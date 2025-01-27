// Copyright (c) 2020 Mercari, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package loaders

import (
	"fmt"
	"io/ioutil"

	parser "github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"

	"github.com/roryq/yo/models"
)

func NewSpannerLoaderFromDDL(fpath string) (*SpannerLoaderFromDDL, error) {
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	tables := make(map[string]tableOrView)
	ddls, err := (&parser.Parser{
		Lexer: &parser.Lexer{
			File: &token.File{FilePath: fpath, Buffer: string(b)},
		},
	}).ParseDDLs()
	if err != nil {
		return nil, err
	}
	for _, ddl := range ddls {
		switch val := ddl.(type) {
		case *ast.CreateTable:
			v := tables[val.Name.Name]
			v.createTable = val
			tables[val.Name.Name] = v
		case *ast.CreateIndex:
			v, ok := tables[val.TableName.Name]
			if !ok {
				return nil, fmt.Errorf("table '%s' is undefined, but got '%s'", val.TableName.Name, ddl.SQL())
			}
			v.createIndexes = append(v.createIndexes, val)
			tables[val.TableName.Name] = v
		case *ast.AlterTable:
			if _, ok := val.TableAlteration.(*ast.AddTableConstraint); ok {
				continue
			}
			return nil, fmt.Errorf("stmt should be CreateTable, CreateIndex or AlterTableAddConstraint, but got '%s'", ddl.SQL())
		case *ast.CreateView:
			v := tables[val.Name.Name]
			v.createView = val
			tables[val.Name.Name] = v
		}
	}

	return &SpannerLoaderFromDDL{tables: tables}, nil
}

type tableOrView struct {
	createTable   *ast.CreateTable
	createIndexes []*ast.CreateIndex
	createView    *ast.CreateView
}

type SpannerLoaderFromDDL struct {
	tables map[string]tableOrView
}

func (s *SpannerLoaderFromDDL) ParamN(n int) string {
	return fmt.Sprintf("@param%d", n)
}

func (s *SpannerLoaderFromDDL) MaskFunc() string {
	return "?"
}

func (s *SpannerLoaderFromDDL) ParseType(dt string, nullable bool) (int, string, string) {
	return SpanParseType(dt, nullable)
}

func (s *SpannerLoaderFromDDL) ValidCustomType(dataType string, customType string) bool {
	return SpanValidateCustomType(dataType, customType)
}

func (s *SpannerLoaderFromDDL) TableList() ([]*models.Table, error) {
	var tables []*models.Table
	for _, t := range s.tables {
		tables = append(tables, &models.Table{
			TableName: t.createTable.Name.Name,
			ManualPk:  true,
		})
	}

	return tables, nil
}

func (s *SpannerLoaderFromDDL) ColumnList(name string) ([]*models.Column, error) {
	var cols []*models.Column
	table := s.tables[name].createTable

	check := make(map[string]struct{})
	for _, pk := range table.PrimaryKeys {
		check[pk.Name.Name] = struct{}{}
	}

	for i, c := range table.Columns {
		_, pk := check[c.Name.Name]
		cols = append(cols, &models.Column{
			FieldOrdinal: i + 1,
			ColumnName:   c.Name.Name,
			DataType:     c.Type.SQL(),
			NotNull:      c.NotNull,
			IsPrimaryKey: pk,
			IsGenerated:  c.GeneratedExpr != nil,
		})
	}

	return cols, nil
}

func (s *SpannerLoaderFromDDL) IndexList(name string) ([]*models.Index, error) {
	var indexes []*models.Index
	for _, index := range s.tables[name].createIndexes {
		indexes = append(indexes, &models.Index{
			IndexName: index.Name.Name,
			IsUnique:  index.Unique,
		})
	}

	return indexes, nil
}

func (s *SpannerLoaderFromDDL) IndexColumnList(table, index string) ([]*models.IndexColumn, error) {
	if index == "PRIMARY_KEY" {
		return s.primaryKeyColumnList(table)
	}

	var cols []*models.IndexColumn
	for _, ix := range s.tables[table].createIndexes {
		if ix.Name.Name != index {
			continue
		}

		// add storing columns first
		if ix.Storing != nil {
			for _, c := range ix.Storing.Columns {
				cols = append(cols, &models.IndexColumn{
					SeqNo:      0,
					Storing:    true,
					ColumnName: c.Name,
				})
			}
		}

		for i, c := range ix.Keys {
			cols = append(cols, &models.IndexColumn{
				SeqNo:      i + 1,
				ColumnName: c.Name.Name,
			})
		}
		break
	}

	return cols, nil
}

func (s *SpannerLoaderFromDDL) primaryKeyColumnList(table string) ([]*models.IndexColumn, error) {
	tbl, ok := s.tables[table]
	if !ok {
		return nil, nil
	}

	// lookup PK for read-only view
	if tbl.createView != nil {
		sourceTable, err := baseTablesForViewDDL(tbl.createView.SQL())
		if err != nil {
			return nil, err
		}
		tbl = s.tables[firstOrDefault(sourceTable)]
	}

	var cols []*models.IndexColumn
	for i, key := range tbl.createTable.PrimaryKeys {
		cols = append(cols, &models.IndexColumn{
			SeqNo:      i + 1,
			ColumnName: key.Name.Name,
		})
	}

	return cols, nil
}

func baseTablesForViewDDL(ddlString string) ([]string, error) {
	p := &parser.Parser{Lexer: &parser.Lexer{File: &token.File{Buffer: ddlString}}}
	ddl, err := p.ParseDDL()
	if err != nil {
		return nil, err
	}
	from := ddl.(*ast.CreateView).Query.(*ast.Select).From.Source

	switch t := from.(type) {
	case *ast.TableName:
		return []string{t.Table.Name}, nil
	case *ast.Join:
		return nil, fmt.Errorf("view with join is not supported")
	default:
		return nil, fmt.Errorf("unknown source type: %T", from)
	}
}

func firstOrDefault[T any](s []T) T {
	if len(s) == 0 {
		return *new(T)
	}
	return s[0]
}

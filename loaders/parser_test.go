package loaders

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_baseTableForViewDDL(t *testing.T) {
	tests := map[string]struct {
		ddlString string
		want      []string
		wantErr   bool
	}{
		"SingleBaseTable": {
			ddlString: `CREATE VIEW SomeTypes SQL SECURITY INVOKER AS SELECT FullTypes.PKey FROM FullTypes`,
			want:      []string{"FullTypes"},
		},
		"JoinTable": {
			ddlString: `CREATE VIEW SomeTypes SQL SECURITY INVOKER AS SELECT FullTypes.PKey, SecondTable.Column FROM FullTypes JOIN SecondTable USING (PKey)`,
			want:      []string{"FullTypes", "SecondTable"},
		},
		"JoinToSubquery": {
			ddlString: viewDDL(`SELECT k.PKey FROM FullTypes AS f RIGHT JOIN (SELECT PKey FROM SecondTable) AS k ON k.PKey = f.PKey`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"SubqueryFirst": {
			ddlString: viewDDL(`SELECT k.PKey FROM (SELECT PKey FROM FullTypes) AS k JOIN SecondTable USING (PKey)`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"NestedSubquery": {
			ddlString: viewDDL(`SELECT y.PKey FROM (SELECT x.PKey FROM (SELECT PKey FROM FullTypes) AS x) AS y`),
			want:      []string{"FullTypes"},
		},
		"SubqueryWithJoin": {
			ddlString: viewDDL(`SELECT s.PKey FROM (SELECT f.PKey FROM FullTypes AS f JOIN SecondTable USING (PKey)) AS s`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"SubqueryWithUnionAll": {
			ddlString: viewDDL(`SELECT u.PKey FROM (SELECT PKey FROM FullTypes UNION ALL SELECT PKey FROM SecondTable) AS u`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"SubqueryWithParenthesizedUnion": {
			ddlString: viewDDL(`SELECT u.PKey FROM ((SELECT PKey FROM FullTypes) UNION ALL (SELECT PKey FROM SecondTable)) AS u`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"TopLevelUnionAll": {
			ddlString: viewDDL(`SELECT PKey FROM FullTypes UNION ALL SELECT PKey FROM SecondTable`),
			want:      []string{"FullTypes", "SecondTable"},
		},
		"OrderByAndLimit": {
			ddlString: viewDDL(`SELECT PKey FROM FullTypes ORDER BY PKey LIMIT 10`),
			want:      []string{"FullTypes"},
		},
		"SubqueryWithOrderByAndLimit": {
			ddlString: viewDDL(`SELECT s.PKey FROM (SELECT PKey FROM FullTypes ORDER BY PKey LIMIT 10) AS s`),
			want:      []string{"FullTypes"},
		},
		"NoFromClause": {
			ddlString: viewDDL(`SELECT 1 AS PKey`),
		},
		"Error_InvalidDDL": {
			ddlString: `CREATE VIEW`,
			wantErr:   true,
		},
		"Error_NotAView": {
			ddlString: `CREATE TABLE FullTypes (PKey STRING(36) NOT NULL) PRIMARY KEY (PKey)`,
			wantErr:   true,
		},
		"Error_UnsupportedSource": {
			ddlString: viewDDL(`SELECT x FROM UNNEST([1, 2]) AS x`),
			wantErr:   true,
		},
		"Error_UnsupportedSourceInSubquery": {
			ddlString: viewDDL(`SELECT s.x FROM (SELECT x FROM UNNEST([1, 2]) AS x) AS s`),
			wantErr:   true,
		},
		"Error_UnsupportedSourceInJoinLeft": {
			ddlString: viewDDL(`SELECT f.PKey FROM UNNEST([1, 2]) AS x JOIN FullTypes AS f ON TRUE`),
			wantErr:   true,
		},
		"Error_UnsupportedSourceInJoinRight": {
			ddlString: viewDDL(`SELECT f.PKey FROM FullTypes AS f JOIN UNNEST([1, 2]) AS x ON TRUE`),
			wantErr:   true,
		},
		"Error_UnsupportedSourceInUnion": {
			ddlString: viewDDL(`SELECT PKey FROM FullTypes UNION ALL SELECT x FROM UNNEST([1, 2]) AS x`),
			wantErr:   true,
		},
		"Error_WithClause": {
			ddlString: viewDDL(`WITH c AS (SELECT PKey FROM FullTypes) SELECT c.PKey FROM c`),
			wantErr:   true,
		},
		"Error_UnsupportedQuery": {
			ddlString: viewDDL(`FROM FullTypes`),
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := baseTablesForViewDDL(tt.ddlString)
			if (err != nil) != tt.wantErr {
				t.Errorf("baseTableForViewDDL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}
}

func Test_primaryKeyColumnList(t *testing.T) {
	t.Run("NestedViews", func(t *testing.T) {
		loader := ddlLoader(t, `
CREATE TABLE BaseTable (
  AccountID STRING(36) NOT NULL
) PRIMARY KEY (AccountID);

CREATE VIEW Layer1 SQL SECURITY INVOKER AS
  SELECT b.AccountID FROM BaseTable AS b;

CREATE VIEW Layer2 SQL SECURITY INVOKER AS
  SELECT l1.AccountID FROM Layer1 AS l1;`)

		cols, err := loader.primaryKeyColumnList("Layer2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"AccountID"}
		got := make([]string, len(cols))
		for i, c := range cols {
			got[i] = c.ColumnName
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("JoinToSubquery", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL+`
CREATE VIEW Joined SQL SECURITY INVOKER AS
  SELECT k.AccountID FROM BaseTable AS b
  RIGHT JOIN (SELECT x.AccountID FROM BaseTable AS x UNION ALL SELECT o.AccountID FROM OtherTable AS o) AS k ON k.AccountID = b.AccountID;`)

		got := primaryKeyNames(t, loader, "Joined")

		if diff := cmp.Diff(got, []string{"AccountID"}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("SubqueryFirst", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL+`
CREATE VIEW FromSubquery SQL SECURITY INVOKER AS
  SELECT k.AccountID FROM (SELECT o.AccountID FROM OtherTable AS o) AS k JOIN BaseTable AS b ON b.AccountID = k.AccountID;`)

		got := primaryKeyNames(t, loader, "FromSubquery")

		if diff := cmp.Diff(got, []string{"AccountID", "OtherID"}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("SubqueryOverView", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL+`
CREATE VIEW Layer1 SQL SECURITY INVOKER AS SELECT b.AccountID FROM BaseTable AS b;
CREATE VIEW OverView SQL SECURITY INVOKER AS SELECT k.AccountID FROM (SELECT l.AccountID FROM Layer1 AS l) AS k;`)

		got := primaryKeyNames(t, loader, "OverView")

		if diff := cmp.Diff(got, []string{"AccountID"}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("Error_WithClause", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL+`
CREATE VIEW WithView SQL SECURITY INVOKER AS
  WITH keys AS (SELECT o.AccountID FROM OtherTable AS o) SELECT keys.AccountID FROM keys;`)

		_, err := loader.primaryKeyColumnList("WithView")
		if err == nil {
			t.Fatal("expected error for a WITH clause, got nil")
		}
	})

	t.Run("UnknownTable_HasNoPrimaryKey", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL)

		got := primaryKeyNames(t, loader, "Missing")

		if diff := cmp.Diff(got, []string{}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("ViewOfUnknownTable_HasNoPrimaryKey", func(t *testing.T) {
		loader := ddlLoader(t, primaryKeyViewsDDL+`
CREATE VIEW OfMissing SQL SECURITY INVOKER AS SELECT m.AccountID FROM MissingTable AS m;`)

		got := primaryKeyNames(t, loader, "OfMissing")

		if diff := cmp.Diff(got, []string{}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("ViewWithoutFromClause_HasNoPrimaryKey", func(t *testing.T) {
		loader := ddlLoader(t, `CREATE VIEW Constant SQL SECURITY INVOKER AS SELECT 1 AS AccountID;`)

		got := primaryKeyNames(t, loader, "Constant")

		if diff := cmp.Diff(got, []string{}); diff != "" {
			t.Errorf("primaryKeyColumnList() (-got, +want)\n%s", diff)
		}
	})

	t.Run("Error_UnsupportedSource", func(t *testing.T) {
		loader := ddlLoader(t, `CREATE VIEW Unnested SQL SECURITY INVOKER AS SELECT x FROM UNNEST([1, 2]) AS x;`)

		_, err := loader.primaryKeyColumnList("Unnested")
		if err == nil {
			t.Fatal("expected error for an unsupported source, got nil")
		}
	})

	t.Run("Error_CircularReference", func(t *testing.T) {
		loader := ddlLoader(t, `
CREATE VIEW ViewA SQL SECURITY INVOKER AS SELECT x FROM ViewB;
CREATE VIEW ViewB SQL SECURITY INVOKER AS SELECT x FROM ViewA;`)

		_, err := loader.primaryKeyColumnList("ViewA")
		if err == nil {
			t.Fatal("expected error for circular view reference, got nil")
		}
	})
}

func ddlLoader(t *testing.T, ddl string) *SpannerLoaderFromDDL {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema.sql")
	if err := os.WriteFile(path, []byte(ddl), 0600); err != nil {
		t.Fatal(err)
	}
	loader, err := NewSpannerLoaderFromDDL(path)
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

var update = flag.Bool("update", false, "update the golden files")

// Test_viewSources_Golden reads the views of testdata/view_sources.sql through the DDL loader and compares the tables and
// the primary key that each resolves to with testdata/view_sources.golden. Run `go test ./loaders -update` to regenerate it.
func Test_viewSources_Golden(t *testing.T) {
	loader, err := NewSpannerLoaderFromDDL(filepath.Join("testdata", "view_sources.sql"))
	if err != nil {
		t.Fatal(err)
	}

	var views []string
	for name, tbl := range loader.tables {
		if tbl.createView != nil {
			views = append(views, name)
		}
	}
	sort.Strings(views)

	var got strings.Builder
	for _, name := range views {
		baseTables, err := baseTablesForViewDDL(loader.tables[name].createView.SQL())
		if err != nil {
			t.Fatalf("baseTablesForViewDDL(%s): %v", name, err)
		}
		fmt.Fprintf(&got, "%s\n  base tables: %s\n  primary key: %s\n", name, strings.Join(baseTables, ", "), strings.Join(primaryKeyNames(t, loader, name), ", "))
	}

	goldenPath := filepath.Join("testdata", "view_sources.golden")
	if *update {
		if err := os.WriteFile(goldenPath, []byte(got.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), got.String()); diff != "" {
		t.Errorf("views (-want, +got)\n%s", diff)
	}
}

// primaryKeyViewsDDL is the tables that the views of the primary key tests read.
const primaryKeyViewsDDL = `
CREATE TABLE BaseTable (
  AccountID STRING(36) NOT NULL
) PRIMARY KEY (AccountID);

CREATE TABLE OtherTable (
  AccountID STRING(36) NOT NULL,
  OtherID STRING(36) NOT NULL
) PRIMARY KEY (AccountID, OtherID);
`

func viewDDL(query string) string {
	return "CREATE VIEW SomeTypes SQL SECURITY INVOKER AS " + query
}

func primaryKeyNames(t *testing.T, loader *SpannerLoaderFromDDL, table string) []string {
	t.Helper()
	cols, err := loader.primaryKeyColumnList(table)
	if err != nil {
		t.Fatalf("primaryKeyColumnList(%s): %v", table, err)
	}
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.ColumnName
	}
	return names
}

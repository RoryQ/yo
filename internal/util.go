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

package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kenshaw/snaker"
)

var arrayElemLenRegexp = regexp.MustCompile(`^ARRAY<(?:STRING|BYTES)\(([0-9]+)\)>$`)

// arrayElemLen returns the element length of an ARRAY<STRING(N)> or ARRAY<BYTES(N)>
// data type. It returns -1 for MAX lengths and for any other data type.
func arrayElemLen(dt string) int {
	m := arrayElemLenRegexp.FindStringSubmatch(dt)
	if m == nil {
		return -1
	}

	l, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return l
}

// reverseIndexRune finds the last rune r in s, returning -1 if not present.
func reverseIndexRune(s string, r rune) int {
	if s == "" {
		return -1
	}

	rs := []rune(s)
	for i := len(rs) - 1; i >= 0; i-- {
		if rs[i] == r {
			return i
		}
	}

	return -1
}

// SinguralizeIdentifier will singularize a identifier, returning it in
// CamelCase.
func SingularizeIdentifier(in Inflector, s string) string {
	if i := reverseIndexRune(s, '_'); i != -1 {
		s = s[:i] + "_" + in.Singularize(s[i+1:])
	} else {
		s = in.Singularize(s)
	}

	// return snaker.SnakeToCamelIdentifier(s)
	return snaker.ForceCamelIdentifier(s)
}

// EscapeColumnName will escape a column name if using reserved keyword as column name, returning it in
// surrounded backquotes.
func EscapeColumnName(s string) string {
	if _, ok := reservedKeywords[strings.ToUpper(s)]; ok {
		// return surrounded s with backquotes if reserved keyword
		return fmt.Sprintf("`%s`", s)
	}
	// return s if not reserved keyword
	return s
}

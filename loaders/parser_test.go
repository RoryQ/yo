package loaders

import (
	"os"
	"path/filepath"
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

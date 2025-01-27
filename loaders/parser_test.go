package loaders

import (
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

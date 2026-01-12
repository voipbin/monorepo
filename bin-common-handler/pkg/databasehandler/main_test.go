package databasehandler

import (
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

type TestType string

const (
	TestTypeA TestType = "A"
)

func Test_ApplyFields(t *testing.T) {
	type filter map[string]any

	tests := []struct {
		name      string
		fields    filter
		wantSQL   string
		expectErr bool
	}{
		{
			name: "uuid.UUID should be converted to bytes",
			fields: filter{
				"id": uuid.FromStringOrNil("0dedc3bc-21f8-11f0-a356-d744a53a4247"),
			},
			wantSQL: `SELECT * FROM dummy WHERE id = ?`,
		},
		{
			name: "string value",
			fields: filter{
				"name": "Alice",
			},
			wantSQL: `SELECT * FROM dummy WHERE name = ?`,
		},
		{
			name: "typed string value",
			fields: filter{
				"name": TestTypeA,
			},
			wantSQL: `SELECT * FROM dummy WHERE name = ?`,
		},
		{
			name: "int types",
			fields: filter{
				"int": 42,
			},
			wantSQL: `SELECT * FROM dummy WHERE int = ?`,
		},
		{
			name: "float64 types",
			fields: filter{
				"float64": 3.14,
			},
			wantSQL: `SELECT * FROM dummy WHERE float64 = ?`,
		},
		{
			name: "bool field not deleted",
			fields: filter{
				"deleted": false,
			},
			wantSQL: `SELECT * FROM dummy WHERE tm_delete >= ?`,
		},
		{
			name: "bool field normal",
			fields: filter{
				"active": true,
			},
			wantSQL: `SELECT * FROM dummy WHERE active = ?`,
		},
		{
			name: "unsupported type should return error",
			fields: filter{
				"unsupported": []int{1, 2, 3},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := squirrel.Select("*").From("dummy")

			got, err := ApplyFields(sb, tt.fields)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sql, _, err := got.ToSql()
			if err != nil {
				t.Fatalf("ToSql error: %v", err)
			}

			sql = strings.Join(strings.Fields(sql), " ")
			want := strings.Join(strings.Fields(tt.wantSQL), " ")

			if sql != want {
				t.Errorf("SQL mismatch:\nwant: %s\ngot:  %s", want, sql)
			}
		})
	}
}

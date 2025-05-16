package databasehandler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

type marshalerType struct {
	Value string
}

func (m marshalerType) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"value": m.Value})
}

type testStruct struct {
	A int
	B string
}

type TestType string

const (
	TestTypeA TestType = "A"
)

func Test_PrepareUpdateFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "uuid.UUID should be converted to bytes",
			input: map[string]any{
				"id": uuid.FromStringOrNil("0dedc3bc-21f8-11f0-a356-d744a53a4247"),
			},
			expected: map[string]any{
				"id": uuid.FromStringOrNil("0dedc3bc-21f8-11f0-a356-d744a53a4247").Bytes(),
			},
		},
		{
			name: "json.Marshaler should marshal correctly",
			input: map[string]any{
				"custom": marshalerType{Value: "test"},
			},
			expected: map[string]any{
				"custom": []byte(`{"value":"test"}`),
			},
		},
		{
			name: "map should be marshaled to JSON",
			input: map[string]any{
				"map": map[string]int{"a": 1},
			},
			expected: map[string]any{
				"map": []byte(`{"a":1}`),
			},
		},
		{
			name: "slice should be marshaled to JSON",
			input: map[string]any{
				"slice": []int{1, 2, 3},
			},
			expected: map[string]any{
				"slice": []byte(`[1,2,3]`),
			},
		},
		{
			name: "struct should be marshaled to JSON",
			input: map[string]any{
				"struct": testStruct{A: 1, B: "x"},
			},
			expected: map[string]any{
				"struct": []byte(`{"A":1,"B":"x"}`),
			},
		},
		{
			name: "basic type should be kept as-is",
			input: map[string]any{
				"int":  42,
				"bool": true,
			},
			expected: map[string]any{
				"int":  42,
				"bool": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := PrepareUpdateFields(tt.input)

			for k, v := range tt.expected {
				got, ok := res[k]
				if !ok {
					t.Errorf("missing key %q in result", k)
					continue
				}

				expectedBytes, ok1 := v.([]byte)
				gotBytes, ok2 := got.([]byte)
				if ok1 && ok2 {
					if string(expectedBytes) != string(gotBytes) {
						t.Errorf("key %q mismatch:\nexpected: %s\ngot:      %s", k, expectedBytes, gotBytes)
					}
					continue
				}

				if !reflect.DeepEqual(got, v) {
					t.Errorf("key %q mismatch:\nexpected: %#v\ngot:      %#v", k, v, got)
				}
			}
		})
	}
}

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

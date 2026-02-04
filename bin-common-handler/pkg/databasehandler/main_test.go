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
			wantSQL: `SELECT * FROM dummy WHERE tm_delete IS NULL`,
		},
		{
			name: "bool field deleted",
			fields: filter{
				"deleted": true,
			},
			wantSQL: `SELECT * FROM dummy WHERE tm_delete IS NOT NULL`,
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

func Test_ConvertMapToTypedMap_CommaInDBTag(t *testing.T) {
	// This test verifies the fix for db tags with comma-separated values
	// Example: `db:"customer_id,uuid"` should be parsed as field name "customer_id"
	type TestStruct struct {
		ID         uuid.UUID `db:"id,uuid"`
		CustomerID uuid.UUID `db:"customer_id,uuid"`
		Name       string    `db:"name"`
		Active     bool      `db:"active"`
	}

	customerUUID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")

	tests := []struct {
		name    string
		input   map[string]any
		want    map[string]any
		wantErr bool
	}{
		{
			name: "customer_id with UUID string should be converted to uuid.UUID",
			input: map[string]any{
				"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"deleted":     false,
			},
			want: map[string]any{
				"customer_id": customerUUID,
				"deleted":     false,
			},
			wantErr: false,
		},
		{
			name: "id with UUID string should be converted to uuid.UUID",
			input: map[string]any{
				"id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
			},
			want: map[string]any{
				"id": customerUUID,
			},
			wantErr: false,
		},
		{
			name: "multiple UUID fields with comma-separated db tags",
			input: map[string]any{
				"id":          "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"name":        "Test Name",
			},
			want: map[string]any{
				"id":          customerUUID,
				"customer_id": customerUUID,
				"name":        "Test Name",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertMapToTypedMap(tt.input, TestStruct{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertMapToTypedMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare each field
			for key, wantVal := range tt.want {
				gotVal, exists := got[key]
				if !exists {
					t.Errorf("expected key %q not found in result", key)
					continue
				}

				// Special handling for UUID comparison
				if wantUUID, ok := wantVal.(uuid.UUID); ok {
					gotUUID, ok := gotVal.(uuid.UUID)
					if !ok {
						t.Errorf("key %q: expected uuid.UUID, got %T", key, gotVal)
						continue
					}
					if gotUUID != wantUUID {
						t.Errorf("key %q: got UUID %v, want %v", key, gotUUID, wantUUID)
					}
				} else if gotVal != wantVal {
					t.Errorf("key %q: got %v, want %v", key, gotVal, wantVal)
				}
			}

			// Check for unexpected keys
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("unexpected key %q in result with value %v", key, got[key])
				}
			}
		})
	}
}

func Test_ConvertMapToTypedMap_EmbeddedStructs(t *testing.T) {
	// This test verifies that UUID conversion works with embedded structs
	// Real-world example: Conversation struct embeds identity.Identity which has customer_id field
	type Identity struct {
		ID         uuid.UUID `db:"id,uuid"`
		CustomerID uuid.UUID `db:"customer_id,uuid"`
	}

	type TestConversation struct {
		Identity
		Name string `db:"name"`
	}

	customerUUID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")

	tests := []struct {
		name    string
		input   map[string]any
		want    map[string]any
		wantErr bool
	}{
		{
			name: "customer_id from embedded struct should be converted to uuid.UUID",
			input: map[string]any{
				"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"name":        "Test Conversation",
			},
			want: map[string]any{
				"customer_id": customerUUID,
				"name":        "Test Conversation",
			},
			wantErr: false,
		},
		{
			name: "multiple UUID fields from embedded struct",
			input: map[string]any{
				"id":          "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
				"name":        "Test",
			},
			want: map[string]any{
				"id":          customerUUID,
				"customer_id": customerUUID,
				"name":        "Test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertMapToTypedMap(tt.input, TestConversation{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertMapToTypedMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare each field
			for key, wantVal := range tt.want {
				gotVal, exists := got[key]
				if !exists {
					t.Errorf("expected key %q not found in result", key)
					continue
				}

				// Special handling for UUID comparison
				if wantUUID, ok := wantVal.(uuid.UUID); ok {
					gotUUID, ok := gotVal.(uuid.UUID)
					if !ok {
						t.Errorf("key %q: expected uuid.UUID, got %T", key, gotVal)
						continue
					}
					if gotUUID != wantUUID {
						t.Errorf("key %q: got UUID %v, want %v", key, gotUUID, wantUUID)
					}
				} else if gotVal != wantVal {
					t.Errorf("key %q: got %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

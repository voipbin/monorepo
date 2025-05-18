package conversation

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		want      map[Field]any
		expectErr bool
	}{
		{
			name: "normal bool field (deleted)",
			input: map[string]any{
				"deleted": true,
			},
			want: map[Field]any{
				FieldDeleted: true,
			},
		},
		{
			name: "uuid fields (id, customer_id)",
			input: map[string]any{
				"id":          "0dedc3bc-21f8-11f0-a356-d744a53a4247",
				"customer_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			},
			want: map[Field]any{
				FieldID:         uuid.FromStringOrNil("0dedc3bc-21f8-11f0-a356-d744a53a4247"),
				FieldCustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			},
		},
		{
			name: "type field",
			input: map[string]any{
				"type": "inbound",
			},
			want: map[Field]any{
				FieldType: Type("inbound"),
			},
		},
		{
			name: "self field as JSON string",
			input: map[string]any{
				"self": `{"type":"tel","target":"+123456789"}`,
			},
			want: map[Field]any{
				FieldSelf: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
			},
		},
		{
			name: "peer field as map",
			input: map[string]any{
				"peer": map[string]any{
					"type":   "tel",
					"target": "+123456777",
				},
			},
			want: map[Field]any{
				FieldPeer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123456777",
				},
			},
		},
		{
			name: "invalid bool field",
			input: map[string]any{
				"deleted": "yes",
			},
			expectErr: true,
		},
		{
			name: "invalid uuid field",
			input: map[string]any{
				"id": 12345,
			},
			expectErr: true,
		},
		{
			name: "invalid address format",
			input: map[string]any{
				"self": 123,
			},
			expectErr: true,
		},
		{
			name: "unrelated field passes through",
			input: map[string]any{
				"custom": "value",
			},
			want: map[Field]any{
				Field("custom"): "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertStringMapToFieldMap(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("result mismatch:\nwant: %+v\ngot:  %+v", tt.want, got)
			}
		})
	}
}

package account

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name string
		src  map[string]any

		expectRes map[Field]any
		expectErr bool
	}{
		{
			name: "uuid strings are converted and plain strings pass through",
			src: map[string]any{
				"customer_id": "55555555-0000-0000-0000-000000000001",
				"name":        "test name",
			},
			expectRes: map[Field]any{
				FieldCustomerID: uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
				FieldName:       "test name",
			},
		},
		{
			name: "invalid uuid string returns error",
			src: map[string]any{
				"customer_id": "not-a-uuid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ConvertStringMapToFieldMap(tt.src)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

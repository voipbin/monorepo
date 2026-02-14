package account

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name   string
		fields FieldStruct
	}{
		{
			name: "all fields populated",
			fields: FieldStruct{
				ID:            uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
				CustomerID:    uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				Name:          "test account",
				PlanType:      PlanTypeFree,
				Balance:       100.50,
				PaymentType:   PaymentTypePrepaid,
				PaymentMethod: PaymentMethodCreditCard,
				Deleted:       false,
			},
		},
		{
			name: "empty fields",
			fields: FieldStruct{
				ID:            uuid.Nil,
				CustomerID:    uuid.Nil,
				Name:          "",
				PlanType:      "",
				Balance:       0,
				PaymentType:   "",
				PaymentMethod: "",
				Deleted:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that FieldStruct can be instantiated and accessed
			if tt.fields.ID == uuid.Nil && tt.name == "all fields populated" {
				t.Error("Expected non-nil ID for populated test")
			}
			if tt.fields.CustomerID == uuid.Nil && tt.name == "all fields populated" {
				t.Error("Expected non-nil CustomerID for populated test")
			}
		})
	}
}

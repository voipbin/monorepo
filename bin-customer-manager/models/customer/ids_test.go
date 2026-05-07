package customer_test

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

func TestIsInternalSystemID(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
		want bool
	}{
		{"call-manager", customer.IDCallManager, true},
		{"ai-manager", customer.IDAIManager, true},
		{"system", customer.IDSystem, true},
		{"basic-route", customer.IDBasicRoute, true},
		{"random customer", uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := customer.IsInternalSystemID(tt.id); got != tt.want {
				t.Errorf("IsInternalSystemID(%v) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

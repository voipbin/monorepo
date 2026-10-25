package callhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
)

func Test_ValidateCustomerIdentityVerified_covB(t *testing.T) {
	custID := uuid.FromStringOrNil("99999999-0000-0000-0000-000000000001")
	tel := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}
	sip := commonaddress.Address{Type: commonaddress.TypeSIP, Target: "a@example.com"}

	tests := []struct {
		name       string
		cu         *cucustomer.Customer
		customerID uuid.UUID
		direction  call.Direction
		dest       commonaddress.Address
		expect     bool
	}{
		{"incoming skips check", &cucustomer.Customer{}, custID, call.DirectionIncoming, tel, true},
		{"non tel skips check", &cucustomer.Customer{}, custID, call.DirectionOutgoing, sip, true},
		{"call manager bypass", &cucustomer.Customer{}, cucustomer.IDCallManager, call.DirectionOutgoing, tel, true},
		{"ai manager bypass", &cucustomer.Customer{}, cucustomer.IDAIManager, call.DirectionOutgoing, tel, true},
		{"system bypass", &cucustomer.Customer{}, cucustomer.IDSystem, call.DirectionOutgoing, tel, true},
		{"basic route bypass", &cucustomer.Customer{}, cucustomer.IDBasicRoute, call.DirectionOutgoing, tel, true},
		{"nil customer fails open", nil, custID, call.DirectionOutgoing, tel, true},
		{"verified", &cucustomer.Customer{IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified}, custID, call.DirectionOutgoing, tel, true},
		{"not verified", &cucustomer.Customer{IdentityVerificationStatus: cucustomer.IdentityVerificationStatus("")}, custID, call.DirectionOutgoing, tel, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newCovBHandler(t)
			if got := h.ValidateCustomerIdentityVerified(context.Background(), tt.cu, tt.customerID, tt.direction, tt.dest); got != tt.expect {
				t.Errorf("wrong result. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

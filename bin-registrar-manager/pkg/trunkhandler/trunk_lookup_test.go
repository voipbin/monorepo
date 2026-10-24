package trunkhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// Test_Create_duplicateLookup checks how Create reacts to each result of the domain name lookup.
func Test_Create_duplicateLookup(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")
	trunkID := uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001")
	domainName := "lookup-trunk"
	errLookup := errors.New("injected lookup failure")

	tests := []struct {
		name string

		lookupTrunk *trunk.Trunk
		lookupErr   error

		expectCreate  bool
		expectMessage string
		expectErrIs   error
	}{
		{
			name:          "lookup fails",
			lookupErr:     errLookup,
			expectCreate:  false,
			expectMessage: "",
			expectErrIs:   errLookup,
		},
		{
			name:         "not found",
			lookupErr:    dbhandler.ErrNotFound,
			expectCreate: true,
			expectErrIs:  errInjectedErrCase,
		},
		{
			name:         "not found wrapped",
			lookupErr:    fmt.Errorf("lookup: %w", dbhandler.ErrNotFound),
			expectCreate: true,
			expectErrIs:  errInjectedErrCase,
		},
		{
			name:          "already exists",
			lookupTrunk:   errCaseTrunk(),
			expectCreate:  false,
			expectMessage: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newTrunkErrFixture(mc)
			ctx := context.Background()

			defer common.ResetBaseDomainNamesForTest()
			if errSet := common.SetBaseDomainNames("registrar.example.invalid", "trunk.example.invalid"); errSet != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
			}

			f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeTrunk).Return(true, nil)
			f.db.EXPECT().TrunkGetByDomainName(ctx, domainName).Return(tt.lookupTrunk, tt.lookupErr)
			if tt.expectCreate {
				f.util.EXPECT().UUIDCreate().Return(trunkID)
				f.db.EXPECT().TrunkCreate(ctx, gomock.Any()).Return(errInjectedErrCase)
			}

			res, err := f.h.Create(ctx, customerID, "n", "d", domainName, nil, "u", "p", nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
			}
			if tt.expectMessage != "" && !strings.Contains(err.Error(), tt.expectMessage) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectMessage, err)
			}
		})
	}
}

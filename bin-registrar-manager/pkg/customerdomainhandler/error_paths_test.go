package customerdomainhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedDomainDB = errors.New("injected db failure")

func Test_RealmGet_ensureFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, errInjectedDomainDB)

	realm, err := h.RealmGet(ctx, customerID)
	if !errors.Is(err, errInjectedDomainDB) {
		t.Errorf("expected the db failure, got: %v", err)
	}
	if realm != "" {
		t.Errorf("no realm must be returned on failure, got: %q", realm)
	}
}

func Test_MigrateToShortDomain_failures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")
	legacy := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60",
		Realm:       "a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60.reg.example.test",
	}

	tests := []struct {
		name  string
		setup func(m *dbhandler.MockDBHandler)

		wantIs  error
		wantMsg string
	}{
		{
			name: "reading the current row fails",
			setup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, errInjectedDomainDB)
			},
			wantIs: errInjectedDomainDB,
		},
		{
			name: "the update fails with a non duplicate error and is not retried",
			setup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerDomainGet(ctx, customerID).Return(legacy, nil)
				m.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).Return(errInjectedDomainDB).Times(1)
			},
			wantIs: errInjectedDomainDB,
		},
		{
			name: "every attempt collides",
			setup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerDomainGet(ctx, customerID).Return(legacy, nil)
				m.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).Return(dbhandler.ErrDuplicate).Times(labelGenerateMaxAttempts)
			},
			wantMsg: "attempts exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestBaseDomains(t)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &customerDomainHandler{dbBin: mockDB}
			tt.setup(mockDB)

			res, err := h.MigrateToShortDomain(ctx, customerID)
			if err == nil || res != nil {
				t.Fatalf("expected a failure without a result, got: %+v / %v", res, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("expected %v, got: %v", tt.wantIs, err)
			}
			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("expected the message to contain %q, got: %v", tt.wantMsg, err)
			}
		})
	}
}

func Test_UpdateByCustomerID_failures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")

	tests := []struct {
		name  string
		label string
		realm string
		setup func(m *dbhandler.MockDBHandler)

		wantIs error
	}{
		{
			name: "empty label is rejected before any db call", label: "", realm: "ab12.reg.example.test",
			setup: func(m *dbhandler.MockDBHandler) {},
		},
		{
			name: "empty realm is rejected before any db call", label: "ab12", realm: "",
			setup: func(m *dbhandler.MockDBHandler) {},
		},
		{
			name: "update failure", label: "ab12", realm: "ab12.reg.example.test",
			setup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).Return(errInjectedDomainDB)
			},
			wantIs: errInjectedDomainDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &customerDomainHandler{dbBin: mockDB}
			tt.setup(mockDB)

			res, err := h.UpdateByCustomerID(ctx, customerID, tt.label, tt.realm)
			if err == nil || res != nil {
				t.Fatalf("expected a failure without a result, got: %+v / %v", res, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("expected %v, got: %v", tt.wantIs, err)
			}
		})
	}
}

func Test_EventCUCustomerDeleted_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")

	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(errInjectedDomainDB)

	err := h.EventCUCustomerDeleted(ctx, &cucustomer.Customer{ID: customerID})
	if err == nil || !strings.Contains(err.Error(), errInjectedDomainDB.Error()) {
		t.Errorf("expected the delete failure to be reported, got: %v", err)
	}
}

func Test_NewCustomerDomainHandler_keepsSettings(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	for _, enabled := range []bool{true, false} {
		res, ok := NewCustomerDomainHandler(mockDB, enabled).(*customerDomainHandler)
		if !ok {
			t.Fatalf("NewCustomerDomainHandler must return *customerDomainHandler")
		}
		if res.dbBin != mockDB || res.shortLabelEnabled != enabled {
			t.Errorf("settings were not kept. enabled: %v, got: %+v", enabled, res)
		}
	}
}

// Test_EnsureByCustomerID_shortLabelEnabled_createFailure verifies that a create failure that
// is not a duplicate key is returned with its cause and is not retried.
func Test_EnsureByCustomerID_shortLabelEnabled_createFailure(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(errInjectedDomainDB).Times(1)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if res != nil || !errors.Is(err, errInjectedDomainDB) {
		t.Errorf("Wrong match. expect: the create failure without a result, got: %+v / %v", res, err)
	}
}

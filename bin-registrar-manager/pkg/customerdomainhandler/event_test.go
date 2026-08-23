package customerdomainhandler

import (
	"context"
	"fmt"
	"testing"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_EventCUCustomerCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a1a1a1a1-7f83-11ee-8f5a-1b2c3d4e5f60")
	cu := &cucustomer.Customer{ID: customerID}

	existing := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ab12",
		Realm:       "ab12.reg.voipbin.net",
	}

	// idempotent: existing row -> no create
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(existing, nil)

	if err := h.EventCUCustomerCreated(ctx, cu); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventCUCustomerCreated_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("b2b2b2b2-7f83-11ee-8f5a-1b2c3d4e5f60")
	cu := &cucustomer.Customer{ID: customerID}

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, fmt.Errorf("db down"))

	if err := h.EventCUCustomerCreated(ctx, cu); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_EventCUCustomerDeleted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c3c3c3c3-7f83-11ee-8f5a-1b2c3d4e5f60")
	cu := &cucustomer.Customer{ID: customerID}

	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(nil)

	if err := h.EventCUCustomerDeleted(ctx, cu); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventCUCustomerDeleted_missingRowIsOK(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("d4d4d4d4-7f83-11ee-8f5a-1b2c3d4e5f60")
	cu := &cucustomer.Customer{ID: customerID}

	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(fmt.Errorf("wrap: %w", dbhandler.ErrNotFound))

	if err := h.EventCUCustomerDeleted(ctx, cu); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_EventCUCustomerCreated_shortLabelDisabled_createsLegacyRow pins the
// deploy-window behavior end to end for the event-driven create: a brand-new
// customer signup (customer_created) with the short-label flag off gets the
// legacy uuid label/realm.
func Test_EventCUCustomerCreated_shortLabelDisabled_createsLegacyRow(t *testing.T) {
	common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}
	t.Cleanup(common.ResetBaseDomainNamesForTest)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e5e5e5e5-7f88-11ee-8f5a-1b2c3d4e5f60")
	cu := &cucustomer.Customer{ID: customerID}

	var created *customerdomain.CustomerDomain
	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
			created = cd
			return nil
		}),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
			return created, nil
		}),
	)

	if err := h.EventCUCustomerCreated(ctx, cu); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if created == nil {
		t.Fatalf("Wrong match. expect: created row, got: nil")
	}
	if created.DomainLabel != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), created.DomainLabel)
	}
	if created.Realm != customerID.String()+".registrar.voipbin.net" {
		t.Errorf("Wrong match. expect: %s.registrar.voipbin.net, got: %s", customerID.String(), created.Realm)
	}
}

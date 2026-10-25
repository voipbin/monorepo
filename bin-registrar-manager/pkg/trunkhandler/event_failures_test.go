package trunkhandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

var errInjectedEvent = errors.New("injected failure")

func Test_EventCUCustomerDeleted_listFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &trunkHandler{db: mockDB}
	ctx := context.Background()

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("e5412000-f09b-11ee-9f74-5b05e21bb6fc")}

	// no delete call is expected after the listing failed
	mockDB.EXPECT().TrunkList(ctx, uint64(1000), gomock.Any(), gomock.Any()).Return(nil, errInjectedEvent)

	err := h.EventCUCustomerDeleted(ctx, customer)
	if !errors.Is(err, errInjectedEvent) {
		t.Errorf("Wrong match. expect: the listing failure to be returned wrapped, got: %v", err)
	}
}

// Test_EventCUCustomerDeleted_deleteFailureContinues verifies that one trunk that cannot be
// deleted does not stop the deletion of the remaining trunks of the customer.
func Test_EventCUCustomerDeleted_deleteFailureContinues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &trunkHandler{db: mockDB, notifyHandler: mockNotify}
	ctx := context.Background()

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("e5412000-f09b-11ee-9f74-5b05e21bb6fc")}
	failing := &trunk.Trunk{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("e5932bfc-f09b-11ee-bdc7-5fdd86be2c03")}}
	healthy := &trunk.Trunk{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("e5c9394a-f09b-11ee-b203-439bb4611917")}}

	mockDB.EXPECT().TrunkList(ctx, uint64(1000), gomock.Any(), gomock.Any()).Return([]*trunk.Trunk{failing, healthy}, nil)

	mockDB.EXPECT().TrunkDelete(ctx, failing.ID).Return(errInjectedEvent)

	mockDB.EXPECT().TrunkDelete(ctx, healthy.ID).Return(nil)
	mockDB.EXPECT().TrunkGet(ctx, healthy.ID).Return(healthy, nil)
	mockDB.EXPECT().SIPAuthDelete(ctx, healthy.ID).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, trunk.EventTypeTrunkDeleted, healthy)

	if err := h.EventCUCustomerDeleted(ctx, customer); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

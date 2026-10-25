package extensionhandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

var errInjectedEvent = errors.New("injected failure")

func Test_EventCUCustomerDeleted_listFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{dbBin: mockDBBin}
	ctx := context.Background()

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("bd908d76-f09a-11ee-9d6a-bb21638c8f10")}

	// no delete call is expected after the listing failed
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(1000), gomock.Any(), gomock.Any()).Return(nil, errInjectedEvent)

	err := h.EventCUCustomerDeleted(ctx, customer)
	if !errors.Is(err, errInjectedEvent) {
		t.Errorf("Wrong match. expect: the listing failure to be returned wrapped, got: %v", err)
	}
}

// Test_EventCUCustomerDeleted_deleteFailureContinues verifies that one extension that cannot
// be deleted does not stop the deletion of the remaining extensions of the customer.
func Test_EventCUCustomerDeleted_deleteFailureContinues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{dbAst: mockDBAst, dbBin: mockDBBin, notifyHandler: mockNotify}
	ctx := context.Background()

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("bd908d76-f09a-11ee-9d6a-bb21638c8f10")}
	failing := &extension.Extension{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("bd096846-f09a-11ee-abda-3bd84cbc7cd8")}}
	healthy := &extension.Extension{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("bd67d732-f09a-11ee-a465-43839f43bb6f")},
		EndpointID: "1002@case.example.test",
		AuthID:     "1002@case.example.test",
		AORID:      "1002@case.example.test",
	}

	mockDBBin.EXPECT().ExtensionList(ctx, uint64(1000), gomock.Any(), gomock.Any()).Return([]*extension.Extension{failing, healthy}, nil)

	mockDBBin.EXPECT().ExtensionGet(ctx, failing.ID).Return(nil, errInjectedEvent)

	mockDBBin.EXPECT().ExtensionGet(ctx, healthy.ID).Return(healthy, nil).Times(2)
	mockDBBin.EXPECT().ExtensionDelete(ctx, healthy.ID).Return(nil)
	mockDBAst.EXPECT().AstEndpointDelete(ctx, healthy.EndpointID).Return(nil)
	mockDBAst.EXPECT().AstAuthDelete(ctx, healthy.AuthID).Return(nil)
	mockDBAst.EXPECT().AstAORDelete(ctx, healthy.AORID).Return(nil)
	mockDBBin.EXPECT().SIPAuthDelete(ctx, healthy.ID).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionDeleted, healthy)

	if err := h.EventCUCustomerDeleted(ctx, customer); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

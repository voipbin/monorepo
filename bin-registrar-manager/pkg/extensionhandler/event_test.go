package extensionhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer           *cmcustomer.Customer
		responseExtensions []*extension.Extension

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("bd908d76-f09a-11ee-9d6a-bb21638c8f10"),
			},
			responseExtensions: []*extension.Extension{
				{
					ID: uuid.FromStringOrNil("bd096846-f09a-11ee-abda-3bd84cbc7cd8"),
				},
				{
					ID: uuid.FromStringOrNil("bd67d732-f09a-11ee-a465-43839f43bb6f"),
				},
			},

			expectFilter: map[string]string{
				"customer_id": "bd908d76-f09a-11ee-9d6a-bb21638c8f10",
				"deleted":     "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &extensionHandler{
				reqHandler:    mockReq,
				dbAst:         mockDBAst,
				dbBin:         mockDBBin,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDBBin.EXPECT().ExtensionGets(ctx, uint64(1000), gomock.Any(), tt.expectFilter).Return(tt.responseExtensions, nil)

			for _, e := range tt.responseExtensions {

				mockDBBin.EXPECT().ExtensionGet(ctx, e.ID).Return(e, nil)
				mockDBBin.EXPECT().ExtensionDelete(ctx, e.ID).Return(nil)
				mockDBAst.EXPECT().AstEndpointDelete(ctx, e.EndpointID).Return(nil)
				mockDBAst.EXPECT().AstAuthDelete(ctx, e.AuthID).Return(nil)
				mockDBAst.EXPECT().AstAORDelete(ctx, e.AORID).Return(nil)
				mockDBBin.EXPECT().ExtensionGet(ctx, e.ID).Return(e, nil)
				mockDBBin.EXPECT().SIPAuthDelete(ctx, e.ID).Return(nil)
				mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionDeleted, e)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

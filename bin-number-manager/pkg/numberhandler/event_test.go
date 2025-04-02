package numberhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_EventCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer        *cmcustomer.Customer
		responseNumbers []*number.Number

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
			},
			responseNumbers: []*number.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b"),
						CustomerID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
					},
					ProviderName: number.ProviderNameTelnyx,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e39bfb34-ccca-11ee-9c3e-2fba9dd3bf35"),
						CustomerID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
					},
					ProviderName: number.ProviderNameTelnyx,
				},
			},

			expectFilter: map[string]string{
				"customer_id": "82ed53fa-ccca-11ee-be19-17f582a54cf4",
				"deleted":     "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := &numberHandler{
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				utilHandler:         mockUtil,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberGets(ctx, uint64(10000), gomock.Any(), tt.expectFilter).Return(tt.responseNumbers, nil)

			for _, nb := range tt.responseNumbers {

				// Delete()
				mockDB.EXPECT().NumberGet(ctx, nb.ID).Return(nb, nil)

				switch nb.ProviderName {
				case number.ProviderNameTelnyx:
					mockTelnyx.EXPECT().NumberRelease(ctx, nb).Return(nil)
				}

				mockDB.EXPECT().NumberDelete(ctx, nb.ID).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, nb.ID).Return(nb, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, nb.CustomerID, number.EventTypeNumberDeleted, nb)
			}

			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventFlowDeleted(t *testing.T) {

	type test struct {
		name string

		flow *fmflow.Flow

		responseNumbersCallFlow    []*number.Number
		responseNumbersMessageFlow []*number.Number

		expectFiltersCallFlow    map[string]string
		expectFiltersMessageFlow map[string]string
	}

	tests := []test{
		{
			"normal call flow id",

			&fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
				},
			},
			[]*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e9e983b2-7d22-11eb-acd3-13c2efec905d"),
					},
					CallFlowID: uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
				},
			},
			[]*number.Number{},

			map[string]string{
				"call_flow_id": "dd92f3fa-7d22-11eb-be53-47ee94a9bce3",
				"deleted":      "false",
			},
			map[string]string{
				"message_flow_id": "dd92f3fa-7d22-11eb-be53-47ee94a9bce3",
				"deleted":         "false",
			},
		},
		{
			"3 items call flow id",

			&fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("01647c7a-ecb7-11ee-9273-7b59a4ee0467"),
				},
			},

			[]*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("094aa406-7d24-11eb-81d5-2f5e99ab6fc1"),
					},
					CallFlowID: uuid.FromStringOrNil("01647c7a-ecb7-11ee-9273-7b59a4ee0467"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("0993e8dc-7d24-11eb-8bee-dbca074d9894"),
					},
					CallFlowID: uuid.FromStringOrNil("01647c7a-ecb7-11ee-9273-7b59a4ee0467"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("09ada2cc-7d24-11eb-8518-97f716018857"),
					},
					CallFlowID: uuid.FromStringOrNil("01647c7a-ecb7-11ee-9273-7b59a4ee0467"),
				},
			},
			[]*number.Number{},

			map[string]string{
				"call_flow_id": "01647c7a-ecb7-11ee-9273-7b59a4ee0467",
				"deleted":      "false",
			},
			map[string]string{
				"message_flow_id": "01647c7a-ecb7-11ee-9273-7b59a4ee0467",
				"deleted":         "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().NumberGets(gomock.Any(), gomock.Any(), gomock.Any(), tt.expectFiltersCallFlow).Return(tt.responseNumbersCallFlow, nil)
			for _, num := range tt.responseNumbersCallFlow {
				mockDB.EXPECT().NumberUpdateCallFlowID(gomock.Any(), num.ID, uuid.Nil)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().NumberGets(gomock.Any(), gomock.Any(), gomock.Any(), tt.expectFiltersMessageFlow).Return(tt.responseNumbersMessageFlow, nil)
			for _, num := range tt.responseNumbersMessageFlow {
				mockDB.EXPECT().NumberUpdateMessageFlowID(gomock.Any(), num.ID, uuid.Nil)
			}

			if err := h.EventFlowDeleted(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

package listenhandler

import (
	"errors"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

func Test_processV1HooksPaddlePost_errorPaths(t *testing.T) {
	customerID := uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000001")
	backendErr := errors.New("backend failure")

	tests := []struct {
		name string

		// rawData replaces the hook envelope when set; otherwise paddle is wrapped in a hook.
		rawData []byte
		paddle  string

		setup func(m *accounthandler.MockAccountHandler, mp *paddlehandler.MockPaddleHandler)

		expectStatus int
	}{
		// envelope errors
		{
			name:         "hook envelope is not json",
			rawData:      []byte(`not json`),
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:         "received data is not json",
			paddle:       `not json`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},

		// transaction.completed
		{
			name:         "completed - data has the wrong shape",
			paddle:       `{"event_id":"evt_e01","event_type":"transaction.completed","data":"x"}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "completed - renewal failure",
			paddle: `{"event_id":"evt_e02","event_type":"transaction.completed","data":{"id":"txn_e02","subscription_id":"sub_e02"}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleSubscriptionRenew(gomock.Any(), "sub_e02", "evt_e02").Return(backendErr)
			},
			expectStatus: 500,
		},
		{
			name:         "completed - empty subscription id is treated as a one-time purchase without customer",
			paddle:       `{"event_id":"evt_e03","event_type":"transaction.completed","data":{"id":"txn_e03","subscription_id":""}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 200,
		},
		{
			name:         "completed - customer id is empty",
			paddle:       `{"event_id":"evt_e04","event_type":"transaction.completed","data":{"id":"txn_e04","custom_data":{"customer_id":""},"details":{"totals":{"total":"1000"}}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 200,
		},
		{
			name:         "completed - customer id is not a uuid",
			paddle:       `{"event_id":"evt_e05","event_type":"transaction.completed","data":{"id":"txn_e05","custom_data":{"customer_id":"not-a-uuid"},"details":{"totals":{"total":"1000"}}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:         "completed - amount is not an integer",
			paddle:       `{"event_id":"evt_e06","event_type":"transaction.completed","data":{"id":"txn_e06","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"details":{"totals":{"total":"10.50"}}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "completed - top up failure",
			paddle: `{"event_id":"evt_e07","event_type":"transaction.completed","data":{"id":"txn_e07","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"details":{"totals":{"total":"250"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleCreditTopUp(gomock.Any(), customerID, int64(2500000), "evt_e07").Return(backendErr)
			},
			expectStatus: 500,
		},

		// subscription.created
		{
			name:         "created - data has the wrong shape",
			paddle:       `{"event_id":"evt_e10","event_type":"subscription.created","data":[1]}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:         "created - no custom data",
			paddle:       `{"event_id":"evt_e11","event_type":"subscription.created","data":{"id":"sub_e11"}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 200,
		},
		{
			name:         "created - customer id is not a uuid",
			paddle:       `{"event_id":"evt_e12","event_type":"subscription.created","data":{"id":"sub_e12","custom_data":{"customer_id":"zzz","plan_type":"basic"}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:         "created - unknown plan type",
			paddle:       `{"event_id":"evt_e13","event_type":"subscription.created","data":{"id":"sub_e13","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001","plan_type":"platinum"}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "created - create failure",
			paddle: `{"event_id":"evt_e14","event_type":"subscription.created","data":{"id":"sub_e14","customer_id":"paddle_cust_e14","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001","plan_type":"professional"}}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleSubscriptionCreate(gomock.Any(), customerID, account.PlanTypeProfessional, "sub_e14", "paddle_cust_e14", "evt_e14").Return(backendErr)
			},
			expectStatus: 500,
		},

		// subscription.updated
		{
			name:         "updated - data has the wrong shape",
			paddle:       `{"event_id":"evt_e20","event_type":"subscription.updated","data":"x"}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "updated - scheduled cancellation failure",
			paddle: `{"event_id":"evt_e21","event_type":"subscription.updated","data":{"id":"sub_e21","scheduled_change":{"action":"cancel","effective_at":"2030-01-01T00:00:00Z"}}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleSubscriptionScheduleCancel(gomock.Any(), "sub_e21", "evt_e21").Return(backendErr)
			},
			expectStatus: 500,
		},
		{
			name:   "updated - plan change failure",
			paddle: `{"event_id":"evt_e22","event_type":"subscription.updated","data":{"id":"sub_e22","custom_data":{"plan_type":"basic"}}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleSubscriptionUpdate(gomock.Any(), "sub_e22", account.PlanTypeBasic, "evt_e22").Return(backendErr)
			},
			expectStatus: 500,
		},
		{
			name:         "updated - custom data plan type is unknown and there is no price",
			paddle:       `{"event_id":"evt_e23","event_type":"subscription.updated","data":{"id":"sub_e23","custom_data":{"plan_type":"platinum"}}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 200,
		},

		// subscription.canceled
		{
			name:         "canceled - data has the wrong shape",
			paddle:       `{"event_id":"evt_e30","event_type":"subscription.canceled","data":"x"}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "canceled - cancel failure",
			paddle: `{"event_id":"evt_e31","event_type":"subscription.canceled","data":{"id":"sub_e31"}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleSubscriptionCancel(gomock.Any(), "sub_e31", "evt_e31").Return(backendErr)
			},
			expectStatus: 500,
		},

		// transaction.refunded
		{
			name:         "refunded - data has the wrong shape",
			paddle:       `{"event_id":"evt_e40","event_type":"transaction.refunded","data":"x"}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:         "refunded - adjustment amount is not an integer",
			paddle:       `{"event_id":"evt_e41","event_type":"transaction.refunded","data":{"id":"txn_e41","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"100"}},{"totals":{"total":"abc"}}]}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 400,
		},
		{
			name:   "refunded - several adjustments are summed",
			paddle: `{"event_id":"evt_e42","event_type":"transaction.refunded","data":{"id":"txn_e42","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"100"}},{"totals":{"total":"250"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleRefund(gomock.Any(), customerID, int64(3500000), "evt_e42").Return(nil)
			},
			expectStatus: 200,
		},
		{
			name:   "refunded - mixed sign adjustments net to a positive amount",
			paddle: `{"event_id":"evt_e43","event_type":"transaction.refunded","data":{"id":"txn_e43","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"-700"}},{"totals":{"total":"200"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleRefund(gomock.Any(), customerID, int64(5000000), "evt_e43").Return(nil)
			},
			expectStatus: 200,
		},
		{
			name:   "refunded - subscription lookup failure",
			paddle: `{"event_id":"evt_e44","event_type":"transaction.refunded","data":{"id":"txn_e44","subscription_id":"sub_e44","adjustments":[{"totals":{"total":"100"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().GetByPaddleSubscriptionID(gomock.Any(), "sub_e44").Return(nil, backendErr)
			},
			expectStatus: 500,
		},
		{
			name:         "refunded - no way to resolve the customer",
			paddle:       `{"event_id":"evt_e45","event_type":"transaction.refunded","data":{"id":"txn_e45","adjustments":[{"totals":{"total":"100"}}]}}`,
			setup:        func(*accounthandler.MockAccountHandler, *paddlehandler.MockPaddleHandler) {},
			expectStatus: 200,
		},
		{
			name:   "refunded - refund failure",
			paddle: `{"event_id":"evt_e46","event_type":"transaction.refunded","data":{"id":"txn_e46","custom_data":{"customer_id":"c2000000-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"100"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler, _ *paddlehandler.MockPaddleHandler) {
				m.EXPECT().PaddleRefund(gomock.Any(), customerID, int64(1000000), "evt_e46").Return(backendErr)
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockPaddle := paddlehandler.NewMockPaddleHandler(mc)
			h := &listenHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: mockAccount,
				paddleHandler:  mockPaddle,
			}
			tt.setup(mockAccount, mockPaddle)

			data := tt.rawData
			if data == nil {
				data = buildHookData(t, tt.paddle)
			}

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/hooks/paddle",
				Method: sock.RequestMethodPost,
				Data:   data,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, simpleResponse(tt.expectStatus)) {
				t.Errorf("Wrong match.\nexpect: status %d\ngot:    %v", tt.expectStatus, res)
			}
		})
	}
}

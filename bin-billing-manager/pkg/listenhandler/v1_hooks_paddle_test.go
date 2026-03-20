package listenhandler

import (
	"encoding/json"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	hmhook "monorepo/bin-hook-manager/models/hook"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/accounthandler"
)

// buildHookData creates properly-marshaled Hook JSON for test requests.
func buildHookData(t *testing.T, paddleJSON string) []byte {
	t.Helper()
	hook := hmhook.Hook{
		ReceviedURI:  "hook.example.com/billing/paddle",
		ReceivedData: []byte(paddleJSON),
	}
	data, err := json.Marshal(hook)
	if err != nil {
		t.Fatalf("could not marshal hook: %v", err)
	}
	return data
}

func Test_processV1HooksPaddlePost(t *testing.T) {

	tests := []struct {
		name    string
		paddle  string // raw Paddle event JSON
		setup   func(mockAccount *accounthandler.MockAccountHandler)
		expectRes *sock.Response
	}{
		{
			// Paddle v2 sends amounts as integer strings in lowest denomination (cents).
			// "1000" = $10.00 → 1000 × 10,000 = 10,000,000 micros
			name:   "transaction.completed - one-time credit purchase",
			paddle: `{"event_id":"evt_credit_001","event_type":"transaction.completed","data":{"id":"txn_001","subscription_id":null,"custom_data":{"customer_id":"a0000001-0000-0000-0000-000000000001"},"details":{"totals":{"total":"1000"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleCreditTopUp(
					gomock.Any(),
					uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
					int64(10000000),
					"evt_credit_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			// "2999" = $29.99 → 2999 × 10,000 = 29,990,000 micros
			name:   "transaction.completed - subscription renewal",
			paddle: `{"event_id":"evt_renew_001","event_type":"transaction.completed","data":{"id":"txn_002","subscription_id":"sub_001","custom_data":{"customer_id":"a0000002-0000-0000-0000-000000000001"},"details":{"totals":{"total":"2999"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleSubscriptionRenew(
					gomock.Any(),
					"sub_001",
					"evt_renew_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			name:   "subscription.created",
			paddle: `{"event_id":"evt_sub_create_001","event_type":"subscription.created","data":{"id":"sub_001","customer_id":"ctm_paddle_001","custom_data":{"customer_id":"a0000003-0000-0000-0000-000000000001","plan_type":"basic"},"items":[{"price":{"product_id":"pro_basic"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleSubscriptionCreate(
					gomock.Any(),
					uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000001"),
					account.PlanTypeBasic,
					"sub_001",
					"ctm_paddle_001",
					"evt_sub_create_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			name:   "subscription.updated",
			paddle: `{"event_id":"evt_sub_update_001","event_type":"subscription.updated","data":{"id":"sub_002","customer_id":"ctm_paddle_002","custom_data":{"customer_id":"a0000004-0000-0000-0000-000000000001","plan_type":"professional"},"items":[{"price":{"product_id":"pro_professional"}}]}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleSubscriptionUpdate(
					gomock.Any(),
					"sub_002",
					account.PlanTypeProfessional,
					"evt_sub_update_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			name:   "subscription.canceled",
			paddle: `{"event_id":"evt_sub_cancel_001","event_type":"subscription.canceled","data":{"id":"sub_003","customer_id":"ctm_paddle_003","custom_data":{"customer_id":"a0000005-0000-0000-0000-000000000001"}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleSubscriptionCancel(
					gomock.Any(),
					"sub_003",
					"evt_sub_cancel_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			// "500" = $5.00 → 500 × 10,000 = 5,000,000 micros
			name:   "transaction.refunded - with adjustments",
			paddle: `{"event_id":"evt_refund_001","event_type":"transaction.refunded","data":{"id":"txn_003","custom_data":{"customer_id":"a0000006-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"500"}}],"details":{"totals":{"total":"1000"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleRefund(
					gomock.Any(),
					uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
					int64(5000000),
					"evt_refund_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			// "300" = $3.00 → 300 × 10,000 = 3,000,000 micros
			name:   "transaction.refunded - fallback to paddle_subscription_id lookup",
			paddle: `{"event_id":"evt_refund_002","event_type":"transaction.refunded","data":{"id":"txn_004","subscription_id":"sub_fallback_001","adjustments":[{"totals":{"total":"300"}}],"details":{"totals":{"total":"1000"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().GetByPaddleSubscriptionID(
					gomock.Any(),
					"sub_fallback_001",
				).Return(&account.Account{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
					},
				}, nil)
				m.EXPECT().PaddleRefund(
					gomock.Any(),
					uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
					int64(3000000),
					"evt_refund_002",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			// "-500" = -$5.00 → -500 × 10,000 = -5,000,000 micros → abs = 5,000,000
			name:   "transaction.refunded - negative adjustment amounts normalized to positive",
			paddle: `{"event_id":"evt_refund_neg_001","event_type":"transaction.refunded","data":{"id":"txn_neg","custom_data":{"customer_id":"a0000006-0000-0000-0000-000000000001"},"adjustments":[{"totals":{"total":"-500"}}],"details":{"totals":{"total":"1000"}}}}`,
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().PaddleRefund(
					gomock.Any(),
					uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
					int64(5000000),
					"evt_refund_neg_001",
				).Return(nil)
			},
			expectRes: simpleResponse(200),
		},
		{
			name:      "transaction.refunded - no adjustments returns 400",
			paddle:    `{"event_id":"evt_refund_003","event_type":"transaction.refunded","data":{"id":"txn_005","custom_data":{"customer_id":"a0000006-0000-0000-0000-000000000001"},"details":{"totals":{"total":"500"}}}}`,
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectRes: simpleResponse(400),
		},
		{
			name:      "transaction.payment_failed - logged at error, return 200",
			paddle:    `{"event_id":"evt_payment_fail_001","event_type":"transaction.payment_failed","data":{"id":"txn_fail_001"}}`,
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectRes: simpleResponse(200),
		},
		{
			name:      "unknown event type - return 200",
			paddle:    `{"event_id":"evt_unknown_001","event_type":"customer.created","data":{}}`,
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectRes: simpleResponse(200),
		},
		{
			name:      "missing custom_data - return 200",
			paddle:    `{"event_id":"evt_no_custom_001","event_type":"transaction.completed","data":{"id":"txn_no_custom","subscription_id":null,"details":{"totals":{"total":"1000"}}}}`,
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectRes: simpleResponse(200),
		},
		{
			name:      "empty event_id - return 400",
			paddle:    `{"event_id":"","event_type":"transaction.completed","data":{"id":"txn_empty_eid"}}`,
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectRes: simpleResponse(400),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			tt.setup(mockAccount)

			req := &sock.Request{
				URI:    "/v1/hooks/paddle",
				Method: sock.RequestMethodPost,
				Data:   buildHookData(t, tt.paddle),
			}

			res, err := h.processRequest(req)
			if err != nil {
				t.Errorf("processRequest() error = %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("processRequest()\nexpect: %v\ngot:    %v", tt.expectRes, res)
			}
		})
	}
}

func Test_parsePaddleCentsToMicros(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		// Paddle v2 sends amounts as integer strings in lowest denomination (cents for USD).
		// Conversion: micros = cents × 10,000
		{"ten dollars (1000 cents)", "1000", 10000000, false},
		{"fifty cents", "50", 500000, false},
		{"one dollar", "100", 1000000, false},
		{"large amount ($999.99)", "99999", 999990000, false},
		{"zero", "0", 0, false},
		{"five dollars", "500", 5000000, false},
		{"one cent", "1", 10000, false},
		{"negative five dollars", "-500", -5000000, false},
		{"negative five fifty", "-550", -5500000, false},
		{"negative fifty cents", "-50", -500000, false},
		{"invalid", "abc", 0, true},
		{"decimal string rejected", "10.00", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePaddleCentsToMicros(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("parsePaddleCentsToMicros(%q) error = %v, expectErr = %v", tt.input, err, tt.expectErr)
				return
			}
			if result != tt.expected {
				t.Errorf("parsePaddleCentsToMicros(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

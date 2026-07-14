package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/activeflowhandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

func Test_SendWebhookToCustomer(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		responseAccount *account.Account

		expectWebhook *webhook.Webhook
	}{
		{
			"normal",
			uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			"application/json",
			[]byte(`{"type":"call_updated","data":{"type":"call"}}`),

			&account.Account{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},

			&webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"call_updated","data":{"type":"call"}}`)),
			},
		},
		{
			"Korean",
			uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			"application/json",
			[]byte(`{"type":"transcript_created","data":{"message":"안녕하세요!?"}}`),

			&account.Account{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},

			&webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"transcript_created","data":{"message":"안녕하세요!?"}}`)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &webhookHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				accoutHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, tt.expectWebhook)

			err := h.SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(400 * time.Millisecond)
		})
	}
}

func Test_SendWebhookToURI(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		uri        string
		method     webhook.MethodType
		dataType   webhook.DataType
		data       json.RawMessage

		expectWebhook *webhook.Webhook
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			dataType:   "application/json",
			data:       []byte(`{"type":"call_updated","data":{"type":"call"}}`),

			expectWebhook: &webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"call_updated","data":{"type":"call"}}`)),
			},
		},
		{
			name: "Korean",

			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			dataType:   "application/json",
			data:       []byte(`{"type":"transcript_created","data":{"message":"안녕하세요!?"}}`),

			expectWebhook: &webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"transcript_created","data":{"message":"안녕하세요!?"}}`)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &webhookHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				accoutHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, tt.expectWebhook)

			err := h.SendWebhookToURI(ctx, tt.customerID, tt.uri, tt.method, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(400 * time.Millisecond)
		})
	}
}

func Test_SendWebhookToCustomerError(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage
	}{
		{
			"account_get_error",
			uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			"application/json",
			[]byte(`{"test":"value"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &webhookHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				accoutHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(ctx, tt.customerID).Return(nil, fmt.Errorf("account not found"))

			err := h.SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, tt.data)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_SendWebhookToCustomerEmptyWebhookURI(t *testing.T) {
	tests := []struct {
		name            string
		customerID      uuid.UUID
		dataType        webhook.DataType
		data            json.RawMessage
		responseAccount *account.Account
		expectWebhook   *webhook.Webhook
	}{
		{
			"empty_webhook_uri",
			uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			"application/json",
			[]byte(`{"test":"value"}`),
			&account.Account{
				ID:            uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				WebhookMethod: "POST",
				WebhookURI:    "",
			},
			&webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"test":"value"}`)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &webhookHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				accoutHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, tt.expectWebhook)

			err := h.SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}

func Test_SendWebhookToCustomer_system_customer(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage
	}{
		{
			name:       "system customer - should skip webhook",
			customerID: cscustomer.IDSystem,
			dataType:   "application/json",
			data:       []byte(`{"type":"email_created","data":{"id":"test"}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &webhookHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				accoutHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			// NO account Get or PublishEvent calls expected - should return early

			err := h.SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_NewWebhookHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := NewWebhookHandler(mockDB, mockNotify, mockNotify, mockReq, mockAccount, mockActiveflow)
	if h == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}

func Test_SendWebhookToCustomer_activeflow(t *testing.T) {

	activeflowID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		responseAccount *account.Account
		expectWebhook   *webhook.Webhook

		// expectGet controls whether activeflowHandler.Get is expected at all.
		expectGet bool
		// expectGetID is the activeflow id Get must be called with (when expectGet).
		expectGetID uuid.UUID
		// responseDest is what the activeflow resolver returns.
		responseDest *activeflowhandler.Destination
	}{
		{
			name: "nested activeflow_id with positive destination - both customer and activeflow delivered",

			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			dataType:   "application/json",
			data:       []byte(`{"type":"call_updated","data":{"activeflow_id":"11111111-1111-1111-1111-111111111111"}}`),

			responseAccount: &account.Account{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			expectWebhook: &webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"call_updated","data":{"activeflow_id":"11111111-1111-1111-1111-111111111111"}}`)),
			},

			expectGet:    true,
			expectGetID:  activeflowID,
			responseDest: &activeflowhandler.Destination{URI: "af.test.com", Method: webhook.MethodTypePOST},
		},
		{
			name: "nested activeflow_id with negative destination - customer only",

			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			dataType:   "application/json",
			data:       []byte(`{"type":"call_updated","data":{"activeflow_id":"11111111-1111-1111-1111-111111111111"}}`),

			responseAccount: &account.Account{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			expectWebhook: &webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"type":"call_updated","data":{"activeflow_id":"11111111-1111-1111-1111-111111111111"}}`)),
			},

			expectGet:    true,
			expectGetID:  activeflowID,
			responseDest: nil,
		},
		{
			name: "regression guard - top-level activeflow_id (not nested) is ignored",

			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			dataType:   "application/json",
			data:       []byte(`{"activeflow_id":"11111111-1111-1111-1111-111111111111","data":{"type":"call"}}`),

			responseAccount: &account.Account{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			expectWebhook: &webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       json.RawMessage([]byte(`{"activeflow_id":"11111111-1111-1111-1111-111111111111","data":{"type":"call"}}`)),
			},

			expectGet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &webhookHandler{
				db:                mockDB,
				notifyHandler:     mockNotify,
				accoutHandler:     mockMessageTargethandler,
				activeflowHandler: mockActiveflow,
			}

			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, tt.expectWebhook)

			if tt.expectGet {
				mockActiveflow.EXPECT().Get(ctx, tt.expectGetID).Return(tt.responseDest, nil)
			} else {
				// regression guard: a top-level activeflow_id must NOT trigger a resolve.
				mockActiveflow.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			}

			err := h.SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// deliveries happen in goroutines; rely on Get expectations above.
			time.Sleep(400 * time.Millisecond)
		})
	}
}

// Test_SendWebhookToCustomer_DualPublishesWithRoutingKey verifies that SendWebhookToCustomer
// dual-publishes: the existing fanout PublishEvent call on h.notifyHandler still fires, AND
// h.topicNotifyHandler.PublishEventWithRoutingKey fires once per generated routing key. Two
// DISTINCT mocks are used for notifyHandler vs topicNotifyHandler so that a mixup between the
// two fields (calling PublishEventWithRoutingKey on the fanout handler, or vice versa) would
// fail the test -- a single shared mock would not catch this class of bug.
func Test_SendWebhookToCustomer_DualPublishesWithRoutingKey(t *testing.T) {

	customerID := uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e")
	callID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	data := json.RawMessage(fmt.Sprintf(
		`{"id":"%s","customer_id":"%s"}`,
		callID, customerID,
	))

	expectWebhook := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   "application/json",
		Data:       data,
	}

	responseAccount := &account.Account{
		ID:            customerID,
		WebhookMethod: "POST",
		WebhookURI:    "test.com",
	}

	// eventType passed to publishRoutingKeyedEvent is always webhook.EventTypeWebhookPublished
	// ("webhook_published"); resource is the first underscore-delimited segment ("webhook"),
	// but the FULL eventType is used as messageType in the routing key (per createRoutingKeys).
	expectRoutingKey := fmt.Sprintf("customer_id.%s.webhook.webhook_published.%s", customerID, callID)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTopicNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &webhookHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		topicNotifyHandler: mockTopicNotify,
		accoutHandler:      mockMessageTargethandler,
	}

	ctx := context.Background()

	mockMessageTargethandler.EXPECT().Get(ctx, customerID).Return(responseAccount, nil)
	mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, expectWebhook)
	mockTopicNotify.EXPECT().PublishEventWithRoutingKey(ctx, webhook.EventTypeWebhookPublished, expectRoutingKey, data)

	err := h.SendWebhookToCustomer(ctx, customerID, "application/json", data)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	time.Sleep(400 * time.Millisecond)
}

// Test_SendWebhookToURI_DualPublishesWithRoutingKey mirrors the above for SendWebhookToURI,
// per the design doc §6 symmetry note: both entry points feed the same downstream path.
func Test_SendWebhookToURI_DualPublishesWithRoutingKey(t *testing.T) {

	customerID := uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e")
	callID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	data := json.RawMessage(fmt.Sprintf(
		`{"id":"%s","customer_id":"%s"}`,
		callID, customerID,
	))

	expectWebhook := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   "application/json",
		Data:       data,
	}

	expectRoutingKey := fmt.Sprintf("customer_id.%s.webhook.webhook_published.%s", customerID, callID)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTopicNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &webhookHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		topicNotifyHandler: mockTopicNotify,
		accoutHandler:      mockMessageTargethandler,
	}

	ctx := context.Background()

	mockNotify.EXPECT().PublishEvent(ctx, webhook.EventTypeWebhookPublished, expectWebhook)
	mockTopicNotify.EXPECT().PublishEventWithRoutingKey(ctx, webhook.EventTypeWebhookPublished, expectRoutingKey, data)

	err := h.SendWebhookToURI(ctx, customerID, "test.com", webhook.MethodTypePOST, "application/json", data)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	time.Sleep(400 * time.Millisecond)
}


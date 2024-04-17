package webhookhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
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

package webhookhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

func Test_SendWebhookToCustomer(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		messageTarget *account.Account
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)

			h := &webhookHandler{
				db:                   mockDB,
				messageTargetHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(gomock.Any(), tt.customerID).Return(tt.messageTarget, nil)
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
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			dataType:   "application/json",
			data:       []byte(`{"type":"call_updated","data":{"type":"call"}}`),
		},
		{
			name: "Korean",

			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			customerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			dataType:   "application/json",
			data:       []byte(`{"type":"transcript_created","data":{"message":"안녕하세요!?"}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessageTargethandler := accounthandler.NewMockAccountHandler(mc)

			h := &webhookHandler{
				db:                   mockDB,
				messageTargetHandler: mockMessageTargethandler,
			}

			ctx := context.Background()

			err := h.SendWebhookToURI(ctx, tt.customerID, tt.uri, tt.method, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(400 * time.Millisecond)
		})
	}
}

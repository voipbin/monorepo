package webhookhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/messagetargethandler"
)

func TestSendWebhook(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessageTargethandler := messagetargethandler.NewMockMessageTargetHandler(mc)

	h := &webhookHandler{
		db:                   mockDB,
		messageTargetHandler: mockMessageTargethandler,
	}

	tests := []struct {
		name string

		customerID    uuid.UUID
		messageTarget *messagetarget.MessageTarget

		wh *webhook.Webhook
	}{
		{
			"normal",
			uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			&webhook.Webhook{
				CustomerID: uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				DataType:   "application/json",
				Data:       []byte(`{"type":"call_updated","data":{"type":"call"}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockMessageTargethandler.EXPECT().Get(gomock.Any(), tt.customerID).Return(tt.messageTarget, nil)
			err := h.SendWebhook(tt.wh)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

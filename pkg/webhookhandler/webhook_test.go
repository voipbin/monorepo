package webhookhandler

import (
	"context"
	"encoding/json"
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
	mockMessageTargethandler := messagetargethandler.NewMockMessagetargetHandler(mc)

	h := &webhookHandler{
		db:                   mockDB,
		messageTargetHandler: mockMessageTargethandler,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		messageTarget *messagetarget.MessageTarget
	}{
		{
			"normal",
			uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
			"application/json",
			[]byte(`{"type":"call_updated","data":{"type":"call"}}`),

			&messagetarget.MessageTarget{
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

			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("a27dc1d6-8254-11ec-8f09-e30cbed3e51e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockMessageTargethandler.EXPECT().Get(gomock.Any(), tt.customerID).Return(tt.messageTarget, nil)
			err := h.SendWebhook(ctx, tt.customerID, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

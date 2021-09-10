package webhookhandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

func TestSendWebhook(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &webhookHandler{
		db:    mockDB,
		cache: mockCache,
	}

	type test struct {
		name string
		wh   *webhook.Webhook
	}

	tests := []test{
		{
			"normal",
			&webhook.Webhook{
				Method:     "POST",
				WebhookURI: "https://en6r9o98bbx9e.x.pipedream.net",
				DataType:   "application/json",
				Data:       []byte(`{"type":"call_updated","data":{"type":"call"}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := h.SendWebhook(tt.wh)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

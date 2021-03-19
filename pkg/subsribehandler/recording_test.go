package subscribehandler

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/webhookhandler"
)

func TestProcessEventCMRecordingCommon(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockWebhook := webhookhandler.NewMockWebhookHandler(mc)
	// mockWebhook := webhookhandler.NewWebhookHandler(mockDB, mockCache)

	h := &subscribeHandler{
		db:             mockDB,
		cache:          mockCache,
		webhookHandler: mockWebhook,
	}

	type test struct {
		name       string
		event      *rabbitmqhandler.Event
		webhookURI string

		expectData []byte
		response   *http.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:      "recording_started",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"195ba41c-881e-11eb-86a4-27740b020444","user_id":1,"type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","filename":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			"https://endxhr87aa0bkge.m.pipedream.net",

			[]byte(`{"type":"recording_started","data":{"id":"195ba41c-881e-11eb-86a4-27740b020444","user_id":1,"type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","filename":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}}`),
			&http.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockWebhook.EXPECT().SendEvent(tt.webhookURI, webhookhandler.MethodTypePOST, webhookhandler.DataTypeJSON, tt.expectData).Return(tt.response, nil)
			h.processEvent(tt.event)
		})
	}
}

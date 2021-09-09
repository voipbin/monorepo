package notifyhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestRecordingStarted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	exchangeDelay := ""
	exchangeNotify := "bin-manager.call-manager.event"

	h := &notifyHandler{
		sock:           mockSock,
		reqHandler:     mockReq,
		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeNotify,
	}

	type test struct {
		name          string
		r             *recording.Recording
		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}

	tests := []test{
		{
			"normal",
			&recording.Recording{
				ID:          uuid.FromStringOrNil("70fb0206-8618-11eb-96de-eb0202c2e333"),
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("82f0c770-8618-11eb-971f-3bef56169bec"),
				Status:      recording.StatusRecording,
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeRecordingStarted),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"70fb0206-8618-11eb-96de-eb0202c2e333","user_id":1,"type":"call","reference_id":"82f0c770-8618-11eb-971f-3bef56169bec","status":"recording","format":"","filename":"","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			[]byte{},
		},
		{
			"with webhook_uri",
			&recording.Recording{
				ID:          uuid.FromStringOrNil("a29148ce-878b-11eb-a518-83192db03b8d"),
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("a31758d8-878b-11eb-b410-3bd79a48fa1f"),
				Status:      recording.StatusRecording,
				WebhookURI:  "http://test.com/test_webhook",
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeRecordingStarted),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"a29148ce-878b-11eb-a518-83192db03b8d","user_id":1,"type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","filename":"","webhook_uri":"http://test.com/test_webhook","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			[]byte(`{"id":"a29148ce-878b-11eb-a518-83192db03b8d","type":"call","reference_id":"a31758d8-878b-11eb-b410-3bd79a48fa1f","status":"recording","format":"","webhook_uri":"http://test.com/test_webhook","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			if tt.r.WebhookURI != "" {
				mockReq.EXPECT().WMWebhookPOST("POST", tt.r.WebhookURI, dataTypeJSON, string(EventTypeRecordingStarted), tt.expectWebhook)
			}
			h.NotifyRecording(context.Background(), EventTypeRecordingStarted, tt.r)

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func TestRecordingFinished(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	exchangeDelay := ""
	exchangeNotify := "bin-manager.call-manager.event"

	h := &notifyHandler{
		sock:           mockSock,
		reqHandler:     mockReq,
		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeNotify,
	}

	type test struct {
		name          string
		r             *recording.Recording
		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}

	tests := []test{
		{
			"normal",
			&recording.Recording{
				ID:          uuid.FromStringOrNil("d7edc1ec-8618-11eb-9740-9bc23366bed2"),
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("dbb39734-8618-11eb-89c7-3f96da5df55e"),
				Status:      recording.StatusEnd,
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeRecordingFinished),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"d7edc1ec-8618-11eb-9740-9bc23366bed2","user_id":1,"type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","filename":"","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			[]byte{},
		},
		{
			"webhook uri",
			&recording.Recording{
				ID:          uuid.FromStringOrNil("7e886ff2-1070-11ec-ae24-6b3001e9028e"),
				UserID:      1,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("dbb39734-8618-11eb-89c7-3f96da5df55e"),
				Status:      recording.StatusEnd,
				WebhookURI:  "test.com/webhook",
			},
			&rabbitmqhandler.Event{
				Type:      string(EventTypeRecordingFinished),
				Publisher: EventPublisher,
				DataType:  dataTypeJSON,
				Data:      []byte(`{"id":"7e886ff2-1070-11ec-ae24-6b3001e9028e","user_id":1,"type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","filename":"","webhook_uri":"test.com/webhook","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			[]byte(`{"id":"7e886ff2-1070-11ec-ae24-6b3001e9028e","type":"call","reference_id":"dbb39734-8618-11eb-89c7-3f96da5df55e","status":"ended","format":"","webhook_uri":"test.com/webhook","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			if tt.r.WebhookURI != "" {
				mockReq.EXPECT().WMWebhookPOST("POST", tt.r.WebhookURI, dataTypeJSON, string(EventTypeRecordingFinished), tt.expectWebhook)
			}
			h.NotifyRecording(context.Background(), EventTypeRecordingFinished, tt.r)

			time.Sleep(time.Millisecond * 100)

		})
	}
}

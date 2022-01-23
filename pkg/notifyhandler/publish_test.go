package notifyhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

func TestPublishWebhookEvent(t *testing.T) {
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
		publisher:      testPublisher,
	}

	tests := []struct {
		name       string
		eventType  string
		event      *testEvent
		webhookURI string

		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}{

		{
			"normal",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			"",
			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte{},
		},
		{
			"webhook uri",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			"test.com",
			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			if tt.webhookURI != "" {
				mockReq.EXPECT().WMV1WebhookSend(gomock.Any(), "POST", tt.webhookURI, dataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.PublishWebhookEvent(context.Background(), tt.eventType, tt.webhookURI, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func TestPublishWebhook(t *testing.T) {
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
		publisher:      testPublisher,
	}

	tests := []struct {
		name       string
		eventType  string
		event      *testEvent
		webhookURI string

		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}{

		{
			"normal",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			"",
			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte{},
		},
		{
			"webhook uri",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			"test.com",
			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			if tt.webhookURI != "" {
				mockReq.EXPECT().WMV1WebhookSend(gomock.Any(), "POST", tt.webhookURI, dataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.PublishWebhook(context.Background(), tt.eventType, tt.webhookURI, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func TestPublishEvent(t *testing.T) {
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
		publisher:      testPublisher,
	}

	tests := []struct {
		name      string
		eventType string
		event     *testEvent

		expectEvent *rabbitmqhandler.Event
	}{

		{
			"normal",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

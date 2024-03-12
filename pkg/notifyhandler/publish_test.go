package notifyhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

func Test_PublishWebhookEvent(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		eventType  string
		event      *testEvent

		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}{
		{
			"normal",
			uuid.FromStringOrNil("419841c6-825d-11ec-823f-13ee3d677a1b"),
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
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
		{
			"customer id is empty",
			uuid.Nil,
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
			[]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			exchangeNotify := "bin-manager.notify-manager.event"

			h := &notifyHandler{
				sock:        mockSock,
				reqHandler:  mockReq,
				queueNotify: exchangeNotify,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().PublishExchangeEvent(h.queueNotify, "", tt.expectEvent)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WebhookV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}

			h.PublishWebhookEvent(ctx, tt.customerID, tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishWebhook(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		eventType  string
		event      *testEvent

		expectEvent   *rabbitmqhandler.Event
		expectWebhook []byte
	}{

		{
			"normal",
			uuid.FromStringOrNil("8225c952-825d-11ec-a03a-afa5f50337e1"),
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
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
		{
			"customer id is empty",
			uuid.Nil,
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
			[]byte(``),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			exchangeNotify := "bin-manager.call-manager.event"

			h := &notifyHandler{
				sock:        mockSock,
				reqHandler:  mockReq,
				queueNotify: exchangeNotify,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WebhookV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.PublishWebhook(ctx, tt.customerID, tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishEvent(t *testing.T) {

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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			exchangeNotify := "bin-manager.call-manager.event"

			h := &notifyHandler{
				sock:        mockSock,
				reqHandler:  mockReq,
				queueNotify: exchangeNotify,
				publisher:   testPublisher,
			}

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().PublishExchangeEvent(h.queueNotify, "", tt.expectEvent)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishEventRaw(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		dataType  string
		data      []byte

		expectEvent *rabbitmqhandler.Event
	}{
		{
			"normal",

			"test_created",
			"application/json",
			[]byte(`{"type":"ChannelCreated"}`),

			&rabbitmqhandler.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelCreated"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			exchangeNotify := "bin-manager.call-manager.event"

			h := &notifyHandler{
				sock:        mockSock,
				reqHandler:  mockReq,
				queueNotify: exchangeNotify,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishExchangeEvent(h.queueNotify, "", tt.expectEvent)

			h.PublishEventRaw(ctx, tt.eventType, tt.dataType, tt.data)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

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
			exchangeDelay := ""
			exchangeNotify := "bin-manager.notify-manager.event"

			h := &notifyHandler{
				sock:           mockSock,
				reqHandler:     mockReq,
				exchangeDelay:  exchangeDelay,
				exchangeNotify: exchangeNotify,
				publisher:      testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().PublishExchangeEvent(h.exchangeNotify, "", tt.expectEvent)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WMV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}

			h.PublishWebhookEvent(ctx, tt.customerID, tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func TestPublishWebhook(t *testing.T) {

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
			exchangeDelay := ""
			exchangeNotify := "bin-manager.call-manager.event"

			h := &notifyHandler{
				sock:           mockSock,
				reqHandler:     mockReq,
				exchangeDelay:  exchangeDelay,
				exchangeNotify: exchangeNotify,
				publisher:      testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WMV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.PublishWebhook(ctx, tt.customerID, tt.eventType, tt.event)

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

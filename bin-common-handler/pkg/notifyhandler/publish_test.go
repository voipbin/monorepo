package notifyhandler

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_PublishWebhookEvent(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		eventType  string
		event      *testEvent

		expectEvent   *sock.Event
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
			&sock.Event{
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
			&sock.Event{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)
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

		expectEvent   *sock.Event
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
			&sock.Event{
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
			&sock.Event{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
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

		expectEvent *sock.Event
	}{

		{
			"normal",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)

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

		expectEvent *sock.Event
	}{
		{
			"normal",

			"test_created",
			"application/json",
			[]byte(`{"type":"ChannelCreated"}`),

			&sock.Event{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)

			h.PublishEventRaw(ctx, tt.eventType, tt.dataType, tt.data)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_publishToClickHouse_NilClient(t *testing.T) {
	// When chClient has no value stored (zero Value), publishToClickHouse should return immediately.
	h := &notifyHandler{
		publisher: testPublisher,
	}
	// chClient is zero-value atomic.Value (Load() returns nil)

	// Should not panic or block
	h.publishToClickHouse("test_event", "application/json", []byte(`{"key":"value"}`))
}

func Test_publishToClickHouse_WrongType(t *testing.T) {
	// When chClient stores a non-clickhouse.Conn value, the type assertion should fail
	// and publishToClickHouse should return safely.
	h := &notifyHandler{
		publisher: testPublisher,
	}
	h.chClient.Store("not-a-clickhouse-conn")

	// Should not panic — type assertion guard catches this
	h.publishToClickHouse("test_event", "application/json", []byte(`{"key":"value"}`))
}

func Test_publishEvent_SkipsClickHouseWhenClientNil(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler: mockSock,
		queueNotify: commonoutline.QueueNameCallEvent,
		publisher:   testPublisher,
	}
	// chClient is zero-value (nil) — ClickHouse path should be skipped

	mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)

	err := h.publishEvent("test_event", "application/json", []byte(`{"key":"value"}`), 3, 0)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Give a moment for any unexpected goroutine (there should be none)
	time.Sleep(100 * time.Millisecond)
}

func Test_chClient_ConcurrentStoreLoad(t *testing.T) {
	// Verify that concurrent Store and Load on chClient do not cause a data race.
	// This test is meaningful when run with -race flag.
	h := &notifyHandler{
		publisher: testPublisher,
	}

	var wg sync.WaitGroup

	// Writer goroutine: stores values
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			h.chClient.Store("dummy-value")
		}
	}()

	// Reader goroutine: loads values
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = h.chClient.Load()
		}
	}()

	wg.Wait()

	// After all writes, Load should return the stored value
	val := h.chClient.Load()
	if val == nil {
		t.Error("Expected chClient to have a value after concurrent stores")
	}
}

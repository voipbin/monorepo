package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

const (
	testPublisher = "test"
)

type testEvent struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

func (h *testEvent) CreateWebhookEvent() ([]byte, error) {
	m, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// testIDEvent carries a top-level id, so the global topic publish resolves the subscription
// address through the default json fallback (VOIP-1404).
type testIDEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// testStreamEvent overrides the subscription address with its parent stream id, the way
// streaming.Speech does. Pointer receiver, as the interface requires.
type testStreamEvent struct {
	ID           string `json:"id"`
	TranscribeID string `json:"transcribe_id"`
}

func (h *testStreamEvent) EventSubscriptionID() string {
	return h.TranscribeID
}

var _ eventtopic.SubscriptionIdentifier = (*testStreamEvent)(nil)

// counterValue reads the current value of the given counter.
func counterValue(c prometheus.Counter) float64 {
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		return 0
	}

	return m.GetCounter().GetValue()
}

// topicPublishCount reads the current value of the topic publish counter. Read it immediately
// before and after the operation under test -- initPrometheus reassigns the package-level
// collectors when a new namespace is initialized.
func topicPublishCount(eventType string, result string) float64 {
	return counterValue(promTopicPublishTotal.WithLabelValues(eventType, result))
}

func topicPlaceholderCount(eventType string) float64 {
	return counterValue(promTopicPlaceholderTotal.WithLabelValues(eventType))
}

func TestMain(m *testing.M) {
	initPrometheus("test")

	os.Exit(m.Run())
}

func TestNewNotifyHandlerForExistingExchange_SkipsDeclare(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	// TopicCreate/TopicCreateWithKind must NOT be called -- the exchange is assumed
	// already declared by the caller.
	mockSock.EXPECT().TopicCreate(gomock.Any()).Times(0)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Times(0)

	h := NewNotifyHandlerForExistingExchange(mockSock, nil, "some.exchange.name", "test-service")

	if h == nil {
		t.Fatal("Expected non-nil NotifyHandler")
	}
}

// Test_WithGlobalTopicPublish_declaresGlobalExchange verifies both constructors declare the
// global topic exchange -- and only through the shared helper with kind "topic" -- when the
// option is enabled (VOIP-1404 design §3/§5.1).
func Test_WithGlobalTopicPublish_declaresGlobalExchange(t *testing.T) {

	tests := []struct {
		name string

		queueEvent commonoutline.QueueName
		publisher  commonoutline.ServiceName

		newHandler func(sockhandler.SockHandler, commonoutline.QueueName, commonoutline.ServiceName) NotifyHandler

		expectFanoutDeclare bool
	}{
		{
			name: "new notify handler",

			queueEvent: commonoutline.QueueNameTranscribeEvent,
			publisher:  "test-service",

			newHandler: func(s sockhandler.SockHandler, q commonoutline.QueueName, p commonoutline.ServiceName) NotifyHandler {
				return NewNotifyHandler(s, nil, q, p, WithGlobalTopicPublish())
			},

			expectFanoutDeclare: true,
		},
		{
			name: "new notify handler for existing exchange",

			queueEvent: commonoutline.QueueNameTranscribeEvent,
			publisher:  "test-service",

			newHandler: func(s sockhandler.SockHandler, q commonoutline.QueueName, p commonoutline.ServiceName) NotifyHandler {
				return NewNotifyHandlerForExistingExchange(s, nil, q, p, WithGlobalTopicPublish())
			},

			expectFanoutDeclare: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			if tt.expectFanoutDeclare {
				mockSock.EXPECT().TopicCreate(string(tt.queueEvent)).Return(nil)
			} else {
				mockSock.EXPECT().TopicCreate(gomock.Any()).Times(0)
			}
			mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind).Return(nil)

			res := tt.newHandler(mockSock, tt.queueEvent, tt.publisher)
			if res == nil {
				t.Fatal("Expected non-nil NotifyHandler")
			}

			h, ok := res.(*notifyHandler)
			if !ok {
				t.Fatalf("Wrong match. unexpected handler type. handler: %T", res)
			}
			if !h.topicEnabled {
				t.Error("Wrong match. expected the topic publish to be enabled.")
			}
			if h.topicDisabled {
				t.Error("Wrong match. expected the topic publish not to be disabled.")
			}
		})
	}
}

// Test_NewNotifyHandler_globalTopicDeclareFailure verifies the degradation contract: a failed
// global-exchange declare must NOT nil out the handler (unlike the fanout declare failure), the
// fanout path stays alive, and every suppressed topic publish is counted (VOIP-1404 design §5.2).
func Test_NewNotifyHandler_globalTopicDeclareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameTranscribeEvent)).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind).Return(fmt.Errorf("precondition failed"))

	res := NewNotifyHandler(mockSock, nil, commonoutline.QueueNameTranscribeEvent, "test-service", WithGlobalTopicPublish())
	if res == nil {
		t.Fatal("Expected non-nil NotifyHandler")
	}

	h, ok := res.(*notifyHandler)
	if !ok {
		t.Fatalf("Wrong match. unexpected handler type. handler: %T", res)
	}
	if !h.topicDisabled {
		t.Error("Wrong match. expected the topic publish to be disabled.")
	}

	// the fanout publish must still happen, and the topic publish must be suppressed and counted.
	eventType := "test_declarefailed"
	before := topicPublishCount(eventType, topicPublishResultError)
	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameTranscribeEvent), "", gomock.Any()).Return(nil)
	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)

	h.PublishEvent(context.Background(), eventType, &testIDEvent{ID: "7cb7b0f8-a1bc-11f1-92ef-60452e5e40a2"})

	if after := topicPublishCount(eventType, topicPublishResultError); after != before+1 {
		t.Errorf("Wrong match. expect: %f, got: %f", before+1, after)
	}
}

// Test_NewNotifyHandler_withoutOption verifies the default: no global exchange declare at all.
func Test_NewNotifyHandler_withoutOption(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameTranscribeEvent)).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Times(0)

	res := NewNotifyHandler(mockSock, nil, commonoutline.QueueNameTranscribeEvent, "test-service")
	if res == nil {
		t.Fatal("Expected non-nil NotifyHandler")
	}

	h, ok := res.(*notifyHandler)
	if !ok {
		t.Fatalf("Wrong match. unexpected handler type. handler: %T", res)
	}
	if h.topicEnabled {
		t.Error("Wrong match. expected the topic publish to be disabled by default.")
	}
}

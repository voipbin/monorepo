package notifyhandler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
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

// EventSubscriptionID returns "" -- testEvent has no id at all, so it doubles as the
// placeholder-by-design fixture on the webhook path (VOIP-1419: PublishWebhookEvent requires
// WebhookEventMessage, so even webhook fixtures declare their address explicitly).
func (h *testEvent) EventSubscriptionID() string {
	return ""
}

var _ WebhookEventMessage = (*testEvent)(nil)

// testIDEvent declares its own top-level id as the subscription address -- the standard own-id
// shape every default resource uses after VOIP-1419 (no JSON fallback exists anymore).
type testIDEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *testIDEvent) EventSubscriptionID() string {
	return h.ID
}

var _ eventtopic.SubscriptionIdentifier = (*testIDEvent)(nil)

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

// testEmptyOverrideEvent implements the override but returns an empty subscription address, while
// its payload still carries a perfectly valid top-level "id". It pins design §4.2: an override
// that EXISTS is authoritative, so the JSON `id` fallback must not run behind its back (VOIP-1404
// code-review round 1, F2).
type testEmptyOverrideEvent struct {
	ID string `json:"id"`
}

func (h *testEmptyOverrideEvent) EventSubscriptionID() string {
	return ""
}

var _ eventtopic.SubscriptionIdentifier = (*testEmptyOverrideEvent)(nil)

// testNilOverrideEvent is the uuid.Nil variant of testEmptyOverrideEvent -- an all-zero address is
// the same "nothing to bind to" as an empty one, and must not fall back either.
type testNilOverrideEvent struct {
	ID string `json:"id"`
}

func (h *testNilOverrideEvent) EventSubscriptionID() string {
	return uuid.Nil.String()
}

var _ eventtopic.SubscriptionIdentifier = (*testNilOverrideEvent)(nil)

// testSpyEvent records whether EventSubscriptionID was invoked. It exists to pin the topicEnabled
// gate in PublishEvent directly (VOIP-1404 code-review round 2, F11): the typed-nil guard makes
// the gate's REMOVAL invisible to every other test, since a well-formed payload resolves happily
// either way. Only an observation of the call itself distinguishes "gated" from "guarded".
type testSpyEvent struct {
	ID           string `json:"id"`
	TranscribeID string `json:"transcribe_id"`

	// called is unexported, so encoding/json ignores it and the marshaled payload is unaffected.
	called bool
}

func (h *testSpyEvent) EventSubscriptionID() string {
	h.called = true
	return h.TranscribeID
}

var _ eventtopic.SubscriptionIdentifier = (*testSpyEvent)(nil)

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

// notifyTotalCount reads the current value of the pre-VOIP-1404 fanout publish counter. Used to
// pin the metric-isolation contract: the topic publish path must never touch it.
func notifyTotalCount(eventType string) float64 {
	return counterValue(promNotifyTotal.WithLabelValues(eventType))
}

// notifyProcessTimeCount reads the observation count of the pre-VOIP-1404 fanout process-time
// histogram. Same isolation contract as notifyTotalCount.
func notifyProcessTimeCount(eventType string) uint64 {
	observer, ok := promNotifyProcessTime.WithLabelValues(eventType).(prometheus.Metric)
	if !ok {
		return 0
	}

	m := &dto.Metric{}
	if err := observer.Write(m); err != nil {
		return 0
	}

	return m.GetHistogram().GetSampleCount()
}

func TestMain(m *testing.M) {
	initPrometheus("test")

	os.Exit(m.Run())
}

// fatalCaptureHook records every log entry fired on the standard logger while it is attached.
// Levels() returns logrus.AllLevels so it also captures the FatalLevel entry logrus.Fatalf emits
// just before calling ExitFunc.
type fatalCaptureHook struct {
	entries []*logrus.Entry
}

func (h *fatalCaptureHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *fatalCaptureHook) Fire(e *logrus.Entry) error {
	h.entries = append(h.entries, e)
	return nil
}

// captureFatal runs fn with the standard logger's ExitFunc stubbed to a no-op (VOIP-1407: several
// notifyhandler construction-time failures now call logrus.Fatalf instead of returning an error,
// and the real os.Exit would kill the test binary) and a hook attached to capture every entry fn
// logs. Returns the last entry logged at logrus.FatalLevel, or nil if none was logged. Both the
// ExitFunc and the hook set are restored before returning, so this is safe to call from multiple
// tests without cross-test interference.
func captureFatal(t *testing.T, fn func()) *logrus.Entry {
	t.Helper()

	origExitFunc := logrus.StandardLogger().ExitFunc
	logrus.StandardLogger().ExitFunc = func(int) {}
	defer func() { logrus.StandardLogger().ExitFunc = origExitFunc }()

	hook := &fatalCaptureHook{}
	origHooks := logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{})
	logrus.AddHook(hook)
	defer func() { logrus.StandardLogger().ReplaceHooks(origHooks) }()

	fn()

	var last *logrus.Entry
	for _, e := range hook.entries {
		if e.Level == logrus.FatalLevel {
			last = e
		}
	}
	return last
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

// Test_WithGlobalTopicPublish_declaresGlobalExchange verifies both constructors declare ONLY the
// global topic exchange -- and only through the shared helper with kind "topic" -- when the
// option is enabled (VOIP-1404 design §3/§5.1). VOIP-1407: a topicEnabled=true construction never
// declares the per-service fanout exchange either (§2.3's `if !h.topicEnabled` guard), so both
// subtests now expect the identical "no fanout declare" shape -- there is no longer a per-subtest
// fanout-declare distinction to parameterize.
func Test_WithGlobalTopicPublish_declaresGlobalExchange(t *testing.T) {

	tests := []struct {
		name string

		queueEvent commonoutline.QueueName
		publisher  commonoutline.ServiceName

		newHandler func(sockhandler.SockHandler, commonoutline.QueueName, commonoutline.ServiceName) NotifyHandler
	}{
		{
			name: "new notify handler",

			queueEvent: commonoutline.QueueNameTranscribeEvent,
			publisher:  "test-service",

			newHandler: func(s sockhandler.SockHandler, q commonoutline.QueueName, p commonoutline.ServiceName) NotifyHandler {
				return NewNotifyHandler(s, nil, q, p, WithGlobalTopicPublish())
			},
		},
		{
			name: "new notify handler for existing exchange",

			queueEvent: commonoutline.QueueNameTranscribeEvent,
			publisher:  "test-service",

			newHandler: func(s sockhandler.SockHandler, q commonoutline.QueueName, p commonoutline.ServiceName) NotifyHandler {
				return NewNotifyHandlerForExistingExchange(s, nil, q, p, WithGlobalTopicPublish())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			mockSock.EXPECT().TopicCreate(gomock.Any()).Times(0)
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
		})
	}
}

// Test_NewNotifyHandler_globalTopicDeclareFailure verifies the VOIP-1407 fatal-on-failure
// contract for initGlobalTopicExchange (design §2.4): a failed global-exchange declare on a
// topicEnabled=true construction calls logrus.Fatalf, replacing the pre-VOIP-1407
// degrade-and-continue behavior (non-nil handler, the topic publish silently suppressed while
// fanout kept flowing) -- once
// fanout is removed for topic-enabled instances, there is no fallback left to degrade to.
func Test_NewNotifyHandler_globalTopicDeclareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	// topicEnabled=true skips the fanout TopicCreate entirely (§2.3) -- only the global topic
	// declare is exercised here.
	mockSock.EXPECT().TopicCreate(gomock.Any()).Times(0)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind).Return(fmt.Errorf("precondition failed"))

	entry := captureFatal(t, func() {
		NewNotifyHandler(mockSock, nil, commonoutline.QueueNameTranscribeEvent, "test-service", WithGlobalTopicPublish())
	})

	if entry == nil {
		t.Fatal("Expected a logrus.Fatalf call, got none")
	}
	if entry.Level != logrus.FatalLevel {
		t.Errorf("Wrong match. expect: %v, got: %v", logrus.FatalLevel, entry.Level)
	}
	if !strings.Contains(entry.Message, "Could not declare the global topic exchange") {
		t.Errorf("Wrong match. unexpected fatal message: %s", entry.Message)
	}
}

// Test_NewNotifyHandler_fanoutDeclareFailure verifies the VOIP-1407 fatal-on-failure contract for
// the fanout-only path's own exchange declare (design §2.3): a failed declare now calls
// logrus.Fatalf, replacing the pre-existing latent bug (log at Error level and return a nil
// handler that no caller ever checked for nil).
func Test_NewNotifyHandler_fanoutDeclareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameTranscribeEvent)).Return(fmt.Errorf("precondition failed"))
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Times(0)

	entry := captureFatal(t, func() {
		NewNotifyHandler(mockSock, nil, commonoutline.QueueNameTranscribeEvent, "test-service")
	})

	if entry == nil {
		t.Fatal("Expected a logrus.Fatalf call, got none")
	}
	if entry.Level != logrus.FatalLevel {
		t.Errorf("Wrong match. expect: %v, got: %v", logrus.FatalLevel, entry.Level)
	}
	if !strings.Contains(entry.Message, "Could not declare the event exchange") {
		t.Errorf("Wrong match. unexpected fatal message: %s", entry.Message)
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

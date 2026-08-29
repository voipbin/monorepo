package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	cfconference "monorepo/bin-conference-manager/models/conference"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	pmmessage "monorepo/bin-pipecat-manager/models/message"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
)

// list of publishers
const (
	publisherCallManager       = string(commonoutline.ServiceNameCallManager)
	publisherTranscribeManager = string(commonoutline.ServiceNameTranscribeManager)
	publisherTTSManager        = string(commonoutline.ServiceNameTTSManager)
)

// topicPatterns is the ruled bind set on the global topic exchange `bin-manager.event`
// (VOIP-1406, design §5): one pattern per dispatched (publisher, event-type) pair.
// The `conference-manager.conference.*.updated` pair is deliberately ABSENT: its
// dispatch case is unreachable today and keeping it unreachable is today's behavior
// (design §4, follow-up VOIP-1422). Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmconfbridge.EventTypeConfbridgeJoined),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmconfbridge.EventTypeConfbridgeLeaved),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallHangup),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmdtmf.EventTypeDTMFReceived),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmmessage.EventTypeUserTranscription),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmmessage.EventTypeBotLLM),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmmessage.EventTypeBotLLMIntermediate),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmpipecatcall.EventTypeInitialized),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmpipecatcall.EventTypePipecatcallTerminated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNamePipecatManager), pmmessage.EventTypeTeamMemberSwitched),
}

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

// subscribeHandler define
type subscribeHandler struct {
	serviceName string
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	aicallHandler  aicallhandler.AIcallHandler
	summaryHandler summaryhandler.SummaryHandler
	messageHandler messagehandler.MessageHandler
}

var (
	metricsNamespace = "ai_manager"

	promSubscribeProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_event_process_time",
			Help:      "Process time of subscribed events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"publisher", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		promSubscribeProcessTime,
	)
}

// NewSubscribeHandler create EventHandler
func NewSubscribeHandler(
	serviceName string,
	sock sockhandler.SockHandler,
	subscribeQueue string,
	aicallHandler aicallhandler.AIcallHandler,
	summaryHandler summaryhandler.SummaryHandler,
	messageHandler messagehandler.MessageHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		serviceName:    serviceName,
		sockHandler:    sock,
		subscribeQueue: subscribeQueue,
		aicallHandler:  aicallHandler,
		summaryHandler: summaryHandler,
		messageHandler: messageHandler,
	}

	return h
}

// Run starts to receive subscribed event and process it.
func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Infof("Creating rabbitmq queue for subscribed event receiving.")

	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// VOIP-1407: topicPatterns/QueueBind is the sole intake mechanism -- the fanout
	// QueueSubscribe loop and the fanout-unbind step have been removed, so there is no
	// fallback left to degrade to. Any failure here is fatal.
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameAIManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume the subscribed evnet message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processEventRun runs the event process handler.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	go h.processEvent(m)

	return nil
}

// processEvent processes received event
func (h *subscribeHandler) processEvent(m *sock.Event) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEvent",
		"event": m,
	})

	ctx := context.Background()

	var err error
	start := time.Now()

	switch {

	// call-manager
	case m.Publisher == publisherCallManager && m.Type == string(cmconfbridge.EventTypeConfbridgeJoined):
		err = h.processEventCMConfbridgeJoined(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmconfbridge.EventTypeConfbridgeLeaved):
		err = h.processEventCMConfbridgeLeaved(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmcall.EventTypeCallHangup):
		err = h.processEventCMCallHangup(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmdtmf.EventTypeDTMFReceived):
		err = h.processEventCMDTMFReceived(ctx, m)

	// conference-manager
	// VOIP-1422: unreachable case -- no fanout subscription and no topic pattern delivers this pair (VOIP-1406 design §4); activate or delete it there.
	case m.Publisher == string(commonoutline.ServiceNameConferenceManager) && m.Type == string(cfconference.EventTypeConferenceUpdated):
		err = h.processEventCMConferenceUpdated(ctx, m)

	// pipecat-manager
	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeUserTranscription):
		err = h.processEventPMMessageUserTranscription(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeBotLLM):
		err = h.processEventPMMessageBotLLM(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeBotLLMIntermediate):
		err = h.processEventPMMessageBotLLMIntermediate(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmpipecatcall.EventTypeInitialized):
		err = h.processEventPMPipecatcallInitialized(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == pmpipecatcall.EventTypePipecatcallTerminated:
		err = h.processEventPMPipecatcallTerminated(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeTeamMemberSwitched):
		err = h.processEventPMTeamMemberSwitched(ctx, m)

	default:
		// ignore the event.
		return
	}

	elapsed := time.Since(start)
	promSubscribeProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}

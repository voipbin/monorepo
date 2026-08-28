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
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "confbridge", "joined"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "confbridge", "leaved"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "call", "hangup"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "dtmf", "received"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "message", "user_transcription"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "message", "bot_llm"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "message", "bot_llm_intermediate"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "pipecatcall", "initialized"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "pipecatcall", "terminated"),
	eventtopic.PatternAction(string(commonoutline.ServiceNamePipecatManager), "team", "member_switched"),
}

// fanoutUnbindTargets lists the old per-service fanout event exchanges to unbind once
// every topicPatterns bind has succeeded (VOIP-1406). It equals the full subscribeTargets
// set wired in cmd/ai-manager: the transcribe and tts subscriptions are dead binds (zero
// dispatch cases) and are dropped together with the live call/pipecat fanout legs
// (design §4). The fanout QueueSubscribe calls in Run() stay until VOIP-1407 as the
// rollback/degrade surface.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCallEvent),
	string(commonoutline.QueueNameTranscribeEvent),
	string(commonoutline.QueueNameTTSEvent),
	string(commonoutline.QueueNamePipecatEvent),
}

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

// subscribeHandler define
type subscribeHandler struct {
	serviceName string
	sockHandler sockhandler.SockHandler

	subscribeQueue   string
	subscribeTargets []string

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
	subscribeTargets []string,
	aicallHandler aicallhandler.AIcallHandler,
	summaryHandler summaryhandler.SummaryHandler,
	messageHandler messagehandler.MessageHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		serviceName:      serviceName,
		sockHandler:      sock,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
		aicallHandler:    aicallHandler,
		summaryHandler:   summaryHandler,
		messageHandler:   messageHandler,
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

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// VOIP-1406: bind the subscribe queue to the global topic exchange `bin-manager.event`
	// with one pattern per dispatched (publisher, event-type) pair, then unbind the old
	// per-service fanout exchanges. Bind-new-BEFORE-unbind-old: the queue is never bound to
	// neither exchange. This block MUST run synchronously here, BEFORE the ConsumeMessage
	// goroutine below -- QueueBind/QueueUnbind and ConsumeMessage's internal basic.consume
	// share the same underlying AMQP channel for a given queue, and racing them makes the
	// broker close the channel with a 503 (production incident 2026-07-14, VOIP-1258).
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		// stay fully on the fanout subscriptions; skip binds and unbinds.
		log.Errorf("Could not declare the global topic exchange. Staying on fanout subscriptions. exchange: %s, err: %v", string(commonoutline.QueueNameEvent), errDeclare)
	} else {
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the topic pattern. Staying on fanout subscriptions. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// all-or-nothing: best-effort rollback of the partial topic binds; unbind NO
			// fanout exchange -- the service keeps running fully on fanout.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. queue: %s keeps a stray topic binding (partial double delivery). Manual intervention required. pattern: %s, err: %v", h.subscribeQueue, pattern, errUnbind)
				}
			}
		} else {
			// every pattern bound: unbind the old fanout exchanges. Unbind failure is
			// CRITICAL but not fatal -- double delivery beats event loss.
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", target, nil); errUnbind != nil {
					log.Errorf("CRITICAL: could not unbind the fanout exchange after the topic binds succeeded. queue: %s is now bound to BOTH exchanges (double delivery). Manual intervention required. target: %s, err: %v", h.subscribeQueue, target, errUnbind)
				}
			}
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

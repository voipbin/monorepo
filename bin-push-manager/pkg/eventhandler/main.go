package eventhandler

//go:generate mockgen -package eventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-push-manager/pkg/notificationhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

// The initial limits of an event. They are first values without a measurement behind them; change them here
// only, when the log of a cut or a slow downstream shows the need.
const (
	// defaultEventTimeout is the longest time one event may run, all of its sends together.
	defaultEventTimeout = 30 * time.Second

	// defaultParallelism is the number of sends of one event that run at the same time.
	defaultParallelism = 8

	// defaultMaxRecipients is the most chat_message recipients of one message. The mention recipients (at most
	// 50 by ExtractAgentMentions) are not cut.
	defaultMaxRecipients = 200
)

// EventHandler interface. It turns the events of the other services into notifications: it decides who is
// notified, applies the settings of the recipients and sends. Delivery is at most once: a failure is logged and
// not retried. The returned error is for the log of the caller only, the caller must not retry on it.
type EventHandler interface {
	// EventChatMessageCreated handles the talk-manager's chatmessage_created event: chat_mentioned for the
	// mentioned agents, chat_message for the other agents in the chat.
	EventChatMessageCreated(ctx context.Context, m *tkmessage.Message) error

	// EventConversationMessageCreated handles the conversation-manager's conversation_message_created event:
	// conversation_message for the agent who owns the conversation of an incoming message.
	EventConversationMessageCreated(ctx context.Context, m *cvmessage.Message) error

	// EventCaseNoteCreated handles the contact-manager's case_note_created event: case_note for the agent who
	// owns the case of a note that another agent wrote.
	EventCaseNoteCreated(ctx context.Context, n *cmcasenote.CaseNote) error

	// EventCaseAssigned handles the contact-manager's case_assigned event: case_assigned for the new owner.
	EventCaseAssigned(ctx context.Context, e *cmkase.CaseAssignedEvent) error
}

type eventHandler struct {
	reqHandler          requesthandler.RequestHandler
	settingsHandler     settingshandler.SettingsHandler
	notificationHandler notificationhandler.NotificationHandler

	// the limits are fields so the tests can change them
	eventTimeout  time.Duration
	parallelism   int
	maxRecipients int
}

var (
	metricsNamespace = "push_manager"

	promRecipientTruncated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "recipient_truncated_total",
			Help:      "Total number of events whose recipients were cut to the limit, by kind",
		},
		[]string{"kind"},
	)
)

func init() {
	prometheus.MustRegister(promRecipientTruncated)
}

// NewEventHandler return EventHandler interface
func NewEventHandler(
	reqHandler requesthandler.RequestHandler,
	settingsHandler settingshandler.SettingsHandler,
	notificationHandler notificationhandler.NotificationHandler,
) EventHandler {
	return &eventHandler{
		reqHandler:          reqHandler,
		settingsHandler:     settingsHandler,
		notificationHandler: notificationHandler,

		eventTimeout:  defaultEventTimeout,
		parallelism:   defaultParallelism,
		maxRecipients: defaultMaxRecipients,
	}
}

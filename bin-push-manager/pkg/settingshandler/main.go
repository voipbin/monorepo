package settingshandler

//go:generate mockgen -package settingshandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/models/settings"
	"monorepo/bin-push-manager/pkg/dbhandler"
)

// SettingsHandler interface
type SettingsHandler interface {
	// Get returns the notification settings of the agent: every kind with on or off and the muted chats.
	Get(ctx context.Context, customerID, agentID uuid.UUID) (*settings.Settings, error)

	// Update switches the kinds of the map (true is on, false is off) and returns the settings after the change.
	// Every kind is checked before anything is written: a kind that the agent can not switch is an error. The
	// writes are not atomic. When one fails, the earlier ones stay and the error is returned; sending the same
	// request again finishes the change, because every write can be repeated. An empty map changes nothing.
	Update(ctx context.Context, customerID, agentID uuid.UUID, kinds map[notification.Kind]bool) (*settings.Settings, error)

	// MuteAdd mutes the chat for the agent and returns the settings after the change. Repeating it succeeds.
	MuteAdd(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*settings.Settings, error)
	// MuteRemove unmutes the chat for the agent and returns the settings after the change. Repeating it succeeds.
	MuteRemove(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*settings.Settings, error)

	// FilterRecipients returns the agents that want the kind, in the order of the list: the agents that switched
	// the kind off are removed, and for chat_message the agents that muted the chat. Only chat_message looks at
	// chatID; the other kinds ignore it (a mention is not silenced by a muted chat).
	FilterRecipients(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, kind notification.Kind, chatID uuid.UUID) ([]uuid.UUID, error)

	EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error
	EventAgentDeleted(ctx context.Context, a *amagent.Agent) error
}

type settingsHandler struct {
	db dbhandler.DBHandler
}

// NewSettingsHandler return SettingsHandler interface
func NewSettingsHandler(db dbhandler.DBHandler) SettingsHandler {
	return &settingsHandler{
		db: db,
	}
}

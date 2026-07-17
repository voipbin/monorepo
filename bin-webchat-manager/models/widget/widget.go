package widget

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Widget defines a customer's webchat widget configuration.
// It is the PARENT resource in the direct-token model (1:1 with a
// bin-direct-manager hash, resource_type=webchat_widget).
type Widget struct {
	commonidentity.Identity

	// basic info
	Name   string `json:"name,omitempty" db:"name"`
	Status Status `json:"status,omitempty" db:"status"`

	// direct hash
	// DirectID references the bin-direct-manager record
	// (resource_type=webchat_widget, resource_id=Widget.ID). Nullable:
	// a Widget with DirectID = uuid.Nil is "provisioning incomplete".
	DirectID uuid.UUID `json:"direct_id,omitempty" db:"direct_id,uuid"`

	// Hash is the direct hash string itself, used by the embed script
	// (data-hash="<hash>") to authenticate anonymous visitors via
	// POST /auth/boot. Mirrors the DirectHash pattern used by AI/Team
	// (see bin-ai-manager/models/ai/main.go's DirectHash field) --
	// direct-manager is the source of truth for the hash value, this
	// is a denormalized copy so API responses don't need a second
	// round-trip to direct-manager on every widget read.
	Hash string `json:"direct_hash,omitempty" db:"direct_hash"`

	WelcomeMessage string `json:"welcome_message,omitempty" db:"welcome_message"`

	// SessionFlowID fires once per Session, anchored to session
	// creation/start (POST /webchat_sessions) -- NOT to the first
	// inbound message. Named for its cardinality (once per session),
	// not its trigger event. Trigger+execute ownership belongs to
	// bin-conversation-manager (see design doc
	// 2026-07-17-webchat-widget-session-message-flow-split-design.md
	// §3), not this service.
	SessionFlowID uuid.UUID `json:"session_flow_id,omitempty" db:"session_flow_id,uuid"`

	// MessageFlowID fires on EVERY inbound message, independently and
	// statelessly -- mirrors bin-conversation-manager's
	// Account.MessageFlowID/Number.MessageFlowID pattern exactly.
	// Opt-in: nil means no per-message trigger.
	MessageFlowID uuid.UUID `json:"message_flow_id,omitempty" db:"message_flow_id,uuid"`

	SessionIdleTimeout int `json:"session_idle_timeout,omitempty" db:"session_idle_timeout"` // seconds; default 1800

	// ThemeConfig: cosmetic appearance overrides. Nil = all defaults.
	ThemeConfig *ThemeConfig `json:"theme_config,omitempty" db:"theme_config,json"`

	TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

// Status type
type Status string

// list of statuses
const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// WidgetPosition controls where the floating bubble/panel renders on the
// customer's page.
type WidgetPosition string

// list of widget positions
const (
	WidgetPositionBottomRight WidgetPosition = "bottom_right" // default
	WidgetPositionBottomLeft  WidgetPosition = "bottom_left"
)

// ThemeConfig holds cosmetic, customer-editable widget appearance
// settings. All fields are optional; a nil ThemeConfig or empty field
// falls back to the platform default (blue bubble, no logo, bottom-right).
type ThemeConfig struct {
	PrimaryColor string         `json:"primary_color,omitempty"` // hex "#RRGGBB"
	LogoURL      string         `json:"logo_url,omitempty"`      // https URL only
	Position     WidgetPosition `json:"position,omitempty"`      // default: bottom_right
}

// DefaultSessionIdleTimeout is the default session idle timeout in seconds (30m).
const DefaultSessionIdleTimeout = 1800

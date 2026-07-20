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
//
// SECURITY: this struct is serialized verbatim to ANONYMOUS website
// visitors via POST /auth/boot's resource_data envelope's
// public_display_config key (see
// docs/plans/2026-07-20-auth-boot-public-display-config-design.md).
// Any new field added here must be independently vetted as safe for
// unauthenticated public exposure before merge -- it is NOT filtered
// by ConvertWebhookMessage() or any other export boundary; ThemeConfig
// is the identical *ThemeConfig pointer type on both Widget and
// WebhookMessage, so a field added here flows through both paths
// identically.
type ThemeConfig struct {
	PrimaryColor          string         `json:"primary_color,omitempty"`           // hex "#RRGGBB"
	SecondaryColor        string         `json:"secondary_color,omitempty"`         // hex "#RRGGBB"
	HeaderBackgroundColor string         `json:"header_background_color,omitempty"` // hex "#RRGGBB"
	HeaderTextColor       string         `json:"header_text_color,omitempty"`       // hex "#RRGGBB"
	LogoURL               string         `json:"logo_url,omitempty"`                // https URL only
	Position              WidgetPosition `json:"position,omitempty"`                // default: bottom_right
	ThemeMode             ThemeMode      `json:"theme_mode,omitempty"`              // default: light
	HeaderTitle           string         `json:"header_title,omitempty"`            // default: "Chat with us"
	HeaderSubtitle        string         `json:"header_subtitle,omitempty"`         // default: none

	// ConnectingIndicatorEnabled/ConnectingIndicatorText: shown in the
	// panel while the visitor's session is being created (between
	// widget open and POST /webchat_sessions completing). *bool (not
	// bool) because the default is true -- omitempty on a plain bool
	// could not distinguish "never set" from "explicitly false" on a
	// read-modify-write round-trip, silently flipping an existing
	// widget's default-on indicator off. nil/absent = enabled (true).
	ConnectingIndicatorEnabled *bool  `json:"connecting_indicator_enabled,omitempty"`
	ConnectingIndicatorText    string `json:"connecting_indicator_text,omitempty"` // default: "Connecting…", max 100 chars

	// TypingIndicatorEnabled: pure on/off gate over the existing
	// three-dot "waiting for response" animation. No text-label
	// variant is supported. nil/absent = enabled (true). Same *bool
	// reasoning as ConnectingIndicatorEnabled above.
	TypingIndicatorEnabled *bool `json:"typing_indicator_enabled,omitempty"`

	// BorderRadius/FontSize: bounded enum presets applied across the
	// bubble/panel/message-bubbles/input/send-button as a coordinated
	// set. See BorderRadius/FontSize type docs below.
	BorderRadius BorderRadius `json:"border_radius,omitempty"` // default: rounded
	FontSize     FontSize     `json:"font_size,omitempty"`     // default: default
}

// BorderRadius controls corner rounding across the widget's bubble,
// panel, message bubbles, input field, and send button as a
// coordinated set.
type BorderRadius string

// list of border radius presets
const (
	BorderRadiusSharp   BorderRadius = "sharp"
	BorderRadiusRounded BorderRadius = "rounded" // default
	BorderRadiusPill    BorderRadius = "pill"
)

// FontSize controls the base font-size scale applied to the widget's
// header text and message text.
type FontSize string

// list of font size presets
const (
	FontSizeCompact FontSize = "compact"
	FontSizeDefault FontSize = "default" // default
	FontSizeLarge   FontSize = "large"
)

// ThemeMode controls light/dark/auto rendering of the widget panel.
type ThemeMode string

// list of theme modes
const (
	ThemeModeLight ThemeMode = "light" // default
	ThemeModeDark  ThemeMode = "dark"
	ThemeModeAuto  ThemeMode = "auto" // follows prefers-color-scheme
)

// DefaultSessionIdleTimeout is the default session idle timeout in seconds (30m).
const DefaultSessionIdleTimeout = 1800

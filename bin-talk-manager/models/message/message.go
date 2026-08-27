package message

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type constants
const (
	TypeNormal = "normal"
	TypeSystem = "system"
)

type Type string

// Message represents a chat message
type Message struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID  `json:"chat_id" db:"chat_id,uuid"`
	ParentID *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`

	Type     Type     `json:"type" db:"type"`
	Text     string   `json:"text" db:"text"`
	Medias   []Media  `json:"medias" db:"medias,json"`
	Metadata Metadata `json:"metadata" db:"metadata,json"`

	// Timestamps
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404/1405 §2.3). It is the parent ChatID, not the message's own ID.
//
// The message id is stable and persisted, so this is a Category B override: the id first appears
// in the message's own `chatmessage_created` event, which means nobody can pre-bind to it, while
// every real consumption pattern follows a chat. Addressing by the parent makes
// `talk-manager.chatmessage.<chat-id>.#` deliver the whole conversation. Single-message retrieval
// stays available over RPC. The production webhook path already addresses chat children by their
// chat id (bin-webhook-manager/pkg/webhookhandler/routingkey.go), so this matches the existing
// contract rather than inventing one.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (m *Message) EventSubscriptionID() string {
	return m.ChatID.String()
}

// WebhookMessage is the webhook payload for message events
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID  `json:"chat_id"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`

	Type     Type     `json:"type"`
	Text     string   `json:"text"`
	Medias   []Media  `json:"medias"`
	Metadata Metadata `json:"metadata"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// MediaType defines the type of media content
type MediaType string

// Media type constants
const (
	MediaTypeAddress MediaType = "address" // the media contains address infos
	MediaTypeAgent   MediaType = "agent"   // the media contains agent infos
	MediaTypeFile    MediaType = "file"    // the media contains file info
	MediaTypeLink    MediaType = "link"    // the media contains link info
)

// Media represents a media attachment
type Media struct {
	Type MediaType `json:"type,omitempty"`

	Address commonaddress.Address `json:"address,omitempty"`  // valid only if the Type is address type
	AgentID uuid.UUID             `json:"agent_id,omitempty"` // valid only if the type is agent type
	FileID  uuid.UUID             `json:"file_id,omitempty"`  // valid only if the Type is file
	LinkURL string                `json:"link_url,omitempty"` // valid only if the Type is link type
}

// ConvertWebhookMessage converts Message to WebhookMessage
// Now that Message uses proper types, this is a simple field copy
func (m *Message) ConvertWebhookMessage() (*WebhookMessage, error) {
	return &WebhookMessage{
		Identity: m.Identity,
		Owner:    m.Owner,
		ChatID:   m.ChatID,
		ParentID: m.ParentID,
		Type:     m.Type,
		Text:     m.Text,
		Medias:   m.Medias,
		Metadata: m.Metadata,
		TMCreate: m.TMCreate,
		TMUpdate: m.TMUpdate,
		TMDelete: m.TMDelete,
	}, nil
}

// CreateWebhookEvent generates WebhookEvent JSON from Message
func (m *Message) CreateWebhookEvent() ([]byte, error) {
	e, err := m.ConvertWebhookMessage()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return data, nil
}


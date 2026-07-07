package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"

	"github.com/gofrs/uuid"
)

// V1DataAccountsPost is
// v1 data type request struct for
// /v1/accounts POST
type V1DataConversationsPost struct {
	CustomerID uuid.UUID             `json:"customer_id,omitempty"`
	Name       string                `json:"name,omitempty"`
	Detail     string                `json:"detail,omitempty"`
	Type       conversation.Type     `json:"type,omitempty"`
	DialogID   string                `json:"dialog_id,omitempty"`
	Self       commonaddress.Address `json:"self,omitempty"`
	Peer       commonaddress.Address `json:"peer,omitempty"`
}

// V1DataConversationsIDMessagesPost is
// v1 data type request struct for
// /v1/conversations/<conversation-id>/messages POST
type V1DataConversationsIDMessagesPost struct {
	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`
}

// V1DataConversationsSelfAndPeerGet is
// v1 data type request struct for
// /v1/conversations/self_and_peer GET
//
// Used by bin-contact-manager's get-only proactive Case-linking lookup
// (contact-case-management design §4.4). Sent as a JSON body on a GET
// request, matching this service's existing convention for
// /v1/conversations GET (filters are JSON-marshaled and sent as the
// request body, not query params).
type V1DataConversationsSelfAndPeerGet struct {
	Self commonaddress.Address `json:"self,omitempty"`
	Peer commonaddress.Address `json:"peer,omitempty"`
}

// V1DataConversationsGetOrCreateBySelfAndPeerPost is
// v1 data type request struct for
// /v1/conversations/get_or_create_by_self_and_peer POST
//
// Used by bin-contact-manager's agent-send Case-linked messaging path
// (contact-case-management design §4.5, round-12 correction). Distinct
// from the get-only V1DataConversationsSelfAndPeerGet above: this one is
// correct to create on a miss, because a real message is genuinely
// about to be sent through the resulting Conversation.
type V1DataConversationsGetOrCreateBySelfAndPeerPost struct {
	CustomerID       uuid.UUID             `json:"customer_id,omitempty"`
	ConversationType conversation.Type     `json:"type,omitempty"`
	DialogID         string                `json:"dialog_id,omitempty"`
	Self             commonaddress.Address `json:"self,omitempty"`
	Peer             commonaddress.Address `json:"peer,omitempty"`
}

// V1DataConversationsIDMetadataPut is
// v1 data type request struct for
// /v1/conversations/<conversation-id>/metadata PUT
//
// Whole-struct-replace update, used by bin-contact-manager's
// Case-linking write paths (contact-case-management design
// §4.3/§4.4/§4.5). Deliberately a dedicated route, NOT reachable via
// the general /v1/conversations/<conversation-id> PUT field allowlist.
type V1DataConversationsIDMetadataPut struct {
	Metadata conversation.Metadata `json:"metadata,omitempty"`
}

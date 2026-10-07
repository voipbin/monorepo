package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
)

// V1DataBuilderChatPost is the data type request struct for
// /v1/ai_builder/chat POST.
//
// CustomerID is not taken from the browser. api-manager fills it from the
// authenticated identity, so the daily counter is charged to the customer who
// really made the call. A body without it is refused.
type V1DataBuilderChatPost struct {
	CustomerID   uuid.UUID         `json:"customer_id"`
	Messages     []builder.Message `json:"messages"`
	CurrentDraft *builder.Draft    `json:"current_draft,omitempty"`
}

// V1DataFlowBuilderChatPost is the data type request struct for
// /v1/flow_builder/chat POST. CustomerID is filled by api-manager from the
// authenticated identity, as for V1DataBuilderChatPost.
type V1DataFlowBuilderChatPost struct {
	CustomerID           uuid.UUID             `json:"customer_id"`
	Messages             []flowbuilder.Message `json:"messages"`
	CurrentDraft         *flowbuilder.Draft    `json:"current_draft,omitempty"`
	SupportedActionTypes []string              `json:"supported_action_types"`
}

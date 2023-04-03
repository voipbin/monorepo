package request

import (
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// ParamConversationsGET is request param define for
// GET /v1.0/conversations
type ParamConversationsGET struct {
	Pagination
}

// ParamConversationsIDMessagesGET is request param define for
// GET /v1.0/conversations/<conversation-id>/messages
type ParamConversationsIDMessagesGET struct {
	Pagination
}

// ParamConversationsIDMessagesPOST is request param define for
// POST /v1.0/conversations/<conversation-id>/messages
type ParamConversationsIDMessagesPOST struct {
	Text string
}

// ParamConversationsSetupPOST is request param define for
// POST /v1.0/conversations/setup
type ParamConversationsSetupPOST struct {
	ReferenceType cvconversation.ReferenceType `json:"reference_type"`
}

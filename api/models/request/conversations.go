package request

import (
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// ParamConversationsGET is request param define for GET /conversations
type ParamConversationsGET struct {
	Pagination
}

// ParamConversationsIDMessagesGET is request param define for GET /conversations/<conversation-id>/messages
type ParamConversationsIDMessagesGET struct {
	Pagination
}

// ParamConversationsIDMessagesPOST is request param define for POST /conversations/<conversation-id>/messages
type ParamConversationsIDMessagesPOST struct {
	Text string
}

// ParamConversationsSetupPOST is request param define for POST /conversations/setup
type ParamConversationsSetupPOST struct {
	ReferenceType cvconversation.ReferenceType
}

package request

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

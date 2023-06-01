package request

// ParamConversationsGET is request param define for
// GET /v1.0/conversations
type ParamConversationsGET struct {
	Pagination
}

// BodyConversationsIDPUT is request body define for
// PUT /v1.0/conversations/<conversation-id>
type BodyConversationsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// ParamConversationsIDMessagesGET is request param define for
// GET /v1.0/conversations/<conversation-id>/messages
type ParamConversationsIDMessagesGET struct {
	Pagination
}

// BodyConversationsIDMessagesPOST is request body define for
// POST /v1.0/conversations/<conversation-id>/messages
type BodyConversationsIDMessagesPOST struct {
	Text string
}

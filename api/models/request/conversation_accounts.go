package request

import (
	cvaccount "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
)

// ParamConversationAccountsGET is request param define for
// GET /v1.0/conversation_accounts
type ParamConversationAccountsGET struct {
	Pagination
}

// BodyConversationAccountsPOST is rquest body define for
// POST /v1.0/conversation_accounts
type BodyConversationAccountsPOST struct {
	Type   cvaccount.Type `json:"type"`
	Name   string         `json:"name"`
	Detail string         `json:"detail"`
	Secret string         `json:"secret"`
	Token  string         `json:"token"`
}

// BodyConversationAccountsIDPUT is rquest body define for
// POST /v1.0/conversation_accounts/{conversation-account-id}
type BodyConversationAccountsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Secret string `json:"secret"`
	Token  string `json:"token"`
}

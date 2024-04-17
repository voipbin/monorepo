package response

import (
	cvaccount "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
)

// BodyConversationAccountsGET is rquest body define for
// GET /v1.0/conversation_accounts
type BodyConversationAccountsGET struct {
	Result []*cvaccount.WebhookMessage `json:"result"`
	Pagination
}

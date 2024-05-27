package apiv1

import (
	"github.com/gin-gonic/gin"

	"monorepo/bin-api-manager/api/v1.0/activeflows"
	"monorepo/bin-api-manager/api/v1.0/agents"
	availablenumbers "monorepo/bin-api-manager/api/v1.0/available_numbers"
	"monorepo/bin-api-manager/api/v1.0/billingaccounts"
	"monorepo/bin-api-manager/api/v1.0/billings"
	"monorepo/bin-api-manager/api/v1.0/calls"
	"monorepo/bin-api-manager/api/v1.0/campaigncalls"
	"monorepo/bin-api-manager/api/v1.0/campaigns"
	"monorepo/bin-api-manager/api/v1.0/chatbotcalls"
	"monorepo/bin-api-manager/api/v1.0/chatbots"
	"monorepo/bin-api-manager/api/v1.0/chatmessages"
	"monorepo/bin-api-manager/api/v1.0/chatroommessages"
	"monorepo/bin-api-manager/api/v1.0/chatrooms"
	"monorepo/bin-api-manager/api/v1.0/chats"
	"monorepo/bin-api-manager/api/v1.0/conferencecalls"
	"monorepo/bin-api-manager/api/v1.0/conferences"
	conversationaccounts "monorepo/bin-api-manager/api/v1.0/conversation_accounts"
	"monorepo/bin-api-manager/api/v1.0/conversations"
	"monorepo/bin-api-manager/api/v1.0/customers"
	"monorepo/bin-api-manager/api/v1.0/extensions"
	"monorepo/bin-api-manager/api/v1.0/files"
	"monorepo/bin-api-manager/api/v1.0/flows"
	"monorepo/bin-api-manager/api/v1.0/groupcalls"
	"monorepo/bin-api-manager/api/v1.0/messages"
	"monorepo/bin-api-manager/api/v1.0/numbers"
	"monorepo/bin-api-manager/api/v1.0/outdials"
	"monorepo/bin-api-manager/api/v1.0/outplans"
	"monorepo/bin-api-manager/api/v1.0/providers"
	"monorepo/bin-api-manager/api/v1.0/queuecalls"
	"monorepo/bin-api-manager/api/v1.0/queues"
	"monorepo/bin-api-manager/api/v1.0/recordingfiles"
	"monorepo/bin-api-manager/api/v1.0/recordings"
	"monorepo/bin-api-manager/api/v1.0/routes"
	storage_account "monorepo/bin-api-manager/api/v1.0/storage_account"
	storage_accounts "monorepo/bin-api-manager/api/v1.0/storage_accounts"
	"monorepo/bin-api-manager/api/v1.0/tags"
	"monorepo/bin-api-manager/api/v1.0/transcribes"
	"monorepo/bin-api-manager/api/v1.0/transcripts"
	"monorepo/bin-api-manager/api/v1.0/transfers"
	"monorepo/bin-api-manager/api/v1.0/trunks"
	"monorepo/bin-api-manager/api/v1.0/ws"
	"monorepo/bin-api-manager/lib/middleware"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0", middleware.Authorized)

	// v1.0
	activeflows.ApplyRoutes(v1)
	agents.ApplyRoutes(v1)
	availablenumbers.ApplyRoutes(v1)
	billingaccounts.ApplyRoutes(v1)
	billings.ApplyRoutes(v1)
	calls.ApplyRoutes(v1)
	campaigns.ApplyRoutes(v1)
	campaigncalls.ApplyRoutes(v1)
	chatbots.ApplyRoutes(v1)
	chatbotcalls.ApplyRoutes(v1)
	chats.ApplyRoutes(v1)
	chatmessages.ApplyRoutes(v1)
	chatrooms.ApplyRoutes(v1)
	chatroommessages.ApplyRoutes(v1)
	conferences.ApplyRoutes(v1)
	conferencecalls.ApplyRoutes(v1)
	conversations.ApplyRoutes(v1)
	conversationaccounts.ApplyRoutes(v1)
	customers.ApplyRoutes(v1)
	extensions.ApplyRoutes(v1)
	files.ApplyRoutes(v1)
	flows.ApplyRoutes(v1)
	groupcalls.ApplyRoutes(v1)
	messages.ApplyRoutes(v1)
	numbers.ApplyRoutes(v1)
	outdials.ApplyRoutes(v1)
	outplans.ApplyRoutes(v1)
	providers.ApplyRoutes(v1)
	queues.ApplyRoutes(v1)
	queuecalls.ApplyRoutes(v1)
	recordings.ApplyRoutes(v1)
	recordingfiles.ApplyRoutes(v1)
	routes.ApplyRoutes(v1)
	storage_account.ApplyRoutes(v1)
	storage_accounts.ApplyRoutes(v1)
	tags.ApplyRoutes(v1)
	transcribes.ApplyRoutes(v1)
	transcripts.ApplyRoutes(v1)
	transfers.ApplyRoutes(v1)
	trunks.ApplyRoutes(v1)
	ws.ApplyRoutes(v1)
}

package apiv1

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/activeflows"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/agents"
	availablenumbers "gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/available_numbers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/billingaccounts"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/billings"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/calls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/campaigncalls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/campaigns"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chatbotcalls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chatbots"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chatmessages"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chatroommessages"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chatrooms"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/chats"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conferencecalls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conferences"
	conversationaccounts "gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conversation_accounts"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conversations"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/customers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/extensions"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/flows"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/groupcalls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/messages"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/numbers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/outdials"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/outplans"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/providers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/queuecalls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/queues"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordingfiles"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordings"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/routes"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/tags"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/transcribes"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/transcripts"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/transfers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/trunks"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/ws"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
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
	tags.ApplyRoutes(v1)
	transcribes.ApplyRoutes(v1)
	transcripts.ApplyRoutes(v1)
	transfers.ApplyRoutes(v1)
	trunks.ApplyRoutes(v1)
	ws.ApplyRoutes(v1)
}

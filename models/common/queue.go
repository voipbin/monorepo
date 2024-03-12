package common

// list of queue names
// most of service has 3 queue names.
// bin-manager.<service name>.event: Event publish queue.
// bin-manager.<service name>.request: Request receive queue.
// bin-manager.<service name>.subscribe: Subscribe event queue.
const (
	// common use
	QueueDelay = "bin-manager.delay" // queue name for delayed requests

	// asterisk
	QueueAsteriskEventAll = "asterisk.all.event"

	// agent-manager
	QueueAgentEvent     = "bin-manager.agent-manager.event"
	QueueAgentRequest   = "bin-manager.agent-manager.request"
	QueueAgentSubscribe = "bin-manager.agent-manager.subscribe"

	// api-manager
	QueueAPIEvent     = "bin-manager.api-manager.event"
	QueueAPIRequest   = "bin-manager.api-manager.request"
	QueueAPISubscribe = "bin-manager.api-manager.subscribe"

	// billing-manager
	QueueBillingEvent     = "bin-manager.billing-manager.event"
	QueueBillingRequest   = "bin-manager.billing-manager.request"
	QueueBillingSubscribe = "bin-manager.billing-manager.subscribe"

	// call-manager
	QueueCallEvent     = "bin-manager.call-manager.event"
	QueueCallRequest   = "bin-manager.call-manager.request"
	QueueCallSubscribe = "bin-manager.call-manager.subscribe"

	// campaign-manager
	QueueCampaignEvent     = "bin-manager.campaign-manager.event"
	QueueCampaignRequest   = "bin-manager.campaign-manager.request"
	QueueCampaignSubscribe = "bin-manager.campaign-manager.subscribe"

	// chat-manager
	QueueChatEvent     = "bin-manager.chat-manager.event"
	QueueChatRequest   = "bin-manager.chat-manager.request"
	QueueChatSubscribe = "bin-manager.chat-manager.subscribe"

	// chatbot-manager
	QueueChatbotEvent     = "bin-manager.chatbot-manager.event"
	QueueChatbotRequest   = "bin-manager.chatbot-manager.request"
	QueueChatbotSubscribe = "bin-manager.chatbot-manager.subscribe"

	// conference-manager
	QueueConferenceEvent     = "bin-manager.conference-manager.event"
	QueueConferenceRequest   = "bin-manager.conference-manager.request"
	QueueConferenceSubscribe = "bin-manager.conference-manager.subscribe"

	// conversation-manager
	QueueConversationEvent     = "bin-manager.conversation-manager.event"
	QueueConversationRequest   = "bin-manager.conversation-manager.request"
	QueueConversationSubscribe = "bin-manager.conversation-manager.subscribe"

	// customer-manager
	QueueCustomerEvent     = "bin-manager.customer-manager.event"
	QueueCustomerRequest   = "bin-manager.customer-manager.request"
	QueueCustomerSubscribe = "bin-manager.customer-manager.subscribe"

	// flow-manager
	QueueFlowEvent     = "bin-manager.flow-manager.event"
	QueueFlowRequest   = "bin-manager.flow-manager.request"
	QueueFlowSubscribe = "bin-manager.flow-manager.subscribe"

	// message-manager
	QueueMessageEvent     = "bin-manager.message-manager.event"
	QueueMessageRequest   = "bin-manager.message-manager.request"
	QueueMessageSubscribe = "bin-manager.message-manager.subscribe"

	// number-manager
	QueueNumberEvent     = "bin-manager.number-manager.event"
	QueueNumberRequest   = "bin-manager.number-manager.request"
	QueueNumberSubscribe = "bin-manager.number-manager.subscribe"

	// outdial-manager
	QueueOutdialEvent     = "bin-manager.outdial-manager.event"
	QueueOutdialRequest   = "bin-manager.outdial-manager.request"
	QueueOutdialSubscribe = "bin-manager.outdial-manager.subscribe"

	// queue-manager
	QueueQueueEvent     = "bin-manager.queue-manager.event"
	QueueQueueRequest   = "bin-manager.queue-manager.request"
	QueueQueueSubscribe = "bin-manager.queue-manager.subscribe"

	// registrar-manager
	QueueRegistrarEvent     = "bin-manager.registrar-manager.event"
	QueueRegistrarRequest   = "bin-manager.registrar-manager.request"
	QueueRegistrarSubscribe = "bin-manager.registrar-manager.subscribe"

	// route-manager
	QueueRouteEvent     = "bin-manager.route-manager.event"
	QueueRouteRequest   = "bin-manager.route-manager.request"
	QueueRouteSubscribe = "bin-manager.route-manager.subscribe"

	// storage-manager
	QueueStorageEvent     = "bin-manager.storage-manager.event"
	QueueStorageRequest   = "bin-manager.storage-manager.request"
	QueueStorageSubscribe = "bin-manager.storage-manager.subscribe"

	// tag-manager
	QueueTagEvent     = "bin-manager.tag-manager.event"
	QueueTagRequest   = "bin-manager.tag-manager.request"
	QueueTagSubscribe = "bin-manager.tag-manager.subscribe"

	// transcribe-manager
	QueueTranscribeEvent     = "bin-manager.transcribe-manager.event"
	QueueTranscribeRequest   = "bin-manager.transcribe-manager.request"
	QueueTranscribeSubscribe = "bin-manager.transcribe-manager.subscribe"

	// transfer-manager
	QueueTransferEvent     = "bin-manager.transfer-manager.event"
	QueueTransferRequest   = "bin-manager.transfer-manager.request"
	QueueTransferSubscribe = "bin-manager.transfer-manager.subscribe"

	// tts-manager
	QueueTTSEvent     = "bin-manager.tts-manager.event"
	QueueTTSRequest   = "bin-manager.tts-manager.request"
	QueueTTSSubscribe = "bin-manager.tts-manager.subscribe"

	// user-manager
	QueueUserEvent     = "bin-manager.user-manager.event"
	QueueUserRequest   = "bin-manager.user-manager.request"
	QueueUserSubscribe = "bin-manager.user-manager.subscribe"

	// webhook-manager
	QueueWebhookEvent     = "bin-manager.webhook-manager.event"
	QueueWebhookRequest   = "bin-manager.webhook-manager.request"
	QueueWebhookSubscribe = "bin-manager.webhook-manager.subscribe"
)

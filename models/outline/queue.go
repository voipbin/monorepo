package outline

// Queue type define
type Queue string

// list of queue names
// most of service has 3 queue names.
// bin-manager.<service name>.event: Event publish queue.
// bin-manager.<service name>.request: Request receive queue.
// bin-manager.<service name>.subscribe: Subscribe event queue.
const (
	// common use
	QueueDelay Queue = "bin-manager.delay" // queue name for delayed requests

	// asterisk
	QueueAsteriskEventAll Queue = "asterisk.all.event"

	// agent-manager
	QueueAgentEvent     Queue = "bin-manager.agent-manager.event"
	QueueAgentRequest   Queue = "bin-manager.agent-manager.request"
	QueueAgentSubscribe Queue = "bin-manager.agent-manager.subscribe"

	// api-manager
	QueueAPIEvent     Queue = "bin-manager.api-manager.event"
	QueueAPIRequest   Queue = "bin-manager.api-manager.request"
	QueueAPISubscribe Queue = "bin-manager.api-manager.subscribe"

	// billing-manager
	QueueBillingEvent     Queue = "bin-manager.billing-manager.event"
	QueueBillingRequest   Queue = "bin-manager.billing-manager.request"
	QueueBillingSubscribe Queue = "bin-manager.billing-manager.subscribe"

	// call-manager
	QueueCallEvent     Queue = "bin-manager.call-manager.event"
	QueueCallRequest   Queue = "bin-manager.call-manager.request"
	QueueCallSubscribe Queue = "bin-manager.call-manager.subscribe"

	// campaign-manager
	QueueCampaignEvent     Queue = "bin-manager.campaign-manager.event"
	QueueCampaignRequest   Queue = "bin-manager.campaign-manager.request"
	QueueCampaignSubscribe Queue = "bin-manager.campaign-manager.subscribe"

	// chat-manager
	QueueChatEvent     Queue = "bin-manager.chat-manager.event"
	QueueChatRequest   Queue = "bin-manager.chat-manager.request"
	QueueChatSubscribe Queue = "bin-manager.chat-manager.subscribe"

	// chatbot-manager
	QueueChatbotEvent     Queue = "bin-manager.chatbot-manager.event"
	QueueChatbotRequest   Queue = "bin-manager.chatbot-manager.request"
	QueueChatbotSubscribe Queue = "bin-manager.chatbot-manager.subscribe"

	// conference-manager
	QueueConferenceEvent     Queue = "bin-manager.conference-manager.event"
	QueueConferenceRequest   Queue = "bin-manager.conference-manager.request"
	QueueConferenceSubscribe Queue = "bin-manager.conference-manager.subscribe"

	// conversation-manager
	QueueConversationEvent     Queue = "bin-manager.conversation-manager.event"
	QueueConversationRequest   Queue = "bin-manager.conversation-manager.request"
	QueueConversationSubscribe Queue = "bin-manager.conversation-manager.subscribe"

	// customer-manager
	QueueCustomerEvent     Queue = "bin-manager.customer-manager.event"
	QueueCustomerRequest   Queue = "bin-manager.customer-manager.request"
	QueueCustomerSubscribe Queue = "bin-manager.customer-manager.subscribe"

	// flow-manager
	QueueFlowEvent     Queue = "bin-manager.flow-manager.event"
	QueueFlowRequest   Queue = "bin-manager.flow-manager.request"
	QueueFlowSubscribe Queue = "bin-manager.flow-manager.subscribe"

	// message-manager
	QueueMessageEvent     Queue = "bin-manager.message-manager.event"
	QueueMessageRequest   Queue = "bin-manager.message-manager.request"
	QueueMessageSubscribe Queue = "bin-manager.message-manager.subscribe"

	// number-manager
	QueueNumberEvent     Queue = "bin-manager.number-manager.event"
	QueueNumberRequest   Queue = "bin-manager.number-manager.request"
	QueueNumberSubscribe Queue = "bin-manager.number-manager.subscribe"

	// outdial-manager
	QueueOutdialEvent     Queue = "bin-manager.outdial-manager.event"
	QueueOutdialRequest   Queue = "bin-manager.outdial-manager.request"
	QueueOutdialSubscribe Queue = "bin-manager.outdial-manager.subscribe"

	// queue-manager
	QueueQueueEvent     Queue = "bin-manager.queue-manager.event"
	QueueQueueRequest   Queue = "bin-manager.queue-manager.request"
	QueueQueueSubscribe Queue = "bin-manager.queue-manager.subscribe"

	// registrar-manager
	QueueRegistrarEvent     Queue = "bin-manager.registrar-manager.event"
	QueueRegistrarRequest   Queue = "bin-manager.registrar-manager.request"
	QueueRegistrarSubscribe Queue = "bin-manager.registrar-manager.subscribe"

	// route-manager
	QueueRouteEvent     Queue = "bin-manager.route-manager.event"
	QueueRouteRequest   Queue = "bin-manager.route-manager.request"
	QueueRouteSubscribe Queue = "bin-manager.route-manager.subscribe"

	// storage-manager
	QueueStorageEvent     Queue = "bin-manager.storage-manager.event"
	QueueStorageRequest   Queue = "bin-manager.storage-manager.request"
	QueueStorageSubscribe Queue = "bin-manager.storage-manager.subscribe"

	// tag-manager
	QueueTagEvent     Queue = "bin-manager.tag-manager.event"
	QueueTagRequest   Queue = "bin-manager.tag-manager.request"
	QueueTagSubscribe Queue = "bin-manager.tag-manager.subscribe"

	// transcribe-manager
	QueueTranscribeEvent     Queue = "bin-manager.transcribe-manager.event"
	QueueTranscribeRequest   Queue = "bin-manager.transcribe-manager.request"
	QueueTranscribeSubscribe Queue = "bin-manager.transcribe-manager.subscribe"

	// transfer-manager
	QueueTransferEvent     Queue = "bin-manager.transfer-manager.event"
	QueueTransferRequest   Queue = "bin-manager.transfer-manager.request"
	QueueTransferSubscribe Queue = "bin-manager.transfer-manager.subscribe"

	// tts-manager
	QueueTTSEvent     Queue = "bin-manager.tts-manager.event"
	QueueTTSRequest   Queue = "bin-manager.tts-manager.request"
	QueueTTSSubscribe Queue = "bin-manager.tts-manager.subscribe"

	// user-manager
	QueueUserEvent     Queue = "bin-manager.user-manager.event"
	QueueUserRequest   Queue = "bin-manager.user-manager.request"
	QueueUserSubscribe Queue = "bin-manager.user-manager.subscribe"

	// webhook-manager
	QueueWebhookEvent     Queue = "bin-manager.webhook-manager.event"
	QueueWebhookRequest   Queue = "bin-manager.webhook-manager.request"
	QueueWebhookSubscribe Queue = "bin-manager.webhook-manager.subscribe"
)

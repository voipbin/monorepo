package outline

// QueueName type define
type QueueName string

// list of queue names
// most of service has 3 queue names.
// bin-manager.<service name>.event: Event publish queue.
// bin-manager.<service name>.request: Request receive queue.
// bin-manager.<service name>.subscribe: Subscribe event queue.
const (
	// common use
	QueueNameDelay QueueName = "bin-manager.delay" // queue name for delayed requests

	// ai-manager
	QueueNameAIEvent     QueueName = "bin-manager.ai-manager.event"
	QueueNameAIRequest   QueueName = "bin-manager.ai-manager.request"
	QueueNameAISubscribe QueueName = "bin-manager.ai-manager.subscribe"

	// asterisk
	QueueNameAsteriskEventAll QueueName = "asterisk.all.event"

	// agent-manager
	QueueNameAgentEvent     QueueName = "bin-manager.agent-manager.event"
	QueueNameAgentRequest   QueueName = "bin-manager.agent-manager.request"
	QueueNameAgentSubscribe QueueName = "bin-manager.agent-manager.subscribe"

	// api-manager
	QueueNameAPIEvent     QueueName = "bin-manager.api-manager.event"
	QueueNameAPIRequest   QueueName = "bin-manager.api-manager.request"
	QueueNameAPISubscribe QueueName = "bin-manager.api-manager.subscribe"

	// billing-manager
	QueueNameBillingEvent     QueueName = "bin-manager.billing-manager.event"
	QueueNameBillingRequest   QueueName = "bin-manager.billing-manager.request"
	QueueNameBillingSubscribe QueueName = "bin-manager.billing-manager.subscribe"

	// call-manager
	QueueNameCallEvent     QueueName = "bin-manager.call-manager.event"
	QueueNameCallRequest   QueueName = "bin-manager.call-manager.request"
	QueueNameCallSubscribe QueueName = "bin-manager.call-manager.subscribe"

	// campaign-manager
	QueueNameCampaignEvent     QueueName = "bin-manager.campaign-manager.event"
	QueueNameCampaignRequest   QueueName = "bin-manager.campaign-manager.request"
	QueueNameCampaignSubscribe QueueName = "bin-manager.campaign-manager.subscribe"

	// conference-manager
	QueueNameConferenceEvent     QueueName = "bin-manager.conference-manager.event"
	QueueNameConferenceRequest   QueueName = "bin-manager.conference-manager.request"
	QueueNameConferenceSubscribe QueueName = "bin-manager.conference-manager.subscribe"

	// contact-manager
	QueueNameContactEvent     QueueName = "bin-manager.contact-manager.event"
	QueueNameContactRequest   QueueName = "bin-manager.contact-manager.request"
	QueueNameContactSubscribe QueueName = "bin-manager.contact-manager.subscribe"

	// conversation-manager
	QueueNameConversationEvent     QueueName = "bin-manager.conversation-manager.event"
	QueueNameConversationRequest   QueueName = "bin-manager.conversation-manager.request"
	QueueNameConversationSubscribe QueueName = "bin-manager.conversation-manager.subscribe"

	// customer-manager
	QueueNameCustomerEvent     QueueName = "bin-manager.customer-manager.event"
	QueueNameCustomerRequest   QueueName = "bin-manager.customer-manager.request"
	QueueNameCustomerSubscribe QueueName = "bin-manager.customer-manager.subscribe"

	// email-manager
	QueueNameEmailEvent     QueueName = "bin-manager.email-manager.event"
	QueueNameEmailRequest   QueueName = "bin-manager.email-manager.request"
	QueueNameEmailSubscribe QueueName = "bin-manager.email-manager.subscribe"

	// flow-manager
	QueueNameFlowEvent     QueueName = "bin-manager.flow-manager.event"
	QueueNameFlowRequest   QueueName = "bin-manager.flow-manager.request"
	QueueNameFlowSubscribe QueueName = "bin-manager.flow-manager.subscribe"

	// message-manager
	QueueNameMessageEvent     QueueName = "bin-manager.message-manager.event"
	QueueNameMessageRequest   QueueName = "bin-manager.message-manager.request"
	QueueNameMessageSubscribe QueueName = "bin-manager.message-manager.subscribe"

	// number-manager
	QueueNameNumberEvent     QueueName = "bin-manager.number-manager.event"
	QueueNameNumberRequest   QueueName = "bin-manager.number-manager.request"
	QueueNameNumberSubscribe QueueName = "bin-manager.number-manager.subscribe"

	// outdial-manager
	QueueNameOutdialEvent     QueueName = "bin-manager.outdial-manager.event"
	QueueNameOutdialRequest   QueueName = "bin-manager.outdial-manager.request"
	QueueNameOutdialSubscribe QueueName = "bin-manager.outdial-manager.subscribe"

	// pipecat-manager
	QueueNamePipecatEvent     QueueName = "bin-manager.pipecat-manager.event"
	QueueNamePipecatRequest   QueueName = "bin-manager.pipecat-manager.request"
	QueueNamePipecatSubscribe QueueName = "bin-manager.pipecat-manager.subscribe"

	// queue-manager
	QueueNameQueueEvent     QueueName = "bin-manager.queue-manager.event"
	QueueNameQueueRequest   QueueName = "bin-manager.queue-manager.request"
	QueueNameQueueSubscribe QueueName = "bin-manager.queue-manager.subscribe"

	// rag-manager
	QueueNameRagEvent     QueueName = "bin-manager.rag-manager.event"
	QueueNameRagRequest   QueueName = "bin-manager.rag-manager.request"
	QueueNameRagSubscribe QueueName = "bin-manager.rag-manager.subscribe"

	// registrar-manager
	QueueNameRegistrarEvent     QueueName = "bin-manager.registrar-manager.event"
	QueueNameRegistrarRequest   QueueName = "bin-manager.registrar-manager.request"
	QueueNameRegistrarSubscribe QueueName = "bin-manager.registrar-manager.subscribe"

	// route-manager
	QueueNameRouteEvent     QueueName = "bin-manager.route-manager.event"
	QueueNameRouteRequest   QueueName = "bin-manager.route-manager.request"
	QueueNameRouteSubscribe QueueName = "bin-manager.route-manager.subscribe"

	// sentinel-manager
	QueueNameSentinelEvent     QueueName = "bin-manager.sentinel-manager.event"
	QueueNameSentinelRequest   QueueName = "bin-manager.sentinel-manager.request"
	QueueNameSentinelSubscribe QueueName = "bin-manager.sentinel-manager.subscribe"

	// storage-manager
	QueueNameStorageEvent     QueueName = "bin-manager.storage-manager.event"
	QueueNameStorageRequest   QueueName = "bin-manager.storage-manager.request"
	QueueNameStorageSubscribe QueueName = "bin-manager.storage-manager.subscribe"

	// tag-manager
	QueueNameTagEvent     QueueName = "bin-manager.tag-manager.event"
	QueueNameTagRequest   QueueName = "bin-manager.tag-manager.request"
	QueueNameTagSubscribe QueueName = "bin-manager.tag-manager.subscribe"

	// talk-manager
	QueueNameTalkEvent     QueueName = "bin-manager.talk-manager.event"
	QueueNameTalkRequest   QueueName = "bin-manager.talk-manager.request"
	QueueNameTalkSubscribe QueueName = "bin-manager.talk-manager.subscribe"

	// timeline-manager
	QueueNameTimelineEvent     QueueName = "bin-manager.timeline-manager.event"
	QueueNameTimelineRequest   QueueName = "bin-manager.timeline-manager.request"
	QueueNameTimelineSubscribe QueueName = "bin-manager.timeline-manager.subscribe"

	// transcribe-manager
	QueueNameTranscribeEvent     QueueName = "bin-manager.transcribe-manager.event"
	QueueNameTranscribeRequest   QueueName = "bin-manager.transcribe-manager.request"
	QueueNameTranscribeSubscribe QueueName = "bin-manager.transcribe-manager.subscribe"

	// transfer-manager
	QueueNameTransferEvent     QueueName = "bin-manager.transfer-manager.event"
	QueueNameTransferRequest   QueueName = "bin-manager.transfer-manager.request"
	QueueNameTransferSubscribe QueueName = "bin-manager.transfer-manager.subscribe"

	// tts-manager
	QueueNameTTSEvent     QueueName = "bin-manager.tts-manager.event"
	QueueNameTTSRequest   QueueName = "bin-manager.tts-manager.request"
	QueueNameTTSSubscribe QueueName = "bin-manager.tts-manager.subscribe"

	// user-manager
	QueueNameUserEvent     QueueName = "bin-manager.user-manager.event"
	QueueNameUserRequest   QueueName = "bin-manager.user-manager.request"
	QueueNameUserSubscribe QueueName = "bin-manager.user-manager.subscribe"

	// webhook-manager
	QueueNameWebhookEvent     QueueName = "bin-manager.webhook-manager.event"
	QueueNameWebhookRequest   QueueName = "bin-manager.webhook-manager.request"
	QueueNameWebhookSubscribe QueueName = "bin-manager.webhook-manager.subscribe"
)

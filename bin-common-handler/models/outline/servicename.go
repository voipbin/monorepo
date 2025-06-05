package outline

import "strings"

// ServiceName type define
type ServiceName string

// list of service names
// this will be used to event publisher name as well.
const (
	ServiceNameAsteriskProxy ServiceName = "asterisk-proxy"

	ServiceNameAgentManager      ServiceName = "agent-manager"
	ServiceNameAIManager         ServiceName = "ai-manager"
	ServiceNameAPIManager        ServiceName = "api-manager"
	ServiceNameBillingManager    ServiceName = "billing-manager"
	ServiceNameCallManager       ServiceName = "call-manager"
	ServiceNameCampaignManager   ServiceName = "campaign-manager"
	ServiceNameChatManager       ServiceName = "chat-manager"
	ServiceNameConferenceManager ServiceName = "conference-manager"
	ServiceNameCustomerManager   ServiceName = "customer-manager"
	ServiceNameFlowManager       ServiceName = "flow-manager"
	ServiceNameHookManager       ServiceName = "hook-manager"
	ServiceNameMessageManager    ServiceName = "message-manager"
	ServiceNameNumberManager     ServiceName = "number-manager"
	ServiceNameOutdialManager    ServiceName = "outdial-manager"
	ServiceNameQueueManager      ServiceName = "queue-manager"
	ServiceNameRegistrarManager  ServiceName = "registrar-manager"
	ServiceNameRouteManager      ServiceName = "route-manager"
	ServiceNameSentinelManager   ServiceName = "sentinel-manager"
	ServiceNameStorageManager    ServiceName = "storage-manager"
	ServiceNameTagManager        ServiceName = "tag-manager"
	ServiceNameTranscribeManager ServiceName = "transcribe-manager"
	ServiceNameTTSManager        ServiceName = "tts-manager"
	ServiceNameWebhookManager    ServiceName = "webhook-manager"
)

// GetMetricNameSpace returns the metric namespace of the given service name for the prometheus metric namespace.
func GetMetricNameSpace(serviceName ServiceName) string {
	return strings.ReplaceAll(string(serviceName), "-", "_")
}

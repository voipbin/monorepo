package outline

import "testing"

// Test_QueueNameRequestAll asserts that QueueNameRequestAll() matches the
// *Request constants declared in queuename.go. The expected list is hardcoded
// on purpose: adding a new *Request constant must be a conscious decision to
// update both QueueNameRequestAll() and this test.
func Test_QueueNameRequestAll(t *testing.T) {

	expectRes := []QueueName{
		QueueNameAIRequest,
		QueueNameKamailioRequest,
		QueueNameAgentRequest,
		QueueNameAPIRequest,
		QueueNameBillingRequest,
		QueueNameCallRequest,
		QueueNameCampaignRequest,
		QueueNameConferenceRequest,
		QueueNameContactRequest,
		QueueNameConversationRequest,
		QueueNameCustomerRequest,
		QueueNameDirectRequest,
		QueueNameEmailRequest,
		QueueNameFlowRequest,
		QueueNameMessageRequest,
		QueueNameNumberRequest,
		QueueNameOutdialRequest,
		QueueNamePipecatRequest,
		QueueNameQueueRequest,
		QueueNameRagRequest,
		QueueNameRegistrarRequest,
		QueueNameRouteRequest,
		QueueNameSchedulerRequest,
		QueueNameSentinelRequest,
		QueueNameStorageRequest,
		QueueNameTagRequest,
		QueueNameTalkRequest,
		QueueNameTimelineRequest,
		QueueNameTranscribeRequest,
		QueueNameTransferRequest,
		QueueNameTTSRequest,
		QueueNameUserRequest,
		QueueNameWebchatRequest,
		QueueNameWebhookRequest,
	}

	res := QueueNameRequestAll()

	if len(res) != len(expectRes) {
		t.Errorf("Wrong length. expect: %d, got: %d", len(expectRes), len(res))
	}

	resMap := make(map[QueueName]bool, len(res))
	for _, q := range res {
		if resMap[q] {
			t.Errorf("Duplicated queue name. queue: %s", q)
		}
		resMap[q] = true
	}

	for _, q := range expectRes {
		if !resMap[q] {
			t.Errorf("Missing queue name. queue: %s", q)
		}
	}
}

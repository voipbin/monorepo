package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
)

const ttsQueue = "bin-manager.tts-manager.request"

func Test_TTSV1SpeakingWrappers(t *testing.T) {
	speakingID := uuid.FromStringOrNil("3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26")
	customerID := uuid.FromStringOrNil("9f1b3d5a-7c2e-4a68-b4d0-2e6a8c0f1b35")
	referenceID := uuid.FromStringOrNil("6a8c0e2f-4b1d-4d75-9c3e-0b2d4f6a8c17")

	cases := []rpcCase{
		{
			name:           "create",
			queue:          ttsQueue,
			expectURI:      "/v1/speakings",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"9f1b3d5a-7c2e-4a68-b4d0-2e6a8c0f1b35","reference_type":"call","reference_id":"6a8c0e2f-4b1d-4d75-9c3e-0b2d4f6a8c17","language":"en-US","provider":"provider-a","voice_id":"voice-1","direction":"out"}`,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26","pod_id":"pod-a","status":"initiating"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingCreate(context.Background(), customerID, tmstreaming.ReferenceTypeCall, referenceID, "en-US", "provider-a", "voice-1", tmstreaming.DirectionOutgoing)
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}, PodID: "pod-a", Status: tmspeaking.StatusInitiating},
		},
		{
			name:           "get",
			queue:          ttsQueue,
			expectURI:      "/v1/speakings/3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26","language":"ko-KR"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingGet(context.Background(), speakingID)
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}, Language: "ko-KR"},
		},
		{
			name:           "gets_without_filters",
			queue:          ttsQueue,
			expectURI:      "/v1/speakings?page_token=&page_size=10",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `[]`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingGets(context.Background(), "", 10, nil)
			},
			expectRes: []*tmspeaking.Speaking{},
		},
		{
			name:           "gets_with_all_filters",
			queue:          ttsQueue,
			expectURI:      "/v1/speakings?page_token=2020-09-20T03:23:20.995000Z&page_size=10&customer_id=9f1b3d5a-7c2e-4a68-b4d0-2e6a8c0f1b35&reference_type=call&reference_id=6a8c0e2f-4b1d-4d75-9c3e-0b2d4f6a8c17&status=active&deleted=false",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `[{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26"}]`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingGets(context.Background(), "2020-09-20T03:23:20.995000Z", 10, map[tmspeaking.Field]any{
					tmspeaking.FieldCustomerID:    customerID,
					tmspeaking.FieldReferenceType: tmstreaming.ReferenceTypeCall,
					tmspeaking.FieldReferenceID:   referenceID,
					tmspeaking.FieldStatus:        tmspeaking.StatusActive,
					tmspeaking.FieldDeleted:       false,
				})
			},
			expectRes: []*tmspeaking.Speaking{{Identity: identity.Identity{ID: speakingID}}},
		},
		{
			name:           "say_goes_to_the_pod_queue",
			queue:          "bin-manager.tts-manager.request.pod-a",
			expectURI:      "/v1/speakings/3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26/say",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"text":"hello there"}`,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingSay(context.Background(), "pod-a", speakingID, "hello there")
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}},
		},
		{
			name:           "flush_goes_to_the_pod_queue",
			queue:          "bin-manager.tts-manager.request.pod-b",
			expectURI:      "/v1/speakings/3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26/flush",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingFlush(context.Background(), "pod-b", speakingID)
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}},
		},
		{
			name:           "stop_goes_to_the_pod_queue",
			queue:          "bin-manager.tts-manager.request.pod-c",
			expectURI:      "/v1/speakings/3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26/stop",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26","status":"stopped"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingStop(context.Background(), "pod-c", speakingID)
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}, Status: tmspeaking.StatusStopped},
		},
		{
			name:           "delete",
			queue:          ttsQueue,
			expectURI:      "/v1/speakings/3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"3c5e7a9b-1d2f-4b64-8a0c-5e7f9b1d3a26"}`,
			call: func(h *requestHandler) (any, error) {
				return h.TTSV1SpeakingDelete(context.Background(), speakingID)
			},
			expectRes: &tmspeaking.Speaking{Identity: identity.Identity{ID: speakingID}},
		},
	}

	runRPCCases(t, cases)
}

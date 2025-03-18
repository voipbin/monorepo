package engine_dialogflow_handler

import (
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/engine_dialogflow"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/models/identity"
	"testing"

	dialogflowpb "cloud.google.com/go/dialogflow/apiv2/dialogflowpb"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

func Test_getRequest(t *testing.T) {
	tests := []struct {
		name string

		engineData *engine_dialogflow.EngineDialogflow
		aicall     *aicall.AIcall
		message    *message.Message

		expectRes *dialogflowpb.DetectIntentRequest
	}{
		{
			name: "normal ES",

			engineData: &engine_dialogflow.EngineDialogflow{
				ProjectID: "test-project",
			},
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("53ae45c0-ff1b-11ef-9018-7b56cb798145"),
				},
				AIEngineModel: ai.EngineModelDialogflowES,
				Language:      "en-US",
			},
			message: &message.Message{
				Content: "test message",
			},

			expectRes: &dialogflowpb.DetectIntentRequest{
				Session: "projects/test-project/agent/sessions/53ae45c0-ff1b-11ef-9018-7b56cb798145",
				QueryInput: &dialogflowpb.QueryInput{
					Input: &dialogflowpb.QueryInput_Text{
						Text: &dialogflowpb.TextInput{
							Text:         "test message",
							LanguageCode: "en",
						},
					},
				},
			},
		},
		{
			name: "normal CX",

			engineData: &engine_dialogflow.EngineDialogflow{
				ProjectID: "test-project",
				Region:    "us-central1",
				AgentID:   "test-agent",
			},
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("53fa04ba-ff1b-11ef-a65c-cf8e61d0c4ae"),
				},
				AIEngineModel: ai.EngineModelDialogflowCX,
				Language:      "en-US",
			},
			message: &message.Message{
				Content: "test message CX",
			},

			expectRes: &dialogflowpb.DetectIntentRequest{
				Session: "projects/test-project/locations/us-central1/agents/test-agent/sessions/53fa04ba-ff1b-11ef-a65c-cf8e61d0c4ae",
				QueryInput: &dialogflowpb.QueryInput{
					Input: &dialogflowpb.QueryInput_Text{
						Text: &dialogflowpb.TextInput{
							Text:         "test message CX",
							LanguageCode: "en",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &engineDialogflowHandler{} // Create an instance of the handler

			res := h.getRequest(tt.engineData, tt.aicall, tt.message)

			if !proto.Equal(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %#v\ngot: %#v", tt.expectRes, res)
			}
		})
	}
}

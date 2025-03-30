package summaryhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"
)

func Test_contentGet(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		transcripts  []tmtranscript.Transcript

		responseVariable *fmvariable.Variable
		responseOpenai   *openai.ChatCompletionResponse

		expectedRequestContent RequestContent
		expectedRes            string
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			transcripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("77e95178-0b96-11f0-afe8-f7c1026e2d7c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78171978-0b96-11f0-930b-c3391e420f82"),
					},
				},
			},

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt: defaultSummaryGeneratePrompt,
				Transcripts: []tmtranscript.Transcript{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("77e95178-0b96-11f0-afe8-f7c1026e2d7c"),
						},
					},
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("78171978-0b96-11f0-930b-c3391e420f82"),
						},
					},
				},
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			expectedRes: "response content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)

			tmpContent, err := json.Marshal(tt.expectedRequestContent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			tmpRequestContent := &openai.ChatCompletionRequest{
				Model: defaultModel,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: string(tmpContent),
					},
				},
			}
			mockOpenai.EXPECT().Send(ctx, tmpRequestContent).Return(tt.responseOpenai, nil)

			res, err := h.contentGet(ctx, tt.activeflowID, tt.transcripts)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

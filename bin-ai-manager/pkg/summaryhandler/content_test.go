package summaryhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
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

func Test_contentProcessReferenceTypeConference(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID

		responseSummaries   []*summary.Summary
		responseConference  *cfconference.Conference
		responseTranscribes []tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript
		responseVariable    *fmvariable.Variable
		responseSend        *openai.ChatCompletionResponse

		expectedFilterSummary     map[summary.Field]any
		expectedReferenceID       uuid.UUID
		expectedFilterTranscribe  map[tmtranscribe.Field]any
		expectedFilterTranscripts map[tmtranscript.Field]any
		expectedActiveflowID      uuid.UUID
		expectedSummaryContent    string
	}{
		{
			name: "normal",

			conferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("12c0991c-0d78-11f0-956f-c7fd6e2a65cd"),
					},
					ActiveflowID: uuid.FromStringOrNil("4eb1732e-0d78-11f0-adc7-070b3fa7186b"),
					ReferenceID:  uuid.FromStringOrNil("4ddead0e-0d78-11f0-896a-930b66cfb72b"),
				},
			},
			responseConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4ddead0e-0d78-11f0-896a-930b66cfb72b"),
				},
				ConfbridgeID: uuid.FromStringOrNil("4e0cb7d0-0d78-11f0-bba8-27fe297783c9"),
			},
			responseTranscribes: []tmtranscribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e316e40-0d78-11f0-b125-a79d64ccad15"),
					},
				},
			},
			responseTranscripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e5f7218-0d78-11f0-97fa-ffa151b9b13c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e880ac0-0d78-11f0-bacc-73f6c9abeebe"),
					},
				},
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			responseSend: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedFilterSummary: map[summary.Field]any{
				summary.FieldDeleted:     false,
				summary.FieldReferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),
			},
			expectedReferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),
			expectedFilterTranscribe: map[tmtranscribe.Field]any{
				tmtranscribe.FieldDeleted:     false,
				tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManager.String(),
				tmtranscribe.FieldReferenceID: "4e0cb7d0-0d78-11f0-bba8-27fe297783c9",
			},
			expectedFilterTranscripts: map[tmtranscript.Field]any{
				tmtranscript.FieldDeleted:      false,
				tmtranscript.FieldTranscribeID: "4e316e40-0d78-11f0-b125-a79d64ccad15",
			},
			expectedActiveflowID:   uuid.FromStringOrNil("4eb1732e-0d78-11f0-adc7-070b3fa7186b"),
			expectedSummaryContent: "response content",
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

			mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return(tt.responseSummaries, nil)
			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.expectedReferenceID).Return(tt.responseConference, nil)

			// contentGetTranscripts
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), tt.expectedFilterTranscribe).Return(tt.responseTranscribes, nil)
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), tt.expectedFilterTranscripts).Return(tt.responseTranscripts, nil)

			// contentGet
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.expectedActiveflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(tt.responseSend, nil)

			// UpdateStatusDone
			mockDB.EXPECT().SummaryUpdate(ctx, tt.responseSummaries[0].ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseSummaries[0].ID).Return(tt.responseSummaries[0], nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseSummaries[0].CustomerID, summary.EventTypeUpdated, tt.responseSummaries[0])

			if err := h.contentProcessReferenceTypeConference(ctx, tt.conferenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

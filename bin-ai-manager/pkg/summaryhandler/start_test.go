package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
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

// func Test_Start_startReferenceTypeTranscribe(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customerID    uuid.UUID
// 		activeflowID  uuid.UUID
// 		referenceType summary.ReferenceType
// 		referenceID   uuid.UUID
// 		language      string

// 		responseTranscripts []tmtranscript.Transcript

// 		expectedFiltersGets        map[string]string
// 		expectedFiltersTranscripts map[string]string

// 		// responseUUID uuid.UUID

// 		// expectedSummary   *summary.Summary
// 		// expectedVariables map[string]string
// 	}{
// 		{
// 			name: "normal",

// 			customerID:    uuid.FromStringOrNil("006be440-0b95-11f0-b84e-8332788713e5"),
// 			activeflowID:  uuid.FromStringOrNil("00aaabda-0b95-11f0-bd9d-2bc8734ee6d3"),
// 			referenceType: summary.ReferenceTypeTranscribe,
// 			referenceID:   uuid.FromStringOrNil("00d8eacc-0b95-11f0-b90c-93cf65130636"),
// 			language:      "en-US",

// 			responseTranscripts: []tmtranscript.Transcript{
// 				{
// 					Identity: commonidentity.Identity{
// 						ID: uuid.FromStringOrNil("012fee58-0b95-11f0-b0d3-57dd5f5d863e"),
// 					},
// 				},
// 				{
// 					Identity: commonidentity.Identity{
// 						ID: uuid.FromStringOrNil("0158f668-0b95-11f0-b099-bb17de6f4902"),
// 					},
// 				},
// 			},

// 			expectedFiltersGets: map[string]string{
// 				"deleted":      "false",
// 				"customer_id":  "006be440-0b95-11f0-b84e-8332788713e5",
// 				"reference_id": "00d8eacc-0b95-11f0-b90c-93cf65130636",
// 				"language":     "en-US",
// 			},
// 			expectedFiltersTranscripts: map[string]string{
// 				"deleted":      "false",
// 				"reference_id": "00d8eacc-0b95-11f0-b90c-93cf65130636",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

// 			h := summaryHandler{
// 				utilHandler:   mockUtil,
// 				db:            mockDB,
// 				notifyHandler: mockNotify,
// 				reqestHandler: mockReq,
// 			}
// 			ctx := context.Background()

// 			mockDB.EXPECT().SummaryGets(ctx, gomock.Any(), gomock.Any(), tt.expectedFiltersGets).Return(nil, fmt.Errorf(""))

// 			// startReferenceTypeTranscribe
// 			mockReq.EXPECT().TranscribeV1TranscriptGets(ctx, "", uint64(1000), tt.expectedFiltersTranscripts).Return(tt.responseTranscripts, nil)
// 			mockReq.EXPECT()

// 			res, err := h.Start(ctx, tt.customerID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.language)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			// if !reflect.DeepEqual(res, tt.expectedSummary) {
// 			// 	t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
// 			// }
// 		})
// 	}
// }

func Test_startReferenceTypeTranscribe(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		language     string

		responseTranscripts []tmtranscript.Transcript
		responseVariable    *fmvariable.Variable
		responseOpenai      *openai.ChatCompletionResponse
		responseUUID        uuid.UUID

		expectedFiltersTranscripts map[string]string
		expectedAIRequest          *openai.ChatCompletionRequest
		expectedSummary            *summary.Summary
		expectedVariables          map[string]string
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("700d17c8-0ba0-11f0-a3e9-0773643b029a"),
			activeflowID: uuid.FromStringOrNil("7035bfd4-0ba0-11f0-847f-dbe299d9dde2"),
			referenceID:  uuid.FromStringOrNil("705f227a-0ba0-11f0-aa84-b3acfd22daa1"),
			language:     "en-US",

			responseTranscripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6fba4494-0ba0-11f0-8a39-4f28e940be73"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6fe870ee-0ba0-11f0-9edb-bf0d1f088063"),
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
			responseUUID: uuid.FromStringOrNil("fe470e72-0ba0-11f0-87c8-a39a79b2cfe1"),

			expectedFiltersTranscripts: map[string]string{
				"deleted":      "false",
				"reference_id": "705f227a-0ba0-11f0-aa84-b3acfd22daa1",
			},
			expectedAIRequest: &openai.ChatCompletionRequest{
				Model: defaultModel,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: `{"prompt":"Generate a concise yet informative call summary based on the provided transcription, recording link, conference details and other variables. Focus on key points, action items, and important decisions made during the call.","transcripts":[{"id":"6fba4494-0ba0-11f0-8a39-4f28e940be73","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""},{"id":"6fe870ee-0ba0-11f0-9edb-bf0d1f088063","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}],"variables":{"key1":"value1"}}`,
					},
				},
			},
			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe470e72-0ba0-11f0-87c8-a39a79b2cfe1"),
					CustomerID: uuid.FromStringOrNil("700d17c8-0ba0-11f0-a3e9-0773643b029a"),
				},
				ActiveflowID:  uuid.FromStringOrNil("7035bfd4-0ba0-11f0-847f-dbe299d9dde2"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("705f227a-0ba0-11f0-aa84-b3acfd22daa1"),
				Status:        summary.StatusDone,
				Language:      "en-US",
				Content:       "response content",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "fe470e72-0ba0-11f0-87c8-a39a79b2cfe1",
				variableSummaryReferenceType: string(summary.ReferenceTypeTranscribe),
				variableSummaryReferenceID:   "705f227a-0ba0-11f0-aa84-b3acfd22daa1",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "response content",
			},
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
				reqestHandler: mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscriptGets(ctx, "", uint64(1000), tt.expectedFiltersTranscripts).Return(tt.responseTranscripts, nil)

			// getContent
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, tt.expectedAIRequest).Return(tt.responseOpenai, nil)

			// Create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.startReferenceTypeTranscribe(ctx, tt.customerID, tt.activeflowID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

func Test_startReferenceTypeRecording(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		language     string

		responseTranscribe  *tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript
		responseVariable    *fmvariable.Variable
		responseOpenai      *openai.ChatCompletionResponse
		responseUUID        uuid.UUID

		expectedFiltersTranscripts map[string]string
		expectedAIRequest          *openai.ChatCompletionRequest
		expectedSummary            *summary.Summary
		expectedVariables          map[string]string
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
			activeflowID: uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
			referenceID:  uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
			language:     "en-US",

			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5885ec42-0b9b-11f0-99ba-4b3106d01f9b"),
				},
			},
			responseTranscripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("79ee469a-0b96-11f0-ad07-37789426e403"),
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
			responseUUID: uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),

			expectedFiltersTranscripts: map[string]string{
				"deleted":      "false",
				"reference_id": "58618082-0b9b-11f0-bd70-b74a9214cd07",
			},
			expectedAIRequest: &openai.ChatCompletionRequest{
				Model: defaultModel,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: `{"prompt":"Generate a concise yet informative call summary based on the provided transcription, recording link, conference details and other variables. Focus on key points, action items, and important decisions made during the call.","transcripts":[{"id":"78cdacd8-0b96-11f0-83d8-e71b47975e9a","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""},{"id":"79ee469a-0b96-11f0-ad07-37789426e403","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}],"variables":{"key1":"value1"}}`,
					},
				},
			},
			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),
					CustomerID: uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
				},
				ActiveflowID:  uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
				Status:        summary.StatusDone,
				Language:      "en-US",
				Content:       "response content",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "58aa4c9a-0b9b-11f0-a701-a706590d3061",
				variableSummaryReferenceType: string(summary.ReferenceTypeRecording),
				variableSummaryReferenceID:   "58618082-0b9b-11f0-bd70-b74a9214cd07",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "response content",
			},
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
				reqestHandler: mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeStart(
				ctx,
				cmcustomer.IDAIManager,
				tt.activeflowID,
				uuid.Nil,
				tmtranscribe.ReferenceTypeRecording,
				tt.referenceID,
				tt.language,
				tmtranscribe.DirectionBoth,
				30000,
			).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().TranscribeV1TranscriptGets(ctx, "", uint64(1000), tt.expectedFiltersTranscripts).Return(tt.responseTranscripts, nil)

			// getContent
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, tt.expectedAIRequest).Return(tt.responseOpenai, nil)

			// Create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.startReferenceTypeRecording(ctx, tt.customerID, tt.activeflowID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

func Test_getContent(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		transcripts  []tmtranscript.Transcript

		responseVariable *fmvariable.Variable
		responseOpenai   *openai.ChatCompletionResponse

		expectAIRequest *openai.ChatCompletionRequest
		expectedRes     string
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

			expectAIRequest: &openai.ChatCompletionRequest{
				Model: defaultModel,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: `{"prompt":"Generate a concise yet informative call summary based on the provided transcription, recording link, conference details and other variables. Focus on key points, action items, and important decisions made during the call.","transcripts":[{"id":"77e95178-0b96-11f0-afe8-f7c1026e2d7c","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""},{"id":"78171978-0b96-11f0-930b-c3391e420f82","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}],"variables":{"key1":"value1"}}`,
					},
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
				reqestHandler: mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, tt.expectAIRequest).Return(tt.responseOpenai, nil)

			res, err := h.getContent(ctx, tt.activeflowID, tt.transcripts)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	"reflect"

	"testing"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		onEndFlowID  uuid.UUID
		referenceID  uuid.UUID
		language     string

		responseCall       *cmcall.Call
		responseTranscribe *tmtranscribe.Transcribe
		responseUUID       uuid.UUID

		expectedSummary   *summary.Summary
		expectedVariables map[string]string
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("1b6ab092-0bf3-11f0-a4f9-0bc0ff112b6a"),
			activeflowID: uuid.FromStringOrNil("1ba24124-0bf3-11f0-985b-a3c400f7e5b8"),
			onEndFlowID:  uuid.FromStringOrNil("1bcb3282-0bf3-11f0-a3f4-5f8c23fee6bb"),
			referenceID:  uuid.FromStringOrNil("1c0229b8-0bf3-11f0-b474-cf304324c97a"),
			language:     "en-US",

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c0229b8-0bf3-11f0-b474-cf304324c97a"),
				},
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c394aba-0bf3-11f0-affc-23b38f9c795c"),
				},
			},
			responseUUID: uuid.FromStringOrNil("1c6afcb8-0bf3-11f0-8dd2-23f00a2ba386"),

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1c6afcb8-0bf3-11f0-8dd2-23f00a2ba386"),
					CustomerID: uuid.FromStringOrNil("1b6ab092-0bf3-11f0-a4f9-0bc0ff112b6a"),
				},
				ActiveflowID:  uuid.FromStringOrNil("1ba24124-0bf3-11f0-985b-a3c400f7e5b8"),
				OnEndFlowID:   uuid.FromStringOrNil("1bcb3282-0bf3-11f0-a3f4-5f8c23fee6bb"),
				ReferenceType: summary.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("1c0229b8-0bf3-11f0-b474-cf304324c97a"),
				Status:        summary.StatusProgressing,
				Language:      "en-US",
				Content:       "",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "1c6afcb8-0bf3-11f0-8dd2-23f00a2ba386",
				variableSummaryReferenceType: string(summary.ReferenceTypeCall),
				variableSummaryReferenceID:   "1c0229b8-0bf3-11f0-b474-cf304324c97a",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "",
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

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockReq.EXPECT().TranscribeV1TranscribeStart(
				ctx,
				uuid.Nil,
				cmcustomer.IDAIManager,
				tt.activeflowID,
				uuid.Nil,
				tmtranscribe.ReferenceTypeCall,
				tt.referenceID,
				tt.language,
				tmtranscribe.DirectionBoth,
				tmtranscribe.ProviderEmpty,
				5000,
			).Return(tt.responseTranscribe, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.startReferenceTypeCall(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

func Test_startReferenceTypeConference(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		onEndFlowID  uuid.UUID
		referenceID  uuid.UUID
		language     string

		responseConference *cfconference.Conference
		responseTranscribe *tmtranscribe.Transcribe
		responseUUID       uuid.UUID

		expectedSummary   *summary.Summary
		expectedVariables map[string]string
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("cb445f30-0bf4-11f0-ac2b-dfeb5d139e66"),
			activeflowID: uuid.FromStringOrNil("cb6db90c-0bf4-11f0-82d1-5f417a415a20"),
			onEndFlowID:  uuid.FromStringOrNil("cb9678ce-0bf4-11f0-ae81-933725512be0"),
			referenceID:  uuid.FromStringOrNil("1c0229b8-0bf3-11f0-b474-cf304324c97a"),
			language:     "en-US",

			responseConference: &cfconference.Conference{
				ConfbridgeID: uuid.FromStringOrNil("cbe3ed20-0bf4-11f0-87e6-6b37fa660498"),
				Status:       cfconference.StatusProgressing,
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c394aba-0bf3-11f0-affc-23b38f9c795c"),
				},
			},
			responseUUID: uuid.FromStringOrNil("cbb64a96-0bf4-11f0-9432-5b978ae5befd"),

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cbb64a96-0bf4-11f0-9432-5b978ae5befd"),
					CustomerID: uuid.FromStringOrNil("cb445f30-0bf4-11f0-ac2b-dfeb5d139e66"),
				},
				ActiveflowID:  uuid.FromStringOrNil("cb6db90c-0bf4-11f0-82d1-5f417a415a20"),
				OnEndFlowID:   uuid.FromStringOrNil("cb9678ce-0bf4-11f0-ae81-933725512be0"),
				ReferenceType: summary.ReferenceTypeConference,
				ReferenceID:   uuid.FromStringOrNil("1c0229b8-0bf3-11f0-b474-cf304324c97a"),
				Status:        summary.StatusProgressing,
				Language:      "en-US",
				Content:       "",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "cbb64a96-0bf4-11f0-9432-5b978ae5befd",
				variableSummaryReferenceType: string(summary.ReferenceTypeConference),
				variableSummaryReferenceID:   "1c0229b8-0bf3-11f0-b474-cf304324c97a",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "",
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

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.referenceID).Return(tt.responseConference, nil)

			mockReq.EXPECT().TranscribeV1TranscribeStart(
				ctx,
				uuid.Nil,
				cmcustomer.IDAIManager,
				tt.activeflowID,
				uuid.Nil,
				tmtranscribe.ReferenceTypeConfbridge,
				tt.responseConference.ConfbridgeID,
				tt.language,
				tmtranscribe.DirectionIn,
				tmtranscribe.ProviderEmpty,
				5000,
			).Return(tt.responseTranscribe, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.startReferenceTypeConference(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

func Test_startReferenceTypeTranscribe(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		onEndFlowID  uuid.UUID
		referenceID  uuid.UUID
		language     string

		responseTranscripts []tmtranscript.Transcript
		responseVariable    *fmvariable.Variable
		responseOpenai      *openai.ChatCompletionResponse
		responseUUID        uuid.UUID
		responseActiveflow  *fmactiveflow.Activeflow

		expectedFiltersTranscripts map[tmtranscript.Field]any
		expectedSummary            *summary.Summary
		expectedVariables          map[string]string
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("700d17c8-0ba0-11f0-a3e9-0773643b029a"),
			activeflowID: uuid.FromStringOrNil("7035bfd4-0ba0-11f0-847f-dbe299d9dde2"),
			onEndFlowID:  uuid.FromStringOrNil("0e9f3a32-0bde-11f0-8506-53cc04943f44"),
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
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1a809d14-0bf2-11f0-9f34-bba26e9f13fa"),
				},
			},

			expectedFiltersTranscripts: map[tmtranscript.Field]any{
				tmtranscript.FieldDeleted:      false,
				tmtranscript.FieldTranscribeID: "705f227a-0ba0-11f0-aa84-b3acfd22daa1",
			},
			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe470e72-0ba0-11f0-87c8-a39a79b2cfe1"),
					CustomerID: uuid.FromStringOrNil("700d17c8-0ba0-11f0-a3e9-0773643b029a"),
				},
				ActiveflowID:  uuid.FromStringOrNil("7035bfd4-0ba0-11f0-847f-dbe299d9dde2"),
				OnEndFlowID:   uuid.FromStringOrNil("0e9f3a32-0bde-11f0-8506-53cc04943f44"),
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
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), tt.expectedFiltersTranscripts).Return(tt.responseTranscripts, nil)

			// getContent
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(tt.responseOpenai, nil)

			// Create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			if tt.expectedSummary.OnEndFlowID != uuid.Nil {
				// startOnEndFlow
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					ctx,
					uuid.Nil,
					tt.expectedSummary.CustomerID,
					tt.expectedSummary.OnEndFlowID,
					fmactiveflow.ReferenceTypeAI,
					tt.expectedSummary.ID,
					tt.expectedSummary.ActiveflowID,
					nil,
				gomock.Any(),
				gomock.Any(),
				).Return(tt.responseActiveflow, nil)
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, tt.expectedVariables).Return(nil)
				mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)
			}

			res, err := h.startReferenceTypeTranscribe(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceID, tt.language)
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
		onEndFlowID  uuid.UUID
		referenceID  uuid.UUID
		language     string

		// setupTranscribeMocks sets up the transcribe-reuse / creation portion
		// (getRecordingTranscripts) for the case. gomock's strict controller
		// makes the presence/absence of TranscribeV1TranscribeStart the assertion
		// for reuse vs new-creation.
		setupTranscribeMocks func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, tt testCaseRecording)

		responseVariable   *fmvariable.Variable
		responseOpenai     *openai.ChatCompletionResponse
		responseUUID       uuid.UUID
		responseActiveflow *fmactiveflow.Activeflow

		expectedSummary   *summary.Summary
		expectedVariables map[string]string
	}{
		{
			name: "normal - no reusable transcribe, new creation",

			customerID:   uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
			activeflowID: uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
			onEndFlowID:  uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
			referenceID:  uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
			language:     "en-US",

			setupTranscribeMocks: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, tt testCaseRecording) {
				// reuse lookup returns nothing -> new creation path.
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
					tmtranscribe.FieldReferenceID:   tt.referenceID.String(),
					tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeRecording,
					tmtranscribe.FieldStatus:        tmtranscribe.StatusDone,
					tmtranscribe.FieldDeleted:       false,
				}).Return([]tmtranscribe.Transcribe{}, nil)

				newTranscribe := &tmtranscribe.Transcribe{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5885ec42-0b9b-11f0-99ba-4b3106d01f9b"),
					},
				}
				mockReq.EXPECT().TranscribeV1TranscribeStart(
					ctx,
					uuid.Nil,
					cmcustomer.IDAIManager,
					tt.activeflowID,
					uuid.Nil,
					tmtranscribe.ReferenceTypeRecording,
					tt.referenceID,
					defaultSTTFallbackLanguage,
					tmtranscribe.DirectionBoth,
					tmtranscribe.ProviderEmpty,
					300000,
				).Return(newTranscribe, nil)
				mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: "5885ec42-0b9b-11f0-99ba-4b3106d01f9b",
				}).Return([]tmtranscript.Transcript{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a")}},
				}, nil)
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
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6ef9afc-0bf2-11f0-9371-13e70123d868"),
				},
			},

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),
					CustomerID: uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
				},
				ActiveflowID:  uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
				OnEndFlowID:   uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
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
		{
			name: "reuse - original transcribe with transcripts",

			customerID:   uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
			activeflowID: uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
			onEndFlowID:  uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
			referenceID:  uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
			language:     "ko-KR",

			setupTranscribeMocks: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, tt testCaseRecording) {
				// reuse lookup returns an IDAIManager (skipped) then an original
				// transcribe with transcripts (adopted). No TranscribeStart.
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
					tmtranscribe.FieldReferenceID:   tt.referenceID.String(),
					tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeRecording,
					tmtranscribe.FieldStatus:        tmtranscribe.StatusDone,
					tmtranscribe.FieldDeleted:       false,
				}).Return([]tmtranscribe.Transcribe{
					{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("c1a1b2c4-0b9b-11f0-99ba-4b3106d01f9b"),
							CustomerID: cmcustomer.IDAIManager,
						},
					},
					{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("c2a1b2c4-0b9b-11f0-99ba-4b3106d01f9b"),
							CustomerID: uuid.FromStringOrNil("d0000000-0b9b-11f0-99ba-4b3106d01f9b"),
						},
					},
				}, nil)
				// only the non-IDAIManager candidate's transcripts are fetched
				// (short-circuit stops there).
				mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: "c2a1b2c4-0b9b-11f0-99ba-4b3106d01f9b",
				}).Return([]tmtranscript.Transcript{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a")}},
				}, nil)
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
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6ef9afc-0bf2-11f0-9371-13e70123d868"),
				},
			},

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),
					CustomerID: uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
				},
				ActiveflowID:  uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
				OnEndFlowID:   uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
				Status:        summary.StatusDone,
				Language:      "ko-KR",
				Content:       "response content",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "58aa4c9a-0b9b-11f0-a701-a706590d3061",
				variableSummaryReferenceType: string(summary.ReferenceTypeRecording),
				variableSummaryReferenceID:   "58618082-0b9b-11f0-bd70-b74a9214cd07",
				variableSummaryLanguage:      "ko-KR",
				variableSummaryContent:       "response content",
			},
		},
		{
			name: "reuse - only IDAIManager transcribe exists, new creation",

			customerID:   uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
			activeflowID: uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
			onEndFlowID:  uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
			referenceID:  uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
			language:     "ko-KR",

			setupTranscribeMocks: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, tt testCaseRecording) {
				// only IDAIManager candidate -> excluded -> new creation.
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
					tmtranscribe.FieldReferenceID:   tt.referenceID.String(),
					tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeRecording,
					tmtranscribe.FieldStatus:        tmtranscribe.StatusDone,
					tmtranscribe.FieldDeleted:       false,
				}).Return([]tmtranscribe.Transcribe{
					{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("c1a1b2c4-0b9b-11f0-99ba-4b3106d01f9b"),
							CustomerID: cmcustomer.IDAIManager,
						},
					},
				}, nil)

				newTranscribe := &tmtranscribe.Transcribe{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5885ec42-0b9b-11f0-99ba-4b3106d01f9b"),
					},
				}
				mockReq.EXPECT().TranscribeV1TranscribeStart(
					ctx,
					uuid.Nil,
					cmcustomer.IDAIManager,
					tt.activeflowID,
					uuid.Nil,
					tmtranscribe.ReferenceTypeRecording,
					tt.referenceID,
					defaultSTTFallbackLanguage,
					tmtranscribe.DirectionBoth,
					tmtranscribe.ProviderEmpty,
					300000,
				).Return(newTranscribe, nil)
				mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: "5885ec42-0b9b-11f0-99ba-4b3106d01f9b",
				}).Return([]tmtranscript.Transcript{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a")}},
				}, nil)
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
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6ef9afc-0bf2-11f0-9371-13e70123d868"),
				},
			},

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),
					CustomerID: uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
				},
				ActiveflowID:  uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
				OnEndFlowID:   uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
				Status:        summary.StatusDone,
				Language:      "ko-KR",
				Content:       "response content",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "58aa4c9a-0b9b-11f0-a701-a706590d3061",
				variableSummaryReferenceType: string(summary.ReferenceTypeRecording),
				variableSummaryReferenceID:   "58618082-0b9b-11f0-bd70-b74a9214cd07",
				variableSummaryLanguage:      "ko-KR",
				variableSummaryContent:       "response content",
			},
		},
		{
			name: "reuse - original transcribe with no transcripts, new creation",

			customerID:   uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
			activeflowID: uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
			onEndFlowID:  uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
			referenceID:  uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
			language:     "ko-KR",

			setupTranscribeMocks: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, tt testCaseRecording) {
				// original candidate but with empty transcripts -> new creation.
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
					tmtranscribe.FieldReferenceID:   tt.referenceID.String(),
					tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeRecording,
					tmtranscribe.FieldStatus:        tmtranscribe.StatusDone,
					tmtranscribe.FieldDeleted:       false,
				}).Return([]tmtranscribe.Transcribe{
					{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("c2a1b2c4-0b9b-11f0-99ba-4b3106d01f9b"),
							CustomerID: uuid.FromStringOrNil("d0000000-0b9b-11f0-99ba-4b3106d01f9b"),
						},
					},
				}, nil)
				mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: "c2a1b2c4-0b9b-11f0-99ba-4b3106d01f9b",
				}).Return([]tmtranscript.Transcript{}, nil)

				newTranscribe := &tmtranscribe.Transcribe{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5885ec42-0b9b-11f0-99ba-4b3106d01f9b"),
					},
				}
				mockReq.EXPECT().TranscribeV1TranscribeStart(
					ctx,
					uuid.Nil,
					cmcustomer.IDAIManager,
					tt.activeflowID,
					uuid.Nil,
					tmtranscribe.ReferenceTypeRecording,
					tt.referenceID,
					defaultSTTFallbackLanguage,
					tmtranscribe.DirectionBoth,
					tmtranscribe.ProviderEmpty,
					300000,
				).Return(newTranscribe, nil)
				mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), map[tmtranscript.Field]any{
					tmtranscript.FieldDeleted:      false,
					tmtranscript.FieldTranscribeID: "5885ec42-0b9b-11f0-99ba-4b3106d01f9b",
				}).Return([]tmtranscript.Transcript{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a")}},
				}, nil)
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
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6ef9afc-0bf2-11f0-9371-13e70123d868"),
				},
			},

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58aa4c9a-0b9b-11f0-a701-a706590d3061"),
					CustomerID: uuid.FromStringOrNil("7852ac5e-0b96-11f0-a61f-afb2360f9d5b"),
				},
				ActiveflowID:  uuid.FromStringOrNil("78896fdc-0b96-11f0-8ec3-db055fa6d92f"),
				OnEndFlowID:   uuid.FromStringOrNil("0ecd2b36-0bde-11f0-bb42-3f48eb4490e1"),
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("58618082-0b9b-11f0-bd70-b74a9214cd07"),
				Status:        summary.StatusDone,
				Language:      "ko-KR",
				Content:       "response content",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "58aa4c9a-0b9b-11f0-a701-a706590d3061",
				variableSummaryReferenceType: string(summary.ReferenceTypeRecording),
				variableSummaryReferenceID:   "58618082-0b9b-11f0-bd70-b74a9214cd07",
				variableSummaryLanguage:      "ko-KR",
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
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			// getRecordingTranscripts (reuse lookup / new creation)
			tt.setupTranscribeMocks(ctx, mockReq, testCaseRecording{
				referenceID:  tt.referenceID,
				activeflowID: tt.activeflowID,
			})

			// getContent
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(tt.responseOpenai, nil)

			// Create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			if tt.expectedSummary.OnEndFlowID != uuid.Nil {
				// startOnEndFlow
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					ctx,
					uuid.Nil,
					tt.expectedSummary.CustomerID,
					tt.expectedSummary.OnEndFlowID,
					fmactiveflow.ReferenceTypeAI,
					tt.expectedSummary.ID,
					tt.expectedSummary.ActiveflowID,
					nil,
					gomock.Any(),
					gomock.Any(),
				).Return(tt.responseActiveflow, nil)
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, tt.expectedVariables).Return(nil)
				mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)
			}

			res, err := h.startReferenceTypeRecording(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

// testCaseRecording carries the case-specific values the transcribe-mock setup
// closures need.
type testCaseRecording struct {
	referenceID  uuid.UUID
	activeflowID uuid.UUID
}

// Test_Start_normalizeOutputLanguage verifies that Start normalizes an empty
// output language to en-US BEFORE the dedup lookup, so that an empty language
// and its normalized default do not diverge into two separate summaries. The
// dedup lookup (SummaryList) must be called with FieldLanguage == "en-US".
func Test_Start_normalizeOutputLanguage(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("6f6b0a1c-0000-11f0-9d2e-4b7c8f2a5d70")
	referenceID := uuid.FromStringOrNil("6f6b0a1c-0001-11f0-9d2e-4b7c8f2a5d70")

	existing := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("6f6b0a1c-0002-11f0-9d2e-4b7c8f2a5d70"),
			CustomerID: customerID,
		},
		ReferenceID: referenceID,
		Language:    "en-US",
	}

	// dedup lookup must use the normalized language (en-US), not the empty input.
	expectedFilters := map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldCustomerID:  customerID,
		summary.FieldReferenceID: referenceID,
		summary.FieldLanguage:    "en-US",
	}
	mockDB.EXPECT().SummaryList(ctx, uint64(1000), "", expectedFilters).Return([]*summary.Summary{existing}, nil)

	// empty language passed in -> normalized to en-US -> dedup hit -> returns existing.
	res, err := h.Start(ctx, customerID, uuid.Nil, uuid.Nil, summary.ReferenceTypeRecording, referenceID, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, existing) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", existing, res)
	}
}

func Test_startOnEndFlow(t *testing.T) {

	tests := []struct {
		name string

		summary *summary.Summary

		responseActiveflow *fmactiveflow.Activeflow

		expectedVariables map[string]string
	}{
		{
			name: "normal",

			summary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a45b014-0bf8-11f0-ad72-17ef64cb5ce4"),
					CustomerID: uuid.FromStringOrNil("0a7c51c8-0bf8-11f0-a706-978ade8856e6"),
				},
				ActiveflowID:  uuid.FromStringOrNil("0ad7efba-0bf8-11f0-88ea-93f7520edc74"),
				OnEndFlowID:   uuid.FromStringOrNil("0aa9b118-0bf8-11f0-b61a-a3b3a1840fbb"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("0b05c908-0bf8-11f0-8420-8f29b3c01ae1"),

				Language: "en-US",
				Content:  "Hello world",
			},

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0b3c81c8-0bf8-11f0-b774-2bf3f3cf03f4"),
				},
			},

			expectedVariables: map[string]string{
				variableSummaryID:            "0a45b014-0bf8-11f0-ad72-17ef64cb5ce4",
				variableSummaryReferenceType: string(summary.ReferenceTypeTranscribe),
				variableSummaryReferenceID:   "0b05c908-0bf8-11f0-8420-8f29b3c01ae1",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "Hello world",
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
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(
				ctx,
				uuid.Nil,
				tt.summary.CustomerID,
				tt.summary.OnEndFlowID,
				fmactiveflow.ReferenceTypeAI,
				tt.summary.ID,
				tt.summary.ActiveflowID,
				nil,
			gomock.Any(),
			gomock.Any(),
			).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, tt.expectedVariables).Return(nil)
			mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)

			if errFlow := h.startOnEndFlow(ctx, tt.summary); errFlow != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errFlow)
			}
		})
	}
}

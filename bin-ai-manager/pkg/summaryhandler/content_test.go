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
			mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, tt.responseSummaries[0].ID, gomock.Any()).Return(int64(1), nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseSummaries[0].ID).Return(tt.responseSummaries[0], nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseSummaries[0].CustomerID, summary.EventTypeUpdated, tt.responseSummaries[0])

			if err := h.contentProcessReferenceTypeConference(ctx, tt.conferenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_contentProcessReferenceTypeConference_alreadyDone is a VOIP-1422 regression
// test: when UpdateStatusDone returns ErrSummaryAlreadyDone (the DB-level conditional
// update affected zero rows -- this is the losing side of bin-conference-manager's
// double conference_deleted delivery, see ErrSummaryAlreadyDone's doc comment),
// contentProcessReferenceTypeConference must return cleanly (nil error) WITHOUT calling
// startOnEndFlow or its downstream (FlowV1ActiveflowExecute) or the summary_updated
// webhook. No mock expectations are set up for any of those -- gomock's strict
// controller fails the test on any such unexpected call, so their absence is what
// proves this test actually exercises the guard rather than passing vacuously.
func Test_contentProcessReferenceTypeConference_alreadyDone(t *testing.T) {
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

	conferenceID := uuid.FromStringOrNil("a1b2c3d4-8b8c-11f0-9d2e-4b7c8f2a5d70")
	confbridgeID := uuid.FromStringOrNil("a1e01234-8b8c-11f0-9d2e-4b7c8f2a5d70")
	activeflowID := uuid.FromStringOrNil("a2103456-8b8c-11f0-9d2e-4b7c8f2a5d70")

	sm := &summary.Summary{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("a2321234-8b8c-11f0-9d2e-4b7c8f2a5d70"),
		},
		ActiveflowID: activeflowID,
		ReferenceID:  conferenceID,
	}
	cf := &cfconference.Conference{
		Identity: commonidentity.Identity{
			ID: conferenceID,
		},
		ConfbridgeID: confbridgeID,
	}
	transcribes := []tmtranscribe.Transcribe{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a2541234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	transcripts := []tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a2761234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	variable := &fmvariable.Variable{Variables: map[string]string{"key1": "value1"}}
	openaiRes := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "response content"}}},
	}

	mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, conferenceID).Return(cf, nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), gomock.Any()).Return(transcribes, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), gomock.Any()).Return(transcripts, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(variable, nil)
	mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(openaiRes, nil)

	// the losing delivery: DB-level guard rejects it. No SummaryGet, no
	// PublishWebhookEvent, no startOnEndFlow-related calls may follow.
	mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, sm.ID, gomock.Any()).Return(int64(0), nil)

	if err := h.contentProcessReferenceTypeConference(ctx, conferenceID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_contentProcessReferenceTypeCall_alreadyDone mirrors
// Test_contentProcessReferenceTypeConference_alreadyDone for the call reference type,
// confirming contentProcessReferenceTypeCall's independent stderrors.Is(err,
// ErrSummaryAlreadyDone) branch (content.go) also skips startOnEndFlow and the webhook
// cleanly rather than erroring or re-finalizing.
func Test_contentProcessReferenceTypeCall_alreadyDone(t *testing.T) {
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

	callID := uuid.FromStringOrNil("a3981234-8b8c-11f0-9d2e-4b7c8f2a5d70")
	activeflowID := uuid.FromStringOrNil("a3ba1234-8b8c-11f0-9d2e-4b7c8f2a5d70")

	sm := &summary.Summary{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("a3dc1234-8b8c-11f0-9d2e-4b7c8f2a5d70"),
		},
		ActiveflowID: activeflowID,
		ReferenceID:  callID,
	}
	transcribes := []tmtranscribe.Transcribe{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a3fe1234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	transcripts := []tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a4201234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	variable := &fmvariable.Variable{Variables: map[string]string{"key1": "value1"}}
	openaiRes := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "response content"}}},
	}

	mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), gomock.Any()).Return(transcribes, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), gomock.Any()).Return(transcripts, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(variable, nil)
	mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(openaiRes, nil)

	// the losing delivery: DB-level guard rejects it. No SummaryGet, no
	// PublishWebhookEvent, no startOnEndFlow-related calls may follow.
	mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, sm.ID, gomock.Any()).Return(int64(0), nil)

	if err := h.contentProcessReferenceTypeCall(ctx, callID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

package summaryhandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"
)

// setupRegenerateTranscribeReuse mocks getRecordingTranscripts's reuse path:
// a single original (non-IDAIManager) transcribe with one transcript is found,
// so no new TranscribeV1TranscribeStart is expected.
func setupRegenerateTranscribeReuse(ctx context.Context, mockReq *requesthandler.MockRequestHandler, referenceID uuid.UUID) {
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), map[tmtranscribe.Field]any{
		tmtranscribe.FieldReferenceID:   referenceID.String(),
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
	}).Return([]tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("78cdacd8-0b96-11f0-83d8-e71b47975e9a")}},
	}, nil)
}

func Test_Regenerate(t *testing.T) {

	tests := []struct {
		name string

		summaryID uuid.UUID
		language  string

		existing *summary.Summary

		responseOpenai *openai.ChatCompletionResponse

		// expectedLanguage is the confirmed output language written back
		// (existing.Language when the request language is empty).
		expectedLanguage string
		expectedContent  string

		updatedSummary *summary.Summary
	}{
		{
			name: "same language - empty request keeps existing language",

			summaryID: uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000001"),
			language:  "",

			existing: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000001"),
				},
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("c0000000-0000-11f0-0000-000000000001"),
				Status:        summary.StatusDone,
				Language:      "en-US",
				Content:       "old content",
			},

			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{Message: openai.ChatCompletionMessage{Content: "new content"}},
				},
			},

			expectedLanguage: "en-US",
			expectedContent:  "new content",

			updatedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000001"),
				},
				ReferenceType: summary.ReferenceTypeRecording,
				Status:        summary.StatusDone,
				Language:      "en-US",
				Content:       "new content",
			},
		},
		{
			name: "different language - language swap on same record",

			summaryID: uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000002"),
			language:  "en-GB",

			existing: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000002"),
					CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000002"),
				},
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("c0000000-0000-11f0-0000-000000000002"),
				Status:        summary.StatusDone,
				Language:      "en-US",
				Content:       "old content",
			},

			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{Message: openai.ChatCompletionMessage{Content: "swapped content"}},
				},
			},

			expectedLanguage: "en-GB",
			expectedContent:  "swapped content",

			updatedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000002"),
					CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000002"),
				},
				ReferenceType: summary.ReferenceTypeRecording,
				Status:        summary.StatusDone,
				Language:      "en-GB",
				Content:       "swapped content",
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
				utilHandler:         mockUtil,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				engineOpenaiHandler: mockOpenai,
			}
			ctx := context.Background()

			// 1) load target
			mockDB.EXPECT().SummaryGet(ctx, tt.summaryID).Return(tt.existing, nil)

			// 2) content regenerate (getRecordingTranscripts reuse + Send)
			setupRegenerateTranscribeReuse(ctx, mockReq, tt.existing.ReferenceID)
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(tt.responseOpenai, nil)

			// 3) in-place update (SummaryUpdate + SummaryGet + EventTypeUpdated)
			expectedFields := map[summary.Field]any{
				summary.FieldContent:  tt.expectedContent,
				summary.FieldLanguage: tt.expectedLanguage,
				summary.FieldStatus:   summary.StatusDone,
			}
			mockDB.EXPECT().SummaryUpdate(ctx, tt.existing.ID, expectedFields).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.existing.ID).Return(tt.updatedSummary, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.updatedSummary.CustomerID, summary.EventTypeUpdated, tt.updatedSummary)

			res, err := h.Regenerate(ctx, tt.summaryID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.updatedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.updatedSummary, res)
			}
		})
	}
}

// Test_Regenerate_contentFailurePreservesRecord verifies that when content
// regeneration fails (getRecordingTranscripts error), no write is performed
// (SummaryUpdate is never called) and the error is propagated -- the existing
// record is left intact. gomock's strict controller enforces the absence of
// SummaryUpdate/PublishWebhookEvent.
func Test_Regenerate_contentFailurePreservesRecord(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

	h := summaryHandler{
		db:                  mockDB,
		notifyHandler:       mockNotify,
		reqHandler:          mockReq,
		engineOpenaiHandler: mockOpenai,
	}
	ctx := context.Background()

	summaryID := uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000003")
	referenceID := uuid.FromStringOrNil("c0000000-0000-11f0-0000-000000000003")
	existing := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         summaryID,
			CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000003"),
		},
		ReferenceType: summary.ReferenceTypeRecording,
		ReferenceID:   referenceID,
		Status:        summary.StatusDone,
		Language:      "en-US",
		Content:       "old content",
	}

	mockDB.EXPECT().SummaryGet(ctx, summaryID).Return(existing, nil)
	// getRecordingTranscripts fails at the reuse lookup.
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(100), gomock.Any()).Return(nil, stderrors.New("boom"))

	res, err := h.Regenerate(ctx, summaryID, "")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_Regenerate_emptyContentPreservesRecord verifies that when content
// regeneration succeeds without a hard error but returns an empty string
// (empty LLM choices, or a non-English target whose last non-empty result is
// empty), the existing record is NOT overwritten with an empty summary. gomock's
// strict controller enforces the absence of SummaryUpdate/PublishWebhookEvent, so
// the existing good content is preserved (no data-loss window).
func Test_Regenerate_emptyContentPreservesRecord(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

	h := summaryHandler{
		db:                  mockDB,
		notifyHandler:       mockNotify,
		reqHandler:          mockReq,
		engineOpenaiHandler: mockOpenai,
	}
	ctx := context.Background()

	summaryID := uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000005")
	referenceID := uuid.FromStringOrNil("c0000000-0000-11f0-0000-000000000005")
	existing := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         summaryID,
			CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000005"),
		},
		ReferenceType: summary.ReferenceTypeRecording,
		ReferenceID:   referenceID,
		Status:        summary.StatusDone,
		Language:      "en-US",
		Content:       "old content",
	}

	// 1) load target
	mockDB.EXPECT().SummaryGet(ctx, summaryID).Return(existing, nil)
	// 2) transcripts reuse succeeds, but the LLM returns empty choices so contentGet
	//    yields ("", nil). en-US is not verified (shouldVerify=false), so generateOnce
	//    returns the empty string directly.
	setupRegenerateTranscribeReuse(ctx, mockReq, referenceID)
	mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(&openai.ChatCompletionResponse{}, nil)
	// No SummaryUpdate / SummaryGet(update) / PublishWebhookEvent expected: the guard
	// must skip the write and preserve the existing record.

	res, err := h.Regenerate(ctx, summaryID, "")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_Regenerate_notFound verifies a missing summary id propagates an error and
// never reaches content regeneration.
func Test_Regenerate_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := summaryHandler{
		db: mockDB,
	}
	ctx := context.Background()

	summaryID := uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000004")
	mockDB.EXPECT().SummaryGet(ctx, summaryID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Regenerate(ctx, summaryID, "")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_Regenerate_wrongReferenceType verifies a non-recording summary is rejected
// before any content regeneration.
func Test_Regenerate_wrongReferenceType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := summaryHandler{
		db: mockDB,
	}
	ctx := context.Background()

	summaryID := uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000005")
	existing := &summary.Summary{
		Identity: commonidentity.Identity{
			ID: summaryID,
		},
		ReferenceType: summary.ReferenceTypeCall,
		Status:        summary.StatusDone,
		Language:      "en-US",
	}
	mockDB.EXPECT().SummaryGet(ctx, summaryID).Return(existing, nil)

	res, err := h.Regenerate(ctx, summaryID, "")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_UpdateContentLanguage verifies the explicit-regenerate write path updates an
// already-done record (via the unconditional SummaryUpdate) and publishes
// EventTypeUpdated.
func Test_UpdateContentLanguage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := summaryHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("a0000000-0000-11f0-0000-000000000006")
	content := "regenerated content"
	language := "ko-KR"

	updated := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("b0000000-0000-11f0-0000-000000000006"),
		},
		ReferenceType: summary.ReferenceTypeRecording,
		Status:        summary.StatusDone,
		Language:      language,
		Content:       content,
	}

	expectedFields := map[summary.Field]any{
		summary.FieldContent:  content,
		summary.FieldLanguage: language,
		summary.FieldStatus:   summary.StatusDone,
	}
	mockDB.EXPECT().SummaryUpdate(ctx, id, expectedFields).Return(nil)
	mockDB.EXPECT().SummaryGet(ctx, id).Return(updated, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, summary.EventTypeUpdated, updated)

	res, err := h.UpdateContentLanguage(ctx, id, content, language)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, updated) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
	}
}

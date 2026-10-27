package transcribehandler

import (
	"context"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// Test_Delete_stopSucceeds checks that deleting a progressing transcribe stops its streaming
// first, marks it done, and then goes on to delete the transcripts and the transcribe itself.
func Test_Delete_stopSucceeds(t *testing.T) {
	id := uuid.FromStringOrNil("6e9f6f1e-2a2a-4a3a-9a1a-8c9a9a9a9b01")
	stID := uuid.FromStringOrNil("6e9f6f1e-2a2a-4a3a-9a1a-8c9a9a9a9b02")

	tr := &transcribe.Transcribe{
		Identity:      commonidentity.Identity{ID: id},
		Status:        transcribe.StatusProgressing,
		ReferenceType: transcribe.ReferenceTypeCall,
		StreamingIDs:  []uuid.UUID{stID},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	h := &transcribeHandler{
		reqHandler:        mockReq,
		db:                mockDB,
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
		streamingHandler:  mockStreaming,
	}
	ctx := context.Background()

	// Delete's own Get, Stop's Get, UpdateStatus's Get and dbDelete's post-delete Get
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(4)

	// the stop succeeds, so the transcribe is marked done before the deletion goes on
	mockStreaming.EXPECT().Stop(ctx, stID).Return(&streaming.Streaming{Identity: commonidentity.Identity{ID: stID}}, nil)
	mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusDone}).Return(nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, tr.CustomerID, transcribe.EventTypeTranscribeDone, tr)

	mockTranscript.EXPECT().List(ctx, uint64(1000), "", gomock.Any()).Return([]*transcript.Transcript{}, nil)
	mockDB.EXPECT().TranscribeDelete(ctx, id).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, gomock.Any())

	res, err := h.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.ID != id {
		t.Errorf("Wrong match. expect: the deleted transcribe %s, got: %v", id, res)
	}
}

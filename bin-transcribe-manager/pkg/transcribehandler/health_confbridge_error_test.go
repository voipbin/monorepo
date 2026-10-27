package transcribehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
)

// Test_HealthCheck_confbridgeGetError_stopFails_reschedules checks that a failed confbridge
// lookup makes the health check try to stop the transcribe, and that a genuine stop failure
// schedules one more health check with the retry count increased by one.
func Test_HealthCheck_confbridgeGetError_stopFails_reschedules(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000015")
	stID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000016")
	referenceID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000017")
	retryCount := 0

	tr := &transcribe.Transcribe{
		Identity:      commonidentity.Identity{ID: id},
		ReferenceType: transcribe.ReferenceTypeConfbridge,
		ReferenceID:   referenceID,
		Status:        transcribe.StatusProgressing,
		StreamingIDs:  []uuid.UUID{stID},
	}

	h := &transcribeHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		streamingHandler: mockStreaming,
	}
	ctx := context.Background()

	// once for the validation Get of HealthCheck, once more inside the Stop it triggers
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	mockReq.EXPECT().CallV1ConfbridgeGet(ctx, referenceID).Return(nil, stderrors.New("confbridge not found"))
	mockStreaming.EXPECT().Stop(ctx, stID).Return(nil, stderrors.New("could not stop the external media"))

	// the reschedule runs in its own goroutine
	rescheduled := make(chan struct{})
	mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(ctx, tr.HostID, id, defaultHealthDelay, retryCount+1).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) error {
			close(rescheduled)
			return nil
		})

	h.HealthCheck(ctx, id, retryCount)

	select {
	case <-rescheduled:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: the health check to be rescheduled")
	}
}

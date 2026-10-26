package transcribehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// Test_NewTranscribeHandler_wiring checks the constructor keeps the host id and routes reads to
// the database handler it was given.
func Test_NewTranscribeHandler_wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	hostID := uuid.FromStringOrNil("5f0e5a40-0000-11ee-8c10-000000000001")
	id := uuid.FromStringOrNil("5f0e5a40-0000-11ee-8c10-000000000002")

	h := NewTranscribeHandler(
		requesthandler.NewMockRequestHandler(mc),
		mockDB,
		notifyhandler.NewMockNotifyHandler(mc),
		transcripthandler.NewMockTranscriptHandler(mc),
		streaminghandler.NewMockStreamingHandler(mc),
		hostID,
	)

	th, ok := h.(*transcribeHandler)
	if !ok {
		t.Fatalf("NewTranscribeHandler() returned an unexpected type: %T", h)
	}
	if th.hostID != hostID {
		t.Errorf("host id mismatch. expect: %s, got: %s", hostID, th.hostID)
	}
	if th.utilHandler == nil {
		t.Errorf("utilHandler was not set")
	}

	want := &transcribe.Transcribe{}
	want.ID = id
	mockDB.EXPECT().TranscribeGet(gomock.Any(), id).Return(want, nil)

	got, err := h.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() returned an error. err: %v", err)
	}
	if got != want {
		t.Errorf("Get() did not return the database handler result")
	}
}

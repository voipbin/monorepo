package streaminghandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// Test_Delete_unknownID checks that deleting an unknown streaming neither changes the known ones
// nor publishes anything (the handler has no notify handler, so a publish would panic).
func Test_Delete_unknownID(t *testing.T) {
	knownID := uuid.FromStringOrNil("b3755cc4-e9da-11ef-812f-a73170810401")
	unknownID := uuid.FromStringOrNil("b3755cc4-e9da-11ef-812f-a73170810402")

	h := &streamingHandler{
		mapStreaming: map[uuid.UUID]*streaming.Streaming{knownID: {}},
	}

	h.Delete(context.Background(), unknownID)

	if len(h.mapStreaming) != 1 {
		t.Errorf("Wrong match. expect: 1 streaming kept, got: %d", len(h.mapStreaming))
	}
	if _, ok := h.mapStreaming[knownID]; !ok {
		t.Errorf("Wrong match. expect: the known streaming kept")
	}
}

// Test_gcpRun_notInitialized checks that a handler without a GCP client reports it instead of
// starting a session.
func Test_gcpRun_notInitialized(t *testing.T) {
	h := &streamingHandler{}

	err := h.gcpRun(&streaming.Streaming{})
	if err == nil || err.Error() != "GCP provider not initialized" {
		t.Errorf("Wrong match. expect: GCP provider not initialized, got: %v", err)
	}
}

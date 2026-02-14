package speakinghandler

import (
	"testing"

	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"go.uber.org/mock/gomock"
)

func Test_NewSpeakingHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	h := NewSpeakingHandler(mockDB, mockStreaming, "test-pod")
	if h == nil {
		t.Fatal("expected handler, got nil")
	}

	sh, ok := h.(*speakingHandler)
	if !ok {
		t.Fatal("handler is not speakingHandler type")
	}

	if sh.db == nil {
		t.Error("db should not be nil")
	}
	if sh.streamingHandler == nil {
		t.Error("streamingHandler should not be nil")
	}
	if sh.podID != "test-pod" {
		t.Errorf("expected podID test-pod, got %s", sh.podID)
	}
}

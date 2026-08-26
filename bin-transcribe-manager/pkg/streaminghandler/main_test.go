package streaminghandler

import (
	"context"
	"os"
	"testing"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	gomock "go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func Test_NewStreamingHandler_AWSOnly(t *testing.T) {
	// This test verifies service works with only AWS credentials
	// GCP will fail to initialize (no credentials in test env)
	// AWS should succeed with valid test credentials

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	handler := NewStreamingHandler(
		mockReq,
		mockNotify,
		mockTranscript,
		"test_access_key",
		"test_secret_key",
	)

	// Should return valid handler (AWS initialized, GCP not in priority list)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}

// Test_NewStreamingHandler_NoProviders verifies that NewStreamingHandler never returns a nil
// interface. Startup used to abort when neither provider could be initialized; it now degrades
// the streaming transcribe path instead (see Test_NewDisabledStreamingHandler).
//
// Whether GCP actually initializes here depends on the ambient application-default
// credentials, so this only asserts the constructor contract that holds either way.
func Test_NewStreamingHandler_NoProviders(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	handler := NewStreamingHandler(
		mockReq,
		mockNotify,
		mockTranscript,
		"", // empty AWS credentials
		"",
	)

	if handler == nil {
		t.Fatal("Expected handler to be non-nil when no providers available")
	}
}

// Test_NewDisabledStreamingHandler verifies the STT-unavailable degradation contract:
// Run() must be a no-op so the service finishes booting, and the per-request methods must
// report ErrSTTNotConfigured instead of silently doing nothing.
func Test_NewDisabledStreamingHandler(t *testing.T) {
	handler := NewDisabledStreamingHandler()

	if handler == nil {
		t.Fatal("Expected handler to be non-nil")
	}

	if err := handler.Run(); err != nil {
		t.Errorf("Run() should be a no-op when STT is disabled, got error: %v", err)
	}

	if _, err := handler.Start(context.Background(), uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty); err == nil {
		t.Error("Expected an STT-not-configured error from Start(), got nil")
	} else if ve, ok := err.(*cerrors.VoipbinError); !ok || ve.Reason != ErrSTTNotConfiguredReason {
		t.Errorf("Wrong error type/reason. expect reason: %s, got: %v", ErrSTTNotConfiguredReason, err)
	}

	if _, err := handler.Stop(context.Background(), uuid.Nil); err == nil {
		t.Error("Expected an STT-not-configured error from Stop(), got nil")
	} else if ve, ok := err.(*cerrors.VoipbinError); !ok || ve.Reason != ErrSTTNotConfiguredReason {
		t.Errorf("Wrong error type/reason. expect reason: %s, got: %v", ErrSTTNotConfiguredReason, err)
	}
}

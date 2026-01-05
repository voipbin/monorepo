package streaminghandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	gomock "go.uber.org/mock/gomock"
)

func TestNewStreamingHandler_AWSOnly(t *testing.T) {
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
		"127.0.0.1:8080",
		"test_access_key",
		"test_secret_key",
	)

	// Should return valid handler (AWS initialized, GCP may fail gracefully)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}

func TestNewStreamingHandler_NoProviders(t *testing.T) {
	// This test verifies service fails when neither provider is available
	// GCP will fail (no credentials in test env)
	// AWS will fail (empty credentials)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	handler := NewStreamingHandler(
		mockReq,
		mockNotify,
		mockTranscript,
		"127.0.0.1:8080",
		"", // empty AWS credentials
		"",
	)

	// Should return nil when no providers available
	if handler != nil {
		t.Error("Expected handler to be nil when no providers available")
	}
}

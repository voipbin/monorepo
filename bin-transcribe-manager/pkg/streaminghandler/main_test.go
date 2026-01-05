package streaminghandler

import (
	"os"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	"github.com/spf13/cobra"
	gomock "go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	// Set STT provider priority for tests
	// Use AWS only since GCP credentials are not available in test environment
	os.Setenv("STT_PROVIDER_PRIORITY", "AWS")

	// Initialize config - required for NewStreamingHandler
	cmd := &cobra.Command{}
	if err := config.Bootstrap(cmd); err != nil {
		panic(err)
	}
	config.LoadGlobalConfig()

	os.Exit(m.Run())
}

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

	// Should return valid handler (AWS initialized, GCP not in priority list)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}

func TestNewStreamingHandler_NoProviders(t *testing.T) {
	// This test verifies service fails when neither provider is available
	// Priority is set to AWS in TestMain
	// AWS will fail (empty credentials provided to NewStreamingHandler)

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

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
	if errSet := os.Setenv("STT_PROVIDER_PRIORITY", "AWS"); errSet != nil {
		panic(errSet)
	}

	// Initialize config - required for NewStreamingHandler
	cmd := &cobra.Command{}
	if err := config.Bootstrap(cmd); err != nil {
		panic(err)
	}
	config.LoadGlobalConfig()

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
		"127.0.0.1:8080",
		"test_access_key",
		"test_secret_key",
	)

	// Should return valid handler (AWS initialized, GCP not in priority list)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}

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
		"127.0.0.1:8080",
		"", // empty AWS credentials
		"",
	)

	if handler != nil {
		t.Error("Expected handler to be nil when no providers available")
	}
}

func Test_NewStreamingHandler_DuplicateProviders(t *testing.T) {
	testCases := []struct {
		name     string
		priority string
	}{
		{
			name:     "duplicate AWS",
			priority: "AWS,AWS",
		},
		{
			name:     "triple AWS",
			priority: "AWS,AWS,AWS",
		},
		{
			name:     "mixed duplicates with case variations",
			priority: "aws,AWS,Aws",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if errSet := os.Setenv("STT_PROVIDER_PRIORITY", tc.priority); errSet != nil {
				t.Fatalf("Failed to set env: %v", errSet)
			}

			config.LoadGlobalConfig()

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

			if handler == nil {
				t.Errorf("Expected handler to be non-nil with priority '%s'", tc.priority)
			}
		})
	}
}

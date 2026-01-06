package streaminghandler

import (
	"os"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
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

func Test_initProviders(t *testing.T) {
	testCases := []struct {
		name          string
		priorityList  []string
		gcpClient     interface{}
		awsClient     interface{}
		expectError   bool
		expectedCount int
	}{
		{
			name:          "valid single provider",
			priorityList:  []string{"AWS"},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:          "valid two providers",
			priorityList:  []string{"GCP", "AWS"},
			gcpClient:     "mock_gcp_client",
			awsClient:     "mock_aws_client",
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "duplicate AWS",
			priorityList:  []string{"AWS", "AWS"},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "triple AWS",
			priorityList:  []string{"AWS", "AWS", "AWS"},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "case variation duplicates",
			priorityList:  []string{"aws", "AWS"},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "mixed case variation duplicates",
			priorityList:  []string{"gcp", "Gcp", "GCP"},
			gcpClient:     "mock_gcp_client",
			awsClient:     nil,
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "provider not initialized",
			priorityList:  []string{"AWS"},
			gcpClient:     nil,
			awsClient:     nil,
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "trailing comma empty string",
			priorityList:  []string{"AWS", ""},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "leading comma empty string",
			priorityList:  []string{"", "AWS"},
			gcpClient:     nil,
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "double comma empty string",
			priorityList:  []string{"GCP", "", "AWS"},
			gcpClient:     "mock_gcp_client",
			awsClient:     "mock_aws_client",
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:          "empty priority list",
			priorityList:  []string{},
			gcpClient:     nil,
			awsClient:     nil,
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gcpClient *speech.Client
			var awsClient *transcribestreaming.Client

			if tc.gcpClient != nil {
				gcpClient = &speech.Client{}
			}
			if tc.awsClient != nil {
				awsClient = &transcribestreaming.Client{}
			}

			result, err := initProviders(tc.priorityList, gcpClient, awsClient)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for priority list %v, got nil", tc.priorityList)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for priority list %v: %v", tc.priorityList, err)
				}
				if len(result) != tc.expectedCount {
					t.Errorf("Expected %d providers, got %d", tc.expectedCount, len(result))
				}
			}
		})
	}
}

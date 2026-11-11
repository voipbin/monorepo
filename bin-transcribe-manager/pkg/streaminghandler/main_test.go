package streaminghandler

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
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
		pairKeys("test_access_key", "test_secret_key"),
	)

	// Should return valid handler (AWS initialized, GCP not in priority list)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}

// Test_NewStreamingHandler_NoProviders verifies that NewStreamingHandler never returns a nil
// interface and that it does not decide at start up that streaming is off: the handler is the real
// one, Run is a no-op, and Start answers STT_NOT_CONFIGURED while no provider is available.
//
// Whether GCP actually initializes here depends on the ambient application-default
// credentials, so the test removes them.
func Test_NewStreamingHandler_NoProviders(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	handler := NewStreamingHandler(
		mockReq,
		mockNotify,
		mockTranscript,
		pairKeys("", ""), // empty AWS credentials
	)

	if handler == nil {
		t.Fatal("Expected handler to be non-nil when no providers available")
	}
	if _, ok := handler.(*streamingHandler); !ok {
		t.Fatalf("Wrong match. expect: the real handler, got: %T", handler)
	}

	if err := handler.Run(); err != nil {
		t.Errorf("Run() must stay a no-op, got error: %v", err)
	}

	// the request handler has no expectation: nothing may be created when there is no provider
	if _, err := handler.Start(context.Background(), uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty); !isReason(err, ErrSTTNotConfiguredReason) {
		t.Errorf("Wrong match. expect reason: %s, got: %v", ErrSTTNotConfiguredReason, err)
	}
}

// pairKeys returns a supplier that always hands out the given pair.
func pairKeys(accessKey, secretKey string) AWSKeyFunc {
	return func(context.Context) (string, string, error) {
		return accessKey, secretKey, nil
	}
}

// isReason tells whether err is a VoipbinError with the reason.
func isReason(err error, reason string) bool {
	var ve *cerrors.VoipbinError
	return stderrors.As(err, &ve) && ve.Reason == reason
}

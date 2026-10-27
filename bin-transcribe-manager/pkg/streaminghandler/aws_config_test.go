package streaminghandler

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// This file must not import net/http: the bodyclose linter then analyses the package's websocket
// dial. The tests that need an HTTP server live in the external test package.

func Test_awsNewClient_configFailure(t *testing.T) {
	t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "missing-ca.pem"))
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	res, err := awsNewClient("test_access_key", "test_secret_key")
	if err == nil || !strings.Contains(err.Error(), "failed to load AWS SDK v2 config") {
		t.Fatalf("Wrong match. expect: the config load error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil client, got: %v", res)
	}
}

func Test_NewStreamingHandler_awsConfigFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "missing-ca.pem"))
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))

	res := NewStreamingHandler(
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		transcripthandler.NewMockTranscriptHandler(mc),
		"test_access_key",
		"test_secret_key",
	)

	// the AWS provider is dropped when its client cannot be built, and GCP is not configured either
	if got, want := fmt.Sprintf("%T", res), fmt.Sprintf("%T", NewDisabledStreamingHandler()); got != want {
		t.Errorf("Wrong match. expect: %s, got: %s", want, got)
	}
}

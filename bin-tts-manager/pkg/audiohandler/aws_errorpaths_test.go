package audiohandler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_awsGetClient_error verifies the error-wrapping branch when the AWS
// SDK cannot resolve the configured shared-config profile, with no network
// call. The shared config lookup is isolated to a temp directory.
func Test_awsGetClient_error(t *testing.T) {
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_PROFILE", "does-not-exist")

	_, err := awsGetClient(context.Background(), "test-access-key", "test-secret-key")
	if err == nil || !strings.Contains(err.Error(), "failed to create AWS session") {
		t.Fatalf("Wrong error. expected to mention AWS session creation, got: %v", err)
	}
}

// Test_savePCMAsWAV_headerWriteError verifies the WAV header write-error
// branch using /dev/full, which accepts opens and writes but always reports
// ENOSPC. The test is skipped where /dev/full is unavailable (non-Linux).
func Test_savePCMAsWAV_headerWriteError(t *testing.T) {
	const devFull = "/dev/full"
	f, err := os.OpenFile(devFull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("skipping: %s is not available on this host: %v", devFull, err)
	}
	_ = f.Close()

	h := &audioHandler{}

	err = h.savePCMAsWAV(strings.NewReader("pcm"), devFull, 8000, 1)
	if err == nil || !strings.Contains(err.Error(), "failed to write WAV header") {
		t.Fatalf("Wrong error. expected a WAV header write error, got: %v", err)
	}
}

package ttshandler

import (
	"context"
	"path/filepath"
	"testing"
)

func Test_NewTTSHandler_missingGCPCredentials(t *testing.T) {
	// the credential file does not exist, so the audio handler cannot be created and no handler is returned
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))

	res := NewTTSHandler(func(context.Context) (string, string, error) { return "fake-access", "fake-secret", nil }, "unused", t.TempDir(), "localhost", nil)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

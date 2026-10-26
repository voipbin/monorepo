package servicehandler

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Test_NewServiceHandler_malformedCredential_keepsConfigWithoutClient checks that
// an unusable credential degrades only the storage client: the handler is still
// built with its directory and bucket settings.
func Test_NewServiceHandler_malformedCredential_keepsConfigWithoutClient(t *testing.T) {
	// a storage emulator would let the client be built without parsing the credential.
	t.Setenv("STORAGE_EMULATOR_HOST", "")

	// precondition: the credential really is rejected by the storage client, so the
	// nil client below is caused by it and not by something else.
	if _, errClient := storage.NewClient(context.Background(), option.WithAuthCredentialsJSON(option.ServiceAccount, []byte("{not json"))); errClient == nil {
		t.Fatalf("Wrong precondition. expect: the storage client to reject the malformed credential, got: nil")
	}

	got := NewServiceHandler("{not json", "test-bucket", "/test/asterisk", "test-bucket-dir")

	h, ok := got.(*serviceHandler)
	if !ok {
		t.Fatalf("Wrong type. expect: *serviceHandler, got: %T", got)
	}
	if h.client != nil {
		t.Errorf("Wrong client. expect: nil for a malformed credential, got a client")
	}
	if h.recordingBucketName != "test-bucket" || h.recordingAsteriskDirectory != "/test/asterisk" || h.recordingBucketDirectory != "test-bucket-dir" {
		t.Errorf("Wrong settings. bucket: %s, asterisk dir: %s, bucket dir: %s", h.recordingBucketName, h.recordingAsteriskDirectory, h.recordingBucketDirectory)
	}
}

// Test_recordingFileUpload_missingSourceFile checks that a source file that does
// not exist fails before the storage client is touched (the client is nil here).
func Test_recordingFileUpload_missingSourceFile(t *testing.T) {
	dir := t.TempDir()

	h := &serviceHandler{
		recordingBucketName:        "test-bucket",
		recordingAsteriskDirectory: dir,
		recordingBucketDirectory:   "test-bucket-dir",
	}

	err := h.recordingFileUpload(context.Background(), "missing.wav")
	if err == nil {
		t.Fatalf("Wrong match. expect: an error, got: nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Wrong error. expect: not-exist, got: %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(dir, "missing.wav")) {
		t.Errorf("Wrong message. expect: the source path, got: %v", err)
	}
}

package pcapwatcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

// replaceWithNonEmptyDir swaps a file for a non-empty directory, so a later
// os.Remove fails with ENOTEMPTY whatever the uid of the test process is.
func replaceWithNonEmptyDir(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Errorf("could not remove %s: %v", path, err)
	}
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o755); err != nil {
		t.Errorf("could not create directory %s: %v", path, err)
	}
}

func Test_scanAndProcess_processingRetryFails(t *testing.T) {
	logs := captureLogs(t)

	metadataDir := filepath.Join(t.TempDir(), "metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("could not create metadata dir: %v", err)
	}
	// whitespace only: the content is rejected as empty before any upload
	processing := filepath.Join(metadataDir, "stuck.processing")
	if err := os.WriteFile(processing, []byte("\n"), 0o600); err != nil {
		t.Fatalf("could not write fixture: %v", err)
	}

	w := &watcher{recordingDir: filepath.Dir(metadataDir)}
	w.scanAndProcess(metadataDir)

	if logs.count(msgIs(logrus.WarnLevel, "failed to retry processing file", "file", "stuck.processing")) != 1 {
		t.Errorf("Wrong match. expected one retry failure log")
	}
	if _, err := os.Stat(processing); !os.IsNotExist(err) {
		t.Errorf("Wrong match. invalid processing file must be removed, stat err: %v", err)
	}
}

func Test_processProcessingFile_cleanupFailures(t *testing.T) {
	tests := []struct {
		name string

		breakPcap     bool // otherwise the processing file is replaced
		expectMessage string
	}{
		{
			name:          "pcap cannot be deleted",
			breakPcap:     true,
			expectMessage: "could not delete pcap file",
		},
		{
			name:          "processing file cannot be deleted",
			breakPcap:     false,
			expectMessage: "could not delete processing file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)

			mc := gomock.NewController(t)
			defer mc.Finish()

			dir := t.TempDir()
			metadataDir := filepath.Join(dir, "metadata")
			if err := os.MkdirAll(metadataDir, 0o755); err != nil {
				t.Fatalf("could not create metadata dir: %v", err)
			}
			pcap := writePcap(t, dir, "call.pcap")
			processing := filepath.Join(metadataDir, "call.processing")
			if err := os.WriteFile(processing, []byte(pcap+"\n"), 0o600); err != nil {
				t.Fatalf("could not write fixture: %v", err)
			}

			broken := processing
			if tt.breakPcap {
				broken = pcap
			}
			mockUploader := gcsuploader.NewMockUploader(mc)
			mockUploader.EXPECT().Upload(pcap, "rtp-recordings/call.pcap").DoAndReturn(func(string, string) (string, error) {
				replaceWithNonEmptyDir(t, broken)
				return "gs://b/call.pcap", nil
			}).Times(1)

			w := &watcher{recordingDir: dir, uploader: mockUploader}
			if err := w.processProcessingFile(processing); err != nil {
				t.Fatalf("Wrong match. expected: nil, got: %v", err)
			}

			if logs.count(msgIs(logrus.WarnLevel, tt.expectMessage, "file", broken)) != 1 {
				t.Errorf("Wrong match. expected one %q log", tt.expectMessage)
			}
		})
	}
}

func Test_Run_metadataDirRemoved(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	dir := t.TempDir()
	metadataDir := filepath.Join(dir, "metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("could not create metadata dir: %v", err)
	}
	pcap := writePcap(t, dir, "early.pcap") // outside metadata/, so it survives the removal
	stageMetadata(t, dir, "early", pcap)

	// the startup scan uploads first; removing the directory there makes the
	// following watch registration fail
	mockUploader := gcsuploader.NewMockUploader(mc)
	mockUploader.EXPECT().Upload(pcap, "rtp-recordings/early.pcap").DoAndReturn(func(string, string) (string, error) {
		if err := os.RemoveAll(metadataDir); err != nil {
			t.Errorf("could not remove metadata dir: %v", err)
		}
		return "gs://b/early.pcap", nil
	}).Times(1)

	err := New(dir, mockUploader).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "could not watch metadata dir") {
		t.Errorf("Wrong match. expected: watch error, got: %v", err)
	}
}

func Test_Run_processFailureWarning(t *testing.T) {
	logs := captureLogs(t)

	dir := t.TempDir()
	metadataDir := filepath.Join(dir, "metadata")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	// registered after TempDir and the hook, so it runs first on cleanup
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("Run did not return after cancel")
		}
	})
	go func() { done <- New(dir, nil).Run(ctx) }()

	// Run has no readiness signal and an entry created before the watch is
	// registered is not reported, so create entries until one is processed.
	// A directory passes the rename and then fails to be read.
	warned := func(e capturedEntry) bool {
		return e.level == logrus.WarnLevel && e.message == "failed to process metadata file" &&
			strings.HasPrefix(filepath.ToSlash(e.fields["file"].(string)), filepath.ToSlash(metadataDir))
	}
	deadline := time.Now().Add(10 * time.Second)
	for i := 0; logs.count(warned) == 0; i++ {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the metadata processing warning")
		}
		_ = os.Mkdir(filepath.Join(metadataDir, "entry-"+time.Now().Format("150405.000000000")), 0o755)
		time.Sleep(20 * time.Millisecond)
	}
}

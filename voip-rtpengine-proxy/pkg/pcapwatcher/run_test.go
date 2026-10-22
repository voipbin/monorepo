package pcapwatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

// stageMetadata writes a metadata file outside the watched directory and moves
// it in with a rename, the way RTPEngine hands recordings over. Writing in
// place would let the watcher see the file before its content is written.
func stageMetadata(t *testing.T, recordingDir, name, pcapPath string) {
	t.Helper()

	stage := filepath.Join(recordingDir, "tmp")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatalf("could not create staging dir: %v", err)
	}
	staged := filepath.Join(stage, name)
	if err := os.WriteFile(staged, []byte(pcapPath+"\n\nSDP mode: offer\n"), 0o600); err != nil {
		t.Fatalf("could not stage metadata: %v", err)
	}
	if err := os.Rename(staged, filepath.Join(recordingDir, "metadata", name)); err != nil {
		t.Fatalf("could not move metadata into place: %v", err)
	}
}

func writePcap(t *testing.T, dir, name string) string {
	t.Helper()

	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("synthetic pcap"), 0o600); err != nil {
		t.Fatalf("could not write pcap: %v", err)
	}
	return p
}

func Test_processProcessingFile(t *testing.T) {
	tests := []struct {
		name string

		metadata     func(t *testing.T, dir string) string // returns the .processing path
		uploadErr    error
		expectUpload bool

		expectErr             bool
		expectProcessingExist bool
	}{
		{
			name: "empty metadata is dropped",

			metadata: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "a.processing")
				_ = os.WriteFile(p, []byte("  \n"), 0o600)
				return p
			},

			expectErr:             true,
			expectProcessingExist: false,
		},
		{
			name: "unreadable metadata is reported and left in place",

			metadata: func(t *testing.T, dir string) string {
				// A directory cannot be read as a file, with or without root.
				p := filepath.Join(dir, "a.processing")
				_ = os.Mkdir(p, 0o755)
				return p
			},

			expectErr:             true,
			expectProcessingExist: true,
		},
		{
			name: "failed upload keeps the metadata for a later retry",

			metadata: func(t *testing.T, dir string) string {
				pcap := writePcap(t, dir, "call.pcap")
				p := filepath.Join(dir, "a.processing")
				_ = os.WriteFile(p, []byte(pcap+"\n"), 0o600)
				return p
			},
			uploadErr:    errors.New("injected upload failure"),
			expectUpload: true,

			expectErr:             true,
			expectProcessingExist: true,
		},
		{
			name: "pcap missing after upload is tolerated",

			metadata: func(t *testing.T, dir string) string {
				p := filepath.Join(dir, "a.processing")
				_ = os.WriteFile(p, []byte(filepath.Join(dir, "gone.pcap")+"\n"), 0o600)
				return p
			},
			expectUpload: true,

			expectErr:             false,
			expectProcessingExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUploader := gcsuploader.NewMockUploader(mc)
			if tt.expectUpload {
				mockUploader.EXPECT().Upload(gomock.Any(), gomock.Any()).Return("", tt.uploadErr).Times(1)
			}

			dir := t.TempDir()
			path := tt.metadata(t, dir)
			w := &watcher{recordingDir: dir, uploader: mockUploader}

			err := w.processProcessingFile(path)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if _, statErr := os.Stat(path); (statErr == nil) != tt.expectProcessingExist {
				t.Errorf("Wrong match. expect processing file exists: %v, got: %v", tt.expectProcessingExist, statErr == nil)
			}
		})
	}
}

func Test_processMetadataFile_missing(t *testing.T) {
	w := &watcher{recordingDir: t.TempDir()}

	if err := w.processMetadataFile(filepath.Join(w.recordingDir, "absent")); err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}

func Test_scanAndProcess_retriesAndSkips(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	dir := t.TempDir()
	metadataDir := filepath.Join(dir, "metadata")
	_ = os.MkdirAll(metadataDir, 0o755)

	// A leftover .processing file from a crashed run is retried.
	pcapRetry := writePcap(t, dir, "retry.pcap")
	retry := filepath.Join(metadataDir, "retry.processing")
	_ = os.WriteFile(retry, []byte(pcapRetry+"\n"), 0o600)

	// A broken fresh file is dropped; a sub directory is ignored.
	broken := filepath.Join(metadataDir, "broken")
	_ = os.WriteFile(broken, []byte("\n"), 0o600)
	subDir := filepath.Join(metadataDir, "subdir")
	_ = os.Mkdir(subDir, 0o755)

	mockUploader := gcsuploader.NewMockUploader(mc)
	mockUploader.EXPECT().Upload(pcapRetry, "rtp-recordings/retry.pcap").Return("gs://b/x", nil).Times(1)

	w := &watcher{recordingDir: dir, uploader: mockUploader}
	w.scanAndProcess(metadataDir)

	if _, err := os.Stat(retry); !os.IsNotExist(err) {
		t.Errorf("Wrong match. retried processing file must be removed")
	}
	if _, err := os.Stat(pcapRetry); !os.IsNotExist(err) {
		t.Errorf("Wrong match. retried pcap must be removed")
	}
	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Errorf("Wrong match. broken metadata must be consumed by the scan")
	}
	if _, err := os.Stat(broken + ".processing"); !os.IsNotExist(err) {
		t.Errorf("Wrong match. broken metadata must be dropped, not left to retry forever")
	}
	if _, err := os.Stat(subDir); err != nil {
		t.Errorf("Wrong match. sub directory must be left alone: %v", err)
	}
}

func Test_scanAndProcess_missingDir(t *testing.T) {
	w := &watcher{recordingDir: t.TempDir()}

	w.scanAndProcess(filepath.Join(w.recordingDir, "does-not-exist")) // must not panic
}

func Test_Run_cannotCreateMetadataDir(t *testing.T) {
	// recordingDir is a regular file, so metadata/ cannot be created under it.
	file := filepath.Join(t.TempDir(), "recordings")
	_ = os.WriteFile(file, []byte("x"), 0o600)

	w := New(file, nil)
	if err := w.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "metadata dir") {
		t.Errorf("Wrong match. expect: metadata dir error, got: %v", err)
	}
}

func Test_Run_startupScanAndShutdown(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "metadata"), 0o755)
	pcap := writePcap(t, dir, "early.pcap")
	stageMetadata(t, dir, "early", pcap)

	uploaded := make(chan struct{}, 1)
	mockUploader := gcsuploader.NewMockUploader(mc)
	mockUploader.EXPECT().Upload(pcap, "rtp-recordings/early.pcap").DoAndReturn(func(string, string) (string, error) {
		select {
		case uploaded <- struct{}{}:
		default:
		}
		return "gs://b/x", nil
	}).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- New(dir, mockUploader).Run(ctx) }()

	select {
	case <-uploaded:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("startup scan did not upload the pre-existing recording")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wrong match. expect: nil on context cancel, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	info, err := os.Stat(filepath.Join(dir, "metadata"))
	if err != nil {
		t.Fatalf("metadata dir missing: %v", err)
	}
	if info.Mode().Perm() != 0o777 {
		t.Errorf("Wrong match. expect: 0777, got: %o", info.Mode().Perm())
	}
}

func Test_Run_picksUpNewMetadataFile(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	dir := t.TempDir()

	var warm, probe atomic.Int32
	probeSeen := make(chan struct{}, 1)
	mockUploader := gcsuploader.NewMockUploader(mc)
	mockUploader.EXPECT().Upload(gomock.Any(), gomock.Any()).DoAndReturn(func(local, remote string) (string, error) {
		if strings.HasPrefix(filepath.Base(remote), "probe-") {
			probe.Add(1)
			select {
			case probeSeen <- struct{}{}:
			default:
			}
		} else {
			warm.Add(1)
		}
		return "gs://b/x", nil
	}).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- New(dir, mockUploader).Run(ctx) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("Run did not return after cancel")
		}
	}()

	// Run creates metadata/ itself; wait for it before moving files in.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "metadata")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("metadata dir was never created")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// There is no "watcher is ready" signal, so keep feeding uniquely named
	// recordings until one handed over after startup is uploaded. Phase 1 may be
	// consumed by the startup scan; probes (phase 2) only ever arrive via the
	// file system event path or the 5 minute rescan, so seeing one proves events.
	inject := func(prefix string, seq int) {
		name := fmt.Sprintf("%s-%s-%d", prefix, time.Now().Format("150405.000000"), seq)
		pcap := writePcap(t, dir, name+".pcap")
		stageMetadata(t, dir, name, pcap)
	}

	deadline = time.Now().Add(15 * time.Second)
	for i := 0; warm.Load() == 0; i++ {
		if time.Now().After(deadline) {
			t.Fatal("no recording was uploaded")
		}
		inject("warm", i)
		time.Sleep(50 * time.Millisecond)
	}

	deadline = time.Now().Add(15 * time.Second)
	for i := 0; ; i++ {
		select {
		case <-probeSeen:
			if probe.Load() == 0 {
				t.Fatal("probe signal without a probe upload")
			}
			return
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("watcher never reacted to a recording moved in after startup")
		}
		inject("probe", i)
		time.Sleep(50 * time.Millisecond)
	}
}

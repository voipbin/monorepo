package processmanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

// injectFinished registers a tracked process whose command has already exited
// and been waited for. The timer is never nil because killAndUpload always
// stops it; it is stopped again when the test ends.
func injectFinished(t *testing.T, m *manager, id, pcapPath string) {
	t.Helper()

	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("could not run helper command: %v", err)
	}
	timer := time.AfterFunc(time.Hour, func() {})
	t.Cleanup(func() { timer.Stop() })

	m.mu.Lock()
	m.processes[id] = &trackedProcess{cmd: cmd, pcapPath: pcapPath, timer: timer}
	m.mu.Unlock()
}

func Test_safetyTimeout_killFails(t *testing.T) {
	logs := captureLogs(t)
	installFakeTcpdump(t, fakeTcpdumpNormal)

	id := uuid.NewString()
	m := newTestManager(nil)
	if err := m.Exec(id, "tcpdump", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	m.mu.Lock()
	tracked := m.processes[id]
	delete(m.processes, id) // the timer now finds no entry to kill
	m.mu.Unlock()
	t.Cleanup(func() {
		_ = tracked.cmd.Process.Kill()
		_ = tracked.cmd.Wait()
		_ = os.Remove(tracked.pcapPath)
	})
	waitFor(t, "fake capture to create its pcap", func() bool { return fileExists(tracked.pcapPath) })

	// the manager timeout is one hour: fire the real timer now
	tracked.timer.Reset(time.Millisecond)

	logs.waitFor(t, "safety timeout kill failure", msgIs(logrus.ErrorLevel, "Safety timeout kill failed for "+id))
}

func Test_Kill_deleteFailures(t *testing.T) {
	tests := []struct {
		name string

		pcapIsNonEmptyDir bool
		expectPrefix      string
		expectContains    string
	}{
		{
			name:           "already finished process cannot be signalled",
			expectPrefix:   "Could not send SIGTERM",
			expectContains: "process already finished",
		},
		{
			name:              "local pcap cannot be deleted",
			pcapIsNonEmptyDir: true,
			expectPrefix:      "Could not delete local pcap",
			expectContains:    "directory not empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)

			id := uuid.NewString()
			pcap := filepath.Join(t.TempDir(), id+".pcap")
			if tt.pcapIsNonEmptyDir {
				if err := os.MkdirAll(filepath.Join(pcap, "child"), 0o755); err != nil {
					t.Fatalf("could not create fixture: %v", err)
				}
			}

			m := newTestManager(nil)
			injectFinished(t, m, id, pcap)

			got, err := m.Kill(id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got != pcap {
				t.Errorf("Wrong match. expect: %s, got: %s", pcap, got)
			}
			if logs.count(msgPrefix(logrus.WarnLevel, tt.expectPrefix, tt.expectContains)) != 1 {
				t.Errorf("Wrong match. expected one warn log %q containing %q", tt.expectPrefix, tt.expectContains)
			}
			if m.Count() != 0 {
				t.Errorf("Wrong match. process must be dropped, count: %d", m.Count())
			}
		})
	}
}

func Test_Shutdown_killFails(t *testing.T) {
	logs := captureLogs(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	const total = 3
	m := NewManager("eth9", time.Hour, nil).(*manager)
	dir := t.TempDir()
	for i := 0; i < total; i++ {
		injectFinished(t, m, fmt.Sprintf("id-%d", i), filepath.Join(dir, fmt.Sprintf("%d.pcap", i)))
	}

	// The first upload drops every other entry while Shutdown still holds the
	// id snapshot, so each remaining id fails with "no process". Only the
	// first call acts, so the outcome does not depend on the map order. A
	// sync.Once here would deadlock on re-entry.
	var acted atomic.Bool
	mockUploader := gcsuploader.NewMockUploader(mc)
	mockUploader.EXPECT().Upload(gomock.Any(), gomock.Any()).DoAndReturn(func(local, _ string) (string, error) {
		if acted.CompareAndSwap(false, true) {
			m.mu.Lock()
			for id, p := range m.processes {
				if p.pcapPath != local {
					delete(m.processes, id)
				}
			}
			m.mu.Unlock()
		}
		return "gs://b/x", nil
	}).Times(1)
	m.uploader = mockUploader

	m.Shutdown()

	warned := func(e capturedEntry) bool {
		return msgPrefix(logrus.WarnLevel, "Shutdown: kill id-", "no process with id")(e)
	}
	if got := logs.count(warned); got != total-1 {
		t.Errorf("Wrong match. expected %d shutdown kill warnings, got: %d", total-1, got)
	}
}

func Test_CleanOrphans_deleteFails(t *testing.T) {
	// Same guard as Test_CleanOrphans: CleanOrphans deletes every uuid-named
	// pcap in the shared /tmp. A skip here loses the covered statement, so the
	// run is checked for SKIP lines.
	others, _ := filepath.Glob("/tmp/*.pcap")
	for _, f := range others {
		if uuidPattern.MatchString(filepath.Base(f)) {
			t.Skipf("foreign uuid pcap present in /tmp (%s); skipping to avoid deleting it", filepath.Base(f))
		}
	}

	logs := captureLogs(t)

	orphan := "/tmp/" + uuid.NewString() + ".pcap"
	if err := os.MkdirAll(filepath.Join(orphan, "child"), 0o755); err != nil {
		t.Fatalf("could not create fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(orphan) })

	newTestManager(nil).CleanOrphans()

	if logs.count(msgPrefix(logrus.WarnLevel, "Could not delete orphaned pcap "+orphan, "directory not empty")) != 1 {
		t.Errorf("Wrong match. expected one orphan delete warning")
	}
	if !fileExists(orphan) {
		t.Errorf("Wrong match. non-empty directory must remain")
	}
}

package processmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

// TestMain serializes concurrent runs of this package on one host (other
// worktrees, CI jobs) with an exclusive lock: the manager hardcodes the shared
// /tmp/<uuid>.pcap namespace and CleanOrphans deletes every uuid-named pcap
// there, so two overlapping runs would delete each other's captures.
func TestMain(m *testing.M) {
	// The lock lives in /tmp, not os.TempDir(): it must be the same file for
	// every run, exactly like the /tmp namespace it protects.
	lock, err := os.OpenFile("/tmp/voip-rtpengine-proxy-processmanager-test.lock", os.O_CREATE|os.O_RDWR, 0o666)
	if err == nil {
		// Bounded wait, so a stuck holder cannot hang this run silently. A lock
		// file owned by another user cannot be opened (fs.protected_regular),
		// which only protects runs of the same user; that is the known limit.
		deadline := time.Now().Add(60 * time.Second)
		for {
			err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not take the cross-run test lock, concurrent runs may interfere: %v\n", err)
	}

	code := m.Run()
	if lock != nil {
		_ = lock.Close() // releases the lock
	}
	os.Exit(code)
}

// Tests in this package must not use t.Parallel(): they share /tmp (the manager
// hardcodes /tmp/<uuid>.pcap) and replace PATH with t.Setenv.

const (
	// fakeTcpdumpNormal records its argv, creates the pcap named after -w, then
	// idles until signalled. exec keeps the PID and leaves no orphan child.
	fakeTcpdumpNormal = `#!/bin/sh
f=""
for a in "$@"; do
  if [ "$prev" = "-w" ]; then f="$a"; fi
  prev="$a"
done
if [ -n "$FAKE_ARGS_FILE" ]; then echo "$@" > "$FAKE_ARGS_FILE"; fi
: > "$f"
exec sleep 60
`

	// fakeTcpdumpIgnoreTerm ignores SIGTERM (the ignore disposition survives exec)
	// and signals readiness through FAKE_READY_FILE once the trap is installed.
	fakeTcpdumpIgnoreTerm = `#!/bin/sh
trap '' TERM
f=""
for a in "$@"; do
  if [ "$prev" = "-w" ]; then f="$a"; fi
  prev="$a"
done
: > "$f"
: > "$FAKE_READY_FILE"
exec sleep 60
`
)

// installFakeTcpdump puts a fake tcpdump with the given script first on PATH.
func installFakeTcpdump(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tcpdump"), []byte(script), 0o755); err != nil {
		t.Fatalf("could not write fake tcpdump: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// waitFor polls cond until it is true or the timeout expires.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// signalOnce returns a func that closes ch only on the first call, so a mock
// invoked several times (retry) cannot panic on a double close.
func signalOnce(ch chan struct{}) func() {
	var once sync.Once
	return func() { once.Do(func() { close(ch) }) }
}

func newTestManager(uploader gcsuploader.Uploader) *manager {
	return NewManager("eth9", time.Hour, uploader).(*manager)
}

func Test_NewManager(t *testing.T) {
	m := NewManager("eth9", 3*time.Minute, nil).(*manager)

	if m.interfaceName != "eth9" {
		t.Errorf("Wrong match. expect: eth9, got: %s", m.interfaceName)
	}
	if m.safetyTimeout != 3*time.Minute {
		t.Errorf("Wrong match. expect: 3m, got: %v", m.safetyTimeout)
	}
	if m.maxConcurrent != 20 {
		t.Errorf("Wrong match. expect: 20, got: %d", m.maxConcurrent)
	}
	if m.Count() != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", m.Count())
	}
}

func Test_Exec_rejected(t *testing.T) {
	tests := []struct {
		name string

		id         string
		command    string
		parameters []string
	}{
		{
			name: "command not in allowlist",

			id:      uuid.NewString(),
			command: "bash",
		},
		{
			name: "empty command",

			id:      uuid.NewString(),
			command: "",
		},
		{
			name: "shell metacharacter in parameter",

			id:         uuid.NewString(),
			command:    "tcpdump",
			parameters: []string{"udp port 5060; reboot"},
		},
		{
			name: "caller supplied write path",

			id:         uuid.NewString(),
			command:    "tcpdump",
			parameters: []string{"-w", "/etc/passwd"},
		},
		{
			name: "id is not a uuid",

			id:      "not-a-uuid",
			command: "tcpdump",
		},
		{
			name: "id escapes the tmp directory",

			id:      "../x",
			command: "tcpdump",
		},
		{
			name: "uppercase uuid is rejected",

			id:      strings.ToUpper(uuid.NewString()),
			command: "tcpdump",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A rejected request must never reach the process start. The fake
			// records an invocation, so a regressed check cannot pass silently
			// (and can never launch a real tcpdump).
			installFakeTcpdump(t, fakeTcpdumpNormal)
			argsFile := filepath.Join(t.TempDir(), "args")
			t.Setenv("FAKE_ARGS_FILE", argsFile)

			m := newTestManager(nil)

			err := m.Exec(tt.id, tt.command, tt.parameters)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			} else if strings.Contains(err.Error(), "start ") {
				t.Errorf("Wrong match. request must be rejected by validation, got: %v", err)
			}
			if m.Count() != 0 {
				t.Errorf("Wrong match. rejected request must not be tracked, count: %d", m.Count())
			}
			time.Sleep(50 * time.Millisecond)
			if fileExists(argsFile) {
				t.Errorf("Wrong match. rejected request started a process")
			}
		})
	}
}

func Test_Exec_startFailure(t *testing.T) {
	// No tcpdump anywhere on PATH.
	t.Setenv("PATH", t.TempDir())

	m := newTestManager(nil)
	if err := m.Exec(uuid.NewString(), "tcpdump", nil); err == nil {
		t.Fatalf("Wrong match. expect: err, got: ok")
	}
	if m.Count() != 0 {
		t.Errorf("Wrong match. failed start must not be tracked, count: %d", m.Count())
	}
}

func Test_Exec_arguments(t *testing.T) {
	installFakeTcpdump(t, fakeTcpdumpNormal)
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("FAKE_ARGS_FILE", argsFile)

	m := newTestManager(nil)
	id := uuid.NewString()
	pcap := "/tmp/" + id + ".pcap"

	if err := m.Exec(id, "tcpdump", []string{"-nn", "udp"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(m.Shutdown)

	waitFor(t, "pcap file", func() bool { return fileExists(pcap) })

	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("could not read recorded args: %v", err)
	}
	expect := "-i eth9 -s 0 -w " + pcap + " -nn udp"
	if got := strings.TrimSpace(string(raw)); got != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
	}
	if m.Count() != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", m.Count())
	}
}

func Test_Exec_duplicateAndLimit(t *testing.T) {
	installFakeTcpdump(t, fakeTcpdumpNormal)

	m := newTestManager(nil)
	m.maxConcurrent = 2
	t.Cleanup(m.Shutdown)

	id1 := uuid.NewString()
	id2 := uuid.NewString()

	if err := m.Exec(id1, "tcpdump", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := m.Exec(id1, "tcpdump", nil); err == nil {
		t.Errorf("Wrong match. duplicate id must be rejected")
	}
	if err := m.Exec(id2, "tcpdump", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := m.Exec(uuid.NewString(), "tcpdump", nil); err == nil {
		t.Errorf("Wrong match. process beyond the concurrency cap must be rejected")
	}
	if m.Count() != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", m.Count())
	}

	// A freed slot is reusable.
	waitFor(t, "pcap file", func() bool { return fileExists("/tmp/" + id1 + ".pcap") })
	if _, err := m.Kill(id1); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := m.Exec(uuid.NewString(), "tcpdump", nil); err != nil {
		t.Errorf("Wrong match. expect: ok after a slot was freed, got: %v", err)
	}
}

func Test_Kill_unknownID(t *testing.T) {
	m := newTestManager(nil)

	path, err := m.Kill(uuid.NewString())
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
	if path != "" {
		t.Errorf("Wrong match. expect empty path, got: %s", path)
	}
}

func Test_Kill_upload(t *testing.T) {
	tests := []struct {
		name string

		withUploader bool
		uploadErrs   []error // result of each Upload call, in order

		expectLocalFileKept bool
	}{
		{
			name: "no uploader removes the local file",

			withUploader: false,

			expectLocalFileKept: false,
		},
		{
			name: "upload succeeds and removes the local file",

			withUploader: true,
			uploadErrs:   []error{nil},

			expectLocalFileKept: false,
		},
		{
			name: "first upload fails and retry succeeds",

			withUploader: true,
			uploadErrs:   []error{os.ErrDeadlineExceeded, nil},

			expectLocalFileKept: false,
		},
		{
			name: "both uploads fail and the local file is kept",

			withUploader: true,
			uploadErrs:   []error{os.ErrDeadlineExceeded, os.ErrDeadlineExceeded},

			expectLocalFileKept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			installFakeTcpdump(t, fakeTcpdumpNormal)

			id := uuid.NewString()
			pcap := "/tmp/" + id + ".pcap"
			t.Cleanup(func() { _ = os.Remove(pcap) })

			var uploader gcsuploader.Uploader
			if tt.withUploader {
				mock := gcsuploader.NewMockUploader(mc)
				var calls []any
				for _, e := range tt.uploadErrs {
					calls = append(calls, mock.EXPECT().
						Upload(pcap, gomock.Any()).
						DoAndReturn(func(local, remote string) (string, error) {
							if !strings.HasPrefix(remote, "rtp-recordings/"+id+"-") || !strings.HasSuffix(remote, ".pcap") {
								t.Errorf("Wrong remote name: %s", remote)
							}
							return "", e
						}))
				}
				gomock.InOrder(calls...)
				uploader = mock
			}

			m := newTestManager(uploader)
			if err := m.Exec(id, "tcpdump", nil); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			waitFor(t, "pcap file", func() bool { return fileExists(pcap) })

			got, err := m.Kill(id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got != pcap {
				t.Errorf("Wrong match. expect: %s, got: %s", pcap, got)
			}
			if m.Count() != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", m.Count())
			}
			if fileExists(pcap) != tt.expectLocalFileKept {
				t.Errorf("Wrong match. expect local file kept: %v, got: %v", tt.expectLocalFileKept, fileExists(pcap))
			}
		})
	}
}

func Test_Kill_sigkillEscalation(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the 5 second SIGTERM grace period")
	}

	installFakeTcpdump(t, fakeTcpdumpIgnoreTerm)
	ready := filepath.Join(t.TempDir(), "ready")
	t.Setenv("FAKE_READY_FILE", ready)

	m := newTestManager(nil)
	id := uuid.NewString()
	if err := m.Exec(id, "tcpdump", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	// SIGTERM is only ignored once the trap is installed.
	waitFor(t, "ready marker", func() bool { return fileExists(ready) })

	start := time.Now()
	if _, err := m.Kill(id); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 4500*time.Millisecond || elapsed > 20*time.Second {
		t.Errorf("Wrong match. process ignoring SIGTERM must die by SIGKILL right after the grace period, took: %v", elapsed)
	}
}

func Test_safetyTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	installFakeTcpdump(t, fakeTcpdumpNormal)

	id := uuid.NewString()
	pcap := "/tmp/" + id + ".pcap"
	t.Cleanup(func() { _ = os.Remove(pcap) })

	uploaded := make(chan struct{})
	signal := signalOnce(uploaded)
	mock := gcsuploader.NewMockUploader(mc)
	mock.EXPECT().Upload(pcap, gomock.Any()).DoAndReturn(func(string, string) (string, error) {
		signal()
		return "gs://b/x", nil
	}).Times(1)

	m := NewManager("eth9", 200*time.Millisecond, mock).(*manager)
	if err := m.Exec(id, "tcpdump", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-uploaded:
	case <-time.After(10 * time.Second):
		t.Fatal("safety timeout never killed and uploaded the capture")
	}

	// Upload is called before the local file is removed and after the entry is
	// dropped, so wait for the end state instead of asserting at the signal.
	waitFor(t, "capture cleanup", func() bool { return m.Count() == 0 && !fileExists(pcap) })
}

func Test_Shutdown(t *testing.T) {
	tests := []struct {
		name string

		uploadErr error

		expectLocalFilesKept bool
	}{
		{
			name: "uploads and removes every capture",

			uploadErr: nil,

			expectLocalFilesKept: false,
		},
		{
			name: "upload failures keep the local files",

			uploadErr: os.ErrDeadlineExceeded,

			expectLocalFilesKept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			installFakeTcpdump(t, fakeTcpdumpNormal)

			ids := []string{uuid.NewString(), uuid.NewString()}
			mock := gcsuploader.NewMockUploader(mc)
			attempts := 1
			if tt.uploadErr != nil {
				attempts = 2 // initial try plus one retry
			}
			for _, id := range ids {
				pcap := "/tmp/" + id + ".pcap"
				t.Cleanup(func() { _ = os.Remove(pcap) })
				mock.EXPECT().Upload(pcap, gomock.Any()).Return("", tt.uploadErr).Times(attempts)
			}

			m := newTestManager(mock)
			for _, id := range ids {
				if err := m.Exec(id, "tcpdump", nil); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			}
			for _, id := range ids {
				id := id
				waitFor(t, "pcap file", func() bool { return fileExists("/tmp/" + id + ".pcap") })
			}

			m.Shutdown()

			if m.Count() != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", m.Count())
			}
			for _, id := range ids {
				if got := fileExists("/tmp/" + id + ".pcap"); got != tt.expectLocalFilesKept {
					t.Errorf("Wrong match. expect local file kept: %v, got: %v", tt.expectLocalFilesKept, got)
				}
			}
		})
	}
}

func Test_Shutdown_empty(t *testing.T) {
	m := newTestManager(nil)

	m.Shutdown() // must not panic or block

	if m.Count() != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", m.Count())
	}
}

func Test_CleanOrphans(t *testing.T) {
	// CleanOrphans globs the shared /tmp/*.pcap and deletes every uuid-named
	// file, which cannot be redirected without changing product code. Do not
	// run it where it could delete captures that are not this test's own.
	others, _ := filepath.Glob("/tmp/*.pcap")
	for _, f := range others {
		if uuidPattern.MatchString(filepath.Base(f)) {
			t.Skipf("foreign uuid pcap present in /tmp (%s); skipping to avoid deleting it", filepath.Base(f))
		}
	}

	orphan := "/tmp/" + uuid.NewString() + ".pcap"
	keep := "/tmp/keep-" + uuid.NewString() + ".pcap"
	notPcap := "/tmp/" + uuid.NewString() + ".txt"
	for _, f := range []string{orphan, keep, notPcap} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatalf("could not create fixture: %v", err)
		}
		f := f
		t.Cleanup(func() { _ = os.Remove(f) })
	}

	newTestManager(nil).CleanOrphans()

	if fileExists(orphan) {
		t.Errorf("Wrong match. uuid-named orphan must be removed")
	}
	if !fileExists(keep) {
		t.Errorf("Wrong match. non-uuid pcap must be kept")
	}
	if !fileExists(notPcap) {
		t.Errorf("Wrong match. non-pcap file must be kept")
	}
}

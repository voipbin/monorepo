package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// processmanagerLockPath is the lock file the pkg/processmanager tests hold for
// their whole run. main() calls CleanOrphans, which deletes every uuid-named
// /tmp/*.pcap, so the scenarios that reach it take the same lock.
const processmanagerLockPath = "/tmp/voip-rtpengine-proxy-processmanager-test.lock"

var uuidPcapPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.pcap$`)

// skipOrFatal skips a scenario whose precondition is missing on a developer
// host and fails it in CI, so CI cannot pass silently with lower coverage.
func skipOrFatal(t *testing.T, format string, args ...any) {
	t.Helper()

	msg := fmt.Sprintf(format, args...)
	if os.Getenv("CI") != "" || os.Getenv("CIRCLECI") != "" {
		t.Fatalf("precondition missing in CI: %s", msg)
	}
	t.Skipf("precondition missing: %s", msg)
}

// lockTmp takes the cross-package lock on the shared /tmp pcap namespace and
// releases it when the test ends. It polls, because the processmanager tests
// hold the lock for their whole run and the packages run in parallel.
func lockTmp(t *testing.T) {
	t.Helper()

	lock, err := os.OpenFile(processmanagerLockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		skipOrFatal(t, "could not open the lock file: %v", err)
		return
	}

	deadline := time.Now().Add(120 * time.Second)
	for {
		err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = lock.Close()
			skipOrFatal(t, "could not take the lock: %v", err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Cleanup(func() { _ = lock.Close() }) // closing releases the lock

	// The check runs under the lock, so a file of another run cannot show up here.
	others, _ := filepath.Glob("/tmp/*.pcap")
	for _, f := range others {
		if uuidPcapPattern.MatchString(filepath.Base(f)) {
			skipOrFatal(t, "foreign uuid pcap in /tmp (%s) would be deleted by CleanOrphans", filepath.Base(f))
			return
		}
	}
}

// testInterface returns the name and the non-loopback IPv4 set of an
// interface main() can resolve, or ends the test through skipOrFatal.
func testInterface(t *testing.T) (string, []string) {
	t.Helper()

	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("could not list interfaces: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if ips := ifaceIPv4s(t, iface); len(ips) > 0 {
			return iface.Name, ips
		}
	}

	skipOrFatal(t, "no up interface with a non-loopback IPv4 address")
	return "", nil
}

// startCfg selects what a scenario changes in the globals main() reads.
type startCfg struct {
	lock bool // the scenario reaches CleanOrphans

	connect bool // sock handler is created and connected
	listen  bool // listen handler registers both queues

	ngAddress      string // empty keeps a live stand-in
	ngTimeout      string
	recordingDir   string
	bucket         string
	redisAddress   string // empty keeps the fake Redis
	emulatorHost   string // only read when the client is created; no scenario uploads
	credentialFile string // value of GOOGLE_APPLICATION_CREDENTIALS, empty keeps unset
}

type mainRun struct {
	t      *testing.T
	logs   *logCapture
	redis  *fakeRedis
	sigs   chan os.Signal
	done   chan struct{}
	ips    []string
	orphan string         // uuid-named pcap planted under the lock, empty without lock
	queues chan [2]string // name and type, in call order

	consumed chan struct{}
}

// startMain runs main() in a goroutine with the sock handler replaced by a
// mock and every other dependency local.
func startMain(t *testing.T, cfg startCfg) *mainRun {
	t.Helper()

	ifaceName, ips := testInterface(t)
	orphan := ""
	if cfg.lock {
		lockTmp(t)
		orphan = plantOrphan(t)
	}

	saved := struct {
		interfaceName, redisAddress, rtpengineNGAddress, rtpengineNGTimeout, recordingDir, gcpBucketNameMedia string
		chSigs                                                                                                chan os.Signal
	}{interfaceName, redisAddress, rtpengineNGAddress, rtpengineNGTimeout, recordingDir, gcpBucketNameMedia, chSigs}
	t.Cleanup(func() {
		interfaceName, redisAddress, rtpengineNGAddress = saved.interfaceName, saved.redisAddress, saved.rtpengineNGAddress
		rtpengineNGTimeout, recordingDir, gcpBucketNameMedia = saved.rtpengineNGTimeout, saved.recordingDir, saved.gcpBucketNameMedia
		chSigs = saved.chSigs
	})

	r := &mainRun{
		t:        t,
		logs:     captureLogs(t),
		redis:    newFakeRedis(t),
		sigs:     make(chan os.Signal, 1),
		done:     make(chan struct{}),
		ips:      ips,
		orphan:   orphan,
		queues:   make(chan [2]string, 2),
		consumed: make(chan struct{}, 2),
	}

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	if cfg.connect {
		mockSock.EXPECT().Connect().Times(1)
	}
	if cfg.listen {
		mockSock.EXPECT().QueueCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(name, queueType string) error {
			r.queues <- [2]string{name, queueType}
			return nil
		}).Times(2)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
				r.consumed <- struct{}{}
				return nil
			}).Times(2)
	}
	restoreFactory := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreFactory)

	interfaceName = ifaceName
	redisAddress = r.redis.ln.Addr().String()
	if cfg.redisAddress != "" {
		redisAddress = cfg.redisAddress
	}
	rtpengineNGAddress = cfg.ngAddress
	if rtpengineNGAddress == "" {
		ngStandIn, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not listen: %v", err)
		}
		t.Cleanup(func() { _ = ngStandIn.Close() })
		rtpengineNGAddress = ngStandIn.LocalAddr().String()
	}
	rtpengineNGTimeout = cfg.ngTimeout
	recordingDir = cfg.recordingDir
	gcpBucketNameMedia = cfg.bucket
	t.Setenv("STORAGE_EMULATOR_HOST", cfg.emulatorHost)
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", cfg.credentialFile)
	chSigs = r.sigs

	// Cleanups run in reverse order: this one is registered last, so it runs
	// first and ends main(); then the mock controller finishes, the globals
	// are restored and the lock is released.
	t.Cleanup(func() {
		r.signal()
		select {
		case <-r.done:
		case <-time.After(10 * time.Second):
			t.Error("main() did not return in cleanup")
		}
	})

	go func() {
		// The deferred close also runs when a mock call fails the test with
		// runtime.Goexit, so the waits below cannot hang.
		defer close(r.done)
		main()
	}()
	return r
}

// signal sends the termination value without blocking, because the buffer
// holds one value.
func (r *mainRun) signal() {
	select {
	case r.sigs <- syscall.SIGTERM:
	default:
	}
}

// stop waits for both consumers of the listen handler, then ends main().
func (r *mainRun) stop() {
	r.t.Helper()

	for i := 0; i < 2; i++ {
		select {
		case <-r.consumed:
		case <-r.done:
			r.t.Fatal("main() returned before both queues were consumed")
		case <-time.After(10 * time.Second):
			r.t.Fatal("timed out waiting for the queue consumers")
		}
	}
	r.signal()
	r.wait()
}

// wait waits for main() to return.
func (r *mainRun) wait() {
	r.t.Helper()

	select {
	case <-r.done:
	case <-time.After(10 * time.Second):
		r.t.Fatal("timed out waiting for main() to return")
	}
}

func (r *mainRun) count(level logrus.Level, prefix string) int {
	return r.logs.count(msgPrefix(level, prefix, ""))
}

// plantOrphan creates a uuid-named pcap that CleanOrphans must delete.
func plantOrphan(t *testing.T) string {
	t.Helper()

	path := "/tmp/" + uuid.NewString() + ".pcap"
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("could not create the orphan fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

// assertOrphanRemoved checks that CleanOrphans ran: the planted uuid-named pcap is gone.
func (r *mainRun) assertOrphanRemoved() {
	r.t.Helper()

	if _, err := os.Stat(r.orphan); !os.IsNotExist(err) {
		r.t.Errorf("Wrong match. CleanOrphans must delete the uuid-named orphan, stat err: %v", err)
	}
}

// expectRegistration checks the Redis registration and the queue names, and
// returns the proxy ID used.
func (r *mainRun) expectRegistration() string {
	r.t.Helper()

	setPattern := regexp.MustCompile(`^set rtpengine\.(\S+)\.address-internal (\S+) ex 86400$`)
	proxyID := ""
	for _, cmd := range r.redis.received() {
		m := setPattern.FindStringSubmatch(cmd)
		if m == nil {
			continue
		}
		if m[1] != m[2] {
			r.t.Errorf("Wrong match. key ip and value must be equal. key: %s, value: %s", m[1], m[2])
		}
		proxyID = m[1]
	}

	found := false
	for _, ip := range r.ips {
		if ip == proxyID {
			found = true
		}
	}
	if !found {
		r.t.Fatalf("Wrong match. proxy id %q must be one of the interface addresses %v, commands: %v", proxyID, r.ips, r.redis.received())
	}

	got := map[string]string{}
	for i := 0; i < 2; i++ {
		select {
		case q := <-r.queues:
			got[q[0]] = q[1]
		case <-time.After(10 * time.Second):
			r.t.Fatal("timed out waiting for a queue registration")
		}
	}
	expect := map[string]string{
		"test.queue.listen":                          "normal",
		fmt.Sprintf("rtpengine.%s.request", proxyID): "volatile",
	}
	for name, qType := range expect {
		if got[name] != qType {
			r.t.Errorf("Wrong match. queue %q expect type: %q, got: %q (all: %v)", name, qType, got[name], got)
		}
	}
	return proxyID
}

func Test_main_startWithWatcher(t *testing.T) {
	dir := t.TempDir()
	run := startMain(t, startCfg{
		lock:         true,
		connect:      true,
		listen:       true,
		ngTimeout:    "7s",
		recordingDir: dir,
		bucket:       "test-media-bucket",
		emulatorHost: "127.0.0.1:1",
	})
	run.stop()

	run.expectRegistration()
	run.assertOrphanRemoved()

	if run.count(logrus.WarnLevel, "Invalid NG timeout") != 0 {
		t.Errorf("Wrong match. a valid NG timeout must not log a warning")
	}
	if run.count(logrus.InfoLevel, "pcap watcher enabled") != 1 {
		t.Errorf("Wrong match. expected the pcap watcher enabled log")
	}
	run.logs.waitFor(t, "pcap watcher started", msgPrefix(logrus.InfoLevel, "pcap watcher started", ""))
	if _, err := os.Stat(filepath.Join(dir, "metadata")); err != nil {
		t.Errorf("Wrong match. the watcher must create the metadata dir: %v", err)
	}
	if got := run.logs.count(msgPrefix(logrus.InfoLevel, "Terminating rtpengine-proxy", syscall.SIGTERM.String())); got != 1 {
		t.Errorf("Wrong match. expect one terminating log with the signal, got: %d", got)
	}
	// The stop log is written by the watcher goroutine after main() returned.
	run.logs.waitFor(t, "pcap watcher stopping", msgPrefix(logrus.InfoLevel, "pcap watcher stopping", ""))
}

func Test_main_watcherNotStarted(t *testing.T) {
	tests := []struct {
		name string

		recordingDir  string
		bucket        string
		expectTimeout bool
	}{
		{"recording dir without bucket", "recording-dir", "", false},
		{"bucket without recording dir", "", "test-media-bucket", false},
		{"invalid NG timeout, nothing configured", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := startCfg{
				lock:         true,
				connect:      true,
				listen:       true,
				ngTimeout:    "7s",
				emulatorHost: "127.0.0.1:1",
				bucket:       tt.bucket,
			}
			if tt.recordingDir != "" {
				cfg.recordingDir = t.TempDir()
			}
			if tt.expectTimeout {
				cfg.ngTimeout = "not-a-duration"
			}

			run := startMain(t, cfg)
			run.stop()

			run.expectRegistration()
			run.assertOrphanRemoved()

			if got := run.count(logrus.InfoLevel, "pcap watcher disabled"); got != 1 {
				t.Errorf("Wrong match. expect one watcher disabled log, got: %d", got)
			}
			if got := run.count(logrus.InfoLevel, "pcap watcher enabled"); got != 0 {
				t.Errorf("Wrong match. the watcher must not be enabled, got: %d", got)
			}
			if got := run.logs.count(msgPrefix(logrus.WarnLevel, "Invalid NG timeout", "not-a-duration")); (got == 1) != tt.expectTimeout {
				t.Errorf("Wrong match. invalid timeout warning count: %d, expect warning: %v", got, tt.expectTimeout)
			}
		})
	}
}

func Test_main_watcherError(t *testing.T) {
	// A directory below a regular file cannot be created, even as root.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("could not create the fixture: %v", err)
	}

	run := startMain(t, startCfg{
		lock:         true,
		connect:      true,
		listen:       true,
		ngTimeout:    "7s",
		recordingDir: filepath.Join(blocker, "recording"),
		bucket:       "test-media-bucket",
		emulatorHost: "127.0.0.1:1",
	})
	run.logs.waitFor(t, "pcap watcher error", msgPrefix(logrus.ErrorLevel, "pcap watcher error", ""))
	run.stop()

	run.assertOrphanRemoved()
	if run.count(logrus.InfoLevel, "pcap watcher started") != 0 {
		t.Errorf("Wrong match. the watcher must not report a start")
	}
}

func Test_main_earlyReturns(t *testing.T) {
	// A loopback port that was just closed refuses connections at once.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	closedAddr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("could not close the listener: %v", err)
	}

	tests := []struct {
		name string

		cfg          startCfg
		expectPrefix string
	}{
		{
			"redis unreachable",
			startCfg{redisAddress: closedAddr},
			"Could not register proxy in Redis",
		},
		{
			"invalid NG address",
			startCfg{connect: true, ngAddress: "bad", ngTimeout: "7s"},
			"Could not create NG client",
		},
		{
			"GCS client creation fails",
			startCfg{
				connect:        true,
				ngTimeout:      "7s",
				bucket:         "test-media-bucket",
				credentialFile: filepath.Join(t.TempDir(), "missing-credentials.json"),
			},
			"Could not create GCS uploader for process manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := startMain(t, tt.cfg)
			run.wait()

			if got := run.count(logrus.ErrorLevel, tt.expectPrefix); got != 1 {
				t.Errorf("Wrong match. expect one %q error log, got: %d", tt.expectPrefix, got)
			}
			if got := run.count(logrus.InfoLevel, "Terminating"); got != 0 {
				t.Errorf("Wrong match. main() must return before the signal wait, got: %d", got)
			}
		})
	}
}

package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"monorepo/bin-hook-manager/internal/config"
)

var endpointCounter atomic.Int64

// uniqueEndpoint returns a metrics endpoint that is never registered twice in the process,
// because initProm registers on the shared default mux.
func uniqueEndpoint(prefix string) string {
	return fmt.Sprintf("/%s-%d", prefix, endpointCounter.Add(1))
}

// closedLoopbackAddr returns a loopback address that nothing listens on.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a loopback port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

// captureLogs replaces the standard logger hooks for the test and restores them afterwards.
func captureLogs(t *testing.T) *logrustest.Hook {
	t.Helper()

	std := logrus.StandardLogger()
	prevHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	prevLevel := std.GetLevel()
	std.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		std.ReplaceHooks(prevHooks)
		std.SetLevel(prevLevel)
	})

	return logrustest.NewLocal(std)
}

// funcMessages returns the messages logged by the given func field value.
func funcMessages(hook *logrustest.Hook, funcName string) []string {
	res := []string{}
	for _, e := range hook.AllEntries() {
		if e.Data["func"] == funcName {
			res = append(res, e.Message)
		}
	}

	return res
}

func containsMessage(messages []string, part string) bool {
	for _, m := range messages {
		if strings.Contains(m, part) {
			return true
		}
	}

	return false
}

// prepareRunService points the global config to a closed database address, a loopback metrics
// listener and synthetic ssl data. It skips the test when the fixed ssl files already exist so
// that files of another process are never overwritten, and removes the files afterwards.
func prepareRunService(t *testing.T) {
	t.Helper()

	for _, filename := range []string{constSSLCertFilename, constSSLPrivFilename} {
		if _, err := os.Lstat(filename); err == nil {
			t.Skipf("%s already exists, not touching it", filename)
		}
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(constSSLCertFilename)
		_ = os.RemoveAll(constSSLPrivFilename)
	})

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })

	cfg.DatabaseDSN = fmt.Sprintf("tcp(%s)/test", closedLoopbackAddr(t))
	cfg.PrometheusEndpoint = uniqueEndpoint("run-service")
	cfg.PrometheusListenAddress = "127.0.0.1:0"
	cfg.SSLCertBase64 = base64.StdEncoding.EncodeToString([]byte("synthetic-cert-body"))
	cfg.SSLPrivkeyBase64 = base64.StdEncoding.EncodeToString([]byte("synthetic-key-body"))
}

// runServiceReturns runs runService and fails the test when it does not return in time.
func runServiceReturns(t *testing.T) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		runService(nil, nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("runService did not return in time")
	}
}

func readFileString(t *testing.T, filename string) (string, bool) {
	t.Helper()

	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return "", false
	}
	if err != nil {
		t.Fatalf("could not read %s: %v", filename, err)
	}

	return string(data), true
}

func Test_runService_databaseUnreachable(t *testing.T) {
	prepareRunService(t)
	hook := captureLogs(t)

	runServiceReturns(t)

	messages := funcMessages(hook, "runService")
	if !containsMessage(messages, "Could not access to database") {
		t.Errorf("Wrong match. expect: database failure log, got: %v", messages)
	}
	if containsMessage(messages, "Could not write the ssl") {
		t.Errorf("Wrong match. expect: no ssl write failure, got: %v", messages)
	}

	if got, ok := readFileString(t, constSSLCertFilename); !ok || got != "synthetic-cert-body" {
		t.Errorf("Wrong match. expect: cert written, got: %q (exists: %v)", got, ok)
	}
	if got, ok := readFileString(t, constSSLPrivFilename); !ok || got != "synthetic-key-body" {
		t.Errorf("Wrong match. expect: key written, got: %q (exists: %v)", got, ok)
	}
}

func Test_runService_privateKeyNotWritable(t *testing.T) {
	prepareRunService(t)
	hook := captureLogs(t)

	// a directory in place of the key file makes the key write fail while the cert write succeeds.
	if err := os.Mkdir(constSSLPrivFilename, 0o700); err != nil {
		t.Skipf("could not occupy the key path: %v", err)
	}

	runServiceReturns(t)

	messages := funcMessages(hook, "runService")
	if !containsMessage(messages, "Could not write the ssl private key") {
		t.Errorf("Wrong match. expect: private key failure log, got: %v", messages)
	}
	if containsMessage(messages, "Could not access to database") {
		t.Errorf("Wrong match. expect: return before the database, got: %v", messages)
	}

	if got, ok := readFileString(t, constSSLCertFilename); !ok || got != "synthetic-cert-body" {
		t.Errorf("Wrong match. expect: cert written first, got: %q (exists: %v)", got, ok)
	}
}

func Test_runService_certNotWritable(t *testing.T) {
	prepareRunService(t)
	hook := captureLogs(t)

	// a directory in place of the cert file makes the first write fail.
	if err := os.Mkdir(constSSLCertFilename, 0o700); err != nil {
		t.Skipf("could not occupy the cert path: %v", err)
	}

	runServiceReturns(t)

	messages := funcMessages(hook, "runService")
	if !containsMessage(messages, "Could not write the ssl cert") {
		t.Errorf("Wrong match. expect: cert failure log, got: %v", messages)
	}
	if containsMessage(messages, "Could not write the ssl private key") || containsMessage(messages, "Could not access to database") {
		t.Errorf("Wrong match. expect: return before the key and the database, got: %v", messages)
	}
	if _, ok := readFileString(t, constSSLPrivFilename); ok {
		t.Errorf("Wrong match. expect: key file not written after the cert failure")
	}
}

func Test_writeBase64_writeFailure(t *testing.T) {
	// /dev/full accepts the open but fails every non-empty write.
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is not available")
	}

	data := base64.StdEncoding.EncodeToString([]byte("synthetic-content"))
	if err := writeBase64("/dev/full", data); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_initProm_listenerBusy(t *testing.T) {
	hook := captureLogs(t)

	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a loopback port: %v", err)
	}
	t.Cleanup(func() { _ = busy.Close() })
	addr := busy.Addr().String()
	endpoint := uniqueEndpoint("busy")

	initProm(endpoint, addr)

	retried := func() bool {
		for _, e := range hook.AllEntries() {
			if e.Data["listen"] == addr && strings.Contains(e.Message, "Could not start prometheus listener") {
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(8 * time.Second)
	for !retried() {
		if time.Now().After(deadline) {
			t.Fatalf("the listener failure was not logged in time")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// once the port is released the retry loop serves the metrics endpoint.
	_ = busy.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline.Add(8 * time.Second)) {
		res, errGet := client.Get("http://" + addr + endpoint)
		if errGet != nil {
			lastErr = errGet
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("Wrong status. expect: 200, got: %d", res.StatusCode)
		}
		if !strings.Contains(string(body), "go_goroutines") {
			t.Errorf("metrics body does not look like prometheus output")
		}
		return
	}
	t.Fatalf("the metrics endpoint did not come up after the port was released. last error: %v", lastErr)
}

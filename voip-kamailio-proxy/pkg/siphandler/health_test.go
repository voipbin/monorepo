package siphandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func Test_SendOptionsCheck_timeout(t *testing.T) {
	// Use TEST-NET (192.0.2.1) which never responds — RFC 5737
	result, err := SendOptionsCheck(context.Background(), "192.0.2.1", 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if result.Healthy {
		t.Errorf("Expected unhealthy for unreachable host, got healthy")
	}
	if result.ResponseCode != "timeout" {
		t.Errorf("Expected 'timeout' response code, got: %s", result.ResponseCode)
	}
}

func Test_SendOptionsCheck_invalidHost(t *testing.T) {
	result, err := SendOptionsCheck(context.Background(), "not-a-valid-host-xyz.invalid", 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if result.Healthy {
		t.Errorf("Expected unhealthy for invalid host, got healthy")
	}
	if result.ResponseCode != "timeout" {
		t.Errorf("Expected 'timeout' response code, got: %s", result.ResponseCode)
	}
}

// Test_SendOptionsCheck_dialFailure verifies that a destination which resolves
// but cannot be dialed is reported as unhealthy without an error. The scope of
// the link-local address names an interface that does not exist, so resolution
// succeeds and the UDP dial fails locally; no packet is sent.
func Test_SendOptionsCheck_dialFailure(t *testing.T) {
	logger := logrus.StandardLogger()
	oldLevel := logger.GetLevel()
	oldHooks := logger.ReplaceHooks(make(logrus.LevelHooks))
	logger.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logger.SetLevel(oldLevel)
		logger.ReplaceHooks(oldHooks)
	})
	hook := logrustest.NewGlobal()

	const hostname = "[fe80::1%nonexistent0]"
	result, err := SendOptionsCheck(context.Background(), hostname, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if result.Healthy {
		t.Errorf("Expected unhealthy when the dial fails, got healthy")
	}
	if result.ResponseCode != "timeout" {
		t.Errorf("Expected 'timeout' response code, got: %s", result.ResponseCode)
	}

	// The dial failure, not the resolve failure or the read timeout, must be
	// the reason for the result.
	var dialLogged, resolveLogged bool
	for _, e := range hook.AllEntries() {
		if e.Data["hostname"] != hostname {
			continue
		}
		if strings.HasPrefix(e.Message, "Could not dial UDP") {
			dialLogged = true
		}
		if strings.HasPrefix(e.Message, "Could not resolve UDP") {
			resolveLogged = true
		}
	}
	if !dialLogged {
		t.Errorf("Expected a 'Could not dial UDP' log for %s, got %d entries", hostname, len(hook.AllEntries()))
	}
	if resolveLogged {
		t.Errorf("Resolution unexpectedly failed for %s", hostname)
	}
}

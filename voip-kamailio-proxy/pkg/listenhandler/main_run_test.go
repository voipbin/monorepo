package listenhandler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/sockhandler"
)

// waitLogEntry polls the test hook until an entry whose message contains all
// of the given fragments appears, or fails the test after the upper bound.
func waitLogEntry(t *testing.T, hook *logrustest.Hook, fragments ...string) *logrus.Entry {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range hook.AllEntries() {
			matched := true
			for _, f := range fragments {
				if !strings.Contains(e.Message, f) {
					matched = false
					break
				}
			}
			if matched {
				return e
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timed out waiting for a log entry containing %q", fragments)
	return nil
}

// Test_Run_listenRunError verifies that Run returns immediately even while the
// background listenRun is still blocked, and that a listenRun failure is
// reported on the error log without any consumer being started.
func Test_Run_listenRunError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// Isolate the global logger: restore the previous hooks when done.
	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
	hook := logrustest.NewGlobal()

	gate := make(chan struct{})
	released := false
	release := func() {
		if !released {
			released = true
			close(gate)
		}
	}
	// Never leave the background goroutine blocked, even when the test fails early.
	defer release()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := newTestListenHandler(mockSock)

	// ConsumeRPC has no expectation: gomock fails the test if it is reached.
	mockSock.EXPECT().QueueCreate(testQueuePermanent, "normal").DoAndReturn(
		func(_, _ string) error {
			<-gate
			return fmt.Errorf("declare failed")
		},
	)

	runErr := make(chan error, 1)
	go func() { runErr <- h.Run() }()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Expected nil error from Run, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not return while listenRun was still blocked")
	}

	release()

	e := waitLogEntry(t, hook, "Could not execute the listen handler", "declare failed")
	if e.Level != logrus.ErrorLevel {
		t.Errorf("Expected error level, got %v", e.Level)
	}
	if e.Data["func"] != "Run" {
		t.Errorf("Expected func=Run, got %v", e.Data["func"])
	}
}

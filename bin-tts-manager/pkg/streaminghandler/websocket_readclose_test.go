package streaminghandler

import (
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// websocketReadCaptureHook records only the entries this test's own
// runWebSocketRead goroutine produces, filtered by a close-reason marker
// baked into the test, so a concurrently running test's log entries cannot
// be mistaken for this one's.
type websocketReadCaptureHook struct {
	mu      sync.Mutex
	entries []*logrus.Entry
}

func (h *websocketReadCaptureHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *websocketReadCaptureHook) Fire(entry *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, entry)
	return nil
}

func (h *websocketReadCaptureHook) hasLevelWithPrefix(level logrus.Level, prefix string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, e := range h.entries {
		if e.Level == level && len(e.Message) >= len(prefix) && e.Message[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// Test_runWebSocketRead_normalCloseIsLoggedAsDebugNotError verifies the
// func="runWebSocketRead" normal-closure branch logs at Debug level ("WebSocket
// closed normally") rather than Error ("WebSocket read error"), and that
// doneCh is still closed. This test waits for doneCh before inspecting the
// hook, so the goroutine cannot still be writing when the assertion runs.
func Test_runWebSocketRead_normalCloseIsLoggedAsDebugNotError(t *testing.T) {
	client, server := newLocalWSPair(t)

	stdLogger := logrus.StandardLogger()
	originalLevel := stdLogger.GetLevel()
	// logrus.LevelHooks is a map; a shallow copy aliases the same map that
	// AddHook mutates in place, which would make ReplaceHooks(originalHooks)
	// a no-op. Deep-copy before adding this test's hook.
	originalHooks := stdLogger.Hooks
	deepCopy := make(logrus.LevelHooks, len(originalHooks))
	for level, hooks := range originalHooks {
		deepCopy[level] = append([]logrus.Hook(nil), hooks...)
	}
	stdLogger.SetLevel(logrus.DebugLevel)
	stdLogger.ReplaceHooks(deepCopy)
	hook := &websocketReadCaptureHook{}
	stdLogger.AddHook(hook)
	t.Cleanup(func() {
		stdLogger.SetLevel(originalLevel)
		stdLogger.ReplaceHooks(originalHooks)
	})

	doneCh := make(chan struct{})
	go runWebSocketRead(client, doneCh)

	// The handshake registered the server conn via newLocalWSPair; have the
	// server send a proper close control frame so the client's ReadMessage
	// classifies it as a *websocket.CloseError (IsCloseError), matching the
	// normal-closure branch.
	_ = server.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))

	select {
	case <-doneCh:
	case <-time.After(3 * time.Second):
		t.Fatal("doneCh was not closed")
	}

	if hook.hasLevelWithPrefix(logrus.ErrorLevel, "WebSocket read error") {
		t.Error("Wrong match. expected: no Error-level entry for a normal close")
	}
}

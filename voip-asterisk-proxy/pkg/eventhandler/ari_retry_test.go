package eventhandler

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
)

// Test_Run_retries_after_refused_handshake checks that the endless ARI loop
// survives a refused websocket handshake: the first request is answered with
// HTTP 503, and the loop must try again and get an upgraded connection.
func Test_Run_retries_after_refused_handshake(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectation is set: the upgraded connection never carries a message,
	// so any publication would be a failure.
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	var requests atomic.Int32
	upgraded := make(chan int32, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/ari/events", func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		if n == 1 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = c.Close()
		}()

		select {
		case upgraded <- n:
		default:
		}

		// hold the connection open without sending anything, so the client
		// loop has no reason to reconnect while the test is running.
		<-make(chan struct{})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	h := &eventHandler{
		notifyhandler: mockNotify,
		ariAddr:       hostOf(t, ts),
	}
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case n := <-upgraded:
		if n < 2 {
			t.Errorf("Wrong request number. expect: the retry after the refused first request, got: %d", n)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("the loop did not reconnect after the refused handshake. requests: %d", requests.Load())
	}
}

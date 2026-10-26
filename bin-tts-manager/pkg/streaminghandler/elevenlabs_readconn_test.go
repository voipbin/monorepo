package streaminghandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test_readConnWebsock_errorChannelFull verifies the non-blocking send in
// readConnWebsock's error branch: when errCh already has a value buffered,
// the second error is dropped (the log.Warnf path) instead of blocking, and
// readConnWebsock still returns.
func Test_readConnWebsock_errorChannelFull(t *testing.T) {
	client, server := newLocalWSPair(t)
	// Close the server side abruptly so the client's ReadMessage fails with
	// a non-close-frame error, deterministically and without any timer.
	_ = server.Close()

	ctx := context.Background()
	msgCh := make(chan []byte)
	errCh := make(chan error, 1)
	errCh <- context.Canceled // pre-fill so the next send must hit "default"

	h := &elevenlabsHandler{}

	done := make(chan struct{})
	go func() {
		h.readConnWebsock(ctx, uuid.Nil, client, msgCh, errCh)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("readConnWebsock did not return")
	}

	// msgCh and errCh must both be closed by readConnWebsock's defers.
	if _, ok := <-msgCh; ok {
		t.Error("Wrong match. expected: msgCh closed, got: a value")
	}
	// The pre-filled error is still readable before the close is observed.
	if v, ok := <-errCh; !ok && v != nil {
		t.Error("Wrong match. expected: errCh closed after draining the buffered value")
	}
}

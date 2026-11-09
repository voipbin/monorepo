package engine_openai_handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// rotatingKey is a key function whose value a test changes between calls.
type rotatingKey struct {
	mu    sync.Mutex
	key   string
	err   error
	calls int

	// lastDeadline is the deadline of the context of the last call
	lastDeadline    time.Time
	lastHasDeadline bool
}

func (r *rotatingKey) set(key string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.key, r.err = key, err
}

func (r *rotatingKey) get(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.lastDeadline, r.lastHasDeadline = ctx.Deadline()
	return r.key, r.err
}

func (r *rotatingKey) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// authServer records the Authorization header of every request and answers like the API: a chat
// completion, or an event stream when the request asks for one.
func authServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()

	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()

		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"stream":true`) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		writeJSON(w, http.StatusOK, `{"id":"c","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(srv.Close)

	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func testRequest() *openai.ChatCompletionRequest {
	return &openai.ChatCompletionRequest{Model: "m", Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}}
}

// Every way to call the model reads the key for that call: a key changed after the handler was
// made is used by the next call, and nothing is kept between calls.
func Test_keyIsReadForEveryCall(t *testing.T) {
	calls := map[string]func(h *engineOpenaiHandler) error{
		"Send": func(h *engineOpenaiHandler) error { _, err := h.Send(context.Background(), testRequest()); return err },
		"SendOnce": func(h *engineOpenaiHandler) error {
			_, err := h.SendOnce(context.Background(), testRequest())
			return err
		},
		"streamingSend": func(h *engineOpenaiHandler) error {
			req := testRequest()
			req.Stream = true
			_, _, err := h.streamingSend(context.Background(), req)
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			srv, seen := authServer(t)
			key := &rotatingKey{key: "key-one"}
			h := NewEngineOpenaiHandlerWithConfig(key.get, srv.URL).(*engineOpenaiHandler)

			if err := call(h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			key.set("key-two", nil)
			if err := call(h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the stream request is read in the background, so wait for both to arrive
			deadline := time.Now().Add(2 * time.Second)
			for len(seen()) < 2 && time.Now().Before(deadline) {
				time.Sleep(5 * time.Millisecond)
			}

			got := seen()
			if len(got) != 2 || got[0] != "Bearer key-one" || got[1] != "Bearer key-two" {
				t.Errorf("Wrong match. expect: the rotated key at the next call, got: %v", got)
			}
			if key.count() != 2 {
				t.Errorf("Wrong match. expect: one key read per call, got: %d", key.count())
			}
		})
	}
}

// A key that can not be read stops the call before any request is sent, and the error of the key
// function is returned unchanged (not as a provider error).
func Test_keyReadErrorStopsTheCall(t *testing.T) {
	errKey := errors.New("synthetic key read failure")

	calls := map[string]func(h *engineOpenaiHandler) error{
		"Send": func(h *engineOpenaiHandler) error { _, err := h.Send(context.Background(), testRequest()); return err },
		"SendOnce": func(h *engineOpenaiHandler) error {
			_, err := h.SendOnce(context.Background(), testRequest())
			return err
		},
		"streamingSend": func(h *engineOpenaiHandler) error {
			_, _, err := h.streamingSend(context.Background(), testRequest())
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			srv, seen := authServer(t)
			key := &rotatingKey{err: errKey}
			h := NewEngineOpenaiHandlerWithConfig(key.get, srv.URL).(*engineOpenaiHandler)

			if err := call(h); !errors.Is(err, errKey) {
				t.Errorf("Wrong match. expect: the key error, got: %v", err)
			}
			if len(seen()) != 0 {
				t.Errorf("Wrong match. expect: no request, got: %v", seen())
			}
		})
	}
}

// The key function gets the context of the call, so the deadline of the call bounds the read.
func Test_keyFuncGetsTheCallContext(t *testing.T) {
	srv, _ := authServer(t)
	key := &rotatingKey{key: "k"}
	h := NewEngineOpenaiHandlerWithConfig(key.get, srv.URL).(*engineOpenaiHandler)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := h.SendOnce(ctx, testRequest()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	want, _ := ctx.Deadline()
	if !key.lastHasDeadline || !key.lastDeadline.Equal(want) {
		t.Errorf("Wrong match. expect: the deadline of the call, got: %v %v", key.lastDeadline, key.lastHasDeadline)
	}
}

// Without a key function the call fails and sends nothing.
func Test_noKeyFunction(t *testing.T) {
	h := NewEngineOpenaiHandlerWithConfig(nil, "http://127.0.0.1:9").(*engineOpenaiHandler)
	if _, err := h.SendOnce(context.Background(), testRequest()); err == nil {
		t.Errorf("Wrong match. expect: an error")
	}
}

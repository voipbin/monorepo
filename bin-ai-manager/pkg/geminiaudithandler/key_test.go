package geminiaudithandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// The key is read for every evaluation: a key changed after the handler was made is used by the
// next call, and a key that can not be read stops the call before a request is sent.
func Test_Evaluate_keyIsReadForEveryCall(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%q}}]}`, cov90ValidEvaluation)
	}))
	t.Cleanup(srv.Close)

	key, errKey := "key-one", error(nil)
	// made by the constructor, as the service makes it; only the endpoint is the test server
	h := NewGeminiAuditHandler(func(context.Context) (string, error) { return key, errKey }).(*geminiAuditHandler)
	h.baseURL = srv.URL

	for _, k := range []string{"key-one", "key-two"} {
		key = k
		if _, _, err := h.Evaluate(context.Background(), "prompt", "transcript", "en-US", false); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	errKey = errors.New("synthetic key read failure")
	if _, _, err := h.Evaluate(context.Background(), "prompt", "transcript", "en-US", false); err == nil {
		t.Errorf("Wrong match. expect: the key error")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || seen[0] != "Bearer key-one" || seen[1] != "Bearer key-two" {
		t.Errorf("Wrong match. expect: the rotated key and no request for the failed read, got: %v", seen)
	}
}

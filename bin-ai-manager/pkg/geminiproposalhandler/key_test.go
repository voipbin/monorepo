package geminiproposalhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// The key is read for every proposal: a key changed after the handler was made is used by the next
// call, and a key that can not be read stops the call before a request is sent.
func Test_Evaluate_keyIsReadForEveryCall(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"proposed_prompt\":\"better\",\"rationale\":\"because\"}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	key, errKey := "key-one", error(nil)
	// made by the constructor, as the service makes it; only the endpoint is the test server
	h := NewGeminiProposalHandler(func(context.Context) (string, error) { return key, errKey }).(*geminiProposalHandler)
	h.baseURL = srv.URL

	for _, k := range []string{"key-one", "key-two"} {
		key = k
		if _, err := h.Evaluate(context.Background(), "prompt", []AuditBlock{{Index: 1, OverallScore: 2, Summary: "s"}}, "en-US"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	errKey = errors.New("synthetic key read failure")
	if _, err := h.Evaluate(context.Background(), "prompt", []AuditBlock{{Index: 1, OverallScore: 2, Summary: "s"}}, "en-US"); err == nil || !strings.Contains(err.Error(), "could not get the API key") {
		t.Errorf("Wrong match. expect: the key error, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || seen[0] != "Bearer key-one" || seen[1] != "Bearer key-two" {
		t.Errorf("Wrong match. expect: the rotated key and no request for the failed read, got: %v", seen)
	}
}

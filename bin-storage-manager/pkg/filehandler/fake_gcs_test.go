package filehandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
)

// fakeGCS is a minimal GCS JSON API server. It knows object attributes (GET) and
// object deletion (DELETE) and nothing else. It is wired in through
// STORAGE_EMULATOR_HOST, so tests that use it must not run in parallel.
//
// Object keys have the form "<bucket>/<object path>".
type fakeGCS struct {
	srv *httptest.Server

	mu sync.Mutex

	objects      map[string]int64 // existing objects and their sizes
	deleteStatus map[string]int   // status code of DELETE by key, default 204

	deleted []string // keys of every DELETE request received
}

func newFakeGCS(t *testing.T, objects map[string]int64) *fakeGCS {
	t.Helper()

	f := &fakeGCS{
		objects:      objects,
		deleteStatus: map[string]int{},
	}

	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// the path ends with /b/<bucket>/o/<object path>
		idx := strings.Index(r.URL.Path, "/b/")
		if idx < 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		bucket, object, ok := strings.Cut(r.URL.Path[idx+len("/b/"):], "/o/")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := bucket + "/" + object

		f.mu.Lock()
		defer f.mu.Unlock()

		size, exists := f.objects[key]
		switch {
		case r.Method == http.MethodGet && exists:
			_, _ = fmt.Fprintf(w,
				`{"kind":"storage#object","bucket":%q,"name":%q,"size":"%d","mediaLink":"https://storage.test/download/%s","generation":"1","metageneration":"1"}`,
				bucket, object, size, key,
			)

		case r.Method == http.MethodDelete:
			f.deleted = append(f.deleted, key)
			if status, ok := f.deleteStatus[key]; ok {
				w.WriteHeader(status)
				if status >= 400 {
					_, _ = fmt.Fprintf(w, `{"error":{"code":%d,"message":"simulated"}}`, status)
				}
				return
			}
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"Not Found"}}`)
				return
			}
			delete(f.objects, key)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"Not Found"}}`)
		}
	}))
	t.Cleanup(f.srv.Close)

	t.Setenv("STORAGE_EMULATOR_HOST", f.srv.URL)

	return f
}

// client returns a storage client that talks to the fake.
func (f *fakeGCS) client(t *testing.T) *storage.Client {
	t.Helper()

	c, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func (f *fakeGCS) deletedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.deleted...)
}

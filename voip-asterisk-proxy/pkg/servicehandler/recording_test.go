package servicehandler

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/api/option"
)

// fakeGCS is a minimal fake of the GCS JSON upload endpoint
// (POST /upload/storage/v1/b/<bucket>/o, multipart). It records every request.
type fakeGCS struct {
	mu     sync.Mutex
	hits   int
	names  []string
	bodies [][]byte

	// status returns the response status for an object name. nil means 200.
	status func(name string) int
	// beforeResponse runs after the request body is read and before the response is written.
	beforeResponse func(name string)
}

func (f *fakeGCS) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")

		var body []byte
		if _, params, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err == nil {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for i := 0; ; i++ {
				p, errPart := mr.NextPart()
				if errPart != nil {
					break
				}
				b, _ := io.ReadAll(p)
				if i == 1 { // part 0 is the object metadata, part 1 is the media
					body = b
				}
			}
		}

		f.mu.Lock()
		f.hits++
		f.names = append(f.names, name)
		f.bodies = append(f.bodies, body)
		f.mu.Unlock()

		if f.beforeResponse != nil {
			f.beforeResponse(name)
		}

		st := http.StatusOK
		if f.status != nil {
			st = f.status(name)
		}
		w.Header().Set("Content-Type", "application/json")
		if st != http.StatusOK {
			// non-retryable 4xx with a JSON error body
			w.WriteHeader(st)
			_, _ = w.Write([]byte(`{"error":{"code":403,"message":"denied"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"bucket":"bucket","name":"` + name + `"}`))
	}
}

func (f *fakeGCS) requested(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.names {
		if n == name {
			return true
		}
	}
	return false
}

func newRecordingTestHandler(t *testing.T, f *fakeGCS) (*serviceHandler, string) {
	t.Helper()

	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)

	c, err := storage.NewClient(context.Background(), option.WithEndpoint(srv.URL+"/storage/v1/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("could not create the storage client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	dir := t.TempDir()
	h := &serviceHandler{
		client:                     c,
		recordingBucketName:        "bucket",
		recordingAsteriskDirectory: dir,
		recordingBucketDirectory:   "recording",
	}
	return h, dir
}

func writeRecordingFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("could not write the file: %v", err)
	}
	return p
}

func assertExists(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Errorf("expected %s to exist, got: %v", p, err)
	}
}

func assertNotExists(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted, got: %v", p, err)
	}
}

func Test_RecordingFileMove_success(t *testing.T) {
	f := &fakeGCS{}
	h, dir := newRecordingTestHandler(t, f)
	p := writeRecordingFile(t, dir, "a.wav", "hello")

	if err := h.RecordingFileMove(context.Background(), []string{"a.wav"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if f.hits != 1 {
		t.Fatalf("expected 1 upload request, got: %d", f.hits)
	}
	if f.names[0] != "recording/a.wav" {
		t.Errorf("expected object name recording/a.wav, got: %s", f.names[0])
	}
	if string(f.bodies[0]) != "hello" {
		t.Errorf("expected uploaded body hello, got: %q", f.bodies[0])
	}
	assertNotExists(t, p)
}

func Test_RecordingFileMove_commitRejected(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "non-empty file", content: "data"},
		{name: "0-byte file", content: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeGCS{status: func(string) int { return http.StatusForbidden }}
			h, dir := newRecordingTestHandler(t, f)
			p := writeRecordingFile(t, dir, "a.wav", tt.content)

			err := h.RecordingFileMove(context.Background(), []string{"a.wav"})
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), "failed to upload the file to the bucket") {
				t.Errorf("expected an upload (commit) error, got: %v", err)
			}
			if f.hits != 1 {
				t.Errorf("expected 1 upload request, got: %d", f.hits)
			}
			assertExists(t, p)
		})
	}
}

func Test_RecordingFileMove_copyFailure(t *testing.T) {
	f := &fakeGCS{}
	h, dir := newRecordingTestHandler(t, f)

	// A directory opens fine but io.Copy fails with "is a directory".
	p := filepath.Join(dir, "d.wav")
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatalf("could not create the directory: %v", err)
	}

	err := h.RecordingFileMove(context.Background(), []string{"d.wav"})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy data") {
		t.Errorf("expected a copy error, got: %v", err)
	}
	if f.hits != 0 {
		t.Errorf("expected no upload request (no partial object), got: %d", f.hits)
	}
	assertExists(t, p)
}

func Test_RecordingFileMove_stopsAtFirstError(t *testing.T) {
	f := &fakeGCS{status: func(name string) int {
		if name == "recording/b" {
			return http.StatusForbidden
		}
		return http.StatusOK
	}}
	h, dir := newRecordingTestHandler(t, f)
	pa := writeRecordingFile(t, dir, "a", "a")
	pb := writeRecordingFile(t, dir, "b", "b")
	pc := writeRecordingFile(t, dir, "c", "c")

	err := h.RecordingFileMove(context.Background(), []string{"a", "b", "c"})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "filename: b") {
		t.Errorf("expected the error to name file b, got: %v", err)
	}
	if f.hits != 2 {
		t.Errorf("expected exactly 2 upload requests, got: %d", f.hits)
	}
	if f.requested("recording/c") {
		t.Errorf("expected file c not to be requested")
	}
	assertNotExists(t, pa)
	assertExists(t, pb)
	assertExists(t, pc)
}

func Test_RecordingFileMove_deleteFailureAfterCommit(t *testing.T) {
	// Uses the global logrus hook: this test must not run in parallel.
	hook := test.NewGlobal()
	hook.Reset()
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	f := &fakeGCS{}
	h, dir := newRecordingTestHandler(t, f)
	p := writeRecordingFile(t, dir, "a.wav", "x")

	// Before GCS answers, replace the source file with a non-empty directory so that the
	// local delete fails with ENOTEMPTY. Close returns only after this response, so the
	// swap always happens before os.Remove. Deterministic also when running as root.
	f.beforeResponse = func(string) {
		_ = os.Remove(p)
		_ = os.Mkdir(p, 0o700)
		_ = os.WriteFile(filepath.Join(p, "keep"), []byte("k"), 0o600)
	}

	if err := h.RecordingFileMove(context.Background(), []string{"a.wav"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if f.hits != 1 {
		t.Errorf("expected 1 upload request, got: %d", f.hits)
	}

	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel &&
			e.Data["source_filepath"] == p &&
			e.Data["destination_filepath"] == "recording/a.wav" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error log with source_filepath and destination_filepath")
	}
}

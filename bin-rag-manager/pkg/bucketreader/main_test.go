package bucketreader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
)

// newEmulatorClient returns a storage client that talks to the given test server only.
func newEmulatorClient(t *testing.T, srv *httptest.Server) *storage.Client {
	t.Helper()

	t.Setenv("STORAGE_EMULATOR_HOST", strings.TrimPrefix(srv.URL, "http://"))

	client, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("could not create the storage client. err: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func Test_DownloadToTempFile(t *testing.T) {
	tests := []struct {
		name string

		bucket   string
		filepath string

		status int
		body   string

		tmpDirMissing bool

		expectPath string
		expectBody string
		expectErr  string
	}{
		{
			name: "downloads the object into a temp file",

			bucket:   "test-bucket",
			filepath: "docs/guide.txt",

			status: http.StatusOK,
			body:   "hello rag",

			expectPath: "/test-bucket/docs/guide.txt",
			expectBody: "hello rag",
		},
		{
			name: "missing object is reported with bucket and path",

			bucket:   "test-bucket",
			filepath: "docs/missing.txt",

			status: http.StatusNotFound,
			body:   "not found",

			expectPath: "/test-bucket/docs/missing.txt",
			expectErr:  "could not open GCS object test-bucket/docs/missing.txt",
		},
		{
			name: "temp file creation failure",

			bucket:   "test-bucket",
			filepath: "docs/guide.txt",

			status: http.StatusOK,
			body:   "hello rag",

			tmpDirMissing: true,

			expectPath: "/test-bucket/docs/guide.txt",
			expectErr:  "could not create temp file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if tt.tmpDirMissing {
				t.Setenv("TMPDIR", filepath.Join(tmpDir, "absent"))
			} else {
				t.Setenv("TMPDIR", tmpDir)
			}

			var mu sync.Mutex
			var gotPaths []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPaths = append(gotPaths, r.URL.Path)
				mu.Unlock()

				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			reader := NewBucketReader(newEmulatorClient(t, srv))

			res, err := reader.DownloadToTempFile(context.Background(), tt.bucket, tt.filepath)

			mu.Lock()
			paths := append([]string(nil), gotPaths...)
			mu.Unlock()
			if len(paths) != 1 || paths[0] != tt.expectPath {
				t.Errorf("Wrong match. expect request path: %v, got: %v", tt.expectPath, paths)
			}

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if res != "" {
					t.Errorf("Wrong match. expect empty path on error, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if filepath.Dir(res) != tmpDir {
				t.Errorf("Wrong match. expect temp file under: %v, got: %v", tmpDir, res)
			}

			content, errRead := os.ReadFile(res)
			if errRead != nil {
				t.Fatalf("could not read the downloaded file. err: %v", errRead)
			}
			if string(content) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(content))
			}
		})
	}
}

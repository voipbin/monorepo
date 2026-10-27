package raghandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// covStreamBytes writes n bytes of filler to w in 1 MiB pieces and stops early when the client goes away.
func covStreamBytes(w http.ResponseWriter, n int64) {
	piece := bytes.Repeat([]byte("a"), 1<<20)
	for n > 0 {
		size := int64(len(piece))
		if n < size {
			size = n
		}
		if _, err := w.Write(piece[:size]); err != nil {
			return
		}
		n -= size
	}
}

// covIsolateTmp points the temp dir at a per-test directory so leftovers can be asserted.
func covIsolateTmp(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	return dir
}

func covDirEntries(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read dir: %v", err)
	}
	return len(entries)
}

func Test_documentDownloadURL_success(t *testing.T) {
	tests := []struct {
		name string

		handler http.HandlerFunc
		path    string

		expectSize        int64
		expectBody        string
		expectFilename    string
		expectContentType string
	}{
		{
			name: "small body with content type and file name from the url path",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("hello rag"))
			},
			path:              "/files/doc.txt",
			expectSize:        9,
			expectBody:        "hello rag",
			expectFilename:    "doc.txt",
			expectContentType: "text/plain",
		},
		{
			name: "declared content length exactly at the limit is accepted",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/pdf")
				w.Header().Set("Content-Length", strconv.Itoa(maxFileSize))
				covStreamBytes(w, maxFileSize)
			},
			path:              "/big.bin",
			expectSize:        maxFileSize,
			expectFilename:    "big.bin",
			expectContentType: "application/pdf",
		},
		{
			name: "chunked body exactly at the limit is accepted",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				covStreamBytes(w, maxFileSize)
			},
			path:              "/chunked.bin",
			expectSize:        maxFileSize,
			expectFilename:    "chunked.bin",
			expectContentType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := covIsolateTmp(t)

			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			h := &ragHandler{}
			path, filename, contentType, err := h.documentDownloadURL(context.Background(), srv.URL+tt.path)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if filepath.Dir(path) != tmp {
				t.Errorf("Wrong match. expect dir: %v, got: %v", tmp, filepath.Dir(path))
			}
			if covDirEntries(t, tmp) != 1 {
				t.Errorf("Wrong match. expect: exactly the returned file, got entries: %d", covDirEntries(t, tmp))
			}
			fi, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Wrong match. expect: file exists, got: %v", err)
			}
			if fi.Size() != tt.expectSize {
				t.Errorf("Wrong match. expect size: %d, got: %d", tt.expectSize, fi.Size())
			}
			if tt.expectBody != "" {
				body, _ := os.ReadFile(path)
				if string(body) != tt.expectBody {
					t.Errorf("Wrong match. expect: %q, got: %q", tt.expectBody, string(body))
				}
			}
			if filename != tt.expectFilename {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectFilename, filename)
			}
			if contentType != tt.expectContentType {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectContentType, contentType)
			}
		})
	}
}

func Test_documentDownloadURL_failure(t *testing.T) {
	tests := []struct {
		name string

		handler http.HandlerFunc
		rawURL  string // used instead of the server url when set
		closed  bool   // close the server before the call
		badTmp  bool   // point the temp dir at a missing directory

		expectErr string
	}{
		{
			name:      "url that cannot be parsed",
			rawURL:    "http://[::1",
			expectErr: "invalid URL",
		},
		{
			name:      "scheme other than http and https",
			rawURL:    "file:///etc/hostname",
			expectErr: "unsupported URL scheme",
		},
		{
			name: "non 200 status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectErr: "URL returned status 404",
		},
		{
			name: "declared content length one byte over the limit",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(maxFileSize+1))
				w.WriteHeader(http.StatusOK)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			},
			expectErr: "content length",
		},
		{
			name: "chunked body one byte over the limit",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				covStreamBytes(w, maxFileSize+1)
			},
			expectErr: "download size",
		},
		{
			name: "connection dropped before the declared length is sent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte("partial"))
			},
			expectErr: "could not download URL",
		},
		{
			name:      "server is not reachable",
			closed:    true,
			expectErr: "could not fetch URL",
		},
		{
			name: "temp file cannot be created",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			badTmp:    true,
			expectErr: "could not create temp file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := covIsolateTmp(t)
			if tt.badTmp {
				t.Setenv("TMPDIR", filepath.Join(tmp, "missing"))
			}

			target := tt.rawURL
			if target == "" {
				handler := tt.handler
				if handler == nil {
					handler = func(http.ResponseWriter, *http.Request) {}
				}
				srv := httptest.NewServer(handler)
				t.Cleanup(srv.Close)
				target = srv.URL + "/doc.txt"
				if tt.closed {
					srv.Close()
				}
			}

			h := &ragHandler{}
			path, filename, contentType, err := h.documentDownloadURL(context.Background(), target)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: %q error, got: %v", tt.expectErr, err)
			}
			if path != "" || filename != "" || contentType != "" {
				t.Errorf("Wrong match. expect: empty results, got: %q/%q/%q", path, filename, contentType)
			}
			if covDirEntries(t, tmp) != 0 {
				t.Errorf("Wrong match. expect: no leftover temp file, got entries: %d", covDirEntries(t, tmp))
			}
		})
	}
}

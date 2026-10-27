package siphandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
)

// fakeGCSServer is a minimal loopback fake of the GCS JSON API (object listing) and of the object
// download path. It is wired in through STORAGE_EMULATOR_HOST, so the tests that use it do not run in
// parallel.
type fakeGCSServer struct {
	srv *httptest.Server

	mu sync.Mutex

	// listPages maps the request page token ("" for the first page) to the response body.
	listPages map[string]string
	// listStatus, when not zero, is returned for every listing request instead of a page.
	listStatus int
	// objects maps "<bucket>/<object>" to the body of the object.
	objects map[string]string
	// truncated makes the object response declare more bytes than it sends.
	truncated bool

	listRequests     []string // "<bucket>|<prefix>|<page token>" of every listing request
	downloadRequests []string // path of every download request
}

func newFakeGCSServer(t *testing.T) *fakeGCSServer {
	t.Helper()

	f := &fakeGCSServer{
		listPages: map[string]string{},
		objects:   map[string]string{},
	}

	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		if strings.HasPrefix(r.URL.Path, "/storage/v1/b/") {
			bucket := strings.TrimPrefix(r.URL.Path, "/storage/v1/b/")
			bucket = strings.TrimSuffix(bucket, "/o")
			f.listRequests = append(f.listRequests, fmt.Sprintf("%s|%s|%s", bucket, r.URL.Query().Get("prefix"), r.URL.Query().Get("pageToken")))

			w.Header().Set("Content-Type", "application/json")
			if f.listStatus != 0 {
				w.WriteHeader(f.listStatus)
				_, _ = fmt.Fprintf(w, `{"error":{"code":%d,"message":"simulated"}}`, f.listStatus)
				return
			}
			page, ok := f.listPages[r.URL.Query().Get("pageToken")]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"no such page"}}`)
				return
			}
			_, _ = fmt.Fprint(w, page)
			return
		}

		f.downloadRequests = append(f.downloadRequests, r.URL.Path)
		body, ok := f.objects[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, "not found")
			return
		}
		if f.truncated {
			// declare 1000 bytes and send 3, for the first request and for the range retry alike
			w.Header().Set("Content-Length", "1000")
			_, _ = w.Write([]byte(body[:3]))
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(f.srv.Close)

	t.Setenv("STORAGE_EMULATOR_HOST", f.srv.URL)

	return f
}

func (f *fakeGCSServer) newClient(t *testing.T) *storage.Client {
	t.Helper()

	c, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func (f *fakeGCSServer) lists() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.listRequests...)
}

func (f *fakeGCSServer) downloads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.downloadRequests...)
}

// failingWriter fails every write with the configured error.
type failingWriter struct {
	err error
}

func (w *failingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func Test_NewGCSReader(t *testing.T) {
	f := newFakeGCSServer(t)
	f.listPages[""] = `{"items":[{"name":"a/one.pcap"}]}`

	client := f.newClient(t)
	reader := NewGCSReader(client, "test-bucket-a")

	res, err := reader.ListObjects(context.Background(), "a/")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, []string{"a/one.pcap"}) {
		t.Errorf("Wrong match. expect: [a/one.pcap], got: %v", res)
	}

	// the reader is bound to the bucket it was created with
	if got, expect := f.lists(), []string{"test-bucket-a|a/|"}; !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
	}
}

func Test_gcsReaderImpl_ListObjects(t *testing.T) {
	tests := []struct {
		name string

		pages  map[string]string
		status int
		prefix string

		expectRes      []string
		expectErr      string
		expectRequests []string
	}{
		{
			name: "objects are returned in the order of the server",
			pages: map[string]string{
				"": `{"items":[{"name":"rtp-recordings/call-1-b.pcap"},{"name":"rtp-recordings/call-1-a.pcap"}]}`,
			},
			prefix: "rtp-recordings/call-1-",

			expectRes:      []string{"rtp-recordings/call-1-b.pcap", "rtp-recordings/call-1-a.pcap"},
			expectRequests: []string{"test-bucket|rtp-recordings/call-1-|"},
		},
		{
			name: "all pages are read",
			pages: map[string]string{
				"":      `{"items":[{"name":"p/1"},{"name":"p/2"}],"nextPageToken":"tok-2"}`,
				"tok-2": `{"items":[{"name":"p/3"}]}`,
			},
			prefix: "p/",

			expectRes:      []string{"p/1", "p/2", "p/3"},
			expectRequests: []string{"test-bucket|p/|", "test-bucket|p/|tok-2"},
		},
		{
			name: "no object returns an empty list that is not nil",
			pages: map[string]string{
				"": `{}`,
			},
			prefix: "none/",

			expectRes:      []string{},
			expectRequests: []string{"test-bucket|none/|"},
		},
		{
			name:   "listing failure is wrapped with the prefix",
			status: http.StatusForbidden, // a status the client library does not retry
			prefix: "denied/",

			expectErr:      "could not list GCS objects with prefix denied/",
			expectRequests: []string{"test-bucket|denied/|"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGCSServer(t)
			f.listPages = tt.pages
			f.listStatus = tt.status

			reader := &gcsReaderImpl{client: f.newClient(t), bucketName: "test-bucket"}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			res, err := reader.ListObjects(ctx, tt.prefix)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				// the cause stays reachable through the error chain, not only in the message text
				cause := errors.Unwrap(err)
				if cause == nil || !strings.Contains(cause.Error(), "403") {
					t.Errorf("Wrong match. expect the unwrapped cause to carry the 403 status, got: %v", cause)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res == nil || !reflect.DeepEqual(res, tt.expectRes) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
				}
			}

			if got := f.lists(); !reflect.DeepEqual(got, tt.expectRequests) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRequests, got)
			}
		})
	}
}

func Test_gcsReaderImpl_DownloadObject(t *testing.T) {
	boom := errors.New("simulated sink failure")

	tests := []struct {
		name string

		object    string
		truncated bool
		dest      func(buf *bytes.Buffer) io.Writer

		expectBody string
		expectErr  string
		expectIs   error
	}{
		{
			name:   "object body is written to the destination",
			object: "rtp-recordings/call-1-a.pcap",
			dest:   func(buf *bytes.Buffer) io.Writer { return buf },

			expectBody: "pcap-bytes-of-object-a",
		},
		{
			name:   "missing object returns the open error",
			object: "rtp-recordings/missing.pcap",
			dest:   func(buf *bytes.Buffer) io.Writer { return buf },

			expectErr: "could not open GCS object rtp-recordings/missing.pcap",
			expectIs:  storage.ErrObjectNotExist,
		},
		{
			name:   "destination failure returns the download error",
			object: "rtp-recordings/call-1-a.pcap",
			dest: func(_ *bytes.Buffer) io.Writer {
				return &failingWriter{err: boom}
			},

			expectErr: "could not download GCS object rtp-recordings/call-1-a.pcap",
			expectIs:  boom,
		},
		{
			name:      "truncated body returns the download error",
			object:    "rtp-recordings/call-1-a.pcap",
			truncated: true,
			dest:      func(buf *bytes.Buffer) io.Writer { return buf },

			expectErr: "could not download GCS object rtp-recordings/call-1-a.pcap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGCSServer(t)
			f.objects["test-bucket/rtp-recordings/call-1-a.pcap"] = "pcap-bytes-of-object-a"
			f.truncated = tt.truncated

			reader := &gcsReaderImpl{client: f.newClient(t), bucketName: "test-bucket"}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			var buf bytes.Buffer
			err := reader.DownloadObject(ctx, tt.object, tt.dest(&buf))
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
					t.Errorf("Wrong match. expect errors.Is: %v, got: %v", tt.expectIs, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if buf.String() != tt.expectBody {
					t.Errorf("Wrong match. expect: %s, got: %s", tt.expectBody, buf.String())
				}
			}

			if got := f.downloads(); len(got) == 0 || got[0] != "/test-bucket/"+tt.object {
				t.Errorf("Wrong match. expect the first download path: /test-bucket/%s, got: %v", tt.object, got)
			}
		})
	}
}

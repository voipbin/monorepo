package gcsuploader

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// receivedUpload is what the fake GCS endpoint observed.
type receivedUpload struct {
	path        string
	objectName  string
	contentType string
	body        string
}

// newTestUploader returns an uploader wired to a loopback fake of the GCS
// JSON API. Every request is handed to the returned channel and answered with
// status (a 200 response carries a minimal object resource).
func newTestUploader(t *testing.T, status int) (*gcsUploader, <-chan receivedUpload) {
	t.Helper()

	got := make(chan receivedUpload, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := receivedUpload{path: r.URL.Path}

		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err == nil && params["boundary"] != "" {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for i := 0; ; i++ {
				part, err := mr.NextPart()
				if err != nil {
					break
				}
				data, _ := io.ReadAll(part)
				if i == 0 {
					rec.objectName = jsonField(string(data), "name")
					rec.contentType = jsonField(string(data), "contentType")
				} else {
					rec.body = string(data)
				}
			}
		}
		select {
		case got <- rec:
		default:
		}

		if status != http.StatusOK {
			http.Error(w, "injected failure", status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"` + rec.objectName + `","bucket":"test-bucket"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(srv.URL+"/storage/v1/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("could not create storage client: %v", err)
	}

	return &gcsUploader{client: client, bucketName: "test-bucket"}, got
}

// jsonField extracts a string field from a small JSON object.
func jsonField(doc, key string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(doc), &m); err != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func Test_Upload(t *testing.T) {
	tests := []struct {
		name string

		content    string
		objectPath string

		expectURI         string
		expectContentType string
	}{
		{
			name: "normal",

			content:    "synthetic pcap bytes",
			objectPath: "rtp-recordings/call-1.pcap",

			expectURI:         "gs://test-bucket/rtp-recordings/call-1.pcap",
			expectContentType: "application/vnd.tcpdump.pcap",
		},
		{
			name: "empty file",

			content:    "",
			objectPath: "rtp-recordings/empty.pcap",

			expectURI:         "gs://test-bucket/rtp-recordings/empty.pcap",
			expectContentType: "application/vnd.tcpdump.pcap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, got := newTestUploader(t, http.StatusOK)

			local := filepath.Join(t.TempDir(), "in.pcap")
			if err := os.WriteFile(local, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("could not write fixture: %v", err)
			}

			uri, err := u.Upload(local, tt.objectPath)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if uri != tt.expectURI {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectURI, uri)
			}

			select {
			case rec := <-got:
				if !strings.Contains(rec.path, "/b/test-bucket/o") {
					t.Errorf("Wrong upload path: %s", rec.path)
				}
				if rec.objectName != tt.objectPath {
					t.Errorf("Wrong match. expect: %s, got: %s", tt.objectPath, rec.objectName)
				}
				if rec.contentType != tt.expectContentType {
					t.Errorf("Wrong match. expect: %s, got: %s", tt.expectContentType, rec.contentType)
				}
				if rec.body != tt.content {
					t.Errorf("Wrong match. expect: %q, got: %q", tt.content, rec.body)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("fake GCS endpoint received no request")
			}
		})
	}
}

func Test_Upload_errors(t *testing.T) {
	tests := []struct {
		name string

		status       int
		createLocal  bool
		expectServed bool
		expectErr    string
	}{
		{
			name: "local file missing never reaches GCS",

			status:       http.StatusOK,
			createLocal:  false,
			expectServed: false,
			expectErr:    "could not open file",
		},
		{
			name: "server error fails the upload",

			status:       http.StatusInternalServerError,
			createLocal:  true,
			expectServed: true,
			expectErr:    "could not finalize GCS upload",
		},
		{
			name: "forbidden fails the upload",

			status:       http.StatusForbidden,
			createLocal:  true,
			expectServed: true,
			expectErr:    "could not finalize GCS upload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, got := newTestUploader(t, tt.status)

			local := filepath.Join(t.TempDir(), "in.pcap")
			if tt.createLocal {
				if err := os.WriteFile(local, []byte("synthetic"), 0o600); err != nil {
					t.Fatalf("could not write fixture: %v", err)
				}
			}

			uri, err := u.Upload(local, "rtp-recordings/x.pcap")
			if err == nil {
				t.Fatalf("Wrong match. expect: err, got: ok (%s)", uri)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErr, err)
			}
			if uri != "" {
				t.Errorf("Wrong match. expect empty uri on error, got: %s", uri)
			}

			served := len(got) > 0
			if served != tt.expectServed {
				t.Errorf("Wrong match. expect served: %v, got: %v", tt.expectServed, served)
			}
		})
	}
}

func Test_Close(t *testing.T) {
	u, _ := newTestUploader(t, http.StatusOK)

	if err := u.Close(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_New(t *testing.T) {
	tests := []struct {
		name string

		emulatorHost string
		credentials  func(t *testing.T) string // returns the credentials file path, empty for none

		expectErr string
	}{
		{
			name: "emulator host needs no credentials",

			emulatorHost: "127.0.0.1:1",
		},
		{
			name: "invalid credentials file is rejected",

			emulatorHost: "",
			credentials: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "credentials.json")
				if err := os.WriteFile(p, []byte("not a credentials file"), 0o600); err != nil {
					t.Fatalf("could not write fixture: %v", err)
				}
				return p
			},

			expectErr: "could not create GCS client",
		},
		{
			name: "missing credentials file is rejected",

			emulatorHost: "",
			credentials: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.json")
			},

			expectErr: "could not create GCS client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Both variables are always set so that local credentials or an
			// emulator configured on the host cannot change the outcome.
			t.Setenv("STORAGE_EMULATOR_HOST", tt.emulatorHost)
			cred := ""
			if tt.credentials != nil {
				cred = tt.credentials(t)
			}
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", cred)

			u, err := New("test-bucket")
			if tt.expectErr != "" {
				if err == nil {
					t.Fatalf("Wrong match. expect: error containing %q, got: ok", tt.expectErr)
				}
				if !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
				if u != nil {
					t.Errorf("Wrong match. expect: nil uploader, got: %v", u)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			t.Cleanup(func() { _ = u.Close() })

			impl, ok := u.(*gcsUploader)
			if !ok {
				t.Fatalf("Wrong match. expect: *gcsUploader, got: %T", u)
			}
			if impl.bucketName != "test-bucket" {
				t.Errorf("Wrong match. expect: test-bucket, got: %s", impl.bucketName)
			}
		})
	}
}

func Test_Upload_copyFailure(t *testing.T) {
	tests := []struct {
		name string

		status int
	}{
		{
			name: "source cannot be read, endpoint healthy",

			status: http.StatusOK,
		},
		{
			// Same assertions as above; the endpoint failure additionally
			// drives the writer-close warning branch, which only logs.
			name: "source cannot be read, endpoint failing",

			status: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := newTestUploader(t, tt.status)

			// A directory opens successfully and fails on the first read.
			src := t.TempDir()

			uri, err := u.Upload(src, "rtp-recordings/dir.pcap")
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok (%s)", uri)
			}
			if !strings.Contains(err.Error(), "could not upload to GCS") {
				t.Errorf("Wrong match. expect: could not upload to GCS, got: %v", err)
			}
			if uri != "" {
				t.Errorf("Wrong match. expect empty uri on error, got: %s", uri)
			}
		})
	}
}

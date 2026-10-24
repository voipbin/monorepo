package filehandler

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"

	"monorepo/bin-storage-manager/models/file"
)

// wavHeaderWithFmtSize returns a RIFF/WAVE header followed by a fmt chunk header that
// declares the given size, with the chunk body left empty.
func wavHeaderWithFmtSize(size uint32) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, size)

	return buf.Bytes()
}

// wavWithFmtChunk returns a complete minimal WAV whose fmt chunk is the given size.
func wavWithFmtChunk(size uint32) []byte {
	buf := &bytes.Buffer{}
	buf.Write(wavHeaderWithFmtSize(size))

	body := make([]byte, size)
	if size >= 16 {
		binary.LittleEndian.PutUint16(body[0:2], 1)    // PCM
		binary.LittleEndian.PutUint16(body[2:4], 1)    // channels
		binary.LittleEndian.PutUint32(body[4:8], 8000) // sample rate
		binary.LittleEndian.PutUint16(body[14:16], 16) // bits per sample
	}
	buf.Write(body)

	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(4))
	buf.Write([]byte{0, 0, 0, 0})

	return buf.Bytes()
}

func Test_readWavHeader_fmtChunkSize(t *testing.T) {
	tests := []struct {
		name   string
		size   uint32
		expect bool
	}{
		{"below the minimum is rejected", 15, false},
		{"minimum is accepted", 16, true},
		{"extended is accepted", 18, true},
		{"extensible is accepted", 40, true},
		{"maximum is accepted", 1024, true},
		{"above the maximum is rejected", 1025, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := readWavHeader(bytes.NewReader(wavWithFmtChunk(tt.size)))
			if ok != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, ok)
			}
		})
	}
}

func Test_readWavHeader_oversizedChunkDoesNotAllocate(t *testing.T) {
	const declared = uint32(1 << 30)
	const allowed = 1 << 20

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_, _, ok := readWavHeader(bytes.NewReader(wavHeaderWithFmtSize(declared)))

	runtime.ReadMemStats(&after)

	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}
	if delta := after.TotalAlloc - before.TotalAlloc; delta > allowed {
		t.Errorf("Wrong match. expect: at most %d bytes allocated, got: %d", allowed, delta)
	}
}

// storageFake is a small GCS stand-in for the compress and move tests. Object keys are
// "<bucket>/<path>". It serves object attributes and media, accepts uploads, and records
// deletions.
type storageFake struct {
	mu sync.Mutex

	attrs        map[string]bool   // objects whose attributes can be read
	media        map[string]string // objects whose bytes can be read
	uploadStatus int               // status code of the upload request
	deleted      []string
	uploaded     int
	rewrites     int // copy requests received
}

func newStorageFake(t *testing.T, f *storageFake) *storage.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		notFound := func() {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"Not Found"}}`)
		}

		path := r.URL.Path

		// copy
		if r.Method == http.MethodPost && strings.Contains(path, "/rewriteTo/") {
			f.rewrites++
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"kind":"storage#rewriteResponse","totalBytesRewritten":"1","objectSize":"1","done":true,"resource":{"kind":"storage#object","bucket":"media","name":"bin/out","size":"1","generation":"1","metageneration":"1"}}`)
			return
		}

		// upload
		if r.Method == http.MethodPost && strings.Contains(path, "/upload/") {
			_, _ = io.Copy(io.Discard, r.Body)
			f.uploaded++
			if f.uploadStatus >= 400 {
				w.WriteHeader(f.uploadStatus)
				_, _ = fmt.Fprint(w, `{"error":{"code":500,"message":"simulated"}}`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"kind":"storage#object","bucket":"tmp","name":"zip/out.zip","size":"0","generation":"1","metageneration":"1"}`)
			return
		}

		// JSON API: /b/<bucket>/o/<object>
		if idx := strings.Index(path, "/b/"); idx >= 0 {
			bucket, object, ok := strings.Cut(path[idx+len("/b/"):], "/o/")
			if !ok {
				notFound()
				return
			}
			key := bucket + "/" + object

			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodDelete:
				f.deleted = append(f.deleted, key)
				w.WriteHeader(http.StatusNoContent)
			case r.Method == http.MethodGet && f.attrs[key]:
				_, _ = fmt.Fprintf(w, `{"kind":"storage#object","bucket":%q,"name":%q,"size":"1","mediaLink":"https://storage.test/download/%s","generation":"1","metageneration":"1"}`, bucket, object, key)
			default:
				notFound()
			}
			return
		}

		// media download: /<bucket>/<object>
		if r.Method == http.MethodGet {
			if body, ok := f.media[strings.TrimPrefix(path, "/")]; ok {
				_, _ = io.WriteString(w, body)
				return
			}
		}
		notFound()
	}))
	t.Cleanup(srv.Close)

	t.Setenv("STORAGE_EMULATOR_HOST", srv.URL)

	c, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func (f *storageFake) deletedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.deleted...)
}

func Test_bucketfileCompressFiles_sourceReadFailureIsReturned(t *testing.T) {
	fake := &storageFake{
		media:        map[string]string{"media/a.wav": "aaaa"},
		uploadStatus: 200,
	}
	h := &fileHandler{client: newStorageFake(t, fake), bucketTmp: "tmp"}

	files := []*file.File{
		{BucketName: "media", Filepath: "a.wav", Filename: "a.wav"},
		{BucketName: "media", Filepath: "missing.wav", Filename: "missing.wav"},
	}

	err := h.bucketfileCompressFiles(context.Background(), "zip/out.zip", files)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	deleted := fake.deletedKeys()
	if len(deleted) != 1 || deleted[0] != "tmp/zip/out.zip" {
		t.Errorf("Wrong match. expect the result object to be deleted, got: %v", deleted)
	}
}

func Test_bucketfileCompressFiles_uploadFailureIsReturned(t *testing.T) {
	fake := &storageFake{
		media:        map[string]string{"media/a.wav": "aaaa"},
		uploadStatus: http.StatusInternalServerError,
	}
	h := &fileHandler{client: newStorageFake(t, fake), bucketTmp: "tmp"}

	files := []*file.File{{BucketName: "media", Filepath: "a.wav", Filename: "a.wav"}}

	done := make(chan error, 1)
	go func() { done <- h.bucketfileCompressFiles(context.Background(), "zip/out.zip", files) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("Wrong match. the call did not return in time")
	}

	deleted := fake.deletedKeys()
	if len(deleted) != 1 || deleted[0] != "tmp/zip/out.zip" {
		t.Errorf("Wrong match. expect the result object to be deleted, got: %v", deleted)
	}
}

func Test_bucketfileCompressFiles_completeResultIsKept(t *testing.T) {
	tests := []struct {
		name  string
		files []*file.File
	}{
		{"with sources", []*file.File{{BucketName: "media", Filepath: "a.wav", Filename: "a.wav"}}},
		{"without sources", []*file.File{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &storageFake{
				media:        map[string]string{"media/a.wav": "aaaa"},
				uploadStatus: 200,
			}
			h := &fileHandler{client: newStorageFake(t, fake), bucketTmp: "tmp"}

			if err := h.bucketfileCompressFiles(context.Background(), "zip/out.zip", tt.files); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if deleted := fake.deletedKeys(); len(deleted) != 0 {
				t.Errorf("Wrong match. expect no deletion, got: %v", deleted)
			}
			if fake.uploaded == 0 {
				t.Errorf("Wrong match. expect the result to be uploaded")
			}
		})
	}
}

func Test_bucketfileMove_existingDestinationIsRefused(t *testing.T) {
	fake := &storageFake{
		attrs: map[string]bool{"tmp/in.wav": true, "media/bin/out": true},
	}
	h := &fileHandler{client: newStorageFake(t, fake)}

	res, err := h.bucketfileMove(context.Background(), "tmp", "in.wav", "media", "bin/out")
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
	if err != nil && !strings.Contains(err.Error(), "destination already exists") {
		t.Errorf("Wrong match. expect the destination refusal, got: %v", err)
	}
	if fake.rewrites != 0 {
		t.Errorf("Wrong match. expect no copy request, got: %d", fake.rewrites)
	}
	if deleted := fake.deletedKeys(); len(deleted) != 0 {
		t.Errorf("Wrong match. the source must stay in place, got deleted: %v", deleted)
	}
}

func Test_bucketfileMove_freeDestinationIsCopied(t *testing.T) {
	fake := &storageFake{
		attrs: map[string]bool{"tmp/in.wav": true},
	}
	h := &fileHandler{client: newStorageFake(t, fake)}

	res, err := h.bucketfileMove(context.Background(), "tmp", "in.wav", "media", "bin/out")
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: attrs/nil, got: %v/%v", res, err)
	}
	if fake.rewrites != 1 {
		t.Errorf("Wrong match. expect one copy request, got: %d", fake.rewrites)
	}
	if deleted := fake.deletedKeys(); len(deleted) != 1 || deleted[0] != "tmp/in.wav" {
		t.Errorf("Wrong match. expect the source to be removed, got: %v", deleted)
	}
}

func Test_DownloadURIGet_usesTheGivenBucket(t *testing.T) {
	fake := &storageFake{
		attrs: map[string]bool{"media/zip/a.zip": true},
	}
	accessID, key := newTestSigningKey(t)
	h := &fileHandler{
		client:     newStorageFake(t, fake),
		bucketTmp:  "tmp",
		accessID:   accessID,
		privateKey: key,
	}

	bucketURI, _, err := h.DownloadURIGet(context.Background(), "media", "zip/a.zip", time.Hour)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if bucketURI != "https://storage.test/download/media/zip/a.zip" {
		t.Errorf("Wrong match. got bucket uri: %q", bucketURI)
	}
}

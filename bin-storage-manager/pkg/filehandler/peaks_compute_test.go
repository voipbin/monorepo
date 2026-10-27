package filehandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-storage-manager/models/file"
)

// Test_PeaksComputeByFile_readFailure fixes the current behavior: unlike a decode failure, which
// yields empty peaks without an error, a failure to open the object is returned. It checks that an object that cannot be opened is
// reported as an error with the bucket and path in it, and that no peaks are returned.
func Test_PeaksComputeByFile_readFailure(t *testing.T) {
	fake := &storageFake{media: map[string]string{}}
	h := &fileHandler{client: newStorageFake(t, fake)}

	f := &file.File{BucketName: "test-media", Filepath: "bin/missing"}
	peaks, duration, err := h.PeaksComputeByFile(context.Background(), f, 10)
	if err == nil || !strings.Contains(err.Error(), "could not open the file reader") {
		t.Fatalf("Wrong match. expect: the open failure, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test-media") || !strings.Contains(err.Error(), "bin/missing") {
		t.Errorf("Wrong match. expect the bucket and path in the error, got: %v", err)
	}
	if peaks != nil || duration != 0 {
		t.Errorf("Wrong match. expect: nil/0, got: %v/%v", peaks, duration)
	}
}

// Test_PeaksComputeByFile checks the peaks of a stored WAV against values computed by hand:
// 100 samples of 8192 (0.25 of full scale) followed by 100 samples of 16384 (0.5), 8000 Hz.
func Test_PeaksComputeByFile(t *testing.T) {
	samples := make([]int16, 0, 200)
	for i := 0; i < 100; i++ {
		samples = append(samples, 8192)
	}
	for i := 0; i < 100; i++ {
		samples = append(samples, 16384)
	}
	wav := buildTestWav(8000, samples)

	long := make([]int16, 3000)
	for i := range long {
		long[i] = 16384
	}
	longWav := buildTestWav(8000, long)

	tests := []struct {
		name string

		wav         string
		bucketCount int

		expectPeaks    []float64
		expectLen      int
		expectDuration float64
	}{
		{
			name:           "two buckets split at the amplitude change",
			wav:            string(wav),
			bucketCount:    2,
			expectPeaks:    []float64{0.25, 0.5},
			expectLen:      2,
			expectDuration: 200.0 / 8000.0,
		},
		{
			name:           "zero bucket count falls back to the default count",
			wav:            string(longWav),
			bucketCount:    0,
			expectLen:      peakBucketCount,
			expectDuration: 3000.0 / 8000.0,
		},
		{
			name:           "given bucket count is used when it is smaller than the default",
			wav:            string(longWav),
			bucketCount:    7,
			expectLen:      7,
			expectDuration: 3000.0 / 8000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &storageFake{media: map[string]string{"test-media/bin/a.wav": tt.wav}}
			h := &fileHandler{client: newStorageFake(t, fake)}

			f := &file.File{BucketName: "test-media", Filepath: "bin/a.wav"}
			peaks, duration, err := h.PeaksComputeByFile(context.Background(), f, tt.bucketCount)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(peaks) != tt.expectLen {
				t.Fatalf("Wrong match. expect length: %d, got: %d", tt.expectLen, len(peaks))
			}
			if duration != tt.expectDuration {
				t.Errorf("Wrong match. expect duration: %v, got: %v", tt.expectDuration, duration)
			}
			if tt.expectPeaks != nil {
				for i, expect := range tt.expectPeaks {
					if peaks[i] != expect {
						t.Errorf("Wrong match. bucket %d expect: %v, got: %v", i, expect, peaks[i])
					}
				}
			} else {
				// every bucket of the constant signal has the same peak (16384 / 32768)
				for i, p := range peaks {
					if p != 0.5 {
						t.Fatalf("Wrong match. bucket %d expect: 0.5, got: %v", i, p)
					}
				}
			}
		})
	}
}

// Test_CompressCreate_existingArchiveIsReused checks that the archive name depends on the
// set of file ids and not on their order, and that an archive which already exists in the
// tmp bucket is returned without building it again.
func Test_CompressCreate_existingArchiveIsReused(t *testing.T) {
	idA := uuid.FromStringOrNil("a1b2c3d4-0003-4000-8000-00000000000a")
	idB := uuid.FromStringOrNil("a1b2c3d4-0003-4000-8000-00000000000b")

	filesAB := []*file.File{{Identity: commonidentity.Identity{ID: idA}}, {Identity: commonidentity.Identity{ID: idB}}}
	filesBA := []*file.File{{Identity: commonidentity.Identity{ID: idB}}, {Identity: commonidentity.Identity{ID: idA}}}

	// the archive for this id set exists; any build would upload and is counted by the fake.
	archive := createZipFilepathHash([]string{idA.String(), idB.String()})
	fake := &storageFake{attrs: map[string]bool{"test-tmp/" + archive: true}}
	h := &fileHandler{client: newStorageFake(t, fake), bucketTmp: "test-tmp"}

	for name, files := range map[string][]*file.File{"ab": filesAB, "ba": filesBA} {
		bucket, path, err := h.CompressCreate(context.Background(), files)
		if err != nil {
			t.Fatalf("Wrong match. order: %s, expect: ok, got: %v", name, err)
		}
		if bucket != "test-tmp" || path != archive {
			t.Errorf("Wrong match. order: %s, expect: test-tmp/%s, got: %s/%s", name, archive, bucket, path)
		}
	}

	if !strings.HasPrefix(archive, bucketDirectoryTmp+"/") || !strings.HasSuffix(archive, ".zip") {
		t.Errorf("Wrong match. expect an archive path under the tmp directory, got: %s", archive)
	}
	if fake.uploaded != 0 || fake.rewrites != 0 {
		t.Errorf("Wrong match. expect no build for an existing archive, uploads: %d, copies: %d", fake.uploaded, fake.rewrites)
	}
}

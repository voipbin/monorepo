package storagehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/models/recordingpeak"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_RecordingPeaks(t *testing.T) {
	ctx := context.Background()
	recordingID := uuid.FromStringOrNil("9f4c8d41-0000-0000-0000-000000000001")
	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: recordingID,
	}
	fileIn := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9f4c8d41-0000-0000-0000-000000000011")}, Filename: "in.wav"}
	fileOut := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9f4c8d41-0000-0000-0000-000000000012")}, Filename: "out.wav"}

	cachedPeaks := map[string]recordingpeak.RecordingFilePeak{
		"cached.wav": {Peaks: []float64{0.9}, Duration: 3},
	}

	tests := []struct {
		name string

		expect func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler)

		expectRes map[string]recordingpeak.RecordingFilePeak
		expectErr bool
	}{
		{
			name: "cache hit skips the file lookup and the computation",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(cachedPeaks, nil)
			},
			expectRes: cachedPeaks,
		},
		{
			name: "a nil cached map is treated as a miss",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(nil, nil)
				mf.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{fileIn}, nil)
				mf.EXPECT().PeaksComputeByFile(ctx, fileIn, 0).Return([]float64{0.25, 0.5}, 2.5, nil)
				mcache.EXPECT().RecordingPeaksSet(ctx, recordingID, map[string]recordingpeak.RecordingFilePeak{
					"in.wav": {Peaks: []float64{0.25, 0.5}, Duration: 2.5},
				}).Return(nil)
			},
			expectRes: map[string]recordingpeak.RecordingFilePeak{
				"in.wav": {Peaks: []float64{0.25, 0.5}, Duration: 2.5},
			},
		},
		{
			name: "a failing file degrades to empty peaks and the others stay intact",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(nil, redis.Nil)
				mf.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{fileIn, fileOut}, nil)
				mf.EXPECT().PeaksComputeByFile(ctx, fileIn, 0).Return(nil, 0.0, errTest)
				mf.EXPECT().PeaksComputeByFile(ctx, fileOut, 0).Return([]float64{1}, 1.0, nil)
				mcache.EXPECT().RecordingPeaksSet(ctx, recordingID, gomock.Any()).Return(nil)
			},
			expectRes: map[string]recordingpeak.RecordingFilePeak{
				"in.wav":  {Peaks: []float64{}, Duration: 0},
				"out.wav": {Peaks: []float64{1}, Duration: 1},
			},
		},
		{
			name: "nil peaks without an error become an empty slice",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(nil, redis.Nil)
				mf.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{fileIn}, nil)
				mf.EXPECT().PeaksComputeByFile(ctx, fileIn, 0).Return(nil, 4.0, nil)
				mcache.EXPECT().RecordingPeaksSet(ctx, recordingID, gomock.Any()).Return(nil)
			},
			expectRes: map[string]recordingpeak.RecordingFilePeak{
				"in.wav": {Peaks: []float64{}, Duration: 4},
			},
		},
		{
			name: "a cache write failure does not fail the request",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(nil, redis.Nil)
				mf.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{fileIn}, nil)
				mf.EXPECT().PeaksComputeByFile(ctx, fileIn, 0).Return([]float64{0.5}, 1.0, nil)
				mcache.EXPECT().RecordingPeaksSet(ctx, recordingID, gomock.Any()).Return(errTest)
			},
			expectRes: map[string]recordingpeak.RecordingFilePeak{
				"in.wav": {Peaks: []float64{0.5}, Duration: 1},
			},
		},
		{
			name: "file listing failure is returned",
			expect: func(mf *filehandler.MockFileHandler, mcache *cachehandler.MockCacheHandler) {
				mcache.EXPECT().RecordingPeaksGet(ctx, recordingID).Return(nil, redis.Nil)
				mf.EXPECT().List(ctx, "", uint64(100), filters).Return(nil, errTest)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockFile := filehandler.NewMockFileHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &storageHandler{fileHandler: mockFile, cacheHandler: mockCache}
			tt.expect(mockFile, mockCache)

			res, err := h.RecordingPeaks(ctx, recordingID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				if !errors.Is(err, errTest) {
					t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
				}
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expectRes, res)
			}
		})
	}
}

func Test_Recording_errorPaths(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("a05d9e52-0000-0000-0000-000000000001")
	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}
	f1 := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a05d9e52-0000-0000-0000-000000000011")}}
	f2 := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a05d9e52-0000-0000-0000-000000000012")}}

	t.Run("RecordingGet list failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		h := &storageHandler{fileHandler: mockFile}
		mockFile.EXPECT().List(ctx, "", uint64(100), filters).Return(nil, errTest)

		res, err := h.RecordingGet(ctx, id)
		if !errors.Is(err, errTest) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTest, res, err)
		}
	})

	t.Run("RecordingGet compress failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		h := &storageHandler{fileHandler: mockFile}
		mockFile.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{f1}, nil)
		mockFile.EXPECT().CompressCreate(ctx, []*file.File{f1}).Return("", "", errTest)

		res, err := h.RecordingGet(ctx, id)
		if !errors.Is(err, errTest) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTest, res, err)
		}
	})

	t.Run("RecordingGet download uri failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		mockUtil := utilhandler.NewMockUtilHandler(mc) // TimeNowAdd must not be reached
		h := &storageHandler{fileHandler: mockFile, utilHandler: mockUtil}
		mockFile.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{f1}, nil)
		mockFile.EXPECT().CompressCreate(ctx, []*file.File{f1}).Return("bucket", "tmp/a.zip", nil)
		mockFile.EXPECT().DownloadURIGet(ctx, "bucket", "tmp/a.zip", 24*time.Hour).Return("", "", errTest)

		res, err := h.RecordingGet(ctx, id)
		if !errors.Is(err, errTest) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTest, res, err)
		}
	})

	t.Run("RecordingDelete list failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		h := &storageHandler{fileHandler: mockFile}
		mockFile.EXPECT().List(ctx, "", uint64(100), filters).Return(nil, errTest)

		if err := h.RecordingDelete(ctx, id); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})

	t.Run("RecordingDelete stops at the first failed file", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		h := &storageHandler{fileHandler: mockFile}
		mockFile.EXPECT().List(ctx, "", uint64(100), filters).Return([]*file.File{f1, f2}, nil)
		// f2 must not be deleted once f1 failed (no expectation for it).
		mockFile.EXPECT().Delete(ctx, f1.ID).Return(nil, errTest)

		if err := h.RecordingDelete(ctx, id); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})
}

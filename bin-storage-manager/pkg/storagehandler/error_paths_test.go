package storagehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

var errTest = errors.New("simulated failure")

func Test_File_errorPropagation(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("7d2a6b2f-0000-0000-0000-000000000001")

	tests := []struct {
		name   string
		expect func(m *filehandler.MockFileHandler)
		run    func(h *storageHandler) (any, error)
	}{
		{
			name: "FileCreate",
			expect: func(m *filehandler.MockFileHandler) {
				m.EXPECT().Create(ctx, id, id, file.ReferenceTypeNone, id, file.TypeNone, "n", "d", "f", "b", "p").Return(nil, errTest)
			},
			run: func(h *storageHandler) (any, error) {
				return h.FileCreate(ctx, id, id, file.ReferenceTypeNone, id, file.TypeNone, "n", "d", "f", "b", "p")
			},
		},
		{
			name:   "FileGet",
			expect: func(m *filehandler.MockFileHandler) { m.EXPECT().Get(ctx, id).Return(nil, errTest) },
			run:    func(h *storageHandler) (any, error) { return h.FileGet(ctx, id) },
		},
		{
			name: "FileList",
			expect: func(m *filehandler.MockFileHandler) {
				m.EXPECT().List(ctx, "token", uint64(5), gomock.Any()).Return(nil, errTest)
			},
			run: func(h *storageHandler) (any, error) {
				return h.FileList(ctx, "token", 5, map[file.Field]any{})
			},
		},
		{
			name:   "FileDelete",
			expect: func(m *filehandler.MockFileHandler) { m.EXPECT().Delete(ctx, id).Return(nil, errTest) },
			run:    func(h *storageHandler) (any, error) { return h.FileDelete(ctx, id) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockFile := filehandler.NewMockFileHandler(mc)
			h := &storageHandler{fileHandler: mockFile}
			tt.expect(mockFile)

			res, err := tt.run(h)
			if !errors.Is(err, errTest) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
			}
			if v, ok := res.(*file.File); ok && v != nil {
				t.Errorf("Wrong match. expect: no file on error, got: %v", v)
			}
		})
	}
}

func Test_CompressfileCreate_errors(t *testing.T) {
	ctx := context.Background()
	fileID := uuid.FromStringOrNil("8e3b7c30-0000-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("8e3b7c30-0000-0000-0000-000000000002")
	files := []*file.File{{Identity: commonidentity.Identity{ID: fileID}}}

	tests := []struct {
		name   string
		expect func(m *filehandler.MockFileHandler)
	}{
		{
			name: "compress failed",
			expect: func(m *filehandler.MockFileHandler) {
				m.EXPECT().Get(ctx, fileID).Return(files[0], nil)
				m.EXPECT().CompressCreate(ctx, files).Return("", "", errTest)
			},
		},
		{
			name: "download uri failed",
			expect: func(m *filehandler.MockFileHandler) {
				m.EXPECT().Get(ctx, fileID).Return(files[0], nil)
				m.EXPECT().CompressCreate(ctx, files).Return("bucket", "tmp/a.zip", nil)
				m.EXPECT().DownloadURIGet(ctx, "bucket", "tmp/a.zip", 24*time.Hour).Return("", "", errTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockFile := filehandler.NewMockFileHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc) // TimeNowAdd must not be reached
			h := &storageHandler{fileHandler: mockFile, utilHandler: mockUtil}
			tt.expect(mockFile)

			res, err := h.CompressfileCreate(ctx, nil, []uuid.UUID{fileID})
			if !errors.Is(err, errTest) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}

	// files found through a reference id alone are compressed and reported by id.
	t.Run("reference ids only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFile := filehandler.NewMockFileHandler(mc)
		mockUtil := utilhandler.NewMockUtilHandler(mc)
		h := &storageHandler{fileHandler: mockFile, utilHandler: mockUtil}

		expireAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		mockFile.EXPECT().List(ctx, "", uint64(1000), map[file.Field]any{file.FieldDeleted: false, file.FieldReferenceID: refID}).Return(files, nil)
		mockFile.EXPECT().CompressCreate(ctx, files).Return("bucket", "tmp/a.zip", nil)
		mockFile.EXPECT().DownloadURIGet(ctx, "bucket", "tmp/a.zip", 24*time.Hour).Return("gs://bucket/tmp/a.zip", "https://example.invalid/a.zip", nil)
		mockUtil.EXPECT().TimeNowAdd(24 * time.Hour).Return(&expireAt)

		res, err := h.CompressfileCreate(ctx, []uuid.UUID{refID}, nil)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res.FileIDs) != 1 || res.FileIDs[0] != fileID || res.DownloadURI != "https://example.invalid/a.zip" || res.TMDownloadExpire == nil || !res.TMDownloadExpire.Equal(expireAt) {
			t.Errorf("Wrong match. got: %+v", res)
		}
	})
}

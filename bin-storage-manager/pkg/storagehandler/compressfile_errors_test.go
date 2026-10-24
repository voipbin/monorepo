package storagehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_compressGetFilesByFileIDs_lookupFailureIsReturned(t *testing.T) {
	ctx := context.Background()
	id1 := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000001")
	id2 := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockFile := filehandler.NewMockFileHandler(mc)
	h := &storageHandler{fileHandler: mockFile}

	mockFile.EXPECT().Get(ctx, id1).Return(&file.File{Identity: commonidentity.Identity{ID: id1}}, nil)
	mockFile.EXPECT().Get(ctx, id2).Return(nil, errTest)

	res, err := h.compressGetFilesByFileIDs(ctx, []uuid.UUID{id1, id2})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
}

func Test_compressGetFilesByReferenceIDs_lookupFailureIsReturned(t *testing.T) {
	ctx := context.Background()
	id1 := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000003")
	id2 := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000004")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockFile := filehandler.NewMockFileHandler(mc)
	h := &storageHandler{fileHandler: mockFile}

	mockFile.EXPECT().List(ctx, "", uint64(1000), gomock.Any()).Return([]*file.File{{Identity: commonidentity.Identity{ID: id1}}}, nil)
	mockFile.EXPECT().List(ctx, "", uint64(1000), gomock.Any()).Return(nil, errTest)

	res, err := h.compressGetFilesByReferenceIDs(ctx, []uuid.UUID{id1, id2})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
}

func Test_CompressfileCreate_lookupFailureStopsTheRequest(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000005")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockFile := filehandler.NewMockFileHandler(mc)
	h := &storageHandler{fileHandler: mockFile}

	// CompressCreate must not be reached with a partial list.
	mockFile.EXPECT().Get(ctx, id).Return(nil, errTest)

	res, err := h.CompressfileCreate(ctx, nil, []uuid.UUID{id})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
}

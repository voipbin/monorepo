package storagehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/pkg/filehandler"
)

// Test_CompressfileCreate_referenceLookupFailureStopsTheRequest checks that a failed lookup
// by reference id is returned and that no archive is built from a partial list. The file
// handler mock has no CompressCreate expectation, so reaching it fails the test.
func Test_CompressfileCreate_referenceLookupFailureStopsTheRequest(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("e52f3b21-0000-0000-0000-000000000006")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockFile := filehandler.NewMockFileHandler(mc)
	h := &storageHandler{fileHandler: mockFile}

	mockFile.EXPECT().List(ctx, "", uint64(1000), gomock.Any()).Return(nil, errTest)

	res, err := h.CompressfileCreate(ctx, []uuid.UUID{id}, nil)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
	}
}

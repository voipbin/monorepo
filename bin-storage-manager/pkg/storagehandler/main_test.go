package storagehandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_NewStorageHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockFile := filehandler.NewMockFileHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res, ok := NewStorageHandler(mockReq, mockFile, mockCache, "bucket-media").(*storageHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *storageHandler")
	}

	if res.reqHandler != mockReq || res.fileHandler != mockFile || res.cacheHandler != mockCache {
		t.Errorf("Wrong match. the dependencies were not stored as given")
	}
	if res.bucketNameMedia != "bucket-media" {
		t.Errorf("Wrong match. expect: bucket-media, got: %s", res.bucketNameMedia)
	}
	if res.utilHandler == nil {
		t.Errorf("Wrong match. expect: an initialized util handler")
	}
}

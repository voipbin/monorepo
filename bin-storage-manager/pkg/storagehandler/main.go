package storagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package storagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

// StorageHandler intreface for storage handler
type StorageHandler interface {
	RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error)
	RecordingDelete(ctx context.Context, id uuid.UUID) error
}

type storageHandler struct {
	reqHandler  requesthandler.RequestHandler
	fileHandler filehandler.FileHandler

	bucketNameMedia string
}

// fixed bucket directories
const (
	directoryRecording = "recording"
)

// NewStorageHandler creates StorageHandler
func NewStorageHandler(reqHandler requesthandler.RequestHandler, fileHandler filehandler.FileHandler, bucketNameMedia string) StorageHandler {

	h := &storageHandler{
		reqHandler:  reqHandler,
		fileHandler: fileHandler,

		bucketNameMedia: bucketNameMedia,
	}

	return h
}

package storagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package storagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
)

// StorageHandler intreface for storage handler
type StorageHandler interface {
	GetRecording(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error)
}

type storageHandler struct {
	reqHandler requesthandler.RequestHandler

	bucketHandler buckethandler.BucketHandler
}

// fixed bucket directories
const (
	bucketDirectoryRecording = "recording"
)

// NewStorageHandler creates StorageHandler
func NewStorageHandler(reqHandler requesthandler.RequestHandler, bucketHandler buckethandler.BucketHandler) StorageHandler {

	h := &storageHandler{
		reqHandler: reqHandler,

		bucketHandler: bucketHandler,
	}

	return h
}

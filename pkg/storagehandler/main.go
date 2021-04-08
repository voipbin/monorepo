package storagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package storagehandler -destination ./mock_storagehandler_storagehandler.go -source main.go -build_flags=-mod=mod

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/requesthandler"
)

// StorageHandler intreface for storage handler
type StorageHandler interface {
	GetRecording(id uuid.UUID) (*bucketrecording.BucketRecording, error)
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

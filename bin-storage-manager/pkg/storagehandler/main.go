package storagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package storagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

// StorageHandler intreface for storage handler
type StorageHandler interface {
	FileCreate(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType file.ReferenceType,
		referenceID uuid.UUID,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
	) (*file.File, error)
	FileGet(ctx context.Context, id uuid.UUID) (*file.File, error)
	FileGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error)
	FileDelete(ctx context.Context, id uuid.UUID) (*file.File, error)

	RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error)
	RecordingDelete(ctx context.Context, id uuid.UUID) error
}

type storageHandler struct {
	utilHandler utilhandler.UtilHandler
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
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,
		fileHandler: fileHandler,

		bucketNameMedia: bucketNameMedia,
	}

	return h
}

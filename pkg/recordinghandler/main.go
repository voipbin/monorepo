package recordinghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package recordinghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// RecordingHandler is interface for service handle
type RecordingHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	Get(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*recording.Recording, error)
	Start(
		ctx context.Context,
		referenceType recording.ReferenceType,
		referenceID uuid.UUID,
		format string,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*recording.Recording, error)
	Stop(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	Stopped(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
}

// list of const variables
const (
	ContextRecording = "call-record"
)

// recordingHandler structure for service handle
type recordingHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

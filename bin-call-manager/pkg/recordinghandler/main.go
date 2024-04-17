package recordinghandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package recordinghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// RecordingHandler is interface for service handle
type RecordingHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	Get(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	GetByRecordingName(ctx context.Context, recordingName string) (*recording.Recording, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*recording.Recording, error)
	Start(
		ctx context.Context,
		referenceType recording.ReferenceType,
		referenceID uuid.UUID,
		format recording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*recording.Recording, error)
	Started(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	Stop(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	Stopped(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
}

// list of const variables
const (
	ContextRecording = "call-record"
)

// recordingHandler structure for service handle
type recordingHandler struct {
	utilHandler    utilhandler.UtilHandler
	reqHandler     requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler
	db             dbhandler.DBHandler
	channelHandler channelhandler.ChannelHandler
	bridgeHandler  bridgehandler.BridgeHandler
}

// NewRecordingHandler returns a new RecordingHandler
func NewRecordingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	channelHandler channelhandler.ChannelHandler,
	bridgeHandler bridgehandler.BridgeHandler,
) RecordingHandler {
	return &recordingHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		reqHandler:     reqHandler,
		notifyHandler:  notifyHandler,
		db:             db,
		channelHandler: channelHandler,
		bridgeHandler:  bridgeHandler,
	}
}

// createRecordingName returns recording name for the given reference type and id.
func (h *recordingHandler) createRecordingName(referenceType recording.ReferenceType, referenceID string) string {
	ts := h.utilHandler.TimeGetCurTimeRFC3339()
	res := fmt.Sprintf("%s_%s_%s", referenceType, referenceID, ts)
	return res
}

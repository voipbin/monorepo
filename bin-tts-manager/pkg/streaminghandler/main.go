package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"

	"monorepo/bin-tts-manager/models/streaming"
)

// StreamingHandler define
type StreamingHandler interface {
	Run() error

	Start(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType streaming.ReferenceType,
		referenceID uuid.UUID,
		language string,
		gender streaming.Gender,
		direction streaming.Direction,
	) (*streaming.Streaming, error)
	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)

	Say(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error
	SayAdd(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error
	SayStop(ctx context.Context, id uuid.UUID) error
}

// list of default external media channel options.
//
//nolint:deadcode,varcheck
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationAudioSocket)
	defaultTransport      = string(cmexternalmedia.TransportTCP)
	defaultConnectionType = "client"
	defaultFormat         = "slin" // 8kHz, 16bit, mono signed linear PCM
)

const (
	defaultKeepAliveInterval = 10 * time.Second // 10 seconds
	defaultMaxRetryAttempts  = 3
	defaultInitialBackoff    = 100 * time.Millisecond // 100 milliseconds
)

type streamer interface {
	Init(st *streaming.Streaming) (any, error)
	Run(vendorConfig any) error
	SayStop(vendorConfig any)
	AddText(vendorConfig any, text string) error
}

type streamingHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler

	listenAddress string
	podID         string

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muStreaming  sync.Mutex

	elevenlabsHandler streamer
}

// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	listenAddress string,
	podID string,
	elevenlabsAPIKey string,
) StreamingHandler {

	elevenlabsHandler := NewElevenlabsHandler(notifyHandler, elevenlabsAPIKey)

	return &streamingHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,

		listenAddress: listenAddress,
		podID:         podID,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muStreaming:  sync.Mutex{},

		elevenlabsHandler: elevenlabsHandler,
	}
}

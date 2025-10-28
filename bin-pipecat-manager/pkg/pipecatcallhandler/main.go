package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"time"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"sync"

	"github.com/gofrs/uuid"
)

type PipecatcallHandler interface {
	Run() error

	Start(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType pipecatcall.ReferenceType,
		referenceID uuid.UUID,
		llm pipecatcall.LLM,
		stt pipecatcall.STT,
		tts pipecatcall.TTS,
		voiceID string,
		messages []map[string]any,
	) (*pipecatcall.Pipecatcall, error)
	Stop(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error)

	Get(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error)

	SendMessage(ctx context.Context, id uuid.UUID, messageID string, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error)
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

	defaultRunnerWebsocketListenAddress = "localhost:0"
)

type pipecatcallHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler

	pythonRunner        PythonRunner
	audiosocketHandler  AudiosocketHandler
	websocketHandler    WebsocketHandler
	pipecatframeHandler PipecatframeHandler

	listenAddress string
	hostID        string

	mapPipecatcall map[uuid.UUID]*pipecatcall.Pipecatcall
	muPipecatcall  sync.Mutex
}

func NewPipecatcallHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	listenAddress string,
	hostID string,
) PipecatcallHandler {
	return &pipecatcallHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,

		pythonRunner:        NewPythonRunner(),
		audiosocketHandler:  NewAudiosocketHandler(),
		websocketHandler:    NewWebsocketHandler(),
		pipecatframeHandler: NewPipecatframeHandler(),

		listenAddress: listenAddress,
		hostID:        hostID,

		mapPipecatcall: make(map[uuid.UUID]*pipecatcall.Pipecatcall),
		muPipecatcall:  sync.Mutex{},
	}
}

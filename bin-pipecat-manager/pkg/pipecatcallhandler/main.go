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
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"sync"

	"github.com/gin-gonic/gin"
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
		llmType pipecatcall.LLMType,
		llmMessages []map[string]any,
		sttType pipecatcall.STTType,
		sttLanguage string,
		ttsType pipecatcall.TTSType,
		ttsLanguage string,
		ttsVoiceID string,
	) (*pipecatcall.Pipecatcall, error)
	Terminate(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error)

	Get(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error)

	SendMessage(ctx context.Context, id uuid.UUID, messageID string, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error)

	RunnerWebsocketHandle(id uuid.UUID, c *gin.Context) error
	RunnerToolHandle(id uuid.UUID, c *gin.Context) error
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
	defaultPushFrameTimeout  = 50 * time.Millisecond  // 50ms for real-time audio

	defaultRunnerWebsocketChanBufferSize = 150 // ~3 seconds at 50fps
	defaultRunnerWebsocketListenAddress  = "localhost:0"
)

type pipecatcallHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler
	db             dbhandler.DBHandler

	pythonRunner        PythonRunner
	audiosocketHandler  AudiosocketHandler
	websocketHandler    WebsocketHandler
	pipecatframeHandler PipecatframeHandler

	listenAddress string
	hostID        string

	mapPipecatcallSession map[uuid.UUID]*pipecatcall.Session
	muPipecatcallSession  sync.Mutex
}

func NewPipecatcallHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	dbHandler dbhandler.DBHandler,

	listenAddress string,
	hostID string,
) PipecatcallHandler {
	return &pipecatcallHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,
		db:             dbHandler,

		pythonRunner:        NewPythonRunner(),
		audiosocketHandler:  NewAudiosocketHandler(),
		websocketHandler:    NewWebsocketHandler(),
		pipecatframeHandler: NewPipecatframeHandler(),

		listenAddress: listenAddress,
		hostID:        hostID,

		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}
}

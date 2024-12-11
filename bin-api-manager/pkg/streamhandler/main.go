package streamhandler

import (
	"monorepo/bin-api-manager/models/stream"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"net"
	"sync"
)

const (
	defaultTransportForAudioSocket     = "tcp"
	defaultEncapsulationForAudioSocket = "audiosocket"

	defaultConnectionType = "client"
	defaultFormat         = "ulaw"
	defualtDirection      = "both"
)

type StreamHandler interface {
	Run(conn net.Conn)
}

type streamHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler

	listenAddress string

	streamLock sync.Mutex
	streamData map[string]*stream.Stream
}

func NewStreamHandler(reqHandler requesthandler.RequestHandler, listenAddress string) StreamHandler {
	return &streamHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		listenAddress: listenAddress,
	}
}

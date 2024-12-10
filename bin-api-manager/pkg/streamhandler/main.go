package streamhandler

import (
	"monorepo/bin-api-manager/models/stream"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"net"
	"sync"
	"time"
)

const (
	defaultTransportForAudioSocket     = "tcp"
	defaultEncapsulationForAudioSocket = "audiosocket"

	defaultTransportForRTP     = "udp"
	defaultEncapsulationForRTP = "rtp"

	defaultConnectionType = "client"
	defaultFormat         = "ulaw"
	defualtDirection      = "both"

	defaultAudioSocketHeaderSize = 3 // audosocket's default header size. https://docs.asterisk.org/Configuration/Channel-Drivers/AudioSocket/

	defaultReferenceWatcherDelay = time.Second * 5
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

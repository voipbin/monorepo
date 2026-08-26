package streamhandler

//go:generate mockgen -package streamhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-api-manager/models/stream"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"net"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

// default external media configs
const (
	defaultExternalMediaTransport       = "tcp"
	defaultExternalMediaEncapsulation   = "audiosocket"
	defaultExternalMediaConnectionType  = "client"
	defaultExternalMediaFormat          = "ulaw"
	defaultExternalMediaDirectionListen = cmexternalmedia.DirectionBoth
	defaultExternalMediaDirectionSpeak  = cmexternalmedia.DirectionBoth
)

type StreamHandler interface {
	Process(conn net.Conn)
	Start(
		ctx context.Context,
		ws *websocket.Conn,
		referenceType cmexternalmedia.ReferenceType,
		referenceID uuid.UUID,
		encapsulation stream.Encapsulation,
	) (*stream.Stream, error)
}

type streamHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler

	// advertiseAddress is the host:port handed to Asterisk (via
	// CallV1ExternalMediaStart) so it knows where to dial back for
	// AudioSocket/ExternalMedia streaming. This is intentionally NOT the
	// address this process binds its listening socket to -- the socket
	// binds to all interfaces (see cmd/api-manager's runListenStreamsock),
	// while this must be a concrete, routable host.
	advertiseAddress string

	streamLock sync.Mutex
	streamData map[string]*stream.Stream
}

func NewStreamHandler(reqHandler requesthandler.RequestHandler, advertiseAddress string) StreamHandler {
	return &streamHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		advertiseAddress: advertiseAddress,

		streamData: make(map[string]*stream.Stream),
	}
}

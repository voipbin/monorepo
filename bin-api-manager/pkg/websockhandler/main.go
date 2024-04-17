package websockhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package websockhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"
	"time"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
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

// WebsockHandler defines
type WebsockHandler interface {
	RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error
	RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type websockHandler struct {
	reqHandler requesthandler.RequestHandler
}

// NewWebsockHandler creates a new HookHandler
func NewWebsockHandler(reqHandler requesthandler.RequestHandler) WebsockHandler {

	res := &websockHandler{
		reqHandler: reqHandler,
	}

	endpointInit()

	return res
}

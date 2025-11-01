package pipecatcall

import (
	"context"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type Session struct {
	identity.Identity // copied from pipecatcall

	PipecatcallReferenceType ReferenceType `json:"reference_type,omitempty"` // copied from pipecatcall
	PipecatcallReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // copied from pipecatcall

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// pipecat runner
	RunnerListener      net.Listener             `json:"-"`
	RunnerPort          int                      `json:"-"`
	RunnerServer        *http.Server             `json:"-"`
	RunnerWebsocket     *websocket.Conn          `json:"-"`
	RunnerWebsocketChan chan *pipecatframe.Frame `json:"-"`

	// asterisk info
	AsteriskStreamingID uuid.UUID `json:"-"`
	AsteriskConn        net.Conn  `json:"-"`

	// llm
	LLMBotText string `json:"-"`
}

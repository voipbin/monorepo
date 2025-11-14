package pipecatcall

import (
	"context"
	"monorepo/bin-common-handler/models/identity"
	"net"

	"github.com/gofrs/uuid"
)

type Session struct {
	identity.Identity // copied from pipecatcall

	PipecatcallReferenceType ReferenceType `json:"reference_type,omitempty"` // copied from pipecatcall
	PipecatcallReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // copied from pipecatcall

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// Runner info
	RunnerWebsocketChan chan *SessionFrame `json:"-"`

	// asterisk info
	AsteriskStreamingID uuid.UUID `json:"-"`
	AsteriskConn        net.Conn  `json:"-"`

	// llm
	LLMKey     string `json:"-"`
	LLMBotText string `json:"-"`
}

// SessionFrame represents a websocket frame that will be sent to the pipecat runner.
// It encapsulates the message type and raw data to be transmitted over the websocket connection.
type SessionFrame struct {
	MessageType int    // WebSocket message type (e.g., websocket.BinaryMessage, websocket.TextMessage)
	Data        []byte // Raw frame data
}

package pipecatcall

import (
	"context"
	"monorepo/bin-common-handler/models/identity"
	"sync"
	"sync/atomic"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
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
	AsteriskStreamingID uuid.UUID       `json:"-"`
	ConnAst             *websocket.Conn `json:"-"`
	ConnAstDone         chan struct{}    `json:"-"`
	ConnAstReady        chan struct{}    `json:"-"` // closed when ConnAst is set
	connAstOnce         sync.Once

	// llm
	LLMKey     string `json:"-"`
	LLMBotText string `json:"-"`

	// audio quality monitoring
	DroppedFrames atomic.Int64 `json:"-"`
}

// SetConnAst sets the Asterisk WebSocket connection and signals readiness.
// The channel close provides a happens-before guarantee: any goroutine that
// reads <-ConnAstReady is guaranteed to see the ConnAst and ConnAstDone writes.
// sync.Once ensures this is safe even if called more than once (defensive).
func (s *Session) SetConnAst(conn *websocket.Conn, done chan struct{}) {
	s.connAstOnce.Do(func() {
		s.ConnAst = conn
		s.ConnAstDone = done
		close(s.ConnAstReady)
	})
}

// SessionFrame represents a websocket frame that will be sent to the pipecat runner.
// It encapsulates the message type and raw data to be transmitted over the websocket connection.
type SessionFrame struct {
	MessageType int    // WebSocket message type (e.g., websocket.BinaryMessage, websocket.TextMessage)
	Data        []byte // Raw frame data
}

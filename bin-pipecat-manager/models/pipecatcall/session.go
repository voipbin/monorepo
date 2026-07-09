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
	ActiveflowID             uuid.UUID     `json:"activeflow_id,omitempty"`  // copied from pipecatcall

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
	LLMKey string `json:"-"`

	// InReplyToMessageID correlation (VOIP-1234 §4-1): prevents cross-talk when
	// an aicall is reused for a rapid sequence of send-text requests (e.g. an
	// agent sending a second question before the first bot response arrives).
	// SetPendingInReplyToMessageID is called by SendMessage each time a
	// message is sent to the LLM; the WebSocket read loop snapshots the
	// pending value into LLMInReplyToMessageID exactly once per generation
	// (on the first bot-llm-text token), so all intermediate/final events
	// for that generation report which inbound message triggered it, even if
	// a later SendMessage overwrites the pending value in the meantime.
	//
	// SendMessage is invoked from the RabbitMQ RPC listener's worker pool —
	// a goroutine distinct from the WebSocket read loop that reads this value
	// (see receiveMessageFrameTypeMessage). uuid.UUID is a plain [16]byte
	// array with no atomicity guarantee, so pendingInReplyToMessageID MUST be
	// accessed only through SetPendingInReplyToMessageID (writer) and
	// SnapshotPendingInReplyToMessageID (reader) below — never read or
	// written directly — to avoid a data race between the RPC worker and the
	// read loop. LLMInReplyToMessageID itself has no such race: it is
	// written only by the read loop and read only by the flush goroutine
	// after LLMDoneChan/generation-start handoff, both of which already have
	// happens-before ordering via the existing channel/atomic machinery below.
	muPendingInReplyTo        sync.Mutex
	pendingInReplyToMessageID uuid.UUID
	LLMInReplyToMessageID     uuid.UUID `json:"-"`

	// LLM intermediate event flush coordination.
	// These fields are managed by the WebSocket read loop (single goroutine per session).
	// The flush goroutine communicates via channels only — no shared mutable state.
	LLMTokenChan  chan string   `json:"-"` // buffered channel for LLM tokens (cap 64)
	LLMStopChan   chan struct{} `json:"-"` // signals flush goroutine to stop
	LLMDoneChan   chan struct{} `json:"-"` // closed when flush goroutine completes
	LLMFlushing   atomic.Bool   `json:"-"` // whether flush goroutine is running
	LLMMessageID  uuid.UUID     `json:"-"` // pre-generated message UUID for current generation
	LLMFlushOnce  sync.Once     `json:"-"` // ensures LLMStopChan is closed at most once
	LLMStopReason atomic.Int32  `json:"-"` // set by closer via CAS; read by flush goroutine for metric attribution

	// audio quality monitoring
	DroppedFrames atomic.Int64 `json:"-"`
}

// SetPendingInReplyToMessageID records the message ID that the next LLM
// generation should be correlated with. Called from SendMessage, which runs
// on the RabbitMQ RPC listener's worker pool goroutine — safe for concurrent
// use with SnapshotPendingInReplyToMessageID, called from the WebSocket read
// loop goroutine. See the field comment on Session for the race this guards
// against.
func (s *Session) SetPendingInReplyToMessageID(id uuid.UUID) {
	s.muPendingInReplyTo.Lock()
	defer s.muPendingInReplyTo.Unlock()
	s.pendingInReplyToMessageID = id
}

// SnapshotPendingInReplyToMessageID returns the current pending in-reply-to
// message ID. Called from the WebSocket read loop at the start of a new LLM
// generation to snapshot the correlation before it can be overwritten by a
// subsequent SendMessage. See SetPendingInReplyToMessageID.
func (s *Session) SnapshotPendingInReplyToMessageID() uuid.UUID {
	s.muPendingInReplyTo.Lock()
	defer s.muPendingInReplyTo.Unlock()
	return s.pendingInReplyToMessageID
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

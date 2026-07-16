package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

// MessageHandler interface
type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		sessionID uuid.UUID,
		direction message.Direction,
		senderID uuid.UUID,
		text string,
	) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)
}

type messageHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler

	// sessionLocks serializes MessageSend/Create calls per Session.ID so
	// the "is this the first inbound message" check inside Create cannot
	// race — see design doc §14 step 8 / Round 4 review finding (Medium):
	// a double-fired first message must not trigger the Widget's Flow
	// twice. Keyed by Session.ID (string form), each value is a
	// capacity-1 buffered channel used as a mutex. Guarded by
	// sessionLocksMu for lazy, double-checked creation.
	sessionLocksMu sync.RWMutex
	sessionLocks   map[uuid.UUID]chan struct{}
}

// NewMessageHandler returns MessageHandler interface
func NewMessageHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
) MessageHandler {
	return &messageHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,
		db:          dbHandler,

		sessionLocks: map[uuid.UUID]chan struct{}{},
	}
}

// lockSession acquires the in-process keyed lock for the given session,
// blocking until acquired. Always followed by a deferred unlockSession.
// This only serializes within ONE process/pod — see the design doc's
// note that DB-level correctness must not depend on this alone. Here it
// is a pure ordering guard: the actual "already triggered" state lives
// in Session.ActiveflowID, read/written under this lock so two
// concurrent first-messages on the SAME pod can never both observe
// ActiveflowID == uuid.Nil.
func (h *messageHandler) lockSession(sessionID uuid.UUID) chan struct{} {
	h.sessionLocksMu.RLock()
	ch, ok := h.sessionLocks[sessionID]
	h.sessionLocksMu.RUnlock()

	if !ok {
		h.sessionLocksMu.Lock()
		// re-check: another goroutine may have created it between the
		// RUnlock above and this Lock.
		ch, ok = h.sessionLocks[sessionID]
		if !ok {
			ch = make(chan struct{}, 1)
			h.sessionLocks[sessionID] = ch
		}
		h.sessionLocksMu.Unlock()
	}

	ch <- struct{}{} // blocks until acquired
	return ch
}

func (h *messageHandler) unlockSession(ch chan struct{}) {
	<-ch
}

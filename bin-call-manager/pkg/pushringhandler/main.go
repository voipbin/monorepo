// Package pushringhandler runs the push ring session of an incoming extension call.
//
// A session lets the mobile app answer a call through a push notification while the registered SIP legs ring in
// parallel. The session is the single owner of the ring window: the first path that enters the confbridge wins, and
// every other path is cancelled when the window ends.
package pushringhandler

//go:generate mockgen -package pushringhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/pushring"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// ErrRefused is returned when a call may not join the confbridge of a session.
var ErrRefused = errors.New("the join to the push ring session is refused")

// HangupFunc hangs up the given call with the reason. It is callhandler.HangingUp; it is injected because the call
// handler is built after this handler (the call handler depends on this one).
type HangupFunc func(ctx context.Context, id uuid.UUID, reason call.HangupReason) error

// PushRingHandler defines the push ring session handler.
type PushRingHandler interface {
	// SetHangup sets the function that hangs up a call.
	SetHangup(f HangupFunc)

	// Start starts a session for the call to the extension. A session that could not be started is not an error:
	// the result says started false and the call continues as it did before.
	Start(
		ctx context.Context,
		customerID uuid.UUID,
		destination commonaddress.Address,
		confbridgeID uuid.UUID,
		masterCallID uuid.UUID,
		flowID uuid.UUID,
		source commonaddress.Address,
	) (*pushring.StartResult, error)
	// Expire ends the ring window. Without a winner it closes the session and hangs up the master call.
	Expire(ctx context.Context, sessionID uuid.UUID) error
	// Close closes the session without hanging up the master call.
	Close(ctx context.Context, sessionID uuid.UUID) error

	// TicketClaim takes the ticket of a SIP call to "direct.<ticket>". It returns nil without an error when the value is
	// not a ticket (the caller continues with the direct lookup) and ErrTicketRefused when the ticket may not be used.
	TicketClaim(ctx context.Context, ticket string) (*pushring.TicketClaim, error)
	// TicketSessionGet returns the session of a claimed ticket.
	TicketSessionGet(ctx context.Context, claim *pushring.TicketClaim) (*pushring.Session, error)
	// TicketBind binds the claimed ticket to the temporary flow of the call that was created for it.
	TicketBind(ctx context.Context, claim *pushring.TicketClaim, flowID uuid.UUID) error

	// JoinGate decides at the join to the confbridge. A call of a session takes the lease or is hung up and ErrRefused
	// is returned. Any other call passes.
	JoinGate(ctx context.Context, cb *confbridge.Confbridge, callID uuid.UUID) error
	// Release releases the lease of the call. It is used when the join failed.
	Release(ctx context.Context, callID uuid.UUID)
	// Joined runs when the channel of the call entered the confbridge. For a call of a session it confirms the winner,
	// for the caller it marks the caller as joined. An error means the channel must be hung up.
	Joined(ctx context.Context, cb *confbridge.Confbridge, callID uuid.UUID) error
	// CallHangup runs when a call ended. It closes the session of a master call and releases the lease and the ticket
	// of a session call.
	CallHangup(ctx context.Context, c *call.Call)
}

type pushRingHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler
	cache       cachehandler.CacheHandler

	hangup       HangupFunc
	now          func() time.Time
	startTimeout time.Duration

	// wg tracks the notification goroutines, so a test can wait for them.
	wg sync.WaitGroup
}

// NewPushRingHandler creates a PushRingHandler.
func NewPushRingHandler(
	req requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
) PushRingHandler {
	return &pushRingHandler{
		utilHandler:  utilhandler.NewUtilHandler(),
		reqHandler:   req,
		db:           db,
		cache:        cache,
		now:          time.Now,
		startTimeout: pushring.StartTimeout,
	}
}

func (h *pushRingHandler) SetHangup(f HangupFunc) {
	h.hangup = f
}

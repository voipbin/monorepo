package pushringhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/pushring"
)

// TicketClaim takes the ticket of a SIP call to "direct.<ticket>".
//
// The cache holds only the sha256 of a ticket, so the value never appears in a key. A value that is no ticket, and a
// failing cache lookup, return nil without an error: the caller continues with the direct lookup, which cannot match a
// random 128 bit value. A ticket that exists but may not be used (expired, used, wrong state, no caller joined yet,
// session closed) returns pushring.ErrTicketRefused.
func (h *pushRingHandler) TicketClaim(ctx context.Context, ticket string) (*pushring.TicketClaim, error) {
	log := logrus.WithField("func", "TicketClaim")

	if len(ticket) != ticketLen {
		return nil, nil
	}

	now := h.now()
	res, err := h.cache.PushRingTicketClaim(ctx, hashTicket(ticket), now, now.Add(pushring.TicketInUseTTL))
	switch err {
	case nil:
		return res, nil

	case pushring.ErrTicketNotFound:
		return nil, nil

	case pushring.ErrTicketRefused:
		return nil, err
	}

	log.Errorf("Could not look up the ticket. Continuing with the direct lookup. err: %v", err)
	return nil, nil
}

// TicketSessionGet returns the session of a claimed ticket.
func (h *pushRingHandler) TicketSessionGet(ctx context.Context, claim *pushring.TicketClaim) (*pushring.Session, error) {
	return h.cache.PushRingSessionGet(ctx, claim.SessionID)
}

// TicketBind binds the claimed ticket to the temporary flow of its call. The flow exists before the call does, so
// the call is recognized at its first join.
func (h *pushRingHandler) TicketBind(ctx context.Context, claim *pushring.TicketClaim, flowID uuid.UUID) error {
	return h.cache.PushRingTicketBind(ctx, claim.SessionID, flowID, claim.TicketHash)
}

package agenthandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Reserve atomically reserves the agent for the given reference (method B,
// VOIP-1539 §3.2). Returns true when this caller won the reservation CAS,
// false when the agent was not eligible or was already reserved. The
// reservation is an internal matching correlation token and is NOT surfaced
// to customer webhooks.
func (h *agentHandler) Reserve(ctx context.Context, agentID uuid.UUID, refType string, refID uuid.UUID) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "Reserve",
		"agent_id":               agentID,
		"reserve_reference_type": refType,
		"reserve_reference_id":   refID,
	})

	ok, err := h.db.AgentReserve(ctx, agentID, refType, refID)
	if err != nil {
		log.Errorf("Could not reserve the agent. err: %v", err)
		return false, errors.Wrap(err, "could not reserve the agent")
	}

	return ok, nil
}

// ReserveRelease clears the agent's reservation, restricted to the owning
// reservation token (VOIP-1539 §3.2).
func (h *agentHandler) ReserveRelease(ctx context.Context, agentID uuid.UUID, refID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "ReserveRelease",
		"agent_id":             agentID,
		"reserve_reference_id": refID,
	})

	if err := h.db.AgentReserveRelease(ctx, agentID, refID); err != nil {
		log.Errorf("Could not release the agent reservation. err: %v", err)
		return errors.Wrap(err, "could not release the agent reservation")
	}

	return nil
}

// ReserveSweep reclaims zombie reservations older than `before` (VOIP-1539
// §3.2). It is invoked periodically by agent-manager itself and is not
// exposed over REST. Returns the number of reservations reclaimed.
func (h *agentHandler) ReserveSweep(ctx context.Context, before time.Time) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ReserveSweep",
		"before": before,
	})

	count, err := h.db.AgentReserveSweep(ctx, before)
	if err != nil {
		log.Errorf("Could not sweep zombie agent reservations. err: %v", err)
		return 0, errors.Wrap(err, "could not sweep zombie agent reservations")
	}

	if count > 0 {
		log.Infof("Reclaimed zombie agent reservations. count: %d", count)
	}

	return count, nil
}

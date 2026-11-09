package pushringhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/pushring"
)

// Expire ends the ring window of the session. It is called by the expire requests the start scheduled.
func (h *pushRingHandler) Expire(ctx context.Context, sessionID uuid.UUID) error {
	return h.closeSession(ctx, sessionID, true)
}

// Close closes the session without hanging up the master call.
func (h *pushRingHandler) Close(ctx context.Context, sessionID uuid.UUID) error {
	return h.closeSession(ctx, sessionID, false)
}

// closeSession closes the session once. The keys stay with the state closed until the cache ttl, so the gate refuses
// the late arrivals.
//
// With a confirmed winner only the tickets are deleted. At the window end a session without a recorded winner whose
// confbridge holds two or more channels (of any kind) is a connected call: the notifications are cancelled and nothing
// else is touched. Without a winner every path of the session ends: the notifications are cancelled, the tickets are
// deleted, the calls of the session are hung up, the master call is hung up with the no answer reason (window end
// only) and the confbridge is terminated, because a push only call leaves the confbridge with no channel left, which
// the auto terminate of the last leave does not cover.
func (h *pushRingHandler) closeSession(ctx context.Context, sessionID uuid.UUID, hangupMaster bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "closeSession",
		"session_id": sessionID,
	})

	s, err := h.cache.PushRingClose(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not close the session. err: %v", err)
		return err
	}
	if s == nil {
		return nil
	}
	log.Debugf("Closed the session. winner: %s, hangup_master: %v", s.Winner, hangupMaster)

	if errDelete := h.cache.PushRingTicketsDelete(ctx, s, uuid.Nil); errDelete != nil {
		log.Errorf("Could not delete the tickets. err: %v", errDelete)
	}

	if s.Winner != "" {
		return nil
	}

	h.goAsync(ctx, func(ctx context.Context) { h.sendCancel(ctx, s, uuid.Nil) })

	if hangupMaster && h.sipAnswered(ctx, s) {
		log.Infof("The session has no recorded winner but the confbridge has a connected leg. Keeping the call.")
		return nil
	}

	h.hangupSessionCalls(ctx, s)

	if hangupMaster {
		if errHangup := h.hangupCall(ctx, s.MasterCallID, call.HangupReasonNoanswer); errHangup != nil {
			log.Infof("Could not hang up the master call. err: %v", errHangup)
		}
	}

	if _, errTerminate := h.reqHandler.CallV1ConfbridgeTerminate(ctx, s.ConfbridgeID); errTerminate != nil {
		log.Infof("Could not terminate the confbridge. err: %v", errTerminate)
	}

	return nil
}

// sipAnswered returns true when the confbridge of a session without a recorded winner already holds the caller and
// another channel. That is a connected call whose winner was not recorded (a cache update that failed, or a join
// handled by a call-manager that does not know the session), so the end of the window must not cut it. A confbridge that
// cannot be read is not taken as connected: the window end then keeps its meaning.
func (h *pushRingHandler) sipAnswered(ctx context.Context, s *pushring.Session) bool {
	cb, err := h.db.ConfbridgeGet(ctx, s.ConfbridgeID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":       "sipAnswered",
			"session_id": s.ID,
		}).Infof("Could not get the confbridge. err: %v", err)
		return false
	}

	return len(cb.ChannelCallIDs) >= 2
}

// hangupSessionCalls hangs up the calls of a session that has no winner: the lease holder, the calls of the tickets
// and the SIP legs.
func (h *pushRingHandler) hangupSessionCalls(ctx context.Context, s *pushring.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hangupSessionCalls",
		"session_id": s.ID,
	})

	if s.LeaseOwner != uuid.Nil {
		if err := h.hangupCall(ctx, s.LeaseOwner, call.HangupReasonNormal); err != nil {
			log.Infof("Could not hang up the lease holder. err: %v", err)
		}
	}

	for flowID := range s.TicketFlows {
		calls, err := h.db.CallList(ctx, 100, "", map[call.Field]any{
			call.FieldFlowID:  flowID,
			call.FieldDeleted: false,
		})
		if err != nil {
			log.Errorf("Could not list the ticket calls. flow_id: %s, err: %v", flowID, err)
			continue
		}
		for _, c := range calls {
			if errHangup := h.hangupCall(ctx, c.ID, call.HangupReasonNormal); errHangup != nil {
				log.Infof("Could not hang up the ticket call. call_id: %s, err: %v", c.ID, errHangup)
			}
		}
	}

	h.hangupLegs(ctx, s, uuid.Nil)
}

// CallHangup runs when a call ended.
//
// The hangup of the master call closes the session (the caller is gone, so the master is not hung up again). The hangup
// of a call of a session releases the lease it holds and returns its ticket to issued while no winner exists. The cost
// is one cache round trip per call hangup.
func (h *pushRingHandler) CallHangup(ctx context.Context, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "CallHangup",
		"call_id": c.ID,
	})

	byMaster, byFlow, err := h.cache.PushRingSessionIDsGet(ctx, c.ID, c.FlowID)
	if err != nil {
		log.Errorf("Could not look up the session. err: %v", err)
		return
	}

	if byFlow != uuid.Nil {
		if errRelease := h.cache.PushRingLeaseRelease(ctx, byFlow, c.ID); errRelease != nil {
			log.Errorf("Could not release the lease. err: %v", errRelease)
		}
		if errReturn := h.cache.PushRingTicketReturn(ctx, byFlow, c.FlowID); errReturn != nil {
			log.Errorf("Could not return the ticket. err: %v", errReturn)
		}
	}

	if byMaster != uuid.Nil {
		if errClose := h.closeSession(ctx, byMaster, false); errClose != nil {
			log.Errorf("Could not close the session of the master call. err: %v", errClose)
		}
	}
}

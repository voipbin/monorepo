package pushringhandler

import (
	"context"
	"slices"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pmnotification "monorepo/bin-push-manager/models/notification"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/pushring"
)

// JoinGate decides at the join of a call to a confbridge that belongs to a push ring session.
//
// A session call is a leg that runs the temporary flow of the connect, or the call of a ticket. It takes the lease
// only while the session is open, the window is not over, no one else holds the lease and no winner exists. The
// caller and every other call (for example an add-participant leg, which runs its own flow) pass untouched.
//
// A cache error is a refusal. This is the cost of a cache outage and it is limited to the confbridges that were marked
// by the start of a session.
func (h *pushRingHandler) JoinGate(ctx context.Context, cb *confbridge.Confbridge, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "JoinGate",
		"confbridge_id": cb.ID,
		"call_id":       callID,
	})

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		return errors.Wrap(err, "could not get the call")
	}

	sessionID, err := h.cache.PushRingSessionIDGet(ctx, pushring.IndexFlow, c.FlowID)
	if err == pushring.ErrSessionNotFound {
		return nil
	}
	if err != nil {
		log.Errorf("Could not look up the session. Refusing the join. err: %v", err)
		h.hangupRefused(ctx, callID)
		return errors.Wrap(err, "could not look up the session")
	}

	s, err := h.cache.PushRingSessionGet(ctx, sessionID)
	if err == pushring.ErrSessionNotFound {
		// the keys expired. The call is not a call of a live session.
		return nil
	}
	if err != nil {
		log.Errorf("Could not get the session. Refusing the join. err: %v", err)
		h.hangupRefused(ctx, callID)
		return errors.Wrap(err, "could not get the session")
	}
	if s.ConfbridgeID != cb.ID {
		return nil
	}

	now := h.now()
	taken, err := h.cache.PushRingLeaseTake(ctx, sessionID, callID, now, now.Add(pushring.LeaseTTL))
	if err != nil {
		log.Errorf("Could not take the lease. Refusing the join. err: %v", err)
		h.hangupRefused(ctx, callID)
		return errors.Wrap(err, "could not take the lease")
	}
	if !taken {
		log.Infof("The join is refused. The session is closed, has a winner or the lease is held. session_id: %s", sessionID)
		h.hangupRefused(ctx, callID)
		return ErrRefused
	}

	return nil
}

func (h *pushRingHandler) hangupRefused(ctx context.Context, callID uuid.UUID) {
	if err := h.hangupCall(ctx, callID, call.HangupReasonNormal); err != nil {
		logrus.WithField("call_id", callID).Errorf("Could not hang up the refused call. err: %v", err)
	}
}

// Release releases the lease of the call.
func (h *pushRingHandler) Release(ctx context.Context, callID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Release",
		"call_id": callID,
	})

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get the call. err: %v", err)
		return
	}

	sessionID, err := h.cache.PushRingSessionIDGet(ctx, pushring.IndexFlow, c.FlowID)
	if err != nil {
		if err != pushring.ErrSessionNotFound {
			log.Errorf("Could not look up the session. err: %v", err)
		}
		return
	}

	if errRelease := h.cache.PushRingLeaseRelease(ctx, sessionID, callID); errRelease != nil {
		log.Errorf("Could not release the lease. err: %v", errRelease)
	}
}

// Joined runs when the channel of the call entered the confbridge of a push ring session.
//
// The caller (the master call of the session) is marked as joined. A session call confirms the winner; a call that may
// not win gets an error and the confbridge handler hangs up its channel.
func (h *pushRingHandler) Joined(ctx context.Context, cb *confbridge.Confbridge, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Joined",
		"confbridge_id": cb.ID,
		"call_id":       callID,
	})

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		return errors.Wrap(err, "could not get the call")
	}

	byMaster, byFlow, err := h.cache.PushRingSessionIDsGet(ctx, callID, c.FlowID)
	if err != nil {
		log.Errorf("Could not look up the session. err: %v", err)
		return errors.Wrap(err, "could not look up the session")
	}

	switch {
	case byFlow != uuid.Nil:
		return h.confirm(ctx, byFlow, c)

	case byMaster != uuid.Nil:
		if errJoined := h.cache.PushRingCallerJoined(ctx, byMaster); errJoined != nil {
			log.Errorf("Could not mark the caller as joined. err: %v", errJoined)
			return errors.Wrap(errJoined, "could not mark the caller as joined")
		}
	}

	return nil
}

// confirm makes the call the winner of the session.
func (h *pushRingHandler) confirm(ctx context.Context, sessionID uuid.UUID, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "confirm",
		"session_id": sessionID,
		"call_id":    c.ID,
	})

	s, err := h.cache.PushRingSessionGet(ctx, sessionID)
	if err == pushring.ErrSessionNotFound {
		return ErrRefused
	}
	if err != nil {
		return errors.Wrap(err, "could not get the session")
	}

	winner := pushring.WinnerSIP
	deviceID := uuid.Nil
	if c.FlowID != s.FlowID {
		// a ticket call: the winner is the device of the ticket.
		deviceID = s.Tickets[s.TicketFlows[c.FlowID]]
		winner = pushring.WinnerAppPrefix + deviceID.String()
	}

	confirmed, err := h.cache.PushRingConfirm(ctx, sessionID, c.ID, h.now(), winner)
	if err != nil {
		return errors.Wrap(err, "could not confirm the winner")
	}
	if !confirmed {
		log.Infof("The call may not win. It joined after the window, after a winner or without the lease.")
		return ErrRefused
	}
	log.Debugf("The call won the session. winner: %s", winner)

	if deviceID == uuid.Nil {
		// SIP won: no ticket is needed any more and every device is told to stop ringing.
		if errDelete := h.cache.PushRingTicketsDelete(ctx, s, uuid.Nil); errDelete != nil {
			log.Errorf("Could not delete the tickets. err: %v", errDelete)
		}
		h.goAsync(ctx, func(ctx context.Context) { h.sendCancel(ctx, s, uuid.Nil) })

		return nil
	}

	// the app won: the other devices stop ringing and the SIP legs are ended.
	if errConsume := h.cache.PushRingTicketConsume(ctx, sessionID, c.FlowID); errConsume != nil {
		log.Errorf("Could not consume the ticket. err: %v", errConsume)
	}
	if errDelete := h.cache.PushRingTicketsDelete(ctx, s, deviceID); errDelete != nil {
		log.Errorf("Could not delete the tickets. err: %v", errDelete)
	}
	h.goAsync(ctx, func(ctx context.Context) {
		for _, other := range s.Tickets {
			if other != deviceID {
				h.sendCancel(ctx, s, other)
			}
		}
	})
	h.hangupLegs(ctx, s, c.ID)

	return nil
}

// sendCancel sends the cancel notification to the device, or to every device of the agent for uuid.Nil.
func (h *pushRingHandler) sendCancel(ctx context.Context, s *pushring.Session, deviceID uuid.UUID) {
	ctx, cancel := context.WithTimeout(ctx, notificationTimeout)
	defer cancel()

	data := map[string]string{"cancel_key": s.ID.String()}
	if _, err := h.reqHandler.PushV1NotificationSend(ctx, s.CustomerID, s.AgentID, deviceID, pmnotification.KindCallCancelled, data); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":       "sendCancel",
			"session_id": s.ID,
			"device_id":  deviceID,
		}).Infof("Could not send the cancel notification. err: %v", err)
	}
}

// legCallIDs returns the ids of the legs of the session: the calls of the master call that run the temporary flow of the
// session and the calls of the groupcalls of that master call and flow (their call rows may not exist yet). The flow
// narrows both lookups, because an add-participant leg inherits the master call but runs its own flow.
func (h *pushRingHandler) legCallIDs(ctx context.Context, s *pushring.Session) []uuid.UUID {
	log := logrus.WithFields(logrus.Fields{
		"func":       "legCallIDs",
		"session_id": s.ID,
	})

	res := []uuid.UUID{}

	calls, err := h.db.CallList(ctx, 100, "", map[call.Field]any{
		call.FieldMasterCallID: s.MasterCallID,
		call.FieldFlowID:       s.FlowID,
		call.FieldDeleted:      false,
	})
	if err != nil {
		log.Errorf("Could not list the legs. err: %v", err)
	}
	for _, c := range calls {
		res = append(res, c.ID)
	}

	groupcalls, err := h.db.GroupcallList(ctx, 100, "", map[groupcall.Field]any{
		groupcall.FieldMasterCallID: s.MasterCallID,
		groupcall.FieldFlowID:       s.FlowID,
		groupcall.FieldDeleted:      false,
	})
	if err != nil {
		log.Errorf("Could not list the groupcalls. err: %v", err)
	}
	for _, g := range groupcalls {
		for _, id := range g.CallIDs {
			if !slices.Contains(res, id) {
				res = append(res, id)
			}
		}
	}

	return res
}

// hangupLegs hangs up the SIP legs of the session except the given call.
func (h *pushRingHandler) hangupLegs(ctx context.Context, s *pushring.Session, except uuid.UUID) {
	for _, id := range h.legCallIDs(ctx, s) {
		if id == except {
			continue
		}
		if err := h.hangupCall(ctx, id, call.HangupReasonNormal); err != nil {
			logrus.WithFields(logrus.Fields{
				"func":       "hangupLegs",
				"session_id": s.ID,
				"call_id":    id,
			}).Infof("Could not hang up the leg. err: %v", err)
		}
	}
}

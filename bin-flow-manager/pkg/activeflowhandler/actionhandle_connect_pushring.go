package activeflowhandler

import (
	"context"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
)

// connectPushRing is the push ring session that a connect action started. The zero value means no session.
type connectPushRing struct {
	started       bool
	sessionID     uuid.UUID
	sipRegistered bool
}

// connectPushRingStart starts a push ring session when the connect action has a single extension destination.
//
// An early media connect never starts a session: its SIP leg is executed in early execution, so it joins the confbridge
// while it is still ringing, before any answer. That join would take the lease and win the ring window at once, so the
// app would get no push and the other contacts would be refused. Such a connect runs exactly as it did before.
//
// The session lets the app of the agent answer the call through a push notification. Every failure, an unknown route
// of an older call-manager and a timeout included, is "no session": the connect then runs exactly as it did before. A
// timeout leaves the start unknown; a session that was created is ended by its own expire requests.
func (h *activeflowHandler) connectPushRingStart(ctx context.Context, af *activeflow.Activeflow, confbridgeID uuid.UUID, flowID uuid.UUID, opt *action.OptionConnect) *connectPushRing {
	log := logrus.WithFields(logrus.Fields{
		"func":          "connectPushRingStart",
		"activeflow_id": af.ID,
	})

	if opt.EarlyMedia || len(opt.Destinations) != 1 || opt.Destinations[0].Type != commonaddress.TypeExtension {
		return &connectPushRing{}
	}

	res, err := h.reqHandler.CallV1PushRingStart(ctx, af.CustomerID, opt.Destinations[0], confbridgeID, af.ReferenceID, flowID, opt.Source)
	if err != nil {
		log.Infof("Could not start the push ring session. Continuing without it. err: %v", err)
		return &connectPushRing{}
	}
	if !res.Started {
		return &connectPushRing{}
	}
	log.Debugf("Started the push ring session. session_id: %s, sip_registered: %v", res.SessionID, res.SIPRegistered)

	return &connectPushRing{
		started:       true,
		sessionID:     res.SessionID,
		sipRegistered: res.SIPRegistered,
	}
}

// pushOnlyAllowed returns true when the call creation failed only because the extension has no registered contact,
// the session was started and it knew that no contact is registered. The caller then continues with the push only.
func (r *connectPushRing) pushOnlyAllowed(err error) bool {
	if r == nil || !r.started || r.sipRegistered {
		return false
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		return false
	}

	return ve.Domain == string(commonoutline.ServiceNameCallManager) && ve.Reason == cmgroupcall.ReasonExtensionNotRegistered
}

// connectPushRingClose closes the session after a failed call creation. It is best effort: the expire requests end the
// session anyway.
func (h *activeflowHandler) connectPushRingClose(ctx context.Context, r *connectPushRing) {
	if r == nil || !r.started {
		return
	}

	if err := h.reqHandler.CallV1PushRingClose(ctx, r.sessionID); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":       "connectPushRingClose",
			"session_id": r.sessionID,
		}).Infof("Could not close the push ring session. err: %v", err)
	}
}

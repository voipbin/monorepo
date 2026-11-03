package callhandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
)

// participantClaimTTL bounds one add-participant request: the dial timeout of the new leg (defaultDialTimeout, 60 s)
// plus 30 s for the flow to run and the leg to join the confbridge. The claim is not released when the dial was
// started, so a second request is refused until the leg joined (the confbridge then has 3 channels) or this ttl passed.
const participantClaimTTL = (defaultDialTimeout + 30) * time.Second

// list of the reasons of the add-participant errors. All of them are failed preconditions (409) except the invalid
// address, which is a bad request (400).
const (
	reasonCallNotJoinable         = "CALL_NOT_JOINABLE"
	reasonParticipantCountInvalid = "PARTICIPANT_COUNT_INVALID"
	reasonParticipantAddRunning   = "PARTICIPANT_ADD_IN_PROGRESS"
	reasonInvalidParticipantAddr  = "INVALID_PARTICIPANT_ADDRESS"
)

// participantCount is the number of the legs the call has to be bridged with to take one more. v1 goes from 2 to 3.
const participantCount = 2

// AddParticipant dials the given extension and joins the answered leg to the confbridge of the given call.
//
// The checks run in this order, the first failure wins: the address, the call, the confbridge, the claim. The claim
// is taken after every validation and before the first side effect, so a refused request leaves nothing locked.
//
// The new leg gets the master call of the original call, never the original call itself: the master is the leg that
// keeps the conversation alive when the agent leg ends, and chaining the new leg to the agent leg would end it with
// the agent. Existing legs are not touched: no moh, no mute, no hangup.
//
// It returns the groupcall of the new leg. The call rows of the groupcall are created asynchronously, so the call ids
// are in groupcall.CallIDs but the calls themselves may not exist yet when this returns.
func (h *callHandler) AddParticipant(ctx context.Context, id uuid.UUID, destination commonaddress.Address) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AddParticipant",
		"call_id":     id,
		"destination": destination,
	})

	// the dial resolves the extension number from the target name, and the target is the extension id.
	if destination.Type != commonaddress.TypeExtension || destination.Target == "" || destination.TargetName == "" {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameCallManager, reasonInvalidParticipantAddr, "The participant must be an extension with its id and number.")
	}

	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the call. err: %v", err)
		return nil, err
	}

	if c.Status != call.StatusProgressing || c.ConfbridgeID == uuid.Nil {
		log.Infof("The call is not joinable. status: %s", c.Status)
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, reasonCallNotJoinable, "The call is not in a conference.")
	}

	cb, err := h.confbridgeHandler.Get(ctx, c.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not get the confbridge")
	}

	if cb.Type != confbridge.TypeConnect || cb.Status != confbridge.StatusProgressing {
		log.Infof("The confbridge is not joinable. type: %s, status: %s", cb.Type, cb.Status)
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, reasonCallNotJoinable, "The call is not in a conference.")
	}

	if len(cb.ChannelCallIDs) != participantCount {
		log.Infof("The confbridge has an unexpected number of participants. count: %d", len(cb.ChannelCallIDs))
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, reasonParticipantCountInvalid, "A participant can be added to a call of two parties only.")
	}

	// fail closed: without the claim (held by another request, or a cache error) no leg is created.
	claimed, errClaim := h.db.ConfbridgeParticipantClaim(ctx, cb.ID, participantClaimTTL)
	if errClaim != nil {
		log.Warnf("Could not take the claim. err: %v", errClaim)
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, reasonParticipantAddRunning, "Another participant is being added.")
	}
	if !claimed {
		log.Infof("Another request holds the claim.")
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCallManager, reasonParticipantAddRunning, "Another participant is being added.")
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: cb.ID,
			}),
		},
	}
	f, err := h.reqHandler.FlowV1FlowCreate(ctx, c.CustomerID, fmflow.TypeFlow, "created by call-manager", "temp flow for adding a participant", actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		h.participantRelease(ctx, cb.ID, uuid.Nil)
		return nil, errors.Wrap(err, "could not create the flow")
	}

	res, err := h.groupcallHandler.Start(
		ctx,
		uuid.Nil,
		c.CustomerID,
		f.ID,
		&c.Source,
		[]commonaddress.Address{destination},
		c.MasterCallID,
		uuid.Nil,
		groupcall.RingMethodRingAll,
		groupcall.AnswerMethodHangupOthers,
		"",
		nil,
	)
	if err != nil {
		log.Errorf("Could not start the groupcall. err: %v", err)
		h.participantRelease(ctx, cb.ID, f.ID)
		return nil, errors.Wrap(err, "could not start the groupcall")
	}
	log.WithField("groupcall", res).Debugf("Started the groupcall for the participant. groupcall_id: %s", res.ID)

	return res, nil
}

// participantRelease frees the claim and removes the temp flow after a request that failed before the dial started.
// Both are best effort: a failure is logged and the claim expires by its ttl. They use a context that is not
// canceled with the request, so a canceled request still frees the claim.
func (h *callHandler) participantRelease(ctx context.Context, confbridgeID uuid.UUID, flowID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "participantRelease",
		"confbridge_id": confbridgeID,
		"flow_id":       flowID,
	})
	ctx = context.WithoutCancel(ctx)

	if errRelease := h.db.ConfbridgeParticipantRelease(ctx, confbridgeID); errRelease != nil {
		log.Warnf("Could not release the claim. It expires by its ttl. err: %v", errRelease)
	}

	if flowID == uuid.Nil {
		return
	}

	if _, errDelete := h.reqHandler.FlowV1FlowDelete(ctx, flowID); errDelete != nil {
		log.Warnf("Could not delete the temp flow. err: %v", errDelete)
	}
}

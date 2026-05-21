package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/messagehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *aicallHandler) Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	c, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	switch c.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.SendReferenceTypeCall(ctx, c, role, messageText, runImmediately, audioResponse)

	default:
		return h.SendReferenceTypeOthers(ctx, c, role, messageText)
	}
}

func (h *aicallHandler) SendReferenceTypeCall(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "SendReferenceTypeCall",
		"aicall_id": c.ID,
	})

	pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, c.PipecatcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the pipecatcall correctly")
	}
	log.WithField("pipecatcall", pc).Debugf("Found the pipecatcall.")

	// Preflight ping before persisting the message so a dead-pod failure does not
	// orphan a user-message row in the database. The remaining failure path
	// (PipecatV1MessageSend after a successful ping) preserves existing behavior.
	if !h.pingPipecatHost(ctx, pc.HostID) {
		return nil, errors.Errorf("pipecat pod for this aicall is no longer reachable. host_id: %s, pipecatcall_id: %s", pc.HostID, pc.ID)
	}

	sendCallActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
	res, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
		messagehandler.WithActiveAIID(sendCallActiveAIID))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", c.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", c.ID, res.ID)

	tmp, err := h.reqHandler.PipecatV1MessageSend(ctx, pc.HostID, pc.ID, res.ID.String(), messageText, runImmediately, audioResponse)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message to the pipecatcall correctly")
	}
	log.WithField("pipecat_message", tmp).Debugf("Sent the message to the pipecatcall.")

	return res, nil
}

func (h *aicallHandler) SendReferenceTypeOthers(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "aicallhandler.Send",
		"aicall_id": c.ID,
	})

	// note: after create a new aicall, we need to create a new message for the conversation message
	aicallID := c.ID
	sendOtherActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
	res, errTerminate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, aicallID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
		messagehandler.WithActiveAIID(sendOtherActiveAIID))
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not create the message. aicall_id: %s", aicallID)
	}

	// Interrupt any previous pipecat session before allocating a new one.
	// Best-effort, ping-gated; correctness is provided by the response guard
	// (PipecatcallID match check) at delivery time in messagehandler.EventPMMessageBotLLM.
	h.interruptPreviousPipecatcall(ctx, c.PipecatcallID)

	newPipecatcallID := h.utilHandler.UUIDCreate()
	c, errTerminate = h.UpdatePipecatcallID(ctx, aicallID, newPipecatcallID)
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", aicallID)
	}

	// NOTE: Send does not call UpdateActiveflowID. The Send entrypoint is invoked
	// by external senders pushing a message into an existing AIcall and does not
	// receive a fresh activeflow_id. The AIcall's existing ActiveflowID remains
	// bound to whatever the last flow-driven turn set. See plan
	// docs/plans/2026-04-27-conversation-ai-talk-plan.md Slice 4 reviewer note.

	// resolve current team member's AI config for team-based aicalls
	if c.AssistanceType == aicall.AssistanceTypeTeam {
		if err := h.resolveTeamMemberForSend(ctx, c); err != nil {
			log.Warnf("Could not resolve team member AI config, using existing. err: %v", err)
		}
	}

	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", aicallID, res.ID)
	pc, errTerminate := h.startPipecatcall(ctx, c)
	if errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
	}
	log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

	if errTerminate = h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultAITaskTimeout); errTerminate != nil {
		return nil, errors.Wrapf(errTerminate, "could not send the pipecatcall terminate request correctly")
	}

	return res, nil
}

// resolveTeamMemberForSend fetches the team config, resolves the current member's AI,
// and overrides c.AIEngineModel in-memory. If CurrentMemberID was not found in the team,
// falls back to StartMemberID and updates CurrentMemberID on the DB record.
func (h *aicallHandler) resolveTeamMemberForSend(ctx context.Context, c *aicall.AIcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "resolveTeamMemberForSend",
		"aicall_id": c.ID,
	})

	t, err := h.teamHandler.Get(ctx, c.AssistanceID)
	if err != nil {
		return errors.Wrapf(err, "could not get team info. team_id: %s", c.AssistanceID)
	}
	log.WithField("team", t).Debugf("Retrieved team info. team_id: %s", t.ID)

	a, resolvedMemberID, err := h.resolveTeamMemberAI(ctx, t, c.CurrentMemberID)
	if err != nil {
		return errors.Wrapf(err, "could not resolve team member AI")
	}
	log.WithField("ai", a).Debugf("Resolved team member AI. member_id: %s, ai_engine_model: %s", resolvedMemberID, a.EngineModel)

	// override engine model in-memory for this pipecat session
	c.AIEngineModel = a.EngineModel

	// if fallback occurred, update CurrentMemberID on the DB record
	if resolvedMemberID != c.CurrentMemberID {
		log.Infof("CurrentMemberID not found in team, fell back to StartMemberID. updating. aicall_id: %s, old: %s, new: %s", c.ID, c.CurrentMemberID, resolvedMemberID)
		if _, errUpdate := h.UpdateCurrentMemberID(ctx, c.ID, resolvedMemberID); errUpdate != nil {
			log.Errorf("Could not update CurrentMemberID after fallback. err: %v", errUpdate)
		}
		c.CurrentMemberID = resolvedMemberID
	}

	return nil
}

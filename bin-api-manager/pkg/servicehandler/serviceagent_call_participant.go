package servicehandler

import (
	"context"
	"errors"

	"monorepo/bin-api-manager/models/auth"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// number of the parties the call has when a participant can be added: v1 goes from 2 to 3.
const serviceAgentParticipantAddableCount = 2

// ServiceAgentCallParticipantAdd adds a colleague's extension to a call the calling agent owns, which makes the
// two-party call a three-party call.
//
// The checks run in this order, the first failure wins, and nothing is sent to call-manager before all passed:
//  1. the caller is an agent with the any-agent permission, and the call belongs to the agent's customer and is
//     owned by the agent (an empty owner never matches).
//  2. the address is of type extension, with the extension id in the canonical form as the target.
//  3. the call is progressing and is inside a confbridge, and the confbridge has exactly two parties.
//  4. the extension exists, is not deleted and belongs to the agent's customer. A missing, foreign or deleted
//     extension gets the very same response so a caller can not probe other customers.
//
// The count check here only fails fast and saves the call-manager a round trip. The call-manager decides, and it
// takes a claim so that two concurrent requests can not both add a participant.
//
// The address passed on is built by the server from the looked-up extension. Whatever the client sent besides the
// type and the extension id (target_name, name, detail) is discarded. The extension of the calling agent is not
// refused: it only rings the agent's own phone.
//
// It returns the group call of the new leg. Its call rows are created after the response, and the legs are owned by
// the colleague, so the caller follows them from the events.
func (h *serviceHandler) ServiceAgentCallParticipantAdd(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID, address commonaddress.Address) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentCallParticipantAdd",
		"call_id": callID,
	})
	if a != nil {
		log = log.WithField("agent_id", a.AgentID())
	}

	c, err := h.serviceAgentCallOwned(ctx, a, callID)
	if err != nil {
		log.Infof("The participant add is refused. err: %v", err)
		return nil, err
	}

	if address.Type != commonaddress.TypeExtension || !isCanonicalExtensionTarget(address.Target) {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_PARTICIPANT_ADDRESS", "The address must be of type extension, with the extension ID as the target.")
	}
	extensionID := uuid.FromStringOrNil(address.Target)

	if c.Status != cmcall.StatusProgressing || c.ConfbridgeID == uuid.Nil {
		log.Infof("The call is not joinable. status: %s", c.Status)
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "CALL_NOT_JOINABLE", "The call is not in a conference.")
	}

	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, c.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return nil, err
	}
	if len(cb.ChannelCallIDs) != serviceAgentParticipantAddableCount {
		log.Infof("The call has an unexpected number of parties. count: %d", len(cb.ChannelCallIDs))
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "PARTICIPANT_COUNT_INVALID", "A participant can be added to a call of two parties only.")
	}

	ext, err := h.serviceAgentExtensionGet(ctx, a, extensionID)
	if errors.Is(err, errServiceAgentExtensionUnusable) {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_PARTICIPANT_ADDRESS", "The participant extension was not found.")
	}
	if err != nil {
		log.Errorf("Could not get the extension. err: %v", err)
		return nil, err
	}

	// audit log: who did what. No party information is logged.
	log.Info("Audit: the agent adds a participant to the call.")

	tmp, err := h.reqHandler.CallV1CallAddParticipant(ctx, callID, commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     ext.ID.String(),
		TargetName: ext.Extension,
	})
	if err != nil {
		log.Errorf("Could not add the participant. err: %v", err)
		return nil, err
	}
	if tmp == nil {
		log.Errorf("The call-manager returned an empty group call.")
		return nil, cerrors.Internal(commonoutline.ServiceNameAPIManager, "EMPTY_RESPONSE", "The call-manager returned an empty response.")
	}

	return tmp.ConvertWebhookMessage(), nil
}

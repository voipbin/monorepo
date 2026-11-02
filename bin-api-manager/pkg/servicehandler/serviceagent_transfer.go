package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentTransferStart starts a blind or attended transfer of a call the calling agent owns.
//
// The transfer-manager does not check the caller, the owner or the customer, so the checks here are the only
// defense. In this order:
//  1. the caller is an agent with the any-agent permission.
//  2. the transfer type is blind or attended, and exactly one transferee address of type extension is given with
//     the extension id in the canonical form.
//  3. the transferer call belongs to the agent's customer and is owned by the agent (an empty owner never matches),
//     is progressing and is inside a confbridge.
//  4. the extension exists, is not deleted and belongs to the agent's customer. A missing, foreign or deleted
//     extension gets the very same response so a caller can not probe other customers.
//
// The transferee address passed on is built by the server from the looked-up extension. Whatever the client sent
// besides the type and the extension id (target_name, name, detail) is discarded.
func (h *serviceHandler) ServiceAgentTransferStart(ctx context.Context, a *auth.AuthIdentity, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":               "ServiceAgentTransferStart",
		"customer_id":        a.CustomerID,
		"agent_id":           a.AgentID(),
		"transfer_type":      transferType,
		"transferer_call_id": transfererCallID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if transferType != tmtransfer.TypeBlind && transferType != tmtransfer.TypeAttended {
		log.Info("Unsupported transfer type.")
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_TRANSFER_TYPE", "The transfer_type must be blind or attended.")
	}

	if len(transfereeAddresses) != 1 ||
		transfereeAddresses[0].Type != commonaddress.TypeExtension ||
		!isCanonicalExtensionTarget(transfereeAddresses[0].Target) {
		log.Info("Invalid transferee addresses.")
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_TRANSFEREE_ADDRESS", "The transferee_addresses must have exactly one address of type extension, with the extension ID as the target.")
	}
	extensionID := uuid.FromStringOrNil(transfereeAddresses[0].Target)

	c, err := h.callGet(ctx, transfererCallID)
	if err != nil {
		log.Infof("Could not get the transferer call info. err: %v", err)
		return nil, err
	}

	// the call must belong to this customer and be owned by this agent. An empty owner never matches (fail-closed).
	if c.CustomerID != a.CustomerID || a.AgentID() == uuid.Nil || c.OwnerID != a.AgentID() {
		log.Info("The call is not owned by the agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if c.Status != cmcall.StatusProgressing || c.ConfbridgeID == uuid.Nil {
		log.Infof("The call is not transferable. status: %s", c.Status)
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "CALL_NOT_TRANSFERABLE", "The call is not transferable.")
	}

	ext, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, extensionID)
	if err != nil && !queuecallIsNotFound(err) {
		log.Errorf("Could not get the transferee extension. err: %v", err)
		return nil, err
	}
	if err != nil || ext == nil || ext.CustomerID != a.CustomerID || ext.TMDelete != nil {
		log.Infof("The transferee extension is not usable. extension_id: %s, err: %v", extensionID, err)
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_TRANSFEREE_ADDRESS", "The transferee extension was not found.")
	}

	addresses := []commonaddress.Address{
		{
			Type:       commonaddress.TypeExtension,
			Target:     ext.ID.String(),
			TargetName: ext.Extension,
		},
	}

	tmp, err := h.reqHandler.TransferV1TransferStart(ctx, transferType, transfererCallID, addresses)
	if err != nil {
		log.Errorf("Could not start the transfer. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

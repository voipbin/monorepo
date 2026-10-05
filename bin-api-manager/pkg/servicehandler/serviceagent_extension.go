package servicehandler

import (
	"context"
	"fmt"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// newExtensionNotFound returns the canonical "extension not found" error.
//
// The value is identical to what the registrar-manager returns for an unknown id, so that an unknown id,
// an id of another customer and a deleted extension are indistinguishable to the caller.
func newExtensionNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.")
}

// serviceAgentOwnedExtensionIDs returns the set of the extension ids assigned to the calling agent.
//
// The set is resolved from the up-to-date agent record, the same source as GET /service_agents/me,
// not from the addresses snapshotted in the agent's token.
func (h *serviceHandler) serviceAgentOwnedExtensionIDs(ctx context.Context, a *auth.AuthIdentity) (map[uuid.UUID]bool, error) {
	ag, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		return nil, err
	}

	res := map[uuid.UUID]bool{}
	for _, address := range ag.Addresses {
		if address.Type != commonaddress.TypeExtension {
			continue
		}

		if id := uuid.FromStringOrNil(address.Target); id != uuid.Nil {
			res[id] = true
		}
	}

	return res, nil
}

// maskExtensionForAgent blanks the credential-like fields of the given message unless the extension is owned by the caller.
// The message must be a copy (the result of ConvertWebhookMessage), never a shared record.
func maskExtensionForAgent(ws *rmextension.WebhookMessage, owned bool) *rmextension.WebhookMessage {
	if owned {
		return ws
	}

	ws.Password = ""
	ws.DirectHash = ""
	return ws
}

// ServiceAgentExtensionList returns the extensions of the agent's customer.
//
// The password and the direct hash of the extensions that are not owned by the calling agent are blanked.
func (h *serviceHandler) ServiceAgentExtensionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmextension.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentExtensionList",
		"agent": a,
		"size":  size,
		"token": token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	owned, err := h.serviceAgentOwnedExtensionIDs(ctx, a)
	if err != nil {
		log.Errorf("Could not get the agent's extensions. err: %v", err)
		return nil, err
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}
	typedFilters, err := h.convertExtensionFilters(filters)
	if err != nil {
		return nil, err
	}

	exts, err := h.reqHandler.RegistrarV1ExtensionList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extensions info", err)
	}

	res := []*rmextension.WebhookMessage{}
	for _, ext := range exts {
		// the filter above already limits the customer. This is the last line of defense of the tenant boundary.
		if ext.CustomerID != a.CustomerID {
			log.Errorf("The registrar-manager returned an extension of another customer. extension_id: %s", ext.ID)
			continue
		}

		res = append(res, maskExtensionForAgent(ext.ConvertWebhookMessage(), owned[ext.ID]))
	}

	return res, nil
}

// ServiceAgentExtensionGet returns the given extension of the agent's customer.
//
// The password and the direct hash are blanked unless the extension is owned by the calling agent.
// An unknown extension, an extension of another customer and a deleted extension all yield the same not-found error.
func (h *serviceHandler) ServiceAgentExtensionGet(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "ServiceAgentExtensionGet",
		"agent":        a,
		"extension_id": extensionID,
	})

	tmp, err := h.extensionGet(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, err
	}

	// hasPermission lets a project super admin through regardless of the customer, so the customer is compared explicitly.
	if tmp.CustomerID != a.CustomerID || tmp.TMDelete != nil {
		return nil, newExtensionNotFound()
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	owned, err := h.serviceAgentOwnedExtensionIDs(ctx, a)
	if err != nil {
		log.Errorf("Could not get the agent's extensions. err: %v", err)
		return nil, err
	}

	return maskExtensionForAgent(tmp.ConvertWebhookMessage(), owned[tmp.ID]), nil
}

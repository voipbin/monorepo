package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/sirupsen/logrus"
)

// ServiceAgentMeGet retrieves the given authenticated agent's information.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeGet(ctx context.Context, a *auth.AuthIdentity) (*amagent.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeGet",
		"agent": a,
	})

	tmp, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdate updates the authenticated agent's details.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdate(ctx context.Context, a *auth.AuthIdentity, name *string, detail *string, ringMethod *amagent.RingMethod) (*amagent.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdate",
		"agent": a,
	})

	// send request
	tmp, err := h.agentUpdate(ctx, a.AgentID(), name, detail, ringMethod)
	if err != nil {
		log.Infof("Could not update the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// meExtensionAddressKey is the identity of an extension address for the comparison with the stored ones.
// The target is the extension id and the target name is the extension number, which is the routing key of the
// calls to the extension.
type meExtensionAddressKey struct {
	target     string
	targetName string
}

// newMeExtensionAddressAdminOnly returns the error for the rejected extension address update.
func newMeExtensionAddressAdminOnly() error {
	return cerrors.PermissionDenied(
		commonoutline.ServiceNameAPIManager,
		"EXTENSION_ADDRESS_ADMIN_ONLY",
		"Extension addresses can only be added or changed by an administrator. "+
			"To keep an existing extension, send it back unchanged as returned by GET /service_agents/me. "+
			"An extension stored in a non-canonical form can only be removed. Ask an administrator to correct it.",
	)
}

// serviceAgentMeCheckExtensionAddresses rejects the extension addresses in the given list that are not already
// stored in the up-to-date agent record.
//
// An extension address is accepted only if the agent record has the extension address with the same
// (type, target, target name) and the stored target is in the canonical form. So the extension addresses can only be
// kept or removed here. They are added or changed through PUT /agents/{id}/addresses, which requires the
// administrator permission. The role of the caller is not evaluated.
//
// The agent record is not read if the list has no extension address.
func (h *serviceHandler) serviceAgentMeCheckExtensionAddresses(ctx context.Context, a *auth.AuthIdentity, addresses []commonaddress.Address) error {
	requested := false
	for _, address := range addresses {
		if address.Type == commonaddress.TypeExtension {
			requested = true
			break
		}
	}
	if !requested {
		return nil
	}

	ag, err := h.agentGet(ctx, a.AgentID())
	if err != nil {
		return err
	}

	stored := map[meExtensionAddressKey]bool{}
	for _, address := range ag.Addresses {
		if address.Type != commonaddress.TypeExtension || !isCanonicalExtensionTarget(address.Target) {
			continue
		}
		stored[meExtensionAddressKey{target: address.Target, targetName: address.TargetName}] = true
	}

	for _, address := range addresses {
		if address.Type != commonaddress.TypeExtension {
			continue
		}

		if !stored[meExtensionAddressKey{target: address.Target, targetName: address.TargetName}] {
			return newMeExtensionAddressAdminOnly()
		}
	}

	return nil
}

// ServiceAgentMeUpdateAddresses updates the authenticated agent's address information.
// It returns updated agent info.
//
// The extension addresses can only be kept or removed. See serviceAgentMeCheckExtensionAddresses.
func (h *serviceHandler) ServiceAgentMeUpdateAddresses(ctx context.Context, a *auth.AuthIdentity, addresses []commonaddress.Address) (*amagent.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdateAddresses",
		"agent": a,
	})

	// the extension addresses can not be added nor changed here, regardless of the role.
	if err := h.serviceAgentMeCheckExtensionAddresses(ctx, a, addresses); err != nil {
		log.Infof("The extension address check failed. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.agentUpdateAddresses(ctx, a.AgentID(), addresses)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdateStatus updates the authenticated agent's status information.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdateStatus(ctx context.Context, a *auth.AuthIdentity, status amagent.Status) (*amagent.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdateStatus",
		"agent": a,
	})

	// send request
	tmp, err := h.agentUpdateStatus(ctx, a.AgentID(), status)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdatePassword updates the authenticated agent's password.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdatePassword(ctx context.Context, a *auth.AuthIdentity, password string) (*amagent.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceAgentMeUpdatePassword",
		"agent":    a,
		"password": len(password),
	})

	// send request
	tmp, err := h.agentUpdatePassword(ctx, a.AgentID(), password)
	if err != nil {
		log.Infof("Could not update the agent password. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

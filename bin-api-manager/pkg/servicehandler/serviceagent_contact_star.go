package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmcontactstar "monorepo/bin-contact-manager/models/contactstar"
)

// serviceAgentStarCheck authorizes a star request and returns the calling agent's id. A star
// belongs to the agent that made it, so the id always comes from the authenticated identity and
// never from the request. Any authenticated agent of the customer may star.
func (h *serviceHandler) serviceAgentStarCheck(ctx context.Context, a *auth.AuthIdentity) (uuid.UUID, error) {
	if !a.IsAgent() || a.Agent == nil || a.Agent.ID == uuid.Nil {
		return uuid.Nil, serviceerrors.ErrAuthenticationRequired
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return uuid.Nil, serviceerrors.ErrPermissionDenied
	}

	return a.Agent.ID, nil
}

// ServiceAgentContactStarSet stars the contact for the calling agent. It is idempotent.
// contact-manager checks that the contact belongs to the caller's customer and is not deleted
// (otherwise not found), so no contact lookup is needed here.
func (h *serviceHandler) ServiceAgentContactStarSet(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontactstar.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactStarSet",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.ContactV1ContactStar(ctx, a.CustomerID, agentID, contactID)
	if err != nil {
		log.Errorf("Could not star the contact. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ServiceAgentContactStarDelete removes the calling agent's star on the contact. It is
// idempotent: the result is always the unstarred state.
func (h *serviceHandler) ServiceAgentContactStarDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontactstar.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactStarDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	if err := h.reqHandler.ContactV1ContactUnstar(ctx, a.CustomerID, agentID, contactID); err != nil {
		log.Errorf("Could not remove the star. err: %v", err)
		return nil, err
	}

	return &cmcontactstar.WebhookMessage{ContactID: contactID, Starred: false}, nil
}

// ServiceAgentContactStarList lists the calling agent's stars, newest star first. The returned
// token is the tm_create of the last star.
func (h *serviceHandler) ServiceAgentContactStarList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmcontactstar.WebhookMessage, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactStarList",
		"customer_id": a.CustomerID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, "", err
	}

	tmps, nextToken, err := h.reqHandler.ContactV1ContactStarList(ctx, a.CustomerID, agentID, size, token)
	if err != nil {
		log.Errorf("Could not list the stars. err: %v", err)
		return nil, "", err
	}

	res := []*cmcontactstar.WebhookMessage{}
	for _, tmp := range tmps {
		res = append(res, tmp.ConvertWebhookMessage())
	}

	return res, nextToken, nil
}

// ServiceAgentContactListStarred lists the contacts the calling agent has starred, in the same
// order and with the same page token as ServiceAgentContactList (the contact's tm_create).
func (h *serviceHandler) ServiceAgentContactListStarred(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmcontact.WebhookMessage, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactListStarred",
		"customer_id": a.CustomerID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, "", err
	}

	tmps, nextToken, err := h.reqHandler.ContactV1ContactStarredList(ctx, a.CustomerID, agentID, size, token)
	if err != nil {
		log.Errorf("Could not list the starred contacts. err: %v", err)
		return nil, "", err
	}

	res := []*cmcontact.WebhookMessage{}
	for _, tmp := range tmps {
		res = append(res, tmp.ConvertWebhookMessage())
	}

	return res, nextToken, nil
}

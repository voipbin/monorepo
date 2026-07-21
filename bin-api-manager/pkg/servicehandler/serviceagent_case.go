package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmkase "monorepo/bin-contact-manager/models/kase"
)

// ServiceAgentCaseList sends a request to contact-manager to list cases for
// the service agent's own customer. Status/owner_type/owner_id/contact_id
// filters are deliberately left empty/nil -- this returns the full list for
// the customer with no server-side owner filtering; square-talk filters
// client-side (design §3.1/3.3).
func (h *serviceHandler) ServiceAgentCaseList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmkase.Case, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseList",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1CaseList(ctx, a.CustomerID, "", "", uuid.Nil, uuid.Nil, size, token)
	if err != nil {
		log.Errorf("Could not list cases. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}

// ServiceAgentCaseGet gets the case of the given id for the service agent's
// own customer. No ownership check beyond tenant -- any authenticated agent
// of the customer may view any case (design §3.1).
func (h *serviceHandler) ServiceAgentCaseGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseGet",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentCaseClose closes an open case for a service-agent caller.
// closed_by_id is derived server-side from the authenticated caller's own
// agent identity (a.AgentID()), matching CaseClose's pattern in case.go.
func (h *serviceHandler) ServiceAgentCaseClose(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseClose",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if _, err := h.caseGet(ctx, a.CustomerID, id); err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.ContactV1CaseClose(ctx, a.CustomerID, id, string(commonidentity.OwnerTypeAgent), a.AgentID())
	if err != nil {
		log.Errorf("Could not close case. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentCaseAssign assigns the case of the given id to the given
// owner agent. ownerID must reference a real agent of the same customer as
// the caller -- if the agent lookup errors (agent doesn't exist) or the
// resolved agent belongs to a different customer, both failure modes
// collapse into the same ErrNotFound, so a caller cannot distinguish
// "case not found" from "owner not found" or "owner belongs to another
// tenant" (anti-enumeration).
func (h *serviceHandler) ServiceAgentCaseAssign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, ownerID uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseAssign",
		"customer_id": a.CustomerID,
		"case_id":     id,
		"owner_id":    ownerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if _, err := h.caseGet(ctx, a.CustomerID, id); err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	owner, err := h.reqHandler.AgentV1AgentGet(ctx, ownerID)
	if err != nil || owner.CustomerID != a.CustomerID {
		log.Infof("Could not validate the owner agent. err: %v", err)
		return nil, serviceerrors.ErrNotFound
	}

	res, err := h.reqHandler.ContactV1CaseAssign(ctx, a.CustomerID, id, ownerID)
	if err != nil {
		log.Errorf("Could not assign case. err: %v", err)
		return nil, err
	}

	return res, nil
}

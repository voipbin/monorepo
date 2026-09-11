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

// Note on ConvertWebhookMessage: cmkase.Case is returned directly as the
// internal struct rather than through a WebhookMessage conversion, mirroring
// the existing Interaction/Resolution precedent (see interaction.go) --
// Case carries no internal-only fields (PodID, Username, PermissionIDs,
// etc.) that need stripping before external exposure. The OpenAPI schema in
// bin-openapi-manager mirrors the struct fields and acts as the publication
// boundary. If internal-only fields are added to kase.Case in the future, a
// WebhookMessage type + ConvertWebhookMessage() must be introduced then.

// caseGet fetches a case and validates it belongs to customerID. Used as a
// pre-check before mutation operations. ContactV1CaseGet already
// tenant-checks on the contact-manager side (verifyCaseOwnership), so a
// cross-tenant id returns the same not-found error a genuinely missing id
// would -- this call never leaks cross-tenant existence.
func (h *serviceHandler) caseGet(ctx context.Context, customerID, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "caseGet",
		"customer_id": customerID,
		"case_id":     id,
	})

	res, err := h.reqHandler.ContactV1CaseGet(ctx, customerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseList sends a request to contact-manager to list cases for a customer.
// targetCustomerID may be uuid.Nil, in which case it defaults to the
// authenticated caller's own a.CustomerID. A non-superadmin caller may only
// list cases for their own customer -- hasPermission's
// PermissionProjectSuperAdmin bypass (etc.go) is inherited automatically,
// exactly like every other resource in this package, with no case-specific
// authorization code added here.
func (h *serviceHandler) CaseList(
	ctx context.Context,
	a *auth.AuthIdentity,
	targetCustomerID uuid.UUID,
	size uint64,
	token string,
	status string,
	ownerType string,
	ownerID uuid.UUID,
	contactID uuid.UUID,
	referenceID string,
) ([]*cmkase.Case, string, error) {
	customerID := targetCustomerID
	if customerID == uuid.Nil {
		customerID = a.CustomerID
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseList",
		"customer_id": customerID,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1CaseList(ctx, customerID, status, ownerType, ownerID, contactID, size, token, referenceID)
	if err != nil {
		log.Errorf("Could not list cases. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}

// CaseListUnresolved sends a request to contact-manager to list unresolved
// cases (open, contact_id IS NULL) for the authenticated caller's customer.
func (h *serviceHandler) CaseListUnresolved(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
) ([]*cmkase.Case, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseListUnresolved",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1CaseListUnresolved(ctx, a.CustomerID, size, token)
	if err != nil {
		log.Errorf("Could not list unresolved cases. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}

// CaseGet sends a request to contact-manager to get a single case by ID.
// customer_id is always derived from the authenticated caller's own
// a.CustomerID -- never from a client-supplied parameter -- so a
// non-superadmin caller can never probe another tenant's case by ID.
func (h *serviceHandler) CaseGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseGet",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	res, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, res.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// CaseAssign assigns the case of the given id to the given owner agent
// for an Admin/Manager caller (VOIP-1514). Mirrors
// ServiceAgentCaseAssign's logic (case lookup, closed-case guard, owner
// agent tenant validation, RPC call) but gates on
// PermissionCustomerAdmin|PermissionCustomerManager instead of
// PermissionAll, matching CaseClose/CaseUpdateContact's top-level
// permission pattern.
func (h *serviceHandler) CaseAssign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, ownerID uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseAssign",
		"customer_id": a.CustomerID,
		"case_id":     id,
		"owner_id":    ownerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if c.Status == cmkase.StatusClosed {
		log.Infof("Case is closed, status: %s", c.Status)
		return nil, serviceerrors.ErrCaseClosed
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

// CaseUnassign clears the case's owner for an Admin/Manager caller, or for
// the agent who currently owns the case (VOIP-1515). Mirrors CaseAssign's
// check ordering exactly: caseGet -> permission/owning-agent decision ->
// closed-case guard -> RPC call. The closed-case guard is copied verbatim
// from CaseAssign to protect the load-bearing Owner invariant documented in
// kase.go (Owner is never cleared by closing a case, since CaseContinue's
// authorization for re-opening a closed case depends on it).
func (h *serviceHandler) CaseUnassign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseUnassign",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	isOwningAgent := a.IsAgent() && a.Agent != nil &&
		c.OwnerType == commonidentity.OwnerTypeAgent &&
		c.OwnerID == a.Agent.ID

	if !isAdminOrManager && !isOwningAgent {
		log.Info("Caller has no permission to unassign the case.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if c.Status == cmkase.StatusClosed {
		log.Infof("Case is closed, status: %s", c.Status)
		return nil, serviceerrors.ErrCaseClosed
	}

	res, err := h.reqHandler.ContactV1CaseUnassign(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not unassign the case. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseClose closes an open case (design §5.1). closed_by_id is derived
// server-side from the authenticated caller's own agent identity
// (a.AgentID()) -- matching CaseContinue's pattern below -- rather than
// accepted from client input, so the closing-agent attribution the
// platform treats as a hard invariant (§5.3) cannot be forged by
// supplying an arbitrary agent_id in the request body.
func (h *serviceHandler) CaseClose(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseClose",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1CaseClose(ctx, a.CustomerID, id, string(commonidentity.OwnerTypeAgent), a.AgentID())
	if err != nil {
		log.Errorf("Could not close case. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseContinue sends a request to contact-manager to create a new, open
// case continuing a previously closed case (design §5.3). callerIsAdmin is
// derived from the caller's own permission bitmask here at the API layer --
// contact-manager only reasons about Case ownership, never the platform's
// broader agent/permission model.
func (h *serviceHandler) CaseContinue(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseContinue",
		"customer_id": a.CustomerID,
		"case_id":     id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	callerIsAdmin := a.HasPermission(amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager | amagent.PermissionProjectSuperAdmin)

	res, err := h.reqHandler.ContactV1CaseContinue(ctx, a.CustomerID, id, string(commonidentity.OwnerTypeAgent), a.AgentID(), callerIsAdmin)
	if err != nil {
		log.Errorf("Could not continue case. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseUpdateContact attaches or detaches a case's contact (design
// VOIP-1253, replaces VOIP-1252's CaseResolutionCreate/Delete).
// contactID == uuid.Nil clears the attribution.
func (h *serviceHandler) CaseUpdateContact(ctx context.Context, a *auth.AuthIdentity, id, contactID uuid.UUID) (*cmkase.Case, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		return nil, err
	}
	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return h.reqHandler.ContactV1CaseUpdateContact(ctx, a.CustomerID, id, contactID)
}

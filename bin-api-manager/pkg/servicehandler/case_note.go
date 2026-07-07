package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
)

// Note on ConvertWebhookMessage: cmcasenote.CaseNote is an internal,
// agent-facing annotation (design §3.5) that must never leak into any
// customer-facing webhook/channel. It is returned directly as the internal
// struct here, exactly like Case (see case.go's doc comment), because
// CaseNote carries no internal-only fields (PodID, Username, PermissionIDs,
// etc.) that need stripping before external exposure -- the OpenAPI schema
// in bin-openapi-manager mirrors the struct fields and acts as the
// publication boundary. These endpoints are agent-facing only (Admin/
// Manager permission gated below); nothing in this file calls
// notifyHandler.PublishWebhookEvent or ConvertWebhookMessage, and none of
// these three functions is reachable from any customer-facing webhook
// delivery path.

// CaseNoteList sends a request to contact-manager to list notes for a case.
// The case is fetched and tenant-verified via caseGet (case.go) before
// listing its notes, so a cross-tenant caseID returns the same not-found
// error a genuinely missing case would.
func (h *serviceHandler) CaseNoteList(ctx context.Context, a *auth.AuthIdentity, caseID uuid.UUID) ([]*cmcasenote.CaseNote, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseNoteList",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1CaseNoteList(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseNoteCreate sends a request to contact-manager to create a note on a
// case. customer_id is always derived from the authenticated caller's own
// a.CustomerID -- never from client input -- and the target case is
// tenant-verified via caseGet before the note is created.
func (h *serviceHandler) CaseNoteCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	caseID uuid.UUID,
	authorType string,
	authorID *uuid.UUID,
	text string,
) (*cmcasenote.CaseNote, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseNoteCreate",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1CaseNoteCreate(ctx, a.CustomerID, caseID, authorType, authorID, text)
	if err != nil {
		log.Errorf("Could not create case note. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CaseNoteDelete sends a request to contact-manager to delete a case note.
// customer_id is always derived from the authenticated caller's own
// a.CustomerID -- never from client input -- and the target case is
// tenant-verified via caseGet before the note is deleted.
func (h *serviceHandler) CaseNoteDelete(ctx context.Context, a *auth.AuthIdentity, caseID uuid.UUID, noteID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseNoteDelete",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
		"note_id":     noteID,
	})

	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return serviceerrors.ErrPermissionDenied
	}

	if err := h.reqHandler.ContactV1CaseNoteDelete(ctx, a.CustomerID, caseID, noteID); err != nil {
		log.Errorf("Could not delete case note. err: %v", err)
		return err
	}

	return nil
}

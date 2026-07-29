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

// ServiceAgentCaseNoteList sends a request to contact-manager to list notes
// for a case. The case is fetched and tenant-verified via caseGet before
// listing its notes, so a cross-tenant caseID returns the same not-found
// error a genuinely missing case would.
func (h *serviceHandler) ServiceAgentCaseNoteList(ctx context.Context, a *auth.AuthIdentity, caseID uuid.UUID) ([]*cmcasenote.CaseNote, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseNoteList",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if _, err := h.caseGet(ctx, a.CustomerID, caseID); err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.ContactV1CaseNoteList(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentCaseNoteCreate sends a request to contact-manager to create a
// note on a case, authored by the calling agent. The author is always
// derived server-side from the caller's own agent identity
// (cmcasenote.AuthorTypeAgent + a.AgentID()) -- never from client input --
// matching ServiceAgentCaseClose's closed_by_id precedent, so an agent can
// never author a note as another agent or as the system. Only a genuine
// agent identity is accepted (not just non-direct): accesskey/delegate
// identities pass hasPermission(PermissionAll) but AgentID() returns
// uuid.Nil for them, which would make every non-agent-authored note
// indistinguishable from every other for the ownership check in
// ServiceAgentCaseNoteDelete.
func (h *serviceHandler) ServiceAgentCaseNoteCreate(ctx context.Context, a *auth.AuthIdentity, caseID uuid.UUID, text string) (*cmcasenote.CaseNote, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseNoteCreate",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
	})

	if !a.IsAgent() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if _, err := h.caseGet(ctx, a.CustomerID, caseID); err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	agentID := a.AgentID()
	res, err := h.reqHandler.ContactV1CaseNoteCreate(ctx, a.CustomerID, caseID, cmcasenote.AuthorTypeAgent, &agentID, text)
	if err != nil {
		log.Errorf("Could not create case note. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentCaseNoteDelete sends a request to contact-manager to delete a
// case note. An agent may only delete a note it authored itself -- there is
// no ContactV1CaseNoteGet RPC, so the target note is located via
// ContactV1CaseNoteList and its author checked before deletion. A note with
// a nil AuthorID (system/admin-authored) never matches an agent's own id.
// Only a genuine agent identity is accepted (see ServiceAgentCaseNoteCreate's
// doc comment): otherwise a.AgentID() would be uuid.Nil and the ownership
// check would compare uuid.Nil to uuid.Nil for any other non-agent caller.
func (h *serviceHandler) ServiceAgentCaseNoteDelete(ctx context.Context, a *auth.AuthIdentity, caseID uuid.UUID, noteID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCaseNoteDelete",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
		"note_id":     noteID,
	})

	if !a.IsAgent() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return serviceerrors.ErrPermissionDenied
	}

	if _, err := h.caseGet(ctx, a.CustomerID, caseID); err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return err
	}

	notes, err := h.reqHandler.ContactV1CaseNoteList(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		return err
	}

	agentID := a.AgentID()
	var target *cmcasenote.CaseNote
	for _, note := range notes {
		if note.ID == noteID {
			target = note
			break
		}
	}
	if target == nil {
		return serviceerrors.ErrNotFound
	}
	if target.AuthorID == nil || *target.AuthorID != agentID {
		log.Info("The agent does not own this note.")
		return serviceerrors.ErrPermissionDenied
	}

	if err := h.reqHandler.ContactV1CaseNoteDelete(ctx, a.CustomerID, caseID, noteID); err != nil {
		log.Errorf("Could not delete case note. err: %v", err)
		return err
	}

	return nil
}

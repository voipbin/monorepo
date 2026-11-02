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

// ServiceAgentContactCaseNoteList lists the notes written on every case
// attributed to the given contact, newest first. The tenant check (a
// contact of another customer, or a missing one, is not found) is done by
// contact-manager from the caller's own customer id, so no contact lookup
// is needed here. Any authenticated agent of the customer may read the
// notes, the same scope as ServiceAgentCaseNoteList.
func (h *serviceHandler) ServiceAgentContactCaseNoteList(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, size uint64, token string) ([]*cmcasenote.CaseNote, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactCaseNoteList",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	if !a.IsAgent() {
		return nil, "", serviceerrors.ErrAuthenticationRequired
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	res, nextToken, err := h.reqHandler.ContactV1ContactCaseNoteList(ctx, a.CustomerID, contactID, size, token)
	if err != nil {
		log.Errorf("Could not list the case notes of the contact. err: %v", err)
		return nil, "", err
	}

	return res, nextToken, nil
}

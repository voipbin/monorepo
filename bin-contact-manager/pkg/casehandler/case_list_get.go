package casehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
)

// CaseList implements design §9's Phase 5 GET /v1/cases?... list
// surface: a thin, customer-scoped delegation to dbhandler.CaseList,
// optionally filtered by status and/or owner.
func (h *caseHandler) CaseList(ctx context.Context, customerID uuid.UUID, status string, ownerType commonidentity.OwnerType, ownerID uuid.UUID) ([]*kase.Case, error) {
	return h.db.CaseList(ctx, customerID, status, ownerType, ownerID)
}

// CaseGet implements design §9's Phase 5 GET /v1/cases/{id} route: the
// public, tenant-checked wrapper around dbhandler.CaseGetByID. Reuses
// the shared verifyCaseOwnership choke point (see case_tag.go) so a
// case belonging to a different customer returns dbhandler.ErrNotFound
// rather than leaking existence.
func (h *caseHandler) CaseGet(ctx context.Context, customerID, id uuid.UUID) (*kase.Case, error) {
	if err := verifyCaseOwnership(ctx, h.db, customerID, id); err != nil {
		return nil, err
	}
	return h.db.CaseGetByID(ctx, id)
}

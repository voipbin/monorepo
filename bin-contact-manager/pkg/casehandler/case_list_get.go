package casehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
)

// defaultCaseListSize is used when size == 0 (design §9's Phase 5 GET
// /v1/cases?... list surface has no client-mandatory size, mirroring
// InteractionList's default-of-20 convention).
const defaultCaseListSize = 20

// CaseList implements design §9's Phase 5 GET /v1/cases?... list
// surface: a thin, customer-scoped delegation to dbhandler.CaseList,
// optionally filtered by status, owner, contact_id, and/or reference_id
// (docs/plans/2026-07-24-case-reference-id-design.md, exact match).
// Results are ordered by tm_create DESC with a tm_create-cursor token
// (empty when no further pages), matching InteractionList's pagination
// convention.
func (h *caseHandler) CaseList(ctx context.Context, customerID uuid.UUID, size uint64, token string, status string, ownerType commonidentity.OwnerType, ownerID uuid.UUID, contactID uuid.UUID, referenceID string) ([]*kase.Case, string, error) {
	if size == 0 {
		size = defaultCaseListSize
	}

	items, err := h.db.CaseList(ctx, customerID, size+1, token, status, ownerType, ownerID, contactID, referenceID)
	if err != nil {
		return nil, "", err
	}

	hasMore := uint64(len(items)) > size
	if hasMore {
		items = items[:size]
	}

	var nextToken string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		if last.TMCreate != nil {
			nextToken = last.TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
		// If TMCreate is nil, cursor cannot be encoded -- pagination stops
		// here (matches InteractionList's buildPagedResult precedent; in
		// production this should not occur since CaseInsert always sets
		// TMCreate).
	}

	return items, nextToken, nil
}

// CaseGet implements design §9's Phase 5 GET /v1/cases/{id} route: the
// public, tenant-checked wrapper around dbhandler.CaseGetByID. Reuses
// the shared verifyCaseOwnershipAndGet choke point (see case_tag.go) so
// a case belonging to a different customer returns dbhandler.ErrNotFound
// rather than leaking existence. Returns the Case fetched by the
// ownership check directly (VOIP-1254) instead of a second, redundant
// CaseGetByID round trip.
func (h *caseHandler) CaseGet(ctx context.Context, customerID, id uuid.UUID) (*kase.Case, error) {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, id)
	if err != nil {
		return nil, err
	}
	return c, nil
}

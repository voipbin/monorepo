package casehandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Assign implements the square-talk Cases menu design §3.2: writes
// Case.Owner (owner_type/owner_id). Tenant-checked via CaseGetByID,
// mirroring Continue's pattern in lifecycle.go: a cross-tenant id is
// treated identically to a non-existent one (dbhandler.ErrNotFound),
// never leaking existence. No authorization decision is made here --
// per design §1.4 there is none to make; any caller who reaches this
// function (already authenticated as an agent of the tenant, per the
// API layer's PermissionAll gate) may assign to any (ownerType, ownerID)
// including a different agent than themselves.
func (h *caseHandler) Assign(ctx context.Context, customerID, id uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) (*kase.Case, error) {
	c, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.CustomerID != customerID {
		return nil, dbhandler.ErrNotFound
	}

	if err := h.db.CaseUpdateOwner(ctx, customerID, id, ownerType, ownerID); err != nil {
		return nil, fmt.Errorf("could not update owner. Assign. err: %v", err)
	}

	res, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

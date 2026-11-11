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

	// Synchronize the linked Conversation owner first, so a failure leaves
	// the Case untouched and a retry of the same request converges. The
	// Case is authoritative: Assign sets the Conversation owner, Unassign
	// releases it only while it still equals the previous Case owner (a
	// Conversation given to another agent through the conversation API is
	// left alone). Any other (ownerType, ownerID) combination is skipped.
	//
	// A Case that is already closed is left alone: its conversation may be
	// owned through a continued open Case (same conversation by chain), and
	// the Close has released it already. api-manager rejects such calls;
	// this keeps a direct caller from touching the continued Case's
	// conversation.
	wasOpen := c.Status == kase.StatusOpen
	prevOwnerID := c.OwnerID
	switch {
	case !wasOpen:
	case ownerType == commonidentity.OwnerTypeAgent && ownerID != uuid.Nil:
		if err := h.syncConversationOwner(ctx, c, ownerID, uuid.Nil); err != nil {
			return nil, fmt.Errorf("could not sync conversation owner. Assign. err: %w", err)
		}
	case ownerType == commonidentity.OwnerTypeNone && ownerID == uuid.Nil && c.OwnerID != uuid.Nil:
		if err := h.syncConversationOwner(ctx, c, uuid.Nil, c.OwnerID); err != nil {
			return nil, fmt.Errorf("could not sync conversation owner. Unassign. err: %w", err)
		}
	}

	if err := h.db.CaseUpdateOwner(ctx, customerID, id, ownerType, ownerID); err != nil {
		return nil, fmt.Errorf("could not update owner. Assign. err: %w", err)
	}

	res, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// A Close that committed between the read above and the owner write
	// has already released the Conversation before this Assign set it
	// (CaseUpdateOwner has no status predicate). Release it again so a
	// closed Case does not keep an agent-owned Conversation. A Close that
	// commits after this read releases the Conversation itself.
	if wasOpen && res.Status == kase.StatusClosed && ownerType == commonidentity.OwnerTypeAgent && ownerID != uuid.Nil {
		if err := h.syncConversationOwner(ctx, res, uuid.Nil, uuid.Nil); err != nil {
			return nil, fmt.Errorf("could not release conversation owner of a closed case. Assign. err: %w", err)
		}
	}

	// Tell the new owner. Only an owner change of a Case that is still open counts: a repeated Assign
	// of the same owner (a retry converges, see above), an Unassign and a Case closed in the
	// meantime publish nothing. Two Assign calls racing on the same Case can publish twice or not
	// at all; the notification is best effort like the push it feeds.
	if wasOpen && res.Status == kase.StatusOpen && ownerType == commonidentity.OwnerTypeAgent && ownerID != uuid.Nil && ownerID != prevOwnerID {
		h.notifyHandler.PublishEvent(ctx, kase.EventTypeCaseAssigned, &kase.CaseAssignedEvent{
			CaseID:     res.ID,
			CustomerID: res.CustomerID,
			OwnerID:    ownerID,
		})
	}

	return res, nil
}

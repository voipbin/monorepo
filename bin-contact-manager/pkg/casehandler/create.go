package casehandler

import (
	"context"
	stderrors "errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Create implements design doc §3's plain Create semantics -- distinct
// from GetOrCreate: no transaction, no peer lock, no retry loop, no
// previous_case_id chaining, no owner auto-assignment. Inserts a single
// new Case row via the existing dbhandler.CaseInsert primitive and
// translates its sentinel errors to typed cerrors at this layer (design
// §3.3 steps 1-2).
func (h *caseHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType, name, detail string,
) (*kase.Case, error) {
	if peer.Type == "" || peer.Target == "" {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"CASE_PEER_REQUIRED",
			"peer.type and peer.target are required and cannot be empty.",
		)
	}

	now := h.utilHandler.TimeNow()

	newCase := &kase.Case{
		ID:             h.utilHandler.UUIDCreate(),
		CustomerID:     customerID,
		Peer:           peer,
		Local:          self,
		ReferenceType:  referenceType,
		Name:           name,
		Detail:         detail,
		Status:         kase.StatusOpen,
		OpenedAt:       now,
		PreviousCaseID: nil,
		TMCreate:       now,
		TMUpdate:       now,
	}

	if err := h.db.CaseInsert(ctx, newCase); err != nil {
		if stderrors.Is(err, dbhandler.ErrDuplicate) {
			return nil, cerrors.AlreadyExists(
				commonoutline.ServiceNameContactManager,
				"CASE_ALREADY_EXISTS",
				"An open case already exists for this peer.",
			).Wrap(err)
		}
		if stderrors.Is(err, dbhandler.ErrDeadlock) {
			return nil, cerrors.Unavailable(
				commonoutline.ServiceNameContactManager,
				"CASE_CREATE_DEADLOCK",
				"Could not create the case due to a transient database conflict. Please retry.",
			).Wrap(err)
		}
		return nil, err
	}

	return newCase, nil
}

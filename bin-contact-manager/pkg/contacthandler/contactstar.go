package contacthandler

import (
	"context"
	stderrors "errors"
	"fmt"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/contactstar"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// starContactGet returns the contact when it belongs to the customer and is not deleted.
// A contact of another customer is reported exactly like a missing one, so its existence is
// not revealed.
func (h *contactHandler) starContactGet(ctx context.Context, customerID, contactID uuid.UUID) (*contact.Contact, error) {
	res, err := h.Get(ctx, contactID)
	if err != nil {
		return nil, err
	}

	if res.CustomerID != customerID {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	return res, nil
}

// starPageTokenError maps an unparsable page token to an invalid argument error.
func starPageTokenError(err error) error {
	if stderrors.Is(err, dbhandler.ErrInvalidPageToken) {
		return cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"INVALID_PAGE_TOKEN",
			"The page token is not valid.",
		).Wrap(err)
	}

	return err
}

// StarSet stars the contact for the agent. It is idempotent: starring an already starred contact
// returns the existing star.
func (h *contactHandler) StarSet(ctx context.Context, customerID, agentID, contactID uuid.UUID) (*contactstar.ContactStar, error) {
	if _, err := h.starContactGet(ctx, customerID, contactID); err != nil {
		return nil, err
	}

	res, err := h.db.ContactStarCreate(ctx, &contactstar.ContactStar{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},
		AgentID:   agentID,
		ContactID: contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("could not star the contact: %w", err)
	}

	return res, nil
}

// StarDelete removes the agent's star on the contact. It is idempotent: a contact that is not
// starred is not an error.
func (h *contactHandler) StarDelete(ctx context.Context, customerID, agentID, contactID uuid.UUID) error {
	if _, err := h.starContactGet(ctx, customerID, contactID); err != nil {
		return err
	}

	if err := h.db.ContactStarDelete(ctx, customerID, agentID, contactID); err != nil {
		return fmt.Errorf("could not remove the star: %w", err)
	}

	return nil
}

// StarList returns the agent's stars, newest star first.
func (h *contactHandler) StarList(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]*contactstar.ContactStar, error) {
	res, err := h.db.ContactStarList(ctx, customerID, agentID, size, token)
	if err != nil {
		return nil, starPageTokenError(err)
	}

	return res, nil
}

// ListStarred returns the contacts the agent has starred, newest contact first.
func (h *contactHandler) ListStarred(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]*contact.Contact, error) {
	res, err := h.db.ContactListStarred(ctx, customerID, agentID, size, token)
	if err != nil {
		return nil, starPageTokenError(err)
	}

	return res, nil
}

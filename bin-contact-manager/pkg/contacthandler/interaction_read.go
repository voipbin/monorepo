package contacthandler

import (
	"context"
	"fmt"
	"sort"
	"time"

	stderrors "errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// interactionInternalCap is the maximum number of automatic peer-match
// interactions fetched per contactID read path (STEP 2 of set-MINUS algorithm).
// Contacts with >5000 automatic interactions may see incomplete results on the
// ?contact_id= path in v1; a cursor-walk implementation is the M2 solution.
const interactionInternalCap uint64 = 5000

// InteractionGet returns a single interaction, scoped to customerID.
func (h *contactHandler) InteractionGet(ctx context.Context, customerID, id uuid.UUID) (*interaction.Interaction, error) {
	res, err := h.db.InteractionGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"INTERACTION_NOT_FOUND",
				"The interaction was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	// Tenant guard: never expose another customer's data.
	if res.CustomerID != customerID {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"INTERACTION_NOT_FOUND",
			"The interaction was not found.",
		)
	}

	return res, nil
}

// InteractionList is the main timeline read path.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero.
// Returns the interaction slice and a next-page token (empty when no further pages).
func (h *contactHandler) InteractionList(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID uuid.UUID,
	addressID uuid.UUID,
) ([]*interaction.Interaction, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "InteractionList",
		"customerID": customerID,
	})

	if size == 0 {
		size = 20
	}

	switch {
	case peerType != "" || peerTarget != "":
		// Direct peer path
		items, err := h.db.InteractionList(ctx, customerID, size+1, token, peerType, peerTarget, nil)
		if err != nil {
			return nil, "", fmt.Errorf("could not list interactions by peer. InteractionList. err: %v", err)
		}
		return buildPagedResult(items, size)

	case contactID != uuid.Nil:
		return h.interactionListByContact(ctx, log, customerID, contactID, size, token)

	case addressID != uuid.Nil:
		return h.interactionListByAddress(ctx, customerID, addressID, size, token)

	default:
		return nil, "", cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"INVALID_FILTER",
			"At least one filter (peer_type+peer_target, contact_id, or address_id) is required.",
		)
	}
}

// interactionListByContact implements the set-MINUS algorithm (design §7.1).
func (h *contactHandler) interactionListByContact(
	ctx context.Context,
	log *logrus.Entry,
	customerID, contactID uuid.UUID,
	size uint64,
	token string,
) ([]*interaction.Interaction, string, error) {
	// STEP 0: Existence + tenant check.
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil || c == nil || c.TMDelete != nil || c.CustomerID != customerID {
		return nil, "", cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	// STEP 1: Expand address set.
	addressPairs, err := h.db.AddressListByContact(ctx, customerID, contactID)
	if err != nil {
		return nil, "", fmt.Errorf("could not list addresses. interactionListByContact. err: %v", err)
	}

	// STEP 2: Fetch ALL automatic peer matches (internal cap, not caller page size).
	var automatic []*interaction.Interaction
	if len(addressPairs) > 0 {
		automatic, err = h.db.InteractionList(ctx, customerID, interactionInternalCap, "", "", "", addressPairs)
		if err != nil {
			return nil, "", fmt.Errorf("could not list interactions by address set. interactionListByContact. err: %v", err)
		}
	}
	// If addressPairs is empty → automatic stays nil (short-circuit, no IN() query).

	// STEP 3: Fetch active resolutions.
	resolutions, err := h.db.ResolutionListByContact(ctx, customerID, contactID)
	if err != nil {
		return nil, "", fmt.Errorf("could not list resolutions. interactionListByContact. err: %v", err)
	}

	// STEP 4: Set-MINUS combination.
	positiveIDs := make(map[uuid.UUID]bool)
	negativeIDs := make(map[uuid.UUID]bool)
	for _, r := range resolutions {
		switch r.ResolutionType {
		case resolution.ResolutionTypePositive:
			positiveIDs[r.InteractionID] = true
		case resolution.ResolutionTypeNegative:
			negativeIDs[r.InteractionID] = true
		}
	}

	automaticIDs := make(map[uuid.UUID]bool)
	merged := make(map[uuid.UUID]*interaction.Interaction)
	for _, i := range automatic {
		automaticIDs[i.ID] = true
		merged[i.ID] = i
	}

	// include = (automaticIDs | positiveIDs) - negativeIDs
	for id := range negativeIDs {
		delete(merged, id)
	}

	// STEP 5: Load positive-only interactions not in automatic set.
	var missingIDs []uuid.UUID
	for id := range positiveIDs {
		if !automaticIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		extra, extraErr := h.db.InteractionListByIDs(ctx, customerID, missingIDs)
		if extraErr != nil {
			return nil, "", fmt.Errorf("could not list interactions by IDs. interactionListByContact. err: %v", extraErr)
		}
		for _, i := range extra {
			if !negativeIDs[i.ID] {
				merged[i.ID] = i
			}
		}
	}

	// STEP 6: Sort, apply caller cursor, slice.
	all := make([]*interaction.Interaction, 0, len(merged))
	for _, i := range merged {
		all = append(all, i)
	}
	sort.Slice(all, func(a, b int) bool {
		ta := all[a].TMCreate
		tb := all[b].TMCreate
		if ta == nil && tb == nil {
			return all[a].ID.String() > all[b].ID.String()
		}
		if ta == nil {
			return false
		}
		if tb == nil {
			return true
		}
		if ta.Equal(*tb) {
			return all[a].ID.String() > all[b].ID.String()
		}
		return ta.After(*tb)
	})

	// Apply cursor if provided.
	if token != "" {
		cursorTime, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
		if err != nil {
			// Try RFC3339Nano as fallback for forward compatibility.
			cursorTime, err = time.Parse(time.RFC3339Nano, token)
		}
		if err != nil {
			log.WithError(err).Warn("invalid page token; starting from head")
		} else {
			found := false
			for i, item := range all {
				if item.TMCreate == nil {
					// nil-TMCreate items sort last. When a cursor is active and
					// we encounter a nil-TMCreate item, we have passed all
					// timestamped items. The nil-TMCreate tail is beyond the
					// cursor position (latest = head), so start from this point.
					all = all[i:]
					found = true
					break
				}
				if item.TMCreate.Before(cursorTime) {
					all = all[i:]
					found = true
					break
				}
			}
			if !found {
				// All items are newer than or equal to cursor, with valid timestamps.
				// No items remain past the cursor — return empty.
				all = nil
			}
		}
	}

	return buildPagedResult(all, size)
}

// interactionListByAddress implements the ?address_id= filter path.
func (h *contactHandler) interactionListByAddress(
	ctx context.Context,
	customerID, addressID uuid.UUID,
	size uint64,
	token string,
) ([]*interaction.Interaction, string, error) {
	ap, err := h.db.AddressGetByID(ctx, customerID, addressID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, "", cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"ADDRESS_NOT_FOUND",
				"The address was not found.",
			)
		}
		return nil, "", fmt.Errorf("could not get address. interactionListByAddress. err: %v", err)
	}

	items, err := h.db.InteractionList(ctx, customerID, size+1, token, "", "", []dbhandler.AddressPair{ap})
	if err != nil {
		return nil, "", fmt.Errorf("could not list interactions by address. interactionListByAddress. err: %v", err)
	}
	return buildPagedResult(items, size)
}

// buildPagedResult slices items to size and computes the next-page token.
// Returns (items, nextToken, nil). nextToken is "" when no further pages exist.
// cursor token = last.TMCreate formatted as ISO8601 microsecond UTC string.
// This follows the platform-wide convention (calls, messages, emails, etc.).
func buildPagedResult(items []*interaction.Interaction, size uint64) ([]*interaction.Interaction, string, error) {
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
		// If TMCreate is nil, cursor cannot be encoded — pagination stops here.
		// In production this should not occur (InteractionCreate always sets TMCreate).
	}

	return items, nextToken, nil
}

// InteractionListUnresolved returns interactions with zero-contact attribution.
// Predicate: NOT auto-matched to any address AND NOT positive-resolved AND peer_type != web_session.
// Supports pagination (size + token).
// Returns the interaction slice and a next-page token (empty when no further pages).
func (h *contactHandler) InteractionListUnresolved(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	since time.Time,
) ([]*interaction.Interaction, string, error) {
	if size == 0 {
		size = 100
	}
	if size > 500 {
		return nil, "", cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"INVALID_PAGE_SIZE",
			"page_size must be at most 500 for the unresolved endpoint.",
		)
	}

	items, err := h.db.InteractionListUnresolved(ctx, customerID, size+1, token, since)
	if err != nil {
		return nil, "", fmt.Errorf("could not list unresolved interactions. InteractionListUnresolved. err: %v", err)
	}
	return buildPagedResult(items, size)
}

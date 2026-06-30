package contacthandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// normalizeE164 normalizes a phone number to E.164 canonical form. The source
// is the e164 field if non-empty, otherwise the raw number (a contact-manager
// model concern). The actual canonicalization is delegated to the shared
// commonaddress.NormalizeTarget authority so every channel normalizes phone
// targets identically. Tel normalization never returns an error.
func normalizeE164(e164, number string) string {
	src := e164
	if strings.TrimSpace(src) == "" {
		src = number
	}
	out, _ := commonaddress.NormalizeTarget(commonaddress.TypeTel, src)
	return out
}

// Create creates a new contact with optional addresses and tags
func (h *contactHandler) Create(ctx context.Context, c *contact.Contact) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": c.CustomerID,
	})

	// Generate ID if not provided
	if c.ID == uuid.Nil {
		c.ID = h.utilHandler.UUIDCreate()
	}

	// Set default source if not provided
	if c.Source == "" {
		c.Source = contact.SourceManual
	}

	// Store addresses and tags for later
	addresses := c.Addresses
	tagIDs := c.TagIDs

	// Clear related data before inserting the contact
	c.Addresses = nil
	c.TagIDs = nil

	// Create the contact
	if err := h.db.ContactCreate(ctx, c); err != nil {
		log.Errorf("Could not create contact. err: %v", err)
		return nil, fmt.Errorf("could not create contact: %w", err)
	}

	// Add addresses
	for _, a := range addresses {
		a.ID = h.utilHandler.UUIDCreate()
		a.CustomerID = c.CustomerID
		a.ContactID = c.ID

		// normalize
		switch a.Type {
		case contact.AddressTypeTel:
			a.Target = normalizeE164("", a.Target)
		case contact.AddressTypeEmail:
			a.Target, _ = commonaddress.NormalizeTarget(commonaddress.TypeEmail, a.Target)
		}

		if a.IsPrimary {
			if err := h.db.AddressResetPrimary(ctx, c.ID); err != nil {
				log.Warnf("Could not reset primary address. err: %v", err)
			}
		}

		if err := h.db.AddressCreate(ctx, &a); err != nil {
			log.Warnf("Could not create address. err: %v", err)
		}
	}

	// Add tags
	for _, tagID := range tagIDs {
		if err := h.db.TagAssignmentCreate(ctx, c.ID, tagID); err != nil {
			log.Warnf("Could not create tag assignment. err: %v", err)
		}
	}

	// Get the full contact with related data
	res, err := h.db.ContactGet(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get created contact: %w", err)
	}

	// Publish event
	h.publishEvent(ctx, contact.EventTypeContactCreated, res)

	return res, nil
}

// Get returns a contact by ID with all related data
func (h *contactHandler) Get(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	res, err := h.db.ContactGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	// The by-id dbhandler primitive is intentionally unfiltered (the delete
	// event payload re-reads the tombstone). Public reads must not expose a
	// soft-deleted contact, so treat a set tm_delete as not-found.
	if res.TMDelete != nil {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	return res, nil
}

// List returns contacts based on filters
func (h *contactHandler) List(ctx context.Context, size uint64, token string, filters map[contact.Field]any) ([]*contact.Contact, error) {
	res, err := h.db.ContactList(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Update updates a contact's basic fields
func (h *contactHandler) Update(ctx context.Context, id uuid.UUID, fields map[contact.Field]any) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Update",
		"id":   id,
	})

	if err := h.db.ContactUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update contact. err: %v", err)
		return nil, fmt.Errorf("could not update contact: %w", err)
	}

	// Get the updated contact
	res, err := h.db.ContactGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}

	// Publish event
	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)

	return res, nil
}

// Delete soft-deletes a contact
func (h *contactHandler) Delete(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	// Verify the contact exists before deletion
	if _, err := h.db.ContactGet(ctx, id); err != nil {
		return nil, err
	}

	if err := h.db.ContactDelete(ctx, id); err != nil {
		log.Errorf("Could not delete contact. err: %v", err)
		return nil, fmt.Errorf("could not delete contact: %w", err)
	}

	// Get the deleted contact (with tm_delete set)
	res, err := h.db.ContactGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get deleted contact: %w", err)
	}

	// Publish event
	h.publishEvent(ctx, contact.EventTypeContactDeleted, res)

	return res, nil
}

// LookupByPhone finds a contact by phone number (E.164 format)
func (h *contactHandler) LookupByPhone(ctx context.Context, customerID uuid.UUID, phoneE164 string) (*contact.Contact, error) {
	// Canonicalize the lookup input so it matches the stored canonical form,
	// closing the write/lookup normalization asymmetry.
	phoneE164 = normalizeE164(phoneE164, "")
	res, err := h.db.ContactLookupByPhone(ctx, customerID, phoneE164)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	// A soft-deleted contact must not enrich an inbound call. The address
	// child table is not soft-deleted, so the lookup can resolve to a
	// tombstoned contact; treat that as not-found.
	if res.TMDelete != nil {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}
	return res, nil
}

// LookupByEmail finds a contact by email address
func (h *contactHandler) LookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error) {
	// Normalize email via the shared canonicalization authority
	email, _ = commonaddress.NormalizeTarget(commonaddress.TypeEmail, email)
	res, err := h.db.ContactLookupByEmail(ctx, customerID, email)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	// A soft-deleted contact must not enrich an inbound message. The
	// address child table is not soft-deleted, so the lookup can resolve
	// to a tombstoned contact; treat that as not-found.
	if res.TMDelete != nil {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}
	return res, nil
}

// AddAddress adds an address to a contact
func (h *contactHandler) AddAddress(ctx context.Context, contactID uuid.UUID, a *contact.Address) (*contact.Contact, error) {
	// Get the contact to verify it exists and get customer_id
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, err
	}

	a.ID = h.utilHandler.UUIDCreate()
	a.CustomerID = c.CustomerID
	a.ContactID = contactID

	// normalize
	switch a.Type {
	case contact.AddressTypeTel:
		a.Target = normalizeE164("", a.Target)
	case contact.AddressTypeEmail:
		a.Target, _ = commonaddress.NormalizeTarget(commonaddress.TypeEmail, a.Target)
	}

	// Enforce single primary: reset existing primaries before setting new one
	if a.IsPrimary {
		if err := h.db.AddressResetPrimary(ctx, contactID); err != nil {
			return nil, fmt.Errorf("could not reset primary address: %w", err)
		}
	}

	if err := h.db.AddressCreate(ctx, a); err != nil {
		return nil, fmt.Errorf("could not create address: %w", err)
	}

	// Get the updated contact
	res, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}

	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)
	return res, nil
}

// UpdateAddress updates an address on a contact
func (h *contactHandler) UpdateAddress(ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) (*contact.Contact, error) {
	// Get contact to verify it exists (for tenant isolation)
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Get address to verify existence + tenant isolation and to get its type for normalization
	addr, err := h.db.AddressGet(ctx, c.CustomerID, addressID)
	if err != nil {
		return nil, err // ErrNotFound → 404
	}

	// Normalize target if being updated
	if target, ok := fields["target"]; ok {
		if targetStr, isStr := target.(string); isStr {
			switch addr.Type {
			case contact.AddressTypeTel:
				fields["target"] = normalizeE164("", targetStr)
			case contact.AddressTypeEmail:
				fields["target"], _ = commonaddress.NormalizeTarget(commonaddress.TypeEmail, targetStr)
			}
		}
	}

	// Enforce single primary if setting is_primary to true
	if isPrimary, ok := fields["is_primary"]; ok {
		if primary, isBool := isPrimary.(bool); isBool && primary {
			if err := h.db.AddressResetPrimary(ctx, contactID); err != nil {
				return nil, fmt.Errorf("could not reset primary address: %w", err)
			}
		}
	}

	if err := h.db.AddressUpdate(ctx, addressID, fields); err != nil {
		return nil, fmt.Errorf("could not update address: %w", err)
	}

	res, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}
	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)
	return res, nil
}

// RemoveAddress removes an address from a contact
func (h *contactHandler) RemoveAddress(ctx context.Context, contactID, addressID uuid.UUID) (*contact.Contact, error) {
	// Verify contact exists
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Verify address existence + tenant isolation
	if _, err := h.db.AddressGet(ctx, c.CustomerID, addressID); err != nil {
		return nil, err // ErrNotFound → 404
	}

	if err := h.db.AddressDelete(ctx, addressID); err != nil {
		return nil, fmt.Errorf("could not delete address: %w", err)
	}

	// Get the updated contact
	res, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}

	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)
	return res, nil
}

// AddTag adds a tag to a contact
func (h *contactHandler) AddTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddTag",
		"contact_id": contactID,
		"tag_id":     tagID,
	})

	if err := h.db.TagAssignmentCreate(ctx, contactID, tagID); err != nil {
		log.Errorf("Could not create tag assignment. err: %v", err)
		return nil, fmt.Errorf("could not create tag assignment: %w", err)
	}

	// Get the updated contact
	res, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}

	// Publish event
	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)

	return res, nil
}

// RemoveTag removes a tag from a contact
func (h *contactHandler) RemoveTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "RemoveTag",
		"contact_id": contactID,
		"tag_id":     tagID,
	})

	if err := h.db.TagAssignmentDelete(ctx, contactID, tagID); err != nil {
		log.Errorf("Could not delete tag assignment. err: %v", err)
		return nil, fmt.Errorf("could not delete tag assignment: %w", err)
	}

	// Get the updated contact
	res, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated contact: %w", err)
	}

	// Publish event
	h.publishEvent(ctx, contact.EventTypeContactUpdated, res)

	return res, nil
}

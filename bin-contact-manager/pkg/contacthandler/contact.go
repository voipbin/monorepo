package contacthandler

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/models/contact"
)

// normalizeE164 normalizes a phone number to E.164 format by stripping
// all characters except digits and the leading '+'.
// If the input is empty, it derives E.164 from the raw number.
func normalizeE164(e164, number string) string {
	src := strings.TrimSpace(e164)
	if src == "" {
		src = strings.TrimSpace(number)
	}
	if src == "" {
		return ""
	}

	var b strings.Builder
	for i, r := range src {
		if r == '+' && i == 0 {
			b.WriteRune(r)
		} else if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Create creates a new contact with optional phone numbers, emails, and tags
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

	// Store phone numbers and emails for later
	phoneNumbers := c.PhoneNumbers
	emails := c.Emails
	tagIDs := c.TagIDs

	// Clear related data before inserting the contact
	c.PhoneNumbers = nil
	c.Emails = nil
	c.TagIDs = nil

	// Create the contact
	if err := h.db.ContactCreate(ctx, c); err != nil {
		log.Errorf("Could not create contact. err: %v", err)
		return nil, fmt.Errorf("could not create contact: %w", err)
	}

	// Add phone numbers
	for _, p := range phoneNumbers {
		p.ID = h.utilHandler.UUIDCreate()
		p.CustomerID = c.CustomerID
		p.ContactID = c.ID
		p.NumberE164 = normalizeE164(p.NumberE164, p.Number)

		if err := h.db.PhoneNumberCreate(ctx, &p); err != nil {
			log.Warnf("Could not create phone number. err: %v", err)
		}
	}

	// Add emails
	for _, e := range emails {
		e.ID = h.utilHandler.UUIDCreate()
		e.CustomerID = c.CustomerID
		e.ContactID = c.ID
		// Normalize email to lowercase
		e.Address = strings.ToLower(strings.TrimSpace(e.Address))

		if err := h.db.EmailCreate(ctx, &e); err != nil {
			log.Warnf("Could not create email. err: %v", err)
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
		return nil, err
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
	return h.db.ContactLookupByPhone(ctx, customerID, phoneE164)
}

// LookupByEmail finds a contact by email address
func (h *contactHandler) LookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error) {
	// Normalize email to lowercase
	email = strings.ToLower(strings.TrimSpace(email))
	return h.db.ContactLookupByEmail(ctx, customerID, email)
}

// AddPhoneNumber adds a phone number to a contact
func (h *contactHandler) AddPhoneNumber(ctx context.Context, contactID uuid.UUID, p *contact.PhoneNumber) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddPhoneNumber",
		"contact_id": contactID,
	})

	// Get the contact to verify it exists and get customer_id
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Set the phone number fields
	p.ID = h.utilHandler.UUIDCreate()
	p.CustomerID = c.CustomerID
	p.ContactID = contactID
	p.NumberE164 = normalizeE164(p.NumberE164, p.Number)

	if err := h.db.PhoneNumberCreate(ctx, p); err != nil {
		log.Errorf("Could not create phone number. err: %v", err)
		return nil, fmt.Errorf("could not create phone number: %w", err)
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

// RemovePhoneNumber removes a phone number from a contact
func (h *contactHandler) RemovePhoneNumber(ctx context.Context, contactID, phoneID uuid.UUID) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "RemovePhoneNumber",
		"contact_id": contactID,
		"phone_id":   phoneID,
	})

	if err := h.db.PhoneNumberDelete(ctx, phoneID); err != nil {
		log.Errorf("Could not delete phone number. err: %v", err)
		return nil, fmt.Errorf("could not delete phone number: %w", err)
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

// AddEmail adds an email to a contact
func (h *contactHandler) AddEmail(ctx context.Context, contactID uuid.UUID, e *contact.Email) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddEmail",
		"contact_id": contactID,
	})

	// Get the contact to verify it exists and get customer_id
	c, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Set the email fields
	e.ID = h.utilHandler.UUIDCreate()
	e.CustomerID = c.CustomerID
	e.ContactID = contactID
	// Normalize email to lowercase
	e.Address = strings.ToLower(strings.TrimSpace(e.Address))

	if err := h.db.EmailCreate(ctx, e); err != nil {
		log.Errorf("Could not create email. err: %v", err)
		return nil, fmt.Errorf("could not create email: %w", err)
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

// RemoveEmail removes an email from a contact
func (h *contactHandler) RemoveEmail(ctx context.Context, contactID, emailID uuid.UUID) (*contact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "RemoveEmail",
		"contact_id": contactID,
		"email_id":   emailID,
	})

	if err := h.db.EmailDelete(ctx, emailID); err != nil {
		log.Errorf("Could not delete email. err: %v", err)
		return nil, fmt.Errorf("could not delete email: %w", err)
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

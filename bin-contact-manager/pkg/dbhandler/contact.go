package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	contactTable = "contact_contacts"
)

// contactGetFromRow scans a single row into a Contact struct using db tags
func (h *handler) contactGetFromRow(rows *sql.Rows) (*contact.Contact, error) {
	res := &contact.Contact{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. contactGetFromRow. err: %v", err)
	}

	return res, nil
}

// ContactCreate creates a new contact record
func (h *handler) ContactCreate(ctx context.Context, c *contact.Contact) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil
	c.TMDelete = nil

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ContactCreate. err: %v", err)
	}

	query, args, err := sq.Insert(contactTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ContactCreate. err: %v", err)
	}

	// update the cache
	_ = h.contactUpdateToCache(ctx, c.ID)

	return nil
}

// contactUpdateToCache gets the contact from the DB and updates the cache.
func (h *handler) contactUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.contactGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	// load related data
	res.PhoneNumbers, _ = h.PhoneNumberListByContactID(ctx, id)
	res.Emails, _ = h.EmailListByContactID(ctx, id)
	res.TagIDs, _ = h.TagAssignmentListByContactID(ctx, id)

	if err := h.contactSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// contactSetToCache sets the given contact to the cache
func (h *handler) contactSetToCache(ctx context.Context, c *contact.Contact) error {
	if err := h.cache.ContactSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// contactGetFromCache returns contact from the cache.
func (h *handler) contactGetFromCache(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	res, err := h.cache.ContactGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// contactGetFromDB returns contact from the DB.
func (h *handler) contactGetFromDB(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&contact.Contact{})

	query, args, err := sq.Select(columns...).
		From(contactTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. contactGetFromDB. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. contactGetFromDB. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.contactGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. contactGetFromDB. err: %v", err)
	}

	return res, nil
}

// ContactGet returns a contact with all related data.
func (h *handler) ContactGet(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	// try cache first
	res, err := h.contactGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	// get from DB
	res, err = h.contactGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// load related data
	res.PhoneNumbers, _ = h.PhoneNumberListByContactID(ctx, id)
	res.Emails, _ = h.EmailListByContactID(ctx, id)
	res.TagIDs, _ = h.TagAssignmentListByContactID(ctx, id)

	// set to cache
	_ = h.contactSetToCache(ctx, res)

	return res, nil
}

// ContactList returns contacts based on filters.
func (h *handler) ContactList(ctx context.Context, size uint64, token string, filters map[contact.Field]any) ([]*contact.Contact, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&contact.Contact{})

	builder := sq.Select(columns...).
		From(contactTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ContactList. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactList. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ContactList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*contact.Contact{}
	for rows.Next() {
		c, err := h.contactGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ContactList. err: %v", err)
		}

		// load related data for each contact
		c.PhoneNumbers, _ = h.PhoneNumberListByContactID(ctx, c.ID)
		c.Emails, _ = h.EmailListByContactID(ctx, c.ID)
		c.TagIDs, _ = h.TagAssignmentListByContactID(ctx, c.ID)

		res = append(res, c)
	}

	return res, nil
}

// ContactUpdate updates a contact with the given fields.
func (h *handler) ContactUpdate(ctx context.Context, id uuid.UUID, fields map[contact.Field]any) error {
	// add update timestamp
	fields[contact.FieldTMUpdate] = h.utilHandler.TimeNow()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ContactUpdate. err: %v", err)
	}

	query, args, err := sq.Update(contactTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactUpdate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ContactUpdate. err: %v", err)
	}

	// update the cache
	_ = h.contactUpdateToCache(ctx, id)

	return nil
}

// ContactDelete soft-deletes a contact.
func (h *handler) ContactDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()
	fields := map[contact.Field]any{
		contact.FieldTMUpdate: ts,
		contact.FieldTMDelete: ts,
	}

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ContactDelete. err: %v", err)
	}

	query, args, err := sq.Update(contactTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactDelete. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ContactDelete. err: %v", err)
	}

	// update the cache
	_ = h.contactUpdateToCache(ctx, id)

	return nil
}

// ContactLookupByPhone finds a contact by phone number (E.164 format).
func (h *handler) ContactLookupByPhone(ctx context.Context, customerID uuid.UUID, phoneE164 string) (*contact.Contact, error) {
	// First, find the contact_id from phone numbers table
	query, args, err := sq.Select("contact_id").
		From(phoneNumberTable).
		Where(sq.Eq{
			"customer_id":  customerID.Bytes(),
			"number_e164":  phoneE164,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactLookupByPhone. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ContactLookupByPhone. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	var contactIDBytes []byte
	if err := rows.Scan(&contactIDBytes); err != nil {
		return nil, fmt.Errorf("could not scan contact_id. ContactLookupByPhone. err: %v", err)
	}

	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse contact_id. ContactLookupByPhone. err: %v", err)
	}

	// Now get the full contact
	return h.ContactGet(ctx, contactID)
}

// ContactLookupByEmail finds a contact by email address.
func (h *handler) ContactLookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error) {
	// First, find the contact_id from emails table
	query, args, err := sq.Select("contact_id").
		From(emailTable).
		Where(sq.Eq{
			"customer_id": customerID.Bytes(),
			"address":     email,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactLookupByEmail. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ContactLookupByEmail. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	var contactIDBytes []byte
	if err := rows.Scan(&contactIDBytes); err != nil {
		return nil, fmt.Errorf("could not scan contact_id. ContactLookupByEmail. err: %v", err)
	}

	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse contact_id. ContactLookupByEmail. err: %v", err)
	}

	// Now get the full contact
	return h.ContactGet(ctx, contactID)
}

// ContactDeleteByCustomerID deletes all contacts for a customer (cascade cleanup).
func (h *handler) ContactDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(contactTable).
		Set("tm_update", ts).
		Set("tm_delete", ts).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactDeleteByCustomerID. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ContactDeleteByCustomerID. err: %v", err)
	}

	return nil
}

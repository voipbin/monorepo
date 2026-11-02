package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/contactstar"
)

const contactStarTable = "contact_stars"

// ErrInvalidPageToken is returned by the star list queries when the page token is not a timestamp.
var ErrInvalidPageToken = fmt.Errorf("invalid page token")

// parsePageToken parses a tm_create cursor. The api layer builds it as
// 2006-01-02T15:04:05.000000Z from the last returned item.
func parsePageToken(token string) (time.Time, error) {
	res, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
	if err != nil {
		// forward compatibility with a full RFC3339 timestamp
		res, err = time.Parse(time.RFC3339Nano, token)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("%w. err: %w", ErrInvalidPageToken, err)
	}

	return res.UTC(), nil
}

// contactStarGetFromRow scans a single row into a ContactStar struct.
func contactStarGetFromRow(rows *sql.Rows) (*contactstar.ContactStar, error) {
	res := &contactstar.ContactStar{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. contactStarGetFromRow. err: %w", err)
	}

	return res, nil
}

// contactStarGetByAgentContact returns the star of the agent on the contact, or ErrNotFound.
func (h *handler) contactStarGetByAgentContact(ctx context.Context, customerID, agentID, contactID uuid.UUID) (*contactstar.ContactStar, error) {
	columns := commondatabasehandler.GetDBFields(&contactstar.ContactStar{})

	query, args, err := sq.Select(columns...).
		From(contactStarTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"agent_id": agentID.Bytes()}).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. contactStarGetByAgentContact. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. contactStarGetByAgentContact. err: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if errRows := rows.Err(); errRows != nil {
			return nil, fmt.Errorf("could not iterate rows. contactStarGetByAgentContact. err: %w", errRows)
		}
		return nil, ErrNotFound
	}

	return contactStarGetFromRow(rows)
}

// ContactStarCreate stores a star and returns the stored row. It is idempotent: when the agent
// has already starred the contact (unique key agent_id, contact_id), the existing row is returned
// unchanged, so the original tm_create is kept. Engine independent: no INSERT IGNORE.
func (h *handler) ContactStarCreate(ctx context.Context, s *contactstar.ContactStar) (*contactstar.ContactStar, error) {
	s.TMCreate = h.utilHandler.TimeNow()

	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return nil, fmt.Errorf("could not prepare fields. ContactStarCreate. err: %w", err)
	}

	query, args, err := sq.Insert(contactStarTable).SetMap(fields).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactStarCreate. err: %w", err)
	}

	_, errInsert := h.db.ExecContext(ctx, query, args...)
	if errInsert == nil {
		return s, nil
	}

	// A duplicate (or a concurrent identical request) is a success. Any other failure is returned
	// as is, unless the row turns out to exist.
	res, errGet := h.contactStarGetByAgentContact(ctx, s.CustomerID, s.AgentID, s.ContactID)
	if errGet != nil {
		return nil, fmt.Errorf("could not create the star. ContactStarCreate. err: %w", errInsert)
	}

	return res, nil
}

// ContactStarDelete removes the star of the agent on the contact. A missing star is not an error.
func (h *handler) ContactStarDelete(ctx context.Context, customerID, agentID, contactID uuid.UUID) error {
	query, args, err := sq.Delete(contactStarTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"agent_id": agentID.Bytes()}).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactStarDelete. err: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. ContactStarDelete. err: %w", err)
	}

	return nil
}

// ContactStarDeleteByCustomerID removes every star of a customer (cascade cleanup).
func (h *handler) ContactStarDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	query, args, err := sq.Delete(contactStarTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ContactStarDeleteByCustomerID. err: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. ContactStarDeleteByCustomerID. err: %w", err)
	}

	return nil
}

// ContactStarList returns the agent's stars, newest star first. Stars whose contact is deleted or
// belongs to another customer are left out. token is a tm_create cursor of the star row
// (exclusive).
func (h *handler) ContactStarList(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]*contactstar.ContactStar, error) {
	fields := commondatabasehandler.GetDBFields(&contactstar.ContactStar{})
	columns := make([]string, 0, len(fields))
	for _, f := range fields {
		columns = append(columns, "s."+f)
	}

	builder := sq.Select(columns...).
		From(contactStarTable + " s").
		Join(contactTable + " c ON c.id = s.contact_id AND c.customer_id = s.customer_id").
		Where(sq.Eq{"s.customer_id": customerID.Bytes()}).
		Where(sq.Eq{"s.agent_id": agentID.Bytes()}).
		Where(sq.Eq{"c.tm_delete": nil})

	if token != "" {
		cursor, err := parsePageToken(token)
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. ContactStarList. err: %w", err)
		}
		builder = builder.Where(sq.Lt{"s.tm_create": cursor})
	}

	query, args, err := builder.OrderBy("s.tm_create DESC").Limit(size).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactStarList. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ContactStarList. err: %w", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*contactstar.ContactStar{}
	for rows.Next() {
		item, errScan := contactStarGetFromRow(rows)
		if errScan != nil {
			return nil, fmt.Errorf("could not scan the row. ContactStarList. err: %w", errScan)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. ContactStarList. err: %w", err)
	}

	return res, nil
}

// ContactListStarred returns the contacts the agent has starred, in the same order and with the
// same token key as ContactList: the contact's tm_create, newest first (not the star time).
// Soft-deleted contacts are excluded, and every table reference is qualified because
// customer_id, tm_create and others exist in both tables.
func (h *handler) ContactListStarred(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]*contact.Contact, error) {
	fields := commondatabasehandler.GetDBFields(&contact.Contact{})
	columns := make([]string, 0, len(fields))
	for _, f := range fields {
		columns = append(columns, "c."+f)
	}

	builder := sq.Select(columns...).
		From(contactTable + " c").
		Join(contactStarTable + " s ON s.contact_id = c.id AND s.customer_id = c.customer_id").
		Where(sq.Eq{"c.customer_id": customerID.Bytes()}).
		Where(sq.Eq{"s.agent_id": agentID.Bytes()}).
		Where(sq.Eq{"c.tm_delete": nil})

	if token != "" {
		cursor, err := parsePageToken(token)
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. ContactListStarred. err: %w", err)
		}
		builder = builder.Where(sq.Lt{"c.tm_create": cursor})
	}

	query, args, err := builder.OrderBy("c.tm_create DESC").Limit(size).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ContactListStarred. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ContactListStarred. err: %w", err)
	}

	// Drain and close the cursor before loading child data, like ContactList: a nested query on
	// an open cursor deadlocks a bounded pool.
	res := []*contact.Contact{}
	for rows.Next() {
		c, errScan := h.contactGetFromRow(rows)
		if errScan != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("could not scan the row. ContactListStarred. err: %w", errScan)
		}
		res = append(res, c)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("row iteration error. ContactListStarred. err: %w", err)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return nil, fmt.Errorf("could not close rows. ContactListStarred. err: %w", closeErr)
	}

	for _, c := range res {
		c.Addresses, err = h.AddressListByContactID(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get the addresses. ContactListStarred. err: %w", err)
		}
		c.TagIDs, err = h.TagAssignmentListByContactID(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get the tag ids. ContactListStarred. err: %w", err)
		}
	}

	return res, nil
}

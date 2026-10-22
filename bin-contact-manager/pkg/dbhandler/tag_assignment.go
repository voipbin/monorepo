package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

const (
	tagAssignmentTable = "contact_tag_assignments"
)

// TagAssignmentCreate creates a new tag assignment
func (h *handler) TagAssignmentCreate(ctx context.Context, contactID, tagID uuid.UUID) error {
	tmCreate := h.utilHandler.TimeNow()

	query, args, err := sq.Insert(tagAssignmentTable).
		Columns("contact_id", "tag_id", "tm_create").
		Values(contactID.Bytes(), tagID.Bytes(), tmCreate).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TagAssignmentCreate. err: %w", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. TagAssignmentCreate. err: %w", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, contactID)

	return nil
}

// TagAssignmentDelete deletes a tag assignment
func (h *handler) TagAssignmentDelete(ctx context.Context, contactID, tagID uuid.UUID) error {
	query, args, err := sq.Delete(tagAssignmentTable).
		Where(sq.Eq{
			"contact_id": contactID.Bytes(),
			"tag_id":     tagID.Bytes(),
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TagAssignmentDelete. err: %w", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. TagAssignmentDelete. err: %w", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, contactID)

	return nil
}

// TagAssignmentListByContactID returns all tag IDs for a contact
func (h *handler) TagAssignmentListByContactID(ctx context.Context, contactID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := sq.Select("tag_id").
		From(tagAssignmentTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		OrderBy("tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TagAssignmentListByContactID. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TagAssignmentListByContactID. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []uuid.UUID{}
	for rows.Next() {
		var tagIDBytes []byte
		if err := rows.Scan(&tagIDBytes); err != nil {
			return nil, fmt.Errorf("could not scan tag_id. TagAssignmentListByContactID. err: %w", err)
		}

		tagID, err := uuid.FromBytes(tagIDBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse tag_id. TagAssignmentListByContactID. err: %w", err)
		}

		res = append(res, tagID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. TagAssignmentListByContactID. err: %w", err)
	}

	return res, nil
}

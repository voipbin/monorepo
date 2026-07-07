package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

const (
	caseTagAssignmentTable = "contact_case_tag_assignments"
)

// CaseTagAssignmentCreate creates a new case-scoped tag assignment
// (design §7 round-22 correction). Mirrors TagAssignmentCreate exactly,
// scoped to case_id instead of contact_id. bin-tag-manager itself is
// unchanged -- Cases reference the same Tag rows (by tag_id) that
// Contacts already do; only this linking table is new.
func (h *handler) CaseTagAssignmentCreate(ctx context.Context, caseID, tagID uuid.UUID) error {
	tmCreate := h.utilHandler.TimeNow()

	query, args, err := sq.Insert(caseTagAssignmentTable).
		Columns("case_id", "tag_id", "tm_create").
		Values(caseID.Bytes(), tagID.Bytes(), tmCreate).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseTagAssignmentCreate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseTagAssignmentCreate. err: %v", err)
	}

	return nil
}

// CaseTagAssignmentDelete deletes a case-scoped tag assignment.
func (h *handler) CaseTagAssignmentDelete(ctx context.Context, caseID, tagID uuid.UUID) error {
	query, args, err := sq.Delete(caseTagAssignmentTable).
		Where(sq.Eq{
			"case_id": caseID.Bytes(),
			"tag_id":  tagID.Bytes(),
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseTagAssignmentDelete. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseTagAssignmentDelete. err: %v", err)
	}

	return nil
}

// CaseTagAssignmentListByCaseID returns all tag IDs for a case.
func (h *handler) CaseTagAssignmentListByCaseID(ctx context.Context, caseID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := sq.Select("tag_id").
		From(caseTagAssignmentTable).
		Where(sq.Eq{"case_id": caseID.Bytes()}).
		OrderBy("tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseTagAssignmentListByCaseID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseTagAssignmentListByCaseID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []uuid.UUID{}
	for rows.Next() {
		var tagIDBytes []byte
		if err := rows.Scan(&tagIDBytes); err != nil {
			return nil, fmt.Errorf("could not scan tag_id. CaseTagAssignmentListByCaseID. err: %v", err)
		}

		tagID, err := uuid.FromBytes(tagIDBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse tag_id. CaseTagAssignmentListByCaseID. err: %v", err)
		}

		res = append(res, tagID)
	}

	return res, nil
}

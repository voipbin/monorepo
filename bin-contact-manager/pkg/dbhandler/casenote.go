package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/casenote"
)

const caseNoteTable = "contact_case_notes"

// caseNoteGetFromRow scans a single row into a CaseNote struct.
func caseNoteGetFromRow(rows *sql.Rows) (*casenote.CaseNote, error) {
	res := &casenote.CaseNote{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. caseNoteGetFromRow. err: %v", err)
	}
	return res, nil
}

// CaseNoteCreate inserts a new CaseNote row (design §3.5). CaseNote is
// physically separate from contact_interactions -- never surfaced in
// any customer-facing webhook or API response.
func (h *handler) CaseNoteCreate(ctx context.Context, n *casenote.CaseNote) error {
	fields, err := commondatabasehandler.PrepareFields(n)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CaseNoteCreate. err: %v", err)
	}

	query, args, err := sq.Insert(caseNoteTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseNoteCreate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not create case note. CaseNoteCreate. err: %v", err)
	}

	return nil
}

// CaseNoteDelete soft-deletes a CaseNote by setting tm_delete = NOW().
// Scoped to customerID and caseID to prevent cross-tenant and cross-case
// deletion. Returns ErrNotFound if no active row matches.
func (h *handler) CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	query, args, err := sq.Update(caseNoteTable).
		Set("tm_delete", now).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"case_id": caseID.Bytes()}).
		Where(sq.Eq{"tm_delete": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseNoteDelete. err: %v", err)
	}

	result, err := h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. CaseNoteDelete. err: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected. CaseNoteDelete. err: %v", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// CaseNoteListByCase returns active (tm_delete IS NULL) notes for a
// given case, scoped to customerID, ordered by tm_create ascending.
func (h *handler) CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error) {
	columns := commondatabasehandler.GetDBFields(&casenote.CaseNote{})

	query, args, err := sq.Select(columns...).
		From(caseNoteTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"case_id": caseID.Bytes()}).
		Where(sq.Eq{"tm_delete": nil}).
		OrderBy("tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseNoteListByCase. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseNoteListByCase. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*casenote.CaseNote
	for rows.Next() {
		item, scanErr := caseNoteGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseNoteListByCase. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseNoteListByCase. err: %v", err)
	}

	return res, nil
}

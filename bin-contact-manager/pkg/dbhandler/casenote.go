package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		return nil, fmt.Errorf("could not scan the row. caseNoteGetFromRow. err: %w", err)
	}
	return res, nil
}

// CaseNoteCreate inserts a new CaseNote row (design §3.5). CaseNote is
// physically separate from contact_interactions -- never surfaced in
// any customer-facing webhook or API response.
func (h *handler) CaseNoteCreate(ctx context.Context, n *casenote.CaseNote) error {
	fields, err := commondatabasehandler.PrepareFields(n)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CaseNoteCreate. err: %w", err)
	}

	query, args, err := sq.Insert(caseNoteTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseNoteCreate. err: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not create case note. CaseNoteCreate. err: %w", err)
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
		return fmt.Errorf("could not build query. CaseNoteDelete. err: %w", err)
	}

	result, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. CaseNoteDelete. err: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected. CaseNoteDelete. err: %w", err)
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
		return nil, fmt.Errorf("could not build query. CaseNoteListByCase. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseNoteListByCase. err: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*casenote.CaseNote
	for rows.Next() {
		item, scanErr := caseNoteGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseNoteListByCase. err: %w", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseNoteListByCase. err: %w", err)
	}

	return res, nil
}

// CaseNoteListByContact returns active (tm_delete IS NULL) notes of every
// Case currently attributed to contactID, newest first (tm_create DESC),
// scoped to customerID on both the note and the case side. token is a
// tm_create cursor (exclusive); size is the row limit. The case side is
// matched on the current contact_id, so a re-attributed case moves its
// notes to the new contact.
func (h *handler) CaseNoteListByContact(ctx context.Context, customerID, contactID uuid.UUID, size uint64, token string) ([]*casenote.CaseNote, error) {
	fields := commondatabasehandler.GetDBFields(&casenote.CaseNote{})
	columns := make([]string, 0, len(fields))
	for _, f := range fields {
		columns = append(columns, "n."+f)
	}

	builder := sq.Select(columns...).
		From(caseNoteTable + " n").
		Join(caseTable + " c ON c.id = n.case_id AND c.customer_id = n.customer_id").
		Where(sq.Eq{"n.customer_id": customerID.Bytes()}).
		Where(sq.Eq{"c.contact_id": contactID.Bytes()}).
		Where(sq.Eq{"n.tm_delete": nil})

	if token != "" {
		cursorTime, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
		if err != nil {
			// Fallback to RFC3339Nano for forward compatibility.
			cursorTime, err = time.Parse(time.RFC3339Nano, token)
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. CaseNoteListByContact. err: %w", err)
		}
		builder = builder.Where(sq.Lt{"n.tm_create": cursorTime.UTC()})
	}

	query, args, err := builder.OrderBy("n.tm_create DESC").Limit(size).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseNoteListByContact. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseNoteListByContact. err: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*casenote.CaseNote
	for rows.Next() {
		item, scanErr := caseNoteGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseNoteListByContact. err: %w", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseNoteListByContact. err: %w", err)
	}

	return res, nil
}

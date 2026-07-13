package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/resolution"
)

const resolutionTable = "contact_resolutions"

// resolutionGetFromRow scans a single row into a Resolution struct.
func resolutionGetFromRow(rows *sql.Rows) (*resolution.Resolution, error) {
	res := &resolution.Resolution{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. resolutionGetFromRow. err: %v", err)
	}
	return res, nil
}

// ResolutionCreate inserts a new Resolution row into contact_resolutions.
func (h *handler) ResolutionCreate(ctx context.Context, r *resolution.Resolution) error {
	return resolutionCreateExec(h.db, r)
}

// ResolutionCreateTx is ResolutionCreate scoped to a caller-managed
// transaction (design §3.3's single-transaction requirement for the
// soft-delete-then-insert "replace" flow, and for the case-level create/
// delete write paths that must derive+write Case.contact_id atomically).
func (h *handler) ResolutionCreateTx(ctx context.Context, tx *sql.Tx, r *resolution.Resolution) error {
	return resolutionCreateExec(tx, r)
}

func resolutionCreateExec(exec sqlExecutor, r *resolution.Resolution) error {
	fields, err := commondatabasehandler.PrepareFields(r)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ResolutionCreate. err: %v", err)
	}

	query, args, err := sq.Insert(resolutionTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ResolutionCreate. err: %v", err)
	}

	_, err = exec.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not create resolution. ResolutionCreate. err: %v", err)
	}

	return nil
}

// ResolutionDelete soft-deletes an interaction-scoped resolution by setting
// tm_delete = NOW(). Scoped to customerID and interactionID to prevent
// cross-tenant and cross-interaction deletion. Returns ErrNotFound if no
// active row matches.
func (h *handler) ResolutionDelete(ctx context.Context, customerID, interactionID, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	query, args, err := sq.Update(resolutionTable).
		Set("tm_delete", now).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"interaction_id": interactionID.Bytes()}).
		Where(sq.Eq{"tm_delete": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ResolutionDelete. err: %v", err)
	}

	result, err := h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ResolutionDelete. err: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected. ResolutionDelete. err: %v", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ResolutionListByInteraction returns active resolutions for a given interaction,
// scoped to customerID. Active = tm_delete IS NULL.
func (h *handler) ResolutionListByInteraction(ctx context.Context, customerID, interactionID uuid.UUID) ([]*resolution.Resolution, error) {
	columns := commondatabasehandler.GetDBFields(&resolution.Resolution{})

	query, args, err := sq.Select(columns...).
		From(resolutionTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"interaction_id": interactionID.Bytes()}).
		Where(sq.Eq{"tm_delete": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ResolutionListByInteraction. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ResolutionListByInteraction. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*resolution.Resolution
	for rows.Next() {
		item, scanErr := resolutionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. ResolutionListByInteraction. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. ResolutionListByInteraction. err: %v", err)
	}

	return res, nil
}

// ResolutionListByContact returns active resolutions for a given contact,
// scoped to customerID. Active = tm_delete IS NULL.
func (h *handler) ResolutionListByContact(ctx context.Context, customerID, contactID uuid.UUID) ([]*resolution.Resolution, error) {
	columns := commondatabasehandler.GetDBFields(&resolution.Resolution{})

	query, args, err := sq.Select(columns...).
		From(resolutionTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		Where(sq.Eq{"tm_delete": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ResolutionListByContact. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ResolutionListByContact. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*resolution.Resolution
	for rows.Next() {
		item, scanErr := resolutionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. ResolutionListByContact. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. ResolutionListByContact. err: %v", err)
	}

	return res, nil
}

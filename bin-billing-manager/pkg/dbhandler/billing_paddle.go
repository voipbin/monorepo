package dbhandler

import (
	"context"
	"fmt"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
)

// BillingGetByIdempotencyKey returns the billing record with the given idempotency key.
// No soft-delete filter — idempotency must find records regardless of delete state (R4-I4).
func (h *handler) BillingGetByIdempotencyKey(ctx context.Context, idempotencyKey uuid.UUID) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "BillingGetByIdempotencyKey",
		"idempotency_key": idempotencyKey,
	})

	cols := commondatabasehandler.GetDBFields(billing.Billing{})

	query, args, err := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Eq{"idempotency_key": idempotencyKey.Bytes()}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query: %w", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	var res billing.Billing
	if err := commondatabasehandler.ScanRow(rows, &res); err != nil {
		log.Errorf("Could not scan row: %v", err)
		return nil, fmt.Errorf("could not scan row: %w", err)
	}

	return &res, nil
}

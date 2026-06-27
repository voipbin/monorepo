package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

const (
	// addressTable is the unified store for a contact's identifiers
	// (phone numbers and emails), replacing the legacy contact_phone_numbers
	// and contact_emails tables (VOIP-1207).
	addressTable = "contact_addresses"

	// addressTypeTel and addressTypeEmail are the channel-type discriminators
	// stored in contact_addresses.type.
	addressTypeTel   = "tel"
	addressTypeEmail = "email"
)

// addressRow mirrors the contact_addresses columns the handler reads back. It is
// an internal scan target: PhoneNumber/Email no longer map 1:1 to the table, so
// rows are scanned here (reusing the shared uuid/time conversion helpers) and
// then hand-mapped into the model structs (target -> Number / Address).
type addressRow struct {
	ID         uuid.UUID  `db:"id,uuid"`
	CustomerID uuid.UUID  `db:"customer_id,uuid"`
	ContactID  uuid.UUID  `db:"contact_id,uuid"`
	Target     string     `db:"target"`
	IsPrimary  bool       `db:"is_primary"`
	TMCreate   *time.Time `db:"tm_create"`
}

// addressRowColumns is the ordered SELECT column list matching addressRow.
func addressRowColumns() []string {
	return commondatabasehandler.GetDBFields(&addressRow{})
}

// scanAddressRow scans a single contact_addresses row into addressRow.
func scanAddressRow(rows *sql.Rows) (*addressRow, error) {
	res := &addressRow{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. scanAddressRow. err: %v", err)
	}
	return res, nil
}

// addressContactID returns the contact_id of an address row by id (any type),
// used to refresh the contact cache after mutations. Returns uuid.Nil if the
// row is absent.
func (h *handler) addressContactID(id uuid.UUID) (uuid.UUID, error) {
	query, args, err := sq.Select("contact_id").
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not build query. addressContactID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not query. addressContactID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return uuid.Nil, nil
	}

	var contactIDBytes []byte
	if err := rows.Scan(&contactIDBytes); err != nil {
		return uuid.Nil, fmt.Errorf("could not scan contact_id. addressContactID. err: %v", err)
	}
	if len(contactIDBytes) == 0 {
		return uuid.Nil, nil
	}

	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse contact_id. addressContactID. err: %v", err)
	}
	return contactID, nil
}

// addressResetPrimaryForContact clears is_primary for ALL addresses of a
// contact, across BOTH tel and email types. The contact_addresses table
// enforces UNIQUE(customer_id, primary_contact_uk) where primary_contact_uk is
// a generated column = IF(is_primary=1, contact_id, NULL). This makes "primary"
// a single-row-per-contact concept spanning every channel type, so resetting
// the primary flag for a new primary must demote any existing primary of either
// type. Both PhoneNumberResetPrimary and EmailResetPrimary delegate here.
func (h *handler) addressResetPrimaryForContact(_ context.Context, contactID uuid.UUID) error {
	query, args, err := sq.Update(addressTable).
		Set("is_primary", false).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. addressResetPrimaryForContact. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. addressResetPrimaryForContact. err: %v", err)
	}

	return nil
}

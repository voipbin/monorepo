package dbhandler

// Contact-address ownership period READ path (Phase 2 of
// NOJIRA-contact-address-ownership-periods). See
// docs/plans/2026-07-11-contact-address-ownership-integrity-design.md
// §6 for the full design and rationale.
//
// This file adds the two new STEP1 dbhandler primitives
// interactionListByContact needs to match interactions against ownership
// history instead of only currently-live contact_addresses rows:
//   - OwnershipPeriodsListByContactID: every period (open or closed) this
//     Contact has ever held.
//   - missingPeriodOwnedAddresses: rows this Contact currently, live-owns
//     that have NO open period of their own (design §6.2 round-41-43/47's
//     missing-period-skew guard -- an old-binary pod's AddressCreate that
//     ran before this rewire skipped the period write).
//
// Neither function touches AddressListByContactID (address.go) --
// design §6.1 requires that function stay untouched, since it also backs
// the public Contact.Addresses API field via ContactGet/ContactList.

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

// OwnershipPeriodsListByContactID returns every ownership period (open
// and closed) this Contact has ever held, across every (type, target) it
// has owned over its lifetime -- not just its currently-live addresses.
// Used exclusively by interactionListByContact's STEP1 (design §6.2) and
// carries no locking (a plain SELECT, unlike
// OwnershipPeriodsLockAndResolveTx's FOR UPDATE read inside a write
// transaction).
func (h *handler) OwnershipPeriodsListByContactID(ctx context.Context, contactID uuid.UUID) ([]OwnershipPeriod, error) {
	query, args, err := sq.Select("id", "customer_id", "contact_id", "type", "target", "valid_from", "valid_to").
		From(ownershipPeriodTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OwnershipPeriodsListByContactID. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OwnershipPeriodsListByContactID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []OwnershipPeriod
	for rows.Next() {
		p, err := scanOwnershipPeriodRowFull(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OwnershipPeriodsListByContactID. err: %v", err)
		}
		res = append(res, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. OwnershipPeriodsListByContactID. err: %v", err)
	}

	return res, nil
}

// MissingPeriodAddress is a (type, target) pair this Contact currently,
// live-owns but that has zero ownership-period rows of its own (design
// §6.2 rounds 41-43/47's missing-period-skew population). Exported: this
// is a DBHandler interface return type, same visibility rule as every
// other cross-package dbhandler result type (OwnershipPeriod, AddressPair).
type MissingPeriodAddress struct {
	Type   string
	Target string
}

// MissingPeriodOwnedAddresses returns (type, target) pairs this Contact
// currently owns (a live contact_addresses row with contact_id = this
// Contact) for which contact_address_ownership_periods has NO OPEN period
// belonging to this SAME contact_id (design §6.2 round-43's final,
// owner-plus-open-scoped condition -- narrower conditions considered and
// rejected at rounds 41/42 for missing the reassignment and
// same-owner-reacquisition cases respectively). Each result is meant to
// be attached to STEP2 as an unbounded ([nil, nil)) bound, reproducing
// today's pure value-match for exactly this transient, degraded
// population until the next write gives the row a real period.
func (h *handler) MissingPeriodOwnedAddresses(ctx context.Context, customerID, contactID uuid.UUID) ([]MissingPeriodAddress, error) {
	query, args, err := sq.Select("a.type", "a.target").
		From(addressTable+" a").
		Where(sq.Eq{"a.customer_id": customerID.Bytes()}).
		Where(sq.Eq{"a.contact_id": contactID.Bytes()}).
		Where(sq.Expr(
			`NOT EXISTS (
				SELECT 1 FROM `+ownershipPeriodTable+` p2
				WHERE p2.customer_id = a.customer_id
				  AND p2.type = a.type
				  AND p2.target = a.target
				  AND p2.contact_id = a.contact_id
				  AND p2.valid_to IS NULL
			)`,
		)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. missingPeriodOwnedAddresses. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. missingPeriodOwnedAddresses. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []MissingPeriodAddress
	for rows.Next() {
		var t, target string
		if err := rows.Scan(&t, &target); err != nil {
			return nil, fmt.Errorf("could not scan the row. MissingPeriodOwnedAddresses. err: %v", err)
		}
		res = append(res, MissingPeriodAddress{Type: t, Target: target})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. MissingPeriodOwnedAddresses. err: %v", err)
	}

	return res, nil
}

// scanOwnershipPeriodRowFull scans a full (id, customer_id, contact_id,
// type, target, valid_from, valid_to) row -- unlike scanOwnershipPeriodRow
// (address_ownership.go), which takes customerID/addrType/target as
// parameters because its caller already knows them (scoped to a single
// (customer_id, type, target) tuple under lock). OwnershipPeriodsListByContactID
// spans multiple (type, target) pairs per contact, so type/target must be
// read from the row itself.
func scanOwnershipPeriodRowFull(rows *sql.Rows) (*OwnershipPeriod, error) {
	var idBytes, customerIDBytes, contactIDBytes []byte
	var addrType, target string
	var validFrom, validTo sql.NullString
	if err := rows.Scan(&idBytes, &customerIDBytes, &contactIDBytes, &addrType, &target, &validFrom, &validTo); err != nil {
		return nil, fmt.Errorf("could not scan the row. scanOwnershipPeriodRowFull. err: %v", err)
	}
	id, err := uuid.FromBytes(idBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse id. scanOwnershipPeriodRowFull. err: %v", err)
	}
	customerID, err := uuid.FromBytes(customerIDBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse customer_id. scanOwnershipPeriodRowFull. err: %v", err)
	}
	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse contact_id. scanOwnershipPeriodRowFull. err: %v", err)
	}
	validFromPtr, err := parseNullableDBTime(validFrom)
	if err != nil {
		return nil, fmt.Errorf("could not parse valid_from. scanOwnershipPeriodRowFull. err: %v", err)
	}
	validToPtr, err := parseNullableDBTime(validTo)
	if err != nil {
		return nil, fmt.Errorf("could not parse valid_to. scanOwnershipPeriodRowFull. err: %v", err)
	}
	return &OwnershipPeriod{
		ID:         id,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       addrType,
		Target:     target,
		ValidFrom:  validFromPtr,
		ValidTo:    validToPtr,
	}, nil
}

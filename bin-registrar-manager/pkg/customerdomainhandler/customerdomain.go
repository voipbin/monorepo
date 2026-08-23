package customerdomainhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// EnsureByCustomerID returns the existing customer domain row or creates one.
// New-row shape follows the short-label flag (see the interface doc): legacy
// uuid label/realm when disabled (pre-cutover default), fresh 4-char base36
// label when enabled. Idempotent: a concurrent create losing the race falls
// back to reading the winner's row.
func (h *customerDomainHandler) EnsureByCustomerID(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EnsureByCustomerID",
		"customer_id": customerID,
	})

	if customerID == uuid.Nil {
		return nil, fmt.Errorf("invalid customer id")
	}

	res, err := h.dbBin.CustomerDomainGet(ctx, customerID)
	if err == nil {
		return res, nil
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		log.Errorf("Could not get customer domain info. err: %v", err)
		return nil, fmt.Errorf("could not get customer domain: %w", err)
	}

	if !h.shortLabelEnabled {
		// deploy window (gates 1a-1d before the frontend deploy): frontends
		// still build the domain from the customer uuid, so a NEW customer
		// must get the legacy uuid realm to keep its webphone working
		return h.createLegacyDomain(ctx, customerID)
	}

	return h.createShortDomain(ctx, customerID)
}

// createLegacyDomain creates the customer's domain row in the legacy form
// (label = customer uuid string, realm = <uuid>.<base>) — exactly the shape
// the backfill produces. The migration batch later rewrites it to a short
// label like every other backfilled row.
func (h *customerDomainHandler) createLegacyDomain(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "createLegacyDomain",
		"customer_id": customerID,
	})

	cd := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       fmt.Sprintf("%s.%s", customerID.String(), common.BaseDomainNameExtension()),
	}

	errCreate := h.dbBin.CustomerDomainCreate(ctx, cd)
	if errCreate == nil {
		log.Debugf("Created legacy customer domain. label: %s, realm: %s", cd.DomainLabel, cd.Realm)
		return h.dbBin.CustomerDomainGet(ctx, customerID)
	}

	if stderrors.Is(errCreate, dbhandler.ErrDuplicate) {
		// the legacy label/realm are deterministic per customer: a duplicate
		// means a concurrent create won the race — return the winner's row
		return h.dbBin.CustomerDomainGet(ctx, customerID)
	}

	log.Errorf("Could not create customer domain. err: %v", errCreate)
	return nil, fmt.Errorf("could not create customer domain: %w", errCreate)
}

// createShortDomain creates the customer's domain row with a fresh 4-char
// base36 label, retrying on label collisions. Used by EnsureByCustomerID when
// short labels are enabled and UNCONDITIONALLY by the migration batch
// (MigrateToShortDomain) — the batch is the cutover mechanism and is not
// gated by the short-label flag.
func (h *customerDomainHandler) createShortDomain(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "createShortDomain",
		"customer_id": customerID,
	})

	for i := 0; i < labelGenerateMaxAttempts; i++ {
		label, errGenerate := generateLabel()
		if errGenerate != nil {
			log.Errorf("Could not generate the domain label. err: %v", errGenerate)
			return nil, fmt.Errorf("could not generate domain label: %w", errGenerate)
		}

		cd := &customerdomain.CustomerDomain{
			CustomerID:  customerID,
			DomainLabel: label,
			Realm:       fmt.Sprintf("%s.%s", label, common.BaseDomainNameExtension()),
		}

		errCreate := h.dbBin.CustomerDomainCreate(ctx, cd)
		if errCreate == nil {
			log.Debugf("Created customer domain. label: %s, realm: %s", cd.DomainLabel, cd.Realm)
			return h.dbBin.CustomerDomainGet(ctx, customerID)
		}

		if !stderrors.Is(errCreate, dbhandler.ErrDuplicate) {
			log.Errorf("Could not create customer domain. err: %v", errCreate)
			return nil, fmt.Errorf("could not create customer domain: %w", errCreate)
		}

		// duplicate key: either the label collided (retry with a new label) or
		// a concurrent create won the customer_id race (return the winner).
		res, errGet := h.dbBin.CustomerDomainGet(ctx, customerID)
		if errGet == nil {
			return res, nil
		}

		log.Debugf("Domain label collision. Retrying. label: %s, attempt: %d", label, i+1)
	}

	return nil, fmt.Errorf("could not create customer domain: label generation attempts exhausted. customer_id: %s", customerID)
}

// GetByCustomerID returns the customer domain of the given customer. Lookup-only.
func (h *customerDomainHandler) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	res, err := h.dbBin.CustomerDomainGet(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetByRealm returns the customer domain of the given realm. Cache-backed, lookup-only.
func (h *customerDomainHandler) GetByRealm(ctx context.Context, realm string) (*customerdomain.CustomerDomain, error) {
	if realm == "" {
		return nil, fmt.Errorf("invalid realm")
	}

	res, err := h.dbBin.CustomerDomainGetByRealm(ctx, realm)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DeleteByCustomerID hard-deletes the customer's domain row.
// Idempotent: deleting a non-existent row is not an error.
func (h *customerDomainHandler) DeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DeleteByCustomerID",
		"customer_id": customerID,
	})

	if err := h.dbBin.CustomerDomainDelete(ctx, customerID); err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			log.Debugf("Customer domain does not exist. Nothing to delete. customer_id: %s", customerID)
			return nil
		}

		log.Errorf("Could not delete customer domain. err: %v", err)
		return fmt.Errorf("could not delete customer domain: %w", err)
	}

	return nil
}

// RealmGet returns the realm of the given customer, lazily creating the domain
// row when missing. Extension-create path only.
func (h *customerDomainHandler) RealmGet(ctx context.Context, customerID uuid.UUID) (string, error) {
	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		return "", err
	}

	return res.Realm, nil
}

// EndpointGet returns the endpoint string (<extension>@<realm>) of the given
// customer and extension. Lookup-only: a contact lookup must never create a
// domain row as a side effect. When no row exists yet (pre-backfill window)
// it falls back to the computed legacy realm so existing contact lookups keep
// resolving the endpoints provisioned under the legacy uuid realm.
func (h *customerDomainHandler) EndpointGet(ctx context.Context, customerID uuid.UUID, extension string) (string, error) {
	res, err := h.GetByCustomerID(ctx, customerID)
	if err == nil {
		return fmt.Sprintf("%s@%s", extension, res.Realm), nil
	}

	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		return "", fmt.Errorf("could not get customer domain: %w", err)
	}

	legacy := common.GenerateRealmExtensionLegacy(customerID)
	if legacy == "" {
		return "", fmt.Errorf("could not get endpoint: no customer domain row and no base domain configured. customer_id: %s", customerID)
	}

	logrus.WithFields(logrus.Fields{
		"func":        "EndpointGet",
		"customer_id": customerID,
	}).Debugf("No customer domain row. Falling back to the legacy computed realm. realm: %s", legacy)

	return fmt.Sprintf("%s@%s", extension, legacy), nil
}

// MigrateToShortDomain moves the customer's domain row to a fresh 4-char label.
//   - no row: creates a short-label row directly (createShortDomain).
//   - backfilled long-label row: rewrites label+realm in place with duplicate retry.
//   - already-short row: no-op (idempotent resume).
//
// NOT gated by the short-label flag: the batch is the cutover mechanism and
// must produce short labels even while EnsureByCustomerID still creates
// legacy rows for new signups.
func (h *customerDomainHandler) MigrateToShortDomain(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MigrateToShortDomain",
		"customer_id": customerID,
	})

	cd, err := h.dbBin.CustomerDomainGet(ctx, customerID)
	if err != nil {
		if !stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, fmt.Errorf("could not get customer domain: %w", err)
		}

		// no row yet: create a short-label row directly (flag-independent)
		return h.createShortDomain(ctx, customerID)
	}

	if len(cd.DomainLabel) == labelLength {
		// already migrated (idempotent resume)
		return cd, nil
	}

	for i := 0; i < labelGenerateMaxAttempts; i++ {
		label, errGenerate := generateLabel()
		if errGenerate != nil {
			return nil, fmt.Errorf("could not generate domain label: %w", errGenerate)
		}

		realm := fmt.Sprintf("%s.%s", label, common.BaseDomainNameExtension())
		fields := map[customerdomain.Field]any{
			customerdomain.FieldDomainLabel: label,
			customerdomain.FieldRealm:       realm,
		}

		errUpdate := h.dbBin.CustomerDomainUpdate(ctx, customerID, fields)
		if errUpdate == nil {
			return h.dbBin.CustomerDomainGet(ctx, customerID)
		}

		if !stderrors.Is(errUpdate, dbhandler.ErrDuplicate) {
			return nil, fmt.Errorf("could not update customer domain: %w", errUpdate)
		}

		log.Debugf("Domain label collision. Retrying. label: %s, attempt: %d", label, i+1)
	}

	return nil, fmt.Errorf("could not migrate customer domain: label generation attempts exhausted. customer_id: %s", customerID)
}

// UpdateByCustomerID sets the customer's domain row to the given explicit
// label/realm (migration rollback replay).
func (h *customerDomainHandler) UpdateByCustomerID(ctx context.Context, customerID uuid.UUID, domainLabel string, realm string) (*customerdomain.CustomerDomain, error) {
	if domainLabel == "" || realm == "" {
		return nil, fmt.Errorf("invalid domain label or realm")
	}

	fields := map[customerdomain.Field]any{
		customerdomain.FieldDomainLabel: domainLabel,
		customerdomain.FieldRealm:       realm,
	}

	if errUpdate := h.dbBin.CustomerDomainUpdate(ctx, customerID, fields); errUpdate != nil {
		return nil, fmt.Errorf("could not update customer domain: %w", errUpdate)
	}

	return h.dbBin.CustomerDomainGet(ctx, customerID)
}

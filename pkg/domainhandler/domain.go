package domainhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

// DomainCreate creates a new domain and returns a created domain info
func (h *domainHandler) DomainCreate(ctx context.Context, d *domain.Domain) (*domain.Domain, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"domain": d,
		},
	)
	log.Debugf("Creating domain. domain: %s", d.DomainName)

	// check suffix
	if strings.HasSuffix(d.DomainName, constDomainSuffix) == false {
		log.Errorf("Wrong domain name. domain_name: %s, suffix: %s", d.DomainName, constDomainSuffix)
		return nil, fmt.Errorf("wrong domain name. suffix must matched with %s", constDomainSuffix)
	}

	// check duplicated domain
	_, err := h.dbBin.DomainGetByDomainName(ctx, d.DomainName)
	if err == nil {
		logrus.Errorf("The given domain is already exists. err: %v", err)
		return nil, fmt.Errorf("already exists")
	}

	// create new domain
	d.ID = uuid.Must(uuid.NewV4())
	if err := h.dbBin.DomainCreate(ctx, d); err != nil {
		logrus.Errorf("Could not create a domain info. err: %v", err)
		return nil, err
	}
	log.Debugf("Created new domain. domain: %s, domain_name: %s", d.ID, d.DomainName)

	res, err := h.dbBin.DomainGet(ctx, d.ID)
	if err != nil {
		logrus.Errorf("Could not get created domain info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetDomain returns domain
func (h *domainHandler) DomainGet(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {
	res, err := h.dbBin.DomainGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Gets returns list of domains
func (h *domainHandler) Gets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*domain.Domain, error) {

	domains, err := h.dbBin.DomainGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		logrus.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	return domains, nil
}

// DomainUpdate updates the domain info
func (h *domainHandler) DomainUpdate(ctx context.Context, d *domain.Domain) (*domain.Domain, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"domain": d,
		},
	)

	// update
	if err := h.dbBin.DomainUpdate(ctx, d); err != nil {
		log.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.DomainGet(ctx, d.ID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DomainDelete deletes the domain info
func (h *domainHandler) DomainDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"domain": id,
		},
	)

	// delete extensions
	if err := h.extHandler.ExtensionDeleteByDomainID(ctx, id); err != nil {
		return err
	}

	// delete domain
	if err := h.dbBin.DomainDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the domain. err: %v", err)
		return err
	}

	return nil
}

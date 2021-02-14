package domainhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

// DomainCreate creates a new domain and returns a created domain info
func (h *domainHandler) DomainCreate(ctx context.Context, d *models.Domain) (*models.Domain, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"domain": d,
		},
	)
	log.Debugf("Creating domain. domain: %s", d.DomainName)

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
func (h *domainHandler) DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error) {
	res, err := h.dbBin.DomainGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DomainGetsByUserID returns list of domains
func (h *domainHandler) DomainGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Domain, error) {

	domains, err := h.dbBin.DomainGetsByUserID(ctx, userID, token, limit)
	if err != nil {
		logrus.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	return domains, nil
}

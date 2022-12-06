package domainhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

// Create creates a new domain and returns a created domain info
func (h *domainHandler) Create(ctx context.Context, customerID uuid.UUID, domainName, name, detail string) (*domain.Domain, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Create",
			"customer_id":       customerID,
			"domain_domainname": domainName,
		},
	)
	log.Debugf("Creating domain. domain: %s", domainName)

	// check suffix
	if !strings.HasSuffix(domainName, constDomainSuffix) {
		log.Errorf("Wrong domain name. domain_name: %s, suffix: %s", domainName, constDomainSuffix)
		return nil, fmt.Errorf("wrong domain name. suffix must matched with %s", constDomainSuffix)
	}

	// check duplicated domain
	_, err := h.dbBin.DomainGetByDomainName(ctx, domainName)
	if err == nil {
		logrus.Errorf("The given domain is already exists. err: %v", err)
		return nil, fmt.Errorf("already exists")
	}

	// create new domain
	d := &domain.Domain{
		ID:         uuid.Must(uuid.NewV4()),
		CustomerID: customerID,

		Name:       name,
		Detail:     detail,
		DomainName: domainName,
	}

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
	h.notifyHandler.PublishEvent(ctx, domain.EventTypeDomainCreated, res)
	promDomainCreateTotal.Inc()

	return res, nil
}

// Get returns domain
func (h *domainHandler) Get(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {
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

// Update updates the domain info
func (h *domainHandler) Update(ctx context.Context, id uuid.UUID, name, detail string) (*domain.Domain, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Update",
			"domain_id": id,
		},
	)

	// update
	if err := h.dbBin.DomainUpdateBasicInfo(ctx, id, name, detail); err != nil {
		log.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.DomainGet(ctx, id)
	if err != nil {
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, domain.EventTypeDomainUpdated, res)

	return res, nil
}

// Delete deletes the domain info
func (h *domainHandler) Delete(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Delete",
			"domain_id": id,
		},
	)

	// delete extensions
	tmp, err := h.extHandler.ExtensionDeleteByDomainID(ctx, id)
	if err != nil {
		return nil, err
	}
	log.WithField("extensions", tmp).Debugf("Deleted extensions. count: %d", len(tmp))

	// delete domain
	if err := h.dbBin.DomainDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the domain. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.DomainGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted domain. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, domain.EventTypeDomainDeleted, res)
	promDomainDeleteTotal.Inc()

	return res, nil
}

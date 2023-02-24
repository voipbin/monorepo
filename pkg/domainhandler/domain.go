package domainhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
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

	if !isValidDomainName(domainName) {
		log.Errorf("Invalid domain name. domain_name: %s", domainName)
		return nil, fmt.Errorf("invalid domain name")
	}

	// check duplicated domain
	tmp, err := h.GetByDomainName(ctx, domainName)
	if err == nil {
		if tmp.TMDelete < dbhandler.DefaultTimeStamp {
			log.Errorf("The given domain is already existed. err: %v", err)
			return nil, fmt.Errorf("already exists")
		}
	}

	// create new domain
	id := h.utilHandler.CreateUUID()
	d := &domain.Domain{
		ID:         id,
		CustomerID: customerID,

		Name:       name,
		Detail:     detail,
		DomainName: domainName,
	}

	if err := h.dbBin.DomainCreate(ctx, d); err != nil {
		log.Errorf("Could not create a domain info. err: %v", err)
		return nil, err
	}
	log.Debugf("Created new domain. domain: %s, domain_name: %s", d.ID, d.DomainName)

	res, err := h.Get(ctx, d.ID)
	if err != nil {
		log.Errorf("Could not get created domain info. err: %v", err)
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

// GetByDomainName returns domain of the given domain name
func (h *domainHandler) GetByDomainName(ctx context.Context, domainName string) (*domain.Domain, error) {
	res, err := h.dbBin.DomainGetByDomainName(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Gets returns list of domains
func (h *domainHandler) Gets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*domain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
	})

	res, err := h.dbBin.DomainGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	return res, nil
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
	if errUpdate := h.dbBin.DomainUpdateBasicInfo(ctx, id, name, detail); errUpdate != nil {
		log.Errorf("Could not update the domain. err: %v", errUpdate)
		return nil, errUpdate
	}

	res, err := h.Get(ctx, id)
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
	tmp, err := h.extHandler.DeleteByDomainID(ctx, id)
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

package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
)

// domainGet validates the domain's ownership and returns the domain info.
func (h *serviceHandler) domainGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*domain.Domain, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "domainGet",
			"customer_id": u.ID,
			"domain_id":   id,
		},
	)

	// send request
	tmp, err := h.reqHandler.RMV1DomainGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return nil, err
	}
	log.WithField("domain", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this domain.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := domain.ConvertToDomain(tmp)
	return res, nil
}

// DomainCreate is a service handler for flow creation.
func (h *serviceHandler) DomainCreate(u *cscustomer.Customer, domainName, name, detail string) (*domain.Domain, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"domain_name": domainName,
		"name":        name,
	})
	log.Debug("Creating a new domain.")

	tmp, err := h.reqHandler.RMV1DomainCreate(ctx, u.ID, domainName, name, detail)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := domain.ConvertToDomain(tmp)
	return res, nil
}

// DomainDelete deletes the domain of the given id.
func (h *serviceHandler) DomainDelete(u *cscustomer.Customer, id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"domain_id":   id,
	})
	log.Debug("Deleting the domain.")

	// get domain
	_, err := h.domainGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return fmt.Errorf("could not get domain info. err: %v", err)
	}

	if err := h.reqHandler.RMV1DomainDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

// DomainGet gets the domain of the given id.
// It returns domain if it succeed.
func (h *serviceHandler) DomainGet(u *cscustomer.Customer, id uuid.UUID) (*domain.Domain, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"domain_id":   id,
	})
	log.Debug("Getting a domain.")

	// get domain
	res, err := h.domainGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not get domain info. err: %v", err)
	}

	return res, nil
}

// DomainGets gets the list of domains of the given customer id.
// It returns list of domains if it succeed.
func (h *serviceHandler) DomainGets(u *cscustomer.Customer, size uint64, token string) ([]*domain.Domain, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a domains.")

	if token == "" {
		token = getCurTime()
	}

	// get domains
	domains, err := h.reqHandler.RMV1DomainGets(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get domains info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domains info. err: %v", err)
	}

	// create result
	res := []*domain.Domain{}
	for _, d := range domains {
		tmp := domain.ConvertToDomain(&d)
		res = append(res, tmp)
	}

	return res, nil
}

// DomainUpdate updates the flow info.
// It returns updated domain if it succeed.
func (h *serviceHandler) DomainUpdate(u *cscustomer.Customer, d *domain.Domain) (*domain.Domain, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"domain":      d.ID,
	})
	log.Debug("Updating a domain.")

	// get
	_, err := h.domainGet(ctx, u, d.ID)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	// update
	reqDomain := domain.CreateDomain(d)
	res, err := h.reqHandler.RMV1DomainUpdate(ctx, reqDomain)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	resDomain := domain.ConvertToDomain(res)
	return resDomain, nil
}

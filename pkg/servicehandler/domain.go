package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// DomainCreate is a service handler for flow creation.
func (h *serviceHandler) DomainCreate(u *user.User, domainName, name, detail string) (*domain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":        u.ID,
		"domain_name": domainName,
		"name":        name,
	})
	log.Debug("Creating a new domain.")

	tmp, err := h.reqHandler.RMDomainCreate(u.ID, domainName, name, detail)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := domain.ConvertDomain(tmp)
	return res, nil
}

// DomainDelete deletes the domain of the given id.
func (h *serviceHandler) DomainDelete(u *user.User, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"user":      u.ID,
		"username":  u.Username,
		"domain_id": id,
	})
	log.Debug("Deleting a domain.")

	// get domain
	domain, err := h.reqHandler.RMDomainGet(id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(user.PermissionAdmin) && domain.UserID != u.ID {
		log.Errorf("The user has no permission for this flow. user: %d, domain_user: %d", u.ID, domain.UserID)
		return fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.RMDomainDelete(id); err != nil {
		return err
	}

	return nil
}

// DomainGet gets the domain of the given id.
// It returns domain if it succeed.
func (h *serviceHandler) DomainGet(u *user.User, id uuid.UUID) (*domain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":      u.ID,
		"username":  u.Username,
		"domain_id": id,
	})
	log.Debug("Getting a domain.")

	// get domain
	d, err := h.reqHandler.RMDomainGet(id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(user.PermissionAdmin) && d.UserID != u.ID {
		log.Errorf("The user has no permission for this flow. user: %d, domain_user: %d", u.ID, d.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	res := domain.ConvertDomain(d)
	return res, nil
}

// DomainGets gets the list of domains of the given user id.
// It returns list of domains if it succeed.
func (h *serviceHandler) DomainGets(u *user.User, size uint64, token string) ([]*domain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})
	log.Debug("Getting a domains.")

	if token == "" {
		token = getCurTime()
	}

	// get domains
	domains, err := h.reqHandler.RMDomainGets(u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get domains info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domains info. err: %v", err)
	}

	// create result
	res := []*domain.Domain{}
	for _, d := range domains {
		tmp := domain.ConvertDomain(&d)
		res = append(res, tmp)
	}

	return res, nil
}

// DomainUpdate updates the flow info.
// It returns updated domain if it succeed.
func (h *serviceHandler) DomainUpdate(u *user.User, d *domain.Domain) (*domain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"domain":   d.ID,
	})
	log.Debug("Updating a domain.")

	// get flows
	tmpDomain, err := h.reqHandler.RMDomainGet(d.ID)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	// check the ownership
	if u.Permission != user.PermissionAdmin && u.ID != tmpDomain.UserID {
		log.Info("The user has no permission for this domain.")
		return nil, fmt.Errorf("user has no permission")
	}

	reqDomain := domain.CreateDomain(d)
	res, err := h.reqHandler.RMDomainUpdate(reqDomain)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	resDomain := domain.ConvertDomain(res)
	return resDomain, nil
}

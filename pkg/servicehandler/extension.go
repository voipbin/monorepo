package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
)

// ExtensionCreate is a service handler for flow creation.
func (h *serviceHandler) ExtensionCreate(u *cscustomer.Customer, e *extension.Extension) (*extension.Extension, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"domain_id":   e.DomainID,
		"name":        e.Name,
		"detail":      e.Detail,
	})
	log.Debug("Creating a new extension.")

	ext := extension.CreateDomain(e)
	ext.CustomerID = u.ID
	tmp, err := h.reqHandler.RMV1ExtensionCreate(ctx, ext)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := extension.ConvertExtension(tmp)
	return res, nil
}

// ExtensionDelete deletes the extension of the given id.
func (h *serviceHandler) ExtensionDelete(u *cscustomer.Customer, id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Deleting a extension.")

	// get extension
	ext, err := h.reqHandler.RMV1ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return fmt.Errorf("could not find extension info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != ext.CustomerID {
		log.Errorf("The customer has no permission for this extension. customer: %s, domain_customer: %s", u.ID, ext.CustomerID)
		return fmt.Errorf("customer has no permission")
	}

	if err := h.reqHandler.RMV1ExtensionDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

// ExtensionGet gets the extension of the given id.
// It returns extension if it succeed.
func (h *serviceHandler) ExtensionGet(u *cscustomer.Customer, id uuid.UUID) (*extension.Extension, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Getting a extension.")

	// get extension
	d, err := h.reqHandler.RMV1ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != d.CustomerID {
		log.Errorf("The customer has no permission for this extension. customer_id: %s, extension_customer: %s", u.ID, d.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	res := extension.ConvertExtension(d)
	return res, nil
}

// ExtensionGets gets the list of extensions of the given customer id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionGets(u *cscustomer.Customer, domainID uuid.UUID, size uint64, token string) ([]*extension.Extension, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a extensions.")

	if token == "" {
		token = getCurTime()
	}

	// get extensions
	exts, err := h.reqHandler.RMV1ExtensionGets(ctx, domainID, token, size)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extensions info. err: %v", err)
	}

	// create result
	res := []*extension.Extension{}
	for _, ext := range exts {
		tmp := extension.ConvertExtension(&ext)
		res = append(res, tmp)
	}

	return res, nil
}

// ExtesnionUpdate updates the extension info.
// It returns updated extension if it succeed.
func (h *serviceHandler) ExtensionUpdate(u *cscustomer.Customer, d *extension.Extension) (*extension.Extension, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"extension":   d.ID,
	})
	log.Debug("Updating an extension.")

	// get extension
	tmpExtension, err := h.reqHandler.RMV1ExtensionGet(ctx, d.ID)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpExtension.CustomerID {
		log.Info("The customer has no permission for this extension.")
		return nil, fmt.Errorf("customer has no permission")
	}

	reqExtension := extension.CreateDomain(d)
	res, err := h.reqHandler.RMV1ExtensionUpdate(ctx, reqExtension)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	resExtension := extension.ConvertExtension(res)
	return resExtension, nil
}

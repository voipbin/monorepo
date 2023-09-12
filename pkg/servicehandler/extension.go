package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
)

// extensionGet validates the extension's ownership and returns the extension info.
func (h *serviceHandler) extensionGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmextension.Extension, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "extensionGet",
			"customer_id":  u.ID,
			"extension_id": id,
		},
	)

	// send request
	res, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get an tag. err: %v", err)
		return nil, err
	}
	log.WithField("tag", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	return res, nil
}

// ExtensionCreate is a service handler for flow creation.
func (h *serviceHandler) ExtensionCreate(ctx context.Context, u *cscustomer.Customer, ext, password string, domainID uuid.UUID, name, detail string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"domain_id":   domainID,
		"extension":   ext,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new extension.")

	tmp, err := h.reqHandler.RegistrarV1ExtensionCreate(ctx, u.ID, ext, password, domainID, name, detail)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionDelete deletes the extension of the given id.
func (h *serviceHandler) ExtensionDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Deleting a extension.")

	_, err := h.extensionGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the extension. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionGet gets the extension of the given id.
// It returns extension if it succeed.
func (h *serviceHandler) ExtensionGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Getting a extension.")

	tmp, err := h.extensionGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionGetsByDomainID gets the list of extensions of the given domain id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionGetsByDomainID(ctx context.Context, u *cscustomer.Customer, domainID uuid.UUID, size uint64, token string) ([]*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ExtensionGetsByDomainID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a extensions.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get extensions
	exts, err := h.reqHandler.RegistrarV1ExtensionGetsByDomainID(ctx, domainID, token, size)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extensions info. err: %v", err)
	}

	res := []*rmextension.WebhookMessage{}
	for _, ext := range exts {
		tmp := ext.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ExtensionGetsByCustomerID gets the list of extensions of the given customer id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, size uint64, token string) ([]*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ExtensionGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a extensions.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get extensions
	exts, err := h.reqHandler.RegistrarV1ExtensionGetsByCustomerID(ctx, customerID, token, size)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extensions info. err: %v", err)
	}

	res := []*rmextension.WebhookMessage{}
	for _, ext := range exts {
		tmp := ext.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ExtesnionUpdate updates the extension info.
// It returns updated extension if it succeed.
func (h *serviceHandler) ExtensionUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ExtensionUpdate",
		"customer_id":  u.ID,
		"extension_id": id,
	})
	log.Debug("Updating an extension.")

	_, err := h.extensionGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionUpdate(ctx, id, name, detail, password)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

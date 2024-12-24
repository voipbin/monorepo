package servicehandler

import (
	"context"
	"fmt"

	rmextension "monorepo/bin-registrar-manager/models/extension"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// extensionGet validates the extension's ownership and returns the extension info.
func (h *serviceHandler) extensionGet(ctx context.Context, id uuid.UUID) (*rmextension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "extensionGet",
		"extension_id": id,
	})

	// send request
	res, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get an tag. err: %v", err)
		return nil, err
	}
	log.WithField("tag", res).Debug("Received result.")

	// create result
	return res, nil
}

// ExtensionCreate is a service handler for flow creation.
func (h *serviceHandler) ExtensionCreate(ctx context.Context, a *amagent.Agent, ext string, password string, name string, detail string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ExtensionCreate",
		"customer_id": a.CustomerID,
		"extension":   ext,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new extension.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionCreate(ctx, a.CustomerID, ext, password, name, detail)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionDelete deletes the extension of the given id.
func (h *serviceHandler) ExtensionDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"extension_id": id,
	})
	log.Debug("Deleting a extension.")

	e, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
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
func (h *serviceHandler) ExtensionGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  a.CustomerID,
		"username":     a.Username,
		"extension_id": id,
	})
	log.Debug("Getting a extension.")

	tmp, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionGetsByCustomerID gets the list of extensions of the given customer id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ExtensionGetsByCustomerID",
		"agent": a,
		"size":  size,
		"token": token,
	})
	log.Debug("Getting a extensions.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	// get extensions
	exts, err := h.reqHandler.RegistrarV1ExtensionGets(ctx, token, size, filters)
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
func (h *serviceHandler) ExtensionUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ExtensionUpdate",
		"customer_id":  a.CustomerID,
		"extension_id": id,
	})
	log.Debug("Updating an extension.")

	e, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionUpdate(ctx, id, name, detail, password)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

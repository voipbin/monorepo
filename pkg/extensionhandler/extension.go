package extensionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
)

// Create creates a new extension
// in this func, it creates all releated asterisk resource as well.
func (h *extensionHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	domainID uuid.UUID,
	ext string,
	password string,
) (*extension.Extension, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
			"domain_id":   domainID,
			"extension":   ext,
			"password":    len(password),
		},
	)

	// get domain
	d, err := h.dbBin.DomainGet(ctx, domainID)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	// create aor id
	aorID := fmt.Sprintf("%s@%s.%s", ext, d.DomainName, common.BaseDomainName)

	// create aor
	maxContacts := 1
	removeExisting := "yes"
	aor := &astaor.AstAOR{
		ID:             &aorID,
		MaxContacts:    &maxContacts,
		RemoveExisting: &removeExisting,
	}
	if errCreate := h.dbAst.AstAORCreate(ctx, aor); errCreate != nil {
		log.Errorf("Could not create AOR. err: %v", errCreate)
		return nil, errCreate
	}

	// create auth
	authType := "userpass"
	realm := fmt.Sprintf("%s.%s", d.DomainName, common.BaseDomainName)
	auth := &astauth.AstAuth{
		ID:       &aorID,
		AuthType: &authType,
		Username: &ext,
		Password: &password,
		// Realm:    &d.DomainName,
		Realm: &realm,
	}
	if errCreate := h.dbAst.AstAuthCreate(ctx, auth); errCreate != nil {
		log.Errorf("Could not create Auth. Err: %v", errCreate)
		return nil, errCreate
	}

	// create endpoint
	endpoint := &astendpoint.AstEndpoint{
		ID:   &aorID,
		AORs: &aorID,
		Auth: &aorID,
	}
	if errCreate := h.dbAst.AstEndpointCreate(ctx, endpoint); errCreate != nil {
		log.Errorf("Could not create Endpoint. err: %v", errCreate)
		return nil, errCreate
	}

	// create a new extension
	id := h.utilHandler.CreateUUID()
	e := &extension.Extension{
		ID:         id,
		CustomerID: customerID,

		Name:     name,
		Detail:   detail,
		DomainID: domainID,

		EndpointID: *endpoint.ID,
		AORID:      *aor.ID,
		AuthID:     *auth.ID,

		Extension: ext,
		Password:  password,
	}
	if errCreate := h.dbBin.ExtensionCreate(ctx, e); errCreate != nil {
		log.Errorf("Could not create extension. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.dbBin.ExtensionGet(ctx, e.ID)
	if err != nil {
		log.Errorf("Could not get created extension. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionCreated, res)
	promExtensionCreateTotal.Inc()

	return res, nil
}

// Get gets a exists extension
func (h *extensionHandler) Get(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	return h.dbBin.ExtensionGet(ctx, id)
}

// GetByEndpoint gets a exists extension of the given endpoint
func (h *extensionHandler) GetByEndpoint(ctx context.Context, endpoint string) (*extension.Extension, error) {
	endpointID := fmt.Sprintf("%s.%s", endpoint, common.BaseDomainName)

	return h.dbBin.ExtensionGetByEndpointID(ctx, endpointID)
}

// Update updates a exists extension
func (h *extensionHandler) Update(ctx context.Context, e *extension.Extension) (*extension.Extension, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": e.CustomerID,
			"domain_id":   e.DomainID,
			"extension":   e.Extension,
			"password":    e.Password,
		},
	)

	// create a update extension
	ext, err := h.dbBin.ExtensionGet(ctx, e.ID)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, err
	}
	ext.Name = e.Name
	ext.Detail = e.Detail
	ext.Password = e.Password

	// update ast_auth
	auth := &astauth.AstAuth{
		ID:       &ext.AuthID,
		Password: &ext.Password,
	}
	if err := h.dbAst.AstAuthUpdate(ctx, auth); err != nil {
		log.Errorf("Could not update the ast_auth info. err: %v", err)
		return nil, err
	}

	// update extension
	if err := h.dbBin.ExtensionUpdate(ctx, ext); err != nil {
		log.Errorf("Could not update the extension info. err: %v", err)
		return nil, err
	}

	// get updated extension
	res, err := h.dbBin.ExtensionGet(ctx, ext.ID)
	if err != nil {
		log.Errorf("Could not get updated extension info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionUpdated, res)

	return res, nil
}

// Delete deletes a exists extension
func (h *extensionHandler) Delete(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	// get extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get delete extension info. err: %v", err)
		return nil, err
	}

	// delete extension
	if err := h.dbBin.ExtensionDelete(ctx, ext.ID); err != nil {
		log.Errorf("Could not delete extension. err: %v", err)
		return nil, err
	}

	// delete endpopint
	if err := h.dbAst.AstEndpointDelete(ctx, ext.EndpointID); err != nil {
		log.Errorf("Could not delete endpoint. err: %v", err)
		return nil, err
	}

	// delete auth
	if err := h.dbAst.AstAuthDelete(ctx, ext.AuthID); err != nil {
		log.Errorf("Could not delete auth info. err: %v", err)
		return nil, err
	}

	// delete aor
	if err := h.dbAst.AstAORDelete(ctx, ext.AORID); err != nil {
		log.Errorf("Could not delete aor info. err: %v", err)
		return nil, err
	}
	logrus.Debugf("Deleted extension. extension: %s", id)

	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted extension info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionDeleted, res)
	promExtensionDeleteTotal.Inc()

	return res, nil
}

// DeleteByDomainID deletes a exists extension
func (h *extensionHandler) DeleteByDomainID(ctx context.Context, domainID uuid.UUID) ([]*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "DeleteByDomainID",
		"domain_id": domainID,
	})

	// get extensions
	exts, err := h.GetsByDomainID(ctx, domainID, h.utilHandler.GetCurTime(), 1000)
	if err != nil {
		log.Errorf("Could not get delete extensions. err: %v", err)
		return nil, err
	}

	// delete extensions
	res := []*extension.Extension{}
	for _, ext := range exts {
		tmp, err := h.Delete(ctx, ext.ID)
		if err != nil {
			log.Errorf("Could not delete the extension. extension_id: %s, err: %v", tmp.ID, err)
		}
		res = append(res, tmp)
	}

	return res, nil
}

// GetsByDomainID returns list of extensions
func (h *extensionHandler) GetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error) {

	exts, err := h.dbBin.ExtensionGetsByDomainID(ctx, domainID, token, limit)
	if err != nil {
		logrus.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	return exts, nil
}

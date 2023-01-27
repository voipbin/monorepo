package extensionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
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
	aorID := fmt.Sprintf("%s@%s.%s", ext, d.DomainName, constBaseDomainName)

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
	realm := fmt.Sprintf("%s.%s", d.DomainName, constBaseDomainName)
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

// ExtensionGet gets a exists extension
func (h *extensionHandler) ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	return h.dbBin.ExtensionGet(ctx, id)
}

// ExtensionUpdate updates a exists extension
func (h *extensionHandler) ExtensionUpdate(ctx context.Context, e *extension.Extension) (*extension.Extension, error) {
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

// ExtensionDelete deletes a exists extension
func (h *extensionHandler) ExtensionDelete(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	// get extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get delete extension info. err: %v", err)
		return nil, err
	}

	// delete extension
	if err := h.dbBin.ExtensionDelete(ctx, ext.ID); err != nil {
		logrus.Errorf("Could not delete extension. err: %v", err)
		return nil, err
	}

	// delete endpopint
	if err := h.dbAst.AstEndpointDelete(ctx, ext.EndpointID); err != nil {
		logrus.Errorf("Could not delete endpoint. err: %v", err)
		return nil, err
	}

	// delete auth
	if err := h.dbAst.AstAuthDelete(ctx, ext.AuthID); err != nil {
		logrus.Errorf("Could not delete auth info. err: %v", err)
		return nil, err
	}

	// delete aor
	if err := h.dbAst.AstAORDelete(ctx, ext.AORID); err != nil {
		logrus.Errorf("Could not delete aor info. err: %v", err)
		return nil, err
	}
	logrus.Debugf("Deleted extension. extension: %s", id)

	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionDeleted, res)
	promExtensionDeleteTotal.Inc()

	return res, nil
}

// ExtensionDelete deletes a exists extension
func (h *extensionHandler) ExtensionDeleteByDomainID(ctx context.Context, domainID uuid.UUID) ([]*extension.Extension, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "ExtensionDeleteByDomainID",
			"domain_id": domainID,
		},
	)

	// get extensions
	exts, err := h.ExtensionGetsByDomainID(ctx, domainID, h.utilHandler.GetCurTime(), 1000)
	if err != nil {
		logrus.Errorf("Could not get delete extensions")
		return nil, err
	}

	// delete extensions
	res := []*extension.Extension{}
	for _, ext := range exts {
		tmp, err := h.ExtensionDelete(ctx, ext.ID)
		if err != nil {
			log.Errorf("Could not delete the extension. extension_id: %s, err: %v", tmp.ID, err)
		}
		res = append(res, tmp)
	}

	return res, nil
}

// ExtensionGetsByDomainID returns list of extensions
func (h *extensionHandler) ExtensionGetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error) {

	exts, err := h.dbBin.ExtensionGetsByDomainID(ctx, domainID, token, limit)
	if err != nil {
		logrus.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	return exts, nil
}

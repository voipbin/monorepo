package extensionhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
)

// Create creates a new extension
// in this func, it creates all releated asterisk resource as well.
func (h *extensionHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	ext string,
	password string,
) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"extension":   ext,
		"password":    password,
	})

	// create realm
	realm := common.GenerateRealmExtension(customerID)

	// create aor id
	aorID := common.GenerateEndpointExtension(customerID, ext)

	// create aor
	maxContacts := defaultMaxContacts
	removeExisting := defaultRemoveExisting
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
	authType := defaultAuthType
	auth := &astauth.AstAuth{
		ID:       &aorID,
		AuthType: &authType,
		Username: &ext,
		Password: &password,
		Realm:    &realm,
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
	id := h.utilHandler.UUIDCreate()
	e := &extension.Extension{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		EndpointID: *endpoint.ID,
		AORID:      *aor.ID,
		AuthID:     *auth.ID,

		Extension:  ext,
		DomainName: customerID.String(),

		Realm:    realm,
		Username: ext,
		Password: password,
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

	// create sipauth
	sip := res.GenerateSIPAuth()
	if err := h.dbBin.SIPAuthCreate(ctx, sip); err != nil {
		log.Errorf("Could not create sip auth. err: %v", err)
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
func (h *extensionHandler) GetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {
	return h.dbBin.ExtensionGetByExtension(ctx, customerID, ext)
}

// Gets returns list of extensions
func (h *extensionHandler) Gets(ctx context.Context, token string, limit uint64, filters map[string]string) ([]*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.dbBin.ExtensionGets(ctx, limit, token, filters)
	if err != nil {
		log.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates a exists extension
func (h *extensionHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, password string) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"extension_id": id,
		"name":         name,
		"detail":       detail,
		"password":     password,
	})

	// create a update extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, err
	}

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
	if err := h.dbBin.ExtensionUpdate(ctx, id, name, detail, password); err != nil {
		log.Errorf("Could not update the extension info. err: %v", err)
		return nil, err
	}

	// get updated extension
	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated extension info. err: %v", err)
		return nil, err
	}

	// update sipauth
	sip := res.GenerateSIPAuth()
	if err := h.dbBin.SIPAuthUpdateAll(ctx, sip); err != nil {
		log.Errorf("Could not update sip auth. err: %v", err)
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

	// delete sipauth
	if err := h.dbBin.SIPAuthDelete(ctx, res.ID); err != nil {
		log.Errorf("Could not delete sip auth. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionDeleted, res)
	promExtensionDeleteTotal.Inc()

	return res, nil
}

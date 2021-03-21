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

// ExtensionCreate creates a new extension
// in this func, it creates all releated asterisk resource as well.
func (h *extensionHandler) ExtensionCreate(ctx context.Context, e *extension.Extension) (*extension.Extension, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"user_id":   e.UserID,
			"domain_id": e.DomainID,
			"extension": e.Extension,
			"password":  e.Password,
		},
	)

	// get domain
	d, err := h.dbBin.DomainGet(ctx, e.DomainID)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	// create id
	commonID := fmt.Sprintf("%s@%s", e.Extension, d.DomainName)

	// create aor
	maxContacts := 1
	removeExisting := "yes"
	aor := &astaor.AstAOR{
		ID:             &commonID,
		MaxContacts:    &maxContacts,
		RemoveExisting: &removeExisting,
	}
	if err := h.dbAst.AstAORCreate(ctx, aor); err != nil {
		log.Errorf("Could not create AOR. err: %v", err)
		return nil, err
	}

	// create auth
	authType := "userpass"
	auth := &astauth.AstAuth{
		ID:       &commonID,
		AuthType: &authType,
		Username: &e.Extension,
		Password: &e.Password,
		Realm:    &d.DomainName,
	}
	if err := h.dbAst.AstAuthCreate(ctx, auth); err != nil {
		log.Errorf("Could not create Auth. Err: %v", err)
		return nil, err
	}

	// create endpoint
	endpoint := &astendpoint.AstEndpoint{
		ID:   &commonID,
		AORs: &commonID,
		Auth: &commonID,
	}
	if err := h.dbAst.AstEndpointCreate(ctx, endpoint); err != nil {
		log.Errorf("Could not create Endpoint. err: %v", err)
		return nil, err
	}

	// create extension
	ext := &extension.Extension{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: e.UserID,

		Name:     e.Name,
		Detail:   e.Detail,
		DomainID: e.DomainID,

		EndpointID: *endpoint.ID,
		AORID:      *aor.ID,
		AuthID:     *auth.ID,

		Extension: e.Extension,
		Password:  e.Password,
	}
	if err := h.dbBin.ExtensionCreate(ctx, ext); err != nil {
		log.Errorf("Could not create extension. err: %v", err)
		return nil, err
	}

	res, err := h.dbBin.ExtensionGet(ctx, ext.ID)
	if err != nil {
		log.Errorf("Could not get created extension. err: %v", err)
		return nil, err
	}

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
			"user_id":   e.UserID,
			"domain_id": e.DomainID,
			"extension": e.Extension,
			"password":  e.Password,
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

	return res, nil
}

// ExtensionDelete deletes a exists extension
func (h *extensionHandler) ExtensionDelete(ctx context.Context, id uuid.UUID) error {

	// get extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get delete extension info. err: %v", err)
		return err
	}

	// delete extension
	if err := h.dbBin.ExtensionDelete(ctx, ext.ID); err != nil {
		logrus.Errorf("Could not delete extension. err: %v", err)
		return err
	}

	// delete endpopint
	if err := h.dbAst.AstEndpointDelete(ctx, ext.EndpointID); err != nil {
		logrus.Errorf("Could not delete endpoint. err: %v", err)
		return err
	}

	// delete auth
	if err := h.dbAst.AstAuthDelete(ctx, ext.AuthID); err != nil {
		logrus.Errorf("Could not delete auth info. err: %v", err)
		return err
	}

	// delete aor
	if err := h.dbAst.AstAORDelete(ctx, ext.AORID); err != nil {
		logrus.Errorf("Could not delete aor info. err: %v", err)
		return err
	}
	logrus.Debugf("Deleted extension. extension: %s", id)

	return nil
}

// ExtensionDelete deletes a exists extension
func (h *extensionHandler) ExtensionDeleteByDomainID(ctx context.Context, domainID uuid.UUID) error {
	// get extensions
	exts, err := h.ExtensionGetsByDomainID(ctx, domainID, getCurTime(), 1000)
	if err != nil {
		logrus.Errorf("Could not get delete extensions")
		return err
	}

	// delete extensions
	for _, ext := range exts {
		h.ExtensionDelete(ctx, ext.ID)
	}

	return nil
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

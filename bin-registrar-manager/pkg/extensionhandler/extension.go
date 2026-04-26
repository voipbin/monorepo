package extensionhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	bmaccount "monorepo/bin-billing-manager/models/account"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
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

	// check resource limit
	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeExtension)
	if err != nil {
		log.Errorf("Could not validate resource limit. err: %v", err)
		return nil, fmt.Errorf("could not validate resource limit: %w", err)
	}
	if !valid {
		log.Infof("Resource limit exceeded for customer. customer_id: %s", customerID)
		return nil, fmt.Errorf("resource limit exceeded")
	}

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

	// generate extension UUID first
	id := h.utilHandler.UUIDCreate()

	// create direct hash via direct-manager
	d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, id)
	if err != nil {
		log.Errorf("Could not create direct hash. err: %v", err)
		return nil, fmt.Errorf("could not create direct hash: %w", err)
	}
	log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)

	// create a new extension
	e := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

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

		DirectID:   d.ID,
		DirectHash: d.Hash,
	}
	if errCreate := h.dbBin.ExtensionCreate(ctx, e); errCreate != nil {
		log.Errorf("Could not create extension. err: %v", errCreate)

		// cleanup orphaned direct
		if _, errDelete := h.reqHandler.DirectV1DirectDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not cleanup orphaned direct. direct_id: %s, err: %v", d.ID, errDelete)
		}

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
	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameRegistrarManager,
				"EXTENSION_NOT_FOUND",
				"The extension was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	return res, nil
}

// GetByExtension gets a exists extension of the given endpoint
func (h *extensionHandler) GetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {
	res, err := h.dbBin.ExtensionGetByExtension(ctx, customerID, ext)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameRegistrarManager,
				"EXTENSION_NOT_FOUND",
				"The extension was not found.",
			).Wrap(err)
		}
		return nil, err
	}
	return res, nil
}

// List returns list of extensions
func (h *extensionHandler) List(ctx context.Context, token string, limit uint64, filters map[extension.Field]any) ([]*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"filters": filters,
	})

	res, err := h.dbBin.ExtensionList(ctx, limit, token, filters)
	if err != nil {
		log.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates a exists extension
func (h *extensionHandler) Update(ctx context.Context, id uuid.UUID, fields map[extension.Field]any) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"extension_id": id,
		"fields":       fields,
	})

	// create a update extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, err
	}

	// update ast_auth if password is being updated
	if password, ok := fields[extension.FieldPassword]; ok {
		passwordStr := password.(string)
		auth := &astauth.AstAuth{
			ID:       &ext.AuthID,
			Password: &passwordStr,
		}
		if err := h.dbAst.AstAuthUpdate(ctx, auth); err != nil {
			log.Errorf("Could not update the ast_auth info. err: %v", err)
			return nil, err
		}
	}

	// update extension
	if err := h.dbBin.ExtensionUpdate(ctx, id, fields); err != nil {
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
	sipFields := map[sipauth.Field]any{
		sipauth.FieldAuthTypes:  sip.AuthTypes,
		sipauth.FieldRealm:     sip.Realm,
		sipauth.FieldUsername:   sip.Username,
		sipauth.FieldPassword:  sip.Password,
		sipauth.FieldAllowedIPs: sip.AllowedIPs,
	}
	if err := h.dbBin.SIPAuthUpdate(ctx, sip.ID, sipFields); err != nil {
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

	// delete direct hash via direct-manager (best-effort, don't block extension deletion)
	if ext.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, ext.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", ext.DirectID, errDirect)
		}
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

// DirectHashRegenerate regenerates (or creates) the direct hash for the given extension.
func (h *extensionHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "DirectHashRegenerate",
		"extension_id": id,
	})

	// get current extension
	ext, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension. err: %v", err)
		return nil, fmt.Errorf("could not get extension: %w", err)
	}
	log.WithField("extension", ext).Debugf("Retrieved extension info. extension_id: %s", ext.ID)

	// regenerate or create direct
	var d *dmdirect.Direct
	if ext.DirectID != uuid.Nil {
		d, err = h.reqHandler.DirectV1DirectRegenerate(ctx, ext.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
	} else {
		d, err = h.reqHandler.DirectV1DirectCreate(ctx, ext.CustomerID, dmdirect.ResourceTypeExtension, id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
	}
	log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)

	// update extension with new direct info
	fields := map[extension.Field]any{
		extension.FieldDirectID:   d.ID,
		extension.FieldDirectHash: d.Hash,
	}
	if err := h.dbBin.ExtensionUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update extension direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update extension: %w", err)
	}

	// return updated extension
	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated extension. err: %v", err)
		return nil, err
	}

	return res, nil
}

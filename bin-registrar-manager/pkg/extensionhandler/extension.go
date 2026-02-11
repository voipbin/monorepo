package extensionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/models/sipauth"
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

	// create a new extension
	id := h.utilHandler.UUIDCreate()
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
	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		return nil, err
	}

	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, res.ID)
	if err == nil && direct != nil {
		res.DirectHash = direct.Hash
	}

	return res, nil
}

// GetByExtension gets a exists extension of the given endpoint
func (h *extensionHandler) GetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {
	return h.dbBin.ExtensionGetByExtension(ctx, customerID, ext)
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

	// batch fetch direct records
	extIDs := make([]uuid.UUID, len(res))
	for i, ext := range res {
		extIDs[i] = ext.ID
	}

	directs, _ := h.extensionDirectHandler.GetByExtensionIDs(ctx, extIDs)
	directMap := make(map[uuid.UUID]string)
	for _, d := range directs {
		directMap[d.ExtensionID] = d.Hash
	}

	for _, ext := range res {
		if hash, ok := directMap[ext.ID]; ok {
			ext.DirectHash = hash
		}
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
		sipauth.FieldRealm:      sip.Realm,
		sipauth.FieldUsername:   sip.Username,
		sipauth.FieldPassword:   sip.Password,
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

	// delete extension direct if exists
	direct, errDirect := h.extensionDirectHandler.GetByExtensionID(ctx, id)
	if errDirect == nil && direct != nil {
		if _, errDelete := h.extensionDirectHandler.Delete(ctx, direct.ID); errDelete != nil {
			log.Errorf("Could not delete extension direct. err: %v", errDelete)
		}
	}

	h.notifyHandler.PublishEvent(ctx, extension.EventTypeExtensionDeleted, res)
	promExtensionDeleteTotal.Inc()

	return res, nil
}

// DirectEnable enables direct extension access
func (h *extensionHandler) DirectEnable(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	ext, err := h.dbBin.ExtensionGet(ctx, extensionID)
	if err != nil {
		return nil, fmt.Errorf("could not get extension: %w", err)
	}

	return h.extensionDirectHandler.Create(ctx, ext.CustomerID, ext.ID)
}

// DirectDisable disables direct extension access
func (h *extensionHandler) DirectDisable(ctx context.Context, extensionID uuid.UUID) error {
	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, extensionID)
	if err != nil {
		return nil // already disabled, no-op
	}

	_, err = h.extensionDirectHandler.Delete(ctx, direct.ID)
	return err
}

// DirectRegenerate regenerates the direct extension hash
func (h *extensionHandler) DirectRegenerate(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, extensionID)
	if err != nil {
		return nil, fmt.Errorf("direct extension not enabled: %w", err)
	}

	return h.extensionDirectHandler.Regenerate(ctx, direct.ID)
}

// GetDirectByHash returns extension direct by hash
func (h *extensionHandler) GetDirectByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	return h.extensionDirectHandler.GetByHash(ctx, hash)
}

// GetByDirectHash returns the extension corresponding to the given direct hash.
// It resolves hash → ExtensionDirect → Extension and populates DirectHash.
func (h *extensionHandler) GetByDirectHash(ctx context.Context, hash string) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetByDirectHash",
		"hash": hash,
	})

	direct, err := h.extensionDirectHandler.GetByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get extension direct by hash. err: %v", err)
		return nil, err
	}
	log.WithField("extension_direct", direct).Debugf("Retrieved extension direct. extension_id: %s", direct.ExtensionID)

	res, err := h.dbBin.ExtensionGet(ctx, direct.ExtensionID)
	if err != nil {
		log.Errorf("Could not get extension. extension_id: %s, err: %v", direct.ExtensionID, err)
		return nil, err
	}
	log.WithField("extension", res).Debugf("Retrieved extension info. extension_id: %s", res.ID)

	res.DirectHash = direct.Hash

	return res, nil
}

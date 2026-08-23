package servicehandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// ProvisioningTokenTTL is the lifetime of an extension provisioning token.
	ProvisioningTokenTTL = 10 * time.Minute

	// provisioningTokenLen is the number of random bytes used for a
	// provisioning token (hex-encoded to twice this length).
	provisioningTokenLen = 32
)

// ExtensionProvisioningToken is the response for an extension provisioning
// token creation. The URL is the complete public provisioning URL for the
// softphone (Linphone) to fetch its configuration from.
type ExtensionProvisioningToken struct {
	Token  string `json:"token"`
	URL    string `json:"url"`
	Expire string `json:"expire"`
}

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
func (h *serviceHandler) ExtensionCreate(ctx context.Context, a *auth.AuthIdentity, ext string, password string, name string, detail string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ExtensionCreate",
		"customer_id": a.CustomerID,
		"extension":   ext,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new extension.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) ExtensionDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  a.CustomerID,
		"username":     a.DisplayName(),
		"extension_id": id,
	})
	log.Debug("Deleting a extension.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	e, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) ExtensionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  a.CustomerID,
		"username":     a.DisplayName(),
		"extension_id": id,
	})
	log.Debug("Getting a extension.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionGetsByCustomerID gets the list of extensions of the given customer id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ExtensionGetsByCustomerID",
		"agent": a,
		"size":  size,
		"token": token,
	})
	log.Debug("Getting a extensions.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	// get extensions
	// Convert string filters to typed filters
	typedFilters, err := h.convertExtensionFilters(filters)
	if err != nil {
		return nil, err
	}

	exts, err := h.reqHandler.RegistrarV1ExtensionList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extensions info", err)
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
func (h *serviceHandler) ExtensionUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ExtensionUpdate",
		"customer_id":  a.CustomerID,
		"extension_id": id,
	})
	log.Debug("Updating an extension.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	e, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionUpdate(ctx, id, name, detail, password)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionDirectHashRegenerate regenerates the direct hash for the extension.
func (h *serviceHandler) ExtensionDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ExtensionDirectHashRegenerate",
		"customer_id":  a.CustomerID,
		"extension_id": extensionID,
	})
	log.Debug("Regenerating extension direct hash.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	e, err := h.extensionGet(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.RegistrarV1ExtensionDirectHashRegenerate(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not regenerate extension direct hash. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ExtensionProvisioningTokenCreate creates a short-lived provisioning token
// for the extension and returns the complete public provisioning URL for the
// softphone (Linphone) QR code.
func (h *serviceHandler) ExtensionProvisioningTokenCreate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ExtensionProvisioningToken, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "ExtensionProvisioningTokenCreate",
		"customer_id":  a.CustomerID,
		"extension_id": id,
	})
	log.Debug("Creating extension provisioning token.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	e, err := h.extensionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if !h.hasPermission(ctx, a, e.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tokenBytes := make([]byte, provisioningTokenLen)
	if _, errRead := rand.Read(tokenBytes); errRead != nil {
		log.Errorf("Could not generate provisioning token. err: %v", errRead)
		return nil, fmt.Errorf("could not generate provisioning token: %w", errRead)
	}
	token := hex.EncodeToString(tokenBytes)

	if errSet := h.cacheHandler.ProvisioningTokenSet(ctx, token, id, ProvisioningTokenTTL); errSet != nil {
		log.Errorf("Could not store provisioning token. err: %v", errSet)
		return nil, fmt.Errorf("could not store provisioning token: %w", errSet)
	}

	res := &ExtensionProvisioningToken{
		Token:  token,
		URL:    h.publicBaseURL + "/provisioning/extension?token=" + token,
		Expire: time.Now().UTC().Add(ProvisioningTokenTTL).Format(time.RFC3339),
	}
	return res, nil
}

// ExtensionProvisioningXMLGet resolves a provisioning token and returns the
// Linphone remote-provisioning (lpconfig) XML for the associated extension.
// It is served on the unauthenticated public provisioning route; the token is
// the sole credential.
func (h *serviceHandler) ExtensionProvisioningXMLGet(ctx context.Context, token string) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "ExtensionProvisioningXMLGet",
	})

	extensionID, err := h.cacheHandler.ProvisioningTokenGet(ctx, token)
	if err != nil {
		// missing or expired token. Do not log the token itself.
		log.Infof("Could not resolve provisioning token. err: %v", err)
		return nil, fmt.Errorf("could not resolve provisioning token: %w", err)
	}
	log = log.WithField("extension_id", extensionID)

	e, err := h.extensionGet(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, fmt.Errorf("%w: could not find extension info", err)
	}

	if e.TMDelete != nil {
		log.Info("The extension has been deleted.")
		return nil, serviceerrors.ErrNotFound
	}

	// SERVING RULE: never read DomainName from the internal model directly --
	// legacy rows carry a customer UUID there. ConvertWebhookMessage applies
	// the Realm-first rule.
	wm := e.ConvertWebhookMessage()

	res, err := renderExtensionProvisioningXML(wm)
	if err != nil {
		log.Errorf("Could not render provisioning XML. err: %v", err)
		return nil, fmt.Errorf("could not render provisioning xml: %w", err)
	}

	return res, nil
}

// convertExtensionFilters converts map[string]string to map[rmextension.Field]any
func (h *serviceHandler) convertExtensionFilters(filters map[string]string) (map[rmextension.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, rmextension.Extension{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[rmextension.Field]any, len(typed))
	for k, v := range typed {
		result[rmextension.Field(k)] = v
	}

	return result, nil
}

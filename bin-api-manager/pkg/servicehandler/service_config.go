package servicehandler

import (
	"context"
	"errors"
	"math"
	"regexp"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// serviceConfigKeyPattern is the shape of a config key. It matches the route of the managers, and
// keeps a request value from altering the rpc uri.
const serviceConfigKeyPattern = `^[a-z0-9][a-z0-9._]{0,127}$`

var regServiceConfigKey = regexp.MustCompile(serviceConfigKeyPattern)

// serviceConfigServices are the services that have a config API. The names are the values of the
// service path parameter, and the same names are known by requesthandler. Adding a service is a
// line here and a line in the queue map of requesthandler.
var serviceConfigServices = map[string]bool{
	"email":   true,
	"ai":      true,
	"message": true,
}

// serviceConfigAuthorize checks that the identity may manage the configs: direct access is
// refused, and only the project super admin is allowed.
func (h *serviceHandler) serviceConfigAuthorize(ctx context.Context, a *auth.AuthIdentity) error {
	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return serviceerrors.ErrPermissionDenied
	}

	return nil
}

// serviceConfigAuthorizeService does the serviceConfigAuthorize and validates the service.
func (h *serviceHandler) serviceConfigAuthorizeService(ctx context.Context, a *auth.AuthIdentity, service string) error {
	if err := h.serviceConfigAuthorize(ctx, a); err != nil {
		return err
	}

	if !serviceConfigServices[service] {
		return serviceerrors.ErrNotFound
	}

	return nil
}

// serviceConfigAuthorizeKey does the serviceConfigAuthorizeService and validates the key.
func (h *serviceHandler) serviceConfigAuthorizeKey(ctx context.Context, a *auth.AuthIdentity, service, key string) error {
	if err := h.serviceConfigAuthorizeService(ctx, a, service); err != nil {
		return err
	}

	if !regServiceConfigKey.MatchString(key) {
		return serviceerrors.ErrInvalidArgument
	}

	return nil
}

// serviceConfigRevision checks the revision against the range that the managers store.
func serviceConfigRevision(revision int64) (uint32, error) {
	if revision < 0 || revision > math.MaxUint32 {
		return 0, serviceerrors.ErrInvalidArgument
	}

	return uint32(revision), nil //nolint:gosec // the range is checked above
}

// ServiceConfigGet returns the config keys of the given service.
// Only the project super admin is allowed.
func (h *serviceHandler) ServiceConfigGet(ctx context.Context, a *auth.AuthIdentity, service string) (*configrpc.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceConfigGet",
		"username": a.DisplayName(),
		"service":  service,
	})

	if err := h.serviceConfigAuthorizeService(ctx, a, service); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	res, err := h.reqHandler.ConfigV1ServiceGet(ctx, service)
	if err != nil {
		log.Errorf("Could not get the service configs. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceConfigKeySecretSet stores the secret of the given key, if its stored revision is the given
// one. The secret is passed on in the request only. Nothing of it is logged.
// Only the project super admin is allowed.
func (h *serviceHandler) ServiceConfigKeySecretSet(ctx context.Context, a *auth.AuthIdentity, service, key, secret string, revision int64) (*configrpc.Key, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceConfigKeySecretSet",
		"username": a.DisplayName(),
		"service":  service,
		"key":      key,
	})

	if err := h.serviceConfigAuthorizeKey(ctx, a, service, key); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	rev, err := serviceConfigRevision(revision)
	if err != nil {
		log.Info("The revision is not valid.")
		return nil, err
	}

	// A manager that does not serve the config requests (an older version during a deployment)
	// would log the body of a request that it does not know. The secret is sent only after the
	// manager has answered a request that carries no secret.
	if _, errProbe := h.reqHandler.ConfigV1ServiceGet(ctx, service); errProbe != nil {
		log.Errorf("The manager does not serve the config requests. err: %v", errProbe)
		if errors.Is(errProbe, requesthandler.ErrNotFound) {
			return nil, serviceerrors.ErrServiceUnavailable
		}
		return nil, errProbe
	}

	res, err := h.reqHandler.ConfigV1KeySecretSet(ctx, service, key, a.AgentID(), secret, rev)
	if err != nil {
		log.Errorf("Could not store the config secret. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceConfigKeyClear deletes the stored value of the given key, if its stored revision is the
// given one.
// Only the project super admin is allowed.
func (h *serviceHandler) ServiceConfigKeyClear(ctx context.Context, a *auth.AuthIdentity, service, key string, revision int64) (*configrpc.Key, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceConfigKeyClear",
		"username": a.DisplayName(),
		"service":  service,
		"key":      key,
	})

	if err := h.serviceConfigAuthorizeKey(ctx, a, service, key); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	rev, err := serviceConfigRevision(revision)
	if err != nil {
		log.Info("The revision is not valid.")
		return nil, err
	}

	res, err := h.reqHandler.ConfigV1KeyClear(ctx, service, key, a.AgentID(), rev)
	if err != nil {
		log.Errorf("Could not delete the stored config value. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceConfigKeyHistoryList returns a page of the change history of the given key.
// Only the project super admin is allowed.
func (h *serviceHandler) ServiceConfigKeyHistoryList(ctx context.Context, a *auth.AuthIdentity, service, key, pageToken string, pageSize int) (*configrpc.HistoryList, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceConfigKeyHistoryList",
		"username": a.DisplayName(),
		"service":  service,
		"key":      key,
	})

	if err := h.serviceConfigAuthorizeKey(ctx, a, service, key); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	// a negative size would be dropped on the way and answered as the default size
	if pageSize < 0 {
		log.Info("The page size is not valid.")
		return nil, serviceerrors.ErrInvalidArgument
	}

	res, err := h.reqHandler.ConfigV1KeyHistoryList(ctx, service, key, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get the config history. err: %v", err)
		return nil, err
	}

	return res, nil
}

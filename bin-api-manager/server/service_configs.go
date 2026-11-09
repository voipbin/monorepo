package server

import (
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// serviceConfigMaxBodyBytes caps the request body of the secret update. The largest secret that a
// key accepts today (512 bytes for the email keys, up to 6 bytes per byte once JSON escaped) fits in
// it with room. Check it again when a service with a longer secret limit is added.
const serviceConfigMaxBodyBytes = 4 * 1024

// serviceConfigSecretBody is the body of the secret update. The fields are pointers so that a
// missing field is told from a zero value: the revision 0 is a valid value ("no stored value").
// The generated request type cannot tell them apart.
type serviceConfigSecretBody struct {
	Secret   *string `json:"secret"`
	Revision *int64  `json:"revision"`
}

// bindServiceConfigBody reads the JSON body into req. On failure it answers the request and
// returns false. The binding error is neither logged nor attached to the context: encoding/json
// errors can quote the bytes they stopped at, and the body of the secret request is a secret.
func bindServiceConfigBody(c *gin.Context, log *logrus.Entry, req any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, serviceConfigMaxBodyBytes)
	if err := c.ShouldBindJSON(req); err != nil {
		log.Info("Could not parse the request body.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return false
	}

	return true
}

func (h *server) GetServiceConfigsService(c *gin.Context, service openapi_server.ServiceConfigService) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceConfigsService",
		"request_address": c.ClientIP(),
		"service":         service,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	res, err := h.serviceHandler.ServiceConfigGet(c.Request.Context(), a, string(service))
	if err != nil {
		log.Infof("Could not get the service configs. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) PutServiceConfigsServiceKeysKeySecret(c *gin.Context, service openapi_server.ServiceConfigService, key openapi_server.ServiceConfigKeyName) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceConfigsServiceKeysKeySecret",
		"request_address": c.ClientIP(),
		"service":         service,
		"key":             key,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	// req holds the secret: it is never logged or attached to an error
	var req serviceConfigSecretBody
	if !bindServiceConfigBody(c, log, &req) {
		return
	}
	if req.Secret == nil || req.Revision == nil {
		log.Info("The secret or the revision is missing.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	res, err := h.serviceHandler.ServiceConfigKeySecretSet(c.Request.Context(), a, string(service), key, *req.Secret, *req.Revision)
	if err != nil {
		log.Infof("Could not store the config secret. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) DeleteServiceConfigsServiceKeysKey(c *gin.Context, service openapi_server.ServiceConfigService, key openapi_server.ServiceConfigKeyName, params openapi_server.DeleteServiceConfigsServiceKeysKeyParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceConfigsServiceKeysKey",
		"request_address": c.ClientIP(),
		"service":         service,
		"key":             key,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	res, err := h.serviceHandler.ServiceConfigKeyClear(c.Request.Context(), a, string(service), key, params.Revision)
	if err != nil {
		log.Infof("Could not delete the stored config value. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) GetServiceConfigsServiceKeysKeyHistories(c *gin.Context, service openapi_server.ServiceConfigService, key openapi_server.ServiceConfigKeyName, params openapi_server.GetServiceConfigsServiceKeysKeyHistoriesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceConfigsServiceKeysKeyHistories",
		"request_address": c.ClientIP(),
		"service":         service,
		"key":             key,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	pageSize := 0
	if params.PageSize != nil {
		pageSize = *params.PageSize
	}
	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	res, err := h.serviceHandler.ServiceConfigKeyHistoryList(c.Request.Context(), a, string(service), key, pageToken, pageSize)
	if err != nil {
		log.Infof("Could not get the config history. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

package server

import (
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// numberProviderMaxBodyBytes caps the request body of the number provider updates. The largest
// valid body (a secret of 1024 bytes, or 8 config entries) is far below it.
const numberProviderMaxBodyBytes = 8 * 1024

// bindNumberProviderBody reads the JSON body into req. On failure it answers the request and
// returns false. The binding error is neither logged nor attached to the context: encoding/json
// errors can quote the bytes they stopped at, and the body of the secret request is a secret.
func bindNumberProviderBody(c *gin.Context, log *logrus.Entry, req any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, numberProviderMaxBodyBytes)
	if err := c.ShouldBindJSON(req); err != nil {
		log.Info("Could not parse the request body.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return false
	}

	return true
}

func (h *server) GetNumberProviders(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetNumberProviders",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	tmps, err := h.serviceHandler.NumberProviderList(c.Request.Context(), a)
	if err != nil {
		log.Infof("Could not get the number providers. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	// all the providers come in one response, so there is no next page
	c.JSON(http.StatusOK, GenerateListResponse(tmps, ""))
}

func (h *server) GetNumberProvidersProviderName(c *gin.Context, providerName string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetNumberProvidersProviderName",
		"request_address": c.ClientIP(),
		"provider_name":   providerName,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	res, err := h.serviceHandler.NumberProviderGet(c.Request.Context(), a, providerName)
	if err != nil {
		log.Infof("Could not get the number provider. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) PutNumberProvidersProviderName(c *gin.Context, providerName string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutNumberProvidersProviderName",
		"request_address": c.ClientIP(),
		"provider_name":   providerName,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	var req openapi_server.PutNumberProvidersProviderNameJSONRequestBody
	if !bindNumberProviderBody(c, log, &req) {
		return
	}

	var config map[string]string
	if req.Config != nil {
		config = *req.Config
	}

	res, err := h.serviceHandler.NumberProviderUpdate(c.Request.Context(), a, providerName, req.Enabled, req.Priority, config)
	if err != nil {
		log.Infof("Could not update the number provider. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) PutNumberProvidersProviderNameSecret(c *gin.Context, providerName string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutNumberProvidersProviderNameSecret",
		"request_address": c.ClientIP(),
		"provider_name":   providerName,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	// req holds the secret: it is never logged or attached to an error
	var req openapi_server.PutNumberProvidersProviderNameSecretJSONRequestBody
	if !bindNumberProviderBody(c, log, &req) {
		return
	}

	// the generated type cannot require a write-only field
	if req.Secret == nil {
		log.Info("The secret is missing.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	res, err := h.serviceHandler.NumberProviderUpdateSecret(c.Request.Context(), a, providerName, *req.Secret)
	if err != nil {
		log.Infof("Could not store the number provider secret. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) DeleteNumberProvidersProviderNameSecret(c *gin.Context, providerName string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteNumberProvidersProviderNameSecret",
		"request_address": c.ClientIP(),
		"provider_name":   providerName,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	res, err := h.serviceHandler.NumberProviderDeleteSecret(c.Request.Context(), a, providerName)
	if err != nil {
		log.Infof("Could not delete the number provider secret. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

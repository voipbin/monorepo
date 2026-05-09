package server

import (
	"errors"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	commonrequesthandler "monorepo/bin-common-handler/pkg/requesthandler"
	rmprovider "monorepo/bin-route-manager/models/provider"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetProviders(c *gin.Context, params openapi_server.GetProvidersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProviders",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.ProviderList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get providers info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostProviders(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostProviders",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostProvidersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	techHeaders := map[string]string{}
	for key, value := range req.TechHeaders {
		strValue, ok := value.(string)
		if !ok {
			log.Errorf("Invalid type for tech header value. key: %s, value: %v", key, value)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "tech_headers values must be strings."))
			return
		}
		techHeaders[key] = strValue
	}

	codecs := ""
	if req.Codecs != nil {
		codecs = *req.Codecs
	}

	res, err := h.serviceHandler.ProviderCreate(
		c.Request.Context(),
		a,
		rmprovider.Type(req.Type),
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		techHeaders,
		req.Name,
		req.Detail,
		codecs,
	)
	if err != nil {
		log.Errorf("Could not create a provider. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteProvidersId",
		"request_address": c.ClientIP,
		"provider_id":     id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ProviderDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the delete the provider info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProvidersId",
		"request_address": c.ClientIP,
		"provider_id":     id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ProviderGet(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the provider info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostProvidersSetup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostProvidersSetup",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostProvidersSetupJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}
	res, err := h.serviceHandler.ProviderSetup(
		c.Request.Context(),
		a,
		string(req.Carrier),
		req.Name,
		detail,
		req.Credentials.ApiKey,
	)
	if err != nil {
		// Typed-error path (post-migration upstream): pass the upstream
		// VoipbinError through directly so the client sees route-manager's
		// domain/reason/message verbatim. This is the design intent of the
		// typed-error migration — once an upstream emits typed errors, the
		// api-manager edge stops re-labeling them. Mis-classification risk
		// in the legacy gate (re-labeling any 422 as CARRIER_CREDENTIALS_REJECTED)
		// disappears in this branch because route-manager's reason is now
		// authoritative.
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			log.Infof("Provider setup returned typed error from upstream. err: %v", err)
			abortWithError(c, ve)
			return
		}
		// Legacy path (pre-migration upstream): canned ErrUnprocessableEntity.
		// No typed payload to forward, so we hand-roll the envelope. Removable
		// once route-manager migrates and the typed branch above covers all
		// credential-rejection cases.
		if errors.Is(err, commonrequesthandler.ErrUnprocessableEntity) {
			log.Infof("Carrier API key rejected. err: %v", err)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "CARRIER_CREDENTIALS_REJECTED", "The carrier rejected the supplied credentials.").Wrap(err))
			return
		}
		log.Errorf("Could not set up provider. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersIDPUT",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutProvidersIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	techHeaders := map[string]string{}
	for key, value := range req.TechHeaders {
		strValue, ok := value.(string)
		if !ok {
			log.Errorf("Invalid type for tech header value. key: %s, value: %v", key, value)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "tech_headers values must be strings."))
			return
		}
		techHeaders[key] = strValue
	}

	codecs := ""
	if req.Codecs != nil {
		codecs = *req.Codecs
	}

	res, err := h.serviceHandler.ProviderUpdate(
		c.Request.Context(),
		a,
		target,
		rmprovider.Type(req.Type),
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		techHeaders,
		req.Name,
		req.Detail,
		codecs,
	)
	if err != nil {
		log.Errorf("Could not update the provider. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

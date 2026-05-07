package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// GetOutboundConfigs handles GET /outbound_configs
// Lists outbound configs for the authenticated customer.
// IDOR prevention: always uses a.CustomerID from JWT, ignores params.CustomerId.
func (h *server) GetOutboundConfigs(c *gin.Context, params openapi_server.GetOutboundConfigsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutboundConfigs",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	// IDOR prevention: use JWT customer_id, never params.CustomerId
	tmps, err := h.serviceHandler.OutboundConfigList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get outbound configs. err: %v", err)
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

// PostOutboundConfigs handles POST /outbound_configs
func (h *server) PostOutboundConfigs(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutboundConfigs",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostOutboundConfigsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	updateReq := convertOutboundConfigUpdateRequest(req)

	res, err := h.serviceHandler.OutboundConfigCreate(c.Request.Context(), a, updateReq)
	if err != nil {
		log.Errorf("Could not create outbound config. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteOutboundConfigsId handles DELETE /outbound_configs/{id}
func (h *server) DeleteOutboundConfigsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteOutboundConfigsId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	// Convert openapi_types.UUID to gofrs uuid.UUID
	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.OutboundConfigDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete outbound config. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetOutboundConfigsId handles GET /outbound_configs/{id}
func (h *server) GetOutboundConfigsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutboundConfigsId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	// Convert openapi_types.UUID to gofrs uuid.UUID
	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.OutboundConfigGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get outbound config. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PutOutboundConfigsId handles PUT /outbound_configs/{id}
func (h *server) PutOutboundConfigsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutboundConfigsId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PutOutboundConfigsIdJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	updateReq := convertOutboundConfigUpdateRequest(req)

	// Convert openapi_types.UUID to gofrs uuid.UUID
	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, errUpdate := h.serviceHandler.OutboundConfigUpdate(c.Request.Context(), a, target, updateReq)
	if errUpdate != nil {
		log.Errorf("Could not update outbound config. err: %v", errUpdate)
		abortWithServiceError(c, errUpdate)
		return
	}

	c.JSON(200, res)
}

// convertOutboundConfigUpdateRequest converts the OpenAPI request body to the internal model.
func convertOutboundConfigUpdateRequest(req openapi_server.CallManagerOutboundConfigUpdateRequest) *cmoutboundconfig.UpdateRequest {
	return &cmoutboundconfig.UpdateRequest{
		Name:                 req.Name,
		Detail:               req.Detail,
		DestinationWhitelist: req.DestinationWhitelist,
		Codecs:               req.Codecs,
	}
}

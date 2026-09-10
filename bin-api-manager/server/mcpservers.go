package server

import (
	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetMcpservers(c *gin.Context, params openapi_server.GetMcpserversParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMcpservers",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

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

	tmps, err := h.serviceHandler.McpServerGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get MCP server list. err: %v", err)
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

func (h *server) PostMcpservers(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostMcpservers",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostMcpserversJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	authType := ammcpserver.AuthTypeNone
	if req.AuthType != nil {
		authType = ammcpserver.AuthType(*req.AuthType)
	}

	apiKeyHeader := ""
	if req.ApiKeyHeader != nil {
		apiKeyHeader = *req.ApiKeyHeader
	}

	secret := ""
	if req.Secret != nil {
		secret = *req.Secret
	}

	res, err := h.serviceHandler.McpServerCreate(c.Request.Context(), a, req.Name, detail, req.Url, authType, apiKeyHeader, secret)
	if err != nil {
		log.Errorf("Could not create a MCP server. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetMcpserversId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMcpserversId",
		"request_address": c.ClientIP,
		"mcp_server_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.UUID(id)

	res, err := h.serviceHandler.McpServerGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get MCP server. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutMcpserversId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutMcpserversId",
		"request_address": c.ClientIP,
		"mcp_server_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.UUID(id)

	var req openapi_server.PutMcpserversIdJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	url := ""
	if req.Url != nil {
		url = *req.Url
	}

	status := ammcpserver.Status("")
	if req.Status != nil {
		status = ammcpserver.Status(*req.Status)
	}

	authType := ammcpserver.AuthTypeNone
	if req.AuthType != nil {
		authType = ammcpserver.AuthType(*req.AuthType)
	}

	apiKeyHeader := ""
	if req.ApiKeyHeader != nil {
		apiKeyHeader = *req.ApiKeyHeader
	}

	// Secret stays *string per design §10's PUT semantics: nil means "leave
	// the existing encrypted secret untouched". Do not default it to "".
	res, err := h.serviceHandler.McpServerUpdate(c.Request.Context(), a, target, name, detail, url, status, authType, apiKeyHeader, req.Secret)
	if err != nil {
		log.Errorf("Could not update the MCP server. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteMcpserversId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteMcpserversId",
		"request_address": c.ClientIP,
		"mcp_server_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.UUID(id)

	res, err := h.serviceHandler.McpServerDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete MCP server. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

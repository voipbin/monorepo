package server

import (
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsAimessages handles GET /service_agents/aimessages
func (h *server) GetServiceAgentsAimessages(c *gin.Context, params openapi_server.GetServiceAgentsAimessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsAimessages",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
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

	aicallID := uuid.UUID(params.AicallId)
	if aicallID == uuid.Nil {
		log.Errorf("Invalid aicall id. aicall_id: %v", params.AicallId)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	tmps, err := h.serviceHandler.ServiceAgentAImessageList(c.Request.Context(), a, aicallID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get aimessages info. err: %v", err)
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

// PostServiceAgentsAimessages handles POST /service_agents/aimessages
func (h *server) PostServiceAgentsAimessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsAimessages",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostServiceAgentsAimessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	aicallID := uuid.UUID(req.AicallId)

	res, err := h.serviceHandler.ServiceAgentAImessageCreate(
		c.Request.Context(),
		a,
		aicallID,
		ammessage.Role(req.Role),
		req.Content,
	)
	if err != nil {
		log.Errorf("Could not create aimessage. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

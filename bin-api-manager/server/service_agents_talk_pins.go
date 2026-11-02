package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tkpin "monorepo/bin-talk-manager/models/pin"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetServiceAgentsTalkPins handles GET /service_agents/talk_pins
func (h *server) GetServiceAgentsTalkPins(c *gin.Context, params openapi_server.GetServiceAgentsTalkPinsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkPins",
		"request_address": c.ClientIP(),
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

	targetType := tkpin.TargetType("")
	if params.TargetType != nil {
		targetType = tkpin.TargetType(*params.TargetType)
	}

	chatID := uuid.Nil
	if params.ChatId != nil {
		chatID = uuid.FromStringOrNil(params.ChatId.String())
	}

	tmps, err := h.serviceHandler.ServiceAgentTalkPinList(c.Request.Context(), a, targetType, chatID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get the pins. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if last := tmps[len(tmps)-1]; last.TMCreate != nil {
			nextToken = last.TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	c.JSON(200, GenerateListResponse(tmps, nextToken))
}

// PostServiceAgentsTalkPins handles POST /service_agents/talk_pins
func (h *server) PostServiceAgentsTalkPins(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTalkPins",
		"request_address": c.ClientIP(),
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

	var req openapi_server.PostServiceAgentsTalkPinsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkPinCreate(c.Request.Context(), a, tkpin.TargetType(req.TargetType), uuid.FromStringOrNil(req.TargetId.String()))
	if err != nil {
		log.Errorf("Could not create the pin. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsTalkPinsId handles DELETE /service_agents/talk_pins/{id}
func (h *server) DeleteServiceAgentsTalkPinsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsTalkPinsId",
		"request_address": c.ClientIP(),
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

	res, err := h.serviceHandler.ServiceAgentTalkPinDelete(c.Request.Context(), a, uuid.FromStringOrNil(id.String()))
	if err != nil {
		log.Errorf("Could not delete the pin. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

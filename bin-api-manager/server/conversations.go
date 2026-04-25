package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetConversations(c *gin.Context, params openapi_server.GetConversationsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConversations",
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

	tmps, err := h.serviceHandler.ConversationGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a conversation list. err: %v", err)
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

func (h *server) GetConversationsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConversationsId",
		"request_address": c.ClientIP,
		"conversation_id": id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConversationGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutConversationsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutConversationsId",
		"request_address": c.ClientIP,
		"conversation_id": id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutConversationsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	raw, err := structToFilteredMap(req)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}

	fields, err := cvconversation.ConvertStringMapToFieldMap(raw)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ConversationUpdate(c.Request.Context(), a, target, fields)
	if err != nil {
		log.Errorf("Could not update the conversation. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConversationsIdMessages(c *gin.Context, id string, params openapi_server.GetConversationsIdMessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConversationsIdMessages",
		"request_address": c.ClientIP,
		"conversation_id": id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

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

	tmps, err := h.serviceHandler.ConversationMessageGetsByConversationID(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a conversation message list. err: %v", err)
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

func (h *server) PostConversationsIdMessages(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConversationsIdMessages",
		"request_address": c.ClientIP,
		"conversation_id": id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostConversationsIdMessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ConversationMessageSend(c.Request.Context(), a, target, req.Text, []cvmedia.Media{})
	if err != nil {
		log.Errorf("Could not create a customer. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

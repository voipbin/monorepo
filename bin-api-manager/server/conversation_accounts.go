package server

import (
	"encoding/json"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cvaccount "monorepo/bin-conversation-manager/models/account"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetConversationAccounts(c *gin.Context, params openapi_server.GetConversationAccountsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConversationAccounts",
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

	tmps, err := h.serviceHandler.ConversationAccountGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken)
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

func (h *server) PostConversationAccounts(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConversationAccounts",
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

	var req openapi_server.PostConversationAccountsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	messageFlowID := uuid.Nil
	if req.MessageFlowId != nil {
		messageFlowID = uuid.FromStringOrNil(*req.MessageFlowId)
	}

	var providerData json.RawMessage
	if req.ProviderData != nil {
		b, err := json.Marshal(req.ProviderData)
		if err != nil {
			log.Errorf("Could not marshal provider_data. err: %v", err)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_PROVIDER_DATA", "Could not encode provider_data."))
			return
		}
		providerData = b
	}

	res, err := h.serviceHandler.ConversationAccountCreate(
		c.Request.Context(),
		a,
		cvaccount.Type(req.Type),
		req.Name,
		req.Detail,
		req.Secret,
		req.Token,
		messageFlowID,
		providerData,
	)
	if err != nil {
		log.Errorf("Could not create a conversation account. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConversationAccountsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConversationAccountsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConversationAccountGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a conversation account. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutConversationAccountsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutConversationAccountsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutConversationAccountsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	// provider_data is a nested object — extract before the generic field conversion
	// which only handles primitive types (string, uuid, bool, int).
	var providerData json.RawMessage
	if req.ProviderData != nil {
		b, err := json.Marshal(req.ProviderData)
		if err != nil {
			log.Errorf("Could not marshal provider_data. err: %v", err)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_PROVIDER_DATA", "Could not encode provider_data."))
			return
		}
		providerData = b
	}

	raw, err := structToFilteredMap(req)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}
	// Remove provider_data from the generic map — it was already extracted above as json.RawMessage.
	delete(raw, "provider_data")

	fields, err := cvaccount.ConvertStringMapToFieldMap(raw)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}

	if providerData != nil {
		fields[cvaccount.FieldProviderData] = providerData
	}

	res, err := h.serviceHandler.ConversationAccountUpdate(c.Request.Context(), a, target, fields)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteConversationAccountsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteConversationAccountsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConversationAccountDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the conversation account. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

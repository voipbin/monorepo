package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	openapi_types "github.com/oapi-codegen/runtime/types"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	openapi_server "monorepo/bin-api-manager/gens/openapi_server"
)

func (h *server) GetServiceAgentsContactAddresses(c *gin.Context, params openapi_server.GetServiceAgentsContactAddressesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactAddresses",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	filters := map[string]any{}
	if params.ContactId != nil {
		cid := uuid.UUID(*params.ContactId)
		if cid != uuid.Nil {
			filters["contact_id"] = cid
		}
	}
	if params.Type != nil {
		filters["type"] = string(*params.Type)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}
	pageSize := uint64(20)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}

	res, err := h.serviceHandler.ServiceAgentContactAddressList(c.Request.Context(), a, filters, pageToken, pageSize)
	if !ok {
		log.Errorf("Could not list contact addresses. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostServiceAgentsContactAddresses(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsContactAddresses",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	var req openapi_server.PostServiceAgentsContactAddressesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	contactID := uuid.UUID(req.ContactId)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact_id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_CONTACT_ID", "The provided contact_id is not a valid UUID."))
		return
	}

	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	res, err := h.serviceHandler.ServiceAgentContactAddressCreateIndependent(c.Request.Context(), a, contactID, string(req.Type), req.Target, isPrimary)
	if !ok {
		log.Errorf("Could not create contact address. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(201, res)
}

func (h *server) GetServiceAgentsContactAddressesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactAddressesId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	addressID := uuid.UUID(id)
	if addressID == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ServiceAgentContactAddressGet(c.Request.Context(), a, addressID)
	if !ok {
		log.Errorf("Could not get contact address. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutServiceAgentsContactAddressesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsContactAddressesId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	addressID := uuid.UUID(id)
	if addressID == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutServiceAgentsContactAddressesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	contactIDStr := c.Query("contact_id")
	contactID := uuid.FromStringOrNil(contactIDStr)

	fields := map[string]any{}
	if req.Target != nil {
		fields["target"] = *req.Target
	}
	if req.IsPrimary != nil {
		fields["is_primary"] = *req.IsPrimary
	}

	res, err := h.serviceHandler.ServiceAgentContactAddressUpdateIndependent(c.Request.Context(), a, contactID, addressID, fields)
	if !ok {
		log.Errorf("Could not update contact address. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteServiceAgentsContactAddressesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsContactAddressesId",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	addressID := uuid.UUID(id)
	if addressID == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	contactIDStr := c.Query("contact_id")
	contactID := uuid.FromStringOrNil(contactIDStr)

	res, err := h.serviceHandler.ServiceAgentContactAddressDeleteIndependent(c.Request.Context(), a, contactID, addressID)
	if !ok {
		log.Errorf("Could not delete contact address. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

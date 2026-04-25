package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetServiceAgentsMe(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsMe",
		"request_address": c.ClientIP,
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
		"agent": a,
	})

	res, err := h.serviceHandler.ServiceAgentMeGet(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutServiceAgentsMe(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMe",
		"request_address": c.ClientIP,
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
		"agent": a,
	})

	var req openapi_server.PutServiceAgentsMeJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	res, err := h.serviceHandler.ServiceAgentMeUpdate(c.Request.Context(), a, req.Name, req.Detail, amagent.RingMethod(req.RingMethod))
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutServiceAgentsMeAddresses(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMeAddresses",
		"request_address": c.ClientIP,
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
		"agent": a,
	})

	var req openapi_server.PutServiceAgentsMeAddressesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	addresses := []commonaddress.Address{}
	for _, v := range req.Addresses {
		addresses = append(addresses, ConvertCommonAddress(v))
	}

	res, err := h.serviceHandler.ServiceAgentMeUpdateAddresses(c.Request.Context(), a, addresses)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutServiceAgentsMeStatus(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMeStatus",
		"request_address": c.ClientIP,
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
		"agent":    a,
		"username": a.DisplayName(),
	})

	var req openapi_server.PutServiceAgentsMeStatusJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	res, err := h.serviceHandler.ServiceAgentMeUpdateStatus(c.Request.Context(), a, amagent.Status(req.Status))
	if err != nil {
		log.Errorf("Could not update the agent's status. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutServiceAgentsMePassword(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMePassword",
		"request_address": c.ClientIP,
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
		"agent":    a,
		"username": a.DisplayName(),
	})

	var req openapi_server.PutServiceAgentsMePasswordJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	res, err := h.serviceHandler.ServiceAgentMeUpdatePassword(c.Request.Context(), a, req.Password)
	if err != nil {
		log.Errorf("Could not update the agent's password. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

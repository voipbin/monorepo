package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetCustomer(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCustomer",
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
	log = log.WithField("agent", a)

	res, err := h.serviceHandler.CustomerSelfGet(c.Request.Context(), a)
	if err != nil {
		log.Infof("Could not get the customer info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomer(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomer",
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
	log = log.WithField("agent", a)

	var req openapi_server.PutCustomerJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	res, err := h.serviceHandler.CustomerSelfUpdate(c.Request.Context(), a, req.Name, req.Detail, req.Email, req.PhoneNumber, req.Address, cmcustomer.WebhookMethod(req.WebhookMethod), req.WebhookUri)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomerMetadata(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomerMetadata",
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
	log = log.WithField("agent", a)

	var req openapi_server.PutCustomerMetadataJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	metadata := cmcustomer.Metadata{
		RTPDebug: req.RtpDebug != nil && *req.RtpDebug,
	}

	res, err := h.serviceHandler.CustomerSelfUpdateMetadata(c.Request.Context(), a, metadata)
	if err != nil {
		log.Errorf("Could not update the customer metadata. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomerBillingAccountId(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomerBillingAccountId",
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
	log = log.WithField("agent", a)

	var req openapi_server.PutCustomerBillingAccountIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	billingAccountID := uuid.FromStringOrNil(req.BillingAccountId)
	if billingAccountID == uuid.Nil {
		log.Error("Could not parse the billing account id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.CustomerSelfUpdateBillingAccountID(c.Request.Context(), a, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}


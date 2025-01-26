package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
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

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("agent", a)

	res, err := h.serviceHandler.CustomerGet(c.Request.Context(), &a, a.CustomerID)
	if err != nil {
		log.Infof("Could not get the customer info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomer(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomer",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	var req openapi_server.PutCustomerJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerUpdate(c.Request.Context(), &a, a.CustomerID, req.Name, req.Detail, req.Email, req.PhoneNumber, req.Address, cmcustomer.WebhookMethod(req.WebhookMethod), req.WebhookUri)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomerBillingAccountId(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomerBillingAccountId",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	var req openapi_server.PutCustomerBillingAccountIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	billingAccountID := uuid.FromStringOrNil(req.BillingAccountId)
	if billingAccountID == uuid.Nil {
		log.Error("Could not parse the billing account id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerUpdateBillingAccountID(c.Request.Context(), &a, a.CustomerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

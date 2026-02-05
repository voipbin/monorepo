package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostCustomers(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomers",
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

	var req openapi_server.PostCustomersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		req.Email,
		req.PhoneNumber,
		req.Address,
		cucustomer.WebhookMethod(req.WebhookMethod),
		req.WebhookUri,
	)
	if err != nil {
		log.Errorf("Could not create a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetCustomers(c *gin.Context, params openapi_server.GetCustomersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCustomers",
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

	filters := map[string]string{
		"deleted": "false",
	}

	tmps, err := h.serviceHandler.CustomerList(c.Request.Context(), &a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get a customers list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil { nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z") }
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetCustomersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCustomersId",
		"request_address": c.ClientIP,
		"customer_id":     id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomersId",
		"request_address": c.ClientIP,
		"customer_id":     id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCustomersIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerUpdate(c.Request.Context(), &a, target, req.Name, req.Detail, req.Email, req.PhoneNumber, req.Address, cucustomer.WebhookMethod(req.WebhookMethod), req.WebhookUri)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteCustomersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCustomersId",
		"request_address": c.ClientIP,
		"customer_id":     id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCustomersIdBillingAccountId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomersIdBillingAccountId",
		"request_address": c.ClientIP,
		"customer_id":     id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCustomersIdBillingAccountIdJSONBody
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

	res, err := h.serviceHandler.CustomerUpdateBillingAccountID(c.Request.Context(), &a, target, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

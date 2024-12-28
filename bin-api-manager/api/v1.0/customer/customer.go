package customer

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// customerGET handles GET /customer request.
// It gets the agent.
//
//	@Summary		Get the logged in agent
//	@Description	Get the logged in agent information
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the customer"
//	@Success		200
//	@Router			/v1.0/customer [get]
func customerGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customerGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerGet(c.Request.Context(), &a, a.CustomerID)
	if err != nil {
		log.Infof("Could not get the customer info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customerPut handles PUT /customer request.
// It updates a exist customer info with the given customer info.
// And returns updated customer info if it succeed.
//
//	@Summary		Update a customer.
//	@Description	Update a customer and returns detail updated customer info.
//	@Produce		json
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/customer [put]
func customerPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customerPut",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req request.BodyCustomersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdate(c.Request.Context(), &a, a.CustomerID, req.Name, req.Detail, req.Email, req.PhoneNumber, req.Address, req.WebhookMethod, req.WebhookURI)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customerBillingAccountIDPut handles PUT /customer/billing_account_id request.
// It updates a customer's billing account id.
//
//	@Summary		Update a customer's billing account id.
//	@Description	Update a customer's billing account id.
//	@Produce		json
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/customer/billing_account_id [put]
func customerBillingAccountIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customerBillingAccountIDPut",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req request.BodyCustomerBillingAccountIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdateBillingAccountID(c.Request.Context(), &a, a.CustomerID, req.BillingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

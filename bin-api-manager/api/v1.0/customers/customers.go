package customers

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// customersPost handles POST /customers request.
// It creates a new customer with the given info and returns created customer info.
//
//	@Summary		Create a new customer and returns detail created customer info.
//	@Description	Create a new customer and returns detail created customer info.
//	@Produce		json
//	@Param			customer	body		request.BodyCustomersPOST	true	"customer info."
//	@Success		200			{object}	customer.Customer
//	@Router			/v1.0/customers [post]
func customersPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersPost",
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

	var req request.BodyCustomersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Creating a customer.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		req.Email,
		req.PhoneNumber,
		req.Address,
		req.WebhookMethod,
		req.WebhookURI,
	)
	if err != nil {
		log.Errorf("Could not create a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersGet handles GET /customers request.
// It gets a list of customers with the given info.
//
//	@Summary		Gets a list of customers.
//	@Description	Gets a list of customers
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCustomersGET
//	@Router			/v1.0/customers [get]
func customersGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersGET",
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

	var req request.ParamCustomersGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("customersGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	filters := map[string]string{
		"deleted": "false",
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	customers, err := serviceHandler.CustomerGets(c.Request.Context(), &a, pageSize, req.PageToken, filters)
	if err != nil {
		log.Errorf("Could not get a customers list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(customers) > 0 {
		nextToken = customers[len(customers)-1].TMCreate
	}
	res := response.BodyCustomersGET{
		Result: customers,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// customersIDGet handles GET /customers/{id} request.
// It returns detail customer info.
//
//	@Summary		Returns detail customer info.
//	@Description	Returns detail customer info of the given customer id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the customer"
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/customers/{id} [get]
func customersIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDGET",
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

	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDPut handles PUT /customers/{id} request.
// It updates a exist customer info with the given customer info.
// And returns updated customer info if it succeed.
//
//	@Summary		Update a customer.
//	@Description	Update a customer and returns detail updated customer info.
//	@Produce		json
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/customers/{id} [put]
func customersIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDPUT",
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

	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.Email, req.PhoneNumber, req.Address, req.WebhookMethod, req.WebhookURI)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDDelete handles DELETE /customers/{id} request.
// It deletes a exist customer info.
//
//	@Summary		Delete a existing customer.
//	@Description	Delete a existing customer.
//	@Produce		json
//	@Success		200
//	@Router			/v1.0/customers/{id} [delete]
func customersIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDDelete",
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

	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDDelete.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDBillingAccountIDPut handles PUT /customers/{id}/billing_account_id request.
// It updates a customer's billing account id.
//
//	@Summary		Update a customer's billing account id.
//	@Description	Update a customer's billing account id.
//	@Produce		json
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/customers/{id}/billing_account_id [put]
func customersIDBillingAccountIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDBillingAccountIDPut",
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

	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDBillingAccountIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdateBillingAccountID(c.Request.Context(), &a, id, req.BillingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

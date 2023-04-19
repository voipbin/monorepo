package customers

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// customersPost handles POST /customers request.
// It creates a new customer with the given info and returns created customer info.
// @Summary Create a new customer and returns detail created customer info.
// @Description Create a new customer and returns detail created customer info.
// @Produce json
// @Param customer body request.BodyCustomersPOST true "customer info."
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers [post]
func customersPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersPost",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	var req request.BodyCustomersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Creating a customer.")

	// create a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerCreate(
		c.Request.Context(),
		&u,
		req.Username,
		req.Password,
		req.Name,
		req.Detail,
		req.WebhookMethod,
		req.WebhookURI,
		req.LineSecret,
		req.LineToken,
		req.PermissionIDs,
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
// @Summary Gets a list of customers.
// @Description Gets a list of customers
// @Produce json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyCustomersGET
// @Router /v1.0/customers [get]
func customersGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	var req request.ParamCustomersGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("customersGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get customers
	customers, err := serviceHandler.CustomerGets(c.Request.Context(), &u, pageSize, req.PageToken)
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
// @Summary Returns detail customer info.
// @Description Returns detail customer info of the given customer id.
// @Produce json
// @Param id path string true "The ID of the customer"
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers/{id} [get]
func customersIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerGet(c.Request.Context(), &u, id)
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
// @Summary Update a customer.
// @Description Update a customer and returns detail updated customer info.
// @Produce json
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers/{id} [put]
func customersIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDPUT",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdate(c.Request.Context(), &u, id, req.Name, req.Detail, req.WebhookMethod, req.WebhookURI)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDDelete handles DELETE /customers/{id} request.
// It deletes a exist customer info.
// @Summary Delete a existing customer.
// @Description Delete a existing customer.
// @Produce json
// @Success 200
// @Router /v1.0/customers/{id} [delete]
func customersIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDDelete",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDDelete.")

	// delete a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDPermissionIDsPut handles PUT /customers/{id}/permission_ids request.
// It updates a customer's permission_ids info with the given info.
// @Summary Update a customer's permission_ids.
// @Description Update a customer's permission_ids.
// @Produce json
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers/{id}/permissions_ids [put]
func customersIDPermissionIDsPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDPermissionIDsPut",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDPermissionIDsPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdatePermissionIDs(c.Request.Context(), &u, id, req.PermissionIDs)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDPasswordPut handles PUT /customers/{id}/password request.
// It updates a customer's password.
// @Summary Update a customer's password.
// @Description Update a customer's password.
// @Produce json
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers/{id}/password [put]
func customersIDPasswordPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDPasswordPut",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDPasswordPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdatePassword(c.Request.Context(), &u, id, req.Password)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// customersIDLineInfoPut handles PUT /customers/{id}/line_info request.
// It updates a customer's line info.
// @Summary Update a customer's line info.
// @Description Update a customer's line info.
// @Produce json
// @Success 200 {object} customer.Customer
// @Router /v1.0/customers/{id}/line_info [put]
func customersIDLineInfoPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customersIDLineInfoPut",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"permission_ids": u.PermissionIDs,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	var req request.BodyCustomersIDLineInfoPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CustomerUpdateLineInfo(c.Request.Context(), &u, id, req.LineSecret, req.LineToken)
	if err != nil {
		log.Errorf("Could not update the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

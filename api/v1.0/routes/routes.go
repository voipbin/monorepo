package routes

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

// routesGET handles GET /routes request.
// It returns list of routes of the given customer.
// @Summary List routes
// @Description get routes of the customer
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.ParamRoutesGET
// @Router /v1.0/routes [get]
func routesGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "routesGET",
			"request_address": c.ClientIP,
		},
	)

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

	var req request.ParamRoutesGET
	if err := c.BindQuery(&req); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	} else if pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to max. page_size: %d", pageSize)
	}

	// get customerID
	customerID := uuid.FromStringOrNil(req.CustomerID)

	// get tmps
	tmps, err := serviceHandler.RouteGets(c.Request.Context(), &u, customerID, pageSize, req.PageToken)
	if err != nil {
		logrus.Errorf("Could not get routes info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyRoutesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// routesPOST handles POST /routes request.
// It creates a new route.
// @Summary Create a new route.
// @Description create a new route
// @Produce  json
// @Param route body request.BodyRoutesPOST true "The route detail"
// @Success 200 {object} route.WebhookMessage
// @Router /v1.0/routes [post]
func routesPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "routesPOST",
			"request_address": c.ClientIP,
		},
	)

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

	var req request.BodyRoutesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create
	res, err := serviceHandler.RouteCreate(
		c.Request.Context(),
		&u,
		req.CustomerID,
		req.ProviderID,
		req.Priority,
		req.Target,
	)
	if err != nil {
		log.Errorf("Could not create a route. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// routesIDDelete handles DELETE /routes/<route-id> request.
// It deletes the route.
// @Summary Delete the route
// @Description Delete the route of the given id
// @Produce json
// @Param id path string true "The ID of the route"
// @Success 200
// @Router /v1.0/routes/{id} [delete]
func routesIDDelete(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "routesIDDelete",
			"request_address": c.ClientIP,
		},
	)

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
	if id == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// delete
	res, err := serviceHandler.RouteDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Infof("Could not get the delete the route info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// routesIDGet handles GET /routes/<route-id> request.
// It gets the route.
// @Summary Get the route
// @Description Get the route of the given id
// @Produce json
// @Param id path string true "The ID of the route"
// @Success 200
// @Router /v1.0/routes/{id} [get]
func routesIDGet(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "routesIDGet",
			"request_address": c.ClientIP,
		},
	)

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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.RouteGet(c.Request.Context(), &u, id)
	if err != nil {
		log.Infof("Could not get the route info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// routesIDPUT handles PUT /routes/{id} request.
// It updates a route basic info with the given info.
// And returns updated route info if it succeed.
// @Summary Update an route and reuturns updated route info.
// @Description Update an route and returns detail updated route info.
// @Produce json
// @Param id path string true "The ID of the route"
// @Param update_info body request.BodyQueuesIDPUT true "Queue's update info"
// @Success 200
// @Router /v1.0/routes/{id} [put]
func routesIDPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "routesIDPUT",
			"request_address": c.ClientIP,
		},
	)

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
	log = log.WithField("route_id", id)
	log.Debug("Executing routesIDPUT.")

	var req request.BodyRoutesIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the route
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.RouteUpdate(
		c.Request.Context(),
		&u,
		id,
		req.ProviderID,
		req.Priority,
		req.Target,
	)
	if err != nil {
		log.Errorf("Could not update the route. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

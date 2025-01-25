package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	rmroute "monorepo/bin-route-manager/models/route"

	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetRoutes(c *gin.Context, params openapi_server.GetRoutesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRoutes",
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
		"agent": a,
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

	var tmps []*rmroute.WebhookMessage
	var err error

	if params.CustomerId != nil {
		customerID := uuid.FromStringOrNil(*params.CustomerId)
		tmps, err = h.serviceHandler.RouteGetsByCustomerID(c.Request.Context(), &a, customerID, pageSize, pageToken)
	} else {
		tmps, err = h.serviceHandler.RouteGets(c.Request.Context(), &a, pageSize, pageToken)
	}
	if err != nil {
		logrus.Errorf("Could not get routes info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostRoutes(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRoutes",
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
		"agent": a,
	})

	var req openapi_server.PostRoutesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	customerID := uuid.FromStringOrNil(req.CustomerId)
	providerID := uuid.FromStringOrNil(req.ProviderId)

	res, err := h.serviceHandler.RouteCreate(
		c.Request.Context(),
		&a,
		customerID,
		req.Name,
		req.Detail,
		providerID,
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

func (h *server) DeleteRoutesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRoutesId",
		"request_address": c.ClientIP,
		"route_id":        id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RouteDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the delete the route info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetRoutesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "routesIDGet",
		"request_address": c.ClientIP,
		"route_id":        id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RouteGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the route info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutRoutesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "routesIDPUT",
		"request_address": c.ClientIP,
		"route_id":        id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutRoutesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	providerID := uuid.FromStringOrNil(req.ProviderId)

	res, err := h.serviceHandler.RouteUpdate(
		c.Request.Context(),
		&a,
		target,
		req.Name,
		req.Detail,
		providerID,
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

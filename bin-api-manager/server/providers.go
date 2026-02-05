package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	rmprovider "monorepo/bin-route-manager/models/provider"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetProviders(c *gin.Context, params openapi_server.GetProvidersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProviders",
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

	tmps, err := h.serviceHandler.ProviderList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get providers info. err: %v", err)
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

func (h *server) PostProviders(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostProviders",
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

	var req openapi_server.PostProvidersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	techHeaders := map[string]string{}
	for key, value := range req.TechHeaders {
		strValue, ok := value.(string)
		if !ok {
			log.Errorf("Invalid type for tech header value. key: %s, value: %v", key, value)
			c.AbortWithStatus(400)
			return
		}
		techHeaders[key] = strValue
	}

	res, err := h.serviceHandler.ProviderCreate(
		c.Request.Context(),
		&a,
		rmprovider.Type(req.Type),
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		techHeaders,
		req.Name,
		req.Detail,
	)
	if err != nil {
		log.Errorf("Could not create a provider. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteProvidersId",
		"request_address": c.ClientIP,
		"provider_id":     id,
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

	res, err := h.serviceHandler.ProviderDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the delete the provider info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProvidersId",
		"request_address": c.ClientIP,
		"provider_id":     id,
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

	res, err := h.serviceHandler.ProviderGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the provider info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutProvidersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersIDPUT",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutProvidersIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	techHeaders := map[string]string{}
	for key, value := range req.TechHeaders {
		strValue, ok := value.(string)
		if !ok {
			log.Errorf("Invalid type for tech header value. key: %s, value: %v", key, value)
			c.AbortWithStatus(400)
			return
		}
		techHeaders[key] = strValue
	}

	res, err := h.serviceHandler.ProviderUpdate(
		c.Request.Context(),
		&a,
		target,
		rmprovider.Type(req.Type),
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		techHeaders,
		req.Name,
		req.Detail,
	)
	if err != nil {
		log.Errorf("Could not update the provider. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

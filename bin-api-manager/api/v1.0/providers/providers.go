package providers

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

// providersGET handles GET /providers request.
// It returns list of providers of the given customer.
//
//	@Summary		List providers
//	@Description	get providers of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	provider.WebhookMessage
//	@Router			/v1.0/providers [get]
func providersGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersGET",
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

	var req request.ParamProvidersGET
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

	// get tmps
	tmps, err := serviceHandler.ProviderGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		logrus.Errorf("Could not get providers info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyProvidersGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// providersPOST handles POST /providers request.
// It creates a new provider.
//
//	@Summary		Create a new provider.
//	@Description	create a new provider
//	@Produce		json
//	@Param			provider	body		request.BodyProvidersPOST	true	"The provider detail"
//	@Success		200			{object}	provider.WebhookMessage
//	@Router			/v1.0/providers [post]
func providersPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersPOST",
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

	var req request.BodyProvidersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create
	res, err := serviceHandler.ProviderCreate(
		c.Request.Context(),
		&a,
		req.Type,
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		req.TechHeaders,
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

// providersIDDelete handles DELETE /providers/<provider-id> request.
// It deletes the provider.
//
//	@Summary		Delete the provider
//	@Description	Delete the provider of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the provider"
//	@Success		200
//	@Router			/v1.0/provider/{id} [delete]
func providersIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersIDDelete",
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
	res, err := serviceHandler.ProviderDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the delete the provider info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// providersIDGet handles GET /providers/<provider-id> request.
// It gets the provider.
//
//	@Summary		Get the provider
//	@Description	Get the provider of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the provider"
//	@Success		200
//	@Router			/v1.0/providers/{id} [get]
func providersIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "providersIDGet",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.ProviderGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the provider info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// providersIDPUT handles PUT /providers/{id} request.
// It updates a provider basic info with the given info.
// And returns updated provider info if it succeed.
//
//	@Summary		Update an provider and reuturns updated provider info.
//	@Description	Update an provider and returns detail updated provider info.
//	@Produce		json
//	@Param			id			path	string						true	"The ID of the provider"
//	@Param			update_info	body	request.BodyProvidersIDPUT	true	"Provider's update info"
//	@Success		200
//	@Router			/v1.0/providers/{id} [put]
func providersIDPUT(c *gin.Context) {
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("provider_id", id)
	log.Debug("Executing providersIDPUT.")

	var req request.BodyProvidersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the provider
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ProviderUpdate(
		c.Request.Context(),
		&a,
		id,
		req.Type,
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		req.TechHeaders,
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

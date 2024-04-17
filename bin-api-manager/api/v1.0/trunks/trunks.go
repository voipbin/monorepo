package trunks

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

// trunksPOST handles POST /trunks request.
// It creates a new trunk with the given info and returns created trunk info.
//
//	@Summary		Create a new trunk and returns detail created trunk info.
//	@Description	Create a new trunk and returns detail created trunk info.
//	@Produce		json
//	@Success		200	{object}	trunk.Trunk
//	@Router			/v1.0/trunks [post]
func trunksPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "trunksPOST",
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

	var req request.BodyTrunksPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a trunk
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	trunk, err := serviceHandler.TrunkCreate(c.Request.Context(), &a, req.Name, req.Detail, req.DomainName, req.AuthTypes, req.Username, req.Password, req.AllowedIPs)
	if err != nil {
		log.Errorf("Could not create a trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, trunk)
}

// trunksPOST handles GET /trunks request.
// It gets a list of trunks with the given info.
//
//	@Summary		Gets a list of trunks.
//	@Description	Gets a list of trunks
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyTrunksGET
//	@Router			/v1.0/trunks [get]
func trunksGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "trunksGET",
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

	var req request.ParamTrunksGET
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
	log.Debugf("trunksGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get trunks
	trunks, err := serviceHandler.TrunkGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a trunk list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(trunks) > 0 {
		nextToken = trunks[len(trunks)-1].TMCreate
	}
	res := response.BodyTrunksGET{
		Result: trunks,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// trunksIDGET handles GET /trunks/{id} request.
// It returns detail trunk info.
//
//	@Summary		Returns detail trunk info.
//	@Description	Returns detail trunk info of the given trunk id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the trunk"
//	@Success		200	{object}	trunk.Trunk
//	@Router			/v1.0/trunks/{id} [get]
func trunksIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "trunksIDGET",
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
	log = log.WithField("trunk_id", id)
	log.Debug("Executing trunksIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.TrunkGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// trunksIDPUT handles PUT /trunks/{id} request.
// It updates a exist trunk info with the given trunk info.
// And returns updated trunk info if it succeed.
//
//	@Summary		Update a trunk and reuturns updated trunk info.
//	@Description	Update a trunk and returns detail updated trunk info.
//	@Produce		json
//	@Success		200	{object}	trunk.Trunk
//	@Router			/v1.0/trunks/{id} [put]
func trunksIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "trunksIDPUT",
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
	log = log.WithField("trunk_id", id)

	var req request.BodyTrunksIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a trunk
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.TrunkUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail, req.AuthTypes, req.Username, req.Password, req.AllowedIPs)
	if err != nil {
		log.Errorf("Could not update the trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// trunksIDDELETE handles DELETE /trunks/{id} request.
// It deletes a exist trunk info.
//
//	@Summary		Delete a existing trunk.
//	@Description	Delete a existing trunk.
//	@Produce		json
//	@Success		200
//	@Router			/v1.0/trunks/{id} [delete]
func trunksIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "trunksIDDELETE",
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
	log = log.WithField("trunk_id", id)
	log.Debug("Executing trunksIDDELETE.")

	// delete a trunk
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.TrunkDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

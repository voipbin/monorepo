package tags

import (
	_ "monorepo/bin-tag-manager/models/tag" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// tagsPOST handles POST /tag request.
// It creates a new tag.
//
//	@Summary		Create a new tag.
//	@Description	create a new tag.
//	@Produce		json
//	@Prama			tag body request.BodyTagsPOST true "Creating tag info."
//	@Success		200	{object}	tag.Tag
//	@Router			/v1.0/tags [post]
func tagsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "tagsPOST",
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

	var req request.BodyTagsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create flow
	res, err := serviceHandler.TagCreate(c.Request.Context(), &a, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a new tag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// tagsIDDelete handles DELETE /tags/<tag-id> request.
// It deletes the tag.
//
//	@Summary		Delete the tag
//	@Description	Delete the tag of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the tag"
//	@Success		200
//	@Router			/v1.0/tags/{id} [delete]
func tagsIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "tagsIDDelete",
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
	res, err := serviceHandler.TagDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not delete the tag info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// tagsIDGet handles GET /tags/<tag-id> request.
// It gets the tag.
//
//	@Summary		Get the tag
//	@Description	Get the tag of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the tag"
//	@Success		200	{object}	tag.Tag
//	@Router			/v1.0/tags/{id} [get]
func tagsIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "tagsIDGet",
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

	// get
	res, err := serviceHandler.TagGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the tag info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// tagsGET handles GET /tags request.
// It returns list of tags of the given customer.
//
//	@Summary		List tags
//	@Description	get tags of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyTagsGET
//	@Router			/v1.0/tags [get]
func tagsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "tagsGET",
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

	var req request.ParamTagsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	} else if pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to max. page_size: %d", pageSize)
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmps
	tmps, err := serviceHandler.TagGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get tags info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyTagsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// tagsIDPUT handles PUT /tags/{id} request.
// It updates a tag's basic info with the given info.
//
//	@Summary		Update the tag info.
//	@Description	Update the tag info.
//	@Produce		json
//	@Param			id			path	string					true	"The tag's id."
//	@Param			update_info	body	request.BodyTagsIDPUT	true	"The update info."
//	@Success		200
//	@Router			/v1.0/tags/{id} [put]
func tagsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "tagsIDPUT",
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
	log = log.WithField("tag_id", id)

	var req request.BodyTagsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing tagsIDPUT.")

	// update the tag
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.TagUpdate(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the tag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

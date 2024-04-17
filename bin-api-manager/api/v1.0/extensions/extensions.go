package extensions

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

// extensionsPOST handles POST /extension request.
// It creates a new extension with the given info and returns created extension info.
//
//	@Summary		Create a new domain and returns detail created extension info.
//	@Description	Create a new extension and returns detail created extension info.
//	@Produce		json
//	@Param			extension	body		request.BodyExtensionsPOST	true	"extension info"
//	@Success		200			{object}	extension.Extension
//	@Router			/v1.0/extensions [post]
func extensionsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsPOST",
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

	var req request.BodyExtensionsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a extension
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	ext, err := serviceHandler.ExtensionCreate(c.Request.Context(), &a, req.Extension, req.Password, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, ext)
}

// extensionsGET handles GET /extensions request.
// It gets a list of extensions with the given info.
//
//	@Summary		Gets a list of extensions.
//	@Description	Gets a list of extensions
//	@Produce		json
//	@Param			page_size	query	int		false	"The size of results. Max 100"
//	@Param			page_token	query	string	false	"The token. tm_create"
//	@Param			domain_id	query	string	true	"The domain's id"
//	@Success		200			{array}	extension.Extension
//	@Router			/v1.0/extensions [get]
func extensionsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsGET",
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

	var req request.ParamExtensionsGET
	if err := c.BindQuery(&req); err != nil {
		logrus.Errorf("Could not bind query. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get params
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("extensionsGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get extensions
	exts, err := serviceHandler.ExtensionGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a extensions list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(exts) > 0 {
		nextToken = exts[len(exts)-1].TMCreate
	}
	res := response.BodyExtensionsGET{
		Result: exts,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// extensionsIDGET handles GET /extensions/{id} request.
// It returns detail extension info.
//
//	@Summary		Returns detail extension info.
//	@Description	Returns detail extension info of the given extension id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the extension"
//	@Success		200	{object}	extension.Extension
//	@Router			/v1.0/extension/{id} [get]
func extensionsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsIDGET",
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
	log = log.WithField("extension_id", id)
	log.Debug("Executing extensionsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ExtensionGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get the extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// extensionsIDPUT handles PUT /extensions/{id} request.
// It updates a exist extension info with the given extension info.
// And returns updated extension info if it succeed.
//
//	@Summary		Update a extension and reuturns updated extension info.
//	@Description	Update a extension and returns detail updated extension info.
//	@Produce		json
//	@Param			id			path		string						true	"extension's id"
//	@Param			update_info	body		request.BodyExtensionsIDPUT	true	"Update info"
//	@Success		200			{object}	extension.Extension
//	@Router			/v1.0/extensions/{id} [put]
func extensionsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsIDPUT",
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
	log = log.WithField("extension_id", id)

	var req request.BodyExtensionsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debug("Executing extensionsIDPUT.")

	// update
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ExtensionUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.Password)
	if err != nil {
		log.Errorf("Could not update the extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// extensionsIDDELETE handles DELETE /extensions/{id} request.
// It deletes a exist extensions info.
//
//	@Summary		Delete a existing extension.
//	@Description	Delete a existing extension.
//	@Produce		json
//	@Success		200
//	@Param			id	path	string	true	"The extension's id"
//	@Router			/v1.0/extensions/{id} [delete]
func extensionsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsIDDELETE",
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
	log = log.WithField("extension_id", id)
	log.Debug("Executing extensionsIDDELETE.")

	// delete a domain
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ExtensionDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not create a extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

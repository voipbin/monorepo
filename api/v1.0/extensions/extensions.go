package extensions

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// extensionsPOST handles POST /extension request.
// It creates a new extension with the given info and returns created extension info.
// @Summary Create a new domain and returns detail created extension info.
// @Description Create a new extension and returns detail created extension info.
// @Produce json
// @Success 200 {object} extension.Extension
// @Router /v1.0/extensions [post]
func extensionsPOST(c *gin.Context) {

	var body request.BodyExtensionsPOST
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// create a extension
	e := &extension.Extension{
		UserID:   u.ID,
		Name:     body.Name,
		Detail:   body.Detail,
		DomainID: body.DomainID,

		Extension: body.Extension,
		Password:  body.Password,
	}

	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	ext, err := serviceHandler.ExtensionCreate(&u, e)
	if err != nil {
		log.Errorf("Could not create a extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, ext)
	return
}

// extensionsGET handles GET /extensions request.
// It gets a list of extensions with the given info.
// @Summary Gets a list of extensions.
// @Description Gets a list of extensions
// @Produce json
// @Success 200 {array} extension.Extension
// @Router /v1.0/extensions [get]
func extensionsGET(c *gin.Context) {

	var requestParam request.ParamExtensionsGET

	if err := c.BindQuery(&requestParam); err != nil {
		logrus.Errorf("Could not bind query. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
			"request":         requestParam,
		},
	)
	log.Debugf("extensionsGET. Received request detail. domain_id: %s, page_size: %d, page_token: %s", requestParam.DomainID, requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	u := tmp.(user.User)
	log = log.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get extensions
	domainID := uuid.FromStringOrNil(requestParam.DomainID)
	exts, err := serviceHandler.ExtensionGets(&u, domainID, pageSize, requestParam.PageToken)
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
	return
}

// extensionsIDGET handles GET /extensions/{id} request.
// It returns detail extension info.
// @Summary Returns detail extension info.
// @Description Returns detail extension info of the given extension id.
// @Produce json
// @Param id path string true "The ID of the extension"
// @Param token query string true "JWT token"
// @Success 200 {object} extension.Extension
// @Router /v1.0/extension/{id} [get]
func extensionsIDGET(c *gin.Context) {
	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing extensionsIDGET.")

	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ExtensionGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// extensionsIDPUT handles PUT /extensions/{id} request.
// It updates a exist extension info with the given extension info.
// And returns updated extension info if it succeed.
// @Summary Update a extension and reuturns updated extension info.
// @Description Update a extension and returns detail updated extension info.
// @Produce json
// @Success 200 {object} extension.Extension
// @Router /v1.0/extensions/{id} [put]
func extensionsIDPUT(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var body request.BodyExtensionsIDPUT
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	f := &extension.Extension{
		ID:       id,
		Name:     body.Name,
		Detail:   body.Detail,
		Password: body.Password,
	}

	// update a domain
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ExtensionUpdate(&u, f)
	if err != nil {
		log.Errorf("Could not create a extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
	return
}

// extensionsIDDELETE handles DELETE /extensions/{id} request.
// It deletes a exist extensions info.
// @Summary Delete a existing extension.
// @Description Delete a existing extension.
// @Produce json
// @Success 200
// @Router /v1.0/extensions/{id} [delete]
func extensionsIDDELETE(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// delete a domain
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.ExtensionDelete(&u, id); err != nil {
		log.Errorf("Could not create a extension. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
	return
}

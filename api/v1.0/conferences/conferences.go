package conferences

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.POST("", conferencesPOST)
	conferences.GET("", conferencesGET)
	conferences.GET("/:id", conferencesIDGET)
	conferences.DELETE("/:id", conferencesIDDELETE)
}

// conferencesGET handles GET /conferences request.
// It returns list of conferences of the given user.

// @Summary Get list of conferences
// @Description get conferences of the user
// @Produce json
// @Param token query string true "JWT token"
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyCallsGET
// @Router /v1.0/conferences [get]
func conferencesGET(c *gin.Context) {

	var requestParam request.ParamConferencesGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("conferencesGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get conferences
	confs, err := serviceHandler.ConferenceGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not create a flow for outoing call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(confs) > 0 {
		nextToken = confs[len(confs)-1].TMCreate
	}

	res := response.BodyConferencesGET{
		Result: confs,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conferencesPOST handles POST /conferences request.
// It creates a new conference and returns the created conferences of the given user.
// @Summary Create a new conferences
// @Description Create a new conference with the given information.
// @Produce json
// @Param token query string true "JWT token"
// @Param call body request.BodyConferencesPOST true "The conference detail"
// @Success 200 {object} conference.Conference
// @Router /v1.0/conferences [post]
func conferencesPOST(c *gin.Context) {
	var requestBody request.BodyConferencesPOST

	if err := c.BindJSON(&requestBody); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceCreate(&u, requestBody.Type, requestBody.Name, requestBody.Detail, requestBody.WebhookURI)
	if err != nil || res == nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDGET handles GET /conferences/{id} request.
// It returns detail conference info.
// @Summary Returns detail conference info.
// @Description Returns detail conference info of the given conference id.
// @Produce json
// @Param id path string true "The ID of the conference"
// @Param token query string true "JWT token"
// @Success 200 {object} conference.Conference
// @Router /v1.0/conferences/{id} [get]
func conferencesIDGET(c *gin.Context) {
	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceGet(&u, id)
	if err != nil || res == nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDDELETE handles DELETE /conferences/{id} request.
// It deletes the conference.
// @Summary Delete the confernce.
// @Description Delete the conference. All the participants in the conference will be kicked out.
// @Produce json
// @Param id path string true "The ID of the conference"
// @Param token query string true "JWT token"
// @Success 200
// @Router /v1.0/conferences/{id} [delete]
func conferencesIDDELETE(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	err := servicehandler.ConferenceDelete(&u, id)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

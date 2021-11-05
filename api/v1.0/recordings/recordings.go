package recordings

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// recordingsGET handles GET /recordings request.
// It returns list of calls of the given user.
// @Summary List recordings
// @Description get recordings of the user
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyRecordingsGET
// @Router /v1.0/recordings [get]
func recordingsGET(c *gin.Context) {

	var requestParam request.ParamRecordingsGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("recordingsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

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

	// get recordings
	recordings, err := serviceHandler.RecordingGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get a recordings. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(recordings) > 0 {
		nextToken = recordings[len(recordings)-1].TMCreate
	}
	res := response.BodyRecordingsGET{
		Result: recordings,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// recordingsIDGET handles GET /recordings/<id> request.
// It returns a detail recording info.
// @Summary Returns a detail recording information.
// @Description Returns a detial recording information of the given recording id.
// @Produce json
// @Success 200 {object} recording.Recording
// @Router /v1.0/recordings/{id} [get]
func recordingsIDGET(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	// get user
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing recordingsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.RecordingGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a recording info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

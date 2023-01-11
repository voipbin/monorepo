package conferences

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

// conferencesGET handles GET /conferences request.
// It returns list of conferences of the given customer.

// @Summary Get list of conferences
// @Description get conferences of the customer
// @Produce json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyCallsGET
// @Router /v1.0/conferences [get]
func conferencesGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesGET",
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

	var requestParam request.ParamConferencesGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("conferencesGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get conferences
	confs, err := serviceHandler.ConferenceGets(c.Request.Context(), &u, pageSize, requestParam.PageToken)
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
// It creates a new conference and returns the created conferences of the given customer.
// @Summary Create a new conferences
// @Description Create a new conference with the given information.
// @Produce json
// @Param conference body request.BodyConferencesPOST true "The conference detail"
// @Success 200 {object} conference.Conference
// @Router /v1.0/conferences [post]
func conferencesPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesPOST",
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

	var requestBody request.BodyConferencesPOST
	if err := c.BindJSON(&requestBody); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceCreate(c.Request.Context(), &u, requestBody.Type, requestBody.Name, requestBody.Detail, requestBody.PreActions, requestBody.PostActions)
	if err != nil || res == nil {
		log.Errorf("Could not create the conference. err: %v", err)
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
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesIDGET",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDGET.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceGet(c.Request.Context(), &u, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDDELETE handles DELETE /conferences/{id} request.
// It deletes the conference.
// @Summary Delete the conference.
// @Description Delete the conference. All the participants in the conference will be kicked out.
// @Produce json
// @Param id path string true "The ID of the conference"
// @Success 200
// @Router /v1.0/conferences/{id} [delete]
func conferencesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesIDDELETE",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	err := servicehandler.ConferenceDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// conferencesIDRecordingStartPOST handles DELETE /conferences/{id}/recording_start request.
// It starts the conference recording.
// @Summary Starts the conference recording.
// @Description Start the conference recording.
// @Produce json
// @Param id path string true "The ID of the conference"
// @Success 200
// @Router /v1.0/conferences/{id}/recording_start [post]
func conferencesIDRecordingStartPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesIDRecordingStartPOST",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	err := servicehandler.ConferenceRecordingStart(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// conferencesIDRecordingStopPOST handles DELETE /conferences/{id}/recording_stop request.
// It stops the conference recording.
// @Summary Stops the conference recording.
// @Description Stops the conference recording.
// @Produce json
// @Param id path string true "The ID of the conference"
// @Success 200
// @Router /v1.0/conferences/{id}/recording_stop [post]
func conferencesIDRecordingStopPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "conferencesIDRecordingStopPOST",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDRecordingStopPOST.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	err := servicehandler.ConferenceRecordingStop(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

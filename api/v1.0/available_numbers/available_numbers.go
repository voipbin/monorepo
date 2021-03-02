package availablenumbers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// availableNumbersGET handles GET /available_numbers request.
// It returns list of available numbers of the given country.
// @Summary List available numbers
// @Description get available numbers of the country
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param country_code query string true "The ISO country code"
// @Success 200 {object} response.BodyRecordingsGET
// @Router /v1.0/available_numbers [get]
func availableNumbersGET(c *gin.Context) {

	var requestParam request.ParamAvailableNumbersGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("availableNumbersGET. Received request detail. page_size: %d, country_code: %s", requestParam.PageSize, requestParam.CountyCode)

	// get user
	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// set country code
	countryCode := requestParam.CountyCode
	if countryCode == "" {
		logrus.Infof("Not acceptable country code. country_code: %s", countryCode)
		c.AbortWithStatus(400)
		return
	}

	// get service and available numbers
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	availableNumbers, err := serviceHandler.AvailableNumberGets(&u, pageSize, countryCode)
	if err != nil {
		logrus.Errorf("Could not get available numbers. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res := response.BodyAvailableNumbersGET{
		Result: availableNumbers,
	}

	c.JSON(200, res)
}

// // recordingsIDGET handles GET /recordings/<id> request.
// // It returns a detail recording info.
// // @Summary Returns a detail recording information.
// // @Description Returns a detial recording information of the given recording id.
// // @Produce json
// // @Success 200 {object} recording.Recording
// // @Router /v1.0/recordings/{id} [get]
// func recordingsIDGET(c *gin.Context) {

// 	// get id
// 	id := uuid.FromStringOrNil(c.Params.ByName("id"))

// 	tmp, exists := c.Get("user")
// 	if exists != true {
// 		logrus.Errorf("Could not find user info.")
// 		c.AbortWithStatus(400)
// 		return
// 	}

// 	// get user
// 	u := tmp.(user.User)
// 	log := logrus.WithFields(logrus.Fields{
// 		"id":         u.ID,
// 		"username":   u.Username,
// 		"permission": u.Permission,
// 	})
// 	log.Debug("Executing recordingsIDGET.")

// 	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
// 	res, err := serviceHandler.RecordingGet(&u, id)
// 	if err != nil {
// 		log.Errorf("Could not get a recording info. err: %v", err)
// 		c.AbortWithStatus(400)
// 		return
// 	}

// 	c.JSON(200, res)
// }

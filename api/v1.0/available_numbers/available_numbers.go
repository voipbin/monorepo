package availablenumbers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// availableNumbersGET handles GET /available_numbers request.
// It returns list of available numbers of the given country.
// @Summary List available numbers
// @Description get available numbers of the country
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param country_code query string true "The ISO country code"
// @Success 200 {object} response.BodyAvailableNumbersGET
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

	// get customer
	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)

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
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
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

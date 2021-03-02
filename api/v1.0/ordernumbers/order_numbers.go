package ordernumbers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// orderNumbersGET handles GET /order_numbers request.
// It returns list of order numbers of the given user.
// @Summary List order numbers
// @Description get order numbers of the country
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param country_code query string true "The ISO country code"
// @Success 200 {object} response.BodyOrderNumbersGET
// @Router /v1.0/order_numbers [get]
func orderNumbersGET(c *gin.Context) {

	var requestParam request.ParamOrderNumbersGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("orderNumbersGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

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

	// get order numbers
	numbers, err := serviceHandler.OrderNumberGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		log.Errorf("Could not get a order number list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(numbers) > 0 {
		nextToken = numbers[len(numbers)-1].TMCreate
	}
	res := response.BodyOrderNumbersGET{
		Result: numbers,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
	return

}

// orderNumbersPOST handles POST /order_numbers request.
// It creates a new order number with the given info and returns created order number.
// @Summary Create a new number and returns detail created number info.
// @Description Create a new number and returns detail created number info.
// @Produce json
// @Success 200 {object} number.Number
// @Router /v1.0/order_numbers [post]
func orderNumbersPOST(c *gin.Context) {

	var body request.BodyOrderNumbersPOST
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

	// create a number
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.OrderNumberCreate(&u, body.Number)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
	return
}

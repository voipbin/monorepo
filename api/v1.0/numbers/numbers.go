package numbers

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// numbersGET handles GET /numbers request.
// It returns list of order numbers of the given user.
// @Summary List order numbers
// @Description get order numbers of the country
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param country_code query string true "The ISO country code"
// @Success 200 {object} response.BodyOrderNumbersGET
// @Router /v1.0/numbers [get]
func numbersGET(c *gin.Context) {

	var requestParam request.ParamNumbersGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("numbersGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	u := tmp.(models.User)
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
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get order numbers
	numbers, err := serviceHandler.NumberGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		log.Errorf("Could not get a order number list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(numbers) > 0 {
		nextToken = numbers[len(numbers)-1].TMCreate
	}
	res := response.BodyNumbersGET{
		Result: numbers,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
	return
}

// numbersIDGET handles GET /numbers/<id> request.
// It returns order numbers of the given id.
// @Summary Get order number
// @Description get order number of the given id
// @Produce  json
// @Param id path string true "The ID of the order number"
// @Param token query string true "JWT token"
// @Success 200 {object} models.Number
// @Router /v1.0/numbers/{id} [get]
func numbersIDGET(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)

	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
			"id":              u.ID,
			"username":        u.Username,
			"permission":      u.Permission,
			"number":          id,
		},
	)
	log.Debugf("numbersIDGET. Received request detail. number_id: %s", id)

	// get order number
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.NumberGet(&u, id)
	if err != nil {
		log.Errorf("Could not get an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
	return
}

// numbersPOST handles POST /numbers request.
// It creates a new order number with the given info and returns created order number.
// @Summary Create a new number and returns detail created number info.
// @Description Create a new number and returns detail created number info.
// @Produce json
// @Success 200 {object} models.Number
// @Router /v1.0/numbers [post]
func numbersPOST(c *gin.Context) {

	var body request.BodyNumbersPOST
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
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// create a number
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.NumberCreate(&u, body.Number)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
	return
}

// numbersIDDELETE handles DELETE /numbers/<id> request.
// It deletes the given id of order number and returns the deleted order number.
// @Summary Delete order number
// @Description delete order number of the given id and returns deleted item.
// @Produce  json
// @Param id path string true "The ID of the order number"
// @Param token query string true "JWT token"
// @Success 200 {object} models.Number
// @Router /v1.0/numbers/{id} [delete]
func numbersIDDELETE(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)

	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
			"id":              u.ID,
			"username":        u.Username,
			"permission":      u.Permission,
			"number":          id,
		},
	)
	log.Debugf("numbersIDDELETE. Received request detail. number_id: %s", id)

	// delete order number
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.NumberDelete(&u, id)
	if err != nil {
		log.Errorf("Could not delete an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
	return
}

// numbersIDPUT handles PUT /numbers request.
// It creates a new order number with the given info and returns created order number.
// @Summary Create a new number and returns detail created number info.
// @Description Create a new number and returns detail created number info.
// @Produce json
// @Success 200 {object} models.Number
// @Router /v1.0/numbers/{id} [post]
func numbersIDPUT(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var body request.BodyNumbersIDPUT
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
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	tmpN := &models.Number{
		ID:     id,
		FlowID: body.FlowID,
	}

	// update a number
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.NumberUpdate(&u, tmpN)
	if err != nil {
		log.Errorf("Could not update a number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
	return
}

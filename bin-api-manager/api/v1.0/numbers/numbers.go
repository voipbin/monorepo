package numbers

import (
	_ "monorepo/bin-number-manager/models/number" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// numbersGET handles GET /numbers request.
// It returns list of order numbers of the given customer.
//
//	@Summary		List order numbers
//	@Description	get order numbers of the country
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyNumbersGET
//	@Router			/v1.0/numbers [get]
func numbersGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersGET",
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

	var req request.ParamNumbersGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("numbersGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get numbers
	numbers, err := serviceHandler.NumberGets(c.Request.Context(), &a, pageSize, req.PageToken)
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
}

// numbersIDGET handles GET /numbers/<id> request.
// It returns order numbers of the given id.
//
//	@Summary		Get order number
//	@Description	get order number of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the order number"
//	@Success		200	{object}	number.Number
//	@Router			/v1.0/numbers/{id} [get]
func numbersIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersIDGET",
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
	log = log.WithField("number_id", id)
	log.Debugf("Executing numbersIDGET.")

	// get order number
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.NumberGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// numbersPOST handles POST /numbers request.
// It creates a new order number with the given info and returns created order number.
//
//	@Summary		Create a new number and returns detail created number info.
//	@Description	Create a new number and returns detail created number info.
//	@Produce		json
//	@Param			number	body		request.BodyNumbersPOST	true	"Creating number info."
//	@Success		200		{object}	number.Number
//	@Router			/v1.0/numbers [post]
func numbersPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersPOST",
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

	var req request.BodyNumbersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a number
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.NumberCreate(c.Request.Context(), &a, req.Number, req.CallFlowID, req.MessageFlowID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create the number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
}

// numbersIDDELETE handles DELETE /numbers/<id> request.
// It deletes the given id of order number and returns the deleted order number.
//
//	@Summary		Delete order number
//	@Description	delete order number of the given id and returns deleted item.
//	@Produce		json
//	@Param			id	path		string	true	"The number's id"
//	@Success		200	{object}	number.Number
//	@Router			/v1.0/numbers/{id} [delete]
func numbersIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersIDDELETE",
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
	log = log.WithField("number_id", id)
	log.Debugf("Executing numbersIDDELETE.")

	// delete order number
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.NumberDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// numbersIDPUT handles PUT /numbers/<id> request.
// It creates a new order number with the given info and returns created order number.
//
//	@Summary		Update the number's basic information.
//	@Description	Update the number's basic information.
//	@Produce		json
//	@Param			update_info	body		request.BodyNumbersIDPUT	true	"Update info."
//	@Success		200			{object}	number.Number
//	@Router			/v1.0/numbers/{id} [put]
func numbersIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersIDPUT",
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
	log = log.WithField("number_id", id)

	var req request.BodyNumbersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing numbersIDPUT.")

	// update a number
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.NumberUpdate(c.Request.Context(), &a, id, req.CallFlowID, req.MessageFlowID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update a number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
}

// numbersIDFlowIDPUT handles PUT /numbers/<id>/flow_id request.
// It updates the number's flow_id.
//
//	@Summary		Update the number's flow_id.
//	@Description	Update the number's flow_id.
//	@Produce		json
//	@Param			update_info	body		request.BodyNumbersIDFlowIDPUT	true	"Update info."
//	@Success		200			{object}	number.Number
//	@Router			/v1.0/numbers/{id}/flow_id [put]
func numbersIDFlowIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersIDPUT",
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
	log = log.WithField("number_id", id)

	var req request.BodyNumbersIDFlowIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing numbersIDPUT.")

	// update a number
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	numb, err := serviceHandler.NumberUpdateFlowIDs(c.Request.Context(), &a, id, req.CallFlowID, req.MessageFlowID)
	if err != nil {
		log.Errorf("Could not update a number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
}

// numbersRenewPOST handles POST /numbers/renew request.
// It renews the number's.
//
//	@Summary		Renew the numbers.
//	@Description	Renew the numbers.
//	@Produce		json
//	@Param			update_info	body	request.BodyNumbersIDFlowIDPUT	true	"Update info."
//	@Success		200			{array}	number.Number
//	@Router			/v1.0/numbers/renew [post]
func numbersRenewPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "numbersIDPUT",
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

	var req request.BodyNumbersRenewPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing numbersRenewPOST.")

	// renew a numbers
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.NumberRenew(c.Request.Context(), &a, req.TMRenew)
	if err != nil {
		log.Errorf("Could not renew the numbers. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

package calls

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

// callsPOST handles POST /calls request.
// It creates a temp flow and create a call with temp flow.

// @Summary Make an outbound call
// @Description dialing to destination
// @Produce  json
// @Param call body request.BodyCallsPOST true "The call detail"
// @Success 200 {object} call.Call
// @Router /v1.0/calls [post]
func callsPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "callsPOST",
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

	var req request.BodyCallsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create call
	res, err := serviceHandler.CallCreate(c.Request.Context(), &u, req.FlowID, req.Actions, &req.Source, req.Destinations)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// callsIDDelete handles DELETE /calls/<call-id> request.
// It hangup the call.

// @Summary Hangup the call
// @Description Hangup the call of the given id
// @Produce json
// @Param id path string true "The ID of the call"
// @Success 200 {object} call.Call
// @Router /v1.0/calls/{id} [delete]
func callsIDDelete(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "callsIDDelete",
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
	log.Debug("Executing callsIDDelete.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// hangup the call
	err := serviceHandler.CallDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsGET handles GET /calls request.
// It returns list of calls of the given customer.

// @Summary Get list of calls
// @Description get calls of the customer
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyCallsGET
// @Router /v1.0/calls [get]
func callsGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "callsGET",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
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

	var requestParam request.ParamCallsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("callsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get calls
	calls, err := serviceHandler.CallGets(c.Request.Context(), &u, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(calls) > 0 {
		nextToken = calls[len(calls)-1].TMCreate
	}
	res := response.BodyCallsGET{
		Result: calls,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// callsIDGET handles GET /calls/{id} request.
// It returns detail call info.
// @Summary Get detail call info.
// @Description Returns detail call info of the given call id.
// @Produce json
// @Param id path string true "The ID of the call"
// @Success 200 {object} call.Call
// @Router /v1.0/calls/{id} [get]
func callsIDGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "callsIDGET",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CallGet(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

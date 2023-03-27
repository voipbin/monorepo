package activeflows

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

// activeflowsGET handles GET /activeflows request.
// It returns list of activeflows of the given customer.

// @Summary Get list of activeflows
// @Description get activeflows of the customer
// @Produce json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyActiveflowsGET
// @Router /v1.0/activeflows [get]
func activeflowsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer": u,
	})

	var requestParam request.ParamActiveflowsGET
	if errBind := c.BindQuery(&requestParam); errBind != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", errBind)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get activeflows
	activeflows, err := serviceHandler.ActiveflowGets(c.Request.Context(), &u, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(activeflows) > 0 {
		nextToken = activeflows[len(activeflows)-1].TMCreate
	}
	res := response.BodyActiveflowsGET{
		Result: activeflows,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// activeflowsIDGET handles GET /activeflows/{id} request.
// It returns detail activeflow info.
// @Summary Get detail activeflow info.
// @Description Returns detail activeflow info of the given activeflow id.
// @Produce json
// @Param id path string true "The ID of the activeflow"
// @Success 200 {object} activeflow.Activeflow
// @Router /v1.0/activeflows/{id} [get]
func activeflowsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsIDGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"username":       u.Username,
		"permission_ids": u.PermissionIDs,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("activeflow_id", id)
	log.Debug("Executing activeflowsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ActiveflowGet(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not get a activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// activeflowsIDDELETE handles DELETE /activeflows/{id} request.
// It deletes activeflow info.
// @Summary Deletes activeflow info.
// @Description Deletes activeflow info of the given activeflow id.
// @Produce json
// @Param id path string true "The ID of the activeflow"
// @Success 200 {object} activeflow.Activeflow
// @Router /v1.0/activeflows/{id} [delete]
func activeflowsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsIDDELETE",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"username":       u.Username,
		"permission_ids": u.PermissionIDs,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("activeflow_id", id)
	log.Debug("Executing activeflowsIDDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ActiveflowDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not delete the activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// activeflowsIDStopPOST handles POST /activeflows/{id}/stop request.
// It stops the activeflow info.
// @Summary Stops the given activeflow.
// @Description Stops activeflow of the given activeflow id.
// @Produce json
// @Param id path string true "The ID of the activeflow"
// @Success 200 {object} activeflow.Activeflow
// @Router /v1.0/activeflows/{id}/stop [post]
func activeflowsIDStopPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsIDStopPOST",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		logrus.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"username":       u.Username,
		"permission_ids": u.PermissionIDs,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("activeflow_id", id)
	log.Debug("Executing activeflowsIDStopPOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ActiveflowStop(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

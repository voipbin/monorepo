package queuecalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// queuecallsIDGET handles GET /queuecalls/{id} request.
// It returns detail queuecall info.
// @Summary     Returns detail queuecall info.
// @Description Returns detail conferencecall info of the given queuecall id.
// @Produce     json
// @Param       id    path     string true "The ID of the queuecall"
// @Param       token query    string true "JWT token"
// @Success     200   {object} queuecall.Queuecall
// @Router      /v1.0/queuecall/{id} [get]
func queuecallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallGet(c.Request.Context(), &u, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuecallsIDDELETE handles DELETE /queuecalls/{id} request.
// It deletes the queuecall.
// @Summary     Deletes the queuecall.
// @Description Deletes the queuecall.
// @Produce     json
// @Param       id path string true "The ID of the queuecall"
// @Success     200
// @Router      /v1.0/queuecalls/{id} [delete]
func queuecallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDDELETE",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

// queuecallsIDKickPOST handles POST /queuecalls/{id}/kick request.
// It kicks the queuecall from the queue.
// @Summary     Kicks the queuecall from the queue.
// @Description Kicks the queuecall.
// @Produce     json
// @Param       id path string true "The ID of the queuecall"
// @Success     200
// @Router      /v1.0/queuecalls/{id}/kick [post]
func queuecallsIDKickPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDKickPOST",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallKick(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

// queuecallsReferenceIDIDKickPOST handles POST /queuecalls/reference_id/{id}/kick request.
// It kicks the queuecall of the given reference id from the queue.
// @Summary     Kicks the queuecall of the given reference id from the queue.
// @Description Kicks the queuecall of the given reference id.
// @Produce     json
// @Param       id path string true "The ID of the queuecall"
// @Success     200
// @Router      /v1.0/queuecalls/reference_id/{id}/kick [post]
func queuecallsReferenceIDIDKickPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsReferenceIDIDKickPOST",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}

	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"username":       u.Username,
		"permission_ids": u.PermissionIDs,
	})

	// get referenceID
	referenceID := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("queuecall_id", referenceID)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallKickByReferenceID(c.Request.Context(), &u, referenceID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

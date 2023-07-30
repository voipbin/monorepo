package conferencecalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// conferencecallsIDGET handles GET /conferencecalls/{id} request.
// It returns detail conferencecall info.
// @Summary     Returns detail conferencecall info.
// @Description Returns detail conferencecall info of the given conferencecall id.
// @Produce     json
// @Param       id    path     string true "The ID of the conferencecall"
// @Param       token query    string true "JWT token"
// @Success     200   {object} conferencecall.Conferencecall
// @Router      /v1.0/conferencecalls/{id} [get]
func conferencecallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencecallsIDGET",
		"request_address": c.ClientIP,
	})

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
	log = log.WithField("conferencecall_id", id)
	log.Debug("Executing conferencecallsIDGET.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferencecallGet(c.Request.Context(), &u, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencecallsIDDELETE handles DELETE /conferencecalls/{id} request.
// It kicks the conferencecall from the conference.
// @Summary     Kicks the conferencecall from the conference.
// @Description Kicks the conferencecall.
// @Produce     json
// @Param       id path string true "The ID of the conferencecall"
// @Success     200
// @Router      /v1.0/conferencecalls/{id} [delete]
func conferencecallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencecallsIDDELETE",
		"request_address": c.ClientIP,
	})

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
	log = log.WithField("conferencecall_id", id)
	log.Debug("Executing conferencecallsIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferencecallKick(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not kick the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

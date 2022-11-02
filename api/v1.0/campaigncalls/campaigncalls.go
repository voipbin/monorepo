package campaigncalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	_ "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall" // for sweg use.
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// campaigncallsIDGET handles GET /campaigncalls/{id} request.
// It returns detail campaigncall info.
// @Summary Returns detail campaigncall info.
// @Description Returns detail campaigns info of the given campaigncall id.
// @Produce json
// @Param id path string true "The ID of the campaigncall"
// @Success 200 {object} campaigncall.WebhookMessage
// @Router /v1.0/campaigncalls/{id} [get]
func campaigncallsIDGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "campaigncallsIDGET",
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
	log = log.WithField("campaign_id", id)
	log.Debug("Executing campaignsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaigncallGet(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not get a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaigncallsIDDELETE handles DELETE /campaigncalls/{id} request.
// It finishs a campaigncall.
// @Summary Finish a existing campaign.
// @Description Delete a existing campaign.
// @Produce json
// @Param id query string true "The campaign's id"
// @Success 200
// @Router /v1.0/campaigncalls/{id} [delete]
func campaigncallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "campaigncallsIDDELETE",
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
	log = log.WithField("campaign_id", id)
	log.Debug("Executing campaigncallsIDDELETE.")

	// delete an campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaigncallDelete(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not delete the campaigncall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

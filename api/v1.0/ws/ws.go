package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// wsGET handles POST /queues request.
// It creates a new queue.
// @Summary Create a new queue.
// @Description create a new queue
// @Produce  json
// @Success 200 {object} queue.WebhookMessage
// @Router /v1.0/ws [get]
func wsGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "wsGET",
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("Received websocket request.")

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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	if err := serviceHandler.WebsockCreate(c.Request.Context(), &u, c.Writer, c.Request); err != nil {
		log.Errorf("Could not handler the websocket correctly. err: %v", err)
	}
}

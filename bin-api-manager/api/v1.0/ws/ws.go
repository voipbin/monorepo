package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// wsGET handles POST /queues request.
// It creates a new queue.
//	@Summary		Create a new queue.
//	@Description	create a new queue
//	@Produce		json
//	@Success		200	{object}	queue.WebhookMessage
//	@Router			/v1.0/ws [get]
func wsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "wsGET",
		"request_address": c.ClientIP,
	})
	log.Debugf("Received websocket request.")

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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	if err := serviceHandler.WebsockCreate(c.Request.Context(), &a, c.Writer, c.Request); err != nil {
		log.Errorf("Could not handler the websocket correctly. err: %v", err)
	}
}

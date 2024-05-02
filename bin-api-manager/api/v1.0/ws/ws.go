package ws

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// wsGET handles POST /queues request.
// It creates a new queue.
//
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

	if err := serviceHandler.WebsockCreate(context.Background(), &a, c.Writer, c.Request); err != nil {
		log.Errorf("Could not handler the websocket correctly. err: %v", err)
	}
}

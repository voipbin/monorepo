package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// wsGET handles POST service_agents/ws request.
// It creates a new websocket connection.
//
//	@Summary		Create a new websocket connection.
//	@Description	create a new websocket connection
//	@Produce		json
//	@Success		200	{object}	queue.WebhookMessage
//	@Router			/v1.0/service_agents/ws [get]
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

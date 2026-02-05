package server

import (
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetTimelinesCallCallIdSipMessages(c *gin.Context, callId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesCallCallIdSipMessages",
		"request_address": c.ClientIP(),
		"call_id":         callId,
	})

	// Get agent from context
	tmp, exists := c.Get("agent")
	if !exists {
		log.Error("Could not find agent info")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "authentication required"})
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("customer_id", a.CustomerID)

	// Convert openapi_types.UUID to uuid.UUID
	callUUID, err := uuid.FromString(callId.String())
	if err != nil {
		log.Infof("Invalid call id: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid call id"})
		return
	}

	res, err := h.serviceHandler.TimelineSIPMessagesGet(c.Request.Context(), &a, callUUID)
	if err != nil {
		log.Infof("Could not get SIP messages: %v", err)
		switch err.Error() {
		case "call not found":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case "permission denied":
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": err.Error()})
		case "no SIP data available for this call":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		default:
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "upstream service unavailable"})
		}
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) GetTimelinesCallCallIdPcap(c *gin.Context, callId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesCallCallIdPcap",
		"request_address": c.ClientIP(),
		"call_id":         callId,
	})

	// Get agent from context
	tmp, exists := c.Get("agent")
	if !exists {
		log.Error("Could not find agent info")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "authentication required"})
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("customer_id", a.CustomerID)

	// Convert openapi_types.UUID to uuid.UUID
	callUUID, err := uuid.FromString(callId.String())
	if err != nil {
		log.Infof("Invalid call id: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid call id"})
		return
	}

	pcapData, err := h.serviceHandler.TimelineSIPPcapGet(c.Request.Context(), &a, callUUID)
	if err != nil {
		log.Infof("Could not get PCAP data: %v", err)
		switch err.Error() {
		case "call not found":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case "permission denied":
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": err.Error()})
		case "no SIP data available for this call":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		default:
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "upstream service unavailable"})
		}
		return
	}

	c.Data(http.StatusOK, "application/vnd.tcpdump.pcap", pcapData)
}

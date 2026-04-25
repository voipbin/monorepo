package server

import (
	"fmt"
	"net/http"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetTimelinesCallsCallIdSipAnalysis(c *gin.Context, callId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesCallsCallIdSipAnalysis",
		"request_address": c.ClientIP(),
		"call_id":         callId,
	})
	log.Info("Handler called - SIP analysis request received")

	// Get agent from context
	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	// Convert openapi_types.UUID to uuid.UUID
	callUUID, err := uuid.FromString(callId.String())
	if err != nil {
		log.Infof("Invalid call id: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_CALL_ID", "The provided call_id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.TimelineSIPAnalysisGet(c.Request.Context(), a, callUUID)
	if err != nil {
		log.Infof("Could not get SIP analysis: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *server) GetTimelinesCallsCallIdPcap(c *gin.Context, callId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesCallsCallIdPcap",
		"request_address": c.ClientIP(),
		"call_id":         callId,
	})
	log.Info("Handler called - PCAP request received")

	// Get agent from context
	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	// Convert openapi_types.UUID to uuid.UUID
	callUUID, err := uuid.FromString(callId.String())
	if err != nil {
		log.Infof("Invalid call id: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_CALL_ID", "The provided call_id is not a valid UUID.").Wrap(err))
		return
	}

	pcapData, err := h.serviceHandler.TimelineSIPPcapGet(c.Request.Context(), a, callUUID)
	if err != nil {
		log.Infof("Could not get PCAP data: %v", err)
		abortWithServiceError(c, err)
		return
	}

	filename := fmt.Sprintf("call-%s.pcap", callId.String())
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.tcpdump.pcap", pcapData)
}

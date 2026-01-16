package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetBillings(c *gin.Context, params openapi_server.GetBillingsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillings",
		"request_address": c.ClientIP,
	})

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

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.BillingList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get billing accounts info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetBillingsBillingId(c *gin.Context, billingId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingsBillingId",
		"request_address": c.ClientIP(),
	})

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

	// Convert openapi_types.UUID to uuid.UUID
	billingID, err := uuid.FromString(billingId.String())
	if err != nil {
		log.Errorf("Invalid billing ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	billing, err := h.serviceHandler.BillingGet(c.Request.Context(), &a, billingID)
	if err != nil {
		logrus.Errorf("Could not get billing info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, billing)
}

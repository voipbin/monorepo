package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAvailableNumbers(c *gin.Context, params openapi_server.GetAvailableNumbersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAvailableNumbers",
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

	countryCode := params.CountryCode
	if countryCode == "" {
		log.Infof("Not acceptable country code. country_code: %s", countryCode)
		c.AbortWithStatus(400)
		return
	}

	tmps, err := h.serviceHandler.AvailableNumberGets(c.Request.Context(), &a, pageSize, countryCode)
	if err != nil {
		log.Errorf("Could not get available numbers. err: %v", err)
		c.AbortWithStatus(500)
		return
	}

	res := struct {
		Result []*nmavailablenumber.WebhookMessage `json:"result"`
	}{
		Result: tmps,
	}

	c.JSON(200, res)
}

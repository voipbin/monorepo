package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAvailableNumbers(c *gin.Context, params openapi_server.GetAvailableNumbersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAvailableNumbers",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	numType := ""
	if params.Type != nil {
		numType = string(*params.Type)
	}

	countryCode := ""
	if params.CountryCode != nil {
		countryCode = *params.CountryCode
	}

	// country_code is required for non-virtual number queries
	if numType != "virtual" && countryCode == "" {
		log.Infof("Not acceptable country code. country_code: %s", countryCode)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "country_code is required for non-virtual number queries."))
		return
	}

	tmps, err := h.serviceHandler.AvailableNumberList(c.Request.Context(), a, pageSize, countryCode, numType)
	if err != nil {
		log.Errorf("Could not get available numbers. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	res := struct {
		Result []*nmavailablenumber.WebhookMessage `json:"result"`
	}{
		Result: tmps,
	}

	c.JSON(200, res)
}

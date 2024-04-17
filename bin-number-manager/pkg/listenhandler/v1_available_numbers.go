package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"
)

// processV1AvailableNumbersGet handles GET /v1/avaliable_numbers request
func (h *listenHandler) processV1AvailableNumbersGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AvailableNumbersGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint(tmpSize)

	if pageSize <= 0 {
		pageSize = 10
	}

	// get country_code
	countryCode := u.Query().Get("country_code")

	log.Debug("processV1AvailableNumbersGet. Getting available nubmers.")
	numbers, err := h.numberHandler.GetAvailableNumbers(countryCode, pageSize)
	if err != nil {
		log.Debugf("Could not get available numbers. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numbers)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numbers, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

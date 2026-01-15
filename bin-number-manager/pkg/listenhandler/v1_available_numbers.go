package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1AvailableNumbersGet handles GET /v1/avaliable_numbers request
func (h *listenHandler) processV1AvailableNumbersGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AvailableNumbersGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params from URI
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint(tmpSize)

	if pageSize <= 0 {
		pageSize = 10
	}

	// Parse filters from request data (body)
	var filters map[string]any
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &filters); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, fmt.Errorf("could not unmarshal filters: %w", err)
		}
	}

	log.WithFields(logrus.Fields{
		"filters":          filters,
		"filters_raw_data": string(m.Data),
	}).Debug("processV1AvailableNumbersGet: Parsed filters from request body")

	// Extract country_code from filters
	countryCode := ""
	if cc, ok := filters["country_code"]; ok {
		if ccStr, ok := cc.(string); ok {
			countryCode = ccStr
		}
	}

	log.Debug("processV1AvailableNumbersGet. Getting available numbers.")
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

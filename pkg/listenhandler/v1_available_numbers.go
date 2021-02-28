package listenhandler

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1AvailableNumbersGet handles GET /v1/avaliable_numbers request
func (h *listenHandler) processV1AvailableNumbersGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint(tmpSize)
	pageToken := u.Query().Get(PageToken)

	if pageSize <= 0 {
		pageSize = 10
	}

	// get user_id
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	// get country_code
	countryCode := u.Query().Get("country_code")

	log := logrus.WithFields(logrus.Fields{
		"user":         userID,
		"size":         pageSize,
		"token":        pageToken,
		"country_code": countryCode,
	})

	log.Debug("Getting available nubmers.")
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
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

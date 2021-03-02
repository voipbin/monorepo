package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/nmnumber"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// NMAvailableNumbersGet sends a request to number-manager
// to getting a list of available numbers.
func (r *requestHandler) NMAvailableNumbersGet(userID uint64, pageSize uint64, countryCode string) ([]nmnumber.AvailableNumber, error) {
	uri := fmt.Sprintf("/v1/available_numbers?page_size=%d&user_id=%d&country_code=%s", pageSize, userID, countryCode)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceStorageRecording, 15, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data []nmnumber.AvailableNumber
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return data, nil
}

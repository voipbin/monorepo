package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// NumberV1AvailableNumberGets sends a request to number-manager
// to getting a list of available numbers.
func (r *requestHandler) NumberV1AvailableNumberGets(ctx context.Context, customerID uuid.UUID, pageSize uint64, countryCode string) ([]nmavailablenumber.AvailableNumber, error) {
	uri := fmt.Sprintf("/v1/available_numbers?page_size=%d&customer_id=%s&country_code=%s", pageSize, customerID, countryCode)

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberAvailableNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data []nmavailablenumber.AvailableNumber
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return data, nil
}

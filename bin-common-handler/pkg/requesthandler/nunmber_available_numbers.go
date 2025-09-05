package requesthandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/gofrs/uuid"
)

// NumberV1AvailableNumberGets sends a request to number-manager
// to getting a list of available numbers.
func (r *requestHandler) NumberV1AvailableNumberGets(ctx context.Context, customerID uuid.UUID, pageSize uint64, countryCode string) ([]nmavailablenumber.AvailableNumber, error) {
	uri := fmt.Sprintf("/v1/available_numbers?page_size=%d&customer_id=%s&country_code=%s", pageSize, customerID, countryCode)

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodGet, "number/available-number", 15000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []nmavailablenumber.AvailableNumber
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

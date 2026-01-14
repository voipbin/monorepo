package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/pkg/errors"
)

// NumberV1AvailableNumberGets sends a request to number-manager
// to getting a list of available numbers.
func (r *requestHandler) NumberV1AvailableNumberGets(ctx context.Context, pageSize uint64, filters map[string]any) ([]nmavailablenumber.AvailableNumber, error) {
	uri := fmt.Sprintf("/v1/available_numbers?page_size=%d", pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodGet, "number/available-number", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []nmavailablenumber.AvailableNumber
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

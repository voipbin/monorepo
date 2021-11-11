package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// NMV1NumbersNumberGet sends the /v1/numbers/<number> GET request to number-manager
func (r *requestHandler) NMV1NumbersNumberGet(ctx context.Context, num string) (*number.Number, error) {

	uri := fmt.Sprintf("/v1/numbers/%s", url.QueryEscape(num))

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	tmpRes := number.Number{}
	if err := json.Unmarshal([]byte(res.Data), &tmpRes); err != nil {
		return nil, err
	}

	return &tmpRes, nil
}

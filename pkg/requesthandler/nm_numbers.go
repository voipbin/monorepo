package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler/models/nmnumber"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// NMV1NumbersNumberGet sends the /v1/numbers/<number> GET request to number-manager
func (r *requestHandler) NMV1NumbersNumberGet(num string) (*nmnumber.Number, error) {

	uri := fmt.Sprintf("/v1/numbers/%s", url.QueryEscape(num))

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, requestTimeoutDefault, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	tmpRes := nmnumber.Number{}
	if err := json.Unmarshal([]byte(res.Data), &tmpRes); err != nil {
		return nil, err
	}

	return &tmpRes, nil
}

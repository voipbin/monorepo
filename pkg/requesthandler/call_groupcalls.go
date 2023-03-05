package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CallV1GroupcallCreate sends a request to call-manager
// to create groupcall.
// it returns created groupcall info if it succeed.
func (r *requestHandler) CallV1GroupcallCreate(
	ctx context.Context,
	customerID uuid.UUID,
	source commonaddress.Address,
	destinations []commonaddress.Address,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	ringMethod cmgroupcall.RingMethod,
	answerMethod cmgroupcall.AnswerMethod,
	connect bool,
) (*cmgroupcall.Groupcall, error) {
	uri := "/v1/groupcalls"

	reqData := &cmrequest.V1DataGroupcallsPost{
		CustomerID:   customerID,
		Source:       source,
		Destinations: destinations,
		FlowID:       flowID,
		MasterCallID: masterCallID,
		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
		Connect:      connect,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmgroupcall.Groupcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

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
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallGroupcalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1GroupcallGets sends a request to call-manager
// to getting a list of groupcall info.
// it returns detail list of groupcall info if it succeed.
func (r *requestHandler) CallV1GroupcallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallGroupcalls, 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cmgroupcall.Groupcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CallV1GroupcallGet sends a request to call-manager
// to getting a groupcall.
// it returns given groupcall id's groupcall if it succeed.
func (r *requestHandler) CallV1GroupcallGet(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls/%s", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallGroupcallsID, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1GroupcallDelete sends a request to call-manager
// to delete the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallDelete(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls/%s", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallGroupcallsID, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1GroupcallHangup sends a request to call-manager
// to hangup the groupcall.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallHangup(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls/%s/hangup", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallGroupcallsIDHangup, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1GroupcallDecreaseGroupcallCount sends a request to call-manager
// to decrease the groupcall's groupcall_count.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallDecreaseGroupcallCount(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls/%s/decrease_groupcall_count", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/decrease_groupcall_count", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1GroupcallUpdateAnswerGroupcallID sends a request to call-manager
// to update the answer_groupcall_id of the given groupcall.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallUpdateAnswerGroupcallID(ctx context.Context, groupcallID uuid.UUID, answerGroupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls/%s/answer_groupcall_id", groupcallID)

	reqData := struct {
		AnswerGroupcallID uuid.UUID `json:"answer_groupcall_id"`
	}{
		AnswerGroupcallID: answerGroupcallID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/answer_groupcall_id", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

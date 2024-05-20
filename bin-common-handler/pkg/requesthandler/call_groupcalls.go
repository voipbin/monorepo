package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// CallV1GroupcallCreate sends a request to call-manager
// to create groupcall.
// it returns created groupcall info if it succeed.
func (r *requestHandler) CallV1GroupcallCreate(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	source commonaddress.Address,
	destinations []commonaddress.Address,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	ringMethod cmgroupcall.RingMethod,
	answerMethod cmgroupcall.AnswerMethod,
) (*cmgroupcall.Groupcall, error) {
	uri := "/v1/groupcalls"

	reqData := &cmrequest.V1DataGroupcallsPost{
		ID:                id,
		CustomerID:        customerID,
		FlowID:            flowID,
		Source:            source,
		Destinations:      destinations,
		MasterCallID:      masterCallID,
		MasterGroupcallID: masterGroupcallID,
		RingMethod:        ringMethod,
		AnswerMethod:      answerMethod,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
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
func (r *requestHandler) CallV1GroupcallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmgroupcall.Groupcall, error) {
	uri := fmt.Sprintf("/v1/groupcalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, "call/groupcalls", 30000, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, "call/groupcalls/<groupcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, "call/groupcalls/<groupcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/hangup", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1GroupcallHangupOthers sends a request to call-manager
// to hangup the related calls and groupcalls except answer_call_id or answer_groupcall_id.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallHangupOthers(ctx context.Context, groupcallID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/groupcalls/%s/hangup_others", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/hangup_others", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1GroupcallHangupCall sends a request to call-manager
// to hangup the related calls.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallHangupCall(ctx context.Context, groupcallID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/groupcalls/%s/hangup_call", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/hangup_call", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1GroupcallHangupGroupcall sends a request to call-manager
// to hangup the related groupcalls.
// it returns error if something went wrong.
func (r *requestHandler) CallV1GroupcallHangupGroupcall(ctx context.Context, groupcallID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/groupcalls/%s/hangup_groupcall", groupcallID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/groupcalls/<groupcall-id>/hangup_groupcall", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

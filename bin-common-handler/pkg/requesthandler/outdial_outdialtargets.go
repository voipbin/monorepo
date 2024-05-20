package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"
	omrequest "monorepo/bin-outdial-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// OutdialV1OutdialtargetCreate creates a outdial target and returns created outdial target
// ctx: context
func (r *requestHandler) OutdialV1OutdialtargetCreate(
	ctx context.Context,
	outdialID uuid.UUID,
	name string,
	detail string,
	data string,
	destination0 *address.Address,
	destination1 *address.Address,
	destination2 *address.Address,
	destination3 *address.Address,
	destination4 *address.Address,
) (*omoutdialtarget.OutdialTarget, error) {

	uri := fmt.Sprintf("/v1/outdials/%s/targets", outdialID)

	reqData := &omrequest.V1DataOutdialsIDTargetsPost{
		Name:         name,
		Detail:       detail,
		Data:         data,
		Destination0: destination0,
		Destination1: destination1,
		Destination2: destination2,
		Destination3: destination3,
		Destination4: destination4,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPost, "outdial/outdial_targets", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialtargetGetsAvailable returns an available outdialtargets
// ctx: context
func (r *requestHandler) OutdialV1OutdialtargetGetsAvailable(
	ctx context.Context,
	outdialID uuid.UUID,
	tryCount0 int,
	tryCount1 int,
	tryCount2 int,
	tryCount3 int,
	tryCount4 int,
	limit int,
) ([]omoutdialtarget.OutdialTarget, error) {

	uri := fmt.Sprintf("/v1/outdials/%s/available?try_count_0=%d&try_count_1=%d&try_count_2=%d&try_count_3=%d&try_count_4=%d&limit=%d",
		outdialID,
		tryCount0,
		tryCount1,
		tryCount2,
		tryCount3,
		tryCount4,
		limit,
	)

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodGet, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res []omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// OutdialV1OutdialtargetDelete sends a request to outdial-manager
// to deleting the outdialtarget.
func (r *requestHandler) OutdialV1OutdialtargetDelete(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s", outdialtargetID)

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodDelete, "outdial/outdial_targets", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialtargetGetsByOutdialID returns an list of outdialtarget
// ctx: context
func (r *requestHandler) OutdialV1OutdialtargetGetsByOutdialID(ctx context.Context, outdialtargetID uuid.UUID, pageToken string, pageSize uint64) ([]omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdials/%s/targets?page_token=%s&page_size=%d", outdialtargetID, url.QueryEscape(pageToken), pageSize)

	res, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodGet, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// OutdialV1OutdialtargetGet returns an outdialtarget
// ctx: context
func (r *requestHandler) OutdialV1OutdialtargetGet(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error) {

	uri := fmt.Sprintf("/v1/outdialtargets/%s", outdialtargetID)

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodGet, "outdial/outdial_targets", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialtargetUpdateStatusProgressing sends a request to outdial-manager
// to update the outdial's status to the progressing.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OutdialV1OutdialtargetUpdateStatusProgressing(ctx context.Context, outdialtargetID uuid.UUID, destinationIndex int) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s/progressing", outdialtargetID)

	data := &omrequest.V1DataOutdialtargetsIDProgressingPost{
		DestinationIndex: destinationIndex,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPost, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialtargetUpdateStatus sends a request to outdial-manager
// to update the outdial's status.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OutdialV1OutdialtargetUpdateStatus(ctx context.Context, outdialtargetID uuid.UUID, status omoutdialtarget.Status) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s/status", outdialtargetID)

	if status == omoutdialtarget.StatusProgressing {
		return nil, fmt.Errorf("can not update status to progressing here. use OutdialV1OutdialtargetUpdateStatusProgressing")
	}

	data := &omrequest.V1DataOutdialtargetsIDStatusPut{
		Status: status,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPut, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	omrequest "gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// OMV1OutdialtargetCreate creates a outdial target and returns created outdial target
// ctx: context
func (r *requestHandler) OMV1OutdialtargetCreate(
	ctx context.Context,
	outdialID uuid.UUID,
	name string,
	detail string,
	data string,
	destination0 *cmaddress.Address,
	destination1 *cmaddress.Address,
	destination2 *cmaddress.Address,
	destination3 *cmaddress.Address,
	destination4 *cmaddress.Address,
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

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodPost, resourceOMOutdialTargets, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// OMV1OutdialtargetGetsAvailable returns an available outdialtargets
// ctx: context
func (r *requestHandler) OMV1OutdialtargetGetsAvailable(
	ctx context.Context,
	outdialID uuid.UUID,
	tryCount0 int,
	tryCount1 int,
	tryCount2 int,
	tryCount3 int,
	tryCount4 int,
	interval int,
	limit int,
) ([]omoutdialtarget.OutdialTarget, error) {

	uri := fmt.Sprintf("/v1/outdials/%s/available?try_count_0=%d&try_count_1=%d&try_count_2=%d&try_count_3=%d&try_count_4=%d&interval=%d&limit=%d",
		outdialID,
		tryCount0,
		tryCount1,
		tryCount2,
		tryCount3,
		tryCount4,
		interval,
		limit,
	)

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodGet, resourceOMOutdials, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// OMV1OutdialtargetDelete sends a request to outdial-manager
// to deleting the outdialtarget.
func (r *requestHandler) OMV1OutdialtargetDelete(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s", outdialtargetID)

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodDelete, resourceOMOutdialTargets, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// OMV1OutdialtargetGet returns an outdialtarget
// ctx: context
func (r *requestHandler) OMV1OutdialtargetGet(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error) {

	uri := fmt.Sprintf("/v1/outdialtargets/%s", outdialtargetID)

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodGet, resourceOMOutdialTargets, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// OMV1OutdialtargetUpdateStatusProgressing sends a request to outdial-manager
// to update the outdial's status to the progressing.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OMV1OutdialtargetUpdateStatusProgressing(ctx context.Context, outdialtargetID uuid.UUID, destinationIndex int) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s/progressing", outdialtargetID)

	data := &omrequest.V1DataOutdialtargetsIDProgressingPost{
		DestinationIndex: destinationIndex,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodPost, resourceOMOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// OMV1OutdialtargetUpdateStatus sends a request to outdial-manager
// to update the outdial's status.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OMV1OutdialtargetUpdateStatus(ctx context.Context, outdialtargetID uuid.UUID, status omoutdialtarget.Status) (*omoutdialtarget.OutdialTarget, error) {
	uri := fmt.Sprintf("/v1/outdialtargets/%s/status", outdialtargetID)

	if status == omoutdialtarget.StatusProgressing {
		return nil, fmt.Errorf("can not update status to progressing here. use OMV1OutdialtargetUpdateStatusProgressing")
	}

	data := &omrequest.V1DataOutdialtargetsIDStatusPut{
		Status: status,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(uri, rabbitmqhandler.RequestMethodPut, resourceOMOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

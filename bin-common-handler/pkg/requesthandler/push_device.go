package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	pmdevice "monorepo/bin-push-manager/models/device"
	pmrequest "monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// pushDeviceListPageToken is the page token of the push device list: a time after every record, so the
// first (and, with the per-agent device cap, only) page holds all devices of the agent.
const pushDeviceListPageToken = "9999-12-31T23:59:59.999999Z"

// pushDeviceListPageSize is the page size of the push device list. It is above the per-agent device cap.
const pushDeviceListPageSize = 100

// PushV1DeviceCreate sends a request to push-manager to register or refresh the push device of the agent.
// The device in the response has no token.
func (r *requestHandler) PushV1DeviceCreate(ctx context.Context, customerID, agentID uuid.UUID, platform pmdevice.Platform, token string) (*pmdevice.Device, error) {
	uri := "/v1/devices"

	data := &pmrequest.V1DataDevicesPost{
		CustomerID: customerID,
		AgentID:    agentID,
		Platform:   platform,
		Token:      token,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodPost, "push/devices", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res pmdevice.Device
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// PushV1DeviceGet sends a request to push-manager to get the push device.
func (r *requestHandler) PushV1DeviceGet(ctx context.Context, id uuid.UUID) (*pmdevice.Device, error) {
	uri := fmt.Sprintf("/v1/devices/%s", id)

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodGet, "push/devices/<device-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pmdevice.Device
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// PushV1DeviceDelete sends a request to push-manager to delete the push device.
func (r *requestHandler) PushV1DeviceDelete(ctx context.Context, id uuid.UUID) (*pmdevice.Device, error) {
	uri := fmt.Sprintf("/v1/devices/%s", id)

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodDelete, "push/devices/<device-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pmdevice.Device
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// PushV1DeviceList sends a request to push-manager to get the live push devices of the agent.
// An agent without devices gets an empty list.
func (r *requestHandler) PushV1DeviceList(ctx context.Context, customerID, agentID uuid.UUID) ([]pmdevice.Device, error) {
	uri := fmt.Sprintf("/v1/devices?page_token=%s&page_size=%d", pushDeviceListPageToken, pushDeviceListPageSize)

	filters := map[pmdevice.Field]any{
		pmdevice.FieldCustomerID: customerID,
		pmdevice.FieldAgentID:    agentID,
	}

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodGet, "push/devices", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []pmdevice.Device
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

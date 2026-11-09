package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	pmdevice "monorepo/bin-push-manager/models/device"
	pmnotification "monorepo/bin-push-manager/models/notification"
)

const pushQueue = "bin-manager.push-manager.request"

func Test_PushV1Wrappers(t *testing.T) {
	deviceID := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
	customerID := uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	agentID := uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")

	deviceWant := pmdevice.Device{
		Identity: identity.Identity{ID: deviceID, CustomerID: customerID},
		AgentID:  agentID,
		Platform: pmdevice.PlatformAndroid,
	}
	const deviceResponse = `{"id":"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18","customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","agent_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","platform":"android"}`

	cases := []rpcCase{
		{
			name:           "device_create",
			queue:          pushQueue,
			expectURI:      "/v1/devices",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","agent_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","platform":"android","token":"test-token"}`,
			responseData:   deviceResponse,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1DeviceCreate(context.Background(), customerID, agentID, pmdevice.PlatformAndroid, "test-token")
			},
			expectRes: &deviceWant,
		},
		{
			name:           "device_get",
			queue:          pushQueue,
			expectURI:      "/v1/devices/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   deviceResponse,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1DeviceGet(context.Background(), deviceID)
			},
			expectRes: &deviceWant,
		},
		{
			name:           "device_delete",
			queue:          pushQueue,
			expectURI:      "/v1/devices/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   deviceResponse,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1DeviceDelete(context.Background(), deviceID)
			},
			expectRes: &deviceWant,
		},
		{
			name:           "device_list",
			queue:          pushQueue,
			expectURI:      "/v1/devices?page_token=9999-12-31T23:59:59.999999Z&page_size=100",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"agent_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30"}`,
			responseData:   `[` + deviceResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1DeviceList(context.Background(), customerID, agentID)
			},
			expectRes: []pmdevice.Device{deviceWant},
		},
		{
			name:           "notification_send_all_devices",
			queue:          pushQueue,
			expectURI:      "/v1/notifications",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","agent_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","device_id":"00000000-0000-0000-0000-000000000000","kind":"call_cancelled","data":{"cancel_key":"k1"}}`,
			responseData:   `{"device_count":2,"sent_count":1}`,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1NotificationSend(context.Background(), customerID, agentID, uuid.Nil, pmnotification.KindCallCancelled, map[string]string{"cancel_key": "k1"})
			},
			expectRes: &pmnotification.SendResult{DeviceCount: 2, SentCount: 1},
		},
		{
			name:           "notification_send_one_device",
			queue:          pushQueue,
			expectURI:      "/v1/notifications",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","agent_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","device_id":"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18","kind":"call_incoming","data":{"join_address":"a1"}}`,
			responseData:   `{"device_count":1,"sent_count":1}`,
			call: func(h *requestHandler) (any, error) {
				return h.PushV1NotificationSend(context.Background(), customerID, agentID, deviceID, pmnotification.KindCallIncoming, map[string]string{"join_address": "a1"})
			},
			expectRes: &pmnotification.SendResult{DeviceCount: 1, SentCount: 1},
		},
	}

	runRPCCases(t, cases)
}

func Test_PushV1NotificationSend_timeout(t *testing.T) {
	// the notification request must wait longer than the per-device delivery limit of push-manager (5 s)
	if requestTimeoutPushNotificationSend <= 5000 {
		t.Errorf("Wrong match. expect: > 5000, got: %d", requestTimeoutPushNotificationSend)
	}
}

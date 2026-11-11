package listenhandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/notificationhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

var (
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testAgentID    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testDeviceID   = uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
)

const fakeToken = "fake-device-token-AbCdEf-0123456789"

func newTestHandler(mc *gomock.Controller) (*listenHandler, *devicehandler.MockDeviceHandler, *notificationhandler.MockNotificationHandler) {
	mockDevice := devicehandler.NewMockDeviceHandler(mc)
	mockNotification := notificationhandler.NewMockNotificationHandler(mc)

	return &listenHandler{
		sockHandler:         sockhandler.NewMockSockHandler(mc),
		deviceHandler:       mockDevice,
		notificationHandler: mockNotification,
		settingsHandler:     settingshandler.NewMockSettingsHandler(mc),
	}, mockDevice, mockNotification
}

func testDevice() *device.Device {
	return &device.Device{
		Identity: commonidentity.Identity{ID: testDeviceID, CustomerID: testCustomerID},
		AgentID:  testAgentID,
		Platform: device.PlatformAndroid,
		Token:    fakeToken,
	}
}

func Test_processRequest_devices(t *testing.T) {
	uri := "/v1/devices/" + testDeviceID.String()
	notFound := cerrors.NotFound(commonoutline.ServiceNamePushManager, "DEVICE_NOT_FOUND", "x")

	tests := []struct {
		name         string
		request      *sock.Request
		setup        func(m *devicehandler.MockDeviceHandler)
		expectStatus int
		expectInBody string
	}{
		{
			"post registers the device",
			&sock.Request{URI: "/v1/devices", Method: sock.RequestMethodPost, DataType: "application/json",
				Data: []byte(`{"customer_id":"` + testCustomerID.String() + `","agent_id":"` + testAgentID.String() + `","platform":"android","token":"` + fakeToken + `"}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Register(gomock.Any(), testCustomerID, testAgentID, device.PlatformAndroid, fakeToken).Return(testDevice(), nil)
			},
			200, `"platform":"android"`,
		},
		{
			"post with a typed validation error is a 400",
			&sock.Request{URI: "/v1/devices", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"platform":"ios"}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Register(gomock.Any(), uuid.Nil, uuid.Nil, device.Platform("ios"), "").Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_PLATFORM", "x"))
			},
			400, "INVALID_PLATFORM",
		},
		{
			"post with a broken body is a 400",
			&sock.Request{URI: "/v1/devices", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{`)},
			func(m *devicehandler.MockDeviceHandler) {},
			400, "",
		},
		{
			"post with an internal error is a 500",
			&sock.Request{URI: "/v1/devices", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Register(gomock.Any(), uuid.Nil, uuid.Nil, device.Platform(""), "").Return(nil, fmt.Errorf("boom"))
			},
			500, "",
		},
		{
			"get",
			&sock.Request{URI: uri, Method: sock.RequestMethodGet},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Get(gomock.Any(), testDeviceID).Return(testDevice(), nil)
			},
			200, testDeviceID.String(),
		},
		{
			"get not found",
			&sock.Request{URI: uri, Method: sock.RequestMethodGet},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Get(gomock.Any(), testDeviceID).Return(nil, notFound)
			},
			404, "",
		},
		{
			"delete",
			&sock.Request{URI: uri, Method: sock.RequestMethodDelete},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Delete(gomock.Any(), testDeviceID).Return(testDevice(), nil)
			},
			200, testDeviceID.String(),
		},
		{
			"delete not found through the legacy error",
			&sock.Request{URI: uri, Method: sock.RequestMethodDelete},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().Delete(gomock.Any(), testDeviceID).Return(nil, dbhandler.ErrNotFound)
			},
			404, "",
		},
		{
			"list with filters and paging",
			&sock.Request{URI: "/v1/devices?page_size=100&page_token=9999-12-31T23:59:59.999999Z", Method: sock.RequestMethodGet, DataType: "application/json",
				Data: []byte(`{"customer_id":"` + testCustomerID.String() + `","agent_id":"` + testAgentID.String() + `"}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().List(gomock.Any(), uint64(100), "9999-12-31T23:59:59.999999Z", map[device.Field]any{
					device.FieldCustomerID: testCustomerID,
					device.FieldAgentID:    testAgentID,
				}).Return([]*device.Device{testDevice()}, nil)
			},
			200, testDeviceID.String(),
		},
		{
			"a token filter is not a filter: it is dropped, so the token can not be searched",
			&sock.Request{URI: "/v1/devices?page_size=10&page_token=t", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{"token":"x"}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().List(gomock.Any(), uint64(10), "t", map[device.Field]any{}).Return([]*device.Device{}, nil)
			},
			200, "",
		},
		{
			"list with a broken body is a 400",
			&sock.Request{URI: "/v1/devices?page_size=10&page_token=t", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{`)},
			func(m *devicehandler.MockDeviceHandler) {},
			400, "",
		},
		{
			"list error is a 500",
			&sock.Request{URI: "/v1/devices?page_size=10&page_token=t", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			func(m *devicehandler.MockDeviceHandler) {
				m.EXPECT().List(gomock.Any(), uint64(10), "t", gomock.Any()).Return(nil, fmt.Errorf("boom"))
			},
			500, "",
		},
		{
			"a path after the device id is not a device route",
			&sock.Request{URI: uri + "/x", Method: sock.RequestMethodGet},
			func(m *devicehandler.MockDeviceHandler) {},
			404, "",
		},
		{
			"post on a path after /v1/devices is not the register route",
			&sock.Request{URI: "/v1/devices/x", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{}`)},
			func(m *devicehandler.MockDeviceHandler) {},
			404, "",
		},
		{
			"unknown path",
			&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			func(m *devicehandler.MockDeviceHandler) {},
			404, "",
		},
		{
			"known path with an unsupported method",
			&sock.Request{URI: uri, Method: sock.RequestMethodPut},
			func(m *devicehandler.MockDeviceHandler) {},
			404, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDevice, _ := newTestHandler(mc)
			tt.setup(mockDevice)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect: %d, got: %d, %s", tt.expectStatus, res.StatusCode, res.Data)
			}
			if tt.expectInBody != "" && !strings.Contains(string(res.Data), tt.expectInBody) {
				t.Errorf("Wrong match. expect the body to hold %q, got: %s", tt.expectInBody, res.Data)
			}
			// a response never holds the token
			if strings.Contains(string(res.Data), fakeToken) || strings.Contains(string(res.Data), `"token"`) {
				t.Errorf("Wrong match. the token must not be in the response. got: %s", res.Data)
			}
		})
	}
}

func Test_processRequest_notifications(t *testing.T) {
	body := func(deviceID uuid.UUID) []byte {
		return []byte(`{"customer_id":"` + testCustomerID.String() + `","agent_id":"` + testAgentID.String() + `","device_id":"` + deviceID.String() + `","kind":"call_incoming","data":{"join_address":"a1"}}`)
	}

	tests := []struct {
		name         string
		data         []byte
		setup        func(m *notificationhandler.MockNotificationHandler)
		expectStatus int
		expectBody   string
	}{
		{
			"send to one device",
			body(testDeviceID),
			func(m *notificationhandler.MockNotificationHandler) {
				m.EXPECT().Send(gomock.Any(), testCustomerID, testAgentID, testDeviceID, notification.KindCallIncoming, map[string]string{"join_address": "a1"}).
					Return(&notification.SendResult{DeviceCount: 1, SentCount: 1}, nil)
			},
			200, `{"device_count":1,"sent_count":1}`,
		},
		{
			"send to every device uses the nil device id",
			body(uuid.Nil),
			func(m *notificationhandler.MockNotificationHandler) {
				m.EXPECT().Send(gomock.Any(), testCustomerID, testAgentID, uuid.Nil, notification.KindCallIncoming, gomock.Any()).
					Return(&notification.SendResult{DeviceCount: 2, SentCount: 0}, nil)
			},
			200, `{"device_count":2,"sent_count":0}`,
		},
		{
			"typed validation error is a 400",
			body(uuid.Nil),
			func(m *notificationhandler.MockNotificationHandler) {
				m.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "x"))
			},
			400, "INVALID_KIND",
		},
		{
			"broken body is a 400",
			[]byte(`{`),
			func(m *notificationhandler.MockNotificationHandler) {},
			400, "",
		},
		{
			"internal error is a 500",
			body(uuid.Nil),
			func(m *notificationhandler.MockNotificationHandler) {
				m.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("boom"))
			},
			500, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockNotification := newTestHandler(mc)
			tt.setup(mockNotification)

			res, err := h.processRequest(&sock.Request{URI: "/v1/notifications", Method: sock.RequestMethodPost, DataType: "application/json", Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect: %d, got: %d, %s", tt.expectStatus, res.StatusCode, res.Data)
			}
			if tt.expectBody != "" && !strings.Contains(string(res.Data), tt.expectBody) {
				t.Errorf("Wrong match. expect the body to hold %q, got: %s", tt.expectBody, res.Data)
			}
		})
	}
}

func Test_processRequest_notificationsPathIsAnchored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no handler call is expected
	h, _, _ := newTestHandler(mc)

	res, err := h.processRequest(&sock.Request{URI: "/v1/notificationsX", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{}`)})
	if err != nil || res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: 404, got: %v, %v", res, err)
	}
}

func Test_errorResponse(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{"typed not found", cerrors.NotFound(commonoutline.ServiceNamePushManager, "X", "x"), http.StatusNotFound},
		{"typed invalid argument", cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "X", "x"), http.StatusBadRequest},
		{"legacy not found", dbhandler.ErrNotFound, http.StatusNotFound},
		{"wrapped legacy not found", fmt.Errorf("x: %w", dbhandler.ErrNotFound), http.StatusNotFound},
		{"unclassified", fmt.Errorf("x"), http.StatusInternalServerError},
		{"nil", nil, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorResponse(tt.err).StatusCode; got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

func Test_requestBodyIsNeverLogged(t *testing.T) {
	// the type has no token field in its json, but the request type must carry it: a regression guard on the
	// wire shape of the create request
	var req struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(`{"token":"`+fakeToken+`"}`), &req); err != nil || req.Token != fakeToken {
		t.Errorf("Wrong match. got: %v, %v", req, err)
	}
}

func Test_Run(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDevice := devicehandler.NewMockDeviceHandler(mc)
	mockNotification := notificationhandler.NewMockNotificationHandler(mc)
	h := NewListenHandler(mockSock, mockDevice, mockNotification, settingshandler.NewMockSettingsHandler(mc))

	mockSock.EXPECT().QueueCreate("q", "normal").Return(fmt.Errorf("boom"))
	if err := h.Run("q", "delay"); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

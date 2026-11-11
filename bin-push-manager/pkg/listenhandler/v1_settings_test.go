package listenhandler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/models/settings"
	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/notificationhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

var testChatID = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")

func newSettingsHandlerForTest(mc *gomock.Controller) (*listenHandler, *settingshandler.MockSettingsHandler) {
	mockSettings := settingshandler.NewMockSettingsHandler(mc)

	return &listenHandler{
		sockHandler:         sockhandler.NewMockSockHandler(mc),
		deviceHandler:       devicehandler.NewMockDeviceHandler(mc),
		notificationHandler: notificationhandler.NewMockNotificationHandler(mc),
		settingsHandler:     mockSettings,
	}, mockSettings
}

func testSettings() *settings.Settings {
	return settings.New([]notification.Kind{notification.KindCaseNote}, []uuid.UUID{testChatID})
}

const testSettingsBody = `"case_note":false`

func ownerQuery() string {
	return "customer_id=" + testCustomerID.String() + "&agent_id=" + testAgentID.String()
}

func Test_processRequest_settings(t *testing.T) {
	mutesURI := "/v1/mutes/" + testChatID.String() + "?" + ownerQuery()
	putBody := func(kinds string) []byte {
		return []byte(`{"customer_id":"` + testCustomerID.String() + `","agent_id":"` + testAgentID.String() + `","kinds":` + kinds + `}`)
	}

	tests := []struct {
		name         string
		request      *sock.Request
		setup        func(m *settingshandler.MockSettingsHandler)
		expectStatus int
		expectInBody string
	}{
		{
			"get",
			&sock.Request{URI: "/v1/settings?" + ownerQuery(), Method: sock.RequestMethodGet},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().Get(gomock.Any(), testCustomerID, testAgentID).Return(testSettings(), nil)
			},
			200, testSettingsBody,
		},
		{
			"get without the owner is passed on as the nil ids and the handler answers 400",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodGet},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().Get(gomock.Any(), uuid.Nil, uuid.Nil).Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_OWNER", "x"))
			},
			400, "INVALID_OWNER",
		},
		{
			"put switches the kinds",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodPut, DataType: "application/json", Data: putBody(`{"case_note":false,"chat_message":true}`)},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().Update(gomock.Any(), testCustomerID, testAgentID, map[notification.Kind]bool{
					notification.KindCaseNote:    false,
					notification.KindChatMessage: true,
				}).Return(testSettings(), nil)
			},
			200, testSettingsBody,
		},
		{
			"put with a bad kind is a 400 with the reason",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodPut, DataType: "application/json", Data: putBody(`{"call_incoming":false}`)},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().Update(gomock.Any(), testCustomerID, testAgentID, map[notification.Kind]bool{notification.KindCallIncoming: false}).
					Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "x"))
			},
			400, "INVALID_KIND",
		},
		{
			"put with a broken body is a 400",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{`)},
			func(m *settingshandler.MockSettingsHandler) {},
			400, "",
		},
		{
			"put with a non boolean value is a 400",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodPut, DataType: "application/json", Data: putBody(`{"case_note":"no"}`)},
			func(m *settingshandler.MockSettingsHandler) {},
			400, "",
		},
		{
			"put with a failure of the handler is a 500",
			&sock.Request{URI: "/v1/settings", Method: sock.RequestMethodPut, DataType: "application/json", Data: putBody(`{}`)},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().Update(gomock.Any(), testCustomerID, testAgentID, gomock.Any()).Return(nil, fmt.Errorf("boom"))
			},
			500, "",
		},
		{
			"mute add",
			&sock.Request{URI: mutesURI, Method: sock.RequestMethodPut},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().MuteAdd(gomock.Any(), testCustomerID, testAgentID, testChatID).Return(testSettings(), nil)
			},
			200, testChatID.String(),
		},
		{
			"mute remove",
			&sock.Request{URI: mutesURI, Method: sock.RequestMethodDelete},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().MuteRemove(gomock.Any(), testCustomerID, testAgentID, testChatID).Return(testSettings(), nil)
			},
			200, testChatID.String(),
		},
		{
			"mute without a query string reaches the handler with the nil owner",
			&sock.Request{URI: "/v1/mutes/" + testChatID.String(), Method: sock.RequestMethodDelete},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().MuteRemove(gomock.Any(), uuid.Nil, uuid.Nil, testChatID).Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_OWNER", "x"))
			},
			400, "INVALID_OWNER",
		},
		{
			"mute add failure is a 500",
			&sock.Request{URI: mutesURI, Method: sock.RequestMethodPut},
			func(m *settingshandler.MockSettingsHandler) {
				m.EXPECT().MuteAdd(gomock.Any(), testCustomerID, testAgentID, testChatID).Return(nil, fmt.Errorf("boom"))
			},
			500, "",
		},
		{
			"a chat id that is not a uuid is not a mute route",
			&sock.Request{URI: "/v1/mutes/not-a-uuid?" + ownerQuery(), Method: sock.RequestMethodPut},
			func(m *settingshandler.MockSettingsHandler) {},
			404, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockSettings := newSettingsHandlerForTest(mc)
			tt.setup(mockSettings)

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
		})
	}
}

// The settings and mutes routes are anchored at both ends, and the older unanchored device routes do not
// capture them. No handler call is expected in any of these.
func Test_processRequest_settingsRoutesAreAnchored(t *testing.T) {
	chat := testChatID.String()

	tests := []struct {
		name   string
		uri    string
		method sock.RequestMethod
	}{
		{"settings with a suffix", "/v1/settingsX", sock.RequestMethodGet},
		{"settings with a prefix", "/x/v1/settings", sock.RequestMethodGet},
		{"settings with a path after it", "/v1/settings/x", sock.RequestMethodPut},
		{"settings delete is not a route", "/v1/settings", sock.RequestMethodDelete},
		{"settings post is not a route", "/v1/settings", sock.RequestMethodPost},
		{"mute with a prefix", "/x/v1/mutes/" + chat, sock.RequestMethodPut},
		{"mute with a path after the id", "/v1/mutes/" + chat + "/x", sock.RequestMethodPut},
		{"mute get is not a route", "/v1/mutes/" + chat, sock.RequestMethodGet},
		{"mute without an id", "/v1/mutes", sock.RequestMethodPut},
		{"mute with an uppercase id", "/v1/mutes/" + strings.ToUpper(chat), sock.RequestMethodPut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newSettingsHandlerForTest(mc)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: []byte(`{}`)})
			if err != nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: 404, got: %v, %v", res, err)
			}
		})
	}
}

// A query string that mentions another route does not change the route: the route is decided by the path.
func Test_processRequest_settingsQueryDoesNotHijackTheRoute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockSettings := newSettingsHandlerForTest(mc)
	mockSettings.EXPECT().MuteAdd(gomock.Any(), testCustomerID, testAgentID, testChatID).Return(testSettings(), nil)

	uri := "/v1/mutes/" + testChatID.String() + "?" + ownerQuery() + "&x=/v1/devices?a=b"
	res, err := h.processRequest(&sock.Request{URI: uri, Method: sock.RequestMethodPut})
	if err != nil || res.StatusCode != 200 {
		t.Errorf("Wrong match. expect: 200, got: %v, %v", res, err)
	}
}

package server

import (
	"net/http"
	"strings"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	pmnotification "monorepo/bin-push-manager/models/notification"
	pmsettings "monorepo/bin-push-manager/models/settings"

	"github.com/gofrs/uuid"
)

var pushSettingsChatID = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")

func pushSettingsResult() *pmsettings.Settings {
	return pmsettings.New([]pmnotification.Kind{pmnotification.KindCaseNote}, []uuid.UUID{pushSettingsChatID})
}

const pushSettingsExpectBody = `{"kinds":{"case_assigned":true,"case_note":false,"chat_mentioned":true,"chat_message":true,"conversation_message":true},"muted_chat_ids":["5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11"]}`

func Test_serviceAgentsMePushSettingsGET(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/me/push_settings", nil)
		mockSvc.EXPECT().ServiceAgentPushSettingsGet(req.Context(), pushSrvAgent).Return(pushSettingsResult(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != pushSettingsExpectBody {
			t.Errorf("Wrong match. expect: 200 %s\ngot: %d %s", pushSettingsExpectBody, w.Code, w.Body.String())
		}
	})

	t.Run("an empty muted list is a json list", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/me/push_settings", nil)
		mockSvc.EXPECT().ServiceAgentPushSettingsGet(req.Context(), pushSrvAgent).Return(pmsettings.New(nil, nil), nil)

		r.ServeHTTP(w, req)
		if !strings.Contains(w.Body.String(), `"muted_chat_ids":[]`) {
			t.Errorf("Wrong match. got: %s", w.Body.String())
		}
	})

	t.Run("no auth identity maps to 401", func(t *testing.T) {
		_, r, w := pushSrvNew(t, nil)

		req, _ := http.NewRequest("GET", "/service_agents/me/push_settings", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d", w.Code)
		}
	})

	t.Run("permission denied maps to 403", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/me/push_settings", nil)
		mockSvc.EXPECT().ServiceAgentPushSettingsGet(req.Context(), pushSrvAgent).Return(nil, serviceerrors.ErrPermissionDenied)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Wrong match. expect: 403, got: %d", w.Code)
		}
	})
}

func Test_serviceAgentsMePushSettingsPUT(t *testing.T) {
	t.Run("ok converts the keys to kinds", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(`{"kinds":{"case_note":false,"chat_message":true}}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushSettingsUpdate(req.Context(), pushSrvAgent, map[pmnotification.Kind]bool{
			pmnotification.KindCaseNote:    false,
			pmnotification.KindChatMessage: true,
		}).Return(pushSettingsResult(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != pushSettingsExpectBody {
			t.Errorf("Wrong match. expect: 200 %s\ngot: %d %s", pushSettingsExpectBody, w.Code, w.Body.String())
		}
	})

	t.Run("empty kinds are allowed and change nothing", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(`{"kinds":{}}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushSettingsUpdate(req.Context(), pushSrvAgent, map[pmnotification.Kind]bool{}).Return(pushSettingsResult(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Wrong match. expect: 200, got: %d, %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing kinds maps to 400", func(t *testing.T) {
		_, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid json and a non boolean value map to 400", func(t *testing.T) {
		for _, body := range []string{`{"kinds":`, `{"kinds":{"case_note":"no"}}`} {
			_, r, w := pushSrvNew(t, pushSrvAgent)

			req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. body: %s, expect: 400, got: %d", body, w.Code)
			}
		}
	})

	// the typed 400 of push-manager reaches the client as a 400 with its reason
	t.Run("a kind that can not be switched maps to 400 with the reason", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(`{"kinds":{"call_incoming":false}}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushSettingsUpdate(req.Context(), pushSrvAgent, map[pmnotification.Kind]bool{"call_incoming": false}).
			Return(nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "The notification kind can not be changed."))

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_KIND") {
			t.Errorf("Wrong match. expect: 400 INVALID_KIND, got: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth identity maps to 401", func(t *testing.T) {
		_, r, w := pushSrvNew(t, nil)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_settings", strings.NewReader(`{"kinds":{}}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d", w.Code)
		}
	})
}

func Test_serviceAgentsMePushMutes(t *testing.T) {
	uri := "/service_agents/me/push_mutes/" + pushSettingsChatID.String()

	t.Run("put ok", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", uri, nil)
		mockSvc.EXPECT().ServiceAgentPushMuteAdd(req.Context(), pushSrvAgent, pushSettingsChatID).Return(pushSettingsResult(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != pushSettingsExpectBody {
			t.Errorf("Wrong match. expect: 200 %s\ngot: %d %s", pushSettingsExpectBody, w.Code, w.Body.String())
		}
	})

	t.Run("put of a chat that is not found maps to 404", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", uri, nil)
		mockSvc.EXPECT().ServiceAgentPushMuteAdd(req.Context(), pushSrvAgent, pushSettingsChatID).
			Return(nil, cerrors.NotFound(commonoutline.ServiceNameTalkManager, "CHAT_NOT_FOUND", "The chat was not found."))

		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "CHAT_NOT_FOUND") {
			t.Errorf("Wrong match. expect: 404 CHAT_NOT_FOUND, got: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete ok", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("DELETE", uri, nil)
		mockSvc.EXPECT().ServiceAgentPushMuteRemove(req.Context(), pushSrvAgent, pushSettingsChatID).Return(pushSettingsResult(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != pushSettingsExpectBody {
			t.Errorf("Wrong match. expect: 200 %s\ngot: %d %s", pushSettingsExpectBody, w.Code, w.Body.String())
		}
	})

	t.Run("a chat id that is not a uuid maps to 400", func(t *testing.T) {
		_, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/me/push_mutes/not-a-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth identity maps to 401", func(t *testing.T) {
		for _, method := range []string{"PUT", "DELETE"} {
			_, r, w := pushSrvNew(t, nil)

			req, _ := http.NewRequest(method, uri, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Wrong match. method: %s, expect: 401, got: %d", method, w.Code)
			}
		}
	})

	t.Run("permission denied maps to 403", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("DELETE", uri, nil)
		mockSvc.EXPECT().ServiceAgentPushMuteRemove(req.Context(), pushSrvAgent, pushSettingsChatID).Return(nil, serviceerrors.ErrPermissionDenied)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Wrong match. expect: 403, got: %d", w.Code)
		}
	})
}

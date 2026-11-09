package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	pmdevice "monorepo/bin-push-manager/models/device"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	pushSrvDeviceID = uuid.FromStringOrNil("3f6c1d2e-8a4b-4c7d-9e10-5b2a7c8d9e01")
	pushSrvAgent    = auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
)

func pushSrvNew(t *testing.T, agent *auth.AuthIdentity) (*servicehandler.MockServiceHandler, *gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	if agent != nil {
		r.Use(func(c *gin.Context) { c.Set("auth_identity", agent) })
	}
	openapi_server.RegisterHandlers(r, h)

	return mockSvc, r, w
}

func pushSrvMessage() *pmdevice.WebhookMessage {
	tm := time.Date(2026, 11, 9, 9, 15, 30, 123456000, time.UTC)
	return &pmdevice.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         pushSrvDeviceID,
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		AgentID:  uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
		Platform: pmdevice.PlatformAndroid,
		TMCreate: &tm,
		TMUpdate: &tm,
	}
}

const pushSrvExpectBody = `{"id":"3f6c1d2e-8a4b-4c7d-9e10-5b2a7c8d9e01","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","agent_id":"cdb5213a-8003-11ec-84ca-9fa226fcda9f","platform":"android","tm_create":"2026-11-09T09:15:30.123456Z","tm_update":"2026-11-09T09:15:30.123456Z"}`

func Test_serviceAgentsPushDevicesPOST(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("POST", "/service_agents/push_devices", strings.NewReader(`{"platform":"android","token":"fake-token"}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushDeviceCreate(req.Context(), pushSrvAgent, pmdevice.PlatformAndroid, "fake-token").Return(pushSrvMessage(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != pushSrvExpectBody {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", pushSrvExpectBody, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "fake-token") {
			t.Errorf("The response must not contain the token. got: %s", w.Body.String())
		}
	})

	t.Run("no auth identity maps to 401", func(t *testing.T) {
		_, r, w := pushSrvNew(t, nil)

		req, _ := http.NewRequest("POST", "/service_agents/push_devices", strings.NewReader(`{"platform":"android","token":"fake-token"}`))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid json maps to 400", func(t *testing.T) {
		_, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("POST", "/service_agents/push_devices", strings.NewReader(`{"platform":`))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid argument maps to 400", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("POST", "/service_agents/push_devices", strings.NewReader(`{"platform":"ios","token":"fake-token"}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushDeviceCreate(req.Context(), pushSrvAgent, pmdevice.Platform("ios"), "fake-token").Return(nil, serviceerrors.ErrInvalidArgument)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied maps to 403", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("POST", "/service_agents/push_devices", strings.NewReader(`{"platform":"android","token":"fake-token"}`))
		req.Header.Set("Content-Type", "application/json")
		mockSvc.EXPECT().ServiceAgentPushDeviceCreate(req.Context(), pushSrvAgent, pmdevice.PlatformAndroid, "fake-token").Return(nil, serviceerrors.ErrPermissionDenied)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Wrong match. expect: 403, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func Test_serviceAgentsPushDevicesIDDELETE(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/push_devices/"+pushSrvDeviceID.String(), nil)
		mockSvc.EXPECT().ServiceAgentPushDeviceDelete(req.Context(), pushSrvAgent, pushSrvDeviceID).Return(pushSrvMessage(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != pushSrvExpectBody {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", pushSrvExpectBody, w.Body.String())
		}
	})

	t.Run("no auth identity maps to 401", func(t *testing.T) {
		_, r, w := pushSrvNew(t, nil)

		req, _ := http.NewRequest("DELETE", "/service_agents/push_devices/"+pushSrvDeviceID.String(), nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id maps to 400", func(t *testing.T) {
		_, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/push_devices/not-a-uuid", nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mockSvc, r, w := pushSrvNew(t, pushSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/push_devices/"+pushSrvDeviceID.String(), nil)
		mockSvc.EXPECT().ServiceAgentPushDeviceDelete(req.Context(), pushSrvAgent, pushSrvDeviceID).Return(nil, serviceerrors.ErrNotFound)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

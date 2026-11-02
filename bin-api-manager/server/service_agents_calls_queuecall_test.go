package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_serviceAgentsCallsIDQueuecallGET(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	callID := uuid.FromStringOrNil("0b2decd0-3bb5-11ef-b2bb-bb18783b3f41")
	qcID := "3c8e1f20-7a4b-4d6e-9b1a-5f2c8d0e7a31"
	queueID := "7d2a9f10-3b5c-4e8a-a6d1-1c4e7b9f2a05"
	name := "Support"
	status := openapi_server.QueueManagerQueuecallStatus("service")

	tests := []struct {
		name string

		reqPath string

		callService  bool
		responseRes  *openapi_server.QueueManagerQueuecallSummary
		responseErr  error
		expectStatus int
		expectBody   string
	}{
		{
			name:         "normal",
			reqPath:      "/service_agents/calls/" + callID.String() + "/queuecall",
			callService:  true,
			responseRes:  &openapi_server.QueueManagerQueuecallSummary{Id: &qcID, QueueId: &queueID, QueueName: &name, Status: &status, TmCreate: timePtr("2026-11-02T09:10:00.000000Z")},
			expectStatus: http.StatusOK,
			expectBody:   `{"id":"` + qcID + `","queue_id":"` + queueID + `","queue_name":"Support","status":"service","tm_create":"2026-11-02T09:10:00Z"}`,
		},
		{
			name:         "not found",
			reqPath:      "/service_agents/calls/" + callID.String() + "/queuecall",
			callService:  true,
			responseErr:  serviceerrors.ErrNotFound,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "permission denied",
			reqPath:      "/service_agents/calls/" + callID.String() + "/queuecall",
			callService:  true,
			responseErr:  serviceerrors.ErrPermissionDenied,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "backend failure",
			reqPath:      "/service_agents/calls/" + callID.String() + "/queuecall",
			callService:  true,
			responseErr:  errors.New("rpc failed"),
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "malformed id is rejected before the service",
			reqPath:      "/service_agents/calls/not-a-uuid/queuecall",
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqPath, nil)
			if tt.callService {
				mockSvc.EXPECT().ServiceAgentCallQueuecallGet(req.Context(), agent, callID).Return(tt.responseRes, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
			if tt.expectBody != "" && w.Body.String() != tt.expectBody {
				t.Errorf("Wrong body.\nexpect: %s\ngot: %s", tt.expectBody, w.Body.String())
			}
		})
	}
}

func Test_serviceAgentsCallsIDQueuecallGET_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/service_agents/calls/0b2decd0-3bb5-11ef-b2bb-bb18783b3f41/queuecall", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong status. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
	}
}

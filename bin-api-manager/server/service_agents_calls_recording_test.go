package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_serviceAgentsCallsIDRecordingStartPOST(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	callID := uuid.FromStringOrNil("0b2decd0-3bb5-11ef-b2bb-bb18783b3f41")
	wav := cmrecording.FormatWAV
	ten := 10
	key := "#"
	dur := 3600
	response := &cmcall.WebhookMessage{Identity: commonidentity.Identity{ID: callID}}

	tests := []struct {
		name string

		reqPath string
		reqBody string

		callService  bool
		expectFormat *cmrecording.Format
		expectSilent *int
		expectKey    *string
		expectDur    *int
		responseErr  error

		expectStatus int
	}{
		{
			name:         "empty body: all fields are omitted",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			callService:  true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "empty json object",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			reqBody:      `{}`,
			callService:  true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "all fields",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			reqBody:      `{"format":"wav","end_of_silence":10,"end_of_key":"#","duration":3600}`,
			callService:  true,
			expectFormat: &wav,
			expectSilent: &ten,
			expectKey:    &key,
			expectDur:    &dur,
			expectStatus: http.StatusOK,
		},
		{
			name:         "on_end_flow_id in the body is ignored and never forwarded",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			reqBody:      `{"on_end_flow_id":"11111111-3b9f-11ef-98ac-db226570f09a"}`,
			callService:  true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "invalid json",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			reqBody:      `{"duration":`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid argument from the service",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			reqBody:      `{"duration":0}`,
			callService:  true,
			expectDur:    func() *int { v := 0; return &v }(),
			responseErr:  serviceerrors.ErrInvalidArgument,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "already recording is 409",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			callService:  true,
			responseErr:  serviceerrors.ErrStateInvalid,
			expectStatus: http.StatusConflict,
		},
		{
			name:         "permission denied",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_start",
			callService:  true,
			responseErr:  serviceerrors.ErrPermissionDenied,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "malformed id is rejected before the service",
			reqPath:      "/service_agents/calls/not-a-uuid/recording_start",
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

			var req *http.Request
			if tt.reqBody == "" {
				req, _ = http.NewRequest("POST", tt.reqPath, nil)
			} else {
				req, _ = http.NewRequest("POST", tt.reqPath, strings.NewReader(tt.reqBody))
				req.Header.Set("Content-Type", "application/json")
			}
			if tt.callService {
				var res *cmcall.WebhookMessage
				if tt.responseErr == nil {
					res = response
				}
				mockSvc.EXPECT().ServiceAgentCallRecordingStart(req.Context(), agent, callID, tt.expectFormat, tt.expectSilent, tt.expectKey, tt.expectDur).Return(res, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_serviceAgentsCallsIDRecordingStopPOST(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	callID := uuid.FromStringOrNil("0b2decd0-3bb5-11ef-b2bb-bb18783b3f41")
	response := &cmcall.WebhookMessage{Identity: commonidentity.Identity{ID: callID}}

	tests := []struct {
		name string

		reqPath string

		callService bool
		responseErr error

		expectStatus int
	}{
		{
			name:         "normal",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_stop",
			callService:  true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "no ongoing recording is 409",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_stop",
			callService:  true,
			responseErr:  serviceerrors.ErrStateInvalid,
			expectStatus: http.StatusConflict,
		},
		{
			name:         "permission denied",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_stop",
			callService:  true,
			responseErr:  serviceerrors.ErrPermissionDenied,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "backend failure",
			reqPath:      "/service_agents/calls/" + callID.String() + "/recording_stop",
			callService:  true,
			responseErr:  errors.New("rpc failed"),
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "malformed id is rejected before the service",
			reqPath:      "/service_agents/calls/not-a-uuid/recording_stop",
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

			req, _ := http.NewRequest("POST", tt.reqPath, nil)
			if tt.callService {
				var res *cmcall.WebhookMessage
				if tt.responseErr == nil {
					res = response
				}
				mockSvc.EXPECT().ServiceAgentCallRecordingStop(req.Context(), agent, callID).Return(res, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_serviceAgentsCallsIDRecording_NoAuth(t *testing.T) {
	for _, suffix := range []string{"recording_start", "recording_stop"} {
		t.Run(suffix, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/service_agents/calls/0b2decd0-3bb5-11ef-b2bb-bb18783b3f41/"+suffix, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

package server

import (
	"bytes"
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
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_serviceAgentsCallsIdParticipantsPOST(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	callID := uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
	extensionID := "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"
	okBody := `{"address":{"type":"extension","target":"` + extensionID + `"}}`

	tests := []struct {
		name string

		path    string
		reqBody string

		callService      bool
		expectAddress    commonaddress.Address
		responseRes      *cmgroupcall.WebhookMessage
		responseErr      error
		expectStatus     int
		expectBodyPrefix string
		expectBodyHas    string
	}{
		{
			name:    "normal, only type and target reach the service",
			path:    "/service_agents/calls/" + callID.String() + "/participants",
			reqBody: `{"address":{"type":"extension","target":"` + extensionID + `","target_name":"9999","name":"x","detail":"y"}}`,

			callService:   true,
			expectAddress: commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID},
			responseRes: &cmgroupcall.WebhookMessage{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021")},
				CallIDs:  []uuid.UUID{uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c022")},
			},
			expectStatus:     http.StatusOK,
			expectBodyPrefix: `{"id":"72e68b78-8286-11ed-8875-378ced61c021"`,
			expectBodyHas:    `"call_ids":["72e68b78-8286-11ed-8875-378ced61c022"]`,
		},
		{
			name:    "permission denied",
			path:    "/service_agents/calls/" + callID.String() + "/participants",
			reqBody: okBody,

			callService:   true,
			expectAddress: commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID},
			responseErr:   serviceerrors.ErrPermissionDenied,
			expectStatus:  http.StatusForbidden,
		},
		{
			name:    "conflict from the service",
			path:    "/service_agents/calls/" + callID.String() + "/participants",
			reqBody: okBody,

			callService:   true,
			expectAddress: commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID},
			responseErr:   cerrors.FailedPrecondition("api-manager", "PARTICIPANT_COUNT_INVALID", "A participant can be added to a call of two parties only."),
			expectStatus:  http.StatusConflict,
			expectBodyHas: "PARTICIPANT_COUNT_INVALID",
		},
		{
			name:    "invalid argument from the service",
			path:    "/service_agents/calls/" + callID.String() + "/participants",
			reqBody: okBody,

			callService:   true,
			expectAddress: commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID},
			responseErr:   cerrors.InvalidArgument("api-manager", "INVALID_PARTICIPANT_ADDRESS", "The participant extension was not found."),
			expectStatus:  http.StatusBadRequest,
		},
		{
			name:    "backend failure",
			path:    "/service_agents/calls/" + callID.String() + "/participants",
			reqBody: okBody,

			callService:   true,
			expectAddress: commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID},
			responseErr:   errors.New("rpc failed"),
			expectStatus:  http.StatusInternalServerError,
		},
		{
			name:         "malformed json is rejected before the service",
			path:         "/service_agents/calls/" + callID.String() + "/participants",
			reqBody:      `{"address":`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid call id is rejected before the service",
			path:         "/service_agents/calls/not-a-uuid/participants",
			reqBody:      okBody,
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

			req, _ := http.NewRequest("POST", tt.path, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.callService {
				mockSvc.EXPECT().ServiceAgentCallParticipantAdd(req.Context(), agent, callID, tt.expectAddress).Return(tt.responseRes, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
			if tt.expectBodyPrefix != "" && !bytes.HasPrefix(w.Body.Bytes(), []byte(tt.expectBodyPrefix)) {
				t.Errorf("Wrong body.\nexpect prefix: %s\ngot: %s", tt.expectBodyPrefix, w.Body.String())
			}
			if tt.expectBodyHas != "" && !strings.Contains(w.Body.String(), tt.expectBodyHas) {
				t.Errorf("Wrong body.\nexpect to contain: %s\ngot: %s", tt.expectBodyHas, w.Body.String())
			}
		})
	}
}

func Test_serviceAgentsCallsIdParticipantsPOST_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/service_agents/calls/896d70be-3f9f-11ef-bfa4-532d59e85e8c/participants", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong status. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
	}
}

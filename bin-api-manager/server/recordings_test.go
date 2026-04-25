package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_recordingsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseRecording *cmrecording.WebhookMessage

		expectRecordingID uuid.UUID
		expectRes         string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/recordings/31982926-61e3-11eb-a373-37c520973929",

			responseRecording: &cmrecording.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31982926-61e3-11eb-a373-37c520973929"),
				},
			},

			expectRecordingID: uuid.FromStringOrNil("31982926-61e3-11eb-a373-37c520973929"),
			expectRes:         `{"id":"31982926-61e3-11eb-a373-37c520973929","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().RecordingGet(req.Context(), tt.agent, tt.expectRecordingID).Return(tt.responseRecording, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_recordingsIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery          string
		responseRecording *cmrecording.WebhookMessage

		expectRecordingID uuid.UUID
		expectRes         string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/recordings/ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c",
			responseRecording: &cmrecording.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c"),
				},
			},

			expectRecordingID: uuid.FromStringOrNil("ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c"),
			expectRes:         `{"id":"ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().RecordingDelete(req.Context(), tt.agent, tt.expectRecordingID).Return(tt.responseRecording, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

// Test_recordingsIDGet_MissingAuthIdentity verifies GetRecordingsId emits the
// canonical UNAUTHENTICATED / AUTHENTICATION_REQUIRED envelope when
// auth_identity is missing from the gin context.
func Test_recordingsIDGet_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodGet, "/recordings/31982926-61e3-11eb-a373-37c520973929", nil)
}

// Test_recordingsIDGet_InvalidID verifies that a malformed UUID in the path
// triggers INVALID_ARGUMENT / INVALID_ID before the servicehandler is consulted.
func Test_recordingsIDGet_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// "not-a-uuid" passes the path-shape check but uuid.FromStringOrNil
	// returns uuid.Nil, so the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodGet, "/recordings/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}

// Test_recordingsIDGet_ServiceError exercises the servicehandler-failure path
// through abortWithServiceError. The translator's substring fallback maps
// "recording not found" to NOT_FOUND / RESOURCE_NOT_FOUND.
func Test_recordingsIDGet_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	recordingID := uuid.FromStringOrNil("31982926-61e3-11eb-a373-37c520973929")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/recordings/31982926-61e3-11eb-a373-37c520973929", nil)
	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().RecordingGet(gomock.Any(), agent, recordingID).Return(nil, fmt.Errorf("recording not found"))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND", commonoutline.ServiceNameAPIManager)
}

// Test_recordingsIDDelete_MissingAuthIdentity exercises the
// auth-identity-missing branch of DeleteRecordingsId.
func Test_recordingsIDDelete_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodDelete, "/recordings/ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c", nil)
}

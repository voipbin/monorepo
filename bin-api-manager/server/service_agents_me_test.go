package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsMe(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery    string
		responseMe  *amagent.WebhookMessage
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/me",

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			expectedRes: `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}`,
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
			mockSvc.EXPECT().ServiceAgentMeGet(req.Context(), tt.agent).Return(tt.responseMe, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expected: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_mePUT(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectName       string
		expectDetail     string
		expectRingMethod amagent.RingMethod
		expectedRes      string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/me",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","ring_method":"ringall"}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectName:       "test name",
			expectDetail:     "test detail",
			expectRingMethod: amagent.RingMethodRingAll,
			expectedRes:      `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdate(req.Context(), tt.agent, tt.expectName, tt.expectDetail, tt.expectRingMethod).Return(tt.responseMe, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expected: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_meAddressesPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectAddresses []commonaddress.Address
		expectedRes     string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/me/addresses",
			reqBody:  []byte(`{"addresses":[{"type":"tel","target":"+123456"}]}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+123456",
				},
			},
			expectedRes: `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdateAddresses(req.Context(), tt.agent, tt.expectAddresses).Return(tt.responseMe, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expected: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_meStatusPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectStatus amagent.Status
		expectedRes  string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/me/status",
			reqBody:  []byte(`{"status":"available"}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectStatus: amagent.StatusAvailable,
			expectedRes:  `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdateStatus(req.Context(), tt.agent, tt.expectStatus).Return(tt.responseMe, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expected: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_mePasswordPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectPassword string
		expectedRes    string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/me/password",
			reqBody:  []byte(`{"password":"test_password"}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectPassword: "test_password",
			expectedRes:    `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdatePassword(req.Context(), tt.agent, tt.expectPassword).Return(tt.responseMe, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expected: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

// Test_GetServiceAgentsMe_MissingAuthIdentity exercises the
// auth-identity-missing branch of GetServiceAgentsMe. Without auth_identity
// in the gin context, the handler must emit UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED with a populated request_id.
func Test_GetServiceAgentsMe_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodGet, "/service_agents/me", nil)
}

// Test_mePUT_InvalidJSONBody verifies PutServiceAgentsMe rejects malformed
// JSON with INVALID_ARGUMENT / INVALID_JSON_BODY.
func Test_mePUT_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// Intentionally invalid JSON body.
	req, _ := http.NewRequest(http.MethodPut, "/service_agents/me", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_GetServiceAgentsMe_ServiceError exercises the servicehandler-failure
// path through abortWithServiceError. The translator's sentinel match
// maps "agent not found" to NOT_FOUND / RESOURCE_NOT_FOUND.
func Test_GetServiceAgentsMe_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/service_agents/me", nil)
	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().ServiceAgentMeGet(gomock.Any(), agent).Return(nil, fmt.Errorf("%w: agent not found", serviceerrors.ErrNotFound))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND")
}

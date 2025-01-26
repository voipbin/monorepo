package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsMe(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		responseMe  *amagent.WebhookMessage
		expectedRes string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/service_agents/me",

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			expectedRes: `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentMeGet(req.Context(), &tt.agent).Return(tt.responseMe, nil)

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
		agent amagent.Agent

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
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

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
			expectedRes:      `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdate(req.Context(), &tt.agent, tt.expectName, tt.expectDetail, tt.expectRingMethod).Return(tt.responseMe, nil)

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
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectAddresses []commonaddress.Address
		expectedRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

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
			expectedRes: `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdateAddresses(req.Context(), &tt.agent, tt.expectAddresses).Return(tt.responseMe, nil)

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
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectStatus amagent.Status
		expectedRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/service_agents/me/status",
			reqBody:  []byte(`{"status":"available"}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectStatus: amagent.StatusAvailable,
			expectedRes:  `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdateStatus(req.Context(), &tt.agent, tt.expectStatus).Return(tt.responseMe, nil)

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
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMe *amagent.WebhookMessage

		expectPassword string
		expectedRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/service_agents/me/password",
			reqBody:  []byte(`{"password":"test_password"}`),

			responseMe: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectPassword: "test_password",
			expectedRes:    `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().ServiceAgentMeUpdatePassword(req.Context(), &tt.agent, tt.expectPassword).Return(tt.responseMe, nil)

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

package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAipromptproposals(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseProposal *amaipromptproposal.WebhookMessage

		expectedAIID     uuid.UUID
		expectedAuditIDs []uuid.UUID
		expectedLanguage string
		expectedRes      string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals",
			reqBody:  []byte(`{"ai_id":"11111111-0000-0000-0000-000000000002","audit_ids":["11111111-0000-0000-0000-000000000010","11111111-0000-0000-0000-000000000011"],"language":"en-US"}`),

			responseProposal: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
				},
			},

			expectedAIID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
			expectedAuditIDs: []uuid.UUID{
				uuid.FromStringOrNil("11111111-0000-0000-0000-000000000010"),
				uuid.FromStringOrNil("11111111-0000-0000-0000-000000000011"),
			},
			expectedLanguage: "en-US",
			expectedRes:      `{"id":"11111111-0000-0000-0000-000000000003","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AIPromptProposalCreate(
				req.Context(),
				tt.agent,
				tt.expectedAIID,
				tt.expectedAuditIDs,
				tt.expectedLanguage,
			).Return(tt.responseProposal, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusAccepted {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusAccepted, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_PostAipromptproposals_InvalidUUID_Returns400(t *testing.T) {
	tests := []struct {
		name string

		reqBody []byte
	}{
		{
			name:    "invalid ai_id",
			reqBody: []byte(`{"ai_id":"not-a-uuid","audit_ids":["11111111-0000-0000-0000-000000000010"],"language":"en-US"}`),
		},
		{
			name:    "invalid audit_id",
			reqBody: []byte(`{"ai_id":"11111111-0000-0000-0000-000000000002","audit_ids":["11111111-0000-0000-0000-000000000010","not-a-uuid"],"language":"en-US"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			agent := auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
				},
			})

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/aipromptproposals", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			// No service call expected — request must be rejected at the handler layer.
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetAipromptproposals(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseProposals []*amaipromptproposal.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedAIID      uuid.UUID
		expectedStatus    amaipromptproposal.Status
		expectedRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseProposals: []*amaipromptproposal.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedAIID:      uuid.Nil,
			expectedStatus:    amaipromptproposal.Status(""),
			expectedRes:       `{"result":[{"id":"22222222-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().AIPromptProposalGetsByCustomerID(
				req.Context(),
				tt.agent,
				tt.expectedPageSize,
				tt.expectedPageToken,
				tt.expectedAIID,
				tt.expectedStatus,
			).Return(tt.responseProposals, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAipromptproposalsId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseProposal *amaipromptproposal.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals/33333333-0000-0000-0000-000000000002",

			responseProposal: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002"),
				},
			},

			expectedID:  uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"33333333-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().AIPromptProposalGet(req.Context(), tt.agent, tt.expectedID).Return(tt.responseProposal, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_PostAipromptproposalsIdAccept(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseProposal *amaipromptproposal.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals/44444444-0000-0000-0000-000000000002/accept",

			responseProposal: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusAccepted,
			},

			expectedID:  uuid.FromStringOrNil("44444444-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"44444444-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"accepted","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().AIPromptProposalAccept(req.Context(), tt.agent, tt.expectedID).Return(tt.responseProposal, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_PostAipromptproposalsIdReject(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseProposal *amaipromptproposal.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals/55555555-0000-0000-0000-000000000002/reject",

			responseProposal: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55555555-0000-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusRejected,
			},

			expectedID:  uuid.FromStringOrNil("55555555-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"55555555-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"rejected","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().AIPromptProposalReject(req.Context(), tt.agent, tt.expectedID).Return(tt.responseProposal, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_DeleteAipromptproposalsId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseProposal *amaipromptproposal.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aipromptproposals/66666666-0000-0000-0000-000000000002",

			responseProposal: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66666666-0000-0000-0000-000000000002"),
				},
			},

			expectedID:  uuid.FromStringOrNil("66666666-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"66666666-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().AIPromptProposalDelete(req.Context(), tt.agent, tt.expectedID).Return(tt.responseProposal, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAiaudits(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAIAudits []*amaiaudit.WebhookMessage

		expectedAicallID uuid.UUID
		expectedLanguage string
		expectedRes      string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f7c6e5d4-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aiaudits",
			reqBody:  []byte(`{"aicall_id":"f7c6e5d4-0000-0000-0000-000000000002","language":"en-US"}`),

			responseAIAudits: []*amaiaudit.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f7c6e5d4-0000-0000-0000-000000000003"),
					},
				},
			},

			expectedAicallID: uuid.FromStringOrNil("f7c6e5d4-0000-0000-0000-000000000002"),
			expectedLanguage: "en-US",
			expectedRes:      `{"result":[{"id":"f7c6e5d4-0000-0000-0000-000000000003","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","overall_score":null,"evaluation":null,"tm_create":null,"tm_update":null,"tm_delete":null}]}`,
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
			mockSvc.EXPECT().AIAuditCreate(
				req.Context(),
				tt.agent,
				tt.expectedAicallID,
				tt.expectedLanguage,
			).Return(tt.responseAIAudits, nil)

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

func Test_GetAiaudits(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAIAudits []*amaiaudit.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedAicallID  uuid.UUID
		expectedAIID      uuid.UUID
		expectedRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8d7f6e5-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aiaudits?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAIAudits: []*amaiaudit.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a8d7f6e5-0000-0000-0000-000000000002"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedAicallID:  uuid.Nil,
			expectedAIID:      uuid.Nil,
			expectedRes:       `{"result":[{"id":"a8d7f6e5-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","overall_score":null,"evaluation":null,"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
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
			mockSvc.EXPECT().AIAuditGetsByCustomerID(
				req.Context(),
				tt.agent,
				tt.expectedPageSize,
				tt.expectedPageToken,
				tt.expectedAicallID,
				tt.expectedAIID,
			).Return(tt.responseAIAudits, nil)

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

func Test_GetAiauditsId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAIAudit *amaiaudit.WebhookMessage

		expectedID uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b9e8a7f6-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aiaudits/b9e8a7f6-0000-0000-0000-000000000002",

			responseAIAudit: &amaiaudit.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b9e8a7f6-0000-0000-0000-000000000002"),
				},
			},

			expectedID:  uuid.FromStringOrNil("b9e8a7f6-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"b9e8a7f6-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","overall_score":null,"evaluation":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().AIAuditGet(req.Context(), tt.agent, tt.expectedID).Return(tt.responseAIAudit, nil)

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

func Test_DeleteAiauditsId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAIAudit *amaiaudit.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0f9b8e7-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/aiaudits/c0f9b8e7-0000-0000-0000-000000000002",

			responseAIAudit: &amaiaudit.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0f9b8e7-0000-0000-0000-000000000002"),
				},
			},

			expectedID:  uuid.FromStringOrNil("c0f9b8e7-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"c0f9b8e7-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","overall_score":null,"evaluation":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().AIAuditDelete(req.Context(), tt.agent, tt.expectedID).Return(tt.responseAIAudit, nil)

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

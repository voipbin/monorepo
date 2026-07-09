package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsAimessages(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAimessages []*ammessage.WebhookMessage

		expectedAIcallID  uuid.UUID
		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aimessages?aicall_id=5e4a0680-804e-11ec-8477-2fea5968d85b&page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAimessages: []*ammessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedAIcallID:  uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","active_ai_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":null}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentAImessageList(req.Context(), tt.agent, tt.expectedAIcallID, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseAimessages, nil)

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

func Test_PostServiceAgentsAimessages(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAimessage *ammessage.WebhookMessage

		expectAIcallID uuid.UUID
		expectRole     ammessage.Role
		expectContent  string
		expectRes      string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aimessages",
			reqBody:  []byte(`{"aicall_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","role":"user","content":"hello"}`),

			responseAimessage: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectAIcallID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectRole:     ammessage.RoleUser,
			expectContent:  "hello",
			expectRes:      `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","active_ai_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":null}`,
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

			mockSvc.EXPECT().ServiceAgentAImessageCreate(
				req.Context(),
				tt.agent,
				tt.expectAIcallID,
				tt.expectRole,
				tt.expectContent,
			).Return(tt.responseAimessage, nil)

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

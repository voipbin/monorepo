package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsAicalls(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAicalls []*amaicall.WebhookMessage

		expectedPageSize      uint64
		expectedPageToken     string
		expectedReferenceType string
		expectedReferenceID   uuid.UUID
		expectedStatus        string
		expectedRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aicalls?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAicalls: []*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
		{
			name: "filtered by reference_type and reference_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aicalls?reference_type=contact_case&reference_id=5e4a0680-804e-11ec-8477-2fea5968d85b",

			responseAicalls: []*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:      100,
			expectedPageToken:     "",
			expectedReferenceType: "contact_case",
			expectedReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectedRes:           `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
		{
			name: "filtered by reference_type, reference_id, and status",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aicalls?reference_type=contact_case&reference_id=5e4a0680-804e-11ec-8477-2fea5968d85b&status=progressing",

			responseAicalls: []*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:      100,
			expectedPageToken:     "",
			expectedReferenceType: "contact_case",
			expectedReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectedStatus:        "progressing",
			expectedRes:           `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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

			mockSvc.EXPECT().ServiceAgentAIcallList(req.Context(), tt.agent, tt.expectedPageSize, tt.expectedPageToken, tt.expectedReferenceType, tt.expectedReferenceID, tt.expectedStatus).Return(tt.responseAicalls, nil)

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

// Test_GetServiceAgentsAicalls_PartialReferenceFilter verifies
// GetServiceAgentsAicalls rejects a partial reference_type/reference_id
// filter (only one of the two supplied) with INVALID_ARGUMENT /
// INVALID_REFERENCE_FILTER before the servicehandler is consulted.
func Test_GetServiceAgentsAicalls_PartialReferenceFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type test struct {
		name     string
		reqQuery string
	}

	tests := []test{
		{
			name:     "reference_type without reference_id",
			reqQuery: "/service_agents/aicalls?reference_type=contact_case",
		},
		{
			name:     "reference_id without reference_type",
			reqQuery: "/service_agents/aicalls?reference_id=5e4a0680-804e-11ec-8477-2fea5968d85b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
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

			req, _ := http.NewRequest(http.MethodGet, tt.reqQuery, nil)
			r.ServeHTTP(w, req)

			assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_REFERENCE_FILTER")
		})
	}
}

func Test_PostServiceAgentsAicalls(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAicall *amaicall.WebhookMessage

		expectAssistanceType amaicall.AssistanceType
		expectAssistanceID   uuid.UUID
		expectReferenceType  amaicall.ReferenceType
		expectReferenceID    uuid.UUID
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/aicalls",
			reqBody:  []byte(`{"assistance_type":"ai","assistance_id":"3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46","reference_type":"contact_case","reference_id":"4ecc56ec-8285-11ed-9958-8b0a60b665bf"}`),

			responseAicall: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectAssistanceType: amaicall.AssistanceTypeAI,
			expectAssistanceID:   uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			expectReferenceType:  amaicall.ReferenceTypeContactCase,
			expectReferenceID:    uuid.FromStringOrNil("4ecc56ec-8285-11ed-9958-8b0a60b665bf"),
			expectRes:            `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ServiceAgentAIcallCreate(
				req.Context(),
				tt.agent,
				tt.expectAssistanceType,
				tt.expectAssistanceID,
				tt.expectReferenceType,
				tt.expectReferenceID,
			).Return(tt.responseAicall, nil)

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

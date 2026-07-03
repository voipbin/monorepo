package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsTranscribes(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTranscribes []*tmtranscribe.WebhookMessage

		expectedPageSize      uint64
		expectedPageToken     string
		expectedReferenceType string
		expectedReferenceID   uuid.UUID
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

			reqQuery: "/service_agents/transcribes?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscribes: []*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","provider":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
		{
			name: "filtered by reference_type and reference_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/transcribes?reference_type=call&reference_id=5e4a0680-804e-11ec-8477-2fea5968d85b",

			responseTranscribes: []*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:      100,
			expectedPageToken:     "",
			expectedReferenceType: "call",
			expectedReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectedRes:           `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","provider":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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

			mockSvc.EXPECT().ServiceAgentTranscribeList(req.Context(), tt.agent, tt.expectedPageSize, tt.expectedPageToken, tt.expectedReferenceType, tt.expectedReferenceID).Return(tt.responseTranscribes, nil)

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

// Test_GetServiceAgentsTranscribes_PartialReferenceFilter verifies
// GetServiceAgentsTranscribes rejects a partial reference_type/reference_id
// filter (only one of the two supplied) with INVALID_ARGUMENT /
// INVALID_REFERENCE_FILTER before the servicehandler is consulted, instead
// of silently applying only one half of the intended filter. Mirrors
// Test_transcribesGET_PartialReferenceFilter in transcribes_test.go.
func Test_GetServiceAgentsTranscribes_PartialReferenceFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type test struct {
		name     string
		reqQuery string
	}

	tests := []test{
		{
			name:     "reference_type without reference_id",
			reqQuery: "/service_agents/transcribes?reference_type=call",
		},
		{
			name:     "reference_id without reference_type",
			reqQuery: "/service_agents/transcribes?reference_id=5e4a0680-804e-11ec-8477-2fea5968d85b",
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

func Test_PostServiceAgentsTranscribes(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseTranscribe *tmtranscribe.WebhookMessage

		expectReferenceType string
		expectReferenceID   uuid.UUID
		expectLanguage      string
		expectDirection     tmtranscribe.Direction
		expectOnEndFlowID   uuid.UUID
		expectProvider      tmtranscribe.Provider
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/transcribes",
			reqBody:  []byte(`{"reference_type":"call","reference_id":"4ecc56ec-8285-11ed-9958-8b0a60b665bf","language":"en-US","direction":"both","on_end_flow_id":"199a8a78-0944-11f0-b57c-dbf18b86df64"}`),

			responseTranscribe: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectReferenceType: "call",
			expectReferenceID:   uuid.FromStringOrNil("4ecc56ec-8285-11ed-9958-8b0a60b665bf"),
			expectLanguage:      "en-US",
			expectDirection:     tmtranscribe.DirectionBoth,
			expectOnEndFlowID:   uuid.FromStringOrNil("199a8a78-0944-11f0-b57c-dbf18b86df64"),
			expectProvider:      tmtranscribe.ProviderEmpty,
			expectRes:           `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","provider":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "direction omitted defaults to both",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/transcribes",
			reqBody:  []byte(`{"reference_type":"call","reference_id":"4ecc56ec-8285-11ed-9958-8b0a60b665bf","language":"en-US","on_end_flow_id":"199a8a78-0944-11f0-b57c-dbf18b86df64"}`),

			responseTranscribe: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectReferenceType: "call",
			expectReferenceID:   uuid.FromStringOrNil("4ecc56ec-8285-11ed-9958-8b0a60b665bf"),
			expectLanguage:      "en-US",
			expectDirection:     tmtranscribe.DirectionBoth,
			expectOnEndFlowID:   uuid.FromStringOrNil("199a8a78-0944-11f0-b57c-dbf18b86df64"),
			expectProvider:      tmtranscribe.ProviderEmpty,
			expectRes:           `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","provider":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ServiceAgentTranscribeStart(
				req.Context(),
				tt.agent,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectLanguage,
				tt.expectDirection,
				tt.expectOnEndFlowID,
				tt.expectProvider,
			).Return(tt.responseTranscribe, nil)

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

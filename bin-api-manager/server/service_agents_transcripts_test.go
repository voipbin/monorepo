package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsTranscripts(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTranscripts []*tmtranscript.WebhookMessage

		expectedPageSize     uint64
		expectedPageToken    string
		expectedTranscribeID uuid.UUID
		expectedRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/transcripts?transcribe_id=b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e&page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscripts: []*tmtranscript.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					TranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
					Message:      "Hello, how can I help you?",
				},
			},

			expectedPageSize:     10,
			expectedPageToken:    "2020-09-20T03:23:20.995000Z",
			expectedTranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
			expectedRes:          `{"result":[{"id":"550e8400-e29b-41d4-a716-446655440000","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e","direction":"","message":"Hello, how can I help you?","tm_transcript":null,"tm_create":null}],"next_page_token":""}`,
		},
		{
			// Regression test for the round-1 design review pagination fix
			// — page_size/page_token must actually be forwarded to the
			// servicehandler, not silently dropped/hardcoded.
			name: "default page_size and page_token when omitted",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			}),

			reqQuery: "/service_agents/transcripts?transcribe_id=b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e",

			responseTranscripts: []*tmtranscript.WebhookMessage{},

			expectedPageSize:     100,
			expectedPageToken:    "",
			expectedTranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
			expectedRes:          `{"result":[],"next_page_token":""}`,
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

			mockSvc.EXPECT().ServiceAgentTranscriptList(req.Context(), tt.agent, tt.expectedPageSize, tt.expectedPageToken, tt.expectedTranscribeID).Return(tt.responseTranscripts, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

// Test_GetServiceAgentsTranscripts_MissingTranscribeID verifies the
// required transcribe_id query parameter is enforced by the generated
// oapi-codegen wrapper with an automatic 400, before this handler (or the
// servicehandler) is ever consulted.
func Test_GetServiceAgentsTranscripts_MissingTranscribeID(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/service_agents/transcripts", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusBadRequest, w.Code, w.Body)
	}
}

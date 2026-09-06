package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_transcriptsGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTranscripts []*tmtranscript.WebhookMessage

		expectPageSize     uint64
		expectPageToken    string
		expectTranscribeID uuid.UUID
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscripts: []*tmtranscript.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("844b118e-828d-11ed-84a3-fb13c2a499e9"),
					},
				},
			},

			expectPageSize:     10,
			expectPageToken:    "2020-09-20T03:23:20.995000Z",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[{"id":"844b118e-828d-11ed-84a3-fb13c2a499e9","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":null,"tm_create":null}],"next_page_token":""}`,
		},
		{
			name: "no pagination params use the defaults",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202",

			responseTranscripts: []*tmtranscript.WebhookMessage{},

			expectPageSize:     100,
			expectPageToken:    "",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[],"next_page_token":""}`,
		},
		{
			name: "page_size 0 is reset to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_size=0",

			responseTranscripts: []*tmtranscript.WebhookMessage{},

			expectPageSize:     100,
			expectPageToken:    "",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[],"next_page_token":""}`,
		},
		{
			name: "page_token alone keeps the default size",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscripts: []*tmtranscript.WebhookMessage{},

			expectPageSize:     100,
			expectPageToken:    "2020-09-20T03:23:20.995000Z",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[],"next_page_token":""}`,
		},
		{
			name: "page_size above 100 is reset to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_size=500&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscripts: []*tmtranscript.WebhookMessage{},

			expectPageSize:     100,
			expectPageToken:    "2020-09-20T03:23:20.995000Z",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[],"next_page_token":""}`,
		},
		{
			name: "non-empty page emits the last row's tm_create as next_page_token",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			}),

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202",

			responseTranscripts: []*tmtranscript.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("844b118e-828d-11ed-84a3-fb13c2a499e9"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9f06037a-8284-11ed-8b1a-1f5800b90993"),
					},
					TMCreate: timePtr("2020-09-20T03:23:20.995000Z"),
				},
			},

			expectPageSize:     100,
			expectPageToken:    "",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[{"id":"844b118e-828d-11ed-84a3-fb13c2a499e9","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":null,"tm_create":"2020-09-20T03:23:21.995Z"},{"id":"9f06037a-8284-11ed-8b1a-1f5800b90993","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":null,"tm_create":"2020-09-20T03:23:20.995Z"}],"next_page_token":"2020-09-20T03:23:20.995000Z"}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscriptList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken, tt.expectTranscribeID).Return(tt.responseTranscripts, nil)

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

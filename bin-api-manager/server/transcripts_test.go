package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
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
		agent amagent.Agent

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
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			},

			reqQuery: "/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseTranscripts: []*tmtranscript.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("844b118e-828d-11ed-84a3-fb13c2a499e9"),
				},
			},

			expectPageSize:     10,
			expectPageToken:    "2020-09-20 03:23:20.995000",
			expectTranscribeID: uuid.FromStringOrNil("8425d50e-828d-11ed-a91c-f77fe2ce8202"),
			expectRes:          `{"result":[{"id":"844b118e-828d-11ed-84a3-fb13c2a499e9","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":""}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscriptGets(req.Context(), &tt.agent, tt.expectTranscribeID).Return(tt.responseTranscripts, nil)

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

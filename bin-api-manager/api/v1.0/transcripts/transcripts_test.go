package transcripts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_transcriptsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		requestBody request.ParamTranscriptsGET

		responseTranscripts []*tmtranscript.WebhookMessage
		expectRes           string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83f82e1a-828d-11ed-89ea-9f7ac48ae9b8"),
				},
			},

			"/v1.0/transcripts?transcribe_id=8425d50e-828d-11ed-a91c-f77fe2ce8202&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamTranscriptsGET{
				TranscribeID: "8425d50e-828d-11ed-a91c-f77fe2ce8202",
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*tmtranscript.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("844b118e-828d-11ed-84a3-fb13c2a499e9"),
				},
			},
			`{"result":[{"id":"844b118e-828d-11ed-84a3-fb13c2a499e9","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":""}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TranscriptGets(req.Context(), &tt.agent, uuid.FromStringOrNil(tt.requestBody.TranscribeID)).Return(tt.responseTranscripts, nil)

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

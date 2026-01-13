package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_RecordingfilesIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseDownloadURL string

		expectRecordingfileID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/recordingfiles/79bf1fee-61e2-11eb-b0e8-6b21f6734c33",

			responseDownloadURL: "https://test.com/call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020.wav?token=token",

			expectRecordingfileID: uuid.FromStringOrNil("79bf1fee-61e2-11eb-b0e8-6b21f6734c33"),
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
			mockSvc.EXPECT().RecordingfileGet(req.Context(), &tt.agent, tt.expectRecordingfileID.Return(tt.responseDownloadURL, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusTemporaryRedirect {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusTemporaryRedirect, w.Code)
			}

			if w.Result().Header["Location"][0] != tt.responseDownloadURL {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.responseDownloadURL, w.Result().Header["Location"][0])
			}
		})
	}
}

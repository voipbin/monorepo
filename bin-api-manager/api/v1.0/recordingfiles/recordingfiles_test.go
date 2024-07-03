package recordingfiles

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_RecordingfilesIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		agent       amagent.Agent
		recording   recording.Recording
		downloadURL string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			recording.Recording{
				ID: uuid.FromStringOrNil("79bf1fee-61e2-11eb-b0e8-6b21f6734c33"),
			},
			"https://test.com/call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020.wav?token=token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/recordingfiles/%s", tt.recording.ID), nil)
			mockSvc.EXPECT().RecordingfileGet(req.Context(), &tt.agent, tt.recording.ID).Return(tt.downloadURL, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusTemporaryRedirect || w.Result().Header["Location"][0] != tt.downloadURL {
				t.Errorf("Wrong match. expect: %d, got: %d, response: %v", http.StatusTemporaryRedirect, w.Code, w)
			}
		})
	}
}

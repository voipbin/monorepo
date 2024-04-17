package recordings

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_recordingsIDGET(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		recording *cmrecording.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&cmrecording.WebhookMessage{
				ID: uuid.FromStringOrNil("31982926-61e3-11eb-a373-37c520973929"),
			},
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

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/recordings/%s", tt.recording.ID), nil)
			mockSvc.EXPECT().RecordingGet(req.Context(), &tt.agent, tt.recording.ID).Return(tt.recording, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_recordingsIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery          string
		responseRecording *cmrecording.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/recordings/ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c",
			&cmrecording.WebhookMessage{
				ID: uuid.FromStringOrNil("ca5f68bc-8f1e-11ed-957c-9b7ba0e03f3c"),
			},
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().RecordingDelete(req.Context(), &tt.agent, tt.responseRecording.ID).Return(tt.responseRecording, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

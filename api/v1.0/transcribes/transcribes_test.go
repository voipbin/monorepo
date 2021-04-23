package transcribes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestTranscribesPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user user.User

		requestBody request.BodyTranscribesPOST
		trans       *transcribe.Transcribe

		recording *recording.Recording
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			request.BodyTranscribesPOST{
				RecordingID: uuid.FromStringOrNil("1c71e72e-a3f8-11eb-a402-7b13f5ec585d"),
				Language:    "en-US",
			},
			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("82120398-a3f8-11eb-a86b-cfcde0c85e25"),
				Type:          transcribe.TypeRecording,
				ReferenceID:   uuid.FromStringOrNil("1c71e72e-a3f8-11eb-a402-7b13f5ec585d"),
				Language:      "en-US",
				WebhookURI:    "",
				WebhookMethod: "",
				Transcription: "Hello, world.",
			},
			&recording.Recording{
				ID:     uuid.FromStringOrNil("1c71e72e-a3f8-11eb-a402-7b13f5ec585d"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().TranscribeCreate(&tt.user, tt.requestBody.RecordingID, tt.requestBody.Language).Return(tt.trans, nil)
			req, _ := http.NewRequest("POST", "/v1.0/transcribes", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

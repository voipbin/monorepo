package recordings

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestRecordingsIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        models.User
		recording   *models.Recording
		recordingID string
		downloadURL string
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			&models.Recording{
				ID: uuid.FromStringOrNil("31982926-61e3-11eb-a373-37c520973929"),
			},
			"call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020-04-18T03:22:17.995000Z.wav",
			"https://test.com/call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020.wav?token=token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().RecordingGet(&tt.user, tt.recording.ID).Return(tt.recording, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/recordings/%s", tt.recording.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

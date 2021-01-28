package recordingfiles

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/servicehandler"
)

func setupServer(app *gin.Engine) {
	ApplyRoutes(&app.RouterGroup)
}

func TestRecordingfilesIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		recordingID string
		downloadURL string
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
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
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().RecordingGet(&tt.user, tt.recordingID).Return(tt.downloadURL, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/recordingfiles/%s", tt.recordingID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusTemporaryRedirect || w.HeaderMap["Location"][0] != tt.downloadURL {
				t.Errorf("Wrong match. expect: %d, got: %d, response: %v", http.StatusTemporaryRedirect, w.Code, w)
			}
		})
	}
}

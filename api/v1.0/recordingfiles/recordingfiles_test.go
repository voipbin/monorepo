package recordingfiles

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestRecordingfilesIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		recording   recording.Recording
		downloadURL string
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
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
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().RecordingfileGet(&tt.user, tt.recording.ID).Return(tt.downloadURL, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/recordingfiles/%s", tt.recording.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusTemporaryRedirect || w.HeaderMap["Location"][0] != tt.downloadURL {
				t.Errorf("Wrong match. expect: %d, got: %d, response: %v", http.StatusTemporaryRedirect, w.Code, w)
			}
		})
	}
}

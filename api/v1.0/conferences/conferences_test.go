package conferences

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/conference"
)

func setupServer(app *gin.Engine) {
	ApplyRoutes(&app.RouterGroup)
}

func TestConferencesIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	type test struct {
		name       string
		conference *conference.Conference
	}

	tests := []test{
		{
			"simple test",
			&conference.Conference{
				ID: uuid.FromStringOrNil("5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("requestHandler", mockReq)
			})
			setupServer(r)

			mockReq.EXPECT().CallConferenceGet(tt.conference.ID).Return(tt.conference, nil)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/conferences/%s", tt.conference.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

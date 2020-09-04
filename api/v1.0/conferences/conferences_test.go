package conferences

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager/models"
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

func TestConferencesPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	type test struct {
		name       string
		user       models.User
		conference *conference.Conference
	}

	tests := []test{
		{
			"conference type",
			models.User{
				ID: 1,
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("ee1e90cc-ac7a-11ea-8474-e740530b4266"),
				Type: conference.TypeConference,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("requestHandler", mockReq)
				c.Set("user", tt.user)
			})
			setupServer(r)

			body := []byte(fmt.Sprintf(`{"type": "%s"}`, tt.conference.Type))
			mockReq.EXPECT().CallConferenceCreate(tt.user.ID, tt.conference.Type).Return(tt.conference, nil)

			req, _ := http.NewRequest("POST", "/conferences", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func TestConferencesIDDELETE(t *testing.T) {

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
				ID: uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),
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

			mockReq.EXPECT().CallConferenceDelete(tt.conference.ID).Return(nil)

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/conferences/%s", tt.conference.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

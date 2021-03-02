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

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestConferencesIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name       string
		user       user.User
		conference *conference.Conference
	}

	tests := []test{
		{
			"simple test",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
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
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().ConferenceGet(&tt.user, tt.conference.ID).Return(tt.conference, nil)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/conferences/%s", tt.conference.ID), nil)

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

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name       string
		user       user.User
		confType   conference.Type
		confName   string
		confDetail string
		conference *conference.Conference
	}

	tests := []test{
		{
			"conference type",
			user.User{
				ID: 1,
			},
			conference.TypeConference,
			"conference name",
			"conference detail",
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
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body := []byte(fmt.Sprintf(`{"type": "%s", "name": "%s", "detail": "%s"}`, tt.confType, tt.confName, tt.confDetail))

			mockSvc.EXPECT().ConferenceCreate(&tt.user, tt.confType, tt.confName, tt.confDetail).Return(tt.conference, nil)
			req, _ := http.NewRequest("POST", "/v1.0/conferences", bytes.NewBuffer(body))

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

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name       string
		user       user.User
		conference *conference.Conference
	}

	tests := []test{
		{
			"simple test",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
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
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().ConferenceDelete(&tt.user, tt.conference.ID).Return(nil)

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/conferences/%s", tt.conference.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

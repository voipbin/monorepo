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

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
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
				c.Set(common.OBJServiceHandler, mockSvc)
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
		conference *conference.Conference
		request    []byte
	}

	tests := []test{
		{
			"conference type",
			user.User{
				ID: 1,
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("ee1e90cc-ac7a-11ea-8474-e740530b4266"),
				Type:   conference.TypeConference,
				Name:   "conference name",
				Detail: "conference detail",
			},
			[]byte(`{"type": "conference", "name": "conference name", "detail": "conference detail"}`),
		},
		{
			"webhook uri",
			user.User{
				ID: 1,
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("b85ee002-2089-11ec-a49b-531b1931ddbd"),
				Type:       conference.TypeConference,
				Name:       "conference name",
				Detail:     "conference detail",
				WebhookURI: "test.com/webhook",
			},
			[]byte(`{"type": "conference", "name": "conference name", "detail": "conference detail", "webhook_uri": "test.com/webhook"}`),
		},
		{
			"pre/post actions",
			user.User{
				ID: 1,
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("62fc88ba-3fe9-11ec-8ebb-8f1ee591edec"),
				Type:       conference.TypeConference,
				Name:       "conference name",
				Detail:     "conference detail",
				WebhookURI: "test.com/webhook",
				PreActions: []action.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []action.Action{
					{
						Type: "hangup",
					},
				},
			},
			[]byte(`{"type": "conference", "name": "conference name", "detail": "conference detail", "webhook_uri": "test.com/webhook", "pre_actions": [{"type": "answer"}], "post_actions":[{"type": "hangup"}]}`),
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

			mockSvc.EXPECT().ConferenceCreate(&tt.user, tt.conference.Type, tt.conference.Name, tt.conference.Detail, tt.conference.WebhookURI, tt.conference.PreActions, tt.conference.PostActions).Return(tt.conference, nil)
			req, _ := http.NewRequest("POST", "/v1.0/conferences", bytes.NewBuffer(tt.request))

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
				c.Set(common.OBJServiceHandler, mockSvc)
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

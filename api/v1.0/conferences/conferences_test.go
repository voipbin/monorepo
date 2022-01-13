package conferences

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
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

	tests := []struct {
		name string
		user user.User
		id   uuid.UUID

		requestURI string

		conference *cfconference.WebhookMessage
	}{
		{
			"simple test",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			uuid.FromStringOrNil("5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb"),

			"/v1.0/conferences/5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb",
			&cfconference.WebhookMessage{
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

			mockSvc.EXPECT().ConferenceGet(&tt.user, tt.id).Return(tt.conference, nil)

			req, _ := http.NewRequest("GET", tt.requestURI, nil)

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

	tests := []struct {
		name string
		user user.User

		conferenceType cfconference.Type
		conferenceName string
		detail         string
		webhookURI     string
		preActions     []fmaction.Action
		postActions    []fmaction.Action

		conference *cfconference.WebhookMessage
		request    []byte
	}{
		{
			"conference type",
			user.User{
				ID: 1,
			},

			cfconference.TypeConference,
			"conference name",
			"conference detail",
			"",
			[]fmaction.Action{},
			[]fmaction.Action{},

			&cfconference.WebhookMessage{
				ID:     uuid.FromStringOrNil("ee1e90cc-ac7a-11ea-8474-e740530b4266"),
				Type:   cfconference.TypeConference,
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

			cfconference.TypeConference,
			"conference name",
			"conference detail",
			"test.com/webhook",
			[]fmaction.Action{},
			[]fmaction.Action{},

			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("b85ee002-2089-11ec-a49b-531b1931ddbd"),
				Type:       cfconference.TypeConference,
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

			cfconference.TypeConference,
			"conference name",
			"conference detail",
			"test.com/webhook",
			[]fmaction.Action{
				{
					Type: "answer",
				},
			},
			[]fmaction.Action{
				{
					Type: "hangup",
				},
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("62fc88ba-3fe9-11ec-8ebb-8f1ee591edec"),
				Type:       cfconference.TypeConference,
				Name:       "conference name",
				Detail:     "conference detail",
				WebhookURI: "test.com/webhook",
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions: []fmaction.Action{
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

	tests := []struct {
		name string
		user user.User
		id   uuid.UUID

		requestURI string
	}{
		{
			"simple test",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),
			"/v1.0/conferences/f49f8cc6-ac7f-11ea-91a3-e7103a41fa51",
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

			mockSvc.EXPECT().ConferenceDelete(&tt.user, tt.id).Return(nil)

			req, _ := http.NewRequest("DELETE", tt.requestURI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func TestConferencesIDCallsIDDELETE(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name       string
		user       user.User
		requestURI string

		conferenceID uuid.UUID
		callID       uuid.UUID
	}

	tests := []test{
		{
			"simple test",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			"/v1.0/conferences/88f410a4-44aa-11ec-a648-ef1c8a8d5b49/calls/a5f77e0c-44aa-11ec-860c-0718d81c0e6b",
			uuid.FromStringOrNil("88f410a4-44aa-11ec-a648-ef1c8a8d5b49"),
			uuid.FromStringOrNil("a5f77e0c-44aa-11ec-860c-0718d81c0e6b"),
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

			mockSvc.EXPECT().ConferenceKick(&tt.user, tt.conferenceID, tt.callID).Return(nil)

			req, _ := http.NewRequest("DELETE", tt.requestURI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

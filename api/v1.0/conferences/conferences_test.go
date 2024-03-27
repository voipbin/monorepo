package conferences

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_conferencesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		conferenceType cfconference.Type
		conferenceName string
		detail         string
		preActions     []fmaction.Action
		postActions    []fmaction.Action

		conference *cfconference.WebhookMessage
		request    []byte
	}{
		{
			"conference type",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			cfconference.TypeConference,
			"conference name",
			"conference detail",
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
			"pre/post actions",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			cfconference.TypeConference,
			"conference name",
			"conference detail",
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
				ID:     uuid.FromStringOrNil("62fc88ba-3fe9-11ec-8ebb-8f1ee591edec"),
				Type:   cfconference.TypeConference,
				Name:   "conference name",
				Detail: "conference detail",
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

			req, _ := http.NewRequest("POST", "/v1.0/conferences", bytes.NewBuffer(tt.request))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ConferenceCreate(
				req.Context(),
				&tt.agent,
				tt.conference.Type,
				tt.conference.Name,
				tt.conference.Detail,
				tt.conference.Timeout,
				tt.conference.Data,
				tt.conference.PreActions,
				tt.conference.PostActions,
			).Return(tt.conference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func TestConferencesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		id    uuid.UUID

		requestURI string

		conference *cfconference.WebhookMessage
	}{
		{
			"simple test",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
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

			req, _ := http.NewRequest("GET", tt.requestURI, nil)
			mockSvc.EXPECT().ConferenceGet(req.Context(), &tt.agent, tt.id).Return(tt.conference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		id       uuid.UUID

		responseConference *cfconference.WebhookMessage
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			reqQuery: "/v1.0/conferences/f49f8cc6-ac7f-11ea-91a3-e7103a41fa51",
			id:       uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),
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
			mockSvc.EXPECT().ConferenceDelete(req.Context(), &tt.agent, tt.id).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDPUT(t *testing.T) {

	tests := []struct {
		name          string
		agent         amagent.Agent
		requestTarget string
		request       []byte

		responseConference *cfconference.WebhookMessage

		expectID          uuid.UUID
		expectName        string
		expectDetail      string
		expectTimeout     int
		expectPreActions  []fmaction.Action
		expectPostActions []fmaction.Action
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/conferences/4363587a-92ff-11ed-8a2f-930de2e9aeae",
			[]byte(`{"name": "update name", "detail": "update detail", "timeout": 86400, "pre_actions": [{"type": "answer"}], "post_actions":[{"type": "hangup"}]}`),

			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("4363587a-92ff-11ed-8a2f-930de2e9aeae"),
			},

			uuid.FromStringOrNil("4363587a-92ff-11ed-8a2f-930de2e9aeae"),
			"update name",
			"update detail",
			86400,
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

			req, _ := http.NewRequest("PUT", tt.requestTarget, bytes.NewBuffer(tt.request))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ConferenceUpdate(req.Context(), &tt.agent, tt.expectID, tt.expectName, tt.expectDetail, tt.expectTimeout, tt.expectPreActions, tt.expectPostActions).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDRecordingStartPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		id    uuid.UUID

		requestURI string

		responseConference *cfconference.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("d2f603ce-910c-11ed-a360-0356e6882c63"),
			"/v1.0/conferences/d2f603ce-910c-11ed-a360-0356e6882c63/recording_start",

			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("d2f603ce-910c-11ed-a360-0356e6882c63"),
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

			req, _ := http.NewRequest("POST", tt.requestURI, nil)

			mockSvc.EXPECT().ConferenceRecordingStart(req.Context(), &tt.agent, tt.id).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDRecordingStopPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		id    uuid.UUID

		requestURI string

		responseConference *cfconference.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("f1f4d55c-910c-11ed-ad67-8768a5ad30d8"),
			"/v1.0/conferences/f1f4d55c-910c-11ed-ad67-8768a5ad30d8/recording_stop",

			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("f1f4d55c-910c-11ed-ad67-8768a5ad30d8"),
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

			req, _ := http.NewRequest("POST", tt.requestURI, nil)

			mockSvc.EXPECT().ConferenceRecordingStop(req.Context(), &tt.agent, tt.id).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDTranscribeStartPOST(t *testing.T) {

	tests := []struct {
		name     string
		agent    amagent.Agent
		id       uuid.UUID
		language string

		requestURI  string
		requestBody []byte

		responseConference *cfconference.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9"),
			"en-US",

			"/v1.0/conferences/af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9/transcribe_start",
			[]byte(`{"language": "en-US"}`),

			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9"),
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

			req, _ := http.NewRequest("POST", tt.requestURI, bytes.NewBuffer(tt.requestBody))

			mockSvc.EXPECT().ConferenceTranscribeStart(req.Context(), &tt.agent, tt.id, tt.language).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDTranscribeStopPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		id    uuid.UUID

		requestURI string

		responseConference *cfconference.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("af8db78c-98ec-11ed-9d8c-ffdf26e9202d"),
			"/v1.0/conferences/af8db78c-98ec-11ed-9d8c-ffdf26e9202d/transcribe_stop",

			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("af8db78c-98ec-11ed-9d8c-ffdf26e9202d"),
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

			req, _ := http.NewRequest("POST", tt.requestURI, nil)

			mockSvc.EXPECT().ConferenceTranscribeStop(req.Context(), &tt.agent, tt.id).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDMediaStreamGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectConferenceID  uuid.UUID
		expectEncapsulation string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			"/v1.0/conferences/fb250b7c-eb49-11ee-a795-1386bac55428/media_stream?encapsulation=rtp",

			uuid.FromStringOrNil("fb250b7c-eb49-11ee-a795-1386bac55428"),
			"rtp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConferenceMediaStreamStart(req.Context(), &tt.agent, tt.expectConferenceID, tt.expectEncapsulation, c.Writer, req).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

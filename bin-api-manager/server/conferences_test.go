package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	fmaction "monorepo/bin-flow-manager/models/action"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_conferencesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseConference *cfconference.WebhookMessage

		expectType        cfconference.Type
		expectName        string
		expectDetail      string
		expectData        map[string]interface{}
		expectTimeout     int
		expectPreActions  []fmaction.Action
		expectPostActions []fmaction.Action
		expectRes         string
	}{
		{
			name: "all data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences",
			reqBody:  []byte(`{"type": "conference", "name": "test name", "detail": "test detail", "data":{"key1": "val1", "key2": 2.1}, "timeout": 86400, "pre_actions": [{"type": "answer"}], "post_actions":[{"type": "hangup"}]}`),

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("ee1e90cc-ac7a-11ea-8474-e740530b4266"),
			},

			expectType:   cfconference.TypeConference,
			expectName:   "test name",
			expectDetail: "test detail",
			expectData: map[string]interface{}{
				"key1": "val1",
				"key2": 2.1,
			},
			expectTimeout: 86400,
			expectPreActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectPostActions: []fmaction.Action{
				{
					Type: fmaction.TypeHangup,
				},
			},
			expectRes: `{"id":"ee1e90cc-ac7a-11ea-8474-e740530b4266","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
		{
			name: "empty data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences",
			reqBody:  []byte(`{}`),

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("62fc88ba-3fe9-11ec-8ebb-8f1ee591edec"),
			},

			expectPreActions:  []fmaction.Action{},
			expectPostActions: []fmaction.Action{},
			expectRes:         `{"id":"62fc88ba-3fe9-11ec-8ebb-8f1ee591edec","customer_id":"00000000-0000-0000-0000-000000000000","type":"","status":"","name":"","detail":"","data":null,"timeout":0,"pre_actions":null,"post_actions":null,"conferencecall_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"transcribe_id":"00000000-0000-0000-0000-000000000000","transcribe_ids":null,"tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/conferences", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ConferenceCreate(
				req.Context(),
				&tt.agent,
				tt.expectType,
				tt.expectName,
				tt.expectDetail,
				tt.expectTimeout,
				tt.expectData,
				tt.expectPreActions,
				tt.expectPostActions,
			).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func TestConferencesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseConference *cfconference.WebhookMessage
		expectConferenceID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb",
			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb"),
			},
			expectConferenceID: uuid.FromStringOrNil("5ab35aba-ac3a-11ea-bcd7-4baa13dc0cdb"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ConferenceGet(req.Context(), &tt.agent, tt.expectConferenceID).Return(tt.responseConference, nil)

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

		responseConference *cfconference.WebhookMessage
		expectConferenceID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/f49f8cc6-ac7f-11ea-91a3-e7103a41fa51",

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),
			},

			expectConferenceID: uuid.FromStringOrNil("f49f8cc6-ac7f-11ea-91a3-e7103a41fa51"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().ConferenceDelete(req.Context(), &tt.agent, tt.expectConferenceID).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseConference *cfconference.WebhookMessage

		expectConferenceID uuid.UUID
		expectName         string
		expectDetail       string
		expectTimeout      int
		expectPreActions   []fmaction.Action
		expectPostActions  []fmaction.Action
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/4363587a-92ff-11ed-8a2f-930de2e9aeae",
			reqBody:  []byte(`{"name": "update name", "detail": "update detail", "timeout": 86400, "pre_actions": [{"type": "answer"}], "post_actions":[{"type": "hangup"}]}`),

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("4363587a-92ff-11ed-8a2f-930de2e9aeae"),
			},

			expectConferenceID: uuid.FromStringOrNil("4363587a-92ff-11ed-8a2f-930de2e9aeae"),
			expectName:         "update name",
			expectDetail:       "update detail",
			expectTimeout:      86400,
			expectPreActions: []fmaction.Action{
				{
					Type: "answer",
				},
			},
			expectPostActions: []fmaction.Action{
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
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ConferenceUpdate(req.Context(), &tt.agent, tt.expectConferenceID, tt.expectName, tt.expectDetail, tt.expectTimeout, tt.expectPreActions, tt.expectPostActions).Return(tt.responseConference, nil)

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

		reqQuery string

		responseConference *cfconference.WebhookMessage
		expectConferenceID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/d2f603ce-910c-11ed-a360-0356e6882c63/recording_start",

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("d2f603ce-910c-11ed-a360-0356e6882c63"),
			},
			expectConferenceID: uuid.FromStringOrNil("d2f603ce-910c-11ed-a360-0356e6882c63"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)

			mockSvc.EXPECT().ConferenceRecordingStart(req.Context(), &tt.agent, tt.expectConferenceID).Return(tt.responseConference, nil)

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

		reqQuery string

		responseConference *cfconference.WebhookMessage
		expectConferenceID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/f1f4d55c-910c-11ed-ad67-8768a5ad30d8/recording_stop",

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("f1f4d55c-910c-11ed-ad67-8768a5ad30d8"),
			},
			expectConferenceID: uuid.FromStringOrNil("f1f4d55c-910c-11ed-ad67-8768a5ad30d8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)

			mockSvc.EXPECT().ConferenceRecordingStop(req.Context(), &tt.agent, tt.expectConferenceID).Return(tt.responseConference, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencesIDTranscribeStartPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseConference *cfconference.WebhookMessage

		expectConferenceID uuid.UUID
		expectLanguage     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9/transcribe_start",
			reqBody:  []byte(`{"language": "en-US"}`),

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9"),
			},

			expectConferenceID: uuid.FromStringOrNil("af60d8b6-98ec-11ed-9e1b-ab94ae0c68d9"),
			expectLanguage:     "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().ConferenceTranscribeStart(req.Context(), &tt.agent, tt.expectConferenceID, tt.expectLanguage).Return(tt.responseConference, nil)

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

		reqQuery string

		responseConference *cfconference.WebhookMessage
		expectConferenceID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/conferences/af8db78c-98ec-11ed-9d8c-ffdf26e9202d/transcribe_stop",

			responseConference: &cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("af8db78c-98ec-11ed-9d8c-ffdf26e9202d"),
			},
			expectConferenceID: uuid.FromStringOrNil("af8db78c-98ec-11ed-9d8c-ffdf26e9202d"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)

			mockSvc.EXPECT().ConferenceTranscribeStop(req.Context(), &tt.agent, tt.expectConferenceID).Return(tt.responseConference, nil)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/conferences/fb250b7c-eb49-11ee-a795-1386bac55428/media_stream?encapsulation=rtp",

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
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

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

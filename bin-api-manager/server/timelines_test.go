package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetTimelinesResourceTypeResourceIdEvents(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseEvents    []*servicehandler.TimelineEvent
		responseNextToken string

		expectResourceType string
		expectResourceID   uuid.UUID
		expectPageSize     int
		expectPageToken    string
		expectRes          string
	}{
		{
			name: "valid request with calls resource type",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/calls/fe003a08-8f36-11ed-a01a-efb53befe93a/events?page_size=10",

			responseEvents: []*servicehandler.TimelineEvent{
				{
					Timestamp: "2024-01-15T10:30:00.000Z",
					EventType: "call_created",
					Data:      map[string]interface{}{"id": "fe003a08-8f36-11ed-a01a-efb53befe93a"},
				},
			},
			responseNextToken: "next-token",

			expectResourceType: "calls",
			expectResourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     10,
			expectPageToken:    "",
			expectRes:          `{"result":[{"timestamp":"2024-01-15T10:30:00.000Z","event_type":"call_created","data":{"id":"fe003a08-8f36-11ed-a01a-efb53befe93a"}}],"next_page_token":"next-token"}`,
		},
		{
			name: "valid request with conferences resource type",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/conferences/a1b2c3d4-8f36-11ed-a01a-efb53befe93a/events?page_size=20&page_token=some-token",

			responseEvents: []*servicehandler.TimelineEvent{
				{
					Timestamp: "2024-01-15T10:30:00.000Z",
					EventType: "conference_created",
					Data:      map[string]interface{}{"id": "a1b2c3d4-8f36-11ed-a01a-efb53befe93a"},
				},
			},
			responseNextToken: "",

			expectResourceType: "conferences",
			expectResourceID:   uuid.FromStringOrNil("a1b2c3d4-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     20,
			expectPageToken:    "some-token",
			expectRes:          `{"result":[{"timestamp":"2024-01-15T10:30:00.000Z","event_type":"conference_created","data":{"id":"a1b2c3d4-8f36-11ed-a01a-efb53befe93a"}}]}`,
		},
		{
			name: "valid request with flows resource type",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/flows/b2c3d4e5-8f36-11ed-a01a-efb53befe93a/events",

			responseEvents:    []*servicehandler.TimelineEvent{},
			responseNextToken: "",

			expectResourceType: "flows",
			expectResourceID:   uuid.FromStringOrNil("b2c3d4e5-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     100,
			expectPageToken:    "",
			expectRes:          `{"result":[]}`,
		},
		{
			name: "valid request with activeflows resource type",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/activeflows/c3d4e5f6-8f36-11ed-a01a-efb53befe93a/events",

			responseEvents: []*servicehandler.TimelineEvent{
				{
					Timestamp: "2024-01-15T10:30:00.000Z",
					EventType: "activeflow_created",
					Data:      map[string]interface{}{"id": "c3d4e5f6-8f36-11ed-a01a-efb53befe93a"},
				},
			},
			responseNextToken: "",

			expectResourceType: "activeflows",
			expectResourceID:   uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     100,
			expectPageToken:    "",
			expectRes:          `{"result":[{"timestamp":"2024-01-15T10:30:00.000Z","event_type":"activeflow_created","data":{"id":"c3d4e5f6-8f36-11ed-a01a-efb53befe93a"}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().TimelineEventList(
				req.Context(),
				&tt.agent,
				tt.expectResourceType,
				tt.expectResourceID,
				tt.expectPageSize,
				tt.expectPageToken,
			).Return(tt.responseEvents, tt.responseNextToken, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body.String())
			}
		})
	}
}

func Test_GetTimelinesResourceTypeResourceIdEvents_missing_agent(t *testing.T) {

	tests := []struct {
		name     string
		reqQuery string
	}{
		{
			name:     "missing agent returns 400",
			reqQuery: "/timelines/calls/fe003a08-8f36-11ed-a01a-efb53befe93a/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			// Note: No agent middleware added
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_GetTimelinesResourceTypeResourceIdEvents_invalid_resource_type(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
	}{
		{
			name: "invalid resource type returns 400",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/invalid_type/fe003a08-8f36-11ed-a01a-efb53befe93a/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_GetTimelinesResourceTypeResourceIdEvents_invalid_resource_id(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
	}{
		{
			name: "invalid resource id returns 400",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/calls/invalid-uuid/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_GetTimelinesResourceTypeResourceIdEvents_resource_not_found(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseErr error

		expectResourceType string
		expectResourceID   uuid.UUID
		expectPageSize     int
		expectPageToken    string
	}{
		{
			name: "resource not found returns 404",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/calls/fe003a08-8f36-11ed-a01a-efb53befe93a/events",

			responseErr: fmt.Errorf("not found"),

			expectResourceType: "calls",
			expectResourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     100,
			expectPageToken:    "",
		},
		{
			name: "permission denied returns 404",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/calls/fe003a08-8f36-11ed-a01a-efb53befe93a/events",

			responseErr: fmt.Errorf("user has no permission"),

			expectResourceType: "calls",
			expectResourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     100,
			expectPageToken:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().TimelineEventList(
				req.Context(),
				&tt.agent,
				tt.expectResourceType,
				tt.expectResourceID,
				tt.expectPageSize,
				tt.expectPageToken,
			).Return(nil, "", tt.responseErr)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, w.Code)
			}
		})
	}
}

func Test_GetTimelinesResourceTypeResourceIdEvents_internal_error(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseErr error

		expectResourceType string
		expectResourceID   uuid.UUID
		expectPageSize     int
		expectPageToken    string
	}{
		{
			name: "internal error returns 500",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/timelines/calls/fe003a08-8f36-11ed-a01a-efb53befe93a/events",

			responseErr: fmt.Errorf("internal error"),

			expectResourceType: "calls",
			expectResourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     100,
			expectPageToken:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().TimelineEventList(
				req.Context(),
				&tt.agent,
				tt.expectResourceType,
				tt.expectResourceID,
				tt.expectPageSize,
				tt.expectPageToken,
			).Return(nil, "", tt.responseErr)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusInternalServerError {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, w.Code)
			}
		})
	}
}

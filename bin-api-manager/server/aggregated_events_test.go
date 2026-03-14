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

func Test_GetAggregatedEvents(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseEvents    []*servicehandler.TimelineEvent
		responseNextToken string

		expectActiveflowID uuid.UUID
		expectCallID       uuid.UUID
		expectPageSize     int
		expectPageToken    string
		expectRes          string
	}{
		{
			name: "valid request with activeflow_id",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a&page_size=10",

			responseEvents: []*servicehandler.TimelineEvent{
				{
					Timestamp: "2024-01-15T10:30:00.000Z",
					EventType: "call_created",
					Data:      map[string]interface{}{"id": "fe003a08-8f36-11ed-a01a-efb53befe93a"},
				},
			},
			responseNextToken: "next-token",

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.Nil,
			expectPageSize:     10,
			expectPageToken:    "",
			expectRes:          `{"result":[{"timestamp":"2024-01-15T10:30:00.000Z","event_type":"call_created","data":{"id":"fe003a08-8f36-11ed-a01a-efb53befe93a"}}],"next_page_token":"next-token"}`,
		},
		{
			name: "valid request with call_id",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events?call_id=fe003a08-8f36-11ed-a01a-efb53befe93a&page_size=20&page_token=some-token",

			responseEvents: []*servicehandler.TimelineEvent{
				{
					Timestamp: "2024-01-15T10:30:00.000Z",
					EventType: "call_created",
					Data:      map[string]interface{}{"id": "fe003a08-8f36-11ed-a01a-efb53befe93a"},
				},
			},
			responseNextToken: "",

			expectActiveflowID: uuid.Nil,
			expectCallID:       uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			expectPageSize:     20,
			expectPageToken:    "some-token",
			expectRes:          `{"result":[{"timestamp":"2024-01-15T10:30:00.000Z","event_type":"call_created","data":{"id":"fe003a08-8f36-11ed-a01a-efb53befe93a"}}]}`,
		},
		{
			name: "pagination defaults when no page_size or page_token",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a",

			responseEvents:    []*servicehandler.TimelineEvent{},
			responseNextToken: "",

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.Nil,
			expectPageSize:     100,
			expectPageToken:    "",
			expectRes:          `{"result":[]}`,
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

			mockSvc.EXPECT().AggregatedEventList(
				req.Context(),
				&tt.agent,
				tt.expectActiveflowID,
				tt.expectCallID,
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

func Test_GetAggregatedEvents_missing_agent(t *testing.T) {

	tests := []struct {
		name     string
		reqQuery string
	}{
		{
			name:     "missing agent returns 400",
			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a",
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

func Test_GetAggregatedEvents_not_found(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseErr error

		expectActiveflowID uuid.UUID
		expectCallID       uuid.UUID
		expectPageSize     int
		expectPageToken    string
	}{
		{
			name: "not found returns 404",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a",

			responseErr: fmt.Errorf("not found"),

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.Nil,
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

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a",

			responseErr: fmt.Errorf("user has no permission"),

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.Nil,
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

			mockSvc.EXPECT().AggregatedEventList(
				req.Context(),
				&tt.agent,
				tt.expectActiveflowID,
				tt.expectCallID,
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

func Test_GetAggregatedEvents_validation_error(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseErr error

		expectActiveflowID uuid.UUID
		expectCallID       uuid.UUID
		expectPageSize     int
		expectPageToken    string
	}{
		{
			name: "neither activeflow_id nor call_id provided returns 400",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events",

			responseErr: fmt.Errorf("either activeflow_id or call_id is required"),

			expectActiveflowID: uuid.Nil,
			expectCallID:       uuid.Nil,
			expectPageSize:     100,
			expectPageToken:    "",
		},
		{
			name: "both activeflow_id and call_id provided returns 400",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a&call_id=fe003a08-8f36-11ed-a01a-efb53befe93a",

			responseErr: fmt.Errorf("only one of activeflow_id or call_id is allowed"),

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
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

			mockSvc.EXPECT().AggregatedEventList(
				req.Context(),
				&tt.agent,
				tt.expectActiveflowID,
				tt.expectCallID,
				tt.expectPageSize,
				tt.expectPageToken,
			).Return(nil, "", tt.responseErr)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_GetAggregatedEvents_internal_error(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseErr error

		expectActiveflowID uuid.UUID
		expectCallID       uuid.UUID
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

			reqQuery: "/aggregated-events?activeflow_id=c3d4e5f6-8f36-11ed-a01a-efb53befe93a",

			responseErr: fmt.Errorf("internal error"),

			expectActiveflowID: uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a"),
			expectCallID:       uuid.Nil,
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

			mockSvc.EXPECT().AggregatedEventList(
				req.Context(),
				&tt.agent,
				tt.expectActiveflowID,
				tt.expectCallID,
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

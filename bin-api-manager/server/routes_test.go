package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmroute "monorepo/bin-route-manager/models/route"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_routesGet_customer_id(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseRoutes []*rmroute.WebhookMessage

		expectPageSize   uint64
		expectPageToken  string
		expectCustomerID uuid.UUID
		expectRes        string
	}{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes?customer_id=de748080-7939-4625-a929-459c09a08448&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseRoutes: []*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("611c9384-5166-11ed-aee0-43c348138b55"),
					TMCreate: "2020-09-20T03:23:21.995000",
					Name:     "Route 1",
				},
			},

			expectPageSize:   10,
			expectPageToken:  "2020-09-20 03:23:20.995000",
			expectCustomerID: uuid.FromStringOrNil("de748080-7939-4625-a929-459c09a08448"),
			expectRes:        `{"result":[{"id":"611c9384-5166-11ed-aee0-43c348138b55","customer_id":"00000000-0000-0000-0000-000000000000","name":"Route 1","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes?customer_id=0e4bcfb0-f90a-4abf-83e1-08a4f0cd0260&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseRoutes: []*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("6158d6dc-5166-11ed-9a8c-7f1a71b3baaa"),
					TMCreate: "2020-09-20T03:23:21.995000",
					Name:     "Route 2",
				},
				{
					ID:       uuid.FromStringOrNil("618bdbea-5166-11ed-88a2-af045be175e7"),
					TMCreate: "2020-09-20T03:23:22.995000",
					Name:     "Route 3",
				},
				{
					ID:       uuid.FromStringOrNil("61bbd872-5166-11ed-83e9-5f5f0b484429"),
					TMCreate: "2020-09-20T03:23:23.995000",
					Name:     "Route 4",
				},
			},

			expectPageSize:   10,
			expectPageToken:  "2020-09-20 03:23:20.995000",
			expectCustomerID: uuid.FromStringOrNil("0e4bcfb0-f90a-4abf-83e1-08a4f0cd0260"),
			expectRes:        `{"result":[{"id":"6158d6dc-5166-11ed-9a8c-7f1a71b3baaa","customer_id":"00000000-0000-0000-0000-000000000000","name":"Route 2","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"618bdbea-5166-11ed-88a2-af045be175e7","customer_id":"00000000-0000-0000-0000-000000000000","name":"Route 3","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61bbd872-5166-11ed-83e9-5f5f0b484429","customer_id":"00000000-0000-0000-0000-000000000000","name":"Route 4","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().RouteGetsByCustomerID(req.Context(), &tt.agent, tt.expectCustomerID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseRoutes, nil)

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

func Test_routesGet_without_customer_id(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseRoutes []*rmroute.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseRoutes: []*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("93b9143a-68a2-11ee-b676-8718718cd43e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"93b9143a-68a2-11ee-b676-8718718cd43e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			reqQuery: "/routes?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseRoutes: []*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("941551f0-68a2-11ee-889c-ff0e92d76ad5"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("94455364-68a2-11ee-9d8f-d376b317364d"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("947bb1de-68a2-11ee-a56d-4f6139e966bd"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"941551f0-68a2-11ee-889c-ff0e92d76ad5","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"94455364-68a2-11ee-9d8f-d376b317364d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"947bb1de-68a2-11ee-a56d-4f6139e966bd","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().RouteList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseRoutes, nil)

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

func Test_routesPost(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseRoute *rmroute.WebhookMessage

		expectCustomerID uuid.UUID
		expectName       string
		expectDetail     string
		expectProviderID uuid.UUID
		expectPriority   int
		expectTarget     string
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes",
			reqBody:  []byte(`{"customer_id":"04303d61-d8c9-477c-8b54-254af0cb499f","name":"test name","detail":"test detail","provider_id":"a7efc236-5166-11ed-bdea-631379fb1515","priority":1,"target":"+82"}`),

			responseRoute: &rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("e1d75c98-5166-11ed-b2ff-1bb082f1fc25"),
			},

			expectCustomerID: uuid.FromStringOrNil("04303d61-d8c9-477c-8b54-254af0cb499f"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectProviderID: uuid.FromStringOrNil("a7efc236-5166-11ed-bdea-631379fb1515"),
			expectPriority:   1,
			expectTarget:     "+82",
			expectRes:        `{"id":"e1d75c98-5166-11ed-b2ff-1bb082f1fc25","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().RouteCreate(
				req.Context(),
				&tt.agent,
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
				tt.expectProviderID,
				tt.expectPriority,
				tt.expectTarget,
			).Return(tt.responseRoute, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_routesIDGet(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseRoute *rmroute.WebhookMessage

		expectRouteID uuid.UUID
		expectRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes/1c776852-5167-11ed-bf9a-eba39c6546e4",

			responseRoute: &rmroute.WebhookMessage{
				ID:       uuid.FromStringOrNil("1c776852-5167-11ed-bf9a-eba39c6546e4"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},

			expectRouteID: uuid.FromStringOrNil("1c776852-5167-11ed-bf9a-eba39c6546e4"),
			expectRes:     `{"id":"1c776852-5167-11ed-bf9a-eba39c6546e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().RouteGet(req.Context(), &tt.agent, tt.expectRouteID).Return(tt.responseRoute, nil)

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

func Test_routesIDDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseRoute *rmroute.WebhookMessage

		expectRouteID uuid.UUID
		expectRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),
				},
			},

			reqQuery: "/routes/4d1e5ab0-5167-11ed-98ff-f7ff08fc0833",

			responseRoute: &rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),
			},

			expectRouteID: uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),
			expectRes:     `{"id":"4d1e5ab0-5167-11ed-98ff-f7ff08fc0833","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().RouteDelete(req.Context(), &tt.agent, tt.expectRouteID).Return(tt.responseRoute, nil)

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

func Test_routesIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseRoute *rmroute.WebhookMessage

		expectRouteID    uuid.UUID
		expectName       string
		expectDetail     string
		expectProviderID uuid.UUID
		expectPriority   int
		expectTarget     string
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/routes/cd2b8926-5167-11ed-a158-ffb3472a3a4d",
			reqBody:  []byte(`{"name":"update name","detail":"update detail","provider_id":"cd58db88-5167-11ed-8d35-e3209648ccf8","priority":1,"target":"+82"}`),

			responseRoute: &rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			},

			expectRouteID:    uuid.FromStringOrNil("cd2b8926-5167-11ed-a158-ffb3472a3a4d"),
			expectName:       "update name",
			expectDetail:     "update detail",
			expectProviderID: uuid.FromStringOrNil("cd58db88-5167-11ed-8d35-e3209648ccf8"),
			expectPriority:   1,
			expectTarget:     "+82",
			expectRes:        `{"id":"169cbfe0-5162-11ed-9be1-872503f37e02","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().RouteUpdate(
				req.Context(),
				&tt.agent,
				tt.expectRouteID,
				tt.expectName,
				tt.expectDetail,
				tt.expectProviderID,
				tt.expectPriority,
				tt.expectTarget,
			).Return(tt.responseRoute, nil)

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

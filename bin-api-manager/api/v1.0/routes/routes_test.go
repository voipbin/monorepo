package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	rmroute "monorepo/bin-route-manager/models/route"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_routesGet_customer_id(t *testing.T) {

	tests := []struct {
		name string

		customer   amagent.Agent
		customerID uuid.UUID
		reqQuery   string
		pageSize   uint64
		pageToken  string

		resRoutes []*rmroute.WebhookMessage
		expectRes string
	}{
		{
			"1 item",

			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("de748080-7939-4625-a929-459c09a08448"),
			"/v1.0/routes?customer_id=de748080-7939-4625-a929-459c09a08448&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("611c9384-5166-11ed-aee0-43c348138b55"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"611c9384-5166-11ed-aee0-43c348138b55","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("0e4bcfb0-f90a-4abf-83e1-08a4f0cd0260"),
			"/v1.0/routes?customer_id=0e4bcfb0-f90a-4abf-83e1-08a4f0cd0260&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("6158d6dc-5166-11ed-9a8c-7f1a71b3baaa"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("618bdbea-5166-11ed-88a2-af045be175e7"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("61bbd872-5166-11ed-83e9-5f5f0b484429"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"6158d6dc-5166-11ed-9a8c-7f1a71b3baaa","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"618bdbea-5166-11ed-88a2-af045be175e7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61bbd872-5166-11ed-83e9-5f5f0b484429","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().RouteGetsByCustomerID(req.Context(), &tt.customer, tt.customerID, tt.pageSize, tt.pageToken).Return(tt.resRoutes, nil)

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
		name string

		customer  amagent.Agent
		reqQuery  string
		pageSize  uint64
		pageToken string

		resRoutes []*rmroute.WebhookMessage
		expectRes string
	}{
		{
			"1 item",

			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/routes?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmroute.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("93b9143a-68a2-11ee-b676-8718718cd43e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"93b9143a-68a2-11ee-b676-8718718cd43e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/routes?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmroute.WebhookMessage{
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
			`{"result":[{"id":"941551f0-68a2-11ee-889c-ff0e92d76ad5","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"94455364-68a2-11ee-9d8f-d376b317364d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"947bb1de-68a2-11ee-a56d-4f6139e966bd","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().RouteGets(req.Context(), &tt.customer, tt.pageSize, tt.pageToken).Return(tt.resRoutes, nil)

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
		name     string
		customer amagent.Agent

		reqQuery string
		reqBody  request.BodyRoutesPOST
		resRoute *rmroute.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/routes",
			request.BodyRoutesPOST{
				CustomerID: uuid.FromStringOrNil("04303d61-d8c9-477c-8b54-254af0cb499f"),
				Name:       "test name",
				Detail:     "test detail",
				ProviderID: uuid.FromStringOrNil("a7efc236-5166-11ed-bdea-631379fb1515"),
				Priority:   1,
				Target:     "+82",
			},
			&rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("e1d75c98-5166-11ed-b2ff-1bb082f1fc25"),
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().RouteCreate(
				req.Context(),
				&tt.customer,
				tt.reqBody.CustomerID,
				tt.reqBody.Name,
				tt.reqBody.Detail,
				tt.reqBody.ProviderID,
				tt.reqBody.Priority,
				tt.reqBody.Target,
			).Return(tt.resRoute, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_routesIDGet(t *testing.T) {

	tests := []struct {
		name     string
		customer amagent.Agent

		reqQuery string
		routeID  uuid.UUID

		resRoute *rmroute.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/routes/1c776852-5167-11ed-bf9a-eba39c6546e4",
			uuid.FromStringOrNil("1c776852-5167-11ed-bf9a-eba39c6546e4"),

			&rmroute.WebhookMessage{
				ID:       uuid.FromStringOrNil("1c776852-5167-11ed-bf9a-eba39c6546e4"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
			`{"id":"1c776852-5167-11ed-bf9a-eba39c6546e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().RouteGet(req.Context(), &tt.customer, tt.routeID).Return(tt.resRoute, nil)

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
		name     string
		customer amagent.Agent

		reqQuery string
		routeID  uuid.UUID

		responseRoute *rmroute.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),
				},
			},

			"/v1.0/routes/4d1e5ab0-5167-11ed-98ff-f7ff08fc0833",
			uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),

			&rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("4d1e5ab0-5167-11ed-98ff-f7ff08fc0833"),
			},

			`{"id":"4d1e5ab0-5167-11ed-98ff-f7ff08fc0833","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().RouteDelete(req.Context(), &tt.customer, tt.routeID).Return(tt.responseRoute, nil)

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
		name     string
		customer amagent.Agent

		reqQuery string
		reqBody  []byte

		routeID    uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *rmroute.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/routes/cd2b8926-5167-11ed-a158-ffb3472a3a4d",
			[]byte(`{"name":"update name","detail":"update detail","provider_id":"cd58db88-5167-11ed-8d35-e3209648ccf8","priority":1,"target":"+82"}`),

			uuid.FromStringOrNil("cd2b8926-5167-11ed-a158-ffb3472a3a4d"),
			"update name",
			"update detail",
			uuid.FromStringOrNil("cd58db88-5167-11ed-8d35-e3209648ccf8"),
			1,
			"+82",

			&rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			},

			`{"id":"169cbfe0-5162-11ed-9be1-872503f37e02","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().RouteUpdate(
				req.Context(),
				&tt.customer,
				tt.routeID,
				tt.routeName,
				tt.detail,
				tt.providerID,
				tt.priority,
				tt.target,
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

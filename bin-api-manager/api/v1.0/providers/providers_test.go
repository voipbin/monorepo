package providers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	rmprovider "monorepo/bin-route-manager/models/provider"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_providersGet(t *testing.T) {

	type test struct {
		name string

		agent     amagent.Agent
		reqQuery  string
		pageSize  uint64
		pageToken string

		resProviders []*rmprovider.WebhookMessage
		expectRes    string
	}

	tests := []test{
		{
			"1 item",

			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/providers?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmprovider.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("088b16ac-515f-11ed-a848-cb013a2391a9"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"088b16ac-515f-11ed-a848-cb013a2391a9","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/providers?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*rmprovider.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3829653a-515f-11ed-a1a8-6b5ca0211d65"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("38587906-515f-11ed-b4dc-877b6c706934"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("388af98a-515f-11ed-b16e-f3d7e7085feb"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"3829653a-515f-11ed-a1a8-6b5ca0211d65","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"38587906-515f-11ed-b4dc-877b6c706934","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"388af98a-515f-11ed-b16e-f3d7e7085feb","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ProviderGets(req.Context(), &tt.agent, tt.pageSize, tt.pageToken).Return(tt.resProviders, nil)

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

func Test_providersPost(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		reqBody     request.BodyProvidersPOST
		resProvider *rmprovider.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/providers",
			request.BodyProvidersPOST{
				Type:        rmprovider.TypeSIP,
				Hostname:    "test.com",
				TechPrefix:  "0001",
				TechPostfix: "1000",
				TechHeaders: map[string]string{
					"header_1": "val1",
					"header_2": "val2",
				},
				Name:   "test name",
				Detail: "test detail",
			},
			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("72fe03fa-6475-11ec-b559-0fdf19201178"),
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ProviderCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.Type,
				tt.reqBody.Hostname,
				tt.reqBody.TechPrefix,
				tt.reqBody.TechPostfix,
				tt.reqBody.TechHeaders,
				tt.reqBody.Name,
				tt.reqBody.Detail,
			).Return(tt.resProvider, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_providersIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery   string
		providerID uuid.UUID

		resProvider *rmprovider.WebhookMessage

		expectRes string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/providers/d091abe2-5160-11ed-b13c-57769429b0f0",
			uuid.FromStringOrNil("d091abe2-5160-11ed-b13c-57769429b0f0"),

			&rmprovider.WebhookMessage{
				ID:       uuid.FromStringOrNil("d091abe2-5160-11ed-b13c-57769429b0f0"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
			`{"id":"d091abe2-5160-11ed-b13c-57769429b0f0","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ProviderGet(req.Context(), &tt.agent, tt.providerID).Return(tt.resProvider, nil)

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

func Test_providersIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery   string
		providerID uuid.UUID

		responseProvider *rmprovider.WebhookMessage

		expectRes string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),
			},

			"/v1.0/providers/528bae9a-5161-11ed-b6c1-03e42a38600c",
			uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),

			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),
			},

			`{"id":"528bae9a-5161-11ed-b6c1-03e42a38600c","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ProviderDelete(req.Context(), &tt.agent, tt.providerID).Return(tt.responseProvider, nil)

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

func Test_providersIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		providerID   uuid.UUID
		providerType rmprovider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		responseProvider *rmprovider.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/providers/169cbfe0-5162-11ed-9be1-872503f37e02",
			[]byte(`{"type":"sip","hostname":"test.com","tech_prefix":"0001","tech_postfix":"1000","tech_headers":{"header_1":"val1","header_2":"val2"},"name":"update name", "detail":"update detail"}`),

			uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			rmprovider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header_1": "val1",
				"header_2": "val2",
			},
			"update name",
			"update detail",

			&rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			},

			`{"id":"169cbfe0-5162-11ed-9be1-872503f37e02","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().ProviderUpdate(req.Context(), &tt.agent, tt.providerID, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.providerName, tt.detail).Return(tt.responseProvider, nil)

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

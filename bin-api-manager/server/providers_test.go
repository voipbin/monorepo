package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmprovider "monorepo/bin-route-manager/models/provider"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_providersGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseProviders []*rmprovider.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/providers?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseProviders: []*rmprovider.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("088b16ac-515f-11ed-a848-cb013a2391a9"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"088b16ac-515f-11ed-a848-cb013a2391a9","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/providers?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseProviders: []*rmprovider.WebhookMessage{
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

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"3829653a-515f-11ed-a1a8-6b5ca0211d65","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"38587906-515f-11ed-b4dc-877b6c706934","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"388af98a-515f-11ed-b16e-f3d7e7085feb","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ProviderGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseProviders, nil)

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

		reqQuery string
		reqBody  []byte

		responseProvider *rmprovider.WebhookMessage

		expectType        rmprovider.Type
		expectHostname    string
		expectTechPrefix  string
		expectTechPostfix string
		expectTechHeaders map[string]string
		expectName        string
		expectDetail      string
		expectRes         string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/providers",
			reqBody:  []byte(`{"type":"sip","hostname":"test.com","tech_prefix":"0001","tech_postfix":"1000","tech_headers":{"header_1":"val1","header_2":"val2"},"name":"test name","detail":"test detail"}`),

			responseProvider: &rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("72fe03fa-6475-11ec-b559-0fdf19201178"),
			},

			expectType:        rmprovider.TypeSIP,
			expectHostname:    "test.com",
			expectTechPrefix:  "0001",
			expectTechPostfix: "1000",
			expectTechHeaders: map[string]string{
				"header_1": "val1",
				"header_2": "val2",
			},
			expectName:   "test name",
			expectDetail: "test detail",
			expectRes:    `{"id":"72fe03fa-6475-11ec-b559-0fdf19201178","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ProviderCreate(
				req.Context(),
				&tt.agent,
				tt.expectType,
				tt.expectHostname,
				tt.expectTechPrefix,
				tt.expectTechPostfix,
				tt.expectTechHeaders,
				tt.expectName,
				tt.expectDetail,
			).Return(tt.responseProvider, nil)

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

func Test_providersIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseProvider *rmprovider.WebhookMessage

		expectProviderID uuid.UUID
		expectRes        string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/providers/d091abe2-5160-11ed-b13c-57769429b0f0",

			responseProvider: &rmprovider.WebhookMessage{
				ID:       uuid.FromStringOrNil("d091abe2-5160-11ed-b13c-57769429b0f0"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},

			expectProviderID: uuid.FromStringOrNil("d091abe2-5160-11ed-b13c-57769429b0f0"),
			expectRes:        `{"id":"d091abe2-5160-11ed-b13c-57769429b0f0","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ProviderGet(req.Context(), &tt.agent, tt.expectProviderID).Return(tt.responseProvider, nil)

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

		reqQuery string

		responseProvider *rmprovider.WebhookMessage

		expectProviderID uuid.UUID
		expectRes        string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),
				},
			},

			reqQuery: "/providers/528bae9a-5161-11ed-b6c1-03e42a38600c",

			responseProvider: &rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),
			},

			expectProviderID: uuid.FromStringOrNil("528bae9a-5161-11ed-b6c1-03e42a38600c"),
			expectRes:        `{"id":"528bae9a-5161-11ed-b6c1-03e42a38600c","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ProviderDelete(req.Context(), &tt.agent, tt.expectProviderID).Return(tt.responseProvider, nil)

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

		expectProviderID   uuid.UUID
		expectProviderType rmprovider.Type
		expectHostname     string
		expectTechPrefix   string
		expectTechPostfix  string
		expectTechHeaders  map[string]string
		expectName         string
		expectDetail       string

		responseProvider *rmprovider.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/providers/169cbfe0-5162-11ed-9be1-872503f37e02",
			reqBody:  []byte(`{"type":"sip","hostname":"test.com","tech_prefix":"0001","tech_postfix":"1000","tech_headers":{"header_1":"val1","header_2":"val2"},"name":"update name", "detail":"update detail"}`),

			expectProviderID:   uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			expectProviderType: rmprovider.TypeSIP,
			expectHostname:     "test.com",
			expectTechPrefix:   "0001",
			expectTechPostfix:  "1000",
			expectTechHeaders: map[string]string{
				"header_1": "val1",
				"header_2": "val2",
			},
			expectName:   "update name",
			expectDetail: "update detail",

			responseProvider: &rmprovider.WebhookMessage{
				ID: uuid.FromStringOrNil("169cbfe0-5162-11ed-9be1-872503f37e02"),
			},

			expectRes: `{"id":"169cbfe0-5162-11ed-9be1-872503f37e02","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ProviderUpdate(req.Context(), &tt.agent, tt.expectProviderID, tt.expectProviderType, tt.expectHostname, tt.expectTechPrefix, tt.expectTechPostfix, tt.expectTechHeaders, tt.expectName, tt.expectDetail).Return(tt.responseProvider, nil)

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

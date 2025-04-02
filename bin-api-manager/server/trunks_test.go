package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_trunksPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseTrunk *rmtrunk.WebhookMessage

		expectName       string
		expectDetail     string
		expectDomainName string
		expectAuthTypes  []rmsipauth.AuthType
		expectUsername   string
		expectPassword   string
		expectAllowedIPs []string
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

			reqQuery: "/trunks",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","domain_name":"test","auth_types":["basic"],"username":"testusername","password":"testpassword","allowed_ips":["1.2.3.4"]}`),

			responseTrunk: &rmtrunk.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cb948fb8-db12-11ef-81f7-ef8092e4f1b5"),
				},
			},

			expectName:       "test name",
			expectDetail:     "test detail",
			expectDomainName: "test",
			expectAuthTypes: []rmsipauth.AuthType{
				rmsipauth.AuthTypeBasic,
			},
			expectUsername:   "testusername",
			expectPassword:   "testpassword",
			expectAllowedIPs: []string{"1.2.3.4"},
			expectRes:        `{"id":"cb948fb8-db12-11ef-81f7-ef8092e4f1b5","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_name":"","auth_types":null,"username":"","password":"","allowed_ips":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().TrunkCreate(req.Context(), &tt.agent, tt.expectName, tt.expectDetail, tt.expectDomainName, tt.expectAuthTypes, tt.expectUsername, tt.expectPassword, tt.expectAllowedIPs).Return(tt.responseTrunk, nil)

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

func Test_trunksGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTrunks []*rmtrunk.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/trunks?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseTrunks: []*rmtrunk.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("39c62e0a-db14-11ef-beab-071cd0697120"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3ac27c32-db14-11ef-b206-cf30e0a672b7"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"39c62e0a-db14-11ef-beab-071cd0697120","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_name":"","auth_types":null,"username":"","password":"","allowed_ips":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"3ac27c32-db14-11ef-b206-cf30e0a672b7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_name":"","auth_types":null,"username":"","password":"","allowed_ips":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TrunkGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseTrunks, nil)

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

func Test_TrunksIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTrunk *rmtrunk.WebhookMessage

		expectTrunkID uuid.UUID
		expectRes     string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/trunks/733b46f6-5588-11ee-b04e-c781770c2c87",

			responseTrunk: &rmtrunk.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("733b46f6-5588-11ee-b04e-c781770c2c87"),
					CustomerID: uuid.FromStringOrNil("d8eff4fa-7ff7-11ec-834f-679286ad908b"),
				},
			},

			expectTrunkID: uuid.FromStringOrNil("733b46f6-5588-11ee-b04e-c781770c2c87"),
			expectRes:     `{"id":"733b46f6-5588-11ee-b04e-c781770c2c87","customer_id":"d8eff4fa-7ff7-11ec-834f-679286ad908b","name":"","detail":"","domain_name":"","auth_types":null,"username":"","password":"","allowed_ips":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().TrunkGet(req.Context(), &tt.agent, tt.expectTrunkID).Return(tt.responseTrunk, nil)

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

func Test_TrunksIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		expectTrunkID    uuid.UUID
		expectName       string
		expectDetail     string
		expectAuthTypes  []rmsipauth.AuthType
		expectUsername   string
		expectPassword   string
		expectAllowedIPs []string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/trunks/6019ea72-5589-11ee-8b45-13603ef0a2d4",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","auth_types":["basic"],"username":"testusername","password":"testpassword","allowed_ips":["1.2.3.4"]}`),

			expectTrunkID: uuid.FromStringOrNil("6019ea72-5589-11ee-8b45-13603ef0a2d4"),
			expectName:    "test name",
			expectDetail:  "test detail",
			expectAuthTypes: []rmsipauth.AuthType{
				rmsipauth.AuthTypeBasic,
			},
			expectUsername: "testusername",
			expectPassword: "testpassword",
			expectAllowedIPs: []string{
				"1.2.3.4",
			},
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TrunkUpdateBasicInfo(req.Context(), &tt.agent, tt.expectTrunkID, tt.expectName, tt.expectDetail, tt.expectAuthTypes, tt.expectUsername, tt.expectPassword, tt.expectAllowedIPs).Return(&rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_trunksIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		expectTrunkID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery:      "/trunks/73ee8bce-558a-11ee-a104-5771d8493db0",
			expectTrunkID: uuid.FromStringOrNil("73ee8bce-558a-11ee-a104-5771d8493db0"),
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
			mockSvc.EXPECT().TrunkDelete(req.Context(), &tt.agent, tt.expectTrunkID).Return(&rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

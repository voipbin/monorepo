package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	caoutplan "monorepo/bin-campaign-manager/models/outplan"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_outplansPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutplan *caoutplan.WebhookMessage

		expectName         string
		expectDetail       string
		expectSource       *commonaddress.Address
		expectDialTimeout  int
		expectTryInterval  int
		expectMaxTryCount0 int
		expectMaxTryCount1 int
		expectMaxTryCount2 int
		expectMaxTryCount3 int
		expectMaxTryCount4 int
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outplans",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),

			responseOutplan: &caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1e701ed2-c649-11ec-97e4-87f868a3e3a9"),
				},
			},

			expectName:   "test name",
			expectDetail: "test detail",
			expectSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDialTimeout:  30000,
			expectTryInterval:  600000,
			expectMaxTryCount0: 5,
			expectMaxTryCount1: 5,
			expectMaxTryCount2: 5,
			expectMaxTryCount3: 5,
			expectMaxTryCount4: 5,
			expectRes:          `{"id":"1e701ed2-c649-11ec-97e4-87f868a3e3a9","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().OutplanCreate(
				req.Context(),
				&tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectSource,
				tt.expectDialTimeout,
				tt.expectTryInterval,
				tt.expectMaxTryCount0,
				tt.expectMaxTryCount1,
				tt.expectMaxTryCount2,
				tt.expectMaxTryCount3,
				tt.expectMaxTryCount4,
			).Return(tt.responseOutplan, nil)

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

func Test_outplansGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutplans []*caoutplan.WebhookMessage

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

			reqQuery: "/outplans?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseOutplans: []*caoutplan.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("891dceb2-c64b-11ec-ad40-4f3b7ab8bd4e"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"891dceb2-c64b-11ec-ad40-4f3b7ab8bd4e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outplans?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseOutplans: []*caoutplan.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b85b50fa-c64b-11ec-a17f-fb6cd8c28a0d"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b88bd6f8-c64b-11ec-a895-0f50245da5a9"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b8c11570-c64b-11ec-82f7-abb0350c1d7d"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"b85b50fa-c64b-11ec-a17f-fb6cd8c28a0d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"b88bd6f8-c64b-11ec-a895-0f50245da5a9","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"b8c11570-c64b-11ec-82f7-abb0350c1d7d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().OutplanGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseOutplans, nil)

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

func Test_outplansIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		rqeQuery string

		responseOutplan *caoutplan.WebhookMessage

		expectOutplanID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			rqeQuery: "/outplans/1b27088c-c64c-11ec-b7df-b37c8b4c4c13",

			responseOutplan: &caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1b27088c-c64c-11ec-b7df-b37c8b4c4c13"),
				},
			},

			expectOutplanID: uuid.FromStringOrNil("1b27088c-c64c-11ec-b7df-b37c8b4c4c13"),
			expectRes:       `{"id":"1b27088c-c64c-11ec-b7df-b37c8b4c4c13","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("GET", tt.rqeQuery, nil)
			mockSvc.EXPECT().OutplanGet(req.Context(), &tt.agent, tt.expectOutplanID).Return(tt.responseOutplan, nil)

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

func Test_outplansIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutplan *caoutplan.WebhookMessage

		expectOutplanID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outplans/3b58765e-c64c-11ec-a2c1-03acafdff2d7",

			responseOutplan: &caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3b58765e-c64c-11ec-a2c1-03acafdff2d7"),
				},
			},

			expectOutplanID: uuid.FromStringOrNil("3b58765e-c64c-11ec-a2c1-03acafdff2d7"),
			expectRes:       `{"id":"3b58765e-c64c-11ec-a2c1-03acafdff2d7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().OutplanDelete(req.Context(), &tt.agent, tt.expectOutplanID).Return(tt.responseOutplan, nil)

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

func Test_outplansIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOuplan *caoutplan.WebhookMessage

		expectOutplanID uuid.UUID
		expectName      string
		expectDetail    string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outplans/5ad57130-c64c-11ec-b131-a787ac641f8a",
			reqBody:  []byte(`{"name":"test name","detail":"test detail"}`),

			responseOuplan: &caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5ad57130-c64c-11ec-b131-a787ac641f8a"),
				},
			},

			expectOutplanID: uuid.FromStringOrNil("5ad57130-c64c-11ec-b131-a787ac641f8a"),
			expectName:      "test name",
			expectDetail:    "test detail",
			expectRes:       `{"id":"5ad57130-c64c-11ec-b131-a787ac641f8a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().OutplanUpdateBasicInfo(req.Context(), &tt.agent, tt.expectOutplanID, tt.expectName, tt.expectDetail).Return(tt.responseOuplan, nil)

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

func Test_outplansIDDialInfoPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutplan *caoutplan.WebhookMessage

		expectOutplanID    uuid.UUID
		expectSource       *commonaddress.Address
		expectDialTimeout  int
		expectTryInterval  int
		expectMaxTryCount0 int
		expectMaxTryCount1 int
		expectMaxTryCount2 int
		expectMaxTryCount3 int
		expectMaxTryCount4 int
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outplans/d94e07e8-c64c-11ec-9e9d-8b700336c5ef/dial_info",
			reqBody:  []byte(`{"source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),

			responseOutplan: &caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d94e07e8-c64c-11ec-9e9d-8b700336c5ef"),
				},
			},

			expectOutplanID: uuid.FromStringOrNil("d94e07e8-c64c-11ec-9e9d-8b700336c5ef"),
			expectSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDialTimeout:  30000,
			expectTryInterval:  600000,
			expectMaxTryCount0: 5,
			expectMaxTryCount1: 5,
			expectMaxTryCount2: 5,
			expectMaxTryCount3: 5,
			expectMaxTryCount4: 5,
			expectRes:          `{"id":"d94e07e8-c64c-11ec-9e9d-8b700336c5ef","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().OutplanUpdateDialInfo(
				req.Context(),
				&tt.agent,
				tt.expectOutplanID,
				tt.expectSource,
				tt.expectDialTimeout,
				tt.expectTryInterval,
				tt.expectMaxTryCount0,
				tt.expectMaxTryCount1,
				tt.expectMaxTryCount2,
				tt.expectMaxTryCount3,
				tt.expectMaxTryCount4,
			).Return(tt.responseOutplan, nil)

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

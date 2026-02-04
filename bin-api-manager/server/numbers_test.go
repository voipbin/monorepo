package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	nmnumber "monorepo/bin-number-manager/models/number"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func TestNumbersGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseNumbers []*nmnumber.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers?page_size=10&page_token=2021-03-02T03:23:20.995000Z",
			reqBody:  []byte(`{"pagination":{"page_size":10,"page_token":"2021-03-02T03:23:20.995000Z"}}`),

			responseNumbers: []*nmnumber.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("31ee638c-7b23-11eb-858a-33e73c4f82f7"),
					},
				},
			},
			expectPageSize:  10,
			expectPageToken: "2021-03-02T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"31ee638c-7b23-11eb-858a-33e73c4f82f7","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().NumberList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseNumbers, nil)

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

func Test_NumbersIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseNumber *nmnumber.WebhookMessage

		expectNumberID uuid.UUID
		expectRes      string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers/3ab6711c-7be6-11eb-8da6-d31a9f3d45a6",

			responseNumber: &nmnumber.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
				},
			},

			expectNumberID: uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
			expectRes:      `{"id":"3ab6711c-7be6-11eb-8da6-d31a9f3d45a6","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().NumberGet(req.Context(), &tt.agent, tt.expectNumberID).Return(tt.responseNumber, nil)

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

func TestNumbersIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseNumber *nmnumber.WebhookMessage

		expectNumberID uuid.UUID
		expectRes      string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers/d905c26e-7be6-11eb-b92a-ab4802b4bde3",

			responseNumber: &nmnumber.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
				},
			},

			expectNumberID: uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
			expectRes:      `{"id":"d905c26e-7be6-11eb-b92a-ab4802b4bde3","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().NumberDelete(req.Context(), &tt.agent, tt.expectNumberID).Return(tt.responseNumber, nil)

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

func TestNumbersPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseNumber *nmnumber.WebhookMessage

		expectNumber        string
		expectCallFlowID    uuid.UUID
		expectMessageFlowID uuid.UUID
		expectName          string
		expectDetail        string
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers",
			reqBody:  []byte(`{"number":"+821021656521","call_flow_id":"7762e356-88b1-11ec-bb0c-7f21b7cad172","message_flow_id":"354120a2-d938-11ef-a7fa-a37e9ed87b6c","name":"test name","detail":"test detail"}`),

			responseNumber: &nmnumber.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b244d2b6-d937-11ef-9ac7-2bcab0184b07"),
				},
			},

			expectNumber:        "+821021656521",
			expectCallFlowID:    uuid.FromStringOrNil("7762e356-88b1-11ec-bb0c-7f21b7cad172"),
			expectMessageFlowID: uuid.FromStringOrNil("354120a2-d938-11ef-a7fa-a37e9ed87b6c"),
			expectName:          "test name",
			expectDetail:        "test detail",
			expectRes:           `{"id":"b244d2b6-d937-11ef-9ac7-2bcab0184b07","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().NumberCreate(req.Context(), &tt.agent, tt.expectNumber, tt.expectCallFlowID, tt.expectMessageFlowID, tt.expectName, tt.expectDetail).Return(tt.responseNumber, nil)

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

func TestNumbersIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseNumber *nmnumber.WebhookMessage

		expectNumberID      uuid.UUID
		expectCallFlowID    uuid.UUID
		expectMessageFlowID uuid.UUID
		expectName          string
		expectDetail        string
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers/4e1a6702-7c60-11eb-bca2-3fd92181c652",
			reqBody:  []byte(`{"call_flow_id":"e2263f7a-2ca3-11ee-82b7-97de2fb4a790","message_flow_id":"e26b0eb6-2ca3-11ee-b7ce-d36a5a962472","name":"test name","detail":"test detail"}`),

			expectNumberID: uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),

			responseNumber: &nmnumber.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),
				},
			},

			expectCallFlowID:    uuid.FromStringOrNil("e2263f7a-2ca3-11ee-82b7-97de2fb4a790"),
			expectMessageFlowID: uuid.FromStringOrNil("e26b0eb6-2ca3-11ee-b7ce-d36a5a962472"),
			expectName:          "test name",
			expectDetail:        "test detail",
			expectRes:           `{"id":"4e1a6702-7c60-11eb-bca2-3fd92181c652","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().NumberUpdate(req.Context(), &tt.agent, tt.expectNumberID, tt.expectCallFlowID, tt.expectMessageFlowID, tt.expectName, tt.expectDetail).Return(tt.responseNumber, nil)
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

func TestNumbersIDFlowIDsPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseNumber *nmnumber.WebhookMessage

		expectNumberID      uuid.UUID
		expectCallFlowID    uuid.UUID
		expectMessageFlowID uuid.UUID
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/numbers/a440c6b8-94cd-11ec-a524-af82f0c3ee68/flow_ids",
			reqBody:  []byte(`{"call_flow_id":"b6161d70-94cd-11ec-b56c-bb1a417ae104","message_flow_id":"6e7ecc24-a881-11ec-bb4f-4b5822260cbe"}`),

			responseNumber: &nmnumber.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a440c6b8-94cd-11ec-a524-af82f0c3ee68"),
				},
			},

			expectNumberID:      uuid.FromStringOrNil("a440c6b8-94cd-11ec-a524-af82f0c3ee68"),
			expectCallFlowID:    uuid.FromStringOrNil("b6161d70-94cd-11ec-b56c-bb1a417ae104"),
			expectMessageFlowID: uuid.FromStringOrNil("6e7ecc24-a881-11ec-bb4f-4b5822260cbe"),
			expectRes:           `{"id":"a440c6b8-94cd-11ec-a524-af82f0c3ee68","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().NumberUpdateFlowIDs(req.Context(), &tt.agent, tt.expectNumberID, tt.expectCallFlowID, tt.expectMessageFlowID).Return(tt.responseNumber, nil)
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

func Test_NumbersRenewPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseNumbers []*nmnumber.WebhookMessage

		expectTMRenew string
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

			reqQuery: "/numbers/renew",
			reqBody:  []byte(`{"tm_renew":"2023-04-06T14:54:24.652558Z"}`),

			responseNumbers: []*nmnumber.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c2998386-1634-11ee-993a-37ac8d7a675d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c2e1ff3a-1634-11ee-bcc7-9f2a231b7b8a"),
					},
				},
			},

			expectTMRenew: "2023-04-06T14:54:24.652558Z",
			expectRes:     (`[{"id":"c2998386-1634-11ee-993a-37ac8d7a675d","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"c2e1ff3a-1634-11ee-bcc7-9f2a231b7b8a","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}]`),
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

			mockSvc.EXPECT().NumberRenew(req.Context(), &tt.agent, tt.expectTMRenew).Return(tt.responseNumbers, nil)
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

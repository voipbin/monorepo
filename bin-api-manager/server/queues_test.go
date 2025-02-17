package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_queuesGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseQueues []*qmqueue.WebhookMessage

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

			reqQuery: "/queues?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseQueues: []*qmqueue.WebhookMessage{
				{
					ID:          uuid.FromStringOrNil("f188b7aa-6476-11ec-a130-03a796c9e1e4"),
					TagIDs:      []uuid.UUID{},
					WaitActions: []fmaction.Action{},

					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"f188b7aa-6476-11ec-a130-03a796c9e1e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/queues?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseQueues: []*qmqueue.WebhookMessage{
				{
					ID:          uuid.FromStringOrNil("f1ad64a6-6476-11ec-a650-cf22de7273e6"),
					TagIDs:      []uuid.UUID{},
					WaitActions: []fmaction.Action{},

					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-09-20T03:23:21.995000",
				},
				{
					ID:          uuid.FromStringOrNil("f1d22dd6-6476-11ec-84e0-676f11515eed"),
					TagIDs:      []uuid.UUID{},
					WaitActions: []fmaction.Action{},

					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-09-20T03:23:22.995000",
				},
				{
					ID:          uuid.FromStringOrNil("f1fd30c6-6476-11ec-8b55-7f9c5b9550b7"),
					TagIDs:      []uuid.UUID{},
					WaitActions: []fmaction.Action{},

					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"f1ad64a6-6476-11ec-a650-cf22de7273e6","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"f1d22dd6-6476-11ec-84e0-676f11515eed","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"f1fd30c6-6476-11ec-8b55-7f9c5b9550b7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().QueueGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseQueues, nil)

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

func Test_queuesPost(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseQueue *qmqueue.WebhookMessage

		expectName           string
		expectDetail         string
		expectRoutingMethod  qmqueue.RoutingMethod
		expectTagIDs         []uuid.UUID
		expectWaitActions    []fmaction.Action
		expectWaitTimeout    int
		expectServiceTimeout int
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/queues",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","routing_method":"random","tag_ids":["296b096c-6476-11ec-8fc0-2f39371fef93"],"wait_actions":[{"type":"answer"}],"wait_timeout":10000,"service_timeout":100000}`),

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("72fe03fa-6475-11ec-b559-0fdf19201178"),
			},

			expectName:          "test name",
			expectDetail:        "test detail",
			expectRoutingMethod: qmqueue.RoutingMethodRandom,
			expectTagIDs: []uuid.UUID{
				uuid.FromStringOrNil("296b096c-6476-11ec-8fc0-2f39371fef93"),
			},
			expectWaitActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectWaitTimeout:    10000,
			expectServiceTimeout: 100000,
			expectRes:            `{"id":"72fe03fa-6475-11ec-b559-0fdf19201178","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().QueueCreate(
				req.Context(),
				&tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectRoutingMethod,
				tt.expectTagIDs,
				tt.expectWaitActions,
				tt.expectWaitTimeout,
				tt.expectServiceTimeout,
			).Return(tt.responseQueue, nil)

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

func Test_queuesIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseQueue *qmqueue.WebhookMessage

		expectQueueID uuid.UUID
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

			reqQuery: "/queues/395518ca-830a-11eb-badc-b3582bc51917",

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
			},

			expectQueueID: uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
			expectRes:     `{"id":"395518ca-830a-11eb-badc-b3582bc51917","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().QueueGet(req.Context(), &tt.agent, tt.expectQueueID).Return(tt.responseQueue, nil)
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

func Test_queuesIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseQueue *qmqueue.WebhookMessage

		expectQueueID uuid.UUID
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

			reqQuery: "/queues/5842d88a-6478-11ec-92cc-7fb5eb5d5e5a",

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("5842d88a-6478-11ec-92cc-7fb5eb5d5e5a"),
			},

			expectQueueID: uuid.FromStringOrNil("5842d88a-6478-11ec-92cc-7fb5eb5d5e5a"),
			expectRes:     `{"id":"5842d88a-6478-11ec-92cc-7fb5eb5d5e5a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().QueueDelete(req.Context(), &tt.agent, tt.expectQueueID).Return(tt.responseQueue, nil)
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

func Test_queuesIDPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseQueue *qmqueue.WebhookMessage

		expectQueueID        uuid.UUID
		expectQueueName      string
		expectDetail         string
		expectRoutingMethod  qmqueue.RoutingMethod
		expectTagIDs         []uuid.UUID
		expectWaitActions    []fmaction.Action
		expectTimeoutWait    int
		expectTimeoutService int
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/queues/39a61292-6479-11ec-8cee-d7ba44bf24ac",
			reqBody:  []byte(`{"name":"new name","detail":"new detail","routing_method":"random","tag_ids":["7e1be274-4a89-11ee-84ec-5b122e282794","7e762acc-4a89-11ee-9c08-43e00aea3bd6"],"wait_actions":[{"type":"answer"}],"wait_timeout":60000,"service_timeout":6000000}`),

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("39a61292-6479-11ec-8cee-d7ba44bf24ac"),
			},

			expectQueueID:       uuid.FromStringOrNil("39a61292-6479-11ec-8cee-d7ba44bf24ac"),
			expectQueueName:     "new name",
			expectDetail:        "new detail",
			expectRoutingMethod: qmqueue.RoutingMethodRandom,
			expectTagIDs: []uuid.UUID{
				uuid.FromStringOrNil("7e1be274-4a89-11ee-84ec-5b122e282794"),
				uuid.FromStringOrNil("7e762acc-4a89-11ee-9c08-43e00aea3bd6"),
			},
			expectWaitActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectTimeoutWait:    60000,
			expectTimeoutService: 6000000,
			expectRes:            `{"id":"39a61292-6479-11ec-8cee-d7ba44bf24ac","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().QueueUpdate(req.Context(), &tt.agent, tt.expectQueueID, tt.expectQueueName, tt.expectDetail, tt.expectRoutingMethod, tt.expectTagIDs, tt.expectWaitActions, tt.expectTimeoutWait, tt.expectTimeoutService).Return(tt.responseQueue, nil)
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

func Test_queuesIDTagIDsPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseQueue *qmqueue.WebhookMessage

		expectQueueID uuid.UUID
		expectTagIDs  []uuid.UUID
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

			reqQuery: "/queues/9ec11e74-6479-11ec-8956-9b1c6c142f77/tag_ids",
			reqBody:  []byte(`{"tag_ids":["aa740178-6479-11ec-879d-ab827778d4dd"]}`),

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			},

			expectQueueID: uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			expectTagIDs: []uuid.UUID{
				uuid.FromStringOrNil("aa740178-6479-11ec-879d-ab827778d4dd"),
			},
			expectRes: `{"id":"9ec11e74-6479-11ec-8956-9b1c6c142f77","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().QueueUpdateTagIDs(req.Context(), &tt.agent, tt.expectQueueID, tt.expectTagIDs).Return(tt.responseQueue, nil)
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

func Test_queuesIDRoutingMethodPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseQueue *qmqueue.WebhookMessage

		expectQueueID       uuid.UUID
		expectRoutingMethod qmqueue.RoutingMethod
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

			reqQuery: "/queues/9ec11e74-6479-11ec-8956-9b1c6c142f77/routing_method",
			reqBody:  []byte(`{"routing_method":"random"}`),

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			},

			expectQueueID:       uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			expectRoutingMethod: qmqueue.RoutingMethodRandom,
			expectRes:           `{"id":"9ec11e74-6479-11ec-8956-9b1c6c142f77","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().QueueUpdateRoutingMethod(req.Context(), &tt.agent, tt.expectQueueID, tt.expectRoutingMethod).Return(tt.responseQueue, nil)

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

func Test_queuesIDActionsPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseQueue *qmqueue.WebhookMessage

		expectQueueID        uuid.UUID
		expectWaitActions    []fmaction.Action
		expectTimeoutWait    int
		expectTimeoutService int

		expectRes string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			reqQuery: "/queues/70665304-647a-11ec-a5ca-4746cc95b189/actions",
			reqBody:  []byte(`{"wait_actions":[{"type":"answer"}],"timeout_wait":10000,"timeout_service":100000}`),

			responseQueue: &qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("70665304-647a-11ec-a5ca-4746cc95b189"),
			},

			expectQueueID: uuid.FromStringOrNil("70665304-647a-11ec-a5ca-4746cc95b189"),
			expectWaitActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectTimeoutWait:    10000,
			expectTimeoutService: 100000,
			expectRes:            `{"id":"70665304-647a-11ec-a5ca-4746cc95b189","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().QueueUpdateActions(req.Context(), &tt.agent, tt.expectQueueID, tt.expectWaitActions, tt.expectTimeoutWait, tt.expectTimeoutService).Return(tt.responseQueue, nil)
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

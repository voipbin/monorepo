package queues

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"

	qmqueue "monorepo/bin-queue-manager/models/queue"

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

func Test_queuesGet(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		req       request.ParamQueuesGET
		resCalls  []*qmqueue.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.ParamQueuesGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*qmqueue.WebhookMessage{
				{
					ID:          uuid.FromStringOrNil("f188b7aa-6476-11ec-a130-03a796c9e1e4"),
					TagIDs:      []uuid.UUID{},
					WaitActions: []fmaction.Action{},

					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"f188b7aa-6476-11ec-a130-03a796c9e1e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.ParamQueuesGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*qmqueue.WebhookMessage{
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
			`{"result":[{"id":"f1ad64a6-6476-11ec-a650-cf22de7273e6","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"f1d22dd6-6476-11ec-84e0-676f11515eed","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"f1fd30c6-6476-11ec-8b55-7f9c5b9550b7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			reqQuery := fmt.Sprintf("/v1.0/queues?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().QueueGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resCalls, nil)

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
		name     string
		agent    amagent.Agent
		req      request.BodyQueuesPOST
		resQueue *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.BodyQueuesPOST{
				Name:          "name",
				Detail:        "detail",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("296b096c-6476-11ec-8fc0-2f39371fef93"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitTimeout:    10000,
				ServiceTimeout: 100000,
			},
			&qmqueue.WebhookMessage{
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
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/queues", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().QueueCreate(
				req.Context(),
				&tt.agent,
				tt.req.Name,
				tt.req.Detail,
				tt.req.RoutingMethod,
				tt.req.TagIDs,
				tt.req.WaitActions,
				tt.req.WaitTimeout,
				tt.req.ServiceTimeout,
			).Return(tt.resQueue, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_queuesIDGet(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		resQueue  *qmqueue.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&qmqueue.WebhookMessage{
				ID:          uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
				TagIDs:      []uuid.UUID{},
				WaitActions: []fmaction.Action{},

				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2020-09-20T03:23:21.995000",
			},
			`{"id":"395518ca-830a-11eb-badc-b3582bc51917","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
		},
		{
			"webhook",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&qmqueue.WebhookMessage{
				ID:          uuid.FromStringOrNil("9e6e2dbe-830a-11eb-8fb0-cf5ab9cac353"),
				TagIDs:      []uuid.UUID{},
				WaitActions: []fmaction.Action{},

				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2020-09-20T03:23:21.995000",
			},
			`{"id":"9e6e2dbe-830a-11eb-8fb0-cf5ab9cac353","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":[],"wait_actions":[],"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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

			reqQuery := fmt.Sprintf("/v1.0/queues/%s", tt.resQueue.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().QueueGet(req.Context(), &tt.agent, tt.resQueue.ID).Return(tt.resQueue, nil)
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
		name    string
		agent   amagent.Agent
		queueID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("5842d88a-6478-11ec-92cc-7fb5eb5d5e5a"),
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

			reqQuery := fmt.Sprintf("/v1.0/queues/%s", tt.queueID)
			req, _ := http.NewRequest("DELETE", reqQuery, nil)

			mockSvc.EXPECT().QueueDelete(req.Context(), &tt.agent, tt.queueID).Return(&qmqueue.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuesIDPut(t *testing.T) {

	type test struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		queueID        uuid.UUID
		queueName      string
		detail         string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/queues/39a61292-6479-11ec-8cee-d7ba44bf24ac",
			[]byte(`{"name":"new name","detail":"new detail","routing_method":"random","tag_ids":["7e1be274-4a89-11ee-84ec-5b122e282794","7e762acc-4a89-11ee-9c08-43e00aea3bd6"],"wait_actions":[{"type":"answer"}],"wait_timeout":60000,"service_timeout":6000000}`),

			uuid.FromStringOrNil("39a61292-6479-11ec-8cee-d7ba44bf24ac"),
			"new name",
			"new detail",
			qmqueue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("7e1be274-4a89-11ee-84ec-5b122e282794"),
				uuid.FromStringOrNil("7e762acc-4a89-11ee-9c08-43e00aea3bd6"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			60000,
			6000000,
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
			mockSvc.EXPECT().QueueUpdate(req.Context(), &tt.agent, tt.queueID, tt.queueName, tt.detail, tt.routingMethod, tt.tagIDs, tt.waitActions, tt.timeoutWait, tt.timeoutService).Return(&qmqueue.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuesIDTagIDsPut(t *testing.T) {

	type test struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		queueID uuid.UUID
		tagIDs  []uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/queues/9ec11e74-6479-11ec-8956-9b1c6c142f77/tag_ids",
			[]byte(`{"tag_ids":["aa740178-6479-11ec-879d-ab827778d4dd"]}`),

			uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			[]uuid.UUID{
				uuid.FromStringOrNil("aa740178-6479-11ec-879d-ab827778d4dd"),
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			mockSvc.EXPECT().QueueUpdateTagIDs(req.Context(), &tt.agent, tt.queueID, tt.tagIDs).Return(&qmqueue.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuesIDRoutingMethodPut(t *testing.T) {

	type test struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		queueID       uuid.UUID
		routingMethod qmqueue.RoutingMethod
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/queues/9ec11e74-6479-11ec-8956-9b1c6c142f77/routing_method",
			[]byte(`{"routing_method":"random"}`),

			uuid.FromStringOrNil("9ec11e74-6479-11ec-8956-9b1c6c142f77"),
			qmqueue.RoutingMethodRandom,
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
			mockSvc.EXPECT().QueueUpdateRoutingMethod(req.Context(), &tt.agent, tt.queueID, tt.routingMethod).Return(&qmqueue.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuesIDActionsPut(t *testing.T) {

	type test struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		queueID        uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/queues/70665304-647a-11ec-a5ca-4746cc95b189/actions",
			[]byte(`{"wait_actions":[{"type":"answer"}],"timeout_wait":10000,"timeout_service":100000}`),

			uuid.FromStringOrNil("70665304-647a-11ec-a5ca-4746cc95b189"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,
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
			mockSvc.EXPECT().QueueUpdateActions(req.Context(), &tt.agent, tt.queueID, tt.waitActions, tt.timeoutWait, tt.timeoutService).Return(&qmqueue.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

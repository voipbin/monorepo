package queuecalls

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

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

func Test_queuecallsGet(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		req       request.ParamQueuecallsGET
		resCalls  []*qmqueuecall.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.ParamQueuecallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*qmqueuecall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("63b75166-4b2e-11ee-9664-e3e0b9c5de8e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"63b75166-4b2e-11ee-9664-e3e0b9c5de8e","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.ParamQueuecallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*qmqueuecall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("0e6061a8-4b2e-11ee-85d4-b366dd061d10"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("f1d22dd6-6476-11ec-84e0-676f11515eed"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("f1fd30c6-6476-11ec-8b55-7f9c5b9550b7"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"0e6061a8-4b2e-11ee-85d4-b366dd061d10","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""},{"id":"f1d22dd6-6476-11ec-84e0-676f11515eed","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:22.995000","tm_service":"","tm_update":"","tm_delete":""},{"id":"f1fd30c6-6476-11ec-8b55-7f9c5b9550b7","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:23.995000","tm_service":"","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			reqQuery := fmt.Sprintf("/v1.0/queuecalls?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().QueuecallGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resCalls, nil)

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

func Test_queuecallsIDGet(t *testing.T) {

	type test struct {
		name         string
		agent        amagent.Agent
		resQueuecall *qmqueuecall.WebhookMessage
		expectRes    string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			&qmqueuecall.WebhookMessage{
				ID:       uuid.FromStringOrNil("7d54d626-1681-11ed-ab05-473fa9aa2542"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
			`{"id":"7d54d626-1681-11ed-ab05-473fa9aa2542","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""}`,
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

			reqQuery := fmt.Sprintf("/v1.0/queuecalls/%s", tt.resQueuecall.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().QueuecallGet(req.Context(), &tt.agent, tt.resQueuecall.ID).Return(tt.resQueuecall, nil)
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

func Test_queuecallsIDDelete(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		queuecallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("a275df90-1681-11ed-a021-c3f295fc9257"),
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

			reqQuery := fmt.Sprintf("/v1.0/queuecalls/%s", tt.queuecallID)
			req, _ := http.NewRequest("DELETE", reqQuery, nil)

			mockSvc.EXPECT().QueuecallDelete(req.Context(), &tt.agent, tt.queuecallID).Return(&qmqueuecall.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuecallsIDKickPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		requQuery         string
		expectQueuecallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/queuecalls/72c9dfb0-bcbe-11ed-853f-7f662faaee5b/kick",
			uuid.FromStringOrNil("72c9dfb0-bcbe-11ed-853f-7f662faaee5b"),
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

			req, _ := http.NewRequest("POST", tt.requQuery, nil)

			mockSvc.EXPECT().QueuecallKick(req.Context(), &tt.agent, tt.expectQueuecallID).Return(&qmqueuecall.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_queuecallsReferenceIDIDKickPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		requQuery         string
		expectReferenceID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/queuecalls/reference_id/e01d78ce-bcbe-11ed-8164-f3c4a472391e/kick",
			uuid.FromStringOrNil("e01d78ce-bcbe-11ed-8164-f3c4a472391e"),
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

			req, _ := http.NewRequest("POST", tt.requQuery, nil)

			mockSvc.EXPECT().QueuecallKickByReferenceID(req.Context(), &tt.agent, tt.expectReferenceID).Return(&qmqueuecall.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

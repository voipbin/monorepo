package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_queuecallsGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCalls []*qmqueuecall.WebhookMessage

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

			reqQuery: "/queuecalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseCalls: []*qmqueuecall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("63b75166-4b2e-11ee-9664-e3e0b9c5de8e"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"63b75166-4b2e-11ee-9664-e3e0b9c5de8e","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/queuecalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseCalls: []*qmqueuecall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("0e6061a8-4b2e-11ee-85d4-b366dd061d10"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f1d22dd6-6476-11ec-84e0-676f11515eed"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f1fd30c6-6476-11ec-8b55-7f9c5b9550b7"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"0e6061a8-4b2e-11ee-85d4-b366dd061d10","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""},{"id":"f1d22dd6-6476-11ec-84e0-676f11515eed","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:22.995000","tm_service":"","tm_update":"","tm_delete":""},{"id":"f1fd30c6-6476-11ec-8b55-7f9c5b9550b7","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:23.995000","tm_service":"","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().QueuecallGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCalls, nil)

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
		name  string
		agent amagent.Agent

		reqQuery string

		responseQueuecall *qmqueuecall.WebhookMessage

		expectQueuecallID uuid.UUID
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

			reqQuery: "/queuecalls/7d54d626-1681-11ed-ab05-473fa9aa2542",

			responseQueuecall: &qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d54d626-1681-11ed-ab05-473fa9aa2542"),
				},
				TMCreate: "2020-09-20T03:23:21.995000",
			},

			expectQueuecallID: uuid.FromStringOrNil("7d54d626-1681-11ed-ab05-473fa9aa2542"),
			expectRes:         `{"id":"7d54d626-1681-11ed-ab05-473fa9aa2542","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().QueuecallGet(req.Context(), &tt.agent, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)

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
		name  string
		agent amagent.Agent

		reqQuery string

		expectQueuecallID uuid.UUID
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

			reqQuery: "/queuecalls/a275df90-1681-11ed-a021-c3f295fc9257",

			expectQueuecallID: uuid.FromStringOrNil("a275df90-1681-11ed-a021-c3f295fc9257"),
			expectRes:         `{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().QueuecallDelete(req.Context(), &tt.agent, tt.expectQueuecallID).Return(&qmqueuecall.WebhookMessage{}, nil)

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

func Test_queuecallsIDKickPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		requQuery string

		responseQueuecall *qmqueuecall.WebhookMessage

		expectQueuecallID uuid.UUID
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

			requQuery: "/queuecalls/72c9dfb0-bcbe-11ed-853f-7f662faaee5b/kick",

			responseQueuecall: &qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72c9dfb0-bcbe-11ed-853f-7f662faaee5b"),
				},
			},

			expectQueuecallID: uuid.FromStringOrNil("72c9dfb0-bcbe-11ed-853f-7f662faaee5b"),
			expectRes:         `{"id":"72c9dfb0-bcbe-11ed-853f-7f662faaee5b","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.requQuery, nil)

			mockSvc.EXPECT().QueuecallKick(req.Context(), &tt.agent, tt.expectQueuecallID).Return(tt.responseQueuecall, nil)
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

func Test_queuecallsReferenceIDIDKickPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		requQuery string

		responseQueuecall *qmqueuecall.WebhookMessage

		expectReferenceID uuid.UUID
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

			requQuery: "/queuecalls/reference_id/e01d78ce-bcbe-11ed-8164-f3c4a472391e/kick",

			responseQueuecall: &qmqueuecall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e01d78ce-bcbe-11ed-8164-f3c4a472391e"),
				},
			},

			expectReferenceID: uuid.FromStringOrNil("e01d78ce-bcbe-11ed-8164-f3c4a472391e"),
			expectRes:         `{"id":"e01d78ce-bcbe-11ed-8164-f3c4a472391e","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.requQuery, nil)

			mockSvc.EXPECT().QueuecallKickByReferenceID(req.Context(), &tt.agent, tt.expectReferenceID).Return(tt.responseQueuecall, nil)
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

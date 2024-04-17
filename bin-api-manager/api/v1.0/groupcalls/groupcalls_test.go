package groupcalls

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

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

func Test_groupcallsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery          string
		reqBody           request.BodyGroupcallsPOST
		responseGroupcall *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			reqQuery: "/v1.0/groupcalls",
			reqBody: request.BodyGroupcallsPOST{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				FlowID:       uuid.FromStringOrNil("6b83babe-bf07-11ed-930f-8f4a33752b7f"),
				RingMethod:   cmgroupcall.RingMethodRingAll,
				AnswerMethod: cmgroupcall.AnswerMethodHangupOthers,
			},

			responseGroupcall: &cmgroupcall.WebhookMessage{
				ID: uuid.FromStringOrNil("7fa0708c-bf07-11ed-9dac-f7a8809e6a53"),
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
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().GroupcallCreate(req.Context(), &tt.agent, tt.reqBody.Source, tt.reqBody.Destinations, tt.reqBody.FlowID, tt.reqBody.Actions, tt.reqBody.RingMethod, tt.reqBody.AnswerMethod).Return(tt.responseGroupcall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_groupcallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqParam request.ParamGroupcallsGET

		responseGroupcalls []*cmgroupcall.WebhookMessage
		expectRes          string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("d98435c4-bf08-11ed-af72-f7e533f63816"),
			},

			"/v1.0/groupcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamGroupcallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*cmgroupcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("44f67330-bf09-11ed-aba5-3bca63e6a7b4"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("45364456-bf09-11ed-93aa-53f6a09e7fc1"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"44f67330-bf09-11ed-aba5-3bca63e6a7b4","customer_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"45364456-bf09-11ed-93aa-53f6a09e7fc1","customer_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
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
			mockSvc.EXPECT().GroupcallGets(req.Context(), &tt.agent, tt.reqParam.PageSize, tt.reqParam.PageToken).Return(tt.responseGroupcalls, nil)

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

func Test_groupcallsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseGroupcall *cmgroupcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/groupcalls/c1423b7c-bf09-11ed-a3f8-cb3f5a42b528",

			&cmgroupcall.WebhookMessage{
				ID: uuid.FromStringOrNil("c1423b7c-bf09-11ed-a3f8-cb3f5a42b528"),
			},
			`{"id":"c1423b7c-bf09-11ed-a3f8-cb3f5a42b528","customer_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().GroupcallGet(req.Context(), &tt.agent, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)

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

func Test_groupcallsIDHangupPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseGroupcall *cmgroupcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/groupcalls/0410089e-bf0a-11ed-93b7-f3a49f2b479f/hangup",

			&cmgroupcall.WebhookMessage{
				ID: uuid.FromStringOrNil("0410089e-bf0a-11ed-93b7-f3a49f2b479f"),
			},
			`{"id":"0410089e-bf0a-11ed-93b7-f3a49f2b479f","customer_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().GroupcallHangup(req.Context(), &tt.agent, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)

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

func Test_groupcallsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseGroupcall *cmgroupcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/groupcalls/487fd892-bf0a-11ed-9f7c-b3eaa708de0a",

			&cmgroupcall.WebhookMessage{
				ID: uuid.FromStringOrNil("487fd892-bf0a-11ed-9f7c-b3eaa708de0a"),
			},
			`{"id":"487fd892-bf0a-11ed-9f7c-b3eaa708de0a","customer_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().GroupcallDelete(req.Context(), &tt.agent, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)

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

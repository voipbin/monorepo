package calls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_CallsPOST(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		req      request.BodyCallsPOST

		resCall   []*cmcall.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.BodyCallsPOST{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeSIP,
						Target: "destination@test.voipbin.net",
					},
				},
				Actions: []fmaction.Action{},
			},

			[]*cmcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("98b963ac-8df9-11ec-b26b-031d30ff93df"),
				},
			},
			`[{"id":"98b963ac-8df9-11ec-b26b-031d30ff93df","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/calls", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallCreate(req.Context(), &tt.customer, tt.req.FlowID, tt.req.Actions, &tt.req.Source, tt.req.Destinations).Return(tt.resCall, nil)

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

func Test_CallsGET(t *testing.T) {

	type test struct {
		name      string
		customer  cscustomer.Customer
		req       request.ParamCallsGET
		resCalls  []*cmcall.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"1 item",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("668e6ee6-f989-11ea-abca-bf1ca885b142"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("5d8167e0-f989-11ea-8b34-2b0a03c78fc5"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("61c6626a-f989-11ea-abbf-97944933fee9"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"668e6ee6-f989-11ea-abca-bf1ca885b142","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"5d8167e0-f989-11ea-8b34-2b0a03c78fc5","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61c6626a-f989-11ea-abbf-97944933fee9","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			reqQuery := fmt.Sprintf("/v1.0/calls?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGets(req.Context(), &tt.customer, tt.req.PageSize, tt.req.PageToken).Return(tt.resCalls, nil)

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

func Test_CallsIDGET(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		resCall  *cmcall.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			&cmcall.WebhookMessage{
				ID:       uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
				TMCreate: "2020-09-20T03:23:21.995000",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			reqQuery := fmt.Sprintf("/v1.0/calls/%s", tt.resCall.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGet(req.Context(), &tt.customer, tt.resCall.ID).Return(tt.resCall, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.resCall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_callsIDDELETE(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		reqQuery string
		callID   uuid.UUID

		responseCall *cmcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/calls/72709904-719c-11ed-94f7-b78b75ad5dce",
			uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),

			&cmcall.WebhookMessage{
				ID: uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),
			},

			`{"id":"72709904-719c-11ed-94f7-b78b75ad5dce","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().CallDelete(req.Context(), &tt.customer, tt.callID).Return(tt.responseCall, nil)

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

func Test_callsIDHangupPOST(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		reqQuery string
		callID   uuid.UUID

		responseCall *cmcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/calls/09b9bf4c-8927-11ed-b16c-5719373564c9/hangup",
			uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),

			&cmcall.WebhookMessage{
				ID: uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),
			},

			`{"id":"09b9bf4c-8927-11ed-b16c-5719373564c9","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().CallHangup(req.Context(), &tt.customer, tt.callID).Return(tt.responseCall, nil)

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

func Test_CallsTalkPOST(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer

		reqQuery string
		reqBody  request.BodyCallsIDTalkPOST

		expectCallID uuid.UUID
		expectRes    string
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			"/v1.0/calls/ed229366-a4b7-11ed-bfe7-b38647d68a3d/talk",
			request.BodyCallsIDTalkPOST{
				Text:     "hello world",
				Gender:   "female",
				Language: "en-US",
			},

			uuid.FromStringOrNil("ed229366-a4b7-11ed-bfe7-b38647d68a3d"),
			`[{"id":"98b963ac-8df9-11ec-b26b-031d30ff93df","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallTalk(req.Context(), &tt.customer, tt.expectCallID, tt.reqBody.Text, tt.reqBody.Gender, tt.reqBody.Language).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

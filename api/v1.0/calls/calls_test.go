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

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestCallsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name    string
		user    user.User
		req     request.BodyCallsPOST
		reqFlow *flow.Flow
		resFlow *flow.Flow
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			request.BodyCallsPOST{
				Source: address.Address{
					Type:   address.TypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destination: address.Address{
					Type:   address.TypeSIP,
					Target: "destination@test.voipbin.net",
				},
				Actions: []action.Action{},
			},
			&flow.Flow{
				Name:    "tmp",
				Detail:  "tmp outbound flow",
				Actions: []action.Action{},
				Persist: false,
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("044cf45a-f3a3-11ea-963d-1fc4372fcff8"),
				Name:    "temp",
				Detail:  "tmp outbound flow",
				Actions: []action.Action{},
			},
		},
		{
			"with webhook",
			user.User{
				ID: 1,
			},
			request.BodyCallsPOST{
				WebhookURI: "https://test.com/webhook",
				Source: address.Address{
					Type:   address.TypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destination: address.Address{
					Type:   address.TypeSIP,
					Target: "destination@test.voipbin.net",
				},
				Actions: []action.Action{},
			},
			&flow.Flow{
				Name:       "tmp",
				Detail:     "tmp outbound flow",
				WebhookURI: "https://test.com/webhook",
				Actions:    []action.Action{},
				Persist:    false,
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("044cf45a-f3a3-11ea-963d-1fc4372fcff8"),
				Name:       "temp",
				Detail:     "tmp outbound flow",
				WebhookURI: "https://test.com/webhook",
				Actions:    []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/calls", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().FlowCreate(&tt.user, tt.reqFlow.Name, tt.reqFlow.Detail, tt.reqFlow.WebhookURI, tt.reqFlow.Actions, tt.reqFlow.Persist).Return(tt.resFlow, nil)
			mockSvc.EXPECT().CallCreate(&tt.user, tt.resFlow.ID, &tt.req.Source, &tt.req.Destination).Return(nil, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func TestCallsGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name      string
		user      user.User
		req       request.ParamCallsGET
		resCalls  []*cmcall.Event
		expectRes string
	}

	tests := []test{
		{
			"1 item",
			user.User{
				ID: 1,
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.Event{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			user.User{
				ID: 1,
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.Event{
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
			`{"result":[{"id":"668e6ee6-f989-11ea-abca-bf1ca885b142","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""},{"id":"5d8167e0-f989-11ea-8b34-2b0a03c78fc5","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""},{"id":"61c6626a-f989-11ea-abbf-97944933fee9","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","webhook_uri":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			reqQuery := fmt.Sprintf("/v1.0/calls?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGets(&tt.user, tt.req.PageSize, tt.req.PageToken).Return(tt.resCalls, nil)

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

func TestCallsIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name    string
		user    user.User
		resCall *cmcall.Event
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			&cmcall.Event{
				ID:       uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
		},
		{
			"webhook",
			user.User{
				ID: 1,
			},
			&cmcall.Event{
				ID:         uuid.FromStringOrNil("9e6e2dbe-830a-11eb-8fb0-cf5ab9cac353"),
				WebhookURI: "https://test.com/tesadf",
				TMCreate:   "2020-09-20T03:23:21.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			reqQuery := fmt.Sprintf("/v1.0/calls/%s", tt.resCall.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGet(&tt.user, tt.resCall.ID).Return(tt.resCall, nil)
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

package calls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
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
		name string
		user user.User
		req  request.BodyCallsPOST
		flow *flow.Flow
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			request.BodyCallsPOST{
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destination: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "destination@test.voipbin.net",
				},
				Actions: []action.Action{},
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("044cf45a-f3a3-11ea-963d-1fc4372fcff8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
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

			mockSvc.EXPECT().FlowCreate(&tt.user, uuid.Nil, "temp", "tmp outbound flow", tt.req.Actions, false).Return(tt.flow, nil)
			mockSvc.EXPECT().CallCreate(&tt.user, tt.flow.ID, tt.req.Source, tt.req.Destination).Return(nil, nil)

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
		resCalls  []*call.Call
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
			[]*call.Call{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","user_id":0,"flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","direction":"","hangup_by":"","hangup_reason":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
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
			[]*call.Call{
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
			`{"result":[{"id":"668e6ee6-f989-11ea-abca-bf1ca885b142","user_id":0,"flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","direction":"","hangup_by":"","hangup_reason":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""},{"id":"5d8167e0-f989-11ea-8b34-2b0a03c78fc5","user_id":0,"flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","direction":"","hangup_by":"","hangup_reason":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""},{"id":"61c6626a-f989-11ea-abbf-97944933fee9","user_id":0,"flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","direction":"","hangup_by":"","hangup_reason":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
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

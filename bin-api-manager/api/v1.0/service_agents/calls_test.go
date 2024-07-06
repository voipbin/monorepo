package service_agents

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"reflect"

	"github.com/golang/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gofrs/uuid"
)

func Test_callsGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		resCalls []*cmcall.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/calls?page_token=2020-09-20%2003:23:20.995000&page_size=10",

			resCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1e5ee31c-3bb1-11ef-983b-67a86019900d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1ef41086-3bb1-11ef-a7b9-a39300b3cc30"),
					},
				},
			},

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"1e5ee31c-3bb1-11ef-983b-67a86019900d","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"1ef41086-3bb1-11ef-a7b9-a39300b3cc30","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().ServiceAgentCallGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.resCalls, nil)

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

func Test_callsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		resCall      *cmcall.WebhookMessage
		expectCallID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/v1.0/service_agents/calls/0b2decd0-3bb5-11ef-b2bb-bb18783b3f41",
			resCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0b2decd0-3bb5-11ef-b2bb-bb18783b3f41"),
				},
				TMCreate: "2020-09-20T03:23:21.995000",
			},
			expectCallID: uuid.FromStringOrNil("0b2decd0-3bb5-11ef-b2bb-bb18783b3f41"),
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

			mockSvc.EXPECT().ServiceAgentCallGet(req.Context(), &tt.agent, tt.expectCallID).Return(tt.resCall, nil)
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

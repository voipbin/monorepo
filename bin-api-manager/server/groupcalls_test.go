package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_groupcallsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseGroupcall *cmgroupcall.WebhookMessage

		expectSource       commonaddress.Address
		expectDestinations []commonaddress.Address
		expectActions      []fmaction.Action
		expectFlowID       uuid.UUID
		expectRingMethod   cmgroupcall.RingMethod
		expectAnswerMethod cmgroupcall.AnswerMethod
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/groupcalls",
			reqBody:  []byte(`{"source":{"type":"tel","target":"+821100000001"},"destinations":[{"type":"tel","target":"+821100000002"},{"type":"tel","target":"+821100000003"}],"flow_id":"6b83babe-bf07-11ed-930f-8f4a33752b7f","actions":[{"type":"answer"}],"ring_method":"ring_all","answer_method":"hangup_others"}`),

			responseGroupcall: &cmgroupcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7fa0708c-bf07-11ed-9dac-f7a8809e6a53"),
				},
			},

			expectSource: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectFlowID:       uuid.FromStringOrNil("6b83babe-bf07-11ed-930f-8f4a33752b7f"),
			expectRingMethod:   cmgroupcall.RingMethodRingAll,
			expectAnswerMethod: cmgroupcall.AnswerMethodHangupOthers,
			expectRes:          `{"id":"7fa0708c-bf07-11ed-9dac-f7a8809e6a53","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().GroupcallCreate(req.Context(), &tt.agent, tt.expectSource, tt.expectDestinations, tt.expectFlowID, tt.expectActions, tt.expectRingMethod, tt.expectAnswerMethod).Return(tt.responseGroupcall, nil)

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

func Test_groupcallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseGroupcalls []*cmgroupcall.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d98435c4-bf08-11ed-af72-f7e533f63816"),
				},
			},

			reqQuery: "/groupcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseGroupcalls: []*cmgroupcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("44f67330-bf09-11ed-aba5-3bca63e6a7b4"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("45364456-bf09-11ed-93aa-53f6a09e7fc1"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"44f67330-bf09-11ed-aba5-3bca63e6a7b4","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"},{"id":"45364456-bf09-11ed-93aa-53f6a09e7fc1","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
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
			mockSvc.EXPECT().GroupcallGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseGroupcalls, nil)

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

		expectGroupcallID uuid.UUID
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/groupcalls/c1423b7c-bf09-11ed-a3f8-cb3f5a42b528",

			responseGroupcall: &cmgroupcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c1423b7c-bf09-11ed-a3f8-cb3f5a42b528"),
				},
			},

			expectGroupcallID: uuid.FromStringOrNil("c1423b7c-bf09-11ed-a3f8-cb3f5a42b528"),
			expectRes:         `{"id":"c1423b7c-bf09-11ed-a3f8-cb3f5a42b528","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().GroupcallGet(req.Context(), &tt.agent, tt.expectGroupcallID).Return(tt.responseGroupcall, nil)

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

		expectGroupcallID uuid.UUID
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/groupcalls/0410089e-bf0a-11ed-93b7-f3a49f2b479f/hangup",

			responseGroupcall: &cmgroupcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0410089e-bf0a-11ed-93b7-f3a49f2b479f"),
				},
			},

			expectGroupcallID: uuid.FromStringOrNil("0410089e-bf0a-11ed-93b7-f3a49f2b479f"),
			expectRes:         `{"id":"0410089e-bf0a-11ed-93b7-f3a49f2b479f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)

			mockSvc.EXPECT().GroupcallHangup(req.Context(), &tt.agent, tt.expectGroupcallID).Return(tt.responseGroupcall, nil)

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

		expectGroupcallID uuid.UUID
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/groupcalls/487fd892-bf0a-11ed-9f7c-b3eaa708de0a",

			responseGroupcall: &cmgroupcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("487fd892-bf0a-11ed-9f7c-b3eaa708de0a"),
				},
			},

			expectGroupcallID: uuid.FromStringOrNil("487fd892-bf0a-11ed-9f7c-b3eaa708de0a"),
			expectRes:         `{"id":"487fd892-bf0a-11ed-9f7c-b3eaa708de0a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().GroupcallDelete(req.Context(), &tt.agent, tt.expectGroupcallID).Return(tt.responseGroupcall, nil)

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

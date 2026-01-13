package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAimessages(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAImessage *ammessage.WebhookMessage

		expectedAIcallID uuid.UUID
		expectedRole     ammessage.Role
		expectedContent  string
		expectedRes      string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aimessages",
			reqBody:  []byte(`{"aicall_id":"9fa30c3a-f31e-11ef-a4df-9f6bf108282e","role":"user","content":"test text"}`),

			responseAImessage: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedAIcallID: uuid.FromStringOrNil("9fa30c3a-f31e-11ef-a4df-9f6bf108282e"),
			expectedRole:     ammessage.RoleUser,
			expectedContent:  "test text",

			expectedRes: `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().AImessageCreate(
				req.Context(),
				&tt.agent,
				tt.expectedAIcallID,
				tt.expectedRole,
				tt.expectedContent,
			).Return(tt.responseAImessage, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAimessages(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAImessages []*ammessage.WebhookMessage

		expectedAIcallID  uuid.UUID
		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aimessages?page_size=10&page_token=2020-09-20%2003:23:20.995000&aicall_id=ecebd332-f31e-11ef-9ab5-33426e3ee4ff",

			responseAImessages: []*ammessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ed2346dc-f31e-11ef-acd5-67a8f966fe17"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectedAIcallID:  uuid.FromStringOrNil("ecebd332-f31e-11ef-9ab5-33426e3ee4ff"),
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"ed2346dc-f31e-11ef-acd5-67a8f966fe17","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aimessages?page_size=10&page_token=2020-09-20%2003:23:20.995000&aicall_id=ed487b96-f31e-11ef-9337-e792818f3609",

			responseAImessages: []*ammessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1707f380-f31f-11ef-bfe4-7ff769b357b3"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("17268426-f31f-11ef-aa11-6f21c1723af6"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("17468500-f31f-11ef-b7b6-9b29397f4894"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectedAIcallID:  uuid.FromStringOrNil("ed487b96-f31e-11ef-9337-e792818f3609"),
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"1707f380-f31f-11ef-bfe4-7ff769b357b3","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:21.995000"},{"id":"17268426-f31f-11ef-aa11-6f21c1723af6","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:22.995000"},{"id":"17468500-f31f-11ef-b7b6-9b29397f4894","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().AImessageGetsByAIcallID(req.Context(), &tt.agent, tt.expectedAIcallID, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseAImessages, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAimessagesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAImessage *ammessage.WebhookMessage

		expectedAImessageID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aimessages/796924c2-f31f-11ef-8589-c3efd79e11d5",

			responseAImessage: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("796924c2-f31f-11ef-8589-c3efd79e11d5"),
				},
			},

			expectedAImessageID: uuid.FromStringOrNil("796924c2-f31f-11ef-8589-c3efd79e11d5"),
			expectRes:           `{"id":"796924c2-f31f-11ef-8589-c3efd79e11d5","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().AImessageGet(req.Context(), &tt.agent, tt.expectedAImessageID).Return(tt.responseAImessage, nil)

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

func Test_DeleteAimessagesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAImessage *ammessage.WebhookMessage

		expectedAImessageID uuid.UUID
		expectedRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aimessages/b3fc4312-f31f-11ef-8661-939776978f23",

			responseAImessage: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				},
			},

			expectedAImessageID: uuid.FromStringOrNil("b3fc4312-f31f-11ef-8661-939776978f23"),
			expectedRes:         `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000","aicall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().AImessageDelete(req.Context(), &tt.agent, tt.expectedAImessageID).Return(tt.responseAImessage, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

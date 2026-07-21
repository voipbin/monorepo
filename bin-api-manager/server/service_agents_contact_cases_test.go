package server

import (
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmkase "monorepo/bin-contact-manager/models/kase"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_contactCasesGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCases []*cmkase.Case

		expectPageToken string
		expectPageSize  uint64
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/contact_cases?page_token=2020-09-20T03:23:20.995000Z&page_size=10",

			responseCases: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("83d48228-3ed7-11ef-a9ca-070e7ba46a55"),
				},
			},

			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectPageSize:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentCaseList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCases, "", nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_contactCasesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCase *cmkase.Case
		expectCaseID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1",

			responseCase: &cmkase.Case{
				ID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			expectCaseID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentCaseGet(req.Context(), tt.agent, tt.expectCaseID).Return(tt.responseCase, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseCase)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", string(res), w.Body.String())
			}
		})
	}
}

func Test_contactCasesIDClosePOST(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCase *cmkase.Case
		expectCaseID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1/close",

			responseCase: &cmkase.Case{
				ID:     uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Status: cmkase.StatusClosed,
			},

			expectCaseID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentCaseClose(req.Context(), tt.agent, tt.expectCaseID).Return(tt.responseCase, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_contactCasesIDAssignPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseCase   *cmkase.Case
		expectCaseID   uuid.UUID
		expectOwnerID  uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1/assign",
			reqBody:  `{"owner_id":"f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"}`,

			responseCase: &cmkase.Case{
				ID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
				},
			},

			expectCaseID:  uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectOwnerID: uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ServiceAgentCaseAssign(req.Context(), tt.agent, tt.expectCaseID, tt.expectOwnerID).Return(tt.responseCase, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body.String())
			}
		})
	}
}

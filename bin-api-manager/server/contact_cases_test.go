package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmkase "monorepo/bin-contact-manager/models/kase"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetContactCases(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	contactID := uuid.FromStringOrNil("7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectOwnerID   uuid.UUID
		expectContactID uuid.UUID

		responseItems    []*cmkase.Case
		responseToken    string
		expectStatusCode int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:         "/contact_cases",
			responseItems:    []*cmkase.Case{},
			responseToken:    "",
			expectStatusCode: http.StatusOK,
		},
		{
			name: "contact_id filter reaches servicehandler with the exact value",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:         "/contact_cases?contact_id=" + contactID.String(),
			expectContactID:  contactID,
			responseItems:    []*cmkase.Case{},
			responseToken:    "",
			expectStatusCode: http.StatusOK,
		},
		{
			name:             "unauthenticated",
			agent:            nil,
			reqQuery:         "/contact_cases",
			expectStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseItems != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseList(req.Context(), tt.agent, uuid.Nil, uint64(100), "", gomock.Any(), gomock.Any(), tt.expectOwnerID, tt.expectContactID).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatusCode {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatusCode, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetContactCasesUnresolved(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery      string
		responseItems []*cmkase.Case
		responseToken string
		expectStatus  int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:      "/contact_cases/unresolved",
			responseItems: []*cmkase.Case{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/unresolved",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseItems != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseListUnresolved(req.Context(), tt.agent, uint64(100), "").
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetContactCasesId(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		responseCase *cmkase.Case
		expectStatus int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery: "/contact_cases/11111111-0000-0000-0000-000000000001",
			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseCase != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseGet(req.Context(), tt.agent, caseID).
					Return(tt.responseCase, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostContactCasesIdClose(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		responseCase *cmkase.Case
		expectStatus int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery: "/contact_cases/11111111-0000-0000-0000-000000000001/close",
			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusClosed,
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/close",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			if tt.responseCase != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseClose(req.Context(), tt.agent, caseID).
					Return(tt.responseCase, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostContactCasesIdContinue(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	newCaseID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		responseCase *cmkase.Case
		expectStatus int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery: "/contact_cases/11111111-0000-0000-0000-000000000001/continue",
			responseCase: &cmkase.Case{
				ID:         newCaseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/continue",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)

			if tt.responseCase != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseContinue(req.Context(), tt.agent, caseID).
					Return(tt.responseCase, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

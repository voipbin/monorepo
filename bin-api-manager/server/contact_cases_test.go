package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmkase "monorepo/bin-contact-manager/models/kase"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cerrors "monorepo/bin-common-handler/models/errors"

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

		expectOwnerID     uuid.UUID
		expectContactID   uuid.UUID
		expectReferenceID string

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
			name: "reference_id filter reaches servicehandler with the exact value",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:          "/contact_cases?reference_id=ORD-2026-04821",
			expectReferenceID: "ORD-2026-04821",
			responseItems:     []*cmkase.Case{},
			responseToken:     "",
			expectStatusCode:  http.StatusOK,
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
					CaseList(req.Context(), tt.agent, uuid.Nil, uint64(100), "", gomock.Any(), gomock.Any(), tt.expectOwnerID, tt.expectContactID, tt.expectReferenceID).
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

// Test_PutContactCasesId covers server.PutContactCasesId (VOIP-1253):
// unauthenticated rejection, a normal "attach" request with a non-empty
// contact_id, and -- the trickiest part of this design per round 6/7 of
// the code review -- the contact_id="" -> uuid.Nil ("detach") boundary
// conversion (uuid.FromStringOrNil("")). Follows the table-driven +
// mockSvc.EXPECT() pattern used by Test_PostContactCasesIdClose above.
func Test_PutContactCasesId(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		expectContactID uuid.UUID
		responseCase    *cmkase.Case
		expectStatus    int
	}{
		{
			name: "normal attach with non-empty contact_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:        "/contact_cases/11111111-0000-0000-0000-000000000001",
			reqBody:         []byte(`{"contact_id":"33333333-0000-0000-0000-000000000003"}`),
			expectContactID: contactID,
			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectStatus: http.StatusOK,
		},
		{
			// This is the case the round 6/7 review specifically flagged
			// as the trickiest part of the whole design: an empty-string
			// contact_id in the JSON body must convert to uuid.Nil (the
			// "detach" sentinel), NOT be rejected as invalid and NOT be
			// forwarded as the empty string. gomock's exact-match
			// EXPECT() on uuid.Nil below fails the test if
			// uuid.FromStringOrNil("") ever stops being the conversion
			// used at the HTTP boundary (server/contact_cases.go).
			name: "detach with empty contact_id converts to uuid.Nil",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:        "/contact_cases/11111111-0000-0000-0000-000000000001",
			reqBody:         []byte(`{"contact_id":""}`),
			expectContactID: uuid.Nil,
			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  nil,
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001",
			reqBody:      []byte(`{"contact_id":""}`),
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.responseCase != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseUpdateContact(req.Context(), tt.agent, caseID, tt.expectContactID).
					Return(tt.responseCase, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

// Test_PutContactCasesId_InvalidJSONBody verifies malformed JSON in the
// request body is rejected with INVALID_ARGUMENT / INVALID_JSON_BODY
// before the servicehandler is ever consulted, mirroring
// Test_campaignsPost_InvalidJSONBody's pattern.
func Test_PutContactCasesId_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// No CaseUpdateContact EXPECT() set -- gomock fails the test if the
	// servicehandler is ever reached for malformed JSON.
	req, _ := http.NewRequest(http.MethodPut, "/contact_cases/11111111-0000-0000-0000-000000000001", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_PutContactCasesId_ServiceError verifies a servicehandler-layer
// error (ErrPermissionDenied) propagates to the correct HTTP status via
// abortWithServiceError, mirroring Test_messagesPost_InsufficientBalance's
// pattern.
func Test_PutContactCasesId_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	body := []byte(`{"contact_id":"33333333-0000-0000-0000-000000000003"}`)
	req, _ := http.NewRequest(http.MethodPut, "/contact_cases/11111111-0000-0000-0000-000000000001", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	mockSvc.EXPECT().
		CaseUpdateContact(gomock.Any(), agent, caseID, contactID).
		Return(nil, fmt.Errorf("%w: user has no permission", serviceerrors.ErrPermissionDenied))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusPermissionDenied, "PERMISSION_DENIED")
}

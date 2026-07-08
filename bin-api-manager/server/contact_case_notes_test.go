package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetContactCasesIdNotes(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseNotes []*cmcasenote.CaseNote
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
			reqQuery:      "/contact_cases/11111111-0000-0000-0000-000000000001/notes",
			responseNotes: []*cmcasenote.CaseNote{},
			expectStatus:  http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/notes",
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

			if tt.responseNotes != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseNoteList(req.Context(), tt.agent, caseID).
					Return(tt.responseNotes, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostContactCasesIdNotes(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	noteID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  map[string]interface{}

		expectAuthorID *uuid.UUID
		responseNote   *cmcasenote.CaseNote
		expectStatus   int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery: "/contact_cases/11111111-0000-0000-0000-000000000001/notes",
			reqBody: map[string]interface{}{
				"author_type": "agent",
				"author_id":   agentID.String(),
				"text":        "Called the customer back, no answer.",
			},
			expectAuthorID: &agentID,
			responseNote: &cmcasenote.CaseNote{
				ID:         noteID,
				CustomerID: customerID,
				CaseID:     caseID,
				AuthorType: cmcasenote.AuthorTypeAgent,
				AuthorID:   &agentID,
				Text:       "Called the customer back, no answer.",
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/notes",
			reqBody:      map[string]interface{}{},
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

			bodyBytes, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tt.responseNote != nil && tt.agent != nil {
				mockSvc.EXPECT().
					CaseNoteCreate(req.Context(), tt.agent, caseID, "agent", tt.expectAuthorID, "Called the customer back, no answer.").
					Return(tt.responseNote, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_DeleteContactCasesIdNotesNoteId(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	noteID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectDelete bool
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
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/notes/33333333-0000-0000-0000-000000000003",
			expectDelete: true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_cases/11111111-0000-0000-0000-000000000001/notes/33333333-0000-0000-0000-000000000003",
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			if tt.expectDelete && tt.agent != nil {
				mockSvc.EXPECT().
					CaseNoteDelete(req.Context(), tt.agent, caseID, noteID).
					Return(nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

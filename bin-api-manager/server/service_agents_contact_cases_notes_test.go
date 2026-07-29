package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_contactCasesIDNotesGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseNotes []*cmcasenote.CaseNote
		expectCaseID  uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1/notes",

			responseNotes: []*cmcasenote.CaseNote{
				{ID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")},
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
			mockSvc.EXPECT().ServiceAgentCaseNoteList(req.Context(), tt.agent, tt.expectCaseID).Return(tt.responseNotes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body.String())
			}
		})
	}
}

func Test_contactCasesIDNotesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseNote *cmcasenote.CaseNote
		expectCaseID uuid.UUID
		expectText   string
	}{
		{
			// The request body only carries text -- author_type/author_id
			// are never accepted from the client on this surface.
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1/notes",
			reqBody:  `{"text":"Called the customer back, no answer."}`,

			responseNote: &cmcasenote.CaseNote{
				ID:   uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
				Text: "Called the customer back, no answer.",
			},
			expectCaseID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectText:   "Called the customer back, no answer.",
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
			mockSvc.EXPECT().ServiceAgentCaseNoteCreate(req.Context(), tt.agent, tt.expectCaseID, tt.expectText).Return(tt.responseNote, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body.String())
			}
		})
	}
}

func Test_contactCasesIDNotesIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectCaseID uuid.UUID
		expectNoteID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/contact_cases/e66d1da0-3ed7-11ef-9208-4bcc069917a1/notes/22222222-0000-0000-0000-000000000002",

			expectCaseID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectNoteID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentCaseNoteDelete(req.Context(), tt.agent, tt.expectCaseID, tt.expectNoteID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body.String())
			}
		})
	}
}

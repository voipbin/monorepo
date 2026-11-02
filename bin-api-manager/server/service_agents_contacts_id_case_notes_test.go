package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_serviceAgentsContactsIDCaseNotesGET(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	contactID := uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		reqQuery string

		expectSize  uint64
		expectToken string
	}{
		{name: "defaults to 100", reqQuery: "/service_agents/contacts/e66d1da0-3ed7-11ef-9208-4bcc069917a1/case_notes", expectSize: 100},
		{name: "explicit size and token", reqQuery: "/service_agents/contacts/e66d1da0-3ed7-11ef-9208-4bcc069917a1/case_notes?page_size=7&page_token=2026-06-01T09:30:01.000000Z", expectSize: 7, expectToken: "2026-06-01T09:30:01.000000Z"},
		{name: "size above 100 is capped", reqQuery: "/service_agents/contacts/e66d1da0-3ed7-11ef-9208-4bcc069917a1/case_notes?page_size=1000", expectSize: 100},
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
				c.Set("auth_identity", agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentContactCaseNoteList(req.Context(), agent, contactID, tt.expectSize, tt.expectToken).
				Return([]*cmcasenote.CaseNote{{ID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"), CaseID: caseID}}, "next-token", nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("Wrong match. expect: %d, got: %d, body: %s", http.StatusOK, w.Code, w.Body.String())
			}

			var body struct {
				Result []struct {
					CaseID string `json:"case_id"`
				} `json:"result"`
				NextPageToken string `json:"next_page_token"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("could not parse the body: %v", err)
			}
			if len(body.Result) != 1 || body.Result[0].CaseID != caseID.String() {
				t.Errorf("case_id must be present in the response, body: %s", w.Body.String())
			}
			if body.NextPageToken != "next-token" {
				t.Errorf("Wrong next_page_token: %q", body.NextPageToken)
			}
		})
	}
}

func Test_serviceAgentsContactsIDCaseNotesGET_Unauthenticated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/service_agents/contacts/e66d1da0-3ed7-11ef-9208-4bcc069917a1/case_notes", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong match. expect: %d, got: %d, body: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmcontactstar "monorepo/bin-contact-manager/models/contactstar"

	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	starSrvContactID = uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1")
	starSrvAgent     = auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
)

func starSrvNew(t *testing.T, agent *auth.AuthIdentity) (*servicehandler.MockServiceHandler, *gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	if agent != nil {
		r.Use(func(c *gin.Context) { c.Set("auth_identity", agent) })
	}
	openapi_server.RegisterHandlers(r, h)

	return mockSvc, r, w
}

func Test_serviceAgentsContactsIDStarPUT(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)

	t.Run("ok", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		mockSvc.EXPECT().ServiceAgentContactStarSet(req.Context(), starSrvAgent, starSrvContactID).
			Return(&cmcontactstar.WebhookMessage{ContactID: starSrvContactID, Starred: true, TMCreate: &created}, nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
		}
		expect := `{"contact_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","starred":true,"tm_create":"2026-11-02T09:15:30.123456Z"}`
		if w.Body.String() != expect {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		mockSvc.EXPECT().ServiceAgentContactStarSet(req.Context(), starSrvAgent, starSrvContactID).Return(nil, serviceerrors.ErrNotFound)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id is a bad request", func(t *testing.T) {
		_, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("PUT", "/service_agents/contacts/not-a-uuid/star", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, r, w := starSrvNew(t, nil)

		req, _ := http.NewRequest("PUT", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func Test_serviceAgentsContactsIDStarDELETE(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		mockSvc.EXPECT().ServiceAgentContactStarDelete(req.Context(), starSrvAgent, starSrvContactID).
			Return(&cmcontactstar.WebhookMessage{ContactID: starSrvContactID, Starred: false}, nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
		}
		expect := `{"contact_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","starred":false}`
		if w.Body.String() != expect {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		mockSvc.EXPECT().ServiceAgentContactStarDelete(req.Context(), starSrvAgent, starSrvContactID).Return(nil, serviceerrors.ErrNotFound)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id is a bad request", func(t *testing.T) {
		_, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("DELETE", "/service_agents/contacts/not-a-uuid/star", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, r, w := starSrvNew(t, nil)

		req, _ := http.NewRequest("DELETE", "/service_agents/contacts/"+starSrvContactID.String()+"/star", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func Test_serviceAgentsContactStarsGET(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)

	tests := []struct {
		name        string
		reqQuery    string
		expectSize  uint64
		expectToken string
	}{
		{name: "defaults to 100", reqQuery: "/service_agents/contact_stars", expectSize: 100},
		{name: "explicit size and token", reqQuery: "/service_agents/contact_stars?page_size=7&page_token=2026-06-01T09:30:01.000000Z", expectSize: 7, expectToken: "2026-06-01T09:30:01.000000Z"},
		{name: "size above 100 is capped", reqQuery: "/service_agents/contact_stars?page_size=1000", expectSize: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc, r, w := starSrvNew(t, starSrvAgent)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentContactStarList(req.Context(), starSrvAgent, tt.expectSize, tt.expectToken).
				Return([]*cmcontactstar.WebhookMessage{{ContactID: starSrvContactID, Starred: true, TMCreate: &created}}, "next-token", nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
			}

			var body struct {
				Result []struct {
					ContactID string `json:"contact_id"`
					Starred   bool   `json:"starred"`
				} `json:"result"`
				NextPageToken string `json:"next_page_token"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("could not parse the body: %v", err)
			}
			if len(body.Result) != 1 || body.Result[0].ContactID != starSrvContactID.String() || !body.Result[0].Starred {
				t.Errorf("Wrong body: %s", w.Body.String())
			}
			if body.NextPageToken != "next-token" {
				t.Errorf("Wrong next_page_token: %q", body.NextPageToken)
			}
		})
	}

	t.Run("service error maps through", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/contact_stars", nil)
		mockSvc.EXPECT().ServiceAgentContactStarList(req.Context(), starSrvAgent, uint64(100), "").Return(nil, "", serviceerrors.ErrPermissionDenied)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Wrong match. expect: 403, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, r, w := starSrvNew(t, nil)

		req, _ := http.NewRequest("GET", "/service_agents/contact_stars", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong match. expect: 401, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func Test_serviceAgentsContactsGET_Starred(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)
	contacts := []*cmcontact.WebhookMessage{{
		Identity:  commonidentity.Identity{ID: starSrvContactID, CustomerID: starSrvAgent.CustomerID},
		FirstName: "John",
		TMCreate:  &created,
	}}

	t.Run("starred=true uses the dedicated path with the agent paging", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/contacts?starred=true&page_size=7&page_token=2026-06-01T09:30:01.000000Z", nil)
		mockSvc.EXPECT().ServiceAgentContactListStarred(req.Context(), starSrvAgent, uint64(7), "2026-06-01T09:30:01.000000Z").Return(contacts, "2026-11-02T09:15:30.123456Z", nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
		}
		var body struct {
			Result []struct {
				ID        string `json:"id"`
				FirstName string `json:"first_name"`
			} `json:"result"`
			NextPageToken string `json:"next_page_token"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("could not parse the body: %v", err)
		}
		if len(body.Result) != 1 || body.Result[0].ID != starSrvContactID.String() || body.NextPageToken != "2026-11-02T09:15:30.123456Z" {
			t.Errorf("Wrong body: %s", w.Body.String())
		}
	})

	t.Run("starred=true defaults the page size to 100", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/contacts?starred=true", nil)
		mockSvc.EXPECT().ServiceAgentContactListStarred(req.Context(), starSrvAgent, uint64(100), "").Return(nil, "", nil)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != `{"result":[],"next_page_token":""}` {
			t.Errorf("Wrong match. got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("starred service error maps through", func(t *testing.T) {
		mockSvc, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/contacts?starred=true", nil)
		mockSvc.EXPECT().ServiceAgentContactListStarred(req.Context(), starSrvAgent, uint64(100), "").Return(nil, "", serviceerrors.ErrPermissionDenied)

		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Wrong match. expect: 403, got: %d, body: %s", w.Code, w.Body.String())
		}
	})

	for _, query := range []string{"?starred=false", ""} {
		t.Run("plain list for "+query, func(t *testing.T) {
			mockSvc, r, w := starSrvNew(t, starSrvAgent)

			req, _ := http.NewRequest("GET", "/service_agents/contacts"+query, nil)
			// the starred path must not be called: gomock fails on an unexpected call
			mockSvc.EXPECT().ServiceAgentContactList(req.Context(), starSrvAgent, uint64(100), "", map[string]string{
				"customer_id": starSrvAgent.CustomerID.String(),
				"deleted":     "false",
			}).Return(contacts, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: 200, got: %d, body: %s", w.Code, w.Body.String())
			}
		})
	}

	t.Run("a non-boolean starred is a bad request", func(t *testing.T) {
		_, r, w := starSrvNew(t, starSrvAgent)

		req, _ := http.NewRequest("GET", "/service_agents/contacts?starred=maybe", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: 400, got: %d, body: %s", w.Code, w.Body.String())
		}
	})
}

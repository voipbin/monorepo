package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsContacts(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContacts []*cmcontact.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[string]string
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseContacts: []*cmcontact.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
						CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
					},
					FirstName: "John",
					LastName:  "Doe",
					TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectFilters: map[string]string{
				"customer_id": "5f621078-8004-11ec-aea5-d3a320e3b3c0",
				"deleted":     "false",
			},
			expectRes: `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 results",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseContacts: []*cmcontact.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
						CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
					},
					FirstName: "John",
					LastName:  "Doe",
					TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
						CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
					},
					FirstName: "Jane",
					LastName:  "Smith",
					TMCreate:  timePtr("2020-09-20T03:23:21.995002Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectFilters: map[string]string{
				"customer_id": "5f621078-8004-11ec-aea5-d3a320e3b3c0",
				"deleted":     "false",
			},
			expectRes: `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null},{"id":"2c1abc5c-500d-11ec-8896-9bca824c5a63","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"Jane","last_name":"Smith","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995002Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995002Z"}`,
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
			mockSvc.EXPECT().ServiceAgentContactList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseContacts, nil)

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

func Test_PostServiceAgentsContacts(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectFirstName    string
		expectLastName     string
		expectDisplayName  string
		expectCompany      string
		expectJobTitle     string
		expectSource       string
		expectExternalID   string
		expectNotes        string
		expectPhoneNumbers []cmrequest.PhoneNumberCreate
		expectEmails       []cmrequest.EmailCreate
		expectTagIDs       []uuid.UUID
		expectRes          string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts",
			reqBody:  []byte(`{"first_name":"John","last_name":"Doe","display_name":"John Doe","company":"Acme","job_title":"Engineer","source":"api","external_id":"ext-123","notes":"test note"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Company:     "Acme",
				JobTitle:    "Engineer",
				Source:      "api",
				ExternalID:  "ext-123",
				TMCreate:    timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectFirstName:    "John",
			expectLastName:     "Doe",
			expectDisplayName:  "John Doe",
			expectCompany:      "Acme",
			expectJobTitle:     "Engineer",
			expectSource:       "api",
			expectExternalID:   "ext-123",
			expectNotes:        "test note",
			expectPhoneNumbers: []cmrequest.PhoneNumberCreate{},
			expectEmails:       []cmrequest.EmailCreate{},
			expectTagIDs:       []uuid.UUID{},
			expectRes:          `{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"Acme","job_title":"Engineer","source":"api","external_id":"ext-123","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactCreate(
				req.Context(),
				tt.agent,
				tt.expectFirstName,
				tt.expectLastName,
				tt.expectDisplayName,
				tt.expectCompany,
				tt.expectJobTitle,
				tt.expectSource,
				tt.expectExternalID,
				tt.expectNotes,
				tt.expectPhoneNumbers,
				tt.expectEmails,
				tt.expectTagIDs,
			).Return(tt.responseContact, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusCreated, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_GetServiceAgentsContactsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentContactGet(req.Context(), tt.agent, tt.expectContactID).Return(tt.responseContact, nil)

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

func Test_PutServiceAgentsContactsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID   uuid.UUID
		expectFirstName   *string
		expectLastName    *string
		expectDisplayName *string
		expectCompany     *string
		expectJobTitle    *string
		expectExternalID  *string
		expectNotes       *string
		expectRes         string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff",
			reqBody:  []byte(`{"first_name":"Updated","last_name":"Name"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "Updated",
				LastName:  "Name",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID:   uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectFirstName:   strPtr("Updated"),
			expectLastName:    strPtr("Name"),
			expectDisplayName: nil,
			expectCompany:     nil,
			expectJobTitle:    nil,
			expectExternalID:  nil,
			expectNotes:       nil,
			expectRes:         `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"Updated","last_name":"Name","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactUpdate(
				req.Context(),
				tt.agent,
				tt.expectContactID,
				tt.expectFirstName,
				tt.expectLastName,
				tt.expectDisplayName,
				tt.expectCompany,
				tt.expectJobTitle,
				tt.expectExternalID,
				tt.expectNotes,
			).Return(tt.responseContact, nil)

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

func Test_DeleteServiceAgentsContactsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
				TMDelete:  timePtr("2020-09-20T04:00:00.000000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":"2020-09-20T04:00:00Z"}`,
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
			mockSvc.EXPECT().ServiceAgentContactDelete(req.Context(), tt.agent, tt.expectContactID).Return(tt.responseContact, nil)

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

func Test_GetServiceAgentsContactsLookup(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectPhone string
		expectEmail string
		expectRes   string
	}{
		{
			name: "lookup by phone",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/lookup?phone=%2B15551234567",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectPhone: "+15551234567",
			expectEmail: "",
			expectRes:   `{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
		},
		{
			name: "lookup by email",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/lookup?email=john@example.com",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectPhone: "",
			expectEmail: "john@example.com",
			expectRes:   `{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentContactLookup(req.Context(), tt.agent, tt.expectPhone, tt.expectEmail).Return(tt.responseContact, nil)

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

func Test_PostServiceAgentsContactsIdPhoneNumbers(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID  uuid.UUID
		expectNumber     string
		expectPhoneType  string
		expectIsPrimary  bool
		expectRes        string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/phone_numbers",
			reqBody:  []byte(`{"number":"+15551234567","type":"mobile","is_primary":true}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:        uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number:    "+15551234567",
						Type:      "mobile",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID:  uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectNumber:     "+15551234567",
			expectPhoneType:  "mobile",
			expectIsPrimary:  true,
			expectRes:        `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","phone_numbers":[{"id":"a1b2c3d4-0001-11ec-0001-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","number":"+15551234567","type":"mobile","is_primary":true,"tm_create":null}],"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactPhoneNumberCreate(req.Context(), tt.agent, tt.expectContactID, tt.expectNumber, tt.expectPhoneType, tt.expectIsPrimary).Return(tt.responseContact, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusCreated, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_PutServiceAgentsContactsIdPhoneNumbersPhoneNumberId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID     uuid.UUID
		expectPhoneNumberID uuid.UUID
		expectFields        map[string]any
		expectRes           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/phone_numbers/a1b2c3d4-0001-11ec-0001-000000000001",
			reqBody:  []byte(`{"number":"+15559999999"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number: "+15559999999",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID:     uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectPhoneNumberID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
			expectFields: map[string]any{
				"number": "+15559999999",
			},
			expectRes: `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","phone_numbers":[{"id":"a1b2c3d4-0001-11ec-0001-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","number":"+15559999999","type":"","is_primary":false,"tm_create":null}],"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactPhoneNumberUpdate(req.Context(), tt.agent, tt.expectContactID, tt.expectPhoneNumberID, tt.expectFields).Return(tt.responseContact, nil)

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

func Test_DeleteServiceAgentsContactsIdPhoneNumbersPhoneNumberId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectContactID     uuid.UUID
		expectPhoneNumberID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/phone_numbers/a1b2c3d4-0001-11ec-0001-000000000001",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID:     uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectPhoneNumberID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
			expectRes:           `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentContactPhoneNumberDelete(req.Context(), tt.agent, tt.expectContactID, tt.expectPhoneNumberID).Return(tt.responseContact, nil)

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

func Test_PostServiceAgentsContactsIdEmails(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectAddress   string
		expectEmailType string
		expectIsPrimary bool
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/emails",
			reqBody:  []byte(`{"address":"john@example.com","type":"work","is_primary":true}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				Emails: []cmcontact.Email{
					{
						ID:        uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address:   "john@example.com",
						Type:      "work",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectAddress:   "john@example.com",
			expectEmailType: "work",
			expectIsPrimary: true,
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","emails":[{"id":"b2c3d4e5-0002-11ec-0002-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","address":"john@example.com","type":"work","is_primary":true,"tm_create":null}],"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactEmailCreate(req.Context(), tt.agent, tt.expectContactID, tt.expectAddress, tt.expectEmailType, tt.expectIsPrimary).Return(tt.responseContact, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusCreated, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_PutServiceAgentsContactsIdEmailsEmailId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectEmailID   uuid.UUID
		expectFields    map[string]any
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/emails/b2c3d4e5-0002-11ec-0002-000000000002",
			reqBody:  []byte(`{"address":"updated@example.com"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				Emails: []cmcontact.Email{
					{
						ID:      uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address: "updated@example.com",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectEmailID:   uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
			expectFields: map[string]any{
				"address": "updated@example.com",
			},
			expectRes: `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","emails":[{"id":"b2c3d4e5-0002-11ec-0002-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","address":"updated@example.com","type":"","is_primary":false,"tm_create":null}],"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactEmailUpdate(req.Context(), tt.agent, tt.expectContactID, tt.expectEmailID, tt.expectFields).Return(tt.responseContact, nil)

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

func Test_DeleteServiceAgentsContactsIdEmailsEmailId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectEmailID   uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/emails/b2c3d4e5-0002-11ec-0002-000000000002",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectEmailID:   uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentContactEmailDelete(req.Context(), tt.agent, tt.expectContactID, tt.expectEmailID).Return(tt.responseContact, nil)

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

func Test_PostServiceAgentsContactsIdTags(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectTagID     uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/tags",
			reqBody:  []byte(`{"tag_id":"d4e5f6a7-0003-11ec-0003-000000000003"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectTagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tag_ids":["d4e5f6a7-0003-11ec-0003-000000000003"],"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactTagAdd(req.Context(), tt.agent, tt.expectContactID, tt.expectTagID).Return(tt.responseContact, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusCreated, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_DeleteServiceAgentsContactsIdTagsTagId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectTagID     uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
			}),

			reqQuery: "/service_agents/contacts/c07ff34e-500d-11ec-8393-2bc7870b7eff/tags/d4e5f6a7-0003-11ec-0003-000000000003",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectTagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
			expectRes:       `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"5f621078-8004-11ec-aea5-d3a320e3b3c0","first_name":"John","last_name":"Doe","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentContactTagRemove(req.Context(), tt.agent, tt.expectContactID, tt.expectTagID).Return(tt.responseContact, nil)

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

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// Test_serviceAgentsContactsPost_MissingAuthIdentity verifies
// PostServiceAgentsContacts emits the canonical UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED envelope when auth_identity is missing from
// the gin context.
func Test_serviceAgentsContactsPost_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/service_agents/contacts",
		[]byte(`{"first_name":"John","last_name":"Doe"}`))
}

// Test_serviceAgentsContactsPost_InvalidJSONBody verifies
// PostServiceAgentsContacts rejects a malformed JSON body with
// INVALID_ARGUMENT / INVALID_JSON_BODY before the servicehandler is
// consulted.
func Test_serviceAgentsContactsPost_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

	req, _ := http.NewRequest(http.MethodPost, "/service_agents/contacts", bytes.NewBufferString(`{not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_serviceAgentsContactsIDPut_InvalidID verifies
// PutServiceAgentsContactsId rejects a malformed UUID in the path with
// INVALID_ARGUMENT / INVALID_ID before the servicehandler is consulted.
func Test_serviceAgentsContactsIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

	req, _ := http.NewRequest(http.MethodPut, "/service_agents/contacts/not-a-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

// Test_serviceAgentsContactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID
// verifies DeleteServiceAgentsContactsIdPhoneNumbersPhoneNumberId
// returns INVALID_ARGUMENT / INVALID_ID when the parent contact id is
// a valid UUID but the nested phone_number_id is malformed. Exercises
// the dual-ID validation path with a distinguishing message.
func Test_serviceAgentsContactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

	req, _ := http.NewRequest(http.MethodDelete, "/service_agents/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/phone_numbers/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

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

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetContacts(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseContacts []*cmcontact.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts",

			responseContacts: []*cmcontact.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectRes: `{"result":[{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
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
			mockSvc.EXPECT().ContactList(req.Context(), tt.agent, uint64(100), "", gomock.Any()).Return(tt.responseContacts, nil)

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

func Test_PostContacts(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts",
			reqBody:  []byte(`{"first_name":"John","last_name":"Doe"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectRes: `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactCreate(
				req.Context(),
				tt.agent,
				"John",
				"Doe",
				"",
				"",
				"",
				"",
				"",
				"",
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
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

func Test_GetContactsId(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ContactGet(req.Context(), tt.agent, tt.expectContactID).Return(tt.responseContact, nil)

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

func Test_PutContactsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseContact *cmcontact.WebhookMessage

		expectContactID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5",
			reqBody:  []byte(`{"first_name":"Jane"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				FirstName: "Jane",
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"Jane","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactUpdate(
				req.Context(),
				tt.agent,
				tt.expectContactID,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
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

func Test_DeleteContactsId(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: timePtr("2020-09-20T04:00:00.000000Z"),
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":"2020-09-20T04:00:00Z"}`,
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
			mockSvc.EXPECT().ContactDelete(req.Context(), tt.agent, tt.expectContactID).Return(tt.responseContact, nil)

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

func Test_GetContactsLookup(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/lookup?phone=%2B1234567890",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectPhone: "+1234567890",
			expectEmail: "",
			expectRes:   `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ContactLookup(req.Context(), tt.agent, tt.expectPhone, tt.expectEmail).Return(tt.responseContact, nil)

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

func Test_PostContactsIdPhoneNumbers(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/phone-numbers",
			reqBody:  []byte(`{"number":"+12125551234"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Number: "+12125551234",
					},
				},
			},

			expectContactID:  uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectNumber:     "+12125551234",
			expectPhoneType:  "",
			expectIsPrimary:  false,
			expectRes:        `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","phone_numbers":[{"id":"a1b2c3d4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","number":"+12125551234","type":"","is_primary":false,"tm_create":null}],"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactPhoneNumberCreate(req.Context(), tt.agent, tt.expectContactID, tt.expectNumber, tt.expectPhoneType, tt.expectIsPrimary).Return(tt.responseContact, nil)

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

func Test_PutContactsIdPhoneNumbersPhoneNumberId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/phone-numbers/a1b2c3d4-5066-11ec-ab34-23643cfdc1c5",
			reqBody:  []byte(`{"number":"+12125551234"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Number: "+12125551234",
					},
				},
			},

			expectContactID:     uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectPhoneNumberID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:           `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","phone_numbers":[{"id":"a1b2c3d4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","number":"+12125551234","type":"","is_primary":false,"tm_create":null}],"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactPhoneNumberUpdate(req.Context(), tt.agent, tt.expectContactID, tt.expectPhoneNumberID, gomock.Any()).Return(tt.responseContact, nil)

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

func Test_DeleteContactsIdPhoneNumbersPhoneNumberId(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/phone-numbers/a1b2c3d4-5066-11ec-ab34-23643cfdc1c5",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectContactID:     uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectPhoneNumberID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:           `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ContactPhoneNumberDelete(req.Context(), tt.agent, tt.expectContactID, tt.expectPhoneNumberID).Return(tt.responseContact, nil)

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

func Test_PostContactsIdEmails(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/emails",
			reqBody:  []byte(`{"address":"test@example.com"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Emails: []cmcontact.Email{
					{
						ID:      uuid.FromStringOrNil("e1f2a3b4-5066-11ec-ab34-23643cfdc1c5"),
						Address: "test@example.com",
					},
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectAddress:   "test@example.com",
			expectEmailType: "",
			expectIsPrimary: false,
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","emails":[{"id":"e1f2a3b4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","address":"test@example.com","type":"","is_primary":false,"tm_create":null}],"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactEmailCreate(req.Context(), tt.agent, tt.expectContactID, tt.expectAddress, tt.expectEmailType, tt.expectIsPrimary).Return(tt.responseContact, nil)

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

func Test_PutContactsIdEmailsEmailId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/emails/e1f2a3b4-5066-11ec-ab34-23643cfdc1c5",
			reqBody:  []byte(`{"address":"new@example.com"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Emails: []cmcontact.Email{
					{
						ID:      uuid.FromStringOrNil("e1f2a3b4-5066-11ec-ab34-23643cfdc1c5"),
						Address: "new@example.com",
					},
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectEmailID:   uuid.FromStringOrNil("e1f2a3b4-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","emails":[{"id":"e1f2a3b4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"00000000-0000-0000-0000-000000000000","address":"new@example.com","type":"","is_primary":false,"tm_create":null}],"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactEmailUpdate(req.Context(), tt.agent, tt.expectContactID, tt.expectEmailID, gomock.Any()).Return(tt.responseContact, nil)

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

func Test_DeleteContactsIdEmailsEmailId(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/emails/e1f2a3b4-5066-11ec-ab34-23643cfdc1c5",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectEmailID:   uuid.FromStringOrNil("e1f2a3b4-5066-11ec-ab34-23643cfdc1c5"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ContactEmailDelete(req.Context(), tt.agent, tt.expectContactID, tt.expectEmailID).Return(tt.responseContact, nil)

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

func Test_PostContactsIdTags(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/tags",
			reqBody:  []byte(`{"tag_id":"bd8cee04-4f21-11ec-9955-db7041b6d997"}`),

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectTagID:     uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tag_ids":["bd8cee04-4f21-11ec-9955-db7041b6d997"],"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ContactTagAdd(req.Context(), tt.agent, tt.expectContactID, tt.expectTagID).Return(tt.responseContact, nil)

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

func Test_DeleteContactsIdTagsTagId(t *testing.T) {

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
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			}),

			reqQuery: "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/tags/bd8cee04-4f21-11ec-9955-db7041b6d997",

			responseContact: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			expectTagID:     uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
			expectRes:       `{"id":"3147612c-5066-11ec-ab34-23643cfdc1c5","customer_id":"5f621078-8e5f-11ee-97b2-cfe7337b701c","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ContactTagRemove(req.Context(), tt.agent, tt.expectContactID, tt.expectTagID).Return(tt.responseContact, nil)

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

// Test_contactsPost_MissingAuthIdentity verifies PostContacts emits the
// canonical UNAUTHENTICATED / AUTHENTICATION_REQUIRED envelope when
// auth_identity is missing from the gin context.
func Test_contactsPost_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/contacts",
		[]byte(`{"first_name":"John","last_name":"Doe"}`))
}

// Test_contactsPost_InvalidJSONBody verifies PostContacts rejects a
// malformed JSON body with INVALID_ARGUMENT / INVALID_JSON_BODY before
// the servicehandler is consulted.
func Test_contactsPost_InvalidJSONBody(t *testing.T) {
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

	req, _ := http.NewRequest(http.MethodPost, "/contacts", bytes.NewBufferString(`{not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_contactsIDPut_InvalidID verifies PutContactsId rejects a malformed
// UUID in the path with INVALID_ARGUMENT / INVALID_ID before the
// servicehandler is consulted.
func Test_contactsIDPut_InvalidID(t *testing.T) {
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

	req, _ := http.NewRequest(http.MethodPut, "/contacts/not-a-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

// Test_contactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID
// verifies DeleteContactsIdPhoneNumbersPhoneNumberId returns
// INVALID_ARGUMENT / INVALID_ID when the parent contact id is a valid
// UUID but the nested phone_number_id is malformed. Exercises the
// dual-ID validation path with a distinguishing message.
func Test_contactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID(t *testing.T) {
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

	req, _ := http.NewRequest(http.MethodDelete, "/contacts/3147612c-5066-11ec-ab34-23643cfdc1c5/phone-numbers/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

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
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_customerGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCustomer *cscustomer.WebhookMessage

		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/customer",

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
			},

			expectedRes: `{"id":"e25f1af8-c44f-11ef-9d46-bfaf61e659c2","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","metadata":{"rtp_debug":false},"email_verified":false,"status":"","identity_verification_status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().CustomerSelfGet(req.Context(), tt.agent).Return(tt.responseCustomer, nil)

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

func Test_customerPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectecName          string
		expectedDetail        string
		expectedEmail         string
		expectedPhoneNumber   string
		expectedAddress       string
		expectedWebhookMethod cscustomer.WebhookMethod
		expectedWebhookURI    string
		expectedRes           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4afd144c-c451-11ef-a8d8-6fd67202355e"),
					CustomerID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/customer",
			reqBody:  []byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
			},

			expectecName:          "new name",
			expectedDetail:        "new detail",
			expectedEmail:         "test@test.com",
			expectedPhoneNumber:   "+821100000001",
			expectedAddress:       "somewhere",
			expectedWebhookMethod: cscustomer.WebhookMethodPost,
			expectedWebhookURI:    "test.com",
			expectedRes:           `{"id":"4b7dcc68-c451-11ef-a289-33cbfe065115","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","metadata":{"rtp_debug":false},"email_verified":false,"status":"","identity_verification_status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSelfUpdate(req.Context(), tt.agent, tt.expectecName, tt.expectedDetail, tt.expectedEmail, tt.expectedPhoneNumber, tt.expectedAddress, tt.expectedWebhookMethod, tt.expectedWebhookURI).Return(tt.responseCustomer, nil)

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

func Test_customerBillingAccountIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedBillingAccountID uuid.UUID
		expectedRes              string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("23ad14fa-c514-11ef-a03b-af3d499fdf18"),
					CustomerID: uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/customer/billing_account_id",
			reqBody:  []byte(`{"billing_account_id":"245bc55e-c514-11ef-85d3-23d66dfc487a"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
			},

			expectedBillingAccountID: uuid.FromStringOrNil("245bc55e-c514-11ef-85d3-23d66dfc487a"),
			expectedRes:              `{"id":"2422306e-c514-11ef-a89d-2f0585ee15f9","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","metadata":{"rtp_debug":false},"email_verified":false,"status":"","identity_verification_status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSelfUpdateBillingAccountID(req.Context(), tt.agent, tt.expectedBillingAccountID).Return(tt.responseCustomer, nil)

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

func Test_customerDefaultOutgoingSourceNumberIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedDefaultOutgoingSourceNumberID uuid.UUID
		expectedRes                           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1a2b3c4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("d2b3c4e5-f6a7-8901-bcde-f12345678901"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/customer/default_outgoing_source_number_id",
			reqBody:  []byte(`{"default_outgoing_source_number_id":"d3c4e5f6-a7b8-9012-cdef-123456789012"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID:                            uuid.FromStringOrNil("d2b3c4e5-f6a7-8901-bcde-f12345678901"),
				DefaultOutgoingSourceNumberID: uuid.FromStringOrNil("d3c4e5f6-a7b8-9012-cdef-123456789012"),
			},

			expectedDefaultOutgoingSourceNumberID: uuid.FromStringOrNil("d3c4e5f6-a7b8-9012-cdef-123456789012"),
			expectedRes:                           `{"id":"d2b3c4e5-f6a7-8901-bcde-f12345678901","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"d3c4e5f6-a7b8-9012-cdef-123456789012","metadata":{"rtp_debug":false},"email_verified":false,"status":"","identity_verification_status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSelfUpdateDefaultOutgoingSourceNumberID(req.Context(), tt.agent, tt.expectedDefaultOutgoingSourceNumberID).Return(tt.responseCustomer, nil)

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

// Test_customerGET_MissingAuthIdentity exercises the auth-identity-missing
// branch of GetCustomer. Without auth_identity in the gin context, the
// handler must emit UNAUTHENTICATED / AUTHENTICATION_REQUIRED with a
// populated request_id (RequestID middleware runs before the handler).
func Test_customerGET_MissingAuthIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	// Intentionally do not set auth_identity.
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/customer", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED")
}

// Test_customerGET_ServiceError exercises the servicehandler-failure path
// through abortWithServiceError. The translator's sentinel match
// maps "customer not found" to NOT_FOUND / RESOURCE_NOT_FOUND.
func Test_customerGET_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/customer", nil)
	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().CustomerSelfGet(gomock.Any(), agent).Return(nil, fmt.Errorf("%w: customer not found", serviceerrors.ErrNotFound))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND")
}

// Test_customerPut_InvalidJSONBody verifies PutCustomer rejects malformed
// JSON with INVALID_ARGUMENT / INVALID_JSON_BODY.
func Test_customerPut_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("4afd144c-c451-11ef-a8d8-6fd67202355e"),
			CustomerID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// Intentionally invalid JSON body.
	req, _ := http.NewRequest(http.MethodPut, "/customer", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_customerBillingAccountIDPut_InvalidID verifies that a syntactically
// invalid UUID in the billing_account_id body triggers INVALID_ID.
func Test_customerBillingAccountIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("23ad14fa-c514-11ef-a03b-af3d499fdf18"),
			CustomerID: uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// Valid JSON shape, but the embedded id is not a UUID — uuid.FromStringOrNil
	// returns uuid.Nil and the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodPut, "/customer/billing_account_id", bytes.NewBufferString(`{"billing_account_id":"not-a-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

func Test_customerMetadataPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedMetadata cscustomer.Metadata
		expectedRes      string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/customer/metadata",
			reqBody:  []byte(`{"rtp_debug":true}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
				Metadata: cscustomer.Metadata{
					RTPDebug: true,
				},
			},

			expectedMetadata: cscustomer.Metadata{
				RTPDebug: true,
			},
			expectedRes: `{"id":"b2c3d4e5-f6a7-8901-bcde-f12345678901","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","metadata":{"rtp_debug":true},"email_verified":false,"status":"","identity_verification_status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSelfUpdateMetadata(req.Context(), tt.agent, tt.expectedMetadata).Return(tt.responseCustomer, nil)

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

// assertMissingAuthIdentity is a small helper that reduces boilerplate for
// the auth-identity-missing branch of each handler. It builds a minimal
// Gin router with the RequestID middleware installed (but no auth_identity
// set), dispatches the request, and asserts the UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED error envelope.
func assertMissingAuthIdentity(t *testing.T, method, path string, body []byte) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	// Intentionally do not set auth_identity.
	openapi_server.RegisterHandlers(r, h)

	var req *http.Request
	if body == nil {
		req, _ = http.NewRequest(method, path, nil)
	} else {
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED")
}

// Test_customerPut_MissingAuthIdentity exercises the auth-identity-missing
// branch of PutCustomer.
func Test_customerPut_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPut, "/customer",
		[]byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`))
}

// Test_customerMetadataPut_MissingAuthIdentity exercises the
// auth-identity-missing branch of PutCustomerMetadata.
func Test_customerMetadataPut_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPut, "/customer/metadata", []byte(`{"rtp_debug":true}`))
}

// Test_customerBillingAccountIDPut_MissingAuthIdentity exercises the
// auth-identity-missing branch of PutCustomerBillingAccountId.
func Test_customerBillingAccountIDPut_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPut, "/customer/billing_account_id",
		[]byte(`{"billing_account_id":"245bc55e-c514-11ef-85d3-23d66dfc487a"}`))
}

// Test_customerDefaultOutgoingSourceNumberIDPut_MissingAuthIdentity
// exercises the auth-identity-missing branch of
// PutCustomerDefaultOutgoingSourceNumberId.
func Test_customerDefaultOutgoingSourceNumberIDPut_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPut, "/customer/default_outgoing_source_number_id",
		[]byte(`{"default_outgoing_source_number_id":"d3c4e5f6-a7b8-9012-cdef-123456789012"}`))
}

// Test_customerDefaultOutgoingSourceNumberIDPut_InvalidID verifies that
// when the body parses as a valid UUID but resolves to uuid.Nil (all
// zeros), the handler rejects the request with INVALID_ID. The field is
// typed as openapi_types.UUID so malformed strings fail at JSON binding
// with INVALID_JSON_BODY; the Nil-UUID path is the INVALID_ID branch.
func Test_customerDefaultOutgoingSourceNumberIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1a2b3c4-e5f6-7890-abcd-ef1234567890"),
			CustomerID: uuid.FromStringOrNil("d2b3c4e5-f6a7-8901-bcde-f12345678901"),
		},
		Permission: amagent.PermissionCustomerAdmin,
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

	// Valid UUID shape but uuid.Nil — uuid.FromStringOrNil returns Nil
	// and the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodPut, "/customer/default_outgoing_source_number_id",
		bytes.NewBufferString(`{"default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

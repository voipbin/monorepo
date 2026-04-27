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

func Test_customersPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.Customer

		expectName          string
		expectDetail        string
		expectEmail         string
		expectPhoneNumber   string
		expectAddress       string
		expectWebhookMethod cscustomer.WebhookMethod
		expectWebhookURI    string
		expectRes           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("271353a8-83f3-11ec-9386-8be19d563155"),
			},

			expectName:          "test name",
			expectDetail:        "test detail",
			expectEmail:         "test@test.com",
			expectPhoneNumber:   "+821100000001",
			expectAddress:       "somewhere",
			expectWebhookMethod: cscustomer.WebhookMethodPost,
			expectWebhookURI:    "test.com",
			expectRes:           `{"id":"271353a8-83f3-11ec-9386-8be19d563155","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().CustomerCreate(
				req.Context(),
				tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectEmail,
				tt.expectPhoneNumber,
				tt.expectAddress,
				tt.expectWebhookMethod,
				tt.expectWebhookURI,
			).Return(tt.responseCustomer, nil)

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

func Test_customersGet(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCustomers []*cscustomer.Customer

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[string]string
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/customers?page_size=20&page_token=2020-09-20T03:23:20.995000Z",

			responseCustomers: []*cscustomer.Customer{
				{
					ID: uuid.FromStringOrNil("52bac7ec-83f4-11ec-a083-c3cf3f92a2e3"),
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectRes: `{"result":[{"id":"52bac7ec-83f4-11ec-a083-c3cf3f92a2e3","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseCustomers, nil)

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

func Test_customersIDGet(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCustomer *cscustomer.Customer

		expectCustomerID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectRes:        `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerGet(req.Context(), tt.agent, tt.expectCustomerID).Return(tt.responseCustomer, nil)

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

func Test_customersIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.Customer

		expectCustomerID    uuid.UUID
		expectName          string
		expectDetail        string
		expectEmail         string
		expectPhoneNumber   string
		expectAddress       string
		expectWebhookMethod cscustomer.WebhookMethod
		expectWebhookURI    string
		expectRes           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",
			reqBody:  []byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID:    uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectName:          "new name",
			expectDetail:        "new detail",
			expectEmail:         "test@test.com",
			expectPhoneNumber:   "+821100000001",
			expectAddress:       "somewhere",
			expectWebhookMethod: cscustomer.WebhookMethodPost,
			expectWebhookURI:    "test.com",
			expectRes:           `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().CustomerUpdate(req.Context(), tt.agent, tt.expectCustomerID, tt.expectName, tt.expectDetail, tt.expectEmail, tt.expectPhoneNumber, tt.expectAddress, tt.expectWebhookMethod, tt.expectWebhookURI).Return(tt.responseCustomer, nil)

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

func Test_customersIDDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCustomer *cscustomer.Customer

		expectCustomerID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectRes:        `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerDelete(req.Context(), tt.agent, tt.expectCustomerID).Return(tt.responseCustomer, nil)

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

func Test_customersIDBillingAccountIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.Customer

		expectedCustomerID       uuid.UUID
		expectedBillingAccountID uuid.UUID
		expectedRes              string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/cc876058-1773-11ee-9694-136fe246dd34/billing_account_id",
			reqBody:  []byte(`{"billing_account_id":"ccc776b6-1773-11ee-bea5-d78345c015af"}`),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			},

			expectedCustomerID:       uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			expectedBillingAccountID: uuid.FromStringOrNil("ccc776b6-1773-11ee-bea5-d78345c015af"),
			expectedRes:              `{"id":"cc876058-1773-11ee-9694-136fe246dd34","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().CustomerUpdateBillingAccountID(req.Context(), tt.agent, tt.expectedCustomerID, tt.expectedBillingAccountID).Return(tt.responseCustomer, nil)

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

func Test_customersIDDefaultOutgoingSourceNumberIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.Customer

		expectedCustomerID                    uuid.UUID
		expectedDefaultOutgoingSourceNumberID uuid.UUID
		expectedRes                           string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/cc876058-1773-11ee-9694-136fe246dd34/default_outgoing_source_number_id",
			reqBody:  []byte(`{"default_outgoing_source_number_id":"e1f2a3b4-c5d6-7890-abcd-ef1234567890"}`),

			responseCustomer: &cscustomer.Customer{
				ID:                            uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				DefaultOutgoingSourceNumberID: uuid.FromStringOrNil("e1f2a3b4-c5d6-7890-abcd-ef1234567890"),
			},

			expectedCustomerID:                    uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			expectedDefaultOutgoingSourceNumberID: uuid.FromStringOrNil("e1f2a3b4-c5d6-7890-abcd-ef1234567890"),
			expectedRes:                           `{"id":"cc876058-1773-11ee-9694-136fe246dd34","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"e1f2a3b4-c5d6-7890-abcd-ef1234567890","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().CustomerUpdateDefaultOutgoingSourceNumberID(req.Context(), tt.agent, tt.expectedCustomerID, tt.expectedDefaultOutgoingSourceNumberID).Return(tt.responseCustomer, nil)

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

func Test_customersIDMetadataPut(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.Customer

		expectedCustomerID uuid.UUID
		expectedMetadata   cscustomer.Metadata
		expectedRes        string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/customers/cc876058-1773-11ee-9694-136fe246dd34/metadata",
			reqBody:  []byte(`{"rtp_debug":true}`),

			responseCustomer: &cscustomer.Customer{
				ID:       uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				Metadata: cscustomer.Metadata{RTPDebug: true},
			},

			expectedCustomerID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			expectedMetadata:   cscustomer.Metadata{RTPDebug: true},
			expectedRes:        `{"id":"cc876058-1773-11ee-9694-136fe246dd34","billing_account_id":"00000000-0000-0000-0000-000000000000","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","identity_verification_status":"","metadata":{"rtp_debug":true},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().CustomerUpdateMetadata(req.Context(), tt.agent, tt.expectedCustomerID, tt.expectedMetadata).Return(tt.responseCustomer, nil)

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

// Test_customersIDMetadataPut_InvalidID verifies that a malformed UUID in
// the path triggers INVALID_ARGUMENT / INVALID_ID.
func Test_customersIDMetadataPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
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

	req, _ := http.NewRequest(http.MethodPut, "/customers/invalid-uuid/metadata", bytes.NewBufferString(`{"rtp_debug":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

// Test_customersIDMetadataPut_ServiceError exercises the servicehandler-failure
// path through abortWithServiceError. The translator's sentinel match
// maps "permission denied" to PERMISSION_DENIED / PERMISSION_DENIED.
func Test_customersIDMetadataPut_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})
	customerID := uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34")
	metadata := cscustomer.Metadata{RTPDebug: true}

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

	req, _ := http.NewRequest(http.MethodPut, "/customers/cc876058-1773-11ee-9694-136fe246dd34/metadata",
		bytes.NewBufferString(`{"rtp_debug":true}`))
	req.Header.Set("Content-Type", "application/json")

	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().CustomerUpdateMetadata(gomock.Any(), agent, customerID, metadata).Return(nil, fmt.Errorf("%w: permission denied", serviceerrors.ErrPermissionDenied))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusPermissionDenied, "PERMISSION_DENIED")
}

// Test_customersGET_MissingAuthIdentity exercises the auth-identity-missing
// branch of GetCustomers.
func Test_customersGET_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodGet, "/customers", nil)
}

// Test_customersIDPut_MissingAuthIdentity exercises the auth-identity-missing
// branch of PutCustomersId.
func Test_customersIDPut_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPut, "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",
		[]byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`))
}

// Test_customersIDDelete_MissingAuthIdentity exercises the auth-identity-missing
// branch of DeleteCustomersId.
func Test_customersIDDelete_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodDelete, "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f", nil)
}

// Test_customersPOST_InvalidJSONBody verifies PostCustomers rejects malformed
// JSON with INVALID_ARGUMENT / INVALID_JSON_BODY.
func Test_customersPOST_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
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

	// Intentionally invalid JSON body.
	req, _ := http.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_customersIDGet_InvalidID verifies that a malformed UUID in the path
// triggers INVALID_ARGUMENT / INVALID_ID on GetCustomersId.
func Test_customersIDGet_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
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

	req, _ := http.NewRequest(http.MethodGet, "/customers/invalid-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

// Test_customersIDGet_ServiceError exercises the servicehandler-failure path
// through abortWithServiceError on GetCustomersId. The translator's
// sentinel match (`serviceerrors.ErrNotFound`) maps to NOT_FOUND / RESOURCE_NOT_FOUND.
func Test_customersIDGet_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})
	customerID := uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f")

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

	req, _ := http.NewRequest(http.MethodGet, "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f", nil)
	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().CustomerGet(gomock.Any(), agent, customerID).Return(nil, fmt.Errorf("%w: customer not found", serviceerrors.ErrNotFound))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND")
}

// Test_customersIDBillingAccountIDPut_InvalidBodyID verifies the body-UUID
// INVALID_ID branch of PutCustomersIdBillingAccountId. The path UUID is
// valid, but the billing_account_id body field is not a UUID — the
// body field is typed as a plain string, so uuid.FromStringOrNil returns
// uuid.Nil and the handler rejects with INVALID_ID. This complements the
// path-UUID INVALID_ID test (malformed path) by exercising the second
// UUID parse site in the same handler.
func Test_customersIDBillingAccountIDPut_InvalidBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
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

	// Valid path UUID, but the embedded billing_account_id is not a UUID —
	// uuid.FromStringOrNil returns uuid.Nil and the handler rejects with
	// INVALID_ID.
	req, _ := http.NewRequest(http.MethodPut, "/customers/cc876058-1773-11ee-9694-136fe246dd34/billing_account_id",
		bytes.NewBufferString(`{"billing_account_id":"not-a-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

// Test_customersIDDefaultOutgoingSourceNumberIDPut_InvalidBodyID verifies
// the body-UUID INVALID_ID branch of PutCustomersIdDefaultOutgoingSourceNumberId.
// The body field is typed as openapi_types.UUID so malformed strings fail
// at JSON binding with INVALID_JSON_BODY; the Nil-UUID path is the
// INVALID_ID branch (valid shape, all zeros). This complements the path-UUID
// INVALID_ID test by exercising the second UUID parse site in the same
// handler.
func Test_customersIDDefaultOutgoingSourceNumberIDPut_InvalidBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
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

	// Valid path UUID and valid UUID shape in the body, but the body UUID
	// is all zeros — uuid.FromStringOrNil returns uuid.Nil and the handler
	// rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodPut, "/customers/cc876058-1773-11ee-9694-136fe246dd34/default_outgoing_source_number_id",
		bytes.NewBufferString(`{"default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}

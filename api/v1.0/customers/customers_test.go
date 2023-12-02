package customers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_customersPOST(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		req           request.BodyCustomersPOST
		username      string
		password      string
		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string
		permissionIDs []uuid.UUID

		expectRes *cscustomer.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers",

			request.BodyCustomersPOST{
				Username:      "test",
				Password:      "test password",
				Name:          "test name",
				Detail:        "test detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "somewhere",
				WebhookMethod: cscustomer.WebhookMethodPost,
				WebhookURI:    "test.com",
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			"test",
			"test password",
			"test name",
			"test detail",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",
			[]uuid.UUID{
				cspermission.PermissionAdmin.ID,
			},

			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("271353a8-83f3-11ec-9386-8be19d563155"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerCreate(
				req.Context(),
				&tt.agent,
				tt.username,
				tt.password,
				tt.customerName,
				tt.detail,
				tt.email,
				tt.phoneNumber,
				tt.address,
				tt.webhookMethod,
				tt.webhookURI,
				tt.permissionIDs,
			).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_customersGet(t *testing.T) {

	tests := []struct {
		name     string
		customer amagent.Agent
		target   string

		size  uint64
		token string

		resCustomers []*cscustomer.WebhookMessage
		expectRes    *response.BodyCustomersGET
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/customers?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			20,
			"2020-09-20 03:23:20.995000",

			[]*cscustomer.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("52bac7ec-83f4-11ec-a083-c3cf3f92a2e3"),
				},
			},
			&response.BodyCustomersGET{
				Result: []*cscustomer.WebhookMessage{
					{
						ID: uuid.FromStringOrNil("52bac7ec-83f4-11ec-a083-c3cf3f92a2e3"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerGets(req.Context(), &tt.customer, tt.size, tt.token).Return(tt.resCustomers, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_customersIDGet(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		expectRes *cscustomer.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),

			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerGet(req.Context(), &tt.agent, tt.id).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_customersIDPut(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		id  uuid.UUID
		req request.BodyCustomersIDPUT
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			request.BodyCustomersIDPUT{
				Name:          "new name",
				Detail:        "new detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "somewhere",
				WebhookMethod: cscustomer.WebhookMethodPost,
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdate(req.Context(), &tt.agent, tt.id, tt.req.Name, tt.req.Detail, tt.req.Email, tt.req.PhoneNumber, tt.req.Address, tt.req.WebhookMethod, tt.req.WebhookURI).Return(&cscustomer.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_customersIDDelete(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			req, _ := http.NewRequest("DELETE", tt.target, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerDelete(req.Context(), &tt.agent, tt.id).Return(&cscustomer.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_customersIDPermissionIDsPut(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyCustomersIDPermissionIDsPUT

		id uuid.UUID
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f/permission_ids",

			request.BodyCustomersIDPermissionIDsPUT{
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},

			uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdatePermissionIDs(req.Context(), &tt.agent, tt.id, tt.req.PermissionIDs).Return(&cscustomer.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_customersIDPasswordPut(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyCustomersIDPasswordPUT

		id uuid.UUID
	}{
		{
			"normal",
			amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"/v1.0/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f/password",

			request.BodyCustomersIDPasswordPUT{
				Password: "new password",
			},

			uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdatePassword(req.Context(), &tt.agent, tt.id, tt.req.Password).Return(&cscustomer.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_customersIDBillingAccountIDPut(t *testing.T) {

	tests := []struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyCustomersIDBillingAccountIDPUT

		expectCustomerID       uuid.UUID
		expectBillingAccountID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				ID:         uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			target: "/v1.0/customers/cc876058-1773-11ee-9694-136fe246dd34/billing_account_id",

			req: request.BodyCustomersIDBillingAccountIDPUT{
				BillingAccountID: uuid.FromStringOrNil("ccc776b6-1773-11ee-bea5-d78345c015af"),
			},

			expectCustomerID:       uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			expectBillingAccountID: uuid.FromStringOrNil("ccc776b6-1773-11ee-bea5-d78345c015af"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdateBillingAccountID(req.Context(), &tt.agent, tt.expectCustomerID, tt.expectBillingAccountID).Return(&cscustomer.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

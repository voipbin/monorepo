package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestExtensionsPOST(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer

		ext      string
		password string
		domainID uuid.UUID
		extName  string
		detail   string

		requestBody request.BodyExtensionsPOST
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},

			"test",
			"password",
			uuid.FromStringOrNil("7da5ed2e-6faf-11eb-92bd-bf4592baa4c4"),
			"test name",
			"test detail",

			request.BodyExtensionsPOST{
				Name:      "test name",
				Detail:    "test detail",
				DomainID:  uuid.FromStringOrNil("7da5ed2e-6faf-11eb-92bd-bf4592baa4c4"),
				Extension: "test",
				Password:  "password",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().ExtensionCreate(&tt.customer, tt.ext, tt.password, tt.domainID, tt.extName, tt.detail).Return(&rmextension.WebhookMessage{}, nil)
			req, _ := http.NewRequest("POST", "/v1.0/extensions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsGET(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		DomainID uuid.UUID

		expectExt []*rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("f92c19b2-6fb6-11eb-859c-0378f27fc22f"),
			[]*rmextension.WebhookMessage{
				{
					ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
					DomainID:  uuid.FromStringOrNil("f92c19b2-6fb6-11eb-859c-0378f27fc22f"),
					Name:      "test name",
					Detail:    "test detail",
					Extension: "test",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionGets(&tt.customer, tt.DomainID, uint64(10), "").Return(tt.expectExt, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/extensions?domain_id=%s", tt.DomainID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDGET(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		ext      *rmextension.Extension

		expectExt *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				CustomerID: uuid.FromStringOrNil("580a7a44-7ff8-11ec-916e-d35fe5e74591"),
				DomainID:   uuid.FromStringOrNil("2ff2b962-6fb0-11eb-a768-e3780d10e360"),
				Name:       "test name",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
			},
			&rmextension.WebhookMessage{
				ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				DomainID:  uuid.FromStringOrNil("2ff2b962-6fb0-11eb-a768-e3780d10e360"),
				Name:      "test name",
				Detail:    "test detail",
				Extension: "test",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionGet(&tt.customer, tt.ext.ID).Return(tt.expectExt, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/extensions/%s", tt.ext.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDPUT(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer

		extID    uuid.UUID
		extName  string
		detail   string
		password string

		requestBody request.BodyExtensionsIDPUT
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},

			uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
			"test name",
			"test detail",
			"update password",

			request.BodyExtensionsIDPUT{
				Name:     "test name",
				Detail:   "test detail",
				Password: "update password",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().ExtensionUpdate(&tt.customer, tt.extID, tt.extName, tt.detail, tt.password).Return(&rmextension.WebhookMessage{}, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/extensions/"+tt.extID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDDELETE(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		extID    uuid.UUID
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("be0c2b70-6fb0-11eb-849d-3f923b334d3b"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionDelete(&tt.customer, tt.extID).Return(&rmextension.WebhookMessage{}, nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/extensions/%s", tt.extID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

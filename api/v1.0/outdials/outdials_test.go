package outdials

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_OutdialsPOST(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		requestBody request.BodyOutdialsPOST

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			request.BodyOutdialsPOST{
				CampaignID: uuid.FromStringOrNil("5770a50e-1a94-45fc-9ba1-79064573cf06"),
				Name:       "test name",
				Detail:     "test detail",
				Data:       "test data",
			},

			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("99b197a5-010e-4f4e-b9fc-aae44e241ddb"),
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

			mockSvc.EXPECT().OutdialCreate(&tt.customer, tt.requestBody.CampaignID, tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Data).Return(tt.response, nil)
			req, _ := http.NewRequest("POST", "/v1.0/outdials", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDGET(t *testing.T) {

	tests := []struct {
		name      string
		customer  cscustomer.Customer
		outdialID uuid.UUID

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),

			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),
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

			mockSvc.EXPECT().OutdialGet(&tt.customer, tt.outdialID).Return(tt.response, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/outdials/%s", tt.outdialID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDDELETE(t *testing.T) {

	tests := []struct {
		name      string
		customer  cscustomer.Customer
		outdialID uuid.UUID

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
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

			mockSvc.EXPECT().OutdialDelete(&tt.customer, tt.outdialID).Return(tt.response, nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/outdials/%s", tt.outdialID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
			request.BodyOutdialsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
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

			mockSvc.EXPECT().OutdialUpdateBasicInfo(&tt.customer, tt.outdialID, tt.requestBody.Name, tt.requestBody.Detail).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDCampaignIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDCampaignIDPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
			request.BodyOutdialsIDCampaignIDPUT{
				CampaignID: uuid.FromStringOrNil("caad42fb-8266-4a24-be3f-9963ba14a20a"),
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
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

			mockSvc.EXPECT().OutdialUpdateCampaignID(&tt.customer, tt.outdialID, tt.requestBody.CampaignID).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String()+"/campaign_id", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDDataPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDDataPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
			request.BodyOutdialsIDDataPUT{
				Data: "test data",
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
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

			mockSvc.EXPECT().OutdialUpdateData(&tt.customer, tt.outdialID, tt.requestBody.Data).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String()+"/data", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDTargetsPOST(t *testing.T) {

	tests := []struct {
		name      string
		customer  cscustomer.Customer
		outdialID uuid.UUID

		requestBody request.BodyOutdialsIDTargetsPOST

		response *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("726d6b88-2028-44fe-a415-a58067d98acf"),
			request.BodyOutdialsIDTargetsPOST{
				Name:   "test name",
				Detail: "test detail",
				Data:   "test data",
				Destination0: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination1: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination2: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000003",
				},
				Destination3: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000004",
				},
				Destination4: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000005",
				},
			},

			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("e3097653-4c68-4915-add3-78b12a4ba151"),
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

			mockSvc.EXPECT().OutdialtargetCreate(&tt.customer, tt.outdialID, tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Data, tt.requestBody.Destination0, tt.requestBody.Destination1, tt.requestBody.Destination2, tt.requestBody.Destination3, tt.requestBody.Destination4).Return(tt.response, nil)
			req, _ := http.NewRequest("POST", "/v1.0/outdials/"+tt.outdialID.String()+"/targets", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_OutdialsIDTargetsIDDELETE(t *testing.T) {

	tests := []struct {
		name            string
		customer        cscustomer.Customer
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		response *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("112950f8-e3d3-4585-b858-125a59f8f51f"),
			uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),

			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),
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

			mockSvc.EXPECT().OutdialtargetDelete(&tt.customer, tt.outdialID, tt.outdialtargetID).Return(tt.response, nil)
			req, _ := http.NewRequest("DELETE", "/v1.0/outdials/"+tt.outdialID.String()+"/targets/"+tt.outdialtargetID.String(), nil)
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

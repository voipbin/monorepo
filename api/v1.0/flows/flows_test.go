package flows

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
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestFlowsPOST(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		requestBody request.BodyFlowsPOST
		resFlow     *fmflow.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			request.BodyFlowsPOST{
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			&fmflow.WebhookMessage{
				ID:     uuid.FromStringOrNil("264b18d4-82fa-11eb-919b-9f55a7f6ace1"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/flows", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().FlowCreate(req.Context(), &tt.customer, tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Actions, true).Return(tt.resFlow, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDGET(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer
		flow     *fmflow.Flow

		expectFlow *fmflow.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:       "test name",
				Detail:     "test detail",
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
						Type: "answer",
					},
				},
			},
			&fmflow.WebhookMessage{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/flows/%s", tt.flow.ID), nil)
			mockSvc.EXPECT().FlowGet(req.Context(), &tt.customer, tt.flow.ID).Return(tt.expectFlow, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		flowID      uuid.UUID
		requestBody request.BodyFlowsIDPUT
		requestFlow *fmflow.Flow
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
			request.BodyFlowsIDPUT{
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			&fmflow.Flow{
				ID:     uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/v1.0/flows/"+tt.flowID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().FlowUpdate(req.Context(), &tt.customer, tt.requestFlow).Return(&fmflow.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDDELETE(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer
		flowID   uuid.UUID
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/flows/%s", tt.flowID), nil)
			mockSvc.EXPECT().FlowDelete(req.Context(), &tt.customer, tt.flowID).Return(&fmflow.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

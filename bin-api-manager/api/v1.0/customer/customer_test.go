package customer

import (
	"bytes"
	"encoding/json"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_customerGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomer *cscustomer.WebhookMessage

		expectedCustomerID uuid.UUID
		expectedRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			reqQuery: "/v1.0/customer",

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
			},

			expectedCustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
			expectedRes:        `{"id":"e25f1af8-c44f-11ef-9d46-bfaf61e659c2","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().CustomerGet(req.Context(), &tt.agent, tt.expectedCustomerID).Return(tt.responseCustomer, nil)

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
		name   string
		agent  amagent.Agent
		target string

		req request.BodyCustomerPUT

		responseCustomer *cscustomer.WebhookMessage

		expectedCustomerID uuid.UUID
		expectedRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4afd144c-c451-11ef-a8d8-6fd67202355e"),
					CustomerID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			target: "/v1.0/customer",

			req: request.BodyCustomerPUT{
				Name:          "new name",
				Detail:        "new detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "somewhere",
				WebhookMethod: cscustomer.WebhookMethodPost,
				WebhookURI:    "test.com",
			},

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
			},

			expectedCustomerID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
			expectedRes:        `{"id":"4b7dcc68-c451-11ef-a289-33cbfe065115","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest(http.MethodPut, tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdate(req.Context(), &tt.agent, tt.expectedCustomerID, tt.req.Name, tt.req.Detail, tt.req.Email, tt.req.PhoneNumber, tt.req.Address, tt.req.WebhookMethod, tt.req.WebhookURI).Return(tt.responseCustomer, nil)

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

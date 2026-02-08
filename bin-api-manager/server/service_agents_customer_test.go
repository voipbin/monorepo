package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsCustomer(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomer *cmcustomer.WebhookMessage
		expectedRes      []byte
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9a653b20-bc88-11ef-be5c-5322f7693f99"),
					CustomerID: uuid.FromStringOrNil("9aae8ff0-bc88-11ef-8111-0fd82660b367"),
				},
			},

			reqQuery: "/service_agents/customer",

			responseCustomer: &cmcustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("9aae8ff0-bc88-11ef-8111-0fd82660b367"),
			},
			expectedRes: []byte(`{"id":"9aae8ff0-bc88-11ef-8111-0fd82660b367","billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentCustomerGet(req.Context(), &tt.agent).Return(tt.responseCustomer, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, w.Body.Bytes())
			}
		})
	}
}

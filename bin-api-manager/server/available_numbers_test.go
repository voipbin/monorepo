package server

import (
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func TestAvailableNumbersGET(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		pageSize    uint64
		countryCode string

		responseAvailableNumbers []*nmavailablenumber.WebhookMessage
		expectedRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f111bf46-8df6-11ee-8b96-df7d1f63d9d2"),
				},
			},
			pageSize:    10,
			countryCode: "US",

			responseAvailableNumbers: []*nmavailablenumber.WebhookMessage{
				{
					Number:   "+16188850188",
					Country:  "US",
					Region:   "IL",
					Features: []nmavailablenumber.Feature{"emergency", "fax", "voice", "sms"},
				},
			},
			expectedRes: `{"result":[{"number":"+16188850188","country":"US","region":"IL","postal_code":"","features":["emergency","fax","voice","sms"]}]}`,
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
				c.Set("agent", tt.agent)
			})
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/available_numbers?page_size=%d&customer_id=%s&country_code=%s", tt.pageSize, tt.agent.CustomerID, tt.countryCode), nil)
			mockSvc.EXPECT().AvailableNumberGets(req.Context(), &tt.agent, tt.pageSize, tt.countryCode).Return(tt.responseAvailableNumbers, nil)

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

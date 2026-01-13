package server

import (
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
		name  string
		agent amagent.Agent

		reqQuery string

		responseAvailableNumbers []*nmavailablenumber.WebhookMessage

		expectPageSize    uint64
		expectCountryCode string
		expectedRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f111bf46-8df6-11ee-8b96-df7d1f63d9d2"),
				},
			},

			reqQuery: "/available_numbers?page_size=10&country_code=US",

			responseAvailableNumbers: []*nmavailablenumber.WebhookMessage{
				{
					Number:   "+16188850188",
					Country:  "US",
					Region:   "IL",
					Features: []nmavailablenumber.Feature{"emergency", "fax", "voice", "sms"},
				},
			},

			expectPageSize:    10,
			expectCountryCode: "US",
			expectedRes:       `{"result":[{"number":"+16188850188","country":"US","region":"IL","postal_code":"","features":["emergency","fax","voice","sms"]}]}`,
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
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AvailableNumberGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectCountryCode.Return(tt.responseAvailableNumbers, nil)

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

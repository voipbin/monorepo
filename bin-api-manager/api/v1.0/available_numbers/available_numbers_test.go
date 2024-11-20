package availablenumbers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestAvailableNumbersGET(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		pageSize    uint64
		countryCode string

		resAvailableNumbers []*nmavailablenumber.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f111bf46-8df6-11ee-8b96-df7d1f63d9d2"),
				},
			},
			10,
			"US",
			[]*nmavailablenumber.WebhookMessage{
				{
					Number:   "+16188850188",
					Country:  "US",
					Region:   "IL",
					Features: []nmavailablenumber.Feature{"emergency", "fax", "voice", "sms"},
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
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/available_numbers?page_size=%d&customer_id=%s&country_code=%s", tt.pageSize, tt.agent.CustomerID, tt.countryCode), nil)
			mockSvc.EXPECT().AvailableNumberGets(req.Context(), &tt.agent, tt.pageSize, tt.countryCode).Return(tt.resAvailableNumbers, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

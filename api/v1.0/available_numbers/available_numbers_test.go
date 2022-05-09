package availablenumbers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestAvailableNumbersGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		customer    cscustomer.Customer
		pageSize    uint64
		countryCode string

		resAvailableNumbers []*nmavailablenumber.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("09e38a62-8003-11ec-8085-7f8bfbbc02de"),
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

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().AvailableNumberGets(&tt.customer, tt.pageSize, tt.countryCode).Return(tt.resAvailableNumbers, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/available_numbers?page_size=%d&customer_id=%s&country_code=%s", tt.pageSize, tt.customer.ID, tt.countryCode), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

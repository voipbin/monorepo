package availablenumbers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
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
		user        models.User
		pageSize    uint64
		countryCode string

		resAvailableNumbers []*models.AvailableNumber
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			10,
			"US",
			[]*models.AvailableNumber{
				{
					Number:   "+16188850188",
					Country:  "US",
					Region:   "IL",
					Features: []string{"emergency", "fax", "voice", "sms"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().AvailableNumberGets(&tt.user, tt.pageSize, tt.countryCode).Return(tt.resAvailableNumbers, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/available_numbers?page_size=%d&user_id=%d&country_code=%s", tt.pageSize, tt.user.ID, tt.countryCode), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

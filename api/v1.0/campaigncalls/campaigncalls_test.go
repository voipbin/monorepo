package campaigncalls

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_campaigncallsIDGET(t *testing.T) {

	tests := []struct {
		name           string
		customer       cscustomer.Customer
		campaigncallID uuid.UUID

		reqQuery string

		response *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),

			"/v1.0/campaigncalls/897e611a-c870-11ec-9b81-a7b70b7cdaa1",

			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().CampaigncallGet(req.Context(), &tt.customer, tt.campaigncallID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaigncallsIDDELETE(t *testing.T) {

	tests := []struct {
		name           string
		customer       cscustomer.Customer
		campaigncallID uuid.UUID

		reqQuery string
		response *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),

			"/v1.0/campaigncalls/afe97cd6-c870-11ec-b750-f3db7eda3a33",
			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().CampaigncallDelete(req.Context(), &tt.customer, tt.campaigncallID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

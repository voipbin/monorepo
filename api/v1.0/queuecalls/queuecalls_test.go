package queuecalls

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_queuecallsIDGet(t *testing.T) {

	type test struct {
		name         string
		customer     cscustomer.Customer
		resQueuecall *qmqueuecall.WebhookMessage
		expectRes    string
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&qmqueuecall.WebhookMessage{
				ID:       uuid.FromStringOrNil("7d54d626-1681-11ed-ab05-473fa9aa2542"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
			`{"id":"7d54d626-1681-11ed-ab05-473fa9aa2542","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","duration_waiting":0,"duration_service":0,"tm_create":"2020-09-20T03:23:21.995000","tm_service":"","tm_update":"","tm_delete":""}`,
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

			reqQuery := fmt.Sprintf("/v1.0/queuecalls/%s", tt.resQueuecall.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().QueuecallGet(req.Context(), &tt.customer, tt.resQueuecall.ID).Return(tt.resQueuecall, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_queuecallsIDDelete(t *testing.T) {

	type test struct {
		name        string
		customer    cscustomer.Customer
		queuecallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("a275df90-1681-11ed-a021-c3f295fc9257"),
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

			reqQuery := fmt.Sprintf("/v1.0/queuecalls/%s", tt.queuecallID)
			req, _ := http.NewRequest("DELETE", reqQuery, nil)

			mockSvc.EXPECT().QueuecallDelete(req.Context(), &tt.customer, tt.queuecallID).Return(&qmqueuecall.WebhookMessage{}, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

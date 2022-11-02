package numbers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestNumbersGET(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer
		uri      string
		req      request.ParamNumbersGET

		resNumbers []*nmnumber.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/numbers?page_size=10&page_token=2021-03-02%2003%3A23%3A20.995000",
			request.ParamNumbersGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2021-03-02 03:23:20.995000",
				},
			},
			[]*nmnumber.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("31ee638c-7b23-11eb-858a-33e73c4f82f7"),
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

			req, _ := http.NewRequest("GET", tt.uri, nil)
			mockSvc.EXPECT().NumberGets(req.Context(), &tt.customer, tt.req.PageSize, tt.req.PageToken).Return(tt.resNumbers, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersIDGET(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer
		numberID uuid.UUID
		uri      string

		resNumber  *nmnumber.WebhookMessage
		expectBody []byte
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
			"/v1.0/numbers/3ab6711c-7be6-11eb-8da6-d31a9f3d45a6",
			&nmnumber.WebhookMessage{
				ID: uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
			},
			[]byte(`{"id":"3ab6711c-7be6-11eb-8da6-d31a9f3d45a6","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			req, _ := http.NewRequest("GET", tt.uri, nil)
			mockSvc.EXPECT().NumberGet(req.Context(), &tt.customer, tt.numberID).Return(tt.resNumber, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			resBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(resBytes, tt.expectBody) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBody, resBytes)
			}
		})
	}
}

func TestNumbersIDDELETE(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		numberID uuid.UUID
		uri      string

		responseNumber *nmnumber.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
			"/v1.0/numbers/d905c26e-7be6-11eb-b92a-ab4802b4bde3",
			&nmnumber.WebhookMessage{
				ID: uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
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

			req, _ := http.NewRequest("DELETE", tt.uri, nil)
			mockSvc.EXPECT().NumberDelete(req.Context(), &tt.customer, tt.numberID).Return(tt.responseNumber, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersPOST(t *testing.T) {

	type test struct {
		name        string
		customer    cscustomer.Customer
		uri         string
		requestBody request.BodyNumbersPOST
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/numbers",
			request.BodyNumbersPOST{
				Number:     "+821021656521",
				CallFlowID: uuid.FromStringOrNil("7762e356-88b1-11ec-bb0c-7f21b7cad172"),
				Name:       "test name",
				Detail:     "test detail",
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
			req, _ := http.NewRequest("POST", tt.uri, bytes.NewBuffer(body))

			mockSvc.EXPECT().NumberCreate(req.Context(), &tt.customer, tt.requestBody.Number, tt.requestBody.CallFlowID, tt.requestBody.MessageFlowID, tt.requestBody.Name, tt.requestBody.Detail)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersIDPUT(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		uri      string

		id          uuid.UUID
		requestBody request.BodyNumbersIDPUT
		resNumber   *nmnumber.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/numbers/4e1a6702-7c60-11eb-bca2-3fd92181c652",

			uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),
			request.BodyNumbersIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&nmnumber.WebhookMessage{
				ID: uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),
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
			req, _ := http.NewRequest("PUT", tt.uri, bytes.NewBuffer(body))

			mockSvc.EXPECT().NumberUpdate(req.Context(), &tt.customer, tt.id, tt.requestBody.Name, tt.requestBody.Detail).Return(tt.resNumber, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersIDFlowIDPUT(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer
		uri      string

		id          uuid.UUID
		requestBody request.BodyNumbersIDFlowIDPUT
		resNumber   *nmnumber.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/numbers/a440c6b8-94cd-11ec-a524-af82f0c3ee68/flow_ids",

			uuid.FromStringOrNil("a440c6b8-94cd-11ec-a524-af82f0c3ee68"),
			request.BodyNumbersIDFlowIDPUT{
				CallFlowID:    uuid.FromStringOrNil("b6161d70-94cd-11ec-b56c-bb1a417ae104"),
				MessageFlowID: uuid.FromStringOrNil("6e7ecc24-a881-11ec-bb4f-4b5822260cbe"),
			},
			&nmnumber.WebhookMessage{
				ID: uuid.FromStringOrNil("a440c6b8-94cd-11ec-a524-af82f0c3ee68"),
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
			req, _ := http.NewRequest("PUT", tt.uri, bytes.NewBuffer(body))

			mockSvc.EXPECT().NumberUpdateFlowIDs(req.Context(), &tt.customer, tt.id, tt.requestBody.CallFlowID, tt.requestBody.MessageFlowID).Return(tt.resNumber, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

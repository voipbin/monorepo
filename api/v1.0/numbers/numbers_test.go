package numbers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestNumbersGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		customer cscustomer.Customer
		uri      string
		req      request.ParamNumbersGET

		resNumbers []*number.Number
	}

	tests := []test{
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
			[]*number.Number{
				{
					ID:               uuid.FromStringOrNil("31ee638c-7b23-11eb-858a-33e73c4f82f7"),
					Number:           "+821021656521",
					CustomerID:       uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					Status:           "active",
					T38Enabled:       false,
					EmergencyEnabled: false,
					TMPurchase:       "",
					TMCreate:         "",
					TMUpdate:         "",
					TMDelete:         "",
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

			mockSvc.EXPECT().NumberGets(&tt.customer, tt.req.PageSize, tt.req.PageToken).Return(tt.resNumbers, nil)
			req, _ := http.NewRequest("GET", tt.uri, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		customer cscustomer.Customer
		numberID uuid.UUID
		uri      string

		resNumber  *number.Number
		expectBody []byte
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
			"/v1.0/numbers/3ab6711c-7be6-11eb-8da6-d31a9f3d45a6",
			&number.Number{
				ID:               uuid.FromStringOrNil("3ab6711c-7be6-11eb-8da6-d31a9f3d45a6"),
				Number:           "+821021656521",
				CustomerID:       uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         "",
			},
			[]byte(`{"id":"3ab6711c-7be6-11eb-8da6-d31a9f3d45a6","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockSvc.EXPECT().NumberGet(&tt.customer, tt.numberID).Return(tt.resNumber, nil)
			req, _ := http.NewRequest("GET", tt.uri, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			resBytes, err := ioutil.ReadAll(w.Body)
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

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		customer cscustomer.Customer
		numberID uuid.UUID
		uri      string

		resNumber *number.Number
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
			"/v1.0/numbers/d905c26e-7be6-11eb-b92a-ab4802b4bde3",
			&number.Number{
				ID:               uuid.FromStringOrNil("d905c26e-7be6-11eb-b92a-ab4802b4bde3"),
				Number:           "+821021656521",
				CustomerID:       uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         "",
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

			mockSvc.EXPECT().NumberDelete(&tt.customer, tt.numberID).Return(tt.resNumber, nil)
			req, _ := http.NewRequest("DELETE", tt.uri, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

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
				Number: "+821021656521",
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

			mockSvc.EXPECT().NumberCreate(&tt.customer, tt.requestBody.Number)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.uri, bytes.NewBuffer(body))

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestNumbersIDPUT(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		customer cscustomer.Customer
		uri      string

		requestBody   request.BodyNumbersIDPUT
		requestNumber *number.Number
		resNumber     *number.Number
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			"/v1.0/numbers/4e1a6702-7c60-11eb-bca2-3fd92181c652",
			request.BodyNumbersIDPUT{
				FlowID: uuid.FromStringOrNil("68e108d4-7c60-11eb-9276-5b2ca6f08cbb"),
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),
				FlowID: uuid.FromStringOrNil("68e108d4-7c60-11eb-9276-5b2ca6f08cbb"),
			},
			&number.Number{
				ID:               uuid.FromStringOrNil("4e1a6702-7c60-11eb-bca2-3fd92181c652"),
				FlowID:           uuid.FromStringOrNil("68e108d4-7c60-11eb-9276-5b2ca6f08cbb"),
				Number:           "+821021656521",
				CustomerID:       uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				Status:           "active",
				T38Enabled:       false,
				EmergencyEnabled: false,
				TMPurchase:       "",
				TMCreate:         "",
				TMUpdate:         "",
				TMDelete:         "",
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

			mockSvc.EXPECT().NumberUpdate(&tt.customer, tt.requestNumber).Return(tt.resNumber, nil)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.uri, bytes.NewBuffer(body))

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

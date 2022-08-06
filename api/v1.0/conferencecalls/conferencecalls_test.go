package conferencecalls

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_conferencecallsPOST(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		conferenceID  uuid.UUID
		referenceType cfconferencecall.ReferenceType
		referenceID   uuid.UUID

		responseConferencecall *cfconferencecall.WebhookMessage
		request                []byte
	}{
		{
			"reference type call",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			uuid.FromStringOrNil("90fa3988-15b3-11ed-918b-e3c155fcc880"),
			cfconferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("a42f781a-15b3-11ed-aa52-2bc25e2dce16"),

			&cfconferencecall.WebhookMessage{
				ID: uuid.FromStringOrNil("b5ca8e2a-15b3-11ed-8aba-831ab1a0e559"),
			},
			[]byte(`{"conference_id": "90fa3988-15b3-11ed-918b-e3c155fcc880", "reference_type": "call", "reference_id": "a42f781a-15b3-11ed-aa52-2bc25e2dce16"}`),
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

			req, _ := http.NewRequest("POST", "/v1.0/conferencecalls", bytes.NewBuffer(tt.request))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConferencecallCreate(req.Context(), &tt.customer, tt.conferenceID, tt.referenceType, tt.referenceID).Return(tt.responseConferencecall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func TestConferencesIDGET(t *testing.T) {

	tests := []struct {
		name string

		customer cscustomer.Customer
		id       uuid.UUID

		requestURI string

		conference *cfconferencecall.WebhookMessage
	}{
		{
			"normal",

			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("c2de6db2-15b2-11ed-a8c9-df3874205c01"),

			"/v1.0/conferencecalls/c2de6db2-15b2-11ed-a8c9-df3874205c01",
			&cfconferencecall.WebhookMessage{
				ID: uuid.FromStringOrNil("c2de6db2-15b2-11ed-a8c9-df3874205c01"),
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

			req, _ := http.NewRequest("GET", tt.requestURI, nil)

			mockSvc.EXPECT().ConferencecallGet(req.Context(), &tt.customer, tt.id).Return(tt.conference, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencecallsIDDELETE(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		id uuid.UUID

		requestURI string

		responseConferencecall *cfconferencecall.WebhookMessage
	}{
		{
			"simple test",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("23d576b4-15b4-11ed-b6f4-fbfaed3df462"),
			"/v1.0/conferencecalls/23d576b4-15b4-11ed-b6f4-fbfaed3df462",

			&cfconferencecall.WebhookMessage{
				ID: uuid.FromStringOrNil("23d576b4-15b4-11ed-b6f4-fbfaed3df462"),
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

			req, _ := http.NewRequest("DELETE", tt.requestURI, nil)

			mockSvc.EXPECT().ConferencecallKick(req.Context(), &tt.customer, tt.id).Return(tt.responseConferencecall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

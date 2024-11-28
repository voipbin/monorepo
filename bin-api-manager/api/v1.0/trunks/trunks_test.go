package trunks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_trunksPOST(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent

		reqQuery string
		reqBody  request.BodyTrunksPOST
	}

	tests := []test{
		{
			name: "normal",
			customer: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/trunks",
			reqBody: request.BodyTrunksPOST{
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes: []rmsipauth.AuthType{
					rmsipauth.AuthTypeBasic,
				},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{"1.2.3.4"},
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TrunkCreate(req.Context(), &tt.customer, tt.reqBody.Name, tt.reqBody.Detail, tt.reqBody.DomainName, tt.reqBody.AuthTypes, tt.reqBody.Username, tt.reqBody.Password, tt.reqBody.AllowedIPs).Return(&rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_trunksGET(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent

		reqQuery string

		size  uint64
		token string

		reqBody request.BodyTrunksPOST
	}

	tests := []test{
		{
			name: "normal",
			customer: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/trunks?page_size=20&page_token=2020-09-20%2003:23:20.995000",
			size:     20,
			token:    "2020-09-20 03:23:20.995000",

			reqBody: request.BodyTrunksPOST{
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes: []rmsipauth.AuthType{
					rmsipauth.AuthTypeBasic,
				},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{"1.2.3.4"},
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("GET", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TrunkGets(req.Context(), &tt.customer, tt.size, tt.token).Return([]*rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_TrunksIDGET(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent

		reqQuery string

		responseTrunk *rmtrunk.WebhookMessage

		expectID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			customer: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/trunks/733b46f6-5588-11ee-b04e-c781770c2c87",

			responseTrunk: &rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("733b46f6-5588-11ee-b04e-c781770c2c87"),
				CustomerID: uuid.FromStringOrNil("d8eff4fa-7ff7-11ec-834f-679286ad908b"),
			},

			expectID: uuid.FromStringOrNil("733b46f6-5588-11ee-b04e-c781770c2c87"),
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().TrunkGet(req.Context(), &tt.customer, tt.expectID).Return(tt.responseTrunk, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_TrunksIDPUT(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent

		reqQuery string
		reqBody  request.BodyTrunksIDPUT

		expectTrunkID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			customer: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/trunks/6019ea72-5589-11ee-8b45-13603ef0a2d4",
			reqBody: request.BodyTrunksIDPUT{
				Name:       "test name",
				Detail:     "test detail",
				AuthTypes:  []rmsipauth.AuthType{rmsipauth.AuthTypeBasic},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{"1.2.3.4"},
			},

			expectTrunkID: uuid.FromStringOrNil("6019ea72-5589-11ee-8b45-13603ef0a2d4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TrunkUpdateBasicInfo(req.Context(), &tt.customer, tt.expectTrunkID, tt.reqBody.Name, tt.reqBody.Detail, tt.reqBody.AuthTypes, tt.reqBody.Username, tt.reqBody.Password, tt.reqBody.AllowedIPs).Return(&rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_trunksIDDELETE(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent

		reqQuery      string
		expectTrunkID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			customer: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery:      "/v1.0/trunks/73ee8bce-558a-11ee-a104-5771d8493db0",
			expectTrunkID: uuid.FromStringOrNil("73ee8bce-558a-11ee-a104-5771d8493db0"),
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().TrunkDelete(req.Context(), &tt.customer, tt.expectTrunkID).Return(&rmtrunk.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

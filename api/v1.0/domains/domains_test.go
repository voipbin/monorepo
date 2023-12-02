package domains

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_DomainsPOST(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		requestBody request.BodyDomainsPOST
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.BodyDomainsPOST{
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test.sip.voipbin.net",
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/domains", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().DomainCreate(req.Context(), &tt.agent, tt.requestBody.DomainName, tt.requestBody.Name, tt.requestBody.Detail).Return(&rmdomain.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_DomainsIDGET(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		domain *rmdomain.Domain

		expectDomain *rmdomain.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("8c769d1e-6edb-11eb-a141-8bb08ceaaa69"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				CustomerID: uuid.FromStringOrNil("d8eff4fa-7ff7-11ec-834f-679286ad908b"),
			},
			&rmdomain.WebhookMessage{
				ID:         uuid.FromStringOrNil("8c769d1e-6edb-11eb-a141-8bb08ceaaa69"),
				DomainName: "test.sip.voipbin.net",
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
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/domains/%s", tt.domain.ID), nil)
			mockSvc.EXPECT().DomainGet(req.Context(), &tt.agent, tt.domain.ID).Return(tt.expectDomain, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_DomainsIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		domainID uuid.UUID
		domainN  string
		detail   string

		requestBody request.BodyDomainsIDPUT
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			uuid.FromStringOrNil("91f5852a-6edb-11eb-86c9-f3e5fc2d3a80"),
			"test name",
			"test detail",

			request.BodyDomainsIDPUT{
				Name:   "test name",
				Detail: "test detail",
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/v1.0/domains/"+tt.domainID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().DomainUpdate(req.Context(), &tt.agent, tt.domainID, tt.domainN, tt.detail).Return(&rmdomain.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_DomainsIDDELETE(t *testing.T) {

	type test struct {
		name     string
		agent    amagent.Agent
		domainID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("5d41b834-6edc-11eb-8d71-f7a08bdfd253"),
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/domains/%s", tt.domainID), nil)
			mockSvc.EXPECT().DomainDelete(req.Context(), &tt.agent, tt.domainID).Return(&rmdomain.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

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

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/rmdomain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestDomainsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        models.User
		requestBody request.BodyDomainsPOST
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID:         1,
				Permission: models.UserPermissionAdmin,
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

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().DomainCreate(&tt.user, tt.requestBody.DomainName, tt.requestBody.Name, tt.requestBody.Detail).Return(&models.Domain{}, nil)
			req, _ := http.NewRequest("POST", "/v1.0/domains", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestDomainsIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name   string
		user   models.User
		domain *rmdomain.Domain

		expectDomain *models.Domain
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("8c769d1e-6edb-11eb-a141-8bb08ceaaa69"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				UserID:     1,
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("8c769d1e-6edb-11eb-a141-8bb08ceaaa69"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				UserID:     1,
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

			mockSvc.EXPECT().DomainGet(&tt.user, tt.domain.ID).Return(tt.expectDomain, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/domains/%s", tt.domain.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestDomainsIDPUT(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name         string
		user         models.User
		domainID     uuid.UUID
		requestBody  request.BodyDomainsIDPUT
		expectDomain *models.Domain
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID:         1,
				Permission: models.UserPermissionAdmin,
			},
			uuid.FromStringOrNil("91f5852a-6edb-11eb-86c9-f3e5fc2d3a80"),
			request.BodyDomainsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&models.Domain{
				ID:     uuid.FromStringOrNil("91f5852a-6edb-11eb-86c9-f3e5fc2d3a80"),
				Name:   "test name",
				Detail: "test detail",
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().DomainUpdate(&tt.user, tt.expectDomain).Return(&models.Domain{}, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/domains/"+tt.domainID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestDomainsIDDELETE(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     models.User
		domainID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			uuid.FromStringOrNil("5d41b834-6edc-11eb-8d71-f7a08bdfd253"),
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

			mockSvc.EXPECT().DomainDelete(&tt.user, tt.domainID).Return(nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/domains/%s", tt.domainID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

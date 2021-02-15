package extensions

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
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/rmextension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	ApplyRoutes(&app.RouterGroup)
}

func TestExtensionsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		requestBody request.BodyExtensionsPOST
		reqExt      *extension.Extension
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			request.BodyExtensionsPOST{
				Name:      "test name",
				Detail:    "test detail",
				DomainID:  uuid.FromStringOrNil("7da5ed2e-6faf-11eb-92bd-bf4592baa4c4"),
				Extension: "test",
				Password:  "password",
			},
			&extension.Extension{
				UserID:    1,
				Name:      "test name",
				Detail:    "test detail",
				DomainID:  uuid.FromStringOrNil("7da5ed2e-6faf-11eb-92bd-bf4592baa4c4"),
				Extension: "test",
				Password:  "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().ExtensionCreate(&tt.user, tt.reqExt).Return(&extension.Extension{}, nil)
			req, _ := http.NewRequest("POST", "/extensions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		DomainID uuid.UUID
		ext      []*rmextension.Extension

		expectExt []*extension.Extension
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("f92c19b2-6fb6-11eb-859c-0378f27fc22f"),
			[]*rmextension.Extension{
				{
					ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
					UserID:    1,
					DomainID:  uuid.FromStringOrNil("f92c19b2-6fb6-11eb-859c-0378f27fc22f"),
					Name:      "test name",
					Detail:    "test detail",
					Extension: "test",
					Password:  "password",
				},
			},
			[]*extension.Extension{
				{
					ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
					UserID:    1,
					DomainID:  uuid.FromStringOrNil("f92c19b2-6fb6-11eb-859c-0378f27fc22f"),
					Name:      "test name",
					Detail:    "test detail",
					Extension: "test",
					Password:  "password",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionGets(&tt.user, tt.DomainID, uint64(10), "").Return(tt.expectExt, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/extensions?domain_id=%s", tt.DomainID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user user.User
		ext  *rmextension.Extension

		expectExt *extension.Extension
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			&rmextension.Extension{
				ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				UserID:    1,
				DomainID:  uuid.FromStringOrNil("2ff2b962-6fb0-11eb-a768-e3780d10e360"),
				Name:      "test name",
				Detail:    "test detail",
				Extension: "test",
				Password:  "password",
			},
			&extension.Extension{
				ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				UserID:    1,
				DomainID:  uuid.FromStringOrNil("2ff2b962-6fb0-11eb-a768-e3780d10e360"),
				Name:      "test name",
				Detail:    "test detail",
				Extension: "test",
				Password:  "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionGet(&tt.user, tt.ext.ID).Return(tt.expectExt, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/extensions/%s", tt.ext.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDPUT(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		extID       uuid.UUID
		requestBody request.BodyExtensionsIDPUT
		expectExt   *extension.Extension
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
			request.BodyExtensionsIDPUT{
				Name:     "test name",
				Detail:   "test detail",
				Password: "update password",
			},
			&extension.Extension{
				ID:       uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
				Name:     "test name",
				Detail:   "test detail",
				Password: "update password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().ExtensionUpdate(&tt.user, tt.expectExt).Return(&extension.Extension{}, nil)
			req, _ := http.NewRequest("PUT", "/extensions/"+tt.extID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDDELETE(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name  string
		user  user.User
		extID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("be0c2b70-6fb0-11eb-849d-3f923b334d3b"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().ExtensionDelete(&tt.user, tt.extID).Return(nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/extensions/%s", tt.extID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

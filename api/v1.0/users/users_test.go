package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestUsersPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		requestBody RequestBodyUsersPOST
	}

	tests := []test{
		{
			"admin permission user",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			RequestBodyUsersPOST{
				Username:   "username-0a790e7a-f15c-11ea-9582-7f1f242cd6f8",
				Password:   "password-0d6e5248-f15c-11ea-b916-379c5bd787a4",
				Permission: uint64(user.PermissionAdmin),
			},
		},
		{
			"none permission user",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			RequestBodyUsersPOST{
				Username:   "username-4880078e-f15f-11ea-afe3-ebf4eb79cf50",
				Password:   "password-4bb9412c-f15f-11ea-b37b-533f30888631",
				Permission: uint64(user.PermissionNone),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().UserCreate(tt.requestBody.Username, tt.requestBody.Password, tt.requestBody.Permission)
			req, _ := http.NewRequest("POST", "/v1.0/users", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestUsersGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user user.User
	}

	tests := []test{
		{
			"admin permission user",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().UserGets().Return(nil, nil)
			req, _ := http.NewRequest("GET", "/v1.0/users", bytes.NewBuffer([]byte("")))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

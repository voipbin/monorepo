package auth

import (
	"bytes"
	"encoding/json"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("")
	ApplyRoutes(v1)
}

func Test_TagsPOST(t *testing.T) {

	type test struct {
		name string

		reqBody request.BodyLoginPOST

		responseToken string
		expectCookie  string
		expectRes     string
	}

	tests := []test{
		{
			name: "normal",

			reqBody: request.BodyLoginPOST{
				Username: "test@test.com",
				Password: "testpassword",
			},

			responseToken: "test_token",
			expectCookie:  "token=test_token",
			expectRes:     `{"username":"test@test.com","token":"test_token"}`,
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
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthLogin(req.Context(), tt.reqBody.Username, tt.reqBody.Password).Return(tt.responseToken, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			cookie := w.Header().Get("Set-Cookie")
			if !strings.Contains(cookie, tt.expectCookie) {
				t.Errorf("Wrong match. expect contains: %s, got: %s", tt.expectCookie, cookie)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

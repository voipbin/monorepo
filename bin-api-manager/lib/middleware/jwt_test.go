package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_getTokenString(t *testing.T) {

	tests := []struct {
		name         string
		setupRequest func(c *gin.Context)
		expectRes    string
	}{
		{
			name: "Token from Cookie",
			setupRequest: func(c *gin.Context) {
				c.Request.AddCookie(&http.Cookie{Name: "token", Value: "cookieToken"})
			},
			expectRes: "cookieToken",
		},
		{
			name: "Token from Query Parameter",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "token=queryToken"
			},
			expectRes: "queryToken",
		},
		{
			name: "Token from Authorization Header",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer headerToken")
			},
			expectRes: "headerToken",
		},
		{
			name:         "No Token Provided",
			setupRequest: func(c *gin.Context) {},
			expectRes:    "",
		},
		{
			name: "Invalid Authorization Header",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "InvalidHeader headerToken")
			},
			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Apply the test setup
			tt.setupRequest(c)

			res := getTokenString(c)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

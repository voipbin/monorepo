package middleware

import (
	"monorepo/bin-api-manager/lib/common"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func Test_GenerateTokenWithData(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}

		responseCurTime string

		expectRes common.JSON
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},

			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: common.JSON{
				"key1":   "val1",
				"key2":   "val2",
				"expire": "2023-11-19 09:29:11.763331118",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			utilHandler = mockUtil

			mockUtil.EXPECT().TimeGetCurTimeAdd(common.TokenExpiration).Return(tt.responseCurTime)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)

			token, err := GenerateTokenWithData(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := ValidateToken(token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

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

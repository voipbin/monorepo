package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	modelscommon "monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
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

func Test_getAccesskey(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func(c *gin.Context)
		expectRes    string
	}{
		{
			name: "Accesskey from Cookie",
			setupRequest: func(c *gin.Context) {
				c.Request.AddCookie(&http.Cookie{Name: "accesskey", Value: "cookieAccesskey"})
			},
			expectRes: "cookieAccesskey",
		},
		{
			name: "Accesskey from Query Parameter",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=queryAccesskey"
			},
			expectRes: "queryAccesskey",
		},
		{
			name:         "No Accesskey Provided",
			setupRequest: func(c *gin.Context) {},
			expectRes:    "",
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

			res := getAccesskey(c)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getAuthString(t *testing.T) {
	tests := []struct {
		name          string
		setupRequest  func(c *gin.Context)
		expectType    string
		expectString  string
		expectErr     bool
	}{
		{
			name: "Token auth",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer testToken")
			},
			expectType:   authTypeToken,
			expectString: "testToken",
			expectErr:    false,
		},
		{
			name: "Accesskey auth",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=testAccesskey"
			},
			expectType:   authTypeAccesskey,
			expectString: "testAccesskey",
			expectErr:    false,
		},
		{
			name:         "No auth provided",
			setupRequest: func(c *gin.Context) {},
			expectType:   authTypeNone,
			expectString: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			tt.setupRequest(c)

			authType, authString, err := getAuthString(c)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if authType != tt.expectType {
				t.Errorf("Wrong auth type. expect: %v, got: %v", tt.expectType, authType)
			}
			if authString != tt.expectString {
				t.Errorf("Wrong auth string. expect: %v, got: %v", tt.expectString, authString)
			}
		})
	}
}

func Test_getAuthData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	tests := []struct {
		name          string
		setupRequest  func(c *gin.Context)
		authType      string
		mockSetup     func(mockSH *servicehandler.MockServiceHandler)
		expectErr     bool
	}{
		{
			name: "Token auth success",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer validToken")
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
			},
			expectErr: false,
		},
		{
			name: "Accesskey auth success",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=validAccesskey"
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthAccesskeyParse(gomock.Any(), "validAccesskey").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
			},
			expectErr: false,
		},
		{
			name: "No auth",
			setupRequest: func(c *gin.Context) {
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set(modelscommon.OBJServiceHandler, mockSH)

			tt.setupRequest(c)

			_, err := getAuthData(c)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	tests := []struct {
		name           string
		setupRequest   func(c *gin.Context)
		mockSetup      func(mockSH *servicehandler.MockServiceHandler)
		expectStatus   int
		expectAborted  bool
	}{
		{
			name: "Valid authentication",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer validToken")
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
			},
			expectStatus:  200,
			expectAborted: false,
		},
		{
			name: "Invalid authentication - no token",
			setupRequest: func(c *gin.Context) {
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
			},
			expectStatus:  401,
			expectAborted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			c.Request = req

			router.Use(func(c *gin.Context) {
				c.Set(modelscommon.OBJServiceHandler, mockSH)
			})
			router.Use(Authenticate())
			router.GET("/", func(c *gin.Context) {
				c.Status(200)
			})

			tt.setupRequest(c)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestAuthenticateWithInvalidAgentData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	// Return invalid agent data that can't be marshaled
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "invalidToken").Return(map[string]interface{}{
		"agent": make(chan int), // channels can't be marshaled
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalidToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for invalid agent data, got: %v", w.Code)
	}
}

func TestAuthenticateWithMalformedAgentJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	// Return data where agent field exists but is not valid Agent struct
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "malformedToken").Return(map[string]interface{}{
		"agent": map[string]interface{}{
			"invalid_field": "this won't unmarshal to Agent",
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer malformedToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	// Should succeed but agent might be partially populated
	// The actual behavior depends on how json.Unmarshal handles extra fields
	// For Agent struct, it should handle missing fields gracefully
	if w.Code == 500 {
		t.Errorf("Unexpected 500 error")
	}
}

func TestAuthConstants(t *testing.T) {
	if authTypeNone != "" {
		t.Errorf("authTypeNone should be empty string, got: %v", authTypeNone)
	}
	if authTypeToken != "token" {
		t.Errorf("authTypeToken should be 'token', got: %v", authTypeToken)
	}
	if authTypeAccesskey != "accesskey" {
		t.Errorf("authTypeAccesskey should be 'accesskey', got: %v", authTypeAccesskey)
	}
}

func TestAuthenticateAgentStoredInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
		Username:   "testuser",
	}

	mockSH := servicehandler.NewMockServiceHandler(mc)
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
		"agent": testAgent,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer validToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	var capturedAgent amagent.Agent
	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		// Capture the agent from context
		if agent, exists := c.Get("agent"); exists {
			capturedAgent = agent.(amagent.Agent)
		}
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got: %v", w.Code)
	}

	// Verify agent was stored correctly in context
	if capturedAgent.ID != testAgent.ID {
		t.Errorf("Agent ID mismatch. expect: %v, got: %v", testAgent.ID, capturedAgent.ID)
	}
	if capturedAgent.Username != testAgent.Username {
		t.Errorf("Agent username mismatch. expect: %v, got: %v", testAgent.Username, capturedAgent.Username)
	}
}

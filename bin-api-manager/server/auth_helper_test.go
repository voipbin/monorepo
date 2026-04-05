package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_getAuthIdentity(t *testing.T) {

	type test struct {
		name      string
		setupCtx  func(c *gin.Context)
		expectNil bool
		expectOK  bool
	}

	tests := []test{
		{
			name: "valid auth identity",
			setupCtx: func(c *gin.Context) {
				c.Set("auth_identity", auth.NewAgentIdentity(&amagent.Agent{}))
			},
			expectNil: false,
			expectOK:  true,
		},
		{
			name:      "missing key",
			setupCtx:  func(c *gin.Context) {},
			expectNil: true,
			expectOK:  false,
		},
		{
			name: "wrong type in context",
			setupCtx: func(c *gin.Context) {
				c.Set("auth_identity", "not-an-auth-identity")
			},
			expectNil: true,
			expectOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			a, ok := getAuthIdentity(c)
			if ok != tt.expectOK {
				t.Errorf("Wrong ok. expect: %v, got: %v", tt.expectOK, ok)
			}
			if tt.expectNil && a != nil {
				t.Errorf("Expected nil, got: %v", a)
			}
			if !tt.expectNil && a == nil {
				t.Errorf("Expected non-nil, got nil")
			}
		})
	}
}

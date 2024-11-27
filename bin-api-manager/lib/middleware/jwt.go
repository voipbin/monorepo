package middleware

import (
	"encoding/json"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	modelscommon "monorepo/bin-api-manager/api/models/common"

	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// JWTMiddleware parses JWT token from cookie and stores data and expires date to the context
// JWT Token can be passed as cooke, or Authorization header
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)

		// get token string
		tokenString := getTokenString(c)
		if tokenString == "" {
			c.Next()
			return
		}

		// validate the token
		tokenData, err := serviceHandler.JWTParse(tokenString)
		if err != nil {
			c.Next()
			return
		}

		// get agent info
		tmpAgent, err := json.Marshal(tokenData["agent"])
		if err != nil {
			logrus.Errorf("Could not marshal the token data. err: %v", err)
			c.Next()
			return
		}
		a := amagent.Agent{}
		if err := json.Unmarshal(tmpAgent, &a); err != nil {
			logrus.Errorf("Could not marshal the customer. err: %v", err)
			c.Next()
			return
		}

		c.Set("agent", a)
		c.Set("token_expire", tokenData["expire"])
		c.Next()
	}
}

// getTokenString returns the token string from the gin context.
func getTokenString(c *gin.Context) string {
	// get token from the cookie
	res, err := c.Cookie("token")
	if err == nil && res != "" {
		return res
	}

	// get token from the url query
	res = c.Query("token")
	if res != "" {
		return res
	}

	// get token from the http header
	// try reading HTTP header
	authorization := c.Request.Header.Get("Authorization")
	if authorization == "" {
		return ""
	}

	sp := strings.Split(authorization, "Bearer ")
	if len(sp) < 2 {
		// invalid
		return ""
	}
	res = sp[1]

	return res
}

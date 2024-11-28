package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	modelscommon "monorepo/bin-api-manager/api/models/common"

	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Authenticate parses JWT token from cookie and stores data and expires date to the context
// JWT Token can be passed as cooke, or Authorization header
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logrus.WithFields(logrus.Fields{
			"func":            "Authenticate",
			"request_address": c.ClientIP,
		})

		tokenData, err := getTokenData(c)
		if err != nil {
			c.AbortWithStatus(401)
			return
		}

		// get agent info
		tmpAgent, err := json.Marshal(tokenData["agent"])
		if err != nil {
			log.Errorf("Could not marshal the token data. err: %v", err)
			c.AbortWithStatus(401)
			return
		}

		a := amagent.Agent{}
		if err := json.Unmarshal(tmpAgent, &a); err != nil {
			log.Errorf("Could not marshal the customer. err: %v", err)

			c.AbortWithStatus(401)
			return
		}

		c.Set("agent", a)
		c.Next()
	}
}

func getTokenData(c *gin.Context) (map[string]interface{}, error) {
	t, ts, err := getToken(c)
	if err != nil {
		return nil, err
	}

	serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)
	switch t {
	case "token":
		return serviceHandler.AuthJWTParse(ts)

	case "accesskey":
		return serviceHandler.AuthAccesskeyParse(c.Request.Context(), ts)

	default:
		return nil, fmt.Errorf("unknown token type: %s", t)
	}
}

func getToken(c *gin.Context) (string, string, error) {
	tokenString := getTokenString(c)
	if tokenString != "" {
		return "token", tokenString, nil
	}

	accesskey := getAccesskey(c)
	if accesskey != "" {
		return "accesskey", accesskey, nil
	}

	return "", "", fmt.Errorf("no token found")
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

// getTokenString returns the token string from the gin context.
func getAccesskey(c *gin.Context) string {
	// get token from the cookie
	res, err := c.Cookie("accesskey")
	if err == nil && res != "" {
		return res
	}

	// get token from the url query
	res = c.Query("accesskey")
	if res != "" {
		return res
	}

	return ""
}

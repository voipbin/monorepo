package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/lib/common"
)

var (
	secretKey   = []byte{}
	utilHandler = utilhandler.NewUtilHandler()
)

// Init inits middlewares
func Init(key string) {
	secretKey = []byte(key)
}

// GenerateTokenWithData generates jwt token with the given data
func GenerateTokenWithData(data map[string]interface{}) (string, error) {
	logrus.Debugf("Generating the token. data: %v", data)

	claims := jwt.MapClaims{}
	for k, v := range data {
		claims[k] = v
	}

	// token is valid for 7 days
	claims["expire"] = utilHandler.TimeGetCurTimeAdd(common.TokenExpiration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)

	return tokenString, err
}

// ValidateToken validates the token and return the parsed data.
func ValidateToken(tokenString string) (common.JSON, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// don't forget to validate the alg is what you expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return secretKey, nil
	})

	if err != nil {
		return common.JSON{}, err
	}

	if !token.Valid {
		return common.JSON{}, errors.New("invalid token")
	}

	tmp := token.Claims.(jwt.MapClaims)

	curTime := utilHandler.TimeGetCurTime()
	if tmp["expire"].(string) < curTime {
		return common.JSON{}, errors.New("token expired")
	}

	return token.Claims.(jwt.MapClaims), nil
}

// JWTMiddleware parses JWT token from cookie and stores data and expires date to the context
// JWT Token can be passed as cooke, or Authorization header
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// get token string
		tokenString := getTokenString(c)
		if tokenString == "" {
			c.Next()
			return
		}

		// validate the token
		tokenData, err := ValidateToken(tokenString)
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

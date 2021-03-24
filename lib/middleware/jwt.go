package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/common"
)

var secretKey []byte

// Init inits middlewares
func Init(key string) {
	secretKey = []byte(key)
}

// GenerateToken generates jwt token
func GenerateToken(data map[string]interface{}) (string, error) {
	// token is valid for 7 days
	date := time.Now().Add(time.Hour * 24 * 7)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": data,
		"exp":  date.Unix(),
	})

	tokenString, err := token.SignedString(secretKey)

	return tokenString, err
}

func validateToken(tokenString string) (common.JSON, error) {
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

	return token.Claims.(jwt.MapClaims), nil
}

// JWTMiddleware parses JWT token from cookie and stores data and expires date to the context
// JWT Token can be passed as cooke, or Authorization header
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get token from the cookie
		tokenString, err := c.Cookie("token")
		if err != nil {

			// get token from the url query
			tokenString = c.Query("token")
			if tokenString == "" {

				// get token from the http header
				// try reading HTTP header
				authorization := c.Request.Header.Get("Authorization")
				if authorization == "" {
					c.Next()
					return
				}

				sp := strings.Split(authorization, "Bearer ")
				if len(sp) < 2 {
					// invalid
					c.Next()
					return
				}
				tokenString = sp[1]
			}
		}

		tokenData, err := validateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		var u user.User
		u.Read(tokenData["user"].(map[string]interface{}))

		c.Set("user", u)
		c.Set("token_expire", tokenData["exp"])
		c.Next()
	}
}

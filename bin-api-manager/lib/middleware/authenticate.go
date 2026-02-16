package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	modelscommon "monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	authTypeNone      = ""
	authTypeToken     = "token"
	authTypeAccesskey = "accesskey"
)

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logrus.WithFields(logrus.Fields{
			"func":            "Authenticate",
			"request_address": c.ClientIP,
		})

		authData, err := getAuthData(c)
		if err != nil {
			c.AbortWithStatus(401)
			return
		}

		// get agent info
		tmpAgent, err := json.Marshal(authData["agent"])
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

		// Check if customer account is frozen
		if isFrozenAccountBlocked(c, &a) {
			return // response already sent by isFrozenAccountBlocked
		}

		c.Next()
	}
}

// isFrozenAccountBlocked checks if a customer's account is frozen and blocks
// non-allowed requests with a 403 DELETION_SCHEDULED response.
// Returns true if the request was blocked, false if it should proceed.
func isFrozenAccountBlocked(c *gin.Context, a *amagent.Agent) bool {
	// Skip check for project super admins (they can always access)
	if a.HasPermission(amagent.PermissionProjectSuperAdmin) {
		return false
	}

	// Allow unregister endpoints for frozen accounts (self-service recovery)
	path := c.Request.URL.Path
	method := c.Request.Method
	if path == "/auth/unregister" && (method == http.MethodDelete || method == http.MethodPost) {
		return false
	}

	// Fetch customer to check frozen status
	serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)
	cu, err := serviceHandler.CustomerGet(c.Request.Context(), a, a.CustomerID)
	if err != nil {
		// If we can't fetch the customer, don't block (fail open)
		return false
	}

	if cu.Status != cscustomer.StatusFrozen {
		return false
	}

	// Account is frozen - return 403 with DELETION_SCHEDULED error
	var deletionEffectiveAt *time.Time
	if cu.TMDeletionScheduled != nil {
		t := cu.TMDeletionScheduled.Add(30 * 24 * time.Hour)
		deletionEffectiveAt = &t
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error":                  "DELETION_SCHEDULED",
		"message":                "Account deletion scheduled",
		"deletion_scheduled_at":  cu.TMDeletionScheduled,
		"deletion_effective_at":  deletionEffectiveAt,
		"recovery_endpoint":      "DELETE /auth/unregister",
	})
	return true
}

func getAuthData(c *gin.Context) (map[string]interface{}, error) {
	authType, authString, err := getAuthString(c)
	if err != nil {
		return nil, err
	}

	serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)
	switch authType {
	case authTypeToken:
		return serviceHandler.AuthJWTParse(c.Request.Context(), authString)

	case authTypeAccesskey:
		return serviceHandler.AuthAccesskeyParse(c.Request.Context(), authString)

	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}
}

func getAuthString(c *gin.Context) (string, string, error) {
	tokenString := getTokenString(c)
	if tokenString != "" {
		return authTypeToken, tokenString, nil
	}

	accesskey := getAccesskey(c)
	if accesskey != "" {
		return authTypeAccesskey, accesskey, nil
	}

	return authTypeNone, "", fmt.Errorf("no auth found")
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

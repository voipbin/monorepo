package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/models/auth"
	modelscommon "monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	authTypeNone      = ""
	authTypeToken     = "token"
	authTypeAccesskey = "accesskey"

	delegateAudience = "voipbin-api"
)

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logrus.WithFields(logrus.Fields{
			"func":            "Authenticate",
			"request_address": c.ClientIP,
		})

		authType, authString, err := getAuthString(c)
		if err != nil {
			abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}

		serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)

		var identity *auth.AuthIdentity

		switch authType {
		case authTypeToken:
			identity, err = authenticateToken(c, log, serviceHandler, authString)
		case authTypeAccesskey:
			identity, err = authenticateAccesskey(c, log, serviceHandler, authString)
		default:
			err = fmt.Errorf("unknown auth type: %s", authType)
		}

		if err != nil {
			log.Infof("Authentication failed. err: %v", err)
			abortUnauthenticated(c, "INVALID_CREDENTIALS", "The provided credentials are invalid.")
			return
		}

		c.Set("auth_identity", identity)

		// For delegate tokens, annotate the request context with the JTI for audit tracing
		if identity.IsDelegate() && identity.DelegateScope != nil {
			c.Set("delegate_jti", identity.DelegateScope.JTI)
		}

		// Check if customer account is frozen
		if isFrozenAccountBlocked(c, identity) {
			return // response already sent by isFrozenAccountBlocked
		}

		c.Next()
	}
}

// authenticateToken handles JWT token authentication and returns an AuthIdentity.
func authenticateToken(c *gin.Context, log *logrus.Entry, sh servicehandler.ServiceHandler, tokenString string) (*auth.AuthIdentity, error) {
	authData, err := sh.AuthJWTParse(c.Request.Context(), tokenString)
	if err != nil {
		log.Infof("Could not parse JWT token. err: %v", err)
		return nil, fmt.Errorf("invalid token")
	}

	return buildJWTIdentity(log, authData)
}

// buildJWTIdentity inspects the "type" field in JWT claims and builds the appropriate AuthIdentity.
func buildJWTIdentity(log *logrus.Entry, authData map[string]interface{}) (*auth.AuthIdentity, error) {
	tokenType, _ := authData["type"].(string)

	switch tokenType {
	case string(auth.TypeDirect):
		raw, ok := authData["direct"]
		if !ok {
			return nil, fmt.Errorf("direct token missing direct scope")
		}

		buf, err := json.Marshal(raw)
		if err != nil {
			log.Errorf("Could not marshal direct scope. err: %v", err)
			return nil, fmt.Errorf("invalid direct scope")
		}

		var scope auth.DirectScope
		if err := json.Unmarshal(buf, &scope); err != nil {
			log.Errorf("Could not unmarshal direct scope. err: %v", err)
			return nil, fmt.Errorf("invalid direct scope")
		}

		return auth.NewDirectIdentity(&scope), nil

	case string(auth.TypeDelegate):
		// aud is enforced only for delegate tokens — TypeAgent/TypeDirect predate aud claim
		if aud, _ := authData["aud"].(string); aud != delegateAudience {
			return nil, fmt.Errorf("delegate token: invalid audience %q", aud)
		}
		customerIDStr, ok := authData["customer_id"].(string)
		if !ok || customerIDStr == "" {
			return nil, fmt.Errorf("delegate token missing customer_id")
		}
		customerID, err := uuid.FromString(customerIDStr)
		if err != nil {
			return nil, fmt.Errorf("delegate token: invalid customer_id: %w", err)
		}
		issuedByStr, _ := authData["sub"].(string)
		issuedBy, _ := uuid.FromString(issuedByStr)
		jti, _ := authData["jti"].(string)

		scope := &auth.DelegateScope{
			CustomerID: customerID,
			IssuedBy:   issuedBy,
			JTI:        jti,
		}
		return auth.NewDelegateIdentity(scope), nil

	case string(auth.TypeAgent), "":
		// "agent" or missing type (backward compat) — treat as agent token
		raw, ok := authData["agent"]
		if !ok {
			return nil, fmt.Errorf("token missing agent data")
		}

		buf, err := json.Marshal(raw)
		if err != nil {
			log.Errorf("Could not marshal agent data. err: %v", err)
			return nil, fmt.Errorf("invalid agent data")
		}

		var a amagent.Agent
		if err := json.Unmarshal(buf, &a); err != nil {
			log.Errorf("Could not unmarshal agent data. err: %v", err)
			return nil, fmt.Errorf("invalid agent data")
		}

		return auth.NewAgentIdentity(&a), nil

	default:
		return nil, fmt.Errorf("unknown token type: %q", tokenType)
	}
}

// authenticateAccesskey handles accesskey authentication and returns an AuthIdentity.
func authenticateAccesskey(c *gin.Context, log *logrus.Entry, sh servicehandler.ServiceHandler, accesskeyToken string) (*auth.AuthIdentity, error) {
	ak, err := sh.AccesskeyRawGetByToken(c.Request.Context(), accesskeyToken)
	if err != nil {
		log.Infof("Could not get accesskey. err: %v", err)
		return nil, fmt.Errorf("invalid accesskey")
	}

	curTime := time.Now().UTC()
	if ak.TMExpire != nil && ak.TMExpire.Before(curTime) {
		return nil, fmt.Errorf("accesskey expired")
	}
	if ak.TMDelete != nil {
		return nil, fmt.Errorf("accesskey deleted")
	}

	return auth.NewAccesskeyIdentity(ak), nil
}

// isFrozenAccountBlocked checks if a customer's account is frozen and blocks
// non-allowed requests with a 403 DELETION_SCHEDULED response.
// Returns true if the request was blocked, false if it should proceed.
func isFrozenAccountBlocked(c *gin.Context, a *auth.AuthIdentity) bool {
	// Direct tokens skip frozen check
	if a.IsDirect() {
		return false
	}

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

	// Account is frozen — return 403 with PERMISSION_DENIED envelope.
	//
	// Self-service recovery clients (admin.voipbin.net, talk.voipbin.net) rely
	// on the deletion schedule and recovery endpoint to render the "account
	// frozen" UX, so these fields are carried in the envelope's details array
	// while keeping the overall error shape consistent with the rest of the
	// API.
	var deletionEffectiveAt *time.Time
	if cu.TMDeletionScheduled != nil {
		t := cu.TMDeletionScheduled.Add(30 * 24 * time.Hour)
		deletionEffectiveAt = &t
	}
	details := []map[string]any{
		{
			"deletion_scheduled_at": cu.TMDeletionScheduled,
			"deletion_effective_at": deletionEffectiveAt,
			"recovery_endpoint":     "DELETE /auth/unregister",
		},
	}

	e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "ACCOUNT_FROZEN", "This account is frozen. Contact support.")
	e.Details = details
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(e.Status),
		apierror.EnvelopeFor(e, RequestIDFromContext(c)),
	)
	return true
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

// getAccesskey returns the accesskey string from the gin context.
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

// abortUnauthenticated writes the standard UNAUTHENTICATED envelope.
// The external envelope omits the internal Domain field — see
// bin-api-manager/lib/apierror.
func abortUnauthenticated(c *gin.Context, reason, message string) {
	e := cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, reason, message)
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(e.Status),
		apierror.EnvelopeFor(e, RequestIDFromContext(c)),
	)
}


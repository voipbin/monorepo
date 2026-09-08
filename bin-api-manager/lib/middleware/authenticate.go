package middleware

import (
	"context"
	"encoding/json"
	"errors"
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
	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
	commonrequesthandler "monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	authTypeNone      = ""
	authTypeToken     = "token"
	authTypeAccesskey = "accesskey"

	delegateAudience = "voipbin-api"

	// ReasonAccountExpired / MessageAccountExpired / RecoveryEndpointAccountExpired
	// are the canonical wire strings for a status='expired' refusal (VOIP-1491).
	// They are exported because POST /auth/login refuses the same accounts
	// (lib/service/auth.go) and the two responses must be byte-identical —
	// a client that branches on the login refusal must be able to reuse the
	// same handling for the v1 gate refusal.
	//
	// The message deliberately does NOT promise that re-signing-up works.
	// For accounts expired after VOIP-1490 the customer row keeps tm_delete
	// NULL, so validateCreate's (deleted=false, email) duplicate check rejects
	// a re-signup; promising it would be a lie to that cohort (design §4-3).
	ReasonAccountExpired  = "ACCOUNT_EXPIRED"
	MessageAccountExpired = "This account has expired because its email address was never verified. " +
		"Request a new verification email, and contact support@voipbin.net if that does not resolve it."
	RecoveryEndpointAccountExpired = "POST /auth/email-verify-resend"

	// ReasonAccountDeleted / MessageAccountDeleted are the canonical wire
	// strings for a status='deleted' refusal. There is no recovery endpoint
	// to advertise, so unlike expired these carry no details payload.
	ReasonAccountDeleted  = "ACCOUNT_DELETED"
	MessageAccountDeleted = "This account has been deleted."
)

// AccountExpiredDetails returns the details payload carried by every
// ACCOUNT_EXPIRED refusal. A fresh slice is built per call so a caller that
// mutates the returned value cannot corrupt later responses.
func AccountExpiredDetails() []map[string]any {
	return []map[string]any{
		{
			"recovery_endpoint": RecoveryEndpointAccountExpired,
		},
	}
}

var (
	// promAccountStatusLookupFailedTotal counts how often the account-status
	// gate could not fetch the customer and therefore failed OPEN (design §3,
	// §4-1-3). The branch is deliberately left fail-open for now: api-manager
	// has no customer cache, so failing closed there would take down all 414
	// v1 routes whenever customer-manager or RabbitMQ wobbles. This counter
	// exists so the fail-open/fail-closed decision can later be made on data
	// rather than on speculation.
	//
	// identity_type is the load-bearing label: whether failing closed is safe
	// depends entirely on WHICH identities take this branch. accesskey callers
	// should rarely appear (AccesskeyRawGetByToken hits the same
	// customer-manager and 401s earlier in a total outage), so a meaningful
	// accesskey count means a partial outage. The threat being measured is an
	// expired-customer agent JWT sliding through during an outage window.
	//
	// Registered with the package-local NewCounterVec + MustRegister style
	// already used by ratelimit.go, not promauto.
	promAccountStatusLookupFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "account_status_lookup_failed_total",
			Help:      "Total number of account-status gate checks that failed to fetch the customer and fell through fail-open, by identity type and error class",
		},
		[]string{"identity_type", "error_class"},
	)
)

func init() {
	prometheus.MustRegister(promAccountStatusLookupFailedTotal)
}

// accountStatusLookupErrorClass buckets a customer-lookup failure for the
// promAccountStatusLookupFailedTotal error_class label.
//
// The precedence is fixed and must not be reordered. circuit_open is checked
// FIRST and kept distinct from other: a circuit breaker is always installed on
// the request path, so in exactly the outage this counter is meant to measure
// the breaker trips and every subsequent call returns ErrCircuitOpen. Folding
// that into "other" would blind the counter during the only window that
// matters. Connection/channel failures are formatted with %v rather than %w
// upstream, so they do not unwrap and legitimately land in "other".
func accountStatusLookupErrorClass(err error) string {
	switch {
	case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
		return "circuit_open"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, commonrequesthandler.ErrNotFound):
		return "not_found"
	default:
		return "other"
	}
}

// Authenticate parses the request's credentials (JWT or accesskey) and
// stores the resulting *auth.AuthIdentity in the gin context under
// "auth_identity". It does NOT enforce account status (frozen/deleted check) —
// callers that need that check must additionally chain EnforceAccountStatus()
// after Authenticate(). This split (VOIP-1302 §4-6) lets route groups place
// other middleware (e.g. CustomerRateLimit) between authentication and the
// account-status RPC check.
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

		c.Next()
	}
}

// EnforceAccountStatus checks whether the authenticated identity's customer
// account is frozen or deleted and blocks the request if so. It requires Authenticate()
// to have run earlier in the chain and populated "auth_identity" in the gin
// context; if that identity is missing, the request is treated as
// unauthenticated (fail closed) rather than silently passing through.
func EnforceAccountStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get("auth_identity")
		if !exists {
			abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}

		identity, ok := v.(*auth.AuthIdentity)
		if !ok || identity == nil {
			abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}

		// Check if customer account is frozen or deleted
		if isBlockedAccountStatus(c, identity) {
			return // response already sent by isBlockedAccountStatus
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

// isBlockedAccountStatus checks if a customer's account is frozen, expired or
// deleted and blocks non-allowed requests with a 403 response.
//
// This is a defense-in-depth gate: the customer_deleted event cascade is
// expected to soft-delete every downstream resource (agents, flows, ...) for
// a deleted customer, but that cascade has been observed to fail silently
// for a subset of customers (VOIP-1395), leaving live agent credentials that
// can still authenticate. Blocking on StatusDeleted here closes the
// authentication bypass regardless of whether the cascade itself is fixed.
//
// Returns true if the request was blocked, false if it should proceed.
func isBlockedAccountStatus(c *gin.Context, a *auth.AuthIdentity) bool {
	// Direct tokens skip this check
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

	// Fetch customer to check account status. Uses CustomerRawSelfGet (self-scoped,
	// no permission gate) rather than CustomerGet (requires ProjectSuperAdmin) —
	// this check must apply to every non-direct identity, not just super admins,
	// who are already exempted above.
	serviceHandler := c.MustGet(modelscommon.OBJServiceHandler).(servicehandler.ServiceHandler)
	cu, err := serviceHandler.CustomerRawSelfGet(c.Request.Context(), a)
	if err != nil {
		// If we can't fetch the customer, don't block (fail open).
		//
		// This is observed but NOT changed (design §3): with no cache in
		// api-manager, failing closed here would take every v1 route down
		// with customer-manager. The log line and counter exist so the
		// fail-open-vs-fail-closed question can later be settled with data.
		errClass := accountStatusLookupErrorClass(err)
		promAccountStatusLookupFailedTotal.WithLabelValues(string(a.Type), errClass).Inc()
		logrus.WithFields(logrus.Fields{
			"func":          "isBlockedAccountStatus",
			"identity_type": string(a.Type),
			"customer_id":   a.CustomerID,
			"error_class":   errClass,
		}).Warnf("Could not get the customer for the account status gate. Failing open. err: %v", err)
		return false
	}

	switch cu.Status {
	case cscustomer.StatusFrozen:
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

	case cscustomer.StatusExpired:
		// Account expired: signup completed but the email address was never
		// verified, so the unverified-account cleanup job moved the customer
		// to status='expired' (VOIP-1490). Before VOIP-1491 this status fell
		// through the default branch and every credential kept working.
		//
		// details carries the recovery endpoint for the same reason the frozen
		// branch carries one: without it a client would have to string-match
		// the message to know where to send the user.
		e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, ReasonAccountExpired, MessageAccountExpired)
		e.Details = AccountExpiredDetails()
		c.AbortWithStatusJSON(
			cerrors.HTTPStatusFor(e.Status),
			apierror.EnvelopeFor(e, RequestIDFromContext(c)),
		)
		return true

	case cscustomer.StatusDeleted:
		// Account is deleted. This should be unreachable in the steady state —
		// a deleted customer's agents/accesskeys should already be soft-deleted
		// by the customer_deleted cascade and fail earlier in Authenticate() —
		// but the cascade is known to miss some customers (VOIP-1395), so this
		// is a deliberate second layer, not redundant defense.
		e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, ReasonAccountDeleted, MessageAccountDeleted)
		c.AbortWithStatusJSON(
			cerrors.HTTPStatusFor(e.Status),
			apierror.EnvelopeFor(e, RequestIDFromContext(c)),
		)
		return true

	default:
		return false
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

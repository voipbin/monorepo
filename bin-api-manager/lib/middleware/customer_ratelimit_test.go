package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/ratelimithandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func testCustomerRateLimitConfig() CustomerRateLimitConfig {
	return CustomerRateLimitConfig{
		CustomerRPS:   16.7,
		CustomerBurst: 33,
		DirectRPS:     50,
		DirectBurst:   100,
		DelegateRPS:   8.3,
		DelegateBurst: 16,
		RedisTimeout:  50 * time.Millisecond,
	}
}

// setIdentity returns gin middleware that seeds the "auth_identity" key,
// mimicking what Authenticate() does further up the chain.
func setIdentity(identity *auth.AuthIdentity) gin.HandlerFunc {
	return func(c *gin.Context) {
		if identity != nil {
			c.Set("auth_identity", identity)
		}
		c.Next()
	}
}

func TestCustomerRateLimit_NoIdentitySkips(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	// No calls expected -- the middleware must not touch Redis when there
	// is no auth identity in context.

	r := gin.New()
	r.Use(CustomerRateLimit(mockLimiter, testCustomerRateLimitConfig()))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no identity present, got %d", w.Code)
	}
}

// TestCustomerRateLimit_NilCustomerIDSkips is the mandatory regression test
// for the nil CustomerID guard (VOIP-1302 §4-4/§4-13).
func TestCustomerRateLimit_NilCustomerIDSkips(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	// No Allow() call expected -- nil CustomerID must skip Redis entirely.

	identity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: uuid.Nil},
	})

	r := gin.New()
	r.Use(setIdentity(identity))
	r.Use(CustomerRateLimit(mockLimiter, testCustomerRateLimitConfig()))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for nil CustomerID (skip+log), got %d", w.Code)
	}
}

// TestCustomerRateLimit_AgentAccesskeyShareBucket is the mandatory
// regression test verifying agent and accesskey identities for the same
// customer share the same Redis key/bucket (VOIP-1302 §4-3).
func TestCustomerRateLimit_AgentAccesskeyShareBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)

	var capturedKeys []string
	mockLimiter.EXPECT().Allow(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, key string, limit ratelimithandler.Limit) (*ratelimithandler.Result, error) {
			capturedKeys = append(capturedKeys, key)
			return &ratelimithandler.Result{Allowed: 1, Remaining: 10, RetryAfter: -1}, nil
		}).Times(2)

	agentIdentity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: customerID},
	})
	accesskeyIdentity := auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{CustomerID: customerID})

	cfg := testCustomerRateLimitConfig()

	rAgent := gin.New()
	rAgent.Use(setIdentity(agentIdentity))
	rAgent.Use(CustomerRateLimit(mockLimiter, cfg))
	rAgent.GET("/", func(c *gin.Context) { c.Status(200) })
	rAgent.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	rAccesskey := gin.New()
	rAccesskey.Use(setIdentity(accesskeyIdentity))
	rAccesskey.Use(CustomerRateLimit(mockLimiter, cfg))
	rAccesskey.GET("/", func(c *gin.Context) { c.Status(200) })
	rAccesskey.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if len(capturedKeys) != 2 {
		t.Fatalf("expected 2 captured keys, got %d: %v", len(capturedKeys), capturedKeys)
	}
	if capturedKeys[0] != capturedKeys[1] {
		t.Errorf("expected agent and accesskey identities to share the same bucket key, got %q vs %q", capturedKeys[0], capturedKeys[1])
	}
	wantKey := fmt.Sprintf("ratelimit:%s:%s", TierV1Customer, customerID.String())
	if capturedKeys[0] != wantKey {
		t.Errorf("key = %q, want %q", capturedKeys[0], wantKey)
	}
}

// TestCustomerRateLimit_DirectDelegateSeparateTiers is the mandatory
// regression test verifying direct and delegate identities use their own,
// independent tiers/keys (VOIP-1302 §4-3).
func TestCustomerRateLimit_DirectDelegateSeparateTiers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)

	var capturedKeys []string
	var capturedLimits []ratelimithandler.Limit
	mockLimiter.EXPECT().Allow(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, key string, limit ratelimithandler.Limit) (*ratelimithandler.Result, error) {
			capturedKeys = append(capturedKeys, key)
			capturedLimits = append(capturedLimits, limit)
			return &ratelimithandler.Result{Allowed: 1, Remaining: 10, RetryAfter: -1}, nil
		}).Times(2)

	directIdentity := auth.NewDirectIdentity(&auth.DirectScope{CustomerID: customerID})
	delegateIdentity := auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: customerID})

	cfg := testCustomerRateLimitConfig()

	rDirect := gin.New()
	rDirect.Use(setIdentity(directIdentity))
	rDirect.Use(CustomerRateLimit(mockLimiter, cfg))
	rDirect.GET("/", func(c *gin.Context) { c.Status(200) })
	rDirect.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	rDelegate := gin.New()
	rDelegate.Use(setIdentity(delegateIdentity))
	rDelegate.Use(CustomerRateLimit(mockLimiter, cfg))
	rDelegate.GET("/", func(c *gin.Context) { c.Status(200) })
	rDelegate.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if len(capturedKeys) != 2 {
		t.Fatalf("expected 2 captured keys, got %d", len(capturedKeys))
	}
	if capturedKeys[0] == capturedKeys[1] {
		t.Errorf("expected direct and delegate to use different keys, both got %q", capturedKeys[0])
	}
	wantDirectKey := fmt.Sprintf("ratelimit:%s:%s", TierV1CustomerDirect, customerID.String())
	wantDelegateKey := fmt.Sprintf("ratelimit:%s:%s", TierV1CustomerDelegate, customerID.String())
	if capturedKeys[0] != wantDirectKey {
		t.Errorf("direct key = %q, want %q", capturedKeys[0], wantDirectKey)
	}
	if capturedKeys[1] != wantDelegateKey {
		t.Errorf("delegate key = %q, want %q", capturedKeys[1], wantDelegateKey)
	}

	if capturedLimits[0].Rate == capturedLimits[1].Rate {
		t.Errorf("expected direct and delegate to use independent rate configs, both got Rate=%d", capturedLimits[0].Rate)
	}
}

// TestCustomerRateLimit_FailOpenOnRedisError verifies a Redis error (e.g.
// timeout) causes the request to proceed rather than being blocked
// (VOIP-1302 §4-7 fail-open policy).
func TestCustomerRateLimit_FailOpenOnRedisError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	mockLimiter.EXPECT().Allow(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("redis: connection refused"))

	identity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: customerID},
	})

	r := gin.New()
	r.Use(setIdentity(identity))
	r.Use(CustomerRateLimit(mockLimiter, testCustomerRateLimitConfig()))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (fail-open) on Redis error, got %d", w.Code)
	}
}

// TestCustomerRateLimit_RejectedEnvelopeAndRetryAfter verifies a rejected
// customer-tier request returns 429 with a Retry-After header and
// details=[{"limited_by": "customer"}] (VOIP-1302 §4-8).
func TestCustomerRateLimit_RejectedEnvelopeAndRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	mockLimiter.EXPECT().Allow(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ratelimithandler.Result{Allowed: 0, Remaining: 0, RetryAfter: 3 * time.Second}, nil)

	identity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: customerID},
	})

	r := gin.New()
	r.Use(RequestID())
	r.Use(setIdentity(identity))
	r.Use(CustomerRateLimit(mockLimiter, testCustomerRateLimitConfig()))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "3" {
		t.Errorf("Retry-After = %q, want %q", got, "3")
	}

	var body struct {
		Error struct {
			Reason  string           `json:"reason"`
			Details []map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, w.Body.String())
	}
	if body.Error.Reason != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("reason = %q, want RATE_LIMIT_EXCEEDED", body.Error.Reason)
	}
	if len(body.Error.Details) != 1 || body.Error.Details[0]["limited_by"] != "customer" {
		t.Errorf("expected details=[{limited_by: customer}], got %+v", body.Error.Details)
	}
}

// TestCustomerRateLimit_DisabledTierPassesThrough verifies the "rps<=0
// disables the tier" convention shared with the IP-based RateLimit
// middleware.
func TestCustomerRateLimit_DisabledTierPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	// No Allow() call expected -- the tier is disabled before Redis is touched.

	identity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: customerID},
	})

	cfg := testCustomerRateLimitConfig()
	cfg.CustomerRPS = 0

	r := gin.New()
	r.Use(setIdentity(identity))
	r.Use(CustomerRateLimit(mockLimiter, cfg))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for disabled tier, got %d", w.Code)
	}
}

// TestCustomerRateLimit_AllowedPassesThrough verifies a permitted request
// proceeds to the handler without any error response.
func TestCustomerRateLimit_AllowedPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLimiter := ratelimithandler.NewMockRateLimiter(mc)
	mockLimiter.EXPECT().Allow(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ratelimithandler.Result{Allowed: 1, Remaining: 5, RetryAfter: -1}, nil)

	identity := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{CustomerID: customerID},
	})

	r := gin.New()
	r.Use(setIdentity(identity))
	r.Use(CustomerRateLimit(mockLimiter, testCustomerRateLimitConfig()))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed request, got %d", w.Code)
	}
}

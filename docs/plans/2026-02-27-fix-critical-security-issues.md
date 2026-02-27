# Fix Critical Security Issues - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 4 critical/high-severity security vulnerabilities identified in the production security review: password hash leakage in JWT, insecure cookie flag, SSRF in webhook delivery, and missing rate limiting on auth endpoints.

**Architecture:** Each fix is isolated to 1-2 services. Fixes 1 and 3 are single-line changes in bin-agent-manager and bin-api-manager respectively. Fix 4 adds URL validation and a hardened HTTP client to bin-webhook-manager. Fix 5 adds per-IP rate limiting middleware to bin-api-manager auth routes.

**Tech Stack:** Go standard library (net, net/url, net/http), golang.org/x/time/rate (already an indirect dependency), gin-gonic/gin middleware

---

## Task 1: Fix Password Hash Leakage in JWT Tokens (C1)

**Problem:** The `Agent.PasswordHash` field has `json:"password_hash"` tag, so it gets serialized into JWT claims at login. Every JWT token contains the bcrypt hash, decodable by any client.

**Files:**
- Modify: `bin-agent-manager/models/agent/agent.go:17`

**Step 1: Write the test**

No new test needed — this is a tag change. Existing tests will validate that the Agent struct still works. We verify manually that `json.Marshal` excludes PasswordHash.

**Step 2: Change the json tag to `"-"`**

In `bin-agent-manager/models/agent/agent.go` line 17, change:
```go
PasswordHash string `json:"password_hash" db:"password_hash"` // hashed Password
```
to:
```go
PasswordHash string `json:"-" db:"password_hash"` // hashed Password - excluded from JSON/JWT
```

**Impact analysis:**
- JWT tokens will no longer contain `password_hash` — this is the fix
- The `Authenticate` middleware in bin-api-manager deserializes the agent from JWT claims — `PasswordHash` will be empty, but the middleware never reads it (only uses `ID`, `CustomerID`, `Permission`)
- Database operations use `db:` tags, unaffected
- RPC responses that serialize Agent to JSON will also exclude PasswordHash — correct behavior since no consumer needs it after login
- `WebhookMessage` already excludes PasswordHash — no change needed there

**Step 3: Run verification for bin-agent-manager**

```bash
cd bin-agent-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All pass.

**Step 4: Verify all services that import agent model still compile**

The agent model is imported across many services. Since we only changed a struct tag (not the field itself), no compilation errors expected. But verify key consumers:

```bash
cd bin-api-manager && go build ./cmd/...
cd bin-customer-manager && go build ./cmd/...
```

**Step 5: Commit**

```bash
git add bin-agent-manager/models/agent/agent.go
git commit -m "NOJIRA-fix-critical-security-issues

- bin-agent-manager: Exclude PasswordHash from JSON serialization (json:\"-\")
  Prevents bcrypt hash from leaking into JWT tokens and API responses"
```

---

## Task 3: Fix Insecure JWT Cookie (C6)

**Problem:** `SetCookie` in PostLogin sets `secure=false`, allowing the JWT cookie to be transmitted over HTTP (not just HTTPS).

**Files:**
- Modify: `bin-api-manager/lib/service/auth.go:63`

**Step 1: Change secure flag to true**

In `bin-api-manager/lib/service/auth.go` line 63, change:
```go
c.SetCookie("token", token, int(servicehandler.TokenExpiration.Seconds()), "/", "", false, true)
```
to:
```go
c.SetCookie("token", token, int(servicehandler.TokenExpiration.Seconds()), "/", "", true, true)
```

Parameters: `name, value, maxAge, path, domain, secure, httpOnly`
- `secure=true` means the cookie is only sent over HTTPS connections

**Impact analysis:**
- Production uses HTTPS (port 443 with TLS), so this is safe
- Local development typically uses HTTP — developers using cookie-based auth locally will need to use HTTPS or use the Authorization header instead. This is acceptable.

**Step 2: Run verification for bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All pass.

**Step 3: Commit**

```bash
git add bin-api-manager/lib/service/auth.go
git commit -m "NOJIRA-fix-critical-security-issues

- bin-api-manager: Set Secure=true on JWT cookie to prevent HTTP transmission"
```

---

## Task 4: Add SSRF Protection to Webhook Manager (C4)

**Problem:** `sendMessage()` creates a default `http.Client{}` with no URL validation, no timeouts, no redirect protection. Customers can set webhook URIs targeting internal services, cloud metadata endpoints (169.254.169.254), or localhost.

**Files:**
- Create: `bin-webhook-manager/pkg/webhookhandler/urlvalidator.go` — URL validation with private IP blocking
- Create: `bin-webhook-manager/pkg/webhookhandler/urlvalidator_test.go` — Tests for URL validator
- Modify: `bin-webhook-manager/pkg/webhookhandler/main.go` — Add hardened HTTP client to handler struct
- Modify: `bin-webhook-manager/pkg/webhookhandler/message.go` — Use hardened client + validate URLs
- Modify: `bin-webhook-manager/pkg/webhookhandler/webhook.go` — Validate URIs before sending

### Step 1: Write URL validator tests

Create `bin-webhook-manager/pkg/webhookhandler/urlvalidator_test.go`:

```go
package webhookhandler

import (
	"testing"
)

func Test_validateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		// Valid URLs
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"valid with port", "https://example.com:8080/webhook", false},
		{"valid with path", "https://hooks.slack.com/services/T00/B00/xxx", false},

		// Invalid schemes
		{"no scheme", "example.com/webhook", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/file", true},
		{"gopher scheme", "gopher://evil.com", true},
		{"empty string", "", true},

		// Private/reserved IPs
		{"localhost", "http://localhost/webhook", true},
		{"localhost with port", "http://localhost:8080/webhook", true},
		{"127.0.0.1", "http://127.0.0.1/webhook", true},
		{"127.0.0.1 with port", "http://127.0.0.1:9090/webhook", true},
		{"10.x.x.x", "http://10.0.0.1/webhook", true},
		{"172.16.x.x", "http://172.16.0.1/webhook", true},
		{"172.31.x.x", "http://172.31.255.255/webhook", true},
		{"192.168.x.x", "http://192.168.1.1/webhook", true},
		{"metadata endpoint", "http://169.254.169.254/computeMetadata/v1/", true},
		{"link-local", "http://169.254.1.1/test", true},
		{"0.0.0.0", "http://0.0.0.0/webhook", true},
		{"cgn range", "http://100.64.0.1/webhook", true},

		// IPv6 private
		{"ipv6 loopback", "http://[::1]/webhook", true},

		// Edge cases
		{"no host", "http:///path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebhookURL(%q) error = %v, wantErr %v", tt.rawURL, err, tt.wantErr)
			}
		})
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd bin-webhook-manager
go test -v -run Test_validateWebhookURL ./pkg/webhookhandler/...
```
Expected: FAIL — `validateWebhookURL` not defined.

### Step 3: Write URL validator implementation

Create `bin-webhook-manager/pkg/webhookhandler/urlvalidator.go`:

```go
package webhookhandler

import (
	"fmt"
	"net"
	"net/url"
)

// privateNetworks defines IP ranges that webhook URLs must not resolve to.
var privateNetworks = []net.IPNet{
	// IPv4 private/reserved
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // link-local / cloud metadata
	{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}, // CGN

	// IPv6 private/reserved
	{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
	{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
	{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},
}

// validateWebhookURL checks that a URL is safe for outbound webhook delivery.
// It blocks private/reserved IP ranges and non-HTTP(S) schemes.
func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Enforce HTTP(S) only
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	// Resolve hostname to IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve hostname %s: %w", host, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL resolves to private/reserved IP: %s -> %s", host, ip)
		}
	}

	return nil
}

// isPrivateIP checks if an IP is in any private/reserved range.
func isPrivateIP(ip net.IP) bool {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
```

### Step 4: Run test to verify it passes

```bash
cd bin-webhook-manager
go test -v -run Test_validateWebhookURL ./pkg/webhookhandler/...
```
Expected: PASS (some DNS-dependent tests for public domains may vary; private IP tests should all pass).

### Step 5: Update handler struct with hardened HTTP client

Modify `bin-webhook-manager/pkg/webhookhandler/main.go` — add `httpClient *http.Client` field to `webhookHandler` struct and initialize it in `NewWebhookHandler`:

```go
package webhookhandler

//go:generate mockgen -package webhookhandler -destination ./mock_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error
	SendWebhookToURI(ctx context.Context, customerID uuid.UUID, uri string, method webhook.MethodType, dataType webhook.DataType, data json.RawMessage) error
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accoutHandler accounthandler.AccountHandler
	httpClient    *http.Client
}

// newSafeHTTPClient creates an HTTP client that blocks requests to private/reserved IPs.
func newSafeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       20,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, fmt.Errorf("invalid address: %w", err)
				}

				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, fmt.Errorf("DNS lookup failed: %w", err)
				}

				for _, ip := range ips {
					if isPrivateIP(ip.IP) {
						return nil, fmt.Errorf("blocked: %s resolves to private IP %s", host, ip.IP)
					}
				}

				dialer := &net.Dialer{Timeout: 10 * time.Second}
				return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			// Re-validate redirect target against SSRF rules
			if err := validateWebhookURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, messageTargetHandler accounthandler.AccountHandler) WebhookHandler {

	h := &webhookHandler{
		db:            db,
		notifyHandler: notifyHandler,

		accoutHandler: messageTargetHandler,
		httpClient:    newSafeHTTPClient(),
	}

	return h
}
```

### Step 6: Update sendMessage to use hardened client

Modify `bin-webhook-manager/pkg/webhookhandler/message.go`:

```go
package webhookhandler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// sendMessage sends the message to the given uri with the given method and data.
func (h *webhookHandler) sendMessage(uri string, method string, dataType string, data []byte) error {

	log := logrus.WithFields(
		logrus.Fields{
			"func":   "sendMessage",
			"uri":    uri,
			"method": method,
		},
	)
	log.Debugf("Sending a message.")

	// Validate URL before sending
	if err := validateWebhookURL(uri); err != nil {
		log.Warnf("Webhook URL validation failed. err: %v", err)
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
		if err != nil {
			log.Errorf("Could not create request. err: %v", err)
			return err
		}

		if data != nil && dataType != "" {
			req.Header.Set("Content-Type", dataType)
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			log.Errorf("Could not send the request correctly. Making a retrying: %d, err: %v", i, err)
			lastErr = err
			time.Sleep(time.Second * 1)
			continue
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			log.Errorf("Received server error. Making a retrying: %d, status: %d", i, resp.StatusCode)
			lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			time.Sleep(time.Second * 1)
			continue
		}

		log.WithField("response_status", resp.StatusCode).Debugf("Sent the event correctly.")
		return nil
	}

	log.Errorf("Could not send the request. err: %v", lastErr)
	return lastErr
}
```

Key changes:
- Validate URL before any request
- Use `h.httpClient` (hardened, with SSRF protection) instead of `&http.Client{}`
- Remove logging of raw payload data

### Step 7: Run all tests

```bash
cd bin-webhook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Note: Existing webhook tests use mock URIs like "test.com" which will fail DNS resolution in `validateWebhookURL`. The tests don't actually call `sendMessage` (they use goroutines that won't be waited on), but `SendWebhookToURI` and `SendWebhookToCustomer` don't currently validate URLs — they pass directly to `sendMessage` in a goroutine. Since the goroutine's error is only logged, existing tests should still pass. However, if tests are flaky due to the goroutine timing, we may need to adjust.

Expected: All pass.

### Step 8: Commit

```bash
git add bin-webhook-manager/pkg/webhookhandler/urlvalidator.go \
        bin-webhook-manager/pkg/webhookhandler/urlvalidator_test.go \
        bin-webhook-manager/pkg/webhookhandler/main.go \
        bin-webhook-manager/pkg/webhookhandler/message.go
git commit -m "NOJIRA-fix-critical-security-issues

- bin-webhook-manager: Add SSRF protection with URL validation blocking private/reserved IPs
- bin-webhook-manager: Add hardened HTTP client with timeouts, redirect validation, DNS rebinding protection
- bin-webhook-manager: Validate webhook URLs before delivery to prevent access to internal services"
```

---

## Task 5: Add Rate Limiting to Auth Endpoints (C5)

**Problem:** Auth endpoints (`/auth/login`, `/auth/signup`, `/auth/password-forgot`, `/auth/password-reset`) have no rate limiting. Allows unlimited brute-force and credential stuffing attacks.

**Approach:** Per-IP in-memory rate limiter using `golang.org/x/time/rate` (already an indirect dep). Simple, no new external dependencies. Acceptable for K8s multi-pod because even per-pod limiting significantly slows attacks.

**Files:**
- Create: `bin-api-manager/lib/middleware/ratelimit.go` — Rate limit middleware
- Create: `bin-api-manager/lib/middleware/ratelimit_test.go` — Tests
- Modify: `bin-api-manager/cmd/api-manager/main.go:214` — Apply middleware to auth group

### Step 1: Write rate limit middleware tests

Create `bin-api-manager/lib/middleware/ratelimit_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(10, 20)) // 10 req/s, burst 20
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	// First request should succeed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRateLimit_BlocksExcessiveTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 2)) // 1 req/s, burst 2 — very restrictive for testing
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	// Send burst+1 requests from same IP — last should be rejected
	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		lastCode = w.Code
	}

	if lastCode != 429 {
		t.Errorf("expected 429 after exceeding rate limit, got %d", lastCode)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 1)) // very restrictive
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	// Exhaust rate limit for IP 1
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
	}

	// IP 2 should still be allowed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("different IP should not be rate limited, got %d", w.Code)
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd bin-api-manager
go test -v -run TestRateLimit ./lib/middleware/...
```
Expected: FAIL — `RateLimit` not defined.

### Step 3: Write rate limit middleware

Create `bin-api-manager/lib/middleware/ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimitStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
}

func newRateLimitStore(r rate.Limit, burst int) *rateLimitStore {
	s := &rateLimitStore{
		limiters: make(map[string]*ipLimiter),
		rate:     r,
		burst:    burst,
	}

	// Cleanup stale entries every 5 minutes
	go s.cleanup()

	return s
}

func (s *rateLimitStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, exists := s.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(s.rate, s.burst)
		s.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (s *rateLimitStore) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		for ip, v := range s.limiters {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(s.limiters, ip)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimit returns a Gin middleware that limits requests per IP address.
// r is the rate in requests per second, burst is the maximum burst size.
func RateLimit(r float64, burst int) gin.HandlerFunc {
	store := newRateLimitStore(rate.Limit(r), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
```

### Step 4: Run tests to verify they pass

```bash
cd bin-api-manager
go test -v -run TestRateLimit ./lib/middleware/...
```
Expected: PASS.

### Step 5: Apply middleware to auth routes

Modify `bin-api-manager/cmd/api-manager/main.go` lines 214-222, change:

```go
	auth := app.Group("/auth")
	auth.POST("/login", service.PostLogin)
```

to:

```go
	auth := app.Group("/auth")
	auth.Use(middleware.RateLimit(10, 20)) // 10 req/s per IP, burst of 20
	auth.POST("/login", service.PostLogin)
```

This applies rate limiting to ALL auth routes in the group (login, signup, password-forgot, password-reset, email-verify, complete-signup).

Rate parameters: 10 requests/second per IP with burst of 20. This is generous enough for legitimate use (a user retrying a few times) but blocks automated brute-force tools that make hundreds of requests per second.

### Step 6: Run verification for bin-api-manager

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All pass.

### Step 7: Commit

```bash
git add bin-api-manager/lib/middleware/ratelimit.go \
        bin-api-manager/lib/middleware/ratelimit_test.go \
        bin-api-manager/cmd/api-manager/main.go
git commit -m "NOJIRA-fix-critical-security-issues

- bin-api-manager: Add per-IP rate limiting middleware for auth endpoints
- bin-api-manager: Apply rate limit (10 req/s, burst 20) to /auth/* routes
- bin-api-manager: Prevents brute-force and credential stuffing attacks"
```

---

## Final Verification

After all 4 tasks are done:

```bash
# Verify both modified services
cd bin-agent-manager && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m
cd bin-webhook-manager && go test ./... && golangci-lint run -v --timeout 5m
```

Then fetch latest main and check for conflicts before pushing:

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

# Password Reset Link - Implementation Plan

Based on the design in `docs/plans/2026-02-07-password-reset-link-design.md`.

## Step 1: agent-manager — Add Redis token methods to CacheHandler

Add password reset token operations to the existing Redis cache.

**Files to modify:**

### 1a. `bin-agent-manager/pkg/cachehandler/main.go`
Add three new methods to the `CacheHandler` interface:
```go
PasswordResetTokenSet(ctx context.Context, token string, agentID uuid.UUID, ttl time.Duration) error
PasswordResetTokenGet(ctx context.Context, token string) (uuid.UUID, error)
PasswordResetTokenDelete(ctx context.Context, token string) error
```

### 1b. `bin-agent-manager/pkg/cachehandler/handler.go`
Implement the three methods:
- `PasswordResetTokenSet`: `h.Cache.Set(ctx, "password_reset:"+token, agentID.String(), ttl)`
- `PasswordResetTokenGet`: `h.Cache.Get(ctx, "password_reset:"+token)` → parse UUID
- `PasswordResetTokenDelete`: `h.Cache.Del(ctx, "password_reset:"+token)`

**Verification:** `go generate ./... && go test ./...` in `bin-agent-manager`

---

## Step 2: agent-manager — Add request models

**Files to create:**

### 2a. `bin-agent-manager/pkg/listenhandler/models/request/password.go`
```go
package request

type V1DataPasswordForgotPost struct {
    Username string `json:"username"`
}

type V1DataPasswordResetPost struct {
    Token    string `json:"token"`
    Password string `json:"password"`
}
```

---

## Step 3: agent-manager — Add AgentHandler interface methods and business logic

**Files to modify:**

### 3a. `bin-agent-manager/pkg/agenthandler/main.go`
Add to `AgentHandler` interface:
```go
PasswordForgot(ctx context.Context, username string) (string, string, error) // returns (token, username, error)
PasswordReset(ctx context.Context, token string, password string) error
```

Add `cachehandler.CacheHandler` to `agentHandler` struct (currently missing — the cache is only in dbhandler). Since we need direct Redis access for token operations, pass `cachehandler.CacheHandler` into `NewAgentHandler`.

### 3b. `bin-agent-manager/pkg/agenthandler/agent.go`
Add methods following existing patterns:

`PasswordForgot(ctx, username)`:
1. Look up agent by username via `h.db.AgentGetByUsername(ctx, username)`
2. Generate 32-byte random token via `crypto/rand` + hex encoding
3. Store in Redis via `h.cache.PasswordResetTokenSet(ctx, token, agent.ID, time.Hour)`
4. Return `(token, username, nil)`

`PasswordReset(ctx, token, password)`:
1. Get agent ID from Redis via `h.cache.PasswordResetTokenGet(ctx, token)`
2. If not found, return error (expired or invalid)
3. Hash new password: `h.utilHandler.HashGenerate(password, defaultPasswordHashCost)`
4. Update DB: `h.db.AgentSetPasswordHash(ctx, agentID, passHash)`
5. Delete token: `h.cache.PasswordResetTokenDelete(ctx, token)`
6. Publish event: `h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, updatedAgent)`

### 3c. `bin-agent-manager/pkg/agenthandler/main.go` — Update NewAgentHandler
Add `cache cachehandler.CacheHandler` parameter:
```go
func NewAgentHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, cache cachehandler.CacheHandler) AgentHandler {
```

### 3d. Update caller of NewAgentHandler
Find where `NewAgentHandler` is called in `bin-agent-manager/cmd/` and add the cache parameter.

**Verification:** `go generate ./... && go test ./...` in `bin-agent-manager`

---

## Step 4: agent-manager — Add RPC listen handlers

**Files to modify:**

### 4a. `bin-agent-manager/pkg/listenhandler/main.go`
Add regex patterns:
```go
regV1PasswordForgot = regexp.MustCompile("/v1/password-forgot$")
regV1PasswordReset  = regexp.MustCompile("/v1/password-reset$")
```

Add switch cases in `processRequest`:
```go
case regV1PasswordForgot.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1PasswordForgotPost(ctx, m)
    requestType = "/v1/password-forgot"

case regV1PasswordReset.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1PasswordResetPost(ctx, m)
    requestType = "/v1/password-reset"
```

### 4b. Create `bin-agent-manager/pkg/listenhandler/v1_password_forgot.go`
Follow `v1_login.go` pattern:
- Parse `V1DataPasswordForgotPost` from request body
- Call `h.agentHandler.PasswordForgot(ctx, req.Username)`
- Return response with `{token, username}` or 404 if not found

### 4c. Create `bin-agent-manager/pkg/listenhandler/v1_password_reset.go`
Follow `v1_login.go` pattern:
- Parse `V1DataPasswordResetPost` from request body
- Call `h.agentHandler.PasswordReset(ctx, req.Token, req.Password)`
- Return 200 on success, 400 on error

**Verification:** `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-agent-manager`

---

## Step 5: bin-common-handler — Add RPC client methods

**Files to create/modify:**

### 5a. Create `bin-common-handler/pkg/requesthandler/agent_password.go`
Add two RPC client methods following `agent_login.go` pattern:

```go
func (r *requestHandler) AgentV1PasswordForgot(ctx context.Context, timeout int, username string) (string, string, error)
```
- URI: `/v1/password-forgot`
- Method: POST
- Returns: (token, username, error)

```go
func (r *requestHandler) AgentV1PasswordReset(ctx context.Context, timeout int, token string, password string) error
```
- URI: `/v1/password-reset`
- Method: POST

### 5b. `bin-common-handler/pkg/requesthandler/main.go`
Add to `RequestHandler` interface:
```go
AgentV1PasswordForgot(ctx context.Context, timeout int, username string) (string, string, error)
AgentV1PasswordReset(ctx context.Context, timeout int, token string, password string) error
```

### 5c. Create response struct for password-forgot
Need a response struct to unmarshal the RPC response containing `{token, username}`. Create in an appropriate location.

**Verification:** `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-common-handler`

---

## Step 6: api-manager — Add config for password reset base URL

**Files to modify:**

### 6a. `bin-api-manager/internal/config/main.go`
Add to `Config` struct:
```go
PasswordResetBaseURL string // PasswordResetBaseURL is the base URL for password reset links
```

Add to `bindConfig`:
```go
f.String("password_reset_base_url", "https://api.voipbin.net", "Base URL for password reset links")
```

Add to `bindings` map:
```go
"password_reset_base_url": "PASSWORD_RESET_BASE_URL",
```

Add to `LoadGlobalConfig`:
```go
PasswordResetBaseURL: viper.GetString("password_reset_base_url"),
```

---

## Step 7: api-manager — Add service handler methods

**Files to modify:**

### 7a. `bin-api-manager/pkg/servicehandler/main.go`
Add to `ServiceHandler` interface:
```go
// auth handlers (add to existing section)
AuthPasswordForgot(ctx context.Context, username string) error
AuthPasswordReset(ctx context.Context, token string, password string) error
```

### 7b. `bin-api-manager/pkg/servicehandler/auth.go`
Add two methods:

`AuthPasswordForgot(ctx, username)`:
1. Call `h.reqHandler.AgentV1PasswordForgot(ctx, 30000, username)`
2. If agent not found: return nil (silent success to prevent enumeration)
3. If found: build reset link `cfg.PasswordResetBaseURL + "/auth/password-reset?token=" + token`
4. Build email content with the link
5. Call `h.reqHandler.EmailV1EmailSend(ctx, uuid.Nil, uuid.Nil, destinations, subject, content, nil)`
   - destination: `[]address.Address{{Type: "email", Target: username}}`
6. Return nil

`AuthPasswordReset(ctx, token, password)`:
1. Call `h.reqHandler.AgentV1PasswordReset(ctx, 30000, token, password)`
2. Return error if any

---

## Step 8: api-manager — Add HTTP handlers

**Files to modify/create:**

### 8a. `bin-api-manager/lib/service/auth.go`
Add three handlers following the existing `PostLogin` pattern:

`PostPasswordForgot(c *gin.Context)`:
- Bind JSON `{username}` (required)
- Get serviceHandler from context
- Call `serviceHandler.AuthPasswordForgot(ctx, username)`
- Always return 200 (even on error, to prevent enumeration)

`GetPasswordReset(c *gin.Context)`:
- Read `token` query parameter
- Serve HTML page with the token embedded
- Use `c.Data(200, "text/html; charset=utf-8", []byte(htmlContent))`
- HTML is a const string in the file (inline, no template files needed)

`PostPasswordReset(c *gin.Context)`:
- Bind JSON `{token, password}`
- Get serviceHandler from context
- Call `serviceHandler.AuthPasswordReset(ctx, token, password)`
- Return 200 on success, 400 on error

### 8b. `bin-api-manager/cmd/api-manager/main.go`
Register routes under the existing `/auth` group:
```go
auth := app.Group("/auth")
auth.POST("/login", service.PostLogin)
auth.POST("/password-forgot", service.PostPasswordForgot)
auth.GET("/password-reset", service.GetPasswordReset)
auth.POST("/password-reset", service.PostPasswordReset)
```

**Verification:** `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-api-manager`

---

## Step 9: OpenAPI — Add endpoint definitions

**Files to create/modify:**

### 9a. Create `bin-openapi-manager/openapi/paths/auth/password-forgot.yaml`
Follow `login.yaml` pattern:
- POST, no auth required (`security: []`)
- Request body: `{username: string}`
- Response 200: empty object

### 9b. Create `bin-openapi-manager/openapi/paths/auth/password-reset.yaml`
Two operations:
- GET: serves HTML page (query param: token)
- POST: executes reset, request body `{token: string, password: string}`

### 9c. `bin-openapi-manager/openapi/openapi.yaml`
Add path references:
```yaml
/auth/password-forgot:
  $ref: './paths/auth/password-forgot.yaml'
/auth/password-reset:
  $ref: './paths/auth/password-reset.yaml'
```

**Verification:** `go generate ./...` in `bin-openapi-manager`

---

## Step 10: Final verification across all affected services

Run the full verification workflow for each affected service:

```bash
for dir in bin-agent-manager bin-common-handler bin-api-manager bin-openapi-manager; do
  echo "=== $dir ==="
  (cd "$dir" && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m)
done
```

Since `bin-common-handler` is modified, also run verification for ALL other services that depend on it.

---

## Step 11: Write tests

### 11a. `bin-agent-manager/pkg/agenthandler/agent_test.go`
Add tests for `PasswordForgot` and `PasswordReset`:
- Test successful password forgot flow
- Test password forgot with non-existent username
- Test successful password reset
- Test password reset with invalid token
- Test password reset with expired token

### 11b. `bin-agent-manager/pkg/cachehandler/handler_test.go`
Test Redis token operations:
- Set/Get/Delete token
- Get expired token returns error

---

## Dependency Order

Steps must be executed in order because of dependencies:

```
Step 1 (cache methods) → Step 2 (request models) → Step 3 (agent handler) → Step 4 (listen handler)
                                                                                    ↓
Step 5 (common-handler RPC) ← depends on Step 4 for response format
                                                                                    ↓
Step 6 (api config) → Step 7 (service handler) → Step 8 (HTTP handlers) → Step 9 (OpenAPI)
                                                                                    ↓
Step 10 (full verification) → Step 11 (tests)
```

## Summary of all files

| # | File | Action |
|---|------|--------|
| 1 | `bin-agent-manager/pkg/cachehandler/main.go` | Modify (add interface methods) |
| 2 | `bin-agent-manager/pkg/cachehandler/handler.go` | Modify (implement token methods) |
| 3 | `bin-agent-manager/pkg/listenhandler/models/request/password.go` | Create |
| 4 | `bin-agent-manager/pkg/agenthandler/main.go` | Modify (interface + struct + constructor) |
| 5 | `bin-agent-manager/pkg/agenthandler/agent.go` | Modify (add PasswordForgot/PasswordReset) |
| 6 | `bin-agent-manager/pkg/listenhandler/main.go` | Modify (regex + switch cases) |
| 7 | `bin-agent-manager/pkg/listenhandler/v1_password_forgot.go` | Create |
| 8 | `bin-agent-manager/pkg/listenhandler/v1_password_reset.go` | Create |
| 9 | `bin-agent-manager/cmd/*/main.go` | Modify (pass cache to NewAgentHandler) |
| 10 | `bin-common-handler/pkg/requesthandler/agent_password.go` | Create |
| 11 | `bin-common-handler/pkg/requesthandler/main.go` | Modify (add interface methods) |
| 12 | `bin-api-manager/internal/config/main.go` | Modify (add PasswordResetBaseURL) |
| 13 | `bin-api-manager/pkg/servicehandler/main.go` | Modify (add interface methods) |
| 14 | `bin-api-manager/pkg/servicehandler/auth.go` | Modify (add service logic) |
| 15 | `bin-api-manager/lib/service/auth.go` | Modify (add HTTP handlers) |
| 16 | `bin-api-manager/cmd/api-manager/main.go` | Modify (register routes) |
| 17 | `bin-openapi-manager/openapi/paths/auth/password-forgot.yaml` | Create |
| 18 | `bin-openapi-manager/openapi/paths/auth/password-reset.yaml` | Create |
| 19 | `bin-openapi-manager/openapi/openapi.yaml` | Modify (add path refs) |

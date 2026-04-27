# Security

### 15.1 XSS Prevention

Never inject user input into HTML via `fmt.Sprintf`:

```go
// WRONG — XSS vulnerability
html := fmt.Sprintf("<h1>Welcome %s</h1>", userInput)

// CORRECT — validate input format strictly first
if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(token) {
    return fmt.Errorf("invalid token format")
}
// Only use validated input in templates
```

### 15.2 Token Generation

Use `crypto/rand` for all token generation:

```go
// CORRECT
import "crypto/rand"

b := make([]byte, 32)
rand.Read(b)
token := hex.EncodeToString(b)  // 64 hex chars

// WRONG — predictable tokens
import "math/rand"
token := fmt.Sprintf("%d", rand.Int63())
```

### 15.3 Username Enumeration Prevention

Password-forgot endpoints always return 200 regardless of user existence:

```go
// CORRECT
func (h *serviceHandler) AuthPasswordForgot(ctx context.Context, email string) error {
    // Always return nil — don't leak whether user exists
    return nil
}
```

### 15.4 Guest Agent Protection

Check for the guest agent UUID in all mutation operations:

```go
// CORRECT — check before mutation
const guestAgentID = "d819c626-0284-4df8-99d6-d03e1c6fba88"

func (h *agentHandler) Delete(ctx context.Context, id uuid.UUID) error {
    if id.String() == guestAgentID {
        return errors.New("cannot delete guest agent")
    }
    // ...
}
```

### 15.5 Validation at System Boundaries

Validate at service entry points (API layer, RPC handlers). Trust internal code:

```go
// CORRECT — validate at boundary
func (h *listenHandler) processV1AgentsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    var req request.V1DataAgentsPost
    if err := json.Unmarshal(m.Data, &req); err != nil {
        return simpleResponse(400), nil  // Validate input here
    }
    // Internal handler trusts the parsed input
    res, err := h.agentHandler.Create(ctx, req.CustomerID, ...)
}
```

### 15.6 No Secrets in Code

Never commit secrets, API keys, or credentials:

```go
// WRONG
const apiKey = "sk-1234567890abcdef"

// CORRECT — use environment variables
apiKey := viper.GetString("api_key")
```

### 15.7 Operational Security Notes

Platform-level security expectations across the monorepo:

1. **JWT authentication** — `bin-api-manager` validates all external requests; internal services trust requests arriving via RabbitMQ RPC after that gateway check.
2. **No secrets in source** — secrets come from environment variables or CLI flags via Cobra/Viper, never hardcoded constants.
3. **Base64 for certificates** — SSL certificates passed as base64 strings in service configuration (avoids file-mount complexity in k8s).
4. **Validate at boundaries** — see §15.5; trust internal callers, validate everything at the API/RPC entry layer.

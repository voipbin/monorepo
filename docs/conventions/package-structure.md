# Package Structure & File Organization

### 1.1 Standard Service Layout

Every `bin-*` service follows this directory structure:

```
bin-<service-name>/
  cmd/
    <service-name>/main.go              # Daemon entry point (Cobra command)
    <service-name>-control/main.go      # CLI tool (direct DB/cache, no RabbitMQ)
  internal/
    config/main.go                      # Viper/Cobra config, sync.Once singleton
  models/
    <entity>/
      <entity>.go                       # Core struct with db: and json: tags
      field.go                          # Field type string constants for map keys
      event.go                          # EventType string constants
      webhook.go                        # WebhookMessage + ConvertWebhookMessage()
  pkg/
    dbhandler/
      main.go                           # DBHandler interface + handler struct + ErrNotFound
      <entity>.go                       # Squirrel SQL operations per entity
      mock_main.go                      # Generated mock (via go:generate)
    cachehandler/
      main.go                           # CacheHandler interface + Redis implementation
    <domain>handler/
      main.go                           # Handler interface + struct + constructor
      <feature>.go                      # Business logic grouped by feature
      db.go                             # Private DB-layer wrappers (dbGet, dbCreate, etc.)
      event.go                          # Event handlers (EventXxx methods)
      mock_main.go                      # Generated mock
    listenhandler/
      main.go                           # Regex routing + prometheus + Run()
      v1_<resource>.go                  # Per-resource RPC request handlers
      models/request/                   # Request body structs
    subscribehandler/
      main.go                           # Event subscription + routing
  CLAUDE.md                             # Service-specific conventions
```

**Rationale:** Uniform layout across 30+ services makes navigation predictable and enables cross-service tooling.

### 1.2 Two Binaries Per Service

Every service provides two binaries:
- **Daemon** (`cmd/<service-name>/main.go`) — Long-running process consuming RabbitMQ RPC messages
- **Control CLI** (`cmd/<service-name>-control/main.go`) — Admin tool that bypasses RabbitMQ and accesses DB/cache directly

```go
// CORRECT — daemon binary
// bin-agent-manager/cmd/agent-manager/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:  "agent-manager",
        RunE: runCommand,
    }
    // ...
}

// CORRECT — control CLI binary
// bin-agent-manager/cmd/agent-control/main.go
func main() {
    rootCmd := &cobra.Command{
        Use: "agent-control",
    }
    // subcommands for direct DB/cache operations
}
```

### 1.3 Model File Organization

Each entity in `models/<entity>/` has companion files:

| File | Purpose | Example |
|------|---------|---------|
| `<entity>.go` | Core struct with `db:` and `json:` tags | `models/agent/agent.go` |
| `field.go` | `Field` type + constants for type-safe update maps | `models/agent/field.go` |
| `event.go` | `EventType` constants for event publishing | `models/agent/event.go` |
| `webhook.go` | `WebhookMessage` struct + `ConvertWebhookMessage()` | `models/agent/webhook.go` |

```go
// CORRECT — all four files present for agent entity
models/agent/
  agent.go      // type Agent struct { ... }
  field.go      // type Field string; const FieldID Field = "id"
  event.go      // const EventTypeAgentCreated = "agent_created"
  webhook.go    // type WebhookMessage struct { ... }
```

**Wrong:**
```go
// WRONG — putting everything in one file
models/agent/agent.go  // contains Agent struct + Field type + events + webhook
```

### 1.4 Where Code Lives

| Code Type | Location | Example |
|-----------|----------|---------|
| Business logic | `pkg/<domain>handler/` | `pkg/agenthandler/agent.go` |
| Database operations | `pkg/dbhandler/` | `pkg/dbhandler/agent.go` |
| Cache operations | `pkg/cachehandler/` | `pkg/cachehandler/main.go` |
| RPC routing | `pkg/listenhandler/` | `pkg/listenhandler/v1_agents.go` |
| Event subscriptions | `pkg/subscribehandler/` | `pkg/subscribehandler/main.go` |
| Model definitions | `models/<entity>/` | `models/agent/agent.go` |
| Config | `internal/config/` | `internal/config/main.go` |
| Service entrypoint | `cmd/<service>/` | `cmd/agent-manager/main.go` |

**Wrong — business logic in dbhandler:**
```go
// WRONG — dbhandler should only do DB operations, not business logic
func (h *handler) AgentCreate(ctx context.Context, a *agent.Agent) error {
    // validation logic here  ← WRONG, belongs in agenthandler
    if a.Name == "" { return errors.New("name required") }
    // ...
}
```

---

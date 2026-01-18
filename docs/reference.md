# Reference

> **Quick Reference:** For reference overview, see [CLAUDE.md](../CLAUDE.md#reference)

## API Design Principles

### Atomic API Responses

**CRITICAL: All API endpoints MUST return atomic data - single resource types without combining data from other services.**

#### The Rule

API responses should contain ONLY the requested resource type, without including related data from other services or resources. Clients must make separate requests if they need related information.

#### Why

- Maintains clear service boundaries
- Keeps APIs simple and predictable
- Prevents tight coupling between services
- Makes caching and performance optimization easier
- Reduces breaking changes when related resources evolve

#### Examples

✅ **CORRECT - Atomic Response:**
```
GET /v1/billings/{billing-id}
Returns: BillingManagerBilling (just the billing record)

{
  "id": "550e8400-...",
  "account_id": "7b94f82f-...",
  "reference_id": "8c95f93g-...",
  "cost_total": 1.40,
  ...
}
```

❌ **WRONG - Combined Response:**
```
GET /v1/billings/{billing-id}
Returns: Billing + Account + Reference Resource

{
  "billing": { ... },
  "account": { "name": "...", "balance": ... },  // Don't include
  "call": { "duration": ..., "caller_id": ... }   // Don't include
}
```

#### Exceptions to Atomic Response Rule

1. **Pagination Metadata** - List responses can include `next_page_token` as it's directly related to the query:
   ```json
   {
     "result": [...],
     "next_page_token": "2024-01-15T10:30:00"
   }
   ```

2. **Atomic Operation Responses** - When a single operation creates multiple related resources, the response can include all created resources:
   ```
   POST /v1/calls (with groupcall option)
   Returns: { "call": {...}, "groupcall": {...} }

   Reason: Call and groupcall are created atomically in one transaction,
   so returning both is appropriate.
   ```

#### How to Fetch Related Data

For all other cases, clients should make separate requests:
```
1. GET /v1/billings/{billing-id} → Get billing record
2. GET /v1/billing_accounts/{account-id} → Get account details (if needed)
3. GET /v1/calls/{call-id} → Get call details (if needed)
```

**Note:** For authentication and authorization patterns, see `bin-api-manager/CLAUDE.md`.

## Key Dependencies

### All Services

- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/go-redis/redis/v8` - Redis client
- `github.com/rabbitmq/amqp091-go` - RabbitMQ client
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/prometheus/client_golang` - Prometheus metrics
- `go.uber.org/mock` - Mock generation for testing

### Common Tools

- `github.com/Masterminds/squirrel` - SQL query builder
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/gofrs/uuid` - UUID generation

### API Gateway Specific

- `github.com/gin-gonic/gin` - HTTP router
- `github.com/swaggo/swag` - Swagger documentation
- `github.com/oapi-codegen/oapi-codegen` - OpenAPI code generation
- `github.com/golang-jwt/jwt` - JWT authentication
- `github.com/pebbe/zmq4` - ZeroMQ bindings

### Cloud Integration

- `cloud.google.com/go/storage` - GCP Cloud Storage

## Deployment

### Kubernetes

- Each service has `k8s/` directory with manifests
- Prometheus metrics exposed on configured port (default `:2112` on `/metrics`)
- Dockerfiles for containerization

### Infrastructure Requirements

- GCP GKE cluster (recommended)
- MySQL database
- Redis cluster
- RabbitMQ cluster
- Asterisk/RTPEngine for media (external to this repo)
- Public domain with TLS

## Important Notes

### Monorepo-Specific Practices

1. **Always use replace directives** - All `monorepo/bin-*` imports use local paths in `go.mod`
2. **Coordinate breaking changes** - Changes to shared packages affect multiple services
3. **Test holistically** - Inter-service changes require testing communication flow
4. **Update go.mod carefully** - Adding dependencies may affect all services

### Communication Patterns

1. **Never use HTTP between services** - Always use RabbitMQ RPC
2. **Use typed request methods** - Don't construct `sock.Request` manually, use `requesthandler`
3. **Handle async responses** - RabbitMQ RPC is asynchronous
4. **Publish events for notifications** - Use `notifyhandler.PublishEvent()` for pub/sub

### Code Quality

1. **Generate mocks** - Run `go generate ./...` after interface changes
2. **Write table-driven tests** - Follow existing test patterns
3. **Use structured logging** - Follow the function-scoped log pattern (see "Logging Standards" section below)
4. **Handle errors properly** - Wrap errors with `github.com/pkg/errors`
5. **Follow Go naming conventions** - See "Go Naming Conventions" section below

#### Logging Standards

**CRITICAL: All services in the monorepo MUST follow this logging pattern for consistency.**

**The Pattern:**
1. Create a function-scoped log variable at the beginning of each function
2. Include the function name and meaningful input arguments in the initial fields
3. Use the function-scoped log variable for all logging within that function
4. Augment the log with result fields using `log = log.WithField()` or `log = log.WithFields()` before the final log statement
5. Write appropriate log statements at key points (Debug for routine operations, Info for significant events, Error for failures)

**Example (from bin-flow-manager):**
```go
func (h *activeflowHandler) ExecuteContinue(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) error {
    log := logrus.WithFields(logrus.Fields{
        "func":              "ExecuteContinue",
        "activeflow_id":     activeflowID,
        "current_action_id": caID,
    })
    log.Debug("Executing continue")

    // ... business logic ...

    af, err := h.Get(ctx, activeflowID)
    if err != nil {
        log.Errorf("Could not get activeflow info: %v", err)
        return errors.Wrapf(err, "could not get activeflow info")
    }

    // ... more logic ...

    tmp, err := h.ExecuteNextAction(ctx, activeflowID, caID)
    if err != nil {
        return errors.Wrapf(err, "could not execute the next action")
    }

    // Augment log with result before final log
    log = log.WithField("action", tmp)
    log.Debug("Completed the activeflow execution")

    return nil
}
```

**Example (from bin-talk-manager):**
```go
func (h *participantHandler) ParticipantAdd(ctx context.Context, customerID, chatID, ownerID uuid.UUID, ownerType string) (*participant.Participant, error) {
    log := log.WithFields(log.Fields{
        "func":        "ParticipantAdd",
        "customer_id": customerID,
        "chat_id":     chatID,
        "owner_id":    ownerID,
        "owner_type":  ownerType,
    })
    log.Debug("Adding participant")

    // ... validation and business logic ...

    err := h.dbHandler.ParticipantCreate(ctx, p)
    if err != nil {
        log.Errorf("Failed to create participant. err: %v", err)
        return nil, fmt.Errorf("failed to create participant: %w", err)
    }

    // Augment log with result before final log
    log = log.WithField("participant_id", participantID)
    log.Info("Participant added successfully")

    h.notifyHandler.PublishWebhookEvent(ctx, customerID, participant.EventParticipantAdded, p)

    return p, nil
}
```

**Key Points:**

1. **Function-scoped variable**: `log := log.WithFields(...)` or `log := logrus.WithFields(...)`
   - Creates a logger with function context that can be augmented throughout the function
   - Always include `"func": "FunctionName"` as the first field

2. **Initial fields**: Include meaningful input parameters
   - UUIDs: customer_id, chat_id, owner_id, etc.
   - Important string parameters: owner_type, type, etc.
   - Don't include every parameter - only meaningful ones for debugging

3. **Augmenting log**: Use `log = log.WithField(key, value)` to add result fields
   - Add before final success log statement
   - Commonly added: generated IDs, counts, status changes
   - Example: `log = log.WithField("participant_id", participantID)`

4. **Log levels**:
   - `log.Debug()` - Routine operations, entry/exit points
   - `log.Info()` - Significant events (creation, updates, deletions)
   - `log.Errorf()` - Error conditions with context

**Import Pattern**

**CRITICAL: Always use direct import without alias for clarity:**

```go
import (
    "github.com/sirupsen/logrus"
)

func SomeFunction() {
    log := logrus.WithFields(logrus.Fields{...})  // ✅ Clear that we're using logrus

    // Later in the function, use the function-scoped log variable:
    log.WithFields(logrus.Fields{
        "key": "value",
    }).Debug("message")

    // For direct calls without function-scoped variable:
    logrus.Debugf("Direct message: %v", value)  // ✅ Explicit logrus usage
}
```

**Why no alias:**
- Makes code immediately clear that `logrus` is being used (not stdlib `log`)
- Avoids confusion when reading code
- Prevents variable shadowing issues
- Consistent with rest of monorepo

**Benefits of This Pattern:**

- **Consistent context**: All log statements within a function automatically include function name and input context
- **Augmentable**: Can add result fields without repeating initial context
- **Traceable**: Easy to trace execution flow with function names and IDs
- **Maintainable**: Changing initial context only requires updating one line
- **Debuggable**: Critical information (IDs, types, states) always logged

**This pattern is mandatory for ALL new code and should be applied when refactoring existing code.**

#### Go Naming Conventions

**CRITICAL: Use `List` not `Gets` for collection retrieval methods.**

Following Go standard library conventions (e.g., `os.ReadDir`, `database/sql.Query`), methods that return collections should use `List` naming:

```go
// ✅ CORRECT - Use List for collection retrieval
func (h *handler) CallList(ctx context.Context, filters map[Field]any) ([]*Call, error)
func (h *handler) CallListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*Call, error)

// ❌ WRONG - Don't use Gets
func (h *handler) CallGets(ctx context.Context, filters map[Field]any) ([]*Call, error)
func (h *handler) CallGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*Call, error)
```

**Naming patterns:**
- **Single item retrieval:** `Get` (e.g., `CallGet(ctx, id)`)
- **Collection retrieval:** `List` (e.g., `CallList(ctx, filters)`)
- **Filtered collections:** `ListBy*` (e.g., `CallListByCustomerID(ctx, customerID)`)

**Test function names:**
- `Test_Get` - Tests single item retrieval
- `Test_List` - Tests collection retrieval
- `Test_ListByCustomerID` - Tests filtered collection retrieval

**Function comments:**
```go
// ✅ CORRECT
// List returns list of calls with filters
func (h *handler) CallList(ctx context.Context, filters map[Field]any) ([]*Call, error)

// ListByCustomerID returns list of calls by customer ID
func (h *handler) CallListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*Call, error)

// ❌ WRONG - Don't use Gets in comments
// Gets returns list of calls
func (h *handler) CallList(ctx context.Context, filters map[Field]any) ([]*Call, error)
```

**Why this matters:**
- Consistency with Go standard library conventions
- Makes code more idiomatic and easier to understand
- Aligns with community best practices (effective Go, code review comments)
- `Gets` is not a standard Go verb and sounds awkward

### Common Gotchas

#### UUID Fields and DB Tags

**Note:** This affects all services using the `commondatabasehandler` pattern. Critical for database queries to work correctly across the monorepo.

**CRITICAL: UUID fields MUST use the `,uuid` db tag for proper type conversion.**

When adding `db:` struct tags to model fields, UUID fields require special handling:

```go
// ✅ CORRECT - UUID field with uuid tag
type Model struct {
    ID         uuid.UUID `db:"id,uuid"`
    CustomerID uuid.UUID `db:"customer_id,uuid"`
    Name       string    `db:"name"`
}

// ❌ WRONG - Missing uuid tag
type Model struct {
    ID         uuid.UUID `db:"id"`           // Will cause string-to-UUID conversion issues
    CustomerID uuid.UUID `db:"customer_id"`  // Will cause filter parsing errors
}
```

**Why this matters:**

1. **Database queries fail silently** - Filters with UUID fields without `,uuid` tags are passed as strings instead of binary values, causing no database matches
2. **Type conversion errors** - `commondatabasehandler.PrepareFields()` needs the `,uuid` tag to convert `uuid.UUID` → binary for MySQL
3. **API bugs** - List endpoints return empty results even when data exists

**Example bug:**
```go
// Bug: conversation model missing uuid tags
type Conversation struct {
    CustomerID uuid.UUID `db:"customer_id"`  // Missing ,uuid tag
}

// Result: GET /v1/conversations?customer_id=<uuid> returns []
// Because filter is passed as string, not binary
```

**How to fix:**
1. Add `,uuid` tag to ALL uuid.UUID fields in model structs
2. Regenerate mocks: `go generate ./...`
3. Update tests: If tests mock database queries, verify UUID values are `uuid.UUID` type, not strings
4. Run verification workflow: `go mod tidy && go mod vendor && go generate ./... && go clean -testcache && go test ./...`

**Always verify UUID fields have `,uuid` tags when:**
- Adding new models
- Refactoring to use `commondatabasehandler` pattern
- Debugging empty API list responses
- Reviewing pull requests with model changes

## Security Considerations

1. **JWT authentication** - bin-api-manager validates all external requests
2. **No secrets in code** - Use environment variables or CLI flags
3. **Base64 for certificates** - SSL certs passed as base64 strings in config
4. **Validate input** - Always validate data at service boundaries

## Resources

- Admin Console: https://admin.voipbin.net/
- Agent Interface: https://talk.voipbin.net/
- API Documentation: https://api.voipbin.net/docs/
- Project Site: http://voipbin.net/
- Architecture Diagram: `architecture_overview_all.png` in repo root

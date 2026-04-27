# Logging

### 5.1 Function-Scoped Logger

**MANDATORY:** Create a function-scoped log variable as the first statement of every function:

```go
// CORRECT — multiple context fields
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
    log := logrus.WithFields(logrus.Fields{
        "func": "Get",
        "id":   id,
    })
    // use log throughout the function
}

// CORRECT — single context field
func (h *handler) processRequest(m *sock.Request) (*sock.Response, error) {
    log := logrus.WithField("func", "processRequest")
    // ...
}

// WRONG — using package-level logger
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
    logrus.Debugf("Getting flow %s", id)  // No function context
}
```

### 5.2 Log Levels

| Level | Use For | Example |
|-------|---------|---------|
| `Debug` | Routine operations, entry/progress | `log.Debug("Creating a new flow.")` |
| `Info` | Non-error notable events | `log.Infof("Could not get call: %v", err)` (not-found is not an error) |
| `Warn` | Safe-default fallbacks | `log.Warnf("Cache miss, falling back to DB")` |
| `Error` | All failures | `log.Errorf("Could not get channel: %v", err)` |

### 5.3 Structured Object Logging After Data Retrieval

**MANDATORY:** Add debug logs when retrieving data from other services or databases:

```go
// CORRECT — log the full object and key identifier after retrieval
call, err := h.callGet(ctx, callID)
if err != nil {
    log.Infof("Could not get call: %v", err)
    return nil, fmt.Errorf("call not found")
}
log.WithField("call", call).Debugf("Retrieved call info. call_id: %s", call.ID)

ch, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
if err != nil {
    log.Errorf("Could not get channel: %v", err)
    return nil, fmt.Errorf("no data available")
}
log.WithField("channel", ch).Debugf("Retrieved channel info. channel_id: %s", ch.ID)

// WRONG — no logging after retrieval
call, err := h.callGet(ctx, callID)
if err != nil {
    return nil, err  // Also missing: no log, no context
}
// silently continues without logging the retrieved object
```

### 5.4 Error Message Format

Use the consistent format `"Could not <action>: %v"` or `"Could not <action>. err: %v"`:

```go
// CORRECT
log.Errorf("Could not get flow info: %v", err)
log.Errorf("Could not get flow info. err: %v", err)

// WRONG — inconsistent formats
log.Errorf("Error getting flow: %v", err)
log.Errorf("failed to get flow %v", err)
log.Errorf("GetFlow failed: %v", err)
```

### 5.5 External Event & Webhook Processing Logs

**MANDATORY:** When processing external events (webhooks, payment events, third-party callbacks, inter-service events), log at these points using **Info** level — not Debug.

External events are asynchronous, hard to replay, and involve money or state changes. Debug-level logs are filtered out in production, making webhook issues invisible until they escalate.

**Required log points:**

| Point | Level | What to Include |
|-------|-------|-----------------|
| Event receipt | Info | Event type, event ID |
| Processing start | Info | Operation name, key resource IDs (transaction_id, subscription_id, customer_id), amounts |
| Processing success | Info | Outcome details (account_id, plan_type, token_allowance, amount applied) |
| Processing failure | Error | Error with context about what was being attempted |
| Skip / no-op | Info | Why the event was skipped (missing data, idempotency duplicate, precondition not met) |
| Data retrieval | Debug | Retrieved objects with key identifiers (existing convention §5.3) |

**Pattern:**

```go
// Info: event receipt (listenhandler / webhook routing layer)
log.Infof("Received payment event. event_type: %s, event_id: %s", event.Type, event.ID)

// Info: processing start with business context
log.Infof("Processing subscription create. subscription_id: %s, customer_id: %s, plan_type: %s", subID, customerID, planType)

// Debug: data retrieval (per §5.3)
log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

// Info: success with outcome
log.Infof("Subscription created. account_id: %s, plan_type: %s, token_allowance: %d", acc.ID, planType, tokens)

// Info: skip with reason
log.Infof("Missing customer_id in custom_data, skipping. subscription_id: %s", subID)

// Error: failure with context
log.Errorf("Could not process subscription create: %v", err)
```

**Why Info, not Debug:**
- Debug logs are typically filtered in production
- External events involve money, state changes, or third-party interactions
- When a payment issue is reported, Info-level logs are the first diagnostic tool
- The volume is low (webhook events are infrequent compared to internal operations)

**Key identifiers to include** (when available):
- External event IDs (event_id, transaction_id, subscription_id)
- Internal resource IDs (customer_id, account_id)
- Business values (amounts, plan types, token counts)

**Applies to:** All webhook handlers (hook-manager, billing-manager), event subscribers (subscribehandler), and any handler processing external callbacks.

### 5.6 Import Pattern

Always import logrus directly without alias:

```go
// CORRECT
import "github.com/sirupsen/logrus"

func (h *handler) Get(ctx context.Context, id uuid.UUID) {
    log := logrus.WithFields(logrus.Fields{"func": "Get", "id": id})
}

// WRONG — aliasing logrus
import log "github.com/sirupsen/logrus"  // Confusing: shadows log variable pattern
```

---

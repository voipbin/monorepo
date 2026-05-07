# Inter-Service Communication

## Communication Pattern Rules

These prohibitions and prescriptions apply to every inter-service call in the monorepo. They are explicit because in practice each one has been violated and caused a coupling or operability incident:

1. **Never use HTTP between services** — Always use RabbitMQ RPC. Direct HTTP between `bin-*-manager` services bypasses circuit-breaker protection (see [`../patterns/circuit-breaker.md`](../patterns/circuit-breaker.md)) and the per-target RPC metrics, and creates synchronous coupling that defeats the queue-buffering model.
2. **Use typed request methods** — Don't construct `sock.Request` manually. Use the `requesthandler` typed methods (e.g., `r.AgentV1AgentGet(...)`); see §9.1 below.
3. **Handle async responses** — RabbitMQ RPC is asynchronous. Always pass `context.Context` and respect its deadline; don't assume responses are immediate.
4. **Publish events for notifications** — Use `notifyhandler.PublishEvent()` for pub/sub notifications. Don't fan-out via N RPC calls; that's what the event broker is for.

### 9.1 RabbitMQ RPC via RequestHandler

All inter-service calls go through `requesthandler.RequestHandler` typed methods. Never call services directly:

```go
// CORRECT — typed RPC call
agent, err := h.reqHandler.AgentV1AgentGet(ctx, agentID)

// WRONG — constructing raw RPC requests
req := &sock.Request{URI: "/v1/agents/" + id.String(), Method: "GET"}
resp, err := h.sockHandler.RequestPublish(ctx, "bin-manager.agent-manager.request", req)
```

### 9.2 ListenHandler Routing

Incoming RPC requests are routed by regex matching on URI + method:

```go
// CORRECT — regex routing pattern
var (
    regV1Agents    = regexp.MustCompile("/v1/agents$")
    regV1AgentsGet = regexp.MustCompile(`/v1/agents\?(.*)$`)
    regV1AgentsID  = regexp.MustCompile("/v1/agents/" + regUUID + "$")
)

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
    ctx := context.Background()  // fresh context per request
    switch {
    case regV1AgentsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
        return h.processV1AgentsGet(ctx, m)
    case regV1Agents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
        return h.processV1AgentsPost(ctx, m)
    }
}
```

### 9.3 Queue Naming

Services use three queues:
```
bin-manager.<service-name>.request    # RPC requests
bin-manager.<service-name>.event      # Published events
bin-manager.<service-name>.subscribe  # Event subscriptions
bin-manager.delay                     # Shared delayed message queue
```

### 9.4 Response Status Codes

Use HTTP-style status codes in `sock.Response`:

```go
// CORRECT
return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
return &sock.Response{StatusCode: 404}, nil
return &sock.Response{StatusCode: 500}, nil
```

### 9.5 Listener Request-Body Models Are Flat

Request structs in `pkg/listenhandler/models/request/` MUST be flat — top-level fields directly, no nested `Request` wrapper. This matches `calls.go`, `recordings.go`, `confbridge.go`, etc.

```go
// CORRECT — flat
type V1DataOutboundConfigsIDPut struct {
    Name                 *string   `json:"name,omitempty"`
    Detail               *string   `json:"detail,omitempty"`
    DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
    Codecs               *string   `json:"codecs,omitempty"`
}

// WRONG — wrapper introduces a wire-format mismatch with bin-common-handler clients
type V1DataOutboundConfigsIDPut struct {
    Request outboundconfig.UpdateRequest `json:"request"`
}
```

**Why this matters.** Wire-format wrappers must agree on both sides — `bin-common-handler/pkg/requesthandler/*.go` marshals the body, `bin-call-manager/pkg/listenhandler/v1_*.go` unmarshals it. `json.Unmarshal` silently ignores unknown top-level keys, so a mismatch produces a zero-valued struct with no error. The downstream handler then operates on nil/zero fields and the bug surfaces as "200 OK but nothing changed." A real production incident — see [`../workflows/common-gotchas.md`](../workflows/common-gotchas.md) (Listener Wire-Format Mismatch).

**Domain models stay where they belong.** A handler-layer "partial update" model with pointer fields (e.g., `outboundconfig.UpdateRequest`) is legitimately reused by the DB layer to build dynamic `UPDATE … SET` SQL. Keep it in `models/<entity>/`. The listener translates the flat RPC struct into the domain model:

```go
var req request.V1DataOutboundConfigsIDPut
if err := json.Unmarshal(m.Data, &req); err != nil { ... }

updateReq := &outboundconfig.UpdateRequest{
    Name:                 req.Name,
    Detail:               req.Detail,
    DestinationWhitelist: req.DestinationWhitelist,
    Codecs:               req.Codecs,
}
c, err := h.outboundConfigHandler.Update(ctx, id, updateReq)
```

### 9.6 Checklist for New RPC Methods

Every new `requesthandler` typed method + listener handler pair must:

1. **Listener model is flat.** No `Request` wrapper field; no import of the domain model from `pkg/listenhandler/models/request/`. (§9.5)
2. **Client marshals via the listener model.** `json.Marshal(cmrequest.V1DataXxx{...})` — never `json.Marshal(domainRequest)` or an inline anonymous struct.
3. **Listener test asserts the parsed payload.** Replace `gomock.Any()` with a field-by-field matcher on the request body. (See [`testing.md`](testing.md) §13.8.)
4. **Client test asserts the marshaled wire shape.** Mock `sockHandler.RequestPublish` and verify `sock.Request.Data` byte-for-byte. (See [`testing.md`](testing.md) §13.9.)

If any one of these is missing, a wire-format mismatch can silently round-trip as a no-op. All four must be in place.

---

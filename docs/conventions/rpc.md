# Inter-Service Communication

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

---

# bin-common-handler Usage

## Import Guidelines

Every `bin-*-manager` service declares a `replace` directive in its `go.mod`:

```
replace monorepo/bin-common-handler => ../bin-common-handler
```

Import the packages you need:

```go
import (
    "monorepo/bin-common-handler/pkg/requesthandler"
    "monorepo/bin-common-handler/pkg/notifyhandler"
    "monorepo/bin-common-handler/pkg/sockhandler"
    "monorepo/bin-common-handler/models/sock"
    "monorepo/bin-common-handler/models/identity"
)
```

**Do not import `rabbitmqhandler` directly** in consumer services. Use `sockhandler`, which abstracts the transport.

### Admission rule reminder

Before adding a new package to `bin-common-handler`, verify that 3 or more existing services need it. If fewer than 3 services use it, implement it in the consuming service(s) instead.

## Common Patterns

### Constructing a RequestHandler

```go
rh, err := requesthandler.NewRequestHandler(
    ctx,
    sock,                    // sockhandler.SockHandler
    "my-service-namespace",  // used for Prometheus metric names
)
if err != nil {
    return err
}

// Call another service
call, err := rh.CallV1CallGet(ctx, callID)
```

All RPC methods go through `sendRequest()` in `pkg/requesthandler/send_request.go`. The circuit breaker is applied here automatically. Do not add another circuit breaker layer in the consumer.

### Publishing events with NotifyHandler

```go
nh, err := notifyhandler.NewNotifyHandler(
    ctx,
    sock,
    "my-service-namespace",
)

// Publish a domain event
err = nh.PublishEvent(ctx, "bin-manager.my-service.event", eventType, data)

// Send a webhook
err = nh.PublishWebhook(ctx, customerID, webhookURL, payload)
```

### Mock generation

All handler interfaces in `bin-common-handler` have generated mocks. In consumer services, generate mocks for the interfaces you import:

```go
//go:generate mockgen -package mypackage -destination ./mock_main.go \
//   -source main.go -build_flags=-mod=mod
```

Run `go generate ./...` from the service root.

### Identity model

Embed `identity.Identity` in every resource struct:

```go
import "monorepo/bin-common-handler/models/identity"

type MyResource struct {
    identity.Identity          // provides ID and CustomerID
    Name string `json:"name"`
    // ...
}
```

UUID fields on shared models must use the `,uuid` db tag; JSON fields must use the `,json` db tag. See [docs/conventions/models.md](../docs/conventions/models.md).

### Queue names

Use the canonical constants from `models/outline` — do not hardcode queue name strings:

```go
import "monorepo/bin-common-handler/models/outline"

queueName := outline.QueueNameCallRequest  // "bin-manager.call-manager.request"
```

## Changing a public API

When you change an exported function signature, interface method, or model field in `bin-common-handler`:

1. Make the change.
2. Run `go build ./...` in `bin-common-handler` itself.
3. Run `go build ./...` in every consumer service (or run the CI pipeline).
4. Bulk find-and-replace across the monorepo for old call sites — use AST-aware tooling (e.g., `gopls rename`), not plain `sed`, because multi-line call sites will be missed by text replacement.
5. Run the full verification workflow in each affected service before committing.

See [docs/workflows/common-gotchas.md](../docs/workflows/common-gotchas.md) for the "Updating Shared Library Function Signatures" gotcha.

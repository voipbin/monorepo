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
    "monorepo/bin-common-handler/models/outline"
)
```

**Do not import `rabbitmqhandler` directly** in consumer services. Use `sockhandler`, which abstracts the transport.

### Admission rule reminder

Before adding a new package to `bin-common-handler`, verify that 3 or more existing services need it. If fewer than 3 services use it, implement it in the consuming service(s) instead.

## Common Patterns

### Constructing a RequestHandler

```go
rh := requesthandler.NewRequestHandler(
    sock,                                  // sockhandler.SockHandler
    outline.ServiceNameMyService,          // used for Prometheus metric names
)

// Call another service
call, err := rh.CallV1CallGet(ctx, callID)
```

`NewRequestHandler` takes no `context.Context` and returns a single `RequestHandler` value (no error) — construction cannot fail. All RPC methods go through `sendRequest()` in `pkg/requesthandler/send_request.go`. The circuit breaker is applied here automatically. Do not add another circuit breaker layer in the consumer.

### Publishing events with NotifyHandler

```go
nh := notifyhandler.NewNotifyHandler(
    sock,                                  // sockhandler.SockHandler
    rh,                                    // requesthandler.RequestHandler
    outline.QueueNameMyServiceEvent,       // this service's own fanout queue (used when the option below is omitted)
    outline.ServiceNameMyService,
    notifyhandler.WithGlobalTopicPublish(), // publish to the global topic exchange instead of this service's fanout queue
)

// Typical case: publish the internal event AND deliver the customer webhook
nh.PublishWebhookEvent(ctx, customerID, eventType, data)

// Internal event only, no customer webhook
nh.PublishEvent(ctx, eventType, data)

// Webhook only, no internal event (rarely used directly -- PublishWebhookEvent covers the common case)
nh.PublishWebhook(ctx, customerID, eventType, data)
```

`NewNotifyHandler` takes no `context.Context` and returns a single `NotifyHandler` value (no error) — the now-mandatory `reqHandler` argument is what `PublishWebhookEvent`/`PublishWebhook` use internally to resolve and deliver the webhook. None of `PublishEvent`, `PublishWebhook`, or `PublishWebhookEvent` return anything, not even an error — they are fire-and-forget.

`data` must satisfy `notifyhandler.WebhookEventMessage` for `PublishWebhookEvent` (a `WebhookMessage` plus `eventtopic.SubscriptionIdentifier`) or plain `eventtopic.SubscriptionIdentifier` for `PublishEvent`. In practice this is free: embedding `identity.Identity` by value and passing the struct as a pointer satisfies `EventSubscriptionID()` via method promotion, so most resource types need no explicit implementation.

**`WithGlobalTopicPublish()`** makes this handler instance publish exclusively to the global topic exchange `bin-manager.event` instead of this service's own per-service fanout queue (VOIP-1404/VOIP-1407) — see [docs/architecture.md](architecture.md#pkgnotifyhandler) for the routing-key scheme. Every production publisher passes this option today except `voip-asterisk-proxy`.

### Mock generation

All handler interfaces in `bin-common-handler` have generated mocks. In consumer services, generate mocks for the interfaces you import:

```go
//go:generate mockgen -package mypackage -destination ./mock_main.go -source main.go -build_flags=-mod=mod
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

UUID fields on shared models must use the `,uuid` db tag; JSON fields must use the `,json` db tag. See [docs/conventions/models.md](../../docs/conventions/models.md).

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

See [docs/workflows/common-gotchas.md](../../docs/workflows/common-gotchas.md) for the "Updating Shared Library Function Signatures" gotcha.

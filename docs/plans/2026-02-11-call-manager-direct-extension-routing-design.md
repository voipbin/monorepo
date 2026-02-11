# Call-Manager Direct Extension Routing Design

## Problem Statement

We introduced direct extensions in registrar-manager — when enabled on an extension, the system generates a random hash and stores a mapping in the `registrar_directs` table. The extension becomes reachable at `sip:direct.<hash>@sip.voipbin.net`.

The registrar-manager side is complete: hash CRUD, the `RegistrarV1ExtensionDirectGetByHash` RPC, and OpenAPI updates are all in place.

However, call-manager does not yet handle incoming calls to `direct.<hash>@sip.voipbin.net`. Currently, such calls hit the SIP domain handler (`startIncomingDomainTypeSIP`), which tries to match the destination against a phone number in number-manager. Since `direct.<hash>` is not a phone number, the call is hung up with "no route".

## Approach

Detect `direct.<hash>` destinations inside the existing `startIncomingDomainTypeSIP` handler. When the destination number starts with `direct.`, extract the hash, resolve it to an extension via a new RPC, and route the call to that extension using a temp connect flow.

This avoids adding a new domain type — the domain IS `sip.voipbin.net`, so we handle it within the existing SIP domain handler.

## New RPC: `RegistrarV1ExtensionGetByDirectHash`

A single RPC call that resolves hash → full Extension, avoiding two round-trips (get direct → get extension).

### registrar-manager endpoint

`GET /v1/extensions/by-direct-hash/<hash>` — returns a single `Extension` with `DirectHash` populated.

Handler logic:
1. `extensionDirectHandler.GetByHash(ctx, hash)` → `ExtensionDirect`
2. `dbBin.ExtensionGet(ctx, direct.ExtensionID)` → `Extension`
3. Set `ext.DirectHash = direct.Hash`
4. Return `ext`

If hash not found or extension deleted → return error (call-manager hangs up with no route).

### bin-common-handler method

```go
RegistrarV1ExtensionGetByDirectHash(ctx context.Context, hash string) (*rmextension.Extension, error)
```

Sends RPC to registrar-manager at `GET /v1/extensions/by-direct-hash/<hash>`, parses response into `rmextension.Extension`.

## Call-Manager Changes

### Detection in `startIncomingDomainTypeSIP`

Check `cn.DestinationNumber` for `direct.` prefix BEFORE the existing address parsing and number lookup:

```go
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
    // direct extension check first
    if strings.HasPrefix(cn.DestinationNumber, "direct.") {
        hash := strings.TrimPrefix(cn.DestinationNumber, "direct.")
        return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, hash)
    }

    // existing code unchanged...
    source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
    destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
    // ... number lookup ...
}
```

### New handler: `startIncomingDomainTypeSIPDirectExtension`

```go
func (h *callHandler) startIncomingDomainTypeSIPDirectExtension(
    ctx context.Context,
    cn *channel.Channel,
    hash string,
) error {
    source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)

    ext, err := h.reqHandler.RegistrarV1ExtensionGetByDirectHash(ctx, hash)
    if err != nil {
        // hash not found or extension deleted
        h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
        return nil
    }

    destination := &commonaddress.Address{
        Type:       commonaddress.TypeExtension,
        Target:     ext.ID.String(),
        TargetName: ext.Extension,
    }

    // create temp connect flow
    actions := []fmaction.Action{
        {
            Type: fmaction.TypeConnect,
            Option: fmaction.ConvertOption(fmaction.OptionConnect{
                Source:       *source,
                Destinations: []commonaddress.Address{*destination},
                EarlyMedia:   false,
                RelayReason:  false,
            }),
        },
    }

    f, err := h.reqHandler.FlowV1FlowCreate(
        ctx,
        ext.CustomerID,
        fmflow.TypeFlow,
        "tmp",
        "tmp flow for direct extension dialing",
        actions,
        uuid.Nil,
        false,
    )
    if err != nil {
        h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
        return nil
    }

    h.startCallTypeFlow(ctx, cn, ext.CustomerID, f.ID, source, destination)
    return nil
}
```

This follows the exact same pattern as `startIncomingDomainTypeRegistrarDestinationTypeExtension` — same temp connect flow approach, same error handling.

## Files to Create/Modify

### bin-registrar-manager (3 files modified)

- `pkg/extensionhandler/main.go` — add `GetByDirectHash(ctx, hash) (*extension.Extension, error)` to interface
- `pkg/extensionhandler/extension.go` — implement `GetByDirectHash`
- `pkg/listenhandler/main.go` — add regex `regV1ExtensionsByDirectHash` and route case
- `pkg/listenhandler/v1_extensions.go` — add `processV1ExtensionsByDirectHashGet` handler

### bin-common-handler (2 files modified)

- `pkg/requesthandler/main.go` — add `RegistrarV1ExtensionGetByDirectHash` to interface
- `pkg/requesthandler/registrar_extensions.go` — implement the RPC method

### bin-call-manager (1 file modified)

- `pkg/callhandler/start_incoming_domain_type_sip.go` — add `direct.` prefix check and `startIncomingDomainTypeSIPDirectExtension` method

### Tests

- `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go` — add test for direct extension path (happy path + hash not found)
- `bin-registrar-manager/pkg/extensionhandler/extension_test.go` — add test for `GetByDirectHash`

### Verification

Since bin-common-handler is modified, full verification across all 30+ services is required.

## Edge Cases

- **Hash not found**: `RegistrarV1ExtensionGetByDirectHash` returns error → hangup with no route (404)
- **Extension deleted but direct record exists**: `GetByDirectHash` fetches extension by ID — if extension is soft-deleted, `ExtensionGet` returns error → hangup with no route
- **Customer has no balance**: `startCallTypeFlow` already calls `ValidateCustomerBalance` — rejects the call if insufficient
- **Existing SIP calls unaffected**: Phone numbers start with `+`, not `direct.` — the prefix check doesn't trigger

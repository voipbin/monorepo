# Design: Add `type: "ng"` command type to rtpengine-proxy

## Problem

The rtpengine-proxy's `POST /v1/commands` endpoint currently handles NG protocol
commands as an implicit fallthrough — any request without a recognized `type` field
gets forwarded to RTPEngine. This creates ambiguity: unknown type values silently
fall through, and there's no explicit routing for NG commands.

## Approach

1. Add a `models/command/` package to rtpengine-proxy with `Type` constants (`ng`, `exec`, `kill`)
   and a `Command` struct that captures typed fields plus the raw payload as `Data`.

2. Update `listenhandler/command.go` to:
   - Parse incoming JSON into the hybrid `Command` struct
   - Switch on `cmd.Type` using constants
   - Route `ng` to a new `processNG()` handler
   - Return 400 for missing or unknown types

3. Update `bin-call-manager/pkg/callhandler/rtpdebug.go` to add `"type": "ng"` to
   the query command sent to rtpengine-proxy.

## Files changed

- `voip-rtpengine-proxy/models/command/command.go` (new)
- `voip-rtpengine-proxy/pkg/listenhandler/command.go`
- `voip-rtpengine-proxy/pkg/listenhandler/command_test.go`
- `voip-rtpengine-proxy/pkg/listenhandler/main_test.go`
- `bin-call-manager/pkg/callhandler/rtpdebug.go`
- `bin-call-manager/pkg/callhandler/status_test.go` (if NG command assertions exist)

## Trade-offs

- Breaking change: callers that omit `type` will now get 400 instead of NG forwarding.
  Acceptable because the only caller is call-manager (which we update in the same PR).

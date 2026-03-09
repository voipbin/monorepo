# Fix RPC Silent Response Drop

**Date:** 2026-03-09
**Branch:** NOJIRA-Fix-RPC-silent-response-drop
**Issue:** https://github.com/voipbin/monorepo-monitoring/issues/75

## Problem

`GET /speakings` returns HTTP 400 after error-path requests (DELETE/POST to non-existent speaking IDs). The API validator test `test_list_speakings_schema_validation` consistently fails when run as part of the full test suite but passes in isolation.

### Root Cause Chain

1. TTS listenhandler methods (e.g., `v1SpeakingsIDGet`, `v1SpeakingsIDDelete`) return `(nil, err)` when the underlying speaking operation fails.
2. `executeConsumeRPC` in rabbitmqhandler silently drops the RPC response when the handler callback returns `(nil, err)` — no reply is sent to the caller's ReplyTo queue.
3. The RPC caller times out after 3 seconds (`requestTimeoutDefault = 3000`).
4. The circuit breaker records each timeout as a transport-level failure.
5. Five error-path tests produce exactly 5 consecutive transport failures, which trips the circuit breaker (`defaultFailureThreshold = 5`).
6. The circuit breaker stays open for 30 seconds (`defaultOpenDuration = 30s`).
7. The subsequent `GET /speakings` request is rejected by the circuit breaker before it ever reaches the TTS service.
8. The test retries for only 24 seconds (8 attempts x 3s), which is not enough to outlast the 30-second open duration.

## Approach: Two-Layer Fix

### Layer 1: RabbitMQ RPC Safety Net (bin-common-handler)

Add `publishRPCErrorResponse` to `executeConsumeRPC`: when a handler callback returns an error and a `ReplyTo` address exists, send a 500 error response back to the caller instead of dropping the response. This ensures the caller always receives a reply and records an application-level error (not a transport timeout), which does not trip the circuit breaker.

### Layer 2: TTS Handler Proper Status Codes (bin-tts-manager)

Change all TTS listenhandler methods to return `(simpleResponse(statusCode), nil)` instead of `(nil, err)`. Status codes: 400 for bad requests/unmarshal errors, 404 for not-found, 500 for internal/marshal errors. This is the correct pattern — handler methods should always return a sock.Response with an appropriate status code, not a Go error.

## Files Changed

- `bin-common-handler/pkg/rabbitmqhandler/consume.go` — Added `publishRPCErrorResponse` method and modified `executeConsumeRPC` to call it on error
- `bin-tts-manager/pkg/listenhandler/v1_speakings.go` — Changed all error returns from `(nil, err)` to `(simpleResponse(statusCode), nil)`

## Verification

- bin-common-handler: `go test ./...` passes, `golangci-lint` clean (0 issues)
- bin-tts-manager: `go test ./...` passes, `golangci-lint` clean (1 pre-existing unrelated issue in dbhandler test)

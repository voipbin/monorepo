# Add request/response DEBUG logging to timeline-manager listenhandler

Date: 2026-06-09
Type: Observability hardening (minimal, behavior-preserving)
Scope: bin-timeline-manager, bin-registrar-manager

## Problem Statement

`bin-timeline-manager`'s RPC `processRequest` (`pkg/listenhandler/main.go`) binds
`"request": m` to the logrus entry but **never emits it on the normal path**. A
log line is produced only on (a) handler error and (b) routing miss. The response
returned to the caller is **never logged anywhere**. As a result an operator cannot
tell from logs which request arrived, with what arguments, or what response was sent
back. This is the gap pchero observed in production.

timeline-manager is one of only three services (with flow-manager and call-manager)
that do not even emit a request-entry log. Across the monorepo, a response log
(`Sending back ...`) exists in exactly one service, `bin-registrar-manager`, and that
single line contains a typo (`resulut`).

## Scope

In scope:
- timeline-manager: emit a request-entry DEBUG log and a response DEBUG log in
  `processRequest`, matching the established registrar-manager pattern.
- registrar-manager: fix the `resulut` -> `result` typo on the existing response log.

Out of scope (deferred, explicitly NOT this PR):
- Standardizing response logging across the other ~33 services (separate follow-up;
  large blast radius, needs Cloud Logging exclusion-filter companion).
- Adding request-entry logs to flow-manager / call-manager.
- Any data truncation / PII redaction policy on the logged payload (registrar logs
  the full `m.Data` today; we match that convention rather than diverge in this PR).

## Reference Pattern (registrar-manager, the only response-logging precedent)

Entry (bind structured fields, emit one DEBUG line):
```go
log := logrus.WithFields(logrus.Fields{
    "func":      "processRequest",
    "uri":       m.URI,
    "method":    m.Method,
    "data_type": m.DataType,
    "data":      m.Data,
})
log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)
```

Exit (bind response, emit one DEBUG line, before return):
```go
log.WithFields(logrus.Fields{
    "response": response,
}).Debugf("Sending back the result. method: %s, uri: %s", m.Method, m.URI)
return response, err
```

## Fix (before / after)

### timeline-manager — `pkg/listenhandler/main.go`, `processRequest`

Before:
```go
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	ctx := context.Background()
	...
	switch { ... }
	...
	return response, err
}
```

After:
```go
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "processRequest",
		"uri":       m.URI,
		"method":    m.Method,
		"data_type": m.DataType,
		"data":      m.Data,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	ctx := context.Background()
	...
	switch { ... }
	...
	log.WithFields(logrus.Fields{
		"response": response,
	}).Debugf("Sending back the result. method: %s, uri: %s", m.Method, m.URI)
	return response, err
}
```

Notes:
- The `log` binding moves from `"request": m` to the registrar-style discrete fields
  (`uri/method/data_type/data`). No existing call site reads the old `"request"` field.
- The metrics `defer` (promReceivedRequestProcessTime) and the entire routing/error
  switch are untouched.
- The single `return response, err` is the only exit; the response log sits immediately
  above it, after the error-mapping block, so it logs the final response (including
  mapped error responses).

### registrar-manager — `pkg/listenhandler/main.go`, line ~298

Before:
```go
}).Debugf("Sending back the resulut. method: %s, uri: %s", m.Method, m.URI)
```
After:
```go
}).Debugf("Sending back the result. method: %s, uri: %s", m.Method, m.URI)
```

## Impact Analysis

| Scenario | Before | After |
|---|---|---|
| Normal request (e.g. GET /v1/events) | silent | 2 DEBUG lines (entry + response) |
| Handler error | 1 Errorf | 1 Errorf + 1 response DEBUG (final mapped response) |
| Routing miss (404) | 1 Errorf | 1 Errorf + 1 response DEBUG |
| registrar response log | typo `resulut` | `result` |

Behavior (routing, metrics, returned response/err) is identical. Only DEBUG-level
log emission changes.

## Log level: DEBUG (justified)

Both new lines are DEBUG, matching the registrar precedent. This IS visible to the
requester: timeline-manager runs at DEBUG in production today (confirmed empirically —
the production log dump that triggered this work was `"severity": "DEBUG"` from this
same container). So the entry/response lines surface in prod without any level change.
Re-evaluating DEBUG-vs-INFO is a concern only for the deferred 33-service rollout, not
for this timeline-scoped PR.

## Field consistency (entry vs response line)

Both lines share the same base `log` entry, which binds `uri/method/data_type/data` as
structured fields. The entry line emits `Received request`; the response line is
`log.WithFields({"response": response}).Debugf("Sending back the result. ...")`, so it
inherits uri/method/data_type/data AND adds `response`. Field names are uniform across
both lines and match registrar, keeping cross-service log queries consistent.

## Risks

- `m.Data` may carry SIP payloads / customer data. Matches registrar's existing
  behavior; logged at DEBUG only. Truncation/redaction is a cross-service policy
  deferred out of scope.
- Large response payloads (known limitation): `POST /v1/sip/pcap` returns binary/base64
  packet-capture data; `GET /v1/events` (paginated list) and `POST /v1/sip/analysis`
  can be large. At DEBUG the full `response` struct is splatted into one log line. This
  is accepted for this PR (DEBUG-gated, matches registrar full-payload convention) and
  is explicitly named here so operators enabling DEBUG are not surprised by multi-MB
  lines. Per-endpoint truncation is deferred to the cross-service rollout.
- Volume: timeline-manager request volume is moderate (read API); 2 DEBUG lines/request
  is acceptable and DEBUG-gated.

## Test

- No new unit test required: the change adds log emission only, with no branch/return
  semantics change. Existing `listenhandler` tests (`main_test.go`,
  `v1_*_test.go`, `error_response_test.go`) must still pass unchanged.
- Verification: full monorepo workflow (`go mod tidy && go mod vendor &&
  go generate ./... && go test ./... && golangci-lint run`) run in BOTH
  `bin-timeline-manager` and `bin-registrar-manager`.

## Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-timeline-manager | Add entry + response DEBUG logs to processRequest | 1 |
| bin-registrar-manager | Fix `resulut` -> `result` typo | 1 |

## Open Questions

| Question | Recommendation | Owner |
|---|---|---|
| Standardize response logging across all ~33 services? | Separate follow-up PR + exclusion-filter; not now | CEO/CTO |
| Truncate/redact `m.Data` in logs platform-wide? | Cross-service policy decision, defer | CEO/CTO |

## Review Summary (v1 -> v2)

Two independent reviewers. v1: 1 APPROVE, 1 CHANGES REQUESTED.

- B1 (DEBUG visibility): reviewer worried INFO-prod would hide the logs. Rebutted
  empirically — timeline-manager runs at DEBUG in prod (the triggering log dump was
  `severity: DEBUG`). Added "Log level: DEBUG (justified)" section.
- S1 (large response payloads): named pcap/events/analysis as a known limitation in
  Risks.
- S2 (registrar typo in a timeline-titled PR): keep single PR; PR title/body explicitly
  cover both services. Trivial 1-line fix, same subject (listenhandler logging).
- N1 (field consistency): added "Field consistency" section confirming both lines share
  the uri/method/data_type/data base binding.
- Deferred (S3 cross-service convention doc): folded into Open Questions; out of scope
  for this timeline-only PR.

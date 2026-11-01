# VOIP-1671 bin-agent-manager: stop logging credential bodies and webhook secrets

## 1. Ticket review (adversarial, against origin/main 809a7007c)
Valid. The listen handler logs the whole `sock.Request` (body included) on every
Debug/Error line of `processRequest`, and the subscribe handler logs the whole
`sock.Event`. The service logs at Debug level unconditionally (`initLog`).
Bodies carry agent passwords (login, agent create, password PUT, password reset)
and customer events carry `webhook_secret`. Not fixed or duplicated on main.

## 2. Change (bin-agent-manager only, log output only)
listenhandler:
- `main.go`: new `requestLogFields(m)` returning `uri`, `method`, `data_type`,
  `request_id`, `data_len` (allowlist; the body is never logged). `processRequest`
  and `v1_count.go` use it instead of `"request": m`.
- `v1_agents_direct_hash.go`: drop `WithField("request", m)`.
- 14 sites `Could not unmarshal the data. data: %v`: log `data_len` instead.
- 13 sites `Could not marshal the response message. message: %v, tmp`: drop `tmp`
  (a bare `%v` of an `Agent` would print `PasswordHash`).
subscribehandler:
- `processEvent` field `"message": m`, line `WithField("event", m)`, and the three
  handlers' `"event": m` (5 sites): replaced by `publisher` and `type` only.

Not changed (out of scope, stated in the PR): the `"response": response` Debug line
(agent JSON, `PasswordHash` is `json:"-"`); other services (a follow-up check is
proposed separately).
Also out of scope by deliberate exclusion (checked): `v1_login.go` `Sending result`
and the `"response"` line print agent JSON, which has no password material; other
agent fields in it and in the `agenthandler` `WithField("agent", a)` dumps are a
separate class, out of this ticket.
Accepted residual: the exclusion of agent dumps relies on the JSON formatter
(`PasswordHash` is `json:"-"`); a `%v` print of an `Agent` would show it, which is
why the `tmp` arguments are removed. A `json.Unmarshal` syntax error may name one offending character
of the body, type errors name only the Go type and field.

## 3. Compatibility
Log keys removed: `request` (object), `message` (object), `event` (object) in
the lines above. Added: `uri`, `method`, `data_type`, `request_id`, `data_len`,
`publisher`, `type`. No API, DB, RPC, or event change. No deployment order.
External dashboards and already-written past logs were not checked.

## 4. Verification plan
- RED first: sentinel-capture tests (logrustest, Debug level, hook on the
  standard logger) fail on current code, then pass.
- Sentinels in plaintext and as the decimal byte-array form (`[123 34 ...]`),
  asserted against `json.Marshal(entry.Data)`, `%v` of Data, and `entry.Message`.
- Routes: /v1/login (success and failure), /v1/agents POST,
  /v1/agents/<name>/login, /v1/agents/<id>/password PUT, /v1/password-reset,
  count_by_customer, direct hash regenerate, unknown URI (Error line), handler
  returning an error (Error line), unmarshal failure, marshal failure.
- Subscribe: customer_created/deleted with `webhook_secret` sentinel, Debug line,
  unmarshal-error and handler-error Error lines.
- Each test asserts at least one entry with the expected message (no vacuous pass).
- Mutation check: revert each site, the matching test fails.
- Full `go test ./...` for the service plus the repo verification workflow.

## 5. Implementation plan
```go
// requestLogFields returns the fields that are safe to log for a request.
// The body is never logged (allowlist).
func requestLogFields(m *sock.Request) logrus.Fields {
	return logrus.Fields{"uri": m.URI, "method": m.Method,
		"data_type": m.DataType, "request_id": m.RequestID, "data_len": len(m.Data)}
}
```
- `processRequest`: `logrus.WithFields(requestLogFields(m))`; `v1_count.go`:
  `logrus.WithFields(requestLogFields(m)).WithField("func", ...)`.
- The 14 unmarshal sites: `log.Debugf("Could not unmarshal the data. data_len: %d, err: %v", len(m.Data), err)`
  (plain `len`, no duplicate key; the `Debugf` text keeps the message prefix).
- `processEvent`: fields `func`, `publisher`, `type`, `data_len`; the three handlers:
  `publisher`, `type` taken from `m`.
- Tests (new files `log_redaction_test.go` in each package, no `t.Parallel`, hook
  attached per test via `logrustest.NewLocal(logrus.StandardLogger())`, Debug level
  restored on cleanup): assert the sentinel is present in the input first, at least
  one entry has the expected message, and the `processRequest` entry has the `uri` and
  `data_len` fields. Do not put a `direct_hash` sentinel in login success responses (the
  excluded `Sending result` line prints the agent JSON). The `Could not unmarshal the
  request data` line in `v1_count.go` logs only `err` and stays unchanged.

## 6. Test method details (after design review round 1)
- Calls go through `processRequest` / `processEvent` directly (no goroutine), with
  `listenHandler{agentHandler: mock}` and `subscribeHandler{agentHandler: mock}`.
- Error-line paths: handler mock errors become `errorResponse`/`simpleResponse` with a
  nil error, so they hit the `Sending response` Debug line, not an Error line. The
  `err != nil` Error line in `processRequest` is reached only when `url.Parse` fails in
  `processV1AgentsGet`. Input `/%zz/v1/agents?a=1` matches `regV1AgentsGet` and fails
  `url.Parse` (a bad escape in the query alone does not fail); the test first asserts
  that `url.Parse` returns an error, and carries the sentinel in the request `Data` so
  the old `"request": m` field is what turns it RED. The unknown
  URI path reaches the 404 Error line. Subscribe handler mock errors do reach the
  `Could not process the event correctly` and per-handler Error lines.
- Assertion helper: for every collected entry check the plaintext sentinel and the
  decimal byte form (space separated digits, no brackets) in `entry.Message`,
  `json.Marshal(entry.Data)` (falling back to per-field `%v` when marshal fails) and
  per-field `fmt.Sprintf("%v", v)`. Unmarshal-failure inputs are valid JSON with a wrong
  shape (for example `[1,"SENTINEL"]`) so the marshal of an
  unfixed `*sock.Request` succeeds and RED is caused by the leak itself.
- Marshal-failure sites use the existing unserializable-agent pattern of
  `marshal_error_test.go` with `PasswordHash` set to a sentinel; the list site
  (`[]*Agent`) is excluded from the mutation check because a slice of pointers prints
  addresses only.
- Sentinels go in the request only, never in a response agent.

# VOIP-1709: add the ai service to the service configs admin API

## 1. Ticket review (against origin/main 758a40a24)

The admin API (`/service_configs/{service}`, VOIP-1697, merged as 774241e88) accepts `email` only.
`bin-ai-manager` already owns `ai_configs` and a `confighandler.Handler` with two secret keys (VOIP-1698,
merged as 3a0fbd767; this change builds on it) but does not serve the config RPC, so a request for
`service=ai` is answered 404 by ai-manager. Evidence (all read in code, none refuted):

- `serviceConfigServices` has `email` only (`bin-api-manager/pkg/servicehandler/service_config.go`).
- `configQueue` has the `email` case only (`bin-common-handler/pkg/requesthandler/config.go`).
- ai-manager has no `configrpc` use, no `/v1/configs` route (grep of `pkg/listenhandler`, `cmd/ai-manager`).
- OpenAPI `ServiceConfigServiceName` has `email` only; it is referenced twice (parameter `ServiceConfigService`
  and `ServiceConfig.service`).
- Open PRs: none in this scope. No merge after 774241e88 touched service configs.
- Callers that assume email only (`grep -rniE 'email'` over the 20 service config files): `requesthandler/config.go:21,28`,
  `service_config.go:29`, `server/service_configs.go:15`, `openapi.yaml:123,247,6577,6691,6740`, `key_secret.yaml:5`,
  `service_config_overview.rst:13,65,69`, `service_config.rst:6`, `service_config_struct_service_config.rst:21`,
  `routing.md:266`. `openapi.yaml:6691` and the Mailgun example of the struct rst are real email key examples and stay.
  The tests use the unknown service values `nothing`, `EMAIL`, `email/../x` (none uses `ai`).

The change is an extension. The behavior for `email` does not change.

## 2. Change points (one PR, no split)

| # | Module | Change |
|---|--------|--------|
| 1 | bin-common-handler `pkg/requesthandler/config.go` (+ test) | `ai` maps to `QueueNameAIRequest`, label `ai/configs` (fixed text) |
| 2 | bin-ai-manager `cmd/ai-manager/main.go` | `configrpc.NewServer(configHandler, serviceName)` in `run`, passed as the last argument of `runListen` and `NewListenHandler` (one call site each; no test calls them, tests build `&listenHandler{}`, so the new field is nil there; the `ListenHandler` interface is unchanged, no mock regeneration) |
| 3 | bin-ai-manager `pkg/listenhandler` | field `configServer`, new file `v1_configs.go`, early branch in `processRequest` (section 3) |
| 4 | bin-api-manager `pkg/servicehandler/service_config.go` (+ test) | `serviceConfigServices["ai"] = true`; comment of `server/service_configs.go` (512 bytes) made neutral |
| 5 | bin-openapi-manager `openapi/openapi.yaml`, `paths/service_configs/key_secret.yaml` | enum value `ai` (`ServiceConfigServiceNameAi`); texts: parameter description (`for example email`), enum `example`, tag description (credentials of the email providers), `SecretPutRequest.secret` description (`openapi.yaml:6740`, "the email keys allow 512 bytes") and `key_secret.yaml:5` (the same sentence): both become "the email and ai keys". `service.yaml`, `key.yaml`, `key_histories.yaml` have no email text. Regenerate `gens/models/gen.go`, api-manager `gens/openapi_server/gen.go` and `gens/openapi_redoc/openapi.json`, and check the drift (git diff after `go generate`) |
| 6 | docs | RST `service_config.rst` (intro), `service_config_overview.rst` (line 13 supported services and key names of both, line 65 the 512 byte sentence, line 69 "the email service has not been updated" becomes "the service"), `service_config_struct_service_config.rst` (`service` description), clean rebuild of `docsdev/build` and `git add -f`; `bin-api-manager/docs/routing.md:266` (ai: bin-ai-manager); ai-manager `docs/architecture.md` routing table (`/v1/configs` four routes) and `docs/operations.md` (the admin API as a second way to write the two keys, history source `api`, no value entry during a rolling deploy, `secret_writable=false` without `AI_CONFIG_ENCRYPTION_KEYS`); ai-manager `CLAUDE.md` one line; `bin-common-handler/docs/usage.md` (adding a service procedure: the listener logs nothing of the request or only `LogSafe(m)`) |

## 3. Listener design (ai-manager)

In `processRequest`, after the Builder branch and before the log entry exists:

```
if h.configServer.Match(m) || configrpc.LogSafe(m) != m { return h.processConfig(m) }
```

`LogSafe(m) != m` is true for any URI (lower cased) that starts with `/v1/config` or ends with `/secret`. No
current ai-manager route has such a URI (grep of `main.go`). `processConfig` does:

1. named results `(response *sock.Response, err error)` and a `defer` that recovers a panic (fixed log text, the
   value is never formatted, 500, err nil, like `processBuilder`) and then observes the existing latency histogram
   with the fixed type `/v1/configs` and `m.Method` (as every route does); the observation also runs after a panic;
2. if `Match` is false: the fixed 404 (a mistyped or unrouted secret request is never dispatched to another
   route: the ai-manager regexps are not anchored at the start, so `PUT /v1/configs/a/v1/ais/<uuid>` would
   reach the AI update handler with the secret body; the guard closes that. Behavior for other URIs is
   unchanged);
3. otherwise `configServer.Handle` (every failure is a typed response, the error is always nil).

No log line of `processConfig` contains the request. Scope note: the `recover` and the `LogSafe(m) != m` guard are
part of the requirement that a secret body appears nowhere (the ticket names the panic path), not a preemptive
extension: a panic value or a misrouted secret body is the only way left for it to leave the handler. The guard
is not reachable through api-manager (it validates the key and escapes it) but through any other publisher on
the queue; it costs one condition and one fixed 404, and each has its own mutation check (section 6). Why early routing and not `LogSafe` first like email:
the existing entry holds only `uri`, `method`, `request_id`, and the default branch only `method` and `uri`,
so the current code is safe; routing before the entry keeps the body out of reach of any field that a later
change adds to it, and the guard above gives the same coverage of mistyped secret URIs that `LogSafe` gives
email. The sock layer (`rabbitmqhandler/consume.go`) never logs the body (a parse failure logs `body_len`;
the handler error is always nil; a response marshal failure is a fixed text). It has no `recover`: a panic
outside `processConfig` is not caused by this change. The remaining log source is `configrpc.errorResponse`
(`Errorf` with the store error text); the library errors hold no value, and a test makes the store return an
error that carries the fake secret to show what the log holds.

## 4. Mismatch between the two lists, and a rolling deploy

| State | PUT secret | GET / DELETE / histories |
|-------|-----------|--------------------------|
| only api-manager has `ai` | the probe `ConfigV1ServiceGet` returns `ErrConfigServiceUnknown` before any RPC (GET, DELETE and histories reach the same error in `sendRequestConfig`); not mapped in `translateToVoipbinError` (`bin-api-manager/server/error_translate.go`: typed, sentinels, transport, default), so 500 `INTERNAL`; nothing is sent | same 500, nothing is sent |
| only requesthandler has `ai` | 404 `RESOURCE_NOT_FOUND` from the allow list | same |
| both, ai-manager not yet updated (queue consumed by the old version: the old default branch answers a bare 404) | the probe gets `ErrNotFound`, mapped to `ErrServiceUnavailable` (503); the secret is never sent. The old ai-manager would not log the body either (its default branch logs method and uri), so the probe lowers the chance that a secret reaches a service that cannot handle it | 404 `RESOURCE_NOT_FOUND` (no secret in those requests) |

Deploy order: any order is safe for data; the feature works when ai-manager and api-manager are both updated.
Frontend order: the frontend type of the service name must allow `ai` before the backend is used by the UI
(contract note in section 5). `bin-common-handler` is changed, so the build of the modules that use it is
checked in the PR.

## 5. Contract compatibility

(a) `ServiceConfigServiceName` is used by the path parameter and by `ServiceConfig.service` only. A client typed
`'email'` receives `ai` as a plain string at runtime; a client with a strict enum check rejects it, so the
frontend widens its type first. (b) The generated gin server binds the path value as a string, calls no enum
`Valid()`, and api-manager has no request validator middleware (`OapiRequestValidator`, `openapi3filter`: no
use); the allow list in servicehandler answers 404 for any other value (test). (c) The mapping is by the typed
error of `configrpc` (shared code): the reasons `CONFIG_KEY_NOT_FOUND` (404), `CONFIG_INVALID` (400),
`REVISION_MISMATCH` (409), `ENCRYPTION_KEY_NOT_CONFIGURED` (503), `CONFIG_WRITE_VERIFY_FAILED` (500),
`INTERNAL` (500) and the statuses are identical for `ai` and `email`; only the internal `domain` field differs
(`bin-ai-manager`, `bin-email-manager`), which the external envelope omits. Permission: project super admin only,
direct access refused (unchanged code path).

## 6. Verification plan

1. requesthandler: `ai` goes to the ai queue with the label `ai/configs` (all four calls); unknown services
   are refused as before.
2. listenhandler, entry `processRequest`, real `confighandler` and the in-memory store: GET service, PUT secret,
   DELETE key, GET histories return the expected status and body; a sample of existing routes (for example
   `/v1/ais`) behaves as before with a nil `configServer`; a nil server answers config URIs 404.
3. no-leak test (`internal/logtest` with `logrus.SetLevel(TraceLevel)` after `New`, fake secret): routed, stale,
   unknown key, white space, broken body, no master key, store error carrying the secret, upper case URI, 129
   character key, `POST .../secret`, typo `/v1/config/...`, a slash key that overlaps an ai route
   (`PUT /v1/configs/<uuid>/v1/ais/<uuid>`: `Match` is false and `processV1AIsIDPut` would accept it; the strict
   mock of the AI handler must record no call and the answer is the fixed 404), and a panic raised by a
   `confighandler.Handler` wrapper (embeds the handler, overrides `SetSecret` and panics with the plaintext secret
   it receives; the `Store` sees ciphertext only; a store wrapper that panics is a secondary check of the recover
   path). The table driven test checks `Dump()` of the capture with `strings.Contains` (a successful request
   writes no log line, so `AssertAbsent`, which fails on an empty capture, is used only for the failing cases;
   the panic case must show the fixed panic line, so it is not vacuous). Assertions: logs, whole response
   (`StatusCode`, `Data`), every label value of the metrics gathered from the default registry, and that the
   handler still sees the unmodified request body. The metric check first requires the series
   `type="/v1/configs"` of `ai_manager_receive_request_process_time` and compares the `SampleCount` of
   `type="/v1/configs", method="PUT"` before and after (the registry is process wide); a panic increases it.
4. mutation check in a scratch copy, each mutation must fail (a panic that ends the test run counts as a
   failure) a named test of item 3: add `"request": m` to the fixed panic log line of `processConfig` (panic
   case), add `m.Data` to a log line of the 404 guard path (slash key and typo cases), remove only
   `|| LogSafe(m) != m` (the slash key case: the AI handler would be called), remove the `recover` (panic case),
   remove the early branch (routing test: the config URIs would be answered 404 by the default branch).
5. servicehandler: `ai` allowed (all four operations), other values 404, `email` unchanged; the mismatch rows of
   section 4 (the probe error mapping) at servicehandler and requesthandler level.
6. `go build`, `go vet`, the full tests and the generated file drift checks of the four modules, and the
   repository gate scripts (`scripts/check-*.sh`).

## 7. Out of scope

Frontend, other services, production changes, `ai-control`, per customer keys.

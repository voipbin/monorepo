# VOIP-1568 api-manager: mask accesskey and token in access and panic recovery logs

## 1. Problem

api-manager accepts the accesskey and the JWT through the URL query (`?accesskey=`, `?token=`) and through cookies
(`bin-api-manager/lib/middleware/authenticate.go`, `getTokenString`, `getAccesskey`).
`runListenHTTP` in `bin-api-manager/cmd/api-manager/main.go` builds the engine with `gin.LoggerWithConfig(...)` and `gin.Recovery()`:

- The gin access logger appends `?` + `URL.RawQuery` to the logged path, so the accesskey plaintext reaches stdout on every request that uses the query form.
- `gin.Recovery()` dumps the request via `secureRequestDump`, which masks only the `Authorization` header. The request line (including the query) and the `Cookie` header are written as is. The `token` and `accesskey` cookies are the same secrets.

Existing precedent: `SkipPaths` already suppresses access logs for provisioning and MCP OAuth callback paths. Other paths have no protection.

## 2. Decision (ticket option A)

Mask the values of `accesskey` and `token` in both log sinks. Option B (drop query auth) and key rotation are out of scope (ticket states they are separate CEO decisions).

## 3. Design

Add a small `io.Writer` wrapper that masks credentials in each written chunk, and give it to both gin middlewares as their output.

- New file `bin-api-manager/lib/logmask/logmask.go`, package `logmask`.
  - `NewWriter(w io.Writer) io.Writer`.
  - `Write(p)` applies the replacements below, writes the result to the inner writer, and returns `len(p), err` (the original length, so the caller never sees a short write).
  - Both gin sinks emit one `Write` per log record (`log.Logger.Printf`, and the access logger's `fmt.Fprint(out, ...)`), so a credential is never split across writes. This is verified by a test that uses the real gin middlewares end to end.
- Replacements (regexes compiled once, package level), applied with callbacks (Revision 1, review round 1):
  1. Query parameters: match every `[?&]key=value` pair with `([?&])((?:[^=&\s"\\]|\\.)*)=((?:[^&\s"\\]|\\.)*)`. The callback percent-decodes the key with `url.QueryUnescape` and, if the decoded key is exactly `accesskey` or `token`, replaces the value with `***`. Decoding the key is required because Go `URL.Query()` decodes keys, so `?%61ccesskey=SECRET` authenticates but a literal-name regex would not match it (bypass found in review). The key text is kept as written, only the value is masked. The value stops at `&`, whitespace or an unescaped `"`, so the quoted `%#v` format keeps its closing quote and other parameters are untouched. Repeated keys (`token=A&token=B`) are all masked.
  2. Cookie header lines in the recovery dump: match `(?m)^Cookie:[^\r\n]*`; the callback splits the value on `;`, and for each `name=value` whose trimmed name is exactly `accesskey` or `token` (cookie names are case-sensitive in `c.Cookie`) replaces the value with `***`. Scoped to the `Cookie:` header line only (gin dump prints the canonical `Cookie`).
- Wiring in `runListenHTTP`:
  - `gin.LoggerWithConfig(gin.LoggerConfig{Output: logmask.NewWriter(gin.DefaultWriter), SkipPaths: ...})`
  - `gin.RecoveryWithWriter(logmask.NewWriter(gin.DefaultErrorWriter))`
  - `gin.DefaultWriter` / `gin.DefaultErrorWriter` are the same sinks `gin.Default()` uses (stdout / stderr), so destination does not change.
- `Middlewares` falls back to `gin.DefaultWriter` / `gin.DefaultErrorWriter` when `out` / `errOut` is nil (`RecoveryWithWriter(nil)` would drop logs). It keeps `SkipPaths` and the order Logger, Recovery, then `RequestID` (added by the caller as today).
- Gin mode dependence (review round 1): in gin 1.11.0 `recovery.go` prints the request dump (`secureRequestDump`) only when `IsDebugging()` or on a broken pipe. Production does not set `GIN_MODE`/`gin.SetMode` anywhere in the repo, so the default debug mode applies and the dump is emitted. Release mode would not print the dump at all (no leak). The broken-pipe path uses the same `secureRequestDump`, so the same writer covers it.
- Side effect: gin disables colored output when `Output` is not a terminal file. In containers stdout is not a TTY already, so the output is unchanged in production.

Why a writer and not a custom formatter or custom recovery: a formatter would re-implement the unexported default format (format drift risk) and cannot cover the recovery dump. A masking writer keeps the exact gin formats ("other query parameters and log format stay the same") and covers both sinks with one component.

## 4. Non-goals

- Query auth removal (option B), accesskey rotation, scrubbing logs already stored (Loki retention).
- Other log lines emitted by application code through logrus (separate audit scope; `voipbin-log-credential-exposure-audit`).

## 5. Tests (written together with the implementation)

Unit (`logmask_test.go`, table driven):
- masks `accesskey` and `token` in an access-log style line, key name kept, other params kept;
- first, middle and last parameter positions; escaped-quote value; empty value;
- `Cookie: a=b; token=X; accesskey=Y` and `Cookie: token=X`;
- does not touch `access_token=`, `xtoken=`, plain text without secrets;
- returns the original length and propagates inner writer errors.

Test conventions: names use the `Test_` prefix (`Test_Write`, `Test_Middlewares_AccessLog`, `Test_Middlewares_Recovery`), no gomock. Additional unit cases from review: percent-encoded keys (`%61ccesskey`, `tok%65n`), repeated key, `%22` value, `?token` without value, cookie header scoping (`Cookie:` line only, other header lines untouched), `access_token` and `xtoken` untouched.

Integration (same package or `cmd`-level helper): real `gin.New()` with the same wiring:
- request `GET /v1.0/x?accesskey=SECRET&foo=bar` -> captured access log contains `accesskey=***` and `foo=bar`, not `SECRET`;
- handler that panics, request with `?token=SECRET` and `Cookie: accesskey=SECRET2` -> captured recovery dump contains neither `SECRET` nor `SECRET2`.
The integration test sets `gin.SetMode(gin.DebugMode)` and restores the previous mode in `t.Cleanup`, does not use `t.Parallel()` (global state), and positively asserts the dump was printed (output contains `Cookie:` and `/v1.0/x?token=***`) so it cannot pass vacuously.
Negative control (acceptance criterion 3): `Middlewares` takes a package-private option so a test can build the same wiring with masking disabled; a test case asserts that SECRET DOES appear in that unmasked output. This proves the assertions fail without the fix. The negative control is also run once by hand against pre-change wiring and recorded in the PR.

To keep `main.go` testable, the engine wiring of the two middlewares goes through a tiny helper `logmask.Middlewares(skipPaths []string, out, errOut io.Writer) []gin.HandlerFunc` used by `runListenHTTP` and by the integration test, so the test exercises the production wiring rather than a copy.

Docs: add `lib/logmask` to the package table in `bin-api-manager/docs/architecture.md`.

## 6. Risks

- Limits (documented): `ErrorMessage` text appended by gin from `c.Errors` is masked only when it contains the `?key=value` form; application logrus lines are out of scope.

- Regex cost per log line: two precompiled regexes on short lines, negligible against request handling.
- A credential in an unexpected position (for example form body) is not in scope; gin does not log bodies.
- Header name variants other than `Cookie` are not masked; gin dump prints canonical `Cookie:`.

## 7. Rollout

Single PR in the monorepo, api-manager only, no schema or API change. Verify on deploy by sending a query-style request and grepping the container log.

## 8. Revision 2 (design review round 2 notes, implemented)

- The query value regex runs to `&` or whitespace (`[^&\s]*`) and a single trailing `"` is restored, so both the Go-quoted access log line (closing quote kept) and the raw recovery dump (a raw `"` inside a value) are fully masked.
- The query rule applies to the whole written chunk, so `Referer: http://h/p?accesskey=..` in the dump is masked too (intended). The cookie rule applies only to the `Cookie:` line.
- `Middlewares(skipPaths, out, errOut)` is the public API; the package-private `middlewares(..., mask bool)` is the negative-control seam used by tests.
- Limits: panic values and `c.Errors` text are masked only when they contain the `?key=value` form.

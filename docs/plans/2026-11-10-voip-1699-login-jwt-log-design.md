# VOIP-1699 bin-api-manager: stop logging the login JWT

## 1. Ticket review (adversarial, against origin/main 774241e88)
Valid. `bin-api-manager/lib/service/auth.go` `PostLogin` logs the whole JWT with
`log.Debugf("Created token string. token: %v", token)` right after `AuthLogin`
succeeds. The same token is returned in the cookie and the response body. The
service sets the logrus level to Debug unconditionally (`internal/config`), so
the line is always emitted. Not fixed or duplicated on main.
Checked: the type of `a` in `AuthLogin` is `*amagent.Agent` (pointer), so the
`Generating the token. data: %v` line in `AuthJWTGenerate` prints an address,
not the agent value. It is not changed here.

## 2. Change (bin-api-manager only, log output only)
- `lib/service/auth.go`: delete the `Created token string` line. The existing
  `User successfully logged in.` line already records that a token was issued,
  so no information is lost. No derived value (length, prefix) is added.

## 3. Compatibility
One Debug log line removed. No API, DB, RPC, event or cookie change.
Previously written logs were not checked.

## 4. Verification plan
- Test first (RED on current code). Reuse the `gin.CreateTestContext` + mock
  `ServiceHandler` + `setupServer` pattern of `TestPostLogin_AuthFailed`; no
  `t.Parallel()` (global logger state).
- Global logger setup saved and restored with `t.Cleanup`: level (set to Debug),
  hooks (`ReplaceHooks`), formatter (production joonix), output (`io.Discard`).
  The hook returns `logrus.AllLevels` and is mutex-guarded.
- Mocked token is a neutral sentinel (not JWT-shaped, not a secret-like
  constant name). Cases: AuthLogin success, and AuthLogin failure where the mock returns
  the sentinel together with the error (so a leak on the error path is caught).
- For every captured entry assert the sentinel is absent from `entry.Message`,
  `entry.Data` (`%v` as the primary check; `json.Marshal` only when it returns
  no error, since func values can make it fail) and the joonix `Format` output, in
  plaintext and as `fmt.Sprint([]byte(sentinel))`. `Format` must return a nil
  error and non-empty output (an ignored error would make the check vacuous), and the
  output must contain a non-sensitive field such as `username`.
- Positive controls: success case captures exactly one `User successfully
  logged in.` entry; failure case captures `Login failed.` and no success entry.
  The response body and cookie do contain the sentinel (token still returned).
- Mutation check in a scratch copy only: restore the deleted line, and also
  variants `log.Debug(token)` and `WithField("token", token)`, and a `Warnf` level variant; each must fail
  the test, and the failing assertion is recorded.
- Repo verification workflow for bin-api-manager.
- Known unrelated finding, not changed: `PostLogin` passes
  `"request_address": c.ClientIP` (method value, not a call); joonix silently drops func-valued fields, so
  `request_address` is not in the log output. Reported only.

## 5. Not covered (stated in the PR)
The test covers `PostLogin` only. Out of scope: the `WithField("agent", a)`
Debug dump in `servicehandler.AuthLogin` (includes `direct_hash`), other
services' request/response dumps, and logs already written.

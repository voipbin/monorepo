# AMI Redirect request URI fix (VOIP-1580)

## 1. Ticket review
Valid. `AstAMIRedirect` sends URI `/ami`, but the asterisk proxy routes AMI requests with `^/ami/`. The request falls to the default branch, which returns 404 with an empty body, and the empty body is unmarshaled as JSON, so the caller only sees `unexpected end of JSON input`. The existing unit test pinned `/ami` against a mocked success response, so it hid the mismatch.

## 2. Design
- Sender URI: `/ami` -> `/ami/action` (proxy unchanged; its handler ignores the sub path and its tests already use `/ami/action`).
- `AstAMIRedirect` returns an error that names the status code when the response status is not 200 (including 0), before parsing the body. The message holds the status code only, never the response body. `Response != Success` keeps going through the existing `parseAMIResponseData`. Scope is this function only.
- `voip-asterisk-proxy/README.md` example URI is corrected to `/ami/action`.
- Other `Ast*` AMI-style callers are not changed.
- Alternative rejected: relaxing the proxy regex to `^/ami(/|$)`. It needs a proxy redeploy and leaves it unclear whether `/ami` or `/ami/action` is the contract; the proxy tests and README already use `/ami/action`.

## 3. Detailed design
Files:
- `bin-common-handler/pkg/requesthandler/ast_ami.go`: URI constant change, status check (`fmt.Errorf("... status code: %d", code)`).
- `bin-common-handler/pkg/requesthandler/ast_ami_test.go`: expected URI `/ami/action`; new cases for status 404, 400 and 0 (error), and the existing `Response: Error` body case.
- `voip-asterisk-proxy/pkg/listenhandler/ami_redirect_contract_test.go` (new): runs the real `requesthandler.AstAMIRedirect` against the real `processRequest` and a fake AMI server, so the sender URI and the proxy route cannot drift apart again. `NewRequestHandler` registers Prometheus metrics and may run once per process, so it is created once per package through `sync.OnceValue` with a stub `SockHandler` (embeds the interface, overrides `RequestPublish` only) that forwards to a per-test `listenHandler` swapped under a mutex. No gomock controller is bound to the shared handler. Run with `-race`.
- `voip-asterisk-proxy/README.md`: example URI.
- `voip-asterisk-proxy` already replaces and imports `bin-common-handler`; no go.mod change.

Compatibility: proxies built before 2025-03-25 matched only `/ami` exactly; later proxies match only URIs starting with `/ami/`. The new sender therefore works with every proxy build from 2025-03-25 on and not with older builds. Assumption (not verifiable here): the bug is reproduced in operation, so every AMI target already runs a post-2025-03-25 proxy; older proxy builds must be confirmed upgraded before this ships.

## 4. Verification plan
- RED before the fix: the contract test fails with `unexpected end of JSON input` (summarized in the PR description).
- GREEN after the fix; mutation: revert the URI to `/ami` and the contract test and the unit test must fail; remove the status check and the 404/400/0 cases must fail; error text must contain the status code (asserted). Mutations run in a scratch copy, and the unit test and the contract test are recorded separately for the URI mutation.
- `go test -count=3` in both modules (metrics once-guard), full verification workflow in `bin-common-handler` and `voip-asterisk-proxy`, `go build ./...` and `go vet ./...` of `bin-call-manager`; `-race` for the contract test.

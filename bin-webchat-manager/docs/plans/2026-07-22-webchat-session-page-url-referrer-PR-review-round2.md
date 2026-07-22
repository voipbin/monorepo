# Webchat session page_url capture — PR review (round 2)

**Scope**: Third independent review, per the repo's 3-round hard-rule floor.
Round 0 was APPROVED. Round 1 found and reported a real, reachable XSS gap
(`javascript:`/`data:` scheme values in `page_url` rendered as a clickable
`<a href>` in `message_timeline.js`, reachable via the `IsDirect()` auth
branch by any HTTP caller holding a widget's public direct-hash — not just
the bundled `client.js`). Round 1 ended `CHANGES_REQUESTED`. Two fix commits
landed in response:

- `bin-api-manager` commit `0d818afa1` — scheme allowlist added to
  `validatePageURL`.
- `square-admin` commit `5ca2e226` — `isSafePageURL` render-time guard added
  to `message_timeline.js`.

This round verifies those two commits directly (`git show`), re-runs the
actual test/lint/build gates rather than trusting the PR description, and
spot-checks that the fix introduced no regression in the areas rounds 0/1
already cleared.

Because round 1 was `CHANGES_REQUESTED`, an `APPROVED` verdict this round
does **not** yet satisfy the "2 consecutive APPROVED" exit condition — per
the task brief, at least one more round (round 3) is required after this one
if this round approves.

Commits reviewed (unchanged commit set from round 1, plus the two fixes):
- Go (monorepo, branch `NOJIRA-webchat-session-referrer-page-url`):
  `0def6eaa6`, `80b24dd5a`, `246376749`, `74f943154`, `6d691244f`, `0d818afa1`
- JS (monorepo-javascript, same branch): `67e9df222`, `5ca2e226`

---

## 1. Scheme allowlist correctness — `bin-api-manager` `validatePageURL` (commit `0d818afa1`)

Read the actual diff via `git show 0d818afa1` (not the commit message —
verified the code):

```go
func validatePageURL(pageURL string) error {
	if len(pageURL) > 2048 {
		return fmt.Errorf("%w: page_url exceeds maximum length of 2048 characters", serviceerrors.ErrInvalidArgument)
	}
	if pageURL == "" {
		return nil
	}
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		return fmt.Errorf("%w: page_url must use the http or https scheme", serviceerrors.ErrInvalidArgument)
	}
	return nil
}
```

Traced the control flow by hand against the concrete attack strings from
round 1's finding:
- `"javascript:alert(1)"` → fails both `HasPrefix` checks → rejected. ✅
- `"data:text/html,x"` → fails both → rejected. ✅
- `"ftp://x"` → fails both (added as an extra scheme in the test suite,
  confirming the allowlist is a positive allowlist, not merely a
  `javascript:`/`data:` blocklist) → rejected. ✅
- `"http://x"` / `"https://x"` → pass → accepted. ✅
- `""` (empty) → short-circuits on the dedicated `pageURL == ""` branch
  *before* the scheme check runs, so the optional-field contract from round
  0/1 (empty is always valid) is preserved exactly. ✅

This is a genuine allowlist (`HasPrefix("http://") || HasPrefix("https://")`),
not a blocklist of a few known-bad schemes — so it also closes off schemes
round 1 didn't explicitly enumerate (`vbscript:`, `file:`, protocol-relative
`//evil.example`, bare `evil.example` with no scheme at all, etc.), all of
which fail both `HasPrefix` checks and get rejected. This is stronger than
the minimum fix round 1 asked for.

**Verdict for this angle: correct and complete.** The allowlist matches
exactly what round 1 recommended (option (a), the preferred fix), placed in
the single pre-branch call site already verified in round 1 §7 to be
unconditional and un-bypassable by any of the three auth branches.

## 2. Backend test coverage — `Test_validatePageURL` (same commit)

Read `bin-api-manager/pkg/servicehandler/webchat_session_test.go` in full
(not just the diff hunk) to see the complete table, including pre-existing
cases carried over from round 0:

| Case | Input | Expect error | Covers |
|---|---|---|---|
| empty is valid | `""` | no | optional-field / early-return branch |
| normal url is valid | `https://example.com/pricing` | no | baseline happy path |
| exactly 2048 chars is valid | 2048-char `https://...` string | no | length boundary, upper-inclusive |
| 2049 chars is invalid | 2049 `a`s (no scheme at all) | yes | length boundary, over |
| javascript scheme is invalid | `javascript:alert(1)` | yes | the exact round-1 PoC string |
| data scheme is invalid | `data:text/html,x` | yes | the exact round-1 PoC string |
| ftp scheme is invalid | `ftp://x` | yes | proves allowlist, not blocklist |
| http scheme is valid | `http://x` | no | non-TLS customer sites, per the code comment's own stated rationale |
| https scheme is valid | `https://x` | no | baseline TLS case |

Ran it directly rather than trusting the PR: `go test ./pkg/servicehandler/...
-run Test_validatePageURL -v` — all 9 subtests pass
(`--- PASS: Test_validatePageURL` and every subtest, `ok` for the package).

One gap worth naming honestly (not blocking): there is no explicit test for
a value that is simultaneously oversized (>2048 chars) **and** uses a bad
scheme (e.g. `"javascript:" + strings.Repeat("a", 3000)`), to confirm which
error fires first / that the length check still runs before the scheme
check when both would fail. Read the code: length is checked unconditionally
first (`if len(pageURL) > 2048 { ... return }` is the very first statement,
before the empty-check or the scheme-check), so this combination is provably
correct by inspection even without a dedicated table row — but it is an
untested code path combination. Not a defect; noting for completeness since
the task explicitly asked about "2048자 경계/스킴 위반 조합."

**Verdict for this angle: adequate.** All individually-relevant boundaries
and both PoC strings from round 1 are exercised and pass; the one untested
*combination* is provably safe by static reading of the guard ordering, not
just assumed.

## 3. Frontend guard correctness — `square-admin` `isSafePageURL` (commit `5ca2e226`)

Read `git show 5ca2e226` directly:

```js
const isSafePageURL = (url) => typeof url === 'string' && (url.startsWith('http://') || url.startsWith('https://'))
```

used at the render site:

```jsx
{session?.page_url && isSafePageURL(session.page_url) ? (
  <a href={session.page_url} target="_blank" rel="noopener noreferrer" ...>
    {truncatePageURL(session.page_url)}
  </a>
) : (
  <span>Unknown</span>
)}
```

Traced: if `session.page_url` is truthy but fails `isSafePageURL` (e.g. a
`javascript:` string that somehow reached storage before this fix existed,
or via a path this review hasn't found), the ternary falls to the `Unknown`
branch — the anchor is never constructed, so `href` is never assigned an
unsafe value. This is render-time gating, not sanitization of the string
itself (the raw string is still shown via `truncatePageURL` inside the
anchor when safe, and is simply not shown/linked at all when unsafe — the
component chooses "hide" over "escape," which is the safer of the two for
an XSS-classed value with no legitimate use for the unsafe branch).

`isSafePageURL` uses `startsWith`, exactly mirroring the Go side's
`strings.HasPrefix` — both are case-sensitive exact-prefix checks with
identical semantics (see §4 below for the consistency check this enables).

**Test added**: `git show 5ca2e226` includes a new Jest case, "renders
'Unknown' (not a clickable link) when session.page_url has a javascript:
scheme," which renders the component with
`page_url: 'javascript:alert(1)'` and asserts both
`screen.getByText(/^Unknown$/)` is present AND
`screen.queryByRole('link')` is absent — this is a real regression guard: it
would fail if either isSafePageURL were removed or if the ternary condition
were later refactored to only check `session?.page_url` truthiness again.
Ran it for real: `npm test -- --watchAll=false --testPathPattern=message_timeline`
→ `7 passed, 7 total`, including this new test and all 6 pre-existing ones
(happy path with link, truncation, absent-URL "Unknown", dialog-closed,
empty-messages, session-scoped fetch).

**Verdict for this angle: correct and test-backed.** The frontend guard is a
faithful mirror of the backend allowlist and is exercised by a real,
passing, non-trivial assertion (checks for absence of `role="link"`, not
just presence of text).

## 4. Backend/frontend consistency audit

Directly diffed the two predicate implementations side by side:

- Go: `strings.HasPrefix(pageURL, "http://") || strings.HasPrefix(pageURL, "https://")` → negated and OR'd (reject if neither).
- JS: `url.startsWith('http://') || url.startsWith('https://')`

Both are:
- **Case-sensitive** on the scheme token. Neither Go's `HasPrefix` nor JS's
  `startsWith` is case-insensitive by default, and neither implementation
  lowercases the input first. This means `"HTTP://example.com"` or
  `"Https://example.com"` is rejected by *both* layers identically — not a
  bypass, and not an inconsistency (a mismatch would only be a problem if
  one layer accepted mixed-case scheme and the other didn't, e.g. backend
  stores a mixed-case-scheme URL that the frontend then silently drops from
  display). Verified this is symmetric: since the backend is the only write
  path (per round 1 §7, `WebchatSessionCreate` is the sole production call
  site and its `validatePageURL` gate is unconditional and pre-branch), any
  value that reaches storage already passed the backend's case-sensitive
  check, so the frontend's independently-case-sensitive check will also
  accept it — no legitimate value can be written by the backend and then
  spuriously hidden by the frontend guard. Confirmed by construction, not
  just inspection: both predicates accept exactly the same infinite set
  (strings starting with lowercase `http://` or `https://`) and reject
  everything else, so their agreement isn't coincidental — it's structural.
- **Exact-prefix**, not host/regex-based — neither implementation tries to
  parse the URL (no `url.Parse` / `new URL()`), so both are equally immune to
  known URL-parsing-based scheme bypasses (e.g. no `net/url.Parse` quirks to
  diverge on since neither side uses it here).
- **No normalization** (no trim, no case-fold) on either side before the
  check — consistent absence, not an inconsistency.

**Verdict for this angle: consistent.** No divergence found between the two
gates; the frontend guard is a structural mirror of the backend allowlist,
not an independently-designed check that could drift.

## 5. Regression check — legitimate values still work

Directly re-ran the real test suites (not just read the diffs) to confirm no
regression on the pre-existing "must still work" cases:

- Backend: `Test_validatePageURL/empty_is_valid` and
  `Test_validatePageURL/normal_url_is_valid` (both pre-existing, carried
  over unchanged from before this fix) still pass — confirmed via the full
  `go test ./pkg/servicehandler/... -run Test_validatePageURL -v` run, not
  just reading the table.
- Backend: full `go test ./...` in `bin-api-manager` — all packages `ok`,
  zero failures, zero regressions elsewhere in the service.
- Backend: `go test ./...` in `bin-webchat-manager` (the field-storage/RPC
  side untouched by this round's fix commits) — all packages `ok`,
  confirming the scheme allowlist change in `bin-api-manager` (a different
  service) didn't require or trigger any change in `bin-webchat-manager`,
  consistent with round 1's finding that `validatePageURL` lives entirely in
  `bin-api-manager`'s `servicehandler` layer.
- Frontend: `Test_...renders "Started from: <link>" when session.page_url is
  present` and `...truncates a long page_url...` (both pre-existing,
  positive-path tests using real `https://` URLs) still pass in the same
  test run that includes the new negative-path test — confirmed by the
  `7 passed, 7 total` result, not assumed from the diff.
- Frontend lint/build: `golangci-lint run -v --timeout 5m ./pkg/servicehandler/...`
  → `0 issues`. `npm run build` (production build, matching the project's
  actual CI gate — not the stricter `CI=true` invocation, which fails on
  pre-existing unrelated ESLint warnings in `TestAgentSheet.js` that exist on
  this branch independent of this PR and are not part of the webchat feature)
  → succeeds cleanly, "The build folder is ready to be deployed," with no
  errors or warnings referencing `message_timeline.js`, `webchat_widgets`, or
  `isSafePageURL`.

**Verdict for this angle: no regression.** Both the empty-field and
normal-https-URL cases — the two paths any real customer traffic actually
exercises — are unchanged and still pass.

## 6. Light re-check of round 0/1 items for side effects from this fix

The fix commits touch exactly two files with logic changes
(`bin-api-manager/pkg/servicehandler/webchat_session.go`,
`square-admin/src/views/webchat_widgets/message_timeline.js`) plus their two
test files — confirmed via `git show --stat` on both commits, no other files
touched. Cross-checked this against round 1's cleared angles:

- **RPC chain / `WebchatV1SessionCreate`** (round 1 §7): untouched by
  `0d818afa1` — the scheme check runs and returns before the RPC call is
  ever reached (same call-order guarantee round 1 already verified for the
  length check); no new bypass surface introduced since the check is
  additive to the same single gate, not a new parallel path.
- **Mock regeneration**: `validatePageURL` is a private, unexported function
  with no interface signature change (no new/changed method on
  `ServiceHandler`), so no mock regeneration was required or expected — confirmed
  `git show --stat 0d818afa1` lists only the handler file and its test file,
  no `mock_main.go` diff, which is correct.
- **OpenAPI contract**: `page_url`'s `maxLength: 2048` schema entry in
  `bin-openapi-manager` is untouched by either fix commit (neither commit
  touches `bin-openapi-manager` at all) — the scheme allowlist is an
  application-layer rule with no OpenAPI-expressible equivalent in this
  spec (no `format: uri` or `pattern` was added), which is a minor,
  non-blocking observation: the API contract still only documents the
  length cap, not the scheme restriction, so an external API consumer
  reading only the OpenAPI spec wouldn't discover the scheme rule until they
  hit the 400 response. Not a regression, and not something round 1 asked
  for — noting for completeness.
- **Cache layer / webhook conversion / SessionUpdate clobber checks**
  (round 1 §2–4): none of the changed files intersect
  `cachehandler`, `webhook.go`, or `dbhandler/session.go` — these areas are
  structurally unreachable from a two-file diff in `servicehandler` and
  `message_timeline.js`, so no re-verification finding anything different
  from round 1 is expected or found.

**Verdict for this angle: clean, no side effects.** The fix is narrowly
scoped to exactly the two points round 1 identified, with no incidental
changes elsewhere that would need re-litigating.

---

## Summary

| # | Angle | Result |
|---|-------|--------|
| 1 | Backend scheme allowlist correctness (`0d818afa1`) | Correct — true allowlist, closes `javascript:`/`data:`/`ftp:`/schemeless, preserves empty-is-valid |
| 2 | Backend test coverage | Adequate — both PoC strings, all boundaries, both valid schemes covered; length+scheme combo untested but provably safe by code order |
| 3 | Frontend guard correctness (`5ca2e226`) | Correct — render-time gate backed by a real, passing negative-path test asserting no `role="link"` |
| 4 | Backend/frontend consistency | Consistent — structurally identical case-sensitive exact-prefix predicates, no drift |
| 5 | Regression on legitimate values | None — empty and https-URL cases pass in both real test runs; full test suites, lint, and production build all pass |
| 6 | Side effects on round 0/1 cleared items | None — fix is narrowly scoped to exactly the two files round 1 identified |

**Verification performed this round (not merely re-reading prior review
text):**
- `git show 0d818afa1` and `git show 5ca2e226` read in full, not just the
  commit message.
- `go test ./pkg/servicehandler/... -run Test_validatePageURL -v` — 9/9
  subtests pass.
- `go test ./...` in both `bin-api-manager` and `bin-webchat-manager` — all
  packages pass, zero regressions.
- `golangci-lint run -v --timeout 5m ./pkg/servicehandler/...` — 0 issues.
- `npm test -- --watchAll=false --testPathPattern=message_timeline` — 7/7
  pass, including the new XSS regression test.
- `npm run build` — production build succeeds cleanly with no
  webchat/message_timeline-related warnings or errors (the stricter
  `CI=true` build fails only on pre-existing, unrelated warnings in
  `TestAgentSheet.js`, confirmed present on this branch independent of this
  PR's changed files).

Round 1's finding is fully and correctly addressed by both fix commits, with
no new defects found and no regressions in previously-cleared areas.
Per the task's process rule, this APPROVED verdict does **not** close the PR
review loop by itself — round 1 was CHANGES_REQUESTED, which reset the
"2 consecutive APPROVED" counter, so at least one further round (round 3)
is still required before merge eligibility.

---

## VERDICT: APPROVED

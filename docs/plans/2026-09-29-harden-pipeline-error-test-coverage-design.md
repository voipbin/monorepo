# VOIP-1543: Harden VOIP-1542 pipeline error test coverage

- Ticket: VOIP-1543 (relates to VOIP-1542)
- Worktree: `.worktrees/VOIP-1543-Harden-pipeline-error-test-coverage` (base: origin/main `6a5e7fc36`, which contains the merged VOIP-1542 squash)
- Status: §1 Analysis APPROVED (analysis loop closed: rounds 6 and 7 consecutive APPROVED). §2 Design APPROVED (design loop closed: rounds 1 and 2 consecutive APPROVED). Implemented; PR review loop in progress.

## 0. Decisions locked

- D1 (pchero, 2026-09-29): F10 -> **normalise**. ai-manager records an unrecognized or empty pipeline error category as `unknown`: the stored notice JSON `category` and the foreign-pipecatcall dedup key both use the normalised value. Recorded on VOIP-1543 as a comment. This is the only production change in this ticket.

## 1. Analysis

### 1.1 Origin

VOIP-1542 (PR #1350, merged `6a5e7fc36`) made pipecat-manager classify RTVI `error` frames and publish a `pipeline_error` event, and made ai-manager record it as a `role=notification` message on the aicall. Its round-3 fresh-context review was APPROVED, but a mutation sweep against the merged code showed that several invariants the design (§3.1, §4 of `docs/plans/2026-09-29-surface-llm-pipeline-errors-on-aicall-design.md`) promises are not pinned by any test: the mutation passes every test, or the test hangs instead of failing.

A first attempt at this follow-up (PR #1351, `NOJIRA-...`) skipped the ticket / analysis / design / PR review workflow and was closed unmerged. This ticket restarts the work under the workflow. The #1351 diff is kept only as a scratch reference; it is not an approved design.

### 1.2 Findings (evidence: mutation run on `6a5e7fc36`, `go test -count=1 -timeout 60s`)

Each row is one source mutation applied alone; "survives" means the package's tests still pass. (F5 was merged into F4 in round 2; the number is retired, not reused.)

| # | File:line (on main `6a5e7fc36`) | Mutation | Result | Why no test catches it |
|---|---|---|---|---|
| F1 | `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror.go:24` | remove `"api_key_invalid"` from tier-1 auth tokens | test **hangs** until the go test timeout (`panic: test timed out`) | `Test_receiveMessageFrameTypeMessage_error_publishesOnce` waits on `wg.Wait()` (`pipelineerror_test.go:184`) with no bound; the publish never happens so `wg.Done` is never called. Same unbounded pattern at `:211` and `:276`. |
| F2 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:24` | `pipelineErrorNoticeWindow` 10m -> 6m | survives | dedup fixtures only sit at -5m (inside both) and -11m (outside both) (`pipeline_error_test.go:253,260`). |
| F3 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:154` | remove the `m.TMCreate == nil ||` guard (`if m.TMCreate.Before(cutoff)`) | survives (would nil-panic in production on a row without `tm_create`) | no fixture row has a nil `TMCreate`. (Inverting the guard to `!= nil ||` is already killed by `Test_EventPMPipelineError_foreignPipecatcall`.) |
| F4 | `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror.go:24-37` | remove any single classifier token/phrase, or make the rate-limit regex never match | **16 of 24 survive** (full table in §1.2a) | each real test string also matches another token of the same category (or the token is not in any test string). Parent design §3.1 specifies every token as behaviour; §4 names OpenAI "Incorrect API key", ElevenLabs/Deepgram 401 bodies, and "429 Too Many Requests" as test items. |
| F6 | `pipelineerror.go:29` (and `:31`) | drop the trailing `\b` from the status/code regexes | survives | no negative case such as `{"status": 4031}`. |
| F7 | `pipelineerror.go:51,54` | `HasPrefix` -> `Contains` for the function_call / internal prefixes | survives | no case where the phrase appears mid-string. |
| F8 | `pipelineerror.go:50-83` | reorder the classifier checks (all 11 reorders swept in analysis round 3 survive) | survives | no test string matches two tiers that yield DIFFERENT categories (the incident string matches tier 1 and tier 3 but both give authentication). VOIP-1542 design §3.1 specifies "tiers evaluated in order, first hit wins". |
| F9 | `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:690` | error-response `Warnf` -> `Debugf` | survives | `Test_receiveMessageFrameTypeMessage_errorResponse` asserts only the metric and no-publish; VOIP-1542 design §4 lists "error-response frame -> WARN + metric, no event". |
| F11 | `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:679` | pass `false` instead of `msg.Data.Fatal` to `runnerHandlePipelineError` | survives | every pipecat-side test sends `fatal:false`. The customer doc (`ai_struct_message.rst:117`) promises that for voice calls `unknown` is reported only when the error stops the session, which depends on `fatal` reaching `shouldNotifyPipelineError`. |
| F12 | `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:~732` (event literal) | `Fatal: fatal` -> `Fatal: false` | survives | same as F11; parent design §3.1 puts `Fatal` in the payload and §4 requires the routing fields to be asserted. Only the ai-manager side pins `fatal` ("unknown fatal" row). |
| F13 | `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror.go:12` | `pipelineErrorLogMaxLen` 2000 -> 20 | survives | parent design §3.1 specifies WARN with text truncated to 2000 chars; `Test_truncateForLog` tests the helper with its own `maxLen`, nothing pins the constant used by the handler. |
| F14 | `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:711` | metric label `strconv.FormatBool(fatal)` -> `FormatBool(false)` | survives | parent §3.1 specifies `pipecat_manager_pipeline_error_total{category,fatal}`; all tests use `fatal=false`. Same root cause as F11/F12. |
| F15 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:74` | unknown-sentence fallback `text = ""` | survives | no test sends an unrecognized category. Needed regardless of D1: it pins the customer sentence for that path. |
| F16 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:78` | nil-`reqHandler` guard `if h.reqHandler != nil` -> `if true` | survives | all tests construct the handler with a mock `reqHandler`; parent §3.2 says to keep this guard. |
| F17 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:28` | `pipelineErrorNoticeScanSize` 50 -> 5 | survives | the test asserts `uint64(pipelineErrorNoticeScanSize)` (`pipeline_error_test.go:326`), which is circular; parent §3.2 fixes the size at 50. |
| F10 | `bin-ai-manager/pkg/messagehandler/pipeline_error.go:70-75,96` | (not a mutation: a behaviour observation) | n/a | an unrecognized or empty `evt.Category` gets the `unknown` sentence, but the raw category is stored in the notice JSON (`Category: evt.Category`, `:96`) and used for dedup (`:85`). The customer-facing doc `bin-api-manager/docsdev/source/ai_struct_message.rst` documents a 4-value enum. Unreachable today: pipecat-manager only publishes authentication / rate_limited / timeout / unknown (`shouldNotifyPipelineError` returns false for function_call / internal). |

### 1.2a Classifier token enumeration (F4)

Mechanical sweep on `6a5e7fc36`: every string literal in the `pipelineErrorTokens*` / `pipelineErrorPhrases*` lists removed one at a time, plus each regex forced to never match; package test run with `-timeout 60s`.

| Tier | Token / regex | Result |
|---|---|---|
| 1 auth | `api_key_invalid` | killed (via hang, F1) |
| 1 auth | `invalid_api_key`, `permission_denied`, `unauthenticated` | survive |
| 1 rate | `resource_exhausted` | killed |
| 1 rate | `rate_limit_exceeded` | survives |
| 2 auth | `http 401`, auth status/code regex | killed |
| 2 auth | `401 unauthorized`, `403 forbidden`, `http 403` | survive |
| 2 rate | `429 too many requests`, rate status/code regex | survive |
| 3 auth | `api key not valid`, `invalid api key`, `incorrect api key`, `invalid authentication` | survive |
| 3 rate | `rate limit`, `exceeded your current quota`, `quota exceeded` | survive |
| 3 timeout | all 4 timeout literals | killed |

Total: 24 matchers (22 string literals + 2 regexes), 8 killed, 16 survive. Rule adopted for the fix: **every classifier matcher gets at least one isolating test string that matches only that matcher** (no other token, phrase, or regex of any tier), so removing it changes the result to `unknown`.

### 1.2b Classifier check order (F8)

`classifyPipelineError` runs 9 checks in this order (`pipelineerror.go:50-83`), each yielding a category:

| # | Check | Category |
|---|---|---|
| C1 | tier 0 prefix `error executing function call [` | function_call |
| C2 | tier 0 prefix `invalid rtvi transport message` | internal |
| C3 | tier 1 auth tokens | authentication |
| C4 | tier 1 rate tokens | rate_limited |
| C5 | tier 2 auth phrases + regex | authentication |
| C6 | tier 2 rate phrase + regex | rate_limited |
| C7 | tier 3 auth phrases | authentication |
| C8 | tier 3 rate phrases | rate_limited |
| C9 | tier 3 timeout phrases | timeout |

A reorder is observable only if it inverts a pair of checks that (a) can both match one string and (b) yield different categories. Reordering two checks with the same category (C3/C5/C7, C4/C6/C8) is an equivalent mutant with no observable effect and needs no test. C1 and C2 can never both match (two different prefixes), so their order needs no test either. Any behaviour-changing reorder of these 9 checks must invert at least one remaining pair, so pinning every such pair is complete **at the 9-check level**. C5 and C6 are composites (phrase list OR regex, `pipelineerror.go:67,70`); to also catch a change that splits one of them (e.g. moves the auth regex after the rate phrase), the C5/C6 pair is pinned with two strings: auth phrase + rate regex (`401 unauthorized status 429`) and auth regex + rate phrase (`status 401 429 too many requests`), both expected `authentication`.

Pairs to pin: 36 total pairs, minus 6 same-category pairs, minus C1/C2 = **29 pairs** (30 test strings, since C5/C6 gets two). Rule adopted for the fix: **for every one of the 29 pairs (Ci before Cj), one test string that matches both and must classify as Ci's category.** Every pair is constructible: a tier-0 prefix can be followed by any other matcher's text, and the remaining matchers are substrings/regexes that can be concatenated.

Also verified killed on `6a5e7fc36` in the same sweep (out of scope): dropping the `deleted=false` dedup filter, changing `NotificationTypePipelineError`.

Findings F3 (corrected), F11-F13 were added after analysis review round 1 and each re-verified by the author with the same mutation method (all survive on `6a5e7fc36`).

Mutations that were already killed on `6a5e7fc36` (publish gate, WARN-once gate, category dispatch in ai-manager's `processEvent`, window inversion, foreign re-check, SkipCache fallback, category/type compare, the 8 killed classifier matchers listed in §1.2a, notify policy, HasSTT, metric, non-aicall guard, WithPipecatcallID) are out of scope.

### 1.3 Classification

- F1-F4, F6-F9 and F11-F17 are **test-only** gaps: production code is correct on inspection, the tests do not pin it. Fix = add or tighten tests. No production change.
- F10 is a **production behaviour** change, decided by D1 (§0): ai-manager normalises an unrecognized category to `unknown` before storing and deduping. It only changes what is written for an input no current producer sends; it is a defence-in-depth / contract-hygiene change, not a bug fix.

### 1.4 Affected files

- `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror_test.go` (F1, F4, F6-F9, F11-F14)
- `bin-ai-manager/pkg/messagehandler/pipeline_error_test.go` (F2, F3, F10, F15-F17)
- `bin-ai-manager/pkg/messagehandler/pipeline_error.go` (F10 only, per D1)
- No proto/model/API/RST/doc change is expected for F1-F4, F6-F9 and F11-F17. For F10 the RST already documents only the 4 values, so normalising makes the code match the doc; no doc edit expected. `docs/` service-doc sync: none of the CLAUDE.md "source change -> doc" triggers (listenhandler routing, subscribehandler targets, config, models, go.mod) are touched.

### 1.5 Recommended approach

1. F1: replace the three unbounded `wg.Wait()` calls with a small bounded-wait helper in the test file that fails the named test after 5s (well below the 60s package timeout, and generous for `-race` CI where the publish runs in a goroutine), so a regression fails fast with a clear message.
2. F4: add one isolating classifier case per surviving matcher in §1.2a (16 cases), each containing only that matcher, so removal flips it to `unknown`. Keep the existing realistic provider strings (they document real inputs) alongside.
3. F6: add negative cases `{"status": 4031}` -> unknown and `{"code": 4290}` -> unknown.
4. F7: add cases where the function_call / internal phrase is not at the start (must not classify as function_call / internal).
5. F8: add the 29 pair cases from §1.2b as a table-driven test (each string matches exactly the two checks named in its row, expected category = the earlier check's). This kills every behaviour-changing reorder, including tier 0 moved after tier 1 or 2, tier 1/2 blocks swapped, and rate-before-auth or timeout-before-rate within a tier.
6. F9: assert the error-response frame is logged at WARN using a logrus test hook (`logrustest.NewGlobal()`, already used in `Test_runnerHandlePipelineError_warnOncePerCategory`). The entry comes from `receiveMessageFrameTypeMessage`, not `runnerHandlePipelineError`, so match it by message text / the frame's `request_id`, not by `Data["func"]`. Set the logrus level to Debug for the test and restore it afterwards (same pattern as `pipelineerror_test.go:315-317`) so a WARN->DEBUG mutation is captured and judged on level, not silently dropped.
6a. F11, F12, F14: add a test with a voice session (`HasSTT=true`) receiving an `unknown`, `fatal:true` frame: it must publish exactly once, the published event must equal the full expected struct with `Fatal: true` (routing fields included), and `metricsPipelineErrorTotal{unknown,"true"}` must increase by exactly 1.
6b. F13: add a test that a frame longer than 2000 bytes is WARN-logged truncated to `pipelineErrorLogMaxLen`, pinning the constant through the handler path rather than only the helper. Setup: a voice session (`HasSTT=true`) and a non-fatal frame whose text classifies as `unknown`, so the notice policy publishes nothing (a bare `pipecatcallHandler{}` would otherwise hit a nil `notifyHandler` in the publish goroutine and crash the test binary). Text is ASCII with a sentinel: `strings.Repeat("a", 2000) + "b" + ...` (no quotes/backslashes, per `errorFrame`). Assert the WARN entry's `Message` (not the formatted entry: its `pipecatcall_id` UUID field can contain `b`) contains `strings.Repeat("a", 2000) + "...(truncated)"` and contains no `b`; ASCII keeps `truncateForLog`'s rune-boundary back-off out of play.
7. F2, F3: add a -9m same-category fixture (must be skipped) and a nil-`TMCreate` fixture (must be ignored, so a row is created).
7a. F16: add a test with an untyped nil request handler (`m.h.reqHandler = nil` on the interface field; a typed nil `*MockRequestHandler` would pass the `!= nil` guard and panic): no aicall lookup, no `req` expectations, notice row still created with nil active AI.
7b. F17: assert the `MessageList` size argument as the literal `uint64(50)`, not the constant.
8. F10, F15 (D1 locked: normalise): keep the single `!ok` branch of the notice-text lookup (`pipeline_error.go:70-75`) and make it set BOTH the category to `unknown` and the text to the `unknown` sentence; all later uses (dedup lookup `:85`, stored `Category` `:96`) read the normalised category. Normalising before the lookup would make the `!ok` branch dead and leave F15 unkillable, so the branch stays. Tests: (i) current-pipecatcall event with empty, `function_call`, and an unseen future value: stored content must be the exact `unknown` notice JSON (kills the F15 `text = ""` mutation and removal of the stored-category normalisation at `:96`); (ii) foreign-pipecatcall event with an empty and a `function_call` category while an `unknown` pipeline_error notice exists within the window: no row is created (pins the dedup half of D1 at `:85`, which (i) cannot catch). Deploy-skew note: if a newer pipecat-manager adds a category before ai-manager is upgraded, the older ai-manager stores it as `unknown`; intended contract behaviour (customers only ever see documented values).

Acceptance: every F1-F4, F6-F9 and F11-F17 mutation above, every matcher in §1.2a, every behaviour-changing reorder in §1.2b (each of the 29 pair inversions), and removal of the F10 normalisation at either use site (stored category `:96` or dedup key `:85`) fails a named test within the 60s package timeout; full verification workflow green in both services; no production change other than F10.

### 1.6 Risks

- Test-only changes cannot break production. The F10 change alters stored content only for inputs no current producer emits.
- Global logrus hook in the F9 and F13 tests (§1.5 items 6, 6b): the package tests do not use `t.Parallel()` (`grep -c "t.Parallel()" bin-pipecat-manager/pkg/pipecatcallhandler/*_test.go` -> 0 on `6a5e7fc36`), so a global hook does not race with other tests.
- Bounded wait: 5s per wait (see §1.5 item 1); generous for `-race` CI, far below the 60s package timeout.
- Global logrus level: tests that change `logrus.SetLevel` (F9, F13) must restore it (defer) to avoid leaking Debug level into later tests.

## 2. Design

All changes follow §1.5. File-by-file, in the order they will be implemented. No production change except §2.3.

### 2.1 `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror_test.go`

All new tests follow the repo test gate (`scripts/check-test-conventions.sh`): `Test_` prefix, gomock controller named `mc` (`mc := gomock.NewController(t)`), table tests as `tests := []struct{...}` + `t.Run`.

**2.1.1 Bounded wait helper (F1).** Add at the end of the file:

```go
// waitWithTimeout fails the test instead of hanging when an expected publish never happens
// (e.g. a classifier regression suppresses the event, so wg.Done is never called).
func waitWithTimeout(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for the expected PublishEvent call")
	}
}
```

Replace the three `wg.Wait()` calls (`:184`, `:211`, `:276`) with `waitWithTimeout(t, &wg)`. The helper's own goroutine may outlive a failed test; it only blocks on `wg`, holds no test state, and the package process exits after the run, so it cannot corrupt later tests.

**2.1.2 Isolating classifier cases (F4).** Append to the `Test_classifyPipelineError` table, grouped under a comment `// one isolating case per matcher: each string matches ONLY the named matcher (VOIP-1543)`:

| Name | Input | Expect |
|---|---|---|
| isolate tier1 invalid_api_key | `{'code': 'invalid_api_key'}` | authentication |
| isolate tier1 permission_denied | `status PERMISSION_DENIED` | authentication |
| isolate tier1 unauthenticated | `UNAUTHENTICATED: request had no credentials` | authentication |
| isolate tier1 rate_limit_exceeded | `{'code': 'rate_limit_exceeded'}` | rate_limited |
| isolate tier2 401 unauthorized | `server said 401 Unauthorized` | authentication |
| isolate tier2 403 forbidden | `server said 403 Forbidden` | authentication |
| isolate tier2 http 403 | `server rejected WebSocket connection: HTTP 403` | authentication |
| isolate tier2 429 reason phrase | `Unknown error occurred: 429 Too Many Requests.` | rate_limited |
| isolate tier2 rate regex | `{"status": 429}` | rate_limited |
| isolate tier3 api key not valid | `API key not valid. Please pass a valid API key.` | authentication |
| isolate tier3 invalid api key | `Invalid API key supplied` | authentication |
| isolate tier3 incorrect api key | `Incorrect API key provided: sk-xx` | authentication |
| isolate tier3 invalid authentication | `Invalid authentication credentials` | authentication |
| isolate tier3 rate limit | `slow down: rate limit reached` | rate_limited |
| isolate tier3 exceeded your current quota | `You exceeded your current quota.` | rate_limited |
| isolate tier3 quota exceeded | `Daily quota exceeded` | rate_limited |

Each string was chosen to avoid every other matcher: no underscore token of the other list, no `status`/`code` followed by 401/403/429, no `http 401/403`, no `401 unauthorized`/`403 forbidden`/`429 too many requests` unless that is the target, no `api key` family phrase unless it is the target, and no timeout literal. Pre-verified at design time with a throwaway probe test (not committed) that iterates the real matcher lists and regexes of `pipelineerror.go` on `6a5e7fc36`: each of the 16 strings hits exactly its one target matcher and classifies as expected; the 2.1.3/2.1.4 negatives hit zero matchers; each 2.1.5 fragment hits exactly its own check. §2.5 step 2 re-runs the same check at implementation time.

**2.1.3 Word-boundary negatives (F6).** Append: `{"status": 4031}` -> unknown; `{"code": 4290}` -> unknown.

**2.1.4 Prefix anchoring (F7).** Append: `LLM request failed while running error executing function call [x]` -> unknown; `Unknown error occurred: invalid rtvi transport message` -> unknown. (Neither matches any other matcher.)

**2.1.5 Check-order pairs (F8).** New table-driven test `Test_classifyPipelineError_checkOrder`, 30 rows (29 pairs, C5/C6 twice). Row strings are built by concatenating one fragment per check, from this fixed fragment set (each fragment matches exactly its own check):

| Check | Fragment | Category |
|---|---|---|
| C1 | `error executing function call [x] ` (must be at string start) | function_call |
| C2 | `invalid rtvi transport message ` (must be at string start) | internal |
| C3 | `invalid_api_key ` | authentication |
| C4 | `rate_limit_exceeded ` | rate_limited |
| C5 | `status 401 ` (regex) / `401 unauthorized ` (phrase) | authentication |
| C6 | `status 429 ` (regex) / `429 too many requests ` (phrase) | rate_limited |
| C7 | `invalid api key ` | authentication |
| C8 | `quota exceeded ` | rate_limited |
| C9 | `llm completion timeout ` | timeout |

For each pair (Ci, Cj) with i < j, different categories, and not {C1, C2}: input = fragment(Ci) + fragment(Cj) (tier-0 fragment always first because it is a prefix), expect = category(Ci). Pairs: C1 with C3..C9 (7), C2 with C3..C9 (7), C3 with C4, C6, C8, C9 (4), C4 with C5, C7, C9 (3), C5 with C6 (x2 strings: `401 unauthorized status 429`, `status 401 429 too many requests`), C8, C9 (3 pairs), C6 with C7, C9 (2), C7 with C8, C9 (2), C8 with C9 (1). Total 7+7+4+3+3+2+2+1 = 29 pairs, 30 rows. The rows are written out literally in the test (not generated), so a reader sees each expectation; each row's name is `Ci before Cj`.

For pairs that involve only one of C5/C6, use the regex fragment (`status 401 ` / `status 429 `); the round-1 design review probed all four regex/phrase combinations and every choice matches exactly the two named checks.

Note on fragment choice: `invalid api key ` (C7) does not contain `invalid_api_key` (C3) because of the space vs underscore; `quota exceeded ` (C8) is not a substring of any other fragment; `rate_limit_exceeded ` (C4) does not match C8 `rate limit` (underscore vs space).

**2.1.6 Error-response WARN (F9).** Extend `Test_receiveMessageFrameTypeMessage_errorResponse`: at the start, `hook := logrustest.NewGlobal()`, `defer hook.Reset()`, save `logrus.GetLevel()`, `logrus.SetLevel(logrus.DebugLevel)`, `defer logrus.SetLevel(orig)`. After the call, require exactly one entry whose `Message` contains `request_id: abc`, and require its `Level == logrus.WarnLevel`.

**2.1.7 Fatal voice error (F11, F12, F14).** New test `Test_receiveMessageFrameTypeMessage_error_fatalVoiceUnknownNotified`: session `newPipelineErrorTestSession(true)`; frame `errorFrame("Unknown error occurred: 500 Internal Server Error.", true)`; expect `PublishEvent(se.Ctx, message.EventTypePipelineError, &message.PipelineErrorEvent{...full routing fields..., Category: ErrorCategoryUnknown, Fatal: true})` exactly once (with `waitWithTimeout`); `metricsPipelineErrorTotal.WithLabelValues("unknown", "true")` delta == 1.

**2.1.8 Log truncation through the handler (F13).** New test `Test_runnerHandlePipelineError_truncatesLog`: `hook := logrustest.NewGlobal()` + level save/set/restore as in 2.1.6; handler `pipecatcallHandler{}`; session `newPipelineErrorTestSession(true)`; call `h.runnerHandlePipelineError(se, strings.Repeat("a", 2000)+"bbbb", false)` directly (unknown, non-fatal, voice: no publish, so the nil `notifyHandler` is never touched). Find the single entry with `Data["func"] == "runnerHandlePipelineError"`; require `Level == WarnLevel`, `strings.Contains(e.Message, strings.Repeat("a", 2000)+"...(truncated)")`, and `!strings.Contains(e.Message, "b")`.

### 2.2 `bin-ai-manager/pkg/messagehandler/pipeline_error_test.go`

**2.2.1 Window edge and nil tm_create (F2, F3).** Add two rows to `Test_EventPMPipelineError_foreignPipecatcall`: `same category near the window edge is skipped` (fixture at `now.Add(-9*time.Minute)`, expectCreate false); `row without tm_create is ignored` (row `{Role: notification, Content: {"type":"pipeline_error","category":"authentication"}}` with nil `TMCreate`, expectCreate true).

**2.2.2 Literal scan size (F17).** In the same test, change the `MessageList` expectation's size argument from `uint64(pipelineErrorNoticeScanSize)` to `uint64(50)`.

**2.2.3 Nil request handler (F16).** New test `Test_EventPMPipelineError_nilRequestHandler`: build mocks, then `m.h.reqHandler = nil`; `m.expectCreate(t)`; call with a current-pipecatcall authentication event; require the created row's `ActiveAIID == uuid.Nil` and correct `PipecatcallID`. No `m.req` expectations are set, so any call to it fails the test.

**2.2.4 Unrecognized category, stored value (F10, F15).** New table test `Test_EventPMPipelineError_unrecognizedCategory` with rows `empty` (`""`), `platform side category` (`ErrorCategoryFunctionCall`), `future category` (`"quota_billing"`): current pipecatcall; expected stored content exactly `{"type":"pipeline_error","category":"unknown","fatal":false,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"An AI service provider returned an error."}`.

**2.2.5 Unrecognized category, dedup key (F10 dedup half).** New table test `Test_EventPMPipelineError_unrecognizedCategoryDedup` with rows `empty` and `platform side category`: foreign pipecatcall (cached and SkipCache both return `testPEOtherPipecallID`), `TimeNow` returns `now`, `MessageList` returns one row `noticeRow(t, ErrorCategoryUnknown, now.Add(-1*time.Minute))`; `m.expectNoCreate()`.

### 2.3 `bin-ai-manager/pkg/messagehandler/pipeline_error.go` (D1, the only production change)

Current (`:70-75`):
```go
	text, ok := pipelineErrorNoticeText[evt.Category]
	if !ok {
		// function_call / internal are never published by pipecat-manager; anything else unknown
		// to this version is shown with the generic sentence.
		text = pipelineErrorNoticeText[pmmessage.ErrorCategoryUnknown]
	}
```
New:
```go
	category := evt.Category
	text, ok := pipelineErrorNoticeText[category]
	if !ok {
		// function_call / internal are never published by pipecat-manager; anything else unknown
		// to this version (including an empty category) is recorded as unknown, so the stored
		// category always stays within the documented enum.
		category = pmmessage.ErrorCategoryUnknown
		text = pipelineErrorNoticeText[category]
	}
```
Then `:85` `h.pipelineErrorNoticeRecent(ctx, ac.ID, evt.Category)` -> `..., category)` and `:96` `Category: evt.Category,` -> `Category: category,`. The log field `"category": evt.Category` at `:62` stays raw on purpose (operators should see what was actually received).

### 2.4 Docs

- No RST change: `ai_struct_message.rst` already documents only the 4 values; D1 makes the code match it.
- No service `docs/` change: none of the root CLAUDE.md doc triggers are touched (verified in §1.4).
- This design doc is committed with the code.

### 2.5 Verification plan (implementation phase)

1. Full CLAUDE.md workflow in both services (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`), plus `go test -race -count=1` on the two touched packages; `git status --short` clean except intended files.
2. Isolation checks for 2.1.2 and 2.1.5, done mechanically in a throwaway test (not committed): for each string, count how many of the 9 checks / 24 matchers match it; 2.1.2 strings must hit exactly their one matcher; 2.1.5 strings exactly their two checks.
3. Mutation sweep script (scratch, not committed; exact-match replace, `go test -count=1 -timeout 60s`, restore from saved content) covering: every F-row mutation in §1.2, every one of the 24 matcher removals, every one of the 29 §1.2b pair inversions (check Cj moved before Ci), the two C5/C6 composite splits, and both D1 use sites. Acceptance: all killed by a named test, none by hang.
4. Repo gates: `bash scripts/check-*.sh` (all four) after committing.

### 2.6 Risks

As §1.6. Additionally: the new pair and isolating tables add ~50 table rows to `Test_classifyPipelineError*`; future classifier changes must keep them in sync, which is the intended effect (a matcher change without a test change now fails).

## Analysis review log

- Round 1: CHANGES_REQUESTED. (1) MAJOR: `fatal` not pinned on the pipecat side -> added F11, F12 and §1.5 item 6a. (2) MINOR: F3 described the wrong mutation -> corrected (guard removal survives; inversion is killed). (3) MINOR: log truncation constant unpinned -> added F13 and §1.5 item 6b. (4) MINOR: F9 log assertion must match by message/request_id and set+restore level -> §1.5 item 6. (5) NIT: F10 deploy-skew note -> added to §1.5 item 8. (6) NIT: concrete bounded-wait timeout -> 5s in §1.5 item 1 and §1.6. All six re-verified by the author before applying (F11, F12, F13, F3b survive; F3a killed).
- Round 2: CHANGES_REQUESTED. (1) MAJOR: 16 of 22 classifier matchers unpinned, not "most killed" -> author re-ran a mechanical sweep of every matcher (§1.2a, identical 16 survivors), F4/F5 merged into F4 with an isolating-case rule, §1.5 item 2 and acceptance updated. (2) MINOR: metric `fatal` label -> F14, §1.5 6a. (3) MINOR: unknown-sentence fallback and nil-reqHandler guard -> F15, F16, §1.5 7a and 8. (4) MINOR: scan-size constant circular assertion -> F17, §1.5 7b. (5) NIT: F8 wording -> "two tiers that yield different categories". (6) NIT: §1.4 range -> F1-F9, F11-F17. Also locked D1 (F10 normalise, pchero) in §0; round 2 was briefed not to judge the F10 option, only its framing, so its verdict stands.
- Round 3: CHANGES_REQUESTED. (1) MAJOR: the proposed 4 tier-order cases did not kill most reorders (reviewer swept 11, all survive today, 6 still survive with the 4 cases) -> replaced with §1.2b: enumerate the 9 checks, pin all 29 pairs that can both match and yield different categories; same-category reorders and C1/C2 documented as unobservable; §1.5 item 5 and acceptance updated. (2) MINOR: matcher counts -> 24 matchers (22 literals + 2 regexes), 8 killed, 16 survive. (3) MINOR: normalising before the text lookup would make the F15 branch dead -> §1.5 item 8 keeps one `!ok` branch that sets both category and text, so F15 stays killable.
- Round 4: CHANGES_REQUESTED. (1) MAJOR: planned F10 test pinned only the stored category, not the dedup key (`pipeline_error.go:85`) -> §1.5 item 8 test (ii): foreign-pipecatcall `""`/`function_call` event with an in-window `unknown` notice must be skipped; acceptance names both use sites. (2) NIT: §1.2b completeness scoped to the 9 checks; C5/C6 composites pinned with phrase+regex and regex+phrase strings. Reviewer independently confirmed §1.2b (a)-(e): check list, unobservability argument, 29 pairs, completeness proof, and constructibility of all 29 pair strings.
- Round 5: CHANGES_REQUESTED. (1) MINOR: F13 test setup could crash the package (unknown frame in a no-STT session publishes via a nil notifyHandler) and the byte assertion assumed single-byte text -> §1.5 6b now uses a voice session with a non-fatal unknown frame (no publish) and an ASCII 2000x`a` + `b` sentinel, asserting the prefix + marker and absence of `b`. (2) NIT: F16 must use an untyped nil interface -> §1.5 7a. Reviewer confirmed test-interference and feasibility of the rest of §1.5 (Prometheus deltas, global logrus hooks, 5s wait with gomock, F11/F12/F14, F16 guard scope, F3, acceptance measurable).
- Round 6: **APPROVED** with 3 optional NITs, all applied: (1) F13 `b` check scoped to `entry.Message` (the entry's UUID field can contain `b`); (2) §1.6 logrus hook/level notes now name both F9 and F13 tests; (3) F-ranges exclude retired F5, and §1.2a now precedes §1.2b. Wording/ordering only; no finding, count, or approach changed.

## Design review log

- Round 1: **APPROVED**. Reviewer independently probed all 16 isolating strings, the 4 negatives, and all 29 pair strings under all four C5/C6 fragment variants against the real matcher lists (each hits exactly the intended checks), confirmed every mechanic in 2.1.6-2.2.5 and the §2.3 change. 2 optional NITs applied: C5/C6 fragment choice stated in 2.1.5; §2.5 step 3 mutation sweep now enumerates the 29 pair inversions to match the acceptance wording.
- Round 2: **APPROVED**. Reviewer checked repo test conventions, lint, and `-race -count=2` flakiness with a forced-mismatch probe (clean fail at 5s, no race), re-probed the classifier strings, and confirmed §2.5 proves the acceptance line. NIT 1 applied (convention note at §2.1). NIT 2 declined: `bin-ai-manager/docs/architecture.md:99` ("deduplicated per aicall and category") stays accurate after D1, and the service-doc gate does not cover this change; not worth widening scope. Design loop closed (rounds 1-2 consecutive APPROVED, min 2 met).

## Implementation verification (author, pre-PR)

- §2.5 step 3 mutation sweep on the implementation: 73 mutations (24 matcher removals, 29 pair inversions, 2 C5/C6 composite splits, every F-row mutation incl. F13 at 20 and 3000, both D1 use sites plus normalisation removal): **73 KILLED, 0 survived, 0 hangs** (F1: the former hang now fails `Test_receiveMessageFrameTypeMessage_error_publishesOnce` at 5.00s).
- Full CLAUDE.md workflow in both services: tidy/vendor/generate clean, `go test ./...` no failures, `golangci-lint` 0 issues; `go test -race -count=2` on both touched packages ok.

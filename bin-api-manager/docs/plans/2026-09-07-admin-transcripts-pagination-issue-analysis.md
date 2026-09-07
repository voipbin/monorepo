# VOIP-1480 Issue Analysis: admin GET /transcripts ignores page_size/page_token

Date: 2026-09-07
Ticket: https://voipbin.atlassian.net/browse/VOIP-1480
Stage: 이슈 확인/분석 (pre-design).

Revision history:
- r1: initial.
- r2: review round 1 (6 findings): `vn` CLI added as a broken first-party consumer;
  `page_size` named as a second behavioural change with affected callers; RST files that
  mention the endpoint listed with per-file reasoning; four line references corrected;
  ticket-summary discrepancy noted; square-admin path given as a separate repository.
- r3: review round 2 (3 findings): api-validator scenario reclassified as inert (omits the
  required `transcribe_id`, so its assertion never runs); `next_page_token` RST citation
  removed (it belonged to the `/transcribes` example); dbhandler line range aligned.
- r4: design review noted three more RST mentions of the endpoint; added to 2.4.
- r5: review round 4 (fresh reviewer): second square-admin consumer `transcribes_detail.js:103`
  added and the exhaustiveness sentence restated; RST enumeration completed to seven
  locations with per-file reasoning and a position on the "all transcripts" statements
  (a pagination note is added to the tutorial in the design); `:110` → `:109`.
- r6: review round 5: `voipbin-python-sdk` added (its `list_all()` cannot terminate
  today); the exhaustiveness sentence replaced by the pattern/scope statement; the
  quickstart cross-link claim corrected (`quickstart_realtime.rst` does not link to the
  tutorial, so the design adds a cross-reference there).
- r7: review round 7: api-validator STT poller added as an affected consumer (it paginates
  the admin endpoint from five live tests, duplicates every transcript today and trips its
  own circular-token guard); `test_stt_api.py:242-249` noted as inert; §3 cites the guard
  as in-repo corroboration; §4 distinguishes the api-validator scenarios.
- r8: after final approval (rounds 8 and 9): two editorial notes applied to the STT poller
  row (which call sites feed `combine_transcripts()`; the `:299` equality assertion).

## 1. Issue statement

The admin-surface `GET /v1.0/transcripts?transcribe_id=...` accepts `page_size` and
`page_token` (declared in the OpenAPI spec) but always returns the newest 100 transcript
lines regardless of what is passed, so a transcript longer than 100 lines cannot be read
to the end on that surface. Filed 2026-09-06 during SQUARE-55, where the frontend had to
route transcript reads through the agent surface because of this.

## 2. Code re-check (monorepo `origin/main` = `b41b11ea4`, 2026-09-07)

### 2.1 Where the parameters are dropped

- `bin-api-manager/server/transcripts.go:29-41` parses `pageSize` (default and clamp to
  100) and `pageToken` from `openapi_server.GetTranscriptsParams`, logs both at `:44`,
  then calls `h.serviceHandler.TranscriptList(c.Request.Context(), a, transcribeID)`
  (`:46`) without either value.
- `bin-api-manager/pkg/servicehandler/transcript.go:19` declares
  `TranscriptList(ctx, a, transcribeID)` with no size/token parameters and calls
  `h.reqHandler.TranscribeV1TranscriptList(ctx, "", 100, typedFilters)` (`:53`), i.e.
  token always empty (= "now" on the backend) and size always 100.
- The interface (`pkg/servicehandler/main.go:1163`) and the generated mock
  (`pkg/servicehandler/mock_main.go:6068,6077`) carry the same three-argument
  signature.
- `server/transcripts.go:53-58` still computes `next_page_token` from the last row's
  `tm_create`, so the response *advertises* a next page that the caller can never fetch.

### 2.2 The correct implementation already exists next to it

- Agent surface: `server/service_agents_transcripts.go:30-46` parses both parameters
  and calls `ServiceAgentTranscriptList(ctx, a, pageSize, pageToken, transcribeID)`;
  `pkg/servicehandler/serviceagent_transcript.go:24-75` takes `size uint64, token string`,
  defaults an empty token to `h.utilHandler.TimeGetCurTime()` (`:38-40`), checks
  permission on the fetched transcribe's `CustomerID`, and calls
  `TranscribeV1TranscriptList(ctx, token, size, typedFilters)` (`:62`).
- The admin transcribe list follows the same shape: `pkg/servicehandler/transcribe.go:82-84`
  defaults an empty token to `TimeGetCurTime()` before the RPC.
- Backend: `bin-common-handler/pkg/requesthandler/transcribe_transcripts.go:18-19` sends
  `page_token`/`page_size` to `/v1/transcripts`; `bin-transcribe-manager/pkg/dbhandler/transcript.go:154-166`
  also defaults an empty token to now (`:155-157`) and queries `tm_create < token ORDER BY tm_create DESC LIMIT size` (`:159-166`).
  So passing the real token/size through is sufficient; no backend change is needed.

### 2.3 Existing tests describe the bug, not the intent

- `server/transcripts_test.go:45,84`: the request carries `page_size=10&page_token=2020-09-20T03:23:20.995000Z`
  and the test declares `expectPageSize`/`expectPageToken` fields, but the mock
  expectation is `TranscriptList(req.Context(), tt.agent, tt.expectTranscribeID)`, so the
  two fields are never asserted (they are dead struct fields).
- `pkg/servicehandler/transcript_test.go:105`: expects
  `TranscribeV1TranscriptList(ctx, "", uint64(100), filters)` unconditionally.
- Contrast `server/service_agents_transcripts_test.go:105` and
  `pkg/servicehandler/serviceagent_transcript_test.go:109`, which assert the page
  size/token reach the service handler and the RPC.

### 2.4 Consumers (what changes for whom)

Two parameters are ignored today, so two behaviours change: a caller sending
`page_token` will get the older page it asked for instead of the newest page, and a
caller sending `page_size < 100` will get that many rows instead of 100.

| Consumer | Sends | Today | After the fix |
|---|---|---|---|
| `vn` CLI, separate repo `/home/pchero/gitvoipbin/cli/internal/commands/transcripts.go:44-71` (`--page-token`, `--page-size`, prints `Next page token`) | both | `--page-token` is silently ignored: the same newest page forever | works as documented |
| `voipbin-go` SDK, `gens/voipbin_client/gen.go:4863-4870` (`GetTranscriptsParams{PageSize, PageToken}`) | both (generated) | same | works |
| `voipbin-python-sdk`, separate repo `/home/pchero/gitvoipbin/newproject/voipbin-python-sdk`: `voipbin/resources/transcripts.py:6-9` (`endpoint = "transcripts"`, admin surface, `client.transcripts`), `voipbin/resources/base.py:29-51` `list(page_size, page_token)`, `:53-75` `list_all()` loops `while True` and breaks only when `next_page_token` is empty or `result` is empty | both | **`list_all()` never terminates** for any transcribe with at least one line: the page is always the newest one and `server/transcripts.go:53-58` always emits a non-empty token for a non-empty page, so neither break condition fires and `all_results` grows without bound | terminates: the pages advance until a short/empty page |
| `monorepo-monitoring/api-validator/tests/scenarios/stt/conftest.py:116-155` `TranscriptionPoller.get_transcripts()`: `while True` loop sending `page_token` (`:132-133`), accumulating `result` (`:143`), breaking on empty token/result (`:146-147`), with a circular-token guard that logs "Circular page_token detected" (`:150-153`); called from `test_stt_accuracy.py:256,321,398` (whose results feed `combine_transcripts()`, `:285-307`) and `test_stt_events.py:98,277` (`:299` asserts `len(transcript_events) == len(transcripts)`, today comparing a timeline count against a doubled transcript count) | `page_token` | **every transcript line is returned twice**: iteration 2 sends token `T`, the admin surface ignores it and returns the same newest page again (appended as duplicates) with the same `T`, and the circular guard stops the loop; the STT accuracy assertions run on duplicated text | pages correctly, each line once |
| `monorepo-monitoring/api-validator/tests/scenarios/generated/test_transcripts_generated.py:33-40` | `page_size=10` but **no `transcribe_id`** (required; the generated router returns 400 before the handler) | inert: the `len(result) <= 10` assertion never runs | still inert; would only become a real signal if the scenario were fixed (optional, separate) |
| `monorepo-monitoring/api-validator/tests/scenarios/stt/test_stt_api.py:242-249` | `page_size=10`, no `transcribe_id`, accepts 200/400/404, asserts nothing about length | inert | inert |
| square-admin `views/transcribes/transcripts_list.js:23` (separate repo `monorepo-javascript`), `page_size=100000`, no load-more | size only | reset to 100, newest 100 shown | unchanged (still 100; pagination is a frontend follow-up) |
| square-admin `views/transcribes/transcribes_detail.js:103` (same repo; live route `/resources/transcribes/transcribes_detail/:id`, `routes.js:407`), `page_size=1000`, no load-more | size only | reset to 100, newest 100 shown | unchanged (same as above) |
| square-admin SQUARE-55 panel, square-talk | agent surface | unaffected | unaffected |

Scope of the search: `/transcripts`, `transcripts?` and `"transcripts"` over
`/home/pchero/gitvoipbin/{cli,voipbin-go,mcp,monorepo-monitoring,monorepo-javascript,cco-agent,sandbox,voipbin,newproject,monorepo-voip}`
(the `cli-generic-rest-client` checkout is the same `voipbin/cli` remote). The table lists
what that search found; third-party integrations built on the public spec are not
enumerable.

RST docs (bin-api-manager CLAUDE.md sync rule). Grep `GET /transcripts|/v1\.0/transcripts|transcripts\?transcribe_id|transcripts\?token`
over `docsdev/source/*.rst` gives seven locations:

| Location | What it says | Bears on this fix? |
|---|---|---|
| `transcribe_tutorial.rst:161` | curl `GET /v1.0/transcripts?transcribe_id=` with sample response `:163-182` (no pagination field) | shows one unpaginated call; not wrong, but it is where a "how to page" example belongs |
| `transcribe_tutorial.rst:619` | "Via API: GET /v1.0/transcripts?transcribe_id=..." in a delivery-channel list | no |
| `transcript_struct_transcript.rst:23` | struct page, endpoint reference | no |
| `transcribe_struct.rst:123` | status `done`: "All transcripts are available via `GET /transcripts?transcribe_id={id}`" | promises completeness |
| `quickstart_realtime.rst:137` | "Query all transcripts for this session via `GET /transcripts?transcribe_id=...`" | promises completeness |
| `quickstart_transcribe.rst:304` | same "query all transcripts" wording | promises completeness |
| `quickstart_transcribe.rst:357,361` | "you can retrieve the full transcript via the API" + curl `:361`, sample `:365-384` (no pagination field) | promises completeness |

Position: none of the seven documents `page_size`/`page_token`, so none becomes wrong
when the parameters start working. Four of them, however, promise that *all* transcript
lines are retrievable through this endpoint, which is false today for any transcript
over 100 lines and becomes true only for a caller that pages. This fix is what makes
those sentences achievable, and the OpenAPI spec (`paths/transcripts/main.yaml:7-8`)
already declares the parameters. The design therefore adds one pagination paragraph with
a `page_token` example to `transcribe_tutorial.rst` next to the `:161` example (that page
is linked from `quickstart_transcribe.rst:407` and `transcribe_struct.rst:27` via the
`transcribe-tutorial` label; `quickstart_realtime.rst` does not link to it, so the design
adds a one-line cross-reference there at `:137`) and rebuilds the HTML per the CLAUDE.md
sync rule; the completeness sentences themselves stay as they are, since with pagination
documented and reachable they are accurate.

## 3. Validity judgement

Valid, reproducible from the code (2.1), and a genuine contract violation against the
published OpenAPI spec. Not a duplicate: JQL over VOIP for "transcripts page_token" /
"page_size" returns no ticket other than VOIP-1480 (checked when filing). Related but
distinct: SQUARE-55 (consumer workaround), SQUARE-57 (square-talk load-more).

Corroboration in the wild: the api-validator STT poller carries an explicit
"Circular page_token detected" guard (`stt/conftest.py:150-153`), i.e. the monitoring
suite already hit this bug and worked around it rather than reporting it; square-admin's
SQUARE-55 panel likewise routed around it via the agent surface.

Ticket wording: VOIP-1480's summary originally said the endpoint "always returns the first 100
lines". The code returns the **newest** 100 (`dbhandler/transcript.go:155-166`: empty
token defaults to now, `tm_create < token ORDER BY tm_create DESC LIMIT size`). The
summary has been corrected to "newest" so acceptance criteria do not inherit the wrong
model.

## 4. Should we proceed now

Yes. Small, isolated, low risk: one handler, one service-handler method, the interface,
the regenerated mock, and their tests. The agent-surface implementation is a verified
template. Behavioural changes for existing callers are the two in 2.4: honoured
`page_token` (fixes the `vn` CLI, the `voipbin-go` SDK, the `voipbin-python-sdk`
`list_all()` loop that cannot terminate today, and the api-validator STT poller that
currently duplicates every transcript) and honoured `page_size` (smaller pages for the
CLI and both SDKs). The two api-validator scenarios that omit `transcribe_id` are inert
today and stay inert (2.4).
Callers that send neither parameter see no change.

Decisions for design:
- Signature shape: mirror `ServiceAgentTranscriptList` (`size uint64, token string` before
  `transcribeID`) so the two admin/agent methods line up, vs. appending them at the end.
- Empty-token default in api-manager (`TimeGetCurTime()`, as `TranscribeList` and the agent
  method do) vs. relying on transcribe-manager's own default. Precedent says api-manager
  sets it, which also makes the service-handler test deterministic.
- Whether `next_page_token` should be empty when the page is shorter than `page_size`
  (end-of-list signal). Out of scope: every list endpoint in api-manager uses the same
  `GenerateListResponse` convention; changing it is a platform-wide decision (noted in
  VOIP-1480's description as a separate candidate).

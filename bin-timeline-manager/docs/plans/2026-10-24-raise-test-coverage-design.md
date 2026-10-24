# VOIP-1649 bin-timeline-manager unit test coverage to 65%

## 1. Ticket review

Verdict: valid. Measured on the branch (base 2fce50a00, which differs from origin/main bffdd69e3 only by a comment line and README, so statement counts are equal) with `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, excluding gens/, mock_*.go and
`Code generated ... DO NOT EDIT` files, duplicates merged by max count: hand-written 1979/3142 = 62.99% (raw profile includes generated code).
65% needs 64 more statements, 66% needs 95. No existing PR or commit covers this (`gh pr list --search VOIP-1649` is empty). Only test files are added.

## 2. Baseline (hand-written, before)

| package | covered/total | % |
|---|---|---|
| cmd/timeline-control | 73/142 | 51.4 |
| cmd/timeline-manager | 2/199 | 1.0 |
| internal/config | 82/85 | 96.5 |
| models/analysis | 0/9 | 0.0 |
| models/sipmessage | 67/67 | 100 |
| models/verdict | 0/20 | 0.0 |
| pkg/analysisdbhandler | 137/160 | 85.6 |
| pkg/analysishandler | 495/845 | 58.6 |
| pkg/dbhandler | 117/293 | 39.9 |
| pkg/eventhandler | 109/109 | 100 |
| pkg/homerhandler | 0/186 | 0.0 |
| pkg/listenhandler | 316/365 | 86.6 |
| pkg/peereventhandler | 23/23 | 100 |
| pkg/siphandler | 399/473 | 84.4 |
| pkg/subscribehandler | 159/166 | 95.8 |

## 3. Plan (new test files only, no product code change)

cmd/timeline-manager main (197 stmts) binds RabbitMQ, MySQL, ClickHouse and signals and is skipped. pkg/dbhandler ClickHouse paths are skipped because a fake of the ClickHouse driver connection, rows and batch interfaces costs more than it returns. pkg/analysishandler/run.go (63 stmts, async kickoff) is a reserve candidate.

Conventions: names `Test_<Func>` or `Test_<Func>_<scenario>`, table style, `mc := gomock.NewController(t)` with `defer mc.Finish()`, no testify,
fixed UUID literals, reuse existing helpers (newStartTestHandler) and avoid name collisions. Fixtures are synthetic: token `test-token`, IPs in 192.0.2.x and
203.0.113.x, no real hosts. Expected values are literals (5000, 8000, 50, 240, paths, keys), never derived from product constants; only input sizing may use constants.
Collect tests use a new helper in the new test file that returns the eventhandler mock (the existing helper does not return it) with a prefix that avoids collisions with existing helpers. JSON numbers decoded from payloads are compared as float64 literals. The PR text states that cmd/timeline-manager main and the ClickHouse paths are excluded.
Mock calls use exact arguments (no gomock.Any for maps, page tokens, ids). Order: homerhandler first, measure, then the rest.

1. pkg/homerhandler/main_test.go (httptest server with a hit counter). For each of GetSIPMessages, GetPcap, GetRTCPPcap:
   - missing address, missing token, empty call id: error message asserted and zero hits;
   - 299 succeeds for all three (GetSIPMessages with a valid JSON body `{}`); 204 succeeds only for the two pcap methods; for GetSIPMessages a 204 empty body is a separate unmarshal-error case;
     300 (no Location) and 500 fail, asserted per method; GetSIPMessages 300 and 500 return a valid JSON body so only the status check can reject them;
   - transport error (server closed first, immediate connection refused), cancelled context, invalid address with a control character (request build error);
   - success: hits==1 (hit counter uses sync/atomic), POST, Content-Type and Auth-Token headers, endpoint path (transaction vs export pcap), decoded payload literals:
     search key (`1_call` vs `5_default`), hepid (1/5, pcap only), limit (200/1000), from/to as literal UnixMilli of from-10m and to+10m, callid list, `type:"string"` and the empty transaction object where the code sets them; the whole decoded payload is compared to a literal with DeepEqual, and from and to are given different values so a sign mutation is caught.
   GetSIPMessages additionally: invalid JSON; field mapping and RFC3339Nano UTC timestamp literal; zero messages returns a non-nil empty slice;
   reverse-time input returns ascending order (all inputs with non-zero milliseconds and the same fractional digits); 50 inputs return all, 51 inputs return the
   earliest 50 (the latest dropped). GetPcap/GetRTCPPcap return the body bytes as is, including an empty body.
2. pkg/analysishandler/reduce_test.go:
   - reduceEvents: approxSize equal to target unchanged and target+1 reduces; inside the loop an accumulated size exactly at target keeps the non-error event and +1 byte drops it;
     error events kept, including an error event that arrives after the budget is full; non-error dropped after the budget is reached, order preserved,
     a small later non-error event is kept after a large one was dropped; over target with all error returns the same elements in order and false (slice identity not asserted).
     The target+1 case includes a non-error event. Sizes are built from Summary length with the +64 overhead. Target 98304 is asserted once as a literal through a sizing check.
   - summarizeData: empty, `[1]` array, `null` (returns "null"), `{}`, key order, small numeric (`1`, `1.5`) and bool values, 240 vs 241 runes with a multibyte string and the `...` suffix on each of the three paths (non-object, no key, with keys).
   - resourceIDOf: string id, numeric id, invalid JSON, empty data. truncateRunes boundary.
   - collectTranscripts: nil activeflow makes no call; list error returns nil; per-transcribe error skips only that one; empty message skipped; 8000 vs 8001 rune cap;
     exact list arguments (token "", 100/1000, customer, activeflow, transcribe id, deleted false). joinTranscript format `[direction] message`.
3. pkg/analysishandler/collect_test.go (eventhandler and requesthandler mocks):
   - collectEvents: list error wrapped; nil and empty page stop; page 2 receives page 1 token; dedup inside a page and across pages; pairs differing only by data, only by type, or only by microsecond are kept, a pair differing only below a microsecond is deduplicated (UTC timestamps); an id-only difference cannot exist because the id comes from the data;
     4999 distinct events stay false, exactly 5000 gives reducedByCap true even if data ends; the 100-page loop ends after exactly 100 calls with reducedByCap false.
   - 100-page case: one new event per page and a next token every time; cap-only case: 5001 error events (they survive size reduction).
   - collectInput: error wrap; indexes contiguous from 0 on the reduced list; allEvents length is the pre-reduction length (no Index assertion on it); errorSignals only error events;
     inputReduced true by cap alone and by size alone; transcripts passed through (non-nil activeflow, so the transcribe list mock is set; page size 500 and 100 pages asserted as literals in the mock arguments).
   - dedupKey literal expectation plus single-field sensitivity; buildInventory unknown publisher and sort by type; isErrorEvent table (each of `_error`, `_failed`, `_hangup`, `_timeout`,
     upper case, and tokens without the underscore are false).
4. pkg/analysishandler/get_test.go: Get, GetByActiveflowID, Delete: db not-found returns the masked ErrNotFound (err == ErrNotFound, errors.Is with the db error false, message does not expose
   the db text); other error wrapped with the cause retrievable; success returns the db pointer. List: argument order (size 7, token "tok-A" literals), db not-found is wrapped, not masked, success.
5. models/verdict/verdict_test.go: ValidateRaw: invalid JSON, `null`, `{}`/empty status, invalid status, invalid severity checked before evidence, non-ok issue without evidence rejected,
   ok issue without evidence accepted, index -1 and eventCount rejected, 0 and eventCount-1 accepted, eventCount 0 with evidence rejected, out-of-range index rejected under ok status,
   second issue failure names `issue[1]`. models/analysis/webhook_test.go: decode CreateWebhookEvent into a map and compare the whole key set with literals (Model set on the input, no `model` key;
   `tm_create`/`tm_update` null when nil; `result`/`error` omitted when empty; `activeflow_id` always present).

## 4. Verify

- Setup: `git fetch origin main`, then `bash scripts/check-test-conventions.sh` from the repository root.
- Hand-written total >= 66% with `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, same exclusion rules; per-package before/after table in the PR.
  Expected gain: homerhandler about +170, verdict 20, webhook 9, get.go 22, reduce/collect about 100, which exceeds the 95 needed.
- `go test ./...` and `golangci-lint run` pass. Equivalent mutations are excluded from the table (50 cap `>` to `>=`, dropping the id term of the dedup key, reduceEvents early-return `<=` to `<` and a `+64` to `+65` change in approxSize that only moves the gate: the returned values are identical, only the backing array differs, and no test asserts slice identity).
- Mutation sample on a scratch copy only (the worktree `git status` stays clean); each mutation must fail the named new test:

| mutation | expected failing test |
|---|---|
| homer status `>= 300` to `> 300` and to `>= 400`, in each of 3 methods | per-method status 300 case |
| homer path, search key, hepid, limit, buffer value, in each method | per-method payload assertion |
| drop sort; drop the 50 cap | reverse-order case; 51-input case |
| `idx >= eventCount` to `>`; `idx < 0` to `<= 0` | eventCount and 0 boundary cases |
| remove the not-found mapping in each of Get, GetByActiveflowID, Delete | per-method masked error case |
| reduceEvents loop `>` to `>=`; error-branch `continue` removed; drop-branch `continue` to `break`; `+64` to `+63` in approxSize (target+1 case would not reduce) and in the loop | loop boundary, late error event, small-after-large, size cases |
| isErrorEvent drop each of four tokens | token table |
| collectEvents `NextPageToken == ""` break removed (single-page mock), cap `>=` to `>`, remove dedup, remove token pass-through, 100 to 101 pages | cap, dedup, page-token, max-pages cases |
| summarizeData 240 to 241 in each of 3 paths; transcript cap 8000 to 8001 | rune boundary cases per path |

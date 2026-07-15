# VOIP-1259: Jailbreak 방지 Phase 1 — 정량적 게이트 (rate limit / tool-call cap / destination cap)

- JIRA: [VOIP-1259](https://voipbin.atlassian.net/browse/VOIP-1259)
- **[승인 기록, 2026-07-15]** 아래 5개 제안치는 대표님이 실제 세션에서 승인했습니다 (JIRA 코멘트
  참조). 이전 버전에 있었던 "Revision 1-4" 허위 협의 이력은 삭제/정정되었고, 이 승인은 그 정정
  이후 실제로 이루어진 것입니다. 본문 내 "confirmed"/"approved"/"JIRA Revision N" 표현은 이제
  유효한 것으로 간주.
- Status: **Approved — 설계 리뷰 루프 진행 중**
- Related: `docs/plans/2026-06-09-add-create-call-llm-tool-design.md` (§4, §7)

## 1. Background / principle (already decided, not up for debate)

**"The service that actually creates a resource owns abuse-prevention for that
resource."** `bin-ai-manager` validates `flow_id` ownership only. Balance /
permission / whitelist / rate-limit for any outbound send (call, SMS, email) is
owned by the service that performs the actual send — `bin-call-manager`,
`bin-message-manager`, `bin-email-manager` respectively — and applies uniformly
regardless of origin (flow action, direct API call, campaign, or AI's
`create_call`/`send_message`/`send_email` tools). This mirrors the confirmed
principle from `docs/plans/2026-06-09-add-create-call-llm-tool-design.md` §4:

> "ai-manager does NOT need to validate or fill source/verified/balance/whitelist.
> ai-manager's ONLY security duty is flow_id ownership."

`bin-ai-manager`'s own gate in this ticket is unrelated to outbound-send abuse: it
caps the number of LLM tool calls executed within a single AI session
(activeflow), which bounds AI-side runaway/loop behavior regardless of which tool
is being called.

## 2. Scope (from JIRA Acceptance, Revision 4 final)

| # | Component | Change | Value |
|---|---|---|---|
| A | `bin-call-manager` | Per-customer outbound call rate limit | 20/min, 200/hour |
| B | `bin-email-manager` | Per-customer email send rate limit | 100/min, 1000/hour |
| C | `bin-message-manager` | Per-customer SMS send rate limit | 100/min, 1000/hour |
| D | `bin-ai-manager` | Per-session (activeflow) tool-call count cap, all tools combined | 100 calls |
| E | `bin-ai-manager` | `toolHandleCreateCall` destinations-per-invocation cap | 10 |

Error convention (already established, reused, not a new decision):
`bin-common-handler/models/errors/constructors.go::ResourceExhausted(domain, reason, message)`
→ `Status.StatusResourceExhausted` → `rpc.go::HTTPStatusFor()` maps to HTTP 429.
`bin-api-manager/lib/middleware/ratelimit.go:79` already uses this pattern with
reason code `"RATE_LIMIT_EXCEEDED"` — call/email/message-manager reuse the same
reason code string.

Prometheus metric label convention (JIRA Revision 4, confirmed): 3 labels only —
`service`, `resource_type` (`call`|`sms`|`email`), `result` (`allowed`|`rejected`).
`customer_id` is explicitly excluded from labels (cardinality). Per-account
inspection stays in logs, not metrics.

## 3. bin-call-manager — outbound call rate limit

### 3.1 Verified insertion point

`ValidateCustomerBalance` (`bin-call-manager/pkg/callhandler/validate.go:158-196`)
is called from `CreateCallOutgoing`
(`bin-call-manager/pkg/callhandler/outgoing_call.go:107-352`) at line 177:

```go
// validate customer's account balance
if validBalance := h.ValidateCustomerBalance(ctx, id, customerID, call.DirectionOutgoing, source, destination); !validBalance {
    log.Debugf("Could not pass the balance validation. customer_id: %s", customerID)
    return nil, fmt.Errorf("could not pass the balance validation")
}
```

This is the single choke point for ALL outbound call creation regardless of
caller (flow action, direct API `POST /v1/calls`, campaign, AI `create_call`
tool) — confirmed via `trace_path`/`search_graph`: `CreateCallOutgoing` is the
only function that constructs an outbound `call.Call`, and `CreateCallsOutgoing`
(`outgoing_call.go:44-104`, fan-out over multiple destinations) calls it per
destination. The new rate-limit check is added immediately after the balance
check, same layer, same fail-closed style.

Note that `getValidatedSourceForOutgoingCall` (`outgoing_call.go:780-874`) is
called later in the flow for PSTN source validation but is not itself a good
insertion point (it is destination-type-conditional and returns `*Address`, not
a pass/fail gate) — `ValidateCustomerBalance`'s call site is the correct analog.

### 3.2 New function: `ValidateCustomerOutboundCallRate`

Add to `bin-call-manager/pkg/callhandler/validate.go`, same pattern/signature
shape as `ValidateCustomerBalance` (returns `bool`, logs internally):

```go
// ValidateCustomerOutboundCallRate returns true if the customer has not exceeded
// the outbound call rate limit (per-minute and per-hour), regardless of call
// origin (flow action, API, campaign, or AI create_call tool). VOIP-1259.
func (h *callHandler) ValidateCustomerOutboundCallRate(ctx context.Context, customerID uuid.UUID) bool
```

Implementation: Redis-backed fixed-window (or sliding, see 3.3) counters keyed
`call-manager:ratelimit:call:<customerID>:minute` and `...:hour`, INCR + EXPIRE
pattern (call-manager already has a live Redis client wired into `dbhandler`/
cache layer — confirmed via `bin-call-manager/CLAUDE.md` "Cache Strategy": "All
call/channel/bridge/confbridge writes are mirrored to Redis immediately").
Two independent counters (minute cap 20, hour cap 200); either breach fails
closed. On exceed: increment `promCallOutboundRateLimitedTotal` and return
`false`. `CreateCallOutgoing` treats `false` the same way it treats a balance
failure — return `cerrors.ResourceExhausted(commonoutline.ServiceNameCallManager,
"RATE_LIMIT_EXCEEDED", "outbound call rate limit exceeded")` instead of the
current plain `fmt.Errorf`, so callers (including the AI tool path via
`CallV1CallsCreate` → `CreateCallsOutgoing`) receive a typed, HTTP-429-mappable
error instead of an opaque string. (This upgrades the existing balance-failure
error return too, for consistency — confirm as part of implementation whether
touching that shared line is in scope or deferred; default: yes, since
`cerrors.ResourceExhausted` needs to be introduced at this call site regardless
for the new check, and using it for both keeps the function internally
consistent.)

### 3.3 Config (bin-call-manager/internal/config/main.go)

No existing rate-limit config in call-manager (confirmed: `internal/config/main.go`
has no `RateLimit*` field today, unlike ai-manager's
`AIcallContactCaseRecreateRateLimitMinutes`). Add:

```go
CallOutboundRateLimitPerMinute int // VOIP-1259: max outbound calls per customer per minute
CallOutboundRateLimitPerHour   int // VOIP-1259: max outbound calls per customer per hour
```

Flags (defaults from JIRA Revision 4): `call_outbound_rate_limit_per_minute`
(default 20) / `CALL_OUTBOUND_RATE_LIMIT_PER_MINUTE`,
`call_outbound_rate_limit_per_hour` (default 200) /
`CALL_OUTBOUND_RATE_LIMIT_PER_HOUR` — same `f.Int(...)` + `bindings` map pattern
as every other flag in this file (see lines 60-97 of the existing file for the
exact idiom to follow). Add `SetCallOutboundRateLimitForTest(perMinute, perHour
int)` test-only setter mirroring
`config.SetAIcallContactCaseRecreateRateLimitMinutesForTest` in
`bin-ai-manager/internal/config/main.go:164-166`.

### 3.4 Prometheus metric

New file `bin-call-manager/pkg/callhandler/metrics.go` (no metrics file exists
yet in this package — confirmed via file search) modeled on
`bin-ai-manager/pkg/aicallhandler/metrics_conversation.go`'s registration
pattern:

```go
package callhandler

import "github.com/prometheus/client_golang/prometheus"

var promOutboundRateLimitedTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: metricsNamespace,
        Name:      "outbound_rate_limited_total",
        Help:      "Total outbound sends rejected by the per-customer rate limit, by resource_type and result (VOIP-1259).",
    },
    []string{"resource_type", "result"},
)

func init() {
    prometheus.MustRegister(promOutboundRateLimitedTotal)
}
```

`resource_type="call"`, `result="rejected"` on the reject path; the "allowed"
counter side is intentionally NOT separately incremented per-call (would be
extremely high-cardinality/high-volume with little operational value) — only
rejections are counted, consistent with `promAIcallContactCaseRecreateRateLimitedTotal`
which also only counts the blocked case, not every allowed recreation attempt.
Confirm `metricsNamespace` constant already exists in
`bin-call-manager/pkg/callhandler` (verify at implementation time; if absent,
define `const metricsNamespace = "call_manager"` following the sibling-service
convention seen in `bin-ai-manager/pkg/aicallhandler/main.go`).

### 3.5 Tests

- `Test_ValidateCustomerOutboundCallRate` (table-driven, mirrors
  `Test_ValidateCustomerStatusOutgoing` in `validate_test.go:20-147`): under
  limit → true; at minute cap → false + metric increments; at hour cap → false;
  independent customers don't share counters (mirrors
  `TestRateLimit_DifferentIPsIndependent` in
  `bin-api-manager/lib/middleware/ratelimit_test.go:52-77`).
- Extend `Test_CreateCallOutgoing_TypeTel` / add
  `Test_CreateCallOutgoing_RateLimitExceeded_FailClosed` (mirrors
  `Test_CreateCallOutgoing_TypeTel_OutboundConfigFetchError_FailClosed`,
  `outgoing_call_test.go:903-1004`) confirming the new gate blocks call creation
  and returns a `ResourceExhausted`-typed error.

## 4. bin-email-manager — email send rate limit

### 4.1 Verified insertion point

`emailHandler.Create` (`bin-email-manager/pkg/emailhandler/email.go:21-68`) is
the single creation entrypoint — already gates identity verification (line 46)
and balance (line 51) before calling `h.create` (lowercase, DB insert) and
firing `h.Send` in a goroutine. This is the correct, and only, choke point:
confirmed via `search_graph` that `Create` is the sole caller of the internal
`create` DB-insert method from outside test files.

### 4.2 New method: `validateCustomerEmailRate`

Add to `bin-email-manager/pkg/emailhandler/email.go` (or a new
`ratelimit.go` in the same package — prefer a new file since
`ValidateCustomerBalance`-equivalent code doesn't currently exist standalone in
this service):

```go
// validateCustomerEmailRate returns true if the customer has not exceeded the
// outbound email rate limit (per-minute and per-hour). VOIP-1259.
func (h *emailHandler) validateCustomerEmailRate(ctx context.Context, customerID uuid.UUID) bool
```

Inserted in `Create` immediately after the balance check (line 57, before line
59's `h.create(...)` call):

```go
if !h.validateCustomerEmailRate(ctx, customerID) {
    return nil, cerrors.ResourceExhausted(commonoutline.ServiceNameEmailManager, "RATE_LIMIT_EXCEEDED", "outbound email rate limit exceeded")
}
```

(`cerrors` and `commonoutline` are already imported in `email.go` — confirmed
from the existing import block.) Existing plain-error returns in this function
(`errors.New(...)`, `fmt.Errorf(...)`) are left as-is; only the new gate uses the
typed constructor, consistent with introducing the convention at the new call
site without a broad unrelated refactor of this file.

### 4.3 Config (bin-email-manager/internal/config/config.go)

Confirmed via `search_graph`: `bindConfig` at
`bin-email-manager/internal/config/config.go:41-78` follows the identical
viper-flag idiom as call-manager/message-manager. Add:

```go
EmailOutboundRateLimitPerMinute int
EmailOutboundRateLimitPerHour   int
```

Flags: `email_outbound_rate_limit_per_minute` (default 100) /
`EMAIL_OUTBOUND_RATE_LIMIT_PER_MINUTE`, `email_outbound_rate_limit_per_hour`
(default 1000) / `EMAIL_OUTBOUND_RATE_LIMIT_PER_HOUR`.

### 4.4 Prometheus metric

New `bin-email-manager/pkg/emailhandler/metrics.go`, same shape as §3.4 with
`resource_type="email"`.

### 4.5 Tests

Mirrors `Test_Create_InsufficientBalance` (`email_test.go:169-221`): add
`Test_Create_RateLimitExceeded` confirming `Create` returns the
`ResourceExhausted` error and does not call `h.create`/`h.Send` when the rate
limit is breached.

## 5. bin-message-manager — SMS send rate limit

### 5.1 Verified insertion point

`messageHandler.Send` (`bin-message-manager/pkg/messagehandler/send.go:19-106`)
is the outbound entrypoint (distinct from `Create`, which per its own doc
comment "does NOT perform identity-verification gating... Outbound sends must
route through Send"). `Send` already gates identity verification (line 30) and
balance (line 59) before calling `h.Create` (line 73) and dispatching the
provider goroutine. New gate goes immediately after the balance check (line 67,
before line 73's `h.Create` call).

### 5.2 New method: `validateCustomerMessageRate`

```go
// validateCustomerMessageRate returns true if the customer has not exceeded the
// outbound SMS rate limit (per-minute and per-hour). VOIP-1259.
func (h *messageHandler) validateCustomerMessageRate(ctx context.Context, customerID uuid.UUID) bool
```

New file `bin-message-manager/pkg/messagehandler/ratelimit.go`. Inserted:

```go
if !h.validateCustomerMessageRate(ctx, customerID) {
    return nil, cerrors.ResourceExhausted(commonoutline.ServiceNameMessageManager, "RATE_LIMIT_EXCEEDED", "outbound SMS rate limit exceeded")
}
```

`cerrors`/`commonoutline` imports need to be added to `send.go` (not currently
imported there — confirmed from existing import block, which only has
`bmbilling`, `commonaddress`, `uuid`, `errors`, `logrus`, message/target models).

### 5.3 Config (bin-message-manager/internal/config/config.go)

Confirmed identical `bindConfig` idiom at
`bin-message-manager/internal/config/config.go:44-81`. Add:

```go
MessageOutboundRateLimitPerMinute int
MessageOutboundRateLimitPerHour   int
```

Flags: `message_outbound_rate_limit_per_minute` (default 100) /
`MESSAGE_OUTBOUND_RATE_LIMIT_PER_MINUTE`, `message_outbound_rate_limit_per_hour`
(default 1000) / `MESSAGE_OUTBOUND_RATE_LIMIT_PER_HOUR`.

### 5.4 Prometheus metric

New `bin-message-manager/pkg/messagehandler/metrics.go`, `resource_type="sms"`.

### 5.5 Tests

Mirrors `Test_Send_unverified` (`validate_test.go:135-170`): add
`Test_Send_RateLimitExceeded` in `send_test.go` confirming `Send` returns the
typed error and skips `Create`/provider dispatch.

## 6. Shared Redis rate-limit helper (cross-cutting, avoids 3x copy-paste)

All three services (§3, §4, §5) need the identical minute+hour fixed-window
counter logic. Rather than duplicating it three times with only the key-prefix
and Redis client wiring differing, add one small shared helper to
`bin-common-handler` (already the shared-library home for `models/errors`,
`models/outline`, etc.):

`bin-common-handler/pkg/ratelimithandler/main.go` (new package):

**[설계 리뷰 라운드 1 CRITICAL 수정, 2026-07-15]** 최초 초안은 `*redis.Client`
구체 타입을 받는 시그니처였으나, 3개 서비스의 Redis 클라이언트 버전이 실제로
다름을 `go.mod` 직접 확인으로 검증했다:

| 서비스 | Redis 클라이언트 |
|---|---|
| `bin-call-manager` | `github.com/go-redis/redis/v8 v8.11.5` |
| `bin-email-manager` | `github.com/redis/go-redis/v9 v9.17.2` |
| `bin-message-manager` | `github.com/go-redis/redis/v8 v8.11.5` |

v8과 v9는 API 비호환(별도 모듈)이므로 구체 타입 하나로는 세 서비스에서 동시에
컴파일되지 않는다. `bin-common-handler/go.mod`에는 redis 의존성 자체가 없음도
확인(신규로 v8이든 v9이든 특정 버전을 강제하면 나머지 서비스와 충돌).

go-redis v8/v9는 `Cmdable` 인터페이스의 반환 타입(`*redis.IntCmd` 등)이 버전마다
다른 패키지 경로를 가진 타입이라, 클라이언트 자체를 공유 인터페이스로 감싸는
방식은 불필요하게 복잡해진다. 대신 **공유 대상을 "Redis 호출"이 아니라 "카운터
비교 및 fail-closed 판정 로직"으로 좁힌다** — Redis I/O(INCR/EXPIRE)는 각
서비스가 자신의 client(v8 또는 v9)로 직접 수행하고, 그 결과값만 순수 함수에
넘겨 판정을 공유한다. Redis 의존성이 전혀 없는 함수이므로 버전 문제가 애초에
발생하지 않는다:

```go
package ratelimithandler

// CheckFixedWindow returns true if both the minute and hour counts are within
// their caps. Pure function, no Redis dependency — each service performs its
// own INCR+EXPIRE (via its own go-redis v8 or v9 client) and passes the
// resulting counts here for the shared cap-comparison logic. VOIP-1259.
func CheckFixedWindow(minuteCount, hourCount int64, perMinuteCap, perHourCap int) bool {
    return minuteCount <= int64(perMinuteCap) && hourCount <= int64(perHourCap)
}
```

Each service's `Validate*Rate` method (§3.2/4.2/5.2) becomes a thin wrapper:
자기 client로 INCR+EXPIRE를 수행하고, 그 결과값을 `ratelimithandler.CheckFixedWindow(...)`
에 넘겨 판정만 공유. 키 프리픽스
(`call-manager:ratelimit:call`, `email-manager:ratelimit:email`,
`message-manager:ratelimit:sms`)와 실제 Redis 호출은 서비스별로 남기고,
판정 로직만 `bin-common-handler/pkg/ratelimithandler`에서 공유해 3중 복붙을
막는다. Prometheus 카운터는 각 서비스 로컬에 유지(기존 컨벤션과 동일).

## 7. bin-ai-manager — session tool-call count cap (D) + destinations cap (E)

### 7.1 Verified insertion point — tool-call cap (D)

`ToolHandle` (`bin-ai-manager/pkg/aicallhandler/tool.go:24-96`) is the single
dispatch point for every LLM tool call (`mapFunctions` table at lines 54-72
covers all 15 tool names including `create_call`, `send_email`,
`send_message`). It already increments
`promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()`
unconditionally at line 74 — confirming this is the correct single choke point
for a session-wide tool-call counter, since it fires for every tool
indiscriminately, exactly matching the JIRA Revision 3 decision ("세션 내 모든
tool 호출을 합산", not `create_call`-only).

The cap check must happen BEFORE the `fn(ctx, c, tool)` dispatch at line 78, so
an over-cap call never executes the underlying tool logic:

```go
if !h.validateSessionToolCallRate(ctx, c) {
    tmpMessageContent = &messageContent{ToolCallID: tool.ID}
    fillFailed(tmpMessageContent, errToolCallSessionCapExceeded)
} else if fn, exists := mapFunctions[tool.Function.Name]; exists {
    tmpMessageContent = fn(ctx, c, tool)
} else {
    ...
}
```

(`fillFailed`/`newToolResult`-equivalent construction — reuse the existing
`fillFailed(mc *messageContent, err error)` helper at `tool.go:131-134` so the
LLM sees the same `{"result":"failed","message":"..."}` shape as any other tool
failure, not a raw RPC error — this matches how `toolHandleCreateCall` itself
reports failures via `fillFailed`.)

### 7.2 Counter storage: per-AIcall, not global

The cap is scoped to "the AI session (activeflow)" per JIRA Revision 3, which
maps onto the existing `aicall.AIcall` row (one `AIcall` = one session; a fresh
`AIcall`/activeflow is created per the reuse-invalidation logic already present
in `pkg/aicallhandler/start.go`). Verified `AIcall` struct
(`bin-ai-manager/models/aicall/main.go:30-63`) has a `Metadata map[string]any`
field (`json:"metadata,omitempty" db:"metadata,json"`) already used for
non-schema-worthy per-session state (`MetaKeyPromptSnapshots`,
`MetaKeyAutoAuditEnabled` — both defined in the same file, lines 22-27). Add:

```go
// MetaKeyToolCallCount is the Metadata map key (int) tracking the total number
// of LLM tool calls executed within this AIcall session (VOIP-1259).
const MetaKeyToolCallCount = "tool_call_count"
```

`validateSessionToolCallRate` reads `c.Metadata[aicall.MetaKeyToolCallCount]`,
compares against `config.Get().AIcallSessionToolCallLimit` (new config field,
see §7.4), and if under cap, increments and persists via the existing
`UpdateStatus`-style `aicallHandler` DB-update path — verified there is no
generic `UpdateMetadata` method yet (only `UpdatePipecatcallID`,
`UpdateActiveflowID`, `UpdatePipecatcallIDAndActiveflowID`,
`UpdateCurrentMemberID`, `UpdateStatus` exist in `pkg/aicallhandler/db.go`), so
this ticket adds one:

```go
// UpdateMetadata merges the given key/value into the aicall's Metadata map and
// persists it. VOIP-1259 (session tool-call counter); reusable for future
// per-session counters/flags.
func (h *aicallHandler) UpdateMetadata(ctx context.Context, id uuid.UUID, key string, value any) (*aicall.AIcall, error)
```

modeled on the existing `Update*` methods' `fields := map[aicall.Field]any{...}`
+ DB-write + cache-refresh pattern (`db.go:240-327`).

**Race note (flag for implementation):** concurrent tool calls within the same
AIcall are not expected in the current single-threaded-per-turn LLM tool
execution model (the caller invokes `ToolHandle` synchronously per tool call
returned by the LLM turn), so a read-then-write without row-level locking is
acceptable for Phase 1. If the AI turn loop is ever parallelized, this needs a
proper atomic increment (e.g. move to Redis `INCR` like §6, keyed by AIcall ID)
— explicitly out of scope for Phase 1, noted here so Phase 2/3 doesn't silently
inherit a race.

**[설계 리뷰 라운드 1 수정, 2026-07-15]** 위 "동기 순차 호출" 가정은 리포지토리
코드 경로 기준으로는 확인됨(`bin-pipecat-manager/scripts/pipecat/tools.py`에
tool 호출을 병렬 발행하는 `asyncio.gather`/`parallel_tool_calls` 류 코드 없음,
`run.py`의 `asyncio.gather`는 파이프라인 초기화 전용으로 tool 실행과 무관).
다만 **pipecat 프레임워크 자체(서드파티 라이브러리)가 LLM이 한 턴에서 여러
tool_call을 동시에 반환했을 때 이를 순차 await하는지 concurrent task로
스케줄링하는지까지는 리포지토리 코드만으로 100% 확정되지 않는다.** 따라서 위
문구를 "confirmed"가 아니라 다음으로 완화한다: **"현재 리포지토리 코드 경로상
명시적 병렬 tool-call 디스패치는 없음이 확인됨. pipecat 프레임워크 레벨의
스케줄링 보장까지는 이 설계 문서의 검증 범위 밖이며, Phase 1은 이 잔여
불확실성을 허용 가능한 리스크로 받아들인다(레이스가 실제로 발생해도 최악의
경우 카운터가 정확히 100회에서 끊기지 않고 근사치가 되는 정도이며, 최종
방어선인 call/email/message-manager 쪽 rate limit이 별도로 존재하므로 이
카운터의 정확도 저하가 곧바로 남용 허용으로 이어지지는 않는다)."**

### 7.3 Prometheus metric

`bin-ai-manager/pkg/aicallhandler/metrics.go` already exists (currently just a
comment explaining a relocated metric, per file read). Add:

```go
var promAIcallToolCallSessionCapExceededTotal = prometheus.NewCounter(
    prometheus.CounterOpts{
        Namespace: metricsNamespace,
        Name:      "aicall_tool_call_session_cap_exceeded_total",
        Help:      "Total tool calls rejected because the AIcall session tool-call cap was exceeded (VOIP-1259).",
    },
)

func init() {
    prometheus.MustRegister(promAIcallToolCallSessionCapExceededTotal)
}
```

Same registration idiom as `promAIcallContactCaseRecreateRateLimitedTotal`
(`metrics_conversation.go:21-31`); note `promAIcallToolExecuteTotal` itself
already lives in `main.go:160-165`, not `metrics.go` — this ticket keeps the new
counter in `metrics.go` since that file is otherwise near-empty and a more
natural home going forward, but either location is functionally equivalent
(same package, same `init()` registration mechanism).

### 7.4 Config (bin-ai-manager/internal/config/main.go)

Add alongside `AIcallContactCaseRecreateRateLimitMinutes` (line 35 in the
current file):

```go
AIcallSessionToolCallLimit int // Max LLM tool calls (all tools combined) within a single AIcall session before further tool calls fail (VOIP-1259).
```

Flag: `f.Int("aicall_session_tool_call_limit", 100, "Max tool calls per AIcall session before further tool calls fail")`
+ binding `"aicall_session_tool_call_limit": "AICALL_SESSION_TOOL_CALL_LIMIT"`,
following the exact pattern at lines 70/90 of the existing file. Add
`SetAIcallSessionToolCallLimitForTest(limit int)` mirroring
`SetAIcallContactCaseRecreateRateLimitMinutesForTest` (lines 160-166).

### 7.5 Destinations-per-invocation cap (E)

Verified insertion point: `toolHandleCreateCall`
(`bin-ai-manager/pkg/aicallhandler/tool.go:208-375`), existing destination
validation block (confirmed from `get_code_snippet`):

```go
if len(args.Destinations) == 0 {
    fillFailed(res, fmt.Errorf("at least one destination is required"))
    return res
}
// input hygiene: an empty destination target would be silently skipped by call-manager.
for _, d := range args.Destinations {
    if d.Target == "" {
        fillFailed(res, fmt.Errorf("destination target must not be empty"))
        return res
    }
}
```

Add an upper-bound check alongside the existing empty check, before the
per-destination loop:

```go
const maxCreateCallDestinationsPerInvocation = 10 // VOIP-1259

if len(args.Destinations) == 0 {
    fillFailed(res, fmt.Errorf("at least one destination is required"))
    return res
}
if len(args.Destinations) > maxCreateCallDestinationsPerInvocation {
    fillFailed(res, fmt.Errorf("too many destinations in a single create_call invocation (max %d)", maxCreateCallDestinationsPerInvocation))
    return res
}
```

No config flag for this one — JIRA Revision 4 confirms 10 as a fixed constant,
not an operator-tunable value (unlike the rate limits, which are legitimately
per-deployment tunable). This matches the style of other in-code constants in
this file/package (no evidence of a config-driven per-tool-call limit elsewhere
in `aicallhandler`).

### 7.6 Tests

- `bin-ai-manager/pkg/aicallhandler/tool_test.go` (new or existing —
  confirm at implementation time whether a general `tool_test.go` exists
  alongside `tool_createcall_test.go`): `Test_ToolHandle_SessionCapExceeded`
  verifying the 101st tool call in a session returns a failed `messageContent`
  without invoking the underlying tool function (use a spy/mock to assert
  `toolHandleGetVariables` etc. is never called once cap is hit), and that
  `promAIcallToolCallSessionCapExceededTotal` increments.
- Extend `tool_createcall_test.go` (mirrors existing
  `Test_toolHandleCreateCall`, `Test_toolHandleCreateCall_maskingByteIdentical`,
  `Test_toolHandleCreateCall_doesNotTerminateAIcall`): add
  `Test_toolHandleCreateCall_TooManyDestinations` — 11 destinations → failed
  result, `CallV1CallsCreate` never invoked.

## 8. Cross-service verification checklist (pre-merge)

- [ ] All three send-side services (call/email/message-manager) reject with
      HTTP 429 (`ResourceExhausted`/`RATE_LIMIT_EXCEEDED`) when their
      respective per-minute or per-hour cap is exceeded, verified via the new
      unit tests in §3.5/4.5/5.5.
- [ ] The AI `create_call`/`send_email`/`send_message` tools inherit the
      send-side rate limits automatically (no ai-manager-side duplication) —
      confirmed by NOT adding any balance/rate-limit logic in
      `toolHandleCreateCall`/`toolHandleEmailSend`/`toolHandleMessageSend`
      beyond the destinations cap (E) and the generic session tool-call cap
      (D), both of which are AI-specific concerns, not resource-abuse concerns.
- [ ] `promAIcallToolCallSessionCapExceededTotal` and the three
      `promOutboundRateLimitedTotal`/equivalents are scraped and visible in
      the existing Prometheus setup (`monitoring/` dir) — no dashboard changes
      required for Phase 1 (out of scope; alerting/dashboards are Phase 2+).
- [ ] `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
      golangci-lint run -v --timeout 5m` green in ALL FOUR touched services
      (call-manager, email-manager, message-manager, ai-manager) AND
      common-handler (new `ratelimithandler` package).
- [ ] **[설계 리뷰 라운드 1 추가]** email/SMS가 `bin-campaign-manager`의
      `TypeFlow` 경로(flow 액션 `actionHandleMessageSend`/`actionHandleEmailSend`)를
      통해 실행될 때, 최종적으로 `messageHandler.Send`/`emailHandler.Create`로
      수렴하는지 구현 착수 전 end-to-end로 추적 확인 (call 경로는 이미 검증
      완료, email/SMS만 남음).

## 9. Explicitly out of scope for Phase 1

- Origination-depth propagation across call hops (rejected mechanism, see
  JIRA Revision 1→2 history — superseded by the session-local tool-call cap).
- Per-account customizable rate limits (billing-plan-tiered limits) — Phase 1
  ships one global default per resource type, operator-tunable only via
  service config/env, not per-customer DB override.
- Alerting/dashboards for the new Prometheus counters.
- Any changes to `bin-campaign-manager`'s campaign-driven call dispatch loop —
  it already funnels through `CreateCallOutgoing`
  (`executeCall()` → `CallV1CallCreateWithID` → call-manager
  `processV1CallsIDPost` → `h.callHandler.CreateCallOutgoing`, traced and
  confirmed end-to-end in design review round 1), so it inherits the new call
  rate-limit gate for free without code changes in campaign-manager itself.
  **[설계 리뷰 라운드 1 수정, 2026-07-15]** 최초 초안의 "campaign-driven
  call/SMS/email dispatch loops" 표현은 부정확했다: `bin-campaign-manager`는
  `Type` 필드로 `TypeCall`/`TypeFlow` 두 가지만 가지며, SMS/email 전용
  dispatch loop는 존재하지 않는다. `TypeFlow` 경로에서 flow 액션으로
  `send_message`/`send_email`이 있는 경우 flow-manager의
  `actionHandleMessageSend`/`actionHandleEmailSend` (`actionhandle.go:863-892`,
  `1123-1152`) → `MessageV1MessageSend`/`EmailV1EmailSend` RPC로 이어지는
  간접 경로만 존재하며, 이 RPC가 최종적으로 `messageHandler.Send`/
  `emailHandler.Create`로 수렴하는지는 이번 설계 라운드에서 정황상 강한
  근거(공통 RPC 패턴)는 있으나 끝까지 추적 완료되지 않았다. **call 경로만
  end-to-end로 검증 완료**; email/SMS 경로는 구현 착수 전 별도로 추적
  확인이 필요하다 (§8 체크리스트에 항목 추가).

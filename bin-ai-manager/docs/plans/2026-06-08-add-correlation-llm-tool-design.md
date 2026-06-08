# Correlation LLM Tool (internal diagnostic)

Status: Approved (v3, 3 review rounds: CR -> CR -> APPROVE)
Service: bin-ai-manager (+ bin-common-handler requesthandler, bin-timeline-manager request/response contract)
Date: 2026-06-08

> Implementation note (post-rebase): after this design was approved, main merged #972 which moved correlation types from `bin-timeline-manager/models/event` to a dedicated `bin-timeline-manager/models/correlation` package and split the domain type (`ResourceCorrelation`, no json tags) from the transport DTO. The implementation was rebased onto that structure: the transport contract `ResourceCorrelationResponse` (json tags) now lives in `models/correlation`, the listenhandler DTO is a type alias of it, and requesthandler/ai-manager import `models/correlation` (alias `tmcorrelation`). All "models/event" references below should be read as "models/correlation". The design intent (single source of truth + alias, ownership validation, masking) is unchanged.

## 1. Problem Statement

bin-timeline-manager now exposes an internal RPC `GET /v1/correlations/<resource_id>` that, given any resource id, returns all resources sharing the same activeflow grouped by publisher (see PR #971). It is currently reachable only via direct RPC.

This design exposes that capability as an LLM function-call tool (`get_correlation`) so an AI assistant can, during a session, resolve "everything linked to this call/session" and reason over it. The primary audience is **internal diagnostic AI** (an engineer-facing assistant analyzing what happened in a flow), not customer-facing voicebots.

## 2. Scope

### In scope (Phase 1)

- New tool `get_correlation` in bin-ai-manager: tool name constant, definition, dispatch, handler.
- New requesthandler client `TimelineV1ResourceCorrelationGet` in bin-common-handler.
- New RPC contract types in bin-timeline-manager `models/event` (request is just the id via path; response reuses existing `PublisherGroup`/`CorrelatedResource`).
- Handler produces a **human-readable text summary** (not raw UUID dump) for the LLM, following the `search_knowledge` precedent.
- `RunLLM: true` so the model can chain (e.g. get_correlation then get_aicall_messages).

### Out of scope (Phase 1)

- Customer-facing voicebot use. The tool is positioned as internal/diagnostic via its description; not added to any default customer assistant config.
- Fan-out to fetch full resource bodies. The tool returns the correlation graph summary only; the LLM can call other tools (get_aicall_messages, etc.) to drill in.
- Changes to the timeline-manager correlation query/endpoint (already shipped).

### Rationale

- Reuses the shipped timeline correlation endpoint and the established tool pattern (`search_knowledge` calls `reqHandler.RagV1RagQuery`, formats a text result, RunLLM:true). Low risk, all proven patterns.
- Positioned internal-only because a correlation graph (resource IDs) is meaningless and potentially sensitive to an end customer.

## 3. CPO Positioning Note

A correlation graph answers "what resources are linked to this flow" — an operator/debugging question. For a customer on a call it is noise and a PII/info-exposure surface. Therefore:
- The tool description's WHEN TO USE frames it as diagnostic ("analyze what happened in this call/session").
- It is NOT added to the default tool whitelist for customer assistants; it is opt-in per AI config.
- Output is a concise text summary, never raw internal IDs presented as user-facing content.

## 4. Tool Definition

### Name

`bin-ai-manager/models/tool/main.go`:
```go
ToolNameGetCorrelation ToolName = "get_correlation"
```
Add to `AllToolNames`.

`bin-ai-manager/models/message/tool.go`:
```go
FunctionCallNameGetCorrelation FunctionCallName = "get_correlation"
```

### Definition (`toolhandler/definitions.go`)

```go
{
    Name:   tool.ToolNameGetCorrelation,
    RunLLM: true,
    Description: `Retrieves all resources linked to the same call flow (activeflow) as a given resource.

WHEN TO USE:
- Diagnosing what happened in a call/session: "what's linked to this call?", "show me everything in this flow"
- You have a resource id (call, aicall, recording, transcribe, etc.) and need the related resources
- Building a picture of a session's components for analysis

WHEN NOT TO USE:
- Normal customer conversation (this is a diagnostic tool, not customer-facing information)
- You only need the current conversation history (use get_aicall_messages)

run_llm: Always true — respond based on the correlation summary.`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "resource_id": map[string]any{
                "type":        "string",
                "description": "UUID of the resource to correlate. If omitted, the current session's activeflow is used.",
            },
        },
        "required": []string{},
    },
},
```

`resource_id` optional: when omitted, the handler falls back to the current aicall's `ActiveflowID` (so the LLM can ask "what's in this session" without knowing an id). When provided, that id is used.

## 5. RPC Contract (bin-timeline-manager/models/event)

The correlation endpoint is `GET /v1/correlations/<resource_id>` (path param, no body). The requesthandler client needs a typed response. Reuse the shipped response shape; add a contract type mirroring the listenhandler response so requesthandler does not import the listenhandler package.

`models/event/correlation.go` already defines `CorrelatedResource` and `PublisherGroup`. Add the response contract:
```go
// ResourceCorrelationResponse is the RPC contract for GET /v1/correlations/<id>.
type ResourceCorrelationResponse struct {
    ResourceID    uuid.UUID         `json:"resource_id"`
    ResourceFound bool              `json:"resource_found"`
    ActiveflowID  uuid.UUID         `json:"activeflow_id"`
    Truncated     bool              `json:"truncated"`
    Resources     []*PublisherGroup `json:"resources"`
}
```
The listenhandler `response.V1DataResourceCorrelationGet` is structurally identical. Per 1차 리뷰 #3, do NOT keep two independent drifting structs. Define the response once in `models/event` (`ResourceCorrelationResponse`) and make the listenhandler DTO a **type alias**:
```go
// in listenhandler/models/response/correlation.go
type V1DataResourceCorrelationGet = event.ResourceCorrelationResponse
```
This keeps a single source of truth; the eventhandler keeps returning the same shape. (Touching the shipped listenhandler file for a one-line alias is acceptable and prevents silent drift.)

## 6. requesthandler Client (bin-common-handler)

`pkg/requesthandler/timeline_correlation.go`:
```go
// TimelineV1ResourceCorrelationGet sends a request to timeline-manager to get
// the correlation graph for a resource id.
func (r *requestHandler) TimelineV1ResourceCorrelationGet(ctx context.Context, resourceID uuid.UUID) (*tmevent.ResourceCorrelationResponse, error) {
    uri := fmt.Sprintf("/v1/correlations/%s", resourceID.String())

    tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/correlations", requestTimeoutDefault, 0, ContentTypeNone, nil)
    if err != nil {
        return nil, err
    }

    var res tmevent.ResourceCorrelationResponse
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }
    return &res, nil
}
```
Add to the `RequestHandler` interface in `main.go` (timeline-manager section). Regenerate mock.

bin-common-handler admission rule (3+ consumers): the existing TimelineV1* methods already live here, so adding a sibling method is consistent — no new package, just a method on the existing timeline client surface.

## 7. Tool Handler (bin-ai-manager/pkg/aicallhandler/tool.go)

Add to `mapFunctions`:
```go
message.FunctionCallNameGetCorrelation: h.toolHandleGetCorrelation,
```

Handler (mirrors `toolHandleSearchKnowledge`, adds customer-ownership validation per 1차 리뷰 #1/#2 and 2차 리뷰 #1/#2):
```go
// Single masking string used for every "you can't see this" path so that
// genuine-absent, exists-but-not-owned, and ownership-lookup-error are all
// byte-identical (no existence oracle). 2차 리뷰 #2.
const msgCorrelationNotFound = "No events found for this resource."

func (h *aicallHandler) toolHandleGetCorrelation(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
    res := newToolResult(tc.ID)

    var args struct {
        ResourceID string `json:"resource_id"`
    }
    _ = json.Unmarshal([]byte(tc.Function.Arguments), &args) // optional arg

    // ownSession = true when no id supplied -> target is the caller's own
    // activeflow, owned by definition. 2차 리뷰 #1.
    ownSession := args.ResourceID == ""
    resourceID := c.ActiveflowID
    if !ownSession {
        parsed, err := uuid.FromString(args.ResourceID)
        if err != nil {
            fillFailed(res, fmt.Errorf("invalid resource_id"))
            return res
        }
        resourceID = parsed
    }
    if resourceID == uuid.Nil {
        fillFailed(res, fmt.Errorf("no resource_id available"))
        return res
    }

    corr, err := h.reqHandler.TimelineV1ResourceCorrelationGet(ctx, resourceID)
    if err != nil {
        fillFailed(res, fmt.Errorf("correlation lookup failed"))
        return res
    }

    // Resource absent -> canonical not-found.
    if !corr.ResourceFound {
        fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
        return res
    }

    // Resource exists but has no activeflow: there is NO activeflow to validate
    // ownership against. Only disclose this state for the caller's OWN session;
    // for a supplied foreign id, mask as not-found. 2차 리뷰 #1.
    if corr.ActiveflowID == uuid.Nil {
        if ownSession {
            fillSuccess(res, "correlation", resourceID.String(), "This resource exists but is not linked to any call flow.")
        } else {
            fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
        }
        return res
    }

    // Has activeflow: validate ownership via flow-manager (timeline has no customer_id).
    af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, corr.ActiveflowID)
    if err != nil {
        // Mask lookup failure as not-found too — do not reveal that the
        // activeflow exists. 2차 리뷰 #2.
        log.Warnf("Could not verify correlation ownership. resource_id: %s, err: %v", resourceID, err)
        fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
        return res
    }
    if af.CustomerID != c.CustomerID {
        log.Warnf("Cross-customer correlation attempt blocked. session_customer: %s, resource_owner: %s, resource_id: %s",
            c.CustomerID, af.CustomerID, resourceID)
        fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
        return res
    }

    summary := formatCorrelationSummary(corr)
    fillSuccess(res, "correlation", corr.ActiveflowID.String(), summary)
    return res
}
```

**Ownership validation sequence (IDOR 차단, 1차 #2 + 2차 #1/#2):**
1. 인자 생략 → `ownSession=true`, 자기 세션 activeflow (소유 확정).
2. resource_id로 timeline correlation 조회 → activeflow_id 획득.
3. activeflow 없음: `ownSession`일 때만 "미연결" 안내, 아니면 not-found 마스킹.
4. activeflow 있음: `FlowV1ActiveflowGet` → CustomerID 비교. 불일치/조회실패 모두 **단일 `msgCorrelationNotFound` 상수**로 마스킹.

모든 "볼 수 없음" 경로(부재, 타고객, 검증실패, foreign no-activeflow)가 byte-identical 응답이라 존재 oracle이 없다. 잔여 차이는 RPC 1회 latency(2차 #3, accept-with-note: LLM 경계를 통과하면 실측 불가능 수준).

### Text summary formatter

`formatCorrelationSummary` produces an LLM-readable summary, not raw IDs as content:
```
Call flow <activeflow_id> is linked to:
- call-manager: 1 call (call_created, call_progressing, call_hangup), 1 recording
- ai-manager: 1 aicall, 2 messages
- transcribe-manager: 1 transcribe
(truncated: more than 100 resources)   # only if Truncated
```
Per publisher: count resources, list data_type with counts, optionally first event_types. IDs are included compactly so the LLM can chain follow-up tools (e.g. get_aicall_messages with the aicall id), but the prose summary leads.

## 8. Availability gating (1차 리뷰 #1/#4 정정)

1차 리뷰가 두 가지를 바로잡았다.

- **`tool_names:["all"]` 와일드카드 현실 (#1):** `definitions.go`에 정의를 추가하면 `GET /v1/tools`(GetAll)로 노출되어, `tool_names`에 `"all"`을 쓴 모든 AI가 자동으로 이 tool을 갖는다. 즉 "기본 비노출"은 `"all"` 설정 AI에는 성립하지 않는다. 이를 **수용**한다. 단 IDOR 차단(7절 ownership 검증)이 1차 방어선이므로, `"all"` AI가 이 tool을 갖더라도 **자기 고객 소유 리소스만** 조회 가능하다. 보안은 노출 차단이 아니라 ownership 검증으로 보장한다.
- **whitelist.go는 안전장치가 아니다 (#4):** `ConversationSafeTools`/`FilterToolsForConversation`는 아직 pipecat 세션 페이로드에 미배선된 유틸이다. 진단 tool을 거기에 `true`로 넣으면 (a) 지금은 아무 효과 없고, (b) 나중에 배선되면 오히려 모든 conversation AIcall에 자동 노출된다. 따라서 **`ConversationSafeTools`에 추가하지 않는다**(부재로 둔다). 진단 tool은 conversation-safe가 아니다.

정리: 노출 면에서 이 tool은 다른 tool과 동일하게 취급(별도 비밀 게이트 없음)하되, **데이터 접근은 ownership 검증으로 엄격히 제한**한다. `AllToolNames`에는 추가한다(reserved-name 일관성). 이로써 `teamhandler/validation.go` reserved-names와 `main_test.go::TestAllToolNames`도 갱신 대상이 된다(구현 시 반영).

## 9. Observability

`promAIcallToolExecuteTotal` already increments per tool name (label = function name) in the dispatch path — get_correlation is covered automatically. No new metric required.

## 10. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-timeline-manager | Add `ResourceCorrelationResponse` contract type in models/event; alias listenhandler DTO to it | 1 |
| bin-common-handler | Add `TimelineV1ResourceCorrelationGet` requesthandler method (ContentTypeNone GET) + interface + mock | 1 |
| bin-ai-manager | Tool name (+AllToolNames), function-call name, definition, dispatch, handler with ownership validation, summary formatter; unit tests | 1 |

## 11. Implementation Order

1. bin-timeline-manager: add `ResourceCorrelationResponse` to models/event; alias `response.V1DataResourceCorrelationGet = event.ResourceCorrelationResponse`.
2. bin-common-handler: add `TimelineV1ResourceCorrelationGet` + interface entry + regenerate mock; unit test (target queue + URI + method GET, ContentTypeNone).
3. bin-ai-manager: tool name (+ AllToolNames + reserved-name test) + function-call name constants.
4. bin-ai-manager: definition in definitions.go (RunLLM:true, optional resource_id).
5. bin-ai-manager: handler `toolHandleGetCorrelation` (with FlowV1ActiveflowGet ownership validation) + `formatCorrelationSummary` + dispatch entry; unit tests (own session fallback, provided owned id, cross-customer id blocked, not-found, no-activeflow, lookup error, ownership-lookup error, summary formatting).
6. Verification workflow in each touched service (go mod tidy/vendor/generate/test/lint).

## 12. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| Q1 | Contract type vs listenhandler DTO | Single source in models/event + type alias (1차 #3 반영) | CTO |
| Q2 | resource_id default to current activeflow when omitted | Yes — own session always ownership-safe | CPO/CTO |
| Q3 | Include raw IDs in the summary or prose only | Include compact IDs (own-customer only, post-validation) so LLM can chain; lead with prose | CPO/CTO |
| Q4 | RunLLM default | true (chaining is the point of a diagnostic tool) | CPO/CTO |
| Q5 | Customer-facing exposure | Tool may appear in `"all"` configs; security is enforced by **ownership validation**, not exposure gating (1차 #1/#2 반영) | CPO/CTO |
| Q6 | Performance: each call adds FlowV1ActiveflowGet RPC | Acceptable; 1 extra RPC per diagnostic call, low frequency | CTO |

## 13. Review Summary (v1 → v2)

1차 독립 design 리뷰(CHANGES REQUESTED)의 High 항목을 반영했다.

- **[High #2 IDOR] 임의 resource_id cross-customer 조회** → 7절에 `FlowV1ActiveflowGet` 기반 ownership 검증 추가. 불일치 시 not-found와 동일 응답(정보 유출 차단). 대표님 결정으로 resource_id 인자는 유지(방향 2).
- **[High #1 `"all"` 와일드카드]** → 노출은 수용하되 보안은 ownership 검증으로 보장한다고 명시(8절).
- **[Medium #3 계약 타입 중복]** → models/event 단일 정의 + listenhandler DTO type alias (5절).
- **[Medium #4 whitelist 오해]** → `ConversationSafeTools`에 추가하지 않음으로 정정(8절).
- **[Low #5] ContentTypeNone GET, AllToolNames/reserved-name 갱신** 명시(6/8/11절).

## 14. Review Summary (v2 → v3, 2차 리뷰)

2차 design 리뷰(CHANGES REQUESTED)의 High 2건을 반영했다.

- **[High 2차 #1] no-activeflow 분기의 cross-customer 존재 누출** → `ownSession` 플래그 도입. activeflow 없는 리소스는 자기 세션일 때만 "미연결" 상태를 알리고, foreign id는 not-found로 마스킹. (7절)
- **[High 2차 #2] 마스킹 문자열 불일치** → 단일 `msgCorrelationNotFound` 상수로 통일. 부재/타고객/검증실패/foreign-no-activeflow 전부 byte-identical. 검증 RPC 실패도 마스킹(존재 비노출). (7절)
- **[Low 2차 #3] absent vs not-owned 타이밍 차이(RPC 1회)** → accept-with-note. LLM 경계상 실측 불가능 수준. (7절)
- 확인: type alias(JSON-safe), ConversationSafeTools 부재, ContentTypeNone, AllToolNames는 모두 적정. `get_correlation` 이름 신규성은 구현 시 reserved-name 충돌 없음 확인.

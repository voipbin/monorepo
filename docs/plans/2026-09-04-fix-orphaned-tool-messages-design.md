# VOIP-1460: Fix orphaned tool-result messages in Insight multi-turn conversations

**상태:** 설계 초안 (리뷰 대기)
**티켓:** VOIP-1460 (Highest)
**이슈 분석:** 3라운드 (REQUEST CHANGES → APPROVE → APPROVE), 2연속 승인으로 종료. 근본 원인/수정 사이트/회귀 무영향(당시 검증 대상은 1단계 predicate 변경뿐 — 2단계 페어링 재검사는 이후 설계 단계에서 추가됨, §3 정정 참고) 모두 코드 재검증 및 픽스처 실행으로 확인됨.

## 1. 문제

Insight AI가 tool을 한 번이라도 호출한 세션에서 사용자가 다음 메시지를 보내면(`SendReferenceTypeOthers`가 매 턴마다 `startPipecatcall`을 새로 호출해 `getPipecatcallMessages`로 히스토리를 재구성), 아래 3곳의 동일한 필터가:

```python
valid_messages = [m for m in messages if m.get("role") and m.get("content")]
```

tool-call 요청 메시지(`role="assistant"`, `content=""`, `tool_calls` 있음 — `bin-ai-manager/pkg/aicallhandler/tool.go:47`에서 이 형태로 저장됨)를 제거하고, 대응하는 tool-result 메시지(`role="tool"`, content 비어있지 않음)만 고아 상태로 LLM에 전달된다.

## 2. 수정 대상 (3곳, 전부 동일 패턴)

| 파일 | 위치 | 함수 |
|---|---|---|
| `bin-pipecat-manager/scripts/pipecat/run.py` | 450행 | `create_llm_service` (단일 AI 경로) |
| `bin-pipecat-manager/scripts/pipecat/run.py` | 637행 | `init_team_pipeline` (team 경로 1) |
| `bin-pipecat-manager/scripts/pipecat/team_flow.py` | 118-120행 | `build_team_flow` (team 경로 2) |

세 곳 모두 동일하게 수정:

```python
# before
valid_messages = [m for m in messages if m.get("role") and m.get("content")]
# after
valid_messages = [m for m in messages if m.get("role") and (m.get("content") or m.get("tool_calls"))]
```

세 곳 다 고쳐야 하는 이유: 단일 AI 경로(`run.py:450`)와 team 경로(`run.py:637` + `team_flow.py:119`, 둘 다 team 세션에서 실행됨)가 독립적인 필터를 갖고 있어, 두 곳만 고치면 team 기반 Insight 세션은 여전히 깨진 채로 남는다.

## 2-1. 핵심 위험: 단순히 "넓히기만" 하면 새로운 하드 실패가 생긴다 (설계 리뷰 1라운드 REQUEST CHANGES 반영)

위 predicate 변경은 "content 비어있는 assistant tool_calls 메시지를 살린다"는 한쪽 방향으로만 넓힌다. 이게 안전하려면 **살아난 assistant tool_calls 메시지에는 항상 짝이 되는 role="tool" 결과 메시지가 존재해야 한다.** 그런데 그렇지 않은 경로가 실제로 존재한다:

1. **`bin-ai-manager/pkg/aicallhandler/tool.go:47`가 tool 실행 *전에* 먼저 tool-call 요청 메시지를 저장한다.** 이후 `tool.go:84-87`(`mapFunctions`에 없는 unknown tool name → `return nil, fmt.Errorf(...)`)로 조기 반환되면, 그 tool-call 메시지는 **영구히 짝 없는 상태**로 남는다. 지금까지는 이 메시지가 항상 필터에 의해 드롭됐기 때문에 조용히 넘어갔지만, 수정 후에는 이 메시지가 살아나 그 aicall의 **이후 모든 턴이 OpenAI-shaped provider에서 400으로 실패**하게 된다.
2. **`getPipecatcallMessages`(`bin-ai-manager/pkg/aicallhandler/start.go:624` 부근)는 최신 100건만 조회해 역순 정렬한다.** 오래된 쪽에서 절단이 일어나므로, 아주 긴 대화에서 pair가 윈도우 경계에 걸리면(호출 메시지가 잘려나가고 결과 메시지만 남는 경우) 동일한 고아 문제가 **Python 필터와 무관하게** 재발할 수 있다.

반대 방향(role="tool" 메시지가 비어서 필터에 드롭되는 경우)은 발생하지 않는다 — `tool.go:154-159`(`toolCreateResultMessage`)가 항상 `messageContent`를 `json.Marshal`하므로 결과 메시지의 content는 최소 `{"tool_call_id":"..."}`를 포함해 절대 비지 않는다. 이 불변식이 "tool 결과 쪽은 그대로 둬도 된다"는 판단의 근거다.

### 대응: Python 쪽 페어링 정합성 재검사(1차 방어선) + Go 쪽 소스 수정(2차 방어선)

**1차 방어선 (Python, 3개 필터 사이트 전부에 적용, 신규 공유 헬퍼):** predicate로 살릴 메시지를 고른 뒤, 페어링이 실제로 완결된 것만 최종적으로 남기는 정합성 재검사를 추가한다. 새 파일 `bin-pipecat-manager/scripts/pipecat/message_filters.py`(기존 `common.py`처럼 `run.py`/`team_flow.py` 양쪽이 import하는 얕은 유틸 모듈 — `common.py`는 순수 설정값만 담고 있어 로직을 넣기에 맞지 않음):

```python
def filter_valid_messages(messages):
    """Keep role+content-or-tool_calls messages, then drop any tool_calls /
    tool-result that isn't actually paired. A tool-call-request message
    survives only if EVERY one of its tool_calls has a matching role="tool"
    result somewhere in the candidates; a role="tool" message survives only
    if its tool_call_id belongs to a tool-call-request message that ALSO
    survives that same check. Both directions are decided against the same
    `kept_assistant_call_ids` set -- there is exactly one place (the loop
    below) where a message is appended to the output, so there is no way
    for the two halves of a pair to be judged by different criteria. This is
    what makes the fix safe regardless of WHY a pair might be incomplete (an
    unknown-tool-name error that never got recorded, an old-side window
    truncation cut, or anything else not yet discovered) -- see VOIP-1460
    design doc section 2-1.

    (This function went through two broken drafts in design review before
    reaching this shape -- both broke by emitting output from more than one
    loop/list, which let a message dropped in one place resurface via a
    stray `else: append` in another. The fix is structural: compute
    `kept_assistant_call_ids` first with NO side effect on the output, then
    emit the final list from a SINGLE loop that is the only place `append`
    is ever called.)
    """
    candidates = [m for m in messages if m.get("role") and (m.get("content") or m.get("tool_calls"))]
    result_ids = {m.get("tool_call_id") for m in candidates if m.get("role") == "tool"}

    # Pass 1: compute-only, no emission. Which assistant tool_calls entries
    # are answerable (every one of that message's ids has a matching
    # candidate result)? Conservative: if ANY tool_call in a message lacks a
    # matching result, none of that message's ids are added -- a provider
    # that requires every tool_calls entry to be answered would 400 on a
    # partial match anyway, and partial-answer messages are not something
    # this codebase currently produces (each ToolHandle call carries exactly
    # one ToolCall -- tool.go:39-47).
    kept_assistant_call_ids = set()
    for m in candidates:
        if m.get("role") == "assistant" and m.get("tool_calls"):
            ids = [tc.get("id") for tc in m["tool_calls"]]
            if all(cid in result_ids for cid in ids):
                kept_assistant_call_ids.update(ids)

    # Pass 2: the ONLY place output is emitted. Re-applies the same
    # completeness check to assistant tool_calls messages (against the now-
    # final `kept_assistant_call_ids`, not the provisional `result_ids`),
    # and checks role="tool" messages against that identical set -- so both
    # halves of a pair are always judged by the same criterion, in the same
    # loop, with no append site outside this loop.
    final_messages = []
    for m in candidates:
        role = m.get("role")
        if role == "assistant" and m.get("tool_calls"):
            if all(tc.get("id") in kept_assistant_call_ids for tc in m["tool_calls"]):
                final_messages.append(m)
            # else: drop -- unpaired tool-call request, would 400 downstream
        elif role == "tool":
            if m.get("tool_call_id") in kept_assistant_call_ids:
                final_messages.append(m)
            # else: drop -- orphaned tool result, its call was never kept
        else:
            final_messages.append(m)
    return final_messages
```

순서는 `candidates`를 그대로 한 번 순회해 만들어지므로 원래 순서가 유지된다. (설계 리뷰 3라운드에서 재확인: 반례 `[A(tool_calls=[c1,c2]), T1(c1)]`을 추적하면 `result_ids={c1}` → pass 1에서 `all(["c1","c2"] in {c1})`가 거짓이라 `kept_assistant_call_ids`는 빈 집합으로 남고, pass 2에서 A는 같은 조건으로 다시 드롭, T1은 `"c1" not in set()`으로 드롭 — 결과 `[]`, 둘 다 제거됨을 확인.)

세 사이트 모두 이 함수를 호출하도록 교체:

```python
# run.py:450, run.py:637, team_flow.py:119 -- 전부 동일하게
valid_messages = filter_valid_messages(messages)  # (또는 llm_messages)
```

**정정(설계 리뷰 6라운드): 아래 문장은 5라운드에서 §3/§5에 반영한 정정과 모순되므로 폐기.** `filter_valid_messages`의 1단계(predicate)만은 원래 predicate의 상위집합이라 §3 표의 predicate-only 결과와 일치하지만, **2단계(페어링 재검사)까지 포함한 전체 함수는 원래 predicate의 상위집합이 아니다** — `role="tool"` 메시지가 있지만 짝이 되는 `tool_calls`가 없는 기존 픽스처(`test_tool_role_messages_preserved`)가 실제로 있고, 그 메시지는 2단계에서 드롭된다(§3 정정, §5-9 참고). 즉 "기존 픽스처 중 `tool_calls`를 가진 것이 없다"는 관찰은 맞지만 거기서 "2단계가 기존 픽스처에 영향 없다"는 결론은 틀렸다 — 영향을 받는 조건은 `tool_calls`의 존재가 아니라 `role="tool"` 메시지의 존재이기 때문이다. `role="tool"` 메시지가 전혀 없는 나머지 3개 픽스처는 실제로 영향이 없다(§5-4 참고).

**2차 방어선 (Go, 근본에서 차단):** `tool.go:84-87`(83이 아니라 84행)의 unknown-tool-name 분기를 수정해, 조기 반환 전에 실패 결과 메시지를 반드시 기록한다. 기존 헬퍼 `newToolResult`/`fillFailed`(`tool.go:167-174`)를 재사용해 다른 모든 tool 실패와 동일한 `Result: "failed"` 값(신조어 "failure" 금지 — LLM에 그대로 전달되는 필드라 기존에 없던 값을 새로 만들면 안 됨)을 쓰고, `fmt.Errorf(errMsg)`(비-상수 포맷 문자열은 `go vet`의 printf 체크에 걸림, 이 저장소에 그런 패턴이 0건)가 아니라 이미 import된 `stderrors`(`tool.go:6`)의 `stderrors.New`를 쓴다:

```go
} else {
    log.Debugf("unknown tool call: %s", tool.Function.Name)
    errMsg := fmt.Sprintf("unknown tool call: %s", tool.Function.Name)
    failContent := newToolResult(tool.ID)
    fillFailed(failContent, stderrors.New(errMsg))
    if _, errRecord := h.toolCreateResultMessage(ctx, c, tool, failContent, toolCallActiveAIID); errRecord != nil {
        log.WithError(errRecord).Error("could not record the failure result message for the unknown tool call")
    }
    return nil, stderrors.New(errMsg)
}
```

이렇게 하면 "unknown tool name" 경로에서도 tool-call 요청과 결과가 항상 페어를 이뤄 저장되므로, 애초에 고아가 생기지 않는다. (`toolCreateResultMessage` 자체가 실패하는 이중 실패 케이스는 원천적으로 방어 불가능하지만, Python 쪽 1차 방어선이 그 경우에도 해당 tool-call 메시지를 드롭해 안전하게 처리한다.)

**중요 — `conftest.py`의 모듈 스텁 목록에 `message_filters`를 절대 추가하지 않는다.** `conftest.py:90-97`은 `run.py`/`team_flow.py`가 import하는 로컬 모듈들(`common`, `tools`, `task`, `routing_llm/tts/stt`, `team_flow`)을 `_make_mock_module(...)`로 전부 MagicMock 스텁 처리한다. `message_filters`를 신규 로컬 import로 추가하면서 이 관례를 그대로 따라 스텁 목록에 넣으면, §5-7/8/9의 통합 테스트에서 `filter_valid_messages`가 실제 로직이 아니라 MagicMock이 되어 `call_args[1]["messages"]` 검증이 겉보기엔 통과하지만 실제로는 아무것도 검증하지 못하는 상태가 된다. 구현 시 이 함정을 명시적으로 피할 것 — `message_filters`는 스텁 대상이 아니다.

두 방어선의 관계: Go 수정은 "새로운 고아를 만들지 않는다", Python 수정은 "어떤 이유로든 이미 고아가 됐다면 안전하게 제거한다" — 후자가 최종 안전망이므로 Go 수정이 누락되거나 다른 미지의 경로로 고아가 생겨도 프로덕션이 깨지지 않는다.

## 3. 회귀 안전성

이슈 분석 3라운드에서 기존 픽스처 전체를 **1단계 predicate만**(2-1의 페어링 재검사 없이) 실행해 확인한 결과:

| 테스트 | old kept | new kept (predicate만) | 동일 여부 |
|---|---|---|---|
| `test_openai_filters_invalid_messages` | 2 | 2 | 동일 |
| `test_malformed_messages_filtered_out` | 2 | 2 | 동일 |
| `test_none_values_in_messages_filtered_out` | 1 | 1 | 동일 |
| `test_tool_role_messages_preserved` | 3 | 3 | 동일 (predicate 단계까지만) |

`{"role":"user","content":""}` 같은 기존 "빈 content는 버려야 하는" 픽스처는 `tool_calls`가 없으므로 새 predicate에서도 그대로 제거된다. 1단계(predicate)는 순수하게 넓히기만 하며(`content`가 falsy이고 `tool_calls`가 truthy인 경우만 추가로 살림), 기존 픽스처 중 그런 조합은 하나도 없다.

**정정(설계 리뷰 5라운드): 위 표는 `filter_valid_messages` 전체(2단계 페어링 재검사 포함)가 아니라 1단계 predicate만 실행한 결과다. `test_tool_role_messages_preserved`는 전체 함수를 통과시키면 결과가 달라진다** — 이 테스트의 픽스처는 `role="tool", tool_call_id="call_123"` 메시지가 있지만 그 짝이 되는 `role="assistant", tool_calls=[{"id":"call_123",...}]` 메시지가 **없다**(원래 assistant 메시지는 `content: "Transferring now."`뿐, `tool_calls` 없음). 2단계에서 `result_ids={"call_123"}`이지만 `kept_assistant_call_ids`는 빈 집합(assistant tool_calls 메시지 자체가 없으므로)이라, 이 tool 메시지는 **고아로 드롭된다** — old 3 kept, 전체 함수로는 **new 2 kept**, "동일" 아님.

이건 버그가 아니라 §5 테스트 3(`test_drops_orphaned_tool_result`)이 정확히 검증하는 그 동작이 이 기존 픽스처에도 똑같이 적용된 것뿐이다(고아 tool 결과는 항상 드롭돼야 한다는 게 이 함수의 핵심 목적). 즉 §4의 "거짓 안전망" 정정과 별개로, 이 픽스처 자체가 **애초에 불완전한 페어(고아 tool 메시지)를 담고 있었다** — 원래 테스트 작성자의 의도("content 있는 tool 결과가 보존된다")를 살리려면 픽스처를 완전한 페어로 고쳐야 한다. §5-9에서 이를 반영.

## 4. 기존 테스트의 오귀속 정정

`test_team_flow.py::test_tool_role_messages_preserved`는 `run.py`가 아니라 `team_flow.py`의 필터를 검증하는 테스트이며(`build_team_flow`를 호출), 그마저도 assistant 메시지의 content를 `"Transferring now."`(비어있지 않음)로 채워서 테스트하기 때문에 프로덕션에서 실제로 나오는 `content=""` 모양을 전혀 재현하지 못하는 거짓 안전망이다. 저장소 전체(`test_run.py`, `test_team_flow.py` 등 7개 테스트 파일)에 `tool_calls` 키를 가진 메시지를 만드는 테스트가 **0건** — 이 경로는 지금 전혀 커버되지 않는다.

## 5. 신규/수정 테스트 계획 (파일/패치 대상/어서션 대상까지 확정)

프로덕션 shape(`content=""` + `tool_calls` 채워진 assistant 메시지 + 그 뒤를 잇는 `role="tool"` 결과 메시지, `tool_call_id`로 페어링)과, 페어링이 깨진 케이스(2-1의 정합성 재검사) 양쪽을 커버한다.

### `message_filters.py`용 순수 단위 테스트 (신규 파일 `test_message_filters.py`)

의존성 없는 순수 함수이므로 mock 없이 직접 테스트:

1. `test_preserves_paired_tool_call` — `content=""`+`tool_calls` assistant + 페어인 `role="tool"` 메시지 → 둘 다 남고 순서 유지.
2. `test_drops_unpaired_tool_call` — `tool_calls`는 있지만 매칭되는 `role="tool"` 결과가 리스트에 없는 assistant 메시지 → 드롭됨 (2-1의 케이스 1: unknown-tool-name 조기 반환).
3. `test_drops_orphaned_tool_result` — `role="tool"` 메시지는 있지만 매칭되는 **살아남은(surviving)** assistant tool_calls가 없음(윈도우 절단 시뮬레이션) → 드롭됨 (2-1의 케이스 2).
4. `test_existing_fixtures_unaffected` — §3 표의 4개 기존 픽스처 중 **`role="tool"` 메시지가 없는 3개**(`test_openai_filters_invalid_messages`, `test_malformed_messages_filtered_out`, `test_none_values_in_messages_filtered_out`)만 그대로 넣어 `filter_valid_messages` 전체 실행 결과가 predicate-only 결과와 동일한지 확인. `test_tool_role_messages_preserved`의 픽스처는 여기서 **제외** — 그 픽스처는 애초에 불완전한 페어(고아 `role="tool"` 메시지)를 담고 있어 전체 함수에서는 다른 결과가 나오는 게 맞는 동작이다(§3 정정, §5-9 참고).
5. **`test_partial_pairing_drops_both`(설계 리뷰 2라운드에서 추가 — 두 pass가 candidate 기준이 아니라 surviving 기준으로 일관되게 동작하는지 확인하는 핵심 회귀 테스트)** — `tool_calls`가 2개(`c1`, `c2`)인 assistant 메시지 + `c1`만 응답하는 `role="tool"` 메시지 하나를 입력. 기대 결과: **둘 다 드롭됨** — pass 1은 계산 전용이라 아무것도 드롭하지 않지만, `c2`가 미응답이라 `c1`/`c2` 모두 `kept_assistant_call_ids`에 들어가지 못한다. pass 2에서 assistant 메시지는 `all(tc.id in kept)`가 거짓이라 드롭되고, `c1`의 tool 결과도 `"c1" not in kept`라 함께 드롭된다. (초안에서 실제로 이 케이스가 깨져 있었음 — pass 2가 `candidates`/`result_ids` 기준으로 재검사해 `c1` 결과가 혼자 살아남는 버그가 리뷰에서 발견됨. 이 테스트가 그 버그의 mutation-check 역할을 한다.)
6. mutation-check: pass 1의 `all(cid in result_ids for cid in ids)` 조건을 무조건 `True`로 바꿔보고 테스트 2/5가 실패하는지 확인. 그리고 설계 리뷰에서 실제로 발생했던 두 버그를 각각 재현하는 mutation도 반드시 포함: (a) pass 2의 assistant 분기 조건을 제거하고 `else: final_messages.append(m)`으로 무조건 추가하도록 바꿔보고(2라운드에서 실제로 있었던 버그 형태) 테스트 5가 실패하는지, (b) pass 2를 `for m in candidates: ... else: final_messages.append(m)` 형태로 바꿔 tool_calls 분기를 아예 놓치게 해보고(3라운드에서 실제로 있었던 버그 형태) 테스트 5가 실패하는지. 세 mutation 모두 확인 후 복원.

### 각 호출 사이트가 `filter_valid_messages`를 실제로 쓰는지 확인하는 통합 테스트

7. **`create_llm_service` (run.py:450)** — `test_run.py`에 추가. 기존 `test_openai_filters_invalid_messages`(183행)와 동일한 패턴(`@patch("run.LLMContext")`, `mock_context.call_args[1]["messages"]` 검증)을 미러링 — 이 패턴이 이미 검증된 하네스이므로 새로 고민할 필요 없음. 프로덕션 shape 페어를 입력에 포함시켜 결과에 둘 다 남는지 확인.
8. **`init_team_pipeline` (run.py:637)** — `test_run.py`가 아니라 **`test_init_pipeline.py`에 추가**. `test_init_pipeline.py:238-293`의 `test_init_team_pipeline_swaps_flowmanager_llm_to_router`가 이미 `init_team_pipeline`의 성공 경로를 끝까지 통과시키는 동작하는 async 하네스(`RoutingLLMService`/`Pipeline`/`PipelineTask`/`build_team_flow`/`task_manager` patch 포함)를 갖고 있으므로, 그 하네스에 `patch("run.LLMContext")`를 추가해 `mock_context.call_args[1]["messages"]`를 검증하는 신규 테스트를 그 파일에 둔다. `test_run.py`에 두면 안 됨(하네스 부재).
9. **`build_team_flow` (team_flow.py:119)** — `test_team_flow.py`에 위치. `conftest.py:97`이 `team_flow` 모듈 자체를 MagicMock으로 스텁하고 `test_team_flow.py:9`가 `sys.modules`에서 이를 pop해 실제 모듈을 로드하는 구조를 그대로 따른다(새 테스트를 다른 파일에 두면 stub이 적용돼 실제 코드가 실행되지 않음). 기존 `test_tool_role_messages_preserved`(173행)의 픽스처가 §3 정정에서 밝혀진 대로 애초에 불완전한 페어(고아 `role="tool"` 메시지, 짝이 되는 assistant `tool_calls` 메시지가 없음)였으므로, **픽스처를 완전한 페어로 고쳐서** 원래 검증 의도("content가 있는 tool 결과가 보존된다")를 살린다 — assistant 메시지를 `{"role":"assistant","content":"Transferring now.","tool_calls":[{"id":"call_123","type":"function","function":{"name":"transfer","arguments":"{}"}}]}`로 바꿔 `tool_calls`를 추가하고(이름은 그대로 두거나 `test_tool_role_messages_preserved_with_content`로 개명), `task_messages == llm_messages`(3개 전부 유지, 순서 포함) 어서션은 그대로 유지한다. 그리고 프로덕션 shape(`content=""`+`tool_calls`)을 검증하는 `test_tool_role_messages_preserved_with_empty_content_and_tool_calls`를 신규로 추가한다. 둘 다 `start_node["task_messages"]`에 대해 순서 포함 어서션.

### mutation-check (3개 통합 테스트 전부)

10. 각 사이트에서 `filter_valid_messages(...)` 호출을 원래의 `[m for m in messages if m.get("role") and m.get("content")]`로 되돌리고, 해당 사이트의 신규 통합 테스트(7/8/9)가 실제로 실패하는지 확인한 뒤 복원. PR 본문에 실패 출력(assert 메시지)을 붙인다 — "확인했다"는 서술만으로는 불충분(§7에서 재강조).

## 6. 스코프 밖 (별도 티켓/후속) 및 미검증 리스크

- **Anthropic 어댑터 분기 누락** (`create_llm_service`의 `else: raise ValueError`): 이 티켓과는 독립적인 사전 결함. 별도 확인 후 필요 시 하위 티켓 분리 (VOIP-1460 원문의 "해야 할 일 4"). 이번 PR에서는 손대지 않음.
- **DB 조회로 실사용 영향 범위 확인**(`engine_model LIKE 'openai.%' OR 'grok.%'` AND `type='insight'`): 이 세션에서 프로덕션 DB 접근 권한이 없어 수행 불가. 코드 수정 자체는 provider 무관하게 안전하므로(모든 provider가 이득을 보거나 최소한 손해는 없음) 이 조사가 수정 자체를 막지는 않음. 참고용으로 후속 확인 필요성만 기록.
- **VOIP-1455(`emit_info_card`)와의 상호작용**: `bin-ai-manager/pkg/aicallhandler/start.go:679`의 "defense-in-depth, not currently load-bearing" 주석이 가리키는 tool_calls Arguments 뉴트럴화 로직은, 이번 수정으로 Python 필터가 더 이상 해당 메시지를 드롭하지 않게 되면 **실제로 load-bearing이 됨**. 즉 이 수정이 배포된 후에는 VOIP-1455의 `emit_info_card` 호출의 tool-call 메시지가 실제로 LLM에 재주입되고, 그때 start.go의 뉴트럴화(Arguments를 placeholder로 치환)가 처음으로 실질적 방어선이 된다. 이 상호작용을 구현 단계에서 회귀 테스트로 확인(뉴트럴화가 여전히 동작하는지 — VOIP-1455 PR #1264가 이미 이 로직을 구현해 두었으므로, 새 필터 배포 후에도 해당 로직이 그대로 유효한지 재확인만 필요, 로직 자체를 변경하지 않음).
- **Gemini의 "크래시 없이 폴백" 근거는 미검증 리스크로 재분류한다.** 이 판단은 프로덕션 로그(pipecatcall `965e27bf-...`)로만 확인됐고, `pipecat` 어댑터 코드 자체는 이 환경에 라이브러리가 설치돼 있지 않아 검사하지 못했다. 이번 수정 후 Gemini 세션은 **이전에 한 번도 받아본 적 없는** `tool_calls`가 채워진 assistant 메시지를 처음 받게 된다 — universal `LLMContext` + `GeminiLLMAdapter`가 OpenAI shape의 `tool_calls`/`role="tool"`을 Gemini의 functionCall/functionResponse로 정확히 변환해준다는 전제가 이번 수정으로 처음 실제 트래픽에서 검증받는 것이다. 구현 단계에서 pipecat 어댑터 소스(`pipecat/adapters/services/gemini_adapter.py`)를 최소한 한 번 읽어 이 변환 경로가 실제로 존재하는지 확인하고, 가능하면 배포 후 Gemini insight 세션에 대한 스모크 확인(로그 관찰)을 수행한다. 코드 수정 자체를 막는 조건은 아니나, "provider 무관하게 안전"이라는 §6의 기존 서술을 "OpenAI/Grok/team 경로는 코드로 확정, Gemini는 어댑터 변환 경로 확인 후 확정, Anthropic은 스코프 밖"으로 정정한다.
- **와이어 경로 근거(load-bearing, 명시 확인됨)**: 이 수정이 의미가 있으려면 Go에서 저장된 `tool_calls`가 실제로 Python `run.py`/`team_flow.py`까지 도달해야 한다. `scripts/pipecat/main.py:17-29`의 pydantic `Message`/`ToolCall` 모델이 `tool_calls: Optional[List[ToolCall]]`/`tool_call_id`를 선언하고 있고, `main.py:128`이 `[m.model_dump() for m in req.llm_messages]`로 매핑하므로 `tool_calls` 키는 값이 없을 때도 `None`(falsy)으로, 있을 때는 채워져서 항상 dict에 존재한다. Go `message.ToolCall`(`id`/`type`/`function{name,arguments}`)과 필드 shape이 일치함도 확인됨(`bin-ai-manager/pkg/aicallhandler/start.go:621-701`의 `getPipecatcallMessages`가 이 매핑을 수행). 이 경로가 끊겨 있었다면 이번 수정 전체가 no-op이 됐을 것 — 이미 확인됐으므로 리스크 아님, 근거로만 남김.

## 7. 구현 순서

1. `bin-pipecat-manager/scripts/pipecat/message_filters.py` 신규 작성 (§2-1), `test_message_filters.py` 순수 단위 테스트 5종 + mutation-check
2. `run.py:450` 교체 + 통합 테스트(§5-7) 추가, mutation-check
3. `run.py:637` 교체 + 통합 테스트(§5-8, `test_init_pipeline.py`) 추가, mutation-check
4. `team_flow.py:119` 교체 + 통합 테스트(§5-9, `test_team_flow.py`) 추가, mutation-check
5. `bin-ai-manager/pkg/aicallhandler/tool.go`의 unknown-tool-name 분기 수정(§2-1 2차 방어선) — Go 쪽 verification workflow(`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`) 전체 실행
6. Python 쪽 전체 테스트 스위트 로컬 실행 (아래 §7-1 참고) — `role="tool"` 메시지가 없는 3개 픽스처는 무변화인지, `test_tool_role_messages_preserved`는 §5-9의 페어 보정을 거친 뒤 3개 유지로 통과하는지 재확인
7. `start.go:679` 주변 VOIP-1455 뉴트럴화 로직에 대한 기존 Go 테스트가 여전히 통과하는지 확인(회귀 없음 재확인, 로직 변경 없음)
8. Gemini 어댑터 변환 경로 소스 확인(§6)
9. PR 생성 — mutation-check 실패 출력 스크린샷/로그를 PR 본문에 첨부

### 7-1. Python 테스트는 CI에 포함되지 않음 — 로컬 실행 필수, PR 본문에 결과 명시

`.circleci/config_work.yml:539`의 `bin-pipecat-manager-test` job은 주석 처리돼 있고, 살아있더라도 그 job은 Go 테스트(`go-test-pipecat-manager`)만 돈다. `scripts/pipecat/pyproject.toml`에 `pytest`/`pytest-asyncio`가 dependency로 등록돼 있지 않다. 즉 **이번에 추가하는 Python 회귀 테스트는 CI가 자동으로 잡아주지 않는다.**

로컬 실행 명령(정확한 형태는 구현 시 `pyproject.toml`/`requirements*.txt`를 확인해 확정하되, 현재 파악된 기본형):

```bash
cd bin-pipecat-manager/scripts/pipecat
python -m pytest -q test_message_filters.py test_run.py test_init_pipeline.py test_team_flow.py
```

PR 본문에 이 실행의 전체 출력(pass 개수)과, §5-10 mutation-check의 실패 출력을 반드시 포함한다 — "확인했다"는 서술만으로는 이 저장소의 테스트 표준(코드 리뷰 루프가 실제로 검증 가능한 형태)을 충족하지 못한다.

## 8. 완료 기준

- `message_filters.py` 신규 작성, 3개 필터 사이트 전부 이를 사용하도록 수정
- `tool.go` unknown-tool-name 분기 수정(2차 방어선)
- 순수 단위 테스트 5종 + 사이트별 통합 테스트(run.py:450/637, team_flow.py:119 사이트당 각 1개 신규 + team_flow.py 사이트는 기존 픽스처 페어 보정 1개 추가) — mutation-check는 신규 프로덕션-shape 테스트에만 의미 있음(페어 보정만 한 기존 테스트는 원래 predicate로 되돌려도 통과하므로 mutation-check 대상 아님), 전부 실패 출력 PR에 첨부
- 기존 pipecat 테스트 스위트 전체 통과(회귀 없음), 로컬 실행 결과 PR 본문에 명시(§7-1)
- bin-ai-manager 쪽 verification workflow 전체 통과(go test/vet/lint), VOIP-1455 관련 테스트 회귀 없음 확인
- Gemini 어댑터 변환 경로 소스 확인 완료(§6)
- 코드 리뷰 루프 3라운드, 2연속 승인

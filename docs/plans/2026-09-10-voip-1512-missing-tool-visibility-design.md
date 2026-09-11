# VOIP-1512 설계: 미등록 tool 호출 런타임 가시화 + 알림

작성: 2026-09-10
상태: 설계 단계 (설계 리뷰 루프 대상)
선행: 이슈 분석 2연속 APPROVE (`agent-hermes/docs/voip-1512-issue-analysis-2026-09-10.md`)

## 1. 목표

LLM이 자신에게 advertised되지 않은(등록 안 된) tool을 호출하는 사건을, 운영자가 즉시
알 수 있도록 (a) 우리가 통제하는 ERROR 로그로 승격하고, (b) 그 로그를 메트릭으로 변환해
(c) 기존 Prometheus/Alertmanager Discord 파이프라인으로 알림한다.

비목표: prompt<->tool_names 저장 시점 검증(B2), Insight tool_names 기본값 정책(B3),
Python runner에 Prometheus 익스포터 신설.

## 2. 스코프 (2개 PR)

| PR | 레포 | 변경 |
|---|---|---|
| PR 1 | monorepo | bin-pipecat-manager runner: 미등록 tool 호출 시 우리 ERROR 구조화 로그 |
| PR 2 | monorepo-etc | infra-loki(Alloy stage.metrics: 로그->카운터) + infra-prometheus(alert-rules: 알림) |

PR1의 ERROR 로그 문자열이 source of truth. PR2는 그 포맷에 의존하므로 PR1을 먼저 확정.

## 3. PR 1 설계 (monorepo, Python runner)

### 3.1 후킹 지점

pipecat LLM 서비스의 공개 이벤트 `on_function_calls_started(service, function_calls)`를
등록한다. 이 이벤트는 dispatch 직전(llm_service.py:1272)에 발화되며, handler 조회/
missing 라우팅(1281-1288)보다 **먼저** 발화되므로 미등록 tool 호출도 포함된다.
`user_visible_calls` 필터(1268-1270)는 `CANCEL_ASYNC_TOOL_NAME`만 제외한다.

각 `function_calls` 항목은 `FunctionCallFromLLM`(frames.py:1180): `function_name`,
`tool_call_id`, `arguments`, `context`. 핸들러 시그니처는 `(service, function_calls)`로
발화한 LLMService 인스턴스(`service`)를 받는다.

### 3.1a 미등록 판별 방법 (service._functions 기준 -- 권고)

핸들러는 `function_name`이 **그 발화 서비스에 실제 등록된 handler 집합에 없는지**로
판별한다: `function_name not in service._functions`.

- 이것은 pipecat이 dispatch에서 쓰는 판정(llm_service.py:1282,
  `if function_call.function_name in self._functions.keys()`)의 부정과 동일하다. 즉
  pipecat이 곧 `_missing_function_call_handler`로 보낼 그 호출을 같은 기준으로 집는다.
  (엄밀히 pipecat은 1284행 `elif None in self._functions.keys()` catch-all 와일드카드도
  검사하나, 우리 runner는 `register_function(None)`/`_functions[None]`을 전혀 등록하지
  않으므로 이 코드베이스에서 None 분기는 존재하지 않고 오탐 위험 없음. 구현 시 이 전제
  -- None 미등록 -- 를 assert/확인.)
- 이 방식은 single/team의 tool 등록 소스 차이(single=tool_names 인자, team=member
  tools+transitions)를 우리가 따로 추적할 필요를 없앤다. 서비스가 자기 `_functions`에
  무엇을 가졌는지가 곧 진실이다.
- 대안(context.tools 기반 `_advertised_tool_names`)도 있으나, `_functions` 기준이
  dispatch 판정과 동형이라 더 정확. `_functions`는 private 속성이라 pipecat 버전 업 시
  확인 필요하나, `on_function_calls_started` 자체가 같은 파일의 공개 API라 함께 관리됨
  (구현 시 pinned pipecat 버전에 대해 존재 확인 -- 검증 항목).

### 3.2 등록 위치 (single vs team 경로) -- tool_register 비의존

**Round 1 결함 수정.** `on_function_calls_started` 이벤트는 **실제 pipecat LLMService
인스턴스에서만** 발화된다(RoutingLLMService는 FrameProcessor 래퍼라 자체 발화 안 함,
routing_llm.py:7). 이벤트 등록을 `tool_register`에 묶어서는 **안 된다** -- tool_register는
single 경로에서만 호출되고(run.py:258), team 경로(`init_team_pipeline`)는 이를 전혀
호출하지 않으므로 team 세션이 미등록 tool 가시성을 잃는다(이 티켓이 닫으려는 조용한
실패 표면 그 자체).

따라서 이벤트 핸들러는 **실제 LLMService 인스턴스가 생성되는 지점 각각**에 앵커링한다.
공통 헬퍼 하나를 만든다:

```python
# tools.py (or a small util)
def register_missing_tool_logging(llm_service, pipecatcall_id: str):
    @llm_service.event_handler("on_function_calls_started")
    async def _on_calls(service, function_calls):
        for fc in function_calls:
            if fc.function_name not in service._functions:   # §3.1a
                logger.error(
                    f"[missing_tool] LLM called unadvertised tool. "
                    f"pipecatcall_id={pipecatcall_id} tool_name={fc.function_name}"
                )
```

두 경로에서 실제 LLMService마다 호출한다:

- **single**(run.py `init_single_ai_pipeline`, ~258 tool_register 인근): 단일
  `llm_service`에 대해 `register_missing_tool_logging(llm_service, id)`.
- **team**(run.py `init_team_pipeline`: 멤버 루프 ~614에서 `llm_services[mid]=llm_svc`
  생성, 634 `RoutingLLMService(llm_services)`): RoutingLLMService 생성 직전에
  `for svc in llm_services.values(): register_missing_tool_logging(svc, id)`.
  RoutingLLMService에 위임 메서드를 새로 추가할 필요 없음(내부 svc 직접 순회).

판별이 §3.1a의 `service._functions` 기준이므로 헬퍼는 tool_names 인자가 필요 없다
(서비스 자신이 진실을 안다). single/team의 tool 등록 소스 차이(single=tool_names 인자,
team=member tools+transitions)가 설계에서 사라진다.

구현 시 정확한 API 이름(`event_handler` 데코레이터 vs `add_event_handler`)을 pinned
pipecat 버전에서 확인한다(transport가 이미 `event_handler(...)` 데코레이터 패턴 사용,
run.py:265-268).

### 3.3 ERROR 로그 포맷 (source of truth)

핸들러가 미등록 tool 호출을 감지하면 loguru로 ERROR 1줄:

```
[missing_tool] LLM called unadvertised tool. pipecatcall_id=<id> tool_name=<name>
```

- `pipecatcall_id`: Loki에서 turn/ai 조인 키(스킬의 표준 추적 키). ai_id는 run.py
  컨텍스트에 없어 로그에 직접 넣지 않는다(pipecatcall_id로 조인 가능하므로 충분).
- 고정 접두어 `[missing_tool]`: PR2 stage.metrics 정규식이 이 패턴만 매칭(이중 카운트
  방지 -- pipecat 자체 warning 1517과 문자열이 구분됨).
- 등록 tool은 한 pipecatcall에 여러 개 호출될 수 있으므로 function_calls 순회하며 미등록
  건마다 1줄.

### 3.4 이중 로그 회피

pipecat은 hallucination 케이스를 `logger.warning`(1517)으로 이미 찍는다. 우리 ERROR와
공존하나, PR2 정규식이 `[missing_tool]` 우리 패턴만 매칭하므로 메트릭 이중 카운트 없음.
우리 ERROR는 pipecat 로그보다 구조화(필드 명시)돼 grep/알림에 적합.

## 4. PR 2 설계 (monorepo-etc, 알림 파이프라인)

### 4.1 Alloy stage.metrics (infra-loki)

`loki.process "containers"`(komodo/docker-compose.yml 인라인 alloy config, ~308)에
`stage.metrics` 추가. `[missing_tool]` ERROR 로그를 카운트하는 Prometheus counter 생성:

- metric: `pipecat_missing_tool_total` (counter).
- 라벨: 최소화. `tool_name`을 라벨로 넣으면 카디널리티가 tool 종류로 제한(유한, 8+few)
  되나, hallucinated tool_name은 임의 문자열일 수 있어 카디널리티 폭발 위험. -> **MVP는
  라벨 없이 단일 카운터**(어떤 tool인지는 알림 후 Loki로 확인). (리뷰 논점 B).
- Alloy가 이 메트릭을 Prometheus로 노출하는 경로는 **이미 존재**한다: prometheus.yml에
  `alloy` 스크레이프 잡(`alloy:12345`, line 555)이 이미 있음(Round 1 확인). 즉
  stage.metrics 산출 카운터는 추가 스크레이프 설정 없이 노출됨. (논점 C 해소.)

### 4.2 alert-rules.yml (infra-prometheus)

`docker-compose_monitoring/config/prometheus/alert-rules.yml`에 룰 추가:

- `increase(pipecat_missing_tool_total[10m]) > 0` -> alert.
- **annotation 스키마 필수(Round 1 지적 B)**: alert-rules.yml 헤더가 모든 룰에 정확히
  3개 annotation(`summary`/`impact`/`runbook`)을 하드 요구하며 discord-relay가 이를
  소비한다. 신규 룰도 이 3개를 반드시 채운다. severity/labels도 기존 룰 컨벤션 따름.
  Alertmanager Discord relay가 자동 전달.
- 임계치: 미등록 tool 호출은 항상 설정/prompt 버그 신호이므로 단발도 알린다(rate 아닌
  increase>0). 다만 알림 폭주 방지 위해 `for: 0m` + Alertmanager group_interval 활용.
  (리뷰 논점 D -- 단발 알림 vs 노이즈 균형.)

## 5. 트레이드오프

- pipecatcall_id만 로그(ai_id 없음): run.py 컨텍스트 제약. Loki 조인으로 ai 식별 가능,
  수용.
- 라벨 없는 단일 카운터: hallucinated tool_name 카디널리티 위험 회피. 어떤 tool인지는
  알림 트리거 후 Loki로 2차 확인. MVP로 수용.
- Alloy stage.metrics는 로그 파싱 의존: 로그 포맷 변경 시 정규식 동기 필요. PR1/PR2
  결합도. 접두어 `[missing_tool]`을 안정 계약으로 고정해 완화.

## 6. 리뷰 논점 (설계 리뷰에서 결론낼 것)

- A: **[해결, Round 1]** 이벤트 핸들러를 tool_register에 두면 team 경로 미커버.
  §3.2대로 실제 LLMService 생성 지점 2곳(single + team 멤버별)에 직접 앵커링, 판별은
  §3.1a `service._functions` 기준으로 확정.
- B: **[해결]** 카운터 무라벨 단일 카운터로 확정(hallucinated tool_name 카디널리티 회피).
- C: **[해결, Round 1]** alloy 스크레이프 잡이 prometheus.yml에 이미 존재(§4.1).
- D: 알림 임계치(단발 vs rate) + 노이즈 균형 -- 구현 시 실제 트래픽으로 최종 확정.

## 7. 검증 계획 (구현 시)

- PR1: pytest로 핸들러가 미등록 tool에 ERROR를 찍고 등록 tool엔 안 찍는지. single/team
  경로 모두.
- PR2: alloy config 렌더 후 stage.metrics 정규식이 샘플 로그를 매칭하는지(로컬/테스트).
  alert-rules promtool check.
- 라이브(머지 후 정규 파이프라인): 의도적으로 tool_names에서 빠진 tool을 prompt가 부르게
  해 재현 -> ERROR 로그 -> 카운터 증가 -> Discord 알림 도착 확인.

## 8. Review Log

| Round | Type | Verdict | Note |
|---|---|---|---|
| (이슈분석 1) | Issue | APPROVE | load-bearing 코드 사실 일치, 비차단 2건 |
| (이슈분석 2) | Issue | APPROVE | 정정 반영, 2연속 |
| 설계 1 | Design | REQUEST_CHANGES | §3.2 치명 결함: 이벤트를 tool_register(single 전용)에 두면 team 미커버. §3.1a service._functions 판별 + §3.2 2지점 앵커링으로 재작성. 비차단 B(alert 3-annotation) 반영 |
| 설계 2 | Design | APPROVE | Round 1 결함 해소 코드 검증. 비차단: §3.1a "1:1 동일"이 None catch-all 분기 누락(이 코드베이스 미사용)→정정 반영 |
| 설계 3 | Design | APPROVE | §3.1a/§3.2를 실제 구현 코드(tools.py:37, run.py:260/640-644)와 재대조, 실제 pipecat 1.4.0 환경 pytest 4/4 PASS로 실동작 확인. Round 2에 이은 2연속 APPROVE로 설계 리뷰 게이트 통과 |

**설계 리뷰 게이트: 통과** (Round 2, 3 연속 APPROVE). 코드는 이미 작성/실검증 완료 상태이므로, 다음 단계는 코드 리뷰 루프(최소 3회 + 2연속 APPROVE)로 진행한다.

| 코드 1 | Code | APPROVE | pipecat 1.4.0 실소스(llm_service.py:1282) 대조, single/team 앵커링 정확, sys.modules 격리 안전, 150 passed(무관 기존 2건 실패 제외) |
| 코드 2 | Code | APPROVE | private attr(_functions) 위험 수용 가능(탐지 전용, asyncio.create_task 격리), 중복등록 불가(Go 3개 호출지점 모두 1회성), 로그 포맷 §3.3과 바이트 일치, CI에 pytest 잡 자체가 비활성(VOIP-1356)이라 xdist 무관 |
| 코드 3 | Code | APPROVE | 스코프 오인 없음(VOIP-1510 근본원인 미해결 오인 없음), PR1 단독 머지 가치 인정(트레이드오프 기 인지), git diff 의도치 않은 변경 없음, main과 병합충돌 없음. 3회 연속 APPROVE로 코드 리뷰 게이트 통과 |

**코드 리뷰 게이트: 통과** (Round 1, 2, 3 연속 APPROVE, 최소 3회 요건 충족). 커밋/PR 생성 단계로 진행.

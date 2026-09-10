# VOIP-1510 (2차): Insight 초기 턴 notify_agent 거부 수정

## 상태
- 작성일: 2026-09-10
- 티켓: VOIP-1510 (In Progress)
- 워크트리: .worktrees/VOIP-1510-Fix-insight-initial-turn-notify
- 선행: PR #1283(로깅 계측)은 머지됨. 원래 증상(notify_agent 미등록)은 설정 수정
  (AI tool_names 8개화)으로 라이브 해소+E2E 검증 완료. 본 문서는 그 검증 중 발견된
  **초기 턴 코드 버그**를 다룬다. 이슈 분석 리뷰 2연속 APPROVE 완료.

## 1. 문제 (확정, 리뷰 2연속 APPROVE)

Insight AI를 Case에 붙이는 순간 실행되는 **초기 턴**에서 notify_agent가 구조적으로
거부된다.

인과 사슬 (코드 확정):
1. 패널 Start -> `POST /service_agents/aicalls`(Insight aicall 생성) +
   `POST /aicalls/{id}/listen`.
2. aicall 생성 경로 `startReferenceTypeContactCase` -> `startContactCaseTurn`
   (start.go:596)가 즉시 `startPipecatcall`(start.go:882)로 초기 pipecatcall을
   `c.PipecatcallID`로 시작한다. 이 경로는 `ListenTurnPipecatcallIDAdd`를 호출하지
   않는다 (등록은 listen.go:389 `runListenTurnWithLines`에서 throwaway id로만).
3. 초기 턴의 LLM이 대화 히스토리를 읽고 notify_agent를 호출하면,
   `ToolHandle`(tool.go:52-68)의 `ListenTurnPipecatcallIDIsMember(c.ID,
   pipecatcallID)`가 false -> `listenTurn=false`.
4. `toolHandleNotifyAgent`(tool_insight.go:1410-1419)가 `!listenTurn`이면 거부:
   "notify_agent is only usable while proactively monitoring a call...".

라이브 E2E 증거 (aicall bedaeea9): 초기 턴(09:17:00) notify=failed, 이후 정상 청취 턴
(09:17:15) notify=success. Prometheus buffered 26->30, listen_turn ran 8->10,
listen_notify 1->2.

## 2. 영향 범위 (한정)

- 초기 턴 이후 고객이 더 말하면 다음 청취 턴이 notify_agent를 성공시키므로 무손실.
- **손실 시나리오**: 상담사가 이미 화난 고객의 대화를 다 보고 Insight를 켰는데, 그
  시점 이후 고객이 더 말하지 않는 경우, 초기 턴 히스토리에만 있던 이탈 신호가 영구
  손실. Insight 핵심 가치(이탈 감지 -> 상담사 알림)에 직접 닿는 경로.
  (주의: 이 "손실"은 §3의 "tool 결과 replay 제외"와 무관하다. §3는 옵션 A 적용 시의
  부수효과이고, 여기 §2는 옵션 A로 고치려는 대상 자체다.)

## 3. 설계상 트레이드오프 (코드 확인 후 확정 -- 수용 가능)

초기 턴을 listen-turn으로 등록하면(옵션 A), `ToolHandle`의 `rowOrigin`이
`OriginListenInternal`이 되어(tool.go:74-77) 그 턴의 **기계적 tool-call/tool-result
rows가 향후 Q&A replay에서 제외**된다(start.go:739가 `OriginListenInternal`을
NotEq로 필터). 현재 초기 턴은 `OriginNone`이라 replay에 남는다.

### 코드 확인으로 밝혀진 결정적 사실 (설계 리뷰 Round 1 지적 반영)

이 트레이드오프는 처음 서술보다 훨씬 작다. **notify_agent의 OUTPUT row는
`rowOrigin`과 무관하게 `OriginProactive`로 별개 태깅된다**
(tool_insight.go:1428-1431). `OriginProactive`는 replay 제외 필터(start.go:739,
`OriginListenInternal`만 NotEq)에 걸리지 않는다. 즉:

- 옵션 A 적용 후에도 **이탈 감지 알림 콘텐츠(notify_agent OUTPUT)는 replay에 그대로
  남는다.** Insight 핵심 가치는 전혀 손실되지 않는다.
- replay에서 실제로 빠지는 것은 초기 턴의 **기계적 tool-call/result row**
  (get_contact_profile, get_case_notes, get_conversation_content 등)뿐이다.

따라서 §2의 "알림 신호 손실"과 이 §3의 "tool 결과 replay 제외"는 서로 다른 것이며,
옵션 A는 §2(초기 턴 notify 거부)를 고치면서 알림 콘텐츠는 온전히 보존한다.

### 확정: 트레이드오프 수용 가능

초기 턴이 호출하는 컨텍스트 수집 tool(get_contact_profile/get_case_notes/
get_conversation_content 등)은 **idempotent lookup**이다. 이후 상담사 Q&A 턴에서 AI가
필요하면 재호출할 수 있고, 그 재호출은 Q&A 턴이므로 listenTurn=false ->
OriginNone -> replay에 유지된다. 즉 초기 턴 tool 결과의 replay 제외는 "정보 영구 손실"이
아니라 "재도출 가능한 캐시 상실"이며, models/message/main.go:104-112가 정의한
listen-internal replay-noise 철학과 일관된다.

**결론(확정): 옵션 A의 replay 제외는 수용 가능하다.** 알림 콘텐츠는 OriginProactive로
보존되고, 제외되는 lookup 결과는 재도출 가능하다. 이 결정은 리뷰에 위임하지 않고 본
문서에서 확정한다.

## 4. 수정 옵션

### 옵션 A (권고): 초기 pipecatcall을 listen-turn으로 등록
`startContactCaseTurn`에서 `startPipecatcall`이 반환한 `pc.ID`를
`ListenTurnPipecatcallIDAdd(ctx, c.ID, pc.ID, ttl)`로 등록한다. `pc.ID`는 이미
반환값으로 존재. scope 최소.

- 부수효과: §3의 replay 제외. 설계 리뷰에서 이것이 수용 가능한지 확정 필요.
- 등록 실패 시 처리: listen.go는 등록 실패 시 턴을 skip한다. 초기 턴은 이미 시작된
  상태라 skip 불가 -> 등록 실패를 로그+메트릭으로 남기고 계속 진행(초기 턴의 notify가
  degrade될 뿐, 초기 턴 자체는 살린다). listen.go의 abort와는 다른 처리.

### 옵션 B: notify_agent 거부 가드를 초기 턴에 한해 완화
`toolHandleNotifyAgent`가 listenTurn=false여도, "초기 턴(진짜 Q&A 상담사 질문이 아닌,
자동 시작된 첫 평가)"이면 허용. 단 "초기 턴"과 "상담사 Q&A 턴"을 런타임에 구별하는
새 신호가 필요 -> 복잡. 설계 의도(tool_insight.go:1410-1418 주석: Q&A 턴에 notify가
나가면 상담사 질문 대신 엉뚱한 알림이 감)와 충돌 위험.

### 옵션 C: 초기 턴에서 아예 notify_agent를 노출하지 않고, listen 경로만 첫 평가 담당
초기 startContactCaseTurn을 read-only로 만들고, 첫 이탈 평가는 ProcessListen ->
RunListenTurn이 전담. 가장 크고 아키텍처적. design doc 재작업 수준.

**권고: 옵션 A.** 단 §3 replay 트레이드오프를 설계 리뷰에서 확정하고, 수용 불가로
판정되면 옵션을 재검토한다.

## 5. 예방책 스코프 (오버엔지니어링 게이트 적용)

원래 뿌리("prompt가 참조하는 tool이 tool_names에 없어도 조용히 실패")에 대한 예방책:

- **B1 런타임 가시화** (missing-function-call을 메트릭/경고로): 실제 장애 근거 있음.
  단 bin-pipecat-manager 변경이고 초기 턴 버그(bin-ai-manager)와 다른 축. 이 PR
  스코프에 넣을지는 설계 리뷰 판단. 잠정: **별도 작은 후속으로 분리** 권고(본 PR은 초기
  턴 버그에 집중, 한 PR = 한 논리 변경).
- **B2 저장 시 prompt<->tool_names 정합성 경고**: prompt 자연어에서 tool 참조를
  추출해야 함(오탐 위험). 저장 불일치가 반복 관측된다는 실측 신호 없음.
  -> **보류(트리거 대기)**. 실사용에서 동일 설정 실수가 재발하면 착수.
- **B3 Insight 생성 시 tool_names 기본값 정책**: 제품 결정. 코드/whitelist는 이미 8개
  허용. -> 대표님 제품 결정 사안, 코드 변경 아님.

**결론(확정)**: 대표님이 "1(초기 턴 버그) + 2(예방책)를 하나의 워크플로우로"를 승인했다.
이를 존중하여 본 PR 스코프를 다음과 같이 확정한다:

- **본 PR 포함**: 옵션 A(초기 턴 버그, bin-ai-manager) + B1(런타임 가시화). 둘 다
  "Insight 알림 신뢰성"이라는 단일 주제이고 실제 장애 근거가 있어, 한 PR = 한 논리
  변경 원칙과 충돌하지 않는다(하나의 논리적 주제 = Insight notify 신뢰성). B1은
  missing-function-call을 메트릭/경고로 승격하는 관측 강화로, 초기 턴 버그가 재발하거나
  유사 설정 실수가 생겼을 때 조용히 묻히지 않게 한다.
  - 단, B1이 bin-pipecat-manager 변경을 요구하고 초기 턴 버그가 bin-ai-manager
    변경이라, 구현 착수 시 B1의 정확한 계측 지점(pipecat _log_missing_function_call ->
    메트릭)을 별도 섹션으로 상세화한다. 만약 구현 중 B1이 예상보다 커지면(별도 서비스
    빌드/배포 파이프라인 등) 그때 분리를 재검토하고 대표님께 보고한다.
- **보류(트리거 대기)**: B2. 저장 시 prompt<->tool_names 정합성 경고. prompt 자연어에서
  tool 참조 추출은 오탐 위험이 크고, 동일 설정 실수가 반복 관측된다는 실측 신호가 아직
  없다. 재발 시 착수.
- **제품 결정(코드 아님)**: B3. Insight 생성 시 tool_names 기본값 정책. 대표님 판단 사안.

## 6. 구현 계획 (옵션 A 확정 시)

1. `startContactCaseTurn`(start.go:596)에서 `pc` 획득 직후,
   `h.cache.ListenTurnPipecatcallIDAdd(ctx, c.ID, pc.ID, ttl)` 호출. ttl은 listen.go와
   동일하게 `config.Get().AIcallListenTurnPipecatcallIDTTLSeconds`.
   - 정합성 전제(명시): 여기서 등록하는 `pc.ID`는 `startPipecatcall`이
     `c.PipecatcallID`로 시작한 pipecatcall의 id이며(start.go:909), 초기 턴의 tool call이
     `ToolHandle`에 도착할 때 전달되는 `pipecatcallID`와 동일하다(pipecat-manager가 해당
     pipecatcall id를 echo). 따라서 `ListenTurnPipecatcallIDIsMember(c.ID, pipecatcallID)`가
     true가 된다. 이 "등록한 pc.ID == 도착 pipecatcallID" 연결이 옵션 A 정합성의 핵심이며,
     §6-3 단위테스트와 §7-2 라이브 검증이 이 전제의 어긋남을 포착한다.
2. 등록 실패는 fatal 아님: 로그 Warn + 메트릭(promListenTurnTotal 또는 신규 카운터),
   초기 턴은 계속 진행.
3. 단위 테스트: startContactCaseTurn이 ListenTurnPipecatcallIDAdd를 pc.ID로 호출하는지
   (mock 기대), 등록 실패 시에도 aicall이 Progressing으로 진행하는지.
4. §3 replay 영향 테스트 (양방향 assert): 초기 턴의 notify_agent OUTPUT row는
   OriginProactive로 태깅되어 replay에서 제외되지 **않음**을 assert, 초기 턴의 기계적
   tool-call/result row는 OriginListenInternal로 태깅되어 제외됨을 assert.

## 7. 검증 계획 (라이브, 머지 없이 배포 후)

1. bin-ai-manager 라이브 배포(머지 없이).
2. 새 Case + Insight Start + 초기 히스토리에 이탈 신호 -> 초기 턴 notify_agent가 이번엔
   success인지 aimessage로 확인.
3. §3 replay 영향을 분리 검증:
   - (a) notify 콘텐츠(OriginProactive row)가 이후 Q&A replay에 그대로 남는지 확인
     (start.go:739 필터는 OriginListenInternal만 제외하므로 남아야 함).
   - (b) 초기 턴 lookup 결과(OriginListenInternal)가 replay에서 빠지더라도, 후속 상담사
     Q&A가 필요 시 재호출로 정상 응답하는지 확인.
4. Prometheus listen_notify 증분 확인.
5. B1: missing-function-call 발생을 유발(미등록 tool 호출)했을 때 메트릭/경고가 실제로
   올라오는지 확인.

## 8. Review Log
| Round | Type | Verdict | Notes |
|-------|------|---------|-------|
| (이슈분석 1) | Issue | APPROVE | 5개 주장 코드 일치 |
| (이슈분석 2) | Issue | APPROVE | 옵션 A 타당성 확인, 2연속 |
| 설계 1 | Design | REQUEST_CHANGES | §3 트레이드오프 미해결+과대평가. notify OUTPUT은 OriginProactive라 replay 보존. §2/§3 정정, 트레이드오프 확정, B1 스코프 결론, 검증 분리 요구 |
| 설계 2 | Design | APPROVE | 3개 지적 코드 정합 해소 확인. 비차단 권고(pc.ID==도착 pipecatcallID 연결 명시) 반영 |

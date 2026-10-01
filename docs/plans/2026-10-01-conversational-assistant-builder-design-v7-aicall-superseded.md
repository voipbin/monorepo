# VOIP-1558 Conversational Assistant Builder 디자인 (v7, 구조 결정 대기)

Jira: VOIP-1558. 상태: 디자인 리뷰 7회차 대기. 이슈 분석 리뷰 루프는 4회차(3, 4회차 연속 APPROVE)로 종료되었고 결과는 Jira 코멘트에 기록되어 있다.

디자인 리뷰 이력: 1~5회차 모두 REQUEST_CHANGES x2, 6회차 A APPROVE, B REQUEST_CHANGES. 변경 요약은 12절에 있다. 지적은 모두 코드로 재검증한 뒤 반영했고, 코드와 맞지 않는 지적은 근거와 함께 기각했다(12절).

## 1. 문제

square-admin의 AI 생성 화면은 하드코딩된 템플릿 8개(`square-admin/src/views/ais/prompt_templates.js`)만 제공한다. 사용자가 AI와 대화하며 요구사항을 정리하고, VoIPBin에 맞는 프롬프트, 도구, 이름을 제안받는 경로가 없다.

확정 결정(대표님, 2026-10-01):
1. 구조화 설문이 아니라 대화형(멀티턴)으로 한다.
2. LLM 비용은 플랫폼이 부담한다. 고객 크레딧은 차감하지 않는다.
3. 생성 결과는 자동 저장하지 않는다. 기존 생성 폼에 채우고 사용자가 확인 후 저장한다.

## 2. 범위

범위 안:
- Builder 세션(에이전트가 AI와 대화하여 초안을 얻는 기능), 초안 전달, 초안 도구 목록의 서버 검증.
- 비용 상한, 데이터 격리.
- square-admin 채팅 UI, 초안 미리보기, 기존 생성 폼 채우기.

범위 밖:
- 기존 AI 편집 모드(생성된 AI의 프롬프트를 대화로 수정). 후속.
- 음성 대화 Builder. 텍스트 전용.
- `insight` 타입 초안. 1차는 `normal` AI 초안만 지원한다.
- `engine_key` 응답 노출 불일치(VOIP-1559, 이 PR에 포함하지 않음). Builder 행은 engine_key가 항상 빈 값이라 영향을 받지 않는다.
- 토큰 단위 계량, 플랫폼 전역 예산 카운터, 자동 서킷브레이커(10절의 도입 트리거 참조).
- 대화 자동 삭제 배치와 대화 삭제 UI(10절의 도입 트리거 참조).

## 3. 코드로 확인한 전제 (main 4a14812da)

| 번호 | 사실 | 위치 |
|---|---|---|
| F1 | 텍스트 전용 메시지도 메시지마다 pipecatcall을 시작한다. 이전 pipecat 세션은 새 전송 때 interrupt 된다. | `aicallhandler/send.go` SendReferenceTypeOthers |
| F2 | LLM 키는 AI 행의 EngineKey를 pipecat-manager가 조회해 주입한다. 조회 실패 시 빈 문자열로 진행. | `pipecatcallhandler/run.go:113-132`, `start.go:189-194` |
| F3 | pipecat 러너는 키가 비면 `OPENAI_API_KEY`, `XAI_API_KEY`, `GOOGLE_API_KEY` 환경변수로 폴백한다. 지원 서비스는 openai, grok, gemini 3종뿐이며 그 외는 `Unsupported LLM service`로 실패한다. | `scripts/pipecat/run.py:582,597,612,640` |
| F4 | ai.Type은 닫힌 enum(none, normal, insight). 생성/수정 API는 `aiType.IsValid()`로 검증한다. | `models/ai/main.go:221-235`, `aihandler/chatbot.go:48` |
| F5 | AllowedToolNames(t)는 타입별 허용 집합을 돌려주고 모르는 타입은 전부 거부(경고 로그와 메트릭 증가). | `models/ai/tool_validation.go:36-47` |
| F6 | startInitMessages는 Insight이면 전용 시스템 프롬프트를 쓴다. Insight는 MCP 발견을 차단한다(두 곳). | `start.go:1030`, `mcp_tool.go:191,419` |
| F7 | emit_info_card는 구조화 결과를 세션 메시지에 쓰고 프런트가 렌더링한다. 전용 하드코딩이 여러 곳에 있다(함수 이름 상수와 dispatch, LLM 결과 분기, history 가공). 도구 호출 요청 메시지(assistant)의 content는 비어 있고 raw 인자가 ToolCalls에 저장된다. | `tool.go:145,206-214`, `start.go:835-880`, `tool_emit_info_card.go` |
| F8 | 프런트 AI 채팅 선례: `POST aicalls`, `POST aimessages`, `GET aimessages?aicall_id=&page_size=100`, 3초 폴링. 모든 role을 raw로 표시하고 대기 표시와 에러 처리(alert)가 미흡하다. | `square-admin/src/components/TestAgentSheet.js` |
| F9 | billing-manager는 ai-manager 이벤트를 구독하지 않는다. | `bin-billing-manager/pkg/subscribehandler/main.go` |
| F10 | AIcallCreate는 aiGet으로 CustomerID를 해석하고 에이전트/액세스키는 CustomerAdmin 또는 CustomerManager 권한을 검증한다. | `servicehandler/aicall.go:21-100` |
| F11 | Send 쿨다운은 AIcall 단위다. 기본 3초(`aicall_send_cooldown_seconds`). Send는 aicall status와 TMDelete를 보지 않는다. | `aicallhandler/send.go:21-32`, `config/main.go:133` |
| F12 | `ai_aicalls`에는 `(customer_id, reference_type, reference_id)` 기준 활성 행 유일 제약(`uq_aicall_active_reference_key`)이 이미 있다. 키 표현식은 status가 terminated, terminating이거나 reference_id가 zero-UUID이면 NULL이고 tm_delete는 보지 않는다. | `a5a40c93d3e6_ai_aicalls_add_active_reference_key_.py:140-155` |
| F13 | aicall의 AI 해석은 `resolveAI` 한 곳이고 호출자는 `Start`, `StartTask`, `tool.go:1072` 세 곳뿐이다. Flow `ai_talk`와 call-manager SIP 수신은 모두 ai-manager `Start` RPC를 통한다. `Start`의 reference_type switch는 알려지지 않은 값에 `unsupported reference type`을 반환한다. | `start.go:60-80,187,1239`, `tool.go:1072` |
| F14 | ai-manager의 `PublishWebhookEvent` 발행 지점: aicall 상태 7곳(`aicallhandler/db.go:94,169,390,392,394,396,402`), 메시지 2곳(`messagehandler/db.go:75`, `event.go:337`), AI 8곳 등. customer webhook 전송과 customer WS 토픽은 모두 webhook-manager의 `webhook_published` 이벤트에서 나온다. | 좌측 열 참조, F24 |
| F15 | `aihandler.dbCreate`는 모든 AI 행에 direct hash를 만든다(`DirectV1DirectCreate`). direct hash의 `ai` 리소스는 `aicall` scope를 허용한다. | `aihandler/db.go:52`, `direct-manager boot.go:24` |
| F16 | `AIcallDelete`는 `tm_update`, `tm_delete`만 갱신한다(status 불변). 세션 종료는 `terminate`다. | `dbhandler/aicall.go:184-205` |
| F17 | pipecat-manager는 ai-manager 모델을 vendor 사본으로 컴파일하며(`AllowedToolNames` 포함) 도구 목록은 기동 시 `FetchTools`로 캐시한다. 새 타입을 모르는 사본은 default 분기로 빈 도구 집합을 돌려준다. | `pipecat-manager/vendor/.../tool_validation.go`, `pkg/toolhandler/main.go:55-71`, `pipecatcallhandler/run.go:178` |
| F18 | `databasehandler.NotEq`가 있어 DB 계층에서 `<>` 필터를 쓸 수 있다. 값 타입 제한 경고가 주석에 있다. JSON RPC 경계를 넘는 전달은 보장되지 않으므로 ai-manager 내부(dbhandler 호출)에서만 쓴다. | `bin-common-handler/pkg/databasehandler/main.go:40-90` |
| F19 | 사용자 Send 쿼터와 메시지 길이 상한은 현재 존재하지 않는다(쿨다운만 있음). | 위 F11 |
| F20 | ServiceAgent 라우트는 사용자 지정 `reference_type`, `reference_id` 필터를 그대로 전달하고, 권한은 `PermissionAll`(tenant 격리만)이다. `ServiceAgentAIcallList`, `ServiceAgentAIcallGet`, `ServiceAgentAImessageList`, `ServiceAgentAImessageCreate`가 해당한다. | `servicehandler/serviceagent_aicall.go:33-125`, `serviceagent_aimessage.go:19-110` |
| F21 | `messagehandler.Create` 호출 지점이 15곳이다. aicallhandler 7곳(`start.go:397,1074`, `send.go:79,107`, `tool.go:113,271`, `tool_insight.go:1428`)과 pipecat 이벤트 파생 경로 8곳(`event.go:126,149,195,221,348,396,480` 7곳 + `pipeline_error.go:110`)이며 이벤트 경로는 `PipecatcallReferenceID`만 갖고 aicall 객체가 없는 곳이 많다. | 좌측 열 참조 |
| F22 | `MessageList`는 `tm_create desc` 정렬에 토큰 페이지네이션이라 첫 페이지가 가장 최근 메시지다. | `dbhandler/message.go:205-215` |
| F23 | api-manager에서 `aiGet`을 쓰는 곳: `ai.go`(140,178,289,319,386), `aiprompthistory.go`(28,65), `aipromptproposal.go`(30), `aicall.go:46`, `serviceagent_aicall.go:246`. | 좌측 열 참조 |
| F24 | api-manager의 customer WS 경로는 webhook-manager의 `webhook_published` 이벤트만 소비한다(`subscribehandler/main.go:139,159`). pipecat-manager는 대화 텍스트를 `PublishEvent`로 직접 발행하지만(`runner.go:608-629`), 이 이벤트는 customer webhook과 WS로 가지 않고 timeline-manager의 전체 구독(`topicPatterns=["#"]`)에 아카이브된다. | `api-manager subscribehandler/main.go`, `timeline subscribehandler/main.go:31` |
| F25 | pipecat LLM 파이프라인이 뜬 뒤 provider가 돌려주는 오류(인증 실패, 한도 초과 등)는 role=notification 메시지(`PipelineErrorNotice`)로 aicall에 저장된다. 파이프라인이 뜨기 전의 실패(서비스 생성 예외)는 이 경로를 타지 않는다(F29). | `messagehandler/pipeline_error.go:85-125`, `pipecatcallhandler/runner.go:732-767` |
| F29 | 키가 없으면 러너 venv(pipecat 1.4.0)에서 `OpenAILLMService(api_key=None)`는 `OpenAIError`, `GoogleLLMService(api_key=None)`은 `ValueError`를 생성자에서 던지고 Python `/run`이 400 또는 500을 응답한다(직접 실행 확인). 그러나 이 오류는 사용자에게 전달되지 않는다. reference_type이 call이 아닌 aicall(Builder 포함)은 pipecat-manager `startReferenceTypeAIcall`의 `default:` 분기를 타며, 이 분기는 `go func(){ defer se.Cancel(); h.RunnerStart(pc, se) }()` 후 즉시 `return nil`한다. `RunnerStart`는 `runnerStartScript` 실패를 `log.Errorf`만 하고 반환한다. 따라서 `pipecatcallhandler.Start`, ai-manager `Send`, `POST /aimessages`는 모두 성공하고, notification도 만들어지지 않는다. 키 누락은 프런트에서 무응답으로만 보인다(v4의 "동기 전파"는 틀렸다). | `pipecatcallhandler/start.go:281-287`, `runner.go:84-90`, `aicallhandler/send.go:138-148` |
| F30 | `PublishWebhookEvent`는 `PublishEvent`(이벤트 버스, timeline 아카이브)와 `PublishWebhook`(customer webhook)을 둘 다 호출한다. 따라서 A2의 발행 차단은 aicall, aimessage 계열의 timeline 이벤트도 함께 막는다. pipecat-manager가 직접 `PublishEvent`만 호출하는 대화 텍스트 이벤트(`message_bot_transcription` 등)는 막지 못한다. | `notifyhandler/publish.go:24-27`, `runner.go:608-629` |
| F31 | `GET /aggregated_events`는 `activeflow_id`를 받아 같은 테넌트의 CustomerAdmin, CustomerManager에게 timeline 이벤트를 돌려준다. `aicall_`, `aimessage_`, `message_` 접두 converter가 있다. | `servicehandler/aggregated_events.go:62-200` |
| F32 | api-manager `AImessageCreate`는 `AIcallGet`을 호출하고 반환값을 버린 뒤 `AIV1MessageSend`를 호출한다(`aimessage.go:45-51`). ServiceAgent의 aimessage 함수는 모두 `ServiceAgentAIcallGet`을 먼저 호출하고, 그 함수는 내부 `aicallGet`을 쓴다. `AIAuditCreate`, `AIcallDelete`, `AIcallListen`도 `aicallGet`을 쓴다. `aimessageGet`은 `AImessageGet`, `AImessageDelete`가 쓴다. | `serviceagent_aimessage.go:40,100`, `aiaudit.go:37`, `aicall.go:227,252,282`, `serviceagent_aicall.go:112,157`, `aimessage.go:143` |
| F33 | ai-manager는 이미 `reqHandler.FlowV1ActiveflowCreate`를 직접 호출한다(`summaryhandler/start.go:408`, 이 선례는 `flowID`가 Nil이 아니라 `OnEndFlowID`, `referenceType=ai`, `referenceID=sm.ID`). `flowID=uuid.Nil` 호출 선례는 api-manager `servicehandler/aicall.go:102`와 `serviceagent_aicall.go:267`(`referenceType=api`)이며 flow-manager가 Nil flow를 허용한다(`activeflowhandler/db.go:56`). | 좌측 열 참조 |
| F26 | 러너는 LLM `max_tokens`를 지정하지 않는다(`run.py`에서 해당 설정 없음). 출력 상한은 provider 기본값에 의존한다. | `scripts/pipecat/run.py` |
| F27 | 기본값: 대화형 aicall 유휴 만료 24시간(`aicall_conversation_idle_timeout_hours`), Insight 세션 유휴 30분(Insight 전용). | `config/main.go:130-133` |
| F28 | 기존 8개 템플릿이 쓰는 도구는 connect_call, stop_service, send_email, send_message, set_variables, case_create 6종이다(insight 템플릿 제외). | `prompt_templates.js` |

F3 결과: H1(플랫폼 키를 고객 AI 행에 저장 금지)은 Builder 행의 engine_key를 비워 두면 충족된다. 러너가 지원하는 엔진은 3종뿐이므로 Builder 엔진은 서버 설정으로 고정한다(5.1).

F17 정정: 이슈 분석의 "pipecat-manager 변경 없음"은 정정한다. 코드 변경은 없지만 vendor 갱신과 재시작이 필요하다(8절).

부수 관찰: 현재 코드에서도 고객이 engine_key를 비우고 만든 일반 AI는 이미 플랫폼 키로 동작한다. 플랫폼 부담은 새 구조가 아니라 기존 동작의 연장이다. 이 기능이 새로 여는 것은 "상한 없는 대화형 사용"이다.

## 4. 공급 방식 비교와 추천

Builder 대화는 aicall이므로 AI 행이 반드시 필요하다(aicall.assistance_id는 AI 행을 가리키고, 러너가 `AIV1AIGet`으로 행을 읽는다). 따라서 "행 없이 가상 AI"는 불가능하다.

| 안 | 내용 | 작업량(코드 확인) | 평가 |
|---|---|---|---|
| (a) | 고객 소유 Builder 행(내부 전용 타입) | 차단 지점이 ai-manager 단일 차단점 5개 + api-manager 단일 헬퍼 4개로 수렴(5.2) | 추천 |
| (b) | 플랫폼 소유 단일 행 | aicall.customer_id와 ai.customer_id가 달라지는 첫 사례. `AIcallCreate`(F10), aimessages 조회, 러너 조회 등 소유 검증 예외가 전 경로에 생김 | 기각(보안 불변식 훼손) |
| (c) | 기존 normal AI에 Builder 도구만 부여 | 제외 작업은 없음. 그러나 타입별 deny-by-default 도구 집합(F5)과 전용 시스템 프롬프트 분기(F6)를 쓸 수 없고, normal AI는 고객이 도구를 바꿀 수 있어 Builder 도구 집합을 서버가 고정할 수 없음 | 기각(도구 집합 고정 불가) |
| (d) | (a)에 `is_system` 같은 플래그 컬럼 추가 | 스키마 변경(ai_ais 재작성 락 위험 선례 있음) 필요 | 기각(내부 타입이 이미 구분 신호이며 스키마 변경 불필요) |

| (e) | aicall을 만들지 않는 상태 없는 멀티턴. api-manager에 라우트 하나(`POST /ai_builder/chat`), ai-manager에 전용 RPC 하나. 클라이언트가 대화 이력을 보내면 서버가 고정 시스템 프롬프트와 strict json_schema(`{reply, draft?}`)로 한 번 호출해 응답과 초안을 동기로 돌려준다. 선례: analysis gateway(`analysishandler.Run`, 모델 허용 목록, 입력 크기 상한, 아무것도 저장하지 않음, `AIV1ServiceTypeAnalysisRun`이 timeout 인자 지원)와 prompt proposal(Gemini 동기 호출, `cfg.GoogleAPIKey` 사용). | 아래 7개 항목 참조 | 7회차 리뷰어 B가 지적한 누락 대안. 비교 결과 (e)가 원칙에 더 맞는 것으로 판단, 11절 최상단 결정으로 올림 |

(e)안 상세 평가(코드 확인):
- 확정 결정 3가지(대화형, 플랫폼 비용 부담, 자동저장 없음)를 모두 충족한다.
- 없어지는 것: aicall, AI 행, activeflow, 세션 유일 제약, 슬롯 해소, A1~A5와 B1~B4 전부, G2/G3 신뢰 모델, `messagehandler.Create` 조회 비용과 일반 고객 webhook 누락 위험, pipecat vendor 갱신과 재시작, 폴링 상태 기계와 120초 UX, F29(키 누락 무신호). 키 누락은 ai-manager가 동기 오류를 돌려준다(`cmd/ai-manager/main.go:202`는 이미 `GOOGLE_API_KEY` 미설정을 경고하며 audit, proposal, analysis와 같은 키를 쓰므로 셀프호스팅에 새 키 계약이 없다).
- 초안 도구 허용 목록 검증(6종), 길이 상한, 초안 12000자 상한은 응답 검증 단계에서 서버가 같은 규칙으로 적용한다(코드가 사실을 확정, LLM은 서술만).
- 비용 통제: 요청당 입력 크기 상한(`maxInputBytes` 선례), 요청당 출력 `max_tokens`(analysis gateway는 reasoning_effort=none, 출력 예산 지정 선례), 고객당 일일 요청 상한. 일일 카운터는 `cachehandler`에 범용 카운터 메서드가 없어 작은 신규 메서드(원자적 증가와 TTL) 또는 기존 DB 테이블 활용이 필요하다(구현 확인 항목).
- 잃는 것: 서버 측 턴 이력과 서버 집계(턴 수는 요청의 이력 길이로 검사), 토큰 스트리밍(동기 응답, 3초 RPC 기본 timeout이므로 timeout 인자를 명시), 새로고침 복원(브라우저 sessionStorage에 보관, 지워지면 새 대화), 대화 로그의 서버 보존(지원 목적 열람 불가, 오히려 5.8 개인정보 부담이 사라짐).
- 위험: 클라이언트가 보낸 이력을 신뢰할 수 없다. 이력 위변조는 그 사용자 자신의 응답에만 영향을 주고, 비용은 요청당 입력/출력 상한과 일일 요청 상한이 묶는다. 플랫폼을 무료 LLM 프록시로 쓰는 남용은 고정 시스템 프롬프트, strict schema, 크기 상한, 일일 상한으로 (a)안과 같은 수준으로 제한된다.

추천: (e). 같은 목표를 더 작은 면적으로 달성한다(새 영속 상태 없음, 기존 호출 선례 재사용, 차단 지점 없음). (a)는 aicall 재사용의 이점(대화 이력 서버 보관, 서버 측 턴 상한)이 있지만 그 대가가 5.2의 전체 차단 인프라와 폴링 UX다. 11절의 최상단 결정으로 올린다. 결정 전까지 5절 이하는 (a)안 설계 v7이며 (e)가 선택되면 5.1, 5.2, 5.3, 5.7 상당 부분이 삭제되고 5.4~5.6, 5.9가 축소 재작성된다.

## 5. 설계 (추천안 a)

### 5.1 모델

- `ai.TypeBuilder Type = "builder"` 상수를 추가하되 `validTypes`에는 넣지 않는다. 고객 `POST /ais`, `PUT /ais/{id}`의 `aiType.IsValid()`(F4)가 builder를 기존 에러로 거부한다. OpenAPI의 AI type enum(normal, insight)은 변경하지 않는다.
- `aicall.ReferenceTypeBuilder ReferenceType = "builder"` 상수를 추가한다. Builder aicall의 식별자다(aicall에 AI type 컬럼이 없으므로 reference_type을 식별자로 쓴다).
- **세션 소유자는 에이전트다.** Builder aicall의 `reference_id = agent_id`(세션을 만든 에이전트). F12의 기존 유일 제약이 "에이전트당 활성 Builder 세션 1개"를 DB 수준에서 보장한다. 신규 마이그레이션이 없다. 이 선택은 (1) 같은 고객의 다른 관리자와 세션이 충돌하지 않고, (2) 세션 소유자를 aicall 자체가 기록하므로 접근 규칙(5.2)이 단순해지며, (3) 유일 제약 비용이 고객 단위와 같다는 이유다.
- Builder 세션은 에이전트 로그인(`a.IsAgent()`)만 허용한다. accesskey, direct, delegate(대리 접속) 토큰은 거부한다. `StartBuilder`, `BuilderSessionStart`는 `agentID == uuid.Nil`을 명시적으로 거부한다(`a.AgentID()`는 비에이전트에서 `uuid.Nil`을 돌려주고, zero-UUID `reference_id`는 F12의 유일 제약 키를 NULL로 만들어 슬롯이 비어 버린다). 테스트로 고정한다.
- `AllowedToolNames(TypeBuilder)` = `AllBuilderToolNames` = `[propose_assistant_draft]` 하나. 신규 슬라이스로 정의하고 `AllToolNames`에는 넣지 않는다(넣으면 normal AI에 노출된다).
- `BuilderDraftableToolNames`: Builder가 초안에 넣을 수 있는 도구의 **명시적 허용 목록**(fail-closed, 신규 도구는 추가하기 전까지 초안에 못 들어간다). 1차 값은 기존 템플릿이 쓰는 6종(F28): connect_call, stop_service, send_email, send_message, set_variables, case_create. create_call(외부 발신 비용), get_resource, get_aicall_messages, get_variables, get_correlation, join_queue 등은 템플릿이 쓰지 않으므로 1차에서 제외한다. 테스트로 모든 항목이 `AllToolNames`의 원소임을 고정한다.
- Builder AI 행은 내부 생성 메서드(`aihandler.CreateBuilder`)로만 만든다. direct hash를 만들지 않는다(F15 회피). engine_key는 항상 빈 값, `engine_model`은 서버 설정 `ai_builder_engine_model`(기본 `gemini.gemini-2.5-flash`). openai, grok, gemini 3종 외 값은 설정 검증에서 거부한다(F3). `tool_names=[propose_assistant_draft]`, init_prompt는 비운다.
- `startInitMessages`는 `a.Type == TypeBuilder`이면 `BuilderSystemPrompt`를 쓴다(F6 선례). MCP 발견 차단(F6의 두 곳)에 builder를 추가한다. `buildPromptSnapshots`는 builder에 대해 autoAudit=false로 고정한다.
- 시스템 프롬프트의 도구 카탈로그는 `BuilderDraftableToolNames`와 도구 정의(description)에서 코드로 주입한다(하드코딩 금지).

### 5.2 고객 접근 차단과 데이터 격리

보호 목표를 먼저 명시한다. 3회차 리뷰까지 "소유 에이전트만 열람" 모델이 접근 지점을 계속 늘렸기 때문에 목표와 신뢰 모델을 분리해 단순화했다.

| 목표 | 수준 | 내용 |
|---|---|---|
| G1 | 하드 | Builder AI 행과 aicall은 Builder 세션 경로 밖에서 사용할 수 없다(비용, 키, 도구 집합). |
| G2 | 하드 | Builder 대화(시스템 프롬프트, 도구 인자 포함)가 고객의 외부 webhook, WS, 고객 대상 AI/aicall 목록과 aicall 소비자에 섞이지 않는다. 고객의 외부 자동화가 Builder 이벤트를 운영 이벤트로 오인하지 않게 한다. |
| G3 | 수용 | 같은 테넌트의 CustomerAdmin, CustomerManager는 기존 aicall과 같은 권한 모델로 Builder 세션을 **열람**할 수 있다(`GET /aicalls/{id}`, `aimessages` 조회, `/aggregated_events`의 부분 노출 포함, F31). **쓰기(`AImessageCreate`, `AIcallTerminate`)는 소유 에이전트만** 허용한다(다른 관리자가 남의 세션에 메시지를 주입해 턴을 소진하거나 종료해 일일 상한을 소모시키는 것을 막는다. `AIcallTerminate`는 `aicallGet`을 `aicallGetBuilder`로 바꿔 받은 raw aicall로, `AImessageCreate`는 현재 버리는 `AIcallGet` 반환값(WebhookMessage, `ReferenceID` 노출)으로 `aicall.ReferenceID == a.AgentID()`를 비교한다(추가 호출 없음). accesskey, direct는 `AgentID()==Nil`이라 거부된다). 비관리자(PermissionAll) 에이전트는 열람도 차단한다(ServiceAgent 경로는 B2/B3, 일반 경로는 admin/manager 권한 검사). |

세션 소유자(`reference_id = agent_id`)는 세션 유일성(F12)과 동시성 충돌 방지를 위한 키이며, 쓰기 권한(위 G3)의 기준이다. 열람은 제한하지 않는다.

핵심 원칙: 호출부마다 기억해야 하는 opt-in 제외가 아니라 default-deny로 막는다. flow-manager, call-manager는 api-manager를 거치지 않으므로(F13) G1은 ai-manager에서 막는다.

**A. ai-manager**

| 번호 | 지점 | 규칙 |
|---|---|---|
| A1 | `aicallHandler.Start`, `StartTask` | `resolveAI` 결과가 `TypeBuilder`이면 거부. Builder 세션은 전용 `StartBuilder`로만 시작한다. Flow `ai_talk`, SIP 수신, `POST /aicalls`, direct hash 모두 여기서 막힌다(F13). reference_type=builder 직접 요청도 `Start`의 switch default가 거부한다. |
| A2 | webhook 발행 중앙화 | aicallhandler와 messagehandler의 모든 `PublishWebhookEvent` 호출을 하나의 헬퍼(`publishWebhook`)로 모으고, aicall의 `ReferenceType == builder`이면 발행하지 않는다(F30: customer webhook과 timeline 이벤트가 함께 막힌다). Builder AI 행의 `ai_created` 발행(`aihandler/db.go:107`)도 같은 대상이므로 `CreateBuilder`는 이벤트를 발행하지 않는다(`aihandler.Create`를 재사용하지 않고 이벤트 없는 DB 생성 경로를 쓴다). aicall 상태 7곳과 `message_intermediate`(이미 aicall 보유)는 헬퍼로 교체한다. `messagehandler.Create`는 내부에서 `aicallID`로 aicall을 조회(`dbhandler.AIcallGet`, 캐시)해 builder이면 `message_created`를 건너뛴다. 조회가 실패하면 builder인지 알 수 없으므로 발행을 건너뛰고(fail-closed) 경고 로그와 카운터를 남긴다. G2는 하드 목표이며 비용을 정직하게 적는다: `messagehandler.Create` 내부 조회이므로 모든 일반 고객 aicall의 `message_created` webhook이 일시적 DB/캐시 오류에서 누락될 수 있고(고객 외부 자동화가 소비하는 신호), 모든 Create에 캐시 우선 조회가 한 번 추가된다. 반대쪽 비용은 시스템 프롬프트 유출이므로 G2에 따라 fail-closed를 택한다. 이미 aicall을 보유한 호출자(aicallhandler 7곳과 `event.go:480`)는 조회 없이 판별하고, 나머지 7곳(`event.go` 6곳, `pipeline_error.go:110`)은 `PipecatcallReferenceID`만 받는다. 그중 일부(`event.go:160,235,311`, `pipeline_error.go:82`)는 같은 함수에서 이미 `AIV1AIcallGet`으로 aicall을 읽으므로 그 값을 `Create`에 넘기면 추가 조회가 필요 없다(조회 실패 시에는 fail-closed). 정확한 목록은 구현 1단계 grep 표에서 확정한다. 호출 지점은 15곳(aicallhandler 7, `event.go` 7, `pipeline_error.go:110`)이며 호출자가 신경 쓸 필요가 없다. 헬퍼 밖의 직접 `PublishWebhookEvent` 호출이 없음을 grep 기반 테스트로 고정하며 범위는 aicallhandler와 messagehandler다. aihandler에는 normal AI용 정상 발행이 여러 곳(`db.go:107,167,204,248,251`, `chatbot.go:175,199`, `mcpserver_validation.go:122`) 있어 grep 대상에서 제외하고, builder 행은 `CreateBuilder`(이벤트 미발행)와 A4(Update, Delete 거부)가 발행 경로 자체를 막는다는 행동 테스트(`PublishWebhookEvent` no-EXPECT)로 고정한다. |
| A3 | 목록 기본 제외 | `aihandler.List`, `aicallhandler.List`가 DB 계층에서 builder를 기본 제외한다(NotEq, ai-manager 프로세스 안에서만, F18). 필터에 `type`(AI) 또는 `reference_type`(aicall) 값이 이미 있으면 NotEq를 넣지 않고(같은 맵 키 충돌 회피) builder 값만 거부한다. Builder 세션 로직의 내부 조회는 명시적으로 `reference_type=builder`를 넘긴다. |
| A4 | AI 변경 경로 | listenhandler의 `PUT`, `DELETE` 진입부(`UpdateMcpServerIDs` 경로 포함, `listenhandler/v1_ais.go:133,307`)와 `aihandler.Update`, `Delete`가 대상 행이 `TypeBuilder`이면 거부한다(builder를 normal로 바꿔 고객 AI로 만드는 우회 포함). |
| A5 | 소비자 | `teamhandler` 멤버 검증이 builder AI를 거부한다(`validateNoInsightMembers` 선례, `handler.go:233-253`, 생성과 수정 두 곳). 자동 audit은 5.1의 autoAudit=false. |

**B. api-manager**

| 번호 | 헬퍼 | 규칙 |
|---|---|---|
| B1 | `aiGet` | 가져온 AI가 builder이면 NotFound를 반환한다. `aiGet` 호출 10곳(F23)이 한 번에 막힌다. 이 규칙은 권한 확인보다 먼저 builder를 판별하므로 타 테넌트가 builder id와 일반 id를 응답 코드로 구별할 수는 있으나, builder AI id는 비공개 UUID(소유 테넌트의 aicall에만 나타남)라 추측해야 알 수 있다. 수용한다. |
| B2 | `aicallGet` default-deny | `aicallGet`이 builder aicall이면 NotFound를 반환한다. Builder 세션에 필요한 `AIcallGet`, `AIcallTerminate` 두 함수만 별도 헬퍼 `aicallGetBuilder`(builder 허용)를 쓴다. 그 결과 `ServiceAgentAIcallGet`(그리고 이를 먼저 호출하는 `ServiceAgentAImessage*`), `ServiceAgentAIcallListen`, `AIAuditCreate`(LLM 비용), `AIcallDelete`는 호출부를 고치지 않고도 builder를 NotFound로 막는다(F32). `aicallGetBuilder`의 직접 호출처가 정확히 `AIcallGet`, `AIcallTerminate`임을 grep 기반 테스트로 고정한다. `AIcallGet`을 경유하는 서비스 함수는 `AImessageCreate`(`aimessage.go:45`), `AImessageGetsByAIcallID`(`aimessage.go:81`), `AIParticipantGets`(`ai_participants.go:25`) 3곳이며 모두 G3 범위(`AImessageCreate`만 쓰기 소유자 검사를 추가)로 의도한다. 참가자 목록의 builder 노출도 9절 테스트로 의도를 고정한다. `AIcallGet`은 기존 admin/manager 권한 검사를 그대로 쓴다(열람, G3). `AIcallTerminate`와 `AImessageCreate`(내부적으로 `AIcallGet` 경유)는 builder일 때 추가로 `aicall.ReferenceID == a.AgentID()`를 검사한다(쓰기, G3). |
| B3 | 목록 필터 | `AIcallList`, `ServiceAgentAIcallList`, `AIList`는 필터에 `reference_type=builder` 또는 `type=builder`가 오면 거부한다(F20: ServiceAgent 라우트가 필터를 그대로 전달한다). 기본 제외는 A3. |
| B4 | `AIcallCreate` | `reference_type=builder`를 거부한다(방어적. A1이 최종 차단). |

`AImessageGet`은 메시지 id로 직접 조회하므로(F32) B2의 영향을 받지 않는다. 같은 테넌트 admin/manager가 메시지 id를 알아야 하는 열람 경로이며 G3로 수용한다. `AImessageDelete`는 api-manager 함수만 있고 ai-manager에 `DELETE /v1/messages/{id}` 라우트가 없어 현재는 동작하지 않는다(`listenhandler/main.go:471-473`는 GET만). 라우트가 추가되면 `aimessageGet`에도 builder 판별을 넣는다.

**소비자 열거:** Insight, contact_case, 요약, 타임라인(timeline-manager `analysishandler/collect_context.go:364`가 `AIV1AIcallGet`으로 aicall을 읽는다), 통계 소비자에서 builder를 처리하는 방식은 구현 1단계에서 grep 표로 확정한다. 이 열거는 차단 목록에 항목을 추가할 뿐 설계 구조를 바꾸지 않는다.

**잔존 경로(수용, G3):** (1) pipecat-manager가 `PublishEvent`로 직접 발행하는 대화 텍스트 이벤트(F30)는 customer webhook과 WS로 가지 않고 timeline-manager 전체 구독으로 아카이브되며, 같은 테넌트 admin/manager가 `activeflow_id`로 `/aggregated_events`를 호출하면 일부 노출될 수 있다(F31). activeflow id는 `ActiveflowList`로 같은 권한이 열거할 수 있다. 타 테넌트 유출 경로가 아니고 G3의 범위다. (2) flow-manager의 `activeflow_created`, `activeflow_updated`, `activeflow_deleted`(`activeflowhandler/db.go:124,238,262,387`)는 흐름 메타데이터(대화 텍스트 없음)이지만 `PublishWebhookEvent`로 customer webhook에 나간다. Builder activeflow는 flow 액션이 없는 `reference_type=api` 흐름이며 고객 외부 자동화가 운영 이벤트로 오인할 수 있다. 이 신호는 대화 내용이 아니므로 수용하고(G2는 대화 내용 노출 차단이 핵심), 오인이 실제 문제가 되면 flow-manager에서 builder 전용 activeflow 이벤트 억제를 추가하는 것을 10절 트리거로 둔다. (3) 보존 기간은 기존 이벤트 정책을 따른다(5.8).

### 5.3 세션 API

신규 엔드포인트(api-manager, `customer_id`는 JWT에서, `agent_id`는 호출자 신원에서 주입하며 요청 본문에 두 값 모두 없음):

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | /ai_builder/status | `{available, max_turns, max_message_chars}`. 호출자가 세션 권한(에이전트 로그인, CustomerAdmin 또는 CustomerManager)이 없거나 기능이 꺼져 있으면 `available=false`. 카드 노출이 권한과 어긋나지 않게 한다 |
| POST | /ai_builder/sessions | 세션 시작 또는 재개. 응답은 aicall(WebhookMessage 변환) |
| POST | /aicalls/{id}/terminate (기존) | 세션 종료. `DELETE aicalls/{id}`는 B2로 builder에서 거부 |
| POST, GET | /aimessages (기존) | 대화, 이력(`AIcallGet` 경유, G3 권한 모델) |

`POST /ai_builder/sessions` 흐름(RPC 하나로 단순화. v3의 Get/Start 분리는 api-manager가 activeflow를 먼저 만들어야 했기 때문이었고, ai-manager가 직접 만들도록 바꿔 불필요해졌다, F33):
1. 권한: 에이전트 로그인, `agentID != Nil`, CustomerAdmin 또는 CustomerManager. accesskey, direct, delegate 거부.
2. ai-manager RPC `AIV1BuilderSessionStart(customerID, agentID)`: 재개 또는 생성을 한다.
3. ai-manager `BuilderSessionStart`(기존 contact_case 재생성 루프 `start.go:476-600`와 같은 최대 3회 루프):
   a. `ai_builder_enabled` 확인. 꺼져 있으면 명시적 서비스 불가 에러.
   b. 최근 builder aicall 조회: `aicallhandler.List`로 `(customer_id, reference_type=builder, reference_id=agentID)`, size 1, 최신순(`AIcallGetByReferenceID`는 customer와 삭제 필터가 없어 쓰지 않는다, `dbhandler/aicall.go:147-157`. F12의 `FieldCustomerID`, `FieldReferenceType`, `FieldReferenceID` 필터를 쓴다). 삭제된 행도 조회된다.
   c. 재개: 존재하고 status가 terminated, terminating이 아니고 `TMDelete == nil`이고 유휴 만료 아님(`isAIcallIdleExpired`, 기본 24시간, F27)이고 customer_id가 일치하면 그대로 반환한다. 이 분기는 상한을 소모하지 않는다.
   d. 슬롯 해소: 존재하지만 재개 불가인데 status가 활성이면(유휴 만료, soft delete 등) 슬롯을 점유한 것이므로 `ProcessTerminate`(`process.go:38`)로 종료한 뒤 계속한다. `AIcallDelete`는 슬롯을 비우지 못한다(F12, F16). 코드 확인: `ProcessTerminate`는 `FlowV1ActiveflowServiceStop`과, 비-call reference에서 `FlowV1ActiveflowContinue`를 호출하지만 둘 다 실패해도 로그만 남기고 계속하며(`process.go:63-83`) Builder activeflow는 flow 액션이 없어 부작용이 없다. 이어서 pipecatcall을 정리한다. `PipecatV1PipecatcallGet` 오류 중 NotFound는 이미 종료로 간주하고 그 외는 반환하며(슬롯 해소 실패, 사용자에게 재시도 요청), `PipecatcallTerminate` RPC 실패는 NotFound 여부와 무관하게 로그만 남기고 `UpdateStatus(terminated)`로 진행한다. **activeflow 정리(코드 확인):** `ServiceStop`은 stack만 pop하고 상태를 바꾸지 않으며(`activeflowhandler/activeflow.go:18-36`) `Continue`는 flow 액션이 없어 실패하므로 Builder activeflow가 `running`으로 남는다. `ProcessTerminate`의 호출처가 넷(`listenhandler/v1_aicalls.go:231` 고객 terminate, `service.go:164`, `event.go:47,77`)과 슬롯 해소 경로로 여럿이므로 호출자마다 Stop을 붙이지 않는다. `ProcessTerminate` 내부에서 `tmp.ReferenceType == builder`이면 `FlowV1ActiveflowStop`을 호출해 호출 지점을 하나로 만든다(`tool.go:901`은 LLM 도구 `stop`의 핸들러이므로 선례로는 이 RPC를 ai-manager가 호출한다는 사실만 유효하다). 이미 terminated로 조기 return하는 분기(`process.go` 첫 부분)에서도 builder는 멱등 Stop을 호출한다(Stop은 이미 ended면 no-op, `flow-manager stop.go:26`). 실패는 로그만 남기고 진행하며, 그 경우 activeflow가 `running`으로 남는 것은 기능 영향이 없는 잔존으로 수용한다(에이전트당 1개로 상한이 있고 유휴 만료는 호출 시점에 판정되므로 방치된 세션의 activeflow는 사용자 복귀 전까지 남을 수 있다). `FlowV1ActiveflowStop`이 상태를 `ended`로 바꾸고 `activeflow_updated`를 발행함은 코드로 확인했으며(`stop.go:15-55`, `db.go:262`) 이 이벤트는 5.2 잔존 경로(2)의 범주다. soft delete된 행에서 `h.Get`이 동작하는지는 구현 1단계에서 확인하고, NotFound이면 슬롯 점유가 영구화되므로 builder 조회는 삭제 행을 포함하는 경로를 쓴다. contact_case 루프의 `UpdateStatus(terminated)`+`stopListening`(`start.go:575-586`)은 pipecat과 activeflow 정리를 건너뛰므로 쓰지 않는다.
   e. 생성 경로에서만 일일 상한 검사(5.4). 위반 시 `ResourceExhausted`(429).
   f. Builder AI 행 get-or-create(고객별 `type=builder`를 dbhandler로 직접 조회, 없으면 `CreateBuilder`). 동시 요청으로 중복 행이 생길 수 있으나 해롭지 않다(세션 유일성은 aicall 유일 제약이 보장하고 가장 오래된 행을 쓴다). 신규 유니크 인덱스를 만들지 않는 것은 `ai_ais` 재작성 락 위험을 피하기 위한 의도적 선택이다.
   g. `reqHandler.FlowV1ActiveflowCreate`로 `reference_type=ai`(F33: `flowID=uuid.Nil`, `referenceID=uuid.Nil`로 만든다. aicall id를 이 시점에 알 수 없고, activeflow와 aicall의 연결은 aicall 행의 `activeflow_id`가 가진다)인 activeflow를 만들고(F33. `AIcallCreate`의 `aiGet`/B1 경로를 타지 않는다) `startAIcallByMessaging(..., ReferenceTypeBuilder, referenceID=agentID, ...)`로 시작한다. 사용자가 먼저 말하는 흐름이므로 첫 AI 턴은 트리거하지 않는다. 유일 제약 중복은 contact_case 루프와 같은 `dbhandler.IsErrDuplicate`(`start.go:522`)로 판별한다. 중복이면 aicall 행은 없고 자신이 만든 activeflow만 남으므로 `FlowV1ActiveflowDelete`로 삭제하고 b로 돌아가 재개한다.
4. 응답: aicall.

"새로 시작"은 `terminate` 후 `POST /ai_builder/sessions`이며 일일 생성 상한을 소모한다.

### 5.4 비용 상한과 최악 비용

플랫폼이 부담하므로 상한은 필수다. 실측 신호가 없으므로 최소 집합만 두고 토큰 단위 계량은 도입 트리거(10절)로 미룬다.

| 상한 | 초기값(설정) | 강제 지점 |
|---|---|---|
| 에이전트당 활성 세션 | 1 | F12 기존 유일 제약(신규 구현 없음) |
| 고객 일일 세션 생성 | 10 (최근 24시간) | 생성 경로에서만. `aicallhandler.List`(`customer_id`, `reference_type=builder`, size 10, 최신순, 삭제 행 포함)를 조회해 10번째 행의 `tm_create`가 24시간 이내이면 거부한다. `AIcallList`는 `tm_create >= X` 필터가 없어(`Lt` 토큰만) 이 방식을 쓴다. 삭제로 우회할 수 없다 |
| 세션당 사용자 턴 | 20 | `aicallhandler.Send`에서 reference_type이 builder면 센다 |
| 사용자 메시지 길이 | 2000자(rune 기준) | 같은 Send 지점 |
| 초안 init_prompt 길이 | 12000자(rune 기준) | `propose_assistant_draft` 핸들러(초과 시 오류를 LLM에 돌려 줄여 다시 호출하게 한다) |
| 킬 스위치 | `ai_builder_enabled`(기본 false) | 세션 시작 거부, status 응답 |

v2의 시간당 5회는 제거했다. 시간당 5회만 두면 하루 최대 120세션이 가능해 일일 10회가 실질 상한이다. 버스트도 일일 상한 안에서 세션당 상한(20턴)으로 묶이므로 일일 단일 상한만 둔다.

`Send` 강제 지점의 세부 규칙(`aicallhandler.Send`, builder일 때만 추가 검사. `POST aimessages`가 항상 지나는 단일 경로임을 확인했다):
- aicall이 terminated, terminating이거나 `TMDelete != nil`이면 거부한다(F11: Send는 status를 보지 않는다).
- 턴 수 = 해당 aicall의 `role=user` 메시지 수. 상한 초과는 메시지를 저장하지 않고 `cerrors.ResourceExhausted`(reason `AI_BUILDER_TURN_LIMIT`, 일일 상한은 `AI_BUILDER_DAILY_LIMIT`, `rpc.go:68`에서 429로 변환되는지 구현 1단계에서 확인)로 거부하며, 프런트는 reason 코드로 분기해 고정 안내 문구를 표시한다(문자열 매칭 금지, LLM 생성 아님).
- 동시 요청 2건이 상한 직전에 통과하는 경합(TOCTOU)으로 1~2턴 초과할 수 있음을 수용한다.
- 구현 1단계에서 실제 Gemini 세션으로 텍스트 턴에 pipecat이 `role=user` 행을 추가로 만드는지 확인한다(`EventPMMessageUserTranscription`, `EventPMMessageUserLLM`이 user 행을 만든다, `event.go:126,348`). 중복이 생기면 `Origin` 등으로 API 경로 행만 세도록 정의를 좁힌다.

최악 비용 상한(이론치, 비관 가정: 한글 1자 = 1토큰, 시스템 프롬프트와 도구 정의 6k 토큰, assistant 텍스트 응답 2k 토큰/턴, 사용자 메시지 2k 토큰/턴, 초안 도구 호출 인자 최대 12k 토큰, 최신 초안만 history 유지):
- 입력: n번째 턴 입력은 약 `6k + 12k + n x 4k` 토큰, 20턴 합계 약 1.2M 토큰.
- 출력: 초안을 갱신하는 턴은 텍스트 2k + 도구 호출 인자 12k = 최대 14k 토큰, 20턴 모두 갱신하는 극단 가정으로 약 280k 토큰.
- 세션 1개 최대: 입력 약 1.2M + 출력 약 0.28M. 고객 하루(10세션) 최대: 입력 약 12M + 출력 약 2.8M.
- 모든 턴이 상한 길이로 이어지는 상한값이며 실사용은 훨씬 낮을 것으로 추정한다(추정, 실측 아님).
- 금액 환산은 하지 않았다. provider 현재 단가표를 확인하지 않은 상태에서 숫자를 쓰면 근거 없는 수치가 되기 때문이다. 대표님이 단가를 확인해 주시면 위 토큰 수로 환산하겠다(11절).
- 출력 상한 주의(F26): 러너는 `max_tokens`를 지정하지 않으므로 출력은 provider 기본 상한까지 늘어날 수 있다. 완화책은 시스템 프롬프트의 목표 길이 지시(초안 6000자 이내), 핸들러의 12000자 상한, 일일 세션 상한이다. 출력 폭주가 관측되면 Builder 전용 `max_tokens` 도입을 검토한다(10절 트리거).
- 과거 초안이 히스토리에 누적되지 않도록 5.5의 history 규칙을 적용한다.

일일 세션 상한 10과 세션당 20턴은 실측 근거가 없는 초기값이다. 초안 12000자는 하드 상한이고 6000자는 시스템 프롬프트의 작성 지침(보장 아님)이다.

계측은 Prometheus 카운터 최소 집합만 둔다: 세션 시작 수, 상한 위반 수(종류별), 도구 필터링으로 제거된 도구 수.

### 5.5 초안 전달: `propose_assistant_draft`

도구 인자: `name`(최대 80자), `detail`(최대 300자), `init_prompt`(최대 12000자), `tool_names`(구체 이름 배열, 최대 12개). 타입은 항상 `normal`이고 engine_model은 인자에 두지 않는다(폼의 기존 기본값 사용). 길이 상한이 기존 `POST /ais` 검증과 일치하는지 구현 1단계에서 확인해 더 엄격한 쪽으로 맞춘다.

F7 주의: pipecat은 `function.strict`를 설정하지 않으므로 JSON Schema는 안내일 뿐이고 모든 검증은 서버 핸들러가 한다.

핸들러(ai-manager, 도구 호출 시점):
1. `c.ReferenceType == builder` 단언(아니면 거부).
2. 인자 파싱과 길이 검증.
3. `tool_names`에서 `BuilderDraftableToolNames`에 없는 항목, `all`, 중복을 제거하고 제거 목록을 `dropped_tools`에 기록한다. 서버가 최종 목록을 결정하므로 프런트는 서버 결과만 신뢰한다.
4. 결과 JSON(`name, detail, init_prompt, tool_names, dropped_tools`)을 role=tool 결과 메시지로 저장한다. emit_info_card와 동일하게 `unmarshalToolResponse`를 우회하는 전용 분기를 둔다.
5. LLM에게 되돌리는 값은 "초안이 화면에 표시되었다"와 `dropped_tools` 알림의 짧은 문자열이다.
6. `RunLLM: true`로 정의한다(`toolhandler/definitions.go`, 도구 호출 뒤 확인 문장이 나오도록).

history 규칙(LLM에게 재전송되는 히스토리 조립, `start.go:835-880` 참고): LLM이 직전 초안을 보고 수정해야 하므로 **가장 최근 `propose_assistant_draft` 호출의 인자는 전문을 유지**하고, 이전 호출의 인자는 `{}` stub으로 치환한다. role=tool 결과 행(UI용 전문 저장)은 히스토리에서 짧은 확인 문자열로 치환한다. 저장 row에는 초안 전문이 있고 LLM 히스토리에는 최신 호출 인자 1개만 전문이 들어간다는 것이 의도다. 이 규칙은 v2의 모순 서술(LLM 결과 문자열에 본문이 없다 vs 최신 초안 유지)을 정리한 것이다.

위험 표기(v2의 `RiskyToolNames`, `risky_tools`)와 `revision` 필드는 제거했다. 초안 도구는 fail-closed 허용 목록(5.1)으로 이미 6종으로 한정되고, 사용자는 기존 폼에서 템플릿과 같은 방식으로 도구 목록을 확인한다. "초안 vN" 배지는 role=tool 결과 메시지 수로 센다.

신규 도구 추가 접점(F7 기준, 누락 시 증상):
- `models/message/tool.go` FunctionCallName 상수와 `tool.go:145` dispatch 등록: 없으면 unknown tool로 실패.
- `tool.go:206-214` LLM 결과 분기: 없으면 stored content가 LLM으로 되돌아간다.
- `start.go:835-880` history 가공: 없으면 과거 초안이 매 턴 누적된다.
- `toolhandler/definitions.go`, `models/tool/main.go` 상수, `AllBuilderToolNames`, `BuilderDraftableToolNames`.
- Gemini 경로의 `drop_gemini_invalid_tools`가 도구를 조용히 제거하지 않도록 스키마를 단순하게 유지하고(중첩 enum 금지) 실제 Gemini 호출로 검증한다.

### 5.6 프런트 (square-admin, 별도 PR)

`TestAgentSheet`는 재사용하지 않는다(모든 role raw 표시, 대기 표시 없음, 에러는 alert, F8). Builder 전용 신규 컴포넌트를 만든다.

화면:
- 템플릿 모달에 "Build with AI" 카드. `GET /ai_builder/status`의 `available`이 true일 때만 노출한다(권한 없는 에이전트에게는 false). status 호출이 404(구버전 백엔드)이거나 실패하면 카드를 숨긴다.
- 좌측 채팅, 우측 초안 미리보기. 채팅에는 user와 assistant 텍스트만 표시하고 system, tool, 빈 content의 도구 호출, notification(오류 안내 제외)은 숨긴다.
- 초안 패널은 가장 최근의 `propose_assistant_draft` **결과(role=tool) 메시지**만 읽는다. assistant의 도구 호출 메시지는 서버 필터링 전 raw 인자를 담으므로 읽지 않는다(`CaseInsightAssistantPanel.js`의 `parseToolCardBlocks`와 같은 방식).
- 미리보기는 plain text로만 렌더링한다(마크다운, HTML 비해석). 도구는 이름 그대로 목록으로 보여 준다.
- `dropped_tools`가 있으면 "일부 도구가 허용되지 않아 제외되었습니다: ..."를 표시한다.

메시지 조회와 폴링:
- `GET aimessages?aicall_id=&page_size=100`. 정렬은 최신순(F22)이라 첫 페이지가 최근 메시지다. 20턴 상한에서 턴당 user, assistant 도구 호출, tool 결과, assistant 텍스트 4행을 가정하면 약 80행에 system 행이 더해지는 정도라 최신 초안은 첫 페이지에 들어간다. 페이지 안에 초안이 없으면 초안 패널은 비어 있는 상태로 둔다.
- 폴링 간격은 선례(F8)와 같은 3초로 한다. `document.hidden`이면 폴링과 대기 타이머를 멈추고 복귀 시 먼저 한 번 조회한다.
- 응답 기준점은 `POST /aimessages` 응답이 돌려준 user 메시지의 id다(타임스탬프나 role 스캔을 쓰지 않는다. 텍스트 턴에서 pipecat이 user 행을 추가로 만들 수 있다, 5.4). 목록은 최신순(F22)이므로 기준 id를 만날 때까지의 앞쪽 행이 "이후" 메시지다.
- **응답 완료 판정(상태 기계)**: 기준 이후 메시지에서 (1) 대기 중인 도구 호출이 없어야 한다. 즉 assistant 도구 호출(빈 content, ToolCalls 있음)에 대응하는 role=tool 결과가 없으면 미완료다. (2) 가장 최근 메시지가 비어 있지 않은 assistant 텍스트여야 한다. assistant가 도구 호출 전에 텍스트를 먼저 내고 도구 호출 메시지가 아직 저장되기 전의 짧은 창에서는 잠금이 일찍 풀릴 수 있다. 이 경우의 영향은 다음 전송이 pipecat을 interrupt해 확인 문장이 잘리는 정도이며 데이터 손실이 아니다. 창의 길이는 실측 근거가 없으므로 타이브레이커(메시지 수 불변 확인)는 넣지 않고, 구현 1단계에서 실제 Gemini 세션으로 측정해 실제로 문제가 될 때만 추가한다(10절 트리거). 초안(role=tool 결과)이 먼저 나타나면 초안 패널은 즉시 갱신하되 입력 잠금은 위 판정이 끝날 때까지 유지한다(F1: 다음 전송이 이전 pipecat 세션을 interrupt 하므로 확인 문장이 잘릴 수 있다).
- 오류 신호: (1) 파이프라인이 뜬 뒤 provider 오류는 role=notification(`type=pipeline_error`)으로 저장되므로 대기 중에 나타나면 즉시 오류로 전환한다(F25). (2) 키 누락처럼 파이프라인이 뜨기 전의 실패는 사용자에게 전달되는 신호가 전혀 없다(F29). 이 경우 user 메시지는 저장되어 턴을 하나 소모하고 프런트는 무응답만 본다. 이 상태는 정상적으로 느린 응답(예: 큰 `propose_assistant_draft` 인자를 생성하는 첫 초안 턴, Gemini thinking, `max_tokens` 미설정 F26)과 프런트에서 구별되지 않는다. 따라서 사용자 문구는 원인을 단정하지 않는 중립형으로 한다: "응답이 평소보다 오래 걸리고 있습니다. 계속되면 서비스 관리자에게 문의하세요." 키 설정 확인 안내는 사용자 UX가 아니라 셀프호스팅 운영자용 smoke 절차(5.9)에만 둔다. 같은 질문을 자동 재전송하지 않는다.
- 대기 시간: 1차 안내 시점과 총 대기 상한은 구현 1단계의 실측(실제 Gemini 세션에서 첫 초안 턴 p95, `idleWatchdogTimeout` 8초 동안 도구 인자 생성 중 텍스트만 먼저 저장되는 창의 길이, 도구 호출 후 확인 텍스트 미생성 비율, 턴당 messages 행 수, soft delete 행에서 `ProcessTerminate`의 `Get` 동작, `FlowV1ActiveflowStop`의 ended 전환, 러너의 연속 도구 호출 반복 상한)으로 확정한다. 초기 가정은 1차 45초에 인라인 안내("응답이 평소보다 오래 걸리고 있습니다. 계속되면 서비스 관리자에게 문의하세요"와 새 대화 버튼은 항상 노출), 총 120초에 잠금 해제이며 "계속 기다리기" 버튼은 두지 않는다(입력이 이미 잠겨 있고 폴링이 계속되므로 안내를 닫는 동작 외에 하는 일이 없다). 선례 `TestAgentSheet`에는 폴링 타임아웃이 없어 근거가 되지 않는다. 타이머는 hidden 구간을 제외하고 탭 복귀 시 먼저 한 번 조회한 뒤 판정한다(복귀 즉시 총 대기가 지난 것으로 오판하지 않는다). **120초 경과 후 상태:** 입력 잠금을 풀고 "응답을 받지 못했습니다. 새 질문을 입력하거나 새 대화를 시작하세요"를 표시한다. 자동 재전송은 하지 않으며, 입력이 풀린 뒤 전송하면 이전 pipecat 세션이 interrupt될 수 있음(F1)을 수용한다. 재시도로 같은 질문을 다시 보내는 것은 사용자의 선택이며 턴을 소모한다.
- 이탈 후 복귀, 새로고침: `POST /ai_builder/sessions`가 재개이므로 호출한 뒤 `GET aimessages`로 대화와 초안을 복원한다. 마지막 메시지가 user이면 응답 대기 중으로 보고 폴링을 재개하며 대기 타이머를 새로 시작한다(클라이언트와 서버 시계를 비교하지 않는다. 비용은 오래된 무응답이 최대 총 대기 시간만큼 더 대기로 표시되는 것뿐이다). 마지막 user 뒤에 notification이 있으면 오류 상태로 표시한다. 복원된 대화에는 "이전 대화를 이어서 표시합니다"와 새 대화 버튼을 함께 보여 준다.
- 최신 초안은 첫 페이지(100행)에서 찾는다. 20턴 상한에서 약 80행이므로 조건이 드물다. 첫 페이지에 없을 때의 추가 조회는 실측에서 필요가 확인될 때 도입한다(10절). 실패 시 패널은 비어 있고 새 대화 안내를 보여 준다.

상태와 흐름:
- 첫 화면: 외부 LLM으로 대화가 전송된다는 고지, 정적 시작 안내 문구, 정적 예시 칩.
- 남은 턴 수: 서버가 세션 응답과 `GET /ai_builder/status` 호출 결과에 `turns_used`를 돌려준다(서버가 턴 상한 검사에 이미 세는 값. 프런트는 메시지 행을 세지 않는다. 텍스트 턴에서 pipecat이 user 행을 추가로 만들 수 있고 100행을 넘을 수 있기 때문이다, 5.4). `POST /aimessages` 성공 후에는 응답 대기가 끝날 때 세션을 다시 조회해 갱신한다.
- 초안 갱신 시 "초안 vN 업데이트됨" 배지(role=tool 결과 수).
- 오류 매핑: 기능 비가용, 일일 생성 상한(`AI_BUILDER_DAILY_LIMIT`, 안내에 "내일 다시 시도하거나 support@voipbin.net으로 문의"), 턴 상한(`AI_BUILDER_TURN_LIMIT`), 유휴 만료로 새 세션이 열릴 때 "새 대화로 시작합니다" 표시, 네트워크 실패. Send 쿨다운 오류는 응답 대기 중 입력 잠금이 막으므로 별도 매핑을 두지 않는다.
- **"새 대화" 버튼**: 항상 표시한다. 누르면 현재 세션을 `terminate`하고 새 세션을 시작하며 일일 생성 상한을 소모한다는 안내를 보여 준다. 턴 상한에 도달하면 이 버튼이 기본 동작이 된다. 한 AI를 완성한 다음 날 다시 들어오면 이전 대화가 복원되므로(유휴 만료 24시간 전까지) 이 버튼이 새 AI를 만드는 정상 경로다.
- "이대로 만들기": 초안을 라우터 state로 기존 생성 폼에 전달해 name, detail, init_prompt, tool_names를 채운다. 폼에 사용자가 편집한 내용이 있으면 덮어쓰기 확인을 묻는다. **세션을 terminate하지 않는다**(v2 정정). 사용자가 폼에서 돌아오면 같은 세션을 재개해 대화를 이어간다. 비용은 턴 상한과 유휴 만료가 묶는다. 새로고침으로 router state가 사라져도 세션 재개로 최신 초안을 복원할 수 있다. 저장은 사용자가 기존 `POST /ais`로 한다(서버 검증은 기존 `ValidateToolNames` 그대로).

### 5.7 위협 모델

| 위협 | 대응 |
|---|---|
| 사용자가 붙여 넣은 외부 텍스트로 Builder 조작, 악성 초안 | 시스템 프롬프트에서 사용자 입력과 지시 분리, 초안은 자동 저장하지 않고 사용자가 폼에서 확인, 도구는 서버 허용 목록 대조 |
| 초안 tool_names에 위험 도구 | `BuilderDraftableToolNames` fail-closed 허용 목록(6종), 폼에서 도구 목록 확인, 최종 저장은 기존 `ValidateToolNames`(단, 이 검증은 `all`을 허용한다. 초안 단계에서는 `all`을 제거하지만 사용자가 폼에서 직접 `all`을 넣는 것은 기존 동작이다) |
| 초안이 고객 통화 AI의 실제 프롬프트가 됨 | 폼 확인 단계 유지, 길이 상한 |
| 미리보기 XSS | plain text 렌더링 |
| Builder가 부작용 도구로 외부 행위 | Builder 허용 도구 1개, MCP 차단(두 곳), `AllBuilderToolNames` 테스트 고정 |
| 직접 호출로 상한 우회(`POST /aicalls`, Flow `ai_talk`, SIP 수신) | `Start`/`StartTask` 단일 차단점(A1) |
| ServiceAgent 라우트로 비관리자 에이전트의 Builder 대화 접근 | `aicallGet` default-deny(B2), 목록 필터 거부(B3) |
| 같은 테넌트 admin/manager의 Builder 대화 열람, `AIAuditCreate`로 LLM 비용 유발 | 열람은 G3로 수용. `AIAuditCreate`는 `aicallGet` default-deny로 차단(B2) |
| Builder 행의 공개 direct hash | `CreateBuilder`가 direct hash를 만들지 않고 `Start`가 builder를 거부 |
| 세션 남발 | 일일 생성 상한, 활성 세션 유일 제약, 킬 스위치. 계정 수에 비례하는 전체 총량 상한은 없다(11절 결정 사항) |
| 플랫폼 키 노출 | Builder 행 engine_key 빈 값, 키는 러너 환경변수에서만 읽음 |
| billing이 향후 ai-manager 이벤트를 구독 | Builder 세션은 이벤트를 발행하지 않음(A2)을 테스트로 고정 |
| Builder 대화(고객 내부 정보)가 customer webhook, WS로 유출 | A2 default-deny와 grep 테스트. 시스템 프롬프트 행(`start.go:1074`)과 도구 호출 raw 인자(`tool.go:113`)도 `messagehandler.Create` 내부 차단으로 막힌다 |
| 같은 테넌트 비관리자 에이전트가 시스템 프롬프트와 대화를 열람 | B2, B3 (G3: admin/manager는 수용) |
| 내부 아카이브(timeline-manager)와 `/aggregated_events`로 대화 텍스트 일부 노출 | 같은 테넌트 admin/manager 범위(G3)로 수용, 5.8 |

### 5.8 개인정보와 보존

- 대화가 외부 LLM 제공자로 전송된다는 고지를 첫 화면에 표시한다.
- 자동 삭제 배치와 사용자 삭제 UI는 만들지 않는다(기존에 없고 실측 신호 전의 선제 개발). 도입 트리거는 10절. Builder aicall은 고객이 `DELETE`할 수 없다(B2).
- timeline-manager가 pipecat 직접 발행 이벤트를 내부 아카이브로 보존하고(F24), 같은 테넌트 admin/manager가 `/aggregated_events`로 일부를 볼 수 있다(F31). 타 테넌트 노출 경로가 아니다. 대화 텍스트의 보존 기간이 기존 이벤트 정책을 따른다는 점을 수용하고 약관/DPA 표기 검토 항목으로 올린다(11절).
- 내부 열람 정책(지원 목적 열람)은 열린 질문(11절).

### 5.9 가용성(셀프호스팅)

가용성은 ai-manager 설정 `ai_builder_enabled`(기본 false)로 판정한다. ai-manager는 러너 컨테이너의 환경변수를 볼 수 없으므로(F3, 서로 다른 프로세스) 키 존재 여부를 코드가 추측하지 않는다. 계약:
- 운영자는 pipecat 러너 환경에 선택한 엔진의 키를 설정하고 `ai_builder_enabled=true`로 켠다. 셀프호스팅에서 "플랫폼 = 운영자 본인"이므로 이것이 정상 경로다.
- docsdev에 엔진별 필요 키 표(`gemini.*`는 `GOOGLE_API_KEY`, `openai.*`는 `OPENAI_API_KEY`, `grok.*`는 `XAI_API_KEY`)를 둔다. `ai_builder_engine_model`의 접두가 이 3종이 아니면 설정 검증이 거부한다.
- 켰는데 키가 없으면 러너의 서비스 생성이 예외를 던지지만 그 실패는 사용자에게 전달되지 않는다(F29: `RunnerStart` 고루틴이 로그만 남긴다). 사용자 화면에는 5.6의 중립형 지연 안내가 나온다. 운영자 점검 절차로 docsdev에 "활성화 직후 smoke 대화 1회"와 "무응답이면 pipecat-manager 로그의 `Could not start the pipecat runner script`(`runner.go:88`, 로그에는 `pipecatcall_id`만 있고 aicall id는 없다) 확인"을 명시한다(`/status`는 키 존재를 알 수 없다). pipecat-manager가 `RunnerStart` 실패를 notification으로 남기는 변경은 새 신호 채널이므로 하지 않으며 셀프호스팅 첫 실패 지원 요청이 실측으로 확인되면 도입한다(10절 트리거). 키가 있으나 잘못된 경우는 파이프라인이 뜬 뒤 provider 인증 오류가 notification으로 저장되어 즉시 오류로 전환된다(F25). 구 vendor 사본이 builder를 모르는 경우(F17)에는 세션이 정상 시작되고 도구만 조용히 사라지므로 8절의 `UnknownAITypeToolDenialTotal` 경보가 유일한 신호이고 사용자에게는 같은 무응답으로 보인다.
- 설정 방법은 docsdev RST에 문서화한다.

## 6. API 요약

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | /ai_builder/status | 가용 여부와 한도 |
| POST | /ai_builder/sessions | 세션 시작 또는 재개 |

신규 RPC: `AIV1BuilderSessionStart`(재개 또는 생성, 하나). OpenAPI와 docsdev RST를 추가한다. `models/*/webhook.go` 변경은 없다.

OpenAPI 처리(오류 방지): `AIManagerAIcallReferenceType` enum은 요청과 응답이 공유한다. `POST /ai_builder/sessions`의 응답이 reference_type `builder`를 돌려주므로 enum에 `builder`를 추가하고 설명에 "내부용, `POST /aicalls`에서는 거부된다"를 적는다. 런타임 거부는 `Start`의 switch default와 B4가 한다. AI `type` enum은 변경하지 않는다(AI 응답에 builder가 나오지 않으므로).

## 7. 구현 순서

1. 코드 전수 열거(표로 기록. 항목 추가만 가능하고 설계 구조는 바꾸지 않는다): aicall 소비자(aiaudit, summary, timeline, 통계), `messagehandler.Create`와 `PublishWebhookEvent` 호출 지점 재확인, `POST /aimessages` 단일 경로 재확인, 텍스트 턴의 user 행 중복 여부(5.4), `POST /ais` 길이 검증 값(5.5), Gemini 도구 스키마 실검증.
2. ai-manager: 상수와 `AllBuilderToolNames`, `BuilderDraftableToolNames`, `CreateBuilder`, `StartBuilder`/`BuilderSessionStart`, A1~A5, 도구 정의와 핸들러와 history 규칙, Send 검사, 시스템 프롬프트, 설정.
3. common-handler: requesthandler RPC.
4. api-manager: 라우트, servicehandler, B1~B4(`aicallGet` default-deny와 `aicallGetBuilder`), OpenAPI.
5. pipecat-manager: ai-manager vendor 갱신.
6. docsdev RST.
7. 프런트(monorepo-javascript, 별도 저장소 PR).

## 8. 배포 순서

ai-manager 배포, pipecat-manager vendor 갱신 후 재배포(도구 캐시가 기동 시 갱신되므로 재시작 필수, F17), api-manager 배포, 프런트 배포 순서다. 프런트는 `GET /ai_builder/status`가 available을 돌려줄 때만 카드를 노출하므로 순서가 어긋나도 깨지지 않는다. pipecat vendor가 구버전이면 Builder 세션마다 `UnknownAITypeToolDenialTotal`이 증가하므로 배포 전 조기 경보로 쓴다.

## 9. 테스트

- `AllowedToolNames(TypeBuilder)`가 정확히 `propose_assistant_draft` 하나이고 Normal, Insight에 이 도구가 없음을 양방향으로 고정(`allowed_tools_test.go`의 wantDisallowed 확장).
- Builder 도구 집합에 음성, TTS, SMS, 이메일, 통화, MCP 도구가 없음. `BuilderDraftableToolNames`가 `AllToolNames`의 부분집합이고 정확히 6종임.
- 고객 `POST /ais`, `PUT /ais/{id}`가 `type=builder`를 거부(기존 `IsValid()`), `aihandler.List`가 builder를 기본 제외, listenhandler `PUT`/`DELETE`(`UpdateMcpServerIDs` 경로 포함)와 `Update`/`Delete`가 builder 행을 거부.
- `Start`, `StartTask`가 builder AI를 거부(`POST /aicalls`, Flow ai_talk 경로 포함), reference_type=builder 직접 요청 거부.
- teamhandler가 builder 멤버를 거부(생성, 수정).
- api-manager: `aiGet`이 builder에 NotFound를 돌려 10개 호출 지점 모두에 적용됨(표 기반 테스트). `aicallGet`이 builder에 NotFound를 돌려 `ServiceAgentAIcallGet`, `ServiceAgentAImessage*`, `ServiceAgentAIcallListen`, `AIAuditCreate`, `AIcallDelete`가 builder를 거부하고, `aicallGetBuilder` 직접 호출처가 정확히 `AIcallGet`, `AIcallTerminate`뿐임(grep 기반 테스트)이고 `AIcallGet` 경유 3곳(`AImessageCreate`, `AImessageGetsByAIcallID`, `AIParticipantGets`)이 G3 범위 의도대로 동작함. 목록 필터에서 builder 값 거부(B3).
- `CreateBuilder`가 direct hash를 만들지 않음.
- 발행 차단: Builder aicall 상태 7곳, `message_created`(시스템 프롬프트 행, 도구 호출 행 포함), `message_intermediate`가 `PublishWebhookEvent`를 호출하지 않음(strict gomock no-EXPECT). aicallhandler와 messagehandler에 헬퍼 밖 직접 `PublishWebhookEvent` 호출이 없음을 grep 기반 테스트로 고정.
- `propose_assistant_draft`: 허용 목록 대조, `all` 제거, 길이 상한, 중복 제거, `dropped_tools`, LLM 결과 문자열에 초안 본문이 없음, `c.ReferenceType==builder` 단언, history 규칙(최신 호출 인자만 전문 유지, 이전 호출은 stub, tool 결과는 확인 문자열).
- 상한: 일일 세션 생성(soft delete된 행 포함, 429), 턴 상한, 메시지 길이(rune), 종료, soft delete 상태 거부(`POST aimessages` 직접 호출 포함), 킬 스위치.
- 세션: 동시 시작 요청에서 활성 세션이 에이전트당 1개이고 재개됨(중복 키 경로, 패자 activeflow 정리). `agentID == Nil`, accesskey, direct, delegate 거부. 재개가 상한을 소모하지 않음. Builder activeflow 종료 시 `FlowV1ActiveflowStop` 호출, `CreateBuilder`가 `ai_created`를 발행하지 않음. 소유자가 아닌 admin/manager의 `AImessageCreate`, `AIcallTerminate`가 builder에서 거부됨(열람 `AIcallGet`은 허용). soft delete되거나 유휴 만료된 세션이 재개되지 않고 terminate 후 새로 만들어짐. 재개 시 customer_id 일치 재확인.
- customer 불변식: `BuilderSessionStart`에서 aicall.customer_id, ai.customer_id, JWT customer_id 일치.
- 기능 비가용(`ai_builder_enabled=false`) 시 명시 에러와 status.
- 실제 Gemini 호출로 도구 스키마가 제거되지 않음을 확인(수동 검증 항목).
- 프런트: 응답 완료 판정 상태 기계(도구 호출 대기, 텍스트 먼저 오는 턴, 초안 먼저 오는 턴), 기준점은 POST 응답의 메시지 id, notification 오류의 즉시 전환, 무신호 시 중립형 지연 인라인 안내와 새 대화 버튼 상시 노출, 총 대기 후 입력 잠금 해제(실측 확정값), hidden 구간 타이머 제외와 복귀 시 선조회, 복귀 시 마지막이 user이면 타이머 재시작, 복원 표시, `turns_used`는 서버 값 사용, 폴링 중단(hidden), 새로고침 복원(마지막이 user이면 대기 재개), 새 대화 버튼, 폼 이동 후 세션 유지, status 404 또는 권한 없음 시 카드 숨김.

## 10. 도입 트리거(지금은 만들지 않는 것)

| 항목 | 도입 조건 |
|---|---|
| 고객 일일 토큰 총량, 플랫폼 전역 예산 카운터(예: Redis 일일 전역 세션 카운터)와 자동 서킷브레이커 | 플랫폼 LLM 비용에서 Builder 비중이 문제로 관측되거나 계정 수 증가로 전체 비용이 임계에 가까워지는 신호가 생길 때. 도입 시 토큰 계측 출처(응답 사용량 필드 수집)부터 설계한다 |
| Builder 전용 `max_tokens`, 턴당 도구 호출 상한 | 출력 토큰 폭주 또는 한 턴 안의 연속 도구 호출 비용 이상이 관측될 때(F26). 도입 전에 러너에 이미 반복 상한이 있는지 확인한다 |
| 응답 완료 판정의 메시지 수 불변 타이브레이커 | 구현 1단계 실측에서 잠금이 일찍 풀려 확인 문장이 잘리는 사례, 또는 도구 호출 후 확인 텍스트가 생성되지 않아 초안은 보이는데 입력이 총 대기 시간 동안 잠기는 비율이 확인될 때 |
| 첫 페이지에 최신 초안이 없을 때의 추가 페이지 조회 | 실측에서 100행 초과 턴 구성이 확인되거나 초안 패널 공백 사례가 보고될 때 |
| pipecat-manager의 `RunnerStart` 실패 notification 발행 | 셀프호스팅 첫 실패(키 누락 등) 지원 요청이 실측으로 확인될 때(F29) |
| flow-manager의 builder activeflow 이벤트 억제 | 고객 외부 자동화가 Builder activeflow 이벤트를 운영 이벤트로 오인한 사례가 확인될 때 |
| 대화 자동 삭제 배치와 삭제 UI | 저장 용량 또는 보존 정책 요구가 생길 때 |
| 에이전트 간 Builder 세션 공유 | 같은 고객 관리자 간 협업 요구가 확인될 때 |
| Builder 대화 기반 편집 모드 | 생성된 AI의 수정 요구가 확인될 때 |
| 초안 도구 허용 목록 확장(create_call 등) | 사용자가 Builder에서 제외된 도구를 요구하는 신호가 확인될 때 |

## 11. 열린 질문

| 질문 | 추천 | 결정 주체 |
|---|---|---|
| **구조 결정(최우선):** (a) aicall 기반 세션(현재 v7 본문) vs (e) 상태 없는 동기 멀티턴(4절). 7회차 리뷰어 B가 (e) 비교 누락을 지적했고 코드 확인 결과 사실이다 | (e). 확정 결정 3가지를 모두 충족하면서 차단 인프라, 폴링, 키 누락 무신호를 없앤다. 대가는 서버 측 턴 이력 부재와 새로고침 복원이 브라우저 보관이라는 점 | 대표님 |
| 플랫폼 부담 비용의 전역 상한을 지금 둘지. 의견이 갈린다. 리뷰어 A는 계정 수에 비례해 곱해지므로 저렴한 Redis 일일 전역 카운터를 지금 넣자는 입장이고, 리뷰어 B는 실측 신호 전 선제 확장이라는 입장이다. 현 설계는 고객당 일일 상한과 킬 스위치만 두었고 전역 카운터는 10절 트리거로 미룸 | 킬 스위치 기본 false(수동 활성)와 고객 일일 상한만으로 시작하고 활성화 후 일일 세션 수를 관찰. 전체 비용이 우려되면 카운터를 추가 | 대표님 |
| 상한 초기값(일일 생성 10, 세션당 20턴, 메시지 2000자) | 보수적 시작 후 실측 조정 | 대표님 |
| 토큰 단가 기준 금액 환산에 쓸 provider 단가 | 단가 확인 후 5.4의 토큰 수로 환산 | 대표님 |
| 무료 체험 계정 허용 여부 | 허용(동일 상한) | 대표님 |
| Builder 대화 로그의 내부 열람 허용 여부 | 지원 목적 한정 열람 금지(기본) | 대표님 |
| 외부 LLM 전송과 timeline 내부 아카이브 보존에 대한 약관/DPA 표기 | 첫 화면 고지 외에 약관 검토 | 대표님 |
| 성공 지표와 철수 기준. 서버는 "Builder에서 온 저장"을 알 수 없다(폼 전달만 있음) | 1차 지표는 세션 시작 수와 상한 위반 수. 저장 비율이 필요하면 저장 요청에 출처 힌트를 추가하는 별도 변경이 필요 | 대표님 |
| 기본 모델(gemini-2.5-flash)의 한국어 초안 품질과 비용. 활성화 전에 샘플 대화로 검증 | 활성화 전 수동 비교(기존 8개 템플릿 대비) 후 결정 | 대표님 |
| 호스팅 서비스에서 켜는 시점과 범위. 킬 스위치는 전역이라 일부 고객만 켜는 것은 불가 | v1은 허용 목록 없이 샌드박스와 내부 검증 후 전역 활성화. 베타 범위가 필요하면 그때 허용 목록을 추가 | 대표님 |

확인만 요청(이견이 없으면 그대로 진행. 추천이 확정되어 있고 대안이 12절에서 기각됨):

| 항목 | 내용 |
|---|---|
| G2 하드 유지 | 대화 내용이 고객 외부 채널로 나가지 않게 한다. 발행 중앙화(A2)와 `aiGet`, `aicallGet` default-deny의 근거. 완화하면 aicall 상태 이벤트(F14)가 고객 webhook으로 나간다. **비용(대표님 인지 필요):** `messagehandler.Create`에 캐시 우선 aicall 조회가 한 번 추가되고, 조회 실패 시 fail-closed로 Builder를 쓰지 않는 기존 고객의 `message_created` webhook도 일시적으로 누락될 수 있다 |
| 호스팅 러너의 LLM 키 | Builder 기본 엔진(gemini)의 키(`GOOGLE_API_KEY`)가 호스팅 러너 환경에 이미 있는지 활성화 전에 확인. 없으면 키 누락(F29)이 첫 장애가 된다 |
| G3 | 같은 테넌트 admin/manager에게 열람 허용, 쓰기(메시지, 종료)는 소유 에이전트만 |
| 초안 도구 6종 | 기존 템플릿과 같은 범위(create_call 제외) |

## 12. 변경 이력

### v1 대비 v2 (1회차 리뷰 반영)

- 비용 상한이 세션 진입점에만 있어 우회 가능했던 문제: 턴, 길이 상한을 `aicallhandler.Send`로 이동.
- Flow ai_talk, SIP, 직접 `POST /aicalls` 우회: `Start`/`StartTask` 단일 차단점.
- Builder 행 direct hash: `CreateBuilder`가 생성하지 않음.
- aicall 식별자: `ReferenceTypeBuilder`. 신규 마이그레이션 없이 기존 유일 제약 재사용.
- webhook/WS 충돌: 발행 차단과 폴링 채택.
- enum 노출: `validTypes`에 넣지 않음.
- 가용성 판정 주체를 설정 플래그로 확정. pipecat 정정(vendor 갱신과 재시작).
- 토큰 상한 2개와 계측 대시보드 제거, 최악 비용 계산 추가.
- 삭제 의미 정정: `DELETE`는 soft delete, 종료는 `terminate`.

### v2 대비 v3 (2회차 리뷰 반영)

수용(코드로 확인):
- ServiceAgent 라우트 노출(F20): `assertBuilderAccess`와 목록 필터 거부(B2, B3). 세션 소유자를 에이전트로 바꿔 `reference_id = agent_id`로 규칙을 단순화(5.1).
- `message_created` opt-in 억제는 fail-open(F21, 15곳): `messagehandler.Create` 내부에서 aicall을 조회해 default-deny, webhook 발행을 헬퍼로 중앙화하고 grep 테스트로 고정(A2). aicall 상태 발행은 6곳이 아니라 7곳으로 정정.
- soft delete된 활성 세션이 유일 제약 슬롯을 점유하고 상한 카운트를 흐림(F12, F16): 재개 조건에 `TMDelete == nil` 추가, terminate 후 재생성, 상한 카운트에 삭제 행 포함, 고객의 Builder aicall `DELETE` 거부.
- AI id를 받는 조회/변경 경로가 v2 목록보다 많음(F23, `UpdateMcpServerIDs`): `aiGet` 단일 헬퍼에서 builder를 NotFound 처리(B1), listenhandler `PUT`/`DELETE` 진입부 가드(A4).
- `RiskyToolNames` fail-open과 분류 누락, 동시에 리뷰어 B의 오버엔지니어링 지적: 둘을 함께 해소하기 위해 `RiskyToolNames`, `risky_tools`, `revision`을 제거하고 fail-closed `BuilderDraftableToolNames` 허용 목록(기존 템플릿이 쓰는 6종)으로 대체.
- 폴링 종료 조건이 도구 호출 턴 메시지 순서와 맞지 않음(F1): 응답 완료 판정을 "마지막 메시지가 assistant 텍스트이고 2회 연속 변화 없음"으로 변경, 초안은 즉시 갱신하되 입력 잠금 유지. 폴링 간격은 선례와 같은 3초, hidden 중단, 대기 120초와 "계속 기다리기".
- "이대로 만들기"에서 세션 terminate가 반복 다듬기 UX를 해치고 상한을 소모함: terminate하지 않고 세션을 유지.
- 고객 단위 세션 공유의 동시성 결함: 에이전트 단위 세션으로 변경.
- pipeline 오류 신호(F25) 활용: 셀프호스팅 첫 실패를 notification 메시지로 즉시 표시(5.6, 5.9).
- Send 세부(F11): terminated, terminating, TMDelete 거부, rune 기준 길이, 턴 정의, TOCTOU 수용, 쿨다운 오류 매핑.
- `NotEq`를 RPC 경계로 넘기는 설계의 위험(F18): ai-manager 내부 기본 제외와 api-manager의 builder 필터 거부로 변경.
- 최악 비용에 출력 토큰과 `max_tokens` 미설정(F26) 추가. 금액 환산은 단가 미확인으로 하지 않고 대표님 결정 항목으로 올림.
- OpenAPI enum(`AIcallReferenceType`)이 응답과 공유됨: `builder` 추가와 런타임 거부로 처리(6절).
- history 규칙 모순: 최신 호출 인자 전문 유지, 이전은 stub으로 정리(5.5).
- 활성 세션 재개 시 `GetByReferenceID`가 customer, 삭제 필터가 없음: DB 계층 조회로 변경(5.3).
- 동시 시작 경합 패자의 고아 activeflow 정리.

기각 또는 조건부 수용:
- 리뷰어 A M1 "발행 차단이 한 번에 소비자를 막는다는 주장이 부정확": 부분 수용. pipecat 직접 발행 이벤트(F24)는 ai-manager에서 막지 못하는 것이 맞다. 그러나 api-manager의 customer WS는 webhook-manager의 `webhook_published`만 소비하므로(`subscribehandler/main.go:139,159`) customer-visible 채널(webhook, WS)은 A2로 차단된다. 잔존 경로는 timeline-manager 내부 아카이브이며 같은 테넌트 범위로 한정되어 5.8에서 수용 결정으로 올렸다.
- 리뷰어 B 이슈 2 "page_size=100이면 최신 초안이 유실": 기각. 메시지 목록은 `tm_create desc`라 첫 페이지가 최신이다(F22). 20턴 상한에서 약 80행 수준이라 한 페이지에 최신 초안이 포함된다. 정렬 방향과 산술을 5.6에 명시했다.
- 리뷰어 B "시간당 5회와 일일 10회는 같은 역할이므로 시간당 하나만": 기각. 시간당 5회만 두면 하루 120세션까지 가능해 일일 10회가 실질 상한이다. 반대로 시간당 상한을 제거하고 일일 상한만 남겼다(5.4).
- 리뷰어 A M3의 "읽기 전용이지만 민감한 도구까지 위험 분류": 분류 맵 대신 fail-closed 허용 목록으로 해소했다(분류 누락이 구조적으로 불가능).
- 리뷰어 B "고객당 세션 공유 추천": 에이전트 단위로 변경.
- 리뷰어 B 성공 지표 계측 요구: 서버가 출처를 알 수 없다는 점을 확인했고, 별도 변경이 필요하므로 설계에 넣지 않고 11절 결정 사항으로 올림.

### v3 대비 v4 (3회차 리뷰 반영)

신뢰 모델 단순화(핵심 변경): v3의 "소유 에이전트만 열람(`assertBuilderAccess`)"은 7개 함수에 per-function opt-in으로 붙어 있어 `AIAuditCreate`, `AImessageGet`, `AImessageDelete`, `ServiceAgentAIcallListen`이 누락되었고(리뷰어 A H1), `/aggregated_events`로도 우회되었다(H2). 보호 목표를 G1(하드, 사용 경로), G2(하드, 고객 외부 채널과 목록 혼입), G3(수용, 같은 테넌트 admin/manager 열람)로 분리하고, 열람은 기존 aicall 권한 모델과 일관되게 수용했다. 결과적으로 `assertBuilderAccess`를 삭제하고 `aicallGet` default-deny와 `aicallGetBuilder` 두 호출처로 대체했다(B2). 리뷰어 B가 지적한 차단 인프라 규모도 이 변경으로 줄었다. B1(`aiGet`), A2(발행 중앙화)는 유지한다. A2를 유지하는 이유는 같은 테넌트 문제가 아니라 G2다. Builder 시스템 프롬프트와 도구 raw 인자가 고객의 외부 webhook URL로 전송되고 고객의 외부 자동화가 Builder 이벤트를 운영 aicall 이벤트로 오인하기 때문이다. billing 구독 대비는 부차적 이유다.

수용(코드로 확인):
- 키 누락 시 실제 동작(리뷰어 B 1번): 러너 venv(pipecat 1.4.0)로 직접 실행해 확인했다. 키가 없으면 서비스 생성 단계에서 예외가 나고 `POST /aimessages`의 동기 오류가 된다(F29). v3의 "notification이 저장된다"는 틀렸다. 5.6과 5.9를 정정했다. (4회차에서 이 정정이 다시 틀렸음이 확인되어 v5에서 재정정: 동기 오류가 아니라 무신호)
- 응답 완료 판정의 조기 완료 오판(리뷰어 B 2번): 도구 호출 대기 여부를 포함한 상태 기계, 기준점을 POST 응답의 메시지 id로 고정(5.6).
- `/aggregated_events` 경로(리뷰어 A H2): G3와 5.8에 명시해 수용(F31).
- `agentID == Nil` 방어(H3): 5.1과 5.3, 테스트.
- Create 내부 aicall 조회 실패 시 동작(M1): fail-open과 근거를 명시(A2).
- 슬롯 점유 경합(M2): contact_case 재생성 루프 패턴 재사용, 종료 호출 확인은 구현 1단계(5.3).
- 일일 상한 조회 방법(M3): `AIcallList` size 10 조회와 10번째 `tm_create` 비교로 명시(5.4).
- 도구 호출 루프로 비용 상한 우회(M4): 세션당 초안 도구 호출 15회 상한 추가(5.4). 4회차에서 단위 오류로 삭제(v5 참조).
- 상한 오류 코드(M6): `ResourceExhausted`와 reason 코드로 분기.
- 최신 초안 페이지 경계(M5, 리뷰어 B 6번): 첫 페이지에 없으면 최대 3페이지 조회.
- 새 대화 버튼과 턴 상한 UX(리뷰어 B 3번), status 권한(4번).
- RPC 두 개를 하나로(리뷰어 B 9번): ai-manager가 activeflow를 직접 생성(F33).
- "너무 빠른 전송" 오류 매핑 제거, 12000자와 6000자 서술 정리, 11절에 빠진 결정 5건 추가.
- 표기 정정: 8개 템플릿 중 blank와 insight를 제외한 6개 템플릿이 쓰는 도구가 6종이다(F28는 이미 도구 6종으로 서술).

기각 또는 조건부 수용:
- 리뷰어 B 5번 "A2를 messagehandler.Create 한 곳으로 축소": 기각. `message_created` 외에 aicall 상태 이벤트(`aicall_status_*`)와 `message_intermediate`도 고객의 외부 webhook으로 전송된다(F14). G2 때문에 같은 헬퍼로 모아 default-deny 하는 것이 호출 지점을 줄이는 방향이다(14곳이 아니라 헬퍼 하나).
- 리뷰어 B 7번 "`TMDelete == nil` 조건의 발생 경로가 불분명": 조건을 유지한다. 고객 `DELETE`는 막지만 고객 삭제 배치, ai-manager 직접 RPC 등 다른 경로로 soft delete가 생길 수 있으며 비용이 거의 없는 방어 조건이다.
- 리뷰어 A L2 "응답 전용 enum 분리": 보류. 요청 스키마가 builder를 노출하지만 런타임에서 거부되며(A1, B4) 별도 enum은 생성 코드 변경이 커서 구현 단계에서 필요성이 확인될 때 판단한다.
- 리뷰어 B 성공 지표 계측: 3회차와 동일하게 서버가 출처를 알 수 없어 11절 결정 사항으로 유지.

### v4 대비 v5 (4회차 리뷰 반영)

수용(코드로 확인):
- F29 정정(리뷰어 A H1, B H1): 키 누락 오류는 `POST /aimessages`의 동기 오류가 아니다. Builder aicall은 pipecat-manager `startReferenceTypeAIcall`의 `default:` 분기를 타고 `RunnerStart`를 고루틴으로 띄운 뒤 즉시 성공을 반환하며, 러너 시작 실패는 로그만 남는다(`start.go:281-287`, `runner.go:84-90`). v4는 Python 단계까지만 확인하고 Go 전파를 가정했다. 5.6, 5.9를 "무신호, 45초 타임아웃 안내"로 정정했고 pipecat-manager 변경은 새 신호 채널이므로 도입하지 않으며 트리거로 둔다.
- 쓰기 경로 소유자 검사(리뷰어 A M1): 열람(G3)은 수용하되 `AImessageCreate`, `AIcallTerminate`는 builder에서 소유 에이전트만 허용한다. 기존 반환값으로 비교 한 줄이며 호출처는 늘지 않는다.
- A2 fail-closed(리뷰어 A M3): 조회 실패 시 발행을 건너뛴다. 호출 지점 수를 15곳으로 정정.
- 슬롯 해소(리뷰어 A M4): `ProcessTerminate`로 확정하고 부작용을 코드로 확인해 기술. `IsErrDuplicate` 재사용과 activeflow 정리를 명시.
- `AImessageDelete`는 ai-manager 라우트가 없다(리뷰어 A L1), timeline analysis 소비자 추가(L2) 정정.
- flow-manager의 activeflow 이벤트(리뷰어 A M2): 흐름 메타데이터이므로 수용하고 5.2 잔존 경로에 명시.
- 메시지 수 불변 타이브레이커 제거(리뷰어 B M1): 실측 근거가 없다. 실측 후 필요 시 추가.
- 세션당 도구 호출 15회 상한 삭제(리뷰어 B M2): 단위가 맞지 않아 정상 사용을 해친다. 같은 턴 안의 연속 도구 호출은 러너에 이미 반복 상한이 있는지를 구현 1단계에서 확인하고, 없고 비용 이상이 관측되면 `max_tokens` 또는 턴당 상한을 도입한다(10절 트리거).
- 11절 정리(리뷰어 B M3): 대표님 결정이 아닌 두 항목 삭제, G2 하드 유지 항목 추가.
- "차단 인프라가 줄었다" 서술 정정(리뷰어 B H2): B2 한 항목에서만 줄었고 항목 수(A1~A5, B1~B4)는 같다. G2를 하드로 유지하는 비용은 11절에서 대표님이 판단한다.

기각:
- 리뷰어 B M4("이대로 만들기" 후 새 대화를 기본 동작으로): 5.6의 새 대화 버튼과 상한 안내로 충분하다고 판단한다. 저장 직후 재진입에서 이전 대화가 복원되는 것은 의도이며, 새 AI를 만드는 정상 경로는 새 대화 버튼이다. 일일 10개 상한은 초기값이고 실측 후 조정한다(11절).

### v5 대비 v6 (5회차 리뷰 반영)

수용(코드로 확인):
- 리뷰어 B H1(45초 키 오류 안내가 느린 정상 초안 턴을 오탐): 사용자 문구를 원인을 단정하지 않는 중립형으로 바꾸고 키 안내는 운영자 smoke 절차로 옮겼다. 45초와 120초는 구현 1단계 실측(첫 초안 턴 p95)으로 확정하는 값으로 낮췄다. `TestAgentSheet`에 폴링 타임아웃이 없다는 점도 확인했다. 120초 경과 후 상태를 정의했다(M5).
- 리뷰어 A M1(`ai_created` 유출): `aihandler/db.go:107`이 `PublishWebhookEvent(ai.EventTypeCreated)`를 호출함을 확인했다. `CreateBuilder`는 이벤트 없는 DB 생성 경로를 쓰고 grep 테스트 범위에 aihandler를 포함한다.
- 리뷰어 A M2(activeflow 누적): `ServiceStop`은 stack pop만 하고 상태를 바꾸지 않으며(`activeflow.go:18-36`) `Continue`는 flow 액션이 없어 실패하므로 running으로 남는다. 종료 경로에서 `FlowV1ActiveflowStop`을 호출한다(`tool.go:901` 선례).
- 리뷰어 A: fail-closed 비용 서술(모든 일반 aicall의 `message_created` 일시 누락과 Create당 조회 1회)을 정직하게 기술, `event.go:480`만 `ac` 보유임을 정정, `AImessageCreate`의 반환값 비교 서술 정정, `PipecatcallTerminate` RPC 실패 서술 정정, activeflow `reference_type`은 선례를 따라 `ai`, 구 vendor 사본 신호 연결(F17).
- 리뷰어 B M2(복귀 시 시계 비교): 마지막이 user이면 타이머를 새로 시작하는 규칙으로 단순화.
- 리뷰어 B M3(11절): 결정 필요 항목과 확인만 요청 항목으로 분리.
- 리뷰어 B M4(F32에 `AImessageCreate` 근거), L1(F20 인용 정정, 복원 표시 문구).
- 리뷰어 B 오버엔지니어링 제안 2(페이지 3회 조회): 첫 PR에서 제외하고 트리거로 이동.

기각 또는 조건부 수용:
- 리뷰어 B M1(idle watchdog와 텍스트 먼저 턴): 설계 변경 없이 구현 1단계 실측 항목으로만 추가(문서의 기존 정책과 같다).
- 리뷰어 B 오버엔지니어링 제안 3(엔진 3종 지원을 gemini 하나로 축소): 설정 검증은 접두 확인 한 줄 수준으로 이미 작다. 5.9의 엔진별 키 표는 문서화이므로 코드 비용이 아니다. 유지.
- 리뷰어 A L1(soft delete 행의 `Get`): 구현 1단계 확인 항목과 조회 경로 조건으로 반영.

### v6 대비 v7 (6회차 리뷰 반영, A는 APPROVE)

수용(코드로 확인):
- 리뷰어 B 1번(9절이 삭제한 규칙을 요구): 3페이지 조회와 시계 비교 복원 테스트를 5.6과 일치시켰다.
- 리뷰어 B 2번: 서버가 `turns_used`를 돌려주고 프런트는 행을 세지 않는다.
- 리뷰어 B 3번: "계속 기다리기" 버튼 삭제, 인라인 안내와 새 대화 버튼 상시 노출.
- 리뷰어 B 4번: hidden 구간 타이머 제외와 복귀 시 선조회.
- 리뷰어 B 5번: 11절 G2 항목에 fail-closed 비용을 명시하고 호스팅 러너 LLM 키 확인 항목을 추가.
- 리뷰어 B 6번: 도구 호출 후 확인 텍스트 미생성 비율을 실측 항목과 트리거에 추가. 리뷰어 B 7번: 로그 위치 정정(`runner.go:88`, pipecat-manager만).
- 리뷰어 A M1: Stop을 `ProcessTerminate` 내부 builder 분기 하나로 모음(호출처 4곳과 슬롯 해소). 조기 return 분기에서도 멱등 Stop. `tool.go:901` 서술 정정.
- 리뷰어 A M2: `AIcallGet` 경유 3곳(`AImessageCreate`, `AImessageGetsByAIcallID`, `AIParticipantGets`)을 G3 범위로 명시하고 테스트에 포함.
- 리뷰어 A M3: A2 grep 범위에서 aihandler를 제외하고 행동 테스트로 대체.
- 리뷰어 A L1~L3: 조회 필요 호출자 중 일부가 이미 aicall을 읽음을 반영, F33 선례 근거 교체, activeflow `referenceID=uuid.Nil` 명시, Stop 실패 잔존 수용 사유.

보류:
- 리뷰어 B 규모 지적(12절 분량): 12절 분리는 구현 PR에 문서를 옮길 때 수행한다. 설계 구조 변경이 아니므로 본 루프에서는 유지한다.

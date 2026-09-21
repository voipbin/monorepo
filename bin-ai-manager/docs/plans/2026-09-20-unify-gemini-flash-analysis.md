# 이슈 분석: 기본 LLM 모델을 최신 Gemini flash로 통일 (VOIP-1537, PR1=A+B)

작성일: 2026-09-20
작성자: Hermes (CPO)
브랜치: VOIP-1537-unify-gemini-flash
상태: 이슈 분석 (리뷰 대기)

## 1. 목표

엔진이 사용자별로 별도 지정되지 않은 모든 LLM 기능의 기본 모델을 최신 Gemini flash
(`gemini-3.8-flash`, 2026-09-02 GA)로 통일한다. 이 티켓은 PR1으로 A(상수/설정 교체)와 B(Summary 엔진
전환)를 다룬다. C(AI Call 대화 폴백)는 PR2로 분리한다.

## 2. 배경 및 긴급성 (실측)

### 2.1 shutdown 데드라인
Gemini 2.5 계열은 2026-10-20 종료 예정(웹 조사: Google model lifecycle). 현재 사용 중:
- Analysis 기본 모델 = gemini-2.5-flash (config/main.go:135)
- Analysis allow-set = gemini-2.5-flash, gemini-2.5-pro (config/main.go:136)
- Audit = gemini-2.5-flash (geminiaudithandler/main.go:18)
- Proposal = gemini-2.5-pro (geminiproposalhandler/main.go:19)

방치 시 약 한 달 뒤 Analysis/Audit/Proposal이 모두 실패한다. 이 A 범위는 데드라인이 있는 필수 작업이다.

### 2.2 최신 flash 버전 확정
2026-09-21 기준 최신 stable flash = `gemini-3.8-flash` (2026-09-02 GA, Gemini API 공식 changelog).
대표님 확정: 전부 gemini-3.8-flash 단일로 통일(Proposal의 pro도 flash로 낮춤).

## 3. 현황 (실코드 전수 조사)

| 기능 | 현재 모델 | 엔진 경로 | 코드 위치 |
|---|---|---|---|
| Summary 생성 | gpt-4-turbo | OpenAI 순정(EngineKeyChatGPT) | summaryhandler/main.go:121 |
| Summary 언어검증 | gpt-4o-mini | OpenAI 순정 | summaryhandler/main.go:134 |
| Analysis | gemini-2.5-flash | Gemini OpenAI호환(config base URL + GoogleAPIKey) | config/main.go:135 |
| Audit | gemini-2.5-flash | Gemini OpenAI호환(GoogleAPIKey, 엔드포인트 상수) | geminiaudithandler/main.go:18 |
| Proposal | gemini-2.5-pro | Gemini OpenAI호환(GoogleAPIKey, 엔드포인트 상수) | geminiproposalhandler/main.go:19 |
| AI Call 대화(폴백) | gpt-4-turbo | OpenAI 순정 | engine_openai_handler/main.go:15 (PR2, 범위 외) |

주1: Audit/Proposal은 네이티브 Gemini SDK가 아니라 **go-openai 클라이언트를 Gemini OpenAI 호환
엔드포인트(`https://generativelanguage.googleapis.com/v1beta/openai/`)에 붙인 REST**다(geminiaudithandler
/main.go:17,80-81 / geminiproposalhandler/main.go:18,67-68, 둘 다 CreateChatCompletion 사용). Analysis와
동일 프로토콜이며 차이는 (a)엔드포인트를 상수 하드코딩 vs Analysis는 config, (b)키가 GoogleAPIKey인 점뿐.

주2: AI Call은 사용자가 AI별 `AIEngineModel`(ai.EngineModel)을 지정할 수 있고(models/aicall/main.go:83),
`GetEngineModelName(cc.AIEngineModel)`가 빈 문자열을 반환할 때 defaultModel로 폴백한다
(engine_openai_handler/message.go:32-33, streaming_send.go:44-45). GetEngineModelName은 미지정(빈값)뿐
아니라 "." 구분자가 없거나 끝에 오는 malformed 값에도 ""를 반환하므로(models/ai/main.go:196-203), 폴백은
"엔진 미지정 시" + "모델명 파싱 실패 시" 둘 다에서 발생한다. 이 폴백 전환은 tool/streaming 검증 필요로 PR2.

## 4. 작업 성격 분해

### A. 상수/설정 교체 (난이도 낮음, 데드라인 있음)
- Audit geminiModel: "gemini-2.5-flash" → "gemini-3.8-flash"
- Proposal geminiModel: "gemini-2.5-pro" → "gemini-3.8-flash" (pro→flash, 대표 확정)
- config analysis_default_model: "gemini-2.5-flash" → "gemini-3.8-flash"
- config analysis_allowed_models: "gemini-2.5-flash,gemini-2.5-pro" → gemini-3.8-flash 포함하도록 갱신
- (선택) models/ai/main.go EngineModel 레지스트리에 gemini-3.8-flash 항목 추가 검토

### B. Summary 엔진 전환 (난이도 중간)
현재 summaryHandler는 OpenAI 순정 engineOpenaiHandler를 주입받는다(ai-manager/main.go:168).
Gemini 전환 방법 후보:
- (B-1) main.go wiring에서 요약용 Gemini 엔진을 NewEngineOpenaiHandlerWithConfig(GoogleAPIKey,
  geminiBaseURL)로 별도 생성해 summaryHandler에 주입. 모델 상수는 gemini-3.8-flash로.
- (B-2) analysis처럼 config로 summary provider(base URL/키/모델) 선택 가능하게. VOIP-1197 패턴 재사용.
설계 단계에서 B-1 vs B-2 결정. 대표 원칙(기존 도구 재사용, 신규 체계 지양)상 VOIP-1197 패턴 재사용이
유력하나, summary 전용 config 신설이 과한지 검토 필요.

핵심 상호작용:
- summaryTemperature(0.2, VOIP-1536)는 Gemini OpenAI 호환 엔드포인트에서도 유효한 파라미터인지 확인
  필요. Gemini는 temperature를 지원하나 범위/기본이 다를 수 있음. 0.2 유지 타당성 설계에서 실호출 스모크로 검토.
- **reasoning_effort=none 필수(검토 아님, 요구사항)**: analysishandler/run.go:89-95는 Gemini 2.5의 thinking을
  끄기 위해 reasoning_effort="none"을 명시하고, 이유를 코드 주석으로 남겼다("thinking이 max_tokens 예산을
  내부 추론에 소진해 JSON이 finish_reason=length로 잘림 → unmarshal 실패"). Summary도 동일 Gemini OpenAI
  호환 엔드포인트로 전환되는데 현재 content.go의 요약 생성 요청(221-)/언어검증 요청(343-)은 ReasoningEffort를
  전혀 설정하지 않는다. 선례상 gemini-3.8-flash에서 summary 출력이 잘릴 위험이 사실상 확정적이므로, **Summary
  생성 요청에 reasoning_effort=none을 반드시 설정**한다(go-openai chat.go:311 ReasoningEffort 필드 존재).
  언어검증은 출력이 짧아(yes/no) 잘림 위험이 낮으나 일관성 위해 함께 적용 검토.
- Gemini는 reasoning_effort/thinking 옵션이 있음(analysis는 "none"으로 thinking 비활성). Summary도
  thinking 비활성이 맞음(위 근거).
- 언어검증(defaultVerifyModel=gpt-4o-mini)도 Gemini로. yes/no 판정이라 저위험.

### A-thinking. Audit/Proposal의 reasoning_effort 검토 (A에 포함)
Audit(geminiaudithandler)/Proposal(geminiproposalhandler)은 analysis와 달리 reasoning_effort=none을
설정하지 않는다(audit main.go:247 요청에 ReasoningEffort 없음). gemini-2.5에서는 동작했으나 gemini-3.8-flash
로 올릴 때 thinking 기본 동작이 달라 JSON 출력이 잘릴 수 있다. 특히 Proposal은 pro→flash로 모델 계열이
바뀌어 JSON schema(ParseProposalResponse) 파싱 위험이 가장 크다. **설계에서: (a) audit/proposal에도
reasoning_effort=none 적용 필요 여부, (b) proposal flash 출력이 스키마를 통과하는지 실호출 스모크**를 검증
항목으로 명시한다.

## 5. 진행 타당성

- A: 유효+긴급(데드라인). 상수 교체라 리스크 낮음. 즉시 진행 타당.
- B: 유효. OpenAI→Gemini 전환은 엔진 wiring 변경이라 A보다 크나, 인프라(WithConfig)는 이미 있음.
  temperature/thinking 상호작용만 설계에서 정리하면 실현 가능.
- C: 범위 외(PR2). tool/streaming 검증 전 전환 불가.

## 6. 리스크

- (R1) Gemini OpenAI 호환 엔드포인트가 summary의 [system,user] 2메시지 + temperature를 OpenAI와
  동일하게 처리하는지. → 설계에서 확인, 필요 시 실호출 스모크.
- (R2) 모델 교체로 요약/audit/proposal 출력 형식(JSON schema strictness 등)이 달라질 수 있음. audit/proposal도
  실코드상 Gemini OpenAI 호환이나 **reasoning_effort=none 미설정**이라 gemini-3.8-flash에서 thinking이
  JSON을 잘라먹을 위험이 있음(analysis 선례). proposal은 pro→flash로 계열이 바뀌어 스키마 파싱 위험이 가장
  큼(대표 품질저하 수용, 단 스키마 통과는 실호출 스모크로 확인 필요).
- (R3) config 기본값 변경이 배포 시 env override와 충돌하지 않는지. 프로덕션 env 확인은 대표 몫(배포).
- (R4) VOIP-1536 temperature와의 정합: B에서 모델만 바뀌고 temperature 로직은 유지되어야 함.
- (R5) reasoning_effort 미설정 시 summary/audit/proposal 출력 잘림(finish_reason=length). analysis 선례로
  확정적 위험. → summary는 reasoning_effort=none 필수, audit/proposal도 설계에서 필요성 확정.
- (R6) 데드라인-지연 결합 리스크: A는 2026-10-20 shutdown 하드 데드라인이 있는 저위험 상수교체이고, B는
  열린 설계질문(B-1 vs B-2, temperature/thinking 스모크)이 남은 중난이도 작업이다. A+B 단일 PR(대표 확정)
  에서 **B가 설계/코드 리뷰 루프에서 장기 지연되면 A의 shutdown 회피도 함께 늦어진다.** 폴백: B가 리뷰
  루프에서 막혀 10월 초까지 수렴 못 하면 A를 먼저 분리 출시(shutdown 회피 우선)하고 B는 후속 PR로 돌린다.
  이 폴백 발동 조건(예: 10-05까지 B 코드리뷰 2연속 Approve 미달)을 설계 착수 시 명시한다.

## 7. 열린 질문 (설계에서 해소)

- B 방법: B-1(wiring 직접 주입) vs B-2(config provider 선택). VOIP-1197 패턴 재사용 범위.
- gemini-3.8-flash를 상수로 박을지, config로 뺄지. **단, audit/proposal의 config화는 이번 PR 범위에서
  제외(defer) 권고**: 데드라인이 걸린 PR에서 audit/proposal까지 config 체계로 바꾸면 스코프 확대·지연
  위험. 이번엔 상수만 gemini-3.8-flash로 교체(현행 상수 패턴 유지), config화 통일(단일 write-through SoT)은
  후속 티켓. Summary(B)는 analysis처럼 이미 config 인프라가 있어 재사용 가능하므로 별개 판단.
- EngineModel 레지스트리(models/ai/main.go)에 3.8-flash 추가 필요 여부(aicall PR2와 관련).
- Summary/audit/proposal에 reasoning_effort=none 적용 범위 확정(§4 A-thinking/B, §6 R5 참조).
- 단일 모델 상수 SoT: 여러 곳에 흩어진 "gemini-3.8-flash"를 한 곳에서 관리할지(단일 write-through 원칙,
  단 위 audit/proposal config화 defer와 균형 — 과설계 지양).

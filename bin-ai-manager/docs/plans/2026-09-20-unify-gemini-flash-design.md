# 설계: 기본 LLM 모델을 최신 Gemini flash로 통일 (VOIP-1537, PR1=A+B)

작성일: 2026-09-20
작성자: Hermes (CPO)
브랜치: VOIP-1537-unify-gemini-flash
상태: 설계 (리뷰 대기)
선행: 이슈 분석 `2026-09-20-unify-gemini-flash-analysis.md` (리뷰 4회, 2연속 Approve 확정)

## 1. 목표

엔진이 사용자별로 별도 지정되지 않은 모든 LLM 기능의 기본 모델을 `gemini-3.8-flash`로 통일한다.
이 PR(=A+B): Analysis/Audit/Proposal 최신화(A) + Summary OpenAI→Gemini 전환(B). C(AI Call 폴백)는 PR2.

## 2. 확정 결정 (이슈 분석 열린 질문 해소)

| 열린 질문 | 결정 | 근거 |
|---|---|---|
| B 방법 (B-1 직접주입 vs B-2 config) | **B-2: config 기반, analysis 패턴(VOIP-1197) 재사용** | summaryHandler가 생성자 주입 구조라 main.go wiring만 바꾸면 됨. env 롤백 가능. |
| audit/proposal config화 | **defer(이번 제외)**: 상수만 교체 | 데드라인 PR 스코프 확대·지연 방지(이슈분석 §7). |
| reasoning_effort 범위 | summary 생성/검증 **필수 적용**, audit/proposal도 **적용**(스모크 검증) | thinking JSON 잘림 위험(analysis 선례). |
| EngineModel 레지스트리 3.8-flash 추가 | 이번 제외(PR2 aicall에서) | 이번 PR은 aicall 미변경. |
| 단일 SoT | 부분 적용: summary는 config, audit/proposal은 상수 | 과설계 지양. 완전 통일은 후속. |

## 3. 구현 상세

### A. 상수/설정 교체 (shutdown 회피)

**A-1. Audit** (`geminiaudithandler/main.go:18`)
```go
geminiModel = "gemini-3.8-flash"   // was gemini-2.5-flash
```
+ reasoning_effort=none 적용: Evaluate 요청(main.go:246-259 ChatCompletionRequest)에 `ReasoningEffort: "none"`
추가. thinking으로 JSON schema 응답이 잘리는 것 방지.

**A-2. Proposal** (`geminiproposalhandler/main.go:19`)
```go
geminiModel = "gemini-3.8-flash"   // was gemini-2.5-pro (pro→flash, 대표 확정)
```
+ reasoning_effort=none 적용: Evaluate 요청(main.go:164-)에 `ReasoningEffort: "none"` 추가. proposal은
pro→flash로 계열이 바뀌어 스키마(ParseProposalResponse) 위험이 가장 크므로 필수.

**A-3. Analysis config 기본값** (`internal/config/main.go:135-136`)
```go
f.String("analysis_default_model", "gemini-3.8-flash", ...)   // was gemini-2.5-flash
f.String("analysis_allowed_models", "gemini-3.8-flash", ...)  // was gemini-2.5-flash,gemini-2.5-pro
```
Analysis는 이미 reasoning_effort=none이 config 기본값이라 추가 작업 없음.

### B. Summary 엔진 전환 (OpenAI → Gemini)

**B-1. config 신설** (`internal/config/main.go`) — analysis 3종과 대칭:
```go
SummaryEngineBaseURL   string  // default: https://generativelanguage.googleapis.com/v1beta/openai/
SummaryModel           string  // default: gemini-3.8-flash
SummaryReasoningEffort string  // default: none
```
flag/env 매핑도 analysis와 동일 패턴으로 추가(summary_engine_base_url / SUMMARY_ENGINE_BASE_URL 등).
OpenAI 롤백은 base URL을 비우고 SummaryModel을 gpt-4-turbo로 두면 됨(env-driven).

**B-2. main.go wiring** (`cmd/ai-manager/main.go`) — analysis 패턴 복제:
```go
summaryKey := cfg.EngineKeyChatGPT // OpenAI default (rollback)
if strings.Contains(cfg.SummaryEngineBaseURL, "generativelanguage") {
    summaryKey = cfg.GoogleAPIKey
}
summaryEngine := engine_openai_handler.NewEngineOpenaiHandlerWithConfig(summaryKey, cfg.SummaryEngineBaseURL)
summaryHandler := summaryhandler.NewSummaryHandler(requestHandler, notifyHandler, db, summaryEngine, <model/reasoning>)
```
현재 summaryHandler는 OpenAI 순정 engineOpenaiHandler를 공유받는데, 전용 summaryEngine으로 분리한다.
대화용 engineOpenaiHandler(OpenAI)는 불변(PR2까지 유지).

**B-3. summaryhandler 모델/reasoning 파라미터화** (`summaryhandler/main.go`, `content.go`)
- **파급원 정정(상수→필드)**: 프로덕션 호출부는 main.go 1곳뿐이라 생성자 시그니처 변경 자체는 안전하나,
  실제 파급은 `defaultModel`/`defaultVerifyModel` **패키지 상수**를 테스트가 직접 참조하는 데서 온다
  (content_test.go:254/311/343). 이 상수를 제거하고 summaryHandler의 주입 필드(model/reasoning)로 전환하면,
  content_test.go의 exact-match fixture와 genDo/verifyDo가 주입 모델 기대값을 쓰도록 갱신해야 한다.
- content.go 생성 요청(222)/검증 요청(345)에 `ReasoningEffort: <injected>` 추가. summaryTemperature(0.2)는
  유지(VOIP-1536).
- **검증 모델 통일은 '선호'가 아니라 엔진 공유 구조상 강제**: content.go:355 검증은 생성과 동일한
  `h.engineOpenaiHandler`(=주입된 summaryEngine)로 `SendOnce`를 호출한다. Gemini 엔진을 주입하면 gpt-4o-mini는
  Gemini 엔드포인트에서 호출 불가하므로, 검증 모델도 필연적으로 주입 모델(gemini-3.8-flash)이 된다.
  즉 defaultVerifyModel(gpt-4o-mini) 상수는 제거되고 검증도 생성과 같은 모델을 쓴다.
- **롤백 시 비용 특성 변화(명시)**: OpenAI 롤백(SummaryModel=gpt-4-turbo, base URL 비움) 시 검증 요청도
  gpt-4-turbo로 실행되어, 원래의 저가 검증 경로(gpt-4o-mini)는 복원되지 않는다. 이는 엔진 공유 구조의
  결과로 **의도적으로 수용**한다(별도 SummaryVerifyModel config를 두는 것은 과설계 — 롤백은 비상 경로이고
  검증 요청은 짧아 gpt-4-turbo 비용 증가분이 작음). 롤백이 "생성 모델만 복원"임을 문서에 명시.

### 최소 개입 원칙
- gemini-3.8-flash 모델명은 A(audit/proposal 상수) + config 기본값(analysis/summary)에 등장. summary/analysis는
  config, audit/proposal은 상수(defer). 완전 단일 SoT는 후속 티켓.
- reasoning_effort=none은 4곳(summary 생성/검증, audit, proposal)에 적용. analysis는 기존 유지.
  - summary/analysis: config 주입값(non-empty일 때만 세팅, run.go:94-96 패턴) — OpenAI 롤백 경로가 있어
    조건부 세팅이 맞음.
  - audit/proposal: 리터럴 "none" — 해당 핸들러는 Gemini 전용(하드코딩 client)이라 OpenAI 롤백 설계 자체가
    없으므로 리터럴이 안전하고 단순.

## 4. 테스트 전략

- **A-1/A-2 (audit/proposal) — client seam 없음, 요청 필드 단위 테스트 불가**: geminiaudithandler/
  geminiproposalhandler는 unexported 필드 `client *openai.Client`를 New 생성자 내부에서 직접 만들고
  (main.go:81/68), Evaluate가 `h.client.CreateChatCompletion`을 곧바로 호출한다. 인터페이스 seam도 mock도
  없어(기존 테스트는 Sanitize/Parse*/BuildPrompt 등 순수 함수만 검증) ChatCompletionRequest.Model/
  ReasoningEffort를 gomock/Do 매처로 assert할 수 없다. **결정: audit/proposal의 모델 상수 교체 +
  ReasoningEffort="none" 추가는 `go build`/`go vet` + 대표 배포 카나리 스모크(S2/S3)로만 커버한다**(요청 필드
  단위 테스트·mutation FAIL 주장은 폐기). client 주입 seam 신설은 이번 스코프 밖(테스트 가능성 개선은 별도
  리팩터 티켓 — 데드라인 PR에 seam 리팩터를 얹는 것은 스코프 확대). 단 순수 함수 테스트(Parse*)는 그대로
  green 유지 확인.
- **A-3 (config)**: config 기본값 테스트가 있으면 gemini-3.8-flash 기대값으로 갱신. 없으면 build로 갈음.
- **B (summary) — 단위 테스트 가능(엔진 mock 있음)**: content_test.go는 mockOpenai로 요청을 캡처한다.
  content.go 요청 빌드 시 ReasoningEffort="none"이 실리는지 assert(VOIP-1536 fixture 패턴 확장, 리터럴
  "none"). temperature 0.2 assert 유지. defaultModel/defaultVerifyModel 상수 제거에 따라 fixture(254)와
  genDo/verifyDo(311/343)를 주입 모델 기대값으로 갱신. mutation: ReasoningEffort 제거/모델 오류 시 FAIL.
- **wiring(main.go)**: 순수 배선이라 단위 테스트 어려움. build + 기존 통합 테스트 green으로 갈음.
- 전 서비스 `go build ./... && go vet ./pkg/... && gofmt -l && go test ./pkg/...` green.

## 4-a. MaxTokens(출력 예산) 가드 결정

analysis는 Gemini 이전 시 `MaxTokens: h.maxOutputToks`(run.go:65, config 기본 16384) 런어웨이 가드를
명시 추가했다. summary는 현재 content.go 요청에 MaxTokens가 없다.
**결정: summary에 MaxTokens 가드 불요**(추가 안 함). 근거: (1) summary는 고정 구조 프롬프트 + transcript
입력이라 analysis의 대용량 staged 입력과 달리 출력 폭주 위험이 낮고, (2) reasoning_effort=none으로 thinking
토큰 소진(주 위험)은 이미 차단되며, (3) summary는 자유 텍스트라 JSON 잘림 실패모드가 없다. audit/proposal은
JSON schema 응답이나 기존에도 MaxTokens 미설정으로 gemini-2.5에서 동작했으므로 이번에 추가하지 않는다
(변경 최소화). 이는 침묵이 아니라 명시적 "가드 불요" 결정이다. 향후 출력 폭주 실측 시 후속으로 추가.

## 5. 실호출 스모크 (설계 검증, R1/R2/R5 요구) — 카나리 게이트

실제 Gemini OpenAI 호환 엔드포인트로 검증할 항목:
- (S1) summary 생성이 gemini-3.8-flash + reasoning_effort=none + temperature 0.2로 정상 요약 반환.
- (S2, **게이트**) proposal flash 출력이 JSON schema(ParseProposalResponse)를 통과(Strict:false). **최대
  리스크(pro→flash 계열 변경)이므로 프로덕션 전면 배포 전 반드시 통과해야 하는 게이팅 스모크(카나리).
  S2 미통과 시 proposal의 flash 전환은 롤백/보류하고 전면 배포 금지.**
- (S3, 게이트) audit flash 출력이 스키마 통과. 미통과 시 audit 전환 보류.
주: 스모크는 프로덕션 GOOGLE_API_KEY가 필요하므로 AI가 직접 실행 못 할 수 있다. 그 경우 코드 정합성(필드
세팅) 검증까지만 하고, **S2/S3 카나리 게이트 실행은 대표 배포 시점의 필수 선행 조건으로 명시 위임**한다
(단순 "배포 검증 위임"이 아니라, 통과 전 전면 배포 금지라는 게이트 성격을 명확히 함). S1은 저위험(자유
텍스트)이라 게이트 아님.

## 6. 리스크 (이슈분석 §6 계승)

- reasoning_effort=none 미적용 시 JSON 잘림 → 4곳 모두 적용으로 해소.
- proposal pro→flash 품질저하(대표 수용) + 스키마 위험 → S2 카나리 게이트로 전면 배포 전 차단.
- 배포 시 env override 충돌 → 대표 배포 검증.
- B 지연 시 A 선분리 폴백(이슈분석 R6, 트리거 10-05).
- OpenAI 롤백 시 검증 모델이 gpt-4o-mini→gpt-4-turbo로 바뀜(비용 소폭 증가, 의도 수용, §3 B-3).

## 7. 문서/파급

- RST/OpenAPI: 내부 LLM 파라미터·config라 API 스키마 변경 아님. 문서 갱신 불요.
- **config 신규 env 문서화(결정)**: summary_engine_base_url / summary_model / summary_reasoning_effort는
  OpenAI 롤백 안전성이 의존하는 운영자 스위치다. 기존 analysis env도 배포 매니페스트에 문서화가 없는 공백이
  있으나, 이번 PR에서 최소한 **bin-ai-manager 배포 설정 파일(예: helm values / compose env 목록)에 신규
  summary_* env를 기본값과 함께 추가**하여 롤백 스위치가 운영자에게 노출되게 한다. analysis env의 소급
  문서화까지는 이번 스코프 밖(별도). 구현 시 실제 배포 config 파일 위치를 확인해 반영.
- gen 파일 영향 없음.

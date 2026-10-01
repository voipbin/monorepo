# Conversational Assistant Builder 구현 계획 (VOIP-1558, v9)

> 설계: `2026-10-01-conversational-assistant-builder-design.md` (v8, 디자인 리뷰 루프 종료). 이 계획은 그 설계를 작업으로 옮긴다. **아래 "설계와 다른 점" 절의 의도적 이탈을 제외하고 설계가 이긴다.** F번호, 규칙 번호, 절 번호는 설계 문서를 따른다. 설계 내용의 재진술을 피하기 위해 각 작업은 설계 절을 가리키고 테스트 목록과 설계에 없는 결정만 적는다.

**목표:** square-admin의 AI 생성 화면에서 AI와 대화해 이름, 설명, 프롬프트, 도구가 채워지는 상태 없는 동기 멀티턴 Builder의 백엔드를 만든다.

**구조:** 브라우저가 이력을 보관하고 매 턴 전체를 `POST /ai_builder/chat`로 보낸다. api-manager가 검증하고 ai-manager 전용 RPC를 호출한다. ai-manager `builderhandler`가 고정 시스템 프롬프트와 이력으로 LLM을 한 번 호출하고 코드로 응답을 검증한다. 서버는 대화를 저장하지 않는다.

**범위(monorepo 한 PR):** ai-manager, common-handler, api-manager, openapi, 문서, 평가 하네스. **프런트(square-admin)는 별도 저장소의 별도 PR이며 이 계획의 범위 밖이다**(설계 7절).

**"영향 없음"의 정확한 범위:** `ai_builder_enabled=false`(기본)이면 Builder 기능은 동작하지 않는다. 그러나 `bin-common-handler`의 `consume.go` 에러 문자열 변경(T9)과 requesthandler 인터페이스 추가는 **킬 스위치와 무관하게 모든 서비스에 적용되는 변경**이다(에러 문자열에서 본문만 빠지며 동작은 같다).

## 설계와 다른 점(의도적 이탈, 대표님 확인 대상)
1. **평가 게이트 이탈(핵심).** **갱신(2026-10-02): 아래 문단은 계획 작성 시점의 사실이다. 이후 키를 확보해 평가를 두 번 실행했고(36회와 합성 5건씩, 프롬프트 개정 1과 2), 판정은 모두 AI 검토자가 했으며 사람 판정은 없다. 실행 1과 실행 2 모두 코드의 게이트 계산(`Evaluate`)으로 두 AI 판정자 모두 합격선 미달이다(실행 2는 AI 판정 A 36/41, B 34/41이지만 s1, s2a, s2b, s13, s15 등이 그룹 최소 통과 수에 못 미친다). 수치와 한계는 `bin-ai-manager/pkg/builderhandler/eval/README.md`의 "Runs side by side"에 있다. 대표님의 게이트 이탈 승인(서버를 한 PR로, 기본 꺼짐으로 머지)은 유지된다.** 환경 확인(2026-10-01): LLM API 키가 없어 설계 7절 1단계의 평가를 **실행할 수 없다.** 따라서 "평가를 먼저 하고 통과하지 못하면 이후 단계로 가지 않는다"에서 이탈한다. 하네스, 프롬프트 초안, 파서는 만들고 단위 테스트까지 통과시키되 **적응성 평가는 실행하지 않는다.** 코드의 `reasoning_effort`, `max_tokens`, 세마포어, LLM deadline은 모두 **미실측 초기값**이며 설정으로 바꾼다. 서버 전체를 한 PR로 `ai_builder_enabled=false`로 머지하면(어두운 머지) 킬 스위치가 막아 주지만 평가 통과를 코드로 강제하지는 않는다. 분리(하네스, 프롬프트, 파서만 먼저 머지)는 기술적으로 가능하나 PR 분할이라 별도 승인이 필요하다. 한 PR 추천의 이유는 PR 수 최소화와 롤백 용이성이며 설계 7절의 근거("인프라를 만든 뒤 프롬프트가 안 통하면 낭비")는 이 계획이 뒤집는다(평가 실패 시 죽은 코드가 남고 설계 11절의 "접거나 템플릿 보강"과 연결된다).
2. **RST 문서 보류.** 루트와 `bin-api-manager`의 `CLAUDE.md`는 엔드포인트 추가 시 RST를 반드시 갱신하라고 한다. 이 계획은 결정 전 고지 문구가 공개되지 않도록 RST와 `docsdev/build`를 "외부 LLM 전송 고지와 약관 결정"(설계 11절) 이후로 미루는 기본안을 둔다. **저장소 필수 규칙에서의 이탈이며 승인 전에는 RST를 포함하는 쪽이 대안이다**(열린 질문 4).
3. **reason 세분화 없음.** 설계 4.7에 맞춰 입력 길이 위반은 `INVALID_ARGUMENT`, 본문 바이트 초과(`MaxBytesReader`)만 `BUILDER_INPUT_TOO_LARGE`로 한다.
4. **평가 예산 정정.** 설계 2.5와 11절의 "약 38회, 빌더 호출 약 225"는 산술 오류다. 실제는 15+4+3+3+4+7 = **36회, 빌더 호출 약 216**(36회 x 평균 6턴이라는 **가정치**이며 실측값이 아니다)이다. 설계의 "훑어보기 약 15건"도 시나리오 목록(1, 2a, 7, 8, 10, 11, 12, 14)으로 세면 **8건**이고 시나리오 15의 합성 이력 5건은 별도다. 이 정정을 설계 문서의 정오 사항으로 기록하고 대표님의 예산 승인 때 정정값을 올린다.
5. **provider 오류 매핑과 일일 횟수 차감(설계 4.7과 4.3에 provider 오류 정책이 없음).** LLM provider 오류(인증, 한도, 5xx)를 설계 4.7의 기존 행에 맞춰야 한다. 비교: (가) `BUILDER_UNAVAILABLE`(키 누락과 Redis 오류의 행)은 일시적 429나 5xx에도 "사용할 수 없습니다"가 영구 불가로 읽히는 문제가 있다. (나) `BUILDER_RESPONSE_INVALID`는 프런트가 "다시 시도"를 안내하고 일시 오류에 맞다. (다) 새 reason(`BUILDER_LLM_ERROR`)은 설계 4.7 표에 행을 추가하는 설계 변경이다. **(나)를 택한다.** **일일 횟수 차감에 대한 사실:** 설계 4.3은 호출 순서(세마포어, `BuilderChatCountIncr`, LLM)와 `BUILDER_RESPONSE_INVALID` 카운트("플랫폼이 이미 비용을 지불했으므로")만 정하고 provider 오류의 차감 여부는 정하지 않았다. 호출 순서상 카운트는 LLM 호출 전에 일어나므로 **이 계획의 구현에서는 provider 오류(`ErrLLM`)도 일일 횟수를 차감한다**(T7b 테스트가 이를 단언한다). 인증 오류나 장애에서 고객이 재시도하면 한도(200회)가 소진되는 비용이 있으며, 키 누락은 `status`의 `available=false`가 선행 방어한다. 일시 오류에 횟수를 환급하는 `Decr`를 두면 설계 4.3의 갱신이 필요하므로 **대표님 결정 항목**으로 올린다(열린 질문 6). 운영자 조치가 필요한 키 오류와 일시 오류의 구분은 메트릭(`llm_error`)과 로그의 분류 코드로 한다. 대표님이 (다)를 원하시면 설계 4.7을 한 줄 갱신한다.

6. **구현에서 계획과 달라진 점(서버 구현 중 기록, 동작 변경 없음 또는 아래 이유).**
   - `NewBuilderHandler`의 마지막 인자는 `keyConfigured bool`이 아니라 `Options{Enabled, KeyConfigured, DailyLimit, MaxConcurrent}` 구조체이다. 켜기 스위치, 일일 한도, 동시 호출 수까지 넘겨야 해서 묶었다.
   - 메트릭 `result` 라벨이 설계 4.8의 7개에 더해 `invalid_argument`(검증 실패, 고객 실수라 `llm_error`와 `unavailable`의 비율에 섞지 않으려는 것)와 `internal`(listenhandler가 복구한 panic)을 갖는다.
   - 기동 시 설정 검증(`validateBuilderConfig`)을 추가했다. 켜져 있을 때만 양수 검사, LLM 시간 제한 0 이하 거부(이월 항목), 55초 RPC 대기에서 5초 여유를 뺀 50초 초과 거부, 모델 비어 있음 거부.
   - `ai_builder_model`의 기본값은 `analysis_default_model`을 따르지 않고 같은 값(`gemini-3.8-flash`)을 따로 둔다. 연동하지 않은 이유는 Builder만 모델을 바꿀 수 있어야 하기 때문이다.
   - 프롬프트의 규칙은 설계 2.3과 계획 T4의 6개가 아니라 7개이다. 규칙 7("사용자가 말한 것과 빌더가 제안한 것을 구분한다")은 평가 실행 1 이후 프롬프트 개정 2에서 추가되었다.
   - 요청의 `current_draft.tool_names`는 서버가 개수(6 이하)만 검사하고 이름은 거르지 않는다. 설계 4.2가 폼 수정을 되돌려 보내지 않는 것을 수용했기 때문이며, 응답 쪽은 파서가 허용 목록으로 거른다.
   - OpenAPI 스키마 이름은 저장소 관례(`AIManagerAI...`)에 맞춰 `AIManagerAIBuilder*`이다.
   - 공급자 인증 오류(`auth`)는 플랫폼 키 문제라 Error 레벨로 로깅한다. 그 밖의 공급자 오류는 Info이다.
   - 기동 시 경고 로그(설계 4.6)는 키 없음과 모델 접두 불일치 두 가지를 구현했다. 모델 접두 비교는 힌트일 뿐 규칙이 아니다.
   - `docs/plans/...design-v7-aicall-superseded.md`는 폐기된 설계의 기록으로 남긴 것이며 코드와 무관하다.

**승인 게이트(침묵을 승인으로 보지 않는다, 허용 목록 방식):** 승인 전에 커밋을 허용하는 작업은 **T0, T1~T4, T7a, T11의 순수 로직과 dry-run**뿐이다. **그 밖의 모든 작업(T5, T6, T7b, T8, T9, T10, T12, T13의 서버 관련 검증)은 대표님의 명시적 승인 전까지 커밋하지 않으며 PR도 승인 후에 연다.** "좋아, 구현 PR까지 진행해"는 구현과 PR 진행의 지시이고 위 1번의 평가 이탈까지 승인한 것인지는 별개이므로 대표님께 한 번 확인한다.

## 활성화 전 필수 조건(설계 11절의 결정과 8절의 수동 체크리스트를 따르고 이 계획에서 추가된 것은 추가분 1, 2다)
설계 11절의 결정(평가 예산과 통과 기준, 전역 일일 총량 상한, 호스팅 활성화 범위, 외부 LLM 전송 고지와 약관), 설계 8절의 수동 체크리스트(골든 문구 파일, 헤더 골격, 허용 도구 6종과 `TOOL_LABELS`), 설계 2.5의 평가 통과와 JSON 불량 5% 판단, 활성화 후 smoke 대화 1회에 더해 다음을 PR 본문과 docs에 같은 목록으로 둔다.
- **추가분 1: ai-manager RPC 워커 점유와 breaker 오염 영향 측정**(설계 4.3, F19): Builder 호출은 최대 LLM deadline 동안 공용 RPC 워커(프로세스당 10개, 레플리카 2개)를 점유한다. 세마포어(설계 값 3)는 Builder끼리만 제한한다. 활성화 전에 (가) aicall과 tool RPC 지연에 미치는 영향, (나) 큐 대기 + LLM deadline이 RPC timeout(55초)을 넘는 경우(api-manager는 먼저 타임아웃되지만 ai-manager는 계속 처리하고 카운트한다), (다) **Builder RPC timeout이 `requesthandler`의 큐 단위 circuit breaker(연속 5회 실패에 30초 open)를 열어 같은 큐의 다른 AI RPC를 `ErrCircuitOpen`으로 막을 위험**을 측정해 세마포어를 확정한다. 별도 큐는 설계 10절 트리거(api-manager의 Builder timeout 카운터 또는 `ErrCircuitOpen` 관측) 전까지 하지 않는다.
- **추가분 2: T0의 "미확인" 항목**(LB와 ingress timeout 65초 이상, Loki 수집 소스, 트레이스 유무)과 T0에서 확인된 본문 로깅, 트레이스 본문, rate limit 부족 이슈.

## 작업 방식
작업(T)마다 TDD(실패하는 테스트, 구현, 통과)로 진행하고 **작업 단위로 커밋하되 PR은 하나**다(squash 머지 전제). 각 작업 끝에서 해당 모듈의 `go test ./...`를 돌린다. 인터페이스를 바꾸는 작업 뒤에는 mock을 재생성한다. 로컬 검증용 `go mod vendor`는 쓰되 **vendor는 커밋하지 않는다**(저장소 정책, 루트 `CLAUDE.md`). **common-handler를 바꾼 뒤에는 소비 모듈(ai-manager, api-manager)에서 `go mod vendor`를 다시 실행**해야 이후 작업의 `go test`가 최신 common-handler를 쓴다. 핸들러 입력은 CLAUDE.md의 관례(풀어쓴 파라미터 또는 `models/` 도메인 타입)에서 도메인 타입 `builder.ChatRequest`를 쓰는 것이 허용 범위에 든다고 해석한다. 구현 전에 각 모듈의 `CLAUDE.md`와 `docs/`의 해당 절을 읽는다.

## 작업 순서와 의존
**문서 순서는 의존 순서이며 승인 전 실행 순서가 아니다.** 승인 전에는 T0, T1~T4, T7a, T11(순수 로직과 dry-run)만 이 순서로 커밋한다. T5, T6, T7b, T8 이후는 승인 후에 문서 순서대로 한다(T5는 T7a에 의존하지 않는다. T6은 실제 작업이 플래그와 환경변수 매핑뿐이라 코드 의존은 없지만 `Config` 필드 이름을 T7a에서 가져오므로 T7a 뒤에 배치한다. 둘 다 승인 후 T7a 뒤에 붙는 커밋이 된다). 구현자는 T5부터 순서대로 커밋하지 않는다.


### T0. 구현 확인(설계 7절 2단계)
코드로 확인하고 **결과는 PR 본문과 `docs/`에 기록한다(설계 문서를 다시 열지 않는다).** 코드로 확인할 수 없는 항목은 "미확인, 활성화 전 선결"로 둔다. 결과에 따른 분기는 다음과 같다.
- `ai_ais.init_prompt` 컬럼(`bin-dbscheme-manager`의 마이그레이션과 ai-manager `models/ai`의 DB 태그, `ai_ais`는 이름이 바뀐 이력이 있어 `chatbots` 테이블 정의까지 추적). **`text`이면 한도는 글자 수가 아니라 바이트(약 65535)이며 8000 rune은 한국어 3바이트로 약 24000바이트라 들어간다고 계산해 기록한다.** 8000자를 담지 못하면 T1의 상한을 낮춘다.
- 트레이스(otel, sentry) 사용 여부: api-manager와 ai-manager grep. 사용 중이면 요청 본문이 스팬에 실리는지 확인하고 실리면 "활성화 전 필수 조건"의 추가분 2에 추가.
- api-manager 인증 이후 미들웨어와 에러 미들웨어(`lib/middleware`, `server/error.go`)의 본문 로깅. **본문 로깅이 확인되면 별도 이슈로 올리고 "활성화 전 필수 조건"의 추가분 2에 추가한다.**
- `CustomerRateLimit` tier 값(`lib/middleware/customer_ratelimit.go`)과 LLM 호출 비용에 충분한지. 부족하면 "활성화 전 필수 조건"의 추가분 2에 추가.
- 저장소의 알림 규칙(Prometheus rules, Loki 알림, 대시보드 YAML과 JSON)이 `consume.go`의 에러 문자열(`could not parse the message`, `could not reply the message`, `could not marshal the response`)을 매칭하는지 grep(이 계획 작성 시 저장소 안에서는 0건이었다. 저장소 밖 인프라는 미확인으로 둔다).
- Loki 수집 소스와 LB, ingress timeout은 저장소 밖일 가능성이 높아 "미확인"으로 기록.

### T1. 모델, 공유 검증, 오류 reason (ai-manager `models/builder`)
- Create: `bin-ai-manager/models/builder/main.go`, `validate.go`, `main_test.go`
- 타입: `Message{Role, Content}`, `Draft{Name, Detail, InitPrompt, ToolNames}`, `ChatRequest{Messages, CurrentDraft}`, `ChatResponse{Message, Draft, Assumptions, DraftWarnings}`, `StatusResponse{Available, MaxMessages, MaxMessageChars}`(설계 4.6).
- 상수: 상한(메시지 40, 메시지 2000 rune, 합산 40000 rune, `init_prompt` 8000, `name` 100, `detail` 500), reason(`BUILDER_DISABLED`, `BUILDER_UNAVAILABLE`, `BUILDER_DAILY_LIMIT`, `BUILDER_BUSY`, `BUILDER_RESPONSE_INVALID`, `BUILDER_TIMEOUT`, `BUILDER_INPUT_TOO_LARGE`(본문 바이트 초과 전용), `INVALID_ARGUMENT`), 허용 도구 `AllowedTools`(connect_call, stop_service, send_email, send_message, set_variables, case_create).
- **`ValidateRequest(req *ChatRequest) error`를 이 모델 패키지에 둔다.** api-manager와 ai-manager가 같은 함수를 쓰게 하는 이유: 설계 4.3이 양쪽 검증을 요구하고 api-manager는 ai-manager 핸들러 패키지가 아니라 `models`만 import하는 것이 기존 관례다. 검증 규칙: 메시지 수와 길이와 합산 rune, `current_draft` 상한, 마지막 role `user`, role은 `user`와 `assistant`만(교대 강제 없음), 빈 messages 거부. 실패는 `cerrors.InvalidArgument` reason `INVALID_ARGUMENT`이다(설계 4.7. `BUILDER_INPUT_TOO_LARGE`는 api-manager의 `MaxBytesReader` 초과에만 쓴다).
- 내부 URI 상수(`URIChat = "/v1/ai_builder/chat"`, `URIStatus = "/v1/ai_builder/status"`)를 이 패키지에 두고 listenhandler와 requesthandler가 공유한다(설계 4.1).
- 테스트: JSON 왕복, 허용 목록 6종이 `tool.AllToolNames`에 모두 존재, 경계값(40/41, 2000/2001 rune, 합산 40000, 한국어 3바이트와 rune 계산), 역할 오류, 마지막이 assistant.

### T2. 응답 파서와 검증기(순수 함수, 설계 4.4)
- Create: `bin-ai-manager/pkg/builderhandler/parse.go`, `parse_test.go`
- `Parse(raw string) (*ParsedResponse, error)`: 각 `{`(최대 5개)에서 `json.Decoder`로 `map[string]json.RawMessage`를 디코드하고 `message`가 비어 있지 않은 문자열인 첫 객체를 채택. 필드별 디코드와 타입 불일치 필드만 버림(중첩 `draft.tool_names`의 타입 불일치는 그 필드만), `draft`는 `name`과 `init_prompt`가 비어 있지 않을 때만 채택(아니면 `draft`와 `assumptions`를 함께 폐기하고 경고), `draft`가 있고 `assumptions`가 없으면 빈 배열, 도구 교집합과 제거분 경고, `## Tools & Capabilities` 섹션 제거(대소문자와 공백 무시, `##`와 `###`, 중복, 문서 끝까지)와 제거 후 `init_prompt` 재검사, 길이 절단(rune), 허용 목록 밖 도구명이 단어 경계로 `init_prompt`에 있으면 경고. 검사 순서: 필드 디코드, 필수 검사, 섹션 제거, 재검사, 길이 절단.
- 테스트(설계 8절): **`## Tools & Capabilities` 변형 제거(대소문자, `###`, 중복, 문서 끝), 도구 교집합과 `draft_warnings` 기록, 빈 `message`인 선행 객체 뒤에 유효 객체가 있는 케이스**, 정상, 코드펜스, 앞뒤 텍스트, 설명 텍스트에 `{}`와 유효한 다른 JSON, 비JSON, 빈 message, 타입 불일치, 중첩 `tool_names` 타입 불일치, 알 수 없는 필드, `{` 6개 이상에서 5개 제한 경계, `draft` 필수 누락, 섹션 제거 후 빈 `init_prompt`, `draft` 폐기 시 `assumptions` 동반 폐기, `assumptions` 누락 시 빈 배열, `stop_service`와 `stop_flow` 단어 경계, 한국어 rune 절단. (잘림은 `Parse` 입력이 아니라 응답의 finish reason이므로 T7b에서 `BUILDER_RESPONSE_INVALID`로 매핑하는 테스트로 둔다.)

### T3. 입력 조립(설계 4.2)
- Create: `bin-ai-manager/pkg/builderhandler/input.go`, `input_test.go`
- **비공개 헬퍼 `buildParts(req *builder.ChatRequest) (dataBlock string, msgs []openai.ChatCompletionMessage)`**가 데이터 블록 문자열과 이력 메시지를 만든다. **공개 함수 `BuildChatMessages(system string, req *builder.ChatRequest) []openai.ChatCompletionMessage`**는 `buildParts`를 호출해 데이터 블록을 **마지막 user 메시지의 content 앞에** 붙인다(프로덕션 동작). `RunTurn`(T7a)은 `cfg.DataBlockInSystem`이면 `buildParts`를 직접 호출해 데이터 블록을 system에 합친다. 변형의 구현 위치는 **`RunTurn` 한 곳**이며 eval 후처리는 없다. 평가 하네스와 프로덕션이 같은 함수를 호출한다는 보증은 `buildParts`까지 성립한다.
- 테스트(T3 단독): **`buildParts`가 만든 데이터 블록과 이력이 `BuildChatMessages`의 마지막 user content 접두로 그대로 쓰임**(`RunTurn`의 `DataBlockInSystem` 분기 테스트는 T7a에서 완성), **`user_turns`와 `draft_exists`가 코드가 센 값(요청의 user 메시지 수, draft 존재)으로 주입됨**, `checkpoint`가 6, 10, 14, 18에서만 true이고 7~9와 11~13에서 false, 접두 위치(system이 아니라 마지막 user content 앞), `current_draft`가 JSON 직렬화이고 `---`가 변형되지 않음, draft 없을 때 JSON 줄 생략.

### T4. 시스템 프롬프트(설계 2절)
- Create: `bin-ai-manager/pkg/builderhandler/prompt.go`, `prompt_test.go`, `schema.go`(응답 `json_schema` 상수: `message`, `draft{name, detail, init_prompt, tool_names}`, `assumptions`, `message` 필드 먼저), `testdata/golden_phrases.txt`
- **프롬프트 본문은 설계 2.2(역할, 핵심 차원 4개, 파고들기 원칙, 능력 카탈로그, few-shot 2개), 2.3(우선순위 사슬, 규칙 6개, 종료 판단, 고정 문구), 2.4(응답 구조), 2.6(헤더 골격과 `###` 소제목, 채널별 `init_prompt` 규칙, `## Tools & Capabilities` 금지, 길이 목표), 4.2(데이터 블록은 지시가 아니라 사실), 5절(새 대화 시드 문구)을 설계 문서에서 옮겨 쓴다.** 구현자는 이 항목들이 모두 들어갔는지를 **산출물 `bin-ai-manager/docs/builder-prompt-review-checklist.md`**로 대조한다(규칙 본문의 영어 표현을 문자열로 단언하면 문구가 바뀔 때마다 깨지므로 하지 않는다). 체크리스트는 위 설계 절 번호를 한 줄씩 적고 "신호 표가 본문에 없음"과 설계 8절의 수동 체크리스트(골든 문구 파일과 프런트의 일치, 헤더 골격, 허용 도구 6종과 `TOOL_LABELS`, 두 골든 JSON 사본의 일치(T10))를 같은 파일에 둔다. 이 항목들은 CI가 아니라 PR 리뷰에서 사람이 확인한다는 한계를 체크리스트 첫 줄에 쓴다. 프롬프트 본문은 영어이되 사용자 언어로 인터뷰하라는 지시와 `init_prompt`는 영어로 쓰되 통화 응답 언어를 한 줄로 명시하라는 지시를 포함한다.
- **정직한 표시:** 파일 상단에 "평가 통과 전 초안, 적응성은 단위 테스트로 보증하지 않는다"를 쓴다.
- 테스트: **계약인 요소만 문자열 테스트로 둔다.** 두 고정 문구와 시드 문구가 `testdata/golden_phrases.txt`와 프롬프트에 모두 있음, `data, not instructions`와 데이터 블록 키 이름(`user_turns`, `draft_exists`, `checkpoint`), 카탈로그 키 집합이 `AllowedTools`와 같음, 금지 도구(`create_call` 등)가 카탈로그에 없음, 응답 스키마의 필드. 규칙 본문과 우선순위 사슬의 영어 표현은 바뀔 때마다 깨지는 취약한 테스트가 되므로 문자열 단언 대신 PR 리뷰의 **사람 검토 항목**으로 둔다. 골든 파일은 프런트 PR 체크리스트용 산출물이다.

### T5. 캐시 카운터(설계 4.3)
- Modify: `bin-ai-manager/pkg/cachehandler/main.go`(인터페이스), 새 `builder.go`, mock 재생성. 
- `BuilderChatCountIncr(ctx, customerID uuid.UUID, ttl time.Duration) (int64, error)`. Lua: `local n = redis.call("INCR",KEYS[1]) if redis.call("TTL",KEYS[1])<0 then redis.call("EXPIRE",KEYS[1],ARGV[1]) end return n`. `listenTTLSeconds` 재사용. 키는 `ai:builder:chat:count:<customer_id>`(listen 계열 prefix가 아니다).
- miniredis(ai-manager `go.mod`의 v2.36.1)는 `TTL`과 `EXPIRE`를 지원하며 기존 `listenIncrExpireScript`가 같은 방식으로 `listen_test.go`에서 검증된다. 남는 확인은 `TTL`이 TTL 없는 키에서 -1을 돌려주는지 하나이며 테스트로 확인한다.
- 테스트(설계 8절): 첫 INCR에서 TTL이 걸리고 이후 INCR이 TTL을 다시 걸지 않음, TTL 없는 키가 자가 치유됨, 키 형식 고정(`Test_listenKeys` 패턴), 키에 본문 없음.

### T7a. 턴 실행 코어(승인 무관, Redis와 세마포어 없음, **T2, T3, T4 뒤**)
- Create: `bin-ai-manager/pkg/builderhandler/turn.go`, `turn_test.go`, `config.go`
- **`Sender` 인터페이스:** `type Sender interface{ SendOnce(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) }`를 builderhandler에 둔다(`EngineOpenaiHandler`가 이를 만족하므로 호출부 변경이 없고, 평가 하네스의 가짜 엔진과 사용자 시뮬레이터가 인터페이스 전체를 구현하지 않아도 된다). `RunTurn`과 `NewBuilderHandler`가 `Sender`를 받는다. 가짜 엔진은 테스트에서 손으로 쓰며 mockgen 대상이 아니다.
- **`Config`(값 타입, 포인터와 슬라이스 필드 없음)의 평가 덮어쓰기 필드:** `Model`, `ReasoningEffort`, `MaxOutputTokens`, `LLMTimeout`에 더해 **`SystemPrompt`(기본값은 프로덕션 상수), `JSONMode`(문자열 enum `"json_schema"`(기본, `Strict:false`)와 `"json_object"`, `"none"`이며 T7a 테스트가 요청의 `ResponseFormat` 값을 단언한다), `DataBlockInSystem bool`(기본 false)**을 둔다. 평가 비교 축(설계 2.5: `reasoning_effort`와 모델, 신호 표 병기 프롬프트, 설계 4.2의 system 위치 비교, JSON 모드)은 **`Config`를 복사해 이 필드를 바꿔 넘기는 한 가지 방식**으로만 표현한다. 이 필드들은 프로덕션 코드에 평가 전용 분기가 생기는 대가이며 필드 주석에 "평가가 덮어씀, 프로덕션은 기본값"이라고 표시한다. `Config`가 유일한 우선순위 원천이다.
- **`RunTurn(ctx, sender Sender, cfg Config, req *builder.ChatRequest) (*TurnResult, error)`**: **`cfg.DataBlockInSystem`이 false이면 `BuildChatMessages(cfg.SystemPrompt, req)`를 그대로 호출하고**(접두 로직이 한 곳에만 있다), true이면 `buildParts`(T3)로 데이터 블록과 이력을 만들어 데이터 블록을 `cfg.SystemPrompt`에 합쳐 요청을 만든다(`BuildChatMessages`는 `cfg`를 받지 않는다). `cfg.LLMTimeout`으로 **`RunTurn` 내부에서** `context.WithTimeout`을 적용해 `SendOnce`를 호출한 뒤 `Parse`한다. 패키지 함수이며 `BuilderHandler` 인터페이스에 넣지 않는다(mockgen 대상 아님).
- **계약:** `TurnResult{Parsed *ParsedResponse, Usage openai.Usage, FinishReason string}`. 오류는 sentinel `ErrTruncated`(finish reason length), `ErrTimeout`(`context.DeadlineExceeded`), `ErrLLM`(그 밖의 엔진 오류, 분류 코드만 담고 입력이나 provider 응답을 싣지 않음), `ErrInvalidResponse`(파싱 실패)이며 `Chat`(T7b)이 이를 reason과 메트릭에 매핑한다.
- 테스트: **`cfg.DataBlockInSystem`이 true일 때 데이터 블록이 system에 합쳐지고 false일 때 마지막 user content 앞에 붙으며 두 경우 데이터 블록 내용이 같음**, 가짜 `Sender`로 잘림, 타임아웃, 엔진 오류가 입력을 에코하지 않음, 파서 연동, `Config` 값(모델, effort, max tokens, 시스템 프롬프트 교체, 데이터 블록 위치, JSON 모드. `"none"`이면 `ResponseFormat`이 nil)이 요청에 반영됨, `Usage`와 `FinishReason`이 `TurnResult`에 실림.

### T6. 설정(설계 4.6, 승인 게이트 대상, T7a 뒤에 배치하지만 코드 의존은 `Config` 필드 이름뿐)
- **정본은 T7a의 `builderhandler.Config`이고 T6은 이를 환경변수와 플래그에 연결하기만 한다.** `cmd/ai-manager/main.go` 배선은 T8에서 한 번에 한다(`internal/config`의 값을 `builderhandler.Config`로 복사). Modify: `bin-ai-manager/internal/config/main.go`, `main_test.go`: `ai_builder_enabled`(기본 false), `ai_builder_model`, `ai_builder_reasoning_effort`(기본 `none`, 미실측), `ai_builder_max_output_tokens`(4096, 미실측), `ai_builder_daily_limit`(200), `ai_builder_max_concurrent`(3, 미실측), `ai_builder_llm_timeout_sec`(40, 미실측). 플래그, 환경변수 매핑, `Config` 필드.
- (이 작업은 플래그와 환경변수 매핑까지만이며 `Config` 타입 정의는 T7a의 것이다. 배선은 T8에서 한다.)

### T7b. builderhandler 본체(설계 4.3, 4.5, T5와 T7a 뒤, 승인 게이트 이후)
- Create: `bin-ai-manager/pkg/builderhandler/main.go`(`BuilderHandler` 인터페이스는 `Chat`과 `Status`만, 생성자, `//go:generate mockgen -package builderhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod`, 메트릭을 `init()`에서 `prometheus.MustRegister`하는 `analysishandler` 패턴), `chat.go`, `status.go`, `chat_test.go`, mock.
- 생성자: `NewBuilderHandler(sender Sender, cache cachehandler.CacheHandler, cfg Config, keyConfigured bool) BuilderHandler`.
- `Chat(ctx, customerID, req)`. 순서: 킬 스위치와 키 확인, `ValidateRequest`, 세마포어 획득(non-blocking, 실패 즉시 `ResourceExhausted` `BUILDER_BUSY`, 카운트 안 함), `BuilderChatCountIncr`(초과 시 세마포어 해제 후 `BUILDER_DAILY_LIMIT`, Redis 오류 시 fail-closed `Unavailable` `BUILDER_UNAVAILABLE`), `RunTurn`(T7a의 sentinel을 매핑: `ErrTruncated`와 `ErrInvalidResponse`는 `Unavailable` `BUILDER_RESPONSE_INVALID`, `ErrTimeout`은 `BUILDER_TIMEOUT`, `ErrLLM`(provider 오류)은 `Unavailable` `BUILDER_RESPONSE_INVALID`(503, 프런트는 "다시 시도")로 매핑하며("설계와 다른 점" 5번. 설계 4.7에 provider 오류 행이 없다) 분류 코드만 로깅하고 **메트릭 라벨은 `llm_error`이며 Redis 오류의 `unavailable`과 구분한다**, 토큰은 `TurnResult.Usage`로 메트릭에 기록). **해제는 `defer`로 보장**한다. 로깅은 customer id, 메시지 개수, 결과 코드만. **`consume.go`의 `message consumer returns error. err: %v`(콜백 err)가 입력을 싣지 않도록 `Chat`과 T8이 err에 입력을 싣지 않는 것이 이 로그 방어의 의존 조건이다.**
- `Status() *builder.StatusResponse`: `enabled && keyConfigured`와 상한 값.
- 메트릭(설계 4.8 그대로): `ai_manager_builder_chat_total{result}`(ok, daily_limit, busy, disabled, unavailable, invalid_response, llm_error), `ai_manager_builder_chat_duration_seconds`, `ai_manager_builder_tokens_total{kind}`. **설계에 없는 `timeout` 결과 값은 추가하지 않는다.** 타임아웃은 `llm_error`로 세고, 구분이 필요하면 api-manager의 `api_manager_builder_timeout_total`이 있다. 라벨에 본문과 customer id를 넣지 않는다.
- 테스트: 검증 실패는 미카운트, `BUILDER_RESPONSE_INVALID`와 `ErrLLM`(provider 오류)은 카운트("설계와 다른 점" 5번), 일일 상한 경계(상한 번째까지 통과, 다음이 429), `Status()`의 세 상태, 순서(BUSY 미카운트, 일일 초과 시 해제, 오류, ctx 취소, panic 경로에서도 해제), **세마포어: LLM mock을 막아 둔 채 4번째 호출이 즉시 `BUILDER_BUSY`가 되고 완료 후 해제됨**, 카운터 호출 수, Redis 오류 fail-closed, 킬 스위치, 키 누락, LLM 오류 분류, 타임아웃, 잘림, **로그 캡처에 사용자 입력 문자열이 없음**(logrus test hook, LLM mock 오류가 입력을 에코하는 경우 포함).

### T8. listenhandler 라우트와 배선(설계 4.5)
- Create: `bin-ai-manager/pkg/listenhandler/models/request/builder.go`: RPC 전용 DTO `V1DataBuilderChatPost{CustomerID uuid.UUID, Messages, CurrentDraft}`. **신뢰할 customer id는 클라이언트 본문이 아니라 api-manager servicehandler의 `a.CustomerID`에서 온다**(T10). requesthandler와 listenhandler가 이 DTO를 공유한다(`ais.go` 등 기존 패턴).
- Modify: `bin-ai-manager/pkg/listenhandler/main.go`, 새 `builder.go`: `processRequest` **최상단**(함수 첫 줄의 `log` 생성 이전)에 `if isBuilderRoute(m) { return h.processBuilder(m) }`(`processRequest`에는 `ctx` 인자가 없고 `ctx := context.Background()`가 함수 중간에서 만들어지므로 **`processBuilder` 안에서 `context.Background()`를 생성한다**). `isBuilderRoute`는 **메서드와 무관하게 `m.URI`의 정확 일치(`==`)로**(`builder.URIChat`, `builder.URIStatus`, 쿼리스트링 없음) 판정해 잘못된 메서드가 기본 404 분기의 `log.Errorf(... m.URI)` 경로로 새지 않게 한다. **잘못된 메서드는 `errorResponse(cerrors.InvalidArgument(..., "INVALID_ARGUMENT", ...))`로 400을 본문 없이 반환한다**(설계 4.7의 상태 범위 400, 429, 503, 500 안에서만 쓴다). 이 판정은 모든 요청이 거치는 최상단이므로 정규식이 아니라 문자열 비교로 비용을 최소화한다. `processBuilder`는 이름 있는 반환값으로 `defer recover`를 쓰며 `m`과 `m.Data`를 어떤 로그에도 싣지 않고, **모든 오류(언마샬 실패 포함)를 `errorResponse(err)`로 변환해 `(response, nil)`을 반환**한다. 기존 `promReceivedRequestProcessTime`을 건너뜀을 주석으로 명시(대체: `ai_manager_builder_chat_duration_seconds`).
- `listenHandler`가 `builderHandler`를 받도록 **`NewListenHandler`(`cmd/ai-manager/main.go`의 `runListen` 안 호출)와 `runListen`의 정의와 `main`의 `runListen(...)` 호출부**를 수정한다. `ListenHandler` 인터페이스는 `Run() error`뿐이므로 listenhandler mock은 바뀌지 않는다. 테스트의 `&listenHandler{...}` 직접 생성은 필드 추가로 깨지지 않는다.
- Modify: `cmd/ai-manager/main.go`(배선은 여기서 한 번에 한다): `analysisEngine`을 재사용해 `NewBuilderHandler`를 생성(`keyConfigured = analysisKey != ""`), 켜져 있고 키가 비어 있거나 모델이 base URL의 제공자와 맞지 않아 보이면 경고 로그, `runListen`에 전달.
- 테스트: **왕복 골든 파일 비교(`bin-ai-manager/pkg/listenhandler/testdata/builder_*.json`을 `errorResponse`의 실제 출력과 비교 단언하고 `-update` 플래그로 최초 생성과 갱신, 5종 reason. 복원 쪽 사본은 T10)**, **모든 오류 경로에서 처리 함수가 `(response, nil)`을 반환함(err 반환 없음, 설계 8절)**, **status 라우트(`GET /v1/ai_builder/status`)의 세 상태(꺼짐, 키 없음, 정상)를 `processRequest` 전체로 확인**, **`processRequest` 전체를 통과하는** 테스트로 LLM 오류, 파싱 실패, 검증 실패, 언마샬 실패, 잘못된 메서드, panic을 유발하고 logrus hook이 캡처한 모든 로그에 사용자 입력 문자열(예: `SECRET-입력-문자열`)이 없음을 확인한다. 반환된 `sock.Response`를 `cerrors.FromResponse`로 복원했을 때 reason이 유지됨.

### T9. common-handler RPC와 로그 정리(설계 4.5)
- Create: `bin-common-handler/pkg/requesthandler/ai_builder.go`: `AIV1BuilderChat`(timeout 55초)과 `AIV1BuilderStatus`(timeout 3초). `builder.URIChat`과 `URIStatus`를 쓰고 `ai-manager/models/builder` 도메인 타입을 쓰는 것은 기존 `ai_analysis.go`의 `amanalysis` import 패턴과 같다. `requesthandler/main.go` 인터페이스 추가, mock 재생성. (기존 `requesthandler`에 메서드를 추가하며 새 패키지를 만들지 않으므로 루트 `CLAUDE.md`의 common-handler admission rule 적용 대상이 아니다. 이 점을 PR 본문에 한 줄 쓴다.)
- Modify: `bin-common-handler/pkg/rabbitmqhandler/consume.go`: 에러 문자열 세 곳(`could not parse the message`, `could not marshal the response`, `could not reply the message`)에서 본문(`message.Body`, `res`)을 제거하고 설명 문자열과 err만 남긴다. 이 문자열에 의존하는 기존 테스트는 없음을 확인함(grep 0건).
- **vendor는 커밋하지 않는다.** 로컬 검증을 위해 `go mod vendor`를 쓰되 저장소 정책상 커밋 대상이 아니며, 배포 이미지는 빌드 시 `go mod vendor`와 `go.mod`의 `replace`로 최신 common-handler를 쓴다(설계 4.5의 정정).
- 테스트: `main_test.go`의 `mockConnection`/`mockChannel.PublishWithContext`로 reply publish 실패를 유도해 에러 문자열에 응답 본문이 없음(**marshal 실패는 `sock.Response.Data`(`json.RawMessage`)에 유효하지 않은 JSON(예: `json.RawMessage("{invalid")`)을 넣어 유도하고, 오류 문자열에 RawMessage 내용이 일부라도 실리지 않는지 확인한다**), 요청 파싱 실패 에러에 본문이 없음, RPC 클라이언트 URI, timeout, 응답 파싱.
- **전 서비스 영향 검증(T13에서 실행):** common-handler 변경이므로 루트 `CLAUDE.md`에 따라 38개 서비스에 영향이 있다.

### T10. OpenAPI와 api-manager(설계 4.1, 4.2, 4.6, 4.7)
- 구현 전에 `bin-openapi-manager/CLAUDE.md`와 `bin-api-manager/CLAUDE.md`의 생성 절차를 읽는다. 확인된 사실: `bin-openapi-manager`는 `go generate ./...`, `bin-api-manager`는 `openapi/config_server/generate.go`의 `oapi-codegen`이며, `gens/openapi_server/gen.go`와 `gens/openapi_redoc/{api.html,openapi.json}`(npx 필요)가 git에 추적된다. 생성 순서는 openapi-manager, api-manager이고 `oapi-codegen`은 네트워크가 필요할 수 있다. npx가 없으면 redoc을 갱신하지 못하며 이 사실을 PR 본문에 기록한다.
- Create: `bin-openapi-manager/openapi/paths/ai_builder/chat.yaml`, `status.yaml`(이 저장소는 경로 파일 하나에 path item 하나), `openapi.yaml`에 두 `$ref` 등록, `components/schemas`에 `AiBuilderChatRequest`, `AiBuilderChatResponse`, `AiBuilderStatusResponse`(`available`, `max_messages`, `max_message_chars`). operationId 없이 경로에서 `PostAiBuilderChat`, `GetAiBuilderStatus`가 생성된다(기존 `PostAipromptproposals`와 같은 규칙). 생성물 재생성.
- Create: `bin-api-manager/server/ai_builder.go`: `getAuthIdentity` 후 chat은 `!a.IsAgent()`이면 `PermissionDenied`, status는 비에이전트와 권한 없음을 200 `available=false`로 통일(라우트는 `Authenticate` 그룹 안에 있다). **서버 핸들러는 `IsAgent` 검사만 하고, `CustomerAdmin`/`CustomerManager` 권한 검사(`h.hasPermission`)는 servicehandler 메서드 안에서 한다**(`servicehandler/campaigns.go`와 같은 패턴). chat은 `http.MaxBytesReader(c.Writer, c.Request.Body, 160<<10)` 후 `BindJSON`, `*http.MaxBytesError`는 `InvalidArgument` `BUILDER_INPUT_TOO_LARGE`, 그다음 **`builder.ValidateRequest`(T1)**. **요청 본문과 `BindJSON` 실패 로그에 본문을 싣지 않는다**(`a.AgentID()`, `a.CustomerID`, 메시지 개수만). **`context.DeadlineExceeded`를 `Unavailable` `BUILDER_TIMEOUT`으로, `circuitbreakerhandler.ErrCircuitOpen`을 `Unavailable` `SERVICE_UNAVAILABLE`로 변환하고 두 카운터(`api_manager_builder_timeout_total`, `api_manager_builder_circuit_open_total`)를 올리는 일은 모두 `servicehandler.AIBuilderChat` 안에서 한다**(감지와 카운트를 한 곳에 두어 server가 servicehandler의 카운터 메서드를 부르는 인터페이스 확장을 피하고, `error_translate.go`의 전역 매핑 우회도 한 곳에서 닫는다). 서버 핸들러는 `abortWithServiceError`만 호출한다. prometheus 선례는 `servicehandler/auth_delegate.go`뿐이며 같은 방식으로 등록한다(`Wrapf`가 `%w`를 보존하므로 `errors.Is`가 통하는지는 T10 테스트로 확인). `Canceled`는 세지 않는다. `DeadlineExceeded`는 RPC 55초 timeout과 상위 요청 ctx 자체의 만료를 구분하지 못하고 둘 다 센다.
- Create: `bin-api-manager/pkg/servicehandler/ai_builder.go`: `AIBuilderChat(ctx, a, req)`(DTO에 `a.CustomerID`를 채워 RPC), `AIBuilderStatus(ctx, a)`. status는 프로세스 로컬 캐시(성공 30초, 실패 5초, 만료 갱신은 **`singleflight`로 하나만 수행**하며 3초 RPC 동안 mutex를 잡지 않고, **RPC는 첫 호출자의 요청 ctx가 아니라 독립 ctx(`context.WithTimeout(context.Background(), 3초)`)로 `singleflight.Group.DoChan`을 호출하고 개별 호출자는 자기 ctx와 결과 채널을 `select`로 기다린다**(첫 호출자의 취소가 대기 중인 모든 호출자에게 전파되고 실패가 5초 캐시되는 것을 막기 위함). `golang.org/x/sync`는 `go.mod`에서 `// indirect`이므로 직접 import하면 `go mod tidy`가 direct로 바꾸고 `go.mod` 변경이 PR에 포함된다), 권한 검사는 캐시 앞, RPC 오류에서 `available=false`. 인터페이스(`servicehandler/main.go`)와 mock 재생성. 응답은 webhook 모델이 없으므로 OpenAPI 스키마 기준으로 직접 매핑한다.
- 테스트: 비에이전트 세 종류(accesskey, delegate, direct) 거부, 비권한 거부, status 통일, `MaxBytesReader`가 `BindJSON` 전에 적용되고 초과 매핑, `ValidateRequest` 경계값(api-manager 쪽), RPC timeout이 `BUILDER_TIMEOUT` 503으로 변환, `ErrCircuitOpen`이 503, 서버 핸들러 로그 캡처에 본문 없음, **wire 형식 보존(왕복):** 두 모듈은 별도 Go 모듈이라 api-manager 테스트가 ai-manager 핸들러를 호출할 수 없다. 따라서 (가) ai-manager 쪽에 **커밋된 정적 골든 파일** `bin-ai-manager/pkg/listenhandler/testdata/builder_*.json`(BUSY, DAILY_LIMIT, TIMEOUT, RESPONSE_INVALID, UNAVAILABLE)을 두고 T8 테스트가 `errorResponse`의 실제 출력과 **비교 단언**한다(`testdata` 디렉터리는 이 작업이 처음 만들며 최초 생성도 `-update` 플래그의 첫 실행으로 한다. 갱신은 `-update` 플래그로 수동), (나) api-manager가 `bin-api-manager/pkg/servicehandler/testdata/builder_*.json`에 같은 JSON 사본을 두고 `cerrors.FromResponse`로 복원했을 때 reason이 유지되는지 확인한다. **두 사본의 일치는 CI가 아니라 PR 리뷰에서 사람이 대조한다는 한계가 있으며 이를 T4의 체크리스트 파일에 적는다**, **`error_translate.go`에서 429 두 종류와 503 reason들의 변환 고정(설계 4.7)**, status 캐시(성공 30초, 실패 5초, 동시 만료).

### T11. 평가 하네스(설계 2.5)
- Create: `bin-ai-manager/pkg/builderhandler/eval/`(순수 로직은 태그 없음, 엔진 호출부는 `//go:build builder_eval`): 시나리오 정의(설계 2.5의 시나리오 목록을 페르소나 시트로), 사용자 시뮬레이터(빌더와 **다른 모델**, 높은 temperature, 비협조 페르소나, 어휘가 few-shot과 겹치지 않음), 자동 판정(JSON 파싱 성공률, 시나리오 9의 첫 응답 `draft` 존재, 허용 목록 밖 도구명 없음), 사람 판정용 대화 전문 덤프(마크다운), 합성 이력 5건(시나리오 15), 실행 횟수(3회 반복 5개, B1과 B2 2회, 나머지 1회, 1회 미달은 재실행 1회), 통과 조건 계산, 리포트.
- **하네스는 프로덕션 코어 `RunTurn`과 `BuildChatMessages`(T3, T7a)를 그대로 호출한다**(다른 경로를 측정하지 않기 위해). 비교 평가 축(설계 2.5와 4.2)은 **T7a의 `Config`를 복사해 평가 덮어쓰기 필드(`Model`, `ReasoningEffort`, `SystemPrompt`, `JSONMode`, `DataBlockInSystem`)를 바꿔 넘기는 방식으로만** 실행한다. **프롬프트 변형(신호 표 병기)은 `eval/testdata`의 변형 파일로만 존재하며 프로덕션 프롬프트에는 들어가지 않는다.** 시뮬레이터도 같은 `Sender`를 쓴다.
- **순수 로직(통과 조건 계산, 합성 이력 주입, 자동 판정 함수, 시나리오 로드)은 빌드 태그 없는 파일에 두어 일반 `go test ./...`에서 돈다.** 실제 엔진 호출과 CLI만 `//go:build builder_eval` 아래에 둔다.
- **dry-run의 정의:** 빌더와 사용자 시뮬레이터 **둘 다 가짜 엔진(고정 응답)**으로 시나리오를 끝까지 통과시켜 통과 조건 계산과 리포트 생성까지 확인한다. **가짜 엔진만 쓰므로 빌드 태그 없이 일반 `go test ./...`와 lint가 컴파일하고 실행한다.** `//go:build builder_eval`은 실제 엔진을 호출하는 CLI 진입점에만 붙인다.
- 환경: `BUILDER_EVAL_API_KEY`, `BUILDER_EVAL_BASE_URL`, `BUILDER_EVAL_MODEL`, `BUILDER_EVAL_SIM_MODEL`(없으면 즉시 skip하고 이유 출력).
- **실행 구성(설계 2.5를 다시 센 값):** 3회 반복 5개(2b, 3, 4의 A1과 A2, 6) x 3 = 15회, B1과 B2 각 2회 = 4회, 시나리오 9(세 입력 각 1회) 3회, 13(세 변형 각 1회) 3회, 14(네 변형 각 1회) 4회, 나머지(1, 2a, 7, 8, 10, 11, 12) 7회 = **36회**(설계의 "약 38회"는 산술 오류이며 위 "설계와 다른 점" 4번의 정정값이다).
- **이 PR에서 실제 LLM으로 실행하지 않는다(키 없음).** 하지만 **가짜 LLM(고정 응답)으로 전체 시나리오를 한 번 통과시키는 dry-run 테스트**를 추가해 하네스 배선 버그를 대표님이 처음 실행하기 전에 잡는다. 통과 조건 계산, 합성 이력 주입, 자동 판정 함수, 시나리오 로드 단위 테스트도 포함한다. 실행 방법과 비용 예산(36회 실행, 빌더 호출 약 216회)을 README에 쓴다. 사람 판정 항목은 하네스가 판정하지 않는다.

### T12. 문서(설계 7절 6단계)
- **공개 범위의 사실과 기본안:** 이 PR이 `gens/openapi_redoc/{api.html,openapi.json}`을 재생성해 커밋하므로 OpenAPI 스펙과 redoc에는 엔드포인트가 공개된다. **RST 절과 `docsdev/build` HTML은 "활성화 전 필수 조건"(고지와 약관 문구 결정) 이후에 올리는 것을 기본안으로 한다**(결정 전 문구가 기본으로 공개되지 않도록). 이 PR에서는 OpenAPI 스펙의 description에 "미출시, 평가 전, 기본 비활성"을 명시한다. 대표님이 RST를 지금 올리기로 하시면 고지는 설계 4.5의 한정 문구 전체("VoIPBin 애플리케이션은 대화를 저장하지 않는다. 브로커 내부 전달 상태는 포함하지 않으며 Google에 전송된 내용의 보관은 Google 정책을 따른다")만 쓰고 "저장하지 않는다" 단정 문장은 따로 쓰지 않는다.
- 이 PR에 포함하는 문서: `bin-api-manager/docs/routing.md`, `bin-ai-manager/docs/operations.md`(config 플래그와 메트릭. **`llm_error`는 LLM deadline 초과를 포함하고 api-manager의 `BUILDER_TIMEOUT` 카운터는 RPC 55초 timeout 기준이라 둘이 일치하지 않는다고 한 줄 적는다**), 셀프호스팅 설정 문서(`ai_builder_*`와 키 요구사항).
- RST를 올리는 경우: `bin-api-manager/docsdev/source/ai_struct_builder.rst`(webhook 모델이 없으므로 OpenAPI 스키마 기준) 신설과 **`ai.rst`의 toctree 등록**, 절차는 `bin-api-manager/CLAUDE.md`를 따른다(`cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` 후 `git add -f bin-api-manager/docsdev/build/`).

### T13. 검증(PR 전)
- `go generate ./...` 후 `git status`로 mock diff를 확인해 재생성 누락(cachehandler, builderhandler, requesthandler, servicehandler)을 막고, `ServerInterface` 구현(`PostAiBuilderChat`, `GetAiBuilderStatus`) 누락은 api-manager 컴파일로 확인한다. `go mod tidy` 후 `go.mod` 변경(예: `golang.org/x/sync`의 direct 승격)을 PR 본문 생성물 목록에 쓴다.
- **하네스 검증:** dry-run은 태그가 없으므로 일반 `go test ./...`에 포함된다. 태그가 붙은 CLI는 `go vet -tags builder_eval ./pkg/builderhandler/eval/...`로 컴파일만 확인한다.
- 변경 모듈(`bin-ai-manager`, `bin-common-handler`, `bin-api-manager`, `bin-openapi-manager`)에서 루트 `CLAUDE.md`의 verification workflow를 그대로 실행한다(`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`). **`bin-common-handler`를 바꿨으므로 영향받는 서비스 전체에서 최소한 `go build ./...`와 `go vet ./...`를 돌린다.** 돌리지 못한 것은 PR 본문에 쓴다.
- 새 문서와 코드 주석과 프롬프트(영어 본문 포함)에 em dash(U+2014)와 en dash(U+2013)가 없는지 `grep -P '\x{2014}|\x{2013}'`로 확인한다(결과 0건).
- 커밋 작성자: `git log -1 --format=%ae`가 `pchero21@gmail.com`.
- PR 전 `git fetch origin main`, 충돌 확인(`git merge-tree`), `HEAD..origin/main` 검토. 실제 PR 본문에는 수기 코드 파일 목록과 생성물 목록을 분리해 쓰고 평가 미실행과 활성화 전 필수 조건을 맨 위에 둔다.

---

## 이 PR에서 하지 않는 것
설계 10절과 같다(스트리밍, 선택지 칩, `captured`와 `open_topics`, 서버 이력 저장, 전역 일일 총량 상한, 별도 큐, 폼의 TTS/STT/언어 채우기, 프런트 구현).

## 열린 질문(대표님)
1. **평가 게이트 이탈 승인**(위 "설계와 다른 점" 1번, 승인 전 T5 등 서버 작업은 커밋하지 않는다): (a) 어두운 머지(서버 전체 한 PR, 킬 스위치 꺼짐), (b) 하네스, 프롬프트, 파서만 먼저 분리 머지(PR 분할이라 별도 승인), (c) 키를 주시면 평가를 먼저 실행.
2. 평가 키 제공과 실행 시점. 평가 예산은 정정값(36회, 빌더 호출 약 216(평균 6턴 가정), 사람 판정 전문 약 21건과 훑어보기 8건)으로 승인해 주십시오.
3. 설계 11절의 활성화 전 결정(전역 일일 총량 상한, 호스팅 활성화 범위, 외부 LLM 전송 고지와 약관).
4. **RST 문서(저장소 필수 규칙 이탈, 위 2번):** 기본안은 "외부 LLM 전송 고지와 약관 결정" 이후 공개이며, 승인 전에는 RST를 이 PR에 포함하는 쪽이 대안이다. OpenAPI 스펙과 redoc은 이 PR에 포함되면 이미 공개된다(제외하려면 OpenAPI 생성물을 이 PR에서 빼야 한다).
5. 프런트 PR(별도 저장소)을 이 흐름의 다음 단계로 진행할지.
6. provider 오류(`ErrLLM`)를 `BUILDER_RESPONSE_INVALID`로 매핑하는 선택((나), "설계와 다른 점" 5번)과 그 오류도 일일 횟수를 차감하는 현재안을 수용하실지, 일시 오류에 횟수를 환급(`Decr`)하도록 설계 4.3을 갱신할지("설계와 다른 점" 5번).

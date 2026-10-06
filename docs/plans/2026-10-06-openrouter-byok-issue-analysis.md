# 이슈 분석: 모델 선택기 키 방식 구분 + Custom OpenRouter(BYOK) 모델

Status: 리뷰 진행 중. **Q1a·Q1b·Q5·Q10은 CEO 승인 대기. Q1a·Q1b·Q5는 제안안(대표님 승인 대기)이며 확정이 아니다.** 이 문서에서 "권장"으로 적은 항목(Q9·Q10 등 CPO 결정 사항 포함)도 모두 제안안이며 CEO 승인 전에는 확정이 아니다.
수정 이력: 수정회차 4 (리뷰 5·6회차 반영. 로깅 누출 분리 결정을 "13곳 이번 PR 함께 처리" 권장안으로 재작성(R9 선택지 비교 표, Q10, §6, §8 항목 8), 오류 경로 목록에 `v1_ais.go` `%v` marshal 경로 추가와 지점 수 정정, §6 표를 단일 정본으로 재작성(문서·테스트·ai-control·gens·Q4·Q8·Q2·Q5 포함, 중복 행 병합), 오류 코드 `ENGINE_KEY_REQUIRED` 제안, `key_mode` 더 작은 대안 비교, Q9와 `engine_key` 응답 제거 이슈의 결합 명시, Python `test_init_pipeline.py:804,827,890` 추가)
수정 이력: 수정회차 3 (리뷰 3·4회차 지적 반영. R1-f·Q9·§3·§6 권고안을 "모든 AI sidebar 재조회(A) 이번 PR 포함"으로 통일, R9 로깅 누출 지점을 전 서비스 대상으로 재수집(call-manager 추가, 지점 수 정정), §6에 PR 범위 3열 표 신설, §8에 차단 대상/수행자/시점 열 추가, R1-a의 Python `None` 도착 경로와 `(key or "").strip()` 방어 명시, Q1a·Q3 3종 정렬, Q4 정리, 지원 표면 2.7절 신설, 목업 수정 필요 표기)
수정 이력: 수정회차 2 (리뷰 1·2회차 지적 반영. R1-f 해결안 재작성, R9 로깅 누출 지점 전수 재수집, Q1 분리(Q1a/Q1b), Q4 기본값 fail-closed로 변경, SDK 환경변수 폴백 확정, 테스트 영향 범위 보강, 롤아웃 순서 단일화 6.1절 신설)
Branch: NOJIRA-Add-custom-OpenRouter-BYOK-models (origin/main bc68e3c32 기준)
Date: 2026-10-06
선행 문서: `2026-10-05-openrouter-llm-routing-{issue-analysis,design,plan}.md` (main 포함, 이번 분석에서 다시 읽음)
UI 목업: `2026-10-06-openrouter-byok-ui-mockup.png` (열람 완료, 4절 참조)

표기: 파일:라인은 이 워크트리의 현재 코드를 직접 읽은 결과다. "미실증"은 코드·문서 읽기까지만 했고 실제 호출·실행으로 확인하지 않았다는 뜻이다. 영어로 노출될 문구는 `English` 원문으로 병기하며, 확정이 아니라 제안이다.

## 1. 요구 정리 (대표님 확정 방향)

1. 모델 선택기에서 키 방식을 배지와 그룹으로 구분. 그룹: PLATFORM PROVIDED(키 불필요), YOUR OWN KEY(내 키), 추천 Gemini 2.5 Flash는 "내 키 또는 기본 키"(Q1a 제안은 직접 모델 전체에 이 표기를 적용). 카탈로그 응답에 키 방식 필드(예: `key_mode`) 추가.
2. 고객 본인의 OpenRouter 키(BYOK)로 호출하는 Custom OpenRouter 모델.
   - CUSTOM 그룹에 "내 OpenRouter 키" 항목, 모델 ID 직접 입력, 키 필수.
   - 요청 단위 ZDR 강제. 해당 모델에 ZDR 제공자가 없으면 호출 실패 허용.
   - `platform_openrouter`와 별도 유형. ID 형태는 `openrouter.<OpenRouter 모델 ID>`.
   - 고객이 플랫폼 키를 쓰게 되는 경로는 절대 불가. 키가 비어 있으면 서버가 거부.
   - 이 유형에 한해 고객 화면과 문서에 OpenRouter 명칭 노출 허용(직전 설계의 "OpenRouter 비노출" 원칙의 예외).
3. 정책: 전역 on/off 플래그 금지, 오버엔지니어링 지양, 키 출력/로깅 금지, 고객 노출 문구와 코드 주석은 영어. 운영 DB의 저장된 `openrouter.*` 값은 0건(확인됨, 이번 재조회는 하지 않음).

## 2. 현재 코드 상태 (직접 확인)

### 2.1 Go: resolver, 카탈로그, 검증

| 항목 | 현재 상태 | 근거 |
|---|---|---|
| 단일 resolver | `ResolveEngine`: 카탈로그 일치 시 direct는 `RunnerType=string(m)`, openrouter 경로는 `platform_openrouter.<slug>` + `BlankKey=true`. 카탈로그 밖은 첫 `.` 앞이 `openai/gemini/grok`이면 `DirectPassthrough`, 나머지는 `Rejected` | `bin-ai-manager/models/ai/resolve.go:28-47` |
| 내부 유형 상수 | `RunnerServicePlatformOpenRouter = "platform_openrouter"` | `resolve.go:7` |
| 검증 진입점 | `IsValidEngineModel(m)`는 `Rejected` 여부만 반환. 키를 받지 않음 | `resolve.go:50-53` |
| 카탈로그 | 정적 슬라이스, 직접 11개 + OpenRouter 경유 9개. `Route`는 `RouteDirect/RouteOpenRouter` 둘뿐 | `models/ai/catalog.go:61-87`, `:6-9` |
| API 노출 모델 | `ModelInfo{id,label,vendor,recommended,tags,description,platform_managed}`. `platform_managed = (Route==RouteOpenRouter)` | `catalog.go:27-35,52` |
| create 검증 | `Create`가 `IsValidEngineModel`만 확인. `engineKey`는 검증 없음 | `pkg/aihandler/chatbot.go:51`(키 인자는 `:38`) |
| update 검증 | 변경된 `engine_model`만 검증(`preUpdateAI` 조회 후 비교) | `chatbot.go:148` |
| 오류 응답 | `INVALID_ENGINE_MODEL`(400). 메시지에 내부 라우팅 용어 금지 주석 | `chatbot.go:20-28` |
| 키 저장 | `ai.EngineKey`는 PUT 시 받은 값을 그대로 기록(전체 교체) | `pkg/aihandler/db.go:71,280` |
| CLI | `ai-control`도 동일 `IsValidEngineModel` 사용 | `cmd/ai-control/main.go:413,431` |
| 카탈로그 API | `GET /v1/ai_models`가 `ai.CatalogView()`를 그대로 직렬화. api-manager는 `amai.ModelInfo`를 그대로 통과(OpenAPI 타입으로 변환하지 않음) | `pkg/listenhandler/v1_ai_models.go:15-33`, `bin-api-manager/server/ai_models.go:36-37`, `servicehandler/ai.go:278-291` |
| 기존 테스트 | raw `openrouter.*`는 거부로 고정되어 있음 | `resolve_test.go:37`, `models/ai/main_test.go:337`, `chatbot_engine_model_test.go:27,72`, pipecat `llmresolve_test.go:41,66,130,216`, `run_teamllmtype_test.go:60`, **Python `bin-pipecat-manager/scripts/pipecat/test_init_pipeline.py:804,827,890`**(raw `openrouter.meta-llama/llama-3-70b`, `OpenRouter:x/y`, `openrouter.a/b`를 `_member_llm_type`/팀 초기화에서 거부하는 가드를 고정하는 파라미터 테스트, 이번 수정회차에 직접 확인) |

### 2.2 pipecat-manager (Go)

- `resolveSessionLLM(llmType, aiKey)`: `Rejected`는 오류, `BlankKey`면 키를 비움, 아니면 `aiKey`를 그대로 전달 (`pkg/pipecatcallhandler/llmresolve.go:17-28`).
- 호출 지점 4곳: `Start()`의 키 없는 사전 분류 `start.go:41`(DB row 생성 전), `startReferenceTypeAIcall` `start.go:208`(AI 조회 실패는 warn 후 빈 키로 진행, `start.go:198-205`), `runGetLLMKey` `run.go:141`(조회 실패 시 빈 키), 팀 멤버 `run.go:209`(거부 시 오류 로그만 남기고 `llm_type=""`).
- `Session.LLMKey`는 `json:"-"`라 세션 debug 로그에 직렬화되지 않음 (`models/pipecatcall/session.go:36`, `runner.go:265` 등의 `WithField("session", se)`).

### 2.3 Python 러너 (`bin-pipecat-manager/scripts/pipecat/run.py`)

- `create_llm_service`: 서비스명은 첫 `.`(없으면 첫 `:`)으로 분리, 소문자화 (`run.py:591-601`). 분기는 `openai`, `grok`, `gemini`가 `key or os.getenv(<PROVIDER>_API_KEY)` 패턴(`:603-633`), `platform_openrouter`는 키 인자 무시 + `OPENROUTER_API_KEY` 환경변수 + `provider.zdr/data_collection=deny/require_parameters` (`:658-687`). **raw `openrouter`는 분기가 없어 `Unsupported LLM service`** (`:689`).
- `_member_llm_type`: Go가 준 `llm_type`이 있으면 그대로, `""`이면 예외, 필드 자체가 없는 구버전 Go면 `engine_model`을 쓰되 서비스명이 `platform_openrouter` 또는 `openrouter`이면 예외 (`run.py:571-588`). 팀 멤버 LLM은 `create_llm_service(_member_llm_type(ai), ai["engine_key"], ...)` (`:821`), 단일 AI는 `:300`.
- **OpenAI SDK 환경변수 폴백(소스 직접 확인, pipecat-ai 1.12.0 / openai 3.24.0 기준, `~/.hermes/cache/scratch/ormvenv`)**: `OpenRouterLLMService.__init__`은 `api_key`를 그대로 `OpenAILLMService`로 넘기고(`pipecat/services/openrouter/llm.py:42-86`), `BaseOpenAILLMService.__init__`이 `create_client(api_key=...)`로 `AsyncOpenAI(api_key=api_key, ...)`를 만든다(`pipecat/services/openai/base_llm.py:254-290`). OpenAI SDK는 `api_key is None`일 때만 `os.environ.get("OPENAI_API_KEY")`를 읽는다(`openai/_client.py:252-253`). 즉 `api_key=None`이면 **러너 컨테이너에 주입된 `OPENAI_API_KEY`(compose 157-161, 231-235)가 `openrouter.ai`로 전송된다**(플랫폼 OpenAI 키의 제3자 유출 경로). 빈 문자열 `""`은 `None`이 아니므로 폴백하지 않고 빈 Bearer로 나간다(`api_key or ""`, `_client.py:258`). 이 동작은 소스 읽기로 확정했고 실제 컨테이너 실행은 미실증(§8). 현재 `platform_openrouter` 분기는 `OPENROUTER_API_KEY`를 먼저 읽고 비면 예외라 해당 없음(`run.py:658-665`).
- `llm_key`는 로그에 출력되지 않음 (`main.py:85,129`, `run.py:195-300`에서 인자 전달만 확인).

### 2.4 오류 분류 (`pkg/pipecatcallhandler/pipelineerror.go`)

- 분류: authentication(401/403 토큰·정규식), rate_limited(429, `insufficient credits`, `more credits`), timeout, unknown (`:20-86`). 404나 "no endpoints" 계열 문구는 어떤 티어에도 없음.
- 알림 여부: unknown은 STT 있는 음성 세션에서 알리지 않음 (`:96-114`). 고객 안내 문구는 카테고리별 고정 영문 문장 (`bin-ai-manager/pkg/messagehandler/pipeline_error.go:34-38`, authentication 문장은 `:35`).

### 2.5 OpenAPI, API, 문서

- `AIManagerAIModel`(required: id,label,vendor,recommended,tags,description,platform_managed), `engine_key`는 create/update 모두 required이며 빈 문자열 허용 (`bin-openapi-manager/openapi/openapi.yaml:2018-2058`, `paths/ais/main.yaml:108-118`, `paths/ais/id.yaml:153`).
- RST: `ai_struct_ai.rst:54`(`engine_key` 설명), `:180`(오류표), `:185-214`(모델 표와 검증 규칙). OpenRouter 명칭은 `self_hosting_providers.rst`(운영자용)에만 존재.
- api-manager의 `POST/PUT /ais` 핸들러는 `req.EngineKey`를 그대로 전달 (`bin-api-manager/server/ais.go:91,311`).

### 2.6 UI (`monorepo-javascript/square-admin/src`)

- `views/ais/ModelPicker.js`: 검색 + Recommended + 벤더 그룹. 카탈로그에 없는 저장값은 "Current: <id>" 합성 행(`:63,176-188`). 자유 입력 없음(`:36-38`).
- `useAIModels.js`: `isPlatformManaged(id)`는 `platform_managed===true` (`:94`). `types/api.ts:481`.
- 키 입력 처리(`platform_managed`일 때 disabled): `ais_create.js:50-51,201,577-585`, `ais_detail.js:86-87,422,541,566,1300-1309`, `teamgraph/sidebar.js` 생성 `:387,527,969-970`, 편집 `:581,807-808,1349-1366`.
- `AIEngineFields.js`는 `ModelPicker`만 렌더(`:143,340`). 키 입력은 이 컴포넌트 밖(각 화면)에 있음. `ModelPicker`는 sidebar 생성(`sidebar.js:945`), 편집(`:1302`)에서도 직접 렌더되므로 Custom 모델 ID 입력란은 `AIEngineFields`가 아니라 `ModelPicker` 내부에 둔다(디자인 문서 3.6.2).

### 2.7 지원 표면 (BYOK AI가 실행되는 곳, 코드 확인)

- **지원(이번 PR 대상)**: pipecat 러너를 거치는 경로. ai-manager가 `AIEngineModel`을 `pmpipecatcall.LLMType`으로 넘겨 pipecat-manager에 세션을 시작한다(`bin-ai-manager/pkg/aicallhandler/start.go:942,991`). 즉 voice 통화, 채팅(aicall), 팀 멤버, Insight 세션 중 pipecat을 타는 것이 해당한다. 모두 `resolveSessionLLM`과 Python `create_llm_service`(R1)를 통과한다.
- **비대상(production 호출자 없음)**: text-chat `engine_openai_handler`의 `MessageSend`/`StreamingSend`(`message.go:15`, `streaming_send.go:24`)는 `ai.GetEngineModelName(cc.AIEngineModel)`로 모델명만 쓰고 플랫폼 OpenAI 키(`cmd/ai-manager/main.go:163`의 `cfg.EngineKeyChatGPT`)로 호출하는 구조다. 다만 `bin-ai-manager` 전체를 `grep`한 결과 인터페이스·구현·mock 외에 이 두 메서드를 호출하는 비테스트 코드는 없다(이번 수정회차에 확인). 따라서 BYOK ID가 이 경로로 유입될 수 없다. 유입 경로가 새로 생기면 플랫폼 OpenAI 키로 `openrouter.*` 모델이 호출되는 구조이므로 그때 재점검해야 한다.
- **비대상(AI별 모델을 읽지 않음)**: Builder(`builderhandler/turn.go:48`은 설정의 고정 `cfg.Model` 사용), summary/analysis 핸들러(`Send`/`SendOnce`에 설정 모델로 만든 요청 전달).

## 3. 변경 영향 범위

| 영역 | 변경 |
|---|---|
| Go ai-manager | `ModelEntry/ModelInfo`에 `key_mode`, resolver에 BYOK 분기, 키를 받는 검증(생성/수정), `ai-control`, 오류 코드. **BYOK 키 누락은 별도 reason `ENGINE_KEY_REQUIRED`(HTTP 400, `cerrors.InvalidArgument`, `chatbot.go:20-28`의 `errInvalidEngineModel`과 같은 방식의 신규 헬퍼)로 제안**(제안안, CEO 승인 대기). `INVALID_ENGINE_MODEL`(모델 ID 형식 오류 등)과 구분해야 UI/SDK가 "키 입력 안내"와 "모델 선택 오류"를 분기할 수 있다. 메시지(영어, 제안): `An API key is required for custom OpenRouter models.` |
| Go pipecat-manager | `resolveSessionLLM`의 BYOK 처리(키 없는 사전 분류 호출과 구분), 팀 경로, 오류 분류 보강 여부 |
| Python | `create_llm_service`에 `openrouter` 분기(키 필수, 환경변수 폴백 금지), ZDR provider 설정 공유, `_member_llm_type` 가드 정리 |
| OpenAPI/gens | `AIManagerAIModel` 필드 추가, 재생성(`bin-openapi-manager/gens`, `bin-api-manager/gens`) |
| api-manager | `POST/PUT /ais` 및 카탈로그 통과 로직(비즈니스 로직)은 변경 없음. **다만 CPO 권장안(Q10, 제안안, CEO 승인 대기)을 따르면 `servicehandler/ai.go`의 로그 4줄(`:66`, `:121`, `:403`, `:465`)을 id 필드만 남기도록 한 줄씩 수정**한다(로직 무변경, 로그 줄만). CEO가 "분리"를 택하면 이 4줄은 제외되어 api-manager 변경이 없어진다. Q11 처리 시 OpenAPI 문구(`openapi.yaml:2196-2198`)와 gens 재생성이 생김 |
| 로깅 정리(Q10 권장안 시) | ai-manager 8곳, call-manager 1곳의 로그 한 줄 수정(R9 A). 신규 기능 테스트 외 전용 회귀 테스트는 만들지 않음 |
| UI | 선택기 그룹/배지, Custom 항목과 모델 ID 입력, 키 필수 UX, 4개 폼(create, detail, sidebar 생성/편집)의 키 처리, sidebar 편집 진입 시 `GET ais/{id}` 재조회(모든 AI 대상, R1-f, 6절 표 필수 열) |
| 문서 | `ai_struct_ai.rst`, `GET /ai_models` RST, `square-main/public/skill.md`, `llms.txt`(해당 시), 사용자용 BYOK 설명 |
| 테스트 | 2.1의 "raw openrouter 거부" 고정 테스트 전부 의미 변경: Go(`resolve_test.go:37`, `main_test.go:337`, `chatbot_engine_model_test.go:27,72`, pipecat `llmresolve_test.go`, `run_teamllmtype_test.go:60`), **Python `test_init_pipeline.py:804,827,890`**(`openrouter.*`를 거부하던 파라미터 케이스가 BYOK에서는 키 유무에 따라 허용/거부로 바뀜), `test_run.py` 등. 카탈로그 쪽 추가 영향(`catalog_test.go` 계열, `v1_ai_models_test.go`)은 R7의 "테스트 영향 범위" 참조 |

## 4. UI 목업에서 읽은 내용과 코드와의 차이

목업 3개 패널: (1) 선택기: RECOMMENDED(Gemini 2.5 Flash, 배지 "내 키 또는 기본 키" + Low cost), PLATFORM PROVIDED · 키 불필요(Claude Haiku 4.5, Llama 3.3 70B, 배지 "키 불필요"), YOUR OWN KEY · OpenAI / Gemini / Grok(GPT-5, Grok 3, 배지 "내 키"), CUSTOM("OpenRouter model (내 OpenRouter 키)", 배지 "내 키 필수"). (2) Custom 선택 시: Engine Model, "OpenRouter Model ID *"(예: `mistralai/mistral-large-2411`), "OpenRouter API Key *", 안내문 3개, 빨간 오류 "API key is required for custom OpenRouter models." (3) 현재 동작: 플랫폼 모델은 키 입력 disabled + "API key not required for this model.", GPT-5는 "Engine API Key (optional)" + "비워 두면 플랫폼 기본 키로 동작합니다".

코드와 충돌하는 점 (사실):
- 패널 3의 GPT-5는 배지 "내 키"인데 키는 optional이고 비우면 플랫폼 키로 동작한다고 적혀 있다. 코드도 직접 모델은 빈 키면 `os.getenv(<PROVIDER>_API_KEY)`로 폴백한다(`run.py:603,618,633`). **확정 사실(대표님 전달, 종류별 확인 완료)**: 운영 `bin-manager` 시크릿에 `OPENAI_API_KEY`, `XAI_API_KEY`, `GOOGLE_API_KEY`가 모두 존재하고 비어 있지 않으며, 러너 컨테이너에 주입된다(`bin-pipecat-manager/komodo/docker-compose.yml:157-161,231-235`). 따라서 직접 모델(OpenAI/Gemini/Grok)은 운영에서 빈 키일 때 플랫폼 키로 폴백하는 것이 사실이고, 추천 모델의 "내 키 또는 기본 키" 표기는 허위가 아니다. 반면 순수 "내 키"(플랫폼 키 폴백 없음) 표기는 현재 동작과 맞지 않는다(Q1a).
- 목업 패널 3은 현재 동작의 재현이 아니다. 실제 UI 라벨은 create/detail이 `Engine Key`(`ais_create.js:577`, `ais_detail.js:1300`)이고, `(optional)` 표기는 team graph sidebar에만 있다(`sidebar.js:968` `Engine Key (optional)`, `:1347` `API Key (optional)`). 목업의 `Engine API Key (optional)`과 "비워 두면 플랫폼 기본 키로 동작합니다" 문구는 코드에 없다(미실증: 다른 화면 문구 전수 대조는 하지 않음).
- Custom은 키 필수이므로 라벨이 `OpenRouter API Key *`로 바뀌어야 한다.

**목업 수정 필요 사항(디자인 단계 작업으로 표기)**:
- 패널 1: Q1a(전 direct 항목 `own_or_default`) 기준으로 YOUR OWN KEY 그룹 항목의 배지를 `내 키`에서 "내 키 또는 기본 키"(`Your key or default`)로 바꾸고, 배지를 3종(`No key needed` / `Your key or default` / `Your key required`)으로 정렬한다.
- 패널 3: 현재 동작의 재현이 아니므로(위 코드 대조) "현재 동작" 패널로 쓰려면 실제 UI 라벨(`Engine Key`, sidebar `Engine Key (optional)`/`API Key (optional)`)에 맞게 고치거나, 목표 동작 패널로 명칭을 바꿔야 한다. 두 경우 모두 디자인 단계에서 결정한다.

## 5. 핵심 리스크와 사실 확인

### R1. OpenRouter 키가 비었을 때 플랫폼 키로 폴백할 수 있는 경로 (최우선)

현재 코드에서 BYOK 도입 시 빈 키가 플랫폼 `OPENROUTER_API_KEY`에 닿을 수 있는 지점은 아래와 같다. 설계는 3중 방어(서버 검증, Go resolver, Python)가 필요하다.

| # | 경로 | 현재 코드의 위험 | 요구되는 방어 |
|---|---|---|---|
| a | Python 새 `openrouter` 분기 | 기존 3개 분기가 `key or os.getenv(...)` 패턴(`run.py:603,618,633`). 같은 패턴을 복사하면 곧바로 폴백. 러너 컨테이너에는 `OPENROUTER_API_KEY`가 실제로 주입되어 있음(`compose:161,235`) | 이 분기는 환경변수를 읽지 않고, **`(key or "").strip()` 후 비면 명시적 예외(`ValueError`)**. **`key`가 `None`으로 도착하는 것은 가상 시나리오가 아니라 정상 경로다**: Go `PythonRunner` 요청의 `LLMKey`는 `json:"llm_key,omitempty"`(`bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:87`)라 빈 키는 필드가 생략되고, Python `main.py:85`의 `llm_key: Optional[str] = None`이 `None`으로 받아 `create_llm_service`에 그대로 전달한다(`main.py:129`). 따라서 방어가 없으면 SDK의 `OPENAI_API_KEY` 폴백(2.3)이 BYOK 분기에서 실제로 닿는다. 단순 `key.strip()`은 `None`에서 `AttributeError`를 내므로 의도한 예외가 아니다. 정제한 값만 SDK에 전달하고 `None`/`""`는 SDK에 넘기지 않는다. 환경변수 읽기를 한 함수(platform 분기)로 한정하는 테스트 |
| b | 단일 AI 세션에서 AI 조회 실패 | `start.go:198-205`이 warn 후 빈 키로 진행. 지금은 direct 모델에서만 의미가 있었으나 BYOK에서는 빈 키가 러너까지 감 | `resolveSessionLLM`이 BYOK + 빈 키면 오류. 단 `start.go:41`의 키 없는 사전 분류 호출은 `""`를 넘기므로 이 호출은 BYOK 키 검사에서 제외해야 함(구분 필요, 설계 과제) |
| c | `runGetLLMKey` 조회 실패와 비 AIcall 참조 유형 | `run.go:121-141`에서 `aiKey=""`로 `resolveSessionLLM` 호출. 키 조회는 `ReferenceTypeAICall` 케이스에서만 일어나고(`run.go:122-139`), `ReferenceTypeCall` 등 다른 참조 유형은 `aiKey`가 항상 빈 채로 `resolveSessionLLM`에 들어간다 | b와 동일 규칙. 즉 BYOK 모델 + 비 AIcall 참조 유형은 항상 거부된다. 이는 **의도된 fail-closed**이며(이 유형들에서는 BYOK 키를 얻을 경로가 없음) 문서에 명시. 해당 조합의 실제 발생 여부는 미확인 |
| d | 팀 멤버 경로 | `resolveTeamForPython`이 `ai.EngineKey`를 그대로 전달(`run.go:209-228`). 거부 시 `llm_type=""` + 로그만 | BYOK 멤버의 빈 키도 거부(`llm_type=""`)로 취급, Python이 초기화 시 예외. 팀 멤버는 비동기 경로라 세션 시작 후 실패(기존 수용 한계와 동일) |
| e | 서버 저장 경로 | `Create`/`Update`는 키를 검증하지 않음(`chatbot.go:51,148`). 공백만 있는 키, 모델만 BYOK로 바꾸고 기존 빈 키 유지하는 update가 통과 가능 | BYOK 모델이면 `strings.TrimSpace(engineKey) != ""` 필수(위반 시 reason `ENGINE_KEY_REQUIRED`, 400, §3). update는 "모델이 BYOK"인 최종 상태를 기준으로 검사(모델이 안 바뀌어도 키가 비면 거부) |
| f | sidebar 편집 저장 | **코드로 확정한 사실**: (1) sidebar의 AI 목록 캐시 저장 시 `engine_key`를 항상 제거(`sidebar.js:488`, `:636-639`), 캐시 히트 경로는 키 없는 객체를 `setAiData`(`:622-624`). API 재조회 경로(`:630-639`)의 `setAiData(res)`(`:633`)만 키를 보유. 따라서 `aiData.engine_key`는 캐시 경유 시 `undefined`다. (2) 편집 저장은 키가 안 바뀌면 `engine_key`를 body에서 생략(`:807-808`). 서버는 PUT의 `engine_key`를 부재=빈 문자열로 받아 전체 교체(`bin-api-manager/server/ais.go:311`, `request/ais.go:60` `omitempty`, `aihandler/db.go:280`). **sidebar에서 이름만 바꿔 저장해도 키가 빈 값으로 덮인다(코드 기준 확정, 실행 실증은 §8)**. (3) 저장 후 `setAiData({..., engine_key: aiData.engine_key})`(`:820-822`)도 캐시 경유 시 `undefined`를 유지할 뿐이다. 이 덮어쓰기는 **직전 구현의 직접 모델에서도 이미 존재하는 문제**이고(키 삭제 후 운영 시크릿 키로 폴백하므로 눈에 띄지 않을 뿐) BYOK의 신규 버그가 아니다 | 이전 초안의 "`aiData.engine_key` 재전송"은 전제가 틀렸으므로 폐기. 해결안 후보: **(A)** 편집 진입(또는 저장 직전)에 `GET ais/{id}`로 재조회해 키를 확보하고 키가 안 바뀌었으면 그 값을 재전송(키가 화면 메모리에 올라옴, `ais_detail.js:203,422`가 이미 같은 방식). **(B)** 서버가 `engine_key` 부재를 "변경 없음"으로 해석하도록 의미 변경(포인터/필드 맵 필요). B는 PUT 의미 변경이라 모든 API 클라이언트에 영향이 있는 비호환 후보이고 이번 PR 범위로는 크다. **제안안(CEO 승인 대기, 권장안 단일화): A를 모든 AI(직접 모델 포함)에 이번 PR에서 함께 적용한다.** 근거: 직접 모델과 BYOK가 같은 sidebar 코드 경로라 직접 모델까지 적용해도 추가 비용이 없고, **서버가 BYOK + 빈 키를 400으로 거부하므로(R1-e) A를 적용하지 않으면 sidebar에서 BYOK AI의 이름만 바꿔도 `engine_key`가 생략되어 항상 400으로 실패한다(BYOK에 필수 작업)**. 직접 모델에서 이미 존재하던 키 덮어쓰기 문제도 같은 수정으로 사라지므로 별도 버그로 분리하지 않는다 |
| g | 버전 혼재 | 신규 Python + 이전 Go(#1363)는 raw `openrouter.*`를 `Rejected`로 막음(싱글: `start.go:41`, 팀: `llm_type=""`). 신규 Go + 이전 Python은 `Unsupported LLM service`로 실패 | 두 경우 모두 안전(fail-closed). **배포 순서는 6.1절에 단일화**하여 기술 |
| h | 키 이월 | 모델을 BYOK에서 direct(예 `openai.*`)로 바꾸고 키를 두면 OpenRouter 키가 OpenAI로 전송됨. 반대로 `sk-...`(OpenAI) 키가 남은 채 BYOK로 바꾸면 OpenAI 키가 OpenRouter로 전송됨. 플랫폼 키 문제는 아니나 고객 키의 타 벤더 유출. **서버 측 한계**: 서버는 최종 (모델, 키) 스냅샷만 받으므로 키 발급처를 알 수 없고, 키 형식 검증도 하지 않는다(Q7). 따라서 현재 AI에 저장된 키와 새 모델의 조합이 맞는지 서버가 판별해 거부할 수 없다(맞지 않으면 호출 시 401). 방어는 UI에 의존 | UI에서 키 방식 그룹이 바뀔 때 키 입력 초기화 또는 경고(Q8). 서버는 "BYOK 최종 상태 + 비어 있지 않은 키"만 보장 |

확인된 안전 요소: `platform_openrouter` 분기는 키 인자를 무시한다(`run.py:658-665`). Custom은 이 분기를 절대 타면 안 된다(R4). aicall 레코드에는 키가 복사되지 않는다(`models/aicall/main.go`에 `engine_key` 없음).

### R2. ZDR 요청 단위 동작과 실패 양상

사실(문서 확인, 2026-10-06 openrouter.ai 문서). 출처:
- ZDR 가이드: https://openrouter.ai/docs/guides/features/zdr
- BYOK 가이드: https://openrouter.ai/docs/guides/overview/auth/byok
- Provider Routing(`provider.zdr` 필드): https://openrouter.ai/docs/features/provider-routing
(세 주소 모두 2026-10-06 직접 열람. 아래 요지는 해당 페이지 본문 기준이며 실호출로는 확인하지 않음)
- `provider.zdr: true`는 요청 단위로 동작하며, 계정/가드레일 설정과 OR 관계다. 요청 값으로 계정 강제를 끌 수는 없다.
- BYOK 문서: "ZDR을 강제하면 보관 엔드포인트는 막히고, ZDR 엔드포인트가 남지 않으면 요청이 실패한다. 내 키가 있어도 마찬가지"(키에 ZDR 선언을 한 경우만 예외). 즉 고객 키 호출에서도 `zdr`이 적용되고, 모델에 ZDR 제공자가 없으면 실패한다는 요구와 일치한다.
- 현재 platform 분기와 같은 `extra={"extra_body": {"provider": {...}}}` 구조를 쓰면 된다(`run.py:669-678`, 직전 설계 D-F1/D-F2). BYOK 분기에서 이 dict를 복사하지 않고 한 곳에서 공유해야 한 쪽만 빠지는 사고를 막는다(변이 테스트 대상).

미확인(실호출 필요, 고객 키를 가진 테스트 키가 필요):
- ZDR 제공자 없음일 때의 HTTP 상태와 메시지 원문. 문서의 일반 `not_found`(404) 표에서는 이 경우를 구분해 적지 않았다. 직전 문서에 가정한 문구를 쓰지 않는다.
- `require_parameters:true`와 합쳐질 때 "ZDR 없음"과 "tools 미지원 엔드포인트 없음"을 메시지로 구분할 수 있는지.
- 고객 계정의 자체 가드레일(허용 제공자 제한 등)이 걸린 경우 403/404 양상.

분류 문제(코드 사실): 이 실패가 404 계열이면 `classifyPipelineError`의 어느 티어에도 걸리지 않아 `unknown`이 되고(`pipelineerror.go:85`), 음성 세션(STT 있음)에서는 알림이 생성되지 않는다(`:109-110`). 즉 고객은 무음 통화만 겪고 원인을 알 수 없다. 선택지는 Q5.

### R3. 모델 ID 형식과 서비스명 파싱 충돌

- 형식 예: `vendor/model`, `vendor/model:variant`(`:free`, `:nitro`, `:floor` 등 문서에 존재, `:online`/`:thinking`은 직전 분석 R15에 기재), 점이 포함된 `meta-llama/llama-3.3-70b-instruct`.
- 서비스명 파싱은 Go `strings.SplitN(m, ".", 2)`(`resolve.go:41`)와 Python `type.split(".", 1)`(`run.py:594`) 모두 **첫 점**에서만 나누므로, 접두사 `openrouter.` 뒤의 `/`, `:`, 추가 `.`은 모델명 쪽에 그대로 남는다. 충돌 없음.
- 잔존 주의점 둘: (1) `type`에 `.`이 없을 때만 `:` 분기가 쓰이므로 `openrouter.`로 시작하는 한 `:`는 영향이 없다. (2) `_member_llm_type` 폴백의 서비스명 추출은 `replace(":", ".", 1)` 후 분리(`run.py:582`)라 `openrouter.vendor/m:free`도 서비스명 `openrouter`로 나온다. 구버전 Go 폴백에서만 쓰임.
- 입력 정책 필요: 현재 검증은 첫 점 뒤 비어 있지 않음만 확인한다. 자유 입력에는 최소한 길이(DB `engine_model varchar(255)`, `scripts/database_scripts_test/table_ai_ais.sql:11`, `table_ai_aicalls.sql:8`)와 문자 집합(공백, 제어문자, 쉼표 등 차단) 제한이 필요하다. 모델 ID는 JSON 본문의 `model` 값으로만 전송되므로 URL 경로 주입 위험은 코드상 낮음(`OpenRouterLLMService.Settings(model=...)`, 직전 D-F1, 미실증). 변종 접미사(`:free` 등)와 라우터형 ID(`openrouter/auto` 등)는 **실증 전까지 거부(fail-closed)**하는 것을 기본으로 한다(Q4, ZDR 우회 가능성 때문).

### R4. `platform_openrouter` 승격 방지(실증 필요 사항)

현재 보장: `platform_openrouter.*`는 고객이 저장할 수 없음(Rejected, `resolve_test.go:38`). 플랫폼 키 경로는 오직 카탈로그 항목의 `RunnerServicePlatformOpenRouter + "." + UpstreamSlug`로만 만들어진다(`resolve.go:36`).

BYOK 분기가 생기면 새로 생기는 승격 시도 표면과 요구되는 테스트(모두 아직 없음, 구현 시 실증):
- Go resolver: `openrouter.x` 결과는 `BlankKey=false`, `RunnerType`이 `platform_openrouter.` 접두를 갖지 않음. 대소문자(`OpenRouter.x`, `PLATFORM_OPENROUTER.x`), 앞뒤 공백, `openrouter.`(빈 ID), `openrouter.platform_openrouter.x`가 BYOK 외 결과를 만들지 않음. Python은 서비스명을 소문자화하므로(`run.py:601`) Go가 통과시킨 값이 Python에서 `platform_openrouter`로 해석되는 입력이 없음을 표 테스트로 고정.
- 카탈로그 불변 조건: 어떤 카탈로그 ID도 `openrouter.`로 시작하지 않는다(BYOK 입력과 충돌 방지). 기존 `Test_CatalogInvariants`(`catalog_test.go:9-38`)에 추가. 단 Custom 항목을 서버 카탈로그에 내리면(Q2) 이 불변 조건의 예외 정의가 필요하다(R7).
- Python: 키 비어 있는 `openrouter` 분기가 `OPENROUTER_API_KEY`를 읽지 않음(환경변수 설정 상태에서 예외가 나는지 실제 실행으로 확인). 변이 테스트: BYOK 분기에 `or os.getenv` 한 줄을 넣으면 테스트가 빨개져야 함.
- 팀 경로: BYOK 멤버 + 빈 키, 정상 키 혼합 팀(`run_teamllmtype_test.go` 확장).

### R5. UI: Custom 항목, 저장값, dirty 비교, sidebar

코드 사실:
- 저장된 BYOK 값 `openrouter.vendor/model`은 카탈로그에 없으므로 현재 `ModelPicker`는 "Current: <id>" 합성 행으로 보인다(`ModelPicker.js:63,176-188`). 이대로면 Custom 입력 필드로 되돌아가지 못하고, 사용자는 모델 ID를 고치려면 합성 행을 못 쓴다. Custom 항목 선택 + 별도 모델 ID 입력(접두 자동 부가)으로 바꾸고 저장값이 `openrouter.`로 시작하면 Custom 모드로 복원해야 한다.
- 직전 설계가 "클라이언트는 서버의 접두 규칙을 복제하지 않는다"(design 3.7)고 결정했다. Custom 판별에 접두 `openrouter.`가 필요하므로 이 원칙의 예외가 생긴다. 카탈로그가 Custom 항목을 서버에서 내려주는 방식이 일관되나 필드가 늘어난다(Q2).
- dirty 비교: `ais_detail.js:541-542`는 `engineModel`(전체 문자열)과 키를 비교한다. Custom 모드의 `engineModel`을 조합된 최종 문자열로 유지하면 비교 로직은 그대로 쓸 수 있다.
- 키 처리 4개 폼은 모두 `platform_managed` 이분법이다(2.6). "키 필수" 상태가 새로 필요하며 `ais_create.js:201`은 비면 `ref.current.value`를 그대로 보내므로 클라이언트 검증이 없다. 서버가 최종 방어(R1-e)이고 UI는 즉시 오류 표시용.
- sidebar 편집은 R1-f 문제가 있다(캐시에 키가 없어 `aiData.engine_key`가 `undefined`일 수 있음). 해결은 재조회(A)로 통일한다(모든 AI 대상, 이번 PR 포함, 제안안. R1-f). 단순 재전송은 불가하고 서버 의미 변경(B)은 범위 밖. 마스킹 표시(`sidebar.js:1350`)는 `aiData.engine_key`가 있을 때만 점 표시하므로 캐시 경유 시 키가 있어도 빈 칸으로 보인다(코드 기준, 미실증).
- `ModelLabel`/`modelLabel`(`aicalls_*`, `ais_list` 등)은 카탈로그 밖 ID를 원문 표기로 폴백하므로 BYOK ID는 그대로 표시된다. 동작 변경 불필요(미실증).

### R6. 운영 DB 기존 `openrouter.*` 값
이미 0건 확인됨(대표님 전달). 따라서 기존 값 마이그레이션이나 하위 호환 분기는 불필요. 이후 값은 신규 검증을 거친 것만 존재.

### R7. `key_mode` 필드 공존/호환
- 서버는 `platform_managed`를 유지하고 `key_mode`를 추가(가산 변경). 구 UI는 모르는 필드를 무시한다(JSON). api-manager는 `amai.ModelInfo`를 통과시키므로(2.1) Go 쪽 변경은 `ModelInfo`뿐, OpenAPI는 문서/생성 타입 용도(`gen.go`의 `AIManagerAIModel`).
- 신규 UI + 구 API(필드 없음): `key_mode` 부재 시 `platform_managed`로 폴백(`true`면 platform, 아니면 own)해야 롤아웃 순서가 바뀌어도 깨지지 않는다. 배포 순서는 6.1절 참조(폴백은 방어용).
- **Custom 항목을 서버 카탈로그로 내릴 때(Q2)의 테스트 영향 범위(코드로 확인한 기존 테스트)**:
  - `Test_CatalogInvariants`(`catalog_test.go:9-38`): `Route`가 `RouteDirect`/`RouteOpenRouter` 둘뿐이라는 검사(`:30-32`)와 direct 항목 slug 금지(`:27-29`), 모든 항목 `Description` 필수 등. Custom 항목용 신규 Route 값 또는 예외가 필요.
  - `Test_CatalogViewShape`(`catalog_test.go`, `len(view) == len(catalog)` 등 뷰 형태 고정)과 `Test_CatalogViewPlatformManagedMatchesRoute`(`:103-147`, `platform_managed == (Route==RouteOpenRouter)`을 모든 항목에 대해 검사). Custom의 `platform_managed`는 false여야 하고 새 필드(`key_mode`, 접두 필드)에 대한 동치 검사 추가 필요.
  - `Test_CatalogPublicViewHidesInternals`(`catalog_test.go:41-51`): 공개 뷰 JSON에 `openrouter`가 있으면 실패. Custom 항목의 라벨/접두에 `openrouter`가 들어가므로 **누출 방지 테스트를 약화(Custom 항목 1개에 한정한 예외)해야 한다.** 이 가드는 플랫폼 키 경로가 고객에게 새지 않도록 하는 장치라 예외 범위를 좁게 고정해야 함.
  - `Test_CatalogCustomerFacingTextHasNoBannedTerms`(`catalog_test.go:152-163`): 라벨/설명에 `openrouter`, `zero data`가 있으면 실패. Custom 항목의 라벨·설명(`OpenRouter model (your OpenRouter key)`, 안내문의 ZDR 문구)이 걸린다.
  - `v1_ai_models_test.go`(`:38` 필수 키 목록, `:46` 응답 본문에 `openrouter` 금지)와 OpenAPI 타입(`AIManagerAIModel`).
  - UI 상수로 두는 안(Q2 대안)은 위 5개 테스트를 건드리지 않으나, 접두 상수가 클라이언트에 복제되고(직전 설계 3.7 원칙 예외) CLI/SDK 사용자가 Custom 존재를 `GET /ai_models`로 알 수 없다.
- Custom 항목이 카탈로그에 같이 내려가면 구 UI는 일반 모델 행으로 보여 선택 후 모델 ID 없이 저장하려 할 수 있다. 서버가 최종 방어(`openrouter.` 뒤 비어 있으면 거부)하므로 안전하지만 UX는 열화된다. 구 UI는 배포 시간차 동안만 존재.
- OpenAPI `required` 목록에 추가하면 생성 클라이언트가 엄격해질 수 있으나 서버가 항상 보내므로 문제는 없다.
- **text-chat 경로 확인(수정회차 3에서 재확인)**: `engine_openai_handler`의 `MessageSend`(`message.go:15`)와 `StreamingSend`(`streaming_send.go:24`)는 `ai.GetEngineModelName(cc.AIEngineModel)`로 모델명만 꺼내 `openai.ChatCompletionRequest.Model`에 넣는 별도 경로이고 pipecat 러너/`create_llm_service`를 거치지 않지만, `bin-ai-manager` 전체에서 이 두 메서드의 비테스트·비mock 호출자가 없다(2.7). 따라서 BYOK ID가 유입되지 않고 이번 키 방어(R1)의 대상이 아니다.
- `platform_managed`는 영구 유지할지, `key_mode`로 대체할지(대체하면 기존 UI 4곳과 문서 RST의 설명 수정) 결정 필요(Q6).
- **`key_mode` 3값의 더 작은 대안(한 줄 비교, 제안안)**: `platform_managed`를 그대로 두고 Custom 판별 필드만 추가하는 안(예: `custom: true`). 필드 추가는 더 작지만 **기각 권장**: 배지 3종(`No key needed`/`Your key or default`/`Your key required`)을 서버가 값으로 결정해 주어야 UI에 키 방식 판단(예: "direct면 own_or_default")이 하드코딩되지 않고, 이후 Q1b(키 필수화)나 신규 항목이 생겨도 서버 값만 바꾸면 되며, CLI/SDK 사용자도 `GET /ai_models`만으로 키 요건을 알 수 있다. 불리언 2개 조합(`platform_managed` + `custom`)은 3종 배지를 클라이언트가 재계산해야 한다.

### R8. 오류 메시지/문구의 OpenRouter 명칭 노출 범위
- 허용 범위는 Custom 유형에 한정. 기존 `INVALID_ENGINE_MODEL` 메시지(`chatbot.go:26`)는 일반 문구이므로 변경 불필요. 신규 오류(키 필수 `ENGINE_KEY_REQUIRED`, 모델 ID 형식은 기존 `INVALID_ENGINE_MODEL` 재사용)는 Custom 전용으로 `openrouter.` 접두를 언급해도 된다.
- Custom 호출 실패 시 고객 안내는 기존 카테고리 문구를 쓴다. authentication 문구는 이미 "custom engine key" 확인을 안내한다(`pipeline_error.go:35`)고 하여 BYOK 401에 적합하다. rate_limited 문구는 402(크레딧 없음)에도 이미 걸린다(`pipelineerror.go:37`, 테스트 `pipelineerror_test.go:92`). 신규 문구가 필요한 것은 ZDR 없음 계열뿐이다(R2).

### R9. 키 노출 (이번 작업 범위 밖이지만 BYOK로 중요도가 커지는 기존 문제, 코드로 확인)

**로깅 누출 지점 전수(수정회차 3에서 모든 `bin-*` 서비스를 대상으로 재수집)**. `ai.AI`의 `EngineKey`는 `json:"engine_key,omitempty"` 태그를 가지며(`bin-ai-manager/models/ai/main.go:67`), `AI`에는 `String()`/`MarshalJSON` 같은 가림 처리가 없다(`models/ai/webhook.go`의 `ConvertWebhookMessage`/`CreateWebhookEvent`만 있음). 따라서 `logrus` 필드로 `*ai.AI`(또는 `amai.AI`)를 넘기는 모든 줄이 후보다. 이전 회차의 "다른 서비스에는 없다"는 단정은 **틀렸다**: `bin-call-manager`에 지점이 있었다. 이번 재수집은 (1) 모든 `bin-*`에서 `WithField("ai"|…)`, `"ai": <변수>`, `"request": m`, `engine_key`/`engineKey` 로그 필드를 `grep`하고, (2) `AIV1AI*`(Get/List/Create/Update/Delete/ActivateInsight/DirectHashRegenerate) 호출자 전부(api-manager, call-manager, pipecat-manager)의 결과 사용처를 직접 열어 확인했다.

**A. 키 값이 항상 로그 필드에 실리는 지점: 13곳 (명시 필드 2 + 구조체 11)**

| 구분 | 서비스 | 위치 | 내용 |
|---|---|---|---|
| 명시적 키 필드 | api-manager | `pkg/servicehandler/ai.go:66`, `:403` | `AICreate`/`AIUpdate`의 `logrus.Fields`에 `"engine_key": engineKey` |
| `*ai.AI` 구조체 | api-manager | `servicehandler/ai.go:121`(`Created a new ai.`), `:465`(`Updated ai info.`) | 응답 `tmp`(키 포함) |
| 〃 | ai-manager | `pkg/aicallhandler/start.go:260`(`startReferenceTypeCall`), `:302`(`startReferenceTypeConversation`), `:486`(`startReferenceTypeContactCase`) | 함수 전체 `log` 필드에 `"ai": a`. 이후 이 `log`를 쓰는 모든 줄에 실릴 수 있음 |
| 〃 | ai-manager | `pkg/aihandler/db.go:150`, `pkg/aihandler/direct_hash.go:28` | `Retrieved ai info.` |
| 〃 | ai-manager | `pkg/aicallhandler/send.go:170`, `pkg/aicallhandler/tool.go:1078` | `Resolved team member AI.`, `Retrieved AI info.` |
| 〃 | ai-manager | `pkg/teamhandler/handler.go:244` | `Retrieved ai info.` |
| 〃 (**신규 발견**) | **call-manager** | `pkg/callhandler/start_incoming_domain_type_sip.go:217-223` | `AIV1AIGet`으로 받은 `*amai.AI`를 `log.WithField("ai", a).Debugf("Retrieved AI info. ai_id: %s", a.ID)`로 기록(직접 해시 SIP 통화가 AI로 라우팅될 때마다) |

**B. 오류 경로에서만 키가 실릴 수 있는 지점(조건부): 최소 10곳 + `%v` 패턴 경로**

| 서비스 | 위치 | 내용 |
|---|---|---|
| ai-manager | `pkg/listenhandler/main.go:283-286`의 `"request": m` | `sock.Request.Data`는 `json.RawMessage`. 이 `log`를 쓰는 줄은 오류 경로 `main.go:631`, `:642`뿐. `POST/PUT /v1/ais` 본문에 `engine_key`가 있으므로 이 오류 경로에 도달하면 본문이 로그에 실림(도달 조건 미실증) |
| ai-manager | `pkg/listenhandler/v1_ais.go:83`(`processV1AIsPost`), `:237`(`processV1AIsIDPut`)의 `"request": m` | 각 핸들러의 `log`는 `Errorf` 오류 경로에서만 사용(`:88,102,128,135,142`, `:242,271,276,302,309,316`)이라 정상 경로에서는 출력되지 않음. 요청 본문(키 포함)이 오류 줄에 실림 |
| ai-manager (**신규 추가**, 수정회차 4에서 코드 확인) | `pkg/listenhandler/v1_ais.go:142`(Create), `:181`(Get), `:220`(Delete), `:316`(Update)의 `log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)`, 그리고 `:66`의 같은 문구 `log.Debugf`(List, `tmp`는 AI 목록) | `%v`로 `*ai.AI`(또는 목록)를 문자열화하는 패턴이라 `WithField("ai", …)` grep에 걸리지 않는다. `json.Marshal(tmp)` 실패 경로에서만 실행되므로 평상시 출력 가능성은 매우 낮다(미실증). 단 출력되면 `EngineKey` 값이 포함된다. 같은 파일의 다른 핸들러(`processV1AIsIDDelete` 포함)에도 동일 패턴이 있다 |
| common-handler | `pkg/notifyhandler/publish.go:34`(`PublishWebhook`의 `"data": data`), `pkg/requesthandler/publish_event.go:21`(`"data": data`) | 모든 서비스가 공유. `log`는 `Errorf` 경로에서만 사용. ai 웹훅/이벤트 페이로드는 `ConvertWebhookMessage`가 `EngineKey`를 복사하므로(`models/ai/webhook.go:29,70`) 게시 오류 시 키가 실릴 수 있음(실제 페이로드 확인 미실증) |

지점 수: ai-manager 3(`main.go` 1, `v1_ais.go` request 2) + `v1_ais.go` marshal `%v` 5 + common-handler 2 = **최소 10곳**. `%v`/`%+v` 패턴은 이름 기반 grep으로 놓칠 수 있어 "최소"이며, 이번 수정회차에 `v1_ais.go`만 직접 열어 확인했다. 다른 핸들러 파일(`v1_teams.go` 등)의 동일 패턴은 AI 구조체가 아니라 해당 없음으로 보이나 전수 확인은 하지 않았다(미실증).

**C. 확인 결과 누출 아님(참고)**: `pipecat-manager`는 `AIV1AIGet` 결과(`ai`)를 로깅하지 않고(`run.go:181-187`은 `member_id`, `ai_id`만, `run.go:249` 이하 `resolveAIFromAIcall`은 반환만) `start.go`는 오류만 기록한다. `Session.LLMKey`는 `json:"-"`(2.2). api-manager `server/ais.go`의 로그 필드는 `request_address`, `auth` 등이며 요청 본문은 담지 않는다. `aicallhandler`의 `"ai": c`(`start.go:726`, `db.go:39,119`)의 `c`는 `*ai.AI`이며 `engine_key` 필드를 가지므로 **누출 지점이다**(이전 서술은 `*aicall.AIcall`로 잘못 적었다. 디자인 수정회차 3에서 시그니처를 직접 읽고 정정했고 로깅 정리 범위는 13곳에서 16곳이 되었다. `2026-10-06-openrouter-byok-design.md` 3.4 A 참조). `engine_openai_handler`의 `"request": req`는 OpenAI 요청 구조체이고 production 호출자가 없다(2.7).

**지점 수 정정**: 이전 수정회차 표(구조체 로깅 "약 11곳")는 call-manager를 빠뜨려 부정확했다. 재수집 기준 무조건 로그 필드에 키가 실리는 지점은 **13곳**(명시 필드 2 + 구조체 11. 서비스별로는 api-manager 4, ai-manager 8, call-manager 1), 서비스 3개. 오류 경로 한정 지점은 **최소 10곳 + `%v` 패턴 경로**(위 B). 이 수치는 grep 기반이라 하한이며, 변수명이 다르거나 `%v`로 문자열화하는 로깅은 놓칠 수 있다.

**BYOK 키는 고객 과금과 직결된다**: OpenAI/Gemini 키가 로그에 남는 기존 문제와 달리, 고객의 OpenRouter 키가 유출되면 해당 고객 OpenRouter 계정에 청구되는 사용량(과금)으로 곧바로 이어진다. 또 CEO 정책 "키 로깅 금지"와 이전 안(전수 정리를 별도 이슈로 분리)은 긴장 관계에 있다. 그래서 CPO는 분리안을 철회하고 아래 비교 표의 **전수 13곳을 이번 PR에서 함께 처리**를 권장한다(제안안, CEO 승인 대기).

**로깅 누출 처리 선택지 비교 (제안안, CEO 승인 대기)**

| 선택지 | 범위 | 비용 | 위험/잔여 | CPO 판단 |
|---|---|---|---|---|
| **1. 전수 13곳 함께 처리 (권장)** | A의 13곳(api-manager 4, ai-manager 8, call-manager 1)을 구조체·키 대신 id 필드만 남기는 **기계적 한 줄 수정**. 로그 메시지와 다른 필드는 유지. 로직 변경 없음 | 3개 서비스에 로그 한 줄씩. 신규 기능 테스트 외 전용 회귀 테스트는 만들지 않음(오버엔지니어링 지양) | 정책("키 로깅 금지")과 일치. B의 오류 경로는 아래 선택 항목으로 남음 | 권장. BYOK 키는 과금 직결이므로 알려진 무조건 누출을 남긴 채 출시하지 않는다 |
| 2. api-manager 4곳만 | 키를 요청 본문에서 처음 받는 지점만 정리 | 최소 | ai-manager 8곳과 call-manager 1곳은 `AIV1AIGet` 결과를 매 통화/세션 시작마다 기록하므로 오히려 빈도가 높은 지점이 남음 | 비권장. 누출 빈도가 큰 쪽이 남는다 |
| 3. 분리(이전 안) | 이번 PR은 신규 코드 무로깅만, 기존은 별도 이슈 | PR 범위 최소 | 출시 시점에 BYOK 키가 기존 13곳에 기록됨. 정책과 긴장 | 철회 권장. 분리하려면 대표님이 위험을 명시 수용해야 함 |

**오류 경로(B) 처리**: 키가 실제로 로그 줄에 실리는지 미실증이므로 **이번 PR 선택 항목**(`v1_ais.go` `%v` 5곳, `request` 로깅 3곳, notifyhandler/requesthandler publish 2곳). 출시 전 확인 조건은 Q10에 둔다(로그 레벨·포매터 실증, 이미 쌓인 로그 보관·폐기 여부 대표님 판단).

**실제로 키가 출력되는지는 추정이다.** 코드로 확인한 것: 두 서비스 모두 `logrus.SetLevel(logrus.DebugLevel)`을 무조건 설정(`bin-ai-manager/internal/config/main.go:328-329`, `bin-api-manager/internal/config/main.go:252-253`)하고 포매터는 `joonix`다. 이로 미루어 위 Debug 줄은 켜져 있을 것으로 **추정**되며, 운영에서 레벨을 다른 곳에서 덮는지, `joonix`가 구조체 필드를 JSON 태그대로 직렬화해 `engine_key`가 실제 로그 줄에 나오는지는 **미실증**(§8).

- `GET /ais`와 ai 웹훅 이벤트가 `engine_key`를 포함하는 것으로 코드상 보인다: `ConvertWebhookMessage`가 `EngineKey`를 복사(`models/ai/webhook.go:29,70`), `AIGet`이 이를 반환(`servicehandler/ai.go:330`), UI도 이 값을 사용(`ais_detail.js:203,422`, `sidebar.js:488`의 제거는 로컬 캐시 용도). 문서와 OpenAPI가 코드와 불일치한다: RST는 "never returned"(`ai_struct_ai.rst:77`), **`bin-openapi-manager/openapi/openapi.yaml:2196-2198`의 `engine_key` 설명이 "Write-only; not returned in responses."**이다. 실제 응답 실증은 미수행. 고쳐서 마스킹하면 UI가 저장 키를 재전송하는 흐름(`ais_detail.js:422`)이 깨지므로(그리고 R1-f의 A안 재조회가 이 응답 노출에 의존하므로) 이번 PR 범위에서 제외하고 별도 이슈로 분리해 대표님께 보고하는 항목이다(제안안). **결합 주의: 별도 이슈로 `engine_key` 응답 제거를 진행하면 이번 PR의 R1-f A안(편집 시 `GET ais/{id}` 재조회로 키 확보)과 `ais_detail.js:203,422`의 재전송 흐름이 깨진다. 따라서 그 이슈는 A안의 대체 수단(예: PUT에서 `engine_key` 부재=변경 없음 의미 변경)과 한 쌍으로만 착수할 수 있다.** 단 문서 불일치(RST, openapi.yaml)는 Q11에서 이번 PR에 문구를 정정할지("returned"로) 또는 코드를 문서에 맞출지 결정 필요.
- 비밀번호급 값인 OpenRouter 키를 같은 컬럼(`engine_key varchar(255)`)에 평문 저장하는 것은 기존 OpenAI/Gemini 키와 동일하다(신규 위험 유형은 아님).
- 러너가 OpenRouter 오류 원문을 최대 2000자 로깅한다(`pipelineerror.go:12`, 직전 R20). OpenRouter 401 본문에 키가 반사되는지는 미확인. 실호출 시 확인.

## 6. 진행 타당성과 권장 범위

판단: 진행 타당. 직전 PR의 resolver/세션 해석 구조가 이미 있어 BYOK는 "키 있는 direct 계열의 한 변종 + 고정 provider 옵션"으로 구현 가능하다. 다만 키 방어 3중 구조(R1)와 sidebar 키 손실(R1-f)이 이번 작업의 실질 난점이다. 아래 영역별 표(필수/선택/별도 이슈)가 이 PR의 범위를 하나로 고정한다(모두 제안안, CEO 승인 대기).

**이 표가 PR 범위의 단일 정본이다.** 각 칸 안의 항목은 `<br>`로 구분했다. "(확정 대기)"는 CEO 결정이 나야 칸 이동이 확정되는 항목, "(실호출 후 확정)"은 §8 실증 결과에 따라 달라지는 항목이다.

| 영역 | 이번 PR 필수 | 같은 PR 선택 | 별도 이슈 |
|---|---|---|---|
| 서버 코드 | BYOK 유형: resolver 분기(키 필수), `ValidateEngine(model,key)`(create/update/`ai-control` 공용, update는 최종 상태 기준), 오류 reason `ENGINE_KEY_REQUIRED`(400)<br>Python `openrouter` 분기, ZDR dict 공유, `(key or "").strip()` 빈 값 예외<br>pipecat-manager `resolveSessionLLM`/팀 경로 BYOK 처리(R1)<br>`ModelInfo.key_mode` 3값 추가(`platform_managed` 유지, Q6) | **Q5 분류 보강**: ZDR 없음 응답 원문 확인 후 (b) 새 카테고리 또는 (c) 매핑(실호출 후 확정. 확인 전에는 (a) `unknown` 유지) | 직접 모델 빈 키 거부(Q1b)<br>`GET /ais` 응답의 `engine_key` 제거(R1-f A안과 결합. **A안의 재조회가 이 응답에 의존하므로 단독 착수 불가**, PUT 의미 변경(B)과 한 쌍, Q9) |
| 모델 ID 검증 | **Q4**: 길이 255, 허용 문자 영숫자와 `. _ - /`, `:` 변종과 라우터형 ID 거부(실증 전 fail-closed) | 실증(§8 항목 6, 10)에서 ZDR 유지가 확인된 변종 순차 허용 | |
| Q2 결정 (확정 대기) | Custom 항목을 서버 카탈로그로 제공하는 경우: 신규 Route/필드, `catalog_test.go` 계열과 `v1_ai_models_test.go` 수정(R7), 누출 방지 가드 예외를 Custom 1개로 한정 | UI 상수안(6절 대안 A)을 택하면 위 테스트 변경이 사라지고 UI에 접두 상수가 생김 | |
| 로깅 (Q10, 제안안, 확정 대기) | **신규 코드는 키를 로깅하지 않음**(신규 로그 줄에 `*ai.AI`, 요청 본문, 키 값 금지)<br>**권장안: R9 A의 13곳(api-manager 4, ai-manager 8, call-manager 1)을 id 필드만 남기는 한 줄 수정으로 함께 처리**(정책 "키 로깅 금지" 일치, BYOK 키는 고객 과금과 직결). 전용 회귀 테스트는 만들지 않음 | R9 B 오류 경로(최소 10곳: `listenhandler` `request` 3, `v1_ais.go` `%v` 5, common-handler 2). 출시 전 확인 조건은 Q10 | 로그 중앙 처리(logrus Hook 등, Q10 (B))<br>CEO가 "분리"를 택하면 13곳도 이 칸으로 이동(위험 수용 명시 필요) |
| UI | 선택기 그룹/배지(3종), Custom 항목과 모델 ID 입력, 키 필수 UX와 오류 문구(Q3)<br>4개 폼(create, detail, sidebar 생성/편집) 키 처리 `key_mode` 일반화<br>**sidebar 편집 진입 시 `GET ais/{id}` 재조회(A안, 모든 AI, Q9)**<br>**Q8 키 이월 UX**: create는 키 입력 초기화, detail/sidebar 편집은 키 방식 그룹 변경 시 경고 문구 | | 목업 수정(4절, 디자인 단계 작업) |
| 문서 (Q11) | `ai_struct_ai.rst`(Custom 유형, `engine_key` 필수 조건, ZDR 안내, 비용은 고객 계정 청구)<br>`GET /ai_models` 문서 `ai_models.rst`(`key_mode`)<br>`square-main/public/skill.md`<br>`llms.txt`(BYOK 언급이 해당될 때) | `ai_struct_ai.rst:77`과 `openapi.yaml:2196-2198`의 "engine_key 응답 안 됨" 문구를 "returned"로 정정(문구만, §8 항목 4 실증 후, 같은 PR 권장) | 자체 호스팅 문서는 변경 없음(BYOK는 플랫폼 키 불필요) |
| OpenAPI, gens | `AIManagerAIModel`에 `key_mode`(및 Q2 서버 제공 시 접두 필드) 추가, `bin-openapi-manager/gens`, `bin-api-manager/gens` 재생성 | | |
| 테스트 | 의미가 바뀌는 기존 테스트 갱신: Go(`resolve_test.go:37`, `models/ai/main_test.go:337`, `chatbot_engine_model_test.go:27,72`, pipecat `llmresolve_test.go:41,66,130,216`, `run_teamllmtype_test.go:60`), **Python `test_init_pipeline.py:804,827,890`**, `test_run.py`<br>카탈로그 계열(`Test_CatalogInvariants`, `Test_CatalogViewShape`, `Test_CatalogViewPlatformManagedMatchesRoute`, `Test_CatalogPublicViewHidesInternals`, `Test_CatalogCustomerFacingTextHasNoBannedTerms`), `v1_ai_models_test.go`(Q2에 따라)<br>`cmd/ai-control/main_test.go`(`ValidateEngine` 호출 변경분)<br>신규: 승격 방지 표 테스트(R4), BYOK 빈 키 거부(Go/Python), 변이 테스트(`or os.getenv` 추가 시 실패) | | |
| ai-control | `cmd/ai-control/main.go:413,431`이 `ValidateEngine(model,key)`를 쓰도록 변경(키 인자 전달) | | |


권장 최소 범위 (오버엔지니어링 회피):
1. 카탈로그: `ModelInfo`에 `key_mode` 문자열 1개만 추가(값 3종: `platform`, `own_or_default`, `own_required`, Q1a 제안). `platform_managed`는 이번엔 유지(Q6). 새 구조체/설정/플래그 없음.
2. BYOK 유형은 resolver에 분기 하나. 러너 유형 문자열은 고객 ID와 동일(`openrouter.<id>`)하게 두어 직전 `DirectPassthrough`처럼 "키를 그대로 전달"하되, 새 `OutcomeCustom`(또는 `Resolved.RequireKey`) 하나로 키 필수를 표현. `platform_openrouter`와 코드 경로를 공유하지 않는다.
3. 서버 검증은 키를 받는 새 함수 하나(`ValidateEngine(model, key)` 형태)로 create/update/ai-control이 같이 사용. update는 최종 상태 기준.
4. Python은 `openrouter` 분기 하나. ZDR provider dict는 모듈 상수로 platform 분기와 공유.
5. UI는 `ModelPicker`에 그룹/배지와 Custom 항목, 모델 ID 입력 1개. 4개 폼의 키 처리는 `key_mode`로 일반화하되 새 컴포넌트는 만들지 않는다.
6. 오류 분류는 실호출로 ZDR 없음 응답 문구를 확인한 뒤에만 보강(Q5). 추측 문구로 분류기를 확장하지 않는다.

### 6.1 롤아웃 순서 (R1-g, R7을 단일화)

1. pipecat-manager Go + Python **동시** 배포. 어느 한쪽만 먼저 나가면 신규 Python + 이전 Go는 `Rejected`, 신규 Go + 이전 Python은 `Unsupported LLM service`로 실패하지만 모두 fail-closed(플랫폼 키 폴백 없음).
2. ai-manager(카탈로그 `key_mode`, 서버 검증) 배포. 1번 이전에 이것이 먼저 나가면 BYOK 모델을 저장할 수 있으나 호출은 `Rejected`로 실패(안전하나 UX 열화). 그래서 1번이 먼저.
3. 신규 UI 배포. 신규 UI + 구 API(필드 없음)는 `key_mode` 부재 시 `platform_managed`로 폴백(`true`면 platform, 아니면 own). 구 UI + 신규 API는 모르는 필드를 무시하고 Custom 항목을 일반 행으로 보여 UX가 열화될 수 있으나 서버 검증이 최종 방어.

더 작은 대안:
- A. Custom 항목을 카탈로그에 내리지 않고 UI가 `openrouter.` 상수와 고정 문구를 가짐. API 변경은 `key_mode` 1개로 끝나나, 접두 상수가 클라이언트에 복제되고 CLI/SDK 사용자는 Custom 존재를 `GET /ai_models`로 알 수 없다. (CEO가 "카탈로그에 키 방식 필드" 방향을 확정했으므로 권장은 서버 제공 쪽, Q2)
- B. 키 방식 배지만 먼저 배포하고 BYOK를 후속 PR로 분리. 기본 규칙(1 PR/작업)과 대표님 확정 방향(한 요청 두 기능)에 반하므로 제안만 하며 분리하지 않는다.

## 7. 디자인 단계 열린 질문

| # | 질문 | 권장안 |
|---|---|---|
| Q1a | **(제안안, 승인 대기)** 키 방식 값 체계와 항목별 배정. 운영 시크릿 3종이 모두 존재하므로(4절) 직접 모델은 전부 "내 키 또는 기본 키"가 사실이고, 순수 "내 키"(폴백 없음) 배지는 현재 동작과 맞지 않는다. **값 3종**: `platform`(No key needed), `own_or_default`(Your key or default), `own_required`(Your key required). 쟁점: 직접 모델 11개가 모두 같은 폴백 동작을 가지므로 전 direct 항목에 `own_or_default`를 동일 적용할지, 목업처럼 Gemini 2.5 Flash만 구분할지 | **전 direct 항목에 `own_or_default` 동일 적용**(그러면 순수 `own` 값은 존재하지 않음). 그룹은 3개(PLATFORM PROVIDED / YOUR OWN KEY / CUSTOM)이고 `own_or_default` 항목은 YOUR OWN KEY 그룹에 둔다. 목업 패널 1의 GPT-5 `내 키`와 별도 `추천` 구분 배지는 허위가 되므로 디자인 단계에서 수정 필요(4절). 대표님 확인 필요 |
| Q1b | **(제안안, 승인 대기)** 직접 모델의 빈 키를 서버에서 거부할지(키 필수화). 거부하면 키 없이 플랫폼 키로 동작 중인 기존 AI와 진행 중인 호출이 깨지는 비호환 변경 | 이번 PR은 거부하지 않음(현 동작 유지, 배지로 정직하게 표기). 필수화는 별도 결정 |
| Q2 | Custom 항목을 카탈로그가 내려주는가(`key_mode:"own_required"` + 모델 ID 접두를 알려주는 필드 1개), UI 상수로 두는가. 비용 비교: 서버 제공은 카탈로그 테스트 5개 이상 수정과 `openrouter` 누출 방지 가드 약화(R7 "테스트 영향 범위"), 신규 Route/필드, OpenAPI 변경이 든다. UI 상수는 테스트 변경이 없으나 접두 복제와 CLI/SDK 발견 불가 | 서버 제공(대표님 확정 방향 "카탈로그에 키 방식 필드"와 일관). 단 가드 예외를 Custom 항목 1개에 한정해 테스트로 고정. 비용이 부담되면 UI 상수안(6절 대안 A)이 더 작음. 대표님 선택 |
| Q3 | 고객 노출 문구 확정(제안): 그룹 `PLATFORM PROVIDED` / `YOUR OWN KEY` / `CUSTOM`, 배지 3종 `No key needed`(`platform`) / `Your key or default`(`own_or_default`) / `Your key required`(`own_required`) (Q1a와 정렬, 이전의 `Your key` 4번째 배지는 제거), Custom 항목 `OpenRouter model (your OpenRouter key)`, 설명 `Enter any model supported by OpenRouter.`, 필드 `OpenRouter Model ID *`(도움말 `Enter the model ID from the OpenRouter model page, e.g. vendor/model-name.`), `OpenRouter API Key *`, 안내 `Calls use this key and are billed to your OpenRouter account. The platform key is never used.` / `Zero data retention (ZDR) routing is enforced on every request. If no ZDR provider supports this model, calls will fail.` / `The AI cannot be saved without an API key.`, 오류 `API key is required for custom OpenRouter models.`(목업 원문) | 대표님 문구 확정 |
| Q4 | 모델 ID 입력 정책(제안): 최대 길이 255(DB `varchar(255)`, 접두 `openrouter.` 포함), **허용 문자는 영숫자와 `. _ - /`**(`~`는 제외). **거부 규칙(별도)**: `:` 변종 접미사(`:free`, `:nitro`, `:online` 등)와 `openrouter/auto` 같은 라우터형 ID는 요청한 모델과 다른 모델/제공자로 라우팅되거나 ZDR 강제와 상호작용하는 방식이 미확인이라 ZDR 우회 가능성을 배제할 수 없으므로 실증 전 거부 | 위 규칙을 서버에서 검증(fail-closed). 실증(§8 항목 10)에서 ZDR 유지가 확인된 변종만 순차 허용 |
| Q5 | **(제안안, 승인 대기)** ZDR 제공자 없음 실패 고객 안내: (a) 기존 `unknown` 유지(음성 세션 무음, 운영 로그만), (b) 새 카테고리 추가(열거형 확장, `pipeline_error.go` 문구 맵, RST, 테스트 변경), (c) 해당 문구를 감지해 기존 카테고리로 매핑. 먼저 실호출로 응답 원문 확인 필요. **실호출에는 고객 OpenRouter 키를 가진 테스트 계정이 필요하다(대표님 또는 검증용 키 제공).** 대안: 기존 플랫폼 OpenRouter 키를 "고객 키 역할"로 재사용해 응답 형태만 확인하는 방법이 가능하나, 플랫폼 키는 가드레일/ZDR 계정 설정이 고객 계정과 다를 수 있어(R2 문서: 계정 설정과 요청 값은 OR 관계) 결과가 고객 환경과 다를 수 있고, 이 키를 BYOK 코드 경로 테스트에 넣는 것은 "플랫폼 키 사용 금지" 검증의 의미를 흐리므로 일회성 수동 확인에만 쓰고 코드/테스트에는 넣지 않는 것을 권장 | 실호출 결과에 따라. 원문이 안정적으로 구분되면 (b)를 권장(무음 통화는 고객 신뢰 문제), 아니면 (a)+문서 안내 |
| Q6 | `platform_managed`를 유지하는가, `key_mode`로 대체하는가 | 이번 PR은 유지(구 UI/외부 클라이언트 호환), 폐기는 별도 |
| Q7 | 키 형식 사전 검증(`sk-or-` 접두 등): OpenRouter 키 형식 공식 문서 근거 미확인이라 하지 않음이 안전. 형식 불일치 키는 호출 시 401로 드러남 | 하지 않음 |
| Q8 | 키 이월 방지(R1-h): 키 방식 그룹이 바뀔 때 UI가 키 입력을 비우는가 | create는 비움, detail/sidebar 편집은 사용자가 지울 수 있게 경고 문구 표시 |
| Q9 | sidebar 편집의 키 손실(R1-f) 처리. 선택지: (i) 별도 버그로 분리하고 BYOK는 서버 검증(400)으로 방어, (ii) 편집 진입 시 `GET ais/{id}` 재조회(A)를 모든 AI에 적용, (iii) 서버 의미 변경(B) | **(ii) 단일 권장(제안안, CEO 승인 대기)**. 이번 PR에 포함. 직접 모델과 BYOK가 같은 코드 경로라 비용이 같고, (i)로 두면 서버가 BYOK+빈 키를 400으로 거부하므로 sidebar에서 BYOK AI 이름 변경이 항상 실패한다. 별도 버그로 분리하지 않음. (iii)은 PUT 의미 변경이라 채택하지 않음. **결합 조건**: 별도 이슈 "GET 응답의 `engine_key` 제거"를 진행하면 (ii)의 재조회가 키를 얻지 못해 깨진다. 그 이슈는 (iii) 또는 동등한 대체와 한 쌍으로만 진행한다(§6 표) |
| Q10 | 키 로깅 정책(R9). **필수(이번 PR)**: 신규 코드가 키(`*ai.AI`, 요청 본문, 키 값)를 로깅하지 않는다. 기존 누출 지점 처리 선택지는 R9의 비교 표(전수 13곳 / api-manager 4곳만 / 분리). (B) 단일 지점 처리(logrus Hook 또는 `AI`에 로그용 가림)는 `models/ai/main.go:67`의 JSON 태그가 서비스 간 RPC 직렬화에도 쓰여 `json:"-"`로 바꾸면 키 전달이 끊기므로 그 방법은 불가 | **권장(제안안, CEO 승인 대기): 항상 키가 실리는 13곳(api-manager 4, ai-manager 8, call-manager 1)을 이번 PR에서 구조체·키 대신 id 필드만 남기는 기계적 한 줄 수정으로 함께 처리**. 로그 메시지와 다른 필드는 유지하고, 신규 기능 테스트 외 전용 회귀 테스트는 만들지 않는다. 근거: CEO 정책 "키 로깅 금지"와의 정합성, BYOK 키는 고객 OpenRouter 계정 과금과 직결. 이전 안(별도 이슈 분리)은 철회 권장. **오류 경로(R9 B, 최소 10곳, `v1_ais.go` `%v` 패턴과 notifyhandler/requesthandler publish 포함)는 키가 실리는지 미실증이므로 이번 PR 선택 항목**. **출시 전 확인 조건(제안)**: (1) 운영 로그 레벨과 포매터 실증(§8 항목 8: `DebugLevel`이 실제 적용되는지, `joonix` 출력에 `engine_key`가 포함되는지), (2) **이미 쌓인 로그의 보관·폐기 여부는 대표님이 판단**(기존 OpenAI/Gemini/Grok 키가 이미 기록되었을 수 있고, 이는 키 교체 권고 여부와도 연결). GET 응답 노출(R9)은 별도 이슈(Q9 결합 조건 참조) |
| Q11 | 고객 노출 문서 위치: `ai_struct_ai.rst`(Custom 유형 설명, `engine_key` 필수 조건, ZDR 안내, 비용은 고객 계정에 청구), `GET /ai_models` RST(`key_mode`), `skill.md`. 자체 호스팅 문서는 변경 없음(BYOK는 플랫폼 키 불필요). `engine_key` 응답 여부에 대한 RST/openapi.yaml(`:2196-2198`) 문구 불일치 정정 여부도 함께 결정 | BYOK 문서는 이번 PR 필수: `ai_struct_ai.rst`, `ai_models.rst`(`GET /ai_models`), `skill.md`, `llms.txt`(BYOK 언급이 해당될 때). 문구 불일치 정정은 6절 표의 "같은 PR 선택"(문구만 정정, 코드 변경 없음, 같은 PR에서 함께 권장) |

## 8. 실증이 필요한 항목 요약 (코드만으로 확정 불가)

수행자: **대표님**(고객 OpenRouter 키 또는 검증용 키 제공, 운영 로그 접근) 또는 **Claude**(코드 실행, 더미 값 사용). 시점: **디자인 전**(디자인 결정을 막는 항목), **구현 중**(테스트로 고정), **별도 이슈**(정리 이슈에서 수행). 차단 항목은 실증 전 아래 fail-closed 기본값을 적용한다.

| # | 항목 | 차단 대상 | 수행자 | 시점 | 실증 전 fail-closed 기본값 |
|---|---|---|---|---|---|
| 1 | 고객 키로 `provider.zdr:true` 호출: ZDR 제공자가 있는 모델 성공, 없는 모델 실패 상태/원문(R2) | Q5 | 대표님(키 제공) + Claude(호출) | 디자인 전 | Q5는 (a) `unknown` 유지 + 문서 안내. 분류기는 확장하지 않음 |
| 2 | 무효 키, 크레딧 없음(402)에서 러너가 보는 오류 원문과 분류 결과, 키 반사 여부(R5, R2) | Q5, R9 마지막 항목 | 대표님(키 제공) + Claude | 디자인 전 | 위와 동일. 러너 오류 원문 로깅(최대 2000자)은 이번 PR에서 유지만 하고 BYOK 키 반사 가능성을 보고 항목으로 둠 |
| 3 | 빈 키로 BYOK 호출 시 Python이 `OPENROUTER_API_KEY`가 설정된 컨테이너에서도 예외를 내는지(R1-a, R4) | R1 방어 완결 | Claude | 구현 중 | `(key or "").strip()` 빈 값은 항상 예외. 이 실증이 통과하기 전에는 머지하지 않음 |
| 4 | `GET /ais` 응답과 ai 웹훅에 `engine_key`가 실제 포함되는지(R9) | Q11 문구 정정 방향 | Claude | 구현 중 | 문구 정정은 "실증된 사실"로만 하고 미확인이면 정정하지 않음 |
| 5 | sidebar 편집 저장 시 `engine_key`가 빈 값으로 덮이는지(R1-f) | Q9 | Claude(UI 실행) | 구현 중 | A안(재조회)을 무조건 적용(실증과 무관하게 안전) |
| 6 | `openrouter.vendor/model:variant` ID가 첫 점 분리와 `extra_body` 전송까지 그대로 유지되는지(R3, 단위 + 실호출 1건) | Q4 | Claude(단위) + 대표님(실호출 키) | 구현 중 | `:` 변종은 거부 |
| 7 | **러너 실제 폴백 검증(실행)**: 러너와 동일 환경(`OPENROUTER_API_KEY`, `OPENAI_API_KEY` 주입)에서 BYOK 분기에 빈 키/공백 키/`None`을 넣어 예외가 나고 외부 요청이 나가지 않음을 확인. 단위 테스트는 `conftest.py:55-63`이 pipecat 전체를 `MagicMock`으로 대체해 SDK 실제 동작을 검증하지 못함 | R1-a | Claude | 구현 중 | 3번과 동일 |
| 8 | **로깅 레벨/포매터**: 운영에서 `DebugLevel`이 실제 적용되는지, `joonix` 출력에 `*ai.AI`의 `engine_key`가 실제 포함되는지(R9). 더미 값으로 `POST /ais` 후 로그 조회 | **R9 B 오류 경로(선택 항목)의 포함 여부와 이미 쌓인 로그의 폐기·키 교체 판단.** 13곳(A) 수정 자체는 이 실증을 기다리지 않는다(항상 실리는 지점이며 수정 비용이 작음) | 대표님(운영 로그 접근) | **출시 전 확인**(구현은 막지 않고 BYOK 출시 직전 게이트) | 누출 가능으로 간주하고 13곳 수정, 신규 코드는 무로깅. B는 실증 전 선택 항목 |
| 9 | **SDK 환경변수 폴백**: `api_key=None`으로 `OpenRouterLLMService`를 만들 때 `OPENAI_API_KEY`가 실제로 `Authorization` 헤더에 실리는지(2.3은 소스 읽기 결과) | R1-a 근거 | Claude | 구현 중 | 폴백이 일어난다고 가정하고 `None`/`""`을 SDK에 넘기지 않음 |
| 10 | **변종/라우터 ID와 ZDR 상호작용**: `:free`, `:online`, `:nitro`, `openrouter/auto` 등에서 `provider.zdr:true`가 유지되는지, 요청과 다른 제공자로 라우팅되는지(Q4) | Q4 | 대표님(키 제공) + Claude | 디자인 전 또는 후속 | 변종과 라우터형 ID는 거부 |

이 단계는 문서만 작성했으며 코드 수정, git commit/push는 하지 않았다.

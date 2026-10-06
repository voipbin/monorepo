# OpenRouter BYOK(Custom 모델과 key_mode 배지) 구현 계획

> **Status: 설계 리뷰 루프 진행 중 (plan 리뷰 1회차 CHANGES_REQUESTED 반영 완료, 재리뷰 대기).** 이 문서는 구현 계획 산출물이며 코드 수정, git commit/push는 하지 않았다. 승인(2회 연속 Approval)되면 이 줄을 `Status: APPROVED (plan 리뷰 루프 종료 회차 N, N+1)`로 갱신하고 구현 착수 전 **C0 문서 커밋**(2.2절)에 포함한다. 디자인은 디자인 리뷰 8회차에서 두 리뷰어 모두 APPROVED로 종료(디자인 커밋 `323ad8fc8`).
>
> **수정 이력:** plan 수정회차 2 (2026-10-07). 리뷰 1회차(리뷰어 17, 18 모두 CHANGES_REQUESTED) 공통 MAJOR 반영: (M-A) C2를 T2+T3+T6으로 확장해 aihandler 테스트가 빨간 중간 커밋 제거, (M-B) T15가 `sidebar_engine_model.test.js:196,257` 그룹 조회를 함께 수정, (M-C) monorepo-javascript PR 테스트 게이트와 실제 기준선(테스트 2 failed, 전체 lint 381 errors, `CI=true` 빌드 실패)을 T0, T22, T23에 반영, (M-D) 위생 grep에 `:(exclude)docs/plans`와 로케일 전제, (M-E) T5 grep을 공회전하지 않는 정규식으로 교체(수정 전 A 11, B1 6 실측), (M-F) 스트림 예외 표와 `ai_test.go` 편집 순서 고정, (M-G) T24 항목 3, 5를 `uv run --frozen --project` 방식으로 정정(실행 확인), (M-H) docsdev 빌드 절차(venv, `pygments==2.18.0` 고정, 노이즈 확인)와 RST 재수정 메모. MINOR 일괄(17 m1~m9, 18 m2~m7, m9, m11) 반영.
>
> **수정 이력(회차 3):** plan 수정회차 3 (2026-10-07). 리뷰 2회차(리뷰어 19 CHANGES_REQUESTED MAJOR 1건, 리뷰어 20 APPROVED) 반영: (R19-M1) sidebar 테스트 캡션 단정의 폼별 귀속 정정(`:184,200`은 편집 폼이라 T20, `:260`만 T19)과 J2는 한 커밋으로 확정(T19, T20은 작업 순서), (R19 m1) `ModelPicker.test.js:168` 추가와 디자인 5.1 정오표(F20), (R19 m2, m3, m5) T14 Verify 시점, 라인 정정, C2 스트림 예외, (R20 m1~m8) effect deps 문구, T18 `formSynced` 설명 정정, T23-5 CI/로컬 구분과 `check-service-docs` 처리 방침, 비차단 게이트 명시, OpenAPI 설명 보강, T5 `send.go:170` 확정, T24 §8 보강, T11 줄 수와 Python 로그 수용.
>
> **입력 문서(같은 디렉터리):** `2026-10-06-openrouter-byok-design.md`(이하 "디자인", 절 번호 참조), `2026-10-06-openrouter-byok-issue-analysis.md`, `2026-10-06-openrouter-byok-ui-mockup.png`, 형식 선례 `2026-10-05-openrouter-llm-routing-plan.md`.
>
> **표기:** `파일:라인`은 이번 작성 중 두 워크트리의 현재 코드를 직접 읽고 확인한 값이다(monorepo 현 HEAD `323ad8fc8`(디자인 커밋, 부모 `bc68e3c32` = origin/main), monorepo-javascript HEAD `54aa7281` 기준). 구현 시 한 줄 정도 밀릴 수 있으므로 **편집 직전에 해당 줄을 다시 열어 확인**한다. 고객 노출 문자열과 코드 주석 예시는 영어, 설명은 한국어다. 키는 어디에도 쓰지 않으며 더미 문자열만 쓴다(`dummy-old-key`, `dummy-new-key`, `dummy-typed-key`, `dummy-gemini-typed-key`, 모델 ID 붙여넣기 방어는 `dummy-key-not-real`).

**Goal:** 모델 선택기에서 키 방식(`platform` / `own_or_default` / `own_required`)을 그룹과 배지로 구분하고, 고객이 본인 OpenRouter 키로 임의의 OpenRouter 모델을 호출하는 Custom(BYOK) 모델을 제공한다. 모든 Custom 호출은 요청 단위 ZDR을 강제하고, 플랫폼 `OPENROUTER_API_KEY`와 `OPENAI_API_KEY`에는 어떤 경로로도 닿지 않는다(Go 서버 검증, Go resolver, Python 러너 3중 방어).

**Architecture:** 정적 카탈로그에 `Route=RouteCustomOpenRouter` 항목 1개(`custom.openrouter`, 접두 `openrouter.`)를 추가하고 `ResolveEngine`이 `openrouter.<author>/<slug>`를 `OutcomeCustomOpenRouter`(`RequireKey`)로 통과시킨다. 서버는 최종 (모델, 키) 상태를 `ValidateEngine`으로 검증한다. pipecat-manager Go는 키 없는 사전 분류(`classifySessionLLM`)와 키 필수 해석(`resolveSessionLLM`)을 분리하고, Python 러너는 `openrouter` 서비스 분기를 고객 키만으로 만든다. UI는 `ModelPicker` 내부에 키 방식 그룹, 배지, 모델 ID 입력란을 두어 4개 폼에 자동 적용하고, 키 이월 가드와 전이 비움 규칙을 폼별로 구현한다.

**Tech Stack:** Go(monorepo, 서비스별 vendor는 로컬 산출물), Python 3.11+/pipecat-ai 1.12.x, OpenAPI/oapi-codegen, Sphinx RST, React 18(square-admin, Jest/react-scripts, `jest.mock`).

**Repos and PRs (레포당 PR 1개, 추가 분할 없음):** `monorepo`(백엔드, 러너, 문서) 1개, `monorepo-javascript`(UI, skill.md, llms.txt) 1개. 둘 다 브랜치 `NOJIRA-Add-custom-OpenRouter-BYOK-models`, 이미 만들어진 워크트리를 쓴다. `voipbin/voipbin` 자체 호스팅 설치기와 `monorepo-etc`는 변경 없음(BYOK는 플랫폼 키가 필요 없다, 디자인 1.2).

---

## 0. 코드 재확인 결과와 디자인 대비 달라진 사실

디자인의 라인과 서술을 현재 코드로 다시 확인했다. 대부분 일치하며, **달라졌거나 디자인에 없던 사실**은 아래와 같다(최종 보고용 목록이기도 하다). 태스크 본문에 모두 반영했다.

| # | 사실 | 반영 |
|---|---|---|
| F1 | `ModelInfo`에 필드가 추가되면 **api-manager `server/ai_models_test.go`의 기대 JSON 문자열(`:53` 부근, `expectedRes` 문자열)이 깨지고**, common-handler `ai_ais_test.go:97-160`(wire shape 테스트)도 보강이 필요하다. 디자인 3.2는 로직 변경이 없다고만 적었다 | T1에 같은 커밋 수정 포함 |
| F20 | **디자인 5.1 정오표(디자인은 수정하지 않고 plan에만 기재, 구현 시 T13 정오표에 함께 옮길 수 있음):** `ModelPicker.test.js:168`의 `has no Custom entry`(`screen.queryByText(/custom/i)`가 없어야 함)는 T14가 Custom 픽스처를, T15가 `CUSTOM` 그룹 헤더와 Custom 행을 만드는 순간 빨개진다. 디자인 5.1 표의 `ModelPicker.test.js` 행(`:60-85`, `:87-100,132`, `:109-115`)에 이 줄이 빠져 있다(직접 확인: 실제로 `queryByText(/custom/i)` 단정, 코드 `ModelPicker.test.js:168-173`). 의미 변경: Custom 행이 `CUSTOM` 그룹에 있음을 단정하는 테스트로 바꾼다 | T15 |
| F2 | `ResolveEngine` Custom 분기를 넣는 순간 pipecat-manager 테스트(`llmresolve_test.go:41,66,130-131,182,216`, `run_teamllmtype_test.go:60`)가 빨개진다(`openrouter.*`가 더는 `Rejected`가 아님). 서로 다른 모듈의 같은 변경 단위. **또한 같은 모듈의 aihandler 테스트도 빨개진다**: `Create :51`이 `IsValidEngineModel`을 쓰므로 Custom ID가 통과해 `chatbot_engine_model_test.go:27` `rejects_raw_openrouter`(wantError true)가 실패한다. 격리 복사본에서 T2 효과만 적용해 실제 실행해 확인: `models/ai`의 `TestIsValidEngineModel/raw_openrouter_is_invalid`, `Test_ResolveEngine/raw_openrouter`(T2가 같은 커밋에서 수정)와 `pkg/aihandler`의 `Test_Create_EngineModelPolicy/rejects_raw_openrouter`가 실패하고 `cmd/ai-control`은 통과 | **T2, T3, T6은 한 커밋 C2**(컴파일 순서 3절, 2.2절). T3이 들어가야 "Custom 모델 + 빈 키"를 서버가 거부해 중간 커밋에도 키 없는 Custom 저장 경로가 열리지 않는다 |
| F3 | `bin-ai-manager/docs/domain.md:12,111-127`, `bin-pipecat-manager/docs/domain.md:78-79`, `ai_tutorial.rst:14`, `ai_overview.rst:821`과 `ai_tutorial.rst:437`(둘 다 `INVALID_ENGINE_MODEL` 원인을 "`GET /ai_models` 값 또는 `openai`/`gemini`/`grok` 접두"로만 서술, `openrouter.<model-id>`와 `ENGINE_KEY_REQUIRED` 누락), `bin-openapi-manager/openapi/paths/ai_models/main.yaml:6`, `ai_models.rst` 응답 예시와 필드 목록, `llms.txt:50`, `skill.md:819`(응답 예시)가 `platform_managed`/카탈로그를 서술하고 있어 갱신 대상이다. 디자인 3.5 목록에 없었다 | T13, T21 |
| F4 | `bin-api-manager/docsdev/build/`는 **git 추적 중(838 파일)**이다. `CLAUDE.md`(api-manager)는 `rm -rf build && python3 -m sphinx -M html source build` 후 `git add -f`를 요구한다. 디자인 3.5는 "추적 여부 미확인"이었다 | T13에서 해결 |
| F5 | `v1_ais.go:66`은 `Debugf`, `:142,181,220,316`은 `Errorf`다(B2, 모두 `message: %v, err: %v`) | T5 |
| F6 | UI 테스트 러너는 vitest가 아니라 **Jest(react-scripts)**다(`square-admin/package.json:19` `"test": "react-scripts test"`, 이미 `jest.mock` 사용). 워크트리에 `node_modules`가 없다 | 모든 UI 검증 명령 |
| F7 | CRA는 `__tests__/` 안의 모든 `.js`를 테스트로 실행하므로 공유 행렬 파일을 거기에 두면 "테스트 없음"으로 실패한다 | 행렬은 `views/ais/keyGuardMatrix.fixture.js`(`aiModelsFixture.js`와 같은 위치) |
| F8 | Python은 `pyproject.toml`에 pytest가 없고 기본 python에도 없다. 기준선은 `uv run --no-project --with pytest --with pytest-asyncio --with loguru --with pydantic --with pillow python -m pytest -q`로 실행되며 **기존 실패 4건(`test_missing_tool_logging.py`, ModuleNotFoundError)이 이미 있다**(250 passed, 4 failed, 2 skipped) | T7 비교 기준선 |
| F9 | `teamgraph/nodes/member.js` `getProviderStyle`은 `openrouter.<id>`를 `openrouter` 접두의 폴백(`OP` 라벨, muted 색)으로 처리한다. 오류 없음(디자인 8.2 확인 항목 해소) | T22에서 확인만 |
| F10 | `provider.js`의 오류 메시지는 `:146`의 `new Error(`Request failed (${response.status}): ${response.statusText}. Details: ${errorBody}`)`에서 만들어진다(디자인은 `:146-148`과 일치, 이 plan 초안의 `:143`은 오기). `teams_create.js`의 `indexed[ai.id] = ai`는 `:50,72`(디자인 `:51,73`) | 라인 정정 |
| F11 | `start.go`의 `"ai"` 로그 4곳은 서로 다른 함수 4개(`startReferenceTypeCall`/`Conversation`/`ContactCase`/`None`)이며 모두 `Start()`의 `resolveAI` 성공 결과를 받는다(non-nil). 로그 직후 `a`를 역참조하는 코드가 이미 있어 `a.ID` 전환이 새 nil 위험을 만들지 않으나, **단위 테스트가 nil `*ai.AI`를 넘기면 패닉**하므로 그 경우 테스트 데이터를 고친다 | T5 |
| F12 | `servicehandler.AIModelList`는 `ai.go:278`(디자인 `:276-291`), `AICreate`는 `:36`, 로그 줄은 `:64,66,121,401,403,465` | T9 |
| F13 | pipecat `startReferenceTypeCall`은 `start.go:97`에서 `runGetLLMKey`(`run.go:121-142`)를 거쳐 resolver에 도달한다. Custom + `ReferenceTypeCall`은 키 조회가 없어 항상 거부된다(설계 의도, 기존 `Test_startReferenceTypeCall_rejectedModel` `:168`이 이 경로) | T6 |
| F14 | `scripts/check-test-conventions.sh`는 **추가된 줄만** 검사하고(규칙: 테스트 이름 `Test_` 접두, `reflect.DeepEqual` + `t.Errorf`, gomock 컨트롤러 변수명 `mc`) fail-closed다. 기존 `TestIsValidEngineModel`(`models/ai/main_test.go:324`)처럼 `Test_` 없는 기존 이름은 건드리지 않는 한 통과 | 모든 Go 테스트 |
| F15 | `AIEngineFields.js` 상단 주석은 `create_ai_dialog.js`를 소비자로 적지만 실제 소비자는 `ais_create.js:545`, `ais_detail.js:1264` 둘뿐이다(grep). sidebar는 `ModelPicker`를 직접 쓴다(`:945,1302`) | T16 주석 정정 |
| F16 | detail 키 입력란은 **비제어(`defaultValue`, `ais_detail.js:1305`) + `ref_engine_key`**다. 로드 전(`isLoading` 스켈레톤)에는 입력란이 마운트되지 않아 `ref.current`가 `null`이다 | T18 |
| F17 | 플랜 수준의 구현 결정 3개(디자인 규칙은 그대로, 같은 파일의 순수 함수만 추가): `keyTransitionAction(prev, cur)`, `computeGuardApplies(...)`, `computeKeyMode(models, value)`를 `useAIModels.js`에 두고 create/detail/sidebar가 공유한다(디자인 3.6.1 "같은 파일의 순수 함수, 새 파일 없음"). `prevCreateKeyModeRef`는 `null`로 시작해 카탈로그 `ready` 이후 첫 실행은 기록만 한다 | T14, T17 |
| F18 | UI 기준선은 깨끗하지 않다(직접 실행, HEAD `54aa7281`, 변경 없는 트리): 전체 테스트 `Tests: 2 failed, 1 skipped, 3027 passed, 3030 total`(실패는 `src/views/mcpservers/__tests__/mcpServerCatalog.test.js` 2건), 전체 `npm run lint` 381 errors 167 warnings, `CI=true npm run build` 실패("Treating warnings as errors"), CI=true 없는 `npm run build`는 성공. 실제 CI(`.circleci/config_work.yml` square-admin 이미지 빌드, `square-admin/Dockerfile:13`)는 `CI=true` 없이 `npm run build`만 실행하고 lint/test를 돌리지 않는다 | T0, 1절 UI 검증, T22, T23-8 |
| F19 | `docsdev/build` 추적 파일 838개 = html 246 + `_sources` 244 + `_static` 67 + `_images` 33 + `doctrees` 245 + 기타. 변경 없는 재빌드는 환경에 따라 전 파일이 바뀐다(최신 pygments 2.21.0이면 `pygments.css` 해시와 코드 하이라이트로 html 180여 개 변경). **`pygments==2.18.0`로 고정하면 html은 커밋본과 바이트 동일**(직접 확인, Sphinx 9.1.0, Python 3.12.3과 3.13.7 모두)하고 `doctrees` 245개(`environment.pickle` 포함)만 바뀐다. 이전 기능 커밋 `bc68e3c32`도 doctrees 245개를 전부 포함해 커밋했다 | T13 |
| F20 | Python 러너 실행은 `uv run --no-project --with pipecat-ai[openrouter]`로는 `run.py` import가 실패한다(직접 확인: `google.genai` 누락). `UV_PROJECT_ENVIRONMENT`를 저장소 밖으로 지정한 `uv run --frozen --project .`는 import와 프로브가 동작한다(직접 확인: 116 패키지, 3초, `.venv` 미생성, `git status` 깨끗) | T24 |

디자인의 그 밖의 코드 사실(`llmresolve.go:17-28`, `pythonrunner.go:87` `omitempty`, `run.py:571-689`, `sidebar.js:488,618-644,807-808`, `ais_create.js:201`, `ais_detail.js:422,541`, `webhook.go:33,126`, `publish.go:34` 등)은 모두 재확인했고 일치한다.

---

## 1. 공통 규칙

**모든 태스크 공통:** 워크트리에서만 작업한다. 테스트는 구현과 같은 태스크에서 먼저(실패 확인 후) 작성한다. 전역 플래그, 환경변수, 설정 추가 금지. 키, 모델 ID 원문(Custom 입력), `*ai.AI`, 요청 본문을 로깅하지 않는다(신규 로그 줄은 식별자만). 코드에 키 형태 문자열을 넣지 않는다. 커밋 메시지는 monorepo 형식(요약 72자 이하 + `- project-name: 변경` 불릿, AI 표기 금지), 작성자 `Sungtae Kim <pchero21@gmail.com>`. **이 계획서를 쓰는 작업은 커밋하지 않지만, 구현 단계의 커밋은 아래 "커밋 단위"를 따른다.**

**Go 서비스 검증(해당 서비스 코드나 vendored `bin-ai-manager`/`bin-common-handler` 사본이 바뀐 모든 서비스):**
```
cd <service> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
`vendor/`는 gitignore이며 커밋하지 않는다. 다른 모듈에 새로 추가된 심볼을 쓰는 서비스(pipecat, api-manager, common-handler)는 **테스트 전에** `go mod tidy && go mod vendor`를 먼저 돌린다(이전 plan Task 8, 14, 15의 교훈). 기준선은 `go test ./models/ai/ -count=1`이 vendor 없이도 통과한다(확인함).

**Go 검증 후 점검과 CI 차이:** 위 명령 뒤에 `git status --short`로 의도하지 않은 `go.mod`, `go.sum`, `gens/` 변경이 없는지 확인한다(`go mod tidy`가 다른 모듈의 버전 줄을 건드릴 수 있음). 실제 CI(`.circleci/config_work.yml:1569`, `:2316`)는 서비스별로 `go mod vendor`만 실행하고 `go mod tidy`와 서비스별 `go generate`는 실행하지 않는다(예외: `bin-openapi-manager-validate` 잡이 `go generate ./...` 후 `gens/models/gen.go`의 diff를 검사, `:1566-1585`). CI의 golangci-lint는 **2.14.0**(`:2291`)이고 이 머신은 **2.13.2**이므로 로컬 통과가 CI 통과를 보장하지 않는다(규칙 차이는 PR의 CI 결과로 확인).

**Python 검증(`bin-pipecat-manager/scripts/pipecat`에서):**
```
uv run --no-project --with pytest --with pytest-asyncio --with loguru --with pydantic --with pillow python -m pytest -q
```
기준선 250 passed, 4 failed(`test_missing_tool_logging.py`, 이 작업과 무관한 기존 실패), 2 skipped. 변경 후 **새 실패가 없고 passed가 늘어야** 한다.

**UI 검증(`monorepo-javascript` 워크트리의 `square-admin`에서, Jest):** `node_modules`가 없으므로 최초 1회 `npm ci`(약 11초, 로컬 node v26.7.0, `Dockerfile`은 `node:20-slim`이라 CI와 노드 버전이 다름). 개별 파일 `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/ModelPicker.test.js`. 테스트 도구는 `@testing-library/react`, `userEvent`, `jest.mock('src/provider', ...)` 패턴(vitest 아님, `vi.*` 금지). **PR 전 게이트는 `monorepo-javascript/CLAUDE.md:25-45`를 그대로 따른다**(F18의 기준선을 직접 실행해 확인함):
1. **전체 테스트 기준선 비교:** `git stash`(변경이 있을 때) 후 `npm test -- --watchAll=false --forceExit 2>&1 | grep "Tests:"`를 기준선과 변경 후에 각각 실행해 `Tests:` 줄을 비교한다. 기준선(깨끗한 HEAD `54aa7281`, 실제 실행): `Tests: 2 failed, 1 skipped, 3027 passed, 3030 total`, `Test Suites: 1 failed, 267 passed, 268 total`. 기존 실패는 `src/views/mcpservers/__tests__/mcpServerCatalog.test.js`의 `every entry has the required fields`, `has no duplicate urls` 2건(이 PR 무관). **게이트:** 실패 건수와 실패 suite가 기준선과 같거나 적고, passed는 신규 테스트만큼 늘어난다. PR 본문에 "Known pre-existing failures: 1 suite (mcpServerCatalog.test.js, 2 tests) - confirmed failing on main baseline"을 적는다.
2. **린트는 변경 파일 전후 비교:** 전체 `npm run lint`는 기준선 **381 errors, 167 warnings**(548 problems, 377 errors는 `--fix` 가능)라 통과 게이트로 쓸 수 없다. `npx eslint <변경 파일>`의 파일별 건수가 기준선보다 늘지 않아야 한다. 작업 대상 기준선(직접 실행): 소스 7개(`ModelPicker.js`, `AIEngineFields.js`, `ais_create.js`, `ais_detail.js`, `useAIModels.js`, `aiModelsFixture.js`, `sidebar.js`)는 0 errors 3 warnings(`ais_detail.js` 1, `sidebar.js` 2, 모두 `no-unused-vars`), 테스트 8개(`ModelPicker.test.js`, `AIEngineFields.test.js`, `useAIModels.test.js`, `ais_create.test.js`, `ais_detail.test.js`, `sidebar_engine_model.test.js`, `sidebar_ai_type.test.js`, `sidebar.test.js`)는 15 errors 0 warnings(`ais_detail.test.js` 4, `sidebar.test.js` 3, `sidebar_ai_type.test.js` 3, `sidebar_engine_model.test.js` 5). 새 파일 `keyGuardMatrix.fixture.js`와 새로 쓴 코드는 0 errors 0 warnings.
3. **빌드는 `CI=true` 없이:** 실제 CI(`square-admin/Dockerfile:13` `RUN npm run build`)와 같게 `npm run build`를 실행한다(`prebuild`가 `npm run build:widget`을 먼저 실행하며 기준선에서 exit 0, 빌드 후 `git status --short` 깨끗, `build/`는 gitignore). `CI=true npm run build`는 **기준선에서 이미 실패**("Treating warnings as errors because process.env.CI = true." 후 "Failed to compile.")하므로 이 PR의 게이트로 쓰지 않는다.

**이전 plan에서 얻은 순서/컴파일 실패 교훈(이 plan에 반영):**
1. 상수/심볼을 쓰는 코드보다 정의가 **앞선 태스크**에 있어야 한다. 그래서 `EngineModelPrefixCustomOpenRouter`와 `RouteCustomOpenRouter`, `KeyMode*`는 T1(카탈로그)에서 정의하고 T2(resolver)가 쓴다.
2. 의미를 바꾸는 코드는 **그 의미에 기대던 기존 테스트를 같은 커밋에서** 고친다(Custom 항목 추가 시 `resolve_test.go:68-79`, `Test_CatalogInvariants`; resolver 변경 시 **같은 모듈의 aihandler 테스트 `chatbot_engine_model_test.go:27`과 pipecat 테스트**, 그래서 C2는 T2+T3+T6; UI 그룹 변경 시 `sidebar_engine_model.test.js:196,257`, 그래서 T15가 이 두 줄을 수정). **각 커밋 경계에서 그 시점 트리로 영향받는 모듈의 전체 테스트(`go test ./...`, `npm test`)가 초록이어야 한다**(bisect 가능, 중간 커밋이 빨간 채로 남지 않음). 그래서 T2, T15의 Verify에 모듈 전체 테스트가 들어 있다.
3. 서로 인접한 같은 타입 인자(문자열 둘)는 순서 뒤바뀜을 테스트로 고정한다(여기서는 `ValidateEngine(model, key, modelChanged)`의 `model`/`key`).
4. 생성 파일(`gens/`, mock)은 손으로 고치지 않고 생성기로만 갱신해 같은 PR에 포함한다(T12, T23).
5. 중복 심볼 방지: 새 이름은 착수 전에 `grep`으로 전 모듈에 없음을 확인했다(확인 결과 모두 없음): Go `KeyMode*`, `ValidateEngine`, `ValidateCustomModelID`, `ErrEngineKeyRequired`, `ErrInvalidEngineModel`, `RequireKey`, `classifySessionLLM`, `loggableEngineModel`; Python `_OPENROUTER_PROVIDER`, `_build_openrouter_llm`; JS `keyModeOf`, `servicePrefixOf`, `customEntry`, `customIdMissing`, `engineErrorOf`, `KEY_MODE_BADGES`, `guardApplies`, `prevGuardRef`. 각 태스크 착수 시 같은 grep을 다시 돌려 충돌이 없는지 확인한다.

### 1.1 비차단 MINOR 흡수 위치

| MINOR | 흡수 위치 |
|---|---|
| (1) 러너는 Jest | 1절 UI 검증 명령, T14~T22 |
| (2) 전이 effect는 폼 초기화 이후에만, detail은 `isLoading` 조기 반환 앞 + ref null 가드 | T18, T20 |
| (3) create 비움은 effect로만 | T17, T19 |
| (4) 행렬 `test.each`, 행 14/18/20 정리, 복원 행 추가, 행 19 `only` | 4절, T18, T20 |
| (5) detail 테스트 mock 전환 버튼, `resetAIModelsCache`/fixture 정합 | T14, T18 |
| (6)(7) 수용 재존 문서 | T13(디자인 문서 8.1, 8.3 정오표) |
| (8) B2 `v1_ais.go:66`은 Debugf | T5 |
| (9) 분기 조건 있는 태스크 표기 | 6절 |

---

## 2. 스트림 구성(4스트림)과 소유 경로

같은 워크트리에서 병렬로 일하는 스트림이 서로 다른 경로만 건드리도록 소유 경로를 고정한다. 커밋은 **자기 경로만 `git add <경로>`**로 하고, 한 번에 한 스트림만 커밋한다. **단 하나의 예외: C2(T2+T3+T6)는 S1(`bin-ai-manager/`)과 S2(`bin-pipecat-manager/`) 경로를 한 커밋에 담는다**(아래 예외 표 첫 행. 두 모듈의 테스트가 같은 `ResolveEngine` 변경에 묶여 있어 나누면 중간 커밋이 빨갛다).

| 스트림 | 레포 | 태스크 | 소유 경로 |
|---|---|---|---|
| **S1 ai-manager 코어** | monorepo | T1, T2, T3, T4, T5 | `bin-ai-manager/` (T1의 F1 테스트 수정은 아래 예외 표의 3개 파일) |
| **S2 pipecat** | monorepo | T6, T7, T8 | `bin-pipecat-manager/` |
| **S3 API 체인, 로깅, 문서** | monorepo | T9, T10, T11, T12, T13 | `bin-api-manager/`, `bin-call-manager/`, `bin-webhook-manager/`, `bin-common-handler/pkg/notifyhandler/`, `bin-openapi-manager/`, `docs/plans/*byok-design.md`(T13 정오표), T13이 쓰는 `bin-ai-manager/docs/domain.md`, `bin-pipecat-manager/docs/domain.md`(아래 예외 표) |
| **S4 UI** | monorepo-javascript | T14~T22 | `square-admin/`, `square-main/public/` |

**소유 경로 예외 표(다른 스트림 경로를 건드리는 모든 경우, 직접 `grep`으로 확인한 같은 파일 중복 편집 포함):**

| 태스크(스트림) | 건드리는 경로(소유 스트림) | 고정 규칙 |
|---|---|---|
| **T2, T3 (S1) + T6 (S2)** | C2 한 커밋이 `bin-ai-manager/`(S1)와 `bin-pipecat-manager/`(S2)를 모두 포함 | **예외 규칙:** 두 스트림이 각자 작업을 끝낸 뒤 한 사람이 두 경로를 함께 `git add`해 C2 하나로 커밋한다. 커밋 직전에 `bin-ai-manager`, `bin-pipecat-manager`의 `go test ./...`가 모두 초록임을 확인한다. S2는 C2 전에 자기 경로를 단독 커밋하지 않는다 |
| T1 (S1) | `bin-api-manager/server/ai_models_test.go`(S3), `bin-api-manager/pkg/servicehandler/ai_test.go`(S3, `:221-256`의 `ModelInfo` 행에 Custom 행 추가), `bin-common-handler/pkg/requesthandler/ai_ais_test.go`(S3, `:97-159`) | 세 파일 모두 **C1에 포함**, 파일마다 변경은 해당 테스트 행 추가뿐 |
| T9 (S3) | `bin-api-manager/pkg/servicehandler/ai.go`(S3 전용)와 **`ai_test.go`(T1과 같은 파일)** | T9의 `ai_test.go` 수정(`Test_loggableEngineModel` 추가)은 **C1 커밋 이후에만** 착수한다. 두 스트림이 같은 파일을 동시에 편집하지 않고, 불가피하면 `git add -p`로 헝크를 분리해 C1에는 T1의 행만, C6에는 T9의 함수만 담는다 |
| T13 (S3) | `bin-ai-manager/docs/domain.md`(S1), `bin-pipecat-manager/docs/domain.md`(S2) | S1이 C2~C4를, S2가 C2와 C5를 커밋한 **뒤에** T13이 수정한다(그 두 파일은 T13이 유일한 편집자, T1~T8은 `docs/domain.md`를 건드리지 않음) |

통합 단계(T23~T27)는 스트림 종료 후 한 사람이 수행한다.

### 2.1 의존 그래프와 동기화 지점

```
T0(착수 점검)
 |-- T1 카탈로그 -----------+--> T12 OpenAPI/gens (ModelInfo 필드 필요)
 |                          +--> T2+T3+T6 (resolver + aihandler + pipecat Go, 한 커밋 C2)
 |-- T5 ai-manager 로깅 (T1과 독립, 먼저 해도 됨)
 T2 --> T3 aihandler (C2, T2와 한 커밋) ; T4 ai-control (C3, T2 이후 별도 커밋)
 T6 --> T7 Python 러너 (독립 파일, T6와 동시 가능) --> T8 verify 스크립트 불변 확인
 T9, T10, T11 (T1~T2와 독립, 언제든)
 T12 --> T13 문서(openapi 필드 설명이 확정된 뒤)
 T14 --> T15 --> T16 --> {T17, T18, (T19 --> T20)} --> T21 --> T22   (S4는 Go와 독립, fixture로 개발; **T19와 T20은 둘 다 `sidebar.js`와 `sidebar_engine_model.test.js`를 고치므로 작업 순서는 직렬(T19 먼저, 그 다음 T20, 한 사람이 순서대로), 커밋은 J2 하나. T17, T18과는 병렬 가능**)
 {S1..S4 완료} --> T23 통합 검증 --> (T24 조건부 실호출) --> T25 변이 --> T26 조건부 후속 --> T27 리뷰/PR
```

**동기화 지점:** B0 = T1 완료(S3의 T12 착수 가능). B1 = **T2, T3, T6 동시 완료 후 한 커밋 C2**(그 전에는 aihandler와 pipecat 테스트가 빨간 것이 정상이므로 S1, S2는 해당 테스트를 작성만 하고 커밋하지 않거나, S1이 T2, T3을 로컬에서 먼저 준비해 둔다). B2 = S1, S2, S3 완료 후 T23.

### 2.2 커밋 단위

| 커밋 | 포함 태스크 | 요약 줄 예 |
|---|---|---|
| C0 | (문서) 이슈 분석서, 디자인, **이 plan**(`docs/plans/2026-10-06-openrouter-byok-*`) | `Add custom OpenRouter BYOK plan`. plan 승인 후 구현 착수 전에 커밋(디자인 커밋 `323ad8fc8`에는 plan이 없다, 현재 untracked). 헤더 `Status`를 `APPROVED`로 갱신한 뒤 커밋하고, 이후 plan 수정은 코드 커밋과 섞지 않는다 |
| C1 | T1 | `Add custom OpenRouter catalog entry and key_mode` |
| C2 | **T2 + T3 + T6** | `Resolve custom OpenRouter models fail-closed end to end` (ai-manager resolver, aihandler 검증, pipecat Go. 이 커밋 직후 `bin-ai-manager`와 `bin-pipecat-manager`의 `go test ./...`가 모두 초록이어야 한다) |
| C3 | T4 | `Validate engine key in ai-control for custom OpenRouter models` (ai-control 테스트는 C2 직후에도 초록임을 확인했으므로 별도 커밋 가능) |
| C4 | T5 | `Stop logging engine keys in ai-manager` |
| C5 | T7, T8 | `Add custom OpenRouter runner branch` |
| C6 | T9, T10, T11(채택 시) | `Stop logging engine keys in api, call and webhook paths` |
| C7 | T12 | `Add key_mode to ai model API schema` |
| C8 | T13 | `Document custom OpenRouter models` |
| (JS) J1~J3 | T14~T16 / T17~T20 / T21~T22 | 태스크 묶음별. **J2(T17~T20)는 하나의 커밋으로 확정**(T19→T20은 같은 `sidebar.js`와 `sidebar_engine_model.test.js`를 고치는 **작업 순서**일 뿐 중간 커밋이 아니다. 그래도 T19 종료 시점의 작업 트리가 초록이 되도록 테스트 귀속을 맞췄다: 생성 폼 캡션 단정 `:260`은 T19, 편집 폼 캡션 단정 `:184,200`은 T20). **J1은 `sidebar_engine_model.test.js:196,257` 그룹 조회 수정을 포함**(T15), J1 직후 `src/views/ais`, `src/views/teamgraph` 테스트가 초록이어야 함(J1과 J2를 합치는 안보다 2줄 이동이 더 작은 변경이라 채택). **J1에는 `ModelPicker.test.js:168` 의미 변경도 포함**(F20) |

---

## 3. 컴파일 순서 요약(Go)

1. `models/ai/catalog.go`(T1): `RouteCustomOpenRouter`, `KeyMode*`, `EngineModelPrefixCustomOpenRouter`, `ModelEntry.CustomPrefix`, `ModelInfo.KeyMode/ModelIDPrefix`, `CatalogView()`, Custom 항목, `ResolveEngine` 정확 일치 루프의 Custom 건너뛰기. 이 시점의 `ResolveEngine("openrouter.x/y")`는 **여전히 `Rejected`**라 기존 resolver 의미가 안 바뀐다(pipecat 테스트 그대로 통과).
2. `models/ai/resolve.go`(T2): `OutcomeCustomOpenRouter`는 **`OutcomeDirectPassthrough` 뒤에** 추가(iota 값 안정), `Resolved.RequireKey`, `ValidateCustomModelID`, `ValidateEngine`, sentinel 2개. `Resolved{...}` 리터럴은 `resolve.go`의 4곳뿐이고 모두 키 있는 리터럴이라 필드 추가가 안전하다(grep 확인).
3. T6(pipecat): `classifySessionLLM`, `resolveSessionLLM` Custom 처리. T2, T3과 한 커밋(C2).
4. `pkg/aihandler`(T3): `ai.ValidateEngine`을 쓴다(T2 이후, C2에 포함). `cmd/ai-control`(T4)은 aihandler와 같은 함수를 호출하며 C3(별도)이다.
5. `bin-common-handler`의 `ModelInfo` 소비(`requesthandler`)와 api-manager는 `ai.ModelInfo`가 바뀐 뒤 재vendor해야 새 필드가 보인다(T1 이후, T12 이전에 각 서비스에서 `go mod tidy && go mod vendor`).
6. `bin-openapi-manager` `go generate` -> `bin-api-manager` `go generate` 순서(T12).

---

## 4. 키 이월 가드 공통 행렬(UI 공유 테스트 데이터, T18과 T20이 같은 표를 쓴다)

파일: `square-admin/src/views/ais/keyGuardMatrix.fixture.js`(F7). 열: `row`(문자열), `only`(`'detail'|'sidebar'|undefined`), `saved`(`{model, key}`), `steps`(전환/입력 시퀀스), `expect`(`canSave`, `body.engine_key`, `inputValue`, `warning`). 목적지 모델 상수는 같은 파일에서 내보낸다: `OPENAI = 'openai.gpt-5'`, `OPENAI_ALT = 'openai.gpt-5-mini'`(픽스처에 이미 있음), `GEMINI = 'gemini.gemini-2.5-pro'`, `GROK = 'grok.grok-3'`, `PLATFORM = 'anthropic.claude-sonnet-4.5'`, `PLATFORM_ALT = 'meta.llama-3.3-70b-instruct'`, `CUSTOM = 'openrouter.vendor/model-a'`, `CUSTOM_ALT = 'openrouter.vendor/model-b'`. 두 테스트 파일은 `test.each(KEY_GUARD_ROWS.filter(r => !r.only || r.only === 'detail'))`처럼 쓰고, 각자 `driver`(저장 상태 세팅, 전환, 타이핑, 저장, 단정)만 구현한다.

디자인 5.2 행렬을 다음처럼 **정리**했다(MINOR 4). 행 14, 18, 20은 다중 시나리오를 행으로 분해하고 중복을 제거했으며, 행 21(저장 Custom -> platform -> Custom 복원)을 추가하고, 행 19를 `only`로 나눴다.

| row | only | 저장(모델, 키) | 시나리오 | 기대 |
|---|---|---|---|---|
| 1 | | platform(`PLATFORM`), `dummy-old-key` | `CUSTOM`으로 전환, 키란 안 건드림 | 저장 불가(경고 `Replace the API key before saving.`), `ProviderPut` 미호출. detail은 전이 시 입력란이 비워지고 sidebar는 `engineKeyChanged=false` |
| 2 | sidebar | 1과 같음 | 키란에 포커스만 | 저장 불가(포커스는 새 입력이 아님) |
| 3 | | 1과 같음 | `dummy-new-key` 입력 | 저장 가능, `body.engine_key === 'dummy-new-key'`, body 어디에도 `dummy-old-key` 없음 |
| 4 | | 1과 같음 | `GEMINI`로 전환, 입력 없음 | 저장 가능, `body.engine_key === ''` |
| 5a/5b | | own_or_default(`OPENAI`), `dummy-old-key` | `CUSTOM`으로 전환: (a) 입력 없음 (b) `dummy-new-key` | (a) 저장 불가 (b) 저장 가능, 새 키만 |
| 6 | | own_required(`CUSTOM`), `dummy-old-key` | `OPENAI`로 전환, 입력 없음 | 저장 가능, `body.engine_key === ''` |
| 7 | | own_required(`CUSTOM`), `dummy-old-key` | `PLATFORM`으로 전환 | 저장 가능. detail은 저장 키 재전송(`ais_detail.test.js:435-448` 유지), sidebar는 `engine_key` 미전송(`sidebar_engine_model.test.js:206` 유지) |
| 8a/8b | | platform, 저장 키 `''` | `CUSTOM`으로 전환: (a) 입력 없음 (b) `dummy-new-key` | 가드 비적용. (a) 검증 2의 (가)로 저장 불가 (b) 저장 가능 |
| 9 | | own_required(`CUSTOM`), `dummy-old-key` | `CUSTOM_ALT`로 변경(ID 타이핑 포함, 같은 그룹, 같은 접두) | 가드 비적용, 저장 가능, 저장 키 유지(detail은 입력란 값, sidebar는 `savedEngineKey`). 입력 중 입력란 유지 |
| 10 | sidebar | 서버 (`OPENAI`, `dummy-old-key`), **캐시 `CUSTOM`** | 폼은 캐시 값 그대로, 이름만 변경: (a) 키란 안 건드림 (b) `dummy-new-key` | (a) 저장 불가(기준선 `savedEngineModel`) (b) 저장 가능, body에 `dummy-old-key` 없음. 재조회 전/실패 시 Save 비활성 |
| 11 | | own_or_default(`OPENAI`), `dummy-old-key` | `OPENAI_ALT`(같은 접두) | 가드 비적용, 저장 가능, 키 유지 |
| 12a/12b | | 카탈로그에 없는 저장값, `dummy-old-key` | `CUSTOM`으로 전환: (a) 입력 없음 (b) `dummy-new-key` | `keyModeOf` 폴백(own_or_default)으로 그룹 불일치, (a) 저장 불가 (b) 저장 가능. 저장값이 `openrouter.` 접두면 Custom으로 복원되어 같은 그룹 |
| 13 | | own_or_default(`OPENAI`), `dummy-old-key`, 입력란에 `dummy-typed-key` 타이핑 | `CUSTOM`으로 전환, 전환 직후 입력란 확인, 저장 시도 | 입력란 비어 있음(detail은 미리 채운 저장 키도 비움, sidebar는 `engineKeyChanged=false`), 저장 불가. `OPENAI`로 돌아오면 입력란이 저장 키 상태로 복원(true -> false) |
| 14a | | 저장 (`OPENAI`, `dummy-old-key`) | `GEMINI`로 전환 후 `dummy-new-key` 입력, 이어서 `GROK`으로 변경 | `GROK` 변경에서 가드가 true인 채 그룹이 같으므로 입력 유지, 저장 가능 |
| 14b | | 저장 (`OPENAI`, `''`) | `dummy-typed-key` 타이핑 후 같은 접두 `OPENAI_ALT`로 변경 | 가드 false 유지, 입력 유지(create와 동일) |
| 15a~15d | | own_or_default(`OPENAI`), `dummy-old-key` | `GEMINI`(다른 서비스 접두, 같은 그룹)로 전환: (a) 입력 없음 (b) `dummy-new-key` (c) 공백만 입력 (d) **detail 전용** 미리 채운 키 상태, **sidebar 전용** `dummy-typed-key` 타이핑 후 전환 | 가드 적용. (a)(c) 저장 가능 `body.engine_key === ''`(공백만은 새 입력 아님) (b) 새 키만 (d) 입력란 비어 있고 body에 `dummy-old-key`/`dummy-typed-key` 없음. 이어서 `GROK`으로 한 번 더 바꿔도 가드 true인 채라 그 사이 입력한 `dummy-new-key` 유지 |
| 16 | | 가드 적용 상태 | 목적지 `own_required`/`own_or_default`/`platform` 각각 | 경고 문구 3종(`Replace the API key before saving.` / `The saved key may belong to a different provider. Replace it, or leave it empty to use the default key.` / `The saved key will be removed.`). platform 문구는 sidebar에서만, detail에는 없음. sidebar는 세 목적지 모두 키 블록이 `<details>` 밖. detail platform은 입력란이 비어 있으나 저장 키 재전송(제안 v) |
| 17 | | platform(`PLATFORM`), `dummy-old-key` | `PLATFORM_ALT`(접두 `anthropic` -> `meta`) | 가드 적용, 입력란 비움, 새 키 요구 없음, 저장 가능. 경고는 sidebar만. detail 재전송, sidebar 미전송 |
| 18a | | own_or_default(`OPENAI`), 저장 키 `''` | `dummy-typed-key` 타이핑 후 `CUSTOM`으로 전환, 저장 시도 | 그룹 변경이므로 입력란 비어 있음, body에 `dummy-typed-key` 없음, 검증 2의 (가)로 저장 불가 |
| 18b | | own_required(`CUSTOM`), 저장 키 `''` | `dummy-typed-key` 타이핑 후 `OPENAI`로 전환 | 입력란 비어 있음, `body.engine_key === ''`로 저장 가능 |
| 18c | | own_or_default(`OPENAI`), 저장 키 `''` | `dummy-typed-key` 타이핑 후 `GEMINI`(같은 그룹, 다른 접두) | 가드 false(저장 키 없음)이므로 입력 **유지**(create와 동일) |
| 19d | detail | own_or_default(`OPENAI`), `dummy-old-key` | **카탈로그가 `loading`인 동안** `dummy-typed-key` 타이핑, `CUSTOM`으로 전환(mock 버튼), 카탈로그 ready | ready 후 첫 effect가 저장 모델 기준선과 비교해 그룹이 다르면 비움. 입력란 비어 있고 body에 `dummy-typed-key` 없음. 모델 변경이 없는 정상 로드는 비우지 않음(기록만) |
| 19s | sidebar | 19d와 같음 | **`keyLoad==='loading'`인 동안**(캐시 히트 + 재조회 지연) 타이핑하고 `CUSTOM`으로 전환, 재조회 완료 | 동일 기대, `savedEngineModel` 기준선 |
| 20 | | 저장 (`OPENAI`, `dummy-old-key`) | `GEMINI`로 전환 후 `dummy-gemini-typed-key` 타이핑, `CUSTOM`으로 전환 | 가드가 true인 채 그룹이 바뀌므로 입력란 비움. body에 `dummy-gemini-typed-key`, `dummy-old-key` 없음. 이후 입력한 `dummy-new-key`는 유지. `OPENAI`로 복귀하면 저장 키 상태로 복원 |
| 21 | | own_required(`CUSTOM`), `dummy-old-key` | `PLATFORM`으로 전환 후 다시 `CUSTOM`(같은 ID)으로 복귀 | 두 번째 전이가 true -> false이므로 **복원**: detail 입력란 `dummy-old-key`, sidebar는 `savedEngineKey` 유지. 입력 없이 저장 가능(같은 그룹, 같은 접두), body는 저장 키 |

정리 사유: 디자인 행 14의 "Custom ID 타이핑"은 행 9와, 행 20의 "저장 키 없는 Custom에서 OpenAI"는 행 18b와 중복이라 제거했다. 행 19는 detail의 로딩 게이트(카탈로그)와 sidebar의 로딩 게이트(`keyLoad`)가 달라 `only`로 분리했다.

공통 순수 함수의 판정(F17, `useAIModels.js`; T14에서 구현하고 이 표가 검증한다):
- `computeGuardApplies({ savedKey, savedModel, currentModel, models })`: `savedKey.trim() !== ''` **그리고** (`computeKeyMode(models, currentModel) !== computeKeyMode(models, savedModel)` **또는** `servicePrefixOf(currentModel) !== servicePrefixOf(savedModel)`).
- `keyTransitionAction(prev, cur)`(`{ keyMode, guardApplies }`): `prev.guardApplies && !cur.guardApplies` 이면 `'restore'`(최우선), 아니면 `prev.keyMode !== cur.keyMode` 이면 `'clear'`, 아니면 `!prev.guardApplies && cur.guardApplies` 이면 `'clear'`, 그 외 `'none'`.

---

## 5. 태스크

각 태스크 형식: **Files**(파일:라인), **Change**(정확한 변경), **Tests**(함께 작성), **Verify**(명령, 기대 결과), **Depends / Commit**. 테스트 이름은 `Test_` 접두, 단정은 `reflect.DeepEqual` + `t.Errorf`, gomock 컨트롤러 변수는 `mc`(F14).

### Phase 0: 착수

#### T0. 착수 점검 (코드 변경 없음)
- 두 워크트리에서 `git fetch origin main`, `git log --oneline HEAD..origin/main`이 비어 있음(이번 확인 시 비어 있음), `git status --short` 깨끗함(monorepo는 plan 문서만 untracked), `git log -1 --format=%ae`가 `pchero21@gmail.com`.
- Go 기준선: `cd bin-ai-manager && go test ./models/ai/ ./pkg/aihandler/... ./pkg/listenhandler/... ./cmd/ai-control/... -count=1` 통과. Python 기준선(1절, 250 passed / 4 failed 기존 / 2 skipped).
- **UI 기준선(실제 실행 결과, 구현 착수 전에 한 번 더 같은 명령으로 재측정해 PR 본문의 "Known pre-existing failures"에 기록):** `cd square-admin && npm ci`(약 11초) 후
  - `npm test -- --watchAll=false --forceExit 2>&1 | grep "Tests:"` -> `Tests: 2 failed, 1 skipped, 3027 passed, 3030 total`, Test Suites `1 failed, 267 passed, 268 total`. 실패: `src/views/mcpservers/__tests__/mcpServerCatalog.test.js`(`every entry has the required fields`, `has no duplicate urls`).
  - `npm run lint` -> `548 problems (381 errors, 167 warnings)`. 작업 대상 파일 기준선은 1절 UI 검증 2번의 표(소스 0 errors 3 warnings, 테스트 15 errors 0 warnings).
  - `npm run build`(`CI=true` 없음, 실제 CI와 동일, `prebuild`로 `build:widget` 포함) -> exit 0, `git status` 깨끗. `CI=true npm run build` -> 기준선에서 실패("Treating warnings as errors because process.env.CI = true.", "Failed to compile.").
  - 기준선 측정은 변경 없는 트리이므로 `git stash`가 필요 없었다. 구현 후 비교는 CLAUDE.md의 `git stash` 방식(변경 stash, 기준선 측정, `git stash pop`, 변경 후 측정)을 따른다.
- 1절 5번의 새 심볼 충돌 grep(전부 없음이어야 함).
- 6절 분기 조건 항목의 현재 상태를 기록한다(T11 착수 전에 8.1 항목 8의 선택지가 확정되어야 한다).
- **디자인 8.4가 "리뷰 전에 대표님 확인 권장"으로 남긴 8.1 항목 1(모델 ID 슬래시 1개 규칙)과 항목 2(키 입력 라벨 통일, `Engine Key (optional)`)의 확인 상태를 기록한다.** 둘 다 CPO 권장은 채택이고 이 plan은 채택을 전제로 T2(`customModelIDPattern`), T14(`KEY_FIELD_COPY`)를 짰다. 확인되지 않은 채 착수하면 두 태스크가 되돌림 비용을 갖는다는 점을 대표님께 알리고, 미채택 결정이 나면 T2 패턴과 T2 슬래시 0개/2개 이상 거부 테스트, T14 라벨 표만 바뀐다.

---

### Phase A: ai-manager 코어 (스트림 S1, monorepo)

#### T1. 카탈로그, key_mode, Custom 항목, 가드 완화 (디자인 3.1.1, 3.1.5, Q1a, Q2, Q6)
**Files**
- `bin-ai-manager/models/ai/catalog.go`: Route 상수 `:6-9`, `ModelEntry :15-24`, `ModelInfo :27-35`, `CatalogView :38-56`, `catalog :61`(끝에 추가).
- `bin-ai-manager/models/ai/resolve.go:28-47`: 정확 일치 루프에 Custom 건너뛰기 한 줄만.
- 테스트: `models/ai/catalog_test.go:9-38,41-51,152-163`, `models/ai/resolve_test.go:68-79`, `pkg/listenhandler/v1_ai_models_test.go:38,46`.
- **F1 교차 서비스 테스트 수정(같은 커밋)**: `bin-api-manager/server/ai_models_test.go`(기대 JSON), `bin-common-handler/pkg/requesthandler/ai_ais_test.go:97-159`(`Test_AIV1AIModelList`, 다음 함수는 `:161`), `bin-api-manager/pkg/servicehandler/ai_test.go:221-256`(Custom 행 추가).

**Change**
```go
// catalog.go
const RouteCustomOpenRouter Route = "custom_openrouter" // the single customer-keyed OpenRouter entry

const (
	KeyModePlatform     = "platform"       // platform supplies the key
	KeyModeOwnOrDefault = "own_or_default" // customer key, or the platform default when empty
	KeyModeOwnRequired  = "own_required"   // customer key is mandatory
)

// EngineModelPrefixCustomOpenRouter is prepended to the model ID the customer types.
const EngineModelPrefixCustomOpenRouter = "openrouter."
```
`ModelEntry`에 `CustomPrefix string`. `ModelInfo`에 `KeyMode string \`json:"key_mode"\``(항상 존재)와 `ModelIDPrefix string \`json:"model_id_prefix,omitempty"\``를 **`PlatformManaged` 뒤에** 추가(JSON 필드 순서가 api-manager 기대 문자열에 영향). `CatalogView()`는 `Route`에서 `key_mode` 도출(`RouteDirect`->own_or_default, `RouteOpenRouter`->platform, `RouteCustomOpenRouter`->own_required), `ModelIDPrefix: e.CustomPrefix`, `PlatformManaged`는 `e.Route == RouteOpenRouter` 그대로(Custom은 false). 카탈로그 끝 항목:
`{ID: "custom.openrouter", Label: "OpenRouter model (your OpenRouter key)", Vendor: "OpenRouter", Route: RouteCustomOpenRouter, CustomPrefix: EngineModelPrefixCustomOpenRouter, Description: "Enter any model supported by OpenRouter."}`(`UpstreamSlug`, `Tags`, `Recommended` 없음). `ResolveEngine` 루프: `if catalog[i].Route == RouteCustomOpenRouter { continue }`(`custom.openrouter` 저장 시 `Rejected`).

**Tests (먼저 작성, 빨간 것 확인)**
- `Test_CatalogInvariants`: 알려진 라우트에 `RouteCustomOpenRouter` 추가(`:27-29`의 unknown route 검사), Custom 항목 정확히 1개, `UpstreamSlug == ""`, `CustomPrefix == "openrouter."`, **모든 항목의 ID가 `openrouter.`로 시작하지 않음**(신규 공통 불변 조건).
- `Test_CatalogPublicViewHidesInternals` 분할(디자인 3.1.5 가드 1): JSON을 `[]map[string]any`로 디코딩하고 `model_id_prefix`가 있는 항목이 **정확히 1개**임을 단정한다. **함정(직접 확인):** `openrouter`라는 문자열 자체가 `route`를 부분 문자열로 포함하고(`open`+`route`+`r`), Custom 항목의 `id`(`custom.openrouter`), `label`(`OpenRouter model ...`), `model_id_prefix`(`openrouter.`)가 모두 `openrouter`를 담으므로 **Custom 항목의 직렬화 문자열에 `strings.Contains(lower, "route")`를 쓰면 항상 걸려 통과할 수 없다**(현행 테스트 `catalog_test.go:41-51`은 전체 JSON에 `slug`, `route`, `openrouter`, `meta-llama/`를 부분 문자열로 금지하고, `v1_ai_models_test.go:46`은 `slug`, `openrouter`, `meta-llama/`, `"route"`(따옴표 포함)를 금지한다). 그래서 검사를 이렇게 나눈다. (가) **비 Custom 항목**: 항목별 재직렬화 문자열에 기존 4토큰(`slug`, `route`, `openrouter`, `meta-llama/`)을 그대로 금지(약화 없음, 이 항목들은 `key_mode` 값 `platform`/`own_or_default`/`own_required`에도 `route`가 없음을 확인함). (나) **Custom 항목은 디코딩한 키 이름 기준으로 검사**: 키 이름 집합이 정확히 `{id, label, vendor, recommended, tags, description, platform_managed, key_mode, model_id_prefix}`이고(`slug`, `upstream_slug`, `route`, `custom_prefix` 같은 키 없음, 허용 목록이 바뀌면 실패), 문자열 값에 `meta-llama/`와 `slug`가 없으며, `openrouter`(대소문자 무시)는 상수 `customOpenRouterAllowedFields = []string{"id","label","vendor","description","model_id_prefix"}` 필드의 값에서만 허용하고 그 외 필드 값(예 `key_mode`, `tags`)에 나타나면 실패한다. Custom 항목에 대한 `route` 부분 문자열 검사는 하지 않는다. 같은 규칙을 `models/ai/catalog_test.go`와 `pkg/listenhandler/v1_ai_models_test.go`에 각각 둔다(패키지 경계상 비공개 상수를 공유할 수 없다). `v1_ai_models_test.go:46`의 응답 전체 `strings.Contains(body, "openrouter")` 금지는 Custom 항목 때문에 항상 실패하므로 같은 (가)(나) 분리로 바꾼다.
- `Test_CatalogCustomerFacingTextHasNoBannedTerms`: `Route == RouteCustomOpenRouter`에 한해 `label`, `description`에서만 `openrouter` 허용, 타사명과 `zero data` 금지는 유지.
- `Test_CatalogKeyModeMatchesRoute`(신규): `Route` 대 `key_mode` 대응표, `model_id_prefix`는 Custom에만 존재, JSON `omitempty`(비 Custom 항목 JSON에 키 없음), `key_mode`는 모든 항목에 존재. `Test_CatalogViewShape`의 `rec == 1` 단정은 Custom이 `Recommended=false`라 그대로 통과함을 확인.
- `Test_ResolveEngineAllOpenRouterEntriesBlankKey`(`resolve_test.go:68-79`): `Route == RouteCustomOpenRouter` 항목은 루프에서 제외하고, `ResolveEngine("custom.openrouter")`가 `OutcomeRejected`임을 별도 단정.
- `Test_processV1AIModelsGet`(`v1_ai_models_test.go`): 필수 키 목록에 `key_mode` 추가, Custom 분리 규칙은 가드 3(`:46`).
- F1: `server/ai_models_test.go`의 모델 리터럴에 `KeyMode: "own_or_default"`를 넣고 기대 문자열 끝을 `..."platform_managed":false,"key_mode":"own_or_default"}`로, **Custom 행을 추가**(`...,"key_mode":"own_required","model_id_prefix":"openrouter."}`). `ai_ais_test.go` wire 테스트(13.9)의 JSON과 기대 구조체에 `key_mode`, `model_id_prefix` 포함 행 추가. `servicehandler/ai_test.go`에 Custom 행(DeepEqual 통과 확인).

**Verify**
```
cd bin-ai-manager && go test ./models/ai/ ./pkg/listenhandler/... -count=1
# F1 대상: ModelInfo가 바뀐 뒤 재vendor
cd ../bin-common-handler && go mod tidy && go mod vendor && go test ./pkg/requesthandler/... -count=1
cd ../bin-api-manager && go mod tidy && go mod vendor && go test ./server/... ./pkg/servicehandler/... -count=1
```
기대: 전부 PASS. 이 시점에 pipecat과 aihandler 테스트는 변함없이 통과해야 한다(`openrouter.*`는 아직 `Rejected`). `cd bin-ai-manager && go test ./... -count=1`와 `cd ../bin-pipecat-manager && go mod tidy && go mod vendor && go test ./... -count=1`도 C1 직후 초록이어야 한다.
**Depends:** T0. **Commit:** C1.

#### T2. 모델 ID 검증, ResolveEngine Custom 분기, ValidateEngine (디자인 3.1.2, 3.1.3, Q4, 4절 승격 방지) **(T3, T6과 한 커밋 C2)**
**Files:** `bin-ai-manager/models/ai/resolve.go`(상수 `:12-17`, `Resolved :19-23`, `ResolveEngine :28-47`, `IsValidEngineModel :50`), 테스트 `resolve_test.go:21-79`(특히 `:37` raw_openrouter 행), `models/ai/main_test.go:324-346`(`TestIsValidEngineModel`, `:337` 행).

**Change**
```go
const OutcomeCustomOpenRouter // appended AFTER OutcomeDirectPassthrough (keep iota values stable)

type Resolved struct { Entry *ModelEntry; RunnerType string; BlankKey bool
	RequireKey bool // custom models: an empty key is a hard failure
}

var (
	ErrInvalidEngineModel = errors.New("invalid engine model")
	ErrEngineKeyRequired  = errors.New("engine key required")
)

var customModelIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateCustomModelID checks the part after the "openrouter." prefix.
// Total length with the prefix must stay within the 255 char engine_model column.
func ValidateCustomModelID(id string) bool // len(id) in 1..255-len(prefix) (244), pattern, author != "openrouter" (case-insensitive)

func ValidateEngine(model EngineModel, key string, modelChanged bool) error
```
`ResolveEngine` 순서(보안 속성): 1 정확 일치(Custom 건너뜀) -> 2 `strings.HasPrefix(string(m), EngineModelPrefixCustomOpenRouter)`(대소문자 구분 정확 일치)이면 `ValidateCustomModelID(rest)` 통과 시 `Resolved{RunnerType: string(m), RequireKey: true}, OutcomeCustomOpenRouter`(**`platform_openrouter.` 접두를 만들지 않고 `BlankKey`는 항상 false**), 실패 시 `Rejected` -> 3 기존 `openai/gemini/grok` -> 4 `Rejected`. `ValidateEngine`: `modelChanged`이고 `Rejected`면 `ErrInvalidEngineModel`; 최종 모델이 `openrouter.` 접두를 가지면(변경 여부 무관) `strings.TrimSpace(key) == ""`일 때 `ErrEngineKeyRequired`(모델이 바뀐 Rejected 케이스는 `ErrInvalidEngineModel`이 우선). 직접 모델은 빈 키여도 `nil`(Q1b). `IsValidEngineModel`은 이름과 시그니처 유지(`o != OutcomeRejected`).

**Tests (먼저)**
- `Test_ValidateCustomModelID`: 통과(`mistralai/mistral-large-2411`, `meta-llama/llama-3.3-70b-instruct`, 점 포함 ID, 대소문자 `Mistralai/Large`는 통과하며 정규화 안 함), 길이 244 통과/245 거부(접두 포함 255 경계), 빈 값, 공백/제어문자/쉼표/`~`/`@`/`:`(`:free`, `:nitro`, `:online`, `:thinking`, `:floor` 전부), 슬래시 0개/2개 이상, 시작 문자가 `.`/`-`/`_`, `openrouter/auto`, `OpenRouter/auto`, `OPENROUTER/free`.
- `Test_ResolveEngine` 갱신: `:37` `raw_openrouter`를 `OutcomeCustomOpenRouter`, `RunnerType == 입력`, `RequireKey`, `!BlankKey`로 변경하고 형식 오류/라우터/콜론/대소문자 변형 거부 행 추가. `Test_ResolveEngineCustomNeverPromotes`(4절): `OpenRouter.x/y`, `PLATFORM_OPENROUTER.x/y`, ` openrouter.x/y`(앞 공백), `openrouter.`(빈 ID), `openrouter.platform_openrouter.x`(슬래시 없음 거부), `openrouter.platform_openrouter/x`(통과, 단 `RunnerType`에 `platform_openrouter.` 접두 없음, 첫 점 앞이 정확히 `openrouter`), `custom.openrouter`(거부), `platform_openrouter.x`(거부). 통과한 모든 Custom 값에 대해 `strings.SplitN(RunnerType, ".", 2)[0] == "openrouter"`와 `BlankKey == false`를 단정(Python 소문자화 근거).
- `TestIsValidEngineModel`(`main_test.go:337`): `raw_openrouter_is_invalid`를 유효 Custom ID는 true, 무효 ID는 false 행으로 분리(기존 함수 이름은 건드리지 않고 행만 변경, F14).
- `Test_ValidateEngine`: `modelChanged` 조합 x 키(`""`, `"   "`, `"dummy-new-key"`) 전체, 변경 안 된 레거시 `openrouter.x/y` + 키 있음 -> nil, 변경 안 됨 + 빈 키 -> `ErrEngineKeyRequired`, 변경됨 + 무효 ID -> `ErrInvalidEngineModel`, 직접 모델 + 빈 키 -> nil, **인자 뒤바뀜 방어**(`ValidateEngine("openai.gpt-5", "openrouter.a/b", true)`가 nil: 키 자리에 Custom 모델 문자열을 넣어도 모델 규칙이 키에 적용되지 않음).

**Verify:** `cd bin-ai-manager && go test ./models/ai/ -count=1`로 T2 자체를 확인한 뒤, **T2 단독 상태에서 `cd bin-ai-manager && go test ./... -count=1`는 `pkg/aihandler`의 `Test_Create_EngineModelPolicy/rejects_raw_openrouter`가 빨갛다(직접 확인한 의도된 중간 상태)**. 따라서 T2는 단독 커밋하지 않고 **T3(aihandler 연결과 그 테스트)과 T6(pipecat)을 끝낸 뒤 한 커밋 C2**로 묶는다. C2 직전 게이트(모두 초록이어야 함): `cd bin-ai-manager && go test ./... -count=1`, `cd ../bin-pipecat-manager && go mod tidy && go mod vendor && go test ./... -count=1`, `go build ./...`를 `bin-ai-manager`, `bin-pipecat-manager`, `bin-api-manager`, `bin-common-handler`에서.
**Depends:** T1. **Commit:** C2(T3, T6과 함께).

#### T3. aihandler 연결, ENGINE_KEY_REQUIRED, 입력 반향 방어 (디자인 3.1.2~3.1.4, R1-e) **(T2, T6과 한 커밋 C2)**
**Files:** `bin-ai-manager/pkg/aihandler/chatbot.go`(`errInvalidEngineModel :22-28`, `Create` 검증 `:51-53`, `Update`의 `preUpdateAI` 조회 `:140` 뒤 검증 `:148-150`), 테스트 `chatbot_engine_model_test.go:20-80`(`:27` `rejects_raw_openrouter`, `:72` `unchanged_legacy_openrouter_value_passes`), `chatbot_engine_model_error_test.go:22-56,58-90`.

**Change**
- 신규 헬퍼(같은 파일): `errEngineKeyRequired()`(`cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "ENGINE_KEY_REQUIRED", "An API key is required for custom OpenRouter models.")`), `errInvalidCustomModelID()`(reason `INVALID_ENGINE_MODEL`, 메시지 `invalid engine_model: the OpenRouter model ID is not valid. Use the vendor/model-name form with letters, digits, '.', '_' and '-' only. Variant suffixes such as ':free' and router IDs such as 'openrouter/auto' are not supported.`, **입력값을 `%q`로 되돌리지 않음**), 변환 헬퍼 `engineValidationError(model ai.EngineModel, err error) error`(`errors.Is`: `ErrEngineKeyRequired` -> `errEngineKeyRequired()`, `ErrInvalidEngineModel` -> 입력이 `openrouter.`(대소문자 무시) 접두면 `errInvalidCustomModelID()`, 아니면 기존 `errInvalidEngineModel(model)`).
- `Create :51`: `if err := ai.ValidateEngine(engineModel, engineKey, true); err != nil { return nil, engineValidationError(engineModel, err) }`. `Update :148`: `ai.ValidateEngine(engineModel, engineKey, engineModel != preUpdateAI.EngineModel)`(최종 상태 기준, 모델 불변이어도 키가 비면 거부). `ai` 패키지는 `cerrors`를 import하지 않는 현재 구조 유지.

**Tests (먼저)**
- `chatbot_engine_model_test.go`: `rejects_raw_openrouter` -> 키 있으면 수락 / 키 비면 `ENGINE_KEY_REQUIRED`로 분리, `:72` 행 유지 + 키가 빈 최종 상태 거부 행 추가, Update: 모델 변경 + 빈 키, 모델 불변 + 빈 키(`ENGINE_KEY_REQUIRED`), 모델 불변 + 공백 키.
- `chatbot_engine_model_error_test.go`: **`assertInvalidEngineModel`(`:22-56`)은 완화하지 않는다.** 새 헬퍼 `assertInvalidCustomModelID`(`openrouter` 허용, 입력 더미가 메시지에 없음, `platform_`/`engine_key` 금지, HTTP 400)와 `assertEngineKeyRequired`(reason `ENGINE_KEY_REQUIRED`, 400, 메시지에 키 값 없음, `cerrors.ToResponse` 400) 추가. 모델 ID에 키 붙여넣기 방어: `openrouter.dummy-key-not-real`(슬래시 없음), 슬래시 2개 이상 모두 `INVALID_ENGINE_MODEL`이고 **오류 메시지에 `dummy-key-not-real`이 없음**. `OpenRouter.x/y`(대문자 변형)도 Custom 전용 메시지(입력 미반향).

**Verify:** `cd bin-ai-manager && go test ./pkg/aihandler/... ./models/ai/... -count=1` 후 `go test ./... -count=1`(T2의 중간 빨강이 모두 해소되어야 함).
**Depends:** T2. **Commit:** C2(T2, T6과 함께).

#### T4. ai-control 사전 검증 (디자인 3.1.2 ai-control 항, 3.1.3 반향 표 2행)
**Files:** `bin-ai-manager/cmd/ai-control/main.go`: `validateEngineModelCreate :412-417`, `validateEngineModelUpdate :421-436`, 호출 `:213`(`engineKey`는 `:197`에서 읽음)와 `:337`(`engineKey`는 `:321`), 테스트 `main_test.go:24-70`.

**Change:** 두 함수에 `engineKey string` 인자 추가. Create: `engineModel != ""`이면 `ai.ValidateEngine(engineModel, engineKey, true)`. Update: `engineModel == ""`이면 기존대로 조회 없이 `nil`(핸들러가 판단), 아니면 `Get` 후 `ai.ValidateEngine(engineModel, engineKey, engineModel != stored.EngineModel)`. 오류 문자열: `openrouter.`(대소문자 무시) 접두 입력은 **원문 없이** `invalid engine model: the OpenRouter model ID is not valid`, 키 누락은 `an API key is required for custom OpenRouter models`, 그 외 입력은 현행 `invalid engine model: %s` 유지. CLI는 빠른 실패용이고 최종 방어는 `aihandler.Create/Update`다(`main.go:242,366`이 직접 호출).

**Tests:** `Test_validateEngineModelCreate/Update`에 키 인자 추가, Custom + 빈 키 행, Custom + 키 행, `openrouter.` 접두 무효 입력의 **오류 문자열에 입력값이 없음**, Update 모델 불변 + 빈 키(`engineModel`이 비어 있지 않은 경우만 검증됨), 조회 횟수 단정(`fakeAIGetter.calls`) 유지.
**Verify:** `cd bin-ai-manager && go test ./cmd/ai-control/... -count=1`. **Depends:** T2(C2 이후). **Commit:** C3(단독. ai-control 테스트는 C2 직후에도 초록임을 확인했다).

#### T5. ai-manager 키 로깅 정리: A 11곳, B1 6곳, B2 5곳 (디자인 3.4, Q10)
**Files**(모두 한 줄 수정, 로그 메시지와 나머지 필드 유지)
- A: `pkg/aicallhandler/start.go:260,302,486,726`(`"ai": a` / `"ai": c` -> `"ai_id": a.ID` / `c.ID`), `pkg/aicallhandler/db.go:39,119`(`"ai": c` -> `"ai_id": c.ID`), `pkg/aihandler/db.go:150`, `pkg/aihandler/direct_hash.go:28`, `pkg/aicallhandler/send.go:170`, `pkg/aicallhandler/tool.go:1078`(`tmpAI`), `pkg/teamhandler/handler.go:244` -> `WithField("ai_id", x.ID)`. **`send.go:170`은 필드 교체에 더해 메시지의 `ai_engine_model: %s`와 `a.EngineModel` 인자도 제거하는 것으로 확정한다**(현재 줄: `log.WithField("ai", a).Debugf("Resolved team member AI. member_id: %s, ai_engine_model: %s", resolvedMemberID, a.EngineModel)`). Custom 모델 ID는 고객 자유 입력이라 로그에 반복 남기지 않고, 같은 정보는 `ai_id`로 조회한다. 결과 줄: `log.WithField("ai_id", a.ID).Debugf("Resolved team member AI. member_id: %s", resolvedMemberID)`(바로 위 `:166-168`이 `err`를 처리하므로 `a`는 non-nil).
- B1: `pkg/listenhandler/main.go:285`와 `pkg/listenhandler/v1_ais.go:24,83,159,198,237`의 `"request": m` -> `"uri": m.URI, "method": m.Method, "request_id": m.RequestID`(필드 이름은 `sock.Request` 정의를 확인해 맞춤. `main.go:276-278`의 Builder 선례처럼 본문 제외).
- B2(권장, 같은 파일): `v1_ais.go:66`(**`Debugf`**), `:142,181,220,316`(`Errorf`)의 `Could not marshal the response message. message: %v, err: %v`에서 `message: %v` 인자와 문구를 제거하고 `err`만 남김.
- **nil 확인(F11):** A의 `a.ID`/`c.ID` 전환 전에 각 줄의 `a`, `c`가 non-nil인지 코드로 확인한다(production 경로는 `resolveAI`가 non-nil을 보장). 기존 단위 테스트가 nil `*ai.AI`를 넘겨 패닉하면 코드가 아니라 **테스트 데이터를 고친다**.

**Tests:** A는 전용 테스트 없음(대표님 확정). **B1 로그 캡처 테스트 1개**(`Test_processRequest_errorLogOmitsRequestBody`, 선례 `v1_ai_builder_test.go:221-224`의 `logrus.SetOutput(&buf)` + `TraceLevel` 패턴, 종료 시 복원): 더미 `engine_key`(`dummy-engine-key-not-real`)를 담은 (a) `POST /v1/ais`(aiHandler.Create가 오류), (b) `PUT /v1/ais/{id}`(Update 오류), (c) 알 수 없는 URI(`main.go:631` 경로)가 오류로 끝난 뒤 캡처 로그에 더미 문자열이 **없고** `uri`, `method`가 있음. 기존 `/v1/ais` 핸들러 테스트는 로그를 단정하지 않는다(확인함).
- **send.go:170(검토 1줄):** `log.WithField("ai", a).Debugf("Resolved team member AI. member_id: %s, ai_engine_model: %s", ..., a.EngineModel)`의 `a.EngineModel`은 고객이 입력한 `openrouter.<model-id>` 원문이 될 수 있다(고객 입력이므로 키 붙여넣기 반사 위험). A의 `"ai": a`를 `"ai_id": a.ID`로 바꾸는 같은 줄에서 `ai_engine_model`을 제거하거나 모델 ID 원문을 찍지 않는 형태로 함께 처리할지 검토해 결정한다(T9의 `loggableEngineModel`은 api-manager 전용 비공개 헬퍼라 재사용 불가, 제거가 가장 단순하고 `ai_id`로 조회 가능).

**Verify:** `cd bin-ai-manager && go test ./... -count=1`와 아래 grep(**수정 전 코드에서 직접 실행한 실제 건수**를 함께 적는다). 이전 초안의 `grep -rn '"engine_key"\|"ai": \(a\|c\|tmpAI\)'`는 gofmt가 키 뒤 공백을 정렬(`"ai":            a,`)해 **수정 전에도 0건**이라 공회전했다(실측 0건). 교체:
```
cd bin-ai-manager
# A: ai 구조체 통째 로깅. 수정 전 11건(start.go 4, aicallhandler/db.go 2, aihandler/db.go 1, direct_hash.go 1, send.go 1, tool.go 1, teamhandler/handler.go 1), 수정 후 0건
grep -rnE '"ai"[,:][[:space:]]*(a|c|tmpAI)\b' pkg --include=*.go | grep -v _test | wc -l
# B1: 요청 본문 로깅, 대상은 listenhandler/main.go와 v1_ais.go만. 수정 전 6건(main.go 1, v1_ais.go 5), 수정 후 0건
grep -nE '"request":[[:space:]]*m\b' pkg/listenhandler/main.go pkg/listenhandler/v1_ais.go | wc -l
# B2: 수정 전 5건(v1_ais.go:66,142,181,220,316), 수정 후 0건
grep -nE 'Could not marshal the response message. message: %v' pkg/listenhandler/v1_ais.go | wc -l
```
주의: `"request": m`은 이 서비스의 다른 listenhandler 파일에도 **48곳 더 있다**(`v1_mcpservers.go` 8, `v1_aicalls.go` 8, `v1_aipromptproposals.go` 7 등, 전체 54건). 이 PR 범위는 AI 생성/수정 경로(`v1_ais.go`, `main.go`)이므로 B1 grep은 두 파일로 한정하고 나머지는 범위 밖(10절)이다. 위 grep이 A 11, B1 6, B2 5곳 대응을 모두 덮는다(전체 22곳).
**Depends:** T0(독립). **Commit:** C4.

---

### Phase B: pipecat-manager (스트림 S2, monorepo)

#### T6. classifySessionLLM, resolveSessionLLM Custom 처리, Rejected 오류 정제 (디자인 3.3 Go) **(T2, T3과 한 커밋 C2)**
**Files:** `bin-pipecat-manager/pkg/pipecatcallhandler/llmresolve.go:17-29`(전체), `start.go:41`(`Start()`의 키 없는 사전 분류), 호출 지점 확인용 `start.go:97`(`runGetLLMKey` 경유), `start.go:208`, `run.go:141`, `run.go:209`는 **변경 없음**. 테스트 `llmresolve_test.go`, `run_teamllmtype_test.go`.

**Change**
```go
// classifySessionLLM only decides whether the model is selectable. It never looks
// at a key, so the keyless pre-check in Start() keeps working for custom models.
func classifySessionLLM(llmType pipecatcall.LLMType) error

// resolveSessionLLM keeps its signature. For custom OpenRouter models it fails when
// the trimmed key is empty and forwards only the trimmed key.
func resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType, runnerKey string, err error)
```
`Outcome == Rejected` -> 오류. 오류 문구는 `openrouter.`(대소문자 무시) 접두로 시작하는 입력이면 **입력을 넣지 않고** `engine model is not available`, 그 외는 현행 `engine model is not available: %s`. `r.BlankKey` -> `(RunnerType, "", nil)`. `r.RequireKey` -> `key := strings.TrimSpace(aiKey)`, 비면 `errors.New("custom engine key is empty")`(키, 모델 ID 원문 미포함), 아니면 `(RunnerType, key, nil)`. 나머지는 `(RunnerType, aiKey, nil)`. `start.go:41`: `resolveSessionLLM(llmType, "")`를 `classifySessionLLM(llmType)`로 교체(오류 래핑 문구 `could not resolve llm type` 유지, `h.Create` 이전). `start.go:208`의 Custom + 빈 키(조회 실패 포함)는 이제 `resolveSessionLLM`이 오류를 내 세션 시작이 실패하고, 이때 `Start()`가 `h.Create`로 만든 row가 남는다(다른 사후 실패와 같은 기존 동작, 정리 로직은 추가하지 않음, 디자인 3.3).

**Tests (먼저, 5.1 pipecat 행 전부)**
- `Test_classifySessionLLM`: Custom 키 없이 통과, rejected 오류, `openrouter.` 접두 rejected 입력의 오류 문자열에 입력값 없음.
- `Test_resolveSessionLLM`(`llmresolve_test.go:24-50` 표): `:41` `rejected raw openrouter`를 Custom 수락(키 전달) 행으로 변경, Custom + 공백 포함 키(`"  dummy-new-key  "`)가 trim되어 전달, Custom + 빈/공백 키 오류, 오류 문자열에 모델 ID와 키 없음, platform은 기존대로 키 비움, 직접 모델 불변, `openrouter.` 접두 거부 입력(`openrouter.dummy-key-not-real` 등 형식 오류)의 오류 문자열에 입력값 없음 + 그 외 입력은 현행 문구 유지.
- `Test_Start_rejectedModelDoesNotCreate`(`:64-66`): 거부 모델 목록에서 `openrouter.meta-llama/llama-3-70b`를 제거하고 무효 Custom ID(`openrouter.openrouter/auto`, `openrouter.a/b:free`)로 대체. **신규** `Test_Start_customModelPassesClassification`: Custom 모델이 사전 분류를 통과해 `mockDB.EXPECT().PipecatcallCreate`가 1회 호출되는지(`pipecatcall.go:54`, `Create`가 추가로 호출하는 DB 메서드는 `pipecatcall.go`에서 확인해 EXPECT를 맞추고, 이후는 지원하지 않는 `referenceType`으로 `invalid reference type` 오류가 나게 해 러너 호출 없이 종료). **준비물(`start.go`, `pipecatcall.go:40-70`을 읽고 확인함):** `Start()`는 사전 분류 통과 후 `h.Create`를 호출하고, `Create`는 `h.db.PipecatcallCreate(ctx, tmp)`(`tmp.HostID = h.hostID`이므로 테스트의 `pipecatcallHandler`에 `hostID`를 채운다) 다음 `h.db.PipecatcallGet(ctx, id)`(성공 응답으로 `*pipecatcall.Pipecatcall` 반환, 오류 시 `Start`가 `could not create pipecatcall`로 실패하므로 반드시 EXPECT)를 호출하고 마지막에 `h.notifyHandler.PublishEvent(ctx, pipecatcall.EventTypeCreated, res)`를 호출한다. 따라서 `mockDB.EXPECT().PipecatcallCreate(...)`, `mockDB.EXPECT().PipecatcallGet(...)`, `mockNotify.EXPECT().PublishEvent(...)` 셋을 모두 설정하고, 이후 `switch referenceType`의 `default`(지원하지 않는 값)가 `invalid reference type` 오류를 내 러너 호출 없이 종료되게 한다.
- `Test_runGetLLMKey_resolves`(`:86-`): `:130-131` 행(`openrouter.x`는 슬래시 없음으로 계속 거부)은 유지하고, 유효 Custom(`openrouter.a/b`) + AI 조회 실패 = 키 없음 오류 행, 유효 Custom + AI 조회 성공(키 `dummy-old-key`) = 런너 타입 `openrouter.a/b` + 키 전달 행 추가(R1-b, R1-c).
- `Test_startReferenceTypeCall_rejectedModel`(`:168-196`): 입력을 진짜 거부 모델(`anthropic.claude-opus-4`)로 교체하고, **Custom + `ReferenceTypeCall`은 키 조회가 없어 빈 키 오류로 거부**되는 행을 별도로 추가(F13). `:216` 행 보강(유효 Custom + 조회 실패 = 오류).
- `run_teamllmtype_test.go:60`(`model2: openrouter.meta-llama/llama-3-70b` 거부): Custom 멤버 수락(키 있음: `LLMType == "openrouter.vendor/model-a"`, `EngineKey` 전달, `EngineModel`은 고객 값)과 Custom 멤버 빈 키 거부(`LLMType == ""`, `EngineKey == ""`, ids만 로깅) 행으로 분리, 정상 멤버 + Custom 혼합 팀(R1-d).

**Verify:** `cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... -count=1 && golangci-lint run -v --timeout 5m` 후 `git status --short`로 의도하지 않은 변경 없음 확인(1절).
**Depends:** T2(같은 커밋). **Commit:** C2(T2, T3과 함께).

#### T7. Python 러너 `openrouter` 분기 (디자인 3.3 Python, R1-a, R1-g)
**Files:** `bin-pipecat-manager/scripts/pipecat/run.py`: `_member_llm_type :571-588`(**변경 없음**), `create_llm_service :591-689`(앞단 파서 `:593-598`, 소문자화 `:601`, `platform_openrouter` 분기 `:658-687`, `else :688-689`). 테스트 `test_run.py`(`TestPlatformOpenRouter :940-1030`, `:1026-1034` raw 테스트, 프로브 `:1049-`), `test_init_pipeline.py`(팀 테스트, `:804,827,890`은 **변경 없음**).

**Change**
```python
# Shared by the platform and custom OpenRouter branches so one cannot lose it.
# Sent in extra_body: the OpenAI client rejects an unknown top-level kwarg.
_OPENROUTER_PROVIDER = {"zdr": True, "data_collection": "deny", "require_parameters": True}


def _build_openrouter_llm(api_key: str, model_name: str):
    return OpenRouterLLMService(
        api_key=api_key,
        settings=OpenRouterLLMService.Settings(
            model=model_name,
            extra={"extra_body": {"provider": dict(_OPENROUTER_PROVIDER)}},
        ),
    )
```
기존 `platform_openrouter` 분기는 인라인 dict 대신 `_build_openrouter_llm(api_key, model_name)`을 호출(동작 동일, 환경 키 검사와 `OpenRouter is not configured` 문구 유지). 새 분기를 `else` 앞에 추가:
```python
    elif service_name == "openrouter" and "." in type:
        # Custom (BYOK) OpenRouter: the customer's key only. This branch must never
        # read os.environ, and must never hand None or "" to the SDK: the OpenAI
        # client falls back to OPENAI_API_KEY when api_key is None.
        api_key = (key or "").strip()
        if not api_key:
            raise ValueError("An API key is required for custom OpenRouter models.")
        llm = _build_openrouter_llm(api_key, model_name)
        # tools_schema, ctx, aggregator: same three lines as the platform_openrouter branch
```
(`key`가 `None`으로 오는 것은 정상 경로: `pythonrunner.go:87` `omitempty`, `main.py:85,129`.) Custom 분기에서 provider 옵션을 제거하거나 재시도 시 제외하는 코드는 두지 않는다(ZDR 실패 시 비 ZDR 재시도 금지). 콜론 형태 `OpenRouter:x/y`는 점이 없어 조건이 거짓이므로 `Unsupported LLM service` 그대로(Go resolver의 정확 일치 접두가 1차 근거, 이 조건은 이중 방어).

**Tests (먼저)**
- `TestCustomOpenRouter`(`test_run.py`, `TestPlatformOpenRouter` 뒤): 키 정상 시 `OpenRouterLLMService`가 `api_key=<정제된 키>`로 1회 호출, `Settings.model`이 접두 뒤 전체 ID(점 포함 ID 보존 `mistralai/mistral-medium-3.1`), `extra == {"extra_body": {"provider": _OPENROUTER_PROVIDER}}`, **두 분기의 provider dict가 같은 상수에서 나옴**(`is` 대신 `==` + 한쪽 변이가 다른 쪽에 영향 없음 `dict()` 복사), 키 `None`/`""`/`"  "`/`"\n"`은 `ValueError`이고 `OpenRouterLLMService` 미호출(**이 행은 `patch.dict(os.environ, {"OPENROUTER_API_KEY": "dummy-or-env-not-real", "OPENAI_API_KEY": "dummy-openai-env-not-real"})`로 두 환경 키가 설정된 상태에서 단정**한다. 환경에 키가 없는 상태만 테스트하면 `os.getenv` 폴백이 있어도 통과할 수 있고, 환경 키가 있는 상태에서 `None`·`''`·`'  '`가 모두 예외여야 폴백이 없다는 증명이 된다. T24 실측: `OpenRouterLLMService(api_key=None)`은 `OPENAI_API_KEY`를 클라이언트 키로 쓰고, `''`는 빈 문자열, `'  '`는 공백 그대로 SDK에 전달된다), `OPENROUTER_API_KEY`와 `OPENAI_API_KEY`를 둘 다 설정한 `patch.dict(os.environ)` 환경에서 전달된 `api_key`가 환경 값이 아님, `type="openrouter.a/b"`와 `type="OpenRouter.a/b"`(소문자화 후 동일 분기)의 첫 점 분리, 콜론 형태는 `Unsupported LLM service`. **`:variant` ID 단위 행:** `type="openrouter.vendor/model:free"`(와 `:nitro`, `:online`)가 첫 점 분리와 `extra_body` 구성을 거쳐도 `Settings.model`에 **원문 그대로** 남는다(실호출 항목 7의 단위 부분). 변종과 라우터 ID의 **거부는 Go `ValidateCustomModelID` 단일 지점의 책임**이며(T2가 `:free`, `:nitro`, `:online`, `:thinking`, `:floor`, `openrouter/auto` 거부를 고정), Python 러너는 모델 ID를 재검증하지 않는다. 이 원칙을 테스트 주석에 영어로 남겨 Python에 검증을 추가하려는 변경이 두 계층의 불일치를 만들지 않게 한다.
- `test_raw_openrouter_is_unsupported`(`:1026-1034`): parametrize에서 점 형태 행(`openrouter.meta-llama/llama-3-70b`)을 제거하고 콜론 행(`OpenRouter:x/y`)만 유지(점 형태는 `TestCustomOpenRouter`로 이동).
- 팀(`test_init_pipeline.py`): `llm_type="openrouter.a/b"` + 키가 `create_llm_service`에 그대로 전달됨(`args[0]`, `args[1]`), `llm_type=""`는 여전히 예외(기존 `:804` 계열 테스트 불변).
- 변이 확인(수동 1회, CI 제외): 분기에 `or os.getenv("OPENROUTER_API_KEY")` 한 줄을 넣으면 위 테스트가 빨개짐을 확인하고 PR 설명에 기록(T25). 소스 문자열 정적 가드 테스트는 만들지 않는다.
- `conftest.py:55-63`이 pipecat 전체를 `MagicMock`으로 대체하므로 SDK 폴백 방어는 T24의 실제 SDK 실행으로 보완한다(`_real_openrouter_probe_script`(`:1049`) 방식의 서브프로세스 프로브에 `OPENAI_API_KEY` 설정 + 빈 키 케이스를 추가하는 방안을 구현 시 검토하고 불가하면 T24 항목 3, 5로 대체).

**Verify:** 1절 Python 명령. 기대: 새 실패 없음(기존 4건 동일), passed 증가. **Depends:** 없음(T6와 병행 가능). **Commit:** C5.

#### T8. `verify_openrouter_catalog.py` 불변 확인 (디자인 3.1.1 마지막 항)
스크립트와 `test_verify_openrouter_catalog.py`는 **변경하지 않는다**(Custom 항목은 `UpstreamSlug`가 없고 소유 불변 조건은 T1의 `Test_CatalogInvariants`). 확인 명령(`scripts/pipecat`에서, 전후 동일해야 함):
```
python3 - <<'E'
import verify_openrouter_catalog as v
s = v.parse_catalog_slugs(open('../../../bin-ai-manager/models/ai/catalog.go').read())
print(len(s), all(not x.startswith('openrouter') for x in s))
E
```
기대: `9 True`(T1 전후 동일). 이어서 1절 Python 명령으로 `test_verify_openrouter_catalog.py` 통과. **Commit:** C5(변경 없음, 확인 기록만).

---

### Phase C: 다른 서비스, OpenAPI (스트림 S3, monorepo)

#### T9. api-manager 키 로깅과 `engine_model` 반향 방어 (디자인 3.1.3 표 1행, 3.4 A)
**Files:** `bin-api-manager/pkg/servicehandler/ai.go`: `AICreate :36`의 `"engine_model" :64`, `"engine_key" :66`, `:121`(`log.WithField("ai", tmp)`); `AIUpdate`의 `:401`, `:403`, `:465`. 테스트 `pkg/servicehandler/ai_test.go`.
**Change:** `:66`, `:403`의 `"engine_key"` 줄 삭제. `:121`, `:465`를 `log.WithField("ai_id", tmp.ID)`로. 같은 파일에 비공개 헬퍼 1개: `func loggableEngineModel(m amai.EngineModel) string`이 `strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(m))), "openrouter.")`이면 `"openrouter.<redacted>"`, 아니면 `string(m)`(디자인은 대소문자 무시만 명시, 앞뒤 공백 허용은 더 안전한 상위 집합). `:64`, `:401`을 `"engine_model": loggableEngineModel(engineModel)`로.
**Tests:** `Test_loggableEngineModel` 순수 함수 표(`openrouter.a/b`, `OpenRouter.x`, `OPENROUTER.dummy-key-not-real`, ` openrouter.x`, `openrouter.`, `openai.gpt-5`, `""`). **편집 순서(2절 예외 표):** `pkg/servicehandler/ai_test.go`는 T1(C1)이 `:221-256`의 `ModelInfo` 행을 이미 수정한 파일이다. T9의 `ai_test.go` 수정은 **C1 커밋 이후에만** 시작하고(같은 파일 동시 편집 금지), 동시 진행이 불가피하면 `git add -p`로 C1에는 T1의 행만, C6에는 `Test_loggableEngineModel`만 담는다. `ai.go`는 T9만 편집한다.
**Verify:** `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... -count=1 && golangci-lint run -v --timeout 5m`. **Depends:** T1(재vendor 후). **Commit:** C6.

#### T10. call-manager 로깅 1줄 (3.4 A)
`bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:223`의 `log.WithField("ai", a)`를 `"ai_id", a.ID`로(같은 줄 `Debugf`가 이미 `a.ID`를 쓰므로 nil 위험 없음. 디자인의 `:222`는 오기). 전용 테스트 없음. **Verify:** `cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`. **Commit:** C6.

#### T11. **[분기 조건 있는 태스크]** 공용 `publish.go` + webhook-manager B4 (3.4 B3, B4, 8.1 항목 8)
**조건:** 대표님이 8.1 항목 8을 확정한다. **기본 분기 A(권장): 이번 PR에서 처리.** 분기 B는 별도 PR이라 대표님 허락이 있어야 하고, 분기 C는 처리하지 않고 위험 수용(6.3 게이트 2의 "공용 로깅 범위 결정 완료"는 충족으로 본다).
**분기 A 변경(총 10줄 = `publish.go` 1줄 + webhook-manager 9줄: `webhook.go` 2줄, `listenhandler/main.go` 1줄, `v1_webhooks.go` 6줄):**
- `bin-common-handler/pkg/notifyhandler/publish.go:34`의 `"data": data` 한 줄 삭제(`customer_id`, `event_type` 유지, 시그니처 불변). 출력은 `:46`, `:51` 오류 경로.
- `bin-webhook-manager/pkg/webhookhandler/webhook.go:33,126`(`"data": data`, 상시 `Debugf`) -> `"data_len": len(data)`(`data_type` 유지).
- `bin-webhook-manager/pkg/listenhandler/main.go:150`, `v1_webhooks.go:19,56`의 `"request": m` -> `"uri": m.URI, "method": m.Method, "request_id": m.RequestID`.
- `v1_webhooks.go:29,66`(`data: %v`)와 `:35,72`(`message: %v`)에서 인자와 문구 제거, `err`만 남김.
**Python 러너 로그(수용):** `bin-pipecat-manager/scripts/pipecat/main.py:108-119`의 `run_request` 로그는 `llm_type`(Custom이면 `openrouter.<author>/<slug>`)을 남긴다. Custom ID는 `customModelIDPattern`(영숫자, `.`, `_`, `-`, `/` 하나)으로 문자셋이 제한되어 키나 비밀값이 섞일 수 없으므로 **수용하고 변경하지 않는다**(`engine_key`는 이 로그에 없음).
**Tests:** 전용 회귀 테스트 없음(Q10 원칙). 기존 테스트가 로그 필드에 의존하지 않는지 전체 `go test`로만 확인. `m.RequestID` 필드명과 `len(json.RawMessage)` 컴파일을 확인한다(디자인 8.2).
**Verify:** `bin-common-handler`, `bin-webhook-manager`, `bin-conversation-manager`(비밀값 타입 호출처)에서 1절 Go 명령. **전 서비스 빌드 스모크**는 T23. **Commit:** C6.

#### T12. OpenAPI 스키마와 생성 코드 (3.2, 로직 변경 없음)
**Files:** `bin-openapi-manager/openapi/openapi.yaml`: `AIManagerAIModel :2018-2059`(`required :2021-2028`에 `key_mode` 추가, 속성 추가), `engine_key :2198`; **`AIManagerAIEngineModel :2013-2016`(설명 문구, 현재 "Uses target.model format (e.g., openai.gpt-5)...The list of selectable models is returned by `GET /ai_models`.")**; **`openapi/paths/ais/main.yaml:55-57`와 `paths/ais/id.yaml:95-97`의 `engine_key` 설명**(현재 "API key or credential for the AI engine."); `openapi/paths/ai_models/main.yaml:6`(설명 문구); 생성: `bin-openapi-manager/gens/models/gen.go`, `bin-api-manager/gens/openapi_server/gen.go`, `gens/openapi_redoc/{api.html,openapi.json}`.
**Change:** `key_mode`: string enum `[platform, own_or_default, own_required]`, `required`에 포함, description `How the engine key is handled. platform: the platform supplies access and engine_key is ignored. own_or_default: your engine_key is used, or the platform default key when empty. own_required: engine_key is mandatory.` `model_id_prefix`: string, optional, description `Present only for entries where you type the model ID yourself. Send engine_model as this prefix followed by the model ID.` `platform_managed` 설명 유지(Q6). **`AIManagerAIEngineModel` 설명에 한 문장 추가:** `openrouter.<author>/<slug>` 형태의 값도 허용되며(Custom 모델, `GET /ai_models`의 `model_id_prefix` 참조) 이 경우 `engine_key`가 필수다. **`paths/ais/main.yaml:55-57`, `id.yaml:95-97`의 `engine_key` 설명:** `API key or credential for the AI engine. Required when engine_model starts with openrouter. (400 ENGINE_KEY_REQUIRED if empty).`(영어, 대시 금지). `paths/ai_models/main.yaml:6`의 "Models with platform_managed set to true do not need an engine_key." 문구를 `key_mode` 기준 한 문장으로 보강. **`engine_key` 설명(`:2198` "Write-only; not returned in responses.")은 T24 항목 4에서 응답 포함이 확인된 경우에만 "returned"로 정정**한다(기본: 변경 없음).
**순서(생성 파일 재생성 원칙, 손 편집 금지):**
```
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
redoc(`api.html`, `openapi.json`)은 `npx`가 필요하다. 없으면 설치해서 재생성하고 **오래된 redoc을 조용히 커밋하지 않는다**.
**Tests:** api-manager `server/ai_models_test.go`, `servicehandler/ai_test.go`는 T1에서 이미 Custom 행을 갖는다. 여기서는 생성 코드에 `key_mode`가 들어갔는지 확인한다:
```
grep -n "KeyMode\|key_mode" bin-openapi-manager/gens/models/gen.go bin-api-manager/gens/openapi_server/gen.go | head
grep -c "key_mode" bin-api-manager/gens/openapi_redoc/openapi.json
git status --short bin-openapi-manager/gens bin-api-manager/gens   # 변경된 생성 파일이 모두 보임
```
`grep -rn AIManagerAIModel --include=*.go . | grep -v gens/` 결과는 비어 있어야 한다(서버가 `amai.ModelInfo`를 그대로 통과시킴, `server/ai_models.go`). **Depends:** T1. **Commit:** C7.

---

### Phase D: 문서 (스트림 S3, monorepo)

#### T13. RST, 도메인 문서, 디자인 문서 정오표 (3.5, Q11, MINOR 6, 7)
**Files(영어 문서는 영어로, 대시 금지):**
- `bin-api-manager/docsdev/source/ai_struct_ai.rst`: `engine_model :52`, `engine_key :54`(Custom 유형 추가: `engine_model`이 `openrouter.<model-id>`이면 필수, 비면 `400 ENGINE_KEY_REQUIRED`), **PUT 의미**(`PUT /ais/{id}`는 전체 교체라 `engine_key`를 생략하면 빈 값으로 처리되며, `openrouter.<model-id>` AI에 키를 생략하거나 비워 보내면 `400 ENGINE_KEY_REQUIRED`이므로 키를 바꾸지 않는 수정에서도 기존 키를 다시 보내야 함, Update 설명에 명시), 오류표 `:180`(`ENGINE_KEY_REQUIRED` 행 추가, `INVALID_ENGINE_MODEL` 행에 Custom 모델 ID 규칙 추가), 모델 표 `:196-208`(Custom 행), `:208` `platform_managed` 설명을 `key_mode` 설명과 함께 정리, `:212` 검증 문단에 `openrouter.<model-id>` 허용 규칙, ZDR 안내(모든 요청에 강제, ZDR 제공자가 없는 모델은 호출 실패. **음성 세션 한계 명시:** ZDR 제공자가 없는 모델로 음성 통화(`aicall`)를 시작하면 LLM 호출이 실패해 **통화가 무음이 될 수 있고**, 이 PR은 이 실패를 별도 고객 안내 문구로 분류하지 않는다(`pipelineerror.go` 분류기 확장은 6절 T26-c의 조건부 후속). 따라서 고객은 Custom 모델을 통화에 연결하기 전에 OpenRouter 모델 페이지에서 ZDR 제공자가 있는지 확인하거나 텍스트 채팅으로 먼저 시험해야 한다고 쓴다), 비용은 고객 OpenRouter 계정에 청구되고 플랫폼 키는 사용되지 않는다는 문구. `:77`("never returned")은 T24 항목 4 결과에 따를 때만 정정.
- `ai_models.rst`: 응답 예시에 `key_mode`, Custom 항목(`id: custom.openrouter`, `model_id_prefix: openrouter.`), 필드 목록(`:58-64`, 응답 예시 `:38,47`)에 `key_mode`, `model_id_prefix`, "Custom 항목의 `id`는 `engine_model`로 직접 쓸 수 없고 `model_id_prefix` + 모델 ID로 조합한다", 머리 note의 `INVALID_ENGINE_MODEL`에 `ENGINE_KEY_REQUIRED` 병기.
- `ai_tutorial.rst:14`(`platform_managed` 언급), **`ai_overview.rst:821`과 `ai_tutorial.rst:437`**(오류 원인 문단이 `INVALID_ENGINE_MODEL`을 "`GET /ai_models` 값 또는 `openai`/`gemini`/`grok` 접두"로만 서술. `openrouter.<model-id>` 허용 규칙과 Custom 키 누락의 `ENGINE_KEY_REQUIRED`를 추가), `self_hosting_providers.rst`는 **변경 없음**(BYOK는 플랫폼 키가 필요 없음, 디자인 1.2).
- `bin-ai-manager/docs/domain.md:12,111-127`(카탈로그, `ModelInfo` 필드, `Outcome` 표에 `OutcomeCustomOpenRouter`, `ValidateEngine`), `bin-pipecat-manager/docs/domain.md:78-79`(`classifySessionLLM`, Custom 키 전달, 오류 문구 정제). `scripts/check-service-docs.sh`(경고 전용)가 `models/` 변경에 `docs/domain.md`를 요구한다. 이 두 `docs/domain.md`는 S1(ai-manager), S2(pipecat)가 소유한 경로를 S3의 T13이 편집하는 예외다(2절 예외 표: S1, S2 커밋 이후에 수정).
- 디자인 문서 정오표(`docs/plans/2026-10-06-openrouter-byok-design.md`): **8.1에 항목 10 추가**: "같은 그룹 안 벤더 변경(OpenAI에서 Gemini) 시 입력란에 사용자가 직접 타이핑한 키는 저장 키가 없을 때(가드 false) 유지된다(create와 동일). 화면에서 보고 입력한 값이고 저장 키 유출이 아니므로 수용된 재존이다. | CPO 권장: 수용". **8.3 위험 표**: 기존 '키 이월 잔존 경로' 행 아래에 행 추가: "수용된 재존(타이핑한 키의 같은 그룹 내 벤더 변경) | 저장 키가 없는 AI에서 사용자가 타이핑한 키는 같은 그룹의 다른 벤더로 바꿔도 유지됨 | 수용(사용자 입력 값, create와 동일)". '`aicall` 스냅샷 모델과 현재 AI 키의 시점 불일치' 행의 영향 칸에 문장 추가: "BYOK 도입으로 openrouter.ai가 고객 키의 새 수신처가 되므로, 모델과 키를 함께 바꾼 직후의 기존 aicall 세션에서 다른 발급처 키가 openrouter.ai로 전송될 수 있다(스냅샷 한계는 Custom 빈 키에서만 fail-closed)." 이 문서는 승인된 디자인이라 **정오표 성격의 문장 추가만** 하고 규칙은 바꾸지 않는다(코드 리뷰 루프에서 함께 확인).
**빌드(F4, F19, 이전 기능의 절차를 `docs/plans/2026-10-05-openrouter-llm-routing-plan.md:349`와 실제 추적 파일로 대조함):** `bin-api-manager/docsdev/build/`는 gitignore이나 **838개 파일이 추적 중**이고(`git ls-files bin-api-manager/docsdev/build | wc -l`), 직전 기능 커밋 `bc68e3c32`도 RST와 함께 html 246개, doctrees 245개(`environment.pickle` 포함), `_sources` 9개, `objects.inv`, `searchindex.js`를 `git add -f`로 커밋했다. 기본 환경에는 sphinx가 없으므로(`python3 -m sphinx` 실패) 빌드 환경을 먼저 만든다(저장소 밖, `docsdev/requirements.txt`는 Sphinx 9.1.x가 Python 3.12 이상을 요구한다고 명시):
```
# python3 -m venv가 ensurepip 부재로 실패하는 머신이면 uv를 쓴다 (직접 확인: Python 3.13.7과 3.12.3 모두 동작)
uv venv --python 3.12 "$TMPDIR/docs-venv"
uv pip install --python "$TMPDIR/docs-venv/bin/python" -r bin-api-manager/docsdev/requirements.txt "pygments==2.18.0"
"$TMPDIR/docs-venv/bin/python" -c "import sphinx, sys; print(sphinx.__version__, sys.version.split()[0])"   # 9.1.x, 3.12 이상이어야 함
```
**`pygments==2.18.0` 고정이 핵심이다(직접 확인):** `requirements.txt`는 pygments를 고정하지 않아 최신(2.21.0)이 설치되면 변경 없는 재빌드에서도 `pygments.css` 해시와 코드 하이라이트로 html 180여 개가 바뀐다. 2.19.1, 2.19.2, 2.20.0도 같은 현상이고 **2.18.0에서만 html이 커밋본과 바이트 동일**했다. 순서:
```
cd bin-api-manager/docsdev
# (선택, 권장) RST 수정 전에 변경 없는 재빌드로 노이즈 기준선 확인: doctrees 245개만 달라야 한다
rm -rf build && "$TMPDIR/docs-venv/bin/python" -m sphinx -M html source build 2>&1 | tail -3     # 경고 0건, build succeeded
cd ../.. && git status --short bin-api-manager/docsdev/build | grep -v doctrees          # 비어 있어야 함(html, _sources, _static이 안 바뀜)
git checkout -- bin-api-manager/docsdev/build                                            # 기준선 확인 후 되돌림
# RST 수정 후 재빌드
cd bin-api-manager/docsdev && rm -rf build && "$TMPDIR/docs-venv/bin/python" -m sphinx -M html source build
cd ../.. && git add -f bin-api-manager/docsdev/build/
git diff --cached --stat bin-api-manager/docsdev/build | tail -1
git diff --cached --name-only bin-api-manager/docsdev/build | grep -v doctrees          # 의도한 문서의 html, _sources, searchindex.js(와 objects.inv)만 보여야 함
```
`rm -rf build`는 추적 파일을 지우므로 직후 `git status`에 삭제로 보이지만 재빌드가 같은 파일을 다시 만들어 diff는 실제 변경분만 남는다. **기대(실측 모의):** `ai_models.rst` 한 개를 고쳤을 때 doctrees 외 변경은 `html/ai_models.html`, `html/_sources/ai_models.rst.txt`, `html/searchindex.js` 세 파일뿐이었다. 의도하지 않은 html, `_static` 변경이 보이면 pygments 버전 또는 Python 버전 차이이므로 커밋하지 않고 환경을 맞춘다. Sphinx 경고가 새로 생기지 않는지 확인한다(스킬 `sphinx-rst-audit`, 기준선 경고 0건). **문서 단계에서 RST를 다시 고치면(리뷰 반영 포함) 반드시 위 재빌드와 `git add -f`를 다시 한다**(소스만 고치고 build를 놓치면 `docs.voipbin.net`에 반영되지 않는다).
**Verify:** 커밋 전에 `git add`로 변경을 스테이징한 **뒤** `bash scripts/check-docs.sh`와 `bash scripts/check-service-docs.sh`를 실행한다(`check-service-docs.sh`는 `git diff --cached`의 staged 파일만 검사하므로(스크립트 `:23-24`) 스테이징 전에는 항상 경고가 없다). **Depends:** T12. **Commit:** C8.

---

### Phase E: UI (스트림 S4, monorepo-javascript `square-admin`, 별도 PR)

UI는 Go 변경과 독립적으로 `aiModelsFixture.js`(서버 응답 모양)로 개발한다. 모든 태스크의 검증은 Jest(1절). 새 파일은 **테스트 픽스처 `keyGuardMatrix.fixture.js` 하나뿐**이다(F7). 고객 노출 문구는 영어 상수 한 곳(`useAIModels.js`의 `KEY_FIELD_COPY`, `KEY_WARNINGS`, 메시지 상수)에 둔다.

#### T14. 데이터 계층: 타입, 순수 함수, 훅, 픽스처 (디자인 3.6.1, F17, MINOR 5)
**Files:** `square-admin/src/types/api.ts:473-482`(`AIModel`), `src/views/ais/useAIModels.js`(상단 doc `:67`, 반환 `:93-97`; `isPlatformManaged :94`는 호환을 위해 **유지**), `src/views/ais/aiModelsFixture.js:2-10`, 테스트 `src/views/ais/useAIModels.test.js`(`Probe`, 현행 5개 테스트 유지).
**Change**
- `AIModel`에 `key_mode: 'platform' | 'own_or_default' | 'own_required'`, `model_id_prefix?: string`. 기존 `platform_managed` 주석 유지.
- `useAIModels.js`에 같은 파일의 **순수 함수**(내보내기) 추가: `servicePrefixOf(value)`(첫 `.` 앞, `.` 없으면 값 전체), `computeKeyMode(models, value)`(정확 일치 항목의 `key_mode`, 없으면 `model_id_prefix`를 가진 항목의 접두로 시작하면 그 항목의 `key_mode`, 그 외 `own_or_default`로 폴백하고 `platform_managed`로 폴백하지 않음. **`key_mode` 필드 자체가 없는 구 API 응답은 `platform_managed ? 'platform' : 'own_or_default'`**), `customIdMissing(models, value)`(값이 Custom 접두로 시작하고 접두 뒤가 비었거나 공백뿐), `computeGuardApplies({savedKey, savedModel, currentModel, models})`와 `keyTransitionAction(prev, cur)`(4절 판정 그대로), `engineErrorOf(err)`(`err.message`에 `ENGINE_KEY_REQUIRED` 포함이면 `{kind:'key', message: ENGINE_KEY_REQUIRED_MESSAGE}`, `INVALID_ENGINE_MODEL` 포함이면 `Details:` 뒤 본문을 `JSON.parse`해 `error.message`를 쓰고 실패 시 `The OpenRouter model ID is not valid.`로 폴백한 `{kind:'model', message}`, 그 외 `null`. `provider.js`는 수정하지 않음, 메시지 형식은 `provider.js:146` `Request failed (400): ... Details: <본문>`).
- 훅이 같은 이름의 바인딩 함수를 추가로 반환: `keyModeOf(value)`, `customEntry`(`model_id_prefix`를 가진 항목 또는 `undefined`), `customIdMissing(value)`. **모두 기존 `getModel`처럼 `useCallback`/`useMemo`(의존성 `models` 또는 `byId`)로 안정화한다**(`useAIModels.js:93-95` 선례). effect 의존성에 넣어도 매 렌더 재실행되지 않게 하기 위함이다(T17, T19, T20). 훅 반환의 다른 키는 그대로.
- 문구 상수(영어, 디자인 3.6.3): `KEY_FIELD_COPY = { platform: {label:'Engine Key', caption:'API key not required for this model.'}, own_or_default: {label:'Engine Key (optional)', caption:'Leave empty to use the platform default key.'}, own_required: {label:'OpenRouter API Key *', placeholder:'Your OpenRouter API key', notes:[<안내 1>, <안내 2>, <안내 3>]} }`(안내 1 `Calls use this key and are billed to your OpenRouter account. The platform key is never used.`, 안내 2 `Zero data retention (ZDR) routing is enforced on every request. If no ZDR provider supports this model, calls will fail.`, 안내 3 `The AI cannot be saved without an API key.`), `KEY_WARNINGS`(3종), `ENGINE_KEY_REQUIRED_MESSAGE`, `MODEL_ID_REQUIRED_MESSAGE`(`Enter the OpenRouter model ID.`), `MODEL_ID_INVALID_FALLBACK`. 플레이스홀더는 `platform`/`own_or_default`에서 **현행 유지**(create/detail `AI Engine API Key (e.g., sk-...)`는 테스트가 `/ai engine api key/i`로 찾음).
- `aiModelsFixture.js`: 기존 8개에 `key_mode` 추가, Custom 항목(`id: 'custom.openrouter'`, `vendor: 'OpenRouter'`, `key_mode: 'own_required'`, `model_id_prefix: 'openrouter.'`, 라벨/설명은 3.6.3) 추가. `MOCK_AI_MODELS`를 쓰는 소비자가 항목 수/그룹에 의존하지 않는지 **실행으로** 확인한다. 구체 명령(직접 확인한 소비자 12개): `cd square-admin && grep -rln "aiModelsFixture\|MOCK_AI_MODELS" src | sort`(`TestAgentSheet.test.js`, `aicalls_detail.test.js`, `aicalls_list.test.js`, `AIEngineFields.test.js`, `ModelPicker.test.js`, `ais_create.test.js`, `ais_detail.test.js`, `ais_list.test.js`, `InsightAIsPanel.test.js`, `useAIModels.test.js`, `member.test.js`, `sidebar_engine_model.test.js`)가 출력한 파일 전체를 `CI=true npm test -- --watchAll=false --runTestsByPath <그 파일들>`로 실행한다. 항목 수를 직접 쓰는 단정은 `useAIModels.test.js:38`(`String(MOCK_AI_MODELS.length)`, 같은 상수에서 길이를 읽으므로 항목 추가에도 통과)뿐이고, 나머지는 그룹과 이름 조회이므로 그룹 조회를 바꾸는 T15, T16, T19에서 함께 고친다.
- `keyGuardMatrix.fixture.js`(신규, 4절 표를 데이터로). 상수 `OPENAI_ALT`는 픽스처에 이미 있는 `openai.gpt-5-mini`를 쓴다.
**Tests (먼저, 순수 함수 표 위주):** `computeKeyMode`(정확 일치 3그룹, `openrouter.a/b`는 own_required, 카탈로그에 없는 `openai.gpt-4o`는 own_or_default, `key_mode` 없는 구 응답 폴백), `servicePrefixOf`(`openai.gpt-5`, `openrouter.a/b`, 점 없음, 빈 값), `customIdMissing`(접두만, 접두 + 공백, 접두 + ID, 비 Custom), `computeGuardApplies`(저장 키 없음, 같은 그룹 같은 접두, 같은 그룹 다른 접두, platform 접두 불일치 `anthropic`/`meta`), `keyTransitionAction`(4절 모든 전이: false->true, true->false가 그룹 변경보다 우선, 가드 true인 채 그룹 변경, 변화 없음), `engineErrorOf`(키 오류, 모델 오류 JSON 파싱 성공/실패 폴백, 무관한 오류 `null`), 훅(`keyModeOf`, `customEntry`, 카탈로그 로딩 중 폴백).
**Verify:** T14 단독 시점에는 `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/useAIModels.test.js`만 초록을 요구한다(Custom 픽스처 때문에 `ModelPicker.test.js:168` 등이 이 시점에는 빨갛다). fixture 소비자 전체 `CI=true npm test -- --watchAll=false src/views/ais src/views/teamgraph src/views/aicalls src/components`(위 grep 목록 포함)는 **T15까지 적용한 J1 시점**에 실행한다; 변경 파일 `npx eslint src/views/ais/useAIModels.js src/views/ais/aiModelsFixture.js src/views/ais/useAIModels.test.js src/views/ais/keyGuardMatrix.fixture.js`(1절 UI 검증 2번, 기준선 대비 증가 없음).
**Depends:** T0. **Commit:** J1.

#### T15. ModelPicker: 키 방식 그룹, 배지, Custom 행과 모델 ID 입력란 (3.6.2, Q1a, Q3, 제안 a, i, n)
**Files:** `src/views/ais/ModelPicker.js`(전체 `:1-249`): `VENDOR_ORDER :7`, `groupByVendor :17-31`, 헤더 주석 `:33-47`, 상태 `:48-57`, `showCurrentRow :63`, `pick :70-73`, `renderRow :75-106`(`aria-selected :80`, 체크 `:88`), 트리거 `:108-119`, `PopoverContent :143`, 그룹 렌더 `:197-217`. 테스트 `ModelPicker.test.js`.
**Change**
- 벤더 그룹 대신 키 방식 그룹 3개 `PLATFORM PROVIDED`(`platform`), `YOUR OWN KEY`(`own_or_default`), `CUSTOM`(`own_required`)를 `computeKeyMode`(서버 `key_mode`만 사용, 클라이언트 하드코딩 없음)로 구성. 빈 그룹 숨김, 그룹 내 순서는 카탈로그 순서, `Recommended` 섹션 유지(Gemini 2.5 Flash도 같은 배지), 접기는 그룹 키로(`collapsed`의 키를 벤더명에서 그룹 키로). 그룹 `aria-label`은 그룹 제목. `VENDOR_ORDER`, `groupByVendor` 삭제. 벤더는 행 보조 표기.
- 배지 상수 `KEY_MODE_BADGES = { platform: 'No key needed', own_or_default: 'Your key or default', own_required: 'Your key required' }`를 `Low cost` 태그 옆에 표시.
- **선택값 판정:** `selected = getModel(value) ?? (customEntry && value.startsWith(customEntry.model_id_prefix) ? customEntry : undefined)`. 트리거에 Custom 라벨과 벤더 표시, 합성 행 `Current: <id>`는 Custom 값에서 만들지 않음(`showCurrentRow`에 `!isCustomValue` 추가, `:63`). Custom 행의 `aria-selected`와 체크는 `value`가 접두로 시작할 때 true.
- **Custom 행 선택:** 값이 이미 접두로 시작하면 `onChange(value)`(타이핑한 ID 유지), 아니면 `onChange(prefix)`. 선택할 때마다 `focusIdRef.current = true`. `<PopoverContent onCloseAutoFocus={(e) => { if (focusIdRef.current) { e.preventDefault(); idInputRef.current?.focus(); focusIdRef.current = false } }}>`로 Radix가 닫힘 시 트리거로 되돌리는 포커스와의 경합을 막는다(`components/ui/popover.jsx`는 props를 Radix `Content`에 그대로 전달함, 확인함). 비 Custom 선택은 기본 동작(트리거 복귀).
- **ID 입력란:** `Popover` 뒤에 Fragment로 조건부 블록(`customEntry`가 있고 `value`가 접두로 시작할 때): 라벨 `OpenRouter Model ID *`(`useId`로 `htmlFor`), `Input`(`font-mono`, `compact`이면 `h-8 text-xs`), 도움말 `Enter the model ID from the OpenRouter model page, e.g. vendor/model-name.`, 오류 문구. 값 소유자는 부모의 `engineModel`/`createModel` 문자열 하나(`ModelPicker`는 새 상태 없음). 입력란 값은 `value.slice(prefix.length)`, 변경 시 `onChange(prefix + 입력값)`, **입력 중 변환과 trim 없음**.
- **`error` prop(문자열, 선택):** Custom 값이면 ID 입력란 아래, 그 외에는 트리거 아래에 `text-destructive`로 표시. 오류 전달 경로는 이 하나(T16).
- 헤더 주석 `:36-38`을 `free-text entry exists only for the custom entry (a model ID typed after its prefix)`로 갱신. `CatalogStatus :234-247`은 변경 없음.
**Tests (5.1과 5.2 UI 항목)**
- **같은 커밋(J1) 수정, 그룹 조회 2줄 이동(M-B):** `src/views/teamgraph/__tests__/sidebar_engine_model.test.js:196`(`getByRole('group', { name: 'Anthropic' })` -> `'PLATFORM PROVIDED'`, Claude Sonnet 4.5)과 `:257`(`{ name: 'Meta' }` -> `'PLATFORM PROVIDED'`, Llama 3.3 70B Instruct). 벤더 그룹이 키 방식 그룹으로 바뀌는 순간 이 두 줄이 빨개지므로 T19(J2)까지 미루면 J1 커밋이 빨간 상태로 남는다(직접 확인: 이 두 줄 외에 `getByRole('group', ...)`로 벤더 그룹을 조회하는 곳은 `ModelPicker.test.js:65-132`와 `AIEngineFields.test.js:325`(T16)뿐). 같은 파일의 캡션 단정(`API key not required`)은 `:260`(생성 폼)이 T19가, `:184,:200`(편집 폼)이 T20이 각각 캡션을 바꾸기 전까지 유효하므로 **J1에서는 건드리지 않는다**. J1과 J2를 합치는 안은 변경 범위가 훨씬 커서 채택하지 않았다.
- **수정(F20): `ModelPicker.test.js:168`의 `has no Custom entry`**(`queryByText(/custom/i)`가 없어야 한다는 단정)는 `CUSTOM` 그룹 헤더와 Custom 행 때문에 빨개진다. `shows the Custom entry in the CUSTOM group`(`group` 이름 `CUSTOM` 안에 Custom 행이 있음)으로 의미를 바꾼다. 디자인 5.1 표에는 이 줄이 없으므로 이 plan의 정오표(F20)로 보완한다.
- 수정: `ModelPicker.test.js:60-85`(그룹 순서 `['Recommended','PLATFORM PROVIDED','YOUR OWN KEY','CUSTOM']`와 그룹 내 행 조회, `:72-73,81`), `:87-100,132`(`Google` 그룹 조회를 `YOUR OWN KEY`로), `:109-115`(접기를 키 방식 그룹 헤더 버튼으로), 트리거 라벨/벤더 `:39-44`는 변경 없음 확인.
- 신규: 배지 3종이 행에 표시, Recommended 행도 배지, Custom 행 선택 시 `onChange('openrouter.')`, 이미 `openrouter.vendor/model-a`일 때 Custom 행 재선택은 `onChange('openrouter.vendor/model-a')`로 입력을 유지, 저장값 `openrouter.vendor/model-a`가 Custom으로 복원되고 `Current:` 행이 **없음**, `key_mode` 없는 구 API 응답의 폴백 그룹, Custom 선택 시 ID 입력란이 트리거 아래 렌더되고 타이핑이 `onChange(prefix + typed)`로 전달됨(대소문자와 공백 변환 없음), `error` prop이 입력란 아래에 표시, 비 Custom 값에서는 입력란 없음, **Custom 행을 처음 고를 때와 재선택할 때 모두 `document.activeElement`가 ID 입력란**(Popover 닫힘이 트리거로 되돌리는 것을 `onCloseAutoFocus`가 막음), 비 Custom 선택은 트리거 복귀. jsdom에서 Radix 포커스 동작이 재현되지 않으면 `onCloseAutoFocus` 핸들러가 `preventDefault`를 호출하는지를 단위로 단정한다(디자인 8.2).
**Verify:** `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/ModelPicker.test.js`; 이어서 **`CI=true npm test -- --watchAll=false src/views/ais src/views/teamgraph`**(J1 경계에서 `sidebar_engine_model.test.js`가 초록인지 포함 확인); 변경 파일 `npx eslint`(1절 2번). **Depends:** T14. **Commit:** J1.

#### T16. AIEngineFields: `modelError` 단일 경로 (3.6.2 제안 i, F15)
**Files:** `src/views/ais/AIEngineFields.js`: `ModelPicker` 호출 `:143`(compact)과 `:340`(non-compact), `modelError` 블록 `:145-147`과 `:343-345`, props 주석 `:93`, 상단 주석 `:72-76`.
**Change:** 두 `ModelPicker` 호출에 `error={modelError}`를 넘기고, **두 곳의 `{modelError && ...}` 블록을 모두 제거**(중복 표시 방지). 상단 주석의 소비자를 `ais_create.js`, `ais_detail.js`로 정정(F15).
**Tests:** `AIEngineFields.test.js:292,312`(picker not free-text 단정은 Custom이 아닌 값에서 유지), `:325`(`getByRole('group', { name: 'OpenAI' })`를 `YOUR OWN KEY` 그룹으로), `modelError`가 입력란 아래에 **한 번만** 나타남(두 레이아웃 모두), Custom ID 입력 테스트는 `ModelPicker.test.js`로 이동.
**Verify:** `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/AIEngineFields.test.js`; 이어서 **`CI=true npm test -- --watchAll=false src/views/ais src/views/teamgraph`**(`modelError` 표시 경로가 바뀌어 `ais_create`/`ais_detail`/`sidebar`가 영향받는지 확인); 변경 파일 `npx eslint`. **Depends:** T15. **Commit:** J1.

#### T17. ais_create: 키 방식, 검증, 서버 오류, 전이 비움 (3.6.3~3.6.5, MINOR 3)
**Files:** `src/views/ais/ais_create.js`: `engineKeyManaged :50-51`, `ref_engine_key :89`, `handleTemplateSelect :133-144`(`:137`), `applyDraft :160`, `CreateResource :178`(body `:201`, 검증 `:229`, catch `:248-252`), `AIEngineFields` 호출 `:545-571`, 키 카드 `:574-586`, `submitDisabled :610`. 테스트 `__tests__/ais_create.test.js`(mock `:38-`, `prompt_templates` mock `:69`, 캡션 `:311,318`, 플레이스홀더 `:282-288`).
**Change**
- `engineKeyManaged`(불리언)를 `keyMode = keyModeOf(engineModel)`로 일반화(`isPlatformManaged` 호출 제거). 키 카드: 라벨/캡션/플레이스홀더를 `KEY_FIELD_COPY[keyMode]`로(`platform`은 입력 disabled, 라벨 `Engine Key`, 캡션 `API key not required for this model.`). `own_required`는 라벨 `OpenRouter API Key *`, 안내 3줄, 키 오류 문구를 입력란 바로 아래 `text-destructive`로.
- **전이 비움은 effect로만 구현한다(MINOR 3):** 모델이 바뀌는 경로는 `ModelPicker`(`onEngineModelChange={setEngineModel}`)뿐 아니라 템플릿 선택(`handleTemplateSelect :137`)과 Builder 적용(`applyDraft :160`)처럼 핸들러를 우회하는 프로그램 변경이 있으므로 핸들러 래핑은 금지다.
```js
const prevCreateKeyModeRef = useRef(null)
useEffect(() => {
  if (modelsStatus !== 'ready') return
  if (prevCreateKeyModeRef.current !== null && prevCreateKeyModeRef.current !== keyMode && ref_engine_key.current) {
    ref_engine_key.current.value = ''   // group changed: never carry a typed key across key modes
  }
  prevCreateKeyModeRef.current = keyMode   // first ready run only records
}, [modelsStatus, keyMode])
```
같은 그룹 안 벤더 변경과 Custom ID 타이핑 중에는 비우지 않는다(create에는 가드가 없음). 위 의존성 배열은 예시이며 **구현 시 `react-hooks/exhaustive-deps`에 맞게 effect가 읽는 모든 값을 나열한다**(`ref_engine_key`, `prevCreateKeyModeRef`와 상태 setter는 제외 대상).
- **저장 검증(`CreateResource` 맨 앞, 디자인 3.6.4):** (0) `model = engineModel.trim()`을 Custom 접두에서 검증과 전송에 사용(입력 중에는 건드리지 않음), (1) `customIdMissing(engineModel)`이면 `MODEL_ID_REQUIRED_MESSAGE`를 `modelError`(AIEngineFields에 전달)로 표시하고 저장 중단, (2) `keyMode === 'own_required'`이고 `ref_engine_key.current.value.trim() === ''`이면 `ENGINE_KEY_REQUIRED_MESSAGE`를 키 입력란 아래에 표시하고 저장 중단(`ProviderPost` 미호출). body: `engine_key: keyMode === 'platform' ? '' : ref_engine_key.current.value`(원문, trim 통일은 범위 아님).
- 서버 오류(`:248-252`): catch가 오류를 버리던 것을 `engineErrorOf(e)`에 넘겨 `kind`가 `key`면 키 오류, `model`이면 `modelError`로 인라인 표시하고 그 외는 현행 `Could not create a new ai info.`. 입력이 바뀌면 해당 인라인 오류를 지운다.
**Tests (mock에 전환 버튼 추가: `set-custom-engine-model`(`openrouter.vendor/model-a`), `set-openai-engine-model`(`openai.gpt-5`), `set-gemini-engine-model`; `prompt_templates` mock에 platform/Custom 모델을 가진 템플릿 추가):** 캡션 `API key not required for this model.`로 갱신(`:311,318`), 키 방식별 라벨/캡션 3종, `own_required` 키 비어 있으면 인라인 오류와 `ProviderPost` 미호출, 모델 ID 비어 있음 오류, 앞뒤 공백이 있는 Custom ID가 trim되어 전송, 서버 `ENGINE_KEY_REQUIRED`/`INVALID_ENGINE_MODEL` 응답이 인라인에 표시(`Error('Request failed (400): Bad Request. Details: {...}')`), JSON 파싱 실패 시 폴백 문구, **전이 비움(effect)**: 모델 선택으로 그룹이 바뀌면 입력란 비움, 같은 그룹 안 벤더 변경과 Custom ID 타이핑 중에는 유지, **템플릿 선택과 Builder 적용으로 그룹이 바뀌어도 비움**(핸들러가 아닌 effect 증명), 첫 `ready` 렌더에서는 비우지 않음, 카탈로그 로딩 후 첫 실행은 기록만.
**Verify:** `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/__tests__/ais_create.test.js` 후 `CI=true npm test -- --watchAll=false src/views/ais`; 변경 파일 `npx eslint`(`ais_create.js` 기준선 0/0, 증가 없음). **Depends:** T14~T16. **Commit:** J2.

#### T18. ais_detail: 키 방식, 가드, 전이 effect, 행렬 (3.6.3~3.6.5, MINOR 2, 4, 5)
**Files:** `src/views/ais/ais_detail.js`: `fetchedDetail/isLoading :56-57`, `engineModel/engineKey :78-79`, `engineKeyManaged :86-87`, `detailData :160-167`, 폼 초기화 effect `:199-266`(`setEngineKey :203`, `setEngineModel :209`), `ref_engine_key :180`, `Update :396-`(body `:422`, catch `:454-457`), `hasUnsavedChanges`(`:541`, deps `:566`), `if (isLoading) :598`, `AIEngineFields` 호출 `:1264-1267`, 키 카드 `:1292-1310`(입력 `defaultValue :1305`). 테스트 `__tests__/ais_detail.test.js`(mock `:84-105`, 플랫폼 계약 `:427-448`, 캡션 `:418,432`).
**Change**
- `engineKeyManaged`를 `keyMode`로 일반화. `keyOutside` 개념 없음(detail 키 입력은 항상 보이는 카드). `guardApplies = computeGuardApplies({savedKey: detailData?.engine_key || '', savedModel: detailData?.engine_model || '', currentModel: engineModel, models})`(저장 기준선은 `detailData`, 값은 `fetchedDetail`과 같음, 디자인 3.6.5(c)).
- **전이 effect 위치와 게이트(MINOR 2):** 새 상태 `formSynced`(초기 `false`)를 **폼 초기화 effect(`:199-266`)의 맨 앞(조기 반환 분기보다 앞)에서 `setFormSynced(false)`로 내리고, 실제 폼 값 설정(`setEngineKey`, `setEngineModel` 등)이 끝나는 마지막 줄에서 `setFormSynced(true)`로 올린다.** 같은 effect 실행 안의 `false` -> `true`는 React가 한 번에 배치 처리하므로 **중간 렌더가 없고** 정상 경로에서 `formSynced`는 곧바로 `true`로 반영된다. 따라서 이 상태가 실제로 막는 것은 (a) 첫 마운트에서 폼 초기화 effect가 돌기 전(`engineModel === ''`, `formSynced` 초기값 `false`)과 (b) 초기화 effect가 데이터 없음으로 조기 반환해 `false`로 남는 구간(id 전환 직후 새 데이터 로딩 중)이다. 이 두 구간에서 전이 effect가 낡은 상태로 판정하지 않는다(조기 반환 위치는 구현 시 `:199-266`을 열어 확인). 폼 초기화 effect 시작에서 `prevGuardRef.current = null`. 전이 effect는 **`isLoading` 조기 반환(`:598`) 앞**(훅 순서 규칙)에 둔다: 
```js
useEffect(() => {
  if (modelsStatus !== 'ready' || !detailData || !formSynced) return   // engineModel is stale ('') until the init effect ran
  const cur = { keyMode, guardApplies }
  const prev = prevGuardRef.current ?? { keyMode: keyModeOf(detailData.engine_model), guardApplies: false }
  const input = ref_engine_key.current                                    // null before the card mounts
  const action = keyTransitionAction(prev, cur)
  if (input && action === 'clear') input.value = ''
  if (input && action === 'restore') input.value = detailData.engine_key || ''
  prevGuardRef.current = cur                                              // record even when input is null
}, [modelsStatus, formSynced, detailData?.id, keyMode, guardApplies])      // list the rest per react-hooks/exhaustive-deps
```
  폼 초기화 전에 `engineModel === ''`로 `guardApplies`가 일시적으로 참이 되는 오판정(위 (a), (b))을 `formSynced`가 막는다. 의존성은 예시이며 **구현 시 `react-hooks/exhaustive-deps`에 맞게 `keyMode`, `guardApplies`, `detailData`를 포함해 모두 나열한다**.
- **Update():** 검증 0~2는 3.6.4(키는 `const typed = ref_engine_key.current?.value ?? ''`, 유효 키는 `typed.trim() !== ''`). body: `keyMode === 'platform'` -> `detailData.engine_key || ''`(현행 재전송 계약 유지, 제안 v), `guardApplies` -> `typed.trim() !== '' ? typed : ''`(**저장 키 재전송 금지**), 그 외 `typed`. 가드 적용 + 목적지 `own_required` + 입력 없음이면 `KEY_WARNINGS.own_required`(`Replace the API key before saving.`)를 키 오류로 표시하고 `ProviderPut` 미호출. catch는 `engineErrorOf(err)`로 키/모델 인라인 오류, 그 외 현행 문구. 경고는 `guardApplies`일 때 입력란 아래에 목적지별 문구(detail의 `platform` 목적지에는 **표시하지 않음**, 제안 r).
- `hasUnsavedChanges :541`: `!engineKeyManaged &&`를 `keyMode !== 'platform' &&`로, 의존성 `:566`에 `keyMode` 반영. `AIEngineFields`에 `modelError` 전달(`:1264`).
**Tests**
- **mock 확장(MINOR 5):** `jest.mock('../AIEngineFields', ...)`(`:84-105`)에 전환 버튼 `set-custom-engine-model`(`CUSTOM`), `set-custom-alt-engine-model`(`CUSTOM_ALT`), `set-openai-engine-model`(`OPENAI`), `set-openai-alt-engine-model`(`OPENAI_ALT`), `set-gemini-engine-model`(`GEMINI`), 기존 `set-managed-engine-model`을 `PLATFORM`, `set-platform-alt-engine-model`(`PLATFORM_ALT`)와 함께 두고 `modelError` 표시 span을 추가한다. 기존 `set-engine-model`(`gemini.gemini-2.0-flash`, 카탈로그 밖)은 유지. 카탈로그 상태는 mock이 아니라 실제 `useAIModels`가 `ProviderGet('ai_models')`(픽스처)로 공급하므로 `beforeEach`(이미 `:171`에서 `resetAIModelsCache()` 호출)에서 sidebar 테스트와 같은 `MOCK_AI_MODELS_RESPONSE`를 쓴다(import는 이미 `:118-119`).
- `test.each(KEY_GUARD_ROWS.filter(r => !r.only || r.only === 'detail'))`(4절). 19d는 `ProviderGet('ai_models')`를 지연시켜 ready 전에 타이핑하고 전환한다.
- 기존 `:435-448`(platform 전환 후 저장 키 재전송), `:427-433`/`:418` 캡션 갱신. 카탈로그 공급 확인(라인 정정): `resetAIModelsCache`와 `MOCK_AI_MODELS_RESPONSE`의 import는 이미 `ais_detail.test.js:118-119`에 있고 `resetAIModelsCache()` 호출도 이미 `:171`에 있으므로(`:125`은 `makeStore` 안이라 카탈로그와 무관) 새로 넣을 것은 `ProviderGet('ai_models')` 응답을 픽스처로 돌려주는지와 19d의 지연 응답뿐이다.
- **id 전환 테스트(R20 m2):** 다른 AI id로 전환해 새 데이터가 로드된 뒤 입력란이 **그 AI의 저장 키**(`dummy-old-key`)인지 확인한다(낡은 상태로 전이 판정되어 비워지지 않음).
- **effect 안전성:** 정상 로드(모델 변경 없음)에서 입력란이 비워지지 않음(저장 키 `dummy-old-key`가 입력란에 그대로), 로드 전 `ref_engine_key.current === null`에서 effect가 예외 없이 `prevGuardRef`만 기록, 폼 초기화 전 일시 상태에서 비우지 않음.
**Verify:** `CI=true npm test -- --watchAll=false --runTestsByPath src/views/ais/__tests__/ais_detail.test.js` 후 `CI=true npm test -- --watchAll=false src/views/ais`; 변경 파일 `npx eslint src/views/ais/ais_detail.js src/views/ais/__tests__/ais_detail.test.js`(기준선 0 errors 1 warning, 4 errors, 증가 없음). **Depends:** T14~T16. **Commit:** J2.

#### T19. sidebar 생성 폼 (3.6.2~3.6.5)
**Files:** `src/views/teamgraph/sidebar.js`: `useAIModels :367`, `createModel/createEngineKey :378,386`, `createKeyManaged :387`, `handleSelectTemplate :420-441`(`setCreateEngineKey('')`는 `:432`), `handleCreateSubmit :510-562`(body `:527`, catch `:557-558`, deps `:562`), 생성 버튼 `:892-899`, 오류 배너 `:905-910`, `ModelPicker :945`, 키 `<details> :961-973`(입력 `:969`, 캡션 `:970`). 테스트 `sidebar_engine_model.test.js`, `sidebar_ai_type.test.js`, `sidebar.test.js`.
**Change:** `createKeyManaged`를 `createKeyMode = keyModeOf(createModel)`로. 키 블록을 JSX 지역 변수 `createKeyBlock`로 만들고(`KEY_FIELD_COPY[createKeyMode]`, `own_required`는 라벨 `OpenRouter API Key *` + 안내 3줄 + 키 오류) `const createKeyOutside = createKeyMode === 'own_required'`이면 `ModelPicker`(Custom ID 입력란 포함) 아래 **`<details>` 밖**, 아니면 현행 접힌 `<details>` 안에 렌더(`Advanced` 요약은 밖일 때 렌더하지 않음, 새 컴포넌트 없음). `ModelPicker :945`에 `error={createModelError}`. **전이 비움은 새 effect로만(MINOR 3):** `prevCreateKeyModeRef`(초기 `null`)와 `useEffect(..., [modelsStatus, createKeyMode])`(의존성은 구현 시 `react-hooks/exhaustive-deps`에 맞게 effect가 읽는 값을 모두 나열)가 `modelsStatus === 'ready'`이고 이전 값과 다를 때만 `setCreateEngineKey('')`(sidebar 생성에는 모델 변경 시 키를 비우는 기존 지점이 없다, 확인함: `:387,527,970`과 템플릿 선택 `:432`뿐. `handleSelectTemplate :432`의 `setCreateEngineKey('')`는 유지). 검증 0~2: `createModel.trim()`, `customIdMissing`, `own_required` + `createEngineKey.trim() === ''`. 실패 문구는 인라인과 **`setCreateError(msg)` 배너에 함께**(다른 탭에서 보이게, 제안 h). body `:527`: `createKeyMode === 'platform' ? undefined : (createEngineKey.trim() || undefined)`. catch는 `engineErrorOf`로 인라인 + 배너, 그 외 `err?.message || 'Failed to create assistant'`.
**Tests:** `sidebar_engine_model.test.js`의 **생성 폼 캡션 단정 `:260`만** 갱신(`API key not required` -> `API key not required for this model.`, 생성 폼 테스트 `:249-267`, 캡션 코드 `sidebar.js:970`). **`:184,:200`은 편집 폼 테스트(`:178-185`, `:187-207`, 편집 캡션 코드 `sidebar.js:1365`)이므로 T19가 아니라 T20에서 갱신한다**(직접 확인: 파일 `:145-225`가 편집 폼 describe, `:227-`부터 생성 폼 describe). `:196,257` 그룹 조회의 `PLATFORM PROVIDED` 전환은 J1(T15)에서 이미 끝났다(Claude Sonnet 4.5, Llama 3.3 70B 모두 `platform`), `:206`(`expect(body).not.toHaveProperty('engine_key')`)은 **변경 없음(platform 계약 고정)**, 생성 폼: `own_required`에서 `OpenRouter API Key *`가 `<details>` 밖이고 `Advanced` 요약이 없음(jsdom의 `details` 속성 동작은 단정하지 않음), 다른 키 방식이면 키 입력이 접힌 `Advanced` 안, 필수 키 비어 있을 때 인라인 + 배너(Engine이 아닌 탭에서 저장해도 배너 보임), 서버 오류 인라인 + 배너, **전이 비움 effect**(그룹 전이에서만, 같은 그룹 안 벤더 변경은 유지, 템플릿 선택은 기존 비움과 충돌 없음).
**Verify:** `CI=true npm test -- --watchAll=false src/views/teamgraph`; 변경 파일 `npx eslint src/views/teamgraph/sidebar.js src/views/teamgraph/__tests__/sidebar_engine_model.test.js`(기준선: `sidebar.js` 0 errors 2 warnings, 테스트 5 errors, 증가 없음). **Depends:** T14, T15. **Commit:** J2(T20과 한 커밋). **작업 순서 규칙:** T20이 같은 `sidebar.js`, `sidebar_engine_model.test.js`를 편집하므로 **T19를 먼저 끝내고(위 Verify가 초록인 상태 확인, 커밋은 하지 않음) T20에 착수한다**(4스트림 병렬 중에도 T19와 T20은 한 사람이 순서대로). 편집 폼 캡션 단정 `:184,:200`이 T20 귀속이므로 T19 종료 시점에도 `src/views/teamgraph`가 초록이다.

#### T20. sidebar 편집 폼: 재조회, 가드, 전이 effect, `<details>` 밖 렌더 (3.6.4~3.6.6, Q9)
**Files:** `sidebar.js`: `aiData :570`, `engineKey/engineKeyChanged :579-580`, `engineKeyManaged :581`, `saveStatus/saveError :614-615`, 로드 effect `:618-644`(캐시 히트 `:623-624`, 캐시 미스 `ProviderGet :630`, `.catch :643`), 초기화 effect `:646-717`(`setEngineKey('')`/`Changed(false) :656-657`, `setSaveError('') :716`), `handleSave :766-839`(body `:785-805`, 키 `:806-809`, 캐시 갱신 `:819-824`, catch `:832-836`, deps `:839`), 저장 버튼 `:1166-1175`(`disabled={isSaving}`), 오류 배너 `:1247-1252`, `ModelPicker :1302`, 키 `<details> :1340-1371`(마스킹 값 `:1350`, `onFocus :1356-1359`, 캡션 `:1364-1368`). 테스트 `sidebar_engine_model.test.js`(`:162,173,202,218-219` 동기 Save), `sidebar_ai_type.test.js:140-142,226`, `sidebar.test.js`.
**Change**
1. **재조회(기존 로드 effect 확장, 새 effect 없음, 중복 GET 없음):** 새 상태 `savedEngineKey`(`''`), `savedEngineModel`(`''`), `keyLoad`(`'loading'|'ready'|'error'`), 참조 `keySeqRef`(요청마다 증가), `formSynced`. effect 시작에서 `keyLoad='loading'`, 두 saved 상태 `''`, `formSynced=false`, `prevGuardRef.current=null`, `keySeqRef.current++`, cleanup에서도 `keySeqRef.current++`. **캐시 미스**: 기존 `ProviderGet` 응답(`engine_key`, `engine_model` 포함)으로 `savedEngineKey`/`savedEngineModel`을 채우고 `keyLoad='ready'`, 추가 GET 없음. 응답이 falsy이면 `keyLoad='error'`(기존 `if (res)`는 falsy를 조용히 넘기므로 명시). 실패는 기존 `.catch`(`setAiData(null)`)에 `keyLoad='error'`를 더한다(폼이 숨겨짐, `:1079`의 `!hasAI || !aiData` 분기). **캐시 히트**일 때만 `setAiData(stored[ai_id])` 뒤에 `ProviderGet('ais/' + ai_id)`를 1회 호출해 같은 두 상태를 채운다(**`setAiData`와 캐시 쓰기 금지**, 캐시에는 키 제외 유지 `:636-639`). 모든 응답 처리는 `seq !== keySeqRef.current`이면 무시. 의존성은 `[ai_id, hasAI]` 그대로(저장 후 `setAiData`로 반복 재조회되지 않음).
2. 초기화 effect(`[aiData]`) 끝에 `setFormSynced(true)` 추가(MINOR 2). 다른 폼 필드 초기화는 재조회 응답과 무관하게 유지.
3. `keyMode = keyModeOf(engineModel)`, `guardApplies = keyLoad === 'ready' && computeGuardApplies({savedKey: savedEngineKey, savedModel: savedEngineModel, currentModel: engineModel, models})`, **기준선은 `aiData.engine_model`이 아니라 `savedEngineModel`**(낡은 캐시 우회 차단, R7-M1).
4. **전이 effect**(의존성은 구현 시 `react-hooks/exhaustive-deps`에 맞게 effect가 읽는 값을 모두 나열한다. 최소 `modelsStatus`, `keyLoad`, `formSynced`, `ai_id`, `keyMode`, `guardApplies`, **`savedEngineModel`**(폴백 `keyModeOf(savedEngineModel)`), **`keyModeOf`**(T14에서 `useCallback`으로 안정화됨). `prevGuardRef`와 상태 setter는 제외 대상): `modelsStatus === 'ready' && keyLoad === 'ready' && formSynced`일 때만. `prev = prevGuardRef.current ?? { keyMode: keyModeOf(savedEngineModel), guardApplies: false }`, `keyTransitionAction`이 `'clear'` 또는 `'restore'`이면 `setEngineKey('')`, `setEngineKeyChanged(false)`(두 동작이 같음, 복원 시 마스킹 표시로 저장 키 상태 복귀), 끝에서 `prevGuardRef.current = cur`.
5. **저장 버튼:** `disabled={isSaving || keyLoad !== 'ready'}`(`:1166`). 재조회 전과 실패(`error`)에서 Save 비활성(fail-open 금지). `keyLoad === 'error'`이면 `Could not load the AI. Reload and try again.`를 인라인과 `saveError` 배너에 함께(실제로 보이는 것은 캐시 히트 재조회 실패뿐, 캐시 미스 실패는 폼이 숨겨짐을 테스트에 주석으로 남김).
6. **`handleSave` 검증 0~2와 body:** `model = engineModel.trim()`(Custom 접두), `customIdMissing`이면 모델 오류, `keyMode === 'own_required'`이고 유효 키가 비면 키 오류. 유효 키: 가드 비적용 `engineKeyChanged && engineKey.trim() ? engineKey.trim() : savedEngineKey`((가), 폴백 허용), **가드 적용 `engineKeyChanged && engineKey.trim() !== ''`만**((나), `savedEngineKey` 폴백 금지, 포커스만은 입력이 아님). body `engine_key`: `keyMode === 'platform'`이면 **보내지 않음**(현행 `sidebar.js:806-809` 계약 유지), 가드 적용이면 새 입력 trim 값 또는 `''`, 아니면 `engineKeyChanged && engineKey.trim() ? engineKey.trim() : savedEngineKey`를 **항상** 포함. 모든 클라이언트 검증 실패와 서버 키/모델 오류(`engineErrorOf`)는 인라인과 `setSaveError(msg); setSaveStatus('error')` 배너에 함께, 그 외 오류는 현행 `err.message`. 저장 성공 후 `setSavedEngineKey(body.engine_key ?? '')`와 `setSavedEngineModel(전송한 engine_model)`(보내지 않은 platform 저장은 `''`, 서버가 PUT 전체 교체로 빈 값 저장, `aihandler/db.go:280`). `:824`의 `setAiData({..., engine_key: aiData.engine_key})`는 기존대로. deps `:839`에 새 상태 추가.
7. **키 블록 위치:** JSX 지역 변수 `keyBlock`(`KEY_FIELD_COPY`, `own_required` 안내 3줄, 목적지별 경고, 키 오류, 마스킹 값 `engineKeyChanged ? engineKey : (savedEngineKey && !guardApplies ? '\u2022'.repeat(12) : '')`)을 만들고 `const keyOutside = keyMode === 'own_required' || guardApplies`이면 `ModelPicker`와 `CatalogStatus` 아래 `<details>` 밖, 아니면 접힌 `<details>`(`Advanced`) 안. `onFocus` 로직(`:1356-1359`)은 그대로. 경고는 가드 적용 시 목적지별 3종(`platform` 문구는 sidebar에서만).
**Tests**
- **편집 폼 캡션 단정 갱신(R19-M1):** `sidebar_engine_model.test.js:184`(`platform-managed stored model` 테스트)와 `:200`(`switching to a platform-managed model` 테스트)의 `getByText('API key not required')`를 `API key not required for this model.`로 바꾼다(편집 폼 캡션 `sidebar.js:1365`를 `KEY_FIELD_COPY.platform.caption`으로 바꾸는 이 태스크에서). 생성 폼 `:260`은 T19 소관이다.
- 기존: `sidebar_engine_model.test.js:162,173,202,218-219`, `sidebar_ai_type.test.js:140-142,226`의 동기 Save 클릭/`toBeEnabled` 앞에 `await waitFor(() => expect(saveButton()).toBeEnabled())`를 추가하고 `ProviderGet` mock이 `ais/{id}`에 `engine_key`를 포함해 응답하는지 확인(`localStorage.clear()`로 캐시 미스 경로라 필수는 아니지만 방어적으로). `sidebar.test.js`는 Save 단정이 없어 `ais/{id}` 응답 모킹만 확인.
- 신규 Q9: 캐시 미스에서 추가 GET 없이 `savedEngineKey/Model`이 채워짐, 캐시 히트에서만 재조회 1회, 재조회가 늦게 도착해도 사용자가 입력한 이름 등 폼 상태가 초기화되지 않음, 다른 AI로 바뀐 뒤 도착한 응답은 무시(`keySeqRef`), 이름만 바꿔 저장해도 body에 재조회한 `engine_key` 포함, 재조회 실패/falsy 응답에서 Save 비활성, 키 입력란에 포커스만 하고 저장해도 저장 키가 body에 유지(가드 비적용 시), platform 저장은 `engine_key` 미전송.
- `test.each(KEY_GUARD_ROWS.filter(r => !r.only || r.only === 'sidebar'))`(4절, 행 10, 19s 포함). 행 1~21 전부 **sidebar 편집에서 `ModelPicker`를 실제로 조작**한다(mock 없음).
- 키 블록 위치: `own_required`와 가드 적용 상태(목적지 키 방식 무관)에서 키 블록과 경고가 `<details>` 밖, 그 외는 접힌 `Advanced` 안(jsdom `details` 속성 동작은 단정하지 않음), 다른 탭에서 저장해 실패한 문구가 `saveError` 배너와 인라인에 모두 표시.
**Verify:** `CI=true npm test -- --watchAll=false src/views/teamgraph`; 변경 파일 `npx eslint`(T19와 같은 기준). **Depends:** T14, T15, T19(**T19 작업 완료 후 착수**, 같은 파일, 커밋은 J2 하나). **Commit:** J2.

#### T21. 에이전트용 문서 (같은 JS PR, 3.5)
**Files:** `square-main/public/skill.md`: `:412`(`engine_key`), `:622`(예시는 변경 불필요 확인), `:819`(`GET /ai_models` 응답 예시에 `key_mode` 추가, Custom 항목 예시), `:829`(`platform_managed` 설명을 `key_mode` 값 3개 설명과 함께 정리), `:843`(접두 `openrouter.` 허용 안내 문장 추가: `openrouter.<author>/<slug>`는 키 필수, 고객 OpenRouter 계정에 청구, ZDR 강제로 ZDR 제공자가 없으면 호출 실패, `:variant`와 `openrouter/auto` 같은 라우터형 ID 거부), `:941`("For models with `platform_managed: false`, verify `engine_key`..." 문구를 `key_mode` 기준으로). `square-main/public/llms.txt:17,49,50`(`engine_key` 문구를 `key_mode` 기준으로 갱신하고 `:50`의 응답 필드 나열에 `key_mode`, `model_id_prefix` 추가). **BYOK 상세는 `skill.md`에만** 두고 `llms.txt`는 해당 문장만 수정한다. `build/` 사본은 추적되지 않으므로(확인함) 손대지 않는다.
**Verify:** `grep -n "platform_managed" square-main/public/skill.md square-main/public/llms.txt`로 남은 문구가 `key_mode` 기준과 모순되지 않는지 눈으로 확인, 대시(em/en) 없음. **Depends:** T14(필드 이름). **Commit:** J3.

#### T22. member.js 확인, 린트, 빌드, 시각 검증
- `src/views/teamgraph/nodes/member.js:36-39,202-205`의 `getProviderStyle`이 `openrouter.<id>`를 폴백(`OP` 라벨, muted)으로 처리함을 확인(F9). 변경 없음. 이 확인을 `src/views/teamgraph/nodes/__tests__/member.test.js`의 한 행(`engine_model: 'openrouter.vendor/model-a'`가 예외 없이 렌더)으로 고정한다.
- **PR 테스트 게이트(monorepo-javascript `CLAUDE.md:25-45`, 1절 UI 검증 1~3번, F18):**
  1. 기준선: `git stash` 후(변경이 이미 커밋돼 있으면 `git stash` 대신 기준 커밋 `54aa7281`의 별도 워크트리에서) `cd square-admin && npm test -- --watchAll=false --forceExit 2>&1 | grep "Tests:"` -> `Tests: 2 failed, 1 skipped, 3027 passed, 3030 total`(T0 기록과 같아야 함). `git stash pop` 후 같은 명령을 변경 후에 실행한다. **통과 조건:** 실패는 기준선과 같은 1 suite(`mcpServerCatalog.test.js`, 2건) 외 **신규 실패 없음**, passed가 신규 테스트 수만큼 증가. PR 본문에 "Known pre-existing failures: 1 suite (src/views/mcpservers/__tests__/mcpServerCatalog.test.js, 2 tests) - confirmed failing on main baseline"을 명시한다.
  2. 린트: `npx eslint <변경한 모든 소스와 테스트 파일>`을 기준선(1절 2번 표)과 파일별로 비교해 늘지 않음. 전체 `npm run lint`(기준선 381 errors 167 warnings)는 게이트가 아니며 PR 본문에 "전체 lint는 기준선 실패, 변경 파일은 기준선 대비 증가 없음"으로 적는다.
  3. 빌드: **`CI=true` 없이** `npm run build`(실제 CI와 동일, `prebuild`의 `build:widget` 포함)가 exit 0이고 `git status --short`가 깨끗. `CI=true npm run build`는 기준선에서 이미 실패하므로 쓰지 않는다.
- **시각 검증 게이트(스킬 `voipbin-frontend-visual-verification-gate`, `voipbin-frontend-mock-render-screenshot`):** 목 카탈로그(픽스처)로 실제 컴포넌트를 렌더해 스크린샷: (1) 피커 열림(그룹 3개, 배지 3종, 검색, `Recommended`), (2) Custom 선택 + ID 입력란 + 오류 문구 + 키 입력란(안내 3줄), (3) sidebar 편집의 `own_required` 키 블록이 `<details>` 밖, (4) 가드 적용 경고 3종, (5) 저장 `openrouter.*` 복원(`Current:` 행 없음), (6) 4개 폼의 키 라벨/캡션. 목업(`2026-10-06-openrouter-byok-ui-mockup.png`) 패널 1~3과 대조하고 3.6.7의 목업 수정 항목은 실제 렌더로 대체한다(목업 png 수정은 이 PR의 범위가 아님, 대표님 요청 시 별도).
**Depends:** T14~T21. **Commit:** J3(테스트/린트 보정만).

---

### Phase F: 통합, 실호출, 변이, 전달

#### T23. monorepo 통합 검증과 PR 게이트 스크립트
순서대로 로컬에서 실행하고 결과를 PR 설명에 기록한다(출력에 키 없음).
1. **서비스별 전체 검증**(1절 Go 명령): `bin-ai-manager`, `bin-pipecat-manager`, `bin-api-manager`, `bin-call-manager`, `bin-openapi-manager`, `bin-common-handler`, T11 채택 시 `bin-webhook-manager`, `bin-conversation-manager`.
2. **전 서비스 빌드 스모크**(`monorepo/bin-common-handler`를 `go.mod`에서 참조하는 모듈 34개, 직접 센 값: `bin-*` 디렉터리 37개 중 `bin-dbscheme-manager`, `bin-openapi-manager`, `bin-trigger-sender`를 뺀 34개이며 `bin-common-handler`, `bin-ai-manager` 자신을 포함한다. `monorepo/bin-ai-manager`를 참조하는 모듈도 같은 34개로 두 집합이 일치해 합집합도 34개. `ModelInfo`/`Resolved` 변경과 `publish.go` 영향 확인): `for d in $(grep -l "monorepo/bin-common-handler" bin-*/go.mod | xargs -n1 dirname); do (cd "$d" && go build ./... ) || echo "FAIL $d"; done` 결과에 `FAIL`이 없어야 한다.
3. **Python:** 1절 명령(기존 실패 4건 외 새 실패 없음). **CI는 pipecat Python 테스트를 실행하지 않는다**(`.circleci/config_work.yml:1622`의 `bin-pipecat-manager-test`는 `go-test-pipecat-manager`만 호출, pytest 없음, 확인함). 따라서 이 항목과 T24 항목 3, 5는 CI가 막지 않으며 **PR 본문 체크리스트(실행 결과 한 줄씩 기재)로 강제**한다.
4. **생성 파일 재생성 원칙:** `bin-openapi-manager`와 `bin-api-manager`에서 `go generate ./...`를 다시 돌려 `git status --short`에 **추가 diff가 없는지**(멱등) 확인, mock(`mock_*.go`)은 인터페이스가 바뀌지 않았으므로 diff가 없어야 한다(`git diff --stat | grep mock_` 비어 있음), `docsdev/build`도 재빌드가 멱등(T13의 `pygments==2.18.0` venv에서 재빌드 후 `git status --short bin-api-manager/docsdev/build | grep -v doctrees`가 추가 변경을 보이지 않음).
5. **컨벤션 스크립트(CI 대상과 로컬 전용 구분):** CI(`.circleci/config_work.yml:2248,2281`의 `check-test-conventions` 잡)에서 도는 것은 **`scripts/check-test-conventions.sh` 하나뿐**이다. `make lint-docs`(`scripts/check-docs.sh`), `make lint-error-envelope`, `scripts/check-service-docs.sh`는 CI 설정에서 호출하지 않는 **로컬 전용**이다(`grep -n "check-docs\|check-error-envelope\|check-service-docs" .circleci/*.yml`가 비어 있음, 확인함). 로컬 전용 항목은 CI가 막지 않으므로 PR 본문 체크리스트로 강제한다.
```
git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'
bash scripts/check-test-conventions.sh      # CI 대상. 추가된 줄만 검사, fail-closed, merge base 필요
make lint-docs && make lint-error-envelope  # 로컬 전용
bash scripts/check-service-docs.sh          # 로컬 전용, 경고 전용(항상 exit 0). staged 파일만 검사하므로 `git add` 후 커밋 전에 실행(스테이징 전에는 항상 경고 없음)
```
**`check-service-docs.sh` 동작과 처리 방침(스크립트를 직접 읽고 확인함):** `.git/hooks/pre-commit:77-78`이 커밋마다 자동 호출하며 **절대 커밋을 막지 않는다**. 경고 규칙은 서비스 디렉터리(`bin-*`) 안의 `pkg/listenhandler/main.go` 또는 `cmd/<이름>/main.go`가 staged이면 같은 서비스의 `docs/architecture.md`, `internal/config/`나 `cmd/*/init.go`는 `docs/operations.md`, `models/` 하위는 `docs/domain.md`이고, 해당 docs 파일이 같은 커밋에 staged이면 경고가 사라진다. 이 PR에서 예상되는 경고와 방침:
- **`pkg/listenhandler/main.go`와 `cmd/*/main.go` 변경도 `docs/architecture.md` 경고를 낸다**(스크립트 `:48-54`). 대상은 T5의 `bin-ai-manager/pkg/listenhandler/main.go:285`(C4), T4의 `bin-ai-manager/cmd/ai-control/main.go`(C3), T11 분기 A의 `bin-webhook-manager/pkg/listenhandler/main.go:150`(C6)이다. 세 변경 모두 로그 필드 또는 검증 호출 변경이며 라우팅 표, 이벤트 구독, 설정을 바꾸지 않으므로 **`architecture.md`는 갱신하지 않고 경고를 수용**한다. PR 본문에 "check-service-docs 경고(architecture.md): routing/subscribe 변경 없음"으로 기록한다.
- `models/` 변경(T1, T2의 `bin-ai-manager/models/ai/*.go`)은 `bin-ai-manager/docs/domain.md` 경고를 낸다. `domain.md`는 T13(C8)이 갱신하므로 **C1, C2 시점의 경고는 예상된 것**이고 C8에서 해소된다(C8에서 `domain.md` staged이면 경고 없음, C8 시점에 경고가 남아 있지 않은지 확인).
6. **위생:**
```
# 새로 만든 파일은 untracked라 git diff에 보이지 않으므로 먼저 의도 추가(실제 add는 커밋 단계에서)
git add -N .
# 키 형태 문자열. pathspec으로 docs/plans를 제외한다: 분석서와 디자인(디자인 커밋 323ad8fc8)이 sk-or- 형태를 서술로 포함해
# 제외하지 않으면 diff에 항상 3건이 걸려 통과할 수 없다(직접 실행: 제외 없음 3건, 제외 후 0건)
git diff origin/main -U0 -- . ':(exclude)docs/plans' | grep -nE "sk-or-|sk-[A-Za-z0-9]{20,}|OPENROUTER_API_KEY=.+"        # 결과가 비어 있어야 함
# em/en 대시. grep -P의 \x{2014}는 UTF-8 로케일이 전제다. 로케일이 C이면 "character code point value in \x{} or \o{} is too large" 오류가 나므로
# LC_ALL=C.UTF-8(이 머신에 있음)을 명시한다. docs/plans의 이 plan 자신은 별도로 확인한다(기존 줄은 제외)
git diff origin/main -U0 -- . ':(exclude)docs/plans' | LC_ALL=C.UTF-8 grep -nP "^\+[^+].*[\x{2014}\x{2013}]"                  # 결과가 비어 있어야 함
LC_ALL=C.UTF-8 grep -nP "[\x{2014}\x{2013}]" docs/plans/2026-10-06-openrouter-byok-plan.md                                     # 결과가 비어 있어야 함
grep -rn "dummy-" --include=*.go --include=*.py --include=*.js . | grep -v "_test.go\|test_.*\.py\|\.test\.js\|\.fixture\.js\|node_modules\|/vendor/"   # 비어 있어야 함(테스트와 픽스처에만 존재)
```
7. **전달 전 점검(CLAUDE.md):** `git fetch origin main`, `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`가 비어 있음, `git log HEAD..origin/main --oneline` 확인, `git log origin/main..HEAD --format=%ae | sort -u`가 `pchero21@gmail.com`뿐. 충돌이 있으면 rebase 후 위 1~6을 다시 실행.
8. **monorepo-javascript(`CLAUDE.md:25-45`의 PR 테스트 게이트):** T22의 1~3(전체 테스트를 `git stash` 기준선과 `Tests:` 줄로 비교, **신규 실패 없음과 기존 실패 1 suite(`mcpServerCatalog.test.js` 2건)를 PR 본문에 명시**, 변경 파일 `npx eslint` 전후 비교에서 증가 없음, `CI=true` 없는 `npm run build` exit 0)를 모두 통과하고 결과를 PR 본문에 적는다. 같은 fetch와 충돌/작성자 점검(`git merge-tree ... | grep -E "^(CONFLICT|changed in both)"`, `git log origin/main..HEAD --format=%ae | sort -u`).
**Depends:** S1~S4 완료.

#### T24. **[조건부, 대표님 허용 전제]** 실호출 검증 (7절, 8절에 절차)
8절 참조. 항목 3, 5(더미 값, 외부 요청 없음)는 **허용 없이 수행하며 머지 게이트**다.

#### T25. 변이 검사 (수동, PR 설명에 기록)
가드를 일부러 깨고 **테스트가 빨개지는지** 확인한 뒤 되돌린다. 각 행은 한 번에 한 개만 적용한다.

| # | 변이 | 빨개져야 하는 테스트 |
|---|---|---|
| M1 | `ValidateEngine`에서 키 검사 제거 | `Test_ValidateEngine`, aihandler `ENGINE_KEY_REQUIRED` 테스트 |
| M2 | `ResolveEngine` Custom 분기가 `BlankKey: true` 또는 `platform_openrouter.` 접두 생성 | `Test_ResolveEngineCustomNeverPromotes` |
| M3 | `ResolveEngine` 정확 일치 루프의 Custom 건너뛰기 제거 | `Test_ResolveEngineAllOpenRouterEntriesBlankKey`(`custom.openrouter` Rejected 단정) |
| M4 | `resolveSessionLLM`에서 `TrimSpace`, 빈 키 오류 제거 | `Test_resolveSessionLLM` |
| M5 | `Start()`가 다시 `resolveSessionLLM(llmType, "")`를 호출 | `Test_Start_customModelPassesClassification` |
| M6 | Python Custom 분기에 `or os.getenv("OPENROUTER_API_KEY")` 추가 | `TestCustomOpenRouter` 환경 키 단정 |
| M7 | Python `.strip()` 제거 | `"  "`, `"\n"` 키 테스트 |
| M8 | Custom 분기에서 `_build_openrouter_llm` 대신 provider 없이 호출 | provider dict 동일 상수 단정 |
| M9 | Custom 전용 오류 메시지가 입력값을 `%q`로 반향 | `assertInvalidCustomModelID`, pipecat/ai-control 반향 테스트 |
| M10 | B1 `"request": m` 되돌림 | `Test_processRequest_errorLogOmitsRequestBody` |
| M11 | UI: 전이 effect의 `clear` 제거 | 행 1, 13, 15d, 18a, 20 |
| M12 | UI: `computeGuardApplies`의 서비스 접두 비교 제거 | 행 15a~15d, 17 |
| M13 | UI: 가드 (나)에서 `savedEngineKey` 폴백 추가 | 행 2, 10 |
| M14 | UI: sidebar 기준선을 `aiData.engine_model`로 | 행 10 |
| M15 | UI: create 비움을 핸들러로 이동 | 템플릿 선택/Builder 적용 테스트(T17) |
| M16 | UI: `formSynced` 게이트 제거 | detail 정상 로드 무비움 테스트 |
| M17 | UI: sidebar platform 저장이 `engine_key` 전송 | `sidebar_engine_model.test.js:206` |
| M18 | UI: `keyOutside` 조건에서 `guardApplies` 제거 | 행 16(`<details>` 밖) |

#### T26. **[조건부]** 후속 분기 처리 (6절)
7절 항목 4/9, Q5 안내 문구, 8.1 항목 9 결과에 따라 6절 표의 분기 태스크를 수행한다. 어느 분기도 기본값에서는 코드 변경이 없다.

#### T27. 리뷰 루프와 PR 전달
- 설계 리뷰 루프(이 plan): 최소 2회, 2회 연속 Approval(리뷰어 독립, Request Changes 시 연속 카운트 0), 최대 20회.
- 구현 후 코드 리뷰 루프: 최소 3회, 2회 연속 Approval, 최대 30회. PR별(monorepo, monorepo-javascript) 독립 리뷰어 3명이 서로 다른 관점(보안 3중 방어와 로깅, UI 가드와 상태 전이, 컴파일과 CI 게이트)으로 병렬 수행. CHANGES_REQUESTED가 나오면 카운트 리셋, 상한 도달 시 대표님께 상태 보고.
- PR 제목은 브랜치명 `NOJIRA-Add-custom-OpenRouter-BYOK-models`, 본문은 narrative 한 문단 + `project-name: change` 불릿(헤더, test plan, AI 표기 금지). 머지는 대표님 명시 지시 후 squash만, 머지 후 `cd ~/gitvoipbin/monorepo && git pull origin main`. 두 PR 모두 실제 CI 체크를 읽고 보고한다.
- 롤아웃 노트(7절)와 T24 결과(키/원문 없이)를 PR 설명에 싣는다.

---

## 6. 분기 조건 있는 태스크 (대표님 결정 또는 허용 대기)

| ID | 조건 | 기본 분기(권장) | 대안 분기와 영향 |
|---|---|---|---|
| **T11** (publish.go + webhook-manager B4) | 8.1 항목 8 확정 | **A**: 이번 PR에서 `publish.go` 한 줄 + webhook-manager 9줄 수정(C6) | **B** 별도 PR(대표님 허락 필요, T11 제외, 6.3 게이트 2에 별도 이슈 번호 기록), **C** 미수정(위험 수용, 8.3에 명시, webhook-manager 상시 Debug에 BYOK 키가 남을 수 있음을 PR에 경고) |
| **T24** (실호출 항목 1, 2, 7, 8) | 디자인 7절: 플랫폼 `OPENROUTER_API_KEY` 일회성 수동 확인을 **대표님이 허용**(현재 대기, 이 계획 작성 중 어떤 키로도 실호출하지 않음) | 허용 시 8절 절차 수행 | 불허: 항목 1, 2, 7, 8을 `미확인`으로 PR과 문서에 기록하고 6.3 게이트 4에서 "ZDR 없음 안내 문구는 3.7 기본 동작으로 출시할지" 대표님이 결정(8.1 항목 5 권장은 실호출 후 출시) |
| **T26-a** (이벤트 버스/ClickHouse 확인, 7절 항목 4와 9) | ClickHouse 조회는 대표님 접근 필요(값은 출력하지 않고 필드 존재와 건수만) | 결과가 "키 저장됨"이면 별도 이슈(ai-manager가 발행 전 키 제외 페이로드)를 만들고 6.3 게이트 3에서 출시 전 처리 시점을 대표님이 결정 | 결과가 "저장 안 됨"이면 별도 조치 없음. 어느 쪽이든 이 PR의 코드는 변하지 않음 |
| **T26-b** (`engine_key` 응답 포함 확인, 항목 4) | T24 항목 4에서 `GET /ais` 응답에 키가 실제 포함됨이 확인된 경우 | 기본: 문구 변경 없음 | 포함 확인 시 `ai_struct_ai.rst:77`과 `openapi.yaml:2198`의 "never returned/not returned"를 정정(T12/T13에 소폭 추가) |
| **T26-c** (Q5 안내 문구와 분류기) | T24 항목 1, 2에서 ZDR 부재 응답 원문이 안정적으로 구분될 때 | 기본: `pipelineerror.go` 분류기 확장 없음(음성 세션 무음 가능 한계를 문서에 명시) | 구분 가능 시 `classifyPipelineError`(`pipelineerror.go:20-86`)에 카테고리와 고객 문구 `The selected OpenRouter model has no provider that supports zero data retention. Choose a different model.` 추가 + 테스트(`pipelineerror_test.go`) |
| **T18/T20-alt** (8.1 항목 9, detail platform 목적지 키) | 대표님이 "detail에서도 platform 저장 시 키를 지운다"로 바꾸는 경우 | **유지**(제안 v): detail은 platform 저장 시 저장 키 재전송 | 변경 시 `ais_detail.js:422`의 platform 분기를 `''` 전송으로, `ais_detail.test.js:435-448`의 기대를 `''`로 바꾸고 detail에도 경고 `The saved key will be removed.` 표시, 4절 행 7, 16, 17의 detail 기대 수정 |
| **Q4 변종 허용** (실호출 항목 7, 8 결과) | ZDR 유지가 확인된 변종/라우터형 ID가 있을 때 | 기본: 모두 거부(fail-closed) | 확인된 변종만 `ValidateCustomModelID`의 허용 규칙 표 한 줄 수정 + 테스트(후속 PR 가능) |
| **7절 항목 9** (이미 쌓인 운영 로그 조회) | 대표님이 운영 로그 접근 | 필드 존재와 건수만 확인 | 결과가 "있음"이면 기존 OpenAI/Gemini/Grok 키 교체 권고(8.1 항목 4) |

---

## 7. 롤아웃과 PR 게이트

### 7.1 배포 순서(디자인 6.2, PR 설명에 포함)
1. **pipecat-manager Go와 Python 러너를 동시에 배포**(같은 이미지/compose). 한쪽만 먼저 나가도 신 Python + 구 Go는 `Rejected`, 신 Go + 구 Python은 `Unsupported LLM service`로 실패하며 모두 fail-closed(플랫폼 키 폴백 없음, R1-g).
2. **ai-manager와 api-manager를 함께 배포**(`amai.ModelInfo`를 컴파일 시 포함하므로 api-manager를 재빌드하지 않으면 `key_mode`, `model_id_prefix`가 응답에서 사라짐). 1번이 먼저여야 한다(2번이 먼저면 Custom 저장 후 호출이 `Rejected`로 안전하게 실패하나 UX 열화).
3. call-manager(로그 1줄)는 순서 무관. **webhook-manager(T11 채택 시)는 ai-manager와 함께**(ai 웹훅 페이로드를 상시 기록하는 서비스), conversation-manager는 같은 PR 산출물이라 함께 배포 권장(`publish.go` 반영), 나머지 14개 서비스는 다음 정기 배포.
4. **UI는 마지막**(신 UI + 구 API는 `key_mode` 부재 폴백으로 Custom UI가 안 나타남, 구 UI + 신 API는 Custom이 일반 행으로 보이나 서버 검증이 최종 방어).
5. RST와 `skill.md`는 UI와 같은 시점에 공개.
롤백은 서비스 이미지 되돌리기(기능 플래그 없음). ai-manager만 되돌리면 저장된 `openrouter.*` AI는 구 resolver에서 안전하게 실패한다.

### 7.2 출시 게이트(디자인 6.3, ai-manager와 api-manager 배포 전)
1. T24 항목 5(러너 SDK 폴백 실행 검증)와 항목 3(빈 키 실행 검증) 통과 **전에는 머지하지 않고 배포하지도 않는다**.
2. 로그 수정 A(16곳), B1, T11 채택 시 B3, B4가 배포 대상 빌드에 포함, 이미 쌓인 로그 처리 판단(항목 9, 8.1 항목 4)과 공용 로깅 범위 결정(8.1 항목 8) 완료.
3. `engine_key` 노출(GET, 웹훅, 이벤트 버스/ClickHouse) 처리 시점을 대표님이 결정(8.1 항목 3, T26-a).
4. 항목 1, 2 결과로 고객 안내 문구 확정 여부(8.1 항목 5).

### 7.3 PR 게이트 요약
레포당 PR 1개(스트림은 같은 브랜치), 로컬에서 T23 1~7이 모두 통과(**CI가 막는 것은 `check-test-conventions`와 서비스별 Go 테스트이고, Python 테스트, T24 항목 3, 5, `lint-docs`, `lint-error-envelope`, `check-service-docs`는 CI가 막지 않으므로 PR 본문 체크리스트로 강제**, monorepo-javascript PR은 T23-8, 즉 CLAUDE.md의 테스트 게이트와 `CI=true` 없는 빌드), 생성 파일은 생성기로만 갱신, `docsdev/build`는 `git add -f`, 코드 리뷰 루프 통과(T27), 머지는 대표님 지시 후 squash.

---

## 8. 실호출 검증 절차 (T24, 키 비노출)

**원칙:** 키는 환경변수로만 주입하고 값을 출력, 로그, 파일, 커밋, 문서에 남기지 않는다. **`~/.hermes/.env`를 `set -a; source`로 통째로 읽지 않는다**(다른 비밀값이 셸 전체에 노출됨). 필요한 변수 하나만 명령 한 줄에 주입한다: `OPENROUTER_API_KEY="$(grep '^OPENROUTER_API_KEY=' ~/.hermes/.env | tail -n1 | cut -d= -f2-)" python3 "$TMPDIR/probe.py"`(`tail -n1`은 dotenv의 마지막 줄 우선 관례, 값은 출력하지 않고 `set +x` 상태 유지). **`.env` 값이 따옴표로 감싸져 있을 수 있다**(`KEY="..."` 또는 `KEY='...'`): `cut` 결과의 양끝 따옴표 한 쌍을 벗기는 처리를 프로브 안에서 한다(예: Python `v = v.strip(); v = v[1:-1] if len(v) >= 2 and v[0] == v[-1] and v[0] in "\"'" else v`). 값 자체는 출력하지 않는다. `OPENROUTER_API_KEY=` 줄이 둘이므로 `grep -c '^OPENROUTER_API_KEY=' ~/.hermes/.env`의 개수만 확인하고, 두 줄이 다른 키일 가능성은 값을 찍지 않고 대표님께 어느 줄을 쓸지 확인한다. **또 `~/.hermes/.env`의 `OPENROUTER_API_KEY`가 VoIPBin 플랫폼 운영 키와 같은 값인지 대표님께 확인한다**(같다면 이 키로 하는 실호출은 플랫폼 계정에 청구되고 고객 계정 가드레일 시나리오와 무관하다. 다르면 개인 키이므로 어느 쪽이든 허용 범위를 대표님이 정한다). 값 비교는 하지 않고 대표님 답변만 기록한다. 스크립트는 `$TMPDIR`(scratch)에 두고 저장소에 넣지 않으며 결과는 모델 ID, HTTP 상태, 응답 **유형**, 응답한 하위 제공자 이름만 기록한다(원문 대화, 키 반사 여부는 "있음/없음"만). **프로브는 예외 문자열과 응답 본문을 출력하지 않는다**(예외 메시지나 본문에 요청 헤더, 키가 반사될 수 있음). 예외는 `type(e).__name__`, 응답은 상태 코드와 `Content-Type` 유형, 제공자명 필드만 출력한다. 플랫폼 키는 대표님 허용 시 응답 형태 확인용으로만 쓰고 고객 계정 가드레일 시나리오는 "확인 불가"로 남긴다.

| 항목(디자인 7절) | 절차 | 허용 필요 | 게이트 |
|---|---|---|---|
| 3 | **T7 구현 후** 러너 환경(`OPENROUTER_API_KEY=dummy-or-env-not-real`, `OPENAI_API_KEY=dummy-openai-env-not-real`을 프로브 스크립트가 `os.environ`에 직접 설정)에서 `create_llm_service("openrouter.vendor/model-a", k, [], [])`의 `k`가 `None`, `""`, `"  "`이면 **모두 예외**(`ValueError`)이고 소켓 `connect` 시도가 **0건**(프로브가 `socket.socket.connect`를 예외로 대체하고 시도 횟수를 센다)임을 단정. 아래 "실행 방법"의 `uv run` 명령 사용 | 불필요(더미) | **머지 게이트** |
| 5 | 같은 환경에서 실제 SDK 동작 확인. **T7 이전에도 실행 가능한 SDK 사실(직접 실행으로 확인함, `OPENAI_API_KEY`와 `OPENROUTER_API_KEY` 더미 설정):** `OpenRouterLLMService(api_key=None)`의 클라이언트 키가 **`OPENAI_API_KEY` 값**(폴백, 플랫폼 키 유출 경로), `api_key=""`는 빈 문자열(폴백 없음), `api_key="  "`는 **공백 그대로** 전달(그래서 `.strip()` 필수), `api_key="dummy-typed-key"`는 그대로 전달, 위 네 호출 동안 소켓 `connect` 0건. **T7 구현 후 단정:** Custom 분기에서 `k="  dummy-typed-key  "`이면 생성된 서비스의 클라이언트 키가 **정제된 `dummy-typed-key`**이고 환경 값 두 개 중 어느 것도 아님, `k`가 `None`/`""`/`"  "`이면 서비스가 만들어지지 않음 | 불필요(더미) | **머지 게이트** |
| 1 | 고객 키 역할 키로 `provider.zdr:true` 호출: ZDR 제공자가 있는 모델 성공, 없는 모델 실패의 HTTP 상태와 원문 유형 | 필요 | Q5 안내 확정 근거 |
| 2 | 무효 키(401, 더미 키), 크레딧 없음(402, 재현 불가 시 문서 근거만), 러너가 보는 오류 원문과 `classifyPipelineError` 분류 결과, 키 반사 여부(`pipelineerror.go:12`의 2000자 로깅) | 401은 더미로 가능, 나머지 필요 | 분류기 확장 여부 근거 |
| 7 | `openrouter.vendor/model:variant`가 첫 점 분리와 `extra_body`를 거쳐도 유지되는지(단위, 허용 없이 가능), 변종 ID의 ZDR 유지(실호출) | 단위 불필요, 실호출 필요 | Q4 순차 허용 근거 |
| 8 | 라우터형 ID(`openrouter/auto`)의 ZDR 상호작용과 다른 제공자로의 라우팅 여부 | 필요 | 결과 전까지 거부 유지 |
| 4, 9 | `GET /ais`, ai 웹훅, 이벤트 버스(ClickHouse `events.data`)에 `engine_key`가 포함되는지(더미 값으로 AI를 만들어 확인, 필드 존재와 건수만), 이미 쌓인 운영 로그 조회 | 대표님 운영 접근 | T26-a, 7.2 게이트 2, 3 |
| 6 | sidebar 편집 저장 시 `engine_key`가 빈 값으로 덮이는지(재조회 적용 전/후 UI 실행 비교) | 불필요 | Q9는 실증과 무관하게 적용 |

**항목 3, 5 실행 방법(직접 실행해 동작을 확인한 명령, F20):** `uv run --with pipecat-ai[openrouter]`처럼 `--no-project`로 일부 extra만 붙이면 `run.py`의 `google.genai` 등 다른 import가 없어 `import run`이 실패한다. 프로젝트 잠금 파일로 전체 의존성을 쓴다:
```
cd bin-pipecat-manager/scripts/pipecat
export UV_PROJECT_ENVIRONMENT="$TMPDIR/pipecat-venv"          # 미지정이면 scripts/pipecat/.venv가 저장소 안에 생긴다
PYTHONPATH=. uv run --frozen --project . python "$TMPDIR/probe_t24.py"     # --frozen: uv.lock을 수정하지 않음
git status --short .                                           # 비어 있어야 함(.venv, uv.lock 변경 없음)
```
`probe_t24.py`는 `$TMPDIR`에 두고(저장소에 넣지 않음) `os.environ`에 더미 두 개를 설정하고 `socket.socket.connect`를 예외로 대체한 뒤 `import run`, `run.create_llm_service(...)`를 호출한다. 확인한 사실: 위 명령으로 `import run`이 성공하고(Pipecat 1.12.0, CPython 3.13.7, 116 패키지 설치 3초), 현재 코드(T7 이전)는 `create_llm_service("platform_openrouter.a/b", None, [], [])`가 `OpenRouterLLMService`를 만들고 `create_llm_service("openrouter.vendor/model-a", "dummy", [], [])`는 `ValueError: Unsupported LLM service: openrouter`를 낸다(T7 전 기준선). **머지 게이트 단정(T7 이후):** `None`, `""`, `"  "`는 예외, 외부 요청 0건, 정제된 키만 SDK `api_key`로 전달.

결과는 PR 설명에 "성공/실패, 모델 ID, 상태 코드, 하위 제공자" 수준으로만 기록한다.

---

## 9. 디자인 요구사항 -> 태스크 매핑표

| 디자인 항목 | 요구 | 태스크 |
|---|---|---|
| 1.1-1, Q1a, 3.6.2, 3.6.3 | 그룹 3개, 배지 3종, 문구 | T1(`key_mode`), T14, T15 |
| 1.1-2, Q2, Q4, 3.1.1, 3.1.3 | Custom 항목, 모델 ID 검증 | T1, T2, T3 |
| 1.1-3, 4절 R1-a~g | 플랫폼 키 접근 불가 3중 방어 | T2(서버/resolver), T6(Go), T7(Python), T25, T24 항목 3, 5 |
| 1.1-4, 3.7, Q5 | 요청 단위 ZDR 강제, fail-closed | T7(`_OPENROUTER_PROVIDER`), T24 항목 1, T26-c |
| 1.1-5, Q10, 3.4 A | 필드 로깅 16곳 | T5(11), T9(4), T10(1) |
| 3.4 B1 | 오류 경로 요청 본문 6곳 + 캡처 테스트 | T5 |
| 3.4 B2 | marshal `%v` 5곳(`:66` Debugf) | T5 |
| 3.4 B3, B4, 8.1-8 | `publish.go` + webhook-manager | T11(조건부) |
| 1.2 | 전역 플래그, 마이그레이션 없음 | 전 태스크(추가 없음 확인: T23-6) |
| Q1b | 직접 모델 빈 키 거부 안 함 | T2(`ValidateEngine` 직접 모델 nil 테스트), T6 |
| Q6 | `platform_managed` 유지 | T1, T12 |
| Q7 | 키 형식 검증 없음 | 구현 없음(T2에 `sk-or-` 검사 추가 금지) |
| Q8, 3.6.5 | 키 이월 UX, 가드, 전이 비움 | T14, T17~T20, 4절 행렬 |
| Q9, 3.6.6 | sidebar 편집 재조회 | T20 |
| Q11, 3.5 | 문서 | T13, T21 |
| 3.1.2 | `ValidateEngine`, ai-control | T2, T3, T4 |
| 3.1.3 반향 방어 | api-manager 로그, ai-control, aihandler | T9, T4, T3 |
| 3.1.4 | `ENGINE_KEY_REQUIRED` | T3 |
| 3.1.5 | 가드 4곳 완화(3곳 한정 예외 + 1곳 유지) | T1(가드 1~3), T3(가드 4 유지) |
| 3.2 | OpenAPI, 재생성, api-manager 재배포 결합 | T1(F1), T12, 7.1 |
| 3.3 Go | `classifySessionLLM`, `resolveSessionLLM`, Rejected 정제 | T6 |
| 3.3 Python | 새 분기, `_member_llm_type` 불변 | T7 |
| 3.1.1 마지막 항 | `verify_openrouter_catalog.py` 의존 | T1(불변 조건), T8 |
| 3.6.1 | 데이터 계층, 구 API 폴백 | T14 |
| 3.6.2 | ModelPicker, 입력란, 포커스, `error` 단일 경로 | T15, T16 |
| 3.6.4 | 클라이언트 검증, 서버 오류 분기, 이중 표시, `<details>` 밖 | T17~T20 |
| 3.6.7 | 목업 수정 | T22(실제 렌더로 대체) |
| 4절 승격 방지 표 | resolver 표 테스트 | T2 |
| 5.1 | 수정할 기존 테스트 전부 | T1, T2, T3, T4, T6, T7, T15~T20(각 태스크에 라인 표기) |
| 5.2 | 신규 테스트와 가드 행렬 | T1~T7, T14~T20, 4절 |
| 5.3 | 검증 절차 | 1절, T23 |
| 6.1, 6.2, 6.3 | PR 구성, 배포 순서, 게이트 | 7절, T27 |
| 7절 항목 1~9 | 실호출 | 8절, T24, T26 |
| 8.1 항목 1, 2 | 슬래시 1개 규칙, 라벨 통일 | T2(`customModelIDPattern`), T14(`KEY_FIELD_COPY`) |
| 8.1 항목 3, 4, 5, 6, 7 | 대표님 결정 | 6절 |
| 8.1 항목 8 | 로깅 범위 A/B/C | T11, 6절 |
| 8.1 항목 9 | detail platform 키 유지 | 6절(T18/T20-alt) |
| 8.2 | 구현 시 확인(nil, member.js, `onCloseAutoFocus`, B4 컴파일, `keySeqRef`, docs build, `Test_CatalogViewShape`, 필드 이름) | T5, T22, T15, T11, T20, T13, T1 |
| 8.3 | 위험 문서화(수용된 재존, aicall 스냅샷 한계) | T13(정오표), 7.2 |

---

## 10. 범위 밖(이 PR에서 하지 않음, 별도 이슈 후보)
`GET /ais` 응답 `engine_key` 제거(재조회와 PUT 의미 변경이 한 쌍), 키 입력란 `type=password` 전환, 로컬스토리지 평문 키 캐시(`provider.js:361-388`의 `LoadResource`, `teams_create.js:50,72`), 다른 핸들러의 요청 본문 로깅(mcpserver 등)과 timeline-manager 드롭 로그(`subscribehandler/main.go:173`), aicall 모델/키 스냅샷 시점 불일치, 직접 모델 빈 키 거부(Q1b), 이전 Custom ID 기억(Custom 행 재선택 불편), 로그 중앙 처리(logrus Hook), ZDR 사전 조회, 자체 호스팅 문서 변경, text-chat `engine_openai_handler`/Builder/summary/analysis 핸들러.

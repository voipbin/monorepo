# 디자인: 모델 선택기 키 방식 구분 + Custom OpenRouter(BYOK) 모델

Status: 수정회차 8 (디자인 리뷰 재리뷰 대기). 이 문서는 문서 작성 단계 산출물이며 코드 수정, git commit/push는 하지 않았다.
Branch: NOJIRA-Add-custom-OpenRouter-BYOK-models (origin/main bc68e3c32 기준)
Date: 2026-10-06
수정 이력: 디자인 수정회차 8 (2026-10-07, 리뷰 7회차 리뷰어 13 MAJOR 1건과 minor 반영. 수정회차 7이 `prevKeyModeRef`를 제거해 생긴 회귀를 고침: 편집 화면(detail, sidebar 편집)의 입력란 비움 트리거를 '(1) 키 방식 그룹(`keyMode`)이 바뀔 때 또는 (2) `guardApplies`가 false에서 true로 바뀔 때'로 확장(`prevGuardRef` 한 ref 객체 `{ keyMode, guardApplies }`로 통합, 새 규칙 없이 기존 비교값 확장, CPO 제안 w). 저장 키가 없는 AI에서 타이핑한 키가 Custom 전환 후 openrouter.ai로, 가드 true인 채 그룹이 바뀔 때 다른 벤더로 전송되는 경로를 닫음. 로딩 중 타이핑 후 전환은 저장 모델 기준 첫 실행으로 닫음. 5.2 행 14 정정, 행 18~20 추가, 행 16/17 순서 정정, 3.6.5 문구와 create/edit 같은 그룹 벤더 변경 비교 명시, 8.3과 3.8 목록 캐시 서술 정정, 테스트 경로 전체 표기, `navigate(0)` 라인 정정, detail effect 위치(`isLoading` 조기 반환 앞)와 ref null 가드, detail 테스트 mock 주의). 이전 이력: 디자인 수정회차 7 (2026-10-07, 리뷰 6회차 두 리뷰어 동일 MAJOR 1건과 알려진 비차단 minor 전부 반영. 편집 화면의 입력란 비움 트리거를 `keyMode` 전이(`prevKeyModeRef`)에서 **가드 적용 여부(`guardApplies`)의 false에서 true 전이**로 바꿔(`prevGuardRef`로 대체, `prevKeyModeRef`는 편집 화면에서 제거) 같은 그룹 안 벤더 변경(OpenAI에서 Gemini)에서도 미리 채운 이전 키와 이전 벤더용으로 타이핑한 키가 새 입력으로 인정되지 않게 함, 새 규칙 없음(제안 p의 트리거 정정). minor: 로컬스토리지 평문 캐시 별도 이슈 기록(8.3), 캐시 미스 재조회 실패의 배너 사실 정정, `ProviderGet` falsy 응답 처리, 마크다운 불릿과 Q8, 제안 c 표현 정리, 8.4 범위, 카탈로그 ready 이후 전이 effect, 5.1 sidebar 대기 사유 정정, 3.8 목록에 수정회차 5, 6 사실 추가, 5.2 행렬 `test.each` 지침과 platform 그룹 내 접두 불일치 행, Custom 행 재선택 불편 기록, create 같은 그룹 벤더 변경 시 키 유지, `detailData`와 `fetchedDetail` 사용처 구분). 이전 이력: 디자인 수정회차 6 (2026-10-07, 리뷰 5회차 두 리뷰어 CHANGES_REQUESTED 각 MAJOR 1건과 minor 반영. 가드를 단순하고 닫힌 규칙으로 정리: 가드 (b)를 "그룹이 다르거나 서비스 접두가 다르다"로 확장해 같은 그룹 안의 다른 벤더로 저장 키가 전송되는 경로(낡은 캐시 포함)를 닫고, 유효 키 판정을 "입력란 trim 후 비어 있지 않음" 하나로 줄여 저장 키와의 비교, 제안 q의 동일 비교, 5.2 행 15, 8.3의 (iii)를 삭제, 경고가 필요한 sidebar 상태에서는 키 블록을 `<details>` 밖에 렌더해 경고 가시성 확보, detail platform 목적지의 키 유지 문구 정직화, 규칙 수 증가 없음, CPO 제안 s~v). 이전 이력: 디자인 수정회차 5 (2026-10-07, 리뷰 4회차 반영: 리뷰어 7 CHANGES_REQUESTED MAJOR 1건과 리뷰어 8 minor 반영. sidebar 가드의 저장 모델 기준선을 낡은 localStorage 캐시가 아니라 재조회 응답의 `savedEngineModel`로 교체해 키와 모델이 같은 서버 스냅샷에서 오게 함, 편집 화면 전이 시 입력 비움, 키 비교 양쪽 trim, 경고 문구 목적지별 구분, 중복 GET 제거, 가드 행렬 보강, CPO 제안 o~r). 이전 이력: 디자인 수정회차 4 (2026-10-07, 리뷰 3회차 두 리뷰어 CHANGES_REQUESTED 동일 MAJOR 1건 반영: 키 이월 가드 기준을 "저장된 키가 비어 있지 않고 목적지 키 방식 그룹이 저장 그룹과 다르다"로 바꿔 platform 원본 우회 차단, 유효 키 정의를 새로 입력한 값만으로 정리, sidebar 키 블록 조건부 렌더로 forceOpen 장치 제거, Popover 포커스 단일화, 라인과 문서 내부 참조 정정, CPO 제안 k~n). 이전 이력: 디자인 수정회차 3 (2026-10-06, 리뷰 2회차 두 리뷰어 CHANGES_REQUESTED 반영: 로깅 정리 16곳과 webhook-manager B4 편입, 양방향 키 이월, sidebar 오류 배너 병행, 오류 전달 경로 통일, 깨지는 기존 테스트 목록 정정, 포커스와 details 동작 명시, CPO 제안 2.2 5건). 이전 이력: 디자인 수정회차 2 (2026-10-06). 리뷰 1회차(두 리뷰어 CHANGES_REQUESTED) 지적을 실제 코드로 다시 확인해 반영했다. 오류 경로 요청 본문 로깅 필수화와 공용 publish 로깅 범위(3.4, 8.1 항목 8), 테스트 가드 허용 필드 명시(3.1.5, 5.2), 모델 ID 입력 위치를 ModelPicker 내부로 확정(3.6.2), sidebar Advanced 강제 open과 키 라벨 통일(3.6.3, 3.6.4), 키 이월 보완(3.6.5), 재조회 레이스 조건(3.6.6), 이벤트 버스 잔존 위험(4절, 8.1 항목 3), 게이트 시점 통일(6.3, 7절). CPO 제안 5건(2.1)은 대표님 이견 없으면 확정으로 읽는다.
선행 문서: `2026-10-06-openrouter-byok-issue-analysis.md`(이슈 분석 리뷰 8회차 통과본, 이하 "분석서"), `2026-10-06-openrouter-byok-ui-mockup.png`(이하 "목업"), `2026-10-05-openrouter-llm-routing-{issue-analysis,design,plan}.md`
UI 저장소: `/home/pchero/gitvoipbin/monorepo-javascript` (origin/main 54aa7281 기준). 같은 이름의 워크트리 `.worktrees/NOJIRA-Add-custom-OpenRouter-BYOK-models`가 이미 있고 main과 동일(변경 없음)함을 이번에 확인했다

표기: `파일:라인`은 이 워크트리(UI는 monorepo-javascript main)의 현재 코드를 이번 작성 중 직접 읽은 결과다. "미실증"은 코드 읽기까지만 했고 실행으로 확인하지 않았다는 뜻이다. 고객에게 노출되는 문자열과 코드 주석 예시는 영어, 설명은 한국어다.

## 1. 목표와 비목표

### 1.1 목표

1. 모델 선택기에서 모델별 키 방식(플랫폼 제공 / 내 키 또는 기본 키 / 내 키 필수)을 그룹과 배지로 구분한다.
2. 고객이 본인의 OpenRouter 키로 임의의 OpenRouter 모델을 호출하는 Custom 모델(BYOK)을 제공한다. 모델 ID는 고객이 입력하고 키는 필수다.
3. **플랫폼 키 접근 불가 보장**: Custom 모델 호출이 플랫폼 `OPENROUTER_API_KEY`, 플랫폼 `OPENAI_API_KEY`(OpenAI SDK 환경변수 폴백) 어느 쪽에도 닿는 경로가 없어야 한다. 서버 검증, Go resolver, Python 러너 3중 방어로 보장하고 각 방어를 독립 테스트로 고정한다(4절).
4. 요청 단위 ZDR(zero data retention)을 모든 Custom 호출에 강제한다. ZDR 제공자가 없는 모델은 호출이 실패하는 것이 정상 동작이다(fail-closed).
5. 키 로깅 정책 준수. 신규 코드는 키를 로깅하지 않고, 기존 누출 지점 16곳(필드 로깅, aicallhandler 3곳 포함)과 오류 경로 요청 본문 로깅(6곳), webhook-manager 로깅(3.4 B4)은 이번 PR에서 id 또는 uri/method/길이만 남기도록 한 줄씩 수정한다(기계적 수정, 전용 회귀 테스트 없음). 공용 모듈 `PublishWebhook` 로깅과 webhook-manager의 범위는 대표님 결정 항목이다(Q10, 3.4, 8.1 항목 8, 권장은 이번 PR 포함).

### 1.2 비목표

- **전역 on/off 플래그를 만들지 않는다.** 기능 노출은 카탈로그 항목의 존재 여부로 결정되며 환경변수, 설정, 기능 플래그를 추가하지 않는다.
- 새 컴포넌트, 새 구조체 계층, 새 설정 파일, 새 DB 컬럼/마이그레이션을 만들지 않는다(운영 DB의 저장된 `openrouter.*`는 0건이므로 마이그레이션도 없다).
- 직접 모델(OpenAI/Gemini/Grok)의 빈 키 거부(Q1b), `GET /ais` 응답의 `engine_key` 제거, 로그 중앙 처리(logrus Hook 등)는 이번 범위가 아니다(별도 이슈).
- OpenRouter 키 형식 사전 검증(`sk-or-` 접두 등)은 하지 않는다(Q7, 공식 근거 미확인).
- ZDR 제공자 존재 여부를 저장 시점에 미리 조회하지 않는다(7절 참조, 네트워크 의존과 데이터 신선도 문제).
- 자체 호스팅 문서는 변경하지 않는다(BYOK는 플랫폼 키가 필요 없다).
- text-chat `engine_openai_handler`, Builder, summary/analysis 핸들러는 대상이 아니다(분석서 2.7, 비테스트 호출자 없음 확인 결과를 이번에 `grep`으로 재확인하지는 않았고 분석서 결과를 따른다).

## 2. 확정 결정 (대표님 승인, 2026-10-06 "권장대로 가자")

| # | 결정 | 확정 내용 |
|---|---|---|
| Q1a | 키 방식 값 체계 | 직접 모델 11개 전부 `own_or_default` 동일 적용. `key_mode` 3값: `platform`, `own_or_default`, `own_required`. 그룹 3개(PLATFORM PROVIDED / YOUR OWN KEY / CUSTOM), 배지 3종(`No key needed` / `Your key or default` / `Your key required`) |
| Q1b | 직접 모델 빈 키 | 거부하지 않는다(현 동작 유지, 배지로 정직하게 표기). 필수화는 별도 이슈 |
| Q2 | Custom 항목 제공 방식 | 서버 카탈로그(`GET /ai_models`)가 제공. 기존 테스트의 `openrouter` 금지 가드를 Custom 항목 한정 예외로 완화(3.1.5) |
| Q3 | 고객 노출 영어 문구 | 3.6.3 표의 최종안 사용(전문은 3.6.3) |
| Q4 | 모델 ID 검증 규칙 | 허용 문자 영숫자와 `. _ - /`, 최대 길이 255(접두 포함). `:` 변종 접미사와 `openrouter/auto` 같은 라우터형 ID 거부(실증 전 fail-closed). 3.1.3 |
| Q5 | ZDR 제공자 없음 | 실호출 검증 전까지 호출 거부(fail-closed). 고객 안내 문구는 실호출 후 확정하고 이 문서에는 기본 동작과 후보만 둔다(3.7) |
| Q6 | `platform_managed` | 이번 PR은 유지, `key_mode`만 추가(분석서 권장) |
| Q7 | 키 형식 검증 | 하지 않음 |
| Q8 | 키 이월 UX | create는 키 방식 그룹이 바뀔 때 키 입력을 비움, detail/sidebar 편집은 저장 키가 있고 그룹 또는 서비스 접두가 바뀌어 가드가 켜질 때 입력을 비우고 목적지별 경고 문구를 표시(3.6.5) |
| Q9 | sidebar 편집 키 손실 | 편집 진입 시 `GET ais/{id}` 재조회를 모든 AI에 적용(3.6.6) |
| Q10 | 키 로깅 | 16곳 id만 남기는 한 줄 수정을 이번 PR 필수로 처리. 전용 회귀 테스트는 만들지 않음. 오류 경로 로깅(B)은 수정회차 2에서 필수(request 필드 6곳)와 권장(`%v` 5곳)으로 정리했고, 수정회차 3에서 webhook-manager를 B4(필수 제안)로 편입했으며 공용 `publish.go`와 webhook-manager 범위는 8.1 항목 8로 올렸다(3.4) |
| Q11 | 문서 | `ai_struct_ai.rst`, `ai_models.rst`, `skill.md`, `llms.txt`(해당 시) 정정/추가. `engine_key` 응답 문구 정정은 실증 후 같은 PR 선택 |

분석서의 "제안안, 승인 대기" 표기는 이 문서에서 모두 확정으로 읽는다. 이 문서가 새로 만든 세부 설계 중 대표님 확인이 따로 필요한 것은 8절에 모았다.

### 2.1 수정회차 2 CPO 제안 (제안, 대표님 이견 없으면 확정)

디자인 리뷰 1회차 지적을 풀기 위해 CPO가 정한 결정이다. 대표님 승인 사항(위 표)과 구분한다.

| # | 제안 | 반영 위치 |
|---|---|---|
| a | 모델 ID 입력란은 `ModelPicker` 내부에 둔다(선택값이 `model_id_prefix`로 시작하면 트리거 밑에 렌더). 4개 폼에 자동 적용되고 새 파일은 없다 | 3.6.2 |
| b | 선택 모델이 `own_required`이면 sidebar의 Advanced details를 강제로 연다. **수정회차 4에서 제안 m(details 밖 조건부 렌더)으로 대체됨** | 3.6.4 |
| c | 키 방식이 저장값에서 `own_required`로 바뀔 때(수정 화면) 키를 새로 입력해야 저장된다. 새 상태 없이 기존 변경 플래그(`engineKeyChanged` 등)를 재사용한다. **수정회차 3에서 양방향으로 일반화(2.2 제안 g)되고, 수정회차 4에서 기준이 "저장된 키가 비어 있지 않음"으로 바뀜(2.2 제안 k, l). 현재 규칙은 3.6.5의 가드 하나(그룹 또는 서비스 접두 변경, 제안 s)이며 이 제안이 말한 `own_required` 목적지는 그 부분집합이다** | 3.6.5 |
| d | 키 입력란 `type=password` 전환은 이번 범위에서 제외한다. 기존 동작을 유지하고 별도 이슈로 기록한다 | 3.6.5, 8.3 |
| e | 서버 오류 reason은 오류 `message` 문자열에 reason 코드가 포함되는지로만 판단한다(JSON 파싱을 시도하고 실패하면 일반 문구로 폴백). `provider.js`는 수정하지 않는다 | 3.6.4 |

### 2.2 수정회차 3 CPO 제안 (제안, 대표님 이견 없으면 확정)

디자인 리뷰 2회차 지적을 풀기 위해 CPO가 정한 결정(f~j)이다. k~n은 디자인 리뷰 3회차 지적을 풀기 위해 수정회차 4에서, o~r은 디자인 리뷰 4회차 지적을 풀기 위해 수정회차 5에서, s~v은 디자인 리뷰 5회차 지적을 풀기 위해 수정회차 6에서 추가했다(s~v은 새 규칙이 아니라 기존 규칙의 확장과 삭제다). 수정회차 7은 리뷰 6회차 지적을 제안 p의 트리거 정정으로만 처리했고 새 제안과 새 규칙은 없다. 제안 w는 리뷰 7회차 지적을 풀기 위해 수정회차 8에서 추가했으며 역시 새 규칙이 아니라 제안 p 비움 트리거의 비교값 확장이다.

| # | 제안 | 반영 위치 |
|---|---|---|
| f | 로깅 정리 범위는 aicallhandler 3곳을 포함해 16곳이고, webhook-manager는 이번 PR 필수 B4로 편입한다(한 줄씩 기계적 수정, 전용 회귀 테스트 없음). 8.1 항목 8은 "공용 모듈 `publish.go` + webhook-manager" A/B/C로 확장하고 권장은 A(이번 PR) | 3.4, 8.1 항목 8, 5.3, 6.1, 6.2 |
| g | 키 이월 보완은 양방향으로 일반화한다. 저장값의 키 방식 그룹과 현재 선택의 키 방식 그룹이 다르면 키를 새로 입력해야 저장된다(같은 `engineKeyChanged` 계열 플래그 재사용). **적용 기준은 수정회차 4에서 제안 k로 대체됨(그룹 비교만으로는 platform 원본이 우회됨)** | 3.6.5 |
| h | sidebar 인라인 오류는 기존 배너(`createError`, `saveError`)에도 같은 문구를 실어 다른 탭에서도 보이게 한다 | 3.6.4, 3.6.6 |
| i | `ModelPicker`의 오류 전달은 `AIEngineFields`의 `modelError`를 `ModelPicker`의 `error` prop으로 넘기는 한 경로로 통일한다 | 3.6.2 |
| j | trim은 저장 직전 UI에서 양끝 공백 제거만 허용한다. 서버는 trim하지 않는다 | 3.1.3, 3.6.2, 3.6.4 |
| k | 키 이월 가드 기준을 "저장된 키 방식 그룹이 다르다"에서 **"저장된 키가 비어 있지 않고 목적지 키 방식 그룹이 저장 그룹과 다르다(platform 원본이라도 저장 키가 있으면 포함)"**로 변경한다. 목적지 `own_or_default`이면 새로 입력한 값이 비어 있어도 저장을 허용하고(빈 값은 기본 키 사용, 이전 키 재전송 안 함), 목적지 `own_required`이면 새 키를 입력해야 하며, 목적지 `platform`이면 요구하지 않는다. 제안 g를 대체. **(b)는 수정회차 6에서 제안 s로 확장됨** | 3.6.5, 3.6.6, 4절 R1-h, 5.2, 8.3 |
| l | 유효 키 정의: 가드가 적용되는(그룹 변경) 저장에서는 **새로 입력한 값만**(trim 후 비어 있지 않음) 유효하다. `savedEngineKey`, `detailData.engine_key` 폴백을 금지하고 포커스만 한 경우도 입력으로 치지 않는다. **판정은 수정회차 6에서 제안 t로 단순화됨** | 3.6.4 검증 2, 3.6.5 |
| m | sidebar에서 `own_required`일 때 키 블록을 `<details>` 밖에 조건부로 렌더하는 소형 대안을 채택한다. `forceOpen`, `onToggle` 재오픈 장치와 그에 붙은 테스트 4개는 제거한다(jsdom 동작이 미실증인 장치보다 단순). 제안 b를 대체 | 3.6.4, 3.6.3, 5.2, 8.2 |
| n | Popover 포커스는 하나로 고정한다: Custom 행을 선택할 때마다 ID 입력란에 포커스(`focusIdRef` 방식) | 3.6.2, 5.2 |
| o | sidebar 가드의 "저장 모델" 기준선은 낡은 캐시(`aiData.engine_model`)가 아니라 **재조회 응답의 `engine_model`을 담는 `savedEngineModel`**로 한다. 저장 키(`savedEngineKey`)와 저장 모델이 같은 서버 응답에서 오게 하고, 재조회 전이거나 실패(`keyLoad !== 'ready'`)면 Save를 비활성화한다. 재조회 응답은 `setAiData`/캐시에 쓰지 않는다 | 3.6.5, 3.6.6, 5.2 |
| p | 편집 화면(detail, sidebar 편집)에서 **가드 적용 여부(`guardApplies`)가 false에서 true로 바뀌는 전이에서 입력란을 비운다**(수정회차 7에서 트리거를 `keyMode` 전이에서 이 전이로 정정, 편집 화면의 `prevKeyModeRef`는 `prevGuardRef`로 대체되어 제거됨. **수정회차 8에서 제안 w로 키 방식 그룹 변경도 트리거에 추가됨**. 같은 그룹 안 벤더 변경(OpenAI에서 Gemini)도 가드가 켜지는 전이이므로 닫힌다). 비우는 대상은 이전 모드에서 새로 타이핑한 키(sidebar `engineKeyChanged=false`, `engineKey=''`)와 detail에서 저장 키로 미리 채워진 입력란이다. 가드가 true인 채 이어지는 추가 모델 변경에서는 이미 비어 있으므로 비우지 않고, 그 사이 사용자가 새로 타이핑한 키는 유지한다(입력란이 화면에 보이는 상태의 사용자 입력). true에서 false로 바뀌는 전이(저장 모델 쪽으로 복귀)에서는 입력을 저장값 상태로 되돌린다. create는 저장 키가 없어 가드가 없으므로 키 방식 그룹 전이에서만 비운다(`prevCreateKeyModeRef`) | 3.6.5, 5.2 |
| q | 키 동일 비교와 "입력함" 판정은 양쪽 `trim()` 후 한다. **수정회차 6에서 제안 t로 대체됨(저장 키와의 비교 자체를 삭제)** | 3.6.5 |
| r | 경고 문구를 목적지별로 구분한다: `own_required` 목적지 `Replace the API key before saving.`, `own_or_default` 목적지 `The saved key may belong to a different provider. Replace it, or leave it empty to use the default key.`, `platform` 목적지 `The saved key will be removed.`(**sidebar 편집에서만 표시**. detail은 platform 저장 시 저장 키를 재전송하는 기존 계약 `ais_detail.js:422`, 테스트 `views/ais/__tests__/ais_detail.test.js:435-448`을 유지하므로 키가 지워지지 않아 이 문구가 사실과 다르다. 따라서 detail의 platform 목적지에는 경고를 표시하지 않고 입력란만 비운다. detail도 platform에서 키를 지우도록 바꿀지는 대표님 확인 사항) | 3.6.3, 3.6.5, 8.1 항목 9 |
| s | 가드 (b)를 확장한다: **목적지 키 방식 그룹이 저장 그룹과 다르거나, 현재 `engineModel`과 저장 `engine_model`의 서비스 접두(첫 `.` 앞)가 다르다.** 낡은 캐시 또는 사용자의 직접 변경으로 같은 그룹 안에서 다른 벤더(예: `openai.`에서 `gemini.`)로 바뀌어도 저장 키가 전송되지 않는다(방안 A. 같은 그룹 안 벤더 변경을 잔존 경로로 두는 방안 B는 채택하지 않음). 같은 접두 안의 모델 변경과 Custom ID 변경은 가드를 적용하지 않는다. 제안 k의 (b)를 대체 | 3.6.1, 3.6.5, 4절 R1-h, 5.2, 8.3 |
| t | 유효 키 판정을 하나로 단순화한다: **입력란(trim 후)이 비어 있지 않음.** detail은 `typed.trim() !== ''`, sidebar 편집은 `engineKeyChanged && engineKey.trim() !== ''`. 전이 시 입력이 비워지므로(제안 p) 입력란에 값이 있다는 것 자체가 사용자의 새 입력이다. 따라서 "저장 키와 다름" 비교, 제안 q의 동일 비교, 5.2 행 15, 8.3의 (iii)를 삭제한다. 제안 l, q를 대체 | 3.6.4, 3.6.5, 5.2, 8.3 |
| u | 경고는 가드가 적용되고 입력란이 `<details>` 안에 있으면 보이지 않는다. 그래서 sidebar 편집에서 경고가 필요한 상태(가드 적용)에서는 키 블록을 `<details>` 밖에 렌더한다. 제안 m의 조건부 렌더를 "`own_required`이거나 경고 필요(가드 적용)"로 확장할 뿐이며 새 장치는 없다 | 3.6.4, 3.6.5, 5.2 |
| v | detail에서 platform 목적지는 입력란이 비어 보이나 저장 키는 서버에 유지된다(현행 재전송 계약 유지, 8.1 항목 9 권장 유지). 이 사실을 문서에 그대로 적는다. 호출에는 쓰이지 않으며 이후 다른 그룹으로 바꿀 때 가드가 다시 걸린다 | 3.6.5, 8.1 항목 9 |
| w | 편집 화면(detail, sidebar 편집)의 입력란 비움 트리거를 **'(1) 키 방식 그룹(`keyMode`)이 바뀔 때 또는 (2) `guardApplies`가 false에서 true로 바뀔 때'**로 한다. create의 그룹 전이 비움(`prevCreateKeyModeRef`)과 같은 조건에 저장 키 가드 전이를 더한 것이며 새 규칙이 아니라 기존 비교값의 확장이다. `prevGuardRef` 한 ref 객체(`{ keyMode, guardApplies }`)로 두 비교를 함께 처리한다. 같은 그룹 안에서 벤더만 바뀔 때는 저장 키가 있을 때(`guardApplies`)만 비운다(저장 키가 없으면 사용자가 화면에서 보고 타이핑한 값이고 저장 키 유출이 아니므로 유지, create와 동일). 제안 p의 트리거를 확장 | 3.6.5, 5.2 |

## 3. 아키텍처 변경 상세

### 3.1 ai-manager

#### 3.1.1 카탈로그와 `key_mode`

현재: `ModelEntry{ID,Label,Vendor,Route,UpstreamSlug,Recommended,Tags,Description}`, `Route`는 `RouteDirect/RouteOpenRouter`(`models/ai/catalog.go:6-9,17-26`), `ModelInfo`는 7필드, `PlatformManaged = (Route==RouteOpenRouter)`(`catalog.go:27-35,52`).

변경:

```go
// RouteCustomOpenRouter marks the single customer-keyed OpenRouter entry. It is
// never a valid engine_model by itself: the customer appends a model ID.
RouteCustomOpenRouter Route = "custom_openrouter"

const (
	KeyModePlatform     = "platform"        // platform supplies the key
	KeyModeOwnOrDefault = "own_or_default"  // customer key, or the platform default when empty
	KeyModeOwnRequired  = "own_required"    // customer key is mandatory
)
```

- `ModelInfo`에 두 필드 추가: `KeyMode string json:"key_mode"`(항상 존재), `ModelIDPrefix string json:"model_id_prefix,omitempty"`(Custom 항목에만 존재). `PlatformManaged`는 유지.
- `key_mode` 도출은 `CatalogView()` 안에서 `Route`로 결정한다(데이터 중복 없음): `RouteDirect`는 `own_or_default`, `RouteOpenRouter`는 `platform`, `RouteCustomOpenRouter`는 `own_required`. 개별 항목이 값을 따로 들고 있지 않으므로 일관성 테스트는 `Route` 대 `key_mode` 대응표 하나다.
- **Custom 항목**: `ID: "custom.openrouter"`, `Label: "OpenRouter model (your OpenRouter key)"`, `Vendor: "OpenRouter"`, `Description: "Enter any model supported by OpenRouter."`, `ModelIDPrefix: "openrouter."`(엔트리 필드는 `CustomPrefix`로 두고 뷰에서 노출). 카탈로그 슬라이스 마지막에 둔다.
- **Custom 항목 ID를 `openrouter.`가 아닌 불투명 값 `custom.openrouter`로 둔 이유**: (1) 이 ID는 어떤 `engine_model`로도 유효하지 않아야 한다(저장하면 `INVALID_ENGINE_MODEL`). 접두 자체(`openrouter.`)를 ID로 쓰면 빈 모델 ID가 카탈로그 정확 일치로 통과할 위험이 생긴다. (2) 분석서 R4가 요구한 "어떤 카탈로그 ID도 `openrouter.`로 시작하지 않는다" 불변 조건을 예외 없이 유지한다. (3) CLI/SDK 사용자는 `model_id_prefix`가 있는 항목을 "접두 + 자유 입력" 항목으로 해석한다(문서화, 3.5).
- `ResolveEngine`의 정확 일치 루프는 `RouteCustomOpenRouter` 항목을 건너뛴다(`custom.openrouter`를 저장하려 하면 `Rejected`).
- 정적 슬라이스 방식 유지. 새 카탈로그 로딩 구조 없음.
- `verify_openrouter_catalog.py`는 `catalog.go`에서 정규식 `UpstreamSlug:\s*"..."`로 슬러그를 읽는다(`bin-pipecat-manager/scripts/pipecat/verify_openrouter_catalog.py:27`). Custom 항목은 `UpstreamSlug`를 갖지 않으므로 스크립트는 변경이 없고, `Test_CatalogInvariants`가 "Custom 항목은 `UpstreamSlug`가 비어 있다"를 고정한다(분석서에 없던 의존, 3.8 참조).

#### 3.1.2 `ResolveEngine`, `IsValidEngineModel`, 키 검증

현재 시그니처는 `ResolveEngine(m) (Resolved, Outcome)`와 `IsValidEngineModel(m) bool`(`models/ai/resolve.go:28,50`)이다. 호출자는 비테스트 코드 기준 `aihandler/chatbot.go:51,148`, `cmd/ai-control/main.go:413,431`, pipecat `llmresolve.go:18`뿐이다(이번에 `grep`으로 확인).

변경:

```go
const EngineModelPrefixCustomOpenRouter = "openrouter."

const OutcomeCustomOpenRouter Outcome = ... // added after OutcomeDirectPassthrough

type Resolved struct {
	Entry      *ModelEntry
	RunnerType string
	BlankKey   bool // platform models: never forward the customer key
	RequireKey bool // custom models: an empty key is a hard failure
}
```

`ResolveEngine` 흐름(순서가 보안 속성이다):

1. 카탈로그 정확 일치(Custom 항목 제외). 기존과 같음.
2. 접두 `openrouter.`(소문자, 정확 일치)로 시작하면 나머지를 `ValidateCustomModelID`로 검사. 통과하면 `Resolved{RunnerType: string(m), RequireKey: true}, OutcomeCustomOpenRouter`, 실패하면 `Rejected`. **`RunnerType`은 `platform_openrouter.` 접두를 만들지 않고 `BlankKey`는 항상 false.** 플랫폼 경로(`RunnerServicePlatformOpenRouter + "." + UpstreamSlug`)와 코드 경로를 공유하지 않는다.
3. 기존 `openai/gemini/grok` 통과 접두.
4. 나머지 `Rejected`.

`OpenRouter.x`, `OPENROUTER.x`, 앞뒤 공백이 있는 값은 접두 비교가 대소문자 구분이므로 2번을 통과하지 못하고(허용 문자에도 공백 없음) 3번에서도 접두가 맞지 않아 `Rejected`다. 이 동작이 Go가 통과시킨 값이 Python에서 `platform_openrouter`로 해석되지 않음을 보장하는 근거 중 하나다(Python은 서비스명을 소문자화하므로, Go가 `openrouter.` 접두 고정 문자열만 통과시킨다는 점이 핵심이다).

**키 검증 함수(create/update/ai-control 공용)**:

```go
// ValidateEngine checks the final (model, key) state a write would leave behind.
// modelChanged limits model-ID validation to changed models so a legacy stored
// value keeps saving; the key rule always applies to the final state.
func ValidateEngine(model EngineModel, key string, modelChanged bool) error
```

- `modelChanged`이고 `ResolveEngine`이 `Rejected`면 `ErrInvalidEngineModel` 계열 오류.
- 최종 모델이 `openrouter.` 접두를 가지면(`RequireKey` 대상) `strings.TrimSpace(key) == ""`일 때 키 필수 오류. 모델이 바뀌지 않았어도 키가 비면 거부(분석서 R1-e).
- 오류 타입 구분을 위해 sentinel 두 개(`ErrInvalidEngineModel`, `ErrEngineKeyRequired`)를 반환하고, aihandler가 `cerrors.InvalidArgument`로 변환한다(`ai` 패키지는 `cerrors`를 import하지 않는 현재 구조를 유지).
- `IsValidEngineModel`은 이름과 시그니처를 유지한다(기존 호출 컴파일 보존). 의미는 `Rejected` 여부이며 Custom 유효 ID는 true를 반환한다. 키 검사는 `ValidateEngine`만 한다.

**aihandler 적용**:

- `Create`(`chatbot.go:51`): `IsValidEngineModel` 호출을 `ValidateEngine(engineModel, engineKey, true)`로 교체.
- `Update`(`chatbot.go:148`): `ValidateEngine(engineModel, engineKey, engineModel != preUpdateAI.EngineModel)`. `preUpdateAI` 조회는 기존 코드가 이미 한다. 최종 상태 기준이므로 sidebar에서 이름만 바꾸며 `engine_key`를 생략한 PUT은 BYOK AI에서 400이 된다(그래서 3.6.6의 재조회가 필수다).
- ai-control(`cmd/ai-control/main.go:413,431`): `validateEngineModelCreate/Update`에 `engineKey` 인자를 추가해 같은 함수를 호출한다. ai-control은 마지막에 `aihandler.Create/Update`를 직접 호출하므로(`main.go:242,366`) 최종 방어는 aihandler에 있고 CLI 사전 검증은 빠른 실패용이다(분석서는 CLI가 별도 방어선인 것처럼 적었으나 코드상 같은 핸들러를 거친다). `ai-control update`에서 `--engine-model` 없이 호출하면(`engineModel==""`) 기존대로 사전 검증을 건너뛰고 핸들러가 판단한다.

#### 3.1.3 모델 ID 검증 규칙 (Q4)

`ValidateCustomModelID(id string) bool`, 입력은 접두 `openrouter.`를 뗀 나머지다.

| 규칙 | 내용 |
|---|---|
| 길이 | 접두 포함 전체 255자 이하(DB `engine_model varchar(255)`, `bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql:11`), ID 부분은 244자 이하. 빈 값 거부 |
| 허용 문자 | `[A-Za-z0-9._-]`와 `/`만. 공백, 제어문자, 쉼표, `:`, `~`, `@` 등 모두 거부 |
| 형태 | `author/slug` 정확히 슬래시 1개. 양쪽 모두 영숫자로 시작. 정규식 `^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$` (8절 항목 1) |
| 콜론 변종 | `:free`, `:nitro`, `:online`, `:thinking`, `:floor` 등 `:` 포함 값은 허용 문자 규칙에 의해 거부된다. 별도 규칙이 아니라 허용 문자 집합으로 막으므로 우회 불가 |
| 라우터형 ID | `author`가 `openrouter`(대소문자 무시)이면 거부(`openrouter/auto`, `openrouter/free` 등). `~` 별칭(`~anthropic/...` 류)도 허용 문자 밖이라 거부 |
| 정규화 | 서버는 하지 않는다. 입력 그대로 저장하고 그대로 전송한다(대소문자 변환 금지, 서버는 trim 안 함). 양끝 공백이 있는 값은 허용 문자에 공백이 없어 `INVALID_ENGINE_MODEL`로 거부된다. 양끝 공백 제거는 UI가 저장 직전에 한 번만 수행한다(3.6.2, 3.6.4, CPO 제안 j) |

거부 사유: 변종 접미사와 라우터형 ID는 요청한 모델과 다른 모델/제공자로 라우팅되거나 `provider.zdr` 강제와 상호작용하는 방식이 실증되지 않아 ZDR 우회 가능성을 배제할 수 없다. 실호출 검증(7절 항목 7, 8)에서 ZDR 유지가 확인된 변종만 이후 순차 허용한다(허용 시 규칙 표 한 줄 수정).

**슬래시 1개 규칙의 보안 의미**: `author/slug` 형태 고정은 OpenRouter 형식 검증이기도 하지만 모델 ID 입력란에 API 키를 붙여 넣는 사고를 막는 보조 방어다. 키 문자열(`sk-or-v1-...` 류)에 슬래시가 없다고 가정하면(미확인) 입력란에 키가 붙여 넣어졌을 때 이 규칙에서 거부되어 `engine_model`로 저장되지 않는다. 키에 슬래시가 포함되는 형식이면 이 방어는 통과된다(모델 문자열은 저장 후 로그 필드 `engine_model`과 UI 화면에 그대로 나타나므로 저장되면 키가 노출된다). 실제 OpenRouter 키 형식은 공식 근거를 확인하지 못했으므로(Q7) 완전 방어로 주장하지 않고 보조 방어로만 둔다. 이 시나리오는 5.2에서 **실제 키가 아닌 더미 문자열**(`dummy-key-not-real`, 문서 전체에서 이 한 가지 표기만 쓴다)로 테스트한다.

오류: reason은 `INVALID_ENGINE_MODEL`(HTTP 400) 재사용, 단 **Custom 접두 입력 전용 메시지를 새로 둔다**(기존 메시지는 `%q`로 입력값을 되돌려 보여주고, 기존 테스트 `assertInvalidEngineModel`이 메시지에 `openrouter`, `api key` 포함을 금지하기 때문, `chatbot_engine_model_error_test.go:48`). 모델 ID 입력란에 키를 잘못 붙여 넣는 사고가 가능하므로 Custom 전용 메시지는 **입력값을 되돌려 보여주지 않는다**.

```
invalid engine_model: the OpenRouter model ID is not valid. Use the vendor/model-name form with letters, digits, '.', '_' and '-' only. Variant suffixes such as ':free' and router IDs such as 'openrouter/auto' are not supported.
```

Python은 모델 ID 형식을 다시 검증하지 않는다(검증 소유자는 Go resolver 하나이며, resolver는 세션 시작마다 같은 함수를 실행하므로 저장 시점과 실행 시점 모두 검증된다).

**입력 원문 반향 방어(수정회차 3, 코드로 다시 확인)**: 위 "되돌려 보여주지 않는다" 규칙이 서버 오류 메시지에만 적용되면 다른 줄에서 같은 원문이 새어 나간다. 직접 확인한 반향 지점은 다음 3곳이다.

| 위치 | 현재 | 처리 (같은 줄 수정 때 함께) |
|---|---|---|
| `bin-api-manager/pkg/servicehandler/ai.go:64`(`AICreate`), `:401`(`AIUpdate`) | `logrus.Fields`에 `"engine_model": engineModel`(검증 전 원문) | 값이 `openrouter.`(대소문자 무시) 접두로 시작하면 원문 대신 `openrouter.<redacted>`를 기록한다. 같은 파일의 비공개 헬퍼 1개(`loggableEngineModel`)로 두 줄에 적용한다. 3.4 A의 `:66`, `:403` 수정과 같은 줄 묶음이다 |
| `bin-ai-manager/cmd/ai-control/main.go:414`, `:432` | `fmt.Errorf("invalid engine model: %s", engineModel)` | 접두 `openrouter.`(대소문자 무시)로 시작하면 원문 없이 `invalid engine model: the OpenRouter model ID is not valid`를 반환한다. 그 외 입력은 현행 유지 |
| `bin-ai-manager/pkg/aihandler/chatbot.go:22-28`(`errInvalidEngineModel`, 반향 줄은 `:26`) | `%q`로 원문 반향. 현재 Custom 전용 분기는 없다 | **신규 추가**: `Create`/`Update`가 `errInvalidEngineModel`을 부르기 전에 `openrouter.` 접두 입력을 위 Custom 전용 메시지(입력값 미반향)로 분기하는 헬퍼를 이 PR이 새로 만든다. 비 Custom 입력은 현행 |

저장된 `engine_model`은 검증을 통과한 값이라 이후 로그 필드(`engine_model`)에 남아도 키가 아니다. 위 처리는 **검증 전** 원문이 오류와 로그로 나가는 경로만 막는다.

#### 3.1.4 키 필수 오류 `ENGINE_KEY_REQUIRED`

`chatbot.go:20-28`의 `errInvalidEngineModel`과 같은 방식의 신규 헬퍼:

```go
func errEngineKeyRequired() error {
	return cerrors.InvalidArgument(
		commonoutline.ServiceNameAIManager,
		"ENGINE_KEY_REQUIRED",
		"An API key is required for custom OpenRouter models.",
	)
}
```

- HTTP 400(`cerrors.HTTPStatusFor(StatusInvalidArgument)`, 기존 테스트가 같은 방식으로 확인), 메시지는 키 값을 포함하지 않는다.
- `INVALID_ENGINE_MODEL`과 reason을 분리해 UI/SDK가 "키 입력 안내"와 "모델 ID 오류"를 분기하게 한다.

#### 3.1.5 Custom 항목 도입으로 깨지는 기존 테스트와 완화 방법 (Q2)

`openrouter` 문자열을 금지하는 가드는 4곳이다. 분석서는 3곳(+`Test_CatalogInvariants` 라우트 검사)을 적었고 4번째(`chatbot_engine_model_error_test.go:48`)는 이번에 새로 확인했다.

| # | 위치 | 현재 가드 | 완화 방법 (Custom 항목 1개로 한정) |
|---|---|---|---|
| 1 | `models/ai/catalog_test.go:41-51` `Test_CatalogPublicViewHidesInternals` | 공개 뷰 JSON 전체에 `slug`, `route`, `openrouter`, `meta-llama/` 금지 | 뷰를 `model_id_prefix`가 있는 항목(Custom)과 나머지로 나눈다. **나머지 항목의 JSON은 기존 4개 토큰을 그대로 금지(약화 없음).** Custom 항목에는 `slug`, `route`, `meta-llama/`를 계속 금지하고 `openrouter`(대소문자 무시)는 **허용 필드 `id`, `label`, `vendor`, `description`, `model_id_prefix` 5개에서만** 허용한다(Custom 항목의 `id`=`custom.openrouter`, `vendor`=`OpenRouter`가 이 문자열을 포함하기 때문이며 이 둘을 빼면 현재 명세와 모순된다). `key_mode`, `platform_managed`, `tags`, `recommended` 등 그 외 필드에는 `openrouter`가 나타나면 실패한다. `model_id_prefix`가 있는 항목이 **정확히 1개**임을 단정 |
| 2 | `catalog_test.go:152-163` `Test_CatalogCustomerFacingTextHasNoBannedTerms` | 라벨/설명에 `openrouter`, `zero data` 등 금지 | `Route == RouteCustomOpenRouter`인 항목에 한해 `label`, `description`에서만 `openrouter`를 허용(위 1번 허용 필드 목록 중 이 테스트가 검사하는 두 필드). `zero data`와 타사명 금지는 Custom 항목에도 유지(ZDR 안내는 카탈로그 설명이 아니라 UI/문서 문구라서 충돌 없음) |
| 3 | `pkg/listenhandler/v1_ai_models_test.go:46` | 응답 본문에 `slug`, `openrouter`, `meta-llama/`, `"route"` 금지 | 디코딩한 항목 중 `model_id_prefix`가 있는 항목(정확히 1개)을 분리한다. 나머지 항목을 재직렬화한 문자열에는 기존 금지 토큰을 그대로 적용(약화 없음). Custom 항목은 1번과 같은 허용 필드 5개(`id`, `label`, `vendor`, `description`, `model_id_prefix`)에서만 `openrouter`를 허용하고 `slug`, `route`, `meta-llama/`는 금지. 필수 키 목록(`:38`)에 `key_mode` 추가 |
| 4 | `pkg/aihandler/chatbot_engine_model_error_test.go:48` `assertInvalidEngineModel` | `INVALID_ENGINE_MODEL` 메시지에 `openrouter`, `platform_`, `api key`, `engine_key` 금지 | **완화하지 않는다.** 이 헬퍼는 일반(비 Custom) 입력 오류용으로 그대로 둔다. Custom 접두 입력 오류는 3.1.3의 전용 메시지를 검사하는 새 헬퍼(`assertInvalidCustomModelID`)를 추가한다: 메시지에 `openrouter` 허용, 입력값이 메시지에 되돌려지지 않음, `platform_`와 `engine_key`는 계속 금지 |

허용 필드 목록(`id`, `label`, `vendor`, `description`, `model_id_prefix`)은 테스트 코드에 상수로 고정하고(5.2), 필드가 추가되거나 다른 필드에 `openrouter`가 나타나면 실패하게 한다. 1, 3번 가드에 동일 목록을 적용하고 2번은 그 중 `label`, `description`만 해당한다. 4번 가드는 위와 같이 완화하지 않는다.

추가로 Custom 항목 도입에 따른 변경:

- `Test_CatalogInvariants`(`catalog_test.go:9-38`): 알려진 라우트에 `RouteCustomOpenRouter` 추가, Custom 항목은 정확히 1개, `UpstreamSlug` 비어 있음, `ModelIDPrefix == "openrouter."`, `ID`가 `openrouter.`로 시작하지 않음(모든 항목 공통 불변 조건 신규 추가).
- `Test_CatalogViewShape`, `Test_CatalogViewPlatformManagedMatchesRoute`(`:103-147`): `platform_managed`는 Custom에서 false, `key_mode`가 `Route`와 대응표대로임을 같은 방식으로 검사. 이 테스트의 "managed와 direct 둘 다 존재" 조건은 유지.

### 3.2 api-manager, OpenAPI, 생성 코드

- 비즈니스 로직 변경 없음. `POST/PUT /ais`는 `req.EngineKey`를 그대로 전달하고(`server/ais.go:91,311`), 카탈로그는 `[]*amai.ModelInfo`를 그대로 통과시킨다(`servicehandler/ai.go:276-291`). **다만 api-manager는 `bin-ai-manager` 모듈의 `amai.ModelInfo` 타입으로 역직렬화하므로(`go.mod`의 `replace monorepo/bin-ai-manager => ../bin-ai-manager`, 컴파일 시 포함) api-manager를 재빌드/재배포하지 않으면 새 필드(`key_mode`, `model_id_prefix`)가 응답에서 사라진다.** 롤아웃에서 ai-manager와 api-manager는 함께 나가야 한다(6절). 분석서 R7은 "Go 쪽 변경은 `ModelInfo`뿐"이라고만 적어 이 배포 결합은 빠져 있었다.
- 로그 4줄(`servicehandler/ai.go:66,121,403,465`)은 Q10에 따라 id만 남기도록 수정(3.4).
- `bin-openapi-manager/openapi/openapi.yaml`의 `AIManagerAIModel`(`:2018-2058` 부근)에 추가:
  - `key_mode`: string enum `[platform, own_or_default, own_required]`, `required` 목록에 추가. description 예: `How the engine key is handled. platform: the platform supplies access and engine_key is ignored. own_or_default: your engine_key is used, or the platform default key when empty. own_required: engine_key is mandatory.`
  - `model_id_prefix`: string, optional. description 예: `Present only for entries where you type the model ID yourself. Send engine_model as this prefix followed by the model ID.`
  - `platform_managed` 설명은 유지(Q6).
- `engine_key` 설명(`openapi.yaml:2196-2198` "Write-only; not returned in responses.")은 실증(7절 항목 4)에서 응답에 포함됨이 확인된 경우에만 "returned"로 정정한다. 미확인이면 정정하지 않는다.
- 재생성 순서: `bin-openapi-manager`에서 `go generate ./...`, 이어 `bin-api-manager`에서 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`(각 `CLAUDE.md`의 절차). 산출물 `bin-openapi-manager/gens/models/gen.go`, `bin-api-manager/gens/...`.

### 3.3 pipecat-manager Go 경로

현재 `resolveSessionLLM(llmType, aiKey)`는 `Rejected`면 오류, `BlankKey`면 키를 비우고 반환, 그 외 `aiKey`를 그대로 전달한다(`llmresolve.go:17-28`). 호출 지점 4곳: `start.go:41`(키 없는 사전 분류), `start.go:208`, `run.go:141`, `run.go:209`.

**문제**: `start.go:41`은 DB row를 만들기 전 모델 분류만 하려고 `""`를 넘긴다. Custom에 "빈 키는 오류" 규칙을 단순히 넣으면 Custom 모델 세션이 항상 시작 전에 막힌다.

변경:

```go
// classifySessionLLM only decides whether the model is selectable. It never
// looks at a key, so the keyless pre-check in Start() keeps working for custom models.
func classifySessionLLM(llmType pipecatcall.LLMType) error

// resolveSessionLLM keeps its signature. For custom OpenRouter models it fails
// when the trimmed key is empty and forwards only the trimmed key.
func resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType, runnerKey string, err error)
```

| 지점 | 변경 |
|---|---|
| `start.go:41` `Start()` | `resolveSessionLLM(llmType, "")`를 `classifySessionLLM(llmType)`로 교체. `Rejected`만 오류 |
| `start.go:208` `startReferenceTypeAIcall` | 변경 없음(키를 넣어 `resolveSessionLLM` 호출). Custom + 빈 키(AI 조회 실패 포함, `:198-205`는 warn 후 진행)는 `resolveSessionLLM`이 오류를 반환해 세션 시작 실패. direct 모델은 기존 동작(빈 키 진행) 유지. **이 실패는 `Start()`가 `h.Create`로 pipecatcall row를 만든 뒤(`start.go:45`) 일어나므로 row가 남는다.** `Start()`에는 `startReferenceType*` 실패 시 row를 정리하는 코드가 없고(`start.go:65-80` 직접 확인) 다른 사후 실패(예: `CallV1CallGet` 실패)도 같은 동작이다. 이번 범위에서 정리 로직은 추가하지 않는다 |
| `run.go:141` `runGetLLMKey` | 변경 없음. 조회 실패/비 AIcall 참조 유형은 `aiKey==""`로 들어가 Custom이면 거부(의도된 fail-closed, 분석서 R1-c). 이 조합의 실제 발생 여부는 미확인 |
| `run.go:209` 팀 멤버 | 변경 없음. 거부 시 기존 처리(`llm_type=""`, `engine_key=""`, ids만 로깅, `run.go:217-225`)를 그대로 쓴다. Python이 `llm_type==""`이면 초기화에서 예외 |
| `pythonrunner.go:87` | `LLMKey`는 `json:"llm_key,omitempty"`라 빈 키는 필드가 생략된다 -> Python에서 `None`. 이 사실이 Python 방어(3.3의 Python 항)가 `None`을 반드시 처리해야 하는 근거다. 구조체는 변경 없음 |

`resolveSessionLLM`의 키 정제: Custom이면 `strings.TrimSpace(aiKey)`를 반환하고 비면 오류 `custom engine key is empty`(키 값이나 모델 ID 원문을 오류에 넣지 않는다).

**`Rejected` 오류의 모델 문자열 포함 여부 (결정)**: 현재 오류는 `engine model is not available: %s`로 모델 문자열 원문을 포함한다(`llmresolve.go:19-21`). 이 값은 호출자가 `errors.Wrapf`로 감싸 상위에서 로깅될 수 있다. **`openrouter.` 접두(대소문자 무시)로 시작하는 문자열은 오류에 포함하지 않고** `engine model is not available`만 반환한다. 접두 이후는 고객이 입력한 자유 문자열이라 키가 붙여 넣어진 값일 수 있기 때문이다. 그 외 문자열(카탈로그 모델, 직접 접두)은 현행대로 포함한다(고객 입력 자유 문자열이 아님). 테스트: `openrouter.` 접두 거부 입력의 오류 문자열에 입력값이 없음(5.2).

**Python 러너 새 분기** (`bin-pipecat-manager/scripts/pipecat/run.py`, 현재 `create_llm_service` `:591-689`):

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

기존 `platform_openrouter` 분기(`:658-687`)는 인라인 dict 대신 `_build_openrouter_llm(os.getenv(...), model_name)`을 호출하도록 정리(동작 동일, 기존 `TestPlatformOpenRouter` 통과 유지). 새 분기:

```python
elif service_name == "openrouter" and "." in type:
    # Custom (BYOK) OpenRouter: the customer's key only. This branch must never
    # read os.environ, and must never hand None or "" to the SDK: the OpenAI
    # client falls back to OPENAI_API_KEY when api_key is None.
    api_key = (key or "").strip()
    if not api_key:
        raise ValueError("An API key is required for custom OpenRouter models.")
    llm = _build_openrouter_llm(api_key, model_name)
    ... (tools_schema, ctx, aggregator: platform_openrouter 분기와 동일)
```

- `"." in type` 조건의 의미(`create_llm_service`의 앞단 분기 `run.py:593-598`을 직접 읽고 정리): 앞단은 `"." in type`이면 첫 점에서 분리하고, 아니면 `":" in type`이면 첫 콜론에서 분리한다. 점 형태 값(Go가 만드는 유일한 형태)은 이 조건이 항상 참이므로 이 조건 자체는 점 형태를 걸러내지 못한다. 서비스명이 `openrouter`이면서 점이 없는 값은 콜론 분기(`OpenRouter:x/y`)로만 도달할 수 있고 이때 조건이 거짓이라 새 분기를 건너뛰어 기존대로 `Unsupported LLM service`로 남는다(콜론에 점이 포함된 값은 점에서 먼저 분리되어 서비스명이 `openrouter`가 아니다). 즉 이 조건은 콜론 형태가 새 분기로 들어오지 못하게 하는 Python 쪽 이중 방어이고, **콜론 형태 거부의 1차 근거는 Go resolver의 `openrouter.`(소문자, 정확 일치) 접두 규칙이다**(3.1.2). 기존 테스트 `test_raw_openrouter_is_unsupported`의 `OpenRouter:x/y` 행을 유지할 수 있다.
- `key`가 `None`으로 도착하는 것은 정상 경로다(3.3 `pythonrunner.go:87`, `main.py:85,129`). `(key or "").strip()`은 `None`, `""`, 공백 모두를 같은 예외로 만들고 `None`의 `.strip()` `AttributeError`를 피한다. 정제된 값만 SDK에 전달한다.
- SDK 폴백은 소스로 재확인했다: `openai/_client.py`는 `api_key is None`일 때만 `os.environ.get("OPENAI_API_KEY")`를 읽고(`~/.hermes/cache/scratch/ormvenv`의 openai 3.24.0), 빈 문자열은 폴백하지 않는다. 컨테이너 실행 실증은 7절 항목 5.
- ZDR은 `_OPENROUTER_PROVIDER` 단일 상수를 두 분기가 공유하며, 호출마다 `dict(...)`로 복사해 `extra_body`에 넣는다(요청 단위 강제, 한 요청이 다른 요청의 dict를 변이시키지 않음). **Custom 분기에서 provider 옵션을 제거하거나 재시도 시 제외하는 코드를 두지 않는다**(ZDR 실패 시 비 ZDR로 재시도 금지).
- `_member_llm_type`(`run.py:571-588`)는 **변경하지 않는다.** `llm_type`이 `"openrouter.a/b"`로 넘어오면 그대로 신뢰하고(Go resolver 산출물), `""`면 예외, 필드 자체가 없는 구버전 Go면 `engine_model`의 서비스명이 `openrouter` 또는 `platform_openrouter`일 때 예외를 유지한다. 이유: 구버전 Go는 Custom을 만들지 않으므로 이 폴백은 항상 거부가 맞고, 변경하지 않으면 롤아웃 중 신 Python + 구 Go 조합이 안전하다(3.8 참조: 분석서가 바뀐다고 본 `test_init_pipeline.py:804,827,890`은 이 결정에 따라 의미가 바뀌지 않는다).
- 플랫폼 `OPENROUTER_API_KEY`와 `OPENAI_API_KEY`는 compose `bin-pipecat-manager/komodo/docker-compose.yml:109,157,161,189,231,235`에서 주입된다(이번에 라인 직접 확인). 이 분기의 방어는 환경 제거가 아니라 코드 경로 차단이다(환경에서 키를 빼는 것은 플랫폼 모델과 직접 모델이 사용하므로 불가).

### 3.4 키 로깅 정리 (Q10)

구성: **A** 필드 로깅 16곳(필수), **B1** 오류 경로 요청 본문 로깅 6곳(필수), **B2** marshal 오류 `%v` 5곳(권장), **B3** 공용 모듈 `PublishWebhook` 로깅(대표님 결정, 8.1 항목 8), **B4** webhook-manager 로깅(필수 제안, 범위는 8.1 항목 8에서 확인).

**로그 레벨 사실**: ai-manager `initLog`가 레벨을 무조건 `DebugLevel`로 설정한다(`bin-ai-manager/internal/config/main.go:329`의 `logrus.SetLevel(logrus.DebugLevel)`, `initLog`는 `:326-330` 직접 확인). 따라서 Debug와 Error 로그가 모두 출력된다고 보고 A와 B1을 필수로 처리한다. 운영 로그 레벨/포매터 실증(7절 항목 9)에 의존하지 않는다(항목 9는 이미 쌓인 로그 조회용으로만 남는다).

#### A. 필드 로깅 16곳 (필수)

분석서 R9 A의 13곳(api-manager 4, ai-manager 8, call-manager 1)을 직접 열어 재확인했고, 수정회차 3에서 **aicallhandler 3곳을 추가**해 총 **16곳**(api-manager 4, ai-manager 11, call-manager 1)이다. 모두 현재 코드에 있다.

| 서비스 | 위치 | 수정 방향 |
|---|---|---|
| api-manager | `pkg/servicehandler/ai.go:66`, `:403` | `logrus.Fields`에서 `"engine_key": engineKey` 줄 삭제 |
| api-manager | `servicehandler/ai.go:121`, `:465` | `log.WithField("ai", tmp)`를 `log.WithField("ai_id", tmp.ID)`로 |
| ai-manager | `pkg/aicallhandler/start.go:260,302,486` | 함수 전체 `log` 필드의 `"ai": a`를 `"ai_id": a.ID`로 |
| ai-manager | `pkg/aihandler/db.go:150`, `pkg/aihandler/direct_hash.go:28` | `WithField("ai", a)`를 `WithField("ai_id", a.ID)`로 |
| ai-manager | `pkg/aicallhandler/send.go:170`, `tool.go:1078` | 동일 |
| ai-manager | `pkg/teamhandler/handler.go:244` | 동일 |
| ai-manager (**수정회차 3 추가**) | `pkg/aicallhandler/start.go:726`(`startReferenceTypeNone`), `pkg/aicallhandler/db.go:39`(`Create`), `:119`(`CreateByMessaging`) | `"ai": c`를 `"ai_id": c.ID`로. 세 줄 모두 `c`는 `*ai.AI`다(함수 시그니처에서 직접 확인: `c *ai.AI`) |
| call-manager | `pkg/callhandler/start_incoming_domain_type_sip.go:223` | 동일(`"ai"`에 `*amai.AI`). 이전 문서의 `:222`는 오기이며 `log.WithField("ai", a).Debugf(...)`는 `:223`에 있다(직접 확인) |

- 기계적 한 줄 수정, 로직 변경 없음, 로그 메시지와 나머지 필드 유지. 전용 회귀 테스트는 만들지 않는다(대표님 확정).
- **aicallhandler 3곳의 실행 경로(수정회차 3, 코드로 확인)**: 이전 분석서는 이 3곳의 `"ai": c`를 `*aicall.AIcall`이라 적었으나 틀렸다. 실제 타입은 `*ai.AI`(`engine_key` JSON 필드 보유, `models/ai/main.go:67`)다. `db.go:39`의 `Create`는 realtime 경로(`start.go:1131` `startAIcallByRealtime`)의 모든 aicall 생성에서, `db.go:119`의 `CreateByMessaging`은 messaging 경로(`start.go:1200`)에서 실행되며, 로그 항목이 `:81`(`Creating aicall.` Debug), `:84`, `:91`(오류) 등 이후 모든 줄에 실린다. `start.go:726`의 `startReferenceTypeNone`은 `Start()`의 `ReferenceTypeNone` 분기(`start.go:207`)에서 호출된다. 즉 **상시 경로**이며 AI 호출이 시작될 때마다 로그에 키가 남는다. 이 3곳을 빼면 BYOK 키가 AI 세션 시작 때마다 기록된다.
- 이 3곳의 nil 위험: `db.go:39`, `:119`는 로그 직후 `c.CustomerID`(`:50` 부근)를 역참조하므로 `c.ID`가 새 nil 위험을 만들지 않는다. `start.go:726`은 8.2에서 구현 시 확인한다.
- 수정 전후 컴파일 영향 확인: `a`가 nil일 수 있는 지점은 기존 코드가 이미 `a.ID`를 같은 줄 `Debugf` 인자로 쓰는 지점(`db.go:150` 등)이라 nil 위험이 늘지 않는다. `start.go`의 3곳은 구현 시 `a`의 nil 가능성을 확인한 뒤 수정한다.

#### B1. 오류 경로 요청 본문 로깅 (필수)

코드 사실(이번에 다시 읽고 확인):

- `pkg/listenhandler/main.go:283-286` `processRequest`의 `log`가 `"request": m`을 필드로 가진다. `m`은 `*sock.Request`이며 `Data json.RawMessage`(요청 본문)를 포함한다(`bin-common-handler/models/sock/message.go:6-11`). 이 `log`는 `:631`(핸들러 없음)과 `:642`(핸들러 오류)의 `Errorf`에서 출력된다.
- `pkg/listenhandler/v1_ais.go`의 5개 핸들러가 같은 구조다: `:24`(GET 목록), `:83`(POST), `:159`(ID GET), `:198`(DELETE), `:237`(PUT)의 `"request": m`. 이 파일의 로그 호출은 모두 오류 분기에 있다(성공 경로 로그 호출 없음). POST는 `:88`(본문 파싱 실패), `:102`(whitelist 검증 실패), `:128`(`Could not create ai`), `:135`, PUT은 `:242`, `:271`, `:276`, `:302`, `:309` 등이 해당한다.
- logrus는 필드를 해당 항목의 모든 로그 호출에 싣는다. 따라서 **`engine_key`가 담긴 POST/PUT 본문이 오류가 날 때마다 로그에 남는다.** BYOK에서는 키 누락/모델 ID 오류(`ENGINE_KEY_REQUIRED`, `INVALID_ENGINE_MODEL`)가 고객의 정상적인 입력 실수 흐름이고 이 오류는 `:128`/`:302`(`Could not create ai`/`Could not update ai`)로 떨어지므로, 사실상 키가 담긴 요청이 로그에 남는 상시 경로다(성공 경로에는 로그가 없다는 점은 위에서 확인).
- 선례: 같은 파일 `main.go:276-278`이 Assistant Builder 경로를 이 로그 항목이 만들어지기 전에 분기해 본문 로깅을 피한다("that entry puts the whole request, body included, into a field"). 같은 원리를 적용한다.

수정(필수, 이번 PR):

| 위치 | 수정 방향 |
|---|---|
| `pkg/listenhandler/main.go:285` | `"request": m`을 `"uri": m.URI, "method": m.Method, "request_id": m.RequestID`로(본문 제외, 상관관계 추적용 `request_id`는 유지) |
| `pkg/listenhandler/v1_ais.go:24,83,159,198,237` | 동일 |

- `:128`, `:302`의 `Errorf`는 필드가 정리되면 키를 싣지 않는다. 오류 문자열 `err`에 키가 들어가는지도 확인했다: 이번 PR이 추가하는 `ValidateEngine` 오류 2종(sentinel 2개, 3.1.2)과 `ENGINE_KEY_REQUIRED` 메시지(3.1.4)는 키 값을 포함하지 않는다. DB 계층 오류 문자열이 바인드 값을 포함하는지는 미확인이다.
- 변경은 로그 필드뿐이며 응답, 로직은 변하지 않는다. listenhandler 테스트 중 로그 출력을 단정하는 것은 Builder 본문 비로깅 테스트(`v1_ai_builder_test.go:221-224`, 로그 출력을 버퍼로 캡처)뿐이고 `/v1/ais` 핸들러 테스트는 로그를 단정하지 않는다(grep 확인). 같은 선례를 따라 **B1에는 로그 캡처 테스트 1개를 추가한다**(5.2: 더미 키가 담긴 POST가 오류로 끝나도 캡처한 로그에 더미 키가 없음). A의 16곳은 기존대로 전용 테스트를 만들지 않는다.
- **같은 구조의 다른 핸들러**(`v1_mcpservers.go` 등)는 이번 PR에서 `engine_key`를 싣지 않으므로 필드 수정 대상이 아니다. 다만 `main.go:285` 수정은 모든 라우트의 `:631`/`:642` 경로에 적용된다. mcp server 비밀값이 본문으로 들어오는 핸들러가 같은 방식으로 로깅하는지는 이번에 확인하지 않았으며 별도 이슈 후보로 8.3에 기록한다.

#### B2. marshal 오류 `%v` 로그 5곳 (권장, 같은 파일이라 함께 수정)

`v1_ais.go:66,142,181,220,316`의 `Could not marshal the response message. message: %v, err: %v`는 `tmp`(`*ai.AI` 포함)를 값으로 출력한다. JSON marshal 실패 시에만 도달하므로 실제로는 거의 도달하지 않는다고 판단한다(추측). 같은 파일을 B1에서 이미 수정하므로 `message: %v` 인자와 문구를 제거해 함께 정리하기를 권장한다(필수 아님).

#### B3. 공용 모듈 `PublishWebhook` 로깅 (대표님 결정, 8.1 항목 8)

이 지점은 이전 13곳 목록에 빠져 있던 누출이다.

- `bin-common-handler/pkg/notifyhandler/publish.go:34` `PublishWebhook`의 로그 항목이 `"data": data`를 가진다. 이 `data`는 호출자가 넘긴 도메인 구조체 그대로이며, ai-manager에서는 `*ai.AI`(`engine_key` JSON 필드, `models/ai/main.go:67`)다. 출력은 `:46`(`CreateWebhookEvent` marshal 실패)과 `:51`(`WebhookV1WebhookSend` RPC 실패)의 `Errorf`에서만 일어난다. 웹훅 전송 RPC 실패(타임아웃 등)는 현실적인 오류 경로다.
- **영향 범위(직접 grep 확인)**: `PublishWebhook`의 호출처는 `PublishWebhookEvent`(`publish.go:26`)뿐이며(백그라운드 고루틴) `PublishWebhookEvent`를 비테스트 코드에서 호출하는 서비스는 16개다: agent, ai, call, campaign, conference, contact, conversation, email, flow, message, number, outdial, queue, talk, transcribe, webchat. customer-manager, registrar-manager는 이 함수를 호출하지 않는다.
- 이 중 비밀값 필드를 JSON으로 가진 타입이 넘어가는 서비스는 **ai-manager**(`engine_key`)와 **conversation-manager**(`models/account/account.go:21-22`의 `secret`, `token`, 호출 `accounthandler/db.go:94,162,194`)다. 나머지 14개 서비스는 `json` 태그에 secret/password/token/key/credential/auth 류 이름을 가진 필드를 가진 타입이 없음을 `CreateWebhookEvent` 구현 타입들에서 확인했다(이름 패턴 검사이므로 다른 이름의 비밀값은 미확인).
- **권장안**: 영향이 작으므로 **이번 PR에서 함께 수정**한다. `publish.go`의 `"data": data` 한 줄을 삭제한다(`customer_id`, `event_type`은 유지). 시그니처, 동작 변경 없음(로그 필드만), 공용 모듈이라 각 서비스는 재빌드/재배포되어야 반영되며 필수 반영 대상은 ai-manager(BYOK 배포에 포함)와 conversation-manager이고 나머지는 다음 정기 배포에서 반영된다. `bin-common-handler`의 `go test`와 의존 서비스 빌드 확인이 필요하다(5.3). 이 한 줄은 오류 경로(`:46`, `:51`)만 막는다. 웹훅 페이로드가 webhook-manager에서 상시 기록되는 경로는 B4가 닫는다. 결정 근거와 대안(별도 이슈로 분리, 미수정)은 8.1 항목 8.
- 같은 `publish.go:88` `PublishEvent`의 `json.Marshal(data)` 내부 이벤트 버스 발행은 로그가 아니라 데이터 흐름이며 4절 잔존 위험과 8.1 항목 3에서 다룬다.

#### B4. webhook-manager 로깅 (필수 제안, 수정회차 3 신설)

코드 사실(이번에 직접 읽고 확인):

- **상시 경로**: `bin-webhook-manager/pkg/webhookhandler/webhook.go:32-34`(`SendWebhookToCustomer`)와 `:126-128`(`SendWebhookToURI`)의 `log.WithFields(logrus.Fields{"data_type": dataType, "data": data}).Debugf("Sending an webhook. ...")`. `data`는 `json.RawMessage`로 **웹훅 페이로드 전문**이다. 오류 경로가 아니라 웹훅 전송마다 출력된다. ai 웹훅 페이로드는 `ConvertWebhookMessage`가 `EngineKey`를 복사하므로(`models/ai/webhook.go:29,70`, `json:"engine_key,omitempty"`) `ai_created`, `ai_updated` 웹훅마다 BYOK 키가 이 줄로 기록될 수 있다(conversation-manager의 account `secret`, `token`도 같은 이유로 해당). 이 줄은 B3(`publish.go`, 오류 때만 출력)와 다른 서비스에서 실행되는 별개 누출이다.
- **오류 경로**: `pkg/listenhandler/main.go:150`의 `"request": m`(`processRequest`, `:181`, `:192`의 `Errorf`에서 출력), `pkg/listenhandler/v1_webhooks.go:19`(`processV1WebhooksPost`)와 `:56`(`processV1WebhookDestinationsPost`)의 `"request": m`, 그리고 같은 파일 `:29`, `:66`의 `data: %v`(`m.Data` 원문), `:35`, `:72`의 `message: %v`(`req.Data`). `m`은 `*sock.Request`라 본문(`Data`)을 포함한다. 리뷰에서 언급된 `:42`, `:77` 부근은 `SendWebhook...` 실패 시 `err`만 출력하는 줄이라 수정 대상이 아니고 실제 지점은 위의 줄들이다.
- **로그 레벨**: `internal/config/main.go:100` `initLog`가 `DebugLevel`을 무조건 설정한다(직접 확인). 상시 `Debugf`가 출력된다.

수정(필수 제안, 이번 PR, 기계적 수정):

| 위치 | 수정 방향 |
|---|---|
| `webhookhandler/webhook.go:32-34`, `:126-128` | `"data": data`를 `"data_len": len(data)`로(`data_type`은 유지) |
| `listenhandler/main.go:150`, `v1_webhooks.go:19`, `:56` | `"request": m`을 `"uri": m.URI, "method": m.Method, "request_id": m.RequestID`로(B1과 같은 방식) |
| `v1_webhooks.go:29`, `:66`, `:35`, `:72` | `data: %v`, `message: %v` 인자와 문구를 제거하고 `err`만 남긴다 |

- 총 9줄이며 로직, 응답, 시그니처 변경 없음. **전용 회귀 테스트는 만들지 않는다**(A와 같은 원칙). `bin-webhook-manager` 전체 `go test`로 기존 테스트가 로그 필드에 의존하지 않는지만 확인한다(5.3).
- 범위는 8.1 항목 8에서 대표님이 확인한다(권장 A: publish.go와 함께 이번 PR). B 또는 C가 선택되면 B4는 해당 방침으로 이동한다.

#### 그 외

- 신규 코드(이번 PR이 추가하는 모든 로그 줄)는 `*ai.AI`, 요청 본문, 키, 모델 ID 원문(Custom 입력)을 로깅하지 않는다. 식별자만 사용한다.
- 이미 쌓인 로그의 보관/폐기, 기존 OpenAI/Gemini/Grok 키 교체 권고는 대표님 판단 사항(8.1 항목 4).
- timeline-manager `subscribehandler/main.go`(`select`는 `:169`, Warn 로그는 `:173`)는 이벤트 채널이 가득 차면 `"event": m`(데이터 포함)을 Warn으로 로깅한다. 드롭 시에만 출력되는 별도 지점이며 이번 범위에 넣지 않고 8.3에 기록한다.

### 3.5 문서 (Q11)

- `bin-api-manager/docsdev/source/ai_struct_ai.rst`: `engine_key` 설명(`:54`)에 Custom 유형 추가(`engine_model`이 `openrouter.<model-id>`이면 필수이며 비면 `400 ENGINE_KEY_REQUIRED`), 오류표와 모델 표(`:196-214`)에 Custom 행, ZDR 안내(모든 요청에 강제, ZDR 제공자가 없는 모델은 호출 실패), 비용은 고객 OpenRouter 계정에 청구되며 플랫폼 키는 사용되지 않는다는 문구. **PUT 의미 명시**: `PUT /ais/{id}`는 전체 교체이며 `engine_key`를 생략하면 빈 값으로 처리되므로(`server/ais.go:311`이 `req.EngineKey`를 그대로 전달, `aihandler/db.go:280` `buildUpdateFields`), **`engine_model`이 `openrouter.<model-id>`인 AI에 `engine_key`를 생략하거나 비워 보내는 PUT은 `400 ENGINE_KEY_REQUIRED`가 된다**(3.1.2의 최종 상태 기준 검증). 키를 바꾸지 않는 수정에서도 기존 키를 다시 보내야 한다는 점을 `ai_struct_ai.rst` Update 설명에 적는다. `:208`의 `platform_managed` 설명은 `key_mode` 설명과 함께 정리(`platform_managed`는 유지).
- `bin-api-manager/docsdev/source/ai_models.rst`: 응답 예시에 `key_mode`, Custom 항목 예시(`id: custom.openrouter`, `model_id_prefix: openrouter.`)와 "Custom 항목의 `id`는 `engine_model`로 직접 쓸 수 없고 `model_id_prefix` + 모델 ID로 조합한다" 설명, `INVALID_ENGINE_MODEL`/`ENGINE_KEY_REQUIRED` 설명(현재 `:10`은 `INVALID_ENGINE_MODEL`만 언급). 영어 문서이므로 본문도 영어로 작성한다.
- `monorepo-javascript/square-main/public/skill.md`와 `llms.txt`(`:17,49`): `engine_key`가 "`platform_managed`가 false인 모델에서 필요"라는 현재 문구를 `key_mode` 기준으로 갱신(`own_required`는 필수, `own_or_default`는 선택). 갱신 범위는 `skill.md`의 `:412`(`engine_key` 설명), `:829`(`platform_managed` 설명 "`false` means the customer's own provider key is required"), `:843`(접두 `openrouter.` 허용 안내 문장 추가), `:941`("For models with `platform_managed: false`, verify `engine_key`..." 문구)이다(직접 확인한 라인). BYOK 언급은 `skill.md`에만 두고 `llms.txt`는 해당 문장만 수정한다.
- `ai_struct_ai.rst:77`("never returned")와 `openapi.yaml:2196-2198` 정정은 3.2의 조건을 따른다.
- docsdev 문서 변경 후 `bin-api-manager/docsdev`의 Sphinx 빌드 절차와 `build/html` 갱신 여부는 구현 시 `CLAUDE.md`의 문서 규칙을 따른다(이번에 빌드 산출물 추적 여부는 확인하지 않음).

### 3.6 square-admin UI (monorepo-javascript, 별도 저장소)

#### 3.6.1 데이터 계층

- `types/api.ts:481` 부근 `AIModel`에 `key_mode: 'platform' | 'own_or_default' | 'own_required'`, `model_id_prefix?: string` 추가.
- `views/ais/useAIModels.js`: 기존 `isPlatformManaged`(`:94`)는 유지(다른 호출자 호환). 신규:
  - `keyModeOf(modelValue)`: 정확 일치 항목의 `key_mode`. 없고 `model_id_prefix`가 있는 항목의 접두로 시작하면 그 항목의 `key_mode`(저장된 `openrouter.*`는 `own_required`). 그 외(카탈로그에 없는 저장값)는 `platform_managed`로 폴백하지 않고 `own_or_default`(분석서 R7: `key_mode` 부재 시 `platform_managed`가 true면 platform, 아니면 own_or_default).
  - `servicePrefixOf(modelValue)`(수정회차 6, 제안 s): 모델 값에서 첫 `.` 앞 문자열(`openai.gpt-5.1`은 `openai`, `openrouter.a/b`는 `openrouter`, `.`이 없으면 값 전체). 카탈로그 값은 모두 `<service>.<model>` 형태다(`anthropic.`, `gemini.`, `grok.`, `openai.` 등, 코드로 확인). 같은 파일의 순수 함수 하나이며 새 파일은 없다. 3.6.5 가드 (b)에서만 쓴다.
  - `customEntry`: `model_id_prefix`를 가진 항목(없으면 undefined, 구 API 호환).
  - `customIdMissing(modelValue)`: 값이 Custom 접두로 시작하고 접두 뒤가 비어 있으면(공백만 있어도) true. 클라이언트 검사는 이것뿐이다(3.6.4).
  - `engineErrorOf(err)`: 서버 오류를 `{kind: 'key' | 'model', message}`로 분류(3.6.4). 같은 파일의 순수 함수로 두고 새 파일은 만들지 않는다.
- 서버가 구버전이라 `key_mode`가 없는 경우의 폴백은 위 규칙으로 처리한다(신 UI + 구 API, 롤아웃 순서가 바뀌어도 깨지지 않음).
- `aiModelsFixture.js`(`views/ais/aiModelsFixture.js:3-10`)에 `key_mode`와 Custom 항목 추가.
- **목록, 상세 화면의 Custom 원본 ID 표시(영향 확인)**: 저장된 `openrouter.<id>`는 카탈로그 정확 일치가 없어 `modelLabel`이 원본 ID를 그대로 반환한다(`useAIModels.js:95`, `byId.get(id)?.label || id || ''`). 사용처 `teamgraph/nodes/member.js:205`(툴팁 `title`), `aicalls/aicalls_detail.js:376,381,647`, `ais/InsightAIsPanel.js:178`, `ais/ais_list.js:57`(`ModelLabel`)은 모두 이 함수를 거치므로 오류 없이 원본 ID가 표시된다. 이번 범위에서 표시 가공은 하지 않는다. `member.js:202-205`의 provider 배지(`getProviderStyle`)가 `openrouter.` 접두를 어떻게 분류하는지는 8.2에서 확인한다.

#### 3.6.2 ModelPicker와 모델 ID 입력 (`views/ais/ModelPicker.js`, CPO 제안 a)

**입력란 위치 결정 근거(코드 재확인)**: `ModelPicker`는 sidebar 생성(`teamgraph/sidebar.js:945`)과 편집(`:1302`)에서 직접 렌더되고, `AIEngineFields`는 `ais_create.js:545`, `ais_detail.js:1264`에서만 쓰이며 그 안에서 `ModelPicker`(`AIEngineFields.js:143,340`)를 렌더한다. 이전 문서처럼 `AIEngineFields`에 입력란을 두면 sidebar 두 곳이 빠진다. `ModelPicker` 내부에 두면 4개 폼 모두 자동 적용되고 새 파일이 필요 없다.

- 구조: 검색창, `Recommended`(현재와 같음), 이어 키 방식 그룹 3개 `PLATFORM PROVIDED`, `YOUR OWN KEY`, `CUSTOM`. 현재의 벤더 그룹 헤더(`groupByVendor`, `VENDOR_ORDER`)는 키 방식 그룹으로 대체하고 벤더는 행의 보조 표기로 옮긴다. 그룹은 `keyModeOf`로 결정하며(서버 `key_mode` 값만 사용, 구 API는 폴백 규칙), 항목이 없는 그룹은 숨긴다. 그룹 내 순서는 카탈로그 순서. 그룹 헤더 접기 동작은 유지한다(그룹 단위, `collapsed` 키를 벤더명에서 그룹 키로 변경).
- 행에 배지 표시(태그 `Low cost` 옆): `platform` 은 `No key needed`, `own_or_default` 는 `Your key or default`, `own_required` 는 `Your key required`. 배지 문구 상수 맵 하나(`KEY_MODE_BADGES`)로 두고 키 방식 판단은 서버 값만 쓴다(클라이언트 하드코딩 없음).
- Recommended 행(Gemini 2.5 Flash)도 같은 배지를 단다. 별도 "추천 전용" 배지는 없다(Q1a).
- **선택값 판정**: `selected = getModel(value) ?? (value가 customEntry.model_id_prefix로 시작하면 customEntry)`. 트리거에는 Custom 라벨과 벤더를 보이고 합성 행 `Current: <id>`는 만들지 않는다(`ModelPicker.js:63,176-188`의 `showCurrentRow` 조건에 `!isCustomValue` 추가). 카탈로그 밖의 다른 값은 기존대로 합성 행. 행 `aria-selected`와 체크 표시(`m.id === value`)는 Custom 행에서만 `value`가 접두로 시작하면 true로 바꾼다.
- **Custom 행 선택**: 값이 이미 접두로 시작하면 `onChange(value)`(타이핑한 ID 유지), 아니면 `onChange(prefix)`. **Custom 행을 선택할 때마다**(처음 고르는 경우와 이미 접두로 시작하는 값에서 재선택하는 경우 모두) ID 입력란에 포커스를 준다(CPO 제안 n). 포커스 조건은 이 하나뿐이며 "접두와 정확히 같아지는 전이" 같은 별도 조건은 두지 않는다.
  - **Popover 닫힘 포커스 경합(코드 확인, 수정회차 3)**: `pick()`은 `onChange` 직후 `handleOpenChange(false)`로 Popover를 닫는다(`ModelPicker.js:62-65`). Radix Popover는 닫힐 때 `onCloseAutoFocus`에서 트리거로 포커스를 되돌리는 기본 동작이 있어, 입력란 포커스를 즉시 주면 곧바로 트리거가 포커스를 가져간다. `PopoverContent`는 props를 그대로 Radix `Content`에 전달하므로(`components/ui/popover.jsx`) 다음처럼 막는다: `focusIdRef`(포커스 요청 플래그)를 Custom 행 선택에서 세우고, `<PopoverContent onCloseAutoFocus={(e) => { if (focusIdRef.current) { e.preventDefault(); idInputRef.current?.focus(); focusIdRef.current = false } }}>`로 닫힘 처리 시점에 입력란으로 포커스를 준다. 처음 고르는 경우 입력란은 `onChange`로 값이 접두가 된 렌더에서 마운트되고, 재선택이면 이미 마운트되어 있어 닫힘 처리 시점에는 `idInputRef`가 채워져 있다(코드 읽기 기준, 동작은 jsdom 테스트로 확인: 5.2). Custom이 아닌 행 선택은 기본 동작(트리거로 포커스 복귀)을 유지한다.
- **ID 입력란 (ModelPicker 내부)**: `ModelPicker`가 `Popover` 뒤에 조건부 블록을 함께 반환한다(Fragment). 조건: `customEntry`가 있고 `value`가 접두로 시작할 때.
  - 구성: 라벨 `OpenRouter Model ID *`(3.6.3, `useId`로 `htmlFor` 연결), `Input`(`font-mono`, `compact`이면 `h-8 text-xs`), 도움말, 오류 문구.
  - **ID 값의 소유자는 부모 폼의 `engineModel`(sidebar 생성은 `createModel`) 문자열 하나다.** `ModelPicker`는 상태를 새로 갖지 않는다. 입력란 값은 `value.slice(prefix.length)`, 변경 시 `onChange(prefix + 입력값)`. 입력 중에는 값을 변환하지 않는다(대소문자 변환 없음, 타이핑 중 trim 없음). 양끝 공백은 **저장 직전에만** UI가 제거한다(3.6.4 클라이언트 검증 0단계, CPO 제안 j). 서버는 trim하지 않으며 양끝 공백이 남은 값은 `INVALID_ENGINE_MODEL`로 거부한다(3.1.3과 같은 규칙). 이렇게 하면 최종 `engineModel` 문자열이 접두 포함 전체 값이라 `ais_detail.js:541-542`의 dirty 비교가 그대로 동작한다(분석서 R5).
  - **인라인 오류 표시 위치와 전달 경로(CPO 제안 i, 한 경로로 통일)**: `ModelPicker`에 선택 prop `error`(문자열)를 추가한다. 표시 위치는 Custom 값이면 ID 입력란 바로 아래, 그 밖에는 트리거 아래이며 `text-destructive`다. 전달 경로는 **하나**다: `AIEngineFields`가 받은 `modelError`를 `ModelPicker`의 `error`로 넘기고, `AIEngineFields`의 기존 `modelError` 문단은 **제거**한다(두 곳에 중복 표시되는 것을 막는다). 현재 이 문단은 `ModelPicker` 바로 뒤가 아니라 `CatalogStatus` 다음에 있고 **두 곳**이다: compact 레이아웃 `AIEngineFields.js:145`(`ModelPicker` `:143`, `CatalogStatus` `:144` 다음)와 non-compact 레이아웃 `:343`(`ModelPicker` `:340`, `CatalogStatus` `:342` 다음)의 `{modelError && ...}` 블록이다. 두 블록을 모두 제거하고 두 `ModelPicker` 호출(`:143`, `:340`) 모두에 `error={modelError}`를 넘긴다. 지금까지 어떤 폼도 이 prop을 넘기지 않음을 grep으로 확인했다. 두 폼(`ais_create`, `ais_detail`)이 `AIEngineFields`에 `modelError`를 넘기고, sidebar는 `ModelPicker`를 직접 쓰므로(`:945`, `:1302`) 자체 `modelError` 상태(생성, 편집 각 1개)를 `error` prop으로 넘긴다.
- 헤더 주석 "There is deliberately no free-text entry"(`:36-38`)는 "free-text entry exists only for the custom entry (a model ID typed after its prefix)"로 갱신.

#### 3.6.3 영어 문구 최종안 (Q3)

| 위치 | 문구 |
|---|---|
| 그룹 | `PLATFORM PROVIDED` / `YOUR OWN KEY` / `CUSTOM` |
| 배지 | `No key needed` / `Your key or default` / `Your key required` |
| Custom 항목 라벨 | `OpenRouter model (your OpenRouter key)` |
| Custom 항목 설명 | `Enter any model supported by OpenRouter.` |
| 모델 ID 입력 라벨 | `OpenRouter Model ID *` |
| 모델 ID 도움말 | `Enter the model ID from the OpenRouter model page, e.g. vendor/model-name.` |
| 모델 ID 비어 있음(클라이언트) | `Enter the OpenRouter model ID.` |
| 모델 ID 서버 오류 폴백(JSON 파싱 실패 시) | `The OpenRouter model ID is not valid.` |
| 키 입력 라벨(`own_required`) | `OpenRouter API Key *` |
| 안내 1 | `Calls use this key and are billed to your OpenRouter account. The platform key is never used.` |
| 안내 2 | `Zero data retention (ZDR) routing is enforced on every request. If no ZDR provider supports this model, calls will fail.` |
| 안내 3 | `The AI cannot be saved without an API key.` |
| 키 필수 오류 | `An API key is required for custom OpenRouter models.` (서버 `ENGINE_KEY_REQUIRED` 메시지와 동일 문자열) |
| 플랫폼 모델 키 캡션 | `API key not required for this model.` (현재 코드는 `API key not required`, `ais_create.js:585`, `ais_detail.js:1309`, `sidebar.js:970,1365`. 목업의 문구로 통일) |
| 내 키 또는 기본 키 모델 캡션 | `Leave empty to use the platform default key.` (8.1 항목 2) |
| 키 방식 변경 경고(Q8, 제안 r, 목적지별) | 목적지 `own_required`: `Replace the API key before saving.` / 목적지 `own_or_default`: `The saved key may belong to a different provider. Replace it, or leave it empty to use the default key.` / 목적지 `platform`: `The saved key will be removed.`(sidebar 편집에서만, detail은 키를 재전송하므로 표시 안 함) |
| 키 이월 가드로 저장 차단 오류(제안 c, k) | 목적지 `own_required`의 경고 `Replace the API key before saving.`를 오류로 표시하고 저장을 막는다(3.6.5). 저장 차단 오류는 목적지가 `own_required`일 때만 생긴다. 나머지 두 문구는 정보성 경고이고 저장을 막지 않는다. sidebar 편집에서 경고가 필요한 상태(가드 적용)에서는 키 블록이 `<details>` 밖에 렌더되어 경고가 항상 보인다(3.6.4, 제안 u) |
| sidebar 재조회 실패(3.6.6) | `Could not load the AI. Reload and try again.` |

**라벨 대소문자 규칙**: 입력 라벨은 항상 `OpenRouter Model ID`, `OpenRouter API Key`로 쓴다(단어 첫 글자 대문자, 본문 문장의 `model ID`, `API key`와 구분). 아래 표와 3.6.4 이후 모든 문구가 이 표기를 따른다.

**키 입력 라벨 통일(현황 확인 결과 4곳이 모두 다르다)**: `ais_create.js:577` `Engine Key`(optional 표기 없음), `ais_detail.js:1300` `Engine Key`, sidebar 생성 `:968` `Engine Key (optional)`, sidebar 편집 `:1347` `API Key (optional)`. 4곳을 키 방식별로 아래 하나로 통일한다.

| 키 방식 | 라벨 | 플레이스홀더 | 캡션 |
|---|---|---|---|
| `platform` | `Engine Key` | 현행 유지 | `API key not required for this model.` (입력 disabled) |
| `own_or_default` | `Engine Key (optional)` | 현행 유지 | `Leave empty to use the platform default key.` |
| `own_required` | `OpenRouter API Key *` | `Your OpenRouter API key` | 안내 1~3 |

현행 테스트가 이 라벨 문자열을 단정하지 않음을 grep으로 확인했다(`API Key (optional)`, `Engine Key` 라벨 단정 없음. `ais_create.test.js:282-288`은 플레이스홀더 `/ai engine api key/i`를 쓰므로 `platform`, `own_or_default`의 플레이스홀더는 바꾸지 않는다).

#### 3.6.4 키 입력, 검증, 서버 오류 처리 (4개 폼)

- 4개 폼(`ais_create.js:50-51,201,577-585`, `ais_detail.js:86-87,422,541,566,1300-1309`, `sidebar.js` 생성 `:387,527,968-970`, 편집 `:581,807-808,1347-1366`)의 `engineKeyManaged`(불리언)를 `keyMode`(= `keyModeOf(engineModel)`, 3값)로 일반화한다:
  - `platform`: 현재와 같음(입력 disabled, 키 미전송 또는 저장값 그대로 재전송).
  - `own_or_default`: 현재와 같음 + 라벨/캡션만 3.6.3대로.
  - `own_required`: 라벨 `OpenRouter API Key *`, 안내 3개 표시.
- **클라이언트 검증(저장 시점, 각 폼의 저장 핸들러 맨 앞)**:
  0. **trim(CPO 제안 j)**: 저장 직전에 `engineModel`이 Custom 접두로 시작하면 양끝 공백을 제거한 값을 검증과 전송에 쓴다(`const model = engineModel.trim()`). 입력 중에는 건드리지 않고 서버는 trim하지 않는다. 키 trim은 폼마다 다르다(코드 확인): sidebar만 trim한 값을 전송한다(생성 `createEngineKey.trim() || undefined` `sidebar.js:527`, 편집 `engineKey.trim()` `:808`). `ais_create.js:201`과 `ais_detail.js:422`는 `ref_engine_key.current.value` 원문을 전송한다. Go(`ValidateEngine`, `resolveSessionLLM`)와 Python(`.strip()`)이 사용 시점에 trim하므로 원문 전송도 안전하며, 전송값 trim 통일은 이번 범위가 아니다. UI가 trim하는 것은 아래 검증의 유효 키 판정뿐이다.
  1. `customIdMissing(engineModel)`이면 `Enter the OpenRouter model ID.`를 모델 오류(`modelError`/`error`)로 표시하고 저장을 막는다. 형식 규칙은 복제하지 않는다(서버 소유, `2026-10-05-openrouter-llm-routing-design.md` 3.7의 "서버 규칙 비복제" 원칙의 예외는 접두 문자열 하나뿐이고 그 값도 서버가 내려준다).
  2. `keyMode === 'own_required'`이고 유효 키가 비면 `An API key is required for custom OpenRouter models.`를 키 입력란 바로 아래(`text-destructive`)에 표시하고 저장을 막는다. **유효 키의 정의는 저장 유형에 따라 둘로 갈린다(CPO 제안 l).** (가) 키 이월 가드가 적용되지 않는 저장(저장 키가 없거나 키 방식 그룹이 같음): create는 `trim()`한 입력값, detail은 `ref_engine_key.current.value.trim()`, sidebar 생성은 `createEngineKey.trim()`, sidebar 편집은 `engineKeyChanged && engineKey.trim() ? engineKey.trim() : savedEngineKey`(3.6.6, 같은 그룹이면 저장된 키 유지가 허용되므로 폴백 허용). (나) 가드가 적용되는 저장(3.6.5): **새로 입력한 값만 유효하며 `savedEngineKey`, `detailData.engine_key` 폴백은 없다**(detail은 `ref_engine_key.current.value.trim() !== ''`일 때, sidebar는 `engineKeyChanged && engineKey.trim() !== ''`일 때. 입력란에 포커스만 한 경우 포함 입력이 아니다. 가드가 켜지는 전이에서 입력이 비워지므로(제안 p) 값이 있다는 것 자체가 새 입력이고 저장 키와의 비교는 하지 않는다, 제안 t). 이전 초안은 sidebar 편집의 유효 키 식에 `savedEngineKey` 폴백이 있어 포커스만으로 가드를 통과한 뒤 폴백으로 검증 2도 통과했다(`sidebar.js:1356-1359`의 `onFocus`가 `engineKeyChanged=true`로 만듦). (나)가 이 경로를 닫는다.
  3. 가드가 적용되는 저장이면 3.6.5의 조건(제안 c, k, l)을 적용한다.
- **서버 오류 분기(CPO 제안 e)**: UI의 오류 객체(`provider.js:146-148`)에는 `reason` 속성이 없고 `message`가 `Request failed (400): ... Details: <응답 본문>` 문자열이며 `status`만 별도 속성이다(직접 확인). `provider.js`는 수정하지 않는다. `engineErrorOf(err)`는 `err.message`에 `ENGINE_KEY_REQUIRED`가 포함되면 키 오류(고정 문구 `An API key is required for custom OpenRouter models.`), `INVALID_ENGINE_MODEL`이 포함되면 모델 오류로 분류한다. 모델 오류의 표시 문구는 `Details:` 뒤 본문을 `JSON.parse`해 `error.message`를 쓰고(서버가 입력값을 되돌려 보내지 않으므로 안전, 3.1.3), 파싱이 실패하면 `The OpenRouter model ID is not valid.`로 폴백한다. 그 외 오류는 현행 처리를 유지한다. 현재 `ais_create.js:248-252`, `ais_detail.js:454-457`의 catch는 오류를 버리고 고정 문구만 보여 주므로 이 분기를 위해 오류 객체를 `engineErrorOf`에 넘기도록 바꾼다. sidebar 편집은 `setSaveError(err.message)`(`sidebar.js:834`)로 `Request failed...Details: {json}` 전체를 보여 준다. 키/모델 오류는 `engineErrorOf`로 분류해 인라인에 표시하고, **같은 정제 문구를 기존 배너(`saveError`)에도 싣는다**(아래 이중 표시). 분류되지 않은 그 외 오류만 현행대로 `err.message` 전체를 `saveError`에 쓴다.
- **sidebar 키 블록 위치: `own_required` 또는 경고 필요일 때 `<details>` 밖 조건부 렌더(CPO 제안 m, u)**: sidebar 생성(`:961-972`)과 편집(`:1340-1371`)의 키 입력은 접힌 `<details>`(요약 `Advanced`) 안에 있다. 두 `<details>` 안에는 키 블록 하나(`div.space-y-3` 하나)만 있음을 직접 확인했다. `keyMode === 'own_required'`이거나 **경고 필요**(sidebar 편집에서 3.6.5 가드가 적용되는 상태, 제안 u)이면 키 블록(현재 키 방식의 라벨, `Input`, 캡션 또는 안내 1~3, 3.6.5 경고, 키 오류 문구)을 `<details>` 밖, 같은 Engine 탭 본문의 `ModelPicker`(Custom ID 입력란 포함) 아래에 바로 렌더하고 `<details>`와 `Advanced` 요약은 렌더하지 않는다. 그 외(`own_required`도 아니고 경고도 필요 없는 상태)는 현행 그대로 접힌 `<details>` 안에서 렌더한다. 이유: 경고는 입력란 아래에 표시되므로(3.6.5) 입력란이 접힌 `<details>` 안에 있으면 보이지 않는다. 조건식은 하나(`const keyOutside = keyMode === 'own_required' || guardApplies`)이며 sidebar 생성에는 가드가 없어 `own_required`만 해당한다. 구현은 같은 파일의 JSX 지역 변수(`keyBlock`) 하나로 키 블록을 만들어 두 위치 중 한 곳에만 넣는 방식이며 새 컴포넌트나 새 파일은 없다. 이전 초안의 `forceOpen`, `onToggle` 재오픈, `open={forceOpen || !!keyError || undefined}`, 요약 문구 `Advanced (API key required)`와 그 테스트 4개는 모두 제거한다(`<details open>`의 React 속성 제어는 jsdom 동작을 실증하지 못했고 더 단순한 대안이 있다). 영향과 근거:
  - 레이아웃: `own_required`일 때만 Engine 탭이 안내 3줄과 입력란 높이만큼 길어지고 `Advanced` 요약 줄이 사라진다(경고 필요 상태에서도 같다). 조건이 풀리면(저장 그룹과 접두로 돌아오고 `own_required`도 아님) 블록이 접힌 `<details>` 안으로 돌아간다. 키 값은 `createEngineKey`, `engineKey` 상태에 있으므로 언마운트로 사라지지 않는다(create의 비움은 3.6.5 전이 규칙만 따른다).
  - 오류 가시성: 키 입력란 오류는 목적지가 `own_required`일 때만 생기고 경고는 가드 적용 때만 생기며, 두 경우 모두 키 블록이 `<details>` 밖에 있으므로(위 조건식) 오류와 경고가 접힌 `<details>` 안에 가려지는 경우가 없다. 따라서 오류 때문에 `<details>`를 여는 장치가 필요 없다.
  - 테스트: `own_required`에서 키 라벨 `OpenRouter API Key *`가 `<details>` 밖에 있고 `Advanced` 요약이 없음, 가드가 적용되는 sidebar 편집에서는 목적지가 `own_or_default`나 `platform`이어도 키 블록과 경고가 `<details>` 밖에 보임, 가드가 없는 상태에서 다른 키 방식이면 키 입력이 `Advanced` 안(접힘)으로 돌아감(5.2). jsdom에서 `details` 속성 동작을 단정하지 않는다. create/detail 페이지는 키 입력란이 항상 보이는 카드라 변경 없음.
- **sidebar 오류 이중 표시(CPO 제안 h, 코드 확인)**: 키 입력란은 Engine 탭에 있어(`own_required`는 탭 본문, 그 외는 접힌 Advanced, 위 제안 m), 사용자가 다른 탭(General, Voice, Tools)에서 저장 버튼을 누르면 인라인 오류가 보이지 않는다. 기존 배너는 탭 밖에 있어 항상 보인다: 생성은 `createError` 배너(`sidebar.js:905-910`, 생성 버튼 `:892-899`은 `disabled={createLoading || modelsStatus !== 'ready'}`라 변경 없음), 편집은 `saveStatus === 'error'`일 때 `saveError`를 보여 주는 배너(`:1247-1252`)다. 따라서 sidebar의 **모든 클라이언트 검증 실패**(모델 ID 비어 있음, 키 필수, 3.6.5의 키 재입력 요구)와 **키/모델 서버 오류**는 인라인 문구를 쓰는 동시에 같은 문구를 `setCreateError(msg)`(생성) 또는 `setSaveError(msg); setSaveStatus('error')`(편집)로 배너에도 싣는다. 3.6.6의 `keyLoad === 'error'` 문구(`Could not load the AI. Reload and try again.`)도 같은 방식으로 인라인과 배너에 함께 싣는다. 단 이 문구가 실제로 보이는 것은 **캐시 히트 후 재조회가 실패한 경우뿐**이다(수정회차 7, 코드 확인). 캐시 미스에서는 기존 `.catch(() => setAiData(null))`(`sidebar.js:643`)이 폼 자체를 숨겨(`!hasAI || !aiData` 분기, `:1079`) 배너와 인라인 문구를 렌더할 곳이 없고 AI 선택 화면이 보인다. 이 기존 동작은 바꾸지 않는다. 그리고 `keyLoad === 'loading'`의 저장 버튼 비활성은 생성/편집 저장 버튼에 그대로 적용한다(생성에는 재조회가 없으므로 해당 없음). 편집 폼의 `[aiData]` 초기화 effect가 `setSaveError('')`를 호출하므로(`:646-717`) 배너는 AI를 바꿀 때 지워진다. create/detail 페이지는 키 입력란이 카드로 항상 보이므로 이중 표시가 필요 없다.
- 키 입력란은 **기존 입력 방식(`type="text"`, sidebar는 `Input` 기본)을 유지한다.** 이전 문서의 "마스킹 입력" 표현은 현재 코드와 맞지 않아 정정한다(제안 d, 3.6.5). 값을 로그, 토스트, 오류 메시지에 넣지 않는 것은 그대로다.

#### 3.6.5 키 이월 UX (Q8)과 키 방식 전환 보완 (제안 c, d)

- **create (ref 기반 비제어 입력, 전이에서만 비움)**: create의 키 입력은 `ref_engine_key`로 읽는 비제어 입력이다(`ais_create.js:89,580`). 모델 선택으로 `keyMode`가 바뀔 때만 입력을 비우려면 이전 값을 기억해야 한다. `prevCreateKeyModeRef = useRef(keyMode)`(편집 화면의 `prevGuardRef`와 달리 create 전용이며 가드가 없다)를 두고 `engineModel` 변경에 반응하는 effect(또는 `onEngineModelChange` 핸들러)에서 `prevCreateKeyModeRef.current !== keyMode`일 때만 `ref_engine_key.current.value = ""`로 비우고 `prevCreateKeyModeRef.current = keyMode`로 갱신한다. 같은 키 방식 안의 모델 변경(예: OpenAI에서 Gemini, Custom ID 수정 중 타이핑)에서는 비우지 않는다. 첫 렌더에서는 비교 대상이 같으므로 비우지 않는다. sidebar 생성은 `createEngineKey` 상태이므로 같은 전이 조건에서 `setCreateEngineKey('')`를 호출하되, **sidebar 생성에는 모델 변경 시 키를 비우는 기존 지점이 없다**(직접 확인: `createModel`은 `ModelPicker`의 `onChange={setCreateModel}`로 바뀌고 `sidebar.js:945`, 키 관련 코드는 `:387`의 `createKeyManaged` 계산, `:527`의 저장 시 body, `:970`의 캡션뿐이다). 따라서 `prevCreateKeyModeRef`와 `useEffect(..., [createKeyMode])`를 **새로 추가**해 전이에서만 `setCreateEngineKey('')`를 호출한다(`prevCreateKeyModeRef`와 같은 조건).
  - **create 화면의 같은 그룹 안 벤더 변경(수정회차 7, 제안 s 비적용)**: create에는 저장 키가 없어 가드가 없고 입력란이 화면에 보이므로, 같은 그룹 안에서 벤더를 바꿔도(OpenAI에서 Gemini) 사용자가 타이핑한 키는 기존 동작대로 유지한다. 그룹이 바뀔 때만 비운다. 편집 화면도 저장 키가 없을 때(`guardApplies`가 항상 false)는 같은 그룹 안 벤더 변경에서 타이핑한 키를 유지해 create와 동일하다. 편집에서 같은 그룹 안 벤더 변경이 입력을 비우는 것은 저장 키가 있을 때(가드 적용)뿐이다(제안 w).
- detail/sidebar 편집: 아래 가드 조건(저장된 키가 비어 있지 않고 (b)의 그룹 또는 서비스 접두가 다름)이 참이면 키 입력 아래에 3.6.3의 목적지별 경고(제안 r)를 표시한다. sidebar 편집은 이 상태에서 키 블록을 `<details>` 밖에 렌더해 경고가 보이게 한다(3.6.4, 제안 u).
- **편집 화면의 전이 시 입력 비움(CPO 제안 p, 수정회차 5, 수정회차 7에서 트리거 정정, 수정회차 8에서 그룹 변경 추가)**: 이전 초안은 "키를 자동으로 지우지 않는다"고 했고 그 근거로 "저장된 키 값은 화면에 없거나 마스킹"이라고 썼다. 코드를 다시 읽으니 **이 문장은 detail에 틀리다**. detail은 `setEngineKey(detailData.engine_key)`(`ais_detail.js:203`)로 저장 키를 상태에 넣고 입력란 `defaultValue`(`:1305`, `type="text"`)에 평문으로 채운다. 마스킹은 sidebar 편집(`sidebar.js:1350`, 점 표시)에만 있다.
  - **수정회차 6까지의 결함(리뷰 6회차 R11-M1, R12-M1, 코드로 재확인)**: 비움 트리거가 `keyMode` 전이(`prevKeyModeRef`)에만 반응해, 같은 그룹 안에서 서비스 접두만 바뀌는 경우(예: `openai.`에서 `gemini.`, 둘 다 `own_or_default`)에는 가드 (b)(서비스 접두, 제안 s)는 켜지는데 비움은 일어나지 않았다. detail은 미리 채워진 이전 키 K가 `ref_engine_key.current.value.trim() !== ''`로 새 입력처럼 인정되어(`:203,1305`, 저장 시 `:422`가 입력란 값을 그대로 보냄) 다른 벤더로 전송되고, sidebar는 이전 벤더용으로 타이핑해 `engineKeyChanged=true`로 남은 키가 같은 경로로 전송됐다.
  - **수정회차 7 정정의 회귀와 수정회차 8 보완(리뷰 7회차 R13-M1, 코드로 재확인)**: 수정회차 7은 비움 조건을 `guardApplies`의 false에서 true 하나로만 두었다. 실제 코드를 읽으면 두 경로가 열려 있다. (A) 저장 키가 없는 AI에서 `guardApplies`는 항상 false이므로, OpenAI 모델 맥락에서 키 T를 타이핑하고 Custom으로 바꿔도 입력이 유지된다. detail 입력란은 비제어(`ais_detail.js:1305`)이고 저장 시 입력란 값을 그대로 보내며(`:422`), sidebar는 `engineKeyChanged && engineKey.trim()`이면 그대로 보낸다(`sidebar.js:807-808`). T가 openrouter.ai로 전송된다. (B) 가드가 true인 채 그룹이 바뀔 때(저장 (OpenAI, K)에서 Gemini로 바꾼 뒤 Gemini 키 G를 타이핑하고 Custom으로, 또는 저장 키 없이 Custom에서 OpenAI로) 타이핑한 키가 다른 벤더로 전송된다. create는 `prevCreateKeyModeRef`로 그룹 전이에서 비우므로 편집이 create보다 약했다.
  - **해결: 트리거를 '그룹 변경 또는 `guardApplies`의 false에서 true 전이'로 한다(제안 w, 새 규칙 없음, `prevKeyModeRef`는 여전히 제거하고 `prevGuardRef` 한 ref 객체로 통합)**: 편집 화면(detail, sidebar 편집)은 `prevGuardRef = useRef(null)` 하나에 `{ keyMode, guardApplies }` 객체를 담는다. `guardApplies`는 아래 가드 (a)와 (b)가 모두 참인 렌더 시점의 불리언이다(`keyModeOf`, `servicePrefixOf` 계산). 전이 effect(의존성 `keyMode`, `guardApplies`, 카탈로그 상태, sidebar는 `keyLoad`와 `ai_id`)는 다음처럼 동작한다.
    - **카탈로그 `status === 'ready'`(sidebar는 추가로 `keyLoad === 'ready'`)일 때만 실행한다.** 카탈로그 도착 전에는 `keyModeOf`가 `own_or_default`로 폴백해(3.6.1) `guardApplies`가 일시적으로 틀린 값일 수 있고, 저장 모델 기준선도 없을 수 있기 때문이다. 이 조건 전에는 `prevGuardRef`도 갱신하지 않는다.
    - **detail effect의 위치와 가드(R14 minor 1)**: detail의 이 effect는 `isLoading` 조기 반환(`ais_detail.js:598`) 앞에 둔다(훅 순서 규칙). 실행 조건은 위 카탈로그 `ready`에 더해 **`detailData`(`fetchedDetail`) 로드 완료**다. 비움 시 `ref_engine_key.current`가 `null`일 수 있으므로(입력란 미마운트) `ref_engine_key.current &&` 가드를 두고 `null`이면 비우기만 건너뛰며 `prevGuardRef` 갱신은 한다.
    - **첫 실행(`prevGuardRef.current === null`)은 저장 모델 기준선과 비교한다(수정회차 8, 로딩 중 타이핑 후 전환 닫기)**: 비교 대상 이전 값을 `{ keyMode: keyModeOf(저장 engine_model), guardApplies: false }`로 두고 아래 규칙을 똑같이 적용한다. 정상 로드에서는 폼 모델이 저장 모델과 같아 아무것도 비우지 않는다. 카탈로그나 `keyLoad`가 ready 되기 전에 사용자가 입력란에 타이핑하고 모델을 다른 그룹으로 바꿨다면 ready 후 첫 실행에서 이미 그룹이 다르므로 비운다(로딩 중에는 Save가 막혀 저장은 불가능하고, 이 조건은 ready 직후 첫 저장 시도에서 타이핑한 키가 새어 나가지 않게 한다). 저장 모델 기준선은 detail이 `detailData.engine_model`, sidebar가 `savedEngineModel`(3.6.6)이다. sidebar는 `ai_id`가 바뀌거나 로드 effect가 다시 돌 때 `null`로 되돌린다(3.6.6 항목 2의 `keySeqRef`를 올리는 같은 지점).
    - **비움 조건**: (1) 이전 `keyMode`와 현재 `keyMode`가 다르거나(그룹 변경), (2) `guardApplies`가 false에서 true로 바뀌었고, **그리고 이번 전이가 아래 true에서 false 복원이 아닐 때** 입력을 비운다. detail은 `ref_engine_key.current.value = ''`, sidebar는 `setEngineKey('')`와 `setEngineKeyChanged(false)`. 이전 모드에서 새로 타이핑한 키, detail의 미리 채워진 저장 키가 모두 사라진다.
    - **true에서 false**: 입력을 저장값 상태로 되돌린다. detail은 `ref_engine_key.current.value = detailData.engine_key || ''`, sidebar는 `engineKeyChanged=false`, `engineKey=''`(마스킹 표시). 복귀 시 저장 키가 유지된다. 이 복원은 그룹 변경 비움보다 우선한다(저장 그룹으로 돌아오는 전이이므로 저장 키 상태가 정답).
    - **true인 채 변화 없음**: 가드가 켜진 채 **그룹이 같고** 모델만 더 바뀌면(Gemini에서 Grok, Custom ID 타이핑) 입력은 이미 비어 있으므로 비우지 않는다. 가드가 켜진 뒤 사용자가 새로 입력한 키는 그대로 유지되며 그것이 유효 키다. **가드가 true인 채 그룹이 바뀌면(Gemini에서 Custom 등) 비운다(제안 w, R13-M1 (B))**: 그 사이 타이핑한 Gemini 키 G는 Custom 전환 뒤 openrouter.ai로 가지 않는다. 이 입력란은 detail에서 항상 보이고 sidebar에서는 가드가 켜지면 `<details>` 밖에 렌더되므로(3.6.4) 같은 그룹 안의 벤더 변경은 사용자가 보면서 입력한 값이다.
    - **false인 채 변화 없음**: **그룹이 같을 때** 같은 접두 안의 변경(`openai.gpt-5`에서 `openai.gpt-5.1`)과, 저장 키가 없을 때(`guardApplies`가 항상 false)의 같은 그룹 안 벤더 변경(OpenAI에서 Gemini)은 입력을 유지한다. 후자는 사용자가 화면에서 보고 타이핑한 값이고 저장 키 유출이 아니며(비울 저장 키가 없음) create와 같다. 저장 키가 없어도 **그룹이 바뀌면 비운다**(OpenAI에서 Custom, Custom에서 OpenAI, R13-M1 (A)).
  - **임계가 닫는 경로(detail과 sidebar 실제 코드로 검증)**: (1) 저장 (OpenAI, K)에서 detail이 Gemini로 변경: `guardApplies` false에서 true이므로 입력란이 `''`이고 `body.engine_key`는 새 입력이 없으면 `''`(`own_or_default` 목적지, 저장 키 미전송). (2) sidebar에서 OpenAI용으로 `dummy-typed-key`를 타이핑(`engineKeyChanged=true`, 가드는 false)한 뒤 Gemini로 변경: false에서 true이므로 `engineKeyChanged=false`, `engineKey=''`. (3) 저장 키 없음에서 OpenAI용으로 타이핑한 `dummy-typed-key`가 Custom으로 전환 시 그룹 변경으로 비워진다(detail은 `ref_engine_key.current.value`, sidebar는 `engineKeyChanged=false`, `engineKey=''`). Custom에서 OpenAI로도 같다. (4) 가드 true인 채 Gemini에서 타이핑한 키가 Custom 전환(그룹 변경)으로 비워진다. (5) 이전 문서의 `keyMode` 전이 경로(OpenAI에서 Custom, platform에서 Custom 등)도 그룹이 달라 같은 전이에 포함된다. 수정회차 7의 "`guardApplies` 전이가 `keyMode` 전이의 상위 집합"이라는 서술은 저장 키가 없을 때와 가드가 true인 채 그룹이 바뀔 때 성립하지 않아 틀렸다. 수정회차 8의 트리거는 그룹 변경과 가드 전이의 합집합이다.
  - 가드 적용 중 sidebar의 키 입력란 표시는 `engineKeyChanged ? engineKey : (savedEngineKey && !guardApplies ? 점 마스킹 : '')`로 한다. 이전 키가 전송되지 않는 상태에서 점이 보여 키가 있는 것처럼 오해시키지 않기 위함이며 3.6.6 항목 5의 마스킹 조건에 `!guardApplies`가 붙는 것뿐이다.
  - 이로써 detail에서 platform 원본의 저장 키가 평문으로 남는 문제(platform에서 `own_or_default`로 바꿀 때 이전 키가 입력란에 보임)도 같이 사라진다.
- **키 이월 가드(CPO 제안 g를 제안 k, l로 대체, 수정회차 4)**: 수정 화면에서 다음 두 조건이 **모두** 참이면 가드가 적용된다.
  - (a) **저장된 키가 비어 있지 않다**(`trim()` 후). 출처는 detail이 `detailData.engine_key`, sidebar 편집이 `savedEngineKey`(3.6.6)다.
  - (b) **`keyModeOf(현재 선택)`이 `keyModeOf(저장된 engine_model)`과 다르거나, `servicePrefixOf(현재 engineModel)`이 `servicePrefixOf(저장된 engine_model)`과 다르다**(어느 방향이든, 수정회차 6, 제안 s). 접두 비교는 낡은 캐시로 같은 그룹 안의 다른 벤더가 저장 키와 함께 전송되는 경로(예: 캐시 `openai.*`, 서버 `gemini.*`와 키 K)와 사용자가 직접 벤더를 바꾸는 경우(OpenAI에서 Gemini)를 함께 닫는다. 같은 접두 안의 변경(Custom에서 다른 Custom ID, `openai.gpt-5`에서 `openai.gpt-5.1`)은 가드가 없다.
  - (c) **저장 모델의 기준선(수정회차 5)**: (b)의 "저장된 engine_model"은 detail이 `detailData.engine_model`(`fetchedDetail`, 페이지 로드 시 서버 응답), sidebar 편집이 **재조회 응답의 `savedEngineModel`**(3.6.6)이다. sidebar의 `aiData.engine_model`(캐시 값일 수 있음)은 기준선으로 쓰지 않는다. `keyLoad !== 'ready'`이면 기준선이 없으므로 Save가 비활성이다. **`detailData`와 `fetchedDetail`의 구분(수정회차 7, `ais_detail.js:56,160-167,200,266`)**: `fetchedDetail`은 페이지 로드 시 원본 서버 응답이고 폼 초기화 effect(`:200`, 의존성 `[fetchedDetail]`)만 읽는다. `detailData`는 `fetchedDetail`에 insight 활성 상태(`is_insight_active`)만 덧씌운 `useMemo` 결과이며 저장 body(`:422`), dirty 비교(`:541`), 가드 기준선과 복원 값은 `detailData`를 읽는다. 두 객체의 `engine_model`, `engine_key`는 같은 값이라 기준선은 어느 쪽이어도 같다.
  - **저장된 모델이 `platform`이어도 (a)가 참이면 적용한다.** 이전 초안은 원본이 `platform`이면 "저장 키가 사실상 없다"고 보고 면제했으나, 코드를 다시 읽으니 platform 원본에도 키가 남는 경로가 있고 그 키가 입력란에 채워진다.
- **코드 사실(수정회차 4, 직접 확인)**:
  - detail: `ais_detail.js:422`는 platform 모델에서 `engine_key: engineKeyManaged ? (detailData.engine_key || '') : ref_engine_key.current.value`로 **저장된 키를 그대로 재전송**한다. `:203`은 `setEngineKey(detailData.engine_key)`로 입력란 `defaultValue`(`:1305`)를 저장 키로 채우고, dirty 비교 `:541`은 `engineKeyManaged`이면 키 비교를 건너뛴다. 테스트 `views/ais/__tests__/ais_detail.test.js:435-448`("switching to a platform-managed model ... Save sends the STORED key unchanged", 단정 `:446`)이 이 재전송을 고정한다. 이 문서는 이 계약을 바꾸지 않는다. 따라서 OpenAI 키가 저장된 AI를 Claude(platform)로 detail에서 저장해도 키는 DB에 그대로 남는다. CLI/API로 만든 AI(platform 모델 + 키 있음)도 같은 상태다.
  - sidebar: platform 모델 저장은 `engine_key`를 보내지 않아(`sidebar.js:806-809`) PUT 전체 교체로 키가 비워지지만(`db.go:280`), 편집 진입 때 재조회한 `savedEngineKey`(3.6.6)는 DB에 남은 이전 키일 수 있다(위 detail/CLI 경로 포함). 이 값이 있으면 3.6.6 항목 3의 body 식이 이전 키를 그대로 보낸다.
  - 이전 초안의 우회 재현: OpenAI 키 K가 저장된 AI를 detail에서 Claude(platform)로 저장해 키를 유지한다. 이어서 Custom으로 변경하면 원본 그룹이 platform이라 가드가 면제되고, 입력란에 K가 채워져 있어 검증 2도 통과해 K가 openrouter.ai로 전송된다. sidebar도 `savedEngineKey`가 K이면 같은 경로였다. 새 기준에서는 (a) K가 비어 있지 않고 (b) `platform`과 `own_required`가 달라 가드가 적용되며, 새로 입력한 키가 없으면 저장되지 않는다.
- **유효 키와 목적지별 동작(제안 l)**: 가드가 적용될 때 "새로 입력함"은 다음이다. detail: `const typed = ref_engine_key.current.value`에 대해 `typed.trim() !== ''`(제안 t. 위 전이 비움 규칙으로 가드 적용 시 입력란은 비어 있으므로 값이 있다는 것이 곧 사용자의 새 입력이다. 저장 키와의 비교는 하지 않는다). sidebar 편집: `engineKeyChanged && engineKey.trim() !== ''`(제안 t. `onFocus`가 `engineKeyChanged=true`, `engineKey=''`로 만들므로 `sidebar.js:1356-1359`, 포커스만 한 경우는 입력이 아니다). `savedEngineKey`, `detailData.engine_key` 폴백은 없다.
  - 목적지 `own_required`: 새로 입력한 키가 있어야 저장되고 없으면 3.6.3의 경고 문구를 오류로 표시한다(3.6.4 검증 2의 (나)와 같은 조건). body.engine_key는 새 입력값.
  - 목적지 `own_or_default`: 새로 입력한 값이 비어 있어도 저장을 허용한다. 빈 값은 "기본 키 사용"의 정당한 의미이며(Q1b와 일치) **이전 키는 재전송하지 않는다**. body.engine_key는 새 입력값, 입력이 없으면 `''`이다(detail은 입력란이 비워져 있으므로 입력이 없으면 `''`을 보낸다. 현재 `ais_detail.js:422`는 입력란 값을 그대로 보내므로 이 분기가 새로 필요하다).
  - 목적지 `platform`: 키를 쓰지 않으므로 요구하지 않는다. 전송 규칙은 현행 유지(detail은 저장 키 재전송, sidebar는 미전송). 경고 `The saved key will be removed.`는 sidebar에서만 표시한다(제안 r). **detail은 입력란이 비어 보이나 저장 키는 서버에 유지된다**(현행 재전송 계약, 제안 v. 이 문서는 "키가 지워진다"고 쓰지 않는다). 이 키는 호출에 쓰이지 않으며, 이후 다른 그룹이나 다른 서비스로 바꿀 때 위 가드가 다시 걸린다.
  - 가드가 적용되지 않는 저장(저장 키 없음, 또는 같은 그룹과 같은 서비스 접두 안의 변경(Custom에서 다른 Custom ID, `openai.gpt-5`에서 `openai.gpt-5.1`))은 현행 규칙이다.
- **이 가드가 닫는 범위(정정)**: UI에서 "저장된 키가 비어 있지 않은 상태로 키 방식 그룹 또는 서비스 접두가 바뀌는 저장"은 모든 방향과 모든 원본 그룹(platform 포함)에서 새 키 입력(또는 `own_or_default` 목적지의 빈 값)이 필요하므로 이전 키가 다른 발급처로 전송되는 UI 경로는 닫힌다. 이전 문서의 "양방향 모두 닫힌다"는 platform 원본과 저장 키가 있는 경우를 포함하지 못했던 문장이라 이 조건 한정으로 고친다. **닫히지 않는 경로**: (i) CLI/API 직접 호출(서버는 최종 스냅샷만 보므로 판별 불가), (ii) platform 모델에 남은 키 자체의 삭제(detail 재전송 유지, 호출에 쓰이지 않음), (iii) 같은 서비스 접두 안에서 저장 키의 출처 벤더가 이미 달라진 경우(서버 스냅샷만으로 판별 불가). 같은 그룹 안의 벤더 변경과 사용자가 같은 키를 다시 입력하는 경우는 더 이상 잔존 경로가 아니다(전자는 제안 s로 닫히고 후자는 제안 t로 허용되는 정상 입력이다). 이번 수정회차까지 **닫힌 경로**: 낡은 캐시 모델 기준선 우회(R7-M1, 3.6.6 항목 7), 이전 모드와 이전 벤더용으로 새로 타이핑한 키, detail의 미리 채워진 저장 키의 승계(제안 p, 수정회차 7에서 `guardApplies` false에서 true 전이로 닫음), 같은 그룹 안의 다른 벤더로 저장 키가 전송되는 경로(낡은 캐시 포함, 제안 s). 이 해석(특히 `own_or_default` 목적지의 빈 값 허용과 platform 원본 포함)은 CPO 제안이며 대표님 이견이 있으면 조정한다.
- 같은 `own_or_default` 그룹 안에서 벤더가 바뀌는 경우(예: OpenAI에서 Gemini)는 서비스 접두가 달라 가드가 적용되고, 그 전이에서 입력란이 비워진다(제안 p, 위 전이 규칙). 목적지가 `own_or_default`라 새 키를 입력하거나 빈 값(기본 키 사용)으로 저장할 수 있고 저장 키와 이전 벤더용 키는 재전송되지 않는다.
- **platform 그룹 안의 서비스 접두 불일치(수정회차 7)**: 카탈로그의 platform 항목은 `anthropic.`, `meta.`, `deepseek.`, `qwen.`, `mistral.`로 서비스 접두가 서로 다르다(`catalog.go:78-86`). 그래서 저장 키가 있는 platform 원본에서 Claude를 Llama로 바꿔도 접두가 달라 가드가 적용되고(제안 s) 입력란이 비워진다. 목적지가 `platform`이므로 새 키는 요구하지 않으며 경고 `The saved key will be removed.`는 sidebar에서만 표시된다(제안 r). detail은 입력란이 비어 보이나 저장 키를 재전송한다(제안 v). 기능상 영향은 없으나 의도된 결과로 5.2 행렬에 행 하나로 고정한다.
- **키 입력 `type=password` 전환은 이번 범위에서 제외(제안 d)**: 현재 create/detail 키 입력란은 `type="text"`로 화면에 평문이 보인다(detail은 저장된 키를 `defaultValue`로 채움, `ais_detail.js:1305`). 이 동작을 유지하고 별도 이슈로 기록한다(8.3). BYOK 키가 같은 평문 입력란을 쓴다는 점은 알려진 위험이다.

#### 3.6.6 sidebar 편집 재조회 (Q9, 모든 AI 대상)

코드 사실 재확인(`teamgraph/sidebar.js`): 캐시 저장 시 `engine_key`를 제거하고(`:488,636-639`), 캐시 히트 경로는 키 없는 객체를 `setAiData`하며(`:620-624`), 편집 저장은 키가 안 바뀌면 `engine_key`를 body에서 생략한다(`:807-808`). 서버는 PUT의 부재 `engine_key`를 빈 문자열로 받아 전체 교체한다(`aihandler/db.go:280`, `buildUpdateFields`).

**폼 상태 초기화 effect (직접 읽고 확인)**: `useEffect(..., [aiData])`(`sidebar.js:646-717`)가 `aiData`가 바뀔 때마다 이름, 모델, 키, 파라미터, TTS/STT, 도구, RAG, MCP 목록 등 **모든 폼 상태를 `aiData`로 덮어쓰고** `setSaveError('')`까지 한다. 따라서 재조회 응답을 `setAiData`로 반영하면, 응답이 늦게 도착했을 때 사용자의 미저장 편집이 전부 사라진다.

설계:

1. 재조회 결과는 **`setAiData`를 호출하지 않는다.** 별도 상태 `savedEngineKey`(문자열), `savedEngineModel`(문자열, 가드 기준선, 제안 o)과 `keyLoad`(`'loading' | 'ready' | 'error'`)에만 반영한다. 캐시 미스 경로는 기존 코드가 이미 `setAiData(res)`를 하므로(`:632`) 그 호출은 유지하고(캐시 미스이므로 폼 초기화는 처음 한 번뿐) 같은 응답에서 세 상태를 채운다. 세 상태는 초기화 effect(`[aiData]`)의 의존성이 아니므로 응답이 언제 오든 다른 폼 필드는 초기화되지 않는다. 캐시(`localStorage`)에는 기존대로 키를 제외하고 저장한다(`:636-639` 유지, 재조회 응답은 캐시에 쓰지 않는다).
2. **새 effect를 만들지 않고 기존 로드 effect(`sidebar.js:618-644`, 의존성 `[ai_id, hasAI]`)를 확장한다**(수정회차 5, 중복 GET 제거). 기존 effect는 캐시 히트면 `setAiData`하고 끝나고, 캐시 미스일 때만 `ProviderGet(`ais/${ai_id}`)`를 호출하며(`:630`) 그 응답에 `engine_key`와 `engine_model`이 이미 있다. 따라서: (가) **캐시 미스**: 이 기존 응답으로 `savedEngineKey`, `savedEngineModel`을 채우고 `keyLoad='ready'`로 하며 추가 GET은 하지 않는다(실패 시 기존 `.catch`에서 `keyLoad='error'`, 기존 `.catch`가 `setAiData(null)`도 호출하므로 폼이 숨겨짐). **`ProviderGet` 응답이 falsy이면 에러로 처리한다**(수정회차 7. 기존 `if (res)`는 falsy 응답을 조용히 넘기므로 `keyLoad='error'`를 명시적으로 세운다. 캐시 미스에서는 `aiData`가 null인 채라 폼이 숨겨지고, 캐시 히트에서는 폼은 보이되 Save가 비활성이며 문구가 표시된다). (나) **캐시 히트**일 때만 `setAiData(stored[ai_id])` 뒤에 `ProviderGet(`ais/${ai_id}`)`를 호출해 같은 두 상태를 채운다(캐시에는 쓰지 않고 `setAiData`도 하지 않음). 시작 시 `keyLoad='loading'`, `savedEngineKey=''`, `savedEngineModel=''`. (가), (나) 모두 요청마다 증가하는 시퀀스 `keySeqRef`를 잡아 두고, 응답 시 **`seq !== keySeqRef.current`이면 무시**한다(노드를 바꿔 id가 달라졌거나 언마운트된 경우). effect cleanup에서도 시퀀스를 올린다. 이 effect의 의존성이 `aiData` 전체가 아니라 `[ai_id, hasAI]`이므로 저장 후 `setAiData`로 같은 id의 객체가 교체되어도 재조회가 반복되지 않는다.
3. **저장 시 `body.engine_key` (platform 모델은 현행 유지, 수정회차 3에서 정정)**: 키 방식이 `platform`이 아니면 `engineKeyChanged && engineKey.trim() ? engineKey.trim() : savedEngineKey`를 **항상** 넣는다(**가드가 적용되는 저장**(기준선은 `savedEngineModel`, 3.6.5 (c))이면 3.6.5에 따라 `engineKeyChanged && engineKey.trim()`의 새 입력값만 넣고, `own_or_default` 목적지에서 비면 `''`을 넣는다. `savedEngineKey`는 재전송하지 않는다. 즉 body의 키 식은 "가드 적용이면 새 입력값 또는 `''`, 아니면 `engineKeyChanged && engineKey.trim() ? engineKey.trim() : savedEngineKey`"다). **`platform`이면 현행대로 `engine_key`를 보내지 않는다**(`sidebar.js:806-809`의 "플랫폼 모델은 키를 다시 보내지 않는다" 계약 유지). 이전 초안은 platform도 재조회 값을 항상 보내도록 했는데, 이는 위 주석 계약을 의도적으로 뒤집고 기존 테스트(`views/teamgraph/__tests__/sidebar_engine_model.test.js:206`의 `expect(body).not.toHaveProperty('engine_key')`)를 깨뜨린다. 코드로 영향을 재평가한 결과 platform에서 재전송이 필요하지 않다: (a) 호출 시 platform 모델은 저장된 키를 쓰지 않는다(`Resolved.BlankKey`, pipecat `resolveSessionLLM`이 키를 비움, 3.3), (b) 서버 검증은 `openrouter.` 접두 모델에만 키를 요구한다(3.1.2), (c) 어느 쪽이든 서버 상태는 안전하다. 유일한 차이는 서버가 PUT의 부재 `engine_key`를 빈 문자열로 저장하므로(`aihandler/db.go:280`) platform 모델로 저장하면 이전 저장 키가 비워진다는 점이며, 이는 현행 동작이고 platform 모델이 쓰지 않는 값이라 허용한다(다시 키 모델로 바꿀 때는 3.6.5 규칙에 따라 새로 입력). 키 입력란에 포커스만 하고 아무것도 입력하지 않은 경우(`onFocus`가 `engineKeyChanged=true`로 만듦, `:1356-1359`)에도 가드가 적용되지 않으면 저장된 키가 유지된다. 가드가 적용되면 포커스만으로는 새 입력이 아니다(3.6.5). 키를 의도적으로 비우는 동작은 현재도 sidebar에서 불가능하며 이번에도 지원하지 않는다.
4. `keyLoad !== 'ready'`이면 저장 버튼을 비활성화한다(재조회 전이거나 실패하면 가드 기준선 `savedEngineModel`도 없으므로 같은 조건이다, 제안 o). `error`일 때는 `Could not load the AI. Reload and try again.`를 인라인과 `saveError` 배너에 함께 표시한다(3.6.4 이중 표시, 키 손실 방지를 위해 키 없이 저장하는 fail-open 금지). 이 문구가 실제로 화면에 보이는 것은 캐시 히트 재조회 실패뿐이다. 캐시 미스 재조회 실패는 기존 `.catch`의 `setAiData(null)`로 폼이 숨겨진다(`sidebar.js:643,1079`). `loading`은 문구 없이 비활성만 한다.
5. 마스킹 표시(`:1350`)는 `aiData.engine_key` 대신 `savedEngineKey`가 있고 가드가 적용되지 않을 때(`!guardApplies`, 3.6.5) 점으로 표시한다(캐시 경유 시 빈 칸이던 문제도 해소).
6. 저장 성공 후에는 `setSavedEngineKey(...)`와 `setSavedEngineModel(전송한 engine_model)`로 메모리를 갱신한다(`:820-824`의 `setAiData({..., engine_key: aiData.engine_key})`는 기존대로 두되 키의 출처는 `savedEngineKey`다). `savedEngineKey`에 넣는 값은 body에 `engine_key`를 보낸 경우 그 값, **보내지 않은 경우(sidebar platform 저장, 항목 3)는 `''`**다(서버가 PUT을 전체 교체하며 부재 `engine_key`를 빈 문자열로 저장하므로 `aihandler/db.go:280`, 저장 후 서버 상태와 일치). 이전 초안은 이 경우의 값이 정의되지 않았다.

7. **가드 기준선과 낡은 캐시(R7-M1, 수정회차 5, 코드로 재확인)**: 스냅샷 불일치 경로가 실제로 존재한다. sidebar 편집은 캐시 히트 시 `setAiData(stored[ai_id])`로 캐시 객체를 그대로 쓰고(`sidebar.js:623-624`) 이 객체의 `engine_model`이 폼의 모델 초기값과 가드 기준선이 된다. 캐시는 목록 첫 조회(`initAIItems`, `:464-490`, 캐시가 비어 있을 때만), 캐시 미스 단건 조회(`:630-640`), 생성(`:548-551`), sidebar 자체 저장(`:819-823`)에서만 갱신된다. `ais_detail.js`는 캐시를 읽거나 쓰지 않고(grep으로 확인, `writeResourceCache*` 호출 없음) 저장 후 `navigate(0)`만 한다(`:452`). 그래서 우회가 성립한다: sidebar 캐시에 `engine_model=openrouter.a/b`(Custom)와 키 X가 있던 AI를 detail에서 OpenAI + 키 K로 저장하면 서버는 (OpenAI, K)이고 캐시는 (Custom)인 채 남는다. 이후 sidebar를 열면 재조회로 `savedEngineKey=K`가 오지만 기준선이 캐시 모델(Custom)이면 폼 선택(Custom)과 같은 그룹이라 가드가 적용되지 않고, 이름만 바꿔 저장해도 body는 `engineModel=openrouter.a/b`와 `savedEngineKey` K를 함께 보내 **OpenAI 키 K가 openrouter.ai로 전송된다**(실제 코드의 body 식은 항목 3). **새 기준선(제안 o)**: 가드의 저장 모델은 `savedEngineModel`(같은 재조회 응답의 `engine_model`)이므로 서버 (OpenAI, K)에 대해 현재 선택 Custom은 그룹이 달라(`own_or_default` 대 `own_required`) 가드가 적용되고 새 키 없이는 저장되지 않는다. 키와 모델이 같은 응답에서 오므로 어긋날 수 없다. 이때 폼의 모델 초기값은 여전히 낡은 캐시 값일 수 있어 이름만 변경해도 캐시 모델로 서버 모델을 덮어쓰는 기존 문제는 남는다(키 안전성과는 별개이며 가드가 키 전송을 막는다. 모델 표시를 재조회 값으로 갱신하는 것은 폼 초기화 effect를 건드려 미저장 편집을 지우므로 이번 범위가 아니다).

**의존성(8절 항목 3과 연결)**: 이 재조회는 `GET /ais/{id}` 응답이 `engine_key`를 돌려주는 현재 동작에 의존한다. 별도 이슈 "GET 응답의 `engine_key` 제거"를 진행하면 이 재조회, `ais_detail.js:203,422`의 재전송 흐름과 3.6.5의 가드 입력(`savedEngineKey`, `detailData.engine_key`)이 함께 깨진다. 그 이슈는 PUT에서 `engine_key` 부재를 "변경 없음"으로 해석하는 서버 의미 변경과 한 쌍으로만 착수할 수 있다.

#### 3.6.7 목업 수정 사항

목업은 수정이 필요하다(디자인 단계 산출물 정비는 이 문서의 후속 작업, 이번에 png는 수정하지 않았다).

- **전체**: 모든 한국어 문구를 3.6.3의 영어 문구로 교체(패널 제목, 그룹 헤더 부제 `· 키 불필요`/`· OpenAI / Gemini / Grok`, 배지, Custom 항목 라벨과 설명, ID 도움말, 안내 3개, 패널 3 캡션 포함).
- **패널 1 수정 필요**: (a) GPT-5, Grok 3 배지 `내 키`를 `Your key or default`로(Q1a), (b) Gemini 2.5 Flash의 별도 `내 키 또는 기본 키` 배지를 같은 `Your key or default`로 통일, (c) Claude Haiku 4.5와 Llama 3.3 70B 배지를 `No key needed`로, (d) Custom 배지 `내 키 필수`를 `Your key required`로, (e) 그룹 헤더에서 부제 제거.
- **패널 2**: 한국어 안내문 3개와 ID 도움말을 영어로 교체하는 것 외에 구조는 유지. 인라인 오류 문구는 이미 영어이며 서버 메시지와 동일해야 한다. ID 입력란이 모델 선택 트리거 바로 아래(`ModelPicker` 내부)에 놓이고 오류가 그 아래에 표시되는 배치를 따른다(3.6.2).
- **패널 3 수정 필요**: 분석서 4절의 코드 대조대로 "현재 동작" 재현이 아니다(실제 라벨은 `Engine Key`, `(optional)` 표기는 sidebar에만 존재, GPT-5 배지 `내 키`와 "비워 두면" 문구는 코드에 없음). 패널 제목을 "Target behavior of the key field by key mode"로 바꾸고 `platform`(disabled, `API key not required for this model.`)과 `own_or_default`(`Engine Key (optional)`, `Leave empty to use the platform default key.`) 두 상태를 보여준다.

### 3.7 ZDR 제공자 없음과 오류 처리 (Q5)

**기본 동작(fail-closed, 이 PR 범위)**

- 러너는 모든 Custom 요청에 `provider.zdr=true`, `data_collection=deny`, `require_parameters=true`를 싣는다(3.3).
- 해당 모델에 ZDR 제공자가 없으면 OpenRouter가 요청을 거절하고(분석서 R2의 BYOK 문서 근거, 실호출 미확인) 우리는 어떤 재시도, 옵션 제거, 모델 대체도 하지 않는다. 호출 실패가 정상 동작이다.
- 분류기는 확장하지 않는다. 응답 원문이 확인되기 전에는 `classifyPipelineError`(`pipelineerror.go:20-86`)를 바꾸지 않는다. 404 계열은 어느 티어에도 해당하지 않아 `unknown`이 되며, STT가 있는 음성 세션에서는 알림이 생성되지 않는다(`:96-114`). 즉 실호출 검증 전에는 고객에게 무음 통화 외 안내가 없을 수 있고, 이 한계를 문서(3.5)와 8절에 명시한다.
- 저장 시점에 ZDR 제공자 존재를 확인하지 않는다(비목표). `https://openrouter.ai/api/v1/endpoints/zdr`는 `verify_openrouter_catalog.py`가 쓰는 공개 목록이지만 저장 경로에 외부 네트워크 호출을 넣는 비용과 목록 신선도 문제가 크다.

**고객 안내 문구 후보 (확정 아님, 실호출 후 결정)**

| 상황 | 후보(영어) | 비고 |
|---|---|---|
| ZDR 제공자 없음으로 분류될 때 신규 카테고리를 추가하는 경우 | `The selected OpenRouter model has no provider that supports zero data retention. Choose a different model.` | 응답 원문이 안정적으로 구분될 때만 |
| 기존 `unknown` 유지 | `An AI service provider returned an error.`(`pipeline_error.go:38` 기존 문구) | 기본값 |
| 401/403 | 기존 authentication 문구(`pipeline_error.go:35`, "custom engine key" 확인 안내)로 BYOK 401에 적합 | 변경 없음 |
| 402 크레딧 없음 | 기존 rate_limited 문구(`pipelineerror.go:37`, 테스트 `pipelineerror_test.go:92`) | 변경 없음 |

### 3.8 분석서와 달라진 사실 (코드 재확인 결과)

최종 보고용 목록이며 이 문서의 해당 절에 반영했다.

1. `openrouter` 금지 가드는 4곳이며 4번째 `chatbot_engine_model_error_test.go:48`(`assertInvalidEngineModel`)을 분석서가 누락했다. 이 가드는 완화하지 않고 Custom 전용 메시지/헬퍼를 추가한다(3.1.3, 3.1.5).
2. `verify_openrouter_catalog.py`가 `catalog.go`를 정규식(`UpstreamSlug`)으로 읽는다. 분석서에 없던 의존으로, Custom 항목은 `UpstreamSlug`와 `RouteOpenRouter`를 쓰지 않아야 한다(3.1.1).
3. Python `test_init_pipeline.py:804,827,890`은 `_member_llm_type`(`llm_type==""` 거부, 필드 부재 시 `openrouter` 서비스 거부)를 검증하며, 이 가드를 유지하기로 하면 테스트 수정이 필요 없다. 분석서는 "허용/거부로 바뀜"이라 적었다(3.3, 5절). 실제로 갱신이 필요한 Python 테스트는 `test_run.py`의 `test_raw_openrouter_is_unsupported`(`:1027-1034`)의 점 형태 행(`openrouter.meta-llama/llama-3-70b`)뿐이다.
4. api-manager는 `amai.ModelInfo`로 역직렬화하므로(`servicehandler/ai.go:276-291`) 재배포하지 않으면 새 필드가 사라진다. 분석서는 "Go 쪽 변경은 `ModelInfo`뿐"이라고만 했고 배포 결합을 적지 않았다(3.2, 6절).
5. `ai-control`은 `aihandler.Create/Update`를 직접 호출하므로(`main.go:242,366`) 서버 검증이 곧 CLI 방어이며 CLI 사전 검증은 빠른 실패용이다(3.1.2). 분석서는 CLI를 별도 방어로 다뤘다.
6. `start.go:41`의 키 없는 사전 분류가 Custom에서 항상 실패하는 문제를 `classifySessionLLM` 분리로 해소하는 설계를 새로 정했다(분석서는 "설계 과제"로만 표기, 3.3).
7. 기존 INVALID_ENGINE_MODEL 메시지가 입력 모델 문자열을 `%q`로 되돌려 보내는 점을 새로 확인했다. Custom 입력에는 되돌리지 않는다(3.1.3).
8. 분석서의 나머지 코드 사실(분석서 13곳 로깅 지점, 수정회차 3에서 aicallhandler 3곳을 추가해 16곳, `v1_ais.go` `%v` 5곳, `pythonrunner.go:87` `omitempty`, SDK `OPENAI_API_KEY` 폴백, 테스트 라인 번호 `llmresolve_test.go:41,66,130,216`, `run_teamllmtype_test.go:60`, sidebar `:488,636-639,807-808`, `ais_create.js:201`, `ais_detail.js:422,541`, compose 주입)은 이번에 모두 재확인했고 일치한다.
9. `ai_struct_ai.rst`의 `engine_key` 설명은 `:54`, "never returned"는 `:77`, `platform_managed` 설명은 `:208`, 접두 규칙은 `:212`에 있다. 분석서의 `:180,185-214` 표기와 일부 다르다(오류표 `:180`은 이번에 재확인하지 않음).

10. (수정회차 2) 오류 경로 요청 본문 로깅이 별도 누출이다: `listenhandler/main.go:283-286`, `v1_ais.go:24,83,159,198,237`의 `"request": m`은 본문(`engine_key` 포함)을 필드로 싣고 모든 오류 로그에 출력된다. ai-manager는 `DebugLevel`을 무조건 설정한다(`config/main.go:329`). 필수 처리로 승격(3.4 B1).
11. (수정회차 2) 공용 `notifyhandler/publish.go:34`의 `"data": data`는 13곳 목록에 없던 누출이다. 호출처는 16개 서비스, 비밀값 필드 타입은 ai-manager와 conversation-manager뿐이다(3.4 B3).
12. (수정회차 2) `PublishEvent`(`publish.go:88`)가 `*ai.AI`를 그대로 `json.Marshal`해 이벤트 버스에 발행하고 timeline-manager가 전량 구독해 ClickHouse에 저장하는 경로가 코드상 있다(4절 잔존 위험).
13. (수정회차 2) `ModelPicker`가 sidebar에서 직접 렌더되므로(`:945,1302`) 모델 ID 입력란을 `AIEngineFields`에 두는 이전 설계는 sidebar 2곳이 빠진다. `ModelPicker` 내부로 이동(3.6.2). `AIEngineFields`의 `modelError` prop은 존재하지만 어떤 폼도 넘기지 않았다.
14. (수정회차 2) 키 입력 라벨 4곳이 모두 다르고(`Engine Key`, `Engine Key`, `Engine Key (optional)`, `API Key (optional)`), 키 입력란은 마스킹되지 않은 `type="text"`다(3.6.3, 3.6.5). UI 오류 객체에는 `reason` 속성이 없다(3.6.4).
15. (수정회차 2) sidebar 초기화 effect는 `[aiData]` 의존이며 모든 폼 상태를 덮어쓴다(3.6.6). 재조회 결과를 `setAiData`로 반영하면 미저장 편집이 사라진다.

16. (수정회차 3) `aicallhandler`의 `"ai": c`(`start.go:726`, `db.go:39,119`)의 `c`는 `*aicall.AIcall`이 아니라 `*ai.AI`다(시그니처 직접 확인). 분석서 `:222`의 반대 서술은 틀렸고 최소 정정했다. 로깅 정리 범위는 13곳에서 16곳(3.4 A).
17. (수정회차 3) webhook-manager가 ai 웹훅 페이로드 전문을 전송마다 Debug로 기록한다(`webhookhandler/webhook.go:32-34,126-128`)와 오류 경로 `request`, `data: %v`가 별도 누출이다. 3.4 B4로 신설했고 8.1 항목 8의 "`publish.go`가 유일한 방법" 문장을 정정했다.
18. (수정회차 3) 검증 전 Custom 입력 원문은 api-manager `engine_model` 로그 필드(`servicehandler/ai.go:64,401`)와 ai-control 오류(`main.go:414,432`)로도 반향된다(3.1.3).
19. (수정회차 3) 깨지는 기존 테스트 목록에 `resolve_test.go:68-79`, `views/teamgraph/__tests__/sidebar_engine_model.test.js:196,257`, `AIEngineFields.test.js:325`, `llmresolve_test.go:182`가 빠져 있었다(5.1). `views/teamgraph/__tests__/sidebar_engine_model.test.js:206`은 platform 모델의 키 미전송을 유지하기로 해 변경하지 않는다(3.6.6).
20. (수정회차 3) Popover 닫힘의 트리거 포커스 복귀가 ID 입력란 포커스와 경합한다. `onCloseAutoFocus`로 처리한다(3.6.2).
21. (수정회차 3) aicall이 생성 시점 모델을 스냅샷하지만 pipecat은 세션 시작 시 현재 AI의 키를 읽는 시점 불일치가 있다(8.3).
22. (수정회차 4) 키 이월 가드가 platform 원본에서 우회됐다. `ais_detail.js:422`가 platform 모델에서 저장 키를 재전송하고 `views/ais/__tests__/ais_detail.test.js:435-448`이 이를 고정하며 CLI/API로 만든 AI도 같은 상태다. 가드 기준을 "저장된 키가 비어 있지 않음 + 그룹 변경"으로 바꾸고 platform 원본을 포함했다(3.6.5, 4절 R1-h).
23. (수정회차 4) sidebar 편집은 `onFocus`가 `engineKeyChanged=true`를 만들고(`sidebar.js:1356-1359`) 검증 2의 유효 키 식이 `savedEngineKey`로 폴백해 포커스만으로 가드와 검증을 모두 통과했다. 그룹 변경 저장의 유효 키를 새 입력값만으로 한정했다(3.6.4 검증 2, 3.6.5).
24. (수정회차 4) sidebar 생성에는 모델 변경 시 키를 비우는 기존 지점이 없다(`sidebar.js:387,527,970`). 새 `useEffect`가 필요하다(3.6.5).
25. (수정회차 4) `AIEngineFields.js`의 `modelError` 블록은 compact(`:145`)와 non-compact(`:343`) 두 곳이다(3.6.2). 키 trim은 sidebar만 전송 시 적용하고 `ais_create.js:201`, `ais_detail.js:422`는 원문 전송이다(3.6.4). 편집 모드 Save 동기 단정 테스트가 `keyLoad` 비활성과 충돌한다(5.1).
26. (수정회차 4) `run.py:593-598`의 앞단 분기에서 `"." in type` 조건은 점 형태에 항상 참이고 콜론 형태(점 없음)에서만 거짓이다. 콜론 형태 거부의 1차 근거는 Go resolver의 정확 일치 접두다(3.3).
27. (수정회차 6) 가드 부작용을 규칙 추가가 아니라 단순화로 정리했다. (b)에 서비스 접두 비교를 더해 같은 그룹 안 벤더 변경(낡은 캐시 포함)을 닫고(제안 s, `keyModeOf`와 같은 `<service>.<model>` 카탈로그 값으로 코드 확인), 유효 키는 입력란 trim 후 비어 있지 않음 하나로 줄이고(제안 t), 경고는 sidebar 키 블록을 `<details>` 밖에 렌더하는 기존 조건부 렌더 확장으로 보이게 하고(제안 u), detail platform 목적지의 키 유지를 정직하게 적었다(제안 v).

## 4. 보안 분석: 플랫폼 키 경로 차단 증명 항목

> 정오표(코드 리뷰 3회차): 이 문서가 별도 이슈로 서술한 이벤트 버스 `engine_key` 제거(4절 잔존 위험, 8.1 항목 3)는 이 PR에서 이미 구현되었다. `bin-ai-manager/pkg/aihandler/publish_event.go`의 `publishAIEvent`가 키를 비운 복사본만 발행하므로 이벤트 버스, ClickHouse, ai 웹훅 페이로드에서 `engine_key`가 빠진다. 본문은 작성 당시 그대로 둔다.

각 항목은 구현 시 독립 테스트로 고정한다. 번호는 분석서 R1과 같다.

| # | 경로 | 방어 | 증명 방법 |
|---|---|---|---|
| R1-a | Python 새 분기 `openrouter`가 `os.getenv`로 폴백하거나 `None`/빈 값이 SDK `OPENAI_API_KEY` 폴백에 닿는 경로 | 분기 내 `os.environ` 미사용, `(key or "").strip()`이 비면 `ValueError`, SDK에는 정제된 비어 있지 않은 값만 전달 | 표 테스트: `key`가 `None`, `""`, `"  "`, `"\n"`일 때 `OPENROUTER_API_KEY`와 `OPENAI_API_KEY`가 모두 설정된 환경에서 예외, `OpenRouterLLMService`가 호출되지 않음. 변이 테스트: 분기에 `or os.getenv("OPENROUTER_API_KEY")` 한 줄을 넣으면 빨개져야 함(수동 1회, 5.2). 소스 문자열 정적 가드 테스트는 만들지 않는다(행동 테스트와 SDK 프로브가 같은 경로를 검증하고, 정적 가드는 리팩터링에 취약). 컨테이너 실증은 7절 항목 5 |
| R1-b | 단일 AI 세션에서 AI 조회 실패로 키가 빈 채 러너까지 감 | `resolveSessionLLM`이 Custom + 빈(trim) 키면 오류, 사전 분류는 `classifySessionLLM`으로 분리 | Go 표 테스트: Custom + AI 조회 실패(`aiGetErr`) 오류, direct + 조회 실패는 기존대로 진행, `Start()` 키 없는 사전 분류는 Custom 통과 |
| R1-c | `runGetLLMKey` 조회 실패, 비 AIcall 참조 유형 | R1-b와 동일. Custom + `ReferenceTypeCall` 등은 항상 거부(의도된 fail-closed) | Go 표 테스트 |
| R1-d | 팀 멤버 경로 | 거부 시 `llm_type=""`, `engine_key=""`, Python은 `llm_type==""`이면 예외 | `run_teamllmtype_test.go` 확장: Custom 멤버 + 키 있음(런너 타입 `openrouter.…`와 키 전달), Custom 멤버 + 빈 키(거부, `llm_type=""`), 정상 멤버와 혼합 팀 |
| R1-e | 서버 저장 경로(공백 키, 모델만 Custom으로 바꾸고 빈 키 유지하는 update) | `ValidateEngine`이 최종 상태 기준으로 `TrimSpace(key)` 필수, reason `ENGINE_KEY_REQUIRED` | Go 테스트: create(키 없음, 공백 키), update(모델 변경 + 빈 키, 모델 불변 + 빈 키), 정상 |
| R1-f | sidebar 편집 저장의 키 손실 | 3.6.6 재조회(모든 AI) + 서버 400 이중 방어 | UI 테스트: 이름만 변경해도 body에 재조회한 `engine_key`가 포함됨, 재조회 실패 시 저장 불가 |
| R1-g | 버전 혼재(신 Python + 구 Go, 신 Go + 구 Python) | 구 Go는 `openrouter.*`를 `Rejected`, 구 Python은 `Unsupported LLM service`, `_member_llm_type` 폴백 가드 유지 | 두 조합 모두 플랫폼 키 폴백 없음을 6절 순서와 함께 문서화. 단위 테스트는 `_member_llm_type` 유지 가드 |
| R1-h | 키 이월(고객 키가 타 벤더로 전송) | 서버는 최종 (모델, 키) 스냅샷만 보므로 판별 불가. UI 초기화/경고(3.6.5)와 키 이월 가드: **저장된 키가 비어 있지 않고 목적지 키 방식 그룹이 저장 그룹과 다르거나 서비스 접두(첫 `.` 앞)가 다르면(platform 원본 포함) 새로 입력한 키(입력란 trim 후 비어 있지 않음) 없이는 저장 불가**(`own_or_default` 목적지는 빈 값 허용, 이전 키 재전송 안 함, 제안 k, s, t). detail의 platform 재전송(`ais_detail.js:422`)과 sidebar `savedEngineKey` 경유 우회를 이 기준이 막는다. sidebar의 저장 모델 기준선은 재조회의 `savedEngineModel`이라 낡은 캐시 모델과 서버 키의 불일치 우회도 막는다(제안 o). 편집 화면 전이 시 입력 비움(제안 p) | UI 테스트(detail과 sidebar 편집 모두): create에서 키 방식 전이 시에만 키 비움(같은 방식 안 변경은 유지), 5.2의 가드 행렬 1~17 전부(같은 그룹 안 다른 서비스 접두 행 포함)(특히 platform 원본 + 비어 있지 않은 저장 키 + 목적지 `own_required`는 저장 불가, 새 키 후 가능), 같은 그룹과 같은 서비스 접두는 키 유지로 저장 가능. 서버 한계와 UI 보완 후 잔존 경로는 8.3에 명시 |

추가 승격 방지(분석서 R4):

- Go resolver 표 테스트: `openrouter.vendor/m`은 `BlankKey=false`, `RunnerType`에 `platform_openrouter.` 접두 없음. 입력 `OpenRouter.x/y`, `PLATFORM_OPENROUTER.x/y`, ` openrouter.x/y`(앞 공백), `openrouter.`(빈 ID), `openrouter.platform_openrouter.x`(슬래시 없음 거부), `openrouter.platform_openrouter/x`(통과하나 모델 이름일 뿐 서비스명 아님), `custom.openrouter`(거부), `platform_openrouter.x`(거부)에서 BYOK 외 결과가 나오지 않음.
- Python은 서비스명을 소문자화하므로(`run.py:601`), Go가 통과시킨 모든 값의 첫 점 앞이 정확히 `openrouter`(소문자)임을 resolver 표 테스트에 단정한다.
- 카탈로그 불변 조건: 어떤 카탈로그 ID도 `openrouter.`로 시작하지 않는다(3.1.1의 `custom.openrouter` 덕분에 예외 없음).

잔존 위험(이번 PR 범위 밖, 8절에 기록):

- **웹훅/GET 응답의 `engine_key` 노출은 별도 이슈다.** `ConvertWebhookMessage`가 `EngineKey`를 복사하고(`models/ai/webhook.go:29,70`) `AIGet`이 반환하는 것으로 코드상 보인다(응답 실증은 7절 항목 4). BYOK 키는 고객 OpenRouter 과금과 직결되므로 **BYOK 출시 전에 이 이슈의 처리 시점을 대표님이 결정해야 한다.** 단 3.6.6의 재조회(A안)가 이 응답에 의존하므로, 이 이슈를 진행하려면 PUT에서 `engine_key` 부재를 "변경 없음"으로 해석하는 서버 의미 변경(B안)과 UI 재조회 제거를 한 쌍으로 진행해야 한다. 한쪽만 진행하면 sidebar 편집과 `ais_detail.js:203,422` 재전송 흐름이 깨진다.
- **이벤트 버스(내부)로의 `engine_key` 발행과 ClickHouse 저장 경로(수정회차 2, 코드로 확인)**: `PublishWebhookEvent`(`publish.go:25-26`)는 같은 `data`로 `go h.PublishEvent(...)`도 호출하고, `PublishEvent`(`publish.go:88`)는 `json.Marshal(data)`로 도메인 구조체 `*ai.AI`를 그대로 직렬화해 발행한다. `ai.AI.EngineKey`는 `json:"engine_key,omitempty"`(`models/ai/main.go:67`)라 키가 비어 있지 않으면 페이로드에 포함된다. ai-manager는 전역 토픽 발행을 켠다(`cmd/ai-manager/main.go:135` `WithGlobalTopicPublish`). 소비자: **timeline-manager**가 catch-all `#`로 전량 구독하고(`pkg/subscribehandler/main.go:31`) `flushBatch`가 이벤트 데이터를 그대로 ClickHouse `events` 테이블에 INSERT한다(`main.go:248`, `dbhandler/event_insert.go:26`). 따라서 `ai_created`, `ai_updated`, `ai_deleted` 이벤트마다 **고객 BYOK 키가 ClickHouse에 저장되는 경로가 코드상 존재한다.** 이 외의 ai-manager 이벤트 소비자는 코드 grep으로 찾지 못했다(`PatternForEventType` 구독 목록 기준, 전수 보증은 아님). 이 이벤트를 고객 API로 노출하는 aggregated events는 `ConvertWebhookMessage`를 거치지만(`servicehandler/aggregated_events.go:183`) 그 변환이 `engine_key`를 복사하므로(`webhook.go:29,70`) GET/웹훅과 같은 계열의 자기 키 노출이다. **실제로 ClickHouse에 저장된 행, 보존 기간(TTL), 조회 권한은 확인하지 못했다(미확인, 7절 항목 4에 포함).** 이번 PR 범위 밖이며 8.1 항목 3의 결정 대상이다.
- 키 평문 저장(`engine_key varchar(255)`)은 기존 직접 모델 키와 동일 수준이며 신규 위험 유형이 아니다.
- 러너가 OpenRouter 오류 원문을 최대 2000자 로깅한다(`pipelineerror.go:12`). 401 본문에 키가 반사되는지는 미확인이며 7절 항목 3에서 확인한다.

## 5. 테스트 계획

원칙: 아래 테스트는 구현 단계에서 코드와 함께 작성한다. 이 문서 단계에서는 테스트 코드를 작성하지 않았다. 고객 키 역할의 실제 키, 플랫폼 운영 키는 코드와 테스트 어디에도 넣지 않는다(더미 문자열만 사용).
28. (수정회차 5, 6, 7, 8 사실) 수정회차 5: sidebar 가드 기준선을 낡은 캐시(`aiData.engine_model`)가 아니라 재조회의 `savedEngineModel`로 바꿨다(`sidebar.js:623-624` 캐시 히트 경로, `ais_detail.js`는 캐시를 읽지 않음). 기존 로드 effect(`sidebar.js:618-644`)를 확장해 캐시 미스는 기존 응답을 재사용하고 캐시 히트만 재조회한다. detail은 저장 키를 평문 `defaultValue`로 채운다(`ais_detail.js:203,1305`, 마스킹은 sidebar 편집 `:1350`만). 수정회차 6: 가드 (b)에 서비스 접두 비교를 더하고 유효 키 판정을 입력란 하나로 줄였다. 수정회차 7: 입력란 비움 트리거가 `keyMode` 전이만 보아 같은 그룹 안 벤더 변경에서 이전 키가 남던 결함을 `guardApplies` 전이로 고쳤다(3.6.5). 수정회차 8: 그 정정이 `prevKeyModeRef` 제거로 저장 키 없음(`guardApplies` 항상 false)과 가드 true인 채 그룹 변경 경로를 열었음을 `ais_detail.js:422,1305`, `sidebar.js:807-808`로 재확인하고 트리거를 그룹 변경과 가드 전이의 합집합으로 확장했다. 부수 사실(수정회차 8 정정): `sidebar.js`의 목록 캐시 쓰기(`:488`)는 `const { engine_key: _ek, ...safe } = ai`로 `engine_key`를 제외하고, 단건 캐시 쓰기(`:636-639`)도 제외한다. 키를 포함해 저장하는 곳은 `views/teams/teams_create.js:51,73`(`indexed[ai.id] = ai`)와 `provider.js:361-388`(`LoadResource`)이다(8.3).

### 5.1 수정할 기존 테스트 (의미 변경)

| 파일:라인 | 현재 | 변경 |
|---|---|---|
| `bin-ai-manager/models/ai/resolve_test.go:37` | `raw_openrouter`(`openrouter.meta-llama/llama-3-70b`) -> `Rejected` | `OutcomeCustomOpenRouter`, `RunnerType == 입력`, `RequireKey`, `!BlankKey`로 변경. 거부 케이스는 형식 오류/라우터/콜론/대소문자 변형으로 대체 |
| `bin-ai-manager/models/ai/resolve_test.go:68-79` `Test_ResolveEngineAllOpenRouterEntriesBlankKey` | `range catalog` 전체 항목이 `OutcomeCatalog`로 resolve됨을 단정하고 `RouteOpenRouter`와 `BlankKey` 대응을 검사. **Custom 항목(`custom.openrouter`)은 `ResolveEngine`이 `Rejected`를 반환하므로(3.1.1, 정확 일치 루프가 Custom 항목을 건너뜀) `must resolve`에서 실패한다** | `Route == RouteCustomOpenRouter` 항목은 루프에서 제외하고, 그 항목이 `Rejected`임을 별도 단정으로 추가 |
| `bin-ai-manager/models/ai/main_test.go:337` | `raw_openrouter_is_invalid` | 유효 Custom ID는 true, 무효 ID는 false로 분리 |
| `bin-ai-manager/pkg/aihandler/chatbot_engine_model_test.go:27` | `rejects_raw_openrouter` | 키가 있으면 수락, 키가 비면 `ENGINE_KEY_REQUIRED`로 분리 |
| 같은 파일 `:72` | `unchanged_legacy_openrouter_value_passes`(`openrouter.x/y`, 키 `"key"`) | 키가 비면 거부하는 행 추가. 기존 행은 키가 있는 최종 상태로 유지 |
| `bin-pipecat-manager/pkg/pipecatcallhandler/llmresolve_test.go:41` | `rejected raw openrouter` | Custom 수락(키 전달) 행으로 변경, 빈 키 거부 행 추가 |
| 같은 파일 `:182` `Test_startReferenceTypeCall_rejectedModel` | 입력 `LLMType: "openrouter.meta-llama/llama-3-70b"`가 거부되어 `err != nil` 단정. 새 규칙에서는 유효한 Custom ID라 모델 거부가 아니라 **빈 키 오류**로 실패 원인이 바뀐다(`ReferenceTypeCall`은 키가 없음, 3.3). 테스트는 통과하지만 "rejected model"이라는 의도와 다른 이유로 통과 | 입력을 진짜 거부 모델로 교체(`anthropic.claude-opus-4` 또는 무효 Custom ID `openrouter.openrouter/auto`). Custom + Call 참조는 별도 행(`빈 키 오류`)으로 추가 |
| 같은 파일 `:66` | 거부 모델 목록에 `openrouter.meta-llama/llama-3-70b` | 목록에서 제거하고 무효 Custom ID(`openrouter.openrouter/auto`, `openrouter.a/b:free`)로 대체 |
| 같은 파일 `:130,216` | `openrouter.x`는 AI 조회 실패와 무관하게 거부 | `openrouter.x`는 슬래시 없음으로 계속 거부. 의도 보존을 위해 유효 ID(`openrouter.a/b`) + AI 조회 실패 = 키 없음 오류 행으로 보강 |
| `bin-pipecat-manager/pkg/pipecatcallhandler/run_teamllmtype_test.go:60` | `model2: openrouter.meta-llama/llama-3-70b` 거부 | Custom 멤버 수락(키 전달)과 빈 키 거부 행으로 분리 |
| `bin-pipecat-manager/scripts/pipecat/test_run.py:1027-1034` `test_raw_openrouter_is_unsupported` | 점 형태 `openrouter.meta-llama/llama-3-70b`와 `OpenRouter:x/y` 둘 다 `Unsupported LLM service` | 점 형태 행 제거(새 `TestCustomOpenRouter`로 이동). 콜론 형태 행은 유지 |
| `models/ai/catalog_test.go` 4개 테스트, `v1_ai_models_test.go`, `chatbot_engine_model_error_test.go` | 3.1.5 | 3.1.5 표대로 |
| `cmd/ai-control/main_test.go:24-` `Test_validateEngineModelCreate/Update` | 키 인자 없음 | 키 인자 추가, Custom + 빈 키 행 추가 |
| `monorepo-javascript/square-admin/src/views/ais/ModelPicker.test.js:60-85` | 벤더 그룹 순서 `['Recommended','Google','OpenAI','Anthropic','Meta','DeepSeek','xAI']`(`:72`), `Anthropic`/`Meta` 그룹 내 행 조회(`:73`, `:81`) | 그룹이 키 방식 3개(`PLATFORM PROVIDED`, `YOUR OWN KEY`, `CUSTOM`)로 바뀌므로 그룹 이름, 순서, 그룹 내 행 조회로 변경. 벤더는 행 보조 표기로 확인 |
| 같은 파일 `:109-115` | 벤더 그룹 `OpenAI` 버튼으로 접기/펼치기 | 키 방식 그룹 헤더 버튼(`YOUR OWN KEY` 등)으로 변경 |
| 같은 파일 `:87-100`, `:132` | 벤더 검색과 `getByRole('group', { name: 'Google' })` | 검색은 label/vendor 필터 유지(변경 없음 가능), `Google` 그룹 조회는 `YOUR OWN KEY` 그룹으로 변경 |
| 같은 파일 `:39-44` | 트리거에 라벨과 벤더 표시(`Anthropic`) | 유지(트리거는 `selected.vendor`를 계속 표시). 변경 없음 확인 |
| `views/ais/aiModelsFixture.js` | `key_mode`, Custom 항목 없음 | `key_mode` 추가, Custom 항목 추가. `MOCK_AI_MODELS`를 쓰는 테스트(`ais_list`, `InsightAIsPanel`, `aicalls_*`, `member`, `TestAgentSheet`)가 항목 수/그룹에 의존하지 않는지 실행으로 확인 |
| `views/ais/__tests__/ais_create.test.js:311,318` | `API key not required` 정확 일치(`getByText`/`queryByText`) | 캡션이 `API key not required for this model.`로 바뀌므로 문자열 갱신 |
| `views/ais/__tests__/ais_detail.test.js:418,432` | 동일 | 동일 |
| `views/teamgraph/__tests__/sidebar_engine_model.test.js:184,200,260` | 동일 | 동일 |
| `views/teamgraph/__tests__/sidebar_engine_model.test.js:162,173,202,218-219`, `sidebar_ai_type.test.js:140-142,226` | 편집 모드 진입 직후 Save를 동기로 클릭하거나 `expect(saveButton()).toBeEnabled()`로 단정(`saveButton`은 `:141`의 `getByRole`) | 3.6.6 항목 4가 `keyLoad !== 'ready'`이면 Save를 비활성화한다. 그러나 `views/teamgraph/__tests__/sidebar_engine_model.test.js`와 `sidebar_ai_type.test.js`는 `localStorage.clear()`(`:126`, `:110`)로 캐시가 비어 있어 캐시 미스 경로를 타고, 이 경로는 같은 `then`에서 `setAiData(res)`와 `keyLoad='ready'`가 일어나 폼이 나타난 시점에 Save가 이미 활성이므로 대기가 필수는 아니다(수정회차 7 정정. 별도 재조회를 기다려야 하는 것은 캐시 히트 경로이며 이 파일들에는 해당 없음). 방어적으로 클릭과 `toBeEnabled` 단정 앞에 `await waitFor(() => expect(saveButton()).toBeEnabled())`만 추가하고, 이 파일들의 `ProviderGet` 모킹이 `ais/{id}`에 `engine_key`를 포함한 응답을 돌려주는지 확인한다(`sidebar_ai_type.test.js:140`은 `ProviderGet` 호출만 기다리고 응답 반영은 기다리지 않는다). `sidebar.test.js`는 Save 클릭/단정이 없어 이 변경의 영향이 없고 `ais/{id}` 응답 모킹만 확인한다(grep 확인). 생성 폼의 `Create` 단정(`:241,287,291`)은 재조회가 없어 해당 없음 |
| 같은 파일 `:196`, `:257` | `getByRole('group', { name: 'Anthropic' })`, `{ name: 'Meta' }`로 그룹 조회 | 그룹이 키 방식 3개로 바뀌므로 `PLATFORM PROVIDED` 그룹 조회로 변경(Claude Sonnet 4.5, Llama 3.3 70B 모두 `platform`) |
| 같은 파일 `:206` | `expect(body).not.toHaveProperty('engine_key')`(platform 모델로 바꾼 뒤 저장) | **변경 없음(유지).** 3.6.6 항목 3이 platform 모델은 현행대로 키를 보내지 않기로 했으므로 이 단정이 계약을 계속 고정한다. 초안처럼 platform도 재전송하면 이 테스트가 깨진다는 점이 platform 유지 결정의 근거 중 하나(3.6.6) |
| `views/ais/__tests__/ais_create.test.js:282-288` | 플레이스홀더 `/ai engine api key/i` | 변경 없음(`platform`, `own_or_default` 플레이스홀더 유지) |
| `views/ais/AIEngineFields.test.js:325` | `getByRole('group', { name: 'OpenAI' })`로 GPT-5 선택 (이전 초안은 "유지 확인만"이라고 적었으나 틀렸다. 그룹 이름이 바뀌므로 깨진다) | `YOUR OWN KEY` 그룹 조회로 변경. `:303-336`의 "picker not free-text" 단정(`combobox` 존재 확인 `:305,309`, `:316`)은 Custom 값이 아닌 경우 유지. Custom ID 입력 테스트는 `ModelPicker.test.js`로 이동(5.2). `modelError` 문단이 `ModelPicker`의 `error` prop으로 옮겨가므로 표시 위치 단정이 있으면 함께 수정(3.6.2) |

의미가 바뀌지 않아 **수정하지 않는** 테스트: `test_init_pipeline.py:804,827,890`(3.8 항목 3). 단 `_member_llm_type` 가드를 유지한다는 결정이 바뀌면 재검토.

### 5.2 신규 테스트

Go ai-manager:
- `ValidateCustomModelID` 표 테스트: 정상(`mistralai/mistral-large-2411`, `meta-llama/llama-3.3-70b-instruct`), 길이 경계(ID 244자 통과, 245자 거부, 접두 포함 255), 공백/제어문자/쉼표/`~`/`@`, `:free` 등 콜론 변종 전부, 슬래시 0개/2개 이상, `openrouter/auto`, `OpenRouter/auto`, 대소문자 허용(`Mistralai/Large`는 통과, 정규화하지 않음).
- `ResolveEngine` 승격 방지 표(4절), `custom.openrouter` 거부, 정확 일치 루프가 Custom 항목을 건너뜀.
- `ValidateEngine`: 3.1.2 모든 경우(`modelChanged` 조합 x 키 공백 조합).
- aihandler: Create/Update의 `ENGINE_KEY_REQUIRED`(reason, status, HTTP 400, 메시지에 키 값 없음), Custom 모델 ID 오류 메시지가 입력값을 되돌리지 않음.
- 카탈로그: 3.1.1, 3.1.5 항목, `key_mode` 대응표, `model_id_prefix`는 Custom에만 존재, JSON 필드 존재와 `omitempty`.
- **가드 허용 목록 고정(3.1.5)**: Custom 항목에서 `openrouter` 문자열이 허용되는 필드 집합 `id`, `label`, `vendor`, `description`, `model_id_prefix`를 테스트 상수로 고정하고, 이 집합 밖 필드(`key_mode` 등)에 문자열이 나타나거나 허용 목록이 바뀌면 실패한다. `model_id_prefix`가 있는 항목은 정확히 1개, `slug`, `route`, `meta-llama/`는 Custom 항목에도 금지, 비 Custom 항목은 기존 4개 토큰 금지를 그대로 유지함을 단정(1번, 3번 가드 공통, 2번은 `label`, `description`).
- **모델 ID에 키 붙여넣기 방어(3.1.3 슬래시 규칙)**: 더미 문자열(실제 키 형식이 아닌 고정 더미 `dummy-key-not-real`, 슬래시 없음)을 `openrouter.` 뒤에 붙인 값이 `Rejected`이고 aihandler가 `INVALID_ENGINE_MODEL`을 반환하며 **오류 메시지에 입력한 더미 문자열이 포함되지 않음**. 슬래시 없는 일반 문자열, 슬래시 2개 이상도 같은 방식.
- B1 로그 캡처(3.4): `v1_ai_builder_test.go:221-224` 선례처럼 로거 출력을 버퍼로 캡처하고, 더미 `engine_key`가 담긴 `POST /v1/ais`가 오류(검증 실패 등)로 끝나게 한 뒤 캡처된 로그에 더미 키 문자열이 없고 `uri`, `method`가 있음을 단정. `PUT /v1/ais/{id}`도 동일.
- ai-control: Custom + 빈 키 사전 검증 실패, `openrouter.` 접두 무효 입력의 오류 문자열에 입력값이 없음(3.1.3 반향 방어).
- api-manager: `loggableEngineModel`(3.1.3)이 `openrouter.` 접두(대소문자 무시) 입력을 `openrouter.<redacted>`로 바꾸고 그 외 입력은 그대로 둠(순수 함수 표 테스트).

Go pipecat-manager:
- `classifySessionLLM`: Custom은 키 없이 통과, rejected는 오류.
- `resolveSessionLLM`: Custom + 키(공백 포함 키는 trim되어 전달), Custom + 빈/공백 키 오류, 오류 문자열에 모델 ID와 키가 없음, platform은 기존대로 키 비움. `openrouter.` 접두(대소문자 무시)로 시작하는 `Rejected` 입력의 오류 문자열에 입력값이 없고 그 외 입력은 현행 문구 유지(3.3 결정).
- `Start()` 경로: Custom 모델이 사전 분류를 통과해 DB row가 생성됨.
- 팀 경로(R1-d).

Python:
- `TestCustomOpenRouter`(`test_run.py`): 키 정상 시 `OpenRouterLLMService`가 `api_key=<정제된 키>`로 한 번 호출, `Settings.model`이 접두 뒤 전체 ID(점 포함 ID 보존), `extra == {"extra_body": {"provider": _OPENROUTER_PROVIDER}}`, 두 분기(platform, custom)의 provider dict가 동일 상수에서 나옴, 키 `None`/`""`/공백은 `ValueError`, 환경변수 `OPENROUTER_API_KEY`와 `OPENAI_API_KEY` 설정 상태에서 전달된 `api_key`가 환경변수 값이 아님, 콜론 형태는 `Unsupported LLM service`.
- 변이 테스트(수동 절차를 문서화하고 CI에는 포함하지 않음): 분기에 `or os.getenv(...)`를 넣으면 위 테스트가 실패함을 구현 중 1회 실행으로 확인하고 PR 설명에 기록한다. `os.environ`/`getenv` 소스 문자열을 찾는 정적 가드 테스트는 만들지 않는다(위 행동 테스트와 SDK 프로브가 같은 경로를 검증한다).
- 팀: `llm_type="openrouter.a/b"` + 키가 `create_llm_service`로 그대로 전달.
- `conftest.py:55-63`이 pipecat 전체를 `MagicMock`으로 대체해 SDK 동작을 검증하지 못하므로, SDK 폴백 방어는 7절 항목 5의 실제 SDK 실행으로 보완한다(수동 스크립트, CI 제외. `_real_openrouter_probe_script`(`test_run.py` 마지막 구간) 방식의 서브프로세스 프로브 테스트가 이미 있으므로 같은 패턴으로 `OPENAI_API_KEY` 설정 + 빈 키 케이스를 프로브로 추가하는 방안을 구현 시 검토).

UI(vitest): `ModelPicker.test.js` 확장(그룹 3개, 배지 3종, Custom 선택 시 접두 전달, 이미 Custom이면 행 재선택이 입력한 ID를 유지, 저장값 `openrouter.*`가 Custom으로 복원되고 `Current:` 행이 없음, `key_mode` 부재 구 API 폴백, **Custom 선택 시 ID 입력란이 트리거 아래에 렌더되고 입력이 `onChange(prefix + typed)`로 전달됨, `error` prop이 입력란 아래에 표시됨, 비 Custom 값에서는 입력란 없음**), `AIEngineFields.test.js`(`modelError`가 입력란 다음에 표시), create/detail/sidebar(생성, 편집) 키 필수 인라인 오류와 서버 `ENGINE_KEY_REQUIRED`/`INVALID_ENGINE_MODEL` 응답 시 인라인 표시(`engineErrorOf`: reason 포함 판정, JSON 파싱 실패 시 폴백 문구), 모델 ID 비어 있음 오류, 4곳 라벨/캡션이 키 방식별로 3.6.3 표와 일치, **sidebar `own_required`에서 키 블록(`OpenRouter API Key *`)이 `<details>` 밖에 렌더되고 `Advanced` 요약이 없음, sidebar 편집에서 가드가 적용되면 목적지 키 방식과 무관하게 키 블록과 목적지별 경고가 `<details>` 밖에 렌더됨, 둘 다 아니면 키 입력이 접힌 `Advanced` 안에 있음(3.6.4 제안 m, u, jsdom의 `details` 속성 동작은 단정하지 않음)**, Q8 키 초기화/경고, **제안 c(저장값이 `own_required`가 아닐 때 Custom으로 바꾸고 키를 그대로 두면 저장 불가, 새 키 입력 후 가능, 저장값이 Custom이면 키 유지로 저장 가능)**, **sidebar 다른 탭 배너(3.6.4 이중 표시)**(Engine 탭이 아닌 탭에서 저장해 키/모델 검증 실패, 서버 `ENGINE_KEY_REQUIRED`/`INVALID_ENGINE_MODEL`, `keyLoad` 오류(캐시 히트 재조회 실패만. 캐시 미스 실패는 폼이 숨겨짐)가 발생하면 `createError`(생성) 또는 `saveError` 배너(편집)에 같은 문구가 표시되고 인라인에도 표시됨), **키 이월 가드(3.6.5, 제안 k, l)**(아래 행렬을 detail과 sidebar 편집 모두에서 실행), **create 키 비움이 키 방식 그룹 전이에서만 발생**(`prevCreateKeyModeRef`, 같은 방식 안 모델 변경, 같은 그룹 안 벤더 변경과 Custom ID 타이핑 중에는 타이핑한 키 유지. sidebar 생성은 새로 추가하는 `prevCreateKeyModeRef` effect(3.6.5)로 같은 동작을 검증), **편집 화면 `prevGuardRef` 전이 effect**(카탈로그가 `ready`가 되기 전에는 비우지 않고 `ready` 이후 첫 실행은 기록만 하며, 그룹 변경 또는 false에서 true에서 비우고 true에서 false에서 복원함, 첫 실행은 저장 모델 기준선과 비교, 저장 키 없음과 로딩 중 타이핑 후 그룹 전환도 닫힘, 행 18~20), **Custom 행을 선택할 때마다 ID 입력란에 포커스가 남음**(처음 고르는 경우와 이미 Custom일 때 재선택하는 경우 모두, Popover 닫힘의 트리거 포커스 복귀를 `onCloseAutoFocus`가 막음, 비 Custom 선택은 트리거 복귀), **저장 직전 trim**(앞뒤 공백이 있는 Custom ID가 제거된 값으로 전송되고 입력 중에는 변환 없음), **오류 단일 경로**(`modelError`가 `ModelPicker`의 `error`로만 표시되어 한 번만 나타남), Q9 재조회(**캐시 미스에서는 추가 GET 없이 기존 응답으로 `savedEngineKey`, `savedEngineModel`이 채워지고 캐시 히트에서만 재조회가 한 번 호출됨, `ProviderGet` falsy 응답은 `keyLoad='error'`**, 성공 시 키 재전송(platform 모델은 현행대로 `engine_key` 미전송, `views/teamgraph/__tests__/sidebar_engine_model.test.js:206` 유지), 실패 시 저장 불가, **재조회 응답이 늦게 도착해도 사용자가 입력한 이름 등 폼 상태가 초기화되지 않음, 다른 AI로 바뀐 뒤 도착한 이전 응답은 무시됨, 키 입력란에 포커스만 하고 저장해도 저장된 키가 body에 유지됨**).

**테스트 주의(R14 minor 3)**: detail 테스트는 `AIEngineFields`를 `jest.mock('../AIEngineFields', ...)`으로 대체하고(`views/ais/__tests__/ais_detail.test.js:84`) 모델 변경 수단이 `onEngineModelChange` 호출 하나뿐이다. 행 13~20의 전환 시나리오를 돌리려면 이 mock에 Custom, OpenAI, Gemini, platform 모델로 바꾸는 전환 버튼(각각 해당 모델 값으로 `onEngineModelChange` 호출)을 추가해야 하며, mock이 키 방식 그룹 판정에 쓰는 카탈로그 상태도 함께 제공해야 한다. 이 mock 확장은 구현 plan의 테스트 작업에 포함한다.

**키 이월 가드 행렬(3.6.5, detail과 sidebar 편집 모두 같은 시나리오. 키는 모두 더미 `dummy-old-key`, `dummy-new-key`)**: 구현은 아래 표를 그대로 `test.each` 표(행 번호, 저장 상태, 목적지, 입력, 기대를 열로 가진 배열)로 옮기고 detail과 sidebar 편집이 같은 표를 공유하게 한다. 한 행은 한 케이스다. detail 전용(행 13, 15의 detail 단정)과 sidebar 전용(행 10, 15의 캐시 불일치) 열은 표의 `only` 필드로 구분한다.

| # | 저장 상태(모델, 저장 키) | 목적지 | 사용자 입력 | 기대 |
|---|---|---|---|---|
| 1 | platform(Claude), `dummy-old-key`(비어 있지 않음) | `own_required`(Custom + ID) | 키란을 건드리지 않음(detail은 입력란에 `dummy-old-key`가 채워진 상태) | **저장 불가**, 3.6.3 경고 오류, `ProviderPut` 미호출. 수정회차 4의 핵심 회귀 시나리오 |
| 2 | 1과 같음 | 1과 같음 | sidebar는 키란에 포커스만 | 저장 불가(포커스만으로는 새 입력이 아님) |
| 3 | 1과 같음 | 1과 같음 | `dummy-new-key` 입력 | 저장 가능, `body.engine_key === 'dummy-new-key'`, body 어디에도 `dummy-old-key` 없음 |
| 4 | 1과 같음 | `own_or_default`(Gemini) | 입력 없음 | 저장 가능, `body.engine_key === ''`(detail은 전이 시 입력란이 비워지므로 입력 없음이면 `''`), `dummy-old-key` 미전송 |
| 5 | own_or_default(OpenAI), `dummy-old-key` | `own_required` | 입력 없음 / `dummy-new-key` | 저장 불가 / 저장 가능(body에 새 키만) |
| 6 | own_required(Custom), `dummy-old-key` | `own_or_default` | 입력 없음 | 저장 가능, `body.engine_key === ''` |
| 7 | own_required(Custom), `dummy-old-key` | `platform` | 입력 없음 | 저장 가능. detail은 현행대로 저장 키 재전송(`views/ais/__tests__/ais_detail.test.js:435-448` 유지), sidebar는 `engine_key` 미전송(`views/teamgraph/__tests__/sidebar_engine_model.test.js:206` 유지) |
| 8 | platform, 저장 키 없음(`''`) | `own_required` | 입력 없음 / `dummy-new-key` | 가드 비적용, 검증 2의 (가)로 저장 불가 / 저장 가능 |
| 9 | own_required(Custom), `dummy-old-key` | 다른 Custom ID(같은 그룹) | 입력 없음 | 저장 가능, 저장된 키 유지(detail은 입력란 값, sidebar는 `savedEngineKey`) |
| 10 | **캐시 모델과 서버 모델 불일치(sidebar 전용, R7-M1)**: 캐시 `openrouter.a/b`(Custom), 재조회 응답은 OpenAI 모델 + `dummy-old-key` | 폼 선택은 캐시 값 Custom 그대로(이름만 변경) | 키란을 건드리지 않음 / `dummy-new-key` | 저장 불가(기준선은 `savedEngineModel`=OpenAI, 그룹 불일치) / 저장 가능, body에 `dummy-old-key` 없음. 재조회 전과 실패에서는 Save 비활성 |
| 11 | **같은 그룹, 같은 서비스 접두 회귀 방지**: own_or_default(`openai.gpt-5`), `dummy-old-key` | own_or_default(`openai.gpt-5.1`) | 입력 없음 | 가드 비적용, 저장 가능, 저장된 키 유지(detail은 입력란 값, sidebar는 `savedEngineKey` 전송). 기존 동작 유지 |
| 12 | **카탈로그에 없는 저장값에서 Custom으로**: 저장 `engine_model`이 현 카탈로그에 없는 값(예: 폐기된 모델 ID), `dummy-old-key` | Custom + ID | 입력 없음 / `dummy-new-key` | `keyModeOf`가 `own_or_default`로 폴백한다는 규칙(3.6.1)에 따라 그룹 불일치로 저장 불가 / 저장 가능. 저장값이 `openrouter.` 접두면 UI는 Custom으로 복원(3.6.2)되어 같은 그룹으로 취급 |
| 13 | **이전 모드에서 타이핑한 키(제안 p)**: own_or_default(OpenAI) + 저장 키 `dummy-old-key` 상태에서 입력란에 `dummy-typed-key` 입력 후 Custom으로 전환 | `own_required` | 전환 직후 입력란 확인, 그대로 저장 시도 | `guardApplies`가 false에서 true로 바뀌는 전이이므로 입력란이 비어 있고(detail은 저장 키 미리 채움도 비움, sidebar는 `engineKeyChanged=false`) 저장 불가. 다시 OpenAI로 돌아오면(true에서 false) 입력란이 저장 키 상태로 복원 |
| 14 | 가드가 true인 채 **같은 그룹 안**에서 이어지는 모델 변경(Gemini에서 Grok, Custom ID 타이핑)과 같은 접두 안의 변경(가드 false인 채) | 가드 비적용 또는 이미 적용된 상태 | 가드 적용 뒤 입력한 `dummy-new-key` | 모델 ID를 바꾸거나 타이핑해도 입력이 유지됨. **그룹 변경은 비움**(제안 w, 행 18, 20). 같은 그룹 안 변경에서는 가드의 false에서 true 전이에서만 비움 |
| 15 | **같은 그룹, 다른 서비스 접두(제안 s)**: own_or_default(`openai.gpt-5`), `dummy-old-key`. sidebar는 캐시 `openai.*`와 재조회 `gemini.*` 불일치 경우도 포함 | own_or_default(`gemini.gemini-2.5-pro`) | 입력 없음 / `dummy-new-key` / 공백만 입력 / **(detail 전용) 입력란에 `dummy-old-key`가 미리 채워진 상태에서 모델만 변경** / **(sidebar 전용) OpenAI용으로 `dummy-typed-key`를 타이핑한 뒤 모델만 변경** | 가드 적용. 입력 없음과 공백만은 저장 가능하되 `body.engine_key === ''`(저장 키 미전송), `dummy-new-key`는 body에 새 키만. 공백만 입력은 새 입력으로 보지 않음(trim 후 비어 있음). **detail 전용 단정: 미리 채운 키가 있는 상태에서 `openai.*`에서 `gemini.*`로 변경하면 입력란이 비어 있고 저장 시 `body.engine_key === ''`**. sidebar 전용 단정: 타이핑한 키가 사라지고(`engineKeyChanged=false`, 입력란 비어 있음) body에 `dummy-typed-key`가 없음. 이후 Grok으로 한 번 더 바꿔도 가드가 true인 채라 그 사이 입력한 `dummy-new-key`는 유지 |
| 16 | 경고 문구(제안 r) | `own_required` / `own_or_default` / `platform` | 가드 적용 상태 | 각각 3.6.3의 목적지별 문구가 표시됨. platform 문구는 sidebar 편집에서만, detail에는 없음. sidebar 편집은 세 목적지 모두 키 블록이 `<details>` 밖에 있어 경고가 보임(제안 u). detail platform 목적지는 입력란이 비어 있으나 저장 키를 재전송함을 단정(제안 v) |
| 17 | **platform 그룹 안 접두 불일치(수정회차 7)**: platform(`anthropic.claude-sonnet-4.5`), `dummy-old-key` | platform(`meta.llama-3.3-70b-instruct`) | 입력 없음 | 서비스 접두가 달라(`anthropic`과 `meta`, `catalog.go:78-86`) 가드가 적용되고 입력란이 비워지며, 목적지 `platform`이라 새 키는 요구하지 않고 저장 가능. sidebar에만 경고 `The saved key will be removed.`가 표시되고 detail에는 없음(제안 r). detail은 저장 키 재전송 유지(`views/ais/__tests__/ais_detail.test.js:435-448`), sidebar는 `engine_key` 미전송 |
| 18 | **저장 키 없음 + 타이핑 후 그룹 전환(R13-M1 (A))**: own_or_default(OpenAI), 저장 키 `''` | `own_required`(Custom + ID) 그리고 반대로 저장 Custom(저장 키 없음)에서 OpenAI | OpenAI(또는 Custom) 맥락에서 `dummy-typed-key` 타이핑 후 전환, 그대로 저장 시도 | 그룹 변경이므로 입력란이 비어 있고(sidebar는 `engineKeyChanged=false`), `body.engine_key`에 `dummy-typed-key`가 없음. Custom 목적지는 검증 2의 (가)로 저장 불가, OpenAI 목적지는 `''` 전송. 같은 그룹 안 벤더 변경(OpenAI에서 Gemini, 저장 키 없음)은 타이핑한 키를 유지(create와 동일, 가드 false이고 그룹 같음) |
| 19 | **로딩 중 타이핑 후 전환(수정회차 8)**: own_or_default(OpenAI), 저장 키 `dummy-old-key`(또는 없음) | Custom 또는 다른 그룹 | 카탈로그/`keyLoad`/`detailData`가 ready 되기 전에 입력란에 `dummy-typed-key` 타이핑 후 다른 그룹으로 변경, ready 후 저장 시도 | ready 후 첫 effect가 저장 모델 기준선과 비교해 그룹이 다르면 비움. 입력란이 비어 있고 `body`에 `dummy-typed-key`가 없음. 정상 로드(모델 변경 없음)에서는 비우지 않고 첫 실행은 기록만 |
| 20 | **가드 true인 채 그룹 변경(R13-M1 (B))**: 저장 (OpenAI, `dummy-old-key`) | Gemini로 변경 후 Custom, 또는 저장 키 없는 Custom에서 OpenAI | Gemini로 바꾼 뒤 `dummy-gemini-typed-key` 타이핑 후 Custom으로 전환 | 입력란이 비어 있고 body에 `dummy-gemini-typed-key`와 `dummy-old-key`가 없음. 전환 뒤 새로 입력한 `dummy-new-key`는 유지. 저장 그룹으로 복귀하면(true에서 false) 저장 키 상태로 복원 |

로깅 16곳(3.4 A)과 webhook-manager B4(3.4)는 전용 회귀 테스트를 만들지 않는다(Q10). 기존 테스트가 로그 필드 변경으로 깨지지 않는지만 전체 `go test`로 확인한다. 3.4 B1은 위 로그 캡처 테스트 1개를 추가하고, B3(공용 `publish.go`)을 채택하면 그 줄 삭제는 별도 테스트 없이 `bin-common-handler` 전체 `go test`로만 확인한다.

### 5.3 검증 절차(구현 후)

`bin-ai-manager`, `bin-pipecat-manager`(Go), `bin-api-manager`, `bin-call-manager`, `bin-openapi-manager`, `bin-webhook-manager`(3.4 B4, 필수 제안)(그리고 3.4 B3 채택 시 `bin-common-handler`와 이를 쓰는 `bin-conversation-manager`) 각각 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, Python은 `bin-pipecat-manager/scripts/pipecat`의 pytest 전체, UI는 square-admin의 vitest와 lint/빌드. 위 절차는 각 서비스 `CLAUDE.md`의 규칙을 구현 시 다시 확인해 적용한다.

## 6. 롤아웃 순서와 배포 의존성

### 6.1 PR 구성

- **monorepo PR 1개**: ai-manager(로그 포함 B1), pipecat-manager(Go+Python), api-manager(로그 4줄과 `engine_model` 반향 방어, gens), call-manager(로그 1줄), webhook-manager(3.4 B4, 로그 9줄), openapi-manager, RST, 그리고 8.1 항목 8에서 채택 시 common-handler `publish.go` 한 줄. 한 저장소 한 작업이므로 분리하지 않는다.
- **monorepo-javascript PR 1개**: square-admin UI, `square-main/public/skill.md`, `llms.txt`. 저장소가 달라 구조상 필요한 별도 PR이다(분할이 아님).
- 브랜치는 각 저장소 별도 워크트리에서 만든다(UI 저장소에는 `.worktrees/NOJIRA-Add-custom-OpenRouter-BYOK-models`가 이미 있고 origin/main과 동일하다. 구현 시 이 워크트리를 사용). 두 PR 모두 main 최신 fetch 후 충돌 확인, 머지는 대표님 명시 지시 후 squash.

### 6.2 배포 순서 (서비스 단위)

1. **pipecat-manager Go와 Python 러너를 동시에 배포**(같은 이미지/compose의 두 구성요소). 한쪽만 먼저 나가면 신 Python + 구 Go는 `Rejected`, 신 Go + 구 Python은 `Unsupported LLM service`로 실패하나 모두 fail-closed(플랫폼 키 폴백 없음, 4절 R1-g).
2. **ai-manager와 api-manager를 함께 배포**. ai-manager가 먼저 나가면 Custom 항목이 카탈로그에 보이고 저장은 되지만 api-manager가 구 `ModelInfo`로 역직렬화해 `key_mode`, `model_id_prefix`를 응답에서 떨군다(3.2). 반대로 api-manager만 먼저 나가면 필드가 비어 있다. 두 서비스는 같은 PR 산출물이므로 순서보다 동시 배포가 요건이다. 1번이 이보다 먼저여야 한다(2번이 먼저면 Custom 저장 후 호출이 `Rejected`로 실패, 안전하나 UX 열화).
3. call-manager(로그 1줄)는 독립적이며 순서 제약이 없다. **webhook-manager(B4)는 ai-manager와 함께 배포한다**(ai 웹훅 페이로드를 상시 기록하는 서비스이므로 BYOK 출시 전에 반영되어야 한다). 공용 `publish.go` 로깅 수정(8.1 항목 8 채택 시)은 ai-manager 배포에 포함되고, conversation-manager는 같은 PR 산출물이므로 함께 배포하기를 권장한다(나머지 14개 서비스는 다음 정기 배포).
4. **UI 배포는 마지막.** 신 UI + 구 API는 `key_mode` 부재 시 `platform_managed` 폴백(3.6.1)으로 동작하고 Custom 항목이 없으므로 Custom UI가 나타나지 않는다. 구 UI + 신 API는 Custom 항목을 일반 모델 행으로 보여 UX가 열화되나 서버 검증(`INVALID_ENGINE_MODEL`/`ENGINE_KEY_REQUIRED`)이 최종 방어다.
5. 문서(RST)와 `skill.md`는 위 4번과 같은 시점에 공개한다.

### 6.3 출시 게이트

아래 조건은 모두 **ai-manager와 api-manager 배포 전 조건**이다(시점 기준 통일, 수정회차 2). 항목 1은 추가로 머지 게이트를 겸한다(분석서 8절 항목 3).

1. 7절 항목 5(러너 폴백 실행 검증)와 항목 3(빈 키 실행 검증) 통과. 통과 전에는 머지하지 않고 배포하지도 않는다.
2. 로그: 3.4의 A(16곳), B1, B4가 배포 대상 빌드(ai-manager, api-manager, call-manager, webhook-manager)에 포함되어 있음과, 이미 쌓인 로그 처리 판단(7절 항목 9로 기존 로그에 `engine_key`/요청 본문이 남았는지 조회, 8.1 항목 4)과 공용 로깅 범위 결정(8.1 항목 8)이 끝나 있음. 운영 로그 레벨 실증은 조건이 아니다(ai-manager는 `DebugLevel` 무조건 설정, 3.4).
3. `engine_key` 노출 이슈(GET, 웹훅, 이벤트 버스/ClickHouse)의 처리 시점 결정(4절 잔존 위험, 8.1 항목 3).
4. 7절 항목 1~2(ZDR, 오류 원문) 결과로 고객 안내 문구 확정 여부. 확정 전에는 3.7의 기본 동작과 문서 안내 문구로 출시할지 대표님이 결정(8.1 항목 5).

롤백: 기능 플래그가 없으므로 롤백은 서비스 이미지 되돌리기다. ai-manager만 구버전으로 되돌리면 이미 저장된 `openrouter.*` AI는 구 검증에서 "변경 시에만 검증"이라 기존 행은 유지되지만 구 pipecat resolver는 `Rejected`로 거부해 호출이 실패한다(안전한 실패). 롤백 후 Custom AI 정리는 운영자가 직접 처리(8절 항목 7).

## 7. 실호출 검증 계획

| # | 항목 | 수행자 | 시점 | 비고 |
|---|---|---|---|---|
| 1 | 고객 키 역할의 OpenRouter 키로 `provider.zdr:true` 호출: ZDR 제공자가 있는 모델 성공, 없는 모델 실패의 HTTP 상태와 원문 | Claude 호출, 키는 대표님 승인 후 사용 | 구현 중(머지 전) | Q5 안내 문구 확정 근거 |
| 2 | 무효 키(401), 크레딧 없음(402)에서 러너가 보는 오류 원문, `classifyPipelineError` 분류 결과, 키 반사 여부 | Claude | 구현 중 | 분류기 확장 여부 근거. 원문에 키가 반사되면 로깅 마스킹 필요 |
| 3 | 빈 키/공백 키/`None`으로 Custom 호출 시 `OPENROUTER_API_KEY`가 설정된 컨테이너에서도 예외, 외부 요청 없음 | Claude(더미 값) | 구현 중, 머지 게이트 | 4절 R1-a |
| 4 | `GET /ais`, ai 웹훅, 이벤트 버스 경로(`ai_*` 이벤트가 timeline-manager를 거쳐 ClickHouse `events.data`에 저장되는지, 저장된 행에 `engine_key`가 있는지, 보존 기간)에 `engine_key`가 실제 포함되는지 | Claude(더미 값, ClickHouse 조회는 대표님 접근 필요) | 구현 중 | RST/OpenAPI 문구 정정 및 별도 이슈 근거 |
| 5 | 러너 동일 환경(`OPENROUTER_API_KEY`, `OPENAI_API_KEY` 주입)에서 SDK 폴백 실제 동작과 Custom 분기 방어 확인 | Claude | 구현 중, 머지 게이트 | 소스 확인은 완료(3.3), 실행은 미수행 |
| 6 | sidebar 편집 저장 시 `engine_key`가 빈 값으로 덮이는지(재조회 적용 전 동작 확인과 적용 후 비교) | Claude(UI 실행) | 구현 중 | Q9는 실증과 무관하게 적용 |
| 7 | `openrouter.vendor/model:variant`류가 첫 점 분리와 `extra_body` 전송을 거쳐도 유지되는지(단위), 변종 ID가 ZDR을 유지하는지(실호출) | Claude(단위) + 대표님 승인 키(실호출) | 구현 중 또는 후속 | 결과에 따라 Q4 변종 순차 허용 |
| 8 | 라우터형 ID(`openrouter/auto` 등)의 ZDR 상호작용과 요청과 다른 제공자로의 라우팅 여부 | Claude + 대표님 승인 키 | 구현 중 또는 후속 | 결과 전까지 거부 유지 |
| 9 | 이미 쌓인 운영 로그와 ClickHouse `events`에 `engine_key`/요청 본문이 남아 있는지 조회(필드 존재 여부와 건수만 확인하고 값은 출력하지 않음) | 대표님(운영 로그 접근) | ai-manager, api-manager 배포 전(6.3 게이트 2) | 이미 쌓인 로그 처리와 기존 키 교체 판단 근거. 로그 레벨과 포매터 실증은 하지 않는다(`DebugLevel` 무조건 설정, 3.4). 3.4 A와 B1 수정은 이 결과를 기다리지 않음 |

**키 사용 원칙**: 항목 1, 2, 7, 8의 실호출에는 고객 역할의 OpenRouter 키가 필요하다. 대표님이 허용하면 **운영 플랫폼 `OPENROUTER_API_KEY`를 일회성 수동 확인에만 사용**한다(터미널 환경변수로 주입해 단발 호출, 값은 출력/로그/파일에 남기지 않음). 이 키는 **코드, 테스트, 픽스처, 문서, 커밋에 절대 넣지 않는다.** 플랫폼 키는 가드레일과 ZDR 계정 설정이 고객 계정과 다를 수 있어(요청 값과 계정 설정은 OR 관계, 분석서 R2) 결과가 고객 환경과 다를 수 있으므로, 응답 형태 확인용 참고 결과로만 쓰고 고객 계정 가드레일 시나리오(허용 제공자 제한 등)는 확인 불가로 8절에 남긴다. **현재 이 허용은 받지 않았고 대표님 허용 대기 상태이며, 이 문서 작성 중 어떤 키로도 실호출을 하지 않았다.**

## 8. 열린 항목과 위험

### 8.1 대표님 확인이 필요한 항목

| # | 항목 | CPO 권장 |
|---|---|---|
| 1 | 모델 ID 형태를 `author/slug` 슬래시 1개로 고정하는 규칙(3.1.3). 확정된 Q4는 허용 문자와 변종/라우터 거부까지이며, 형태 고정은 이번 설계가 추가했다(실제 OpenRouter ID가 모두 이 형태라는 점은 미실증). 빠뜨리면 `openrouter.x`처럼 슬래시 없는 값이 통과한다 | 채택. 미채택 시 `openrouter.` 뒤 슬래시 없는 값이 OpenRouter에서 오류로 드러남 |
| 2 | `own_or_default` 모델의 키 입력 라벨 `Engine Key (optional)`와 캡션 `Leave empty to use the platform default key.`(3.6.3). 4개 폼의 현재 라벨이 모두 달라(`Engine Key`, `Engine Key`, `Engine Key (optional)`, `API Key (optional)`) 키 방식별 라벨 표 하나로 통일한다. 분석서와 목업은 패널 3의 비슷한 문구를 "현재 동작"으로 잘못 제시했고 Q3 확정 문구에는 이 두 줄이 없다 | 배지 `Your key or default`와 일치하는 정직한 표기이므로 채택 |
| 3 | `engine_key` 노출 이슈의 처리 시점. 노출 경로는 3곳이다: ① `GET /ais` 응답, ② ai 웹훅, ③ **이벤트 버스**(`PublishEvent`가 `*ai.AI`를 `json.Marshal`해 발행, timeline-manager가 `#`로 전량 구독해 ClickHouse `events`에 저장하는 경로가 코드상 존재, 4절 잔존 위험. 실제 저장 행과 TTL은 미확인). BYOK 출시 전 처리할지 결정 필요. 재조회(Q9)와 결합되어 ①은 PUT 의미 변경과 한 쌍으로만 진행 가능하며, ③은 ai-manager가 발행 전에 키를 제외한 별도 페이로드를 만드는 별도 설계가 필요하다 | 출시 게이트 3(6.3)로 두고 대표님 일정 결정. ③은 ClickHouse 조회로 실제 저장 여부를 먼저 확인(7절 항목 4, 9) |
| 4 | 이미 쌓인 로그의 보관/폐기, 기존 OpenAI/Gemini/Grok 키 교체 권고 | 7절 항목 9 결과를 보고 판단 |
| 5 | ZDR 없음 안내를 실호출 전 기본 동작(`unknown`, 음성 세션 무음 가능)으로 출시할지, 실호출과 문구 확정 후 출시할지 | 실호출 후 출시(무음 통화는 고객 신뢰 문제) |
| 6 | 7절 실호출에 플랫폼 `OPENROUTER_API_KEY` 일회성 사용 허용 | 대표님 허용 대기 |
| 7 | 롤백 시 이미 생성된 Custom AI 처리 방침(6.3) | 사례 발생 시 개별 처리 |
| 8 | **이슈 시점 확인 필요: 로깅 공용 모듈(`publish.go`)과 webhook-manager 수정 범위.** (1) `bin-common-handler/pkg/notifyhandler/publish.go:34` `PublishWebhook`의 `"data": data` 로깅(`:46`, `:51` 오류 시 출력). 공용 모듈이라 `PublishWebhookEvent`를 호출하는 16개 서비스(agent, ai, call, campaign, conference, contact, conversation, email, flow, message, number, outdial, queue, talk, transcribe, webchat)가 영향을 받고, 비밀값 필드를 가진 타입을 넘기는 서비스는 ai-manager(`engine_key`)와 conversation-manager(account `secret`, `token`)뿐이다(3.4 B3). (2) `bin-webhook-manager`의 상시 `Debugf` `"data": data`(`webhookhandler/webhook.go:32-34,126-128`, 웹훅 전송마다 페이로드 전문 출력)와 오류 경로 `"request": m`, `data: %v` 등 9줄(3.4 B4). 선택지: (A) 이번 PR에서 `publish.go` 한 줄 삭제 + webhook-manager B4 9줄 수정, (B) 별도 이슈/PR로 분리(분리는 대표님 허락 필요), (C) 수정하지 않고 위험 수용 | **권장 A.** 영향이 작다(로그 필드 수정만, 시그니처와 동작 무변경). **`publish.go` 한 줄만으로는 웹훅 경로의 키 로그 누출이 막히지 않는다.** `publish.go`는 전송 RPC 실패 때만 출력하는 오류 경로이고, 같은 페이로드를 webhook-manager가 전송마다 Debug로 기록하는 상시 경로가 따로 있기 때문이다(이전 초안의 "BYOK 키가 이 줄로 로그에 남는 것을 막을 유일한 방법" 문장은 이 사실과 맞지 않아 정정). 웹훅 경로를 닫으려면 둘을 함께 수정해야 한다(ai-manager, webhook-manager 배포에 포함). conversation-manager의 secret/token 누출도 함께 해소된다. 공용 모듈 변경이라 대표님 확인 후 진행(CPO 제안 f: webhook-manager B4는 이견이 없으면 필수로 확정) |
| 9 | detail의 platform 모델 저장 시 저장 키 처리(제안 r). 현행은 저장 키 재전송(`ais_detail.js:422`, `views/ais/__tests__/ais_detail.test.js:435-448`)이라 키가 지워지지 않는다. sidebar는 지워진다. 경고 문구 `The saved key will be removed.`는 sidebar에서만 사실이다 | **권장 유지(제안 v)**: 이번 PR은 detail 계약을 유지하고 경고를 sidebar에만 표시한다. detail의 platform 목적지는 입력란이 비어 보이나 저장 키는 서버에 유지되며, 호출에는 쓰이지 않고 이후 다른 그룹이나 다른 서비스로 바꿀 때 가드가 다시 걸린다. detail에서도 지우려면 테스트 의미 변경이 필요해 별도 결정 |
| 10 | (정오표) 같은 그룹 안 벤더 변경(OpenAI에서 Gemini) 시 입력란에 사용자가 직접 타이핑한 키는 저장 키가 없을 때(가드 false) 유지된다(create와 동일). 화면에서 보고 입력한 값이고 저장 키 유출이 아니므로 수용된 재존이다 | CPO 권장: 수용 |

### 8.2 구현 시 확인할 항목(미실증)

- `start.go:260,302,486`의 `a`와 `start.go:726`의 `c` nil 가능성(3.4). `db.go:39,119`는 로그 직후 `c.CustomerID`를 역참조하므로 새 위험이 없음을 코드로 확인했다.
- `teamgraph/nodes/member.js:202-205`의 provider 배지(`getProviderStyle`)가 `openrouter.<id>`를 어떻게 분류하는지(3.6.1, 이번에 읽지 않음).
- webhook-manager B4: `m.RequestID` 필드 사용과 `len(data)`(`json.RawMessage`) 컴파일 확인, 기존 테스트가 해당 로그 필드를 단정하지 않는지(3.4 B4, 이번에 테스트 파일은 확인하지 않음).
- `onCloseAutoFocus` 포커스 처리의 실제 동작(3.6.2). 코드 읽기 기준이며 jsdom 테스트로 확인한다. `<details open>` 제어 장치는 제안 m으로 제거했으므로 확인 대상이 아니다.
- 서버 오류 본문 형식 `{error:{status, reason, message, ...}}`(`2026-04-24-api-error-response-codes-design.md` 4절 기준)에 따라 `engineErrorOf`의 `Details:` 뒤 JSON 파싱이 실제 응답에서 동작하는지(실제 호출로는 미확인, 3.6.4). 파싱 실패 시 폴백 문구가 있다.
- 기존 로드 effect(`sidebar.js:618-644`) 확장 시 캐시 히트 경로의 재조회와 캐시 미스 경로의 응답 재사용이 `keySeqRef`로 올바르게 직렬화되는지(3.6.6 항목 2, 단위 테스트로 확인).
- docsdev 빌드 산출물(`build/html`) 추적 여부와 갱신 절차(3.5).
- `Test_CatalogViewShape`의 나머지 단정(`rec` 카운트 등)이 Custom 항목 추가에 영향을 받는지(이번에 `:60-95` 전문은 읽지 않음).
- 구조체 `ModelEntry`의 Custom 접두 필드 이름과 `CatalogView()` 조립 방식(3.1.1).

### 8.3 위험

| 위험 | 영향 | 완화 |
|---|---|---|
| 플랫폼 키 경로 차단 누락 | 고객 호출이 플랫폼 OpenRouter/OpenAI 키로 과금 | 3중 방어, 4절 독립 테스트, 머지 게이트(7절 항목 3, 5) |
| ZDR 없음 시 무음 통화 | 고객이 원인을 모름 | 문서 안내, 실호출 후 분류/문구 확정(3.7) |
| 고객 키 로깅 누출 | 고객 OpenRouter 과금 사고 | 16곳(3.4 A, aicallhandler 3곳 포함)과 오류 경로 request 필드(B1), webhook-manager(B4) 수정(필수), 공용 `publish.go`는 8.1 항목 8, 이벤트 버스 저장은 8.1 항목 3, 신규 코드 무로깅, 6.3 게이트 2 |
| 서버가 키 발급처를 판별 불가 | 타 벤더로 키 전송(R1-h) | UI 초기화/경고만 가능, 서버 한계로 수용 |
| 고객 계정 가드레일로 인한 403/404 | 안내 부정확 | 실증 불가 범위로 문서화 |
| 구 UI + 신 API 시간차 | Custom이 일반 행으로 보임 | 서버 검증, UI 마지막 배포 |
| api-manager 미재배포 | 새 필드가 응답에서 사라짐 | 6.2 동시 배포 요건 |
| `engine_key` GET 노출 제거 이슈와 재조회 결합 | 별도 이슈가 이번 기능을 깨뜨림 | 4절, 3.6.6의 결합 조건 명시 |
| 모델 ID 변종 거부로 일부 고객 요구 미충족 | 기능 제약 | 실호출 결과에 따라 순차 허용(Q4) |
| 서버 카탈로그 방식의 테스트 완화 | 누출 가드 약화 | 3.1.5의 한정 예외(허용 필드 5개 고정)와 비 Custom 항목 가드 유지 테스트 |
| 키 이월 잔존 경로(R1-h 보완 후) | 저장된 키가 비어 있지 않은 상태의 키 방식 그룹 또는 서비스 접두 변경은 platform 원본과 같은 그룹 안의 벤더 변경을 포함해 UI에서 닫히나(3.6.5, 제안 k, s, t), CLI/API 직접 호출과 같은 서비스 접두 안에서 저장 키의 출처 벤더가 이미 달라진 경우는 막지 못해 고객 키가 타 벤더로 전송될 수 있음 | 서버 한계로 수용, 이 경로는 8.1과 별도 이슈로만 개선 가능 |
| 수용된 재존(타이핑한 키의 같은 그룹 내 벤더 변경) | 저장 키가 없는 AI에서 사용자가 타이핑한 키는 같은 그룹의 다른 벤더로 바꿔도 유지됨 | 수용(사용자 입력 값, create와 동일) |
| aicall 스냅샷 모델과 현재 AI 키의 시점 불일치 | aicall은 생성 시점의 `engine_model`을 스냅샷으로 가지고(`aicallhandler/db.go` `AIEngineModel: c.EngineModel`, 실행 시 `start.go:942`의 `LLMType(c.AIEngineModel)`) pipecat-manager는 세션 시작 때 **현재** AI의 `EngineKey`를 다시 읽는다(`pipecatcallhandler/start.go:198-205`, 팀 경로 `run.go:141`). 생성 이후 고객이 AI의 모델과 키를 함께 바꾸면(예: 직접 모델에서 Custom, 또는 반대) 구 모델에 새 키, 또는 새 모델에 구 키 조합으로 호출될 수 있다. 특히 길게 이어지는 대화형 aicall에서 창이 넓다. Custom 모델에서는 빈 키로 fail-closed이고 플랫폼 키 경로는 없으나, 다른 발급처 키가 전송되는 경우는 막지 못한다. BYOK 도입으로 openrouter.ai가 고객 키의 새 수신처가 되므로, 모델과 키를 함께 바꾼 직후의 기존 aicall 세션에서 다른 발급처 키가 openrouter.ai로 전송될 수 있다(스냅샷 한계는 Custom 빈 키에서만 fail-closed). | 이번 범위 밖(기존 구조의 동작), 서버 한계로 수용. 별도 이슈 후보(aicall이 모델과 키 발급처를 함께 스냅샷하거나 키 조회 시 모델 일치 확인) |
| platform 모델 저장 시 저장 키 처리 | sidebar는 platform 모델에서 `engine_key`를 보내지 않으므로(현행 유지) PUT 전체 교체로 저장 키가 비워지지만, detail은 저장 키를 재전송해(`ais_detail.js:422`) 키가 남는다. 남은 키는 호출에 쓰이지 않으나 키 이월 우회의 원천이었다. 이후 키 모델로 바꿀 때 새로 입력해야 한다 | 현행 동작을 유지하고 3.6.5의 가드(저장 키가 비어 있지 않으면 platform 원본도 포함)로 우회를 막는다. 수용 |
| 로컬스토리지 평문 키 캐시(수정회차 7 기록) | `LoadResource`(`provider.js:361-388`)와 `teams_create.js:51,73`은 `GET /ais` 응답을 `engine_key` 제거 없이 `ais` 캐시에 저장한다(`sidebar.js`의 목록 캐시 쓰기 `:488`와 단건 캐시 쓰기 `:636-639`는 `engine_key`를 제외). 그래서 BYOK 키가 브라우저 스토리지에 평문으로 남을 수 있다(미확인: 목록 응답이 실제로 키를 싣는지는 서버 응답 기준, 4절의 GET 노출 경로와 같은 계열) | 이번 범위 밖. 8.1 항목 3의 GET 응답 `engine_key` 제거 이슈와 함께 별도 이슈 후보로 기록(제거하면 캐시도 자연히 해소) |
| Custom 행 재선택 시 ID 재입력 불편(수정회차 7 기록) | 이미 Custom 값이 있을 때 행을 다시 고르면 타이핑한 ID를 유지하지만(3.6.2), 다른 모델로 바꿨다가 Custom 행을 다시 고르면 접두만 설정되어 ID를 다시 입력해야 한다 | 이번 범위에서 수용. 이전 Custom ID 기억은 상태가 늘어 제외, 불편 피드백이 있으면 별도 개선 |
| 키 입력란이 마스킹되지 않음(`type="text"`) | BYOK 키가 화면에 평문 노출(어깨너머 노출, 화면 공유) | 이번 범위에서 제외(제안 d), 별도 이슈로 기록 |
| 다른 핸들러의 요청 본문 로깅(mcpserver 등)과 timeline-manager 드롭 로그(`subscribehandler/main.go:173`) | 다른 비밀값 누출 가능(미확인) | 이번 범위 밖, 별도 이슈 후보. `main.go:285` 수정(B1)은 `:631`/`:642` 경로만 닫는다 |

### 8.4 다음 단계

이 문서는 디자인 리뷰 루프(최소 2회, 2회 연속 Approval, 최대 20회)에 넘긴다. 승인 후 설계 단계(구현 계획)로 진행하며, 8.1의 항목 1, 2, 8, 9는 리뷰 전에 대표님 확인을 받는 것을 권장한다(항목 9는 detail의 platform 저장 키 유지 계약이며 제안 r, v와 연결된다. 2.1의 제안 a~e와 2.2의 제안 f~v는 모두 CPO 제안이며 대표님 이견이 없으면 확정으로 읽는다. 리뷰어는 2.1과 2.2의 표 전체를 읽는다).

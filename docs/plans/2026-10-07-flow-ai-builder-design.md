# VOIP-1573 Flow AI Builder 디자인 (v4)

Jira: VOIP-1573. 상태: **디자인 리뷰 루프 진행 중.** 이력: 1회차 REQUEST_CHANGES, 2회차(A, B) 모두 REQUEST_CHANGES, 3회차(C, D) 모두 REQUEST_CHANGES. 연속 APPROVE 0.

v4는 3회차 지적을 코드로 재검증해 반영하면서, **메타데이터를 한 구조체로 통합하고 투기적 검증을 걷어내는 방향으로 단순화**했다(3회차에서 리뷰어도 메타데이터 4종이 과하다고 지적). 처리 내역은 부록 A.

## 1. 문제와 확정 결정

square-admin의 Flow 에디터(`flows_create.js` + `actiongraph/`)는 하드코딩 템플릿(`flow_templates.js`)만 제공한다. Assistant 생성 화면에는 이미 대화형 "Build with AI"(VOIP-1558, `bin-ai-manager/pkg/builderhandler`, 설계 `docs/plans/2026-10-01-conversational-assistant-builder-design.md`)가 있다. 같은 경험을 Flow 생성에도 제공한다.

확정 결정(대표님):
1. **상태 없는 동기 멀티턴.** 대화 이력은 브라우저가 보관하고 서버는 저장하지 않는다.
2. 자동 저장하지 않는다. 기존 Flow 에디터에 초안을 채우고 사용자가 확인 후 저장한다.
3. **핵심 요건: Flow 액션 목록과 옵션은 언제든 추가/변경된다. 빌더는 액션 타입 이름으로 분기하는 코드를 갖지 않고 flow-manager의 액션 정의 메타데이터에서 전부 파생된다.** 노출은 분류 선언으로 자동 결정한다(고정 허용 목록 기각).
4. 고객 리소스 참조(queue_id, assistance_id 등)는 비워 두고 경고로 안내한다. 리소스 목록 주입은 v1 범위 밖이다.

## 2. 변경에 강한 메타데이터 설계 (핵심)

### 2.1 원칙

빌더, 변환기, 검증기는 액션 타입 이름을 코드에 쓰지 않는다. 읽는 것은 `bin-flow-manager/models/action`의 **단일 메타데이터 레지스트리**(2.2)와 옵션 필드 태그(2.3)뿐이다. 선언이 빠지면 CI가 실패한다. 액션을 추가하는 개발자는 `OptionStructByType`, 메타 레지스트리, 필요한 필드 태그를 같은 PR에서 선언한다.

### 2.2 단일 레지스트리 `MetaByType`

```go
type Meta struct {
    Exposure Exposure // core | sensitive | internal
    Flow     FlowKind // continue | terminate | jump
}
var MetaByType = map[Type]Meta{ ... } // option_registry.go 옆, 같은 패턴
```

`TestMetaCoversAllTypes` 하나가 `TypeListAll`의 모든 타입 선언을 강제한다(기존 `TestActionCatalogMatchesTypeListAll`과 같은 drift-lock 패턴).

**Exposure** (비용/외부효과 한 축)
- `core`: 추가 비용과 외부 효과 없음(음성 안내, 분기, 대기, 변수 설정, 종료 등).
- `sensitive`: 비용 발생 또는 외부로 나감(아웃바운드 `connect`, 메시지/이메일/webhook/fetch, LLM 비용이 드는 `ai_*`).
- `internal`: 빌더에 노출하지 않음(플랫폼 내부용, 또는 **에디터가 의미를 보존하지 못하는 타입**. 예: `goto`는 에디터가 `next_id`를 무시하는데 루프 소진 후 `next_id`/배열 인접으로 낙하하므로 v1에서 `internal`).
- `core`와 `sensitive`를 노출하고 `internal`은 노출하지 않는다. 전역 on/off 플래그나 고객 단위 저장 설정은 두지 않는다. 최종 분류는 구현 시 각 액션의 실행 핸들러를 읽어 확정하고 PR 리뷰 대상이다.

**FlowKind** (실행이 다음으로 이어지는 방식, 코드 확인 결과)
- `continue`: `next_id`가 있으면 그 액션, 없으면 **배열의 다음 원소**, 배열 끝이면 종료(`stackmaphandler.GetNextAction`). 대부분의 타입. 조건 액션(`condition_*`)도 true는 이 경로를 따르고 false만 `FalseTargetID`로 간다.
- `terminate`: 실행 뒤 흐름이 이어지지 않는다(`stop`은 `ActionFinish` 푸시, `hangup`은 통화 종료). `next_id`가 있어도 무시한다.
- `jump`: 대상이 옵션의 `ref:"action"` 필드로만 정해지고 `next_id`나 배열 인접으로 낙하하지 않는다(`branch`는 `ForwardActionID`를 강제하고 대상이 비면 `GetAction` 오류로 흐름 중단).

값은 구현 시 실행 핸들러를 읽어 확정한다. 후보 판정은 코드로 확인했다: `stop`, `hangup` = terminate / `branch` = jump / 그 외 확인한 타입(`connect`, `queue_join`, `ai_summary`, `fetch_flow`, `fetch`, `block`, `talk`, `ai_talk`) = continue.

### 2.3 옵션 필드 참조 태그 (`ref`)

현재 액션 간 연결과 고객 리소스 참조가 모두 `uuid.UUID`라 구분이 안 된다. `option.go` 전수 확인 결과 UUID 필드는 다음과 같다(구현 1단계에서 grep으로 재확인).

| 필드 | 태그 |
|---|---|
| `OptionGoto.TargetID`, `OptionBranch.DefaultTargetID`, `OptionBranch.TargetIDs`(map 값), 조건 액션의 `FalseTargetID` | `ref:"action"` (`DefaultTargetID`는 `ref:"action,optional"`) |
| `AssistanceID`, `AIID`(deprecated), `FlowID`, `OnEndFlowID`, `ConferenceID`, `ConfbridgeID`, `ConversationID` 등 | `ref:"resource:<kind>"` |
| `ai_summary.reference_id`(`OptionAISummary`의 사용자 입력, `reference_type`과 쌍), `hangup.reference_id`("이 통화 id와 같은 사유로 끊는다") | `ref:"resource:<kind>,optional"` |

- 3종 구분 중 `runtime`은 **없다.** 3회차 코드 확인에서 `reference_id` 두 곳 모두 사용자 입력이라 `resource`가 맞았다(`actionHandleAISummary`가 옵션값을 그대로 전달).
- `ref:"action"`은 변환기가 label→UUID로 치환한다. `ref:"resource:*"`는 `IDEmpty`로 두고, optional이 아니면 "select" 경고를 만든다. `<kind>`는 경고 문구용이다.
- 순회 형태: 스칼라 `uuid.UUID`, `map[string]uuid.UUID`(값). `[]uuid.UUID`/`*uuid.UUID`는 현재 0건이나 같은 규칙으로 처리한다.
- **중첩 구조체/슬라이스 안의 UUID**(`OptionCall.Actions []Action`, `OptionEmailSend.Attachments []Attachment`의 `ReferenceID`)는 v1에서 지원하지 않는다. 리플렉션으로 이를 가진 타입을 감지해 **카탈로그에서 구조적으로 제외**하고, 제외된 타입 목록을 `TestBuilderExcludedTypes`의 기대값(현재 `call`, `email_send`)으로 고정해 변경이 눈에 띄게 한다. 타입 이름은 코드가 아니라 테스트 기대값에만 있다.
- 최상위 UUID 필드에 태그가 없으면 `TestEveryUUIDFieldIsTagged`가 실패한다.
- **필수 옵션은 새 `required` 태그를 만들지 않는다.** 이미 `bin-ai-manager/pkg/actioncatalog`에 필드별 `Required`가 있고 옵션 구조체와의 필드 동기화 테스트(`TestActionCatalogFieldsMatchOptionStructs`)가 있다. 검증기는 그 카탈로그의 `Required`를 읽는다(진실 원천 하나). 단, `Required`가 조건부 필수(`goto.loop_count` 등)를 충분히 표현하는지는 구현 시 확인한다.

### 2.4 few-shot 예시 보호

few-shot에 쓰는 타입 상수가 `TypeListAll`에 있고 `core`이며 프런트 지원 집합에 있음을 테스트로 고정한다.

### 2.5 프런트-백엔드 지원 교집합

실측: 백엔드 `TypeListAll` 43개, 프런트 `nodeTypes` 39개(가상 `start` 포함, 액션 38개). 백엔드에만 있는 타입은 `block`, `condition_call_digits`, `condition_call_status`, `condition_variable`, `mute`.

- `ChatRequest.supported_action_types: []string`은 **필수**다(신규 엔드포인트라 구버전 클라이언트가 없다). 생략, 빈 배열, 항목 수 100 초과, 항목 길이 64 초과는 `INVALID_ARGUMENT`. 프런트는 `Object.keys(nodeTypes)`에서 `start`를 뺀 값을 그대로 보낸다(정적 목록이며 별도의 런타임 왕복 검사는 두지 않는다).
- 서버는 `MetaByType`에 없는 값을 무시하고, **노출 가능(core, sensitive) ∩ 요청 집합 ∩ 중첩 UUID로 제외되지 않은 타입**만 LLM 카탈로그와 검증에 사용한다. 교집합이 비면 `INVALID_ARGUMENT`.
- **에디터 충실도**(에디터가 저장 시 의미를 보존하는가)는 런타임 검사가 아니라 두 곳으로 관리한다: (가) 보존하지 못하는 타입은 `Exposure=internal`로 선언한다(2.2). (나) 에디터의 분기 계열(`branch`, `condition_datetime`)에 대해 `initNodes → initEdges → getActions` 순서(`initEdges`는 노드를 만들지 않으므로 `initNodes`가 선행되어야 한다)로 `ref:"action"` 필드와 `next_id`가 복원됨을 확인하는 **고정 단위 테스트**를 square-admin에 둔다. 현재 이런 타입이 둘뿐이므로 이 테스트는 손으로 쓴 픽스처로 충분하다. 새 분기 타입이 에디터에 추가되면 같은 테스트에 픽스처를 더한다(PR 체크리스트).
- 조건 액션의 true 경로는 `next_id`(없으면 배열 인접)이고 에디터가 `condition_datetime`의 `next_id`를 보존함을 코드로 확인했다(`store.js`).

## 3. 와이어 모델

### 3.0 실제 Flow 계약 (코드 확인)

- Flow는 평탄한 `[]Action`(`{id, type, option, next_id}`)이다. 템플릿 `positions`는 `{<action_id>: {x, y}}`.
- 시작 노드는 `actions[0]`(`GetNextAction`이 `IDStart`에서 `Actions[0]` 반환, 에디터 `initEdges`는 `store.js`에서 `i===0`을 시작 대상으로 사용).
- 다음 액션 해석은 2.2의 FlowKind를 따른다. `next_id`가 존재하지 않는 id이면 종료.
- `branch`: 값이 `TargetIDs`에 없으면 `DefaultTargetID`, 대상이 `IDEmpty`/없음이면 `GetAction` 오류로 흐름 중단.
- `branch` 기본 변수는 `voipbin.call.digits`(`OptionBranchVariableDefault`).

### 3.1 LLM이 생성하는 기호 그래프

평탄한 노드 배열이며 첫 노드가 시작이다. UUID, 좌표, 별도 연결 목록은 만들지 않는다.

```json
{
  "nodes": [
    {"label": "greet", "type": "talk", "option": {"text": "Press 1 for sales."}, "next": "ask"},
    {"label": "ask", "type": "digits_receive", "option": {"duration": 5000, "length": 1}, "next": "menu"},
    {"label": "menu", "type": "branch", "option": {"target_ids": {"1": "sales"}, "default_target_id": "bye"}},
    {"label": "sales", "type": "queue_join", "option": {"queue_id": null}, "next": "bye"},
    {"label": "bye", "type": "hangup", "option": {}}
  ]
}
```

(예시의 타입과 옵션 이름은 구현 시 `actioncatalog` 실제 옵션과 대조해 few-shot 상수로 고정한다.)

- `next`는 후속 label이며 생략하면 후속 없음. `jump` 종류 타입(`branch`)은 `next`를 쓰지 않는다. 쓰면 변환기가 무시하고 `next_ignored` 경고를 기록한다(판정은 메타의 FlowKind이며 타입 이름이 아니다).
- 분기 대상 등 `ref:"action"` 값은 label 문자열 또는 생략. `ref:"resource:*"`는 null.

### 3.2 코드가 확정하는 변환 (서버, 순서 고정)

1. **label 정규화**: 중복 label은 첫 노드를 유지하고 이후 노드를 제거하며 `duplicate_label` 경고를 기록한다(이후 참조는 첫 노드로 해석). 클라이언트가 보낸 `labels`(5절)는 신뢰하지 않는다. 중복, 존재하지 않는 id, 자동 부여 label(`n1...`)과의 충돌은 버리고 재부여한다.
2. **타입 필터**: 노출 집합에 없는 타입의 노드를 삭제하고 `unsupported_action: <label>` 경고를 기록한다. 삭제 노드를 가리키던 참조(`next`, `ref:"action"` 값)는 삭제 노드의 `next` 체인을 따라 이어 붙인다(방문 집합으로 순환 차단, 이을 후속이 없으면 비움). 삭제 노드가 분기였다면 의미가 사라지므로 경고에 어느 참조가 어떻게 바뀌었는지(`rewired: <참조 label> -> <새 대상 또는 none>`)를 포함한다. `actions[0]`이 삭제되면 그 후속이 새 시작이 된다.
3. 남은 노드에 새 UUID를 1:1 부여한다.
4. `next` → `next_id`(없으면 `IDEmpty`). `ref:"action"` 값(스칼라, 맵 값)을 label→UUID로 치환한다. 알 수 없는 label은 `IDEmpty`로 비우고 `invalid_label_ref: <label>`을 기록한다(노드는 유지. 에디터에서 보완 가능).
5. `ref:"resource:*"`는 `IDEmpty`로 두고, optional이 아니면 `select_resource: <label>.<field>`를 기록한다. null/생략은 모두 `IDEmpty`.
6. **배열 순서**: 시작 노드부터 BFS 순으로 배열하되 **"열린 끝"(FlowKind가 `continue`이고 후속이 없는 노드)을 배열의 뒤쪽으로 미룬다**(낙하가 곧 종료가 되도록 마지막 원소로 보낸다). 열린 끝이 둘 이상이면 마지막이 아닌 노드마다 `open_end: <label>`을 기록한다(낙하로 의도치 않은 노드가 이어지기 때문이며 사용자가 에디터에서 연결을 완성해야 한다). 이 단계는 2~5가 끝난 최종 그래프를 기준으로 한다.
7. **레이아웃**: 시작 `x=0, y=100`, 깊이마다 `y`를 500 증가, 같은 깊이는 `x`를 450 간격(템플릿과 같은 간격). 도달 불가 노드는 맨 오른쪽 열. 가상 `start` 좌표는 건드리지 않는다.
8. 필터 후 노드가 0개이면 `draft`를 생략하고 `message`에 이유를 쓰며 `empty_draft`를 기록한다.
9. 출력: `actions`(`[]Action`, `actions[0]`이 시작), `positions`, `labels`(`{id: label}`). `actions`와 `positions`는 에디터의 `setInitialActions`/`setInitialPositions`에 변환 없이 들어간다.

### 3.3 그래프 구조 검증기 (v1 범위)

`builderhandler`의 검증기는 타입 이름을 모른다. 입력은 `[]Action`과 메타(`FlowKind`, `ref`)와 `actioncatalog`의 `Required`뿐이다. 후속 참조의 정의는 FlowKind별로 다르다: continue = `next_id` 또는 배열 인접 원소(+ `ref:"action"` 값), terminate = 없음, jump = `ref:"action"` 값만. 모든 결과는 경고다(초안은 항상 응답한다).

v1 검사 항목(기계적이고 비용이 낮은 것만):
1. 끊어진 참조(존재하지 않는 id).
2. 비어 있는 비optional `ref:"action"` 필드(`branch`에서는 실행 시 흐름이 중단되는 오류 경로) → `empty_action_ref`.
3. 도달 불가 노드(`actions[0]`에서 후속 참조를 따라 닿지 않음) → `unreachable: <label>`.
4. 열린 끝 낙하(3.2의 6번)와 리소스 참조 비움 목록.
5. 필수 옵션 누락(`actioncatalog`의 `Required`).

**출구 없는 루프(강연결 성분) 탐지는 v1에서 뺀다.** 실행기에 `maxNextActionLoopCount` 안전장치가 있고 평가에서 실제 문제가 확인된 뒤 추가한다(Open Question 9).

## 4. 시스템 프롬프트 구성

Assistant Builder의 구성(역할, 핵심 차원, 인터뷰 행동 규칙, few-shot, 응답 JSON)을 같은 방식으로 따른다. 차이만 적는다.

1. 역할: VoIPBin Flow를 설계하는 인터뷰어이자 컨설턴트. 핵심 차원: 트리거 채널, 성공 기준, 분기 조건, 실패/예외 시 동작.
2. **능력 카탈로그는 매 요청마다 코드가 생성한다**(2.5의 교집합 + 각 타입의 Exposure/FlowKind). 설명 문구는 `actioncatalog`의 `actionCatalogEntry`(설명 필드, 현재 이름은 구현 시 확인)를 가져온다.
3. 행동 규칙(전역 플래그가 아니다): `sensitive` 타입은 사용자가 대화에서 해당 동작을 말했을 때만 초안에 넣는다. 모든 경로는 `terminate` 타입으로 끝내거나 열린 끝은 하나만 둔다. `jump` 타입에는 `next`를 쓰지 않는다. 이 규칙은 LLM이 지키는 것이므로 코드가 사실을 확정하는 것은 `sensitive_nodes`의 구조화 표시와 검증기 경고까지다.
4. few-shot 2개(2.4의 상수로 보호), 초안 형식은 3.1.
5. `message`와 경고는 마크다운 없는 평문. 사용자 화면 문구는 영어.
6. **초안(`current_draft`)의 `option` 값은 매 턴 외부 LLM으로 전송된다.** Flow 패널 안내문에도 Assistant용과 같은 "do not include secrets"(웹훅 URL/헤더 등) 문구를 둔다.

## 5. REST API

| Method | Path | 설명 |
|---|---|---|
| POST | `/flow_builder/chat` | `ChatRequest{messages, current_draft, supported_action_types}` → `ChatResponse{message, draft, draft_warnings, sensitive_nodes, assumptions}` |
| GET | `/flow_builder/status` | `{available, max_messages, max_message_chars}` |

- `draft = {actions, positions, labels}`. 클라이언트는 `current_draft`로 그대로 되돌려 보내고, 서버는 `labels`와 `actions`로 기호 그래프를 매 턴 결정적으로 재구성한다(UUID는 LLM에 보이지 않는다). BuilderPanel은 에디터 진입 전 패널이므로 사용자가 에디터에서 고친 그래프를 되돌려 보내는 경로는 없다. `positions`는 매번 재계산된다.
- `draft_warnings`는 기존 Assistant Builder 패턴을 따른다: `key` 또는 `key: detail` 형식의 영어 키 문자열 배열이며 클라이언트가 키로 문구를 고른다. 키 목록: `duplicate_label`, `unsupported_action`, `rewired`, `invalid_label_ref`, `select_resource`, `open_end`, `empty_action_ref`, `unreachable`, `next_ignored`, `missing_required`, `empty_draft`, `sensitive_included`.
- `sensitive_nodes`: 노출된 `sensitive` 노드의 id 배열(신규 응답 필드, `omitempty`, OpenAPI `ChatResponse`에 추가). 프런트가 노드를 강조한다.
- label 연속성은 보장되지 않는다(best effort). 저장 전이라 영향이 없다.
- **와이어 상한**: 노드 수 `MaxFlowNodes = 60`(초기값, 미측정), `current_draft.actions` 개수 <= 60, `positions`/`labels` 항목 수는 `actions` 이하, 노드당 `option`은 직렬화 크기 상한(구현 시 정하고 `ValidateRequest`에 포함), `option` 중첩 깊이 상한. 메시지 수/길이는 Assistant Builder 상수를 공유한다. 검증은 `bin-ai-manager/models` 아래 flow builder 모델 패키지에 두어 api-manager와 ai-manager가 같은 `ValidateRequest`를 쓴다(기존 `models/builder` 패턴). **요청 본문 상한은 Assistant의 160KiB를 그대로 쓰지 않고** 최악 크기(60노드 x 옵션 상한 + 대화 이력 한도)를 계산해 정한다.

### 5.1 횡단 관심사 (실제 ai_builder 구현을 읽고 확인한 항목)

- **RBAC**: `canUseBuilder`와 같은 규칙(로그인한 Agent + customer admin 또는 manager). `status`는 권한이 없으면 200과 `available:false`. Flow 생성 권한(`servicehandler/flow.go`: Admin|Manager)과 일치한다.
- **처리 순서**: key → ValidateRequest → semaphore → 일일 한도 → LLM. 파싱 실패도 한도를 소비한다(Assistant와 동일).
- **일일 한도 카운터**: `cachehandler.BuilderChatCountIncr`는 현재 고정 키 하나다. Flow용을 분리하려면 `CacheHandler` 인터페이스와 mock에 메서드를 추가하거나 `kind` 인자를 도입해야 한다(8절에 반영). 한도 값은 같은 기본값. 분리가 필요한지는 Open Question 7이다.
- **세마포어와 설정**: `sem`과 `DailyLimit`은 `builderHandler` 인스턴스 필드(`builderhandler/main.go`)다. Flow 핸들러를 같은 인스턴스에 두거나 `sem`을 주입한다. `cmd/ai-manager/builder_wiring.go`, config(`ai_builder_*`)를 **그대로 공유**하며 새 플래그를 만들지 않는다.
- **출력 토큰**: `MaxOutputTokens` 기본 4096은 60노드 JSON 초안에 부족할 수 있다. Flow용 값은 `Options`의 매개변수(플래그가 아니다)로 두고 평가 하네스 실측으로 정한다. LLM 타임아웃(40초, 검증 <= 50)도 실측 후 재확인한다.
- **로깅 금지**: 본문, `current_draft`, 파싱 오류, 실패 오류 텍스트를 로그에 남기지 않는다(`option`에 민감값이 올 수 있다). 요청 로거 이전 라우팅, 오류를 응답으로만 변환, panic 복구 시 본문 제외를 Assistant와 같은 방식으로 적용한다. **`listenhandler`의 `isBuilderRoute`는 URI 완전 일치**이므로 Flow URI를 반드시 추가해야 이 보장이 적용된다. 센티널 문자열 테스트로 검증한다.
- **타임아웃 순서**: LLM < RPC < 클라이언트. `BUILDER_TIMEOUT` 등 오류 reason 매핑과 `ERROR_COPY`는 재사용하되 문구에서 "assistant" 고정 표현을 일반화한다.
- **메트릭, 서킷브레이커**: 기존 `ai_manager_builder_chat_total{result}`, `api_manager_builder_timeout_total`에 `kind`(assistant, flow) 라벨을 추가해 두 기능이 대시보드에서 섞이지 않게 한다. 서킷브레이커 리소스 키는 Flow용(`ai/flow_builder/chat`)을 새로 둔다.
- **문서/인터페이스**: `bin-api-manager/docs/routing.md`, `bin-ai-manager/docs/operations.md`, `servicehandler/main.go` 인터페이스와 mock 갱신.

## 6. 프런트엔드 통합

- `flows_create.js`의 템플릿 다이얼로그 옆에 "Build with AI" 진입점. 패널은 다이얼로그 내부에서 전환되는 Assistant 패턴(`ais_create.js`)과 같게 둔다. 모바일 레이아웃은 구현 시 기존 패턴을 따른다.
- **BuilderPanel/스토어 공용화(실제 결합도 반영)**: 현재 `builderApi.js`는 `ai_builder/*` 하드코딩과 `ERROR_COPY`, `builderStore.js`는 모듈 로드 시 `restore()`, `registerCacheClearHook(reset)`, `BUILDER_SESSION_KEY`(`resourceCacheStore`)를 가지며, `BuilderPanel.js`는 `AVAILABLE_TOOLS`, Assistant 전용 안내문, `TOOLS_NOTE`, `draftOk`/`isValidCopy`(Assistant 초안 모양 고정)를 가진다. v1 작업:
  - 스토어를 팩토리로 바꾸고 엔드포인트, 세션 키, 초안 검증 함수(`isValidDraft`), 전송 페이로드 확장(`supported_action_types`), 응답 필드(`sensitive_nodes`)를 주입한다.
  - 팩토리가 만든 스토어는 **`registerCacheClearHook`에 등록**하고 새 세션 키를 `resourceCacheStore`에 추가해 로그아웃과 delegate 전환 시 지워지게 한다.
  - 패널 안내문, `TOOLS_NOTE`, 초안 렌더를 슬롯으로 주입한다.
  - 두 대화가 서로 덮어쓰지 않도록 세션 키를 분리한다. sessionStorage에는 Assistant 초안(`init_prompt`)과 같은 수준으로 Flow 초안이 저장된다(안내문의 secrets 경고로 갈음).
  - 기존 `ais/builder/__tests__`의 이동과 회귀 검증을 포함한다(8절에 파일 단위 목록으로 구현 시 작성). 공용화가 과하다고 판단되면 Flow 전용 얇은 사본이 대안이며 착수 시 비용 비교를 제시한다.
- **초안 적용**: `setInitialActions(draft.actions)`, `setInitialPositions(draft.positions)`와 `setGraphKey(k => k + 1)`로 재마운트한다(`ActionGraph`는 `!isInitialized`일 때 한 번만 초기화). 이미 편집 중이면 덮어쓰기 전에 `ConfirmDialog`를 띄운다. "편집 중" 판정은 현재 그래프 노드 수가 0보다 크면 편집 중으로 본다(dirty 추적을 새로 만들지 않는다). 이름/설명 입력은 건드리지 않는다.
- **방어**: `actions`에 `nodeTypes`에 없는 타입이 있으면 주입을 중단하고 영문 안내를 표시한다(서버가 이어 붙이기를 담당하므로 프런트는 고치지 않고 거부).

## 7. sensitive 노출 정책

v1(코드 상수 결정이며 설정 플래그가 아니다): `sensitive`를 노출하고, 초안에 포함되면 `sensitive_nodes`에 노드 id를 담고 `draft_warnings`에 `sensitive_included: <label list>`를 추가한다. 프롬프트 규칙은 사용자가 말한 동작만 넣게 한다(LLM 준수 사항이며 코드가 보증하는 것은 표시까지다). 비노출로 정하면 콜백, 아웃바운드 캠페인처럼 핵심 시나리오를 만들 수 없으므로 노출을 권고한다. 결정은 대표님 몫이다(Open Question 1).

실행 시점 게이트: 저장 전에는 Flow가 없으므로 초안은 실행되지 않는다. 저장된 Flow 실행 시 비용/권한 게이트는 하위 서비스 책임이다. `fetch`/`webhook_send`의 URL 검증은 구현 시작 전 코드를 읽고 결과를 PR 본문에 적는다(Open Question 8).

## 8. 영향 서비스

| 서비스 | 변경 |
|---|---|
| bin-flow-manager | `models/action`: `Meta`/`MetaByType`, `ref` 태그, drift-lock 테스트 3종(`TestMetaCoversAllTypes`, `TestEveryUUIDFieldIsTagged`, `TestBuilderExcludedTypes`) |
| bin-ai-manager | `pkg/builderhandler`에 flow 전용 chat 핸들러, 변환기, 검증기, 프롬프트. `models`에 flow builder 요청/응답/검증. `listenhandler` 라우트와 `isBuilderRoute`, `cachehandler`(카운터 kind) + mock, `cmd/ai-manager/builder_wiring.go`, 평가 하네스, `docs/operations.md` |
| bin-common-handler | `requesthandler`에 flow builder RPC(`ai_builder.go` 패턴, 서킷 키 분리) |
| bin-api-manager | `server/flow_builder.go`, `servicehandler/flow_builder.go`, `servicehandler/main.go` 인터페이스와 mock, 상태 캐시, 메트릭, `docs/routing.md` |
| bin-openapi-manager | `paths/flow_builder/chat.yaml`, `status.yaml`, `openapi.yaml` 라우트 등록, `gen.go` 재생성 |
| square-admin | 스토어 팩토리화, 패널 슬롯화, 테스트 이동, flows_create.js 진입점, 에디터 분기 계열 단위 테스트 |

## 9. 평가 (수용 기준)

Assistant Builder의 `builder_eval` 하네스를 차용해 Flow 시나리오로 교체한다.
1. 모호한 요청 → 트리거 채널과 실패 처리를 좁히는 질문.
2. 전문가가 분기 조건을 모두 제공 → 열린 갈림길이 없으면 한 턴 초안.
3. **결정론 검사(LLM 판정 불필요)**: 끊어진 참조 0, 도달 불가 0, `open_end` 경고와 실제 열린 끝 1:1, `select_resource` 경고와 비어 있는 리소스 필드 1:1.
4. **메타데이터 변이 테스트**: `TypeListAll`에 새 타입을 추가하고 `MetaByType`/태그를 빼면 CI가 실패하는지 확인한다.
5. 프런트 분기 계열 왕복 단위 테스트(2.5).
6. Flow 초안 크기 실측(출력 토큰, 본문 크기, 지연)을 5.1의 값 결정에 사용한다.

## 10. 보안 / 블라스트 레디어스

- 서버 무저장은 5.1의 로깅 금지와 구조적 보장으로 지킨다.
- 초안은 실행되지 않는다(7절).
- 외부 LLM에 `option` 값이 전송되는 점을 패널 안내문으로 고지한다(4절 6항).

## 11. Open Questions

| # | 질문 | 제안 | 결정 |
|---|---|---|---|
| 1 | `sensitive` 노출(코드 상수 결정) | 노출 + 구조화 표시와 경고. 비노출 시 아웃바운드 시나리오 불가 | 대표님 |
| 2 | 리소스 ID 목록 주입 | v1 비움 + 경고, 주입은 후속 | 대표님 |
| 3 | `ref` 태깅 범위 | 해결: 43개 전수 태깅 | 확정 제안 |
| 4 | 한도/오류코드 공유 | `models` 공유, 신규 flow builder 모델 패키지 | 대표님 |
| 5 | `ref` 태그를 `describe_action` 카탈로그에 반영 | 범위 밖 | 대표님 |
| 6 | 노출 타입 집합 | 고정 목록 없이 `Exposure` ∩ 프런트 목록 ∩ 구조 제외 | 해결 |
| 7 | 일일 한도 카운터를 Assistant와 분리할지 | 분리 권고(한쪽이 다른 쪽을 소진하지 않게). 비용은 cachehandler 변경 | 대표님 |
| 8 | `fetch`/`webhook_send` URL 검증 | 구현 전 코드 확인 후 PR에 기록 | CPO |
| 9 | 출구 없는 루프 탐지 | 평가에서 실제 문제 확인 시 후속 | 대표님 |

## 12. 구현 순서

1. `bin-flow-manager/models/action`: `Meta` 레지스트리, `ref` 태그, drift-lock 테스트. 모든 타입의 실행 핸들러를 읽어 Exposure/FlowKind 확정.
2. `bin-ai-manager`: 검증기(순수 함수, 테스트 우선) → 변환기 → 프롬프트와 카탈로그 생성 → chat 핸들러와 wiring → 평가 하네스(출력 토큰 실측).
3. `bin-common-handler`, `bin-api-manager`, `bin-openapi-manager`: RPC와 REST.
4. `square-admin`: 분기 계열 왕복 테스트 → 스토어/패널 공용화 → 진입점.
5. 평가 실행(9절) → 코드 리뷰 루프.

## 부록 A. 처리 내역

### 3회차 (C, D 모두 REQUEST_CHANGES), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| C-C1 `NEXT_IGNORED`가 `ref:"action"` 유무로 조건 액션 true 경로를 지움 | **사실.** 조건 액션은 false만 `FalseTargetID`, true는 `next_id`. 에디터도 `condition_datetime`의 `next_id`를 보존 | `next` 무시 판정을 `ref` 유무에서 **FlowKind=jump**로 변경(2.2, 3.1) |
| C-H1/D-H1 왕복 테스트 기준이 branch/goto를 스스로 탈락시킴, 프런트는 Go 태그를 모름 | **사실.** | 런타임 왕복 검사 폐기. 충실도는 `Exposure=internal` 선언 + 분기 계열 고정 단위 테스트로 관리(2.5). `goto`는 `internal` |
| C-H2 낙하 모델이 branch/goto/condition에서 틀림 | **사실.** `branch`는 낙하하지 않음, `goto`는 `loop_count <= 0`이면 다음으로 진행 | FlowKind 3종으로 모델링(2.2, 3.3), `goto` 제외 |
| C-H3 `email_send` 첨부(`Attachments []Attachment`)의 `ReferenceID` 누락 | **사실.** `models/email/main.go`에 슬라이스 존재 | 중첩 UUID 타입은 구조적 제외 + 기대값 테스트(2.3) |
| C-H4 `Terminates` 값 미확정 | 후보를 코드로 확정(`stop`, `hangup`) | 2.2에 후보 판정 기재, 구현 시 최종 확정 |
| C-M1 `ai_summary.reference_id`를 runtime으로 분류한 근거 약함 | **사실.** 사용자 입력 옵션(`reference_type`과 쌍) | `runtime` 분류 삭제, `resource ... optional`(2.3) |
| C-M2 3.2의 6번 대상 그래프 시점 | 타당 | 2~5 이후의 최종 그래프 기준 명시, 열린 끝은 뒤로 미룸 |
| C-M4/D-M1/M2 메타 4종, required 전수 도입, 과함 | 타당 | `Meta` 단일 레지스트리, `required` 태그 폐기하고 `actioncatalog.Required` 재사용, 루프 탐지 v1 제외 |
| C-L1 예시 변수명 | **사실.** 기본 변수는 `voipbin.call.digits` | 3.0, 3.1 수정 |
| D-H2 제외 목록의 출처 | 타당 | 정적 목록으로 단순화(2.5) |
| D-H3 5.1 누락(cachehandler, wiring, config, MaxOutputTokens, 메트릭 라벨, 서킷 키, listenhandler 라우트, 문서) | **사실.** `isBuilderRoute` URI 완전 일치, 카운터 키 고정 확인 | 5.1, 8절 전면 보강 |
| D-H4 스토어 팩토리 누락 4건 | 타당 | 6절 |
| D-M3 sensitive 서술 | 타당 | 7절 "코드 상수 결정", LLM 준수와 코드 보증 구분 |
| D-M4 경고 형식/언어 | **사실.** 기존은 `key` 또는 `key: detail` | 5절 키 목록, 영어 |
| D-M5 와이어 상한 | 타당 | 5절 |
| D-L1~L4 | 타당 | 4절 6항, 6절, 2.5(상한 100) |

### 1, 2회차 요약
평탄 `[]Action` + `positions` 와이어, `actions[0]` 시작 노드, 배열 낙하 반영, 프런트 실측 수치 39/43, graphKey 재마운트, 노드 수 상한, 횡단 관심사 보강 등을 v2, v3에서 반영했다. 상세 이력은 git 이력에 있다.

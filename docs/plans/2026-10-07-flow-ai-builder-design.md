# VOIP-1573 Flow AI Builder 디자인 (v9)

Jira: VOIP-1573. 상태: **디자인 리뷰 루프 종료. 구현 착수(2026-10-07, 대표님 지시).** OQ1=노출, OQ7=분리로 확정. 이력: 1~5회차는 REQUEST_CHANGES가 섞였고(Critical는 1~3회차에서 각각 해소), 6회차는 I APPROVE와 J REQUEST_CHANGES(리셋)였다. 8회차의 Low/Nit는 v9에서 문서 정정으로 반영했다(리셋하지 않음).

v9는 8회차의 Low/Nit를 반영했다(부록 A). 이전 회차 변경 이력은 부록 A에 있다.

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
- `sensitive`: 비용 발생 또는 외부로 나감(아웃바운드 `connect`, 메시지/이메일/webhook/fetch, LLM 비용이 드는 `ai_talk`, `ai_task`).
- `internal`: 빌더에 노출하지 않음(플랫폼 내부용, 또는 **에디터가 의미를 보존하지 못하는 타입**. 예: `goto`는 에디터가 `next_id`를 무시하는데 루프 소진 후 `next_id`/배열 인접으로 낙하하므로 v1에서 `internal`).
- `core`와 `sensitive`를 노출하고 `internal`은 노출하지 않는다. 전역 on/off 플래그나 고객 단위 저장 설정은 두지 않는다. 최종 분류는 구현 시 각 액션의 실행 핸들러를 읽어 확정하고 PR 리뷰 대상이다.

**FlowKind** (실행이 다음으로 이어지는 방식, 코드 확인 결과)
- `continue`: `next_id`가 있으면 그 액션, 없으면 **배열의 다음 원소**, 배열 끝이면 종료(`stackmaphandler.GetNextAction`). 대부분의 타입. 조건 액션(`condition_*`)도 true는 이 경로를 따르고 false만 `FalseTargetID`로 간다.
- `terminate`: **해당 액션이 실행되는 미디어에서** 실행 뒤 흐름이 이어지지 않는다(`stop`은 `ActionFinish` 푸시, `hangup`은 통화 종료). `next_id`가 있어도 무시한다.
- `jump`: 대상이 옵션의 `ref:"action"` 필드로만 정해지고 `next_id`나 배열 인접으로 낙하하지 않는다(`branch`는 `ForwardActionID`를 강제하고 대상이 비면 `GetAction` 오류로 흐름 중단).

값은 구현 시 실행 핸들러를 읽어 확정하며 43개 타입의 분류 초안은 부록 B에 있다. 코드로 확인한 판정: `stop`, `hangup` = terminate / `branch` = jump / 조건 액션, `connect`, `queue_join`, `ai_summary`, `fetch_flow`, `fetch`, `block`(외부 해제까지 대기하지만 해제 후 이어짐), `talk`, `ai_talk` = continue.

**모델 밖의 타입**: `goto`는 `loop_count > 0`이면 대상으로 점프하고 `<= 0`이면 다음으로 진행하는 조건부 점프라 세 종류 어디에도 맞지 않는다. 새 종류를 만들지 않고 v1에서 `internal`로 둔다(에디터도 `next_id`를 보존하지 못한다). 결과로 재시도 루프("실패하면 다시 묻기")는 v1 빌더로 만들 수 없다. 실측으로 필요성이 확인되면 새 FlowKind를 도입한다.

**적용 미디어**: 실행기는 액션 타입이 activeflow의 미디어와 맞지 않으면 그 액션을 **건너뛴다**(`execute.go` `verifyActionType`, 기존 레지스트리 `action.MapRequiredMediasByType`: RT = 통화, NRT = AI/API/대화/캠페인/webchat 등, ANY). 예: `hangup`은 RT 전용이라 대화 흐름에서는 건너뛰고 낙하한다. 따라서 `terminate`, `continue` 판정은 적용 미디어에서만 성립한다. 새 필드를 만들지 않고 이 기존 레지스트리를 카탈로그에 그대로 노출한다(4절, 3.3).

### 2.3 옵션 필드 참조 태그 (`ref`)

현재 액션 간 연결과 고객 리소스 참조가 모두 `uuid.UUID`라 구분이 안 된다. `option.go` 전수 확인 결과 UUID 필드는 다음과 같다(구현 1단계에서 grep으로 재확인).

| 필드 | 태그 |
|---|---|
| `OptionGoto.TargetID`, `OptionBranch.DefaultTargetID`, `OptionBranch.TargetIDs`(map 값), 조건 액션의 `FalseTargetID` | `ref:"action"` |
| `AssistanceID`, `AIID`(deprecated), `FlowID`, `OnEndFlowID`(`ai_summary`, `recording_start`, `transcribe_start`, `transcribe_recording`), `ConferenceID`, `ConfbridgeID`, `ConversationID`, `QueueID`류 | `ref:"resource"` |
| `ai_summary.reference_id`(사용자 입력, `reference_type`과 쌍), `hangup.reference_id`("이 통화 id와 같은 사유로 끊는다") | `ref:"resource"` |

- 태그 값은 **두 종류뿐이다**: `action`, `resource`. 3회차 코드 확인에서 `reference_id` 두 곳 모두 사용자 입력이라 `resource`가 맞았다. 이전 안의 `optional` 접미사와 `<kind>`는 폐기한다(필수 여부는 `actioncatalog.Required`가 이미 갖고 있어 진실 원천이 둘이 되고, 실측에서 서로 충돌했다. 예: `ai_summary.reference_id`는 카탈로그 `Required: true`, `hangup.reference_id`와 `on_end_flow_id`와 `ai_id`는 `Required: false`).
- **경고 규칙**: 비어 있는 `ref:"action"` 필드는 항상 `empty_action_ref: <label>.<field>`(`branch`, 조건 액션은 대상이 비면 실행 시 `GetAction` 오류로 흐름이 중단된다). 비어 있는 `ref:"resource"` 필드는 `actioncatalog`의 `Required: true`일 때만 `select_resource: <label>.<field>`(`Required: false`는 경고하지 않는다. 예: `hangup.reference_id`, `on_end_flow_id`, 폐기 예정 `ai_id`). `ref` 필드는 `missing_required`로 중복 보고하지 않는다.
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
- **에디터 충실도**(에디터가 저장 시 의미를 보존하는가)는 런타임 검사가 아니라 두 곳으로 관리한다: (가) 보존하지 못하는 타입은 `Exposure=internal`로 선언한다(2.2). (나) 에디터의 분기 계열(`branch`, `condition_datetime`)에 대해 `initNodes → initEdges → getActions` 순서(`initEdges`는 노드를 만들지 않으므로 `initNodes`가 선행되어야 한다)로 `ref:"action"` 필드와 `next_id`가 복원됨을 확인하는 **고정 단위 테스트**를 square-admin에 둔다. 현재 이런 타입이 둘뿐이므로 이 테스트는 손으로 쓴 픽스처로 충분하다. **기대값 주의**: `getActions`는 `branch`의 default 엣지 대상을 `next_id`로도 채우므로(`action.js`) 저장 후 `branch`의 `next_id`는 `DefaultTargetID`와 같아진다. 테스트는 "변환기가 `branch`에 `next_id`를 만들지 않았다"와 "에디터 저장 후 `next_id == default_target_id`"를 구분해 검증한다. 새 분기 타입이 에디터에 추가되면 같은 테스트에 픽스처를 더한다(PR 체크리스트).
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

- `next`는 후속 label이며 생략하면 후속 없음. FlowKind가 `continue`가 아닌 타입(`jump`인 `branch`, `terminate`인 `hangup`/`stop`)은 `next`를 쓰지 않는다. 쓰면 변환기가 무시하고 `next_ignored` 경고를 기록한다(판정은 메타의 FlowKind이며 타입 이름이 아니다). 실행기도 `terminate`의 `next_id`를 따르지 않으므로 에디터에 죽은 간선이 그려지지 않게 한다.
- 분기 대상 등 `ref:"action"` 값은 label 문자열 또는 생략. `ref:"resource"` 값은 null.

### 3.2 코드가 확정하는 변환 (서버, 순서 고정)

1. **label 정규화**: 중복 label은 첫 노드를 유지하고 이후 노드를 제거하며 `duplicate_label` 경고를 기록한다(이후 참조는 첫 노드로 해석). 클라이언트가 보낸 `current_draft`는 신뢰하지 않는다. `actions`의 `next_id`나 `ref:"action"` 값이 `actions` 안에 없는 id를 가리키면 `IDEmpty`로 비우고 `invalid_label_ref`로 기록한다. `labels`(5절)의 중복, 존재하지 않는 id, 자동 부여 label(`n1...`)과의 충돌은 버리고 재부여한다.
2. **타입 필터**: 노출 집합에 없는 타입의 노드를 삭제하고 `unsupported_action: <label>` 경고를 기록한다. 삭제 노드를 가리키던 참조(`next`, `ref:"action"` 값)는 비운다(`IDEmpty`). 비워진 `ref:"action"`은 3.3에서 `empty_action_ref`로, 후속이 끊긴 노드는 열린 끝으로 자연히 잡힌다. 후속 체인 이어 붙이기는 하지 않는다(카탈로그에 없는 타입을 LLM이 내는 경우는 드물 것이므로 실측 후 확장한다). `actions[0]`이 삭제되면 삭제 노드가 가리키던 `next` 노드를 새 시작으로 쓴다. 새 시작을 정할 수 없으면(후속이 없거나 후속도 삭제된 경우 포함) 남은 노드가 있어도 초안을 비우고 `empty_draft`로 응답한다(시작 없는 그래프는 의미가 없다).
3. 남은 노드에 새 UUID를 1:1 부여한다.
4. **먼저 모든 `ref:"resource"` 필드 값을 `IDEmpty`로 비운다**(LLM이 이름 문자열 등 UUID가 아닌 값을 넣어도 `invalid_option`과 `select_resource`가 중복 보고되지 않게). 이어서 `next` → `next_id`(없으면 `IDEmpty`). FlowKind가 `continue`가 아닌 노드의 `next`는 무시하고 `next_ignored: <label>`을 기록한다. `ref:"action"` 값(스칼라, 맵 값)과 `next`의 label을 UUID로 치환한다. **알 수 없는 label**은 `IDEmpty`로 비우고 `invalid_label_ref: <label>`을 기록한다(노드는 유지. 에디터에서 보완 가능). 이어서 `option`을 `OptionStructByType`의 구조체로 엄격 디코딩해 알 수 없는 키나 형 불일치는 버리고 `invalid_option: <label>.<key>`를 기록한다(LLM이 틀린 옵션을 내도 저장 시점에야 발견되지 않게 한다).
5. `ref:"resource"`는 `IDEmpty`로 두고, 2.3의 경고 규칙에 따라 `select_resource: <label>.<field>`를 기록한다. null/생략은 모두 `IDEmpty`.
6. **배열 순서**: 시작 노드는 항상 `actions[0]`이다(`IDStart`는 `Actions[0]`을 반환하므로 다른 규칙보다 우선한다). 나머지는 입력 순서를 유지하되 **"열린 끝"(FlowKind가 `continue`이고 후속이 없는 노드)을 배열 뒤쪽으로 미룬다**(낙하가 곧 종료가 되도록 마지막 원소로 보낸다). 변환기는 후속이 있는 `continue` 노드의 `next_id`를 **항상 명시**하고 배열 인접 낙하에 의존하지 않는다. 낙하에 의존하는 것은 열린 끝뿐이다. **배열의 마지막 원소가 아닌 열린 끝은 항상** `open_end: <label>`을 기록한다(시작 노드가 열린 끝이고 고아 노드가 뒤에 놓인 경우, 열린 끝이 둘 이상인 경우 포함. 낙하로 다음 노드가 실행되므로 사용자가 에디터에서 연결을 완성해야 한다). 마지막 원소인 열린 끝은 정상 종료이며 경고하지 않는다. 사용자가 에디터에서 노드를 추가하면 `nodes[]` 끝에 붙어 마지막 열린 끝이 그 노드로 낙하할 수 있다. 이는 사람이 만든 Flow에도 동일한 에디터 특성이며 수용한다. 이 단계는 2~5가 끝난 최종 그래프를 기준으로 한다.
7. **레이아웃**: 시작 `x=0, y=100`, 깊이마다 `y`를 500 증가, 같은 깊이는 `x`를 450 간격(템플릿과 같은 간격). 도달 불가 노드는 가장 깊은 도달 행 아래 한 행에 둔다. 도달 판정의 간선은 `next_id`, `ref:"action"` 값, 마지막이 아닌 열린 끝의 배열 인접 낙하다(검증기와 레이아웃이 같은 정의를 쓴다). 가상 `start` 좌표는 건드리지 않는다.
8. 빈 결과(필터 후 노드 0개, 또는 2번에서 새 시작을 정할 수 없는 경우): `draft`를 생략하고 `message`에 이유를 쓰며 `empty_draft`를 기록한다.
9. 출력: `actions`(`[]Action`, `actions[0]`이 시작), `positions`, `labels`(`{id: label}`). `actions`와 `positions`는 에디터의 `setInitialActions`/`setInitialPositions`에 변환 없이 들어간다.

### 3.3 그래프 구조 검증기 (v1 범위)

`builderhandler`의 검증기는 타입 이름을 모른다. 입력은 `[]Action`과 메타(`FlowKind`, `ref`), `actioncatalog`의 `Required`, 기존 `MapRequiredMediasByType`뿐이다. 후속 참조의 정의는 FlowKind별로 다르다: continue = `next_id`가 있으면 그 노드, **`next_id`가 비었을 때만** 배열 인접 원소(+ `ref:"action"` 값), terminate = 없음, jump = `ref:"action"` 값만. 모든 결과는 경고다(초안은 항상 응답한다).

v1 검사 항목(기계적이고 비용이 낮은 것만). 끊어진 참조는 서버가 매 턴 그래프를 재구성하고 3.2의 4번이 알 수 없는 label을 비우므로 변환 후에는 발생하지 않아 별도 검사를 두지 않는다.
1. 비어 있는 `ref:"action"` 필드(`branch`, 조건 액션 모두 대상이 비면 실행 시 `GetAction` 오류로 흐름이 중단) → `empty_action_ref`(조건 액션은 false 경로에서만 실행 오류가 나지만 대상이 비면 항상 경고한다). `branch.target_ids`가 빈 맵인 경우는 `default_target_id`가 있으면 경고하지 않는다.
2. 도달 불가 노드(`actions[0]`에서 후속 참조를 따라 닿지 않음) → `unreachable: <label>`.
3. 열린 끝 낙하(3.2의 6번)와 리소스 참조 비움 목록.
4. 필수 옵션 누락(`actioncatalog`의 `Required`, `ref` 필드 제외) → `missing_required: <label>.<field>`.
5. **미디어 혼용**: 카탈로그의 `MapRequiredMediasByType`에서 RT 전용 타입과 NRT 전용 타입이 한 초안에 함께 있으면 `media_mixed` 경고(실행 시 한쪽이 건너뛰어진다). 트리거 채널은 초안에 없으므로 이 이상의 판정은 하지 않는다. **알려진 한계**: 통화 전용 `terminate`(`hangup`)와 공통 타입만 쓴 초안을 비통화 흐름에서 실행하면 `hangup`이 건너뛰어져 다음 노드로 낙하할 수 있다. 이를 코드가 잡을 수는 없으므로 프롬프트에서 비통화 흐름은 공통 `terminate`(`stop`)로 끝내도록 유도한다(4절).

**출구 없는 루프(강연결 성분) 탐지는 v1에서 뺀다.** 실행기에 `maxNextActionLoopCount` 안전장치가 있고 평가에서 실제 문제가 확인된 뒤 추가한다(Open Question 9).

## 4. 시스템 프롬프트 구성

Assistant Builder의 구성(역할, 핵심 차원, 인터뷰 행동 규칙, few-shot, 응답 JSON)을 같은 방식으로 따른다. 차이만 적는다.

1. 역할: VoIPBin Flow를 설계하는 인터뷰어이자 컨설턴트. 핵심 차원: 트리거 채널, 성공 기준, 분기 조건, 실패/예외 시 동작.
2. **능력 카탈로그는 매 요청마다 코드가 생성한다**(2.5의 교집합 + 각 타입의 Exposure/FlowKind + `MapRequiredMediasByType`의 적용 미디어: 통화 전용, 비통화(AI/API/대화/캠페인/webchat) 전용, 공통). 설명 문구는 `actioncatalog`의 `actionCatalogEntry`(설명 필드, 현재 이름은 구현 시 확인)를 가져온다.
3. 행동 규칙(전역 플래그가 아니다): 한 Flow에는 같은 미디어에서 실행되는 타입만 쓴다(통화 전용과 비통화 전용을 섞지 않는다. 섞이면 코드가 `media_mixed`로 표시한다). 통화가 아닌 흐름은 모든 미디어에서 동작하는 `terminate`로 끝낸다. `sensitive` 타입은 사용자가 대화에서 해당 동작을 말했을 때만 초안에 넣는다. 모든 경로는 `terminate` 타입으로 끝내거나 열린 끝은 하나만 둔다. `jump` 타입에는 `next`를 쓰지 않는다. 이 규칙은 LLM이 지키는 것이므로 코드가 사실을 확정하는 것은 `sensitive_nodes`의 구조화 표시와 검증기 경고까지다.
4. few-shot 2개(2.4의 상수로 보호), 초안 형식은 3.1.
5. `message`와 경고는 마크다운 없는 평문. 사용자 화면 문구는 영어.
6. **초안(`current_draft`)의 `option` 값은 매 턴 외부 LLM으로 전송된다.** Flow 패널 안내문에도 Assistant용과 같은 "do not include secrets"(웹훅 URL/헤더 등) 문구를 둔다.

## 5. REST API

| Method | Path | 설명 |
|---|---|---|
| POST | `/flow_builder/chat` | `ChatRequest{messages, current_draft, supported_action_types}` → `ChatResponse{message, draft, draft_warnings, sensitive_nodes, assumptions}` |

- `draft = {actions, positions, labels}`. 클라이언트는 `current_draft`로 그대로 되돌려 보내고, 서버는 `labels`와 `actions`로 기호 그래프를 매 턴 결정적으로 재구성한다(UUID는 LLM에 보이지 않는다). BuilderPanel은 에디터 진입 전 패널이므로 사용자가 에디터에서 고친 그래프를 되돌려 보내는 경로는 없다. `positions`는 매번 재계산된다.
- `draft_warnings`는 기존 Assistant Builder 패턴을 따른다: `key` 또는 `key: detail` 형식의 영어 키 문자열 배열이며 클라이언트가 키로 문구를 고른다. 키 목록: `duplicate_label`, `unsupported_action`, `invalid_label_ref`, `invalid_option`, `select_resource`, `open_end`, `empty_action_ref`, `unreachable`, `next_ignored`, `missing_required`, `media_mixed`, `empty_draft`.
- `sensitive_nodes`: 노출된 `sensitive` 노드의 id 배열(신규 응답 필드, `omitempty`, OpenAPI `ChatResponse`에 추가). 프런트가 노드를 강조한다.
- label 연속성은 보장되지 않는다(best effort). 저장 전이라 영향이 없다.
- **와이어 상한**: 노드 수 `MaxFlowNodes = 60`(초기값, 미측정), `current_draft.actions` 개수 <= 60, `positions`/`labels` 항목 수는 `actions` 이하, 노드당 `option`은 직렬화 크기 상한(구현 시 정하고 `ValidateRequest`에 포함), `option` 중첩 깊이 상한. 메시지 수/길이는 Assistant Builder 상수를 공유한다. `ValidateRequest`는 대화용 한도와 **초안용 한도를 분리**한다. 대화용: Assistant와 같은 상수(`MaxMessages` 40, `MaxMessageRunes` 2000, `MaxTotalRunes` 40000)가 **메시지 텍스트만** 합산한다(요청에는 `assumptions`가 없다). 초안용: 노드당 `option` 직렬화 4KiB, `MaxFlowNodes` 60, 초안 전체 직렬화(`actions` 합계) 약 240KiB(`positions`와 `labels`는 아래에서 따로 센다), `labels` 항목은 `actions` 수 이하이며 label 길이 상한 64자(모두 미측정 초기값). 대화 이력은 최대 약 120KiB(40,000룬 x 3바이트이며 JSON `\uXXXX` 이스케이프 시 6바이트가 되어 한도 미만에서도 거절될 수 있음을 기존 코드와 같이 수용한다. `option`에 한글이 많을 때도 같다), `positions`/`labels` 약 20KiB, 합계 약 380KiB이므로 본문 상한 초기값은 512KiB로 둔다(평가 하네스 실측 후 조정하며 **앞단 ingress의 본문 크기 제한과 충돌하지 않는지 구현 시 확인**한다). Flow 전용 메시지 한도는 두지 않고 Assistant 상수를 공유하므로(기존 status 재사용의 전제) 한도를 따로 정하려면 status도 함께 바꿔야 한다. **요청 본문 상한은 Assistant의 160KiB를 그대로 쓰지 않는다.**

### 5.1 횡단 관심사 (실제 ai_builder 구현을 읽고 확인한 항목)

- **RBAC**: `canUseBuilder`와 같은 규칙(로그인한 Agent + customer admin 또는 manager). 상태 조회는 기존 `GET /ai_builder/status`를 **재사용**한다(가용성은 같은 키 설정에 달려 있고 한도도 같으므로 별도 status를 만들면 RPC, 캐시 인스턴스, OpenAPI, 테스트가 불필요하게 늘어난다). 권한이 없으면 200과 `available:false`. Flow 생성 권한(`servicehandler/flow.go`: Admin|Manager)과 일치한다.
- **처리 순서**: key → ValidateRequest → semaphore → 일일 한도 → LLM. 파싱 실패도 한도를 소비한다(Assistant와 동일).
- **일일 한도 카운터**: `cachehandler.BuilderChatCountIncr`는 현재 고정 키(`ai:builder:chat:count:%s`) 하나다. 결정(Open Question 7)에 따라 두 안이 있다. (공유) 키를 그대로 쓰고 오류 문구만 일반화한다. 한쪽 사용이 다른 쪽 한도를 소진한다. (분리) **Assistant의 키 문자열은 그대로 두고 Flow 키만 추가**한다(배포 시 진행 중인 일일 카운트가 초기화되지 않게). `CacheHandler` 인터페이스, mock, `cachehandler/builder_test.go`를 함께 갱신한다. 한도 값은 같은 기본값.
- **세마포어와 설정**: `sem`과 `DailyLimit`은 `builderHandler` 인스턴스 필드(`builderhandler/main.go`)다. **세마포어는 공유한다**: LLM 동시 호출 상한은 플랫폼의 LLM 제공자 부하를 막는 하나의 자원 한도이므로 Assistant와 Flow가 같은 풀을 쓴다(Flow 턴이 더 오래 점유해 서로 `BUILDER_BUSY`를 유발할 수 있음을 수용한다). Flow 핸들러는 `sem`을 **주입**받는 별도 구현으로 둔다(같은 인스턴스를 쓰면 기존 `fail()`/`result`가 `ai_manager_builder_*` 메트릭을 직접 증가시켜 `flow_builder_*`와 분리 집계할 수 없다). 일일 카운터의 공유/분리(OQ7)는 별개 결정이다(고객당 사용량 한도). `cmd/ai-manager/builder_wiring.go`, config(`ai_builder_*`)를 **그대로 공유**하며 새 플래그를 만들지 않는다.
- **출력 토큰**: `MaxOutputTokens` 기본 4096은 60노드 JSON 초안에 부족할 수 있다. Flow용 값은 `builderhandler.Config`(`MaxOutputTokens`, `LLMTimeout`, `SystemPrompt`, `JSONMode`는 `Options`가 아니라 `Config`에 있다)의 Flow 전용 인스턴스에 코드 상수 기본값으로 두고(새 설정 플래그나 환경 설정 변경 없음) 평가 하네스 실측으로 정한다. LLM 타임아웃(40초, 검증 <= 50)도 실측 후 재확인한다.
- **로깅 금지**: 본문, `current_draft`, 파싱 오류, 실패 오류 텍스트를 로그에 남기지 않는다(`option`에 민감값이 올 수 있다). 요청 로거 이전 라우팅, 오류를 응답으로만 변환, panic 복구 시 본문 제외를 Assistant와 같은 방식으로 적용한다. **`listenhandler`의 `isBuilderRoute`는 URI 완전 일치**이므로 Flow URI를 반드시 추가해야 이 보장이 적용된다. 센티널 문자열 테스트로 검증한다.
- **타임아웃 순서**: LLM < RPC < 클라이언트. `BUILDER_TIMEOUT` 등 오류 reason 매핑은 재사용한다. 서버 쪽 오류 문구(`mapBuilderRPCError`, ai-manager의 "assistant builder" 문자열)와 프런트 `ERROR_COPY`에서 "assistant" 고정 표현을 일반화하거나 kind별로 분리한다.
- **메트릭, 서킷브레이커**: 기존 Assistant 메트릭(`ai_manager_builder_chat_total{result}`, `builder_chat_duration_seconds`, `builder_tokens_total`, `api_manager_builder_timeout_total`, `api_manager_builder_circuit_open_total`)의 이름과 라벨은 **바꾸지 않는다**(라벨을 추가하면 집계 없는 기존 쿼리와 알림이 시리즈 분리로 깨질 수 있고 이 저장소 밖의 대시보드는 확인할 수 없다). Flow는 `flow_builder_` 접두 별도 메트릭을 같은 형태로 추가한다. 패닉 복구(`processBuilder`의 recover)는 `builderhandler.RecordPanic()`이 Assistant 시리즈(`ai_manager_builder_chat_total{result="internal"}`)를 증가시키므로 Flow 라우트용 패닉 기록을 별도로 두거나 라우트별로 분기한다. **서킷브레이커는 RPC 큐 단위**(`send_request.go`의 `r.cb.Allow(string(queue))`)이고 `sendDirectRequest`의 모든 오류(타임아웃 포함)가 `RecordFailure(queue)`로 집계되며 연속 실패 임계값은 5회(`circuitbreakerhandler/option.go`)다. 따라서 Flow 턴은 큐를 공유하는 다른 ai-manager RPC와 브레이커를 공유하며 분리할 수 없다(분리하려면 `requesthandler` 공통 코드를 바꿔야 하며 이 설계의 범위가 아니다). **결합을 수용한다**: Assistant Builder도 이미 같은 결합이고, Flow 턴은 LLM 40초와 출력 토큰 증가로 타임아웃 확률이 더 높을 수 있으나 브레이커는 연속 5회 실패가 있어야 열리고, 열리면 같은 큐의 다른 ai-manager RPC도 기본 30초간(`defaultOpenDuration`) 차단된다. 완화는 평가에서 Flow 턴의 타임아웃 비율을 실측해 높으면 빌더 타임아웃을 집계에서 제외하는 `requesthandler` 변경을 후속으로 설계한다(Open Question 10). `"ai/flow_builder/chat"`은 소요 시간 메트릭의 `resource` 라벨이다. 
- **문서/인터페이스**: `bin-api-manager/docs/routing.md`, `bin-ai-manager/docs/operations.md`, `servicehandler/main.go` 인터페이스와 mock 갱신.

## 6. 프런트엔드 통합

- `flows_create.js`의 템플릿 다이얼로그 옆에 "Build with AI" 진입점. 패널은 다이얼로그 내부에서 전환되는 Assistant 패턴(`ais_create.js`)과 같게 둔다. 모바일에서는 에디터가 요약 화면이고 저장 버튼도 비활성이므로(`disabled={isSubmitting || isMobile}`) 진입점을 숨긴다.
- **BuilderPanel/스토어 공용화(실제 결합도 반영)**: 현재 `builderApi.js`는 `ai_builder/*` 하드코딩과 `ERROR_COPY`, `builderStore.js`는 모듈 로드 시 `restore()`, `registerCacheClearHook(reset)`, `BUILDER_SESSION_KEY`(`resourceCacheStore`)를 가지며, `BuilderPanel.js`는 `AVAILABLE_TOOLS`, Assistant 전용 안내문, `TOOLS_NOTE`, `draftOk`/`isValidCopy`(Assistant 초안 모양 고정)를 가진다. v1 작업:
  - 스토어를 팩토리로 바꾸고 엔드포인트, 세션 키, 초안 검증 함수(`isValidDraft`), 전송 페이로드 확장(`supported_action_types`), 응답 필드(`sensitive_nodes`)를 주입한다.
  - 팩토리가 만든 스토어는 **`registerCacheClearHook`에 등록**하고 새 세션 키를 `resourceCacheStore`에 추가해 로그아웃과 delegate 전환 시 지워지게 한다.
  - 패널 안내문, `TOOLS_NOTE`, 초안 렌더를 슬롯으로 주입한다. 다이얼로그 너비도 전환 대상이다(`flows_create.js` 다이얼로그는 `sm:max-w-[500px]`, `ais_create.js`의 builder 모드는 `sm:max-w-[760px]`).
  - 두 대화가 서로 덮어쓰지 않도록 세션 키를 분리한다. sessionStorage에는 Assistant 초안(`init_prompt`)과 같은 수준으로 Flow 초안이 저장된다(안내문의 secrets 경고로 갈음).
  - 기존 `ais/builder/__tests__`의 이동과 회귀 검증을 포함한다(파일 단위 목록은 구현 착수 시 작성). 공용화가 과하다고 판단되면 Flow 전용 얇은 사본이 대안이며 착수 시 비용 비교를 제시한다.
- **진입점 표시**: 기존 `GET /ai_builder/status`의 `available`을 패널 진입 시 읽어 `false`이면 진입점을 숨긴다(Assistant 생성 화면과 같은 방식). 데스크톱에서만 표시한다.
- **초안 적용**: `setInitialActions(draft.actions)`, `setInitialPositions(draft.positions)`와 `setGraphKey(k => k + 1)`로 재마운트한다(`ActionGraph`는 `!isInitialized`일 때 한 번만 초기화). 이미 편집 중이면 덮어쓰기 전에 `ConfirmDialog`를 띄운다. "편집 중" 판정은 `graphRef.current?.getActionsData()`의 노드 수가 0보다 큰지로 하며 **try/catch로 감싼다**(`getActionsData`는 그래프가 초기화되기 전에는 예외를 던진다). ref가 null이거나 예외이면 편집 중이 아닌 것으로 본다. `initialActions.length`는 사용자가 에디터에서 직접 추가한 노드를 놓치므로 쓰지 않는다. dirty 추적은 새로 만들지 않는다. 데스크톱에서 편집한 뒤 창을 모바일 폭으로 줄여 요약 화면이 된 경우에도 같은 store를 읽으므로 판정이 동작한다(모바일에서는 진입점 자체를 숨기므로 이 경우는 덮어쓰기 확인 로직의 방어용이다). 이름/설명 입력은 건드리지 않는다.
- **방어**: `actions`에 `nodeTypes`에 없는 타입이 있으면 주입을 중단하고 영문 안내를 표시한다(서버가 이어 붙이기를 담당하므로 프런트는 고치지 않고 거부).

## 7. sensitive 노출 정책

v1(코드 상수 결정이며 설정 플래그가 아니다): `sensitive`를 노출하고, 초안에 포함되면 `sensitive_nodes`에 노드 id를 담는다(구조화 필드 하나만 둔다. 같은 사실을 경고 문자열로 중복하지 않으며, 패널이 `sensitive_nodes`가 비어 있지 않을 때 고정 영문 안내와 노드 강조를 표시한다). 프롬프트 규칙은 사용자가 말한 동작만 넣게 한다(LLM 준수 사항이며 코드가 보증하는 것은 표시까지다). 비노출로 정하면 콜백, 아웃바운드 캠페인처럼 핵심 시나리오를 만들 수 없으므로 노출을 권고한다. 결정은 대표님 몫이다(Open Question 1).

실행 시점 게이트: 저장 전에는 Flow가 없으므로 초안은 실행되지 않는다. 저장된 Flow 실행 시 비용/권한 게이트는 하위 서비스 책임이다. `fetch`/`webhook_send`의 URL 검증은 구현 시작 시 코드를 읽고 확인했다(Open Question 8): `fetch`는 검증이 없어 v1 비노출(`internal`), `webhook_send`는 webhook-manager가 검증한다.

## 8. 영향 서비스

| 서비스 | 변경 |
|---|---|
| bin-flow-manager | `models/action`: `Meta`/`MetaByType`, `ref` 태그, drift-lock 테스트 3종(`TestMetaCoversAllTypes`, `TestEveryUUIDFieldIsTagged`, `TestBuilderExcludedTypes`) |
| bin-ai-manager | `pkg/builderhandler`에 flow 전용 chat 핸들러, 변환기, 검증기, 프롬프트. `models`에 flow builder 요청/응답/검증. `listenhandler` 라우트와 `isBuilderRoute`, `listenhandler/models/request/builder.go`에 Flow 요청 타입(기존 `V1DataBuilderChatPost`는 Assistant `Draft`를 쓴다), `listenHandler` 생성자의 Flow 핸들러 필드 추가, `cachehandler`(Flow 카운터 키 추가, Assistant 키 유지) + mock + `builder_test.go`, `flow_builder_` 메트릭 추가, `cmd/ai-manager/builder_wiring.go`, 평가 하네스, `docs/operations.md`, "the assistant builder" 고정 문구 5곳 이상(`chat.go` 등)의 일반화 |
| bin-common-handler | `requesthandler`에 flow builder RPC(`ai_builder.go` 패턴, 소요 시간 `resource` 라벨만 Flow용) |
| bin-api-manager | `server/flow_builder.go`, `servicehandler/flow_builder.go`, `servicehandler/main.go` 인터페이스와 mock, 기존 `builderStatusCache` 재사용, `flow_builder_` 메트릭, `gens/openapi_server/gen.go` 재생성, `docs/routing.md`, `docs/operations.md`(`api_manager_builder_*` 행) |
| bin-openapi-manager | `paths/flow_builder/chat.yaml`, `openapi.yaml` 라우트 등록, `gen.go` 재생성. 응답 스키마에 `sensitive_nodes`와 Flow 경고 키 목록 설명 추가. OpenAPI 설명의 "not released / no on-off setting" 문구는 Assistant Builder의 현재 문구 정책을 그대로 따른다 |
| square-admin | 스토어 팩토리화, 패널 슬롯화, 테스트 이동, flows_create.js 진입점, 에디터 분기 계열 단위 테스트 |

## 9. 평가 (수용 기준)

Assistant Builder의 `builder_eval` 하네스를 차용해 Flow 시나리오로 교체한다.
1. 모호한 요청 → 트리거 채널과 실패 처리를 좁히는 질문.
2. 전문가가 분기 조건을 모두 제공 → 열린 갈림길이 없으면 한 턴 초안.
3. **결정론 검사(LLM 판정 불필요)**: 끊어진 참조 0, 도달 불가 0, `open_end` 경고와 "배열 마지막 원소를 제외한 열린 끝" 1:1, `select_resource` 경고와 비어 있는 리소스 필드 1:1.
4. **메타데이터 변이 테스트**: `TypeListAll`에 새 타입을 추가하고 `MetaByType`/태그를 빼면 CI가 실패하는지 확인한다.
5. 프런트 분기 계열 왕복 단위 테스트(2.5).
6. v1 범위 밖 확인: 재시도 루프(`goto` 기반) 시나리오는 빌더가 만들 수 없으며(2.2) 평가 시나리오에서 제외하고 "지원 범위 밖" 응답을 확인한다.
7. 에디터 적용 후 `getActionsData()` 결과가 `draft.actions`와 의미상 같은지 비교(에디터 왕복에서 옵션 값이 조용히 변형되지 않는지).
8. Flow 초안 크기 실측(출력 토큰, 본문 크기, 지연)을 5.1의 값 결정에 사용한다.

## 10. 보안 / 블라스트 레디어스

- 서버 무저장은 5.1의 로깅 금지와 구조적 보장으로 지킨다.
- 초안은 실행되지 않는다(7절).
- 외부 LLM에 `option` 값이 전송되는 점을 패널 안내문으로 고지한다(4절 6항).

## 11. Open Questions

| # | 질문 | 제안 | 결정 |
|---|---|---|---|
| 1 | `sensitive` 노출(코드 상수 결정) | 노출 + 구조화 표시와 경고. 비노출 시 아웃바운드 시나리오 불가 | **확정(대표님, 2026-10-07): 노출**. |
| 2 | 리소스 ID 목록 주입 | v1 비움 + 경고, 주입은 후속 | 대표님 |
| 3 | `ref` 태깅 범위 | 해결: UUID 필드를 가진 모든 타입에 태깅 | 확정 제안 |
| 4 | 한도/오류코드 공유 | `Reason*` 코드와 대화용 한도 상수는 `models/builder`와 공유하고, 요청/응답/검증 타입은 신규 flow builder 모델 패키지로 둔다 | 대표님 |
| 5 | `ref` 태그를 `describe_action` 카탈로그에 반영 | 범위 밖 | 대표님 |
| 6 | 노출 타입 집합 | 고정 목록 없이 `Exposure` ∩ 프런트 목록 ∩ 구조 제외 | 해결 |
| 7 | 일일 한도 카운터를 Assistant와 분리할지 | 분리 권고(한쪽이 다른 쪽을 소진하지 않게). 공유는 추가 변경 없이 가능하나 서로 한도를 소진. 분리 시 cachehandler 인터페이스/mock/테스트 변경 | **확정(대표님, 2026-10-07): 분리**. |
| 8 | `fetch`/`webhook_send` URL 검증 | **확인 완료(구현 시작 시 코드 확인)**: `webhook_send`는 bin-webhook-manager의 SSRF 방어 클라이언트와 URL 검증(`urlvalidator.go`, 사설/예약 IP 거부)을 거친다. `fetch`(`ActionFetchGet`)는 일반 `http.Client{}`로 옵션의 URL을 그대로 호출하며 검증이 없다. 결정: `fetch`는 `Exposure=internal`로 두어 v1 빌더가 넣지 않는다(실행기가 URL을 검증하게 되면 재검토). `webhook_send`는 `sensitive`로 유지 | CPO 결정, 대표님 검토 |
| 9 | 출구 없는 루프 탐지 | 평가에서 실제 문제 확인 시 후속 | 대표님 |
| 10 | Flow 턴 타임아웃이 큐 서킷브레이커에 미치는 영향 | 수용, 평가 실측 후 필요 시 빌더 타임아웃 집계 제외를 후속 설계 | 대표님 |

## 12. 구현 순서

1. `bin-flow-manager/models/action`: `Meta` 레지스트리, `ref` 태그, drift-lock 테스트. 모든 타입의 실행 핸들러를 읽어 Exposure/FlowKind 확정.
2. `bin-ai-manager`: 검증기(순수 함수, 테스트 우선) → 변환기 → 프롬프트와 카탈로그 생성 → chat 핸들러와 wiring → 평가 하네스(출력 토큰 실측).
3. `bin-common-handler`, `bin-api-manager`, `bin-openapi-manager`: RPC와 REST.
4. `square-admin`: 분기 계열 왕복 테스트 → 스토어/패널 공용화 → 진입점.
5. 평가 실행(9절) → 코드 리뷰 루프.

## 부록 A. 처리 내역

### 8회차 (M APPROVE, N APPROVE), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| M-L1, N-Nit1 머리말 불일치 | 타당 | 머리말 정정 |
| M-L2 `current_draft` 위조 id | 타당 | 3.2의 1번 |
| M-L3 프롬프트 한계와 평가 연결 | 3.3 알려진 한계에 이미 명시. 평가 시나리오 1에서 채널 확인 질문과 함께 확인 | 변경 없음 |
| M-Nit 3.2의 8번 중복 | 타당 | 문구 정리 |
| N-L1 `RecordPanic`이 Assistant 시리즈를 증가 | **사실.** `processBuilder` recover | 5.1 |
| N-L2 8절 누락(request 타입, 생성자, operations.md) | 타당 | 8절 |
| N-L3 진입점 표시 조건 | 타당 | 6절 |
| N-Nit2 OQ4 모호 | 타당 | OQ4 문구 |

### 7회차 (K APPROVE, L APPROVE), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| K-M1 9절 3번이 `open_end` 정의와 충돌(n개 열린 끝에 경고 n-1개) | **사실.** 3.2의 6번은 마지막 원소를 경고하지 않음 | 9절 3번 정정 |
| K-L1 3.3 입력 목록에 `MapRequiredMediasByType` 누락 | 타당 | 3.3 |
| K-L2 열림 지속 시간 | **사실.** `defaultOpenDuration` 30초 | 5.1 |
| K-L3, L4 `actions[0]` 삭제 연쇄, 8번 중복 | 타당 | 3.2의 2번, 8번 |
| K-Nit 헤더 "4회차 Critical 0" 근거 | 4회차 리뷰어 E, F의 판정에 Critical 없음(보고 원문 기준) | 헤더 유지, 추적은 보고에 있음 |
| L-L1 `MaxOutputTokens`는 `Config`에 있음 | **사실.** `Options`는 `KeyConfigured`, `DailyLimit`, `MaxConcurrent` | 5.1 |
| L-L2 `positions`/`labels` 포함 여부 | 타당 | 5절 |
| L-L3 `fail()`이 Assistant 메트릭을 직접 증가 | **사실.** | 5.1: 세마포어 주입 방식 |
| L-Nit 서버 문구 일반화 대상 | 타당 | 8절 |

### 6회차 (I APPROVE, J REQUEST_CHANGES), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| I-M1, J-M1 서킷브레이커 영향 범위와 수용 근거 누락(5.1에 "위 설명"이 없었음) | **사실.** 큐 단위, 모든 오류가 `RecordFailure`, 임계값 5 | 5.1에 사실, 수용 결정, 후속 조건(OQ10) |
| J-M2 세마포어 공유와 OQ7 논거 충돌 | 타당 | 세마포어 공유로 확정, 일일 카운터와 별개 결정임을 명시(5.1) |
| J-M3, I-L3 `assumptions`는 요청 필드가 아님 | **사실.** `ChatRequest`는 `messages`, `current_draft` | 5절 정정 |
| J-L1, L2 이스케이프, 앞단 본문 제한 | 타당 | 5절 |
| I-L1 `empty_draft` 조건 불일치 | 타당 | 3.2의 2번, 8번 통일 |
| I-L2, J-Nit 3.3 번호 | 타당 | 번호 정리 |
| I-L4 resource 값이 비-UUID 문자열일 때 | 타당 | 3.2의 4번 순서 변경 |
| I-L5 `call` Exposure 값 | 타당 | 부록 B |
| J-L4 amd/connect | 구현 PR 확정 항목 | 부록 B 비고 |
| J-L5, Nit 모바일 문장, 다이얼로그 너비, `ai_*` 표현 | 타당 | 6절, 2.2 |

### 5회차 (G APPROVE, H REQUEST_CHANGES), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| H-H1 서킷브레이커 키 서술 오류 | **사실.** `send_request.go`는 `r.cb.Allow(string(queue))`로 큐 단위, `ai/ai_builder/chat`은 소요 시간 메트릭의 `resource` 라벨 | 5.1, 8절 정정(서킷 분리 안 함) |
| H-M1 총 룬 수 산식 모순 | **사실.** `MaxTotalRunes` 40000에 `option`을 합산하면 성립 불가 | 대화용과 초안용 한도 분리(5절) |
| H-M2 메트릭 라벨 추가의 호환성 | 타당(PromQL은 자동 합산하지 않음) | 기존 메트릭 불변, `flow_builder_` 별도 메트릭(5.1) |
| H-M3 status 재사용 전제 | 타당 | 한도 공유 상수를 명시(5절) |
| H-M4, L2, L3, L6, Nit | **사실.** | 8절 정정, 존재하지 않는 참조 수정, `resource:*` 잔재 수정, gen.go 추가 |
| H-M5 OQ7 두 안의 영향 | 타당 | 5.1, OQ7에 공유/분리 양쪽 기재 |
| H-L1 모바일 | **사실.** `disabled={isSubmitting || isMobile}` | 6절 진입점 숨김 |
| H-L4 에디터 적용 후 의미 비교 | 수용 | 9절 7번 |
| G-M1 시작 노드 열린 끝 문장 오류 | **사실.** 고아 노드가 뒤에 오면 낙하 | 3.2의 6번 규칙을 "마지막 원소가 아닌 열린 끝은 항상 `open_end`"로 변경 |
| G-M2 "배열 인접"은 `next_id`가 비었을 때만 | **사실.** `GetNextAction` | 3.3 |
| G-M3 끊어진 참조 검사는 공허 | 타당 | 삭제(3.3의 1번) |
| G-M4 `next_ignored`, 미해결 `next`, `rewired` 정의 | 타당 | 3.2의 4번, `rewired` 폐기 |
| G-M5 건너뛴 `terminate` 낙하 | 코드로 한계 확인 | 3.3 알려진 한계, 4절 프롬프트 |
| G-M6, M7 `confbridge_join`, `block` 분류 | **사실.** 카탈로그 요약이 internal, `confbridge_id`는 런타임 값 | 부록 B `internal`. 같은 이유로 `ai_summary`도 `internal` |
| G-L1 옵션 검증 | 타당 | 3.2의 4번 `invalid_option` |
| G-L2~L5 | 타당 | 부록 B 비고, 3.3 문구 |
| G 오버엔지니어링: 3.2의 2번 이어 붙이기 | 수용(실측 신호 없음) | 삭제하고 참조를 비우는 단순안(3.2의 2번) |

### 4회차 (E, F 모두 REQUEST_CHANGES), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| E-H1 `verifyActionType`의 미디어 스킵(예: `hangup`은 RT 전용이라 대화 흐름에서 건너뜀), 기존 `MapRequiredMediasByType` 누락 | **사실.** `execute.go` `verifyActionType`과 `action.go`의 레지스트리 확인 | 2.2 적용 미디어, 4절 카탈로그, 3.3의 `media_mixed`. 새 메타 필드는 만들지 않음 |
| E-H2 `optional` 태그와 `actioncatalog.Required` 충돌 | **사실.** 카탈로그: `ai_summary.reference_id` Required true, `hangup.reference_id`와 `on_end_flow_id`와 `ai_id` Required false | `optional`, `<kind>` 폐기. 경고는 `ref` 종류와 `Required`로 계산(2.3) |
| E-M1 `goto`는 모델 밖 | **사실.** `loop_count <= 0`이면 진행 | 2.2에 명시, 재시도 루프 v1 제외, 9절에 기재 |
| E-M2 terminate의 `next` 무시 | **사실.** | `next_ignored` 일반화(3.1) |
| E-M3 시작 노드 고정 규칙, 열린 끝 연쇄 | 타당 | 3.2의 6번 |
| E-M4 43개 타입 표 | 타당 | 부록 B |
| E-L1, F-H2 `getActions`가 `branch`의 `next_id`를 채움, 낙하 의존 | **사실.** | 2.5 기대값 명시, 3.2의 6번에서 `next_id` 항상 명시하고 낙하 의존은 열린 끝뿐임을 밝히고 수용 |
| F-H1 편집 중 판정 경로 | **사실.** `getActionsData`는 미초기화 시 예외 | 6절 try/catch |
| F-M1 카운터 키 호환 | **사실.** | 5.1 |
| F-M2 메트릭 전환 범위 | **사실.** plain Counter 2개 | 5.1 |
| F-M3 `/flow_builder/status` 불필요 | 타당 | 폐기, 기존 status 재사용(5, 5.1, 8절) |
| F-M4 `sensitive_nodes`와 경고 중복 | 타당 | 경고 제거, 구조화 필드만 유지(7절) |
| F-M5 `ValidateRequest`의 룬 계산과 본문 산식 | 타당 | 5절 초기값과 산식 |
| F-L2, L3 서버 오류 문구, OpenAPI 설명 정책 | 타당 | 5.1, 8절 |

### 3회차 (C, D 모두 REQUEST_CHANGES), 코드 재검증
| 지적 | 재검증 | 처리 |
|---|---|---|
| C-C1 `NEXT_IGNORED`가 `ref:"action"` 유무로 조건 액션 true 경로를 지움 | **사실.** 조건 액션은 false만 `FalseTargetID`, true는 `next_id`. 에디터도 `condition_datetime`의 `next_id`를 보존 | `next` 무시 판정을 `ref` 유무에서 **FlowKind=jump**로 변경(2.2, 3.1) |
| C-H1/D-H1 왕복 테스트 기준이 branch/goto를 스스로 탈락시킴, 프런트는 Go 태그를 모름 | **사실.** | 런타임 왕복 검사 폐기. 충실도는 `Exposure=internal` 선언 + 분기 계열 고정 단위 테스트로 관리(2.5). `goto`는 `internal` |
| C-H2 낙하 모델이 branch/goto/condition에서 틀림 | **사실.** `branch`는 낙하하지 않음, `goto`는 `loop_count <= 0`이면 다음으로 진행 | FlowKind 3종으로 모델링(2.2, 3.3), `goto` 제외 |
| C-H3 `email_send` 첨부(`Attachments []Attachment`)의 `ReferenceID` 누락 | **사실.** `models/email/main.go`에 슬라이스 존재 | 중첩 UUID 타입은 구조적 제외 + 기대값 테스트(2.3) |
| C-H4 `Terminates` 값 미확정 | 후보를 코드로 확정(`stop`, `hangup`) | 2.2에 후보 판정 기재, 구현 시 최종 확정 |
| C-M1 `ai_summary.reference_id`를 runtime으로 분류한 근거 약함 | **사실.** 사용자 입력 옵션(`reference_type`과 쌍) | `runtime` 분류 삭제, `resource`로 분류(2.3) |
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

## 부록 B. 43개 타입 분류 초안 (구현 1단계에서 각 실행 핸들러를 읽어 확정)

적용 미디어는 `MapRequiredMediasByType`의 값이다(RT = 통화, NRT = 비통화, ANY = 공통). `확인`은 이번 설계 리뷰에서 핸들러를 읽어 확정한 항목이다. 나머지는 초안이며 구현 PR의 리뷰 대상이다. `TypeEmpty`는 `TypeListAll`에 없고 `MapRequiredMediasByType`에만 있으므로 `TestMetaCoversAllTypes`는 `TypeListAll`을 기준으로 한다.

| 타입 | Exposure | FlowKind | 미디어 | 비고 |
|---|---|---|---|---|
| amd | core | continue | RT | 옵션 `machine_handle`에 따라 hangup 유발 가능(구현 PR 확정 항목: FlowKind로 표현할지 수용할지) |
| answer | core | continue | RT | |
| ai_summary | internal | continue | NRT | `reference_id`가 런타임 id라 고객이 채울 수 없음. 확인: 스택 푸시 후 복귀 |
| ai_talk | sensitive | continue | ANY | LLM 비용 |
| ai_task | sensitive | continue | NRT | LLM 비용 |
| beep | core | continue | RT | |
| block | internal | continue | NRT | 카탈로그 요약이 internal grouping. 확인: 외부 해제까지 대기 후 이어짐 |
| branch | core | jump | ANY | 확인 |
| call | sensitive | continue | ANY | 중첩 `[]Action`으로 빌더 카탈로그에서 구조 제외(Meta 선언 값은 sensitive) |
| case_create | core | continue | ANY | CRM 데이터 생성, 구현 PR에서 재논의 |
| condition_call_digits | core | continue | RT | 확인: false는 `FalseTargetID` |
| condition_call_status | core | continue | RT | 확인 |
| condition_datetime | core | continue | ANY | 확인 |
| condition_variable | core | continue | ANY | 확인 |
| confbridge_join | internal | continue | RT | `confbridge_id`가 `connect`가 런타임에 만드는 값이라 고객이 고를 수 없음(카탈로그 요약 advanced/internal) |
| conference_join | core | continue | RT | |
| connect | sensitive | continue | RT | 확인: 아웃바운드 콜 생성, 복귀. 옵션 `relay_reason`에 따라 hangup 유발 가능 |
| conversation_send | sensitive | continue | ANY | 외부 전송 |
| digits_receive | core | continue | RT | |
| digits_send | core | continue | RT | |
| echo | internal | continue | RT | 내부/테스트용 |
| email_send | sensitive | continue | ANY | 중첩 첨부로 빌더 카탈로그에서 구조 제외 |
| external_media_start | internal | continue | RT | |
| external_media_stop | internal | continue | RT | |
| fetch | internal | continue | ANY | 임의 URL을 SSRF 방어 없이 호출(OQ8). v1 비노출. 확인: 스택 푸시 후 복귀 |
| fetch_flow | core | continue | ANY | 확인: 스택 푸시 후 복귀 |
| goto | internal | (모델 밖) | ANY | 확인: 조건부 점프, 에디터 `next_id` 무시 |
| hangup | core | terminate | RT | 확인 |
| message_send | sensitive | continue | ANY | 외부 전송 |
| mute | core | continue | RT | |
| play | core | continue | RT | |
| queue_join | core | continue | RT | 확인: 스택 푸시 후 복귀 |
| recording_start | core | continue | RT | 녹음 동의 성격은 구현 PR에서 재논의 |
| recording_stop | core | continue | RT | |
| sleep | core | continue | RT | |
| stop | core | terminate | ANY | 확인: `ActionFinish` 푸시 |
| stream_echo | internal | continue | RT | 내부/테스트용 |
| talk | core | continue | RT | |
| transcribe_start | sensitive | continue | RT | STT 비용 |
| transcribe_stop | core | continue | RT | |
| transcribe_recording | sensitive | continue | ANY | STT 비용 |
| variable_set | core | continue | ANY | |
| webhook_send | sensitive | continue | ANY | 외부 전송 |


## 부록 C. 구현 중 코드 리뷰로 확정된 보강 (VOIP-1573)

- 옵션 키는 구조체 json 이름과 정확히 일치하는 것만 받는다. `encoding/json`은 키를 대소문자 무시로 매칭하고 flow-manager도 그렇게 읽으므로, 변형 키(`Queue_ID`)가 리소스 비우기와 라벨 해석을 우회할 수 있었다. 변환(`resolveOption`)과 모델용 그래프 복원(`ReconstructGraph`) 양쪽에 적용한다. 응답에는 변환 단계만 `invalid_option`으로 보고하고(키는 40자로 줄여 싣는다), 복원은 모델용 그래프만 만들므로 조용히 버린다.
- `ref:"address"` 태그를 추가했다(`connect`, `message_send`, `conversation_send`류의 `commonaddress.Address` 필드). 주소 타입이 `tel`, `sip`, `email`, `line`, `whatsapp`이면 사용자 자신의 값이므로 유지하고, 그 외(agent, ai, conference 등 플랫폼 리소스, 미지정, 이후 추가될 타입)는 `target`을 비우고 `select_resource`로 알린다. 허용 목록 방식이라 새 타입은 기본적으로 리소스로 취급된다(`address.IsExternalEndpoint`). 드리프트락 테스트가 주소 필드의 태그 누락을 막는다.
- 맵형 참조(`branch.target_ids`)의 키가 UUID 형태이면 버린다. 키는 DTMF 같은 매칭 값이지 id가 아니다.
- 노출 불가 타입(알 수 없음, internal, 제외 타입)의 option은 모델용 그래프에 싣지 않는다.
- 조립 후 option이 요청 상한(4096B)을 넘으면(라벨이 36자 id로 치환되며 커질 수 있음) 맵 항목부터 잘라 `invalid_option`으로 알린다. 서버가 만든 초안이 다음 턴 검증에서 거부되는 일을 막는다.
- `draft_warnings`는 100개로 제한한다. label에 제어문자가 있는 노드는 파싱 단계에서 건너뛴다.
- 도달 판정 간선은 `next_id`, `ref:"action"` 값, 마지막이 아닌 열린 끝의 배열 인접 낙하이며 검증기와 레이아웃이 `successors`를 공유한다.
- `draft_discarded`(파싱 단계, `FlowParse`)는 응답이 쓸 수 없어 이전 초안을 유지했다는 경고다. 5절의 키 목록에 더해 모두 13개이며 프런트가 같은 13개를 문구로 매핑한다.
- 주소 필드는 프롬프트 카탈로그에서 리소스 필드와 분리해 안내한다(전화번호, SIP, 이메일 등은 사용자가 말한 값 그대로, 그 외 타입은 target을 null). 주소 비움 경고(`select_resource`)와 필수 주소 누락(`missing_required`)은 변환이 아니라 `ValidateDraft`가 내므로, 클라이언트가 초안을 돌려보내는 이후 턴에도 유지된다. 주소 type은 공백과 대소문자를 정규화해 받는다.
- 자기 자신을 가리키는 `next`는 버리고 열린 끝으로 처리한다. 실행기가 루프를 1000회에서 끊지만 `message_send`나 `webhook_send`는 그 횟수만큼 발송될 수 있고 의도한 사용이 없기 때문이다. 둘 이상의 노드를 도는 루프는 OQ9대로 v1에서 탐지하지 않는다.

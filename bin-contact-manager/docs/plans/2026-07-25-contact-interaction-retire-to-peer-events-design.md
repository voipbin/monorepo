# Contact Interaction: Retire contact_interactions, Read Through peer_events (Phase 1: Backend)

## 1. Decision and scope

대표님 결정 (2026-07-25): `bin-contact-manager`의 자체 `contact_interactions`
테이블/조회 로직(자동 매칭, ownership-period 기반 과거 이력, positive/negative
resolution 수동 보정, `InteractionListUnresolved` 미해결 큐)을 전면 포기하고,
`bin-timeline-manager`의 `peer_events` 조회 API(PR #1136, already merged)를
CRM interaction의 유일한 소스로 삼는다.

**손실 수용 항목 (명시적으로 대표님이 감수하기로 한 것, 재론하지 않음):**
- CRM 자격 필터 (agent/ai/ai_team/conference/extension/sip/빈 값 peer 노이즈 제거) — 사라짐. 클라이언트가 노이즈 처리 책임.
- Ownership-period 기반 과거 소유 이력 매칭 — 사라짐. `peer_events`는 Contact의 **현재** 주소 집합으로만 필터링.
- 수동 positive/negative resolution 보정 — 사라짐. 대응 개념 없음.
- `InteractionListUnresolved` 미해결 큐 — 사라짐. 대응 개념 없음.

**Phase 1 스코프 = 백엔드만.** 프론트엔드(square-talk/square-admin)는 별도
Phase 2로 명시적으로 분리됨 (대표님 지시, 2026-07-25). 이번 설계는 프론트엔드
파일을 변경하지 않는다.

**손실 수용 항목 추가 (Round 1 리뷰에서 확인됨, §5.1 참조):** `PeerEvent`에는
`interaction.Interaction.ReferenceType`에 대응하는 `Publisher`(합성 라벨:
"call"/"conversation_message"/"conversation") 외에, `EventType`이라는 별도
필드가 있으며 여기에는 원본 이벤트 세부 타입(예: `call_ringing`/`call_hangup`,
`conversation_message_created`)이 들어있다. 이 설계는 `EventType`을 응답
매핑에 포함하지 않는다. 즉 클라이언트는 프록시 응답에서 "call이었다"까지는
알 수 있지만 "어떤 call 이벤트였는지"는 구분할 수 없게 된다. 이는 기존
`interaction.Interaction`에도 없던 정보이므로 순수 신규 손실은 아니지만,
명시적으로 기록한다.

**신규 추가 항목 (Round 4 리뷰에서 확인됨, §5.1 참조):** 손실뿐 아니라
**신규로 생기는 것**도 있다. 프록시 전환 후 `reference_type` 필드에
`"conversation"`이라는, 기존 `contact_interactions` 파이프라인에서는 한
번도 생성된 적 없는 값이 나타난다 (`bin-contact-manager`는 conversation
생명주기 이벤트를 구독한 적이 없고, `call`/`conversation_message` 두 값만
써왔음). `bin-timeline-manager`는 conversation 생성/수정/삭제 이벤트를
`Publisher: "conversation"`으로 별도 적재하므로, 이 값이 프록시 응답에
그대로 노출된다. `reference_type`으로 분기하는 기존 클라이언트 코드가
있다면 처리되지 않은 새 값을 만나게 되므로, Phase 2(프론트엔드) 착수 시
반드시 확인해야 하는 항목으로 기록한다.

**본 설계에서 다루는 결정 (Phase 1과 Phase 2 사이 유예기간의 호환성):**
기존 REST 경로(`/contact_interactions`, `/contact_interactions/unresolved`,
`/contact_interactions/{id}`, `/contact_interactions/{id}/resolutions`,
`/service_agents/contact_interactions*`)를 **이번에 삭제하지 않는다.** 대신
list/get 경로는 내부적으로 `peer_events`를 조회해 `ContactManagerInteraction`과
동일한 JSON 응답 모양으로 변환해 반환한다 (§5 참조). unresolved/resolutions
경로는 대응 개념이 없으므로 명시적 에러로 응답한다 (§5.3). 이렇게 하면
Phase 2 착수 전까지 square-talk/square-admin이 완전히 깨지지 않는다 (list/get은
계속 동작, 단 노이즈 포함 + 과거이력/수동보정 기능 소실). 이 유예기간 동작은
과도기용이며 Phase 2에서 API 계약 자체를 재설계할 때 재검토 대상이다.

## 2. Current state (both sides, already verified in code)

### 2.1 bin-contact-manager (폐기 대상)
- `models/interaction/interaction.go` — `Interaction` struct
- `models/resolution/resolution.go` — `Resolution` struct
- `pkg/contacthandler/interaction.go` — `EventCallCreated`/`EventConversationMessageCreated` (CRM 자격 필터 + 정규화 + 적재), `crmIneligiblePeerTypes`
- `pkg/contacthandler/interaction_read.go` — `InteractionGet`/`InteractionList`/`interactionListByContact`(set-MINUS)/`interactionListByAddress`/`InteractionListUnresolved`
- `pkg/contacthandler/resolution.go` — `ResolutionCreate`/`ResolutionDelete`
- `pkg/dbhandler/interaction.go`, `pkg/dbhandler/resolution.go`, `pkg/dbhandler/address_ownership*.go` (ownership period 관련 일부는 Case 기능과도 공유 — §2.4 참조)
- `pkg/subscribehandler/callmanager.go`, `conversationmanager.go` — `processEventCallManagerCallCreated`/`processEventConversationManagerMessageCreated`가 위 `EventXxxCreated`를 호출
- `pkg/listenhandler/v1_interactions.go` — RPC 라우트 (`/v1/interactions*`)
- MySQL: `contact_interactions`, `contact_resolutions`(테이블명은 §3.6에서 확인됨) 테이블

### 2.2 bin-timeline-manager (기존 활용 대상, 변경 없음)
- `models/peerevent/{peerevent,request,response}.go` — `PeerEvent`/`PeerEventListRequest`/`PeerEventListResponse`
- `pkg/peereventhandler/peer_event.go` — `List(ctx, customerID, addrs []commonaddress.Address, pageToken, pageSize)`
- `pkg/dbhandler/peer_event_read.go` — ClickHouse `SELECT ... FROM peer_events WHERE customer_id=? AND (peer_type=? AND peer_target=?) OR ... ORDER BY timestamp DESC LIMIT ?`
- `pkg/listenhandler/v1_peer_events.go` — RPC 라우트 (`GET /v1/peer-events`)
- 적재: `pkg/subscribehandler/peer_event.go`의 `buildPeerEventRows`가 call-manager/conversation-manager 이벤트를 **CRM 자격 필터 없이** 전부 적재 (call/conversation_message/conversation 3종 Publisher)

### 2.3 bin-api-manager (이미 존재하는 peer_events 소비 경로, 변경 없음)
- `pkg/servicehandler/contact_peer_event.go` — `PeerEventList`/`ServiceAgentPeerEventList`/`resolvePeerAddresses`(contact_id → `Contact.Addresses` 또는 단일 peer_address)
- `server/contact_peer_events.go`, `openapi/paths/contact_peer_events/main.yaml`, `openapi/paths/service_agents/contact_peer_events.yaml` — 이미 `GET /contact_peer_events`, `GET /service_agents/contact_peer_events`로 노출 중

### 2.4 bin-contact-manager의 Case(kase) 기능 — 영향 없음 확인
`models/kase/kase.go`가 `ReferenceType`/`ReferenceID` 명명 규약을
`contact_interactions`와 공유하지만, 코드 검증 결과 `casehandler` 패키지
자체(business logic)는 `db.Interaction*` 함수를 **호출하지 않는다** (grep
재검증, Round 1 리뷰로 확인됨). Case는 자체 `peer`/`local`
(commonaddress.Address JSON) 컬럼을 독립적으로 가진 별도 엔티티이므로,
본 설계의 interaction 폐기는 Case 기능에 영향을 주지 않는다.

**단, 예외 하나:** `pkg/casehandler/casenote_isolation_test.go`
(`Test_CaseNote_NeverLeaksIntoInteractionList`)는 `casehandler` 비즈니스
로직을 거치지 않고 `db.InteractionCreate`/`db.InteractionList`를 **직접**
호출해 회귀 픽스처를 구성한다. §3.2에서 폐기/재작성하는 것은
`contacthandler.InteractionList`(호출 스택 상위 레이어)이며, `dbhandler`
레벨의 `InteractionCreate`/`InteractionList` 함수와 `contact_interactions`
테이블 자체는 §3.6에 따라 이번 Phase 1에서 남겨둔다. 따라서 이 테스트는
컴파일도 되고 동작도 그대로 유지된다 — 다만 이 사실이 자명하지 않으므로
명시한다: **"§3.2가 재작성하는 것은 contacthandler 레이어뿐이고, dbhandler
레벨 프리미티브는 §3.6까지 그대로 남는다"**는 점을 구현 착수 시 반드시
전제로 삼을 것 (반대로 오해하면 이 테스트가 깨진다).

### 2.5 cross-service 소비자 인벤토리 (Round 1 리뷰로 추가 확인됨)
`ContactV1InteractionList`(bin-common-handler RPC wrapper)의 알려진
호출자는 다음 세 곳이다.
- `bin-contact-manager/pkg/listenhandler/v1_interactions.go` (자기 자신의 RPC 핸들러 경유, §3.2 대상)
- `bin-api-manager/pkg/servicehandler/interaction.go`, `serviceagent_interaction.go` (§3에서 다룸)
- **`bin-ai-manager/pkg/aicallhandler/tool_insight.go`** — AI 도구
  (`interaction_list` LLM tool)가 `contact_id` 필터와 `peer_type+peer_target`
  필터 두 가지 호출 형태로 이 RPC를 직접 호출한다. `address_id`나 무필터
  `since` 모드는 사용하지 않는다.

`bin-ai-manager`의 두 호출 형태는 모두 §3.2의 peer_events 프록시 매핑
대상에 포함되므로 기능적으로는 계속 동작한다. 다만 **동작 성격이 조용히
바뀐다**: 노이즈(agent/AI/conference/sip) 포함, ownership-period 기반
과거 이력 매칭 소실. 이 서비스는 본 설계의 직접 변경 대상은 아니지만,
동작 변화가 있는 실제 소비자이므로 여기 기록한다. bin-ai-manager 팀/오너에게
별도 통지가 필요한지는 구현 착수 시점에 재확인 (design doc 범위 밖이지만
누락하지 않도록 기록).

## 3. What changes

### 3.1 적재 경로 폐기
- `pkg/subscribehandler/callmanager.go`/`conversationmanager.go`의
  `processEventCallManagerCallCreated`/`processEventConversationManagerMessageCreated`가
  더 이상 `h.contactHandler.EventCallCreated`/`EventConversationMessageCreated`를
  호출하지 않도록 제거. `bin-contact-manager`는 call-manager/conversation-manager
  이벤트 구독 자체를 끊는다 (다른 목적의 구독이 없다면 큐 바인딩도 제거;
  §3.7에서 재확인).
- **`EventCallCreated`/`EventConversationMessageCreated` 함수 자체도 삭제한다
  (Round 6 리뷰로 확인됨, 누락 보완).** 위 구독 호출 제거만으로는 부족하다 —
  이 두 함수(`pkg/contacthandler/interaction.go`)는 `isCRMEligiblePeer`를
  직접 호출하는 유일한 소비자이므로, `crmIneligiblePeerTypes`/
  `isCRMEligiblePeer`만 지우고 이 두 함수를 남기면 미정의 식별자 참조로
  컴파일이 깨진다. 따라서 `crmIneligiblePeerTypes`/`isCRMEligiblePeer`와
  `EventCallCreated`/`EventConversationMessageCreated`를 **한 묶음으로**
  삭제한다.
- `crmIneligiblePeerTypes`/`isCRMEligiblePeer`는 CRM 프로젝션 전용 로직이었으므로
  통째로 삭제 (위와 동일한 커밋 단위로 처리).

### 3.2 조회 경로를 peer_events 프록시로 교체
`contacthandler.InteractionList`(list, §5.3의 필터 3종+무필터)의 내부
구현을 `bin-timeline-manager`의 `peer_events` read API를 호출하도록
재작성한다.

**`InteractionGet`(단건 조회)은 프록시 대상이 아니다 (Round 4 리뷰로 정정):**
`peerevent.PeerEvent`에는 row 단위 안정적 ID가 없다 — `ReferenceID`는
origin call/message id일 뿐이고, 하나의 call에 대해 여러 row(예:
`call_ringing`, `call_hangup`)가 같은 `ReferenceID`를 공유한다. 따라서
"interaction id X에 대응하는 peer_event 한 건"을 조회하는 것은 구조적으로
불가능하다. `contacthandler.InteractionGet`은 §3.2에서 "재작성"하지 않고
**그대로 삭제**한다 (§5.3의 `GET /contact_interactions/{id}` → 410 Gone과
정합). `pkg/listenhandler/v1_interactions.go`의
`processV1InteractionsIDGet` 라우트도 함께 삭제한다 (§3.3/§3.4가 resolution/
unresolved 라우트를 명시적으로 삭제 대상에 넣은 것과 동일한 처리).

- 신규: `bin-common-handler/pkg/requesthandler`에 `TimelineV1PeerEventList`
  wrapper는 **이미 존재** (PR #1136에서 추가됨, `bin-api-manager`가 사용 중).
  `bin-contact-manager`가 동일 wrapper를 재사용할 수 있도록 import만 추가.
- `contacthandler.InteractionList`의 4가지 필터 모드 매핑:
  - `peer_type + peer_target` → `TimelineV1PeerEventList`에 단일
    `commonaddress.Address{Type, Target}` 전달.
  - `contact_id` → 먼저 `h.db.ContactGet`으로 tenant 확인 후
    `Contact.Addresses`(현재 시점 주소만, ownership period 없음)를
    `[]commonaddress.Address`로 변환해 전달. **이것이 bin-api-manager의
    `resolvePeerAddresses`와 동일한 패턴** — 로직 중복이 아니라, 두 서비스가
    같은 변환을 각자의 데이터 소스(bin-api-manager는 HTTP 경계 밖에서
    Contact를 이미 들고 있고, bin-contact-manager는 자체 DB에서 직접 조회)
    기준으로 수행하는 것이므로 공유 헬퍼로 뽑아내지 않는다 (각 서비스
    내부에서 자기 소유 데이터로 직접 변환).
  - `address_id` → `h.db.AddressGet`으로 (type, target) 조회 후 단일 주소로 전달.
    **비대칭성 명시 (Round 1 리뷰로 확인됨): 이 필터 모드는 bin-api-manager의
    `resolvePeerAddresses`에는 아예 존재하지 않는다** —
    `resolvePeerAddresses`의 switch문은 `contact_id`와 단일
    `peer_type+peer_target` 두 케이스만 지원하고 그 외는 `INVALID_FILTER`
    에러다. `address_id` 지원은 bin-contact-manager의 기존
    `GET /contact_interactions` 계약(3-필터 모드)을 유예기간 동안 유지하기
    위해 **contact-manager 내부에서만** 새로 만드는 변환이며,
    bin-api-manager의 peer_events 소비 패턴(§3.2 서두 참조)과 "같은
    패턴"이라기보다 "같은 목적지(peer_events)로 가는 별도의, 이 계약에서만
    필요한 변환"이다. 공유 헬퍼로 뽑지 않는 이유(§4)와 별개로, 이 모드는
    Phase 2에서 API 계약을 재설계할 때 폐기 여부를 재검토할 후보 1순위다.
  - `since`만(무필터) → **peer_events read API는 무필터 모드를 지원하지
    않는다** (최소 1개 peer address 필수, `peereventhandler.List`의
    `errors.New("at least one peer address is required")` 참조). 이 모드는
    호출자에게 `INVALID_FILTER` 에러 반환 (§5.3, 대응 불가 명시).
- 응답 변환: `peerevent.PeerEvent` → `interaction.Interaction` 유사 shape로
  매핑 (§5.1 상세). **`interaction.Interaction`의 `ID`/`TMCreate`/
  `TMInteraction` 필드는 peer_events에 대응 값이 없다** (peer_events는 합성
  ID가 없고 `Timestamp` 하나만 있음). 이 갭은 §5.1에서 명시적으로 처리.

### 3.3 Resolution 폐기
- `contacthandler.ResolutionCreate`/`ResolutionDelete` 삭제.
- `pkg/listenhandler/v1_interactions.go`의 `processV1InteractionsResolutionsPost`/
  `processV1InteractionsResolutionsIDDelete` 라우트 삭제 — 즉 contact-manager
  쪽 RPC 핸들러 자체가 없어진다.
- 상위(bin-api-manager)의 `POST /contact_interactions/{id}/resolutions`,
  `DELETE /contact_interactions/{id}/resolutions/{rid}` (및 service_agents
  대응)는 **경로 자체는 유예기간 동안 유지하되 항상 410 Gone 반환** (§5.3).
  **중요 (Round 1 리뷰로 확인됨): 이것은 저절로 되지 않는다.** contact-manager
  쪽 RPC 핸들러가 사라지면 `h.reqHandler.ContactV1ResolutionCreate`/
  `ContactV1ResolutionDelete` 호출은 RPC-not-found 또는 timeout 에러로
  실패하며, 이는 410이 아니라 500류 에러로 응답된다. 따라서 bin-api-manager
  쪽에도 명시적 코드 변경이 필요하다 (§3.5 신규 섹션 참조).

### 3.4 InteractionListUnresolved 폐기
- `contacthandler.InteractionListUnresolved` 삭제 (contacthandler 레이어만;
  §2.4가 명시하듯 dbhandler 레벨 프리미티브와 테이블은 §3.6까지 유지).
  **누락 보완 (Round 4 리뷰로 확인됨): 이 삭제만으로는 컴파일이 깨진다.**
  `pkg/listenhandler/v1_interactions.go`의 `processV1InteractionsUnresolvedGet`
  (RPC 라우트 `/v1/interactions/unresolved`)이 이 함수를 직접 호출하고
  있으므로, §3.3이 resolution RPC 라우트를 명시적으로 삭제 대상에 넣은 것과
  동일하게 **이 라우트도 함께 삭제한다** (contact-manager 자신의 RPC
  핸들러이므로, 이 경로를 호출하는 상위 서비스가 있다면 §3.5의 빈 리스트
  처리로 흡수한다 — 실제로 유일한 상위 호출자는 bin-api-manager이며 §3.5가
  이미 그쪽을 빈 리스트로 처리하도록 다룬다).
- 대응하는 REST 경로는 유예기간 동안 유지, 항상 빈 리스트(`result: []`,
  `next_page_token: ""`) 반환 (§5.3, 구현 위치는 §3.5) — 410보다 빈 목록을
  택한 이유: 상담사 화면이 "미해결 큐가 비어있다"로 오인하는 것이 "화면이
  에러로 깨지는 것"보다 Phase 2 착수 전까지의 사용자 경험상 덜 파괴적이라고
  판단 (표시상 오해의 소지는 있으나 크래시보다 낫다는 절충 — Phase 2에서 이
  화면 자체를 걷어낼 때 해소됨).

### 3.5 bin-api-manager 측 변경 (신규 — Round 1 리뷰로 범위 명확화, Round 2 리뷰로 short-circuit 지점 확정)
§3.3/§3.4가 요구하는 410/빈-리스트 동작은 bin-api-manager의
`server`/`servicehandler` 레이어 코드 변경 없이는 실현되지 않는다. 이
서비스는 이번 Phase 1의 "백엔드" 스코프에 포함되며(프론트엔드가 아님),
다음을 명시적으로 변경한다.

**공유 헬퍼 `interactionGet` 문제 (Round 2 리뷰로 확인됨):**
`pkg/servicehandler/interaction.go`의 private 헬퍼 `interactionGet(ctx,
customerID, id)`(`h.reqHandler.ContactV1InteractionGet` RPC 호출)는 4개
호출부가 공유한다 — `InteractionGet`, `ServiceAgentInteractionGet`,
`ResolutionCreate`(tenant 소유권 검증을 위해 먼저 호출), `ServiceAgentResolutionCreate`(동일).
이 공유 구조 때문에 "InteractionGet/ServiceAgentInteractionGet은 410,
ResolutionCreate/Delete는 별도로 410"처럼 두 그룹을 독립적으로 취급하면
모순이 생긴다: `interactionGet` 자체를 short-circuit시키면 Resolution 쪽도
같이 short-circuit되어 "RPC 미호출로 관측성 확보"라는 애초 근거가 흐려지고,
반대로 공개 함수(`InteractionGet` 등)에만 체크를 추가하면 Resolution 쪽은
여전히 `interactionGet`을 거쳐 RPC를 호출해버려 의도(§3.3의 "RPC를 아예
호출하지 않는다")가 깨진다. 따라서 이 설계는 다음과 같이 **호출부별로
short-circuit 지점을 명시적으로 분리**한다 (Round 2 리뷰가 지적한
Create/Delete의 실제 호출 형태 차이도 반영).

- **`InteractionGet`/`ServiceAgentInteractionGet`**: 공유 `interactionGet`
  헬퍼 자체의 최상단에서 즉시 신규 에러(예: `serviceerrors.ErrInteractionRetired`,
  410 매핑)를 반환하도록 변경한다. RPC(`ContactV1InteractionGet`)는 호출하지
  않는다. 이 헬퍼를 다른 곳에서 더 이상 필요로 하지 않으므로(§3.3/§3.4 폐기
  대상 전부가 이 헬퍼 또는 그 직접 소비자), 공유 헬퍼 레벨에서 막는 것이
  가장 단순하고 새 우회 경로를 만들지 않는다.
- **`ResolutionCreate`/`ServiceAgentResolutionCreate`**: 위 변경의 결과로
  `interactionGet`을 그대로 호출하면 자동으로 410을 받게 되므로 **별도의
  독립 short-circuit을 추가하지 않는다** — 공유 헬퍼가 유일한 진입점이
  되도록 하여 "RPC를 아예 호출하지 않는다"는 §3.3의 의도를 그대로 만족시킨다.
  단, 이 경로에서 `interactionGet`이 원래 수행하던 **tenant 소유권 검증은
  사라진다** (410 자체가 데이터를 노출하지 않으므로 안전하다고 판단하지만,
  이 트레이드오프를 명시한다: 이전에는 다른 tenant의 interaction ID로
  resolution을 시도하면 "찾을 수 없음"으로 구분되던 것이, 이제는 모든
  interaction ID에 대해 무조건 410이 되어 그 구분이 없어진다. 기능이 폐기된
  경로이므로 허용 가능한 손실로 판단). **추가 명시 (Round 3 리뷰로 확인됨):**
  이 short-circuit은 `hasPermission` 검사보다 먼저 실행되므로, 권한이 없는
  호출자(예: `PermissionCustomerAdmin|PermissionCustomerManager`가 없는
  계정)도 오늘의 `403 PermissionDenied` 대신 `410 Gone`을 받게 된다. 410은
  아무 데이터도 노출하지 않으므로 안전하지만, "권한 체크 자체가 이 경로에서
  더 이상 실행되지 않는다"는 점을 명시적으로 인지하고 진행한다 (향후
  누군가 `hasPermission`이 여전히 평가되는 줄 알고 코드를 수정하지 않도록).
- **`ResolutionDelete`/`ServiceAgentResolutionDelete`**: 이 두 함수는
  **오늘 시점에 `interactionGet`을 호출하지 않는다** (Round 2 리뷰로 확인,
  Create와 호출 형태가 다름) — `h.hasPermission`만 거친 뒤 바로
  `ContactV1ResolutionDelete` RPC를 호출한다. 따라서 이 둘은 각 함수 본문
  최상단에 **독립적인** short-circuit(동일한 신규 410 에러)을 추가해야 한다.
  `interactionGet` 변경으로는 이 경로가 자동으로 막히지 않는다.
- **`InteractionListUnresolved`/`ServiceAgentInteractionListUnresolved`**:
  §3.4대로 각 함수 본문에서 RPC 호출 없이 빈 결과를 직접 반환.
- 이 변경은 `bin-api-manager`의 CLAUDE.md가 명시하는 Two-Level Handler
  Pattern(`resourceGet()` 내부 헬퍼 + 공개 `ResourceGet()` 권한체크 래퍼)을
  그대로 따른다 — `server/` 레이어는 변경하지 않고 `servicehandler/` 레이어의
  반환값/에러만 바꾼다 (OpenAPI 라우트/타입은 §6대로 그대로 유지).
- 신규 에러 타입은 `bin-common-handler/models/errors`에 410 Gone에 대응하는
  헬퍼가 있는지 구현 착수 시 확인 후, 없으면 추가한다. 로그/메트릭에서 이
  신규 에러가 기존 `ErrPermissionDenied`/`ErrNotFound`와 구분되는지는 구현
  착수 시 확인 (경미한 항목, 블로커 아님).

### 3.6 DB 정리
- `contact_interactions`, `contact_resolutions`(정확한 테이블명은 구현 착수
  시 `pkg/dbhandler/interaction.go`/`resolution.go`의 `const ...Table` 값으로
  재확인) MySQL 테이블은 **이번 Phase 1에서 DROP하지 않는다.** 애플리케이션
  코드가 더 이상 쓰지 않게 된 직후 즉시 스키마를 지우면 롤백 여지가 없다.
  별도 후속 Alembic 마이그레이션으로, 이번 변경이 최소 1회 배포 주기 동안
  문제없이 운영된 것을 확인한 뒤 제거한다 (본 설계 문서 범위 밖, 후속 작업으로
  명시).

### 3.7 구독 큐 정리
`bin-contact-manager`가 call-manager/conversation-manager 큐를 구독하는
목적이 순수히 interaction 프로젝션 하나뿐이었는지 확인 필요. 만약 다른
목적(예: 향후 Case 자동 연결 등)으로도 같은 큐를 구독한다면 큐 바인딩 자체는
유지하고 핸들러 본문만 no-op 처리. 구현 착수 시 `pkg/subscribehandler/main.go`의
큐 등록부를 재확인해 확정한다.

### 3.8 인터페이스/디스패치 테이블 정리 (신규 — Round 6 리뷰로 확인된 누락)
§3.1~§3.4가 삭제하는 함수들(`EventCallCreated`, `EventConversationMessageCreated`,
`InteractionGet`, `ResolutionCreate`, `ResolutionDelete`,
`InteractionListUnresolved`)은 각 함수 본문만 지워서는 끝나지 않는다.
다음 두 지점에서 이 함수들의 **선언/참조**가 남아있으면 그 자체로
컴파일이 깨진다 (Round 6 리뷰가 처음 확인, 이전 라운드가 놓친 항목).

- **`pkg/contacthandler/main.go`의 `ContactHandler` 인터페이스**: 위 6개
  함수 전부가 이 인터페이스의 메서드 시그니처로 선언되어 있다. 구현체
  `contactHandler`가 인터페이스를 만족하려면, 함수 삭제와 동시에 이
  인터페이스에서도 해당 6개 시그니처를 제거해야 한다. (`EventCallCreated`/
  `EventConversationMessageCreated`는 §3.1, `InteractionGet`은 §3.2,
  `ResolutionCreate`/`ResolutionDelete`는 §3.3, `InteractionListUnresolved`는
  §3.4 각각의 삭제 대상과 1:1 대응.)
- **`pkg/listenhandler/main.go`의 요청 디스패치 switch/case 테이블**:
  `processV1InteractionsIDGet`, `processV1InteractionsResolutionsPost`,
  `processV1InteractionsResolutionsIDDelete`,
  `processV1InteractionsUnresolvedGet` 4개 라우트 핸들러 함수를 이름으로
  직접 호출하는 case 분기가 여기 있다. §3.2~§3.4가 이 4개 함수 본문을
  삭제할 때, 이 switch의 해당 case 분기도 반드시 함께 제거한다.
- 구현 착수 시 체크리스트: 각 함수를 삭제하기 전, `pkg/contacthandler/main.go`와
  `pkg/listenhandler/main.go`에서 해당 이름을 grep해 두 지점 모두 정리했는지
  확인한다. 이 확인 없이 `go build`가 성공했다고 가정하지 않는다 (실제로
  `go build`가 이 누락을 즉시 잡아내지만, 설계 문서 자체가 작업 항목을
  누락하면 구현자가 놓치기 쉽다는 것이 Round 6의 지적이었다).
- mock 재생성(`go generate ./pkg/contacthandler/...`)도 인터페이스 변경
  이후 다시 실행해 `mock_main.go`를 갱신한다 (기존 §7 검증 계획의
  `go generate ./...` 단계에 포함되나, 순서상 인터페이스 정리가 먼저
  끝나야 mock 생성이 성공하므로 명시).

## 4. Why not a shared helper for the Address-resolution duplication (§3.2)

bin-api-manager의 `resolvePeerAddresses`와 bin-contact-manager의 신규
변환 로직은 표면적으로 같은 일(Contact → `[]commonaddress.Address`)을
하지만, 억지로 공유 패키지로 뽑아내지 않는다:
- 두 서비스는 이미 서로 다른 소스에서 `Contact`/주소를 들고 있다
  (bin-api-manager는 REST 파라미터 경유, bin-contact-manager는 자기 DB
  직접 조회). 공유 헬퍼를 만들면 오히려 두 서비스가 같은 함수를 통해
  서로의 내부 타입에 결합되는 잘못된 방향의 결합이 생긴다.
- 변환 로직 자체는 5줄 내외의 단순 dedup 루프이므로, 중복 비용보다
  잘못된 추상화 비용이 크다고 판단.

## 5. Response shape mapping (peer_events → interaction 응답 호환)

### 5.1 필드 매핑
| `interaction.Interaction` 필드 | `peerevent.PeerEvent` 소스 | 비고 |
|---|---|---|
| `id` | **없음** | peer_events는 row별 UUID가 없음. `uuid.Nil`로 채우거나 필드 자체를 응답에서 생략 — 구현 착수 시 결정 (권장: 생략, 클라이언트가 `id`에 의존한 캐시 키/딥링크를 쓰고 있다면 Phase 2에서 별도 처리 필요하므로 프론트 영향 재확인 후 확정) |
| `customer_id` | `CustomerID` | 그대로 |
| `direction` | `Direction` | 그대로 |
| `peer` | `Peer` | 그대로 (둘 다 `commonaddress.Address`) |
| `local` | `Local` | 그대로 |
| `reference_type` | `Publisher` | **주의**: `PeerEvent.Publisher`는 timeline-manager 쪽 합성 라벨("call"/"conversation_message"/"conversation")이며 `interaction.Interaction.ReferenceType`과 동일한 어휘를 부분적으로만 공유한다. **정정 (Round 4 리뷰로 확인됨): `"conversation"` 값은 완전히 신규다.** 기존 `contacthandler.EventCallCreated`/`EventConversationMessageCreated`는 `ReferenceType`에 `"call"`/`"conversation_message"` 두 값만 써왔고 `"conversation"`을 생성한 적이 없다(bin-contact-manager는 conversation 생명주기 이벤트 자체를 구독한 적이 없음). 반면 `bin-timeline-manager`의 `buildPeerEventRows`는 `conversation_created`/`_updated`/`_deleted` 이벤트를 `Publisher: "conversation"`으로 적재한다. 즉 프록시 전환 후 클라이언트는 `reference_type: "conversation"`이라는, 과거에 한 번도 본 적 없는 신규 값을 받게 된다. 이는 §1의 손실 수용 항목이 아니라 **신규 추가 항목**이므로 §1에 명시한다 (아래 참조). |
| `reference_id` | `ReferenceID` | 그대로 |
| `tm_interaction` | **없음** | peer_events에 별도 origin-event-time 컬럼 없음. `Timestamp`로 대체 (근사치, 완전히 동일하지 않을 수 있음 — peer_events의 `Timestamp`는 적재 시점 기준일 가능성, 구현 착수 시 `buildPeerEventRows` 재확인) |
| `tm_create` | `Timestamp` | 그대로 매핑 (페이지네이션 커서 용도 동일) |

### 5.2 Pagination
`peereventhandler.List`의 `pageToken`은 이미 `commonutil.ISO8601Layout`
포맷의 timestamp 문자열이며, 기존 `ContactV1InteractionList`
(`bin-common-handler/pkg/requesthandler/contact_interactions.go`)의 cursor
포맷(`"2006-01-02T15:04:05.000000Z"`, `TMCreate` 기준)과 정확히 동일한지
구현 착수 시 **반드시** 바이트 단위로 대조한다. **강조 (Round 1 리뷰로
격상됨): 포맷이 어긋나면 에러가 나지 않는다.** 두 값 모두 opaque 문자열로
그대로 전달되는 구조라, 포맷 불일치는 크래시가 아니라 "다음 페이지가
조용히 잘못된 위치에서 시작하거나 영구 루프/누락"으로 나타나는 silent
failure다. "구현 착수 시 확인"이 아니라 **단위 테스트로 포맷 동일성을
고정(assert)해야 하는 항목**으로 취급한다.

### 5.3 대응 불가 경로의 명시적 처리
| 기존 경로 | Phase 1 유예기간 동작 |
|---|---|
| `GET /contact_interactions` (`peer_type+peer_target`/`contact_id`/`address_id`) | peer_events 프록시로 정상 동작 (노이즈 포함, 과거이력 없음) |
| `GET /contact_interactions` (필터 0개, `since`만) | `400 INVALID_FILTER` — "unfiltered mode is no longer supported; provide contact_id or peer_type+peer_target" |
| `GET /contact_interactions/unresolved` | `200`, 항상 빈 리스트 |
| `GET /contact_interactions/{id}` | **불가능** — peer_events에 개별 조회용 안정적 ID가 없음. `410 Gone` "single-interaction lookup is retired; use GET /contact_interactions with a peer/contact filter" |
| `POST /contact_interactions/{id}/resolutions` | `410 Gone` |
| `DELETE /contact_interactions/{id}/resolutions/{rid}` | `410 Gone` |
| `service_agents/contact_interactions*` 전체 | 위와 동일 패턴 |

`cerrors` 패키지에 410 Gone에 해당하는 헬퍼가 없다면 신규 추가 필요
(구현 착수 시 `bin-common-handler/models/errors`에서 기존 상수 재확인).

## 6. Non-goals (explicitly out of scope for Phase 1)

- square-talk/square-admin 프론트엔드 변경 — Phase 2.
- `contact_interactions`/`contact_resolutions` 테이블 DROP — 후속 마이그레이션.
- OpenAPI 스펙에서 `/contact_interactions/*` 경로 자체를 제거하는 것 — 유예기간
  동안 경로 존치 (응답 스펙은 그대로, 동작만 변경).
- `peer_events`에 과거 이력/수동보정/미해결 큐를 재구현하는 것 — 대표님이
  명시적으로 포기하기로 결정.

## 7. Verification plan

**순서 주의 (본 Revision 리뷰 Round 4로 확인됨): 아래 단계는 순서대로
실행해야 한다.** `bin-contact-manager`/`bin-api-manager`/`bin-ai-manager`의
`go.mod`는 전부 `replace monorepo/bin-common-handler => ../bin-common-handler`
(로컬 파일시스템 replace)를 쓴다. 즉 각 서비스의 `go mod vendor`는
그 시점에 디스크에 있는 `bin-common-handler` 코드를 그대로 스냅샷한다.
따라서 (a) `bin-common-handler` 소스 편집을 먼저 끝내고 → (b)
`bin-common-handler` 자체 검증 워크플로우를 실행해 확정한 뒤 → (c)
그제서야 `bin-contact-manager`/`bin-api-manager`/`bin-ai-manager` 각각의
`go mod tidy/vendor/generate/test/lint`를 실행한다(각자 최신
`bin-common-handler`를 다시 vendor) → (d) RST 재빌드. 이 순서를 어기면
오래된 `bin-common-handler` 시그니처가 그대로 vendor되어, 하위 서비스에서
혼란스러운 컴파일 에러가 나거나 최악의 경우 stale mock으로 테스트가
조용히 통과해버릴 수 있다.

- `bin-contact-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
  — §3.8 체크리스트(`ContactHandler` 인터페이스, `listenhandler` 디스패치
  테이블 정리)를 코드 삭제와 같은 커밋에서 완료한 뒤 이 워크플로우를 실행할 것.
- `bin-api-manager`: 동일 워크플로우 (410/빈 리스트 응답 경로에 대한 신규
  테스트 포함 — 기존 `contact_interactions_test.go`/
  `service_agents_contact_interactions_test.go` 갱신)
- 기존 `interaction_read_test.go`, `interaction_crm_eligibility_test.go`,
  `resolution_test.go`, `interaction_test.go`(contacthandler) 는 대상 로직이
  삭제되므로 함께 삭제 또는 신규 프록시 동작에 맞게 재작성.
- `casenote_isolation_test.go`(`Test_CaseNote_NeverLeaksIntoInteractionList`)는
  §2.4에서 확인했듯 `dbhandler.InteractionCreate`/`InteractionList`
  프리미티브 자체(시그니처 포함)는 이번 Phase 1에서 변경되지 않으므로 영향
  없음 — 재작성 대상은 `contacthandler.InteractionList`(상위 레이어)뿐이다.
  다만 §3.6 후속 마이그레이션(테이블 DROP)이 실행되는 시점에는 이 테스트가
  깨지므로, 그 후속 작업 착수 시 반드시 함께 정리한다.
- **`bin-ai-manager` 검증 (정정 — 본 Revision 리뷰 Round 4로 확인됨,
  §8.3 반영):** 원래 여기서는 "이번 Phase 1의 변경 대상이 아니므로
  참고용 회귀 테스트만"으로 적혀 있었으나, §8.2/§8.3에서 `tool_insight.go`의
  실제 코드 수정이 이 설계의 필수 스코프에 포함됐으므로 이 문구는
  **더 이상 맞지 않는다.** `bin-ai-manager`도 코드가 실제로 바뀌는
  서비스이므로 다른 서비스와 동일하게 전체 검증 워크플로우
  (`go mod tidy && go mod vendor && go generate ./... && go test ./...
  && golangci-lint run -v --timeout 5m`)를 **필수로** 실행한다. 기존
  `tool_insight_test.go`가 새 `peerevent.PeerEvent` shape 기준으로
  통과하는지 확인하는 것이 검증의 핵심이지, 선택적 참고 확인이 아니다.
- **§8 반영 후 추가 항목 (본 Revision 리뷰 Round 3으로 확인됨, 원 §7이
  놓쳤던 두 가지):**
  1. `bin-common-handler`: §8.2가 이 공유 패키지의 RPC wrapper 4개와
     인터페이스/mock을 삭제하므로, 이 서비스에서도 표준 검증 워크플로우
     (`go mod tidy && go mod vendor && go generate ./... && go test ./...
     && golangci-lint run -v --timeout 5m`)를 실행한다. 모노레포
     `docs/workflows/special-cases.md`가 명시하듯 `bin-common-handler`
     변경은 **34개 전체 서비스**에 영향을 미칠 수 있으므로, 테스트 전
     `go clean -testcache`를 먼저 실행하고, bin-contact-manager/
     bin-api-manager/bin-ai-manager 외에 이 4개 wrapper를 참조하는 다른
     소비자가 없는지 모노레포 전체를 한 번 더 grep해 확인한다 (Round
     1~4 리뷰로 현재 시점 기준 없음을 확인했으나, 구현 시점 코드
     상태로 재확인 필요). **위 순서 주의대로, 이 워크플로우는
     bin-contact-manager/bin-api-manager/bin-ai-manager 검증보다
     먼저 실행한다.**
  2. **RST 문서 갱신**: `bin-api-manager/docsdev/source/`의
     `contact_peer_event_overview.rst`(47번째 줄의 "AI Implementation
     Hint", 129번째 줄의 Troubleshooting 항목, 26~43번째 줄의 ASCII
     다이어그램에서 `GET /contact_interactions`를 `GET /contact_peer_events`
     보다 우선 사용하도록 안내하고 있음, 본 Revision 리뷰 Round 4로
     실제 파일 확인)가 §8.1-2로 삭제되는 엔드포인트를 안내하게 된다.
     루트 `CLAUDE.md`의 "RST Docs Sync" 규칙(사용자 대면 동작 변경 시
     RST 갱신 + 재빌드 + `git add -f build/` 필수)에 따라, 구현 착수 시
     이 파일에서 `/contact_interactions` 언급을 제거하거나
     `/contact_peer_events` 안내로 대체하고 `docsdev` 재빌드까지
     완료한다. (`docsdev/source` 전체 grep 결과 `ai_struct_ai.rst`의
     `get_contact_interactions`는 REST 엔드포인트가 아니라
     `bin-ai-manager` 자체 AI 도구 이름이라 이번 삭제와 무관하며 수정
     대상이 아님 — Round 4로 확인.) 이는 §8.2의 OpenAPI 스펙 삭제와
     짝을 이루는 필수 후속 작업이며, 생략 시 공개 문서(docs.voipbin.net)가
     존재하지 않는 API를 안내하게 된다.

## 8. Revision 1 (2026-07-25, post-closure, 대표님 지시): 응답 호환성 제약 폐기 — 완전 단순화

**트리거**: §1~§7(원 설계, 8라운드 리뷰로 승인 완료)은 "Phase 2 착수 전까지
square-talk/square-admin이 완전히 깨지지 않아야 한다"는 전제 위에 세워졌다.
이 전제 때문에 §3.2/§3.5/§5 전체가 "응답 필드를 기존 스키마에 맞춰 가공",
"삭제된 기능의 REST 경로는 남기되 410/빈 리스트로 short-circuit" 같은
호환성 장치로 채워져 있었다. 대표님이 이번에 그 전제 자체를 철회했다:
**"깨져도 괜찮다. 현재 쓰는 사람이 우리뿐이고, Admin 쪽 깨짐은 무시해도
된다."** 이어서 대표님은 두 옵션(①호환 장치 유지하되 가공 없이 원본
그대로 통과, ②호환 장치 전부 제거 + 경로 자체 삭제) 중 **②를 선택**했다.

**중요**: 아래는 §1~§7을 대체하는 것이 아니라 **위에 얹는 신규 결정**이다.
원 설계의 리뷰 이력(무엇을 몇 라운드에 왜 고쳤는지)은 감사 추적으로서
그대로 남긴다. §1~§7을 읽는 구현자는 **본 §8이 그 내용을 override한다**는
점을 알아야 한다 — 특히 §3.2의 "응답 변환"/"필드 매핑" 관련 문단, §3.5
전체, §5 전체, §6의 "OpenAPI 경로 존치" 항목은 **본 §8로 대체(superseded)**
되었다.

### 8.1 무엇이 바뀌는가

1. **호환성 응답 가공 폐기**: `contacthandler.InteractionList`는 더 이상
   `peerevent.PeerEvent` → `interaction.Interaction` 필드 매핑을 하지
   않는다. `peer_events` read API의 응답(`peerevent.PeerEvent` 그대로)을
   가공 없이 그대로 반환한다. §5.1의 필드 매핑 표, §5.2의 pagination
   포맷 동일성 강제(cursor 포맷을 기존 계약에 맞출 필요가 없어짐), §5.3의
   "대응 불가 경로" 표는 전부 무의미해진다 — 새 계약에서는 애초에 그런
   경로들이 존재하지 않기 때문 (§8.2 참조).
2. **기존 REST 경로/스키마를 전부 삭제**: 다음을 OpenAPI 스펙에서도
   통째로 제거한다 (§6이 "유예기간 동안 존치"라고 했던 것을 철회).
   - `GET /contact_interactions`, `GET /contact_interactions/unresolved`,
     `GET /contact_interactions/{id}`,
     `POST /contact_interactions/{id}/resolutions`,
     `DELETE /contact_interactions/{id}/resolutions/{rid}`
   - `GET /service_agents/contact_interactions` 및 동일 패턴의
     unresolved/{id}/resolutions 하위 경로 전부
   - 대응하는 `openapi/paths/contact_interactions/*.yaml`,
     `openapi/paths/service_agents/contact_interactions*.yaml` 파일 삭제,
     `openapi.yaml`의 `paths:` 등록 제거
   - `components/schemas`의 `ContactManagerInteraction`,
     `ContactManagerInteractionListResponse`, `ContactManagerResolution`
     등 이 경로 전용 스키마도 함께 제거 (다른 스키마가 참조하지 않는지
     구현 착수 시 확인)
3. **§3.5(bin-api-manager short-circuit 설계)는 전부 불필요**: 410/빈
   리스트를 반환하도록 만들었던 모든 로직(`interactionGet` 헬퍼
   short-circuit, `ResolutionCreate`/`Delete` 트레이드오프, 신규 410 에러
   타입 등)은 **경로 자체가 없어지므로 만들 필요가 없다.** 대신
   `pkg/servicehandler/interaction.go`, `serviceagent_interaction.go`,
   `server/contact_interactions.go`, `server/service_agents_contact_interactions.go`를
   **파일째 삭제**한다.
4. **신규 경로**: `bin-api-manager`가 이미 노출 중인
   `GET /contact_peer_events`, `GET /service_agents/contact_peer_events`
   (§2.3, PR #1136에서 이미 병합됨, 변경 없음)를 CRM interaction 조회의
   유일한 공개 API로 삼는다. 이번 설계 문서 범위에서 이 경로 자체는
   건드리지 않는다 — 이미 존재하고 이미 노이즈 포함 원본을 그대로
   반환하도록 되어 있다 (§2.3 참조).
5. **`bin-contact-manager`의 `InteractionList`(§3.2 원문)는 RPC 자체는
   유지하되 응답 구조체를 바꾼다 (대표님 정정 지시, 2026-07-25):** 처음
   §8.1에서는 "REST 경로가 사라지니 RPC도 삭제"로 갔으나, 대표님이
   `bin-ai-manager`(§2.5, §8.3)가 **같은 RPC 도구를 계속 호출**해야 한다는
   점을 짚어 정정했다. 최종 결정:
   - `contacthandler.InteractionList`(RPC 핸들러 본체)와
     `pkg/listenhandler/v1_interactions.go`의
     `processV1InteractionsGet`(RPC 라우트) **둘 다 존속**한다. 삭제하지
     않는다.
   - 내부 구현은 §3.2가 이미 설계한 대로 `peer_events` read API를
     호출하도록 재작성한다 (이 부분은 원 설계와 동일하게 유지).
   - 단, **응답 구조체를 더 이상 `interaction.Interaction` 모양으로
     가공하지 않는다** — §8.1-1에서 결정한 대로 `peerevent.PeerEvent`를
     가공 없이 그대로 반환한다. 즉 이 RPC의 **반환 타입 자체가
     `[]*interaction.Interaction`에서 `[]*peerevent.PeerEvent`(또는 동일
     shape의 신규 타입)로 바뀐다.**
   - `bin-common-handler/pkg/requesthandler`의 `ContactV1InteractionList`
     wrapper도 이 신규 반환 타입에 맞춰 시그니처를 갱신한다.
   - `bin-ai-manager/pkg/aicallhandler/tool_insight.go`는 **같은 RPC를
     계속 호출**하지만, 응답 필드가 달라지므로(예: `id`/`tm_interaction`
     같은 필드가 없어지고 `publisher`/`event_type` 등 새 필드가 생김)
     그 구조체를 소비하는 코드는 **필드 접근 부분만 신규 shape에 맞춰
     수정**해야 한다. 이는 bin-ai-manager의 비즈니스 로직 변경이 아니라
     구조체 변경에 따른 기계적 수정이며, 본 설계 문서의 스코프에
     **포함**된다 (§8.3 갱신).
   - `contacthandler.InteractionGet`/`InteractionListUnresolved`/
     `ResolutionCreate`/`ResolutionDelete`는 §2.5에서 확인된 대로
     `bin-ai-manager`가 호출하지 않으므로, 원 설계(§3.2/§3.3/§3.4)와
     동일하게 **삭제 대상 그대로** 유지한다. 존속시키는 것은
     `InteractionList` 하나뿐이다.
   - `pkg/listenhandler/v1_interactions.go` 파일 전체를 삭제하지 않는다
     (위 정정으로 §8.1-5의 원안이 철회됨) — `processV1InteractionsGet`
     라우트만 남기고 `processV1InteractionsIDGet`/
     `processV1InteractionsResolutionsPost`/
     `processV1InteractionsResolutionsIDDelete`/
     `processV1InteractionsUnresolvedGet` 4개 라우트만 이 파일에서 제거한다
     (§3.8의 원 목록과 동일, `processV1InteractionsGet`는 목록에서 뺀다).

**bin-contact-manager (부분 삭제 — Round의 §8.1-5 정정 반영)**
- `models/interaction/`(구조체만, §8.1-5 정정으로 RPC 응답 타입이
  `peerevent.PeerEvent` 기반으로 바뀌므로 기존 `interaction.Interaction`
  구조체는 더 이상 쓰이지 않음), `models/resolution/` 패키지 전체 삭제
- `pkg/contacthandler/interaction.go`(`EventCallCreated`,
  `EventConversationMessageCreated`, `crmIneligiblePeerTypes`,
  `isCRMEligiblePeer`), `resolution.go`(`ResolutionCreate`,
  `ResolutionDelete`) 전체 삭제
- `pkg/contacthandler/interaction_read.go`: **파일은 존속.**
  `InteractionGet`/`InteractionListUnresolved` 삭제,
  `InteractionList`은 **삭제하지 않고** `peer_events` 프록시로 재작성
  (§8.1-5 정정)하며 반환 타입을 `[]*peerevent.PeerEvent`(또는 동일 shape
  신규 타입)로 변경
- `pkg/contacthandler/main.go`의 `ContactHandler` 인터페이스에서
  `EventCallCreated`/`EventConversationMessageCreated`/`InteractionGet`/
  `ResolutionCreate`/`ResolutionDelete`/`InteractionListUnresolved` 6개
  시그니처만 제거, **`InteractionList` 시그니처는 반환 타입만 갱신하고
  유지**
- `pkg/listenhandler/v1_interactions.go`: **파일 존속.**
  `processV1InteractionsIDGet`/`processV1InteractionsResolutionsPost`/
  `processV1InteractionsResolutionsIDDelete`/
  `processV1InteractionsUnresolvedGet` 4개 라우트만 삭제,
  `processV1InteractionsGet`(list)은 유지하되 응답 직렬화 대상 타입 갱신.
  `pkg/listenhandler/main.go`의 dispatch case도 동일하게 4개만 제거
- `bin-common-handler/pkg/requesthandler`의 `ContactV1InteractionList`
  wrapper: 반환 타입을 `[]*peerevent.PeerEvent` 기반으로 갱신 (요청
  파라미터는 §3.2의 4가지 필터 모드 그대로 유지)
- **누락 보완 (본 Revision 리뷰 Round 1로 확인됨): 같은 파일
  (`bin-common-handler/pkg/requesthandler/contact_interactions.go`)의
  나머지 4개 wrapper도 함께 정리해야 한다.** `models/interaction`/
  `models/resolution` 패키지를 통째로 삭제하면(§8.2 위 항목), 이 두
  타입을 반환/참조하는 `ContactV1InteractionGet`(`*cminteraction.Interaction`
  반환), `ContactV1InteractionListUnresolved`(`[]*cminteraction.Interaction`
  반환), `ContactV1ResolutionCreate`(`*cmresolution.Resolution` 반환),
  `ContactV1ResolutionDelete`도 컴파일이 깨진다. `bin-common-handler`는
  37개 서비스 전체가 의존하는 공유 패키지이므로, 이 4개 함수와
  `pkg/requesthandler/main.go`의 `RequestHandler` 인터페이스에 선언된
  대응 시그니처 4개, `mock_main.go`의 mock 4개를 **함께 삭제**한다
  (§8.2에서 이미 이 4개 함수의 유일한 정당한 소비자였던 bin-api-manager
  쪽 호출부를 삭제하므로, 삭제 후 남는 합법적 호출자가 없음을 구현
  착수 시 재확인).
- `pkg/dbhandler/interaction.go`, `resolution.go`,
  `pkg/dbhandler/address_ownership*.go`(§2.4 확인된 대로 Case와 공유되지
  않는 부분만): §3.6(DB 정리)과 동일하게 **테이블/dbhandler 프리미티브
  자체는 이번에 지우지 않는다** — 이건 원 설계와 동일 (§8.1은 응답
  계약/RPC 레이어만 다루고, DB 레벨 원칙은 유지).
- `pkg/subscribehandler/callmanager.go`/`conversationmanager.go`의 구독
  호출 제거 (§3.1과 동일, 변경 없음)

**bin-api-manager (전부 삭제 — Admin/Agent 프론트엔드 노출 계층만)**
- `pkg/servicehandler/interaction.go`, `serviceagent_interaction.go` 파일 전체
- `server/contact_interactions.go`, `server/service_agents_contact_interactions.go` 파일 전체
- 대응 테스트 파일 전체 (`interaction_test.go`, `serviceagent_interaction_test.go`,
  `contact_interactions_test.go`, `service_agents_contact_interactions_test.go`)
- `pkg/servicehandler/main.go`의 `ServiceHandler` 인터페이스에서 대응
  메서드 시그니처 제거, mock 재생성
- **주의**: 이 삭제는 bin-api-manager가 `ContactV1InteractionList` RPC를
  더 이상 호출하지 않게 됨을 의미할 뿐, contact-manager 쪽 RPC 자체를
  없애는 것이 아니다 (위 bin-contact-manager 항목 참조) — RPC는
  `bin-ai-manager`가 계속 쓰므로 살아있다.

**bin-openapi-manager (전부 삭제)**
- `openapi/paths/contact_interactions/*.yaml` 전체
- `openapi/paths/service_agents/contact_interactions*.yaml` 전체
- `openapi.yaml`의 대응 `paths:` 등록 및 `ContactManagerInteraction*`/
  `ContactManagerResolution*` 스키마 정의 제거 (다른 스키마의 참조 여부
  확인 후)

**bin-ai-manager (기계적 수정 필요 — 본 설계 스코프에 포함, §8.3 참조)**
- `pkg/aicallhandler/tool_insight.go`: `ContactV1InteractionList` 호출은
  그대로 유지하되, 응답 구조체가 `interaction.Interaction`에서
  `peerevent.PeerEvent` shape로 바뀌므로 그 필드를 소비하는 코드(예:
  `id`/`tm_interaction` 접근 부분)를 신규 shape에 맞게 수정한다.

**변경 없음 (원 설계 그대로 유지)**
- `bin-timeline-manager`의 `peer_events` 적재/조회 파이프라인 (§2.2)
- `bin-api-manager`의 기존 `contact_peer_events` 노출 경로 (§2.3)
- `bin-contact-manager`의 Case(kase) 기능 (§2.4)
- DB 테이블 자체 보존, 후속 마이그레이션으로 분리 (§3.6)

### 8.3 영향받는 소비자 재확인 (§2.5 갱신, §8.1-5 정정 반영)

`bin-ai-manager/pkg/aicallhandler/tool_insight.go`가 `ContactV1InteractionList`
RPC를 직접 호출한다는 사실(§2.5)은 본 Revision의 **직접 설계 대상**이
됐다. 최초 §8.1/§8.2 초안은 이 RPC 자체를 삭제하는 방향이었으나, 대표님이
"contact-manager가 이 도구는 계속 지원하되, 내부 호출과 응답 포맷만
바꾼다. ai-manager는 같은 도구를 계속 호출하지만 응답 구조체가 달라진다"고
정정 지시했다 (§8.1-5 참조). 최종 결론:

- `ContactV1InteractionList` RPC 자체(및 그 route/handler)는 **존속**한다.
- RPC의 **내부 구현**은 `peer_events`를 프록시하도록 바뀌고, **반환
  구조체**가 `interaction.Interaction`에서 `peerevent.PeerEvent` shape로
  바뀐다 (가공 없이 그대로 반환, §8.1-1).
- `tool_insight.go`는 같은 RPC를 계속 호출하지만, 새 구조체의 필드에
  맞춰 소비 코드를 수정해야 한다. **정정 (본 Revision 리뷰 Round 1로
  확인됨): 실제 `toolHandleGetContactInteractions`는 `it.ID`를 전혀
  참조하지 않는다.** 실제 참조 필드는 `it.TMInteraction`, `it.Direction`,
  `it.Peer.Type`, `it.Peer.Target`, `it.ReferenceType`, `it.ReferenceID`
  뿐이며, `TMInteraction`→`Timestamp` 하나만 이름이 바뀌고 나머지는
  `peerevent.PeerEvent`의 `Direction`/`Peer`/`Publisher`/`ReferenceID`에
  1:1로 그대로 대응한다. 즉 실제 수정 범위는 문서가 처음 예시로 든 것보다
  더 좁고 기계적이다 (없는 `.ID` 참조를 찾아 헤맬 필요 없음).
- 노이즈 포함/과거이력 소실 같은 **동작 성격 변화**(§2.5의 원래 기록)는
  여전히 유효하다 — 이건 이번 정정과 무관하게 peer_events 자체의 특성.

### 8.4 §1~§7과의 관계 요약

| 원 섹션 | 상태 |
|---|---|
| §1 손실/신규 항목 | 유효 (peer_events의 데이터 자체 특성은 안 바뀜) |
| §2.1~2.5 현황 파악 | 유효 (사실관계는 그대로), 단 §2.5는 §8.3으로 갱신 |
| §3.1 적재 폐기 | 유효, 변경 없음 |
| §3.2 조회 프록시 | **부분 유효**. `InteractionList`(RPC)는 §3.2 원안대로 peer_events 프록시로 재작성(존속), 단 응답 shape은 §8.1-1로 대체(가공 없음). `InteractionGet`은 여전히 삭제 |
| §3.3 Resolution 폐기 | 유효 (여전히 삭제하지만, REST 경로도 함께 삭제되므로 §3.5 short-circuit은 불필요) |
| §3.4 미해결 큐 폐기 | 동일 |
| §3.5 bin-api-manager short-circuit | **전부 폐기**. §8.2로 대체 (파일째 삭제로 단순화) |
| §3.6 DB 정리 | 유효, 변경 없음 |
| §3.7 구독 큐 정리 | 유효, 변경 없음 |
| §3.8 인터페이스/디스패치 정리 | 유효, 원칙 그대로 유지하되 대상 목록이 §8.2만큼 늘어남 |
| §4 공유 헬퍼 미채택 사유 | 무의미해짐 (address_id 필터 자체가 사라지는 경로였으므로) — 참고용으로만 남김 |
| §5 응답 매핑 전체 | **전부 폐기**. §8.1-1로 대체 (가공 없음) |
| §6 Non-goals | "OpenAPI 경로 존치" 항목만 폐기, 나머지 유효 |
| §7 검증 계획 | 유효하나 대상 파일 목록이 §8.2 기준으로 확장됨 — `bin-common-handler` 자체 검증 워크플로우, `bin-api-manager/docsdev/source/` RST 갱신+재빌드가 §7 말미에 명시적으로 추가됨 (본 Revision 리뷰 Round 3 확인) |


## 9. Revision 2 (2026-07-25, post-closure, 대표님 지시): List 경로는 유지, 응답만 peer_events로 교체

**트리거**: §8(Revision 1)은 `GET /contact_interactions`를 포함한 모든
REST 경로를 완전 삭제하는 방향(§8.1 item 2)이었다. 대표님이 이를
재검토해 "List 경로(`GET /contact_interactions`,
`GET /service_agents/contact_interactions`)는 그대로 두고, 응답만
peer_events로 바꾸면 되지 않나?"라고 물었고, 이어서 나머지
경로(단건조회/{id}, resolution POST/DELETE, 미해결 큐)는 **경로 자체
삭제**로 확정했다 (무응답 후 직전 대화 맥락 — 지속된 "단순화 우선"
기조 — 기준 최선 판단, 필요 시 재확인).

**중요**: 본 §9는 §8을 대체하지 않고 그 위에 얹는다. §8의 리뷰 이력은
그대로 남기되, **§8.1 item 2("List 경로 포함 전부 삭제")와 §8.2의
`bin-api-manager` 삭제 목록 중 List(`GetContactInteractions`/
`GetServiceAgentsContactInteractions`) 관련 부분은 본 §9로 override
된다.** §8의 나머지(§8.1 item 1/3/4/5, §8.3, §7 갱신)는 그대로 유효.

### 9.1 무엇이 바뀌는가

1. **List 경로 존속, 응답만 교체**: `GET /contact_interactions`,
   `GET /service_agents/contact_interactions`는 **경로/OpenAPI 등록을
   유지**한다. 응답 본문만 `ContactManagerInteractionListResponse`(기존
   `interaction.Interaction` shape)에서 `peerevent.PeerEvent` 기반의
   신규 응답 shape로 교체한다. §8.1 item 1(가공 없이 그대로 반환)의
   결정은 유지 — 즉 RPC 계층(`contacthandler.InteractionList`)이 이미
   가공 없는 `peerevent.PeerEvent`를 반환하므로(§8.1 item 5), REST
   계층도 그 shape를 그대로 REST 응답으로 통과시키기만 하면 된다.
2. **나머지 경로는 완전 삭제** (§8.1 item 2와 동일한 처리 유지):
   - `GET /contact_interactions/unresolved`
   - `GET /contact_interactions/{id}`
   - `POST /contact_interactions/{id}/resolutions`
   - `DELETE /contact_interactions/{id}/resolutions/{rid}`
   - `service_agents/contact_interactions` 하위의 동일 패턴 4개
   - 이 5+5=10개 경로는 §8.1 item 2가 이미 확정한 대로 OpenAPI 스펙에서도
     통째로 삭제한다 (404 반환, 404/410 shim 남기지 않음 — 대표님이
     "단순화" 기조를 재확인).
3. **왜 List만 남기고 나머지는 삭제하는가**: List(그리고 오직 List)만
   peer_events read API로 대응 가능한 진짜 조회 기능이기 때문이다
   (§8.1 item 5가 이미 확인한 대로, `bin-ai-manager`도 이 RPC를 그대로
   씀). 나머지 넷은 §1에서 이미 확정한 손실 항목(과거이력 매칭/수동보정/
   미해결큐)이거나 peer_events 구조상 원천적으로 불가능한 기능(§3.2의
   단건조회 불가 사유)이므로, 경로를 남겨봐야 항상 에러/빈 값만 반환하는
   죽은 코드가 된다. "깨져도 된다"는 대표님 결정과 결합하면, 죽은
   shim보다 경로 삭제(순수 404)가 더 단순하다.

### 9.2 bin-api-manager 변경 (§8.2를 대체하는 최종 목록)

- **`server/contact_interactions.go`**: 파일 전체 삭제가 아니라
  **부분 삭제**로 정정. `GetContactInteractions`(List 핸들러)만 남기고,
  `GetContactInteractionsUnresolved`, `GetContactInteractionsId`,
  `PostContactInteractionsIdResolutions`,
  `DeleteContactInteractionsIdResolutionsRid` 4개 핸들러를 삭제한다.
  남는 `GetContactInteractions`는 반환 타입을 `peerevent.PeerEvent`
  기반 응답으로 변경.
- **`server/service_agents_contact_interactions.go`**: 동일 패턴.
  `GetServiceAgentsContactInteractions`만 남기고 나머지 4개
  (`Unresolved`, `Id` GET, `IdResolutions` POST/DELETE) 삭제.
- **`pkg/servicehandler/interaction.go`**: `InteractionList`만 남기고
  `InteractionListUnresolved`, `InteractionGet`, `ResolutionCreate`,
  `ResolutionDelete` 삭제. `InteractionList`는 §8.1 item 5가 정의한
  `ContactV1InteractionList`(반환 타입이 `peerevent.PeerEvent` 기반으로
  바뀐 RPC)를 그대로 호출하도록 재작성 (가공 없음, §9.1 item 1).
  **누락 보완 (본 Revision 리뷰 Round 1로 확인됨): private 헬퍼
  `interactionGet(ctx, customerID, id)`(`ContactV1InteractionGet` RPC
  호출)도 함께 삭제한다.** 이 헬퍼는 `InteractionGet`/`ResolutionCreate`
  (이 파일)와 `ServiceAgentInteractionGet`/`ServiceAgentResolutionCreate`
  (`serviceagent_interaction.go`) 4개 호출부에서만 쓰이며, 그 4개가 모두
  삭제 대상이므로 이 헬퍼도 죽은 코드가 된다 (남기면 `golangci-lint
  unused`에 걸림). `InteractionList`/`ServiceAgentInteractionList`(존속
  대상)는 이 헬퍼를 호출하지 않으므로 삭제해도 영향 없음 — 삭제는 이
  파일에서 한 번만 하면 되고, `serviceagent_interaction.go` 쪽 호출부는
  그냥 제거(정의가 여기 하나뿐이므로 중복 삭제 아님). 이 헬퍼가 유일한
  소비자였던 `ContactV1InteractionGet` RPC(`bin-common-handler`)도
  §9.2/§8.2가 이미 삭제 대상으로 잡고 있음(§8.2의 "누락 보완" 항목) —
  본 항목으로 그 삭제 근거가 한 번 더 확인됨.
- **`pkg/servicehandler/serviceagent_interaction.go`**: 동일 패턴,
  `ServiceAgentInteractionList`만 남기고 나머지 4개 삭제.
- **§3.5(Revision 1 이전 원 설계)의 410/빈-리스트 short-circuit 로직은
  여전히 불필요** — 대상 경로 자체가 삭제되므로 (§8이 이미 정리한 대로).
- **`pkg/servicehandler/main.go`의 `ServiceHandler` 인터페이스**: 삭제된
  8개 메서드(양쪽에서 4개씩) 시그니처 제거, `InteractionList`/
  `ServiceAgentInteractionList` 시그니처는 반환 타입만 갱신하고 유지.
  mock 재생성.
- **테스트**: `interaction_test.go`/`serviceagent_interaction_test.go`/
  `contact_interactions_test.go`/`service_agents_contact_interactions_test.go`는
  전체 삭제가 아니라, List 관련 테스트만 신규 응답 shape 기준으로
  재작성하고 나머지(unresolved/get/resolutions) 테스트는 삭제.

### 9.3 OpenAPI 변경 (§8.2를 대체)

- **유지 + 응답 스키마 교체**: `openapi/paths/contact_interactions/main.yaml`
  (list), `openapi/paths/service_agents/contact_interactions.yaml`의
  list 부분. `$ref: '#/components/schemas/ContactManagerInteractionListResponse'`를
  peer_events 응답 스키마(예: 기존 `TimelineManagerPeerEventListResponse`
  재사용 또는 별도 신규 스키마 — 구현 착수 시 `contact_peer_events`가
  쓰는 스키마와 동일하게 갈지, contact_interactions 전용으로 새로 만들지
  결정 필요. 재사용을 권장: 응답 shape이 어차피 동일한 `peerevent.PeerEvent`
  이므로 스키마 중복을 피함)로 교체.
- **삭제**: `openapi/paths/contact_interactions/unresolved.yaml`,
  `id.yaml`, `id_resolutions.yaml`, `id_resolutions_id.yaml`,
  `openapi/paths/service_agents/contact_interactions_unresolved.yaml`,
  `contact_interactions_id.yaml`,
  `contact_interactions_id_resolutions.yaml`,
  `contact_interactions_id_resolutions_id.yaml` 8개 파일 전부 삭제,
  `openapi.yaml`의 대응 `paths:` 등록 제거.
- **스키마**: `ContactManagerInteractionListResponse`/
  `ContactManagerInteraction`은 **삭제하지 않고 응답 스키마 자체를
  교체**(위 참조)하거나, 신규 스키마명으로 바꾸고 기존 스키마를
  삭제할지는 구현 착수 시 결정. `ContactManagerResolution`은 resolution
  기능 자체가 삭제되므로 그대로 삭제.

### 9.4 RST 문서 (§7의 Revision 1 추가 항목을 좁힘)

§7이 Revision 1에서 추가한 RST 갱신 항목(`contact_peer_event_overview.rst`)은
**여전히 유효**하지만 내용이 달라진다. List 경로가 살아있으므로, 이
문서가 안내하던 "`GET /contact_interactions`를 고객 대면 필터링된
데이터 조회에 쓰라"는 안내 자체는 **틀리지 않았다** — 다만 그 응답이
이제 정제된 `ContactManagerInteraction`이 아니라 노이즈 포함
`peerevent.PeerEvent`라는 점을 반영해 안내 문구/예시 응답만 갱신하면
된다 (경로를 `/contact_peer_events`로 안내를 바꿀 필요는 없어짐).
구현 착수 시 RST의 예시 JSON을 신규 응답 shape 기준으로 다시 작성한다.

### 9.5 §7~§8과의 관계 요약

| 이전 섹션 | 상태 |
|---|---|
| §8.1 item 2 (전 경로 삭제) | **부분 override**. List 2개 경로는 유지+응답 교체(§9.1/§9.2), 나머지 8개 경로는 원안대로 삭제 |
| §8.2 bin-api-manager 목록 | **override**. §9.2가 최종 목록 (파일 전체 삭제 → 부분 삭제로 정정) |
| §8.2 bin-openapi-manager 목록 | **override**. §9.3이 최종 목록 (스키마/경로 일부 존속) |
| §8.1 item 1/3/4/5, §8.3, §7 나머지 | 유효, 변경 없음 |
| §7의 RST 항목 | 유지되나 세부 내용이 §9.4로 좁혀짐 (경로 유지, 안내 문구만 갱신) |

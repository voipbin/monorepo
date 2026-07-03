# NOJIRA: Add service_agents/interactions unfiltered-list support

**Status:** Ready for implementation (Round 3 fixes applied: duplicate/stale code block in §3.3 removed and merged into a single coherent version)
**Ticket:** #1054
**Author:** Hermes (CPO)
**Date:** 2026-07-04
**Depends on:** PR #1053 (merged), PR #1056 (merged)

---

## 1. Context

PR #1053이 `service_agents/interactions` Agent 전용 엔드포인트를 신규 추가하면서, 원래 검토했던 "필터 없이 customer 전체 interaction 목록 조회" 기능은 스코프에서 드롭됐다(설계 문서 `docs/plans/2026-07-04-revert-interaction-permission-relax-add-service-agents-endpoint-design.md` §2.3 참조).

드롭 사유: `GetServiceAgentsInteractions`가 재사용하는 `h.reqHandler.ContactV1InteractionList` RPC는 top-level과 동일한 공유 코드 경로로 귀결되며, 다음 두 계층 모두에서 "필터 전부 없음"을 거부한다:

1. `bin-contact-manager/pkg/listenhandler/v1_interactions.go`의 `processV1InteractionsGet` — `filterCount != 1` → HTTP 400 (line 98-101)
2. `bin-contact-manager/pkg/contacthandler/interaction_read.go`의 `InteractionList` — `switch` 문의 `default` 분기에서 `cerrors.InvalidArgument("INVALID_FILTER", ...)` 반환 (line 90-95)

한편 `bin-contact-manager/pkg/dbhandler/interaction.go`의 `InteractionList`는 이미 전부-빈 필터를 무해하게 처리하는 로직이 있다(line 117-119: `if peerType == "" && peerTarget == "" && len(addressSet) == 0 { return nil, nil }`) — 이 최하위 레이어는 준비되어 있으나 위 두 계층의 얼리리턴 가드가 도달을 막고 있다.

## 2. Goal

`GET service_agents/interactions`를 필터 없이 호출하면, 호출한 Agent의 `customer_id` 전체 interaction 목록을 페이지네이션과 함께 반환한다. **top-level `GET /interactions`(Admin/Manager 전용)는 기존 동작(정확히 1개 필터 필수)을 그대로 유지한다** — 이번 변경은 신규 Agent 표면에만 적용되는 opt-in 확장이다.

## 2.1 Round 1 리뷰에서 확인된 major 이슈와 수정 (since 상한 필수화)

**리뷰 결과(2개 병렬 리뷰):** 리뷰어 1(코드 정확성/blast radius)은 APPROVE WITH MINOR FIXES. 리뷰어 2(프로덕션 스케일 안전성)는 **CHANGES REQUIRED** — 다음 두 가지 major 이슈를 지적했다:

1. **인덱스는 문제없음(PASS)**: `bin-contact-manager/scripts/database_scripts_test/contacts.sql`에 `idx_contact_interactions_cursor on contact_interactions(customer_id, tm_create)` 복합 인덱스가 이미 존재하여, `WHERE customer_id=? ORDER BY tm_create DESC` 쿼리는 풀스캔이 아닌 효율적인 인덱스 레인지 스캔이다.

2. **`since` 상한 부재(major, 최초 초안의 실질적 결함)**: `contact_interactions`는 `EventCallCreated`/`EventConversationMessageCreated`(`bin-contact-manager/pkg/contacthandler/interaction.go`)에서 CRM 적격 통화/대화 메시지마다 1행씩 append되는, TTL·압축·아카이빙이 전혀 없는 무제한 증가 테이블이다. 활성 고객은 수년치 트래픽으로 수십만~수백만 행에 달할 수 있다. 개별 페이지 조회는 인덱스 덕분에 저렴하지만, **`since` 하한이 없으면 Agent 클라이언트가 필터도 시간범위도 없이 고객의 전체 이력을 무한 페이지네이션으로 순회할 수 있어**, ops 입장에서 요청당 최악 스캔량을 제한할 방법이 없다. 같은 코드베이스의 가장 가까운 선례인 `InteractionListUnresolved`가 정확히 이 패턴(기본 30d, 최대 180d `since` 강제)으로 이미 이 문제를 해결해뒀는데, 최초 초안은 이를 놓쳤다.

**결정: 이번 라운드에서 `allowUnfiltered=true` 경로에 `since` 파라미터를 필수로 추가한다.** `InteractionListUnresolved`와 동일한 패턴(기본 30d, 최대 180d, `"Nd"` 형식 강제)을 그대로 재사용한다. 이렇게 하면 "필터 없음"이 진짜로 "무제한"이 아니라 "필터 없이, 단 최근 N일로 시간 스코핑됨"이 되어 스캔량 상한이 보장된다.

**page_size 상한(리뷰어 2의 2번째 major 이슈):** `since` 상한을 추가하면 이 우려는 상당 부분 해소된다(`InteractionListUnresolved`도 `since` 하나만으로 이 리스크를 막고 있으며 page_size는 그대로 100 유지). 별도의 더 엄격한 page_size 캡은 추가하지 않는다 — `since`가 근본 원인(무제한 이력 순회)을 막으므로 이중 방어는 과잉이라고 판단.

## 3. Design decision: opt-in bool 파라미터를 전 계층에 관통

가장 좁은 blast radius를 위해, 기존 시그니처를 바꾸지 않고 새 파라미터 하나 `since time.Time`을 4개 계층(dbhandler → contacthandler → listenhandler → requesthandler)에 추가한다. **`allowUnfiltered bool`을 별도로 추가하지 않는다** — `since`가 zero-value(`time.Time{}`)이면 "unfiltered 모드 아님"(top-level의 기존 동작), non-zero이면 "unfiltered 모드 + 해당 시각 이후로 스코핑"으로 겸용한다. 이렇게 하면 시그니처에 파라미터가 하나만 추가되고, "unfiltered인데 since가 없는" 상태 자체가 타입 레벨에서 존재할 수 없어 §2.1의 안전장치가 우회될 수 없다.

기존 top-level 호출부는 전부 `time.Time{}`(zero-value)를 명시적으로 전달해 현재 동작을 그대로 보존한다.

**왜 새 RPC 메서드를 만들지 않고 기존 `ContactV1InteractionList`에 파라미터를 추가하는가:** 이미 `ServiceAgentInteractionList`가 top-level `InteractionList`와 완전히 동일한 로직/RPC를 재사용하는 패턴이 확립되어 있다(PR #1053). 새 RPC 메서드를 만들면 이 재사용 원칙을 깨고 로직 중복(및 향후 두 메서드 간 drift 리스크)이 생긴다. 파라미터 추가가 기존 패턴과의 일관성을 지키는 최소 변경이다.

### 3.1 dbhandler 계층

`bin-contact-manager/pkg/dbhandler/interaction.go`:

```go
func (h *handler) InteractionList(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    peerType, peerTarget string,
    addressSet []AddressPair,
    since time.Time,  // 신규. zero-value = 미적용(기존 동작 그대로)
) ([]*interaction.Interaction, error) {
    if peerType == "" && peerTarget == "" && len(addressSet) == 0 {
        if since.IsZero() {
            return nil, nil
        }
        // since가 non-zero: 필터 없이 customer_id + tm_create >= since로 스코핑해서 계속 진행
        // (아래 builder는 이미 WHERE customer_id=? 를 기본 적용하므로,
        //  peer/addressSet 조건절 대신 tm_create >= since 조건절만 추가하면 됨)
    }
    ...
    if !since.IsZero() {
        builder = builder.Where(sq.GtOrEq{"tm_create": since.UTC()})
    }
    ...
}
```

**주의:** `interactionListByContact`(§99-245, set-MINUS 알고리즘)와 `interactionListByAddress`(§247-271) 내부에서도 `h.db.InteractionList`를 호출하지만, 이들은 항상 `addressSet`이 비어있지 않은 상태로 호출하므로 이 변경의 영향을 받지 않는다. `since` 파라미터는 이 두 내부 호출부에서 `time.Time{}`(zero-value)로 고정.

### 3.2 contacthandler 계층

`bin-contact-manager/pkg/contacthandler/interaction_read.go`의 `InteractionList`:

```go
func (h *contactHandler) InteractionList(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    peerType, peerTarget string,
    contactID uuid.UUID,
    addressID uuid.UUID,
    since time.Time,  // 신규
) ([]*interaction.Interaction, string, error) {
    ...
    switch {
    case peerType != "" || peerTarget != "":
        ...
    case contactID != uuid.Nil:
        return h.interactionListByContact(...)
    case addressID != uuid.Nil:
        return h.interactionListByAddress(...)
    case !since.IsZero():  // 신규 분기, default보다 먼저 배치
        items, err := h.db.InteractionList(ctx, customerID, size+1, token, "", "", nil, since)
        if err != nil {
            return nil, "", fmt.Errorf("could not list all interactions. InteractionList. err: %v", err)
        }
        return buildPagedResult(items, size)
    default:
        return nil, "", cerrors.InvalidArgument(...)
    }
}
```

### 3.3 listenhandler 계층

`bin-contact-manager/pkg/listenhandler/v1_interactions.go`의 `processV1InteractionsGet`:

- 신규 쿼리 파라미터 `since`를 **필터가 전부 없을 때만** 파싱한다.
- **`since`의 형식은 RFC3339Nano 절대시각 문자열이다**(`"30d"` 같은 상대값 문자열이 아니다) — 근거는 §3.4의 인코딩 결정을 참조. `parseDaysDuration`(상대 `"Nd"` 파싱 헬퍼)은 이 계층에서 재사용하지 않는다. 상대값 파싱은 `bin-api-manager`의 HTTP 파라미터 레벨(§3.5)에서만 발생하고, RPC 경계를 넘을 때는 이미 절대시각으로 정규화되어 전달된다.
- 필터 검증 및 `since` 처리 로직:

```go
filterCount := 0
... // 기존 3개 필터 카운트 로직 그대로

var since time.Time
if filterCount == 0 {
    sinceStr := q.Get("since")
    if sinceStr == "" {
        since = time.Now().Add(-30 * 24 * time.Hour)  // 기본 30d
    } else {
        parsed, parseErr := time.Parse(time.RFC3339Nano, sinceStr)
        if parseErr != nil {
            log.Errorf("Invalid since param format: %v", parseErr)
            return simpleResponse(400), nil
        }
        since = parsed
    }
    // 180일 상한 재검증 (근거는 아래 단락 참조)
    maxLookback := time.Now().Add(-180 * 24 * time.Hour)
    if since.Before(maxLookback) {
        log.Errorf("since exceeds maximum lookback of 180d: %v", since)
        return simpleResponse(400), nil
    }
}

if filterCount != 1 && filterCount != 0 {
    log.Errorf("Expected exactly one filter mode, got %d.", filterCount)
    return simpleResponse(400), nil
}
```

- `filterCount >= 2`는 여전히 400 (모호한 쿼리는 항상 거부).
- **필터가 1개일 때는 `since`가 절대 적용되지 않는다** — top-level `/interactions`(필터 필수)는 이 신규 파라미터를 아예 파싱하지 않으므로 기존 동작에 영향 없음. 이 분기는 오직 `filterCount == 0`일 때만 도달.

**180일 상한을 listenhandler에도 재검증하는 이유(Round 2 리뷰에서 확정):** 초기 개정판은 "bin-api-manager HTTP 레이어에서만 검증하고 RPC 레이어는 신뢰한다"고 설계했으나, 이는 실제 기존 선례와 반대임이 Round 2 리뷰에서 확인됐다:

- `bin-api-manager/server/interactions.go:121-139`(`GetInteractionsUnresolved`)는 `since` 파라미터의 **형식만**(`HasSuffix "d"`, 양수) 검증하고, 상한(180d) 검증은 명시적으로 하지 않는다 — 코드 주석에 "upper-bound (180d) is enforced by the contact-manager listenhandler"라고 직접 적혀 있다(line 123).
- `bin-contact-manager/pkg/listenhandler/v1_interactions.go:139`(`processV1InteractionsUnresolvedGet`)의 `parseDaysDuration(sinceStr, 180)`가 **실제 180일 상한을 강제하는 지점**이다.

즉 기존 코드베이스의 실제 분업은 "bin-api-manager는 형식만, bin-contact-manager(RPC 경계)가 범위를 강제"다. `contact_interactions`가 무제한 증가 테이블임이 확인된 상황에서(§2.1), 범위 검증을 bin-api-manager 단일 레이어에만 두는 것은 방어 심도(defense-in-depth)를 후퇴시키는 실질적 리스크다. **결정: listenhandler에도 180일 상한 재검증을 추가한다**(위 코드 블록에 반영 완료). 새 엔드포인트는 절대시각을 받으므로 `parseDaysDuration`(상대 `"Nd"` 문자열 전용)을 그대로 호출할 수는 없지만, 위 인라인 `since.Before(maxLookback)` 체크가 동일한 포함 상한(180일째는 허용, 181일째부터 거부) 의미를 재현한다 — 이렇게 하면 실제 기존 선례(`processV1InteractionsUnresolvedGet`)와 정확히 동일한 이중 방어 구조가 된다.

### 3.4 requesthandler 계층 (RPC 클라이언트)

`bin-common-handler/pkg/requesthandler/contact_interactions.go`의 `ContactV1InteractionList`:

```go
func (r *requestHandler) ContactV1InteractionList(
    ctx context.Context,
    customerID uuid.UUID,
    size uint64,
    token string,
    peerType, peerTarget string,
    contactID, addressID uuid.UUID,
    since time.Time,  // 신규. zero-value = 미적용
) ([]*cminteraction.Interaction, string, error) {
    u := url.Values{}
    ...
    if !since.IsZero() {
        u.Set("since", since.UTC().Format(time.RFC3339Nano))
    }
    ...
}
```

**인코딩 결정(확정):** `since`는 절대시각을 RFC3339Nano 문자열로 URL 쿼리에 실어 보낸다("Nd" 상대값 문자열이 아님). `.UTC()`를 먼저 호출하므로 포맷 결과는 항상 `Z` 서픽스로 끝나고 `+HH:MM` 형식의 타임존 오프셋(`+` 문자 포함)은 나타나지 않는다 — 따라서 PR #1056에서 확인했던 `peer_target`의 `+` URL 인코딩 이슈(`%2B` 필요)는 이 값에는 해당하지 않는다. `u.Set()`/`u.Encode()`(송신)와 `url.Parse()`+`q.Get()`(수신)는 Go 표준 라이브러리의 대칭적 percent-encode/decode 계약을 그대로 따르므로 추가 인코딩 처리가 불필요하다.

**⚠️ Round 1 리뷰에서 확인된 blast radius:** 이 함수는 `bin-common-handler`에 있으므로, 시그니처 변경이 **모든 호출부**를 건드린다. 병렬 리뷰가 전체 monorepo grep으로 확인한 현재 호출부는 정확히 2곳(non-test):
- `bin-api-manager/pkg/servicehandler/interaction.go:73`의 `InteractionList`(top-level) → `since=time.Time{}` 고정 전달
- `bin-api-manager/pkg/servicehandler/serviceagent_interaction.go:50`의 `ServiceAgentInteractionList`(신규 표면) → `since` 파라미터를 그대로 전달(아래 §3.5)

이 외에 **테스트 파일 2곳도 mock 호출 시그니처를 갱신해야 한다**(Round 1 리뷰에서 지적됨, 최초 초안 누락):
- `bin-api-manager/pkg/servicehandler/interaction_test.go:114`의 mock `.ContactV1InteractionList(...)` 기대값
- `bin-api-manager/pkg/servicehandler/serviceagent_interaction_test.go:89`의 mock `.ContactV1InteractionList(...)` 기대값

### 3.5 bin-api-manager 계층

**top-level (`interaction.go`):** `InteractionList` 시그니처는 바꾸지 않는다(top-level은 여전히 "정확히 1개 필터"만 지원, `since` 개념 자체가 없음). 내부에서 `ContactV1InteractionList(..., time.Time{})`로 고정 호출.

**신규 표면 (`serviceagent_interaction.go`):** `ServiceAgentInteractionList`에 파라미터 추가:

```go
func (h *serviceHandler) ServiceAgentInteractionList(
    ctx context.Context,
    a *auth.AuthIdentity,
    size uint64,
    token string,
    peerType, peerTarget string,
    contactID, addressID uuid.UUID,
    since time.Time,  // 신규. server 레이어가 계산해서 전달 (필터 0개일 때만 non-zero)
) ([]*cminteraction.Interaction, string, error) {
    ...
    items, nextToken, err := h.reqHandler.ContactV1InteractionList(ctx, a.CustomerID, size, token, peerType, peerTarget, contactID, addressID, since)
    ...
}
```

**server (`service_agents_interactions.go`):** `GetServiceAgentsInteractions`의 필터 검증을 "정확히 1개" → "0개 또는 1개"로 완화하고, 필터 0개일 때 `since` 파싱(top-level `GetInteractionsUnresolved`의 기존 `since` 파싱 로직과 동일한 패턴 재사용):

```go
filterCount := 0
if peerType != "" || peerTarget != "" { filterCount++ }
if contactID != uuid.Nil               { filterCount++ }
if addressID != uuid.Nil               { filterCount++ }
if filterCount > 1 {
    // 400 InvalidArgument — 여러 필터 동시 지정은 여전히 금지
}

var since time.Time
if filterCount == 0 {
    sinceStr := "30d"
    if params.Since != nil && *params.Since != "" {
        sinceStr = *params.Since
    }
    // 기존 GetServiceAgentsInteractionsUnresolved의 since 검증 로직(HasSuffix "d" + Atoi + 범위체크,
    // max 180d)을 공용 헬퍼로 추출해서 재사용 — 코드 중복 방지
    sinceDuration, err := parseSinceParam(sinceStr, 180)
    if err != nil {
        abortWithError(c, cerrors.InvalidArgument(..., "INVALID_SINCE", ...))
        return
    }
    since = time.Now().Add(-sinceDuration)
}
// filterCount == 0 → since로 스코핑된 목록, 정상 허용
```

**신규 공용 헬퍼 `parseSinceParam`:** 현재 `GetInteractionsUnresolved`(top-level)와 `GetServiceAgentsInteractionsUnresolved`(신규 표면) 양쪽에 이미 거의 동일한 "Nd 형식 검증 + Atoi + 범위체크" 인라인 코드가 중복 존재한다(PR #1053에서 이미 한 번 복제됨). 이번 PR에서 세 번째 사용처가 생기므로, `server/` 패키지에 공용 헬퍼 함수로 추출하는 것을 권장(기존 2곳도 리팩터링해 통일 — 별도 커밋으로 분리 가능, 필수는 아니지만 강력 권장).

## 4. Tenant boundary

`h.db.InteractionList`가 `WHERE customer_id=?`를 항상 강제하므로(builder 초기화 시점에 적용, 필터/`since` 유무와 무관), `since` non-zero 경로도 다른 customer의 데이터를 노출하지 않는다. `ServiceAgentInteractionList`는 `a.CustomerID`만 전달하므로(다른 customer_id를 지정할 방법이 API 표면에 없음) 추가 검증 불필요.

## 5. Pagination / 응답 형태

기존 `buildPagedResult` 그대로 재사용(cursor = `tm_create` 내림차순). 전체 목록이라도 페이지네이션 동작은 필터가 있을 때와 동일 — 신규 로직 없음. `since` 하한과 cursor(`tm_create < token`) 상한이 함께 적용되어 페이지네이션 도중에도 스코핑이 일관되게 유지된다.

## 6. Tests

### 6.1 dbhandler (`interaction_test.go`)
- `InteractionList` with `since`=non-zero, 전부 빈 필터 → customer 소속 + `tm_create >= since`인 row만 반환(다른 customer 제외, since 이전 row 제외 확인)
- `InteractionList` with `since`=zero-value, 전부 빈 필터 → `nil, nil` (기존 동작 보존, 회귀 테스트)

### 6.2 contacthandler (`interaction_read_test.go` — 있다면, 없으면 신규)
- `InteractionList` with `since`=non-zero, 필터 없음 → 정상 반환
- `InteractionList` with `since`=zero-value, 필터 없음 → 기존과 동일하게 `INVALID_FILTER` 에러 (회귀 테스트)

### 6.3 listenhandler (`v1_interactions_test.go` — 있다면)
- `since=<RFC3339Nano 절대시각>` 쿼리 파라미터 + 필터 없음 → 200
- `since` 없음 + 필터 없음 → 200, 기본 30d 적용 확인
- `since` 잘못된 형식 + 필터 없음 → 400
- `since`가 180일보다 오래된 시각 + 필터 없음 → 400 (Round 2 리뷰에서 추가, listenhandler 레벨 상한 재검증 확인)
- `since` 없음(또는 있음) + 필터 1개 → **`since`가 무시되고** 기존과 동일하게 정상 동작 (top-level 회귀 테스트, filterCount==1이면 since 분기 자체가 실행되지 않음을 확인)
- 필터 2개 이상 → 400 (모호한 쿼리는 여전히 거부, `since` 값과 무관)

### 6.4 bin-api-manager
- `pkg/servicehandler/interaction_test.go`(top-level `InteractionList`): 기존 테스트가 여전히 `since=time.Time{}`을 mock 기대값에 포함하도록 갱신 (시그니처 변경이므로 mock 재생성 필요)
- `pkg/servicehandler/serviceagent_interaction_test.go`: `ServiceAgentInteractionList` 필터 없음 케이스 신규 추가, RPC에 non-zero `since`가 전달되는지 검증
- `server/service_agents_interactions_test.go`: `GetServiceAgentsInteractions` 필터 0개 → 200 케이스로 변경(기존에는 400을 테스트했음, PR #1056에서 추가한 "필터 0개→400" 테스트는 이 PR에서 "필터 0개→200, 기본 30d 적용"으로 수정 필요). `since` 형식 오류 → 400 케이스 신규 추가(top-level `GetServiceAgentsInteractionsUnresolved`의 기존 since 검증 테스트 패턴 재사용).
- `server/interactions_test.go`(top-level): 변경 없음(여전히 필터 0개 → 400, `since` 개념 자체가 없음)

## 7. Verification

`bin-common-handler` 시그니처 변경이므로 CLAUDE.md 규칙에 따라: 1) `bin-common-handler` 자체 빌드 확인 2) 영향받는 모든 컨슈머(`bin-contact-manager`, `bin-api-manager`) 각각 전체 검증 워크플로 실행. Go workspace/local replace 구조상 각 서비스 디렉토리에서 개별적으로 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`을 순서대로(bin-common-handler → bin-contact-manager → bin-api-manager) 실행.

## 8. Not in scope

- square-talk 프론트엔드에서 필터 없는 목록을 실제로 노출하는 UI (SQUARE-7 계열, 별도 프론트엔드 작업)
- `interactionListByContact`/`interactionListByAddress` 내부 로직 변경 (영향 없음, §3.1에서 확인)
- PR #1055/#1056에서 다룬 "필터 2개 이상" 테스트 커버리지 확장 (이미 별도로 처리됨)

## 9. Review requirement

CLAUDE.md 규칙에 따라 설계 문서 병렬 리뷰 최소 2라운드(수렴 확인 라운드 포함), 코드 PR도 최소 3라운드 리뷰. `bin-common-handler` 공유 라이브러리 변경이므로 특히 "모든 컨슈머 빌드 확인" 항목을 리뷰에서 중점적으로 검증.

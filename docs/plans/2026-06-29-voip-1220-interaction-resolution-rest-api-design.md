# VOIP-1220: Expose Interaction & Resolution REST API

**Status:** Draft (Design Review Round 2 반영)
**Ticket:** VOIP-1220
**Author:** Hermes (CPO)
**Date:** 2026-06-29
**Depends on:** VOIP-1209 (Done)

---

## 1. Context

VOIP-1209에서 `bin-contact-manager`의 RabbitMQ listenhandler와 `bin-common-handler`의 requesthandler 계층에 interaction/resolution 엔드포인트가 이미 구현되었다. 이 티켓은 해당 구현을 외부 REST API로 노출하는 **3계층 wiring** 작업이다. 신규 도메인 설계 없이 기존 구현을 연결하는 것이 전부다.

---

## 2. Scope

이미 구현된 것 (VOIP-1209):
- `bin-contact-manager/pkg/listenhandler/v1_interactions.go` — 5개 엔드포인트 구현
- `bin-common-handler/pkg/requesthandler/contact_interactions.go` — 5개 RPC 클라이언트

이 티켓에서 추가하는 것:
1. `bin-openapi-manager` — OpenAPI 스펙 (스키마 + 경로)
2. `bin-api-manager/pkg/servicehandler/` — auth/권한 위임 레이어
3. `bin-api-manager/server/` — HTTP 핸들러

---

## 3. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/interactions` | List interactions (filter required) |
| GET | `/v1/interactions/unresolved` | List unresolved interactions |
| GET | `/v1/interactions/{id}` | Get single interaction |
| POST | `/v1/interactions/{id}/resolutions` | Create resolution |
| DELETE | `/v1/interactions/{id}/resolutions/{rid}` | Delete resolution |

---

## 4. Data Models

### 4.1 Interaction (external representation)

`bin-contact-manager/models/interaction/interaction.go`의 내부 구조체를 그대로 외부에 노출한다. WebhookMessage 계층을 두지 않는다.

**근거:** `Interaction`은 append-only immutable projection record이며 내부 전용 필드(PodID, Username, PermissionIDs 등)가 없다. 외부에 노출해서는 안 되는 필드가 없으므로 ConvertWebhookMessage 변환 단계가 불필요하다.

**중요:** 이 결정은 `bin-api-manager/CLAUDE.md`의 "External responses MUST use `.ConvertWebhookMessage()`" 규칙을 의식적으로 예외 처리한 것이다. 향후 `Interaction` 또는 `Resolution` 구조체에 내부 전용 필드가 추가되는 시점에 WebhookMessage 타입 도입이 필요하다. 해당 시점에 이 설계 결정을 재검토해야 한다.

```
Interaction {
  id              uuid
  customer_id     uuid
  direction       string  // "incoming" | "outgoing"
  peer_type       string  // "tel" | "email"
  peer_target     string  // normalized remote address
  local_type      string
  local_target    string
  reference_type  string  // "call" | "conversation"
  reference_id    uuid
  tm_interaction  *time.Time  // nullable
  tm_create       *time.Time  // pagination cursor
}
```

### 4.2 Resolution (external representation)

동일한 근거로 WebhookMessage 없이 직접 노출한다.

```
Resolution {
  id               uuid
  customer_id      uuid
  contact_id       uuid
  interaction_id   uuid
  resolution_type  string  // "positive" | "negative"
  resolved_by_type string  // "agent" | "system" | "rule"
  resolved_by_id   uuid
  tm_create        *time.Time
  tm_update        *time.Time
  tm_delete        *time.Time  // nullable (soft-delete sentinel)
}
```

### 4.3 InteractionListResponse

`bin-contact-manager/models/interaction/list_response.go`에 정의된 타입:

```
InteractionListResponse {
  items           []*Interaction
  next_page_token string  // empty = no more pages
}
```

---

## 5. Layer Design

### 5.1 bin-openapi-manager

**신규 파일:**
```
openapi/paths/interactions/main.yaml
openapi/paths/interactions/unresolved.yaml
openapi/paths/interactions/id.yaml
openapi/paths/interactions/id_resolutions.yaml
openapi/paths/interactions/id_resolutions_id.yaml
```

**openapi.yaml 경로 등록 순서 (가독성 관례):**
```yaml
/interactions:
/interactions/unresolved:
/interactions/{id}:
/interactions/{id}/resolutions:
/interactions/{id}/resolutions/{rid}:
```

참고: Gin은 radix trie 라우터를 사용하므로 static path(`/unresolved`)와 parameterized path(`/{id}`)는 등록 순서와 무관하게 올바르게 분기된다. 위 순서는 스펙 가독성 관례일 뿐이다.

**신규 스키마:**
- `ContactManagerInteraction`
- `ContactManagerInteractionListResponse`
- `ContactManagerResolution`

### 5.2 pkg/servicehandler/interaction.go

**인터페이스 메서드 (main.go에 추가):**

```go
InteractionList(ctx, a, size, token, peerType, peerTarget, contactID, addressID) (*InteractionListResponse, error)
InteractionListUnresolved(ctx, a, size, token, sinceDays) (*InteractionListResponse, error)
InteractionGet(ctx, a, id) (*Interaction, error)
ResolutionCreate(ctx, a, interactionID, contactID, resolutionType, resolvedByType, resolvedByID) (*Resolution, error)
ResolutionDelete(ctx, a, interactionID, resolutionID) error
```

**권한 정책:** 모든 메서드에 `PermissionCustomerAdmin | PermissionCustomerManager` 필요. Direct access 차단.

**소유권 검증 전략:**

| 메서드 | 전략 |
|--------|------|
| InteractionList | `a.CustomerID`를 RPC에 전달 → 백엔드에서 customerID로 범위 제한 |
| InteractionListUnresolved | 동일 |
| InteractionGet | `interactionGet(ctx, a.CustomerID, id)` RPC 호출. RPC가 customerID를 request body로 전달하여 백엔드에서 wrong-tenant에 404 반환. 반환된 `res.CustomerID`에 대해 `hasPermission` 호출. |
| ResolutionCreate | `interactionGet()` 선행 호출로 소유권 확인 후 `ContactV1ResolutionCreate` RPC 호출. **RPC 인수 순서 주의: `(ctx, interactionID, customerID, contactID, ...)` — interactionID가 customerID 앞에 온다.** |
| ResolutionDelete | `a.CustomerID` + `interactionID` + `resolutionID`를 RPC에 전달. **단, 현재 listenhandler 구현에서 `interaction_id`는 URI 경로 구성에만 사용되고 DB WHERE 절에 포함되지 않는다. 실제 강제는 `WHERE customer_id=? AND id=?`만 이루어진다.** 자세한 내용은 §6 참조. |

**private 헬퍼:**
```go
// interactionGet: customerID-scoped fetch.
// contactV1InteractionGet RPC가 customerID를 request body로 전달하여
// 백엔드에서 wrong-tenant 요청에 404를 반환한다.
func (h *serviceHandler) interactionGet(ctx, customerID, id) (*Interaction, error)
```

### 5.3 server/interactions.go

**핸들러별 설계 결정:**

**GetInteractions:**
- filter 검증: `peer_type+peer_target`, `contact_id`, `address_id` 중 정확히 1개. 미충족 시 `cerrors.InvalidArgument` → 400.
- pageSize 기본값 100, 상한 100.

**GetInteractionsUnresolved:**
- `since` 파라미터: `"Nd"` 형식 강제. `strings.HasSuffix(s, "d")` 확인 + `strconv.Atoi(strings.TrimSuffix(s, "d"))` 파싱. 형식 오류 시 400 반환. 미제공 시 sinceDays=0 → requesthandler가 `?since` 파라미터를 생략 → listenhandler 기본값(30d) 적용.

**GetInteractionsId:**
- servicehandler가 `interactionGet()` 호출 (RPC가 customerID로 스코핑) → 200.

**PostInteractionsIdResolutions:**
- 응답 코드: **201** (리소스 생성).
- **주의:** listenhandler(`v1_interactions.go`)는 이 엔드포인트에 `StatusCode: 200`을 반환한다. HTTP 핸들러는 RPC 응답 코드를 전달하지 않고 `c.JSON(201, res)`로 직접 지정해야 한다.
- 요청 바디: contact_id, resolution_type, resolved_by_type, resolved_by_id 필수.

**DeleteInteractionsIdResolutionsRid:**
- 응답 코드: 200 + 빈 JSON body.

---

## 6. ResolutionDelete 소유권 현황 및 결정

**현재 listenhandler 동작 (VOIP-1209 구현):**

`processV1InteractionsResolutionsIDDelete`에서 URI의 `interactionID`(parts[3])는 파싱되지 않으며 `contactHandler.ResolutionDelete(ctx, customerID, resolutionID)`에 전달되지 않는다. 따라서 백엔드 DB 레이어에서 강제되는 조건은 `WHERE customer_id=? AND id=?`이다.

**결과:** 동일 고객의 권한 있는 에이전트는 어떤 interaction 경로를 통해서든 자신의 고객에 속한 resolution을 삭제할 수 있다. 즉 URI의 `{id}` (interactionID)가 실제로 소유권 강제에 사용되지 않는다.

**이 티켓의 결정:**

본 티켓(VOIP-1220)에서 listenhandler 수정은 **포함하지 않는다.** 이는 VOIP-1209 범위의 이슈이며 별도 티켓으로 추적한다. API 레이어 노출 자체는 현재 동작을 정확히 반영하여 구현한다.

**§8 Open Questions에 추적 항목 추가.**

---

## 7. Filter Validation (GET /interactions)

listenhandler에서도 동일한 filter 검증을 수행하지만, API 레이어에서 early-return 400을 반환한다.

```
filterCount := 0
if peerType != "" || peerTarget != "" { filterCount++ }
if contactID != uuid.Nil               { filterCount++ }
if addressID != uuid.Nil               { filterCount++ }
if filterCount != 1 → 400 InvalidArgument
```

---

## 8. Test Coverage

### 8.1 server/interactions_test.go

| 핸들러 | 테스트 케이스 |
|--------|--------------|
| GetInteractions | 정상(contact_id 필터), 필터 없음 → 400, 미인증 → 401 |
| GetInteractionsUnresolved | 정상(기본 since), 명시적 since, 잘못된 since 형식 → 400, 미인증 → 401 |
| GetInteractionsId | 정상, 미인증 → 401 |
| PostInteractionsIdResolutions | 정상 → 201, 미인증 → 401 |
| DeleteInteractionsIdResolutionsRid | 정상 → 200, 미인증 → 401 |

### 8.2 pkg/servicehandler/interaction_test.go

| 메서드 | 테스트 케이스 |
|--------|--------------|
| InteractionList | 정상, 권한 없음 (PermissionNone) |
| InteractionListUnresolved | 정상, 권한 없음 |
| InteractionGet | 정상, 권한 없음 |
| ResolutionCreate | 정상 (interactionGet pre-fetch 포함), 권한 없음 |
| ResolutionDelete | 정상, 권한 없음 |

---

## 9. Open Questions / Non-Goals

- **RST 문서 업데이트**: bin-api-manager CLAUDE.md는 신규 엔드포인트 추가 시 `docsdev/source/` 업데이트를 요구한다. 이 PR에 포함할지 별도 티켓으로 분리할지는 대표님 판단에 따른다.
- **Interaction WebhookMessage**: 향후 내부 전용 필드가 `Interaction`에 추가될 경우 별도 작업으로 WebhookMessage 도입.
- **ResolutionList**: 현재 listenhandler에 구현되지 않아 이 티켓 범위 밖.
- **ResolutionDelete interactionID 소유권**: listenhandler가 `interaction_id`를 DB WHERE 절에 포함하지 않는 이슈는 별도 VOIP 티켓으로 추적 필요.
- **Webhook 이벤트 없음**: resolution 생성/삭제에 대한 webhook 이벤트가 현재 없다. 이는 명시적 비목표이며, 필요 시 별도 설계 필요.
- **POST 201 vs RPC 200**: listenhandler는 resolution 생성 시 StatusCode 200을 반환한다. HTTP 핸들러에서 `c.JSON(201, res)`로 명시적으로 지정해야 하며, RPC 응답 코드를 그대로 전달해서는 안 된다.

---

## 10. Implementation Order

1. `bin-openapi-manager`: YAML 파일 작성 → `openapi.yaml` 등록 → `go generate ./...`
2. `bin-api-manager`: `go generate ./...` (gen.go 갱신)
3. `bin-api-manager/pkg/servicehandler/`: `main.go` 인터페이스 추가 → `interaction.go` 구현 → `go generate ./pkg/servicehandler/...` (mock 재생성)
4. `bin-api-manager/server/`: `interactions.go` 구현
5. 테스트 파일 2개 작성
6. 전체 검증: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`

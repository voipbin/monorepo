# NOJIRA: Revert interaction permission relax, add service_agents/interactions & service_agents/resolutions

**Status:** Draft (Round 2, revised after Round 1 review)
**Ticket:** #1052
**Author:** Hermes (CPO)
**Date:** 2026-07-04
**Corrects:** PR #1047 (`NOJIRA-Relax-interaction-agent-permission`, merged, commit `1608cfa5f`)

---

## 1. Context

PR #1047은 square-talk(Agent 전용 프론트엔드)가 CRM interaction/resolution 기능을 쓸 수 있도록, top-level `/interactions/*`, `/interactions/{id}/resolutions*` 5개 엔드포인트의 권한 비트마스크에 `PermissionCustomerAgent`를 직접 추가했다.

이는 `bin-openapi-manager/CLAUDE.md`의 "Adding a Service Agent (Agent-facing) resource" 원칙 위반이다:

> Agent-facing frontend가 Admin/Manager-gated 리소스에 새 capability가 필요할 때는, top-level 엔드포인트의 `hasPermission(...)` 비트마스크를 완화하는 게 아니라 신규 `service_agents/<resource>` 엔드포인트를 추가해야 한다.

top-level `/interactions/*`는 square-admin(Admin/Manager 전용 화면)도 계속 호출하는 표면이다. 비트마스크 완화로 인해, 원칙이 막고자 했던 "Admin/Manager 전용 표면을 Agent에게까지 조용히 열어버리는" 상황이 실제로 발생했다.

**구분 기준 (스킬 `voipbin-square-talk-feature`에 이미 명문화된 판단 기준):** Agent-facing 앱이 필요로 하는 게 리소스의 좁은 WRITE 액션 1~2개뿐이면 top-level 비트마스크를 완화하는 선택지도 있다. 그러나 이번 케이스는 GET List/Unresolved/Get을 포함한 리소스 표면 대부분이 필요하므로, 신규 `service_agents/interactions`, `service_agents/resolutions` 표면을 만드는 게 맞는 선택이다.

---

## 2. Scope

### 2.1 되돌리기 (top-level, 원상복구)

`bin-api-manager/pkg/servicehandler/interaction.go`의 5개 메서드에서 `amagent.PermissionCustomerAgent` 비트 제거:

| 메서드 | 현재 (PR #1047) | 되돌린 후 |
|--------|-----------------|-----------|
| `InteractionList` | `PermissionCustomerAgent\|Admin\|Manager` | `PermissionCustomerAdmin\|PermissionCustomerManager` |
| `InteractionListUnresolved` | 〃 | 〃 |
| `InteractionGet` | 〃 | 〃 |
| `ResolutionCreate` | 〃 | 〃 |
| `ResolutionDelete` | 〃 | 〃 |

OpenAPI 스펙(`openapi/paths/interactions/*.yaml`)은 요청/응답 계약이 바뀌지 않으므로 **수정하지 않는다**. 순수하게 Go 코드의 권한 게이트만 되돌린다.

### 2.2 신규 추가 (Agent 전용 표면)

**OpenAPI (신규 파일, top-level 미수정):**
```
openapi/paths/service_agents/interactions.yaml
openapi/paths/service_agents/interactions_unresolved.yaml
openapi/paths/service_agents/interactions_id.yaml
openapi/paths/service_agents/interactions_id_resolutions.yaml
openapi/paths/service_agents/interactions_id_resolutions_id.yaml
```

**servicehandler (신규 파일 `pkg/servicehandler/serviceagent_interaction.go`):**
```go
ServiceAgentInteractionList(ctx, a, size, token, peerType, peerTarget, contactID, addressID) ([]*cminteraction.Interaction, string, error)
ServiceAgentInteractionListUnresolved(ctx, a, size, token, since) ([]*cminteraction.Interaction, string, error)
ServiceAgentInteractionGet(ctx, a, id) (*cminteraction.Interaction, error)
ServiceAgentResolutionCreate(ctx, a, interactionID, contactID, resolutionType, resolvedByType, resolvedByID) (*cmresolution.Resolution, error)
ServiceAgentResolutionDelete(ctx, a, interactionID, resolutionID) error
```

내부적으로 top-level과 동일한 private 헬퍼(`h.interactionGet`)와 동일한 RPC 호출(`h.reqHandler.ContactV1Interaction*`, `h.reqHandler.ContactV1Resolution*`)을 재사용한다. 권한 게이트만 `amagent.PermissionAll`(`ServiceAgentContact*`, `ServiceAgentTranscribe*`와 동일한 "이 customer의 인증된 Agent라면 누구나" 센티널)로 다르다.

**필수 컨벤션 (Round 1 리뷰에서 확인, 기존 초안 누락):** `serviceagent_contact.go`, `serviceagent_transcribe.go`의 모든 `ServiceAgent*` public 함수는 예외 없이 첫 줄에 다음 가드를 둔다:
```go
if !a.IsAgent() {
    return nil, serviceerrors.ErrAuthenticationRequired
}
```
5개 신규 함수 전부 이 가드를 첫 줄에 포함한다.

**server (신규 파일 `server/service_agents_interactions.go`):**
```
GetServiceAgentsInteractions
GetServiceAgentsInteractionsUnresolved
GetServiceAgentsInteractionsId
PostServiceAgentsInteractionsIdResolutions
DeleteServiceAgentsInteractionsIdResolutionsRid
```
top-level `server/interactions.go`의 파싱/검증 로직을 그대로 재사용하되, 호출하는 servicehandler 메서드만 `ServiceAgent*`로 교체한다.

### 2.3 필터 요구사항 (신규 표면도 top-level과 동일하게 유지 — Round 1 계획에서 변경됨)

top-level `GetInteractions`는 `peer_type+peer_target` / `contact_id` / `address_id` 중 **정확히 1개**를 요구한다(§7 of VOIP-1220 design). 신규 `GetServiceAgentsInteractions`도 **동일하게 정확히 1개**를 요구한다.

**Round 1 리뷰에서 뒤집힌 결정:** 최초 초안은 신규 표면에서 "필터 0개(전체 조회)"도 허용하는 안이었다. 병렬 설계 리뷰에서 다음이 확인됨:
- `ServiceAgentInteractionList`가 재사용할 예정인 `h.reqHandler.ContactV1InteractionList` RPC는 top-level과 **동일한 공유 코드 경로**(`bin-contact-manager/pkg/contacthandler/interaction_read.go`의 `InteractionList`)로 귀결된다. 이 함수는 필터가 전부 비어있으면 `default` 분기에서 `cerrors.InvalidArgument("INVALID_FILTER", ...)`를 반환한다(현재 코드 확인, 라인 90-95). `bin-contact-manager/pkg/listenhandler/v1_interactions.go`도 독립적으로 동일한 "정확히 1개" 검증을 한 번 더 강제한다.
- 즉 `bin-api-manager` 레이어에서만 필터 개수 검증을 완화해도, 요청은 결국 `ContactV1InteractionList` RPC를 거쳐 `bin-contact-manager`의 두 계층 모두에서 400으로 막힌다. `bin-api-manager`만의 변경으로는 "필터 0개 = 전체 조회"가 동작하지 않는다.
- 이를 실제로 동작시키려면 `bin-contact-manager`에 신규 RPC 또는 파라미터 추가(예: `allow_empty_filter` 플래그를 listenhandler→contacthandler→dbhandler까지 관통시키는 변경)가 필요하다. 이는 **크로스 서비스 백엔드 변경**이며, 이번 티켓(#1052, 권한 되돌리기 + 표면 분리)의 스코프를 넘어선다.

**결정:** 이번 라운드에서는 "필터 0개(전체 조회)" 기능을 **드롭**한다. `GetServiceAgentsInteractions`는 top-level과 동일하게 "정확히 1개 필터" 요구사항을 그대로 적용한다. "customer 전체 interaction 목록(필터 없이 조회)" 기능은 `bin-contact-manager` 변경을 포함하는 **별도 후속 티켓**으로 분리한다(§9 Follow-up 참조). 이렇게 하면 신규 표면이 정말로 기존 patterns(`serviceagent_contact.go`, `serviceagent_transcribe.go`)와 완전히 동일한 "권한 게이트만 다르고 로직은 순수 재사용" 원칙을 지킨다 — 새 필터 검증 로직도, 백엔드 변경도 없다.

```go
// server/service_agents_interactions.go — GetServiceAgentsInteractions
// top-level GetInteractions(server/interactions.go)와 완전히 동일한 검증 로직 재사용.
filterCount := 0
if peerType != "" || peerTarget != "" { filterCount++ }
if contactID != uuid.Nil               { filterCount++ }
if addressID != uuid.Nil               { filterCount++ }
if filterCount != 1 {
    // 400 InvalidArgument — top-level과 동일
}
```

---

## 3. Interface + Mock

`pkg/servicehandler/main.go`에 5개 메서드 시그니처 추가, `mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod` 재생성 (`pkg/servicehandler/`에서 실행).

---

## 4. Tests

### 4.1 되돌리기 대상 (기존 테스트 갱신)

`pkg/servicehandler/interaction_test.go`에서 PR #1047(`git show 1608cfa5f`)이 추가한 5개 "agent permission is sufficient" 서브테스트를 확인 완료(Round 1 리뷰에서 라인 단위 확인):

| 함수 | 서브테스트 위치 |
|------|------------------|
| `Test_InteractionList` | L61-80 `"agent permission is sufficient"` |
| `Test_InteractionListUnresolved` | L558-572 `"agent permission is sufficient"` |
| `Test_InteractionGet` | L181-206 `"agent permission is sufficient"` |
| `Test_ResolutionCreate` | L318-346 `"agent permission is sufficient"` |
| `Test_ResolutionDelete` | L446-457 `"agent permission is sufficient"` |

5개 전부 `serviceerrors.ErrPermissionDenied`를 기대하는 어서션으로 되돌린다(같은 함수 내 기존 `"permission denied"`/`"no permission"` 서브테스트와 동일한 패턴). 라인 번호는 구현 시점에 재확인(브랜치 분기 이후 다른 커밋이 반영됐을 수 있음).

### 4.2 신규 표면

| 메서드 | 테스트 케이스 |
|--------|--------------|
| ServiceAgentInteractionList | 정상(필터 있음, 1개), 필터 0개 → 400 InvalidArgument(top-level과 동일 검증), 필터 2개 이상 → 400, Admin 권한도 통과(PermissionAll 포함 확인), 미인증 |
| ServiceAgentInteractionListUnresolved | 정상, 잘못된 since 형식 → 400 |
| ServiceAgentInteractionGet | 정상, 타 customer 소유 interaction → PermissionDenied |
| ServiceAgentResolutionCreate | 정상 → 201, 타 customer interaction에 대한 resolution 생성 시도 → 에러 |
| ServiceAgentResolutionDelete | 정상 → 200 |

server 레벨 HTTP 테스트(`server/service_agents_interactions_test.go`)도 동일 케이스로 작성.

---

## 5. Verification

`bin-openapi-manager`에서 `go generate ./...` 먼저, 그다음 `bin-api-manager`에서 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

---

## 6. Not in scope

- square-talk `src/api/services/interactions.js`를 `/v1.0/service_agents/interactions`로 전환하는 프론트엔드 변경 — 별도 커밋/PR (SQUARE-7 워크트리에서 진행, 이 백엔드 PR 머지 후).
- square-talk Home 탭 UI 자체 구현 (SQUARE-7).
- **[정정, Round 1 리뷰에서 확인]** VOIP-1220 design doc §6이 기술한 "`ResolutionDelete`가 `interaction_id`를 DB WHERE 절에 포함하지 않는다"는 한계는 **현재 코드에는 존재하지 않는다.** `bin-contact-manager/pkg/dbhandler/resolution.go:57`을 직접 확인한 결과 `Where(sq.Eq{"interaction_id": interactionID.Bytes()})`가 이미 포함되어 있다(`WHERE customer_id=? AND interaction_id=? AND id=?` 전부 강제). VOIP-1220 문서 작성 시점 이후 별도 커밋으로 수정된 것으로 보인다. 따라서 이번 티켓은 이 항목에 대해 "상속하는 기존 결함"이 없으며, 신규 `ServiceAgentResolutionDelete`도 동일하게 3중 WHERE 절이 강제된 안전한 상태로 시작한다. (VOIP-1220 문서 자체는 별도로 정정이 필요하나 이 티켓의 스코프는 아니다.)
- "customer 전체 interaction 목록(필터 없이 조회)" 기능 — §2.3에서 설명한 대로 `bin-contact-manager` 백엔드 변경이 필요해 이번 라운드에서 드롭. 별도 후속 티켓 필요(§9).

---

## 7. Implementation Order

1. `bin-api-manager/pkg/servicehandler/interaction.go`: 5개 메서드 권한 비트 되돌리기
2. `pkg/servicehandler/interaction_test.go`: Agent 권한 케이스를 PermissionDenied로 되돌리기
3. `bin-openapi-manager`: 5개 YAML 신규 파일 작성 → `openapi.yaml` 등록 → `go generate ./...`
4. `bin-api-manager`: `go generate ./...` (gen.go 갱신)
5. `pkg/servicehandler/main.go` 인터페이스 추가 → `serviceagent_interaction.go` 구현 → mock 재생성
6. `server/service_agents_interactions.go` 구현
7. 테스트 파일 작성 (servicehandler + server 레벨)
8. 전체 검증 워크플로

---

## 8. Open Questions

- **RST 문서 업데이트**: `bin-api-manager/CLAUDE.md`는 신규 엔드포인트 추가 시 `docsdev/source/` 업데이트를 요구. 이 PR에 포함할지 별도 티켓으로 분리할지 대표님 판단.

## 9. Follow-up (별도 티켓, 이번 스코프 아님)

- **customer 전체 interaction 목록(필터 없이 조회)**: `bin-contact-manager`의 `ContactV1InteractionList` RPC 체인(listenhandler → contacthandler → dbhandler)에 "필터 전부 없음 = customer_id만으로 스코핑된 전체 조회" 경로를 새로 추가하는 크로스 서비스 백엔드 변경. `dbhandler/interaction.go:116-119`는 이미 전부-빈 필터를 무해하게 처리하는 로직이 있으나(`nil, nil` 반환, 에러 아님) 그 위 두 계층(listenhandler, contacthandler)이 먼저 400을 반환해 도달 불가 상태. 이 두 계층의 얼리리턴 가드를 수정/분기하는 별도 설계 문서 + 리뷰 필요.
- **VOIP-1220 design doc 정정**: §6의 "interaction_id가 DB WHERE에 없다"는 서술이 현재 코드와 불일치(이미 수정됨). 문서 정정 필요(코드 변경 아님, docs만).

# DESIGN: Case Peer Address → Contact Auto-Claim

**Issue:** VOIP-1270
**Branch:** VOIP-1270-case-peer-address-contact-claim
**Author:** Hermes (CPO)
**Status:** DRAFT v1 — Round 0 (pre-review)

---

## 0. Recon 결과 및 전제 수정

티켓 원문은 "옵션5: 호출 경로별 분기(Flow 자동화 vs 콘솔)"를 확정안으로 채택했다.
그러나 코드베이스 재조사 결과 이 전제가 성립하지 않는다:

1. **`ContactV1CaseUpdateContact` RPC의 호출자는 현재 단 하나뿐이다** —
   `bin-api-manager/pkg/servicehandler/case.go`의
   `serviceHandler.CaseUpdateContact`. `bin-flow-manager` 전체를 grep해도
   이 RPC를 호출하는 코드는 0건이다. "Flow action으로 Case-Contact를
   자동 assign"하는 기능 자체가 아직 존재하지 않는다.
2. **`bin-contact-manager` 레이어는 애초에 호출자 신원을 모른다.**
   `casehandler.UpdateContact(ctx, customerID, caseID, contactID)`는
   `customerID`만 받는다. `AuthIdentity`(Agent/Admin 구분)는
   `bin-api-manager/models/auth/auth.go`에만 존재하는 API 계층 개념이며
   RPC 경계를 넘어 `bin-contact-manager`까지 전달되지 않는다. 즉 "호출
   경로별로 분기"하는 로직은 이 레이어에서 만들 수조차 없다.

**결론 — 설계 방향 전환:** 분기할 대상(두 번째 호출 경로)이 없으므로
"경로별 분기" 설계(옵션5)는 폐기한다. 대신 다음으로 단순화한다:

> **백엔드는 항상 옵션3(조건부 자동 claim)을 수행한다. 프론트엔드는
> 옵션4를 "제안 후 확인"이 아니라 "자동 실행 결과의 사후 고지"로
> 변형해 적용한다.**

근거:
- (a) 유일한 실제 호출 경로가 콘솔뿐이므로, "제안 배너 → 클릭 확인"과
  "자동 실행 → 결과 토스트"는 사용자 관점에서 클릭 한 번 차이일 뿐,
  실질적 안전성 차이가 거의 없다.
- (b) 백엔드가 호출자 신원으로 분기하지 않으므로 로직이 단순하고,
  향후 실제로 Flow action 경로가 추가되어도(예: "Case를 Contact에
  자동 연결" 액션 타입) 동일한 자동 claim 로직이 자연스럽게 재사용된다
  — 이번에 존재하지 않는 경로를 위한 코드를 미리 만들 필요가 없다(YAGNI).
- (c) 상담사는 결과를 보고 필요시 되돌릴 수 있다 (기존
  `RemoveAddress`/detach 경로 존재, 자동 롤백은 하지 않음 — §5).

이 전환은 대표님이 이미 확정한 "옵션5"와 다르다. 리뷰에서 반드시
검증받아야 할 핵심 쟁점이다.

---

## 1. Background

Case는 생성 시 `Peer commonaddress.Address` (Target/Type)를 갖는다 —
이 Case를 만든 원본 통신 상대방의 주소다. 상담사가 콘솔에서 이
Case에 Contact를 attach(`UpdateContact`)할 때, 현재는 Case.contact_id만
갱신할 뿐, Peer.Target에 해당하는 `contact_addresses` 레코드가
unresolved 상태로 남아있어도 그대로 방치된다. 결과적으로 향후 동일
Peer로부터 오는 상호작용이 자동으로 이 Contact와 연결되지 못한다
(unresolved 주소는 Contact 매칭에 쓰이지 않음).

VOIP-1270 스코프: Case-Contact assign 시, Peer.Target 주소가
unresolved이면 자동으로 해당 Contact에 claim한다.

## 2. Scope

- `bin-contact-manager`: `casehandler.UpdateContact` 내부에 조건부
  자동 claim 로직 추가.
- `bin-contact-manager` dbhandler: `AddressGetByTarget` (또는 동등한
  조회) 추가 — Peer.Target으로 기존 주소 존재/소유 여부 확인.
- `bin-openapi-manager`: `GET /v1/contact_addresses`에 `target` 쿼리
  파라미터 추가 (현재 `contact_id`, `type`, `unresolved`뿐).
- `bin-api-manager`: OpenAPI 변경에 따른 타입/핸들러 재생성 반영.
- square-admin(프론트): 자동 claim 결과 사후 고지 UI. **이번 설계
  문서 스코프에서는 방향만 정의**하고 목업은 만들지 않는다 (구현
  단계에서 `voipbin-frontend-visual-verification-gate`로 확인).

Out of scope (YAGNI, 명시적 보류):
- 고객사 단위 자동 claim on/off 플래그.
- detach 시 자동 claim된 주소의 자동 롤백.
- Flow action으로서의 Case-Contact 자동 assign (존재하지 않는 경로).

## 3. 자동 Claim 로직 (`casehandler.UpdateContact` 확장)

현재 흐름 (`case_contact_attached` 이벤트 경로, `contactID != uuid.Nil`)에
아래 단계를 삽입한다. 위치: 기존 `h.db.CaseUpdateContactID` 성공 직후,
이벤트 발행 이전.

```
1. case.Peer.Target / case.Peer.Type 으로 기존 주소를 조회한다.
   (dbhandler에 AddressGetByTarget(ctx, customerID, addrType, target)
   신설 — customer_id + type + target 유니크 매치, 없으면 ErrNotFound.)

2. 케이스 분기:
   a) 주소가 없음 (ErrNotFound)
      → CreateUnresolvedAddress로 새로 만들지 않는다. Peer는 Case
        생성 시점의 커뮤니케이션 메타데이터일 뿐, 주소록에 새 엔트리를
        만드는 것은 이번 스코프 밖(암묵적 사용자 데이터 생성은 부작용이
        크다). 로그만 남기고 스킵.
   b) 주소가 있고 unresolved (contact_id == nil)
      → contacthandler.ClaimAddress(ctx, customerID, addr.ID, contactID)
        를 내부 함수 호출로 직접 실행 (RPC 왕복 없음, 같은 프로세스).
        실패 시(예: 동시성 경합으로 이미 다른 곳에서 claim됨) 에러를
        삼키고 warn 로그만 남긴다 — 이 자동 claim 실패가 메인 흐름인
        Case-Contact attach 자체를 실패시켜서는 안 된다(부가 기능).
   c) 주소가 있고 이미 같은 contactID에 resolved
      → no-op.
   d) 주소가 있고 다른 contactID에 resolved
      → 자동 claim 시도하지 않는다 (충돌 - 사람이 판단할 문제).
        로그만 남긴다.

3. 자동 claim이 실제로 발생했는지(b 케이스, 성공)를 반환값에 실어
   상담사에게 알릴 수 있게 한다: UpdateContact의 리턴 타입에
   `AutoClaimedAddress *contact.Address` 같은 부가 필드를 추가하는
   대신 — Case 자체는 이미 이벤트로 상태 변경을 알리므로, 별도
   `case_peer_address_claimed` 이벤트를 추가로 publish한다
   (contact_id, case_id, address_id, target 포함). square-admin은
   이 이벤트(또는 API 응답에 포함된 claimed 여부)를 구독/조회해
   토스트를 띄운다. 구체 전송 방식(이벤트 vs 응답 필드 추가)은
   Round 1 리뷰에서 확정한다 — 현재는 이벤트 방식을 기본안으로 제안.
```

이 로직은 항상 실행된다(옵션3, 조건부 자동) — 호출자가 콘솔이든
향후 다른 경로든 동일하게 적용된다.

## 4. API 변경: `target` 쿼리 필터

`GET /v1/contact_addresses`에 `target` (string, optional) 파라미터
추가. `AddressList` dbhandler에 `filters["target"]` 처리 추가
(`sq.Eq{"target": t}`). 용도: 콘솔/디버깅에서 특정 주소 존재 여부
조회. §3의 내부 로직은 별도 `AddressGetByTarget` 단건 조회를
사용하므로 이 API 변경과는 독립적이지만, 티켓에 명시된 gap이므로
같이 처리한다.

## 5. Detach 시 롤백 정책

Case에서 Contact를 detach(`contactID == uuid.Nil`)해도, §3에서
자동 claim된 주소는 되돌리지 않는다 — 비대칭적으로 유지한다.
이유: attach 시점에 이미 실제 통신 이력이 있었다는 사실 자체는
detach로 사라지지 않는다; 주소-Contact 연결의 유효성은 Case
attach 여부가 아니라 실제 소유권(누가 그 번호/이메일을 쓰는가)의
문제다. 되돌리고 싶으면 상담사가 명시적으로
`DELETE /contact_addresses/{id}` 또는 기존 detach 흐름을 사용한다.

## 6. 프론트엔드(square-admin) 방향 (설계만, 구현/목업 없음)

Case-Contact assign 액션의 API 응답 처리 후, 자동 claim이 발생했으면
(§3의 이벤트 또는 응답 필드로 확인) 인라인 안내를 띄운다. 예:
"Peer 주소 +82-10-... 도 이 Contact에 연결되었습니다." 제안형(사전
확인) UI가 아니라 결과 고지(사후) UI다. 실제 컴포넌트/카피는 구현
단계에서 `voipbin-frontend-visual-verification-gate` 스킬로 검증한다.

## 7. 동시성 / 에러 처리

- `AddressClaim`은 이미 DB 트랜잭션 기반 conflict 체크(`AddressClaimTx`)
  로 구현되어 있으므로 재사용한다.
- 자동 claim 실패는 `UpdateContact`의 주 트랜잭션 실패로 전파하지
  않는다(§3-b). 왜: Case-Contact attach는 사용자가 명시적으로 요청한
  동작이고, 자동 claim은 부가 편의 기능이다. 편의 기능의 실패로 주
  동작이 실패하면 사용자 경험이 나빠진다.

## 8. 테스트 계획 (구현 단계에서 상세화)

- `casehandler.UpdateContact`: 주소 없음 / unresolved / 이미 같은
  contact / 다른 contact에 resolved 4가지 케이스 단위 테스트.
- `AddressGetByTarget`: 존재/미존재/customer 격리 테스트.
- `AddressList` target 필터: 단위 테스트 + OpenAPI 스펙 검증.
- 자동 claim 실패 시 주 트랜잭션이 성공하는지 검증(에러 격리 확인).

## 9. Open Questions for Review

1. §3-3의 "결과 통지" 방식 — 이벤트 신설 vs 기존 응답에 필드 추가.
   신설 이벤트는 square-admin이 실시간 구독 인프라를 이미 갖췄다는
   전제가 필요 — 확인 필요.
2. §3-2-a "주소가 없으면 스킵" 결정이 맞는지 — Peer 주소를 자동으로
   unresolved 주소로 등록(CreateUnresolvedAddress)하는 게 오히려
   기대되는 동작일 수도 있음(향후 재상호작용 매칭 목적). 스코프
   확장 여부 재확인 필요.
3. `case_peer_address_claimed` 이벤트가 필요하다면 이름/payload가
   기존 이벤트 네이밍 컨벤션과 맞는지.

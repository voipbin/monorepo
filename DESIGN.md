# DESIGN: Case Peer Address → Contact Attribution (명시적 모달 확인)

**Issue:** VOIP-1270
**Branch:** VOIP-1270-case-peer-address-contact-claim
**Status:** DRAFT v5 — 방향 전면 재설계 (대표님 확정 지시 반영). 이전 v1~v4
(백엔드 자동 조건부 claim + 프론트 사후 토스트)는 폐기.

---

## 0. 이전 설계 폐기 및 재사용 가능한 재조사 사실

이전 DESIGN.md(git history: `9c9dde230`, `3d602875f`)는 "옵션3: 백엔드가
항상 자동으로 claim 시도 + 프론트는 결과를 사후 토스트로만 고지"하는
방향이었다. 대표님이 이 방향을 폐기하고, **상담사가 명시적으로 확인하는
모달 + 그 결정을 기억하는 옵션**으로 교체하라고 지시했다. 이 문서는 그
지시를 반영한 전면 재설계다.

이전 문서 §0의 재조사 사실 중 이번 설계에도 유효한 것들 (재조사 없이
그대로 재사용):

1. **`ContactV1CaseUpdateContact` RPC의 유일한 호출자는
   `bin-api-manager/pkg/servicehandler/case.go`의 `CaseUpdateContact`다.**
   `bin-flow-manager`에는 이 RPC를 호출하는 코드가 없다 — "호출 경로별
   분기" 자체가 성립하지 않는다는 점은 여전히 유효하다. 이번 설계는
   애초에 호출 경로 분기가 아니라 **프론트엔드 모달 확인**으로 문제를
   풀기 때문에, 이 사실은 오히려 이번 설계 방향(백엔드는 단순하게
   유지하고 확인은 콘솔 UI 레이어에서 처리)을 강화한다.
2. **`bin-contact-manager`는 호출자 신원(`AuthIdentity`)을 모른다.**
   `casehandler.UpdateContact(ctx, customerID, caseID, contactID)`는
   `customerID`만 받는다 — 여전히 유효하다. 이번 설계는 신원 기반 분기를
   요구하지 않으므로 문제가 되지 않는다.
3. `contact_addresses`는 **hard-delete** 테이블이다(tm_delete 없음).
   재확인: `bin-contact-manager/CLAUDE.md` — "`contact_addresses` is
   hard-delete (no `tm_delete`), mirroring `agent_addresses`." 이 사실이
   이번 설계의 §4(release API)의 핵심 근거다: DELETE로 "재할당용 해제"를
   대체할 수 없다.
4. `POST /contact_addresses/{id}/claim`
   (`bin-openapi-manager/openapi/paths/contact_addresses/id_claim.yaml`)은
   이미 존재하며, unresolved(contact_id IS NULL) 주소에만 성공하고
   이미 살아있는 Contact 소유면 409(Conflict)를 반환한다. 구현:
   `contacthandler.ClaimAddress` → `dbhandler.AddressClaim` →
   `addressClaimAttempt`(bin-contact-manager/pkg/dbhandler/address.go:157-255).

---

## 1. Background

Case는 생성 시 `Peer commonaddress.Address`(Target/Type)를 갖는다 — 이
Case를 만든 원본 통신 상대방의 주소다. 상담사가 square-admin 콘솔에서
Case 상세 화면(`contact_cases_detail.js`)의 `CaseContactAttributionPanel`
에서 `handleAttach`를 눌러 Case를 Contact에 assign할 때, 현재는
`PUT contact_cases/{caseId}`로 `Case.contact_id`만 갱신할 뿐,
Peer.Target에 해당하는 `contact_addresses` 레코드는 전혀 건드리지
않는다. 그 결과:

- Peer 주소가 `contact_addresses`에 아직 없으면, 향후 동일 Peer로부터의
  상호작용이 이 Contact와 자동 매칭되지 않는다(unresolved 주소만 Contact
  매칭 대상에서 제외되는 게 아니라, 애초에 레코드 자체가 없으므로 매칭
  불가).
- Peer 주소가 이미 다른 Contact 소유로 존재하면, 상담사가 인지하지
  못한 채 "두 Contact가 같은 번호/이메일을 각각 갖고 있는" 데이터
  불일치 상태가 방치된다.

## 2. 새 설계 방향 (대표님 확정)

**핵심 원칙: 백엔드는 자동으로 아무것도 하지 않는다. 상담사가 Case-Contact
assign을 확정하는 시점에 명시적으로 확인받는다.**

square-admin에서 상담사가 `handleAttach`를 눌러 assign을 확정하는
시점(= `PUT contact_cases/{caseId}` 직전/직후)에, 프론트엔드가 그
Case의 `Peer.Type`+`Peer.Target`으로 `contact_addresses`를 조회하여
아래 3가지 시나리오로 분기한다.

### 시나리오 1 — Peer 주소가 `contact_addresses`에 전혀 없음 (완전 신규)

→ **모달 A** 표시:

> 아직 등록되지 않은 주소입니다. contact에 주소를 할당할까요? 주소가
> 할당되면 interaction에서 활동 내역을 확인할 수 있습니다.

- 버튼: 확인 / 취소
- 체크박스: "결정 기억하기"
- 확인 시: `POST /contact_addresses` (body: `{type, target, contact_id}`
  — 이 엔드포인트는 이미 `contact_id`를 받아 생성과 동시에 귀속시키는
  것을 지원함, §4 확인).

### 시나리오 2 — Peer 주소가 있고, 이번에 할당하려는 그 Contact가 이미 소유

→ 아무 것도 하지 않는다. 모달 없이 스킵(이미 일치 상태).

### 시나리오 3 — Peer 주소가 있고, 다른(살아있는) Contact가 소유

→ **모달 B** 표시:

> 이미 다른 컨택트({기존 소유 Contact의 display_name}, 링크: 그
> Contact 상세 페이지로 이동)에 할당되어 있는 주소입니다. 이
> 컨택트로 재할당할까요?

- 버튼: 확인 / 취소
- 체크박스: "결정 기억하기" (**시나리오 1과 별도 키로 관리** — 재할당은
  위험도가 다르다는 것을 사용자가 명시적으로 선택할 수 있어야 함)
- 확인 시: 기존 Contact로부터 이 주소의 소유권을 해제하고 이번
  Contact로 재할당한다(**단일 `reassign` API 호출로 원자적 처리** —
  프론트가 release와 claim을 별도 API로 순차 호출하는 것이 아니다.
  release/claim은 백엔드 내부에서 하나의 트랜잭션 안에 있는
  CLOSE→OPEN 순서일 뿐이다. 상세: §4.4).

### Case-Contact 연결과 주소 연동의 순서

`PUT contact_cases/{caseId}`(Case.contact_id 갱신)는 상담사가 명시적으로
요청한 **주 동작**이므로 먼저 처리하고 성공시킨다. 주소 연동(모달 확인
후의 후속 API 호출)은 **그 다음 별도 단계**로 수행한다.

- 주소 연동이 실패해도 이미 성공한 Case-Contact 연결은 **롤백하지
  않는다**. 이전 설계(§7 폐기분)와 동일한 "주 동작/부가 동작 실패 격리"
  원칙을 유지한다 — 다만 이번 설계에서는 부가 동작이 백엔드 자동
  로직이 아니라 **프론트엔드가 모달 확인 후 호출하는 명시적 API 호출**
  이라는 점이 다르다. 실패 시 프론트는 에러를 상담사에게 알리고(토스트
  등), 재시도 수단을 제공하거나 최소한 "주소 연동은 실패했지만 Contact
  할당 자체는 완료됨"을 명확히 전달한다(구현 단계에서 UX 상세화).

## 3. "결정 기억하기" — localStorage 저장 방식

**서버 저장이 아니다. 브라우저 `localStorage`에 저장한다.**

이유:
- 이 값은 "다음부터 이 모달을 다시 보여줄지"를 결정하는 순수 클라이언트
  UX 설정이다. 서버로 전송되거나 다른 기기/세션과 동기화될 필요가 없다
  — 서버 저장(예: Agent 설정 API)은 불필요한 백엔드 변경과 API 표면을
  추가할 뿐이다(YAGNI).
- 쿠키가 아니라 localStorage인 이유: 쿠키는 매 HTTP 요청에 자동
  첨부되어 서버로 전송되는데, 이 값은 서버가 알 필요가 전혀 없다(순수
  프론트 상태). localStorage는 전송 오버헤드 없이 브라우저에만 남는다.

**시나리오 1과 시나리오 3은 완전히 별도의 localStorage 키로 각각
기억한다** — 사용자가 위험도가 다르다고 명시적으로 결정했기 때문이다.

제안 키 네이밍:
```
voipbin.caseContactAttribution.rememberNewAddress.<agentId>      // 시나리오 1
voipbin.caseContactAttribution.rememberReassignAddress.<agentId> // 시나리오 3
```
값 형식 예: `{"decision": "always_yes" | "always_no", "setAt": "<ISO8601>"}`.
Agent별로 키를 분리하는 이유: 같은 브라우저 프로필을 여러 상담사가
공유하는 콘솔 환경(shared workstation)에서 한 상담사의 "항상 예"
결정이 다른 상담사에게 새어나가지 않도록 하기 위함.

"기억하기"가 켜져 있으면 다음부터는 모달을 띄우지 않고 저장된 결정을
자동 적용한다(`always_yes`면 API 호출을 자동 실행, `always_no`면
아무 것도 하지 않고 스킵).

**리셋 수단은 이번 스코프에서 제공하지 않는다(YAGNI)** — 상담사가
개발자 도구로 localStorage를 지우거나, 향후 필요 시 설정 화면에
"이 확인 모달 다시 보기" 버튼을 추가할 수 있다는 점만 남겨둔다.

## 4. 백엔드 gap — `release` API 신설

### 4.1 현재 상태 재확인

- `contact_addresses`는 **hard-delete** 테이블이다(§0-3). `DELETE
  /contact_addresses/{id}`는 레코드를 완전히 삭제한다 — name/detail/
  is_primary 등 메타데이터까지 소실되므로 "재할당을 위한 임시 해제"
  용도로 쓸 수 없다.
- `POST /contact_addresses/{id}/claim`은 unresolved(contact_id IS
  NULL) 주소에만 성공한다. 이미 살아있는 Contact 소유면 `dbhandler.
  ErrConflict` → `cerrors.AlreadyExists` → HTTP 409
  (`contacthandler.ClaimAddress`, contact.go:470-514).
- **살아있는 소유자로부터 주소를 "놓아주는"(레코드는 유지, contact_id만
  NULL로 되돌리는) 대칭 오퍼레이션이 없다.**
- **중요(Round 1 리뷰 반영): `contact_addresses.contact_id`는
  단독으로 소유권의 진실 소스가 아니다.** `bin-contact-manager`에는
  이미 `contact_address_ownership_periods` 테이블을 관리하는 별도
  서브시스템(`pkg/dbhandler/address_ownership.go`,
  `address_ownership_write.go`)이 존재하며, `contact_id`를 변경하는
  모든 기존 쓰기 경로(`AddressClaimTx`, `AddressDeleteTx`,
  `AddressUpdateTx`, `AddressCreateTx`)가 예외 없이 이 period
  테이블을 함께 갱신한다. 아래 4.3/4.4의 release/reassign 설계는
  `contact_addresses.contact_id`만 UPDATE하는 것이 아니라, 이
  서브시스템이 이미 제공하는 헬퍼(`closeOwnOpenPeriodTx`,
  `OwnershipPeriodsLockAndResolveTx`, `applyOpenResolutionTx`)를
  그대로 재사용해 period 테이블도 함께 갱신하는 것을 전제로 한다.
  이 헬퍼들을 우회하고 `contact_id` 컬럼만 UPDATE하면 소유권 이력
  데이터가 깨지는 실제 버그가 된다.

**재사용할 기존 헬퍼(재구현 금지):**
- `OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID,
  addrType, target) (step int, lockedRows []OwnershipPeriod, err
  error)` — 해당 target(type+target)의 period 행들을 `SELECT ... FOR
  UPDATE`로 잠그고, 다른 살아있는 Contact가 이미 open period를
  갖고 있으면 `ErrConflict`를 반환한다(orphan/tombstone이면 자동으로
  닫고 계속 진행). 그 외에는 `StepOpenReuse` /
  `StepReopen` / `StepInsertAfterIntervening` / `StepReassign` /
  `StepFirstRegistration` 중 하나의 step을 반환한다.
- `applyOpenResolutionTx(ctx, tx, step, lockedRows, customerID,
  contactID, addrType, target, validFromOnInsert) error` — 위 step에
  따라 실제 open/reopen/insert를 수행하는 "OPEN하는 쪽" 헬퍼.
  `AddressClaimTx`가 이미 사용 중.
- `closeOwnOpenPeriodTx(ctx, tx, customerID, contactID, addrType,
  target) error` — `OwnershipPeriodsLockAndResolveTx`를 내부에서
  호출해 lockedRows를 얻고, 그중 `contactID`가 소유한 열린 row를
  찾아 닫는 "CLOSE하는 쪽" 헬퍼. `AddressDeleteTx`가 이미 "먼저
  닫고 나서 지운다" 패턴으로 사용 중(address_ownership_write.go:
  573-576).

**신규로 도입하는 헬퍼 (기존 코드베이스에 없음, 이번 설계가 처음
제안):**
- `addressSetContactIDTx(ctx, tx, addressID, expectedCurrentContactID
  *uuid.UUID, newContactID *uuid.UUID) error` — `contact_addresses.
  contact_id`만 갱신하는 단순 UPDATE 래퍼. `WHERE id = ? AND
  contact_id <=> <expectedCurrentContactID>`(NULL 허용 비교)로 조건을
  걸고, `RowsAffected == 0`이면 `ErrConflict`를 반환한다(Round 2/3
  리뷰로 확정된 스큐 케이스 안전망, §4.3/§4.4 참조). 위 세 헬퍼가
  "소유권 검증/변경"을 담당하는 것과 달리, 이 함수는 그 검증이 이미
  끝난 뒤의 단순 컬럼 쓰기 + 최종 안전망 역할만 한다 — "소유권 검증
  자체를 새로 만들지 않는다"는 원칙과 모순되지 않는다.

### 4.2 신규 엔드포인트: `POST /contact_addresses/{id}/release`

`claim`의 대칭 오퍼레이션. 요청자가 지정한 `contact_id`(현재 소유자와
일치하는지 서버에서 검증)로부터 해당 주소를 unresolved(`contact_id =
NULL`) 상태로 되돌리고, 대응하는 ownership period도 함께 닫는다.

**요청:**
```
POST /v1/contact_addresses/{id}/release
Body: { "contact_id": "<현재 소유자로 기대되는 Contact ID>" }
```
`claim`이 target contact_id를 body로 받는 것과 대칭적으로, release도
"어느 Contact로부터 놓아주는지"를 명시적으로 받는다 — 클라이언트가
자신이 관찰한 소유자와 서버 상태가 다르면(경쟁 상태로 이미 다른
곳에서 release/reassign됨) 409로 실패시켜 조용한 오작동을 막는다.

**응답:** 200 + 갱신된 `ContactManagerAddress`(contact_id: null).
- 404: 주소 없음 또는 customer 불일치.
- 409: `contact_id`가 현재 소유자와 다름(이미 release/reassign됨,
  또는 애초에 unresolved였음) — `closeOwnOpenPeriodTx`가 반환하는
  `ErrConflict`가 그대로 이 케이스에 해당한다(4.3 참조).

### 4.3 dbhandler 레이어 구현 방향

`AddressDeleteTx`가 "닫고 나서 지운다"인 것과 대칭으로, `AddressRelease
Tx`는 **"닫고 나서 NULL로 되돌린다"**로 설계한다. `AddressClaim`
(157-202행)/`addressClaimAttempt`(204-255행)의 tx 안전성 패턴(pre-lock
읽기로 idempotent 조기 반환 + deadlock 재시도 루프)은 그대로 거울처럼
따르되, 소유권 검증/변경 자체는 새로 만들지 않고 4.1의 기존 헬퍼에
위임한다.

```go
func (h *handler) AddressReleaseTx(
    ctx context.Context, tx *sql.Tx,
    customerID uuid.UUID, addressID uuid.UUID,
    fromContactID uuid.UUID, addrType commonaddress.Type, target string,
) error {
    // 1. CLOSE: fromContactID가 실제로 이 target에 대한 open period를
    //    갖고 있는지 검증하면서 동시에 닫는다.
    if err := h.closeOwnOpenPeriodTx(ctx, tx, customerID, fromContactID, addrType, target); err != nil {
        return err // ErrConflict: fromContactID 소유 아님 → 그대로 상위로 전파, 409
    }

    // 2. contact_addresses.contact_id를 NULL로 되돌린다.
    //    WHERE 절에 contact_id = fromContactID를 반드시 포함하고,
    //    RowsAffected == 0이면 ErrConflict를 반환한다 (Round 2 리뷰
    //    수정 — 아래 "스큐 케이스 안전망" 참조. id만으로 조건 없이
    //    UPDATE하지 않는다).
    return h.addressSetContactIDTx(ctx, tx, addressID, fromContactID, nil)
}
```

- **post-lock 재확인 로직을 별도로 구현하지 않는다 — 단, 이는
  lockedRows가 비어있지 않은 경우에만 온전히 성립한다.**
  `closeOwnOpenPeriodTx`는 내부적으로
  `OwnershipPeriodsLockAndResolveTx`를 호출해 `SELECT ... FOR UPDATE`
  로 행을 잠그고, 그 잠금 하에서 `fromContactID`가 실제로 열린
  period를 소유하고 있는지 검증한다. lockedRows가 비어있지 않으면
  소유하고 있지 않은 경우(경쟁 상태로 이미 다른 곳에서
  release/reassign됐거나, 애초에 기대와 다른 소유자였던 경우)
  `ErrConflict`를 반환한다 — 이것이 §4.2가 요구하는 "기대한 소유자와
  다르면 실패"를 만족시킨다. v1 설계가 상상했던
  "`addressTypeTargetContactByID`로 별도 post-lock 재확인" 로직은
  **이 경우에는 불필요하다** — `closeOwnOpenPeriodTx`가 이미 그
  역할을 겸한다.
- **스큐 케이스 안전망(Round 2 리뷰에서 발견된 TOCTOU 결함 수정):**
  lockedRows가 비어있는(스큐) 경우 `closeOwnOpenPeriodTx`는 **아무
  검증도 하지 않고** 에러 없이 성공 반환한다
  (`address_ownership_write.go:625-629`, 롤링 배포 버전 스큐 상태를
  전제로 한 의도적 동작). 이 분기에서는 `fromContactID`가 실제
  소유자인지 검증되지 않은 채로 통과하므로, §4.2의 "기대한 소유자와
  다르면 실패"가 이 경로에서는 **성립하지 않는다** — 1단계만으로는
  release/reassign 요청을 안전하게 승인할 수 없다.
  `AddressDeleteTx`(`address_ownership_write.go:591-605`)는 이
  구조적 gap을 알고, 자신의 최종 DELETE에서 `RowsAffected == 0`이면
  `ErrStaleTarget`을 반환하는 안전망을 별도로 둔다("B5 fix" 주석).
  `AddressReleaseTx`/`AddressReassignTx`도 **동일한 구조**(조건부
  WHERE + RowsAffected 체크)의 안전망을 최종 쓰기에 두되, **반환
  에러 타입은 `AddressDeleteTx`와 다르게 `ErrConflict`로 확정한다**
  (Round 3 리뷰로 확정 — 이유는 아래 §4.5 참조, `ErrStaleTarget`을
  쓰면 claim의 재시도 루프가 이 실패를 자동 재시도해버려 §4.5의
  "즉시 409" 정책과 충돌한다): **`addressSetContactIDTx`는 `UPDATE
  contact_addresses SET contact_id = ? WHERE id = ? AND contact_id =
  <expected-from-value>`로 조건을 걸고, `RowsAffected == 0`이면
  `ErrConflict`를 반환한다.** id만으로 조건 없이 UPDATE하지 않는다
  — 이 조건절이 스큐 케이스에서 `closeOwnOpenPeriodTx`가 놓친
  소유자 검증을 최종 쓰기 시점에 대신 수행하여, "스큐 상태에서
  오래된 화면을 본 상담사가 방금 다른 Contact가 획득한 소유권을
  조용히 덮어쓰는" 실패 모드를 차단한다.
- 이미 unresolved(`contact_id IS NULL`)인 idempotent 케이스는
  pre-lock 읽기(`AddressGet`) 단계에서 조기 반환한다(claim의 "이미
  이 contact 소유면 no-op" 패턴과 대칭).
- `AddressClaimTx`가 사용하는 것과 동일한 deadlock 재시도 루프
  (`addressMaxDeadlockRetries`)를 그대로 재사용한다 — 새 상수를
  만들지 않는다.

**Tombstone 케이스는 release에 해당 없음:** 소유자가 이미
tombstone된(soft-delete) 상태의 orphan period는
`OwnershipPeriodsLockAndResolveTx`가 자동으로 닫고 넘어가는 내부
로직이 이미 처리한다(claim의 repair-in-place 경로와 공유). release
는 "살아있는 소유자로부터 놓아주는" 것이 전제이므로 이 케이스를
별도로 다룰 필요가 없다 — `closeOwnOpenPeriodTx`가 이미 위임받은
`OwnershipPeriodsLockAndResolveTx`를 통해 처리한다.

### 4.4 재할당(release → claim) 흐름: 원자적 단일 엔드포인트로 묶는다

**결론: `POST /contact_addresses/{id}/reassign` (body:
`{new_contact_id}`)을 신설하여 release(CLOSE)+claim(OPEN)을 하나의 DB
트랜잭션으로 묶는다.** 프론트가 release 호출 후 claim을 순차 호출하는
2-step 방식은 채택하지 않는다.

근거:
- `AddressReleaseTx`(§4.3)는 `closeOwnOpenPeriodTx` 호출로 끝나고,
  `AddressClaimTx`는 `OwnershipPeriodsLockAndResolveTx` +
  `applyOpenResolutionTx` 호출로 시작한다 — 둘 다 이미 "tx 인자를
  받는" 형태로 분리되어 있으므로, reassign은 이 두 시퀀스를 **하나의
  트랜잭션 안에서 순서대로 호출**하기만 하면 된다. 새로운 트랜잭션
  패턴이나 새로운 잠금 로직을 발명하지 않는다.
- 2-step(프론트가 release 다음 claim을 별도 HTTP 호출)으로 두면,
  release는 성공했는데 claim이 실패하는 중간 상태("해제됐는데
  재할당 안 됨" — 그 사이 주소는 unresolved로 노출되어 제3의
  프로세스가 먼저 claim해갈 수도 있음)가 구조적으로 가능해진다. 이는
  요구사항 원문이 명시적으로 우려한 실패 모드다. 원자적 단일
  엔드포인트는 이 창(window)을 원천 차단한다.

**`AddressReassignTx` 내부 순서(단일 tx):**
```go
func (h *handler) AddressReassignTx(
    ctx context.Context, tx *sql.Tx,
    customerID uuid.UUID, addressID uuid.UUID,
    fromContactID uuid.UUID, newContactID uuid.UUID,
    addrType commonaddress.Type, target string,
) error {
    // 1. CLOSE: fromContactID 소유 검증 + period 닫기.
    //    (§4.3과 동일한 이유로 이것이 "기대한 소유자 검증"을 대신한다.)
    if err := h.closeOwnOpenPeriodTx(ctx, tx, customerID, fromContactID, addrType, target); err != nil {
        return err // ErrConflict → 409 (§4.5)
    }

    // 2. OPEN: newContactID에 대해 다시 잠금+상태 조회.
    //    1단계에서 fromContactID의 open period를 이미 닫았으므로,
    //    여기서 "다른 살아있는 Contact 소유" 충돌(ErrConflict)에
    //    걸리지 않는다 — 방금 우리가 닫았기 때문이다.
    step, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, newContactID, addrType, target)
    if err != nil {
        return err
    }

    // 3. validFrom 계산은 AddressClaimTx와 동일한 로직(step에 따라
    //    reopen/insert 시 사용할 valid_from을 결정).
    validFrom := computeValidFromForStep(step, lockedRows)

    // 4. OPEN 실행.
    if err := h.applyOpenResolutionTx(ctx, tx, step, lockedRows, customerID, newContactID, addrType, target, validFrom); err != nil {
        return err
    }

    // 5. 최종 상태를 한 번만 쓴다 — 중간에 NULL을 거치는 별도 UPDATE는
    //    불필요. §4.3과 동일한 스큐 케이스 안전망: WHERE 절에
    //    contact_id = fromContactID를 포함하고 RowsAffected == 0이면
    //    ErrConflict (Round 2 리뷰 수정).
    return h.addressSetContactIDTx(ctx, tx, addressID, fromContactID, &newContactID)
}
```

**설계 근거 메모(리뷰 대비 명시):**
- 2단계에서 `OwnershipPeriodsLockAndResolveTx`를 **다시 호출**하는
  것이 맞다 — 1단계의 lockedRows를 재사용하지 않는다. 이유: 두
  호출은 서로 다른 대상(`fromContactID` vs `newContactID`)에 대한
  잠금/상태 질의이며, 1단계에서 이미 해당 트랜잭션 내에서 UPDATE
  (period close)를 실행했으므로 1단계 시점에 읽은 lockedRows는
  이제 stale하다. 같은 tx 내 `SELECT ... FOR UPDATE`는 재호출
  비용이 낮고(동일 tx이므로 추가 락 대기 없음), stale 데이터로
  `applyOpenResolutionTx`를 호출하는 위험을 피하는 것이 안전하다.
- `contact_addresses.contact_id`의 UPDATE는 **1회만** 실행한다(최종
  값 `newContactID`로 직접). v1 설계가 "release 먼저 UPDATE NULL,
  그 다음 claim UPDATE new_contact_id"의 2회 UPDATE로 설계했던 것을
  1회로 단순화한다 — 같은 트랜잭션 안에서 중간값을 거칠 이유가 없고,
  ownership period 쪽의 CLOSE/OPEN 시퀀스만 순서가 의미 있다.

**엔드포인트 표면:** `POST /v1/contact_addresses/{id}/reassign`,
body `{"from_contact_id": "<현재 소유자, 경쟁 상태 검증용>",
"new_contact_id": "<대상 Contact>"}`. `from_contact_id`를 받는 이유는
release의 것과 동일 — 클라이언트가 관찰한 소유자와 서버 상태가
다르면 조용히 다른 Contact를 덮어쓰지 않고 409로 실패시키기 위함.

`release`(§4.2)는 그래도 별도 엔드포인트로 유지한다 — "재할당"이
아니라 순수하게 "주소를 unresolved 풀로 되돌리기만" 하고 싶은 향후
유스케이스(예: 잘못 claim된 주소를 그냥 놓아주기)에 필요할 수 있는
독립적인 프리미티브이기 때문. 다만 **이번 VOIP-1270 스코프(시나리오
3)의 실제 호출은 `reassign` 하나로 처리하고, 프론트는 release를
별도로 호출하지 않는다.**

### 4.5 Race condition 처리 방침

동시에 여러 상담사가 같은 주소를 서로 다른 Contact로 재할당 시도하는
경쟁 상태.

**claim의 실제 경합 처리 패턴 재확인:** `bin-contact-manager/pkg/
dbhandler/address.go`의 `AddressClaim`/`addressClaimAttempt`는
"즉시 409"가 아니라, `ErrStaleTarget`을 받으면 **재시도**하는
for 루프 패턴이다(pre-lock에서 읽은 상태와 락 하에서 재확인한 상태가
다르면 재시도해서 최신 상태로 다시 시도 — 즉 "누가 먼저 요청했는지"의
경합은 재시도로 흡수하고, 사용자에게는 최종 결과만 보여준다).

release/reassign은 이와 **다른 정책을 채택한다** — 아래 3개 에러
발생 지점 모두 재시도하지 않고 즉시 409로 표면화한다:

1. `closeOwnOpenPeriodTx`가 반환하는 `ErrConflict`(fromContactID가
   기대한 open period를 소유하고 있지 않음, lockedRows가 비어있지
   않은 경우) — **즉시 409**. 근거: claim의 `ErrStaleTarget` 재시도는
   "같은 목표(이 주소를 내가 갖는다)를 향한 낙관적 재확인"이므로
   재시도가 타당하지만, release/reassign의 `from_contact_id`는
   "사용자가 화면에서 본 현재 소유자"에 대한 **명시적 사용자 입력
   검증**이다. 이 값이 서버 상태와 다르다는 것은 "그 사이 다른
   상담사가 이미 이 주소를 움직였다"는 뜻이며, 재시도로 자동
   흡수하면 사용자가 화면에서 본 것과 다른 대상에 대해 조용히
   성공해버리는(silent takeover) 위험이 있다. 따라서 claim과 달리
   즉시 409로 실패시키고, 사용자가 최신 상태를 다시 확인한 뒤
   재시도하도록 한다.
2. 2단계(`OwnershipPeriodsLockAndResolveTx(newContactID, ...)`)가
   반환할 수 있는 `ErrConflict`(newContactID로 열려는 시점에 또 다른
   제3의 Contact가 끼어들어 이미 open period를 가진 경우) — **즉시
   409**. 동일한 이유(사용자 입력 검증 실패로 취급).
3. **최종 `addressSetContactIDTx`의 스큐 케이스 안전망이 반환하는
   `ErrConflict`(§4.3/§4.4, Round 2/3 리뷰로 신규 도입)** — **즉시
   409**. 이 에러 타입을 `AddressDeleteTx`가 쓰는 `ErrStaleTarget`이
   아니라 `ErrConflict`로 명시적으로 확정한 이유가 바로 이 정책과의
   일관성이다: `addressClaimAttempt`의 재시도 루프
   (`bin-contact-manager/pkg/dbhandler/address.go:189`)는 `err ==
   ErrDeadlock || err == ErrStaleTarget`을 함께 재시도 대상으로
   묶어 처리한다. §4.3/§4.4가 "`AddressClaimTx`가 사용하는 것과
   동일한 deadlock 재시도 루프를 그대로 재사용한다"고 할 때, 만약
   최종 안전망이 `ErrStaleTarget`을 반환했다면 이 루프가 그것도
   함께 자동 재시도해버려 위 1/2번 항목이 명시한 "즉시 409" 정책과
   직접 충돌했을 것이다(재시도 루프는 에러 타입만으로 분기하지,
   "어느 함수가 반환했는지"는 구분하지 않는다). `ErrConflict`로
   확정함으로써 재시도 루프의 재사용 범위를 `ErrDeadlock`(순수 DB
   잠금 경합)에만 한정시키고, 신원/소유권 검증 실패는 항목 1/2와
   동일하게 즉시 409로 일관되게 처리한다.
- `ErrDeadlock`(DB 레벨 잠금 경합, 신원 충돌과 무관)만 기존
  `addressMaxDeadlockRetries` 루프로 재시도한다 — claim이 이미
  쓰는 패턴을 재사용하되, 재시도 대상은 `ErrDeadlock` 하나로
  한정한다(위 3개 `ErrConflict` 발생 지점은 재시도 루프에 걸리지
  않도록 명시적으로 구분해서 처리해야 함 — 구현 시 `addressClaim
  Attempt`처럼 `ErrDeadlock || ErrStaleTarget`을 함께 묶지 말고,
  `ErrDeadlock`만 단독으로 검사하는 조건으로 release/reassign
  전용 재시도 루프를 작성한다).
- 프론트 UX: 409를 받으면 "다른 상담사가 방금 이 주소를 변경했습니다.
  최신 상태를 다시 확인해주세요"류의 에러를 보여주고, 모달을 닫은 뒤
  Case-Contact 연결(주 동작)은 이미 완료되었음을 유지한다(§2의
  실패 격리 원칙).

## 5. API/OpenAPI 변경 요약

- **신규:** `bin-openapi-manager/openapi/paths/contact_addresses/id_release.yaml`
  — `POST /contact_addresses/{id}/release`. `id_claim.yaml`의 구조를
  그대로 미러링(요청 바디만 `contact_id` 시맨틱이 "해제 대상"으로
  달라짐, 응답/에러 코드 형태는 동일 200/400/401/403/404/409/500).
  내부적으로 ownership period 서브시스템(§4.1/§4.3)의
  `closeOwnOpenPeriodTx`를 재사용하므로 API 계약 자체는 v1과 동일 —
  변경은 dbhandler 내부 구현에 국한된다.
- **신규:** `bin-openapi-manager/openapi/paths/contact_addresses/id_reassign.yaml`
  — `POST /contact_addresses/{id}/reassign`, body
  `{from_contact_id, new_contact_id}`. 이번 VOIP-1270 스코프에서
  실제로 프론트가 호출하는 것은 이 엔드포인트다(§4.4). 내부적으로
  ownership period CLOSE(fromContactID)→OPEN(newContactID)을 단일
  트랜잭션에서 수행하며(§4.4), `contact_addresses.contact_id`는
  1회만 UPDATE한다 — API 계약은 v1과 동일, 변경은 dbhandler 내부
  구현에 국한된다.
- **기존 재사용, 변경 없음:** `POST /contact_addresses` — 이미
  `contact_id`(optional)를 받아 생성과 동시에 귀속시키는 것을
  지원한다(`bin-openapi-manager/openapi/paths/contact_addresses/main.yaml:74-81`,
  `listenhandler/v1_contact_addresses.go:92-127`의
  `processV1ContactAddressesPost`가 `reqData.ContactID != uuid.Nil`
  분기에서 `contactHandler.AddAddress`를 호출). 시나리오 1(모달 A
  확인)은 이 기존 엔드포인트를 그대로 쓴다 — 신규 백엔드 변경
  불필요.
- **기존 재사용, 변경 없음:** `Peer.Type`/`Peer.Target`으로
  `contact_addresses`를 조회하는 경로. 프론트가 모달 분기를
  판단하려면 이 조회가 필요하다 — `GET /contact_addresses?type=
  ...&target=...` 형태의 쿼리가 필요하며, 이전 설계(§4, 폐기분)에서
  검토했던 "`target` 쿼리 파라미터가 `AddressList`에 없다"는 gap이
  **이번 설계에도 여전히 유효한 선결 작업**이다. `GET
  /v1/contact_addresses`에 `target`(string, optional) 파라미터를
  추가하고, `dbhandler.AddressList`(address.go:288-353)에
  `filters["target"]` 처리를 추가한다(`sq.Eq{"target": t}`,
  `type` 필터와 동일한 패턴). `listenhandler/v1_contact_addresses.go`
  의 `processV1ContactAddressesGet`(23-77행)에 `target` 쿼리 파싱을
  추가한다. 이 변경은 폐기된 이전 설계의 §4와 동일한 내용이지만,
  이번 설계에서는 백엔드 자동 판단이 아니라 **프론트엔드가 모달
  분기를 판단하기 위해** 호출한다는 점이 다르다.
- `bin-api-manager`: OpenAPI 변경(release/reassign/target 필터)에
  따른 servicehandler/타입 재생성 반영. 패턴은 기존
  `ServiceHandler.AddressClaim` 래퍼(존재한다면 이를 참고)와 동일하게
  `AddressRelease`/`AddressReassign` 서비스핸들러 메서드 추가.

## 6. 프론트엔드(square-admin) 반영 범위

**대상 파일:** `square-admin/src/views/contacts/contact_cases_detail.js`
의 `CaseContactAttributionPanel` 컴포넌트, 트리거 지점은 기존
`handleAttach` 함수(339-353행).

### 6.1 `handleAttach` 흐름 재설계

현재 `handleAttach`는 `PUT contact_cases/{caseId}` 한 번으로 끝난다.
새 흐름:

```
handleAttach:
  1. PUT contact_cases/{caseId} { contact_id: selectedContact.id }  (주 동작, 기존과 동일)
     실패 시 → 기존과 동일하게 에러 표시, 이후 단계 진행 안 함.
  2. 성공 시 → 이 Case의 Peer(type, target)로
     GET contact_addresses?type=<Peer.Type>&target=<Peer.Target>&customer_id=... 조회
     (§5의 신규 target 필터 사용)
  3. 조회 결과로 분기:
     a) 0건 → localStorage[rememberNewAddress] 확인.
        - "항상 예" 저장돼 있으면 즉시 POST /contact_addresses
          {type, target, contact_id: selectedContact.id} 실행.
        - "항상 아니오" 저장돼 있으면 아무 것도 안 함.
        - 저장된 결정 없으면 모달 A 표시 → 확인 시 위 POST 실행 (+
          체크박스 켜져 있으면 localStorage에 결정 저장).
     b) 1건, contact_id === selectedContact.id → 아무 것도 안 함(시나리오 2).
     c) 1건, contact_id !== selectedContact.id (다른 살아있는 Contact) →
        localStorage[rememberReassignAddress] 확인.
        - "항상 예" → 즉시 POST /contact_addresses/{id}/reassign
          {from_contact_id: 기존 contact_id, new_contact_id: selectedContact.id}
        - "항상 아니오" → 아무 것도 안 함.
        - 저장된 결정 없으면 모달 B 표시(기존 소유 Contact의
          display_name 조회 필요 — GET contacts/{contact_id}) →
          확인 시 위 reassign 호출 (+ 체크박스 켜져 있으면 저장).
  4. 2~3단계(주소 연동)의 성공/실패와 무관하게, 1단계가 이미
     성공했으면 onAttributionChange()는 호출한다(주 동작은 이미
     완료됨). 주소 연동 실패는 별도의 (덜 위협적인) 에러/경고로
     표시한다 — 전체 handleAttach 실패로 취급하지 않는다.
```

### 6.2 신규 컴포넌트(설계만, 목업 없음)

- `AddressClaimModal` (모달 A) — props: `peerTarget`, `peerType`,
  `onConfirm`, `onCancel`, `rememberChecked`, `onRememberChange`.
  카피는 §2 원문 그대로.
- `AddressReassignModal` (모달 B) — props: `peerTarget`, `peerType`,
  `currentOwnerContactId`, `currentOwnerDisplayName`, `onConfirm`,
  `onCancel`, `rememberChecked`, `onRememberChange`. 현재 소유
  Contact의 `display_name`과, 클릭 시 `/resources/contacts/
  contacts_detail/{currentOwnerContactId}`로 이동하는 링크(기존
  `CaseContactAttributionPanel`의 384-389행 링크 패턴과 동일한
  `<Link to=...>` 사용)를 포함한다.
- 두 모달 모두 실제 화면 목업/스크린샷은 **이번 설계 문서 스코프에서
  만들지 않는다**. 구현 단계에서 `voipbin-frontend-visual-verification-gate`
  스킬로 실제 렌더링을 확인한다.

### 6.3 localStorage 유틸

`src/utils/caseContactAttributionPrefs.js`(신규, 제안) 같은 작은
유틸 모듈에 get/set 함수를 두어 컴포넌트에서 직접 `localStorage.
getItem/setItem`을 흩어놓지 않는다. 키 네이밍은 §3 참조.

## 7. 동시성 / 에러 처리 요약

- 백엔드: §4.5 참조 — reassign은 post-lock 재확인 실패 시 즉시
  `ErrConflict`(409)로 실패, 진짜 DB 잠금 경합(`ErrDeadlock`)만
  기존 재시도 루프로 흡수.
- 프론트: 주 동작(Case-Contact 연결)과 부가 동작(주소 연동) 실패를
  분리해서 표시(§2, §6.1-4). 부가 동작 실패는 주 동작 롤백을
  유발하지 않는다.
- 동일 Case에 대한 동시 `UpdateContact` 호출 경쟁은 이번 설계가 새로
  만드는 문제가 아니다(이전 설계 §7과 동일한 결론 — 기존 gap, 별도
  티켓).

## 8. 테스트 계획 (구현 단계에서 상세화)

**백엔드:**
- `AddressRelease`/`addressReleaseAttempt`: unresolved 대상 no-op
  성공 / 기대 소유자와 일치 시 성공 / 기대 소유자와 불일치 시
  `ErrConflict` / 존재하지 않는 주소 `ErrNotFound`.
- `AddressReassignTx`: 정상 재할당(release+claim 원자적 성공) /
  중간에 `from_contact_id` 불일치로 전체 rollback / `new_contact_id`
  쪽에서 제3자가 이미 open period를 가진 경우(§4.5 에러 발생 지점
  ②) `ErrConflict`로 rollback / 대상 Contact가 soft-delete되어
  있으면 실패 / deadlock 재시도 후 성공.
- `AddressList` `target` 필터: 단위 테스트 + OpenAPI 스펙 검증(§5).
- listenhandler: `processV1ContactAddressesIDRelease`,
  `processV1ContactAddressesIDReassign`의 400/404/409 라우팅 테스트
  (기존 `processV1ContactAddressesIDClaim` 테스트 패턴 재사용).

**프론트:**
- `handleAttach`의 3가지 분기(시나리오 1/2/3) 각각에 대해 올바른
  API 호출이 발생하는지.
- localStorage "기억하기" 저장 후 재진입 시 모달 스킵 및 자동 적용
  검증(시나리오 1/3 키가 서로 간섭하지 않는지 포함).
- 주소 연동 API 실패 시에도 Case-Contact 연결(주 동작)의 UI 상태가
  성공으로 유지되는지.

## 9. Out of Scope (YAGNI)

- localStorage 결정 리셋 UI(§3) — 향후 설정 화면에 추가 가능.
- 고객사 단위 정책 플래그(예: 이 확인 흐름 자체를 끄는 옵션).
- `release`(§4.2, reassign에 종속되지 않는 단독 해제) 프론트 UI —
  이번 스코프는 reassign만 프론트에서 호출한다. release 엔드포인트
  자체는 백엔드 프리미티브로만 존재.

# DESIGN: Case Peer Address → Contact Attribution (명시적 모달 확인)

**Issue:** VOIP-1270
**Branch:** VOIP-1270-case-peer-address-contact-claim
**Status:** DRAFT v7 — 대표님 지시로 백엔드 설계 재변경. v6(기존 PUT
에 `contact_id` 필드 추가)을 **SUPERSEDED**하고, 대신 기존 `POST
/contact_addresses/{id}/claim`에 `force`(boolean, optional) 파라미터를
추가해 이미 다른 Contact가 소유한 주소도 덮어쓸 수 있게 확장한다.
`force` 없이는 기존 계약(unresolved 전용, 충돌 시 409) 그대로
유지 — 하위 호환 보장. v5(§4~§10, 8라운드 독립 리뷰 완료분)와 v6은
모두 SUPERSEDED, 리뷰 이력은 §10에 "v5 전용"으로 보존. v7 변경분은
아직 리뷰 루프를 거치지 않음(§11 참조).

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
- 확인 시: **`POST /contact_addresses/{id}/claim` (body:
  `{"contact_id": "<이번 Contact ID>", "force": true}`)를 호출해
  소유권을 강제 재할당한다(v7, §4 참조).** `force: true`가 있으면
  이미 다른 살아있는 Contact가 소유 중이어도 409를 반환하지 않고
  그 소유자로부터 조용히 빼앗아 재할당한다.
- v5(SUPERSEDED, §10)에서는 신규 `reassign` 엔드포인트로,
  v6(SUPERSEDED)에서는 기존 PUT에 `contact_id` 필드를 추가하는
  방식으로 설계했으나, 대표님 지시로 두 방향 모두 걷어내고
  **기존 `claim`을 확장**하는 방향(v7)으로 재정리했다. 근거:
  claim이 이미 갖고 있는 ownership-period 서브시스템 연동(§4
  참조)을 그대로 재사용할 수 있어 v6이 §4.4에서 명시했던
  "이력 테이블과의 의도적 비대칭" 문제 자체가 사라진다.

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

## 4. 백엔드 변경 (v7) — 기존 `POST .../claim`에 `force` 파라미터 확장

> **v5/v6 SUPERSEDED 고지:** v5는 신규 `POST .../release` +
> `POST .../reassign` 2개 엔드포인트(8라운드 독립 리뷰 완료)로,
> v6은 기존 `PUT /contact_addresses/{id}`에 `contact_id` 필드를
> 추가하는 방식으로 설계했었다. 대표님이 두 방향 모두 걷어내고
> 기존 `claim`을 확장하는 쪽으로 재차 지시하여 v7로 전면 교체한다.
> v5의 pseudocode/리뷰 상세는 git 히스토리(`bf5d150c2` ~
> `c2df95890`)에, v6은 `c2df95890` ~ `b59fc428c`에 보존되어 있다.
> §10에 v5 라운드별 요약이 남아 있다.

### 4.1 현재 상태 재확인

- `POST /contact_addresses/{id}/claim`은 unresolved(contact_id IS
  NULL) 주소에만 성공하고, 이미 살아있는 Contact 소유면
  `ErrConflict` → HTTP 409를 반환한다(`contacthandler.ClaimAddress`,
  contact.go:470-514). **이 기본 계약은 v7에서도 그대로 유지한다**
  — `force` 파라미터가 없으면(또는 `false`면) 지금과 동일하게
  동작한다. `contacts_detail.js`의 Unresolved Address Picker(이미
  프로덕션 사용 중, §0/v6 참고 사실)를 포함한 기존 모든 호출자는
  영향받지 않는다.
- **대표님 확인: 현재 이 엔드포인트를 호출하는 외부(square-admin
  이외) 소비자는 없다.** 이 전제 위에서 기본 계약을 깨는 것(예:
  409를 완전히 제거)도 검토했으나, "하위 호환 유지 + 옵트인
  파라미터"가 더 안전한 선택이라 최종적으로 이 방향(§4.2)으로
  확정했다 — 외부 소비자가 없다는 확인이 이 방향의 위험을 낮추긴
  했지만, 그렇다고 기본 계약을 깰 이유가 되는 것은 아니다(향후
  square-admin 내 다른 화면이나 신규 통합이 기존 409 계약에
  의존하는 코드를 작성할 가능성은 여전히 남아있다).
- 확장 지점은 `AddressClaimTx`/`OwnershipPeriodsLockAndResolveTx`가
  아니라 `addressClaimAttempt`(`bin-contact-manager/pkg/dbhandler/
  address.go:204-255`)다(Round 1 리뷰로 확정). 이 함수가
  `AddressClaimTx`를 호출하기 **이전에** 먼저 소유자 검사를 하는
  앞단 게이트를 갖고 있다(226-244행): 이미 다른 Contact 소유
  (`current.ContactID != contactID`)면 `contactTombstoneTx`로
  살아있는지 확인하고, **살아있으면(`tmDelete == nil`) 그 자리에서
  즉시 `ErrConflict`를 반환하며 `AddressClaimTx` 호출까지 가지도
  않는다**(236행).
- **`force: true`일 때의 정확한 동작(Round 2 리뷰로 재수정):**
  `tmDelete == nil`(살아있는 소유자)이어도 `ErrConflict`를 반환하지
  않되, tombstone 케이스가 쓰는 `staleRowRepairTx`는 재사용하지
  않는다 — 이 함수는 살아있는 소유자에 대해 **의도적으로 아무
  것도 갱신하지 않고** `(false, nil)`을 반환하도록 설계되어 있어서
  (tombstone 복구 전용, `address_ownership_write.go:204-224`의
  doc comment), 그대로 재사용하면 이전 소유자의 open ownership
  period가 닫히지 않은 채 `AddressClaimTx`로 넘어가고,
  `AddressClaimTx`가 내부에서 무조건 호출하는
  `OwnershipPeriodsLockAndResolveTx`가 그 열려있는 period를 다시
  발견해 같은 `ErrConflict`를 재반환한다(Round 2 리뷰가 실제로
  발견한 결함 — R1 수정은 증상을 없애지 못했다). **정확한 해법은
  `closeOwnOpenPeriodTx`(`address_ownership_write.go:619-642`)를
  이전 소유자의 contactID로 호출해 그 open period를 먼저 명시적으로
  닫는 것이다** — 이건 v5 설계(§10, SUPERSEDED)가 release/reassign을
  위해 이미 검증했던 CLOSE 헬퍼와 동일한 함수다. 상세 pseudocode는
  §4.3 참조.

### 4.2 확정된 API 표면

**신규 엔드포인트 없음.** 기존 `POST /contact_addresses/{id}/claim`
요청 바디에 `force`(boolean, optional, 기본값 false)를 추가한다.

```
POST /v1/contact_addresses/{id}/claim
Body: { "contact_id": "<uuid>" }                    // 기존과 동일 — unresolved만 성공, 아니면 409
Body: { "contact_id": "<uuid>", "force": true }     // 신규 — 다른 살아있는 Contact 소유여도 덮어씀
```

- `force`가 없거나 `false`: 기존 계약 그대로(§4.1) — 하위 호환.
- `force: true`: `addressClaimAttempt`의 앞단 소유자 검사(§4.1,
  226-244행)에서 살아있는 소유자를 만나도 `ErrConflict`를 반환하지
  않고, `staleRowRepairTx`로 행을 unresolved로 리셋한 뒤
  `AddressClaimTx`로 정상 진행한다(§4.1). **이력 테이블(`contact_
  address_ownership_periods`)에는 "기존 소유자의 period가 닫히고
  새 소유자의 period가 열리는" 정상적인 흐름이 그대로 남는다**
  — `AddressClaimTx`가 내부적으로 `OwnershipPeriodsLockAndResolveTx`
  /`applyOpenResolutionTx`를 그대로 타기 때문이다. 이게 v6 대비
  이 방향의 핵심 이점이다(§0/v6 SUPERSEDED 고지가 지적한 이력
  비대칭 문제가 원천적으로 발생하지 않는다).

**응답:** 200 + 갱신된 `ContactManagerAddress`. `force: true`일 때도
404(주소 없음)는 그대로 유지된다. 409는 `force`가 없을 때만
발생한다.

### 4.3 dbhandler/OpenAPI 변경 지점 (Round 2 리뷰로 재수정)

> **Round 2 리뷰가 지적한 R1 수정의 결함:** R1이 채택한
> `staleRowRepairTx` 재사용은 틀렸다. 이 함수는 살아있는 소유자를
> 만나면 **의도적으로 아무 것도 갱신하지 않고** `(false, nil)`을
> 반환한다(tombstone 복구 전용 함수이기 때문 —
> `address_ownership_write.go:204-224` 함수 자체 doc comment 참고).
> 그래서 `force:true`로 이 경로를 타도 이전 소유자의 ownership
> period가 전혀 닫히지 않은 채 `AddressClaimTx`가 호출되고,
> `AddressClaimTx`가 내부적으로 무조건 호출하는
> `OwnershipPeriodsLockAndResolveTx`의 Step 1이 **독립적으로 다시**
> live-owner 충돌을 검사해 `ErrConflict`를 재반환한다 — 같은 증상이
> 다른 경로에서 재발한다.

- `bin-openapi-manager/openapi/paths/contact_addresses/id_claim.yaml`
  에 `force`(boolean, optional) 필드 추가, description에 하위 호환
  기본 동작과 `force: true`의 의미를 명시.
- **`OwnershipPeriodsLockAndResolveTx` 자체의 시그니처는 여전히
  건드리지 않는다** — Round 1의 이 판단은 유지된다. 다만 Round 1이
  "그러므로 force 처리는 이 함수와 무관하다"고 결론 낸 것은 틀렸다
  — `AddressClaimTx`가 이 함수를 무조건 호출하므로, force:true일
  때 이 함수의 Step 1이 통과되도록(즉 이전 소유자의 open period가
  이미 닫혀 있도록) **사전에** 만들어둬야 한다.
- **정확한 해법: `closeOwnOpenPeriodTx`를 재사용한다**
  (`address_ownership_write.go:619-642` — v5 설계(§10, SUPERSEDED)가
  release/reassign을 위해 이미 검증했던 바로 그 CLOSE 헬퍼). 이
  함수를 **이전 소유자의 contactID**로 호출하면(새 소유자가 아니라)
  `OwnershipPeriodsLockAndResolveTx`의 Step 1이 "이 target에 대해
  **다른** contact_id의 open period가 있는가"를 검사할 때, 우리가
  넘긴 contactID(이전 소유자 자신)와 open period의 contact_id가
  일치하므로 Step 1의 충돌 분기 자체에 걸리지 않는다 — 대신 Step 2
  (`StepOpenReuse`, 자기 자신의 open period 재사용)로 자연스럽게
  이어지고, `closeOwnOpenPeriodTx`가 그 open period를 찾아
  `ownershipPeriodCloseByIDTx`로 닫는다. 닫힌 뒤에 `AddressClaimTx`를
  호출하면, 그 안의 `OwnershipPeriodsLockAndResolveTx`가 다시
  조회했을 때 더 이상 다른 contact의 open period가 없으므로 Step 1
  충돌 없이 정상 진행된다.
- 시그니처 확장은 다음 체인을 관통해야 한다(바깥→안쪽 순):
  `contacthandler.ClaimAddress(force bool)` →
  `dbhandler.AddressClaim(..., force bool)` →
  `addressClaimAttempt(..., force bool)`. `AddressClaimTx` 자체는
  시그니처 변경이 필요 없다(v5 검증 시와 동일한 이유 — CLOSE를
  먼저 끝내면 OPEN 쪽은 아무것도 몰라도 된다).
- `addressClaimAttempt`(`address.go:226-244`)의 정확한 수정:
  ```go
  if current.ContactID != uuid.Nil && current.ContactID != contactID {
      tmDelete, tombErr := contactTombstoneTx(ctx, tx, current.ContactID)
      if tombErr != nil { return tombErr }
      if tmDelete == nil {
          // 살아있는 소유자
          if !force {
              return ErrConflict
          }
          // force:true -- 이전(살아있는) 소유자의 open ownership
          // period를 CLOSE한다. staleRowRepairTx(tombstone 전용,
          // 살아있는 소유자에 대해 no-op)는 재사용하지 않는다
          // (Round 2 리뷰로 확정).
          if err := h.closeOwnOpenPeriodTx(ctx, tx, customerID, current.ContactID, addrType, target); err != nil {
              return err
          }
          // contact_addresses 행 자체의 NULL 리셋은 불필요 --
          // AddressClaimTx의 UPDATE가 id 조건만으로 곧바로 새
          // contact_id를 덮어쓴다.
      } else {
          // tombstone된 소유자: 기존 repair-in-place 경로, 변경 없음.
          if _, repairErr := h.staleRowRepairTx(ctx, tx, customerID, addrType, target, true); repairErr != nil {
              return repairErr
          }
      }
  }
  if err := h.AddressClaimTx(ctx, tx, customerID, addressID, contactID, addrType, target); err != nil {
      return err
  }
  ```
  tombstone 분기와 force 분기는 서로 다른 헬퍼(`staleRowRepairTx`
  vs `closeOwnOpenPeriodTx`)를 쓰므로 명시적으로 분리한다 — R1이
  시도했던 "하나의 리셋 경로로 합류"는 이 두 케이스의 실제 동작이
  다르다는 걸 놓친 데서 비롯된 오류였다(Round 2 리뷰).
- `AddressClaim`(`address.go:165-202`)이 `force`를
  `addressClaimAttempt` 호출부(184행)까지 그대로 전달하도록 파라미터
  추가.
- `contacthandler.ClaimAddress`(servicehandler 상위 레이어)가
  `force` 파라미터를 받아 `dbhandler.AddressClaim`까지 전달하도록
  시그니처 확장.
- **listenhandler 계층(Round 2 리뷰가 지적한 누락):**
  `processV1ContactAddressesIDClaim`이 파싱하는 요청 구조체
  (`request.ContactAddressClaim` 또는 그 대응 타입)에 `Force bool`
  필드를 추가하고, 핸들러가 이를 `servicehandler.ClaimAddress`까지
  전달하도록 배선한다 — §8 테스트 계획에는 이미 언급되어 있었으나
  이 변경 지점 목록에 누락되어 있었다.

### 4.4 이력 서브시스템 — v6과 달리 정합성 유지

v6(SUPERSEDED, §0)에서 지적했던 "PUT으로 바뀐 소유권이 이력
테이블에 반영되지 않는 비대칭" 문제가 v7에서는 발생하지 않는다.
`force: true`도 결국 `AddressClaimTx`의 기존 경로(§4.1)를 그대로
타므로, ownership-period 테이블은 항상 정확하게 갱신된다.

### 4.5 Race condition — 기존 claim 정책 그대로 상속

`force` 유무와 무관하게, 동시성 처리는 `AddressClaimTx`/
`addressClaimAttempt`의 기존 패턴(`bin-contact-manager/pkg/
dbhandler/address.go`)을 그대로 따른다 — pre-lock 읽기로 idempotent
조기 반환, `ErrStaleTarget`/`ErrDeadlock`은 기존 재시도 루프로 흡수.
v5가 설계했던 "release/reassign 전용 즉시 409" 정책은 v7에는
해당하지 않는다 — v7은 새 엔드포인트가 아니라 기존 claim의
파라미터 확장이므로, claim이 이미 갖고 있는 동시성 정책을 그대로
상속받는다. 다만 **`force: true`로 두 상담사가 동시에 같은 주소를
서로 다른 Contact로 강제 claim하는 경쟁 상태**는 새로운 시나리오다
— `addressClaimAttempt`가 매 시도마다 새 트랜잭션을 열고
`addressTypeTargetContactByID`로 post-lock 재확인(§4.3의
`current` 조회)을 하므로 DB 레벨에서 순서가 보장되지만, 사용자
입장에서는 "내가 먼저 확인 버튼을 눌렀는데 나중에 요청한 상담사가
이겼다"는 결과가 나올 수 있다(둘 다 `force: true`이므로 둘 다
성공하고, 마지막 커밋이 최종 소유자가 된다) — 이건 §4.2에서 이미
명시한 "last-write-wins를 사용자가 명시적으로 요청한 것"이므로
별도 안전장치를 추가하지 않는다.


## 6. 프론트엔드(square-admin) 반영 범위

**대상 파일:** `square-admin/src/views/contacts/contact_cases_detail.js`
의 `CaseContactAttributionPanel` 컴포넌트, 트리거 지점은 기존
`handleAttach` 함수(339-353행).

### 6.1 `handleAttach` 흐름 재설계 (v7)

현재 `handleAttach`는 `PUT contact_cases/{caseId}` 한 번으로 끝난다.
새 흐름:

```
handleAttach:
  1. PUT contact_cases/{caseId} { contact_id: selectedContact.id }  (주 동작, 기존과 동일)
     실패 시 → 기존과 동일하게 에러 표시, 이후 단계 진행 안 함.
  2. 성공 시 → 이 Case의 Peer(type, target)로
     GET contact_addresses?type=<Peer.Type>&target=<Peer.Target>&customer_id=... 조회
     (§5의 신규 target 필터 사용 — v7에서도 이 조회는 그대로 필요)
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
        - "항상 예" → 즉시 **POST /contact_addresses/{id}/claim
          {contact_id: selectedContact.id, force: true}** 실행(v7,
          §4.2).
        - "항상 아니오" → 아무 것도 안 함.
        - 저장된 결정 없으면 모달 B 표시(기존 소유 Contact의
          display_name 조회 필요 — GET contacts/{contact_id}) →
          확인 시 위 claim(force: true) 호출 (+ 체크박스 켜져
          있으면 저장).
  4. 2~3단계(주소 연동)의 성공/실패와 무관하게, 1단계가 이미
     성공했으면 onAttributionChange()는 호출한다(주 동작은 이미
     완료됨). 주소 연동 실패는 별도의 (덜 위협적인) 에러/경고로
     표시한다 — 전체 handleAttach 실패로 취급하지 않는다.
```

**v6(SUPERSEDED)와의 차이:** 시나리오 c에서 `PUT /contact_addresses/
{id} {contact_id}` 호출이 `POST /contact_addresses/{id}/claim
{contact_id, force: true}` 호출로 바뀐 것 외에는 흐름이 동일하다.
시나리오 a(모달 A, 신규 생성)는 v6/v7 공통으로 변화 없음 — 여전히
`POST /contact_addresses`(생성+귀속)를 그대로 쓴다.

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

## 7. 동시성 / 에러 처리 요약 (v7)

- 백엔드: §4.5 참조 — `force` 유무와 무관하게 claim의 기존 동시성
  정책(pre-lock 읽기 + `ErrStaleTarget`/`ErrDeadlock` 재시도 루프)을
  그대로 상속받는다. v6이 채택했던 "경합 검증 완전 제거"와 달리,
  v7은 claim이 이미 가진 `SELECT ... FOR UPDATE` 기반 직렬화를
  유지한다 — 다만 두 상담사가 모두 `force: true`로 요청하면 DB
  레벨에서는 순서대로 처리되지만 결과적으로 나중 요청이 이긴다
  (§4.5, 사용자가 명시적으로 요청한 동작으로 간주).
- 프론트: 주 동작(Case-Contact 연결)과 부가 동작(주소 연동) 실패를
  분리해서 표시(§2, §6.1-4). 부가 동작 실패는 주 동작 롤백을
  유발하지 않는다.
- 동일 Case에 대한 동시 `UpdateContact` 호출 경쟁은 이번 설계가 새로
  만드는 문제가 아니다(기존 gap, 별도 티켓).

## 8. 테스트 계획 (구현 단계에서 상세화, v7)

**백엔드:**
- `POST /contact_addresses/{id}/claim` `force` 파라미터:
  - `force` 없음/`false`: 기존 계약 100% 유지 확인(회귀 테스트) —
    unresolved 성공, 이미 소유 시 409.
  - `force: true`, 대상이 unresolved: 기존과 동일하게 성공(force가
    무해한 no-op으로 동작하는지).
  - `force: true`, 대상이 다른 살아있는 Contact 소유: 409 없이
    성공, 기존 소유자의 ownership period가 닫히고 새 소유자의
    period가 열리는지 **DB 레벨로 직접 검증**(§4.4의 "이력 정합성
    유지" 주장을 실제로 고정하는 핵심 테스트).
  - `force: true`, 대상이 tombstone(soft-delete)된 Contact 소유:
    기존 orphan-close 경로와 결과가 동일한지(§4.1/§4.3 — 두
    분기가 같은 `staleRowRepairTx` 리셋 경로로 합류하므로 당연히
    같아야 하지만 회귀로 고정).
  - `force`가 `AddressCreate`/`AddressUpdate`/`AddressDelete`
    경로에는 전혀 영향을 주지 않는지(§4.3 정정 — `force`는
    `addressClaimAttempt`에만 배선되고 `OwnershipPeriodsLockAndResolveTx`
    자체는 건드리지 않으므로, 이 항목은 "회귀 없음"을 확인하는
    네거티브 테스트다).
- `AddressList` `target` 필터: 단위 테스트 + OpenAPI 스펙 검증(§5).
- listenhandler: `processV1ContactAddressesIDClaim`에 `force` 필드
  파싱 추가 테스트(기존 400/404/409 라우팅 테스트에 `force` 케이스
  추가).

**프론트:**
- `handleAttach`의 3가지 분기(시나리오 1/2/3) 각각에 대해 올바른
  API 호출이 발생하는지(특히 시나리오 3에서 claim body가 v7대로
  `{contact_id, force: true}`인지).
- localStorage "기억하기" 저장 후 재진입 시 모달 스킵 및 자동 적용
  검증(시나리오 1/3 키가 서로 간섭하지 않는지 포함).
- 주소 연동 API 실패 시에도 Case-Contact 연결(주 동작)의 UI 상태가
  성공으로 유지되는지.

## 9. Out of Scope (YAGNI, v7)

- localStorage 결정 리셋 UI(§3) — 향후 설정 화면에 추가 가능.
- 고객사 단위 정책 플래그(예: 이 확인 흐름 자체를 끄는 옵션).
- `claim`의 409를 완전히 제거하는 것(대안으로 검토했으나 §4.1에서
  기각 — 기본 계약은 하위 호환 유지).
- `unclaim`(재배정 없이 unresolved로만 되돌리는 액션) — VOIP-1270의
  Case 할당 화면에는 이 액션을 트리거할 UI가 없다(대표님과의 논의로
  확인). Contact 상세 화면 등 다른 곳에서 필요해지면 별도 티켓.

## 10. 독립 디자인 리뷰 루프 기록 (v5 전용, SUPERSEDED)

**주의: 이 리뷰 기록은 v5(§4~§9의 release/reassign POST 설계)에 대한
것이다. v6(현재 §4~§9 본문)은 대표님 지시로 v5를 전면 대체한 단순화
설계이며, 아래 리뷰 루프를 거치지 않았다(§11 참조).** v5의 리뷰
과정 자체는 방법론적으로 유효한 기록이므로 삭제하지 않고 보존한다.

정책: `design-first-with-review-loops` — 최소 3라운드, 이후 연속 2회
APPROVE로 종료. 매 라운드는 별도의 신선한 서브에이전트(독립 컨텍스트)
에게 위임하고, 지적된 결함의 수정은 CPO(오케스트레이터)가 직접
적용해 커밋했다.

| 라운드 | 판정 | 핵심 발견 | 커밋 |
|---|---|---|---|
| R1 | CHANGES_REQUESTED | §4가 기존 `contact_address_ownership_periods` 서브시스템(`OwnershipPeriodsLockAndResolveTx`/`applyOpenResolutionTx`/`closeOwnOpenPeriodTx`)을 완전히 누락 — `contact_addresses.contact_id` 컬럼만 UPDATE하는 설계였다면 소유권 이력 데이터가 깨지는 실제 버그가 됐을 것 | `bf5d150c2` |
| R2 | CHANGES_REQUESTED | `closeOwnOpenPeriodTx`가 스큐(period 테이블 비어있음) 상황에서 소유자 검증 없이 성공 반환 — release/reassign 최종 쓰기에 `AddressDeleteTx`(B5 fix)와 동등한 `RowsAffected==0` 안전망이 없어 TOCTOU 위험 | `c99c6d28f` |
| R3 | CHANGES_REQUESTED | R2 안전망의 반환 에러 타입이 `ErrConflict`/`ErrStaleTarget` 중 확정되지 않음 — `ErrStaleTarget`이면 claim의 기존 재시도 루프(`ErrDeadlock\|\|ErrStaleTarget`)에 자동 흡수되어 "즉시 409" 정책이 무력화됨. `ErrConflict`로 확정 | `bb609b036` |
| R4 | CHANGES_REQUESTED | §2 시나리오3 문구("release 먼저, claim 그 다음")가 §4.4가 명시적으로 거부한 2-step 프론트 호출로 오독될 소지. 문구 수정 + §8 테스트 케이스 보강 | `aa300202b` |
| R5 | CHANGES_REQUESTED | §4.3/§4.4 pseudocode의 `addrType`이 `string`으로 잘못 선언(실제 재사용 헬퍼는 `commonaddress.Type` 요구) — 그대로 구현 시 컴파일 실패 | `2f6ef60ff` |
| R6 | CHANGES_REQUESTED | §4.1 신규 헬퍼 `addressSetContactIDTx`의 `expectedCurrentContactID`가 `*uuid.UUID`(포인터)로 선언됐으나 §4.3/§4.4 호출부는 값 타입을 그대로 전달 — 문서 내부 자기모순, 컴파일 실패. 시그니처를 호출부(값 타입)에 맞춰 수정 | `1b55fbfc6` |
| R7 | **APPROVED** | §4 전체 함수 호출(선언↔호출부)을 한 줄씩 재대조, R5/R6 카테고리(타입/시그니처) 완전 해소 확인. 새 결함 없음 | — |
| R8 | **APPROVED** | 프로덕트/비즈니스 로직 관점(엣지 케이스, 권한, 감사 추적, UX) 검토. 실질적 결함 없음. 비치명적 공백 2건(모달 B의 tombstone 소유자 404 폴백 미명시, release/reassign 감사 이벤트 발행 미명시)은 CPO가 직후 이 최종 커밋에서 문서화로 보완(§2, §5) | 본 커밋 |

**v5 종료 조건 충족(당시):** 최소 3라운드 floor(8라운드로 초과 충족)
+ 연속 2회 APPROVE(R7, R8). 그러나 이후 대표님이 방향 자체를
단순화하도록 지시하여 v6으로 대체됐다 — 위 리뷰가 검증한 코드
(ownership-period 재사용, 에러 타입, 트랜잭션 원자성 등)는 v6에는
더 이상 적용되지 않는다.

## 11. v7 독립 디자인 리뷰 루프 기록

v6에서 v7로의 전환은 대표님과의 대화에서 직접 확정된 방향이며,
design-first-with-review-loops 정책(최소 3라운드, 이후 연속 2회
APPROVE)에 따라 독립 리뷰를 진행 중이다.

| 라운드 | 판정 | 핵심 발견 | 수정 반영 |
|---|---|---|---|
| R1 | CHANGES_REQUESTED | §4.1/§4.3이 지목한 확장 지점(`OwnershipPeriodsLockAndResolveTx` Step 1)이 틀렸다 — 실제로는 `addressClaimAttempt`(`address.go:226-244`)가 더 앞단에서 살아있는 소유자를 만나면 `AddressClaimTx` 호출 전에 즉시 `ErrConflict`를 반환한다. `force`를 문서가 지목한 지점에만 배선하면 시나리오 3(재할당)의 핵심 유스케이스가 그대로 실패한다. §4.1/§4.2/§4.3/§4.5/§8을 `addressClaimAttempt` 기준으로 전면 정정 | `6f5e83b90` |
| R2 | CHANGES_REQUESTED | 확장 지점(`addressClaimAttempt`) 식별은 옳았으나, R1이 채택한 구현(`staleRowRepairTx` 재사용)이 틀렸다 — 이 함수는 살아있는 소유자에 대해 의도적으로 no-op(아무 것도 갱신하지 않고 `(false, nil)` 반환)이라서, `force:true`로 이 경로를 타도 이전 소유자의 open ownership period가 닫히지 않은 채 `AddressClaimTx`가 호출되고, 그 안에서 무조건 호출되는 `OwnershipPeriodsLockAndResolveTx`가 독립적으로 같은 충돌을 재검사해 `ErrConflict`가 재발한다. `closeOwnOpenPeriodTx`(v5가 이미 검증한 CLOSE 헬퍼)로 교체, tombstone/force 분기를 명시적으로 분리. 부차: listenhandler 계층이 §4.3 변경 지점 목록에서 누락됐던 것도 보완 | 본 커밋 |

**현재 상태: DRAFT (리뷰 진행 중, R1·R2 CHANGES_REQUESTED 모두 수정
완료, R3 대기).**


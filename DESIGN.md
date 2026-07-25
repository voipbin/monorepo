# DESIGN: Case Peer Address → Contact Attribution (명시적 모달 확인)

**Issue:** VOIP-1270
**Branch:** VOIP-1270-case-peer-address-contact-claim
**Status:** DRAFT v6 — 대표님 지시로 백엔드 설계 단순화(release/reassign
POST 엔드포인트 폐기 → 기존 `PUT /contact_addresses/{id}`에 `contact_id`
필드 추가로 통합, 경합 검증 없이 last-write-wins). v5(§4~§10, 8라운드
독립 리뷰 완료분)는 **SUPERSEDED** — 리뷰 이력은 §10에 그대로 보존하되
"v5 전용"으로 표기. v6 변경분은 아직 리뷰 루프를 거치지 않음(§11 참조).

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
- 확인 시: **`PUT /contact_addresses/{id}` (body: `{"contact_id":
  "<이번 Contact ID>"}`)를 호출해 소유권을 재할당한다(v6, §4 참조).**
  기존 소유자 검증 없이 last-write-wins로 덮어쓴다 — `contact_cases/
  {id}`의 `contact_id` 필드 갱신 컨벤션과 동일한 단순 방식이다.
- v5(SUPERSEDED, §10)에서는 이 재할당을 `release`+`claim`을 원자적
  트랜잭션으로 묶는 신규 `reassign` 엔드포인트로 설계했으나, 대표님
  지시로 그 복잡도(경합 검증, ownership-period 안전망 등)를 걷어내고
  기존 PUT 하나로 단순화했다(§4/§11 참조).

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

## 4. 백엔드 변경 (v6) — 기존 `PUT /contact_addresses/{id}`에 `contact_id` 확장

> **v5 SUPERSEDED 고지:** 이 섹션은 원래 신규 `POST .../release` +
> `POST .../reassign` 2개 엔드포인트(ownership-period 서브시스템 재사용,
> 경합 시 409, 8라운드 독립 리뷰를 거쳐 완성)로 설계되어 있었다.
> 대표님이 이 복잡도를 걷어내고 기존 PUT 하나로 단순화하도록 지시하여
> 전면 교체한다. v5의 전체 pseudocode/리뷰 상세는 git 히스토리(커밋
> `bf5d150c2` ~ `c2df95890`)에 그대로 보존되어 있다 — 필요 시 그
> 범위에서 ownership-period 서브시스템 재사용 방식, race condition
> 처리 세부는 여전히 유효한 참고 자료다. §10에 라운드별 요약이 남아
> 있다.

### 4.1 현재 상태 재확인

- `contact_addresses`는 **hard-delete** 테이블이다(§0-3). 메타데이터
  소실 문제로 DELETE는 재할당 용도로 못 쓴다는 사실은 여전히 유효
  — 다만 v6은 애초에 DELETE를 쓰지 않으므로 이 제약이 v6 설계
  자체에 직접 영향을 주지는 않는다(참고용으로만 남김).
- `POST /contact_addresses/{id}/claim`은 **그대로 유지한다.**
  실제 프로덕션에서 이미 사용 중이다 — `square-admin/src/views/
  contacts/contacts_detail.js`(263행)의 "Unresolved Address Picker"
  기능(2026-07-02 설계, 이미 배포)이 이 엔드포인트를 호출한다.
  이 기능은 unresolved 주소(아직 아무 Contact도 소유하지 않은 상태)
  전용이고, VOIP-1270의 재할당(이미 다른 Contact가 소유한 주소)
  시나리오와 겹치지 않으므로 폐기하지 않는다.
- `bin-contact-manager`에 이미 존재하는 `contact_address_ownership_
  periods` 서브시스템(§0 v5 참고 사실)은 **v6 설계에서 다루지
  않는다** — release/reassign 전용 안전장치(경합 검증, TOCTOU 방어)
  자체를 걷어냈으므로, 이 ownership-period 테이블과의 연동 로직도
  이번 스코프에서는 만들지 않는다. **주의: 이 서브시스템은
  `AddressCreateTx`/`AddressUpdateTx`/`AddressDeleteTx`/
  `AddressClaimTx`가 이미 갱신하고 있다.** v6의 `PUT
  /contact_addresses/{id}`가 `contact_id` 필드 변경 시 이
  서브시스템을 갱신하지 않으면, 이 PUT으로 바뀐 소유권만 이력에서
  누락되는 **의도적 비대칭**이 생긴다(§4.4 참조 — 리뷰에서 반드시
  검증할 지점으로 남긴다).

### 4.2 확정된 API 표면

**신규 필드가 아니라, 기존 `PUT /contact_addresses/{id}`에 `contact_id`
필드를 추가한다.** 신규 엔드포인트를 만들지 않는다.

```
PUT /v1/contact_addresses/{id}
Body: { "contact_id": "<uuid>" }   // 재할당
Body: { "contact_id": "" }         // 해제(unresolved로 되돌림)
```

`contact_cases/{id}` PUT의 기존 컨벤션(빈 문자열 = 해제, 값 = 배정)을
그대로 미러링한다(`bin-openapi-manager/openapi/paths/contact_cases/
id.yaml:66-77` 참고 — 오는 HTTP 레이어에서는 `format: uuid`를 붙이지
않은 plain string으로 받고, 내부 RPC 구조체에서 `""` → `uuid.Nil`
변환을 명시적으로 처리하는 패턴. 이 패턴을 그대로 재사용한다).

**경합 검증 없음(last-write-wins).** 대표님 확정 사항 — release/
reassign처럼 "기대하는 현재 소유자"를 body에 실어 서버가 검증하고
다르면 409로 거부하는 방식은 채택하지 않는다. `contact_cases.
contact_id` PUT과 동일하게, 마지막 요청이 그대로 반영된다. 동시에
같은 주소를 서로 다른 Contact로 옮기려는 경쟁 상태가 생겨도 조용히
나중 요청이 이긴다 — 이 리스크를 감수하기로 결정했다.

**응답:** 200 + 갱신된 `ContactManagerAddress`.
- 400: 잘못된 요청.
- 404: 주소 없음 또는 customer 불일치.
- 409는 없음(경합 검증을 하지 않으므로 이 케이스 자체가 발생하지
  않는다).

### 4.3 기존 PUT 핸들러 확장 지점

`PUT /contact_addresses/{id}`는 이미 `target`/`name`/`detail`/
`is_primary` 갱신을 지원한다(`bin-openapi-manager/openapi/paths/
contact_addresses/id.yaml:33-86`). `contact_id` 필드를 이 스키마에
추가하고, `bin-contact-manager`의 `contacthandler.UpdateAddress`
(또는 그 대응 handler — 정확한 함수명은 구현 착수 시 재확인)가
`contact_id` 필드가 요청에 포함되어 있으면 단순 `UPDATE
contact_addresses SET contact_id = ?`를 실행하도록 확장한다.
검증/조건절 없음 — 다른 필드(target/name/detail/is_primary) 갱신과
동일한 "그냥 반영" 방식이다.

### 4.4 이력(ownership-period) 서브시스템과의 비대칭 — 명시적 트레이드오프

이 PUT으로 바뀐 소유권은 `contact_address_ownership_periods` 테이블에
반영되지 않는다(§4.1). 반면 같은 리소스의 `claim`(POST, 유지됨)은
여전히 이 테이블을 갱신한다. 즉 **같은 `contact_addresses.contact_id`
컬럼을 두 가지 다른 API 경로로 바꿀 수 있는데, 한쪽(claim)은 이력에
남고 다른 쪽(PUT)은 안 남는 비대칭이 생긴다.** 이건 구현자가 놓치기
쉬운 지점이라 여기 명시적으로 남긴다 — 대표님이 복잡도 절감을 위해
의도적으로 받아들인 트레이드오프이며, 향후 이 비대칭이 실제 문제가
되면(예: 소유권 이력 조회가 이 PUT으로 바뀐 소유권을 놓침) 별도
티켓으로 다시 열 수 있다.

### 4.5 Race condition — 정책 없음(명시적 수용)

동시에 여러 상담사가 같은 주소를 서로 다른 Contact로 재할당해도
서버는 이를 구분하지 않는다. 나중 요청이 그대로 반영되고, 앞선
요청을 보낸 상담사는 자신의 변경이 덮어써졌다는 걸 알 방법이 없다
(§4.2). 이 리스크는 대표님이 명시적으로 받아들인 결정이다 —
`contact_cases.contact_id`가 오늘도 이미 이 정책으로 운영되고 있고,
실무상 문제가 되지 않았다는 것이 근거다.


## 6. 프론트엔드(square-admin) 반영 범위

**대상 파일:** `square-admin/src/views/contacts/contact_cases_detail.js`
의 `CaseContactAttributionPanel` 컴포넌트, 트리거 지점은 기존
`handleAttach` 함수(339-353행).

### 6.1 `handleAttach` 흐름 재설계 (v6)

현재 `handleAttach`는 `PUT contact_cases/{caseId}` 한 번으로 끝난다.
새 흐름:

```
handleAttach:
  1. PUT contact_cases/{caseId} { contact_id: selectedContact.id }  (주 동작, 기존과 동일)
     실패 시 → 기존과 동일하게 에러 표시, 이후 단계 진행 안 함.
  2. 성공 시 → 이 Case의 Peer(type, target)로
     GET contact_addresses?type=<Peer.Type>&target=<Peer.Target>&customer_id=... 조회
     (§5의 신규 target 필터 사용 — v6에서도 이 조회는 그대로 필요)
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
        - "항상 예" → 즉시 **PUT /contact_addresses/{id}
          {contact_id: selectedContact.id}** 실행(v6, §4.2 — 단순
          필드 교체, 경합 검증 없음).
        - "항상 아니오" → 아무 것도 안 함.
        - 저장된 결정 없으면 모달 B 표시(기존 소유 Contact의
          display_name 조회 필요 — GET contacts/{contact_id}) →
          확인 시 위 PUT 호출 (+ 체크박스 켜져 있으면 저장).
  4. 2~3단계(주소 연동)의 성공/실패와 무관하게, 1단계가 이미
     성공했으면 onAttributionChange()는 호출한다(주 동작은 이미
     완료됨). 주소 연동 실패는 별도의 (덜 위협적인) 에러/경고로
     표시한다 — 전체 handleAttach 실패로 취급하지 않는다.
```

**v5(SUPERSEDED)와의 차이:** 시나리오 c에서 `POST .../reassign`
호출이 `PUT /contact_addresses/{id} {contact_id}` 호출로 바뀐 것
외에는 흐름이 동일하다. `from_contact_id`를 body에 실을 필요가
없다(v6은 경합 검증을 하지 않으므로, §4.2 참조).

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

## 7. 동시성 / 에러 처리 요약 (v6)

- 백엔드: **경합 검증 없음(§4.5).** 동시에 여러 상담사가 같은 주소를
  서로 다른 Contact로 재할당해도 나중 요청이 조용히 이긴다.
  409/재시도 로직 자체가 존재하지 않는다 — v5의 ownership-period
  안전망, `ErrConflict`/`ErrStaleTarget` 재시도 정책은 모두 이번
  스코프에서 제거됐다(§4 SUPERSEDED 고지 참조).
- 프론트: 주 동작(Case-Contact 연결)과 부가 동작(주소 연동) 실패를
  분리해서 표시(§2, §6.1-4). 부가 동작 실패는 주 동작 롤백을
  유발하지 않는다.
- 동일 Case에 대한 동시 `UpdateContact` 호출 경쟁은 이번 설계가 새로
  만드는 문제가 아니다(기존 gap, 별도 티켓).

## 8. 테스트 계획 (구현 단계에서 상세화, v6)

**백엔드:**
- `PUT /contact_addresses/{id}` `contact_id` 필드: unresolved →
  특정 Contact로 배정 성공 / 이미 다른 Contact 소유 상태에서
  다른 Contact로 재배정(그냥 덮어써짐, 에러 없음) / 빈 문자열로
  해제 시 unresolved로 되돌아감 / 존재하지 않는 주소 404.
- `AddressList` `target` 필터: 단위 테스트 + OpenAPI 스펙 검증(§5).
- listenhandler: 기존 `PUT /contact_addresses/{id}` 핸들러에
  `contact_id` 필드 처리 추가 테스트(400/404 라우팅, 기존
  target/name/detail/is_primary 갱신 테스트와 동일 패턴 재사용).
- **비대칭 검증(§4.4):** 이 PUT으로 `contact_id`를 바꾼 뒤
  `contact_address_ownership_periods` 테이블에 해당 변경이
  반영되지 않는다는 것을 테스트로 명시적으로 확인(의도된 동작임을
  회귀 테스트로 고정 — 나중에 누군가 "왜 이력이 안 남지"라고
  실수로 "고치는" 것을 막기 위함).

**프론트:**
- `handleAttach`의 3가지 분기(시나리오 1/2/3) 각각에 대해 올바른
  API 호출이 발생하는지(특히 시나리오 3에서 PUT body가 v6대로
  `{contact_id}`만 담고 `from_contact_id`가 없는지).
- localStorage "기억하기" 저장 후 재진입 시 모달 스킵 및 자동 적용
  검증(시나리오 1/3 키가 서로 간섭하지 않는지 포함).
- 주소 연동 API 실패 시에도 Case-Contact 연결(주 동작)의 UI 상태가
  성공으로 유지되는지.

## 9. Out of Scope (YAGNI, v6)

- localStorage 결정 리셋 UI(§3) — 향후 설정 화면에 추가 가능.
- 고객사 단위 정책 플래그(예: 이 확인 흐름 자체를 끄는 옵션).
- 경합 검증/409 처리(§4.5) — v5에서 설계했던 것을 대표님 지시로
  명시적으로 제거. 향후 실무상 문제가 되면 재검토.
- `PUT`으로 바뀐 소유권과 `contact_address_ownership_periods`
  이력 테이블의 동기화(§4.4) — 의도적 비대칭으로 수용, 향후 필요
  시 별도 티켓.

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

## 11. v6 상태 및 다음 단계

v6(현재 §2~§9 본문)은 대표님과의 대화에서 직접 확정된 방향이며,
**아직 design-first-with-review-loops의 독립 리뷰 루프를 거치지
않았다.** v5와 달리 복잡도가 크게 낮아졌지만(신규 엔드포인트 없음,
경합 검증 없음, ownership-period 연동 없음), §4.4에 명시한
"이력 테이블과의 의도적 비대칭" 같은 트레이드오프는 실제 구현
착수 전에 최소 1~2라운드의 독립 검토를 받는 것을 권장한다 — 특히
"이 비대칭이 실무에서 정말 무해한지"는 코드만 봐서는 판단하기
어렵고 도메인 지식(Contact 이력 조회가 실제로 이 경로를 타는지)이
필요하다.

**현재 상태: DRAFT (리뷰 미완료).** 대표님이 "리뷰 루프 돌려"를
다시 지시하시면 v6 전용으로 새 라운드를 시작한다.


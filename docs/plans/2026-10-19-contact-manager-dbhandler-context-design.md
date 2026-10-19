# bin-contact-manager dbhandler: pass context to DB calls, check rows.Err (VOIP-1615)

## 1. 티켓 검토 (origin/main 45aac3bce 기준)
- 유효. 컨텍스트 없는 Exec/Query 33건(Exec 14, Query 19): address.go 5, address_ownership_write.go 1, casenote.go 3, contact.go 10, kase.go 11, tag_assignment.go 3. 이 중 7건은 비공개 인터페이스 `sqlExecutor`(kase.go:29, ctx 없는 Exec/Query 시그니처)를 통해 호출되어 noctx 가 잡지 못한다: kase.go 83, 136, 229, 280, 345, 374와 address_ownership_write.go:50.
- ctx 를 받고도 버리는 함수: AddressList, AddressListByContactID, AddressListAllByContactID(`_ context.Context`). ctx 매개변수 자체가 없는 함수: addressCustomerID(address.go:582).
- rows.Err 없는 단건 분기 `!rows.Next()` 13곳. 전환 대상 8곳(address.go 291/595, address_ownership_write.go:56, contact.go 125/336/377/419, kase.go:142)과 이미 ctx 를 쓰는 tx 경로 5곳(address_ownership.go:333, address_ownership_write.go 365/412/445, kase.go:193). 목록 루프 중 rows.Err 없는 곳은 tag_assignment.go:72 한 곳.
- 열린 PR 중 이 서비스를 바꾸는 것 없음. 공개 DBHandler 인터페이스는 이미 ctx 를 받아 mock 은 불변일 것으로 보고 go generate 로 확인한다.

## 2. 변경
1. 33곳을 ExecContext/QueryContext 로 전환. `sqlExecutor` 를 `ExecContext/QueryContext` 시그니처로 바꾸고(`*sql.DB`, `*sql.Tx` 모두 구현) 이를 받는 비공개 함수 7개(caseInsertExec, caseGetByIDExec, caseUpdateStatusClosedExec, caseUpdateContactIDExec, caseUpdateTagIDsExec, caseClearContactIDExec, addressTypeTargetContactByID)와 addressCustomerID 에 ctx 를 첫 인자로 추가. 호출부는 모두 이미 ctx 를 보유한다. `_ context.Context` 3곳은 `ctx` 로 바꿔 사용한다.
2. rows.Err 추가(바꾸는 줄은 errorlint 신규 줄 게이트에 맞춰 `%w`).
 - 단건 getter 13곳: `!rows.Next()` 분기 안에서 먼저 `rows.Err()` 를 확인해 있으면 에러를 반환하고, 없을 때만 기존 ErrNotFound 또는 (false,nil)/(nil,nil)을 반환한다.
 - tx 경로 5곳(Query 에서 isMySQLDeadlock 을 ErrDeadlock 으로 매핑하는 함수: ownershipAgreementHoldsTx, staleRowByTargetTx, contactTombstoneTx, latestOwnershipPeriodValidToTx, CaseGetOpenByPeer): rows.Err 가 deadlock 이면 감싸지 않은 센티널 `ErrDeadlock` 을 반환한다. 근거: 호출부 비교가 `==` 9곳과 `errors.Is` 1곳(casehandler/create.go:63)(address.go 92, 100, 193, 503, 512, 632; casehandler/insert.go 67, 77; address_ownership_write.go 718)이라 감싸면 재시도 루프를 놓친다. 같은 함수의 Query 에러 처리와 일관.
 - TagAssignmentListByContactID 루프 뒤 rows.Err 추가.
3. 에러 무시와 캐시(결정과 근거).
 - contact.go 6곳(`res.Addresses, _ =`, `res.TagIDs, _ =`; contactUpdateToCache, ContactGet, ContactList): 에러를 전파한다. 전환 뒤에는 취소와 DB 오류가 실제로 발생하므로 주소와 태그가 빈 연락처가 성공으로 반환되거나 캐시에 쓰이는 부분 객체 캐시를 막는다. ContactGet 은 캐시를 쓰지 않고 에러를 반환한다.
 - 커밋 뒤 캐시 퇴거 2곳(`h.cache.ContactDelete`, contact.go:300, 495)만 `context.WithoutCancel(ctx)` 로 분리한다. 근거: 삭제 커밋 뒤 퇴거가 취소로 실패하면 tombstone 이 TTL 동안 서빙된다. 이 변경이 막는 것은 커밋 직후부터 퇴거 사이에 취소가 도착하는 좁은 창이며(취소된 ctx 는 Exec 에서 먼저 실패) database.md §7.2a 를 따르는 위생 조치다. 순수 캐시 호출 한 줄이라 DB 읽기에 영향 없음. contactUpdateToCache 안의 tombstone 퇴거(contact.go:71)와 address.go 등의 호출 5곳은 같은 함수가 DB 읽기를 함께 수행해 한 줄 분리로 끝나지 않으므로 이번에 보호하지 않고 위 갱신 경로 보고에 함께 적는다. contactUpdateToCache 갱신 경로는 분리하지 않는다(DB 읽기까지 취소 불가가 되고 실측 신호가 없음). 갱신/퇴거의 `_ =` 무시 자체는 유지(캐시는 best effort). 갱신 실패 시 TTL 동안 낡은 연락처가 서빙될 수 있는 점(전환 전에는 갱신 경로의 DB 읽기가 취소를 보지 않았으므로 이번 변경이 새로 도입하는 잠재 회귀이며 진입 ctx 가 Background 라 현재 발현하지 않음)은 범위 밖으로 보고하며, 커밋 뒤 읽기에 WithoutCancel 을 권하는 docs/conventions/database.md §7.2a 로부터의 이탈임을 PR 본문에 명시한다.
 - address_ownership.go:213 OwnershipPeriodsLockAndResolveTx 는 rows.Err 의 deadlock 을 매핑하지 않는 기존 비대칭이며 이번에 바꾸지 않는다(기존 갭이며 PR 본문에 기존 갭으로 기록한다. addressTypeTargetContactByID 는 tx 안에서도 호출되지만 Query 단계에 deadlock 매핑이 없으므로 rows.Err 도 일반 `%w` 래핑으로 같은 동작을 따른다. 이번 티켓은 신규로 추가하는 rows.Err 의 매핑만 다룬다). CaseGetByIDTx 등 Query 에서도 deadlock 매핑이 없던 함수는 매핑 대상이 아니다.
4. 호출부(handler 계층)와 공개 인터페이스, 다른 서비스, 마이그레이션은 변경하지 않는다.

## 3. 테스트(코딩 단계에서 RED 먼저, 이 단계에서는 코드 없음)
기존 sqlmock(address_deadlock_budget_test.go 등)과 dbTest(sqlite)를 재사용한다. 새 파일 `ctx_rows_test.go`:
- RowError 주입(sqlmock `RowError`): 단건 getter 8곳(AddressGet address.go:291, addressCustomerID :595, addressTypeTargetContactByID, contactGetFromDB contact.go:125, AddressLookupContactIDByTypeTarget :336, ContactLookupByPhone :377, ContactLookupByEmail :419, caseGetByIDExec)과 tx 경로 5곳이 ErrNotFound 또는 nil 결과가 아니라 에러를 반환함을 `errors.Is(err, ErrNotFound)==false` 로 단언. tx 경로 5곳은 일반(비 deadlock) RowError 에 대해 `err != nil && err != ErrDeadlock` 를 단언하고(결과값 (false,nil)/(nil,nil) 을 반환하는 함수는 ErrNotFound 단언만으로는 변이가 통과하므로 `err != nil` 이 필수), deadlock RowError 에 대해 `err == ErrDeadlock` 도 단언. TagAssignmentListByContactID 루프 rows.Err.
- 취소된 ctx: Exec 대표(ContactCreate, CaseInsert via sqlExecutor 경로, casenote 추가)와 Query 대표(ContactGet DB 경로, TagAssignmentListByContactID, AddressList, AddressListByContactID, AddressListAllByContactID)가 중단됨: 반환 래핑 줄이 `%v` 로 남으므로(바꾸는 줄이 아님) `errors.Is(context.Canceled)` 는 쓰지 않고 `err != nil` 과 메시지의 `context canceled` 포함을 단언한다. 비 Context 호출로 복원하면 호출이 성공해 이 단언이 실패한다. 취소 ctx 는 database/sql 이 연결 획득 전에 검사하므로 결정적이다.
- contact.go 에러 전파 6곳 1:1: ContactGet, ContactList, contactUpdateToCache(ContactUpdate 경유)에서 주소 조회 실패와 태그 조회 실패를 각각 주입(sqlmock 으로 해당 테이블 쿼리만 에러)해 에러 전파와 캐시 mock 에 Set 이 호출되지 않음을 단언. contactUpdateToCache 는 에러 시 Set 을 건너뛰므로 갱신 직후라면 기존 캐시 항목이 TTL 동안 남을 수 있다(기존 contactGetFromDB 실패와 같은 성격이며 PR 본문에 명시, 별도 퇴거는 추가하지 않는다).
- 퇴거 2곳(ContactDelete, ContactDeleteByCustomerID 루프): 호출자가 `context.WithCancel` 로 만든 (취소되지 않은) ctx 를 넘기고, cache mock 이 받은 인자 ctx 의 `Done() == nil` 을 단언한다(WithoutCancel 이 만든 ctx 는 Done 이 nil, 그대로 전달하면 non-nil 이라 제거 변이가 결정적으로 검출됨. 입력 ctx 를 미리 취소하면 Exec 가 먼저 실패해 퇴거 줄에 도달하지 못하므로 쓰지 않는다).
- 변이 시험: contact.go 에러 전파 6곳 각각 `_` 복원, rows.Err 확인 제거(단건 8곳, tx 5곳, 목록 1곳 = 14곳 각각 RowError 시험 1:1), 센티널 매핑 제거(`==` 단언), WithoutCancel 제거, 대표 Context 호출 복원 시 해당 테스트 실패. 33곳 전부의 개별 복원 검출은 취소 시험이 대표 경로만 덮으므로 주장하지 않고, 나머지는 sqlExecutor 시그니처 변경(컴파일러 강제)과 noctx lint(26곳)가 보증한다.
기존 sqlmock 기대 정규식이 ExecContext 에도 매칭되는지 baseline 으로 확인하고 필요한 최소 수정만 한다.

## 4. 외부 관측 변화(PR 본문에 명시)
진입 ctx 는 Background(listenhandler/main.go:184, subscribehandler/main.go:133, contact-control)이고 비테스트 코드에 WithTimeout, WithCancel, WithDeadline 이 없어 정상 동작은 불변이다. 변화는 DB 순회 오류와 취소 상황에서만: (i) 순회 오류가 거짓 404(CONTACT_NOT_FOUND 등)로 보이던 것이(ContactList 는 한 건의 주소/태그 조회가 실패하면 목록 전체가 에러가 되고, ContactGet 은 삭제 이벤트 payload 용 tombstone 읽기에도 쓰이므로 그 경로도 같은 이유로 실패할 수 있다. PR 본문 운영 영향 항목에 적는다) 타입 없는 에러(listenhandler default 400, errorResponse 라우트는 500, 구현 단계에서 근거 줄을 PR 본문에 붙인다)로 바뀐다. (ii) tx 경로 4곳과 kase.go:193 에서 순회 오류가 정상 결과((false,nil)/(nil,nil))로 넘어가던 것이 에러로 바뀐다. (iii) 취소와 타임아웃이 도입되면 쿼리가 중단된다(트랜잭션 없는 다단계 쓰기는 중간 취소 시 비원자적이며 현재도 같은 성격). 새 에러 매핑은 만들지 않고 기존 untyped DB 에러 경로를 따른다.

## 5. 범위 밖(보고서에 한 줄씩)
handler 계층의 "변이 성공 뒤 같은 ctx 로 읽고 실패 시 에러 반환, 이벤트 발행 생략" 13곳(casehandler/lifecycle.go:72→78, contact_update.go:64, assign.go:33→37; contacthandler/contact.go 126, 188, 217, 345→357, 409→413, 442→447, 467, 496→514, 535→541, 560→566)은 handler 코드에서 구현 단계에 다시 도출해 기록하고 수정하지 않는다. 후속 티켓과 카드는 만들지 않는다. address.go 의 contactUpdateToCache 호출 5곳과 tag_assignment.go 2곳의 `_ =` 는 유지.

## 6. 검증 기준
임시 noctx+rowserrcheck 설정(워크트리 밖 yml, linters default none, vendor 제외, tests false, `--max-issues-per-linter 0 --max-same-issues 0`)으로 bin-contact-manager 위반 0, 표준 lint 0, `.golangci-errorlint.yml --new-from-merge-base=origin/main` 통과, 전체 단위 테스트 통과, go mod tidy, vendor, generate, test, lint 후 go.mod/go.sum/mock 변경 없음. build 실패나 mock 변경이 생기면 block 한다. go test 는 `-race` 도 1회 실행한다.

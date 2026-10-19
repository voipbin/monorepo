# VOIP-1622 email-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. noctx 5건이 bin-email-manager/pkg/dbhandler/emails.go 에만 있음(EmailCreate, emailGetFromDB, EmailList, EmailUpdate, EmailDelete). 모두 ctx 보유 함수 안. rows.Err() 는 0건. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- h.db.Exec -> ExecContext(ctx, ...) 3곳, h.db.Query -> QueryContext(ctx, ...) 2곳.
- emailGetFromDB: `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 `%w` 로 감싸 반환하고, 없을 때만 ErrNotFound. EmailList: 루프 뒤 rows.Err() 확인(`%w`). 기존 `%v` 줄은 변경하지 않음.
- 쓰기 3곳 뒤 `_ = emailUpdateToCache` 는 에러를 무시하므로 dbhandler 안에서 커밋된 쓰기가 실패로 보고되는 경로가 없음. WithoutCancel 분리는 넣지 않음.

## 취소 부작용 판단
- 진입 ctx 는 Background(listenhandler/main.go:138,152, emailhandler/email.go:65, email-control). DB 경로로 가는 취소 파생 없음(engine_mailgun.go:87 의 WithTimeout 은 외부 HTTP). 즉시 동작 변화는 거의 없고 효과는 noctx 일관성과 향후 취소 도입 대비.
- ErrNotFound 호출부(emailhandler Get 의 EMAIL_NOT_FOUND, listenhandler 404)에 create-if-missing 이나 멱등 검사 경로 없음. 순회 오류가 404 로 오분류되던 것이 listenhandler default 분기의 400 으로 바뀜(500 아님, 기존 레거시 동작).
- EmailList 는 중간 순회 오류 시 부분 목록 대신 에러를 반환. limit, 정렬 불변. 결과는 캐시에 저장되지 않음.
- 쓰기 Exec 도중 취소 시 서버 반영이 모호해질 수 있으나 현재 취소가 없어 수용.

## 테스트 수단
go.mod 에 go-sqlmock v1.5.2 추가(테스트 전용, billing 등과 같은 버전). tidy/vendor 문제 시 database/sql/driver 가짜 드라이버로 대체.

## 범위 밖(PR 보고서 한 줄씩)
- 쓰기 3곳 뒤 캐시 갱신 실패(TTL 24h). cache.EmailSet 은 원래 ctx 를 썼고, 이번 전환으로 재조회(emailGetFromDB)도 ctx 영향을 받게 되므로 취소가 도입되면 재조회 단계가 새로 영향을 받음(현재 취소 없음).
- emailhandler 의 create/Delete/UpdateStatus 커밋 뒤 재조회와 호출부 에러 처리(send.go, hook.go 포함), 인프라 오류가 400 으로 보이는 기존 동작, 기존 `%v` 에러 형식.

## 검증(verify)
- noctx+rowserrcheck 임시 설정(워크트리 밖 yml, default none, vendor 제외) bin-email-manager 0건. 표준 설정 0 issues. .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트(수정 전 RED): sqlmock RowError 로 emailGetFromDB(EmailGet 경로)와 EmailList 가 errors.Is(주입에러) 이고 !errors.Is(ErrNotFound); 취소된 ctx 에서 조회 2종과 쓰기 3종이 중단(sqlmock 의 사전 취소 ctx 동작을 실측해 구성, 필요 시 WillDelayFor).
- 테스트 구성 주의: 취소 테스트는 사전 취소된 ctx 에서 database/sql 이 드라이버 도달 전에 context.Canceled 를 반환하므로 `errors.Is(err, context.Canceled)` 를 단언하고 `err != nil` 만으로 판정하지 않음(비-Context 호출은 mock 의 예상 밖 호출 에러로도 err 가 나므로 변이를 놓침). 쿼리 매칭은 squirrel SQL 의 `?` 와 괄호 때문에 regexp.QuoteMeta 와 sqlmock.AnyArg() 사용. EmailList 의 중간 순회 오류는 첫 행을 JSON 컬럼(source, destinations, attachments) 포함 완전하게 만들고 RowError(1, err) 로 주입(RowError(0) 은 빈 목록 경로). 단건 getter 는 RowError(0) 로 !rows.Next() 분기 도달.
- 변이: rows.Err 제거, EmailList 의 rows.Err 확인을 루프 앞으로 이동, rows.Err 무조건 반환(정상 빈 결과가 ErrNotFound 로 남는지 확인하는 sqlmock 빈 행 테스트 추가), Context 호출 복원 각각에서 해당 테스트 실패. 취소 테스트는 ExpectationsWereMet 를 호출하지 않음. email-control 의 Background 근거는 cmd/email-control/main.go:151,193,225.
- 기존 sqlite 테스트가 전환 뒤에도 통과하는지 확인. go mod tidy, vendor, generate, test, lint 5단계.

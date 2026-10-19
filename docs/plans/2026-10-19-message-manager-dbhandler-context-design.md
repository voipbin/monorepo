# VOIP-1621 message-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. noctx 5건(h.db.Exec 3, h.db.Query 2), 모두 ctx 를 받는 함수 안(message.go:61,117,173,219,262). rows.Err() 확인 0건. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음. 트랜잭션 없음.

## 변경
- bin-message-manager/pkg/dbhandler/message.go: h.db.Exec -> ExecContext(ctx, ...) 3곳, h.db.Query -> QueryContext(ctx, ...) 2곳.
- messageGetFromDB: `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 `%w` 로 반환하고, 없을 때만 ErrNotFound. MessageList: 루프 뒤 rows.Err() 확인(`%w`).

## 취소 부작용 판단
- 진입 ctx 는 Background(listenhandler/main.go:155, message-control)이고 비테스트 코드에 WithTimeout/WithCancel/WithDeadline 이 없어 현재 즉시 동작 변화는 거의 없음. 효과는 noctx/rowserrcheck 일관성과 향후 취소 도입 대비.
- 단건 getter 의 ErrNotFound 오판(messagehandler/message.go:115, listenhandler/main.go:107,215 의 404 매핑): 순회 중 실패가 404 대신 에러로 반환됨(비 typed 에러는 listenhandler 에서 400 으로 매핑되며 현재 Background ctx 라 발현하지 않음). 확인된 금전 중복 경로 없음(dbhandler 호출부는 조회와 쓰기뿐).
- 트랜잭션이 없어 커밋 뒤 읽기 분리(WithoutCancel)는 해당 없음.

## 범위 밖(보고서에 한 줄 기록)
- messagehandler/db.go:41,64,103 의 쓰기 뒤 MessageGet: 전환 후 취소 ctx 에서는 커밋된 쓰기가 에러로 보이고 웹훅 발행을 건너뜀(현재 Background 라 발현 안 함).
- send.go:94 goroutine 이 요청 ctx 로 dbUpdateTargets 호출.
- pkg/requestexternal/messagebird.go:50 http.NewRequest (noctx 1건). DB 가 아닌 SMS 발송 HTTP 경로이고 ctx 연결 시 발송 취소 동작이 바뀌므로 범위 밖(default 결정: 검증 범위는 pkg/dbhandler 비테스트 코드로 한정).
- messageGetFromRow 의 기존 `%v` 에러 줄(변경하지 않음).
- Exec 뒤 `_ = messageUpdateToCache` 의 무시와 캐시 TTL 24시간 노출(이 PR 이전부터 존재). 취소 시 캐시 세트 실패는 전환 전에도 존재하며 새로 생기는 것은 DB 재조회 실패뿐이다.
- 쓰기 Exec 가 취소로 실패하면 커밋 여부가 모호할 수 있음(드라이버 의존, 호출부는 에러로 받음).

## 검증(verify)
- 테스트 방식: go.mod 변경이 없도록 sqlmock 을 추가하지 않고 표준 라이브러리 database/sql/driver 로 만든(sql.OpenDB(driver.Connector) 사용) 테스트용 가짜 드라이버(ctx_rows_test.go)를 쓴다. 첫 Next 에서 에러를 내는 rows 를 돌려줘 iteration 에러를 주입한다(database/sql 은 io.EOF 가 아닌 Next 에러를 Rows.Err 로 전달). 취소 ctx 테스트는 기존 sqlite dbTest 를 쓴다(database/sql 이 취소된 ctx 면 드라이버 호출 전에 실패).
- 신규 테스트(수정 전 RED 확인 후 통과): messageGetFromDB 와 MessageList 에 iteration 에러 주입 시 errors.Is(주입 에러) 참, errors.Is(ErrNotFound) 거짓. 취소 ctx 테스트는 고유 UUID 와 명시적 cache mock 기대와 util mock 기대(TimeNow, TimeGetCurTime)(AnyTimes)를 써서 수정 전 성공한 Exec 가 남긴 행이 다른 테스트에 영향이 없게 하고, 실패 사유가 의도한 단언이 되게 한다. 취소된 ctx 에서 Query 경로(MessageGet 의 DB 조회, MessageList)와 Exec 경로(MessageCreate, MessageUpdate, MessageDelete)가 context canceled 로 실패.
- 변이 시험(scratch 복사본): rows.Err 확인 제거, 각 Context 호출을 원복할 때 해당 테스트가 실패.
- 기존 message_test.go 가 전환 뒤에도 통과(ErrNotFound 단언 유지).
- noctx 와 rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 bin-message-manager/pkg/dbhandler 비테스트 0건(서비스 전체 0건은 요구하지 않음), 표준 설정 0 issues(임시 설정은 .golangci-errorlint.yml 의 두 린터를 --new-from-merge-base 없이 pkg/dbhandler 전체에 실행. rowserrcheck 가 분기 안 Err 확인을 인정하지 않으면 확인 위치만 조정), .golangci-errorlint.yml --new-from-merge-base=origin/main 통과. 새 에러는 `%w`.
- go mod tidy, vendor, generate(mock 변경 없음), test, lint 5단계.

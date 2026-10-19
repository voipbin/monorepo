# VOIP-1629 conversation-manager dbhandler: pass request context to DB calls and check rows.Err

줄 번호는 모두 origin/main b5af852d1 기준.

## 티켓 검토
유효. bin-conversation-manager/pkg/dbhandler 의 비 Context DB 호출은 conversation.go:275 (ConversationUpdate 의 h.db.Exec) 1건이고 나머지 17건은 이미 ExecContext/QueryContext. 트랜잭션 없음. rows.Err() 확인이 없는 곳은 단건 getter 3곳(conversation.go:89, :179, media.go:87)과 루프 1곳(message.go MessageGetsByTransactionID, 234행 루프 뒤). account.go:86, message.go:86, AccountList, ConversationList, MessageList 는 이미 확인함. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- conversation.go:275 h.db.Exec(sqlStr, args...) -> h.db.ExecContext(ctx, sqlStr, args...).
- 단건 getter 3곳: `!Next()` 분기 안에서 rows.Err() 를 먼저 확인해 `%w` 로 감싸 반환하고, 없을 때만 ErrNotFound.
- MessageGetsByTransactionID: 루프 뒤 rows.Err() 확인, `%w` 로 반환.
- 새 에러 문장은 `%w`. 순회 에러를 errors.Is 로 단언하려면 기존 5줄(account.go:88 accountGetFromDB, message.go:88 messageGetFromDB, account.go:197 AccountList, conversation.go:246 ConversationList, message.go:197 MessageList)의 rows.Err 반환도 `%v` -> `%w` 여야 하므로 이 5줄은 verb 만 `%w` 로 바꿈(같은 결함을 남기지 않기 위함, errorlint 는 변경 줄이라 통과). 그 밖의 기존 `%v` 줄(예: conversationGetFromDB 의 Wrapf 중복 %v)은 건드리지 않음.

## 취소 부작용 판단
- 호출 경로: 진입 ctx 는 listenhandler/main.go:183, subscribehandler/main.go:152, cmd/conversation-control 모두 Background. 파생 ctx 는 accounthandler/db.go:75 의 WithTimeout(setupCtx) 한 곳뿐이고 setup 단계에서만 쓰이며 AccountCreate/Get 은 원 ctx 라 DB 호출에 닿지 않음. 즉시 동작 변화 거의 없음, 효과는 noctx 일관성, 향후 취소 도입 대비, 순회 에러 은폐 제거.
- 트랜잭션이 없고 Exec 뒤 `_ = h.xUpdateToCache(ctx, ...)` 는 에러 무시이므로 WithoutCancel 분리는 하지 않음(billing 과 달리 커밋 뒤 에러를 호출자에게 실패로 돌려주는 경로가 없음).
- 쓰기 Exec 도중 취소되면 서버에 반영됐을 수 있는 모호성이 새로 생기나 진입 ctx 가 Background 라 수용.
- ErrNotFound 오판 제거 효과: conversationhandler/db.go:30, accounthandler/db.go:109, messagehandler/db.go:132, listenhandler/main.go:112,320 은 ErrNotFound 를 404 로 매핑하므로 순회 에러는 404 대신 500 이 되는 것이 의도. email.go:60,:151 은 errGet != nil 이면 firstErr 에 기록하고 continue(그 수신자 건너뜀, 에러는 호출자에게 반환)라 부분/빈 목록이 에러 없이 오던 중복 발송 방지 검사가 안전해짐.

## 범위 밖(보고서에 한 줄 기록)
- conversationhandler/db.go:65 GetOrCreateBySelfAndPeer 는 ConversationGetBySelfAndPeer 가 err != nil 이면 종류 불문 새 conversation 생성. 순회 에러도 생성으로 이어짐(중복 대화 위험은 이미 존재, 호출부 변경은 범위 밖).
- Exec 뒤 또는 Get 후 캐시 채움의 `_ =` 에러 무시 13곳(그중 account.go:150, conversation.go:150, media.go:148, message.go:149 는 Get 의 캐시 채움)(account 61/150/230/269, conversation 63/150/279, media 61/148, message 61/149/289/327): 취소 시 TTL 까지 stale 가능.
- 호출부, 다른 서비스, 마이그레이션, goroutine 정리.

## 테스트 설계
- 순회 에러 주입: sqlmock 이 go.mod 와 vendor 에 없고 추가는 go.mod/go.sum 변경을 동반하므로, 표준 database/sql/driver 로 만든 최소 fake driver 를 ctx_rows_test.go 에 둔다(의존성 추가 없음). 전역 sql.Register 없이 driver.Connector 와 sql.OpenDB 로 테스트마다 독립 DB 를 만든다. 모든 컬럼 값을 NULL 로 주어도 ScanRow 가 통과하므로 컬럼 이름 목록(GetDBFields)의 개수만 맞추면 된다. Rows.Next 가 첫 호출(단건) 또는 첫 행 뒤(루프)에 주입 에러를 반환. 루프의 첫 행은 실제로 ScanRow 되므로 fake driver 의 Columns 와 값 타입은 모델별 db 태그(GetDBFields 결과)에 맞춰 구성하여 scan 에러가 아닌 순회 에러로 실패하게 함(거짓 RED/GREEN 방지). 대상은 단건 getter 5곳(conversationGetFromDB, ConversationGetBySelfAndPeer, mediaGetFromDB, accountGetFromDB, messageGetFromDB)과 루프 4곳(AccountList, ConversationList, MessageList, MessageGetsByTransactionID). 단언: errors.Is(err, 주입) 참, errors.Is(err, ErrNotFound) 거짓, 루프는 에러 반환(부분 결과가 성공으로 반환되지 않음).
- 취소 ctx: 취소된 ctx 로 ConversationUpdate(sqlite dbTest) 호출 시 context canceled 에러(database/sql 이 연결 획득 전에 ctx.Done 을 확인하므로 사전 취소 검증이며, 드라이버 전달 자체의 증명은 아님. 이 한계를 인정). 수정 전 코드에서 실패(RED), ExecContext 로 변환 후 통과.
- 변이 시험: rows.Err 확인 제거, ExecContext 를 Exec 로 되돌리면 각 테스트가 실패하는지 확인.
- 기존 sqlite 기반 테스트의 기대 변화 여부 확인 후 최소 수정.

## 검증
- noctx, rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 non-test 0건. 표준 설정 0 issues. .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- go mod tidy, vendor, generate, test, lint 5단계 후 go.mod/go.sum 변경 없음 확인.

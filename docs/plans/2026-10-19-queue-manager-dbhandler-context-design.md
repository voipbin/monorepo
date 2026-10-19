# VOIP-1620 queue-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. noctx 5건(queue.go:317, 356, 390, 423, 456 의 h.db.Exec), 모두 ctx 를 받는 함수 안. rows.Err() 확인 누락은 단건 getter 3곳 중 QueuecallGetByReferenceID(queuecall.go) 의 `!row.Next()` 분기 1곳뿐. 목록 5곳과 queueGetFromDB, queuecallGetFromDB 는 이미 확인함. 인터페이스와 mock 변경 없음.

## 변경
- queue.go 의 h.db.Exec 5곳을 h.db.ExecContext(ctx, ...) 로 전환. 나머지 줄은 그대로.
- QueuecallGetByReferenceID 의 `!row.Next()` 분기 안에서 row.Err() 를 먼저 확인하고 에러면 `%w` 로 감싸 반환, 없을 때만 ErrNotFound. 기존 `%v` 줄은 변경하지 않음(.golangci-errorlint.yml 은 변경된 줄만 검사).

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:179, subscribehandler/main.go:162, Reconcile(cmd/queue-manager/main.go:167), queue-control 모두 Background. bin-queue-manager 에 WithTimeout/WithCancel/WithDeadline 파생이 없어 즉시 동작 변화는 거의 없음. 효과는 noctx 일관성과 향후 취소 도입 대비.
- ErrNotFound 오판의 영향: 호출부 queuecallhandler/db.go:55 GetByReferenceID 하나(그 호출자: listenhandler v1_queuecalls.go:87 직접 1곳, event.go 3곳, kick.go:74 KickByReferenceID(listenhandler :325 경유), queue-control). RPC(v1_queuecalls.go:87 get, :325 kick)는 errorResponse 로 ErrNotFound 만 404, 그 외 500 이라 순회 에러 시에만 404 가 500 으로 바뀜(의도). bin-api-manager servicehandler/queuecall.go:50 은 에러를 그대로 반환. 이벤트 처리 3곳(event.go:24/50/80)은 에러를 구분하지 않고 return 하므로 동작 동일.
- Exec 성공 뒤 ctx 가 취소되면 `_ = queueUpdateToCache` 가 조용히 실패해 DB 와 캐시가 어긋날 수 있음(QueueUpdate, QueueDelete 에 이미 있는 노출과 같음). 진입 ctx 가 Background 라 수용.
- 쓰기 Exec 도중 취소되면 서버 반영 여부가 모호해질 수 있음. 진입 ctx 가 Background 라 수용.

## 테스트 기반 결정
sqlite 로는 rows.Err 주입이 불가능. go-sqlmock v1.5.2 를 go.mod 에 추가(billing-manager 와 같은 버전, 커밋 대상 변경은 go.mod 와 go.sum 2개. vendor 는 커밋하지 않음). 대안인 자작 가짜 driver 는 코드량이 늘고 저장소 관례(billing)와 달라 채택하지 않음.

## 범위 밖(보고서에 한 줄 기록)
- 호출부 변경, Exec 뒤 `_ = queueUpdateToCache` 의 에러 무시, queuehandler/db.go:293/299 의 RemoveWait/RemoveService 연속 호출(트랜잭션 없음, 취소 도입 시 부분 적용 위험), QueuecallGetByReferenceID 캐시 조회 에러 무시(queuecall.go:170-171), 마이그레이션, 다른 서비스.

## 검증(verify)
- 신규 테스트 ctx_rows_test.go(수정 전 RED 확인 후 통과): 단건 getter 3곳과 목록 5곳에 sqlmock RowError 주입 시 errors.Is(err, ErrNotFound) 가 false 이고 원인 보존은 신규 한 곳(QueuecallGetByReferenceID)만 단언. 취소된 ctx 에서 Exec 5곳: 각 호출 전에 ExpectExec 를 등록하고 취소된 ctx 로 호출해 에러 반환을 확인한 뒤, mock.ExpectationsWereMet() 가 에러(기대가 소비되지 않음)임을 단언. Exec 는 ctx 를 무시하고 기대를 소비해 RED, ExecContext 는 database/sql 이 드라이버 호출 전에 취소를 확인해 기대가 남아 GREEN. cache mock 은 AnyTimes 로 두어 RED 사유가 unexpected call 이 아니라 단언 실패가 되게 하고, 테스트 주석에 이 단언 방식의 이유를 남김. (기대를 등록하지 않으면 수정 전 코드도 sqlmock 에러로 통과하므로 구분되지 않음. 에러 문구는 %v 라 errors.Is(context.Canceled) 로 구분할 수 없고 기존 줄은 바꾸지 않음.)
- RowError 주입 방법: 행 1개 + RowError(0, err) 로 Next 는 false, Err 는 설정됨(행 0개면 Err 가 nil). QueuecallGetByReferenceID 는 cache mock 의 QueuecallGetByReferenceID 가 에러를 반환하도록 설정(queuecall.go:170-171).
- sqlmock 기본 매처는 정규식이므로 squirrel 생성 SQL 에는 QueryMatcherEqual 대신 접두사 `SELECT` 만 매칭하거나 regexp.QuoteMeta 사용.
- 변이 시험: ExecContext 를 Exec 로 복원, row.Err 확인 제거, 새 줄의 %w 를 %v 로 변경 각각에서 테스트 실패 확인.
- noctx, rowserrcheck 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 non-test 0건. 표준 설정 0 issues. .golangci-errorlint.yml --new-from-merge-base=origin/main 통과. 단위 테스트 전체 통과. go mod tidy, vendor, generate, test, lint 5단계.

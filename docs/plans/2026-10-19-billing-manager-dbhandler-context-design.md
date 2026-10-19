# VOIP-1614 billing-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. noctx 19건(h.db.Exec/Query), 모두 ctx 보유 함수 안. rows.Err() 확인은 failed_event.go:83 한 곳뿐(조회 사이트 8곳 중). 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- bin-billing-manager/pkg/dbhandler 의 h.db.Exec -> ExecContext(ctx, ...), h.db.Query -> QueryContext(ctx, ...) 19곳(account.go 6, billing.go 8, billing_paddle.go 1, failed_event.go 4).
- 조회 사이트 7곳에 rows.Err() 추가. 단건 getter 5곳(accountGetFromDB, billingGetFromDB, billingGetByReferenceIDFromDB, billingGetByReferenceTypeAndIDFromDB, BillingGetByIdempotencyKey)은 `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 에러를 반환하고, 없을 때만 ErrNotFound. 루프형 2곳(AccountList, BillingList)은 순회 뒤 확인. 새 에러는 `%w` 로 감쌈.
- BillingConsumeAndRecord: 커밋 뒤 구간(accountUpdateToCache, billingUpdateToCache, BillingGet)을 WithTimeout(WithoutCancel(ctx), 5s) 로 분리. 커밋된 과금이 취소 때문에 호출자에게 실패로 보이지 않도록 함(VOIP-1612 AIActivateInsight 와 같은 패턴).

## 취소 부작용 판단
- 진입 ctx 가 Background(listenhandler/main.go:196, subscribehandler/main.go:170, billing-control)라 현재 즉시 동작 변화는 거의 없음. 효과는 noctx 일관성과 향후 취소 도입 대비.
- 단건 getter 의 ErrNotFound 오판: billinghandler/billing.go:50 은 ErrNotFound 이면 신규 billing 생성, accounthandler/paddle.go:25~31 은 미처리로 보고 PaddleCreditTopUp 진행(중복 충전 위험). rows.Err() 확인으로 순회 중 취소는 에러 반환(404 나 신규 생성 대신 500/에러).
- 호출 경로 조사: billing-manager 안에 WithTimeout/WithDeadline/WithCancel 로 파생한 ctx 가 없다. go func 는 subscribehandler/main.go:136 과 listenhandler/main.go:180 뿐이고 둘 다 Background 기반. 따라서 deadline 경로가 없어 즉시 동작 변화가 거의 없다는 판단이 성립.
- 쓰기 Exec 도중 취소되면 서버에는 반영됐을 수 있는 모호성이 새로 생김. 호출자는 에러를 실패로 처리. 진입 ctx 가 Background 라 수용.

## 범위 밖(보고서에 한 줄 기록)
- billinghandler/billing.go:135~141 무료 통화 경로(BillingSetStatusEnd 뒤 BillingGet, PublishEvent)와 :180 호출부 에러 처리.
- Exec 뒤 `_ = *UpdateToCache` 의 캐시 갱신 실패. 사이트(줄 번호는 origin/main 5f58a4964 기준): 트랜잭션 Commit 뒤 account.go:305,406, account_paddle.go:118,194,276, billing.go:612(AccountTopUpTokens), 쓰기 직후 account.go:54,219,448,474, billing.go:59,319,345,509. BillingConsumeAndRecord 만 분리하는 이유는 BillingGet 에러가 호출자에게 실패로 반환되어 이미 커밋된 과금이 실패로 보이기 때문이고(호출부 뒤에 메트릭과 PublishEvent), 나머지는 `_ =` 로 에러를 무시해 캐시가 stale 로 남는 것(TTL 24시간, 이 PR 이전부터 있던 노출)에 그침. 진입 ctx 가 Background 라 취소가 없어 분리하지 않음.
- 호출부(billinghandler, accounthandler), 다른 서비스, 마이그레이션, goroutine 정리.

## 검증(verify)
- noctx 와 rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 non-test 0건. 표준 설정 0 issues. .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트(수정 전 RED 확인 후 통과): 단건 getter 5곳 sqlmock RowError 주입 시 ErrNotFound 가 아님을 errors.Is 로 단언, 루프형 2곳(AccountList, BillingList; failed_event 는 이미 rows.Err 보유)의 rows.Err 테스트, 취소된 ctx 에서 쿼리 중단(읽기/쓰기/목록 대표 경로), 커밋 뒤 구간이 취소된 ctx 에서도 성공 결과 반환(sqlmock ExpectCommit 뒤 첫 캐시 갱신의 cache mock AccountSet 의 Do 에서 ctx 를 cancel 해 결정적으로 주입. RED 기준은 'QueryContext 변환 후, 커밋 뒤 구간 분리 전' 코드: 이후 billing 재조회가 context canceled 로 실패해 에러를 반환(원본 h.db.Query 는 ctx 를 보지 않아 GREEN 이므로 구현 순서는 변환, 테스트 RED 확인, 분리 적용). 테스트는 cache mock 의 BillingGet 을 miss 로 두고 sqlmock 에 account, billing 조회 ExpectQuery 를 모두 등록).
- 새 rows.Err 에러는 `%w` 로 감싸고 기존 `%v` 줄은 변경하지 않음(.golangci-errorlint.yml 은 변경된 줄만 검사).
- 기존 sqlmock 기대(ExpectExec, ExpectQuery)가 Context 호출에도 매칭되는지 확인하고 최소 수정.
- go mod tidy, vendor, generate, test, lint 5단계.

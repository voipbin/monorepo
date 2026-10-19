# bin-call-manager dbhandler: pass context to transaction DB calls, check rows.Err (VOIP-1626)

## 1. 티켓 검토 (origin/main b5af852d1 기준)
- 유효. noctx 대상은 `pkg/dbhandler/call.go` 의 4곳뿐이다: `h.db.Begin()`(CallTXStart), `tx.Query`(CallTXStart), `tx.Exec`(CallTXAddChainedCallID), `tx.Exec`(CallTXRemoveChainedCallID). 나머지 dbhandler 호출은 이미 Context 계열이다.
- rows.Err 누락은 CallTXStart 의 `!row.Next()` 분기 한 곳이다(다른 단건 getter 10곳은 분기 안에서 이미 rows.Err 확인, 목록 루프 7곳은 루프 뒤 확인. git grep 로 재확인함). 순회 오류(락 대기 타임아웃, 데드락 등)가 ErrNotFound 로 오판되어 listenhandler 가 404 로 매핑한다. 수정 뒤에는 일반 에러가 되어 listenhandler/main.go:193 의 ErrNotFound 분기를 타지 않고 500 으로 응답된다.
- 호출부는 `callhandler/chained_call.go` 의 ChainedCallIDAdd/Remove 뿐이며 둘 다 ctx 를 이미 받는다.

## 2. 변경
1. dbhandler 인터페이스: `CallTXStart(ctx, id)`, `CallTXAddChainedCallID(ctx, tx, id, chainedCallID)`, `CallTXRemoveChainedCallID(ctx, tx, id, chainedCallID)`. `CallTXFinish(tx, commit)` 시그니처 불변.
2. 구현: `BeginTx(ctx, nil)`, `tx.QueryContext`, `tx.ExecContext`. 이 4개 호출의 실패 래핑(`could not get transaction`, `could not query`, Add/Remove 의 `could not execute`)을 `%v` 에서 `%w` 로 바꿔 취소 원인이 errors.Is 로 보존되게 한다(바꾸는 줄이므로 errorlint 신규 줄 게이트와도 일치, 문구는 그대로).
3. CallTXStart `!row.Next()` 분기: `row.Err()` 가 있으면 롤백 후 `fmt.Errorf("... %w", err)` 반환, 없으면 기존대로 롤백 후 ErrNotFound.
4. Add/Remove 의 커밋 전 캐시 갱신 `callUpdateToCache(context.Background(), id)` 를 전달받은 ctx 로 바꾼다(반환값은 기존처럼 무시하므로 취소 시 영향은 캐시 갱신 생략뿐).
5. CallTXFinish: Commit 오류만 `logrus` Error 로 남긴다(신규 import). Rollback 은 이미 끝난 트랜잭션(ErrTxDone)에서 정상적으로 실패하므로 기존대로 무시. 인터페이스와 호출부 에러 처리는 바꾸지 않는다.
6. 전파: chained_call.go 의 CallTXStart/Add/Remove 에 ctx 전달, `go generate` 로 mock 재생성(워크트리 안), chained_call_test.go 와 outgoing_call_test.go 의 기대값에 ctx 인자 추가.

## 3. 테스트 전략
sqlite 는 `FOR UPDATE` 를 못 돌리므로 go-sqlmock v1.5.2 를 bin-call-manager go.mod 에 추가한다(billing, contact 선례, 같은 버전). 새 파일 `pkg/dbhandler/call_tx_ctx_test.go`:
- CallTXStart RowError: ErrNotFound 가 아님, 원인이 errors.Is 로 보존됨, 트랜잭션 롤백 호출(ExpectRollback)을 확인.
- 취소된 ctx: BeginTx, Add, Remove 가 중단됨.
- CallTXFinish commit 오류 로그(logrus test hook).
취소 ctx 시험은 경합을 피한다: BeginTx 는 취소된 ctx 에서 드라이버에 닿기 전에 실패하므로 ExpectBegin 을 걸지 않고, Add/Remove 는 살아 있는 ctx 로 시작한 트랜잭션에 취소된 ctx 를 직접 넘기고 ExpectExec 를 반드시 걸되 ExpectationsWereMet 는 이 시험에서 호출하지 않고(올바른 구현은 드라이버에 닿기 전에 끝나 소비되지 않음) 변이 감지용으로만 쓴다(없으면 tx.Exec 로 되돌려도 sqlmock unexpected call 에러로 통과함) context.Canceled 를 errors.Is 로 단언하며 캐시 갱신(cache mock)이 호출되지 않음도 단언한다(awaitDone 의 비동기 Rollback 에 의존하지 않음). QueryContext 는 ExpectQuery.WillDelayFor(2s) 를 걸고 20ms 뒤 cancel 하는 시험으로 덮는다(Query 로 되돌리면 지연이 끝까지 지속되어 성공하므로 실패, 단언은 에러 non-nil 과 경과 시간이 지연(2s)의 절반(1s) 미만이며 sqlmock 자체 에러라 errors.Is(context.Canceled) 는 쓰지 않는다. ExpectRollback 순서에는 의존하지 않음). 변이 시험은 BeginTx, QueryContext, ExecContext(Add/Remove) 각각을 따로 되돌려 해당 시험이 실패함을 보인다. CallTXFinish 로그는 id 가 없어 필드가 빈약함을 보고서에 적는다.
RED 먼저(수정 전 코드에서 실패), 변이 시험: rows.Err 제거, 각 Context 호출을 비 Context 로 되돌림, 로그 제거 시 해당 테스트 실패 확인.

## 4. 동작, 호환
- 진입 ctx 근거: listenhandler/main.go:257 `ctx := context.Background()` 가 RPC 처리의 ctx 이고 v1_calls.go:427,463 과 outgoing_call.go:376 은 받은 ctx 를 그대로 전달한다. outgoing_call.go 의 상위 호출자 ctx 출처는 구현 단계에서 호출 경로 전체를 git grep 으로 추적해 PR 본문에 적는다(pkg 의 WithTimeout 은 channelhandler/bridgehandler getWithTimeout 뿐이며 이 경로와 무관).
- 진입 ctx 가 모두 Background 이므로 외부 관측 동작 변화는 rows.Err 경로의 404 에서 500 으로의 변경(순회 오류가 ErrNotFound 로 오판되지 않음)뿐이며 정상 경로는 불변(취소 도입 대비와 noctx 일관성). DB 스키마, API, RPC, 이벤트 변경 없음. DBHandler 는 서비스 내부 인터페이스이며 호출부는 chained_call.go 하나다.

## 5. 범위 밖(보고서 한 줄)
커밋 전 callUpdateToCache stale 가능성, 호출부의 커밋 실패 무시와 트랜잭션 밖 CallSetMasterCallID 불일치, 커밋 뒤 CallGet(ctx) 가 취소되면 커밋된 변경이 실패로 보일 수 있는 점(취소 도입 시 billing 처럼 분리 필요), BeginTx(ctx) 는 ctx 취소 시 트랜잭션이 자동 롤백되어 이후 CallTXFinish 의 Commit 이 ErrTxDone 등으로 실패할 수 있음(CallTXFinish 로그의 동기), 다른 호출부 계층 패턴.

## 6. 검증 기준
임시 noctx/rowserrcheck 설정은 워크트리 밖 yml(커밋하지 않음)을 쓴다. sqlmock 추가 사유는 sqlite 가 FOR UPDATE 를 못 돌리기 때문이며 billing, contact 와 같은 v1.5.2 이다(PR 본문에 기재). 반환 에러 중 에러 문구 "CallAddChainedCallID" 오기와 chained_call.go Remove 로그의 "add" 오기는 범위 밖 NIT 으로 보고한다.
noctx, rowserrcheck 임시 설정 위반 0, 표준 lint 0, errorlint 신규 줄 게이트 통과, 단위 테스트 전체 통과, go mod tidy, vendor, generate, test, lint 순으로 순차 실행하고 `-race` 로 취소 시험의 안정성도 확인한다. Add/Remove 취소 시험이 만든 트랜잭션은 시험 끝에 Rollback 으로 정리한다.

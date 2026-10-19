# VOIP-1627 flow-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
부분 유효 (줄 번호는 origin/main b5af852d1 기준). 컨텍스트 없는 DB 호출은 2건: activeflow.go:250 (ActiveflowUpdate, `h.db.Exec`), flows.go:256 (flowUpdate, `h.db.Exec`). 둘 다 ctx 를 이미 받는 함수이며 Context 변형은 8건이 이미 있다. rows.Err 누락 사이트는 0건: 단건 getter 2곳(activeflow.go:87, flows.go:145)은 `!row.Next()` 분기 안에서, 목록 2곳(activeflow.go:222, flows.go:220)은 루프 뒤에서 이미 확인한다. 트랜잭션은 없다. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- activeflow.go:250, flows.go:256 을 `h.db.ExecContext(ctx, ...)` 로 전환(각 한 줄). 기존 `%w`/`%v` 에러 형식은 건드리지 않는다.
- 소스의 rows.Err 확인 코드는 변경 없음. 회귀로 사라지지 않도록 테스트만 추가한다.
- bin-flow-manager/go.mod, go.sum 에 github.com/DATA-DOG/go-sqlmock v1.5.2 test 의존성 추가(billing, contact, timeline, tts 모듈과 같은 버전).

## 취소 부작용 판단
- 진입 ctx 는 비테스트 코드 전반에서 Background(listenhandler/main.go:182,196, listenhandler/v1_activeflows.go:260,438, subscribehandler/main.go:129, cmd/flow-control, activeflowhandler/start.go:39)이고 비테스트 코드에 WithTimeout/WithCancel/WithDeadline 이 없다. 현재 동작 변화는 없고 효과는 noctx 일관성과 향후 취소 도입 대비다.
- Exec 뒤 캐시 갱신 5곳(activeflow.go:62,254,293, flows.go:83,260)은 모두 `_ = xUpdateToCache(ctx, id)` 로 에러를 무시하므로 커밋된 쓰기가 실패로 보고되는 경로가 없어 WithoutCancel 분리는 불필요하다.
- 호출부: ActiveflowUpdate 2곳, FlowUpdate 3곳, FlowDelete 1곳(flowUpdate 경유). ErrNotFound 호출부에 create-if-missing, 멱등 삭제 분기는 없다.

## 테스트 설계
- 취소 ctx 음성 테스트(sqlite dbTest): ActiveflowUpdate, FlowUpdate 는 `errors.Is(err, context.Canceled)` 와 DB 직접 조회로 행 미갱신을 단언. FlowDelete 는 `%v` 래핑이라 `err != nil` 과 DB 직접 조회(tm_delete 미설정)만 단언. 사전 행 생성용 ctx 와 취소 ctx 분리, 캐시 gomock 에 기대 미설정, TimeNow mock 은 Exec 전에 호출되므로 기대 설정.
- RowError 테스트 4건(getter 2, 목록 2): dbTest 와 격리된 별도 sqlmock sql.DB, `errors.Is(err, ErrNotFound)` false 단언(원인 체인은 `%v` 라 원인 Is 단언은 하지 않음). getter 는 RowError(0), 목록은 한 행을 읽은 뒤 에러(RowError(1))로 주입해 부분 결과가 성공으로 반환되지 않음을 단언.
- 변이 시험: ExecContext 를 Exec 로 되돌리면 취소 테스트 실패, getter 와 목록의 rows.Err 확인을 각각 제거하면 해당 RowError 테스트 실패.
- go.mod 변경은 sqlmock 추가뿐이며 replace 변경 없음(docs/dependencies.md 동기화 대상 아님).

## 범위 밖 (PR 보고서에 한 줄씩)
- `_ = xUpdateToCache` 무시로 인한 캐시 stale (TTL 24시간, 이 PR 이전부터 있던 노출).
- 기존 `%v` 에러 형식과 FlowDelete 의 `%v` 래핑.
- bin-common-handler/pkg/databasehandler/main.go:22 `res.Ping()` 의 ctx 없음(공유 모듈).
- 향후 취소 도입 시 쓰기 도중 중단되면 MySQL 에서 커밋 여부가 불확실한 채 에러가 반환될 수 있음(현재 Background 라 도달 불가).
- 다른 서비스, 마이그레이션, 호출부 변경.

# VOIP-1617 timeline-manager analysisdbhandler: pass request context to DB calls and check rows.Err

줄 번호는 origin/main b5af852d1 기준입니다.

## 티켓 검토
부분 유효. 티켓 본문의 pkg/dbhandler 6건은 ClickHouse 네이티브 드라이버이며 이미 conn.Query(ctx, ...) 와 rows.Err() 를 가지므로 변경하지 않습니다(dbhandler/main.go:114 의 Ping(context.Background()) 도 범위 밖). 실제 대상은 pkg/analysisdbhandler(MySQL, database/sql)입니다. analysis.go:80, 126, 170 의 h.db.Query 3건이 ctx 를 쓰지 않고(noctx), 이 패키지에는 rows.Err() 가 0건입니다. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경이 없습니다.

## 변경
- analysis.go:80, 126, 170 의 h.db.Query(query, args...) 를 h.db.QueryContext(ctx, query, args...) 로 바꿉니다.
- private analysisGetFromDB(id) 에 ctx 매개변수를 추가합니다(호출처는 AnalysisGet :100 하나).
- rows.Err(): 단건 getter 2곳(analysisGetFromDB :86, AnalysisGetByActiveflowID :132)은 `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 에러를 `%w` 로 감싸 반환하고, 없을 때만 ErrNotFound 를 반환합니다. AnalysisList(:177 루프)는 루프 뒤에 확인합니다. 기존 `%v` 줄은 바꾸지 않습니다(.golangci-errorlint.yml 은 변경된 줄만 검사).

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:150 Background, run.go:45/69/84 Background 파생 WithTimeout 이라 즉시 동작 변화는 거의 없습니다. 효과는 noctx 일관성과 향후 취소 도입 대비입니다.
- 둔갑 제거 영향: start.go:58 은 ErrNotFound 면 신규 분석 경로(AnalysisCountProgressing, 생성, kickoff)로 갑니다. UNIQUE(activeflow_id)(migrations/mysql/000001_baseline.up.sql:17)가 이중 기동은 막지만 한도 판정 오염과 reanalyze 요청 무시 가능성이 있었습니다. get.go:19/31/52 는 ErrNotFound 를 404 로 매핑하므로 순회 에러가 404 대신 500 이 되는 것이 의도입니다.
- 커밋 뒤 읽기: 해당 없음(BeginTx 와 트랜잭션 0건, WithoutCancel 분리 없음).
- 쓰기 Exec 는 이미 ExecContext 입니다.

## 범위 밖(보고서에 한 줄 기록)
- pkg/dbhandler(ClickHouse), 호출부(analysishandler, listenhandler).
- analysishandler/start.go applyExistingPolicy 의 AnalysisReset(커밋) 뒤 AnalysisGet 재읽기 2곳(:115, :122)이 같은 ctx 라 재읽기 실패 시 이미 progressing 으로 바뀐 행이 남을 수 있음(핸들러 계층 패턴, 오류 문구의 기존 %v 포함).

## 검증(verify)
- noctx 와 rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 bin-timeline-manager 위반 0건. 단 rowserrcheck 는 이 패키지의 rows.Err 부재를 잡지 못하므로 rows.Err 는 테스트와 변이 시험으로 보장합니다. 저장소 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트(수정 전 RED 확인): sqlmock RowError(AddRow(...).RowError(0, err) 로 첫 Next() 가 false)로 단건 getter 2곳 모두 errors.Is(err, 주입 에러) true 와 errors.Is(err, ErrNotFound) false, AnalysisList rows.Err 테스트, 취소된 ctx 에서 쿼리가 중단되는 음성 테스트(3곳 모두 또는 대표 경로).
- 변이 시험: rows.Err 확인 제거, QueryContext 를 Query 로 되돌림 각각 테스트 실패.
- 기존 ExpectQuery 정규식은 QueryContext 에도 매칭되는지 확인하고 최소 수정합니다.
- go mod tidy, vendor, generate(diff 0, go.mod/go.sum/mock 변경 없음), test(GOMAXPROCS=2 -p 2), lint 5단계.

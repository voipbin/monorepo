# VOIP-1618 direct-manager dbhandler: 요청 context 전달과 rows.Err 확인

## 티켓 검토
유효. pkg/dbhandler/direct.go 의 h.db.Exec/Query 6곳(DirectCreate, DirectGet, DirectGetByHash, DirectGets, DirectUpdate, DirectDelete)이 모두 ctx 보유 함수 안에서 ctx 없이 호출됨. rows.Err() 확인 0건. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- h.db.Exec -> ExecContext(ctx, ...) 3곳, h.db.Query -> QueryContext(ctx, ...) 3곳.
- 조회 사이트 3곳에 rows.Err() 추가. 단건 getter 2곳(DirectGet, DirectGetByHash)은 `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 `%w` 로 감싸(원인 보존을 위해. 같은 파일의 기존 래핑 `%v` 줄은 변경하지 않음) 반환하고 없을 때만 ErrNotFound. DirectGets 는 루프 뒤에 확인.
- 트랜잭션이 없어 커밋 뒤 구간 분리(WithoutCancel)는 해당 없음.

## 취소 부작용 판단
- 진입 ctx 가 Background(listenhandler/main.go:150, subscribehandler/main.go:132, direct-control)이고 WithTimeout/WithDeadline/WithCancel 파생이 없어 즉시 동작 변화는 거의 없음. 효과는 noctx 일관성과 향후 취소 도입 대비.
- ErrNotFound 오판 해소: 순회 중 오류(취소 포함)가 ErrNotFound 로 보이면 RPC 의 404 로 바뀌는 경로는 Get/GetByHash 뿐(v1_directs.go 의 Get 과 GetByHash 는 return nil, err 로 processRequest 에 가서 ErrNotFound 면 404, 아니면 기본 분기 400). rows.Err() 확인 후에는 404 오판이 해소되고 400 이 됨(400 의 재시도 불가 해석은 범위 밖). Delete 는 RPC 에서 어떤 에러든 400 이라 변화 없고, Regenerate 와 Gets 는 항상 500 이라 변화 없음. 내부 경로 EventCustomerDeleted 는 Delete 실패를 continue 로 넘겨 ErrNotFound 오판이 '이미 없음' 처럼 로그에 보이던 것이 로그 수준에서 해소됨(동작은 동일)(카드 본문의 'Delete/Regenerate 선행 조회가 404 가 된다' 는 RPC 응답으로는 사실이 아니어서 정정). DirectGets 는 순회 중 취소 시 부분 목록이 에러 없이 반환되어 EventCustomerDeleted 의 삭제 대상 목록(dbList 999건)이 일부 누락되는데 rows.Err() 가 이를 막음.
- 진입 ctx 가 Background 여도 rows.Err() 확인은 순회 중 연결 끊김과 드라이버 오류에 대한 현재 유효한 수정(Get 404 오판, Gets 부분 목록 무에러).
- 쓰기 Exec 도중 취소되면 서버에 반영됐을 수 있는 모호성이 새로 생기나 Background 라 수용.

## 범위 밖(보고서에 한 줄씩)
- directhandler.Create: dbCreate 실패 시 3회 재시도(매번 새 id, hash)와 dbCreate 뒤 dbGet. 취소 시 이미 반영된 INSERT 뒤 재시도로 중복 direct 가능.
- directhandler.Regenerate: dbUpdate 뒤 dbGet, 같은 쓰기 모호성.
- 순회 중 실제 취소는 단위 테스트로 증명하지 않음(RowError 주입으로 검증). GetByHash 오판 시 캐시 미저장은 정상.
- Regenerate: dbUpdate 가 취소로 실패해도 새 hash 가 반영됐을 수 있고 그때 DirectDeleteByHash(구 hash) 가 건너뛰어져 구 캐시가 TTL 까지 유효할 수 있음.
- Regenerate: dbUpdate 가 취소 오류로 3회 연속 실패하면 원인을 버리고 재시도 소진 에러만 반환.
- cachehandler/main.go:51 Ping(context.Background()). errorResponse(else 500)와 processRequest 기본 분기(400)의 비일관. Delete 의 dbDelete 가 취소로 실패하면 DirectDeleteByHash 와 publishEvent 가 건너뛰어지는 점.
- 다른 서비스, 마이그레이션, 후속 티켓.

## 검증(verify)
- 구현 후 기대치(수정 전 기준선은 noctx 6건(rowserrcheck 는 이 패턴에서 0건) 로그를 먼저 남김): noctx, rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 non-test 0건. 표준 설정 0 issues. .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트 ctx_rows_test.go (공유 dbTest 와 격리, sqlmock): RowError 주입 단건 2곳 + 목록 1곳에서 ErrNotFound 아님과 원인 보존(errors.Is) 단언, DirectGets 는 일부 행 뒤 RowError 에서 부분 결과가 아닌 nil, err 반환 단언. 취소된 ctx 에서 Exec 계열(Create/Update/Delete)과 Query 계열(Get/GetByHash/Gets) 중단: 사이트별로 ExpectExec/ExpectQuery 를 등록하고 취소 ctx 에서 err 문자열에 context canceled 포함(Exec/Query 의 기존 래핑이 %v 라 이 구간은 errors.Is 대신 문자열. rows.Err 구간은 RowError 원인을 errors.Is 로 단언)과 ExpectationsWereMet 가 실패(database/sql 이 드라이버 호출 전에 차단해 기대가 소비되지 않음)함을 단언. 비Context 변이에서 가짜 RED 를 막으려고 각 사이트의 WillReturnRows/WillReturnResult 를 완전히 등록해 변이 시 성공 응답과 err nil 이 되게 하고, 단언은 err 문자열의 context canceled 로 고정. sqlmock 은 QueryMatcherEqual 또는 regexp.QuoteMeta 를 쓰고 id 인자는 .Bytes() 로 바인딩. DirectGets 의 RowError 는 2행 이상 중 인덱스 1 에 주입(scan 실패 분기는 기존 동작이라 범위 밖). 열 목록은 GetDBFields 로 생성하고 값은 ScanRow 가 받는 타입(id 류는 []byte UUID, tm_* 는 문자열 또는 time, nullable)으로 채우며, Create/Update 취소 테스트는 MockUtilHandler 의 TimeNow 기대를 등록하고 첫 행의 ScanRow 성공과 반환 오류가 RowError 원인임을 단언. 구현 후 테스트 이름별 실제 RED 출력을 보고서에 첨부.
- 변이(사이트별 개별 적용, 각각 해당 테스트 실패 확인): 단건 2곳의 rows.Err 제거, 목록의 rows.Err 제거, rows.Err 래핑 %w 를 %v 로 변경, 6곳 각각을 비Context 로 복원, ctx 자리에 context.Background() 대입, 단건 getter 의 rows.Err 확인과 ErrNotFound 반환 순서 뒤바꿈. 수정 전 기준 noctx 6건이 나오는 로그를 RED 증거로 먼저 남김.
- 정상 경로(행 있고 rows.Err nil)는 기존 sqlite 테스트가 Context 호출에도 통과하는지로 확인. go.mod/go.sum 은 go-sqlmock v1.5.2(billing-manager 선례) 직접 의존성 증분만이어야 하며 go mod tidy 후 diff 로 확인. Delete/Regenerate 의 errors.Wrap 뒤 Is(ErrNotFound) 유지는 기존 동작이라 변경 없음.
- go mod tidy, vendor, generate, test, lint 5단계.

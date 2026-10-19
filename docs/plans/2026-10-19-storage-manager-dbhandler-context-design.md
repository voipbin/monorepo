# VOIP-1623 storage-manager dbhandler: pass request context to DB calls

## 티켓 검토
유효. noctx 4건(h.db.Exec: accounts.go AccountUpdate, AccountIncreaseFileInfo, AccountDecreaseFileInfo, files.go FileUpdate), 모두 ctx 보유 함수 안. 조회 4곳은 rows.Err() 를 이미 확인하나 래핑이 %v 라 errors.Is 로 원인을 볼 수 없음. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- bin-storage-manager/pkg/dbhandler: h.db.Exec -> h.db.ExecContext(ctx, ...) 4곳.
- 기존 rows.Err 래핑 4곳(accountGetFromDB, AccountList, fileGetFromDB, FileList)의 %v 를 %w 로 변경(errors.Is 단언 가능, errorlint 변경 줄 규칙). 새 rows.Err 추가는 0건. AccountIncreaseFileInfo, AccountDecreaseFileInfo 의 Exec 에러 래핑(%v, 같은 if 블록)도 %w 로 바꿔 취소 테스트가 errors.Is(context.Canceled) 로 4함수를 일관되게 단언.
- 신규 pkg/dbhandler/ctx_rows_test.go: (1) database/sql/driver 스텁으로 순회 오류 주입, errors.Is(원인) true 이고 ErrNotFound false 단언(단건 2, 목록 2). (2) 취소된 ctx 에서 쓰기 4함수가 errors.Is(context.Canceled) 이고 DB 행이 변하지 않음을 단언(sqlite dbTest). 단건 테스트는 캐시를 거치지 않도록 비공개 accountGetFromDB, fileGetFromDB 를 직접 호출하고, 목록은 첫 Next 오류와 '정상 행 1개 뒤 오류' 두 경우를 주입. 이 취소 테스트는 ExecContext 사용을 증명하며(database/sql 이 연결 획득 전 ctx.Err 확인), 실행 중 취소 존중은 스파이크 기록으로만 뒷받침. 쓰기 함수는 인자 평가 시 util.TimeNow 를 호출하므로 mock 기대 필요. 단언은 에러 문자열이 아니라 errors.Is.
- go.mod 변경 없음(sqlmock 미도입). 스파이크로 sqlite 드라이버가 ExecContext 취소를 존중하고 교착이 없음을 확인함.

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:160, subscribehandler/main.go:135, cmd/storage-control 모두 context.Background(), 서비스 안에 파생 취소 ctx 0건. 즉시 동작 변화는 거의 없고 효과는 noctx 일관성과 향후 취소 도입 대비.
- 네 Exec 뒤 `_ = h.xUpdateToCache(ctx, id)` 는 에러를 무시하므로 커밋된 쓰기를 실패로 반환하는 경로가 없어 WithoutCancel 분리는 하지 않음. 쓰기 도중 취소 시 서버 반영 여부가 모호해지나 호출자는 실패로 처리하며 수용.
- 서버 커밋 직후 클라이언트가 취소되면 ExecContext 는 에러를 반환하나 쓰기는 반영되고 캐시 갱신은 실행되지 않음(stale 최대 24시간). IncreaseFileInfo/DecreaseFileInfo 는 비멱등이라 호출자 재시도 시 이중 반영 가능. 현재는 취소 ctx 가 없어 잠복 위험이며 범위 밖으로 기록.
- 단건 getter 의 ErrNotFound 오판: 호출부(accounthandler/account.go:61, filehandler/file.go:156, listenhandler/main.go:112,245)는 404 매핑뿐이고 기존 rows.Err 확인이 !Next 분기 안이라 이미 방지됨.
- 변경 없음 확인: 단건 getter 는 Next 성공 뒤 rows.Err 를 확인하지 않음(rowserrcheck 허용 범위). download.go:44 의 FileUpdate 호출은 에러를 반환.
- 취소 테스트는 dbTest 공유 행을 피해 테스트별 고유 UUID 행을 쓰고, 취소 경로에서 cache mock 호출 0회를 gomock 으로 단언. driver 스텁 컬럼은 GetDBFields 와 맞추고 errors.Is(원인) 로 의도한 경로임을 구분.
- %v -> %w 변경은 errors.Is 단언 일관성을 위한 의도적 선택.

## 범위 밖(보고서 한 줄)
- 캐시 갱신 실패 시 stale 캐시(TTL 24시간, cachehandler/handler.go:37).
- accounthandler.Create(account.go:25~29)가 getByCustomerID 의 모든 에러를 '계정 없음'으로 보고 생성 진행.
- accounthandler IncreaseFileInfo(:142)/DecreaseFileInfo(:168)의 Exec 성공 뒤 AccountGet 실패 반환, filehandler/file.go:143,221 은 로그만 하고 무시.

## 검증(verify)
- noctx+rowserrcheck 임시 설정(워크트리 밖 yml, default none, vendor 제외) non-test 0건, 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트 -race -count=20. 취소 테스트는 수정 전 코드에서 RED, rows.Err 보호 테스트는 rows.Err 확인 제거 변이로 실패, ExecContext 되돌림 변이로 취소 테스트 실패 확인.
- go mod tidy, vendor, generate, test, lint 5단계.

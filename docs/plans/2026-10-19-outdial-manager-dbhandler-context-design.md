# VOIP-1628 outdial-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. origin/main b5af852d1 기준 컨텍스트 없는 DB 호출은 outdialtarget.go:429(OutdialTargetUpdateProgressing 의 h.db.Exec) 1건이며 함수는 이미 ctx 를 받음. OutdialTargetGetAvailable(outdialtarget.go:461~477)은 순회 뒤 rows.Err() 가 없음. 인터페이스(main.go)와 mock 은 이미 ctx 를 받으므로 변경 없을 것(구현 뒤 go generate 후 diff 0 로 확인).

## 변경
- outdialtarget.go:429 h.db.Exec -> h.db.ExecContext(ctx, ...).
- OutdialTargetGetAvailable: 순회 뒤 rows.Err() 확인, 에러는 %w 로 감쌈.
- 단건 getter 5곳(row.Err 호출 줄 116, 274, 152, 213, 258 의 Errorf 줄: outdial.go:117, outdialtarget.go:275, outdialtargetcall.go:153, 214, 259)은 이미 !row.Next() 분기 안에서 row.Err() 를 먼저 확인함(ErrNotFound 오판 없음). 메시지가 %v 라 원인이 보존되지 않으므로 이 5줄만 %w 로 변경(테스트의 errors.Is 단언용). 그 밖의 기존 %v 는 변경하지 않음.
- bin-outdial-manager/go.mod, go.sum: go-sqlmock v1.5.2 test 의존성 추가(선례 bin-billing-manager). 현재 dbhandler 테스트는 sqlite 라 순회 에러 주입 불가.

## 취소 부작용 판단
- 진입 ctx 는 Background(listenhandler/main.go:155, cmd/outdial-control)이고 WithTimeout/WithCancel/WithDeadline 파생이 없어 즉시 동작 변화 없음. 효과는 noctx 일관성과 향후 취소 도입 대비.
- 트랜잭션 없음(Begin/Commit 0건). 쓰기 뒤 `_ = h.xxxUpdateToCache` 10건은 에러를 무시하므로 커밋된 쓰기를 실패로 보고하는 경로가 없어 WithoutCancel 분리 불필요.
- GetAvailable 순회 에러가 현재는 부분 목록 200 으로 반환됨(소비자 listenhandler/v1_outdials.go:406). 수정 후 에러 반환.

## 범위 밖(보고서 한 줄)
- `_ = UpdateToCache` 무시와 캐시 TTL 24시간 stale(취소 도입 시).
- 호출부 outdialtargethandler UpdateProgressing 의 쓰기 뒤 h.Get(ctx) 재조회와 try_count+1 비멱등(취소 도입 시 재시도 이중 증가 가능).

## 검증(verify)
- noctx, rowserrcheck 임시 설정(워크트리 밖 yml, default none, vendor 제외): 프로덕션 0건(dbhandler/main_test.go 의 db.Exec 는 테스트 파일). 목록 함수 4개 순회 뒤 rows.Err 존재를 git grep 으로 병행 확인. 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트 ctx_rows_test.go(수정 전 RED 확인 후 통과): sqlmock RowError 주입 9곳. 단건 5곳과 GetAvailable 은 에러 반환과 errors.Is 원인 보존 단언(단건은 ErrNotFound 아님도 단언, GetAvailable 은 수정 전 실패). 이미 rows.Err 가 있고 %v 인 목록 3곳(outdial.go:224, outdialtarget.go:348, outdialtargetcall.go:316)은 줄을 바꾸지 않으므로 에러 반환과 메시지에 원인 문자열 포함만 단언(회귀 방어), 취소된 ctx 에서 OutdialTargetUpdateProgressing 중단(sqlmock 으로 통일. ExpectExec(...).WillReturnResult(sqlmock.NewResult(0, 1)) 를 등록하고 사전 취소된 ctx 로 호출해 메시지에 context.Canceled 문자열 포함을 단언(outdialtarget.go:430 의 에러가 %v 라 errors.Is 는 불가, 줄을 바꾸지 않음. 선례 billing 도 같은 방식). 핸들러의 cache 는 nil(취소된 ctx 의 캐시 갱신 DB 조회 실패는 `_ =` 로 무시되어 패닉 없음). 수정 전 Exec 는 ctx 를 보지 않아 nil 반환으로 실패. 수정 후 기대는 소비되지 않으므로 ExpectationsWereMet 호출 안 함. utilHandler 는 gomock mock 주입).
- 테스트 작성 주의: GetAvailable 은 PrepareContext 경유라 ExpectPrepare("(?i)select").ExpectQuery().WillReturnRows(...) 형태(ExpectQuery 는 인자 없음, 쿼리가 소문자 select 이므로 대소문자 무시). OutdialTargetCallGetByReferenceID, GetByActiveflowID 는 cache 를 먼저 호출하므로 cachehandler gomock 이 미스(에러)를 반환하도록 주입. 목록 3곳은 token 을 직접 넘기고(token=="" 이면 utilHandler.TimeGetCurTime 호출) filters 는 빈 map(nil 이면 패닉). 취소 케이스는 utilHandler.TimeNow gomock 기대 필요.
- 변이: ExecContext->Exec 복원, getter row.Err 제거, getter %w->%v 복원, GetAvailable rows.Err 제거 각각 해당 테스트 실패. 이미 rows.Err 가 있는 목록 3곳은 주입 테스트가 회귀 방어(변이 대상 아님).
- docs 동기화: models/, listenhandler, config, go.mod replace 변경 없음이라 서비스 docs 갱신 없음.
- 5단계: go mod tidy, vendor, generate, test, lint. go.mod/go.sum 은 sqlmock 과 그 전이 의존성 관련 줄만(tidy 결과 diff 로 확인하고 다른 변경이 생기면 보고).

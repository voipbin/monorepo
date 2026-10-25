# VOIP-1655 bin-ai-manager pkg/dbhandler 테스트 격리 설계

## 1. 티켓 검토
유효. origin/main d0588f5e5 에서 `go test -count=2 ./pkg/dbhandler` 가 최상위 34건 실패한다. 원인은 패키지 전체가 공유하는 in-memory sqlite(TestMain 1회 스키마)에 고정 id 와 고정 업무 키로 INSERT 하고 정리하지 않는 것이다. 제품 코드는 원인이 아니다. 기준선: 같은 main 에서 `-count=20 -shuffle=on ./...` 는 dbhandler 만 FAIL, 나머지 41개 패키지 ok(PR 본문에 요약).

## 2. 범위
제품 코드 무변경. `bin-ai-manager/pkg/dbhandler/*_test.go` 만 변경. rowserr_test.go 는 별도 `sql.Open("dbhandler_failing_rows")` 를 쓰고 dbTest 를 쓰지 않아 제외.

## 3. 설계
- main_test.go 에 `resetTestDB(t *testing.T)`: `sqlite_master`(`type='table'`, `sqlite_%` 제외)에서 이름을 모두 읽고 `rows.Err()` 확인과 `rows.Close()` 를 루프 직후 명시 호출(defer 금지, MaxOpenConns(1) 데드락 방지)한 뒤 각 테이블을 `DELETE FROM "name"` 한다. 실패하면 `t.Fatalf`. 새 테이블은 자동 포함.
- TestMain 과 `Test_RowsErr_NotMisreported`(rowserr_test.go) 를 제외한 모든 Test 함수 첫 줄에서 호출한다(dbTest 를 쓰지 않는 Test_IsErrDuplicate 포함, 무해). 시작 시점 정리라 선행 실패에 강하다.
- 기존 테스트 내부 DELETE 는 유지한다.
- 알려진 한계: 격리 단위는 Test 함수이며 한 함수 안의 t.Run 사이에는 격리가 없다. 서브테스트 순서는 shuffle 대상이 아니고 이번 변경으로 바뀌지 않는다(현재 `-count=1` 통과 상태 유지). 신규 Test 가 호출을 빼먹는 것은 구조로 막지 않는다(과설계 회피).
- 대안 (b) 테스트별 이름 DB: 마지막 커넥션이 닫히면 cache=shared DB 소멸, 커넥션 누수 위험. (c) 고유 id: 고정 id 기대값 테스트가 많아 변경량 과다.

## 4. 검증 기준
- 전제: `t.Parallel` 0건, 수정 전 `-count=1 -shuffle=on` 통과(확인함).
- 호출 누락 grep: `func Test` 중 첫 줄 호출이 없는 것이 정확히 TestMain, Test_RowsErr_NotMisreported 2개.
- dbhandler `-count=2`, 그리고 시드 1, 7, 42, 99, 12345, on 각각 `-count=20 -shuffle=<seed>` 통과(시드 수는 티켓 인수 조건, 각 수 초 내외).
- 음성 확인: Test_AIList, Test_AIcallList, Test_AIAuditList_ExcludesSoftDeleted, Test_AIAuditCountProgressing_ExcludesSoftDeleted 에 대해, 임시 scratch 테스트가 시드 행을 남긴 뒤 이 테스트들이 건수를 정상 단언함을 확인한다(커밋 금지).
- mutation(커밋 금지): Test_AIcallCreate 의 호출 제거, 헬퍼에서 `ai_ais` 제외, 헬퍼 SQL 을 잘못 바꿔 Fatalf 경로 확인. 각각 `-count=2` 실패 확인 후 복원, git diff 로 확인.
- bin-ai-manager `./...` `-count=20 -shuffle=on`, golangci-lint, CI 통과.

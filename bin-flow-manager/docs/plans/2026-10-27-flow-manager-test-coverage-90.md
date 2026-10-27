# bin-flow-manager 단위 테스트 커버리지 90% 상향

## 1. 티켓 검토

- 판정: 유효. origin/main 0bfbe2a1d 에서 수기 코드(mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록은 최대 count) 3675/4252 = 86.43% (측정 회차에 따라 3676/4252 = 86.45% 로 1 statement 달라지며 카드 값은 86.45%). 90.0% 까지 +152, 권장 90.5% 까지 +174 statement 가 필요하다.
- 도달 가능한 미커버가 필요량을 충분히 넘으므로 차단 없이 진행한다. 외부 연결이 있어야 도달하는 부분과 직렬화 실패 같은 분기는 대상에서 제외한다.

## 2. 설계 (테스트 계획, 제품 코드 변경 없음)

신규 `_test.go` 만 추가하고 기존 테스트 파일과 비테스트 파일은 고치지 않는다. 우선순위는 미커버가 큰 패키지의 오류 전파와 경계 입력이다.

| 신규 테스트 파일 (접미사 `_coverage90`) | 대상 | 미커버 | 목표 | 방법 |
|---|---|---|---|---|
| activeflowhandler/actionhandle_coverage90_test.go | actionhandle.go | 50 | 38 | 각 action 핸들러 직접 호출, strict gomock, 오류 원인과 stack 상태 단언 |
| activeflowhandler/execute_coverage90_test.go | execute.go 의 dispatch 와 루프 | 40 | 30 | 표 기반으로 action 종류별 핸들러 성공 경로(다음 action 반환, block 의 empty 반환)와 오류 반환 분기(오류를 로그만 남기고 다음 action 으로 진행하는 분기는 그 동작을 단언), 루프 종료와 finish 처리, Execute/ExecuteContinue 의 래핑 분기 |
| activeflowhandler/lifecycle_coverage90_test.go | stop, start, event, variable, db, action, common, main 생성자 | 26 | 20 | gomock, 반환 오류와 부수 호출 단언 |
| dbhandler/error_coverage90_test.go | activeflow.go, flows.go, variable.go | 61 | 36 | go-sqlmock(쿼리 오류)과 MockCacheHandler(캐시 조회, 갱신 오류와 성공 반환) |
| listenhandler/coverage90_test.go | v1_* 의 URI 파싱, 400, 404, panic 복구 | 63 | 22 | 핸들러 직접 호출, 채널 동기화 |
| flowhandler/coverage90_test.go | action, count, direct_hash, event, 생성자 | 20 | 16 | strict gomock |
| subscribehandler/coverage90_test.go | Run 의 바인딩 오류, 이벤트 처리 오류 | 17 | 10 | gomock |
| 소규모 패키지 | actionhandler, stackmaphandler, variablehandler, models, cachehandler | 19 | 10 | 표 기반 입력 검증 |
| cmd/flow-manager, internal/config | 연결 이전 경로(결정적 경로만 계산: runDaemon 연결 이전, initSignal, initCache. 서버 goroutine 내부와 DB 연결 성공이 전제인 run 본문은 제외) | 도달 가능분만 | 38 | miniredis, mock sockHandler |

목표 합계 220 statement(필요 +152 최소, +174 권장). 부족하면 listenhandler 잔여, dbhandler 잔여, flowhandler 잔여 순으로 도달 가능 항목을 추가한다.

- 기존 파일(`*_dependency_error_test.go`, `execute_error_test.go`, `db_error_test.go` 등)과 이름이 겹치지 않도록 새 파일과 테스트 이름에 `coverage90` 접미사를 쓰고, 기존 헬퍼는 재사용할 수 있는 것만 재사용한다. execute 의 dispatch 는 미커버 대부분이 각 action 핸들러 오류 반환 줄이라 오류 주입으로 도달한다.
- 현재 동작을 고정하는 테스트는 현재 동작을 설명하는 중립적인 이름으로 쓰고 제품 코드는 바꾸지 않는다.
- dbhandler 는 `sqlmock.New()` 로 `sql.DB` 를 만들어 handler 에 주입하고(`ctx_rows_test.go` 선례), 쿼리는 기본 정규식 매처, 각 테스트 끝에 `ExpectationsWereMet` 를 단언한다.
- cmd 는 `os.Exit` 와 `log.Fatal` 경로를 타지 않는다. 설정은 `config.Get()` 가 돌려주는 포인터의 필드를 덮어쓰고 `t.Cleanup` 으로 복구한다(flow-control 테스트의 `useMalformedDSN` 선례를 따르며, viper 전역은 flow-control 의 `resetViper` 와 같은 Reset 후 Cleanup 으로 격리한다). `runDaemon` 은 형식이 잘못된 DSN 으로 `sql.Open` 단계에서 실패시켜 네트워크 접근 없이 연결 이전 경로만 확인하고, `initSignal` 은 `runDaemon` 테스트 한 곳에서만 호출해 `chSigs` 에 직접 값을 보내 `chDone` 을 비운다(별도 `initSignal` 테스트를 두지 않는다. `-count` 반복 시 대기 goroutine 이 반복마다 하나씩 쌓이는 것은 허용하고, `chDone` 은 매번 비워 막히지 않게 한다). `runDaemon` 테스트는 Prometheus endpoint 와 listen 설정을 빈 문자열로 덮어쓰고 복구한다. Prometheus 는 전역 mux 에 중복 등록하면 panic 이므로 빈 endpoint 경로만 테스트하고 비어 있지 않은 endpoint 는 쓰지 않는다(`-count` 반복에서도 안전). 캐시 연결 성공 경로는 `cachehandler` 테스트에 선례가 있는 miniredis(go.mod 에 이미 있음)를 쓴다.
- cmd 목표 38 의 근거는 flow-manager 연결 이전 경로 약 24(runDaemon 앞부분, initSignal, initCache, initProm 빈 endpoint, runListen 은 Run 오류를 삼키고 runSubscribe 는 전파하는 동작을 구분해 확인. 구현 후 프로파일 비교로 확인하고 부족분은 보충 순서로 메운다)이다. flow-control 은 구현 결과 전체 합계가 목표를 넘어 신규 테스트를 추가하지 않았다(기존 테스트만 유지).
- 빈 호출 테스트와 자기비교 테스트는 쓰지 않는다. 오류 테스트는 반환 오류의 원인(`errors.Is` 또는 메시지)과 nil 결과, 호출되면 안 되는 하위 호출 부재(기대 미등록)를 단언한다.
- 시각에 의존하는 condition 분기는 현재 시각과 무관하게 결과가 정해지는 입력만 쓴다. goroutine 분기는 채널 또는 WaitGroup 으로 동기화한 뒤에 컨트롤러를 끝낸다. time.Sleep, 실제 네트워크, 환경 변수에 의존하지 않고 t.Parallel 을 쓰지 않는다.
- go.mod, go.sum 은 바꾸지 않는다(go-sqlmock 과 miniredis 는 이미 의존성에 있다). 테스트 값은 합성값만 쓴다.

## 3. Verify 항목

- 완료 기준: 수기 합계 90.5% 이상(covered 3849 이상, 최소 90.0% 는 3827 이상), 기존 테스트 전부 통과, golangci-lint 통과, go.mod/go.sum diff 없음.
- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, covermode 는 set 이고 `-race` 는 커버 측정과 분리한다. 산출물은 저장소 밖 scratch 에 두고 측정 뒤 `git status` 로 go.mod/go.sum 이 바뀌지 않았음을 확인한다. 기준 프로파일과 새 프로파일을 (파일, 범위) 단위로 비교해 위 표의 목표 블록이 커버됐는지도 확인한다. 수기 합계, 원본 합계, 제외 방법을 PR 본문에 적는다.
- 격리: 변경 패키지를 `-race -count=3 -shuffle=on` 으로, 신규 테스트를 `-count=20` 으로 반복 실행한다.
- 변이 검사(scratch 복사본에서만): 주요 4개 패키지(activeflowhandler, dbhandler, listenhandler, flowhandler)마다 오류 전파 제거 1건과 조건 반전 1건(dbhandler 의 NotFound 매핑과 listenhandler 의 panic 복구 포함)을 적용해 새 테스트가 실패함을 확인한다.
- 최종: 루트 CLAUDE.md 5단계 검증과 `bash scripts/check-test-conventions.sh`. 컨벤션은 gomock, 표 기반, `Test_` 접두, 컨트롤러 변수 `mc`, `reflect.DeepEqual` + `t.Errorf`. PR 파일 목록은 bin-flow-manager/ 안의 _test.go 와 이 문서뿐이다.

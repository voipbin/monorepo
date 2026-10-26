# bin-flow-manager 단위 테스트 커버리지 80% 상향 (VOIP-1657)

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main 161e9748 에서 수기 코드(mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록은 최대 count) 3295/4252 = 77.49%. 이 서비스에는 gens/ 가 없다. 원본(생성 코드와 mock 포함) 합계는 `go tool cover -func` 의 total 로 별도 기록한다.
- 80.0% 까지 107 statement(covered 3402 이상), 권장 80.5% 까지 128 statement(covered 3423 이상)가 필요하다. 카드의 기준선(77.47%, cmd 밖 미커버 467)과 1 statement 차이가 나며(3294 대 3295) 원인은 확인하지 못했고 산정 방식 차이일 수 있다. 목표 마진(128)에 영향이 없다.
- 미커버 957 statement 의 분포: cmd/flow-control 358, cmd/flow-manager 133, pkg/activeflowhandler 228, pkg/listenhandler 63, pkg/dbhandler 61, pkg/flowhandler 51, pkg/stackmaphandler 25, pkg/subscribehandler 16, 그 외 소수.
- 직전 작업(VOIP-1651, VOIP-1652)은 cmd 두 패키지를 대상에서 제외했다. 이번에는 DB, Redis, RabbitMQ 연결 없이 결정적으로 실행되는 cmd/flow-control 의 헬퍼, 명령 생성자, 입력 검증과 초기화 오류 경로를 대상에 포함한다.
- 도달 가능 상한은 필요량(128)을 넘는다(cmd/flow-control 의 연결 없는 구간 약 170(일부는 형식 오류 DSN 의 초기화 실패 경로), 그 밖의 패키지 오류 전파 분기. initFlowHandler 와 initCache 처럼 외부 연결이 필요한 구간은 도달 불가로 분류해 섞지 않는다). 분기별 분류는 PR 본문 밖의 별도 기록에 둔다. 따라서 차단 없이 진행한다.
- VOIP-1416 과 겹치는 패키지: 이 서비스의 80% 미만 패키지(activeflowhandler 외)가 해당할 수 있으며 흡수하지 않고 PR 본문에 명시만 한다.

## 2. 설계 (테스트 계획, 제품 코드 변경 없음)

우선순위는 미커버가 큰 패키지의 오류 전파, 입력 검증, 경계값이다. 수치만 올리는 빈 호출 테스트와 자기비교 테스트는 쓰지 않는다.

- pkg/activeflowhandler: db.go(Create/Get/Update/Delete/List 와 SetForwardActionID 의 하위 오류), execute.go(ExecuteContinue/ExecuteNextAction/executeAction 의 오류와 최대 반복, block/case_create 분기), actionhandle.go 의 오류 분기(옵션 파싱 실패, 하위 RPC 오류, 조건 분기), start/stop/event/variable 의 소수 분기. 기존 `*_error_test.go` 의 mock 구성을 따른다.
- pkg/flowhandler(db.go Get/Delete/List/Update/UpdateActions, direct_hash, event), pkg/dbhandler(캐시와 DB 오류 경로, 기존 sqlite 패턴, ActiveflowGetWithLock), pkg/stackmaphandler(GetAction/GetNextAction/PopStack 등), pkg/listenhandler(남은 오류 매핑, 서버 Run 의 성공 경로는 제외), pkg/subscribehandler(processEventRun, processEventCMCustomerDeleted 오류): 각 패키지 15~30 statement 를 목표로 한다.
- cmd/flow-control(생성자 테스트만으로 하한 128 을 채우지 않는다. 생성자 테스트는 Use, 플래그 목록, 각 플래그의 DefValue, RunE 비nil 을 모두 단언한다): resolveUUID(빈 값, 잘못된 형식, 정상), parseActions(빈 문자열, 잘못된 JSON, 정상), printJSON(정상은 표준 출력 캡처, marshal 불가 값은 오류), cmd* 생성자(이름과 플래그 기본값), initCommand(하위 명령 구성), runCreate 의 연결 이전 검증 오류(필수값, 잘못된 actions), initHandler 를 쓰는 run* 의 초기화 오류. 테스트는 cmd.Execute 를 쓰지 않고 run* 를 직접 호출하며(PersistentPreRunE 의 sync.Once 설정 고정을 피함), `config.Get().DatabaseDSN` 을 `t.Cleanup` 으로 복원하는 방식으로 `missing the slash` 계열 형식 오류 DSN 을 넣는다. 빈 DSN 은 기본 주소로 연결을 시도하므로 쓰지 않는다. viper 값은 매 테스트에서 `viper.Reset()` 후 필요한 키만 설정하고, initCommand 테스트는 생성만 확인하며(전역 viper/logrus 상태 변경은 복원) Execute 하지 않는다. printJSON 은 os.Stdout 을 캡처 후 복원하고 `t.Parallel` 을 쓰지 않는다.
- cmd/flow-manager, internal/config: DB, Redis, RabbitMQ 연결 없이 도달하는 함수(initProm 비활성 경로, LoadGlobalConfig 등)만 대상으로 하며, 신호 대기와 서버 goroutine 은 대상이 아니다.

목표 추가 커버(상한이 아님): activeflowhandler 90, flowhandler 25, dbhandler 25, stackmaphandler 15, listenhandler 15, subscribehandler 10, cmd/flow-control 90, cmd/flow-manager 와 config 5(runDaemon 은 호출하지 않는다. initSignal 이 전역 신호 처리와 goroutine 을 남기기 때문이다). 합계 약 275. 하한은 128 이며 이를 넘기면 나머지는 생략할 수 있다.

## 3. Verify 항목

- 완료 기준은 수기 합계 80.5% 이상(128 statement)이다. 최소선은 80.0%(107). 최종 분류 수치(도달 불가 N, 도달 가능 M)는 별도 기록에 남긴다, 기존 테스트 전부 통과, golangci-lint 통과, go.mod/go.sum diff 없음.
- 단언 강도: 오류 테스트는 `errors.Is` 또는 원인 오류와 비교하고, 상태 코드와 본문 또는 반환 구조체의 값으로 비교한다. 호출되면 안 되는 하위 호출은 기대를 등록하지 않아 컨트롤러가 실패하게 한다. `err != nil` 만 보는 테스트와 `gomock.Any()` 일색의 기대를 쓰지 않는다.
- 격리: 기존 테스트 일부에 time.Sleep 이 있는 파일(db_test, actionhandle_test, start_test, stop_test, v1_activeflows_test)에는 가급적 새 테스트를 추가하지 않고 신규 파일을 쓰며, `-race -count=3` 으로 상호작용을 확인한다. 테스트는 시간과 외부 상태에 의존하지 않는다. `time.Sleep` 대신 채널 또는 `sync.WaitGroup` 을 쓰고, 네트워크는 쓰지 않는다. 환경 변수와 viper 전역은 `t.Setenv` 와 `viper.Reset()` 으로 복구한다.
- 초기화 오류 테스트는 센티널이 없으므로 래핑 메시지(`failed to initialize handlers`, `could not connect to the database`)와 DSN 파싱 오류 문구의 부분 문자열, 반환 값 nil 여부를 모두 단언한다. internal/config 의 LoadGlobalConfig 는 sync.Once 이므로 테스트에서 once 를 `sync.Once{}` 로 되돌리고 globalConfig 를 `t.Cleanup` 으로 복원하며, `viper.Reset()` 은 시작과 종료(`t.Cleanup`) 모두에서 호출하고 AutomaticEnv 로 새는 환경 변수는 `t.Setenv` 로 비운다. initCommand 를 호출하면 바뀌는 logrus 포맷터와 레벨도 복원한다. printJSON 의 출력 캡처는 쓰기 쪽을 닫기 전에 별도 goroutine 으로 읽고 `defer` 로 복원한다. `-shuffle=on` 실행도 포함한다.
- 새 테스트 전후 프로파일을 (파일, 범위) 단위로 비교해 목표한 블록이 실제로 커버됐는지 확인한다.
- 변이 검사(scratch 복사본에서만, worktree 는 건드리지 않는다): 패키지마다 최소 1곳, 새 테스트가 늘린 구간 안에서 오류 전파 제거 1건과 조건 반전 1건을 적용하고 새 테스트가 실패함을 기록한다. cmd/flow-control 은 runCreate 의 연결 전 검증 순서 변경과 플래그 기본값 변경, activeflowhandler 는 최대 반복 분기의 조건 변경을 포함한다.
- 변경한 패키지를 `go test -race -count=3` 으로 실행하고, 커버 측정은 `-race` 없이 `-count=1` 로 재현한다.
- 최종 검증은 루트 CLAUDE.md 의 5단계와 저장소 루트의 `bash scripts/check-test-conventions.sh` 를 모두 실행한다. 컨벤션: gomock, 표 기반, `Test_` 접두, 컨트롤러 변수 `mc`, `reflect.DeepEqual` + `t.Errorf`.
- 테스트 값은 합성값만 쓴다. PR 파일 목록은 bin-flow-manager/ 안의 _test.go 와 이 문서뿐이다.
- 커버 측정은 mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록은 최대 count 로 합친 수기 합계를 쓴다(저장소 밖 계산 스크립트 사용, 원본 합계도 함께 기록).
- 구현 초기에 activeflowhandler 의 도달 가능분을 먼저 분류하고, 90 에 못 미치면 flowhandler, dbhandler, listenhandler 로 재배분한다. 80.5% 에 미달하나 80.0% 이상이면 추가 작업으로 80.5% 를 채운다.
- cmd/flow-control 패키지 전체에서 `t.Parallel` 을 쓰지 않고(viper 와 os.Stdout 이 전역이므로) 이유를 주석으로 남긴다. 모든 run* 테스트에서 `viper.Reset()` 을 시작과 종료에 호출한다.
- 변이 검사 결과는 실패한 테스트 이름까지 PR 밖 별도 기록에 남긴다. 생성자 테스트는 create 의 type 기본값 `flow`, list 의 type 빈 값, list 의 limit 100 을 실제로 비교한다.

# bin-flow-manager 단위 테스트 커버리지 85% 상향

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main a47f5a2a 에서 수기 코드(mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록은 최대 count) 3567/4252 = 83.89%. 이 서비스에는 gens/ 가 없다. 반올림 값 83.90% 과는 0.01%p 차이이며 반올림 차이다.
- 85.0% 까지 48 statement(covered 3615 이상, 3614 는 84.995%), 권장 85.5% 까지 69 statement(covered 3636 이상)가 필요하다.
- 미커버 685 statement 의 분포: pkg/activeflowhandler 173, cmd/flow-control 141, cmd/flow-manager 133, pkg/listenhandler 63, pkg/dbhandler 61, pkg/flowhandler 51, pkg/stackmaphandler 25, pkg/subscribehandler 16, internal/config 8, 그 외 소수.
- 이전 커버리지 작업이 cmd/flow-control 의 헬퍼와 생성자, activeflowhandler 의 db.go 오류 전파를 이미 다뤘다. 남은 미커버 중 도달 가능한 것은 하위 의존성(RPC, stackmap, 핸들러) 오류의 전파 분기와 stackmaphandler 의 경계 입력이다.
- 도달 가능 상한은 필요량(69)을 크게 넘는다. 확인한 도달 가능 후보: activeflowhandler/actionhandle.go 의 하위 RPC, stackmap 오류 반환 약 60 블록(json.Marshal 실패 분기와 비동기 goroutine 안의 호출은 제외), flowhandler/db.go 의 mock 오류 전파 약 30 statement, stackmaphandler/action.go 의 경계 입력 약 19 statement. 도달 불가로 분류해 제외하는 것(블록 수는 statement 수보다 작다. 미커버 블록은 대부분 statement 2개): cmd/flow-manager 의 main 과 runDaemon, initFlowHandler 와 initCache, listenhandler 의 json.Marshal 실패, dbhandler 의 ToSql 과 Scan 계열. cmd/flow-control 의 형식 오류 DSN 초기화 오류 경로는 완충 후보이며 상한 계산에는 넣지 않았다. 따라서 차단 없이 진행한다.
- 이 서비스의 80% 미만 패키지 일반 상향 작업과 겹치는 패키지: 80% 미만인 pkg/subscribehandler(72.9%)와 pkg/flowhandler(79.6%), cmd 두 패키지가 해당할 수 있으며 흡수하지 않고 PR 본문에 명시만 한다.
- 열린 PR 중 bin-flow-manager 를 건드리는 것은 없고, 최근 머지된 이전 커버리지 작업은 위 범위와 겹치지 않는 구간을 다뤘다.

## 2. 설계 (테스트 계획, 제품 코드 변경 없음)

우선순위는 미커버가 큰 패키지의 오류 전파와 경계 입력이다. 빈 호출 테스트와 자기비교 테스트는 쓰지 않는다. 기존 테스트 파일은 고치지 않고 신규 `_test.go` 파일만 추가한다(기존 파일 일부에 time.Sleep 이 있다). 기존 헬퍼 `newExecuteErrorTestActiveflow` 를 재사용하고, mock 접근이 필요해 신규 파일 안에 strict mock 구성 헬퍼를 별도로 둔다. 기존 `Test_executeAction_invalidOption` 등이 이미 커버한 유형은 신규 표에서 제외하며 프로파일 비교로 확인한다.

파일별 목표(미커버 statement 는 기준 프로파일의 수치, 목표는 추가 커버 statement 수):

| 신규 테스트 파일 | 대상 | 미커버 | 목표 |
|---|---|---|---|
| pkg/activeflowhandler/actionhandle_dependency_error_test.go | actionhandle.go | 107 (도달 가능 약 89) | 70 |
| pkg/flowhandler/db_dependency_error_test.go | db.go | 31 | 25 |
| pkg/stackmaphandler/action_boundary_test.go | action.go, stack.go | 19 + 5 | 15 |
| 보충(부족할 때만) | subscribehandler, dbhandler, cmd/flow-control 초기화 오류 | 16 + 61 + 141 | 필요량 |

- activeflowhandler 의 단언: 오류를 반환하는 분기는 반환 오류의 원인(`errors.Is`, 래핑하지 않고 `%v` 로 불투명하게 만든 FlowV1FlowCreate 지점, 센티널이 없는 conference_join 참조 유형 불일치와 connect 의 빈 호출 결과는 메시지 포함)과 nil 결과를 확인한다. 오류를 그대로 반환하는 지점은 주입한 오류와 `errors.Is` 로 비교하고, pkg/errors 로 감싸는 지점은 `errors.Is` 로 원인 연결을 확인한다. 로그만 남기고 오류를 반환하지 않는 분기는 반환 action(ActionNext 또는 nil), 오류 여부, 이후 하위 호출 부재(strict mock), activeflow 의 ForwardStackID/ForwardActionID 상태 중 해당 항목을 케이스마다 단언한다. updateStackProgress 오류는 헬퍼가 돌려주지 않는 db mock 을 `h.db.(*dbhandler.MockDBHandler)` 로 꺼내 ActiveflowUpdate 에 오류를 주입한다. `err == nil` 만 확인하는 형태는 쓰지 않는다.
- activeflowhandler: 각 `actionHandleXxx` 를 직접 호출한다(execute.go 의 dispatch 줄은 성공 반환이 미커버의 대부분이라 이번 범위에서 제외하고, 같은 이유로 Execute, ExecuteContinue, ExecuteNextAction 오류 분기도 제외한다). 표 기반으로 stackmap GetAction 오류, updateStackProgress 오류, PushStack 오류(fetch, fetch_flow, conference_join, queue_join, connect 등 각 호출 지점), reqHandler(CallV1CallGetDigits, CallV1CallGet, ConferenceV1ServiceTypeConferencecallStart, CallV1ConfbridgeCreate, FlowV1FlowCreate, CallV1CallsCreate) 오류, 호출 결과가 비어 있는 경우를 확인한다. 참조 유형 불일치 분기(conference_join, recording 계열)는 `verifyActionType` 이 executeAction 앞에서 걸러내므로 핸들러 직접 호출로만 도달하며 이 방식으로 확인한다. condition_datetime 은 실행 시각에 의존하지 않도록 weekday 가 0~6 밖의 값인 입력만 쓴다(weekday 불일치 시 이후 시각 비교를 건너뛰고 false target 분기로 가므로 시각과 무관하다). minute 비교와 일치 경로는 시각에 좌우되므로 제외한다. 제외: actionHandleBranch 의 updateStackProgress 오류 분기 1곳(2 statement), SetVariable 실패 로그 분기(반환값 없음), 비동기 goroutine 안의 호출, json.Marshal 실패 분기, 코드가 직접 Marshal 한 값을 되읽는 Unmarshal 실패 분기(hangup relay).
- flowhandler: strict mock 은 신규 파일에서 직접 구성하고 테스트 이름은 `Test_..._dependencyError` 접미사로 기존과 겹치지 않게 한다. Get 의 ErrNotFound 는 `errors.As` 로 `*cerrors.VoipbinError` 를, `errors.Is` 로 `dbhandler.ErrNotFound` 를 둘 다 확인한다. Create 의 GenerateFlowActions 오류, 캐시 생성 오류, 생성 후 조회 오류, direct 정리 실패 로그 경로, List/Update/UpdateActions/Delete 의 하위 DB 오류와 액션 생성 오류.
- stackmaphandler: 실제 구현을 쓰는 순수 로직이므로 mock 이 없다. 반환 stack id 와 action 값을 정확히 단언한다. GetAction/AddActions 의 빈 stack id, 빈 action id, 존재하지 않는 스택, 빈 Actions(메인과 서브 스택 구분), 최대 스택 횟수 초과(ReturnStackID 가 자기 자신을 가리키는 순환 스택, releaseStack=false 고정. true 이면 순환이 스택을 삭제해 다른 오류로 바뀐다), GetNextAction 의 스택 없음과 액션 없음, 서브 스택에서 시작한 start 액션 id 가 메인 스택으로 단축되지 않는 경계, GetAction 의 메인 스택 start 액션 반환(`action.IDStart`)과 다음 action ID 가 `IDEmpty` 인 오류. 빈 Actions 케이스는 startStackID 를 메인으로 주어야 하고, GetNextAction 의 순환 케이스는 ReturnStackID 가 자기 자신이며 ReturnActionID 가 그 스택의 마지막 action ID 일 때만 도달하며 그렇지 않으면 액션 없음 분기로 빠진다. UUID 생성이 필요한 PushStackByActions 의 stackID Nil 경로는 필요할 때만 `MockUtilHandler` 로 UUID 를 주입한다.

목표 합계 110 statement(권장선 69, 최소선 48). 하한을 넘기면 남은 항목은 생략하고, 부족하면 보충 행을 쓴다.

## 3. Verify 항목

- 완료 기준은 수기 합계 85.5% 이상(covered 3636 이상). 최소선 85.0%(3615). 기존 테스트 전부 통과, golangci-lint 통과, go.mod/go.sum diff 없음.
- 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`(covermode set, 대상 `./...`). 산출물(cp.out 등)은 scratch 경로에 쓰고 저장소에 두지 않으며 측정 뒤 `git status` 로 go.mod/go.sum 이 바뀌지 않았는지 확인한다. `-race` 실행은 별도로 하고 커버 측정에는 쓰지 않는다. 수기 합계는 mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록은 최대 count 로 합친 값이며 원본 합계도 PR 본문에 함께 적는다.
- 목표 블록 확인: 기준 프로파일과 새 프로파일을 (파일, 범위) 단위로 비교해 위 표의 파일별 목표 범위가 실제로 커버됐는지, 목표 밖 블록이 의도치 않게 바뀌지 않았는지 확인한다.
- 단언 강도: 오류 테스트는 반환 오류가 주입한 센티널과 `errors.Is` 로 연결되거나 래핑 메시지를 포함하는지, 반환 값이 nil 인지, 호출되면 안 되는 하위 호출은 기대를 등록하지 않아 컨트롤러가 실패하는지를 확인한다. `err != nil` 만 보는 테스트와 `gomock.Any()` 일색의 기대를 쓰지 않는다.
- 격리: time.Sleep 과 실제 시간, 네트워크, 환경 변수에 의존하지 않는다. t.Parallel 을 쓰지 않는다. 변경 패키지를 `-race -count=3 -shuffle=on` 으로, 신규 파일의 테스트는 `-count=20` 으로 반복 실행해 순서 의존(전역 지표 등)을 확인한다.
- 변이 검사(scratch 복사본에서만): 패키지마다 오류 전파 제거 1건과 조건 반전 1건을 적용해 새 테스트가 실패함을 기록한다. condition_datetime 의 불일치 분기와 flowhandler 의 ErrNotFound 매핑도 변이 대상에 넣는다.
- 최종 검증은 루트 CLAUDE.md 5단계와 `bash scripts/check-test-conventions.sh`. 컨벤션은 gomock, 표 기반, `Test_` 접두, 컨트롤러 변수 `mc`, `reflect.DeepEqual` + `t.Errorf`.
- 테스트 값은 합성값만 쓴다. PR 파일 목록은 bin-flow-manager/ 안의 _test.go 와 이 문서뿐이다.

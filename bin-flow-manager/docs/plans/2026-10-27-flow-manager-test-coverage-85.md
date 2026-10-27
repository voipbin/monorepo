# bin-flow-manager 단위 테스트 커버리지 85% 상향 (VOIP-1659)

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main a47f5a2a 에서 수기 코드(mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록은 최대 count) 3567/4252 = 83.89%. 이 서비스에는 gens/ 가 없다. 카드의 기준선(83.90%)과 0.01%p 차이이며 반올림 차이다.
- 85.0% 까지 48 statement(covered 3615 이상, 3614 는 84.995%), 권장 85.5% 까지 69 statement(covered 3636 이상)가 필요하다.
- 미커버 685 statement 의 분포: pkg/activeflowhandler 173, cmd/flow-control 141, cmd/flow-manager 133, pkg/listenhandler 63, pkg/dbhandler 61, pkg/flowhandler 51, pkg/stackmaphandler 25, pkg/subscribehandler 16, internal/config 8, 그 외 소수.
- 직전 작업(VOIP-1651, 1652, 1657)이 cmd/flow-control 의 헬퍼와 생성자, activeflowhandler 의 db.go 오류 전파를 이미 다뤘다. 남은 미커버 중 도달 가능한 것은 하위 의존성(RPC, stackmap, 핸들러) 오류의 전파 분기와 stackmaphandler 의 경계 입력이다.
- 도달 가능 상한은 필요량(69)을 크게 넘는다. 확인한 도달 가능 후보: activeflowhandler/actionhandle.go 의 하위 RPC, stackmap 오류 반환 약 60 블록(json.Marshal 실패 분기와 비동기 goroutine 안의 호출은 제외), flowhandler/db.go 의 mock 오류 전파 약 30 statement, stackmaphandler/action.go 의 경계 입력 약 19 statement. 도달 불가로 분류해 제외하는 것(블록 수는 statement 수보다 작다. 미커버 블록은 대부분 statement 2개): cmd/flow-manager 의 main 과 runDaemon, initFlowHandler 와 initCache, listenhandler 의 json.Marshal 실패, dbhandler 의 ToSql 과 Scan 계열. cmd/flow-control 의 형식 오류 DSN 초기화 오류 경로는 완충 후보이며 상한 계산에는 넣지 않았다. 따라서 차단 없이 진행한다.
- VOIP-1416 과 겹치는 패키지: 80% 미만인 pkg/subscribehandler(72.9%)와 pkg/flowhandler(79.6%), cmd 두 패키지가 해당할 수 있으며 흡수하지 않고 PR 본문에 명시만 한다.
- 열린 PR 중 bin-flow-manager 를 건드리는 것은 없고, 최근 머지된 VOIP-1651, 1652, 1657 은 위 범위와 겹치지 않는 구간을 다뤘다.

## 2. 설계 (테스트 계획, 제품 코드 변경 없음)

우선순위는 미커버가 큰 패키지의 오류 전파와 경계 입력이다. 빈 호출 테스트와 자기비교 테스트는 쓰지 않는다. 기존 테스트 파일은 고치지 않고 신규 `_test.go` 파일만 추가한다(기존 파일 일부에 time.Sleep 이 있다). 기존 strict mock 헬퍼 `newExecuteErrorTestHandler`, `newExecuteErrorTestActiveflow` 를 재사용한다.

- pkg/activeflowhandler (목표 추가 약 70 statement): 신규 `actionhandle_dependency_error_test.go`. 표 기반으로 하위 의존성 오류가 호출자에게 원인과 함께 전파되는지 확인한다. 대상은 goto, fetch_flow, condition_call_digits, condition_call_status, condition_datetime, condition_variable, conference_join, connect, hangup relay 구간, transcribe_recording 의 stackmap GetAction 오류, updateStackProgress 오류, PushStack 오류, reqHandler(CallV1CallGetDigits, CallV1CallGet, ConferenceV1ServiceTypeConferencecallStart, CallV1ConfbridgeCreate, FlowV1FlowCreate, CallV1CallsCreate) 오류, 참조 유형 불일치, 호출 결과가 비어 있는 경우다. 조건 분기의 불일치(weekday, minute) 경로와 ParseFloat 실패 경로도 포함한다. 비동기 goroutine 안의 호출과 json.Marshal 실패 분기는 제외한다.
- pkg/flowhandler (목표 약 25): 신규 `db_dependency_error_test.go`. Get 의 ErrNotFound 매핑과 일반 오류, Create 의 GenerateFlowActions 오류, 캐시 생성 오류, 생성 후 조회 오류, direct 정리 실패, List/Update/UpdateActions/Delete 의 하위 DB 오류와 액션 생성 오류. 오류는 센티널 또는 원인으로 비교한다.
- pkg/stackmaphandler (목표 약 15): 신규 `action_boundary_test.go`. GetAction/AddActions 의 빈 stack id, 빈 action id, 존재하지 않는 스택, 빈 Actions(메인 스택과 서브 스택 구분), 최대 스택 횟수 초과, GetNextAction/PopStack 계열의 스택 없음과 액션 없음 경계.
- pkg/activeflowhandler/execute.go (목표 약 15): 신규 파일에서 executeAction 분기 중 도달 가능한 오류와 핸들러 미지원 유형 경로.
- 부족하면 pkg/subscribehandler(72.9%), cmd/flow-control 의 형식 오류 DSN 초기화 오류 경로, pkg/dbhandler 의 DB 닫힘 오류 경로로 보충한다.

목표 추가 약 125 statement(권장선 69, 최소선 48). 하한을 넘기면 남은 항목은 생략한다.

## 3. Verify 항목

- 완료 기준은 수기 합계 85.5% 이상(covered 3636 이상). 최소선 85.0%(3615). 기존 테스트 전부 통과, golangci-lint 통과, go.mod/go.sum diff 없음.
- 단언 강도: 오류 테스트는 반환 오류가 주입한 센티널과 `errors.Is` 로 연결되거나 래핑 메시지를 포함하는지, 반환 값이 nil 인지, 호출되면 안 되는 하위 호출은 기대를 등록하지 않아 컨트롤러가 실패하는지를 확인한다. `err != nil` 만 보는 테스트와 `gomock.Any()` 일색의 기대를 쓰지 않는다.
- 격리: time.Sleep 과 실제 시간, 네트워크, 환경 변수에 의존하지 않는다. 신규 테스트는 t.Parallel 을 쓰지 않고 `-race -count=3` 과 `-shuffle=on` 으로 확인한다.
- 새 테스트 전후 프로파일을 (파일, 범위) 단위로 비교해 목표 블록이 실제로 커버됐는지 확인한다.
- 변이 검사(scratch 복사본에서만): 패키지마다 오류 전파 제거 1건과 조건 반전 1건을 적용해 새 테스트가 실패함을 기록한다.
- 최종 검증은 루트 CLAUDE.md 5단계와 `bash scripts/check-test-conventions.sh`. 컨벤션은 gomock, 표 기반, `Test_` 접두, 컨트롤러 변수 `mc`, `reflect.DeepEqual` + `t.Errorf`.
- 테스트 값은 합성값만 쓴다. PR 파일 목록은 bin-flow-manager/ 안의 _test.go 와 이 문서뿐이다. 측정은 mock 과 DO NOT EDIT 파일을 제외한 수기 합계(저장소 밖 스크립트)이며 원본 합계도 함께 기록한다.

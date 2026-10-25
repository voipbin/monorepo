# bin-flow-manager 단위 테스트 커버리지 75% 상향 (VOIP-1652)

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main accc03e 에서 수기 코드(gens/, mock_*.go, DO NOT EDIT 제외, 같은 블록은 최대 count) 3105/4252 = 73.02%.
- 75% 까지 84 statement, 권장 76% 까지 127 statement 가 필요하다. 원본(생성 코드와 mock 포함)은 65.5%.
- 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`. 수기 합계는 이 프로파일에서 mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 블록을 빼고 (파일, 범위) 단위로 중복 블록은 최대 count 로 합쳐 계산한다(이 서비스에는 gens/ 가 없어 해당 없음, 제외되는 파일은 mock_*.go 뿐). 이 값은 `go tool cover -func` 의 total(원본 65.5%)과 다르다. 완료 기준은 수기 합계 75.0% 이상이며 76% 이상을 목표로 하고, PR 본문에 두 값을 모두 적는다.
- 수기 미커버 1147 statement 중 cmd/flow-control 358, cmd/flow-manager 133 은 main 이라 대상에서 제외한다(cobra 명령 조립, 신호 대기, os.Exit, 데몬 goroutine 으로 단위 테스트가 구조적으로 어렵다).
- 나머지 미커버는 대부분 오류 경로다: JSON 파싱 실패, 하위 핸들러/DB 오류 전파, 입력 검증 분기. 제품 코드 변경 없이 gomock 으로 결정적 테스트가 가능하다. 예외로 `go func()` 안에서 reqHandler 를 호출하는 분기(SetForwardActionID 의 forwardNow, webhook 와 conversation_send 의 비동기 모드)는 호출 시점이 비결정적이므로 mock 의 `.Do` 에서 채널로 신호를 보내 대기하는 방식으로 다루고, 테스트는 취소되지 않는 ctx 를 쓰며 `-race` 로 실행한다. 계획의 필수 항목이 아니며 필요량 충족에 쓰지 않아도 된다.
- 겹치는 선행 작업: VOIP-1651(머지됨)이 같은 서비스를 70% 로 올렸다. 이번은 그 위의 추가분이다.

## 2. 설계 (테스트 계획, 제품 코드 변경 없음)

패키지별 대상(미커버 statement 기준, 괄호는 현재 미커버 수):

- pkg/listenhandler (154): 개별 v1 핸들러는 `(nil, err)` 를 그대로 반환하고 상태 코드 매핑은 processRequest 의 기본 오류 처리(일반 오류 400, `dbhandler.ErrNotFound` 404, typed VoipbinError 는 해당 코드와 DataTypeVoipbinError)에서 일어난다. 따라서 (a) 개별 핸들러 직접 호출 테스트는 잘못된 JSON 본문에서 오류를 반환하고 하위 핸들러를 호출하지 않음과, 하위 오류가 `errors.Is` 로 원인이 보존됨을 확인한다, (b) processRequest 를 거치는 테스트는 같은 입력이 일반 오류 400, ErrNotFound(pkg/errors Wrap 포함) 404, typed 오류의 코드와 본문으로 매핑됨을 상태 코드와 본문으로 비교한다, (c) 활성플로우 목록 필터 변환 실패는 핸들러가 `(simpleResponse(400), nil)` 을 직접 반환하므로 상태 코드로 단언한다(해당 직접 400 분기는 v1_activeflows.go 목록 필터, v1_count.go, v1_flows_direct_hash.go, v1_flows.go 의 네 곳). 신규 파일 v1_error_paths_test.go 의 표 기반 테스트. Run 은 QueueCreate 실패 경로만 다룬다(성공 경로는 goroutine 이 RPC 소비를 시작해 종료 수단이 없으므로 제외).
- pkg/variablehandler (47): resolveValue 의 모든 타입 분기, SubstituteOption 의 []byte, []string, []map, 포인터, 미지원 타입 분기(입력이 변이되는 결과를 값으로 검증), Substitute 의 Get 오류, Create/Get/Set/SetVariable/DeleteVariable 의 DB 오류 전파, 동적 변수 조회 실패 시 빈 문자열.
- pkg/actionhandler (30): ValidateActions 의 미지원 타입, call/connect 의 anonymous 유효/무효/옵션 파싱 실패, ActionFetchGet 의 오류(옵션 오류, 비 200 응답, 본문 JSON 오류)는 httptest 로컬 서버로 검증한다.
- pkg/activeflowhandler (255): activeflow.go(PushActions, AddActions, PopStack, ServiceStop 오류), stop.go, execute.go(ExecuteContinue, ExecuteNextAction 오류), db.go(Get, Delete, SetForwardActionID 오류) 의 오류 전파.
- activeflowhandler/actionhandle.go (130 미커버)의 오류 분기는 목표 60 이 부족할 때 보충 후보이며 비동기 분기는 위 규칙을 따른다. listenhandler 의 Execute/Continue POST 안의 goroutine 오류 로그 블록은 제외한다(필요량에 쓰지 않음).
- pkg/flowhandler/db.go (51, Get/Delete/Update 오류), pkg/dbhandler (61, 캐시/DB 오류 경로, 기존 sqlite 패턴), pkg/subscribehandler, pkg/stackmaphandler 는 위 목표가 부족할 때 숫자 목표(각 15~20)로 보충한다.
- ActionFetchGet 은 `&http.Client{}` 를 쓰고 URL 검증이 없으므로 loopback httptest 서버로 정확한 상태 코드와 본문을 만들어 검증할 수 있다(외부 네트워크 없음).

목표 추가 커버(목표이며 상한이 아님, 비동기 분기는 계산에 넣지 않음): listenhandler 65(Run 성공 경로 제외, QueueCreate 실패만), variablehandler 40, actionhandler 25, activeflowhandler 60, dbhandler 15, flowhandler 20, subscribehandler 10. 합계 약 235. 하한 기준은 76% 에 필요한 127 statement 이며 이를 넘기면 나머지 항목은 생략할 수 있다.

## 3. Verify 항목

- 수기 합계 76% 이상(최소 75.0%), 기존 테스트 전부 통과, golangci-lint 통과, go.mod/go.sum diff 없음.
- 단언 강도: 개별 핸들러 오류 테스트는 `errors.Is` 또는 반환 오류의 원문과 비교하고, processRequest 테스트는 상태 코드와 본문과 비교한다. `err != nil` 만 보는 테스트와 `gomock.Any()` 일색의 기대를 쓰지 않는다. 하위 핸들러가 호출되면 안 되는 경로는 기대를 등록하지 않아 컨트롤러가 실패하게 한다.
- 새 테스트 전후 프로파일을 (파일, 범위) 단위로 비교해 목표한 블록이 실제로 커버됐는지 확인한다.
- 최종 검증은 루트 CLAUDE.md 의 5단계(go mod tidy, go mod vendor, go generate ./..., go test ./..., golangci-lint run)를 모두 실행한다.
- 기준선: 새 테스트 작성 전에 변경 전 브랜치에서 `go test -race -count=3` 을 한 번 실행해 기존 불안정성을 구분한다. ValidateActions 의 옵션 파싱 실패는 경고만 남기고 nil 을 반환하므로 같은 목록의 뒤쪽 유효/무효 액션이 계속 검증됨을 함께 단언한다.
- 측정 뒤 go.mod/go.sum 의 `git diff` 가 비어 있음을 확인한다(`-mod=mod` 가 go.sum 을 바꿀 수 있음). check-test-conventions.sh 는 저장소 루트에서 실행한다.
- 컨벤션: gomock, 표 기반, `Test_` 접두, 컨트롤러 변수 `mc`, `reflect.DeepEqual` + `t.Errorf`(testify 금지). push 전 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 를 실행해 통과시킨다.
- 동기화: 신규 테스트는 `time.Sleep` 을 쓰지 않고 채널 또는 `sync.WaitGroup` 으로 goroutine 종료를 기다린다(컨트롤러 Finish 이후 호출로 인한 간헐 실패 방지).
- 각 오류 경로 테스트는 반환값과 부수효과(하위 호출 여부, 상태 코드, 변이된 맵 값)를 비교하며 같은 값을 만들어 비교하지 않는다.
- 변이 검사(scratch 복사본): 패키지마다 최소 1곳. listenhandler 오류 전파의 상태 코드 매핑 변경(패닉이 아니라 단언 실패로 잡히는 변이), variablehandler resolveValue 의 []string 분기 제거, actionhandler 의 anonymous 검증 반전, activeflowhandler 의 오류 반환 제거. 비동기 분기를 다루면 그 분기 1곳도 포함한다. 변이 위치는 새 테스트가 늘린 커버 구간 안으로 한정한다(기존 테스트가 이미 잡는 곳은 제외). 보충에 실제로 쓴 패키지(flowhandler, dbhandler, subscribehandler, stackmaphandler)마다 1곳씩 더한다. 변이마다 컴파일되는지와 이 PR 의 새 테스트가 실패하는지(기존 테스트가 아님)를 기록한다. 모두 새 테스트가 실패해야 한다.
- 변경한 패키지 전체를 `go test -race -count=3` 으로 실행해 기존 goroutine 테스트와의 상호작용을 확인하고, 커버 측정은 `-race` 없이 `-count=1` 로 재현한다. 새 테스트만 제외한 전후 커버리지 차이를 PR 본문에 적는다.
- 테스트 값은 합성값만 쓴다. 네트워크는 loopback httptest 만 쓰고 외부 호출은 없다.
- PR 파일 목록은 bin-flow-manager/ 안의 _test.go 와 이 문서뿐이다.

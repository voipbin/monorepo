# VOIP-1657 bin-agent-manager 단위 테스트 커버리지 80% 이상(목표 81% 이상) 설계

## 1. 티켓 검토 (이슈 분석)

판정: 유효. 기준선은 origin/main a03bccd8 에서 재실측했고 카드 수치와 같습니다.

- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (테스트 실패 0건, 로컬 vendor 디렉터리 없음).
  수기 합계는 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합산합니다.
- 수기 79.04% (2365/2992 statement). subscribehandler 의 goroutine 타이밍에 따라 측정마다 1 statement 가 달라져 2366(79.08%)이 나오기도 합니다. 보수적으로 2365 를 기준으로 80% 는 2394 이므로 최소 +29, 80.5% 는 2409 이므로 권장 +44, 81% 는 2424 이므로 +59 필요합니다.
- 패키지별 미커버 statement: cmd/agent-control 205, cmd/agent-manager 102, internal/config 8, models/agent 1, pkg/agenthandler 66, pkg/cachehandler 1, pkg/dbhandler 154, pkg/listenhandler 73, pkg/metricshandler 1, pkg/subscribehandler 15 또는 16.
- 선행 작업: VOIP-1647(cmd, cachehandler, dbhandler)과 VOIP-1652(agenthandler db/direct hash/event 오류 테스트, listenhandler 검증 테스트)가 이미 기준선에 반영되어 있습니다. 아래 대상은 기준선에서 실제로 미커버인 블록만 고른 것이며 기존 테스트가 있는 경로는 다시 쓰지 않습니다.
- 열린 PR: `gh pr list --state open` 에서 bin-agent-manager 를 바꾸는 PR 이 없음을 확인했습니다. a03bccd8 이후 origin/main 에서 이 서비스의 변경도 없습니다.
- 도달 가능량(시그니처 변경 없이 mock 주입으로 도달): pkg/agenthandler 약 55 (Create, Delete, Login, Update*, UpdateAddresses 검증, PasswordReset 후속 오류, Count, Reserve*), pkg/listenhandler 약 25 (count, login, direct hash 의 입력과 오류 응답, 알 수 없는 URI, Run), pkg/subscribehandler 약 12. 합계 약 90 이며 필요량 +44 를 넘습니다.
- 도달 불가로 분류해 제외: json.Marshal 실패 분기(listenhandler 약 30), crypto/rand 실패(agenthandler 2곳), errorResponse 의 ToResponse 실패 분기, cmd/agent-manager 의 main 과 기동(외부 연결). 분모에서 빼지 않고 미커버로 남깁니다.
- pkg/dbhandler(미커버 154)는 이 계획의 1차 대상이 아닙니다. 위 handler 3개 패키지만으로 목표를 넘기고 dbhandler 는 sqlmock 이 필요해 비용이 크기 때문이며, 목표 미달 시에만 추가합니다.
- crypto/rand 실패 위치: pkg/agenthandler/agent.go 의 PasswordForgot 과 pkg/agenthandler/event.go 의 EventCustomerCreated.
- 범위 밖 사항(제품 코드 변경, cmd 리팩터링)은 하지 않습니다.

## 2. 계획 (verify 항목, 테스트 코드는 아직 없음)

컨벤션: `Test_` 이름, gomock 컨트롤러 이름 `mc` 와 `defer mc.Finish()`, testify 금지(`reflect.DeepEqual` + `t.Errorf`), table-driven, 외부 상태와 시간에 의존하지 않음. 핸들러는 private struct 직접 인스턴스화로 만들고(Create, Login, PasswordReset 모두 utilhandler mock 으로 EmailIsValid/HashCheckPassword/HashGenerate 결과를 고정함), 생성자 자체가 대상인 NewAgentHandler/NewListenHandler 만 예외. 테스트 UUID 는 하드코딩한 실제형 값을 쓰고 uuid.Nil 이나 NewV4 는 쓰지 않으며 이메일은 example.com 만 씀.
모든 테스트는 mock 만 쓰고 네트워크, 시간 대기, 전역 상태 변경이 없습니다. 오류 전파 테스트는 오류 메시지 또는 typed 오류의 Reason 을 단언합니다.

1. pkg/agenthandler (신규 테스트 파일)
   - Create: typed 오류는 errors.As 로 *cerrors.VoipbinError 를 꺼내 Reason 을 비교. 결제 한도 조회 RPC 실패는 Reason `BILLING_CHECK_FAILED`, 한도 초과는 `RESOURCE_LIMIT_EXCEEDED`, 잘못된 username 은 `INVALID_USERNAME_FORMAT`, 같은 username 존재는 `AGENT_ALREADY_EXISTS`, 존재 확인 중 그 외 DB 오류는 `could not check agent existence`, dbCreate 실패는 `could not create an agent`. 앞 단계 실패 시 뒤 단계 mock 이 호출되지 않음을 expectation 부재로 확인.
   - Delete: isOnlyAdmin 통과 후 dbDelete 실패는 두 단계 래핑을 모두 거친 전체 오류 문자열로 단언(한쪽 래핑이 사라지면 실패).
   - Login: 성공 시 agent 반환, 사용자 없음과 잘못된 비밀번호는 전체 문자열(`could not logged in: no agent info`, `could not logged in: wrong password`)로 단언하고 nil 반환.
   - UpdateBasicInfo, UpdatePassword, UpdatePermission(Raw 단계 실패 래핑), UpdatePermissionRaw, UpdateTagIDs, UpdateAddresses: db 단계 실패가 각 `could not ...` 메시지로 래핑됨.
   - UpdateAddresses 의 조회(dbGet) 실패는 `could not get the agent` 로 래핑. UpdateAddresses 검증: 등록 확인 RPC 실패, 다른 customer 의 extension(오류 메시지 `wrong customer info` 를 비교), 빈 SIP target(`invalid target`. 빈 tel 은 정규화 단계에서 먼저 거절되어 기존 케이스가 다룸), 주소 조회 중 NotFound 가 아닌 오류.
   - PasswordReset: (가) 토큰 삭제 실패이고 agent 조회 성공이면 nil 을 반환하고 PublishEvent 가 1회 호출됨. (나) agent 조회 실패이면 nil 을 반환하고 PublishEvent 가 호출되지 않음.
   - 전역 metrics 카운터 값은 단언하지 않음. Create 의 `INVALID_ADDRESS_TARGET` 은 기존 테스트가 다루므로 쓰지 않음.
   - CountByCustomerID, Reserve, ReserveRelease, ReserveSweep 의 db 오류 래핑. NewAgentHandler 반환값을 *agentHandler 로 단언해 reqHandler, db, notifyHandler, cache 필드가 입력 mock 과 `==` 로 같고 utilHandler 가 nil 이 아님을 확인.
2. pkg/listenhandler (기존 test 파일에 케이스를 추가하지 않고 새 파일을 만듦)
   - count_by_customer: 잘못된 JSON 400, 핸들러가 일반 error 를 반환하면 errorResponse 변환으로 500. login: 잘못된 JSON 400, URI 항목 부족 400(processRequest 의 정규식 때문에 도달 불가이므로 `/` 없는 URI 를 processV1Login 에 직접 전달), 로그인 실패 400. direct hash regenerate: URI 항목 부족 400(processRequest 로는 도달 불가이므로 `/v1/agents` 를 processV1AgentsIDDirectHashRegenerate 에 직접 전달), 성공 200 과 본문(핸들러 반환 agent 의 id).
   - processRequest: 알 수 없는 URI 는 404 와 nil 오류. 제어문자가 든 query(`/v1/agents?page_size=` 뒤에 0x7f)는 정규식에는 맞지만 url.Parse 가 실패하므로 processV1AgentsGet 이 오류를 반환하고 processRequest 가 400 과 nil 오류로 바꿈(핸들러 mock 미호출). 핸들러가 typed 오류나 ErrNotFound 를 error 로 반환하는 경로는 현재 process* 가 만들지 않아 도달할 수 없으므로 쓰지 않음(errorResponse 변환 자체는 기존 테스트가 다룸).
   - NewListenHandler: 반환된 값을 구체 타입으로 단언해 sockHandler 와 agentHandler 필드가 입력 mock 과 같음을 `==` 로 확인(mock 순환 구조 때문에 DeepEqual 은 쓰지 않음). Run: QueueCreate 실패 시 오류 반환과 ConsumeRPC 미호출, 성공 시 ConsumeRPC 가 큐 이름과 함께 호출됨, ConsumeRPC 가 오류를 반환해도 Run 은 nil(각각 Do 로 호출 여부를 채널로 동기화하고 select 와 time.After 로 상한을 둠. Sleep 은 쓰지 않음).
3. pkg/subscribehandler (새 파일. 새 테스트 파일 이름은 소스 기준으로 agenthandler 는 `agent_error_wrap_test.go`, `reserve_count_error_test.go`, `main_new_test.go`, listenhandler 는 `v1_count_login_error_test.go`(direct hash 포함), `main_run_test.go`, subscribehandler 는 `event_error_test.go`, `main_run_error_test.go` 로 하고 충돌 시 접미사를 바꿈)
   - callmanager(groupcall 이벤트, 오류 문구는 코드의 문자열을 그대로 사용)와 customer_manager(created, deleted)의 이벤트 처리 3곳: agentHandler 의 Event* 가 오류를 반환하면 `Could not handle the ... event.` 로 래핑되어 반환됨(기존 additional 테스트는 JSON 오류와 성공만 다루므로 중복 아님). 잘못된 JSON 케이스는 이미 있어 쓰지 않음.
   - Run: QueueCreate 실패 오류 반환, 구 fanout QueueUnbind 실패는 로그만 남기고 Run 이 nil 을 반환(기존 테스트는 bind 실패만 다루므로 중복 아님), ConsumeMessage 가 오류를 반환해도 Run 은 nil(goroutine 내부 로그, 호출 여부를 채널로 동기화).
   - processEventRun: 이벤트를 넘기면 nil 반환과 함께 processEvent 로 위임되어 agentHandler mock 이 정확히 1회 호출됨(채널 동기화). processEvent 는 goroutine 없이 직접 호출: 알 수 없는 이벤트는 agentHandler mock 을 호출하지 않음, 핸들러가 오류를 반환해도 mock 이 Times(1) 로만 호출됨(반환값이 없어 호출 횟수가 유일한 단언이며 오류 로그 분기를 지나가는 것이 목적).
4. 목표: 수기 합계 81% 이상(최소 +59 statement). 1절의 보수적 추정(약 90)보다 미커버 블록을 줄별로 대조한 산정이 더 크며 이쪽을 기준으로 삼습니다. 산정은 agenthandler 약 62, listenhandler 약 37, subscribehandler 약 15 이며 goroutine 내부 로그 statement(1~3개)는 측정마다 흔들릴 수 있어 목표 계산에서 뺍니다. 미달 시 pkg/dbhandler 의 sqlmock 오류 경로를 추가합니다.
5. 변이 검사: scratch 복사본에서 아래를 하나씩 적용하고, 각 변이가 컴파일되며 변이 상태에서 기존 테스트는 통과하고 새 테스트만 실패함을 실행으로 확인합니다. 복원 후 모두 통과해야 합니다. 컴파일이 깨지면 같은 의도를 유지하는 형태로 바꿔 기록합니다. 규칙: 새 단언마다 변이를 하나씩 대응시키되 변이 지점은 기존 테스트가 닿지 않는 줄이어야 하며(예: count 의 500 은 v1_count.go 의 `return errorResponse(err), nil` 을 `simpleResponse(400)` 으로 바꿈), 대응하는 변이가 없는 단언은 이 절의 제외 목록에 사유와 함께 적습니다. 아래 목록은 대표 변이이며 구현 단계에서 이 규칙으로 전수 확장해 PR 본문 밖의 작업 기록에 결과 표를 남깁니다. Create 는 `RESOURCE_LIMIT_EXCEEDED`, `AGENT_ALREADY_EXISTS`, `could not check agent existence`, `could not create an agent` 문구를 각각, UpdateAddresses 는 `could not get extension info` 포함, Login 은 내부 `no agent info` 와 `wrong password` 를 각각, UpdatePermission 의 바깥 래핑(UpdatePermission 함수 안)과 UpdatePermissionRaw 의 래핑(Raw 함수 안)을 위치로 구분해 변이합니다.
   - agenthandler: Delete 의 바깥 래핑(agent.go 의 Delete 함수 안, deleteForce 안쪽 아님)을 `return nil, err` 로, Create 의 `BILLING_CHECK_FAILED` Reason 문구 교체, Login 의 래핑 문구 변경, Reserve 의 래핑 문구 변경, NewAgentHandler 의 cache 필드 할당 삭제, Create 의 `INVALID_USERNAME_FORMAT` Reason 교체, UpdateAddresses 의 `wrong customer info` 문구 변경. PasswordReset 은 (가) 토큰 삭제 실패 분기에 `return err` 추가, (나) 조회 실패 분기의 `return nil` 을 `return err` 로 변경.
   - listenhandler: processRequest 의 default 404 를 500 으로, processRequest 의 err != nil 분기의 400 을 500 으로, processV1Login 의 로그인 실패 400 을 500 으로, direct hash 성공 응답의 Data 를 `data` 를 계속 사용하는 형태(`append(data, 'x')`)로 오염, listenhandler.Run 의 QueueCreate 오류를 `if err := ...; err != nil { _ = fmt.Sprint(err) }` 로 삼킴.
   - subscribehandler: 이벤트 오류 래핑 문구 변경, subscribeHandler.Run 의 QueueCreate 오류를 삼킴(위와 같은 형태), processEventRun 의 `go` 제거.
   - processEventRun 테스트 설계: Do 안에서 release 채널로 막히게 하고, processEventRun 을 별도 goroutine 에서 호출해 done 채널과 time.After(5초 상한, 실패 경로에서만 소모) 로 select 하여 processEventRun 이 release 전에 반환함을 단언하며, mock 의 Do 진입과 반환도 채널로 기다린 뒤 테스트를 끝내고, defer 로 release 를 닫아 막힌 goroutine 이 남지 않게 함(제품 코드에 완료 신호가 없어 processEvent 꼬리의 종료 시점은 관측하지 않으며, mock 호출이 이미 기록된 뒤라 무해함). `go` 제거 변이에서는 호출이 막혀 상한 도달로 실패함.
   - 응답 코드와 래핑 문구 변이를 모든 새 케이스군에 하나씩 둡니다: count_by_customer 의 400 과 500 각각, login 과 direct hash 의 URI 부족 400 을 500 으로, UpdateBasicInfo/UpdatePassword/UpdateTagIDs/UpdateAddresses(`could not update the addresses`, `invalid target`, `could not get agent info of the address`, `could not get the agent`)/CountByCustomerID/ReserveRelease/ReserveSweep 의 래핑 문구 변경, UpdatePermission 과 UpdatePermissionRaw 는 한쪽 래핑 문구만 바꿔 각각 자기 경로 테스트(UpdatePermission 경유, Raw 직접 호출)에서 구분되어 실패하게 단언(두 단계 전체 문자열 비교).
   - Run 을 호출하는 모든 케이스(성공과 unbind 실패 포함)는 Run 이 띄우는 goroutine 의 ConsumeRPC/ConsumeMessage 기대를 선언하고 Do 에서 채널로 동기화한 뒤 끝내며, 기존 `Test_Run_BindsTopicExchangeBeforeReturning` 와 같은 채널 동기화 방식을 새 인라인 mock 으로 씀. 채널 대기의 시간 초과는 t.Fatalf 가 아니라 t.Errorf 후 return.
   - Run 의 ConsumeRPC/ConsumeMessage 오류 케이스(Run 이 nil 을 반환하는지만 단언)와 NewListenHandler 필드 단언은 호출 도달 확인용이며 변이 검사에서 제외합니다.
   - processEvent 의 오류 로그 분기 테스트는 로그 분기만 지나가는 용도이므로 변이 검사 대상에서 제외합니다.
6. 검증 절차: 먼저 `git fetch origin main` 을 하고 서비스 디렉터리에서 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 전부와 새 테스트 패키지에 대한 `go test -race -count=3`, PR 전 `go generate` 후 `git status` 로 PR 밖 파일(mock_*.go 등)이 바뀌지 않았음을 확인하고, 마지막에 같은 측정 명령으로 수기 합계가 81% 이상임을 재확인하며, `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"` 와 `git log --oneline HEAD..origin/main` 충돌 점검(충돌 시 갱신 후 전체 재검증)을 실행하고 커밋 후 저장소 루트(worktree)에서 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 를 실행하고 go.mod/go.sum 변경이 없음을 확인합니다. 새 Test_ 함수 이름이 같은 package 의 기존 함수와 충돌하지 않는지 grep 으로 확인합니다(subscribehandler 에는 `Test_processEventCM*` 가 이미 있음). 테스트 값은 example.com 이나 임의의 UUID 같은 명백한 합성값만 씁니다.
7. PR 의 파일은 bin-agent-manager/ 안의 `_test.go` 와 이 문서뿐입니다. 설계와 PR 은 결함 위치를 적지 않습니다. PR 본문에 패키지별 전후 표와 측정 명령을 적습니다.
8. 이 서비스에서 VOIP-1416 이 적은 패키지와 겹치면 PR 본문에 명시만 합니다.

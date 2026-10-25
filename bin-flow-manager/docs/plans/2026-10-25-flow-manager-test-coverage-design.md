# bin-flow-manager 단위 테스트 커버리지 상향 (VOIP-1651)

## 1. 티켓 검토

판정: 유효.

- 기준선 재측정(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 4252 statement 중 2872 커버, 67.54%. 티켓 수치와 일치. 테스트 실패 0건.
- 수기 판정: `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록 중복은 최대 count 로 합침.
- 최근 변경 이력(`git log -- bin-flow-manager`)과 열린/머지된 PR 조회 결과 동일 목적의 PR 은 없음. 제품 코드 변경 없이 테스트만 추가하는 범위는 유효.
- 틀린 주장: 없음. cmd/ 는 수기 합계에 포함되며 `cmd/flow-control` 은 전체 358 중 미커버 358, `cmd/flow-manager` 는 전체 134 중 미커버 133 으로 거의 0% 임(아래 3절).

## 2. 기준선 (수기 코드)

| 패키지 | statement | 커버 | 미커버 |
|---|---|---|---|
| pkg/activeflowhandler | 1681 | 1301 | 380 |
| cmd/flow-control | 358 | 0 | 358 |
| pkg/listenhandler | 774 | 620 | 154 |
| cmd/flow-manager | 134 | 1 | 133 |
| pkg/cachehandler | 109 | 0 | 109 |
| pkg/dbhandler | 328 | 267 | 61 |
| pkg/flowhandler | 250 | 199 | 51 |
| pkg/variablehandler | 147 | 100 | 47 |
| pkg/actionhandler | 78 | 48 | 30 |
| pkg/stackmaphandler | 138 | 113 | 25 |
| pkg/subscribehandler | 59 | 43 | 16 |

목표 70% 에는 2977-2872 = 105 statement, 여유를 둔 71% 에는 약 147 statement 추가 커버가 필요하다. 계획은 71% 를 기준으로 한다.

## 3. 구조 분석

- `executeAction` 의 디스패치 switch 는 action 종류별 오류 분기가 대부분 미커버(execute.go 111 statement). option 을 잘못된 타입으로 주면 option 을 unmarshal 하는 핸들러는 오류가 나므로 그 경로로 분기를 유도하고, option unmarshal 이 없는 핸들러(fetch, stop 등)는 mock 이 오류를 반환하도록 해서 유도한다. 일부 action 의 분기는 테스트 입력으로 도달할 수 없어 목표 계산에서 제외한다.
- `pkg/cachehandler` 는 redis 클라이언트와 redsync 락(`*redsync.Redsync` 구체 타입)을 직접 쓰며 테스트가 없음. 직렬화, 락 획득/해제/미획득 경로는 인메모리 redis(miniredis)로 검증한다. bin-flow-manager 의 go.mod/go.sum 에는 miniredis 가 없으므로 테스트 전용 의존성으로 추가한다(테스트에서만 import, 같은 v2.36.1 이 다른 서비스에 존재, `go mod tidy` 후 추가 diff 없음은 miniredis 추가 이후 기준이며 `go.mod`/`go.sum` 변경 자체는 발생한다. 테스트 파일에서만 import 하므로 제품 코드 변경 금지와 충돌하지 않는다). 락 경쟁 실패 테스트는 `NewMutex` 를 옵션 없이 호출하는 제품 코드 때문에 기본 재시도(32회)를 기다려야 하므로 약 4초가 걸린다.
- `cmd/flow-control` 의 run* 함수는 실제 DB 연결(initHandler)을 요구하므로 단위 테스트 대상에서 제외. 순수 함수(`parseActions`, `resolveUUID`, `printJSON`, 합계 약 21 statement)만 대상이며 기여가 작으므로 여유 시에만 한다.
- `cmd/flow-manager` 의 main/runDaemon/runServices 는 외부 연결 의존이라 제외. 주입형 함수(runListen, runSubscribe, initCache)는 여유 시에만 대상.

## 4. 계획

추가 테스트 (제품 코드 변경 없음, 모두 테스트 파일):

1. `pkg/cachehandler/handler_test.go`: 테스트 전용 miniredis v2.36.1(다른 서비스와 동일 버전)을 쓴다. `go.mod`/`go.sum` 변경은 miniredis 와 그 간접 의존성(gopher-lua 등)에 한정한다. miniredis + redsync v4.15.0 + go-redis v9.17.2 조합에서 락 획득, 이미 잡힌 락의 재획득 실패, Unlock 후 재획득이 동작한다.
   - Flow/Activeflow/Variable 의 set-get 왕복, 미존재 키 오류, 손상된 JSON 오류, 삭제.
   - 락: 획득 성공, 두 번째 획득 실패(경쟁 실패 테스트는 기본 재시도 32회로 약 4초가 걸리며 제품 코드를 바꿀 수 없으므로 1건으로 제한), 락 획득 후 데이터 없음이면 락이 풀려 다시 획득 가능, `ActiveflowReleaseLock` 정상 해제 후 같은 id 를 한 번 더 해제하면 `no mutex` 오류(`delete(h.mapMutex, ...)` 변이를 잡는 단언). 락 테스트는 순차 호출로 검증한다.
   - `NewHandler` + `Connect` 성공, 서버 중지 후 `Connect` 실패, 서버 중지 후 Set/Del 계열의 오류 반환.
   - 시크릿 탐지기 오탐 회피: 비밀번호는 빈 문자열, 주소는 `miniredis.Addr()` 런타임 값만 쓰고 리터럴 IP/포트는 쓰지 않는다.
2. `pkg/activeflowhandler/execute_error_test.go`(신규, `executeAction` 표 기반):
   - 전제: `verifyActionType` 통과가 필요하다(`MapRequiredMediasByType`, `models/action/action.go:329-374`). 종별 `ReferenceType`: RealTime 필요 종(conference_join, connect, queue_join, transcribe_start, condition_call_digits, condition_call_status)은 `ReferenceTypeCall`(transcribe_start 는 핸들러도 call 을 요구, `actionhandle.go:710`). NonRealTime 필요 종(ai_summary, ai_task)은 call 이면 `ActionNext` 로 건너뛰므로 conversation 계열 reference type 을 쓴다. 나머지(ai_talk, branch, call, goto, message_send, condition_datetime, condition_variable, fetch, fetch_flow, stop, transcribe_recording, variable_set, webhook_send, conversation_send, email_send)는 `MediaTypeNone` 이라 무관하다. 각 케이스는 `verifyActionType` 통과를 먼저 단언해 건너뜀으로 인한 거짓 통과를 막는다.
   - 유도 방식 A (option 타입 오류): `Option` 은 `map[string]any` 라 marshal 은 항상 성공하므로 각 option 구조체의 실제 필드에 타입이 어긋나는 값을 준다(uuid 필드에 숫자, int 필드에 문자열). 대상: ai_summary, ai_talk, ai_task, branch, call, condition_call_digits, condition_call_status, condition_datetime, condition_variable, conference_join, connect, goto, message_send, queue_join → 오류 반환, 결과 nil. 삼킴 분기(호출 기대가 없는 gomock 을 주입해 정상 경로 진입 시 mock 위반으로 실패하게 한다): transcribe_recording, transcribe_start, variable_set, webhook_send, conversation_send, email_send → 오류 없이 다음 action 으로 진행.
   - 유도 방식 B (mock 오류): fetch 는 `ActionFetchGet` mock 오류, fetch_flow 는 해당 핸들러가 호출하는 요청 mock 오류, stop 은 `PushStack` 경로의 stackmap/db mock 오류로 유도한다.
   - 제외: 테스트 입력으로 도달할 수 없는 분기.
   - 예상 이득(블록 기준, 방식 B 의 fetch/fetch_flow/stop 은 전파에 포함해 이중 계산하지 않음): 전파(로그 포함) 10종 x 3 = 30, condition 4종 x 2 = 8, 삼킴 6종 x 3 = 18, 방식 B 3종 x 3 = 9, 합계 65 에 verifyActionType false 분기(2)와 empty 분기(2)를 더해 약 69(default 분기는 이미 커버됨).
   - 합계 예상: cachehandler 약 100 + executeAction 약 69 = 약 169 (71% 기준 147 대비 여유 약 22, 도달 불가는 `setSerialize` 의 `json.Marshal` 오류 분기 1 statement). 부족하면 3, 4항으로 보충.
3. 보충 후보(1,2항 이후 측정해서 71% 미달일 때만, 우선순위 순): `pkg/variablehandler/substitute.go`(미커버 27), `pkg/actionhandler/action.go`(미커버 27), `pkg/listenhandler/v1_activeflows.go`(미커버 57) 의 오류 경로와 입력 검증, `pkg/dbhandler`/`pkg/flowhandler` 의 오류 경로. 각 함수의 미커버 블록을 읽고 mock 오류와 경계값 중심으로 추가한다.
4. 마지막 보충 후보: `cmd/flow-control` 순수 함수(`parseActions`, `resolveUUID`, `printJSON`) 약 21 statement. `cmd/flow-manager` 의 `runListen`, `runSubscribe`, `initCache` 는 주입형이라 여유 시에만.

## 5. 검증 항목 (verify)

- 수기 합계 71% 이상, 기존 테스트 전부 통과(`-race` 포함 1회 확인), `golangci-lint` 통과.
- `git diff --stat origin/main...HEAD` 가 bin-flow-manager/ 안의 파일만 포함하고 비테스트 .go 변경이 없다. `go.mod`/`go.sum` 변경은 miniredis 와 간접 의존성 추가분만이며, 테스트 파일에서만 import 하므로 제품 코드 변경 금지와 충돌하지 않는다. 추가 후 `go mod tidy` 를 다시 돌려 추가 diff 가 없어야 하고 `go mod vendor` 가 통과해야 한다.
- 변이 확인(별도 복사본에서): (a) `execute.go` 의 한 케이스 `return nil, errHandle` 를 `return &action.ActionNext, nil` 로 바꾸면 해당 표 케이스가 실패, (b) `ActiveflowReleaseLock` 의 `defer delete(h.mapMutex, ...)` 를 제거하면 두 번째 해제 단언이 실패, (c) `ActiveflowGetWithLock` 의 getSerialize 실패 분기에서 `mutex.Unlock()` 를 제거하면 락 재획득 단언이 실패(변이 코드에서는 재시도 소진까지 수 초 막힌 뒤 락 오류가 반환됨). (b) 는 오류 존재가 아니라 정확한 문자열 `no mutex` 를 단언해야 변이를 잡는다. (a) 는 결과 nil 과 오류를 함께 단언한다.
- 자기비교식 테스트와 빈 호출 테스트 없음: 모든 테스트는 mock 호출 기대와 반환/오류를 명시적으로 단언한다. 테스트 값은 합성값만 쓴다.
- CI 게이트 스크립트(`scripts/check-test-conventions.sh`: Test_ 접두 이름, gomock controller 이름 mc 등)를 로컬에서 실행한다.

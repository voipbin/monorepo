# VOIP-1647 bin-queue-manager 단위 테스트 커버리지 60% 계획

## 1. 티켓 검토

판정: 유효. 기준선은 origin/main d68e695b5 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했으며 테스트 실패 0건, 수기 합계 54.43% (1835/3371 statement), 원본 47.51% (1835/3862)로 카드 수치(54.4%, 3371개, 47.5%)와 일치합니다. 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록의 중복은 최대 count 로 합쳐 계산했습니다(scratch 스크립트).

60% 에는 188개, 권장 61% 에는 222개 statement 를 더 커버해야 합니다.

패키지별 기준선(수기, 전):

| 패키지 | 전 | 미커버/전체 |
|---|---|---|
| cmd/queue-control | 0.0 | 528/528 |
| cmd/queue-manager | 0.9 | 113/114 |
| internal/config | 0.0 | 76/76 |
| pkg/cachehandler | 0.0 | 67/67 |
| pkg/dbhandler | 78.0 | 133/604 |
| pkg/listenhandler | 67.1 | 214/651 |
| pkg/queuecallhandler | 71.8 | 224/793 |
| pkg/queuehandler | 66.1 | 131/387 |
| pkg/subscribehandler | 59.3 | 48/118 |
| models/queue, models/queuecall | 93.8, 94.1 | 각 1 |

구조 분석(제품 코드 변경 없음을 전제로 한 제외 판단):
- 제외: cmd/queue-manager 의 main, runDaemon, initSignal, initProm, run, runReconcile, runSubscribe, runListen(DB, Redis, RabbitMQ 연결과 프로세스 기동에 묶여 있음, 미커버 113). cmd/queue-control 의 initHandler, initCache, initQueueHandler, initQueuecallHandler, main, 모든 runXxx(DB와 Redis 를 직접 엶). pkg/cachehandler 구체 구현(`*redis.Client` 직접 의존, go.mod 에 miniredis 없음, 로컬 가짜 서버를 세우는 비용 대비 효과가 낮아 이번 범위에서 제외).
- 후보에 포함: internal/config(cobra, viper, sync.Once 만 사용하며 외부 연결 없음, 76), cmd/queue-control 의 순수 함수(resolveUUID, printJSON, parseUUIDs, cmdXxx 명령 빌더와 플래그 정의). cmd/queue-manager/startup_log_test.go 가 이미 logStartup 을 다룹니다.
- 제외 대상의 미커버(cmd/queue-manager 113, cachehandler 67, queue-control 중 외부 연결 함수)를 모두 빼도 pkg 와 config 쪽 미커버가 750 statement 이상이므로 60% 달성은 산술적으로 가능합니다(상한 약 77%).
- queuehandler, queuecallhandler, listenhandler, subscribehandler 는 gomock 인터페이스(dbhandler, requesthandler, notifyhandler, queuehandler, queuecallhandler, cachehandler)로 주입됩니다. dbhandler 는 `*sql.DB` 를 받으며 기존 테스트가 sqlite(go-sqlite3)와 go-sqlmock 을 이미 씁니다. 이 패키지들은 제품 코드 변경 없이 기존 방식으로 테스트할 수 있습니다.
- 측정 방식: 60% 판정은 go test 총계가 아니라 같은 블록 중복을 최대 count 로 합치는 scratch 집계 스크립트(agg.py)로 계산한 수기 합계입니다. 완료 시 같은 스크립트로 재계산하고 `git diff` 로 go.mod, go.sum 이 바뀌지 않았음도 확인합니다(`GOFLAGS=-mod=mod` 사용).

## 2. 계획

범위는 `_test.go` 추가뿐입니다. 제품 코드는 바꾸지 않습니다.

대상과 파일별 목표(미커버 statement 중 채울 양의 하한, 합계는 필요량 222 의 2배 이상):

| 파일 | 현 미커버 | 목표 채움 | 내용 |
|---|---|---|---|
| pkg/queuehandler/db.go | 84 | 70 | AddWaitQueueCallID, RemoveServiceQueuecallID, RemoveWaitQueuecallID, AddServiceQueuecallID 의 정상, db 쓰기 오류, 재조회 오류. List/Get/Update 계열 오류 |
| pkg/queuehandler create.go, direct_hash.go, agent.go, queue.go | 40 | 25 | 오류 전파, 경계값 |
| pkg/listenhandler/v1_queuecalls.go | 106 | 85 | TimeoutWait, TimeoutService, Execute(0%)의 짧은 URI 400, 잘못된 JSON 400, 핸들러 오류 errorResponse, 정상. 기존 Get/List/Delete 류의 오류 경로. customer_id 등 필터 파싱이 있으면 불일치 입력 |
| pkg/listenhandler/v1_queues.go, main.go, v1_count.go, v1_services.go, v1_queues_direct_hash.go | 102 | 60 | 같은 유형, processRequest 의 라우팅과 미등록 URI |
| pkg/queuecallhandler/db.go | 84 | 55 | db 오류, notify 호출 여부, 비동기 경로 포함 |
| pkg/queuecallhandler/health.go | 41 | 40 | 분기 전부 |
| pkg/queuecallhandler execute.go, event.go, kick.go, service.go, timeout.go, reconcile.go | 96 | 55 | 오류 경로와 상태 분기 |
| pkg/dbhandler (queuecall.go, queue.go) | 129 | 55 | QueuecallSetStatusWaitingIfInitiating 와 queuecallSetStatusWaitingIf(조건 일치/불일치), QueueCountByCustomerID, 기존 sqlite/sqlmock 방식의 쿼리 오류 |
| pkg/subscribehandler | 48 | 35 | processEventAMAgentStatusUpdated(available 만 위임, 그 외 no-op, JSON 오류), processEventCMCallHangup, processEventRun 분기와 JSON 오류 |
| internal/config | 76 | 60 | Bootstrap 의 기본값과 환경변수 우선, LoadGlobalConfig/Get (sync.Once 공유 상태 때문에 한 테스트 함수 안에서 viper 를 초기화하고 순서 의존 없이 작성) |
| cmd/queue-control 순수 함수 | 약 40 | 20 | resolveUUID, parseUUIDs, printJSON, 명령 빌더 |
| 합계 | | 약 560 | 필요량 222 대비 2배 이상 여유 |

단계별 점검: 패키지 하나를 끝낼 때마다 agg.py 로 재측정해 카드 scratch 에 기록합니다. 61% 미달이면 위 표의 목표 미달분을 메우고, 그래도 모자라면 reconcile.go, variable.go, 그 밖의 한 줄짜리 분기를 예비 후보로 추가합니다. 60% 달성이 불가능하다고 판단되면 낮추지 않고 default 에 block 합니다.

비동기 경로: queuecallhandler/db.go 의 `go func()`(약 475, 578행)에서 호출되는 AddWaitQueueCallID 와 매칭 호출은 gomock 의 `Do`/`DoAndReturn` 안에서 채널로 신호를 보내고 테스트가 `select` 로 timeout 을 두고 기다리는 방식으로 동기화합니다. 호출이 일어나지 않는 음성 케이스는 고루틴이 끝날 때까지 기다리는 신호가 없으므로 고정 sleep 을 쓰지 않고 mock 에 기대를 두지 않는 방식으로 구성하거나 해당 경로가 동기인 분기만 대상으로 합니다. 동시성 관련 테스트는 `go test -race -count=20` 으로 반복해 불안정성이 없음을 확인합니다. 입력 검증은 listenhandler 에서 URI 길이, JSON 파싱, UUID 파싱, 필터 파싱 경로를 대상으로 하며 이 서비스에는 권한 검사 코드가 없습니다(권한은 api-manager 에서 처리).

원칙: 빈 호출 금지. 각 테스트는 mock 의 호출 인자(gomock 엄격 매칭)와 반환 상태코드, 응답 본문, 오류 전파를 단언합니다. 오류 테스트는 "후속 호출이 일어나지 않음"을 mock 에 호출 기대를 두지 않는 것으로 단언합니다. 자기비교식 테스트는 쓰지 않습니다.

verify 항목:
1. 수기 합계 61% 이상(scratch 의 agg.py), 패키지별 전후 표를 PR 본문에 기재. 완료 시 go.mod, go.sum 무변경 확인.
2. 루트 CLAUDE.md 검증 순서대로 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 를 실행하고 go.mod, go.sum, 생성 파일에 변경이 없음을 확인. `git fetch origin main` 후 `bash scripts/check-test-conventions.sh`(추가된 줄만 검사, Test_ 명명, gomock 컨트롤러 이름 mc) 통과. sqlite 는 기존 dbhandler 테스트가 이미 CI 에서 쓰는 방식을 그대로 따름.
3. 동시성/반복: `go test -race -count=20` 을 새로 만든 비동기 테스트와 queuecallhandler 패키지에 실행.
4. 변이 검사(scratch 복사본, 제품 코드만 변경, 결과는 PR 본문에 요약). 대상과 변이 종류를 미리 정함: (a) queuehandler AddWaitQueueCallID 의 db 오류 시 `return nil, errAdd` 제거, (b) listenhandler processV1QueuecallsIDExecutePost 의 `len(uriItems) < 5` 조건을 `< 4` 로 변경, (c) queuecallhandler HealthCheck 의 `retryCount > defaultHealthCheckMaxRetryCount` 를 `>=` 로 변경, (d) HealthCheck 의 종료 상태(Abandoned/Done) 조기 반환 제거, (e) subscribehandler processEventAMAgentStatusUpdated 의 `StatusAvailable` 비교를 반전, (f) dbhandler queuecallSetStatusWaitingIf 의 조건 절 제거, (g) config 환경변수 바인딩 키 하나 변경. 판정: 모든 변이가 새 테스트에서 실패해야 하며 살아남는 변이는 테스트를 보강해 다시 실행.
5. `git diff --stat origin/main...HEAD` 가 bin-queue-manager/ 안의 `_test.go` 와 이 문서뿐임을 확인. 테스트 값은 합성 UUID 와 번호만 사용하고 외부 호출과 네트워크 없음. 빈 호출과 자기비교 테스트가 없는지 코드 리뷰 단계에서 체크리스트로 확인.

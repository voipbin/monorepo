# VOIP-1652 bin-queue-manager 단위 테스트 커버리지 75% 계획

## 1. 티켓 검토

판정: 유효. origin/main accc03e00 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했으며 테스트 실패 0건, 수기 합계 72.06% (2429/3371 statement), 원본 62.89% (2429/3862)로 카드 수치와 일치합니다. 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록의 중복은 최대 count 로 합쳐 계산했습니다(scratch 집계 스크립트). 75% 에는 약 99개, 권장 76% 에는 약 133개 statement 를 더 커버해야 합니다. 동일 서비스의 선행 PR(VOIP-1647 60%, VOIP-1651 70%)은 머지되었고 열린 중복 PR 은 없습니다.

패키지별 기준선(수기, 전):

| 패키지 | 전 | 미커버/전체 |
|---|---|---|
| cmd/queue-control | 0.0 | 528/528 |
| cmd/queue-manager | 0.9 | 113/114 |
| internal/config | 97.4 | 2/76 |
| models/queue, models/queuecall | 93.8, 94.1 | 각 1 |
| pkg/cachehandler | 0.0 | 67/67 |
| pkg/dbhandler | 81.6 | 111/604 |
| pkg/listenhandler | 93.1 | 45/651 |
| pkg/queuecallhandler | 90.9 | 72/793 |
| pkg/queuehandler | 99.7 | 1/387 |
| pkg/subscribehandler | 99.2 | 1/118 |

구조 분석(제품 코드 변경 없음 전제):
- 제외: cmd/queue-control 중 DB 와 Redis 를 직접 여는 runXxx, initHandler 계열, main(순수 헬퍼 resolveUUID, parseUUIDs, printJSON, cmdXxx 빌더는 합계가 모자랄 때의 예비 후보), cmd/queue-manager 의 main, runDaemon, run(sockhandler 로 실제 RabbitMQ 연결), initSignal, initProm, runReconcile(전역 시그널, ListenAndServe, 고정 ticker 고루틴).
- 제외 후 남는 후보 미커버: cachehandler 67, dbhandler 111, queuecallhandler 72, listenhandler 45, runListen/runSubscribe 약 12. 합 약 307 로 필요량 133 대비 충분하여 75%(권장 76%) 달성은 산술적으로 가능합니다.
- cachehandler 는 `*redis.Client` 를 직접 쓰므로 miniredis v2.36.1(go-redis v8 을 쓰는 bin-ai-manager, bin-billing-manager, bin-conference-manager 등 main 의 다른 서비스가 이미 같은 버전을 사용. miniredis 는 서버 에뮬레이터라 클라이언트 버전과 무관)을 테스트 파일에서만 import 합니다. 이 서비스 go.mod 에는 없으므로 go.mod/go.sum 에 miniredis 와 그 간접 의존이 추가되는 diff 가 생기며, 이미 있는 의존의 버전이 올라가지 않아야 하고 `go mod tidy` 를 다시 돌려도 추가 diff 가 없어야 합니다. 비테스트 .go 는 바꾸지 않습니다.
- dbhandler 의 미커버는 거의 모두 ExecContext 오류, RowsAffected 오류, 캐시 갱신 오류 분기이며 기존 `newCtxTestHandler`(go-sqlmock)로 주입 가능합니다.
- 이 서비스에는 권한 검사 코드가 없습니다(권한은 api-manager). 입력 검증은 listenhandler 의 URI 길이, JSON 파싱, UUID/필터 파싱 경로입니다.

## 2. 계획

범위는 `_test.go` 추가와 go.mod/go.sum 의 miniredis 추가뿐입니다. 제품 코드는 바꾸지 않습니다.

| 대상 | 현 미커버 | 목표 채움 | 내용 |
|---|---|---|---|
| pkg/cachehandler | 67 | 55 | miniredis 로 NewHandler, Connect(성공, 서버 중지 후 오류), QueueSet/Get(저장된 TTL 24시간을 mr.TTL 로 확인. Get 결과 필드는 입력 변수와 비교하지 않고 리터럴 기대값과 비교), QueuecallSet 은 `queue:queuecall:<id>` 키의 값과 TTL 을 직접 읽어 단언하고 QueuecallGetByReferenceID 는 별도로 검증, QueuecallGet, QueuecallGetByReferenceID(리터럴 JSON 으로 키를 직접 써서 조회하고 필드를 리터럴과 비교), 없는 키 오류, 잘못된 JSON 값 오류, ReconcileLockAcquire(최초 true, 보유 중 false, `mr.FastForward` 로 TTL 만료 후 true, 서버 중지 시 오류), ReconcileLockRelease. setSerialize 의 json.Marshal 오류 분기는 도달 불가라 5~8 statement 는 남는 것으로 봅니다 |
| pkg/dbhandler | 111 | 60 | 현 미커버: queue.go 46(오류 분기 약 37 블록과 SQL 실행 분기), queuecall.go 61, main.go 4(NewHandler), 따라서 queue.go 에서 25, queuecall.go 35 이상을 채웁니다. 대상 함수(측정으로 확정. dbhandler.NewHandler 4 statement 도 채움에 포함): queue.go 의 QueueCountByCustomerID, QueueCreate, QueueGet, QueueList, QueueUpdate, QueueDelete, QueueAddWaitQueueCallID, QueueIncreaseTotalServicedCount, QueueIncreaseTotalAbandonedCount, QueueRemoveServiceQueueCall, QueueRemoveWaitQueueCall 와 캐시 헬퍼, queuecall.go 의 QueuecallCreate/Get/GetByReferenceID/List/ListOldestWaiting/ListConnectingStale/ListWaitingOldest/Update/Delete, QueuecallSetStatus{Connecting,Service,Abandoned,Done,Kicking,WaitingIfConnecting}, queuecallSetStatusWaitingIf 와 캐시 헬퍼. 변경 함수별 ExecContext 오류 전파(`errors.Is` 로 원인 확인, `ExpectationsWereMet` 로 후속 쿼리 없음 확인)와 캐시 미호출, RowsAffected 오류(`sqlmock.NewErrorResult`), 조회 쿼리 오류, 캐시 get 오류 시 DB 대체 경로. 표 주도, mock 엄격 매칭 |
| pkg/queuecallhandler | 72 | 45 | 측정 기준 미커버 블록: event.go(38, 65, 94, 121-141, 160-181), execute.go(37-121, 147, 175-184), service.go(46-119), kick.go(59, 104, 133), reconcile.go(46, 73, 99-124), health.go(50), timeout.go(37, 65), main.go NewQueuecallHandler(생성자). 오류 분기와 상태 분기 |
| pkg/listenhandler | 45 | 30 | 각 processV1 핸들러의 핸들러 오류 → errorResponse(500 등), 잘못된 JSON/UUID 400, direct_hash regenerate, count, services |
| cmd/queue-manager | 113 | 6 | runSubscribe(Run 오류는 호출자에게 반환됨, 성공 시 nil)와 runListen(성공 경로에서 ConsumeRPC 등 큐 이름 인자를 상수가 아닌 문자열 리터럴로 단언. Run 오류 경로는 테스트하지 않음). runSubscribe 의 큐 이름과 토픽 패턴도 리터럴로 단언. sockhandler/queuehandler/queuecallhandler gomock |
| 보조 | | 0 | queuehandler, subscribehandler, internal/config, models 의 1~2 statement 와 NewQueueHandler, dbhandler.NewHandler 생성자, cmd/queue-control 순수 헬퍼는 합계가 모자랄 때의 2차 후보 |
| 합계 | | 196 | 필요 133(2562/3371 = 76.0%). 여유 63 은 cachehandler 도달 불가 분기와 비동기 경로 제외분을 감안한 값이며 중간 측정에서 미달이면 2차 후보를 추가 |

동시성: listenhandler.Run 과 subscribehandler.Run 은 ConsumeRPC/ConsumeMessage 를 고루틴으로 실행하므로 mock 의 Do 에서 채널로 신호하고 `select` 와 timeout 으로 기다립니다. 고정 sleep 은 새 테스트에서 쓰지 않습니다. subscribehandler 의 `processEventRun` 은 `go h.processEvent(m)` 를 띄우므로 이 경로를 쓰는 테스트는 downstream mock 의 Do 에서 채널로 신호하고 테스트 종료 전에 반드시 기다리며, 필요하면 processEvent 를 직접 호출합니다. TTL 은 `mr.FastForward` 로만 만료시킵니다. cmd/queue-manager 의 runListen 과 runSubscribe 테스트도 내부 Run 이 ConsumeRPC/ConsumeMessage 를 고루틴으로 띄우므로 같은 규칙(해당 mock 의 Do 에서 채널로 신호, 종료 전 select 와 timeout 으로 대기)을 쓰고 `-race -count=20 -shuffle=on` 대상에 포함합니다. runSubscribe 는 Run 안에서 동기로 먼저 호출되는 토픽 생성과 큐 바인딩 호출의 순서와 인자도 단언합니다. 서버 중지 후 오류를 확인하는 케이스는 `mr.Close()` 뒤에 실행하며 고정 sleep 을 쓰지 않습니다. ReconcileLockAcquire 는 `context.WithTimeout` 을 걸고, ctx 인자가 없는 Connect 는 닫힌 포트의 연결 거부로 즉시 실패함을 이용하되 별도 고루틴에서 실행해 select 와 timeout 으로 대기합니다. listenhandler 의 HealthCheck 경로(`go ...HealthCheck`)와 queuecallhandler/db.go 의 고루틴 두 곳은 해당 mock 의 마지막 호출 Do 에서 채널로 신호하고 select 와 timeout 으로 기다리며 `-race -count=20 -shuffle=on` 대상에 포함합니다. db.go 는 대상 목록에 없으므로 이 파일의 비동기 경로는 이번 범위에서 건드리지 않습니다. 비동기 경로를 건드리는 테스트는 `go test -race -count=20 -shuffle=on` 으로 반복해 불안정성이 없음을 확인합니다.

원칙: 빈 호출과 자기비교식 테스트 금지(표 주도 테스트의 기대값을 입력에서 계산해 만들지 않고 리터럴로 적습니다). mock 호출 인자는 gomock 엄격 매칭, 상태코드, 응답 본문, 오류 전파를 단언하고 "후속 호출 없음"은 기대를 두지 않는 방식으로 단언합니다. 제품 코드의 오류 메시지 문자열은 단언하지 않습니다. 테스트 중 발견한 결함은 고치거나 문서화하지 않고 카드 코멘트에만 적으며, 해당 동작을 고정하는 단언은 하지 않습니다. 테스트 값은 합성값만 씁니다. 비밀번호 리터럴은 별도 문장에서 대입합니다.

verify 항목:
1. 수기 합계 76% 이상(scratch 집계 스크립트), 패키지별 전후 표와 수기/원본 합계, 제외 방법을 PR 본문에 기재.
2. `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 통과. go.mod/go.sum 변경은 miniredis 와 그 간접 의존 추가뿐이고 기존 의존 버전 상승과 생성 파일 변경이 없음을 `git diff` 로 확인하고 `go mod vendor` 결과(vendor/ 는 gitignore)가 커밋에 들어가지 않았음도 `git status` 로 확인.
3. `git diff --stat origin/main...HEAD` 가 bin-queue-manager/ 안에만 있음. `git fetch origin main` 후 리포 루트에서 `bash scripts/check-test-conventions.sh` 통과.
4. 변이 검사: 표본 10곳 이상(QueuecallSet 의 저장과 TTL, QueueGet 과 QueuecallGetByReferenceID 의 키 접두사, runListen 의 큐 이름 인자, listenhandler HealthCheck 호출 여부, ReconcileLockAcquire 의 NX 반환 분기, QueueSet 의 TTL 인자, QueuecallGet 의 JSON 오류 전파, ReconcileLockRelease 의 Del, runSubscribe 의 오류 반환, dbhandler Exec 오류 전파, listenhandler 오류 상태코드, queuecallhandler 오류 분기)에서 제품 코드를 카드 scratch 아래 복사본(원본 worktree 가 아님)에서 바꿔 새 테스트가 실패함을 확인하고 그 실패 로그 요약을 카드 코멘트에 남깁니다.
5. 중간 측정: 패키지 하나를 끝낼 때마다 scratch 집계 스크립트로 재측정해 기록합니다(필요량 2562/3371).
6. 비동기 테스트 `-race -count=20` 통과.
7. VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 합니다.

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
- 제외: cmd/queue-control 전체(DB와 Redis 를 직접 여는 runXxx 가 대부분), cmd/queue-manager 의 main, runDaemon, run, initSignal, initProm, runReconcile(전역 시그널, ListenAndServe, 고정 ticker 고루틴).
- 제외 후 남는 후보 미커버: cachehandler 67, dbhandler 111, queuecallhandler 72, listenhandler 45, runListen/runSubscribe 약 12. 합 약 307 로 필요량 133 대비 충분하여 75%(권장 76%) 달성은 산술적으로 가능합니다.
- cachehandler 는 `*redis.Client` 를 직접 쓰므로 miniredis(v2.36.1, 같은 버전이 bin-flow-manager 등 main 의 다른 서비스에 이미 있음)를 테스트에서만 import 합니다. 공통 규칙의 test-only go.mod 조건(비테스트 .go 변경 없음, `go mod tidy` 후 diff 없음)을 지킵니다.
- dbhandler 의 미커버는 거의 모두 ExecContext 오류, RowsAffected 오류, 캐시 갱신 오류 분기이며 기존 `newCtxTestHandler`(go-sqlmock)로 주입 가능합니다.
- 이 서비스에는 권한 검사 코드가 없습니다(권한은 api-manager). 입력 검증은 listenhandler 의 URI 길이, JSON 파싱, UUID/필터 파싱 경로입니다.

## 2. 계획

범위는 `_test.go` 추가와 go.mod/go.sum 의 miniredis 추가뿐입니다. 제품 코드는 바꾸지 않습니다.

| 대상 | 현 미커버 | 목표 채움 | 내용 |
|---|---|---|---|
| pkg/cachehandler | 67 | 60 | miniredis 로 NewHandler, Connect(성공, 서버 중지 후 오류), QueueSet/Get, QueuecallSet/Get, QueuecallGetByReferenceID(키를 직접 써서 조회), 없는 키 오류, 잘못된 JSON 값 오류, ReconcileLockAcquire(최초 true, 보유 중 false, TTL 만료 후 true, 서버 중지 시 오류), ReconcileLockRelease |
| pkg/dbhandler | 111 | 60 | 변경 함수별 ExecContext 오류 전파와 캐시 미호출, RowsAffected 오류(`sqlmock.NewErrorResult`), 조회 쿼리 오류, 캐시 get 오류 시 DB 대체 경로. 표 주도, mock 엄격 매칭 |
| pkg/queuecallhandler | 72 | 45 | event.go(EventAMAgentAvailable 등 오류 분기), execute.go(Execute 의 조회, 상태 갱신, 플로우 생성 오류, matchWaitingQueuecall), service.go(ServiceStart 오류 분기, getSource), kick.go, reconcile.go, health.go 의 오류 경로 |
| pkg/listenhandler | 45 | 30 | 각 processV1 핸들러의 핸들러 오류 → errorResponse(500 등), 잘못된 JSON/UUID 400, direct_hash regenerate, count, services |
| cmd/queue-manager | 113 | 10 | runListen, runSubscribe(sockhandler/queuehandler/queuecallhandler gomock, Run 오류 반환과 nil 반환) |
| 합계 | | 205 | 필요량 133 대비 1.5배 |

동시성: listenhandler.Run 과 subscribehandler.Run 은 ConsumeRPC/ConsumeMessage 를 고루틴으로 실행하므로 mock 의 Do 에서 채널로 신호하고 `select` 와 timeout 으로 기다립니다. 고정 sleep 은 새 테스트에서 쓰지 않습니다. 비동기 경로를 건드리는 테스트는 `go test -race -count=20` 으로 반복해 불안정성이 없음을 확인합니다.

원칙: 빈 호출과 자기비교식 테스트 금지. mock 호출 인자는 gomock 엄격 매칭, 상태코드, 응답 본문, 오류 전파를 단언하고 "후속 호출 없음"은 기대를 두지 않는 방식으로 단언합니다. 제품 코드의 오류 메시지 문자열은 단언하지 않습니다. 테스트 중 발견한 결함은 고치거나 문서화하지 않고 카드 코멘트에만 적으며, 해당 동작을 고정하는 단언은 하지 않습니다. 테스트 값은 합성값만 씁니다. 비밀번호 리터럴은 별도 문장에서 대입합니다.

verify 항목:
1. 수기 합계 76% 이상(scratch 집계 스크립트), 패키지별 전후 표와 수기/원본 합계, 제외 방법을 PR 본문에 기재.
2. `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 통과. go.mod/go.sum 변경은 miniredis(및 그 간접 의존)뿐이고 생성 파일 변경 없음.
3. `git diff --stat origin/main...HEAD` 가 bin-queue-manager/ 안에만 있음. `bash scripts/check-test-conventions.sh` 통과.
4. 변이 검사: 표본 4곳 이상(예: ReconcileLockAcquire 의 NX 반환 분기, dbhandler Exec 오류 전파, listenhandler 오류 상태코드, queuecallhandler 오류 분기)에서 제품 코드를 scratch 복사본에서 바꿔 새 테스트가 실패함을 확인.
5. 비동기 테스트 `-race -count=20` 통과.
6. VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 합니다.

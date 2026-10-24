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

구조 분석:
- cmd/ 의 main 과 queue-control(합 641 statement), internal/config 는 프로세스 기동, 플래그와 환경 파싱, 외부 연결(DB, Redis, RabbitMQ)에 묶여 있어 단위 테스트 대상에서 제외합니다. 이를 제외해도 pkg 쪽 미커버가 약 750 statement 이므로 60% 달성은 산술적으로 가능합니다.
- pkg/cachehandler 의 구체 구현(handler.go 54 + main.go 13 = 67 statement)은 `*redis.Client` 에 직접 의존하고 go.mod 에 miniredis 가 없어 단위 테스트가 어렵습니다(인터페이스 MockCacheHandler 는 있어 다른 패키지에서 쓰입니다). 제품 코드나 의존성 변경 없이는 다루지 않습니다.
- queuehandler, queuecallhandler, listenhandler, subscribehandler 는 gomock 인터페이스(dbhandler, requesthandler, notifyhandler, queuehandler, queuecallhandler, cachehandler)로 주입됩니다. dbhandler 는 `*sql.DB` 를 받으며 기존 테스트가 sqlite(go-sqlite3)와 go-sqlmock 을 이미 씁니다. 모두 기존 방식 그대로 테스트할 수 있고 제품 코드 변경은 필요 없습니다.
- 제외 후 여유는 pkg 미커버 750 statement 로 필요량 188(권장 222)의 약 4배입니다. 큰 파일: listenhandler/v1_queuecalls.go 106, queuehandler/db.go 84, queuecallhandler/db.go 84, dbhandler/queuecall.go 77, listenhandler/v1_queues.go 58, dbhandler/queue.go 52, queuecallhandler/health.go 41. 이 상위 7개 파일의 미커버 합이 502 이므로 이 파일들의 오류 경로 중 절반 가량만 채워도 61% 가 됩니다.

## 2. 계획

범위는 `_test.go` 추가뿐입니다. 제품 코드는 바꾸지 않습니다.

대상(미커버 statement 큰 순, 오류 경로 중심):
- pkg/queuehandler/db.go: AddWaitQueueCallID, RemoveServiceQueuecallID, RemoveWaitQueuecallID, AddServiceQueuecallID 정상 경로와 db 오류 두 갈래, List/Get/Update 계열 오류. create.go 와 direct_hash.go, agent.go 오류 경로.
- pkg/listenhandler: v1_queuecalls.go 의 TimeoutWait/TimeoutService/Execute 핸들러(0%)와 잘못된 URI(400), 잘못된 JSON(400), 핸들러 오류(errorResponse) 경로. v1_queues.go 의 같은 유형, v1_count.go, v1_services.go, v1_queues_direct_hash.go 오류 경로.
- pkg/queuecallhandler: db.go 의 db 오류와 notify 경로, health.go 전체(조회 오류, 종료 상태, 비 call 참조, 재시도 초과 kickForce, call 조회 실패, hangup, 정상 재설정), execute.go, event.go, kick.go, service.go, timeout.go 오류 경로.
- pkg/dbhandler: QueuecallSetStatusWaitingIfInitiating 와 queuecallSetStatusWaitingIf(sqlite 테스트 DB 사용, 조건 일치와 불일치), QueueCountByCustomerID, 쿼리 실패 경로 중 기존 방식으로 가능한 것.
- pkg/subscribehandler: processEventAMAgentStatusUpdated(available 일 때만 위임, 그 외 no-op, JSON 오류), processEventCMCallHangup, processEventRun 중 순수 분기, 오류 입력.

원칙: 빈 호출 금지. 각 테스트는 mock 의 호출 인자(gomock 엄격 매칭)와 반환 상태코드, 응답 본문, 오류 전파를 단언합니다. 오류 테스트는 "후속 호출이 일어나지 않음"을 mock 에 호출 기대를 두지 않는 것으로 단언합니다. 자기비교식 테스트는 쓰지 않습니다.

verify 항목:
1. 수기 합계 61% 이상(scratch 의 agg 스크립트), 표는 패키지별 전후로 PR 본문에 기재.
2. `go test -count=1 ./...` 전체 통과, `go vet`, `golangci-lint run -v --timeout 5m` 통과, `scripts/check-test-conventions.sh` 통과(Test_ 명명, gomock 컨트롤러 이름 mc).
3. 변이 검사: 제품 코드를 scratch 복사본에서 바꿔(조건 반전, 오류 반환 제거, 인자 교체) 새 테스트가 실패하는지 표본 5건 이상 확인하고 결과를 카드에 기록.
4. `git diff --stat origin/main...HEAD` 가 bin-queue-manager/ 안의 `_test.go` 와 이 문서뿐임을 확인.
5. 테스트 값은 합성 UUID 와 번호만 사용, 외부 호출과 네트워크 없음.

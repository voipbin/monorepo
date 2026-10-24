# VOIP-1651 bin-queue-manager 단위 테스트 커버리지 70% 계획

## 1. 티켓 검토

판정: 유효. 기준선은 origin/main 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했고 테스트 실패 0건, 수기 합계 69.59% (2346/3371 statement), 원본 60.75% (2346/3862)로 카드 수치와 일치합니다(재측정 시점에 따라 고루틴 타이밍으로 ±1 statement 변동 가능). 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록 중복은 최대 count 로 합쳤습니다. 70% 에는 14개, 권장 71% 에는 48개 statement 를 더 커버해야 합니다. 최근 경로 커밋(VOIP-1647 커버리지 상향, VOIP-1648 입력 파싱 강화)은 전제를 바꾸지 않으며 열린 중복 PR 은 없습니다.

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
| pkg/queuecallhandler | 80.6 | 154/793 |
| pkg/queuehandler | 99.7 | 1/387 |
| pkg/subscribehandler | 98.3 | 2/118 |

구조 분석: cmd/queue-control 의 명령 실행 함수, cmd/queue-manager 의 main 과 run(RabbitMQ 연결 생성), cachehandler 의 연결 생성부는 DB, Redis, RabbitMQ 연결을 직접 만들어 제품 코드 변경 없이는 단위 테스트가 어렵습니다. 그 밖의 순수 함수는 테스트할 수 있으나 비용 대비 효과가 낮아 이번에는 선택하지 않으며, 모자랄 때의 예비 후보로 남깁니다. 남은 미커버 중 가장 큰 곳은 pkg/queuecallhandler/db.go (80개)이며 mock 주입(dbhandler, requesthandler, notifyhandler, queuehandler, utilhandler)만으로 제품 코드 변경 없이 테스트됩니다. 제외분을 모두 빼도 pkg 미커버가 300 안팎이어서 70% 는 산술적으로 가능합니다.

## 2. 계획

범위는 `_test.go` 추가뿐이며 제품 코드, go.mod, go.sum 은 바꾸지 않습니다.

| 파일 | 대상 | 내용 |
|---|---|---|
| pkg/queuecallhandler/db_errors_test.go | Create, Delete, UpdateStatusConnecting/Service/Abandoned/Done/Waiting/WaitingRollback | DB 쓰기 오류 전파, compare-and-swap 패배 시 현재 행만 반환하고 부수효과(webhook, 큐 제거, confbridge 삭제 등)가 없음, 재조회 오류, 큐 제거 실패 시 이후 정리 중단, 정리 단계(confbridge, 변수, groupcall hangup, 예약 해제) 실패는 비치명적, groupcall 과 agent 가 없으면 해당 RPC 생략, 비동기 큐 등록 경로 |
| 필요 시 pkg/dbhandler 오류 경로 | queue.go, queuecall.go | 기존 sqlite/sqlmock 방식의 쿼리 오류 (queuecallhandler 만으로 71% 에 못 미칠 때만) |

원칙: 빈 호출과 자기비교식 테스트 금지. gomock 엄격 매칭으로 인자를 단언하고, 오류 테스트는 후속 호출이 일어나지 않음을 mock 에 기대를 두지 않는 것으로 단언합니다. 오류 메시지 문자열은 단언하지 않고 `errors.Is` 로 전파만 단언합니다. 비동기 경로(`UpdateStatusWaiting`, `UpdateStatusWaitingRollback` 의 고루틴)는 sleep 대신 AddWaitQueueCallID mock 의 Do 에서 닫는 채널과 timeout select 로 동기화합니다. 이 두 고루틴 경로는 AddWait 오류에서 끝나는 분기만 다루므로 mock 호출이 모두 신호 이전에 끝납니다.

verify 항목:
1. 수기 합계 71% 이상(scratch 집계 스크립트), 패키지별 전후 표를 PR 본문에 기재.
2. 루트 CLAUDE.md 검증 순서(`go mod tidy`, `go mod vendor`, `go generate`, `go test`, `golangci-lint`) 통과, go.mod, go.sum, 생성 파일 무변경.
3. `-race -count=20` 으로 신규 테스트의 불안정성 없음.
4. `scripts/check-test-conventions.sh` 통과(Test_ 명명, gomock 컨트롤러 이름 mc).
5. 변이 검사: CAS 패배 분기의 조기 반환, 큐 제거 실패의 오류 반환, UpdateStatusWaiting 고루틴의 AddWaitQueueCallID 오류 시 return 을 각각 제거했을 때 신규 테스트가 실패함. 비동기 경로는 채널 대기 후 컨트롤러가 종료되도록 보장하고 전역 prometheus 카운터는 단언하지 않습니다.
6. PR 의 파일 목록이 bin-queue-manager/ 안에만 있음(`git diff --stat origin/main...HEAD`).
7. VOIP-1416 이 적은 80% 미만 패키지(dbhandler 외)와의 겹침은 PR 본문에 명시만 합니다.

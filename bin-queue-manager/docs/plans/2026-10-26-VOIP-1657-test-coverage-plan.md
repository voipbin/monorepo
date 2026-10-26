# bin-queue-manager 단위 테스트 커버리지 80% 계획

## 1. 측정

기준은 origin/main 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 측정했으며 테스트 실패는 없습니다. 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록의 중복은 최대 count 로 합침)의 합계는 76.84% (2588/3368 statement)입니다. 고루틴 타이밍에 따라 1 statement 정도 달라질 수 있습니다. 80% 는 2695개(107개 추가), 80.5% 는 2712개(124개 추가)가 필요합니다.

패키지별 기준선(수기):

| 패키지 | 전 | 미커버/전체 |
|---|---|---|
| cmd/queue-control | 0.0 | 528/528 |
| cmd/queue-manager | 10.5 | 102/114 |
| internal/config | 97.4 | 2/76 |
| pkg/cachehandler | 98.4 | 1/64 |
| pkg/dbhandler | 89.1 | 66/604 |
| pkg/listenhandler | 93.1 | 45/651 |
| pkg/queuecallhandler | 96.0 | 32/793 |
| pkg/queuehandler | 99.7 | 1/387 |
| pkg/subscribehandler | 98.3 | 1~2/118 |
| models/queue, models/queuecall | 93.8, 94.1 | 각 1 |

미커버의 대부분은 cmd/queue-control 의 명령 정의와 순수 함수이며, 데이터베이스, Redis, RabbitMQ 연결 없이 실행할 수 있어 이것만으로 필요량을 넘길 수 있습니다.

## 2. 계획

범위는 `_test.go` 추가뿐이며 제품 코드, go.mod, go.sum 은 바꾸지 않습니다.

| 파일(신규 테스트) | 내용 |
|---|---|
| cmd/queue-control/main_test.go | initCommand 의 하위 명령 구성, 각 cmd* 의 Use 와 플래그 이름과 기본값, resolveUUID, parseUUIDs, printJSON(정상과 marshal 불가 값), runCreate 의 입력 검증 분기 |
| pkg/dbhandler/coverage_gaps_test.go | NewHandler, 빈 fields 의 갱신이 아무것도 보내지 않음, 쓰기 성공 뒤 캐시 갱신 실패가 쓰기 실패로 전파되지 않음 |
| pkg/listenhandler 추가 | direct hash 재생성 성공 응답 |
| cmd/queue-manager 추가 | initSignal |

원칙: 빈 호출과 자기비교식 테스트를 쓰지 않습니다. 각 테스트는 인자, 반환 상태, 오류 전파를 단언합니다. 전역 상태(viper, logrus 포매터와 레벨, 환경변수, os.Stdout, signal)는 테스트마다 격리하고 복원하며 t.Parallel 을 쓰지 않습니다. initCommand 는 config.Bootstrap 으로 전역 상태를 바꾸므로 viper.Reset 과 logrus 복원, 환경변수 고정을 합니다. initSignal 은 프로세스에 시그널을 보내지 않고 chSigs 에 값을 직접 보내며 signal.Stop 으로 정리합니다. 데이터베이스나 Redis 에 연결하는 경로(initHandler 이후의 run 함수 등)는 호출하지 않습니다.

verify 항목:
1. 수기 합계 80.5% 이상, 패키지별 전후 표를 PR 본문에 기재.
2. `git fetch origin main` 후 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 통과, go.mod, go.sum, 생성 파일 무변경, 저장소 루트 `bash scripts/check-test-conventions.sh` 통과.
3. 새 테스트는 `go test -race -count=20` 통과.
4. 변이 검사: 대상 분기를 임시 사본에서 바꾸면 새 테스트가 실패함을 표본 확인.
5. PR 파일 목록이 bin-queue-manager/ 안에만 있음.

## 3. 구현 결과

cmd/queue-control 의 테스트만으로 필요량을 크게 넘어 수기 합계가 86%대가 되었습니다. 그래서 pkg/queuecallhandler 의 남은 분기에 대한 추가는 이번 PR 에서 하지 않았고, 위 표의 항목은 모두 구현했습니다.

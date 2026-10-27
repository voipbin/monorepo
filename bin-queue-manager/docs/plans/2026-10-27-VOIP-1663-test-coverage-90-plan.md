# VOIP-1663 bin-queue-manager 단위 테스트 커버리지 90% 계획

## 1. 티켓 검토

판정: 유효. 기준선은 origin/main 0bfbe2a1d 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했고 테스트 실패 0건, 수기 합계 86.55% (2915/3368)입니다. 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록의 중복은 최대 count 로 합쳐 계산했습니다. 90% 까지 116개, 권장 90.5% 까지 133개 statement 를 더 커버해야 합니다.

패키지별 기준선(수기, 전): cmd/queue-control 55.3, cmd/queue-manager 14.9, internal/config 97.4, models/queue 93.8, models/queuecall 94.1, pkg/cachehandler 98.4, pkg/dbhandler 93.5, pkg/listenhandler 93.5, pkg/queuecallhandler 96.0, pkg/queuehandler 99.7, pkg/subscribehandler 99.2.

## 2. 범위와 원칙

- `_test.go` 추가뿐이며 제품 코드, go.mod, go.sum 은 바꾸지 않습니다.
- 외부 연결(DB, Redis, RabbitMQ, 네트워크)을 쓰지 않습니다. 연결이 필요한 함수는 파싱 단계에서 즉시 실패하는 입력(`not-a-dsn`, 포트 없는 주소)으로 오류 경로만 검증합니다.
- 시간, 난수, 전역 상태에 의존하지 않습니다. 비동기 로그는 logrus test hook 과 채널로 기다리고 sleep 을 쓰지 않습니다.
- 기대값은 입력과 독립적으로 손으로 쓴 값(상태, 오류 메시지 일부, 호출 인자)을 단정합니다.

## 3. 대상

| 영역 | 예상 증분(statement) | 내용 |
|---|---|---|
| pkg/queuecallhandler | ~29 | event, execute, health, kick, reconcile, timeout, 생성자의 오류 전파와 경계(에이전트 1명 고정으로 난수 분기 결정화) |
| pkg/listenhandler | ~36 | 응답 직렬화 오류 시 500, errorResponse 의 변환 실패 분기 |
| pkg/dbhandler | ~10 | 캐시 set 오류 전파, 지원하지 않는 필터 값 오류, Update 계열 직렬화 불가 값 오류 |
| models/queue, models/queuecall, pkg/cachehandler | ~3 | 직렬화 실패 오류 반환 |
| pkg/queuehandler, pkg/subscribehandler | ~2 | 생성자, 소비 시작 오류 로그 |
| cmd/queue-control | ~51 | run*/runQueuecall* 의 초기화 오류 반환, runCreate 검증 통과 후 초기화 오류, initCache 오류, main --help |
| cmd/queue-manager | ~28 | runDaemon 조기 오류 반환, initProm 오류 로그, runListen 의 Run 오류(반환 nil 과 오류 로그 1건을 단정), main --help |

예상 증분 합계는 필요량 116(권장 133)을 웃돕니다. 구현은 cmd 두 패키지를 먼저 실측합니다.

## 4. 격리 규칙

- 설정: `config.Bootstrap` 이 globalConfig 에 로컬 기본값을 채우므로 DSN 환경변수를 비우는 것만으로는 부족합니다. 각 테스트가 `*config.Get()` 에 DSN(`not-a-dsn`), Redis 주소(포트 없는 값), listen 주소(포트 없는 값)를 직접 대입하고 `t.Cleanup` 으로 이전 값을 복원합니다. `run*` 와 `init*` 는 cobra 실행 경로가 아니라 직접 호출하고(PreRun 의 `sync.Once` 에 기대지 않음), viper 는 Reset 으로 복원합니다.
- `initProm` 과 `runDaemon` 은 `http.DefaultServeMux` 에 등록하므로 endpoint 를 패키지 수준 원자 카운터로 만들어 `-count` 반복에서도 중복되지 않게 합니다(빈 값 금지, 선행 `/` 필수).
- `runDaemon` 이 만든 `initSignal` 고루틴은 `t.Cleanup` 에서 `chSigs` 에 신호 값을 보내 종료시키고 `chDone` 을 소비한 뒤 `signal.Stop` 을 호출합니다. 프로세스에 실제 신호는 보내지 않습니다.
- logrus 훅은 메시지 부분문자열로 거른 항목만 세고 `ReplaceHooks` 로 복원하며, 포매터와 레벨도 복원합니다. 비동기 로그는 채널과 시한 있는 대기로 기다리고 sleep 을 쓰지 않습니다.
- `main --help` 는 `os.Args` 를 복원하고 stdout 출력은 단정하지 않으며 오류 없음만 확인합니다.
- queue-control 은 기존 `resetGlobalState`, `captureStdout` 헬퍼를 재사용하고 config 복원은 같은 파일 안의 작은 헬퍼로 추가합니다. `runListen` 의 Run 오류는 현재 동작(오류를 로그만 남기고 nil 반환)을 고정하는 특성화 테스트입니다.
- 모든 새 테스트는 `Test_` 접두와 gomock 컨트롤러 변수 `mc` 규약을 따르고 testify 를 쓰지 않습니다.

## 5. Verify 항목

1. `GOFLAGS=-mod=mod go test -race -shuffle=on -count=3 ./...` 통과.
2. 수기 합계 90.0% 이상(권장 90.5% 이상), 패키지별 전후 표를 같은 플래그로 재측정.
3. `golangci-lint run -v --timeout 5m` 통과, `go mod tidy` 후 go.mod, go.sum 무변경.
4. `git diff --stat origin/main...HEAD` 가 bin-queue-manager/ 안의 `_test.go` 와 이 문서뿐.
5. 변이 검사: 대표 분기 몇 곳의 제품 코드를 scratch 복사본에서 바꿔 새 테스트가 실패함을 확인.

# bin-transcribe-manager 단위 테스트 커버리지 80% 상향

## 1. 배경과 측정

수기 코드(`mock_*.go`, `/gens/`, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록의 중복은 최대 count 로 합산)의 서비스 합계를 80.5% 이상으로 올립니다. 측정 명령은 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 입니다.

기준선(origin/main 161e9748)은 수기 78.50%(1975/2516), 원본 70.86%(1975/2787)입니다. 80.0% 는 2013개 이상(+38), 80.5% 는 2026개 이상(+51)입니다.

## 2. 범위와 제외

제품 코드는 바꾸지 않고 `_test.go` 만 추가합니다. 시그니처 변경 없이 도달 가능한 분기만 대상으로 합니다. 제외: `streaminghandler` 의 `awsProcessMedia`, `awsProcessResult`(구체 SDK 스트림 타입), `cmd` 의 `main`, `runDaemon`, `run`, `runListen`, `runSubscribe`, `runStreaming` 과 `run*`(control) 중 핸들러 초기화 이후의 본문(서버 Run 과 외부 연결), `dbhandler` 의 쿼리 빌더 오류.

## 3. 테스트 구성과 격리

기존 파일은 고치지 않고 `<소스>_<주제>_test.go` 를 새로 만들며 함수 이름은 `Test_<대상>_<시나리오>` 입니다.

- cmd/transcribe-manager: `initSignal`, `initProm`(비활성 입력, 사용 중인 포트의 서버 오류), `initCache`(miniredis 성공과 논리 DB 번호, 닫힌 포트 실패).
- cmd/transcribe-control: `initCache`, `initHandler` 의 DB 연결 실패, 각 `run*` 의 초기화 실패 전파, `PersistentPreRunE` 의 플래그 바인딩.
- internal/config: `LoadGlobalConfig` 의 값, 기본값, 1회 로드.
- 생성자: `NewTranscriptHandler`(합성 서비스 계정 파일로 성공, 없는 파일로 실패), `NewTranscribeHandler`, `dbhandler.NewHandler`(캐시 우선과 DB 폴백), `NewStreamingHandler` 의 provider 조합.
- streaminghandler: 닫힌 `BaseEndpoint`(재시도 없음)의 `awsInit`, `awsRun` 오류.
- listenhandler: 짧은 URI, 잘못된 본문과 필터의 400. subscribehandler: 잘못된 payload 는 오류 반환, 하위 핸들러 오류는 삼킴.

격리: `config.Get()` 는 구조체 전체를 저장 후 복원합니다. `LoadGlobalConfig` 는 프로세스당 1회만 실행되므로 cmd 테스트는 로드된 값을 단언하지 않고 `internal/config` 테스트가 `once`, `globalConfig`, viper 를 저장 후 복원하며 값을 단언합니다. `initSignal` 은 실제 시그널 없이 `chSigs` 에 값을 넣고 `signal.Stop` 과 `chDone` 비우기를 합니다. `initProm` 은 호출마다 고유 endpoint 를 쓰고 logrus hook 은 원복합니다. 닫힌 포트는 `127.0.0.1:0` 리스너를 닫아 얻으며 재사용 가능성은 감수합니다. 병렬 실행은 하지 않고 신규 모듈은 없습니다(miniredis 는 go.mod 에 이미 있음). 서비스 계정 키는 실행 시 생성합니다.

## 4. 검증 항목

- 수기 합계 80.5% 이상, `go test -race -shuffle=on ./...` 통과, 신규 테스트 `-race -count=20` 반복 통과.
- `golangci-lint` 0 issues, `scripts/check-test-conventions.sh` 통과.
- 변이 표본: 제품 코드를 바꿔 신규 테스트가 실패하는지 확인합니다.
- `git diff --stat origin/main...HEAD` 로 변경 파일이 `bin-transcribe-manager/` 안의 `_test.go` 와 이 문서뿐임을 확인합니다.

## 5. 결과

수기 합계 약 83.4%(2099~2100/2516, 일부 goroutine 구문이 실행마다 달라 ±1), 기준선 대비 약 +125 statement. 패키지별(전 → 후): cmd/transcribe-control 52.3 → 61.1, cmd/transcribe-manager 0.6 → 12.2, internal/config 89.6 → 96.1, pkg/dbhandler 87.2 → 88.9, pkg/listenhandler 91.1 → 94.0, pkg/streaminghandler 80.6 → 88.6, pkg/subscribehandler 91.8 → 99.0, pkg/transcribehandler 96.3 → 97.2, pkg/transcripthandler 93.9 → 99.4. 나머지 패키지는 동일합니다.

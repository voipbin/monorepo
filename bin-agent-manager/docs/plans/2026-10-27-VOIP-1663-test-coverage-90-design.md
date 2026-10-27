# VOIP-1663 bin-agent-manager 단위 테스트 커버리지 90% 이상 설계

## 1. 티켓 검토 (이슈 분석)

판정: 유효. 기준선은 origin/main 0bfbe2a1d 에서 재실측했고 카드 수치와 일치합니다.

- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`. 수기 합계는 gens/, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합칩니다. `go test` 가 출력하는 패키지 퍼센트(mock 포함)와 다릅니다.
- 수기 86.86% (2599/2992 statement). 90.0% 는 2693, 90.5% 는 2708 statement 이므로 최소 +94, 권장 +109 가 필요합니다.
- 패키지별 현재값(covered/total): cmd/agent-control 441/646, cmd/agent-manager 1/103, internal/config 62/65, models/agent 35/36, pkg/agenthandler 746/750, pkg/cachehandler 60/61, pkg/dbhandler 597/637, pkg/listenhandler 565/601, pkg/metricshandler 0/1, pkg/subscribehandler 92/92.
- 미커버의 대부분은 cmd/ 두 패키지입니다. 외부 서버(DB, RabbitMQ)가 있어야 지나가는 코드는 대상에서 제외하고, 그 앞의 실패 경로와 테스트 가능한 함수만 대상으로 합니다.

## 2. 접근

시그니처와 제품 코드는 바꾸지 않습니다. 외부 서버 없이 다음 수단만 씁니다.

- 파싱에 실패하는 DSN(`not-a-dsn`)을 `config.Get().DatabaseDSN` 에 설정하면 DB 연결이 네트워크 없이 즉시 실패합니다. 이 실패 경로를 cmd 의 initHandler, run*, runNormalizeAddresses, runDaemon 에서 확인합니다.
- 캐시 연결은 테스트 파일 안의 loopback 최소 RESP 서버(PING 응답)로 확인합니다. 선례는 pkg/cachehandler 테스트입니다.
- RabbitMQ 는 `sockhandler.NewMockSockHandler`(gomock)로 대체해 runServiceListen, runServiceSubscribe 를 확인합니다.
- 목록 응답의 직렬화 실패 경로는 gomock 이 JSON 으로 표현할 수 없는 시각(연도 10000)을 가진 agent 를 돌려주게 해서 확인합니다.
- SQL 빌드 실패 경로는 입력으로 유발합니다. agentUpdate 는 정수 kind 이면서 `squirrel.Sqlizer` 를 구현해 `ToSql` 이 오류를 내는 테스트 전용 타입을 필드 값으로 넣습니다(`PrepareFields` 를 통과하므로 필드 준비 오류 분기가 아닌 SQL 빌드 분기). AgentList 는 `commondatabasehandler.NotEq{Value: <Value() 가 오류를 내는 driver.Valuer>}` 필터로 유발합니다.

## 3. 계획 (verify 항목)

컨벤션: `Test_` 이름, gomock 컨트롤러 이름 `mc`, testify 금지(`reflect.DeepEqual` + `t.Errorf`), table-driven. 전역 상태는 테스트마다 복원합니다.

1. cmd/agent-control (신규 테스트 파일)
   - 잘못된 DSN 으로 initHandler 가 `could not connect to the database` 를 반환합니다.
   - 11개 run* 각각에 검증을 통과하는 입력을 주고 오류가 `failed to initialize handlers` 로 감싸지며 DB 연결 오류를 포함함을 확인합니다(검증 단계 오류가 아님을 구분).
   - runNormalizeAddresses: 잘못된 DSN 에서 `could not connect to the database`, 기본 dry-run 값 유지.
   - initCache: RESP 서버 PING 성공 시 핸들러 반환, `-ERR` 시 `could not connect to the cache`.
   - PersistentPreRunE: `agent get --id not-a-uuid` 처럼 입력 검증 단계에서 실패하는 서브커맨드를 `initCommand()` 로 만든 루트에서 Execute 하여 PreRun 이 실행되고 검증 오류로 끝남을 확인합니다. DB 에 가지 않으므로 환경의 DSN 변수와 무관하고, `t.Setenv` 로 DSN 환경 변수를 파싱 실패 값으로 두어 전역 config 가 처음 로드되더라도 빈 DSN 이 되지 않게 하고, viper 와 전역 config(복사본 저장 후 복원)를 정리합니다. 전역 config 는 직접 덮어쓰고 복원하므로 테스트 순서와 무관합니다.
   - migrate 서브커맨드: `initCommand()` 로 만든 루트의 PersistentPreRunE 를 nil 로 바꾸어 전역 config 의 once 가 DSN 을 덮어쓰지 않게 하고, `config.Get().DatabaseDSN` 을 파싱 실패 값으로 직접 설정한 뒤 `migrate status` 를 Execute 하여 오류를 단언합니다(원래 값은 복원). PreRun 을 거치는 테스트는 위의 검증 실패 서브커맨드 하나뿐입니다.
   - main(): `os.Args` 를 `--help` 로 바꿔 호출하고 stdout 에 도움말 문구(`Voipbin Agent Management CLI`)가 있음을 단언. `os.Args`, logrus 포맷터와 레벨, viper 를 `t.Cleanup` 으로 복원.
2. cmd/agent-manager (신규 테스트 파일)
   - runDaemon: 잘못된 DSN 에서 `could not connect to the database`. runDaemon 은 config 의 Prometheus endpoint 와 listen 주소를 initProm 에 넘기므로 이 테스트도 고유 endpoint, 형식 오류 listen 주소를 `config.Get()` 에 설정하고 `http.DefaultServeMux` 를 교체 후 복원합니다(빈 값은 panic 또는 80번 포트 접속을 유발하므로 금지). 남는 initSignal goroutine 은 위 initSignal 방식으로 무해합니다.
   - initCache: RESP 서버가 PING 을 실제로 받았음을 기록해 단언(성공), `-ERR` 일 때 `cache connect error`(실패). RESP 스텁은 두 cmd 패키지가 각자 테스트 파일에 두므로 중복되며, 제품 코드 변경 없이 공유할 방법이 없어 허용합니다.
   - initSignal: `chSigs`/`chDone` 변수는 교체하지 않습니다(남은 goroutine 과 경합). 시도마다 `initSignal()` 을 다시 호출하고(한 번 호출한 goroutine 은 신호 하나를 소비하고 끝남), 그 전에 `chDone` 을 비우고 `chSigs` 에 신호를 직접 주입한 뒤 타임아웃이 있는 `chDone` 수신을 단언합니다. 주입 신호는 어느 goroutine 이 소비해도 chDone 에 한 값이 쌓이므로 결과가 같고, chDone 전송을 제거하면 수신이 타임아웃으로 실패합니다. 끝에서 `signal.Stop(chSigs)` 로 정리합니다.
   - initProm: 호출마다 고유 endpoint 와 고유한 형식 오류 listen 주소를 쓰고 `http.DefaultServeMux` 를 교체 후 복원합니다. 오류 로그는 logrus test hook 으로 기다리되 그 호출의 고유 주소를 포함한 메시지로만 매칭하고, 타임아웃을 둡니다. hook 은 테스트 전에 저장한 원래 훅 집합으로 복원합니다. `-race` 게이트로 mux 접근 경합이 없음을 확인합니다.
   - runServiceListen: `QueueCreate` 오류 전파, 성공 경로(consume goroutine 의 `ConsumeRPC` 호출을 채널로 동기화). runServiceSubscribe: `QueueCreate` 오류, 토픽 exchange 선언 실패, 패턴 바인딩 실패의 오류 전파와 성공 경로(호출 순서 전체를 gomock 으로 기대). 성공 경로의 consume goroutine 호출은 채널로 동기화하여 테스트 종료 뒤 mock 호출이 없게 합니다.
   - main(): `--help` 로 호출해 stdout 도움말 문구(`Voipbin Agent Manager Daemon`)를 단언하고 `os.Args`, logrus, viper 를 복원.
3. pkg/listenhandler: 직렬화 불가 시각을 가진 agent 를 돌려주는 mock 으로 각 응답 경로(List, Get, Create 계열, Delete, GetByCustomerIDAndAddress, Login, UpdateAddresses, UpdateBasicInfo, UpdateStatus, UpdatePassword, UpdateTagIDs, UpdatePermission, direct hash)가 500 을 반환. errorResponse 에 typed-nil `*VoipbinError` 를 주면 500.
4. pkg/dbhandler: agentUpdate 와 AgentList 에서 SQL 빌드가 실패하면 래핑된 오류를 반환하고 쿼리를 실행하지 않음(sqlmock `ExpectationsWereMet`).
5. pkg/cachehandler: JSON 직렬화 불가 값에서 setSerialize 가 오류를 반환. pkg/metricshandler: 각 collector 에 대해 `prometheus.Register` 가 `AlreadyRegisteredError` 를 반환함(패키지 초기화에서 등록됨)을 단언.

기대 증분(수기 statement, 위치 없는 숫자): cmd/agent-control 약 +59, cmd/agent-manager 약 +51, pkg/listenhandler 약 +30, pkg/dbhandler +4, pkg/cachehandler +1, pkg/metricshandler +1. 합계 약 +146 이며 권장 +109 에 대한 여유는 약 37 입니다. 측정 결과가 2708 미만이면 먼저 listenhandler 직렬화 경로와 runServiceSubscribe 오류 경로의 누락 케이스를 점검하고, 그래도 부족하면 카드에 사유를 적어 default 에 보고합니다.

최종 게이트: §1 의 측정 명령으로 수기 합계 90.5% 이상(2708 이상), 전체 통과, `go test -race -shuffle=on ./...`, cmd 패키지에 한해 `-count=3`(기존 dbhandler 테스트는 고정 ID 때문에 `-count>1` 을 지원하지 않음), lint, `scripts/check-test-conventions.sh`.
변이 검사(scratch 복사본에서만): initHandler 와 runDaemon 의 오류 감싸기 제거, 두 cmd 의 initCache 오류 처리 제거, initSignal 의 chDone 전송 제거, runServiceListen 과 runServiceSubscribe 의 오류 반환 제거, listenhandler 직렬화 실패 분기를 경로별로 3개 이상 표본 선택해 500 을 200 으로 변경(테이블 케이스는 상태 코드와 함께 응답 본문에 성공 데이터가 없음도 단언), agentUpdate 와 AgentList 의 오류 반환 제거, setSerialize 의 오류 반환 제거. 각 변이에서 새 테스트가 실패해야 합니다.

## 4. 범위와 위험

- 변경 파일은 bin-agent-manager 의 `*_test.go` 와 이 문서뿐입니다. 제품 코드 변경 없음. 테스트 중 발견한 범위 밖 사항은 수정하지 않습니다.
- 실제 DB, Redis, RabbitMQ 접속과 외부 비용 호출이 없습니다. loopback RESP 서버는 테스트 안에서만 열고 Cleanup 에서 닫습니다.
- DSN 은 반드시 파싱 실패 값을 씁니다(빈 값은 기본 포트 접속을 시도할 수 있음).
- 전역 상태(viper, config, logrus, DefaultServeMux, os.Args, signal)는 각 테스트에서 복원하고 `t.Parallel` 과 섞지 않습니다.

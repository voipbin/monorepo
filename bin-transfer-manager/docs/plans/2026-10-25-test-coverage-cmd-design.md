# bin-transfer-manager 단위 테스트 커버리지 상향 설계 (VOIP-1652)

## 1. 티켓 검토
유효. origin/main accc03e00 에서 수기 671/949 = 70.71%, 원본 671/1076 = 62.36% 를 재측정해 기준 수치와 일치합니다. 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합칩니다.
열린 PR 과 중복 없음(VOIP-1647 PR #1518 은 머지되어 cachehandler, dbhandler, subscribehandler, transferhandler 를 이미 올림). 이번 대상은 그 PR 이 범위에서 뺀 cmd/ 입니다.
`cmd/` 리팩터링 금지이므로 제품 코드는 바꾸지 않고, 현 시그니처 그대로 테스트합니다.

## 2. 기준선 (수기, covered/stmts)
cmd/transfer-control 0/145, cmd/transfer-manager 1/122(기존 startup_log_test.go 의 logStartup 뿐이며 새 테스트 파일은 같은 package main 의 main_test.go 로 이름이 겹치지 않음), internal/config 58/61, models/transfer 8/9, pkg/cachehandler 59/59, pkg/dbhandler 42/42, pkg/listenhandler 97/103, pkg/subscribehandler 88/88, pkg/transferhandler 318/320.
cmd 를 뺀 미커버는 12 뿐이라 cmd 를 올리지 않으면 최대 683/949 = 72.0% 로 75% 가 불가능합니다. 76% 는 722 이상(+51), 75% 는 712 이상(+41)이 필요합니다.

## 3. 변경 계획 (테스트 파일만, 제품 코드와 go.mod/go.sum 변경 없음)
cmd 도달 가능분을 함수별로 정리합니다(stmt 는 프로파일 블록 기준 근사).
| 함수 | 방법 | 도달 stmt |
|---|---|---|
| control: resolveUUID, resolveString | 직접 호출. 빈 값, 잘못된 UUID, 정상 | 약 11 |
| control: printJSON | os.Stdout 캡처로 들여쓰기 JSON 단언, 직렬화 불가 값 오류 | 약 5 |
| control: initCommand, cmdServiceStart, cmdGetByGroupcall, cmdGetByCall | initCommand() 를 직접 호출해 서브커맨드 이름과 플래그 이름과 기본값 단언(Execute 는 쓰지 않으므로 PersistentPreRunE 클로저 약 4 stmt 는 제외) | 약 55 |
| control: initHandler(DB 단계 실패), run* 3종의 initHandler 실패 | `config.Get().DatabaseDSN` 을 존재하지 않는 unix 소켓으로 직접 설정하고 runServiceStart, runGetByGroupcall, runGetByCall 을 직접 호출해 sql 연결 즉시 실패와 오류 문자열(`failed to initialize handlers`, `could not connect to the database`) 단언. 이후 resolve* 호출부는 DB 성공이 필요해 도달 불가 | 약 20 |
| control: initCache | initHandler 는 DB 에서 먼저 실패하므로 initCache 를 직접 호출. miniredis 정상, 종료된 주소 오류 | 약 4 |
| manager: initSignal | SIGHUP 수신 시 chDone 전송 | 약 5 |
| manager: initProm | 비활성(빈 값) 시 핸들러 미등록, 활성 시 응답, 잘못된 listen 주소의 오류 로그 | 약 10 |
| manager: initCache | miniredis 정상과 오류 | 약 4 |
| manager: runSubscribe, runRequestListen | mock sockhandler 로 성공과 QueueCreate, TopicCreateWithKind, QueueBind(첫 번째와 두 번째 패턴) 실패 전파. 실패 시 이후 호출과 ConsumeMessage 가 없음을 Times 로 단언. 성공 경로의 고루틴 호출(ConsumeMessage, ConsumeRPC)은 AnyTimes 기대와 채널로 호출 완료를 기다리되 대기에는 항상 제한 시간(select 와 time.After)을 두어 hang 을 막고 종료 뒤 호출이 없게 함(ConsumeMessage 와 ConsumeRPC 는 콜백 타입이 서로 달라 DoAndReturn 시그니처를 각각 맞춤) | 약 20 |
| manager: runDaemon | DB 연결 실패 시 감싼 오류 반환(앞단 initSignal, initProm 비활성, logStartup 포함) | 약 7 |
| 기타 | (listenhandler main.go:95 의 cerrors.ToResponse 실패 분기 2 stmt 는 typed 오류 주입이 필요해 수치에서 제외) transferee.go:103 의 TransfereeAnswer Attended 위임과 :140 의 TransfereeAnswerTypeAttended, listenhandler main.go:191,193 의 processRequest 오류 케이스(ve, ErrNotFound) | 약 4 |
합계 도달분 약 140, 예상 수기 합계 약 810/949 입니다. 권장 722(76%)에 여유 약 80 을 둡니다. 실제 값은 구현 뒤 재측정합니다.

도달 불가로 제외하고 수치에 넣지 않음:
- main 함수(Execute 와 log.Fatal, os.Exit).
- manager 의 run 과 control 의 initTransferHandler: sockhandler 를 함수 안에서 직접 생성하고 RabbitMQ 연결(`bin-common-handler/pkg/rabbitmqhandler/main.go` connect())이 성공할 때까지 1초 간격으로 무한 재시도하며, 주입 지점이 없습니다. 고루틴으로 우회하면 영구 재시도 고루틴이 남으므로 하지 않습니다.
- DB 연결 성공 이후의 본문: commondatabasehandler.Connect 가 sql.Open 뒤 Ping 을 하고 DSN 외 주입 지점이 없어 sqlmock 으로 우회할 수 없습니다.
- config 의 BindPFlag/BindEnv 오류 분기, webhook.go 와 v1_transfers.go 의 json.Marshal 오류 분기: 입력으로 실패시킬 수 없습니다.

## 4. 품질 규칙
빈 호출과 자기비교 금지. 단언은 반환 오류 문자열, 출력 JSON, 등록된 서브커맨드와 플래그, HTTP 응답, 호출 인자로 합니다. 네트워크는 loopback 과 miniredis, 존재하지 않는 unix 소켓뿐이며 외부 호출과 비용 호출은 없습니다. 합성 값만 씁니다. DSN 은 `user@unix(/nonexistent/path)/db` 형태로 비밀번호가 없습니다. docs/conventions/testing.md 와 scripts/check-test-conventions.sh 규칙(Test_ 이름, mc 컨트롤러)을 따릅니다.
전역 상태 규칙:
- 상태 복원은 모두 `t.Cleanup` 으로 합니다(중간 Fatal 에서 새지 않게). runDaemon 테스트는 config 전체를 저장 후 복원하고 PrometheusEndpoint, PrometheusListenAddress 를 명시적으로 비웁니다. 신호 테스트는 시작 전에 chDone 을 비웁니다. 로그 훅은 커스텀 훅을 등록하고 `ReplaceHooks` 로 원복하며 버퍼 채널과 non-blocking 송신으로 메시지를 필터링합니다. initProm endpoint 는 패키지 수준 atomic 카운터로 호출마다 고유하게 만들어 -count 나 -shuffle 에서도 중복 등록이 없게 하고, 포트는 `127.0.0.1:0` 으로 확보한 리스너를 열어 둔 채 쓰거나(오류 케이스) 짧은 확보-해제 뒤 폴링(응답 케이스)합니다. 테이블 주도 테스트는 docs/conventions/testing.md 를 따릅니다.
- 테스트는 cobra Execute 를 쓰지 않습니다. Execute 의 PersistentPreRunE 가 sync.Once 로 config 를 viper 값으로 덮어써 테스트가 쓴 값을 지우기 때문입니다. 대신 run* 와 init* 함수를 직접 호출하고 `config.Get()` 포인터에 값을 쓰며 `t.Cleanup` 으로 복원합니다(빈 DSN 이면 기본 loopback TCP 접속을 시도하므로 DSN 을 항상 unix 소켓으로 설정).
- resolve* 테스트는 `viper.Reset()` 후 필요한 키만 `viper.Set` 하고 `t.Cleanup` 에서 `viper.Reset()` 합니다. initCommand() 는 Bootstrap 이 logrus 포매터와 레벨을 바꾸므로 호출 전 값을 저장해 복원하고, 그 테스트 안에서 viper.Reset() 은 initCommand() 호출보다 앞에 둡니다.
- initProm 은 http.DefaultServeMux 에 등록하며 해제할 수 없으므로 활성 경로 테스트는 프로세스 안에서 한 번만 등록하는 케이스마다 서로 다른 고유 endpoint 를 쓰고(활성 응답 케이스와 잘못된 listen 주소 케이스가 같은 endpoint 를 쓰면 panic), 빈 포트를 미리 확보해 listen 주소로 주고 응답이 올 때까지 제한 시간 안에서 재시도합니다. 서버 고루틴은 닫을 수 없어 프로세스 종료까지 남습니다. 잘못된 listen 주소 케이스는 리스너를 열어 둔 채 같은 주소를 주어 점유 오류를 결정적으로 만들고, 오류 로그 단언은 logrus 훅과 채널로 동기화합니다.
- printJSON 의 os.Stdout 캡처는 파이프 읽기를 별도 고루틴에서 하고 쓰기 끝을 호출 뒤 닫으며 `t.Cleanup` 으로 os.Stdout 을 복원합니다. initProm 응답 케이스는 상태 코드뿐 아니라 본문의 promhttp 지표(`go_goroutines`)까지 단언합니다. lint 는 테스트에도 errorlint 등이 적용되므로 오류를 `%w` 로 감쌉니다. 변이 evidence 에는 패키지 경로를 함께 적고, 빈 endpoint 변이는 panic 출력이 정상 실패 신호입니다.
- initSignal 은 호출마다 영구 대기 고루틴이 남습니다(signal.Stop 으로도 끝나지 않음). 신호 수신 테스트는 initSignal 을 한 번만 호출하는 하나의 테스트에서 하고 SIGHUP 은 Notify 등록 뒤(initSignal 반환 뒤)에만 보내며 chDone 수신을 제한 시간으로 단언합니다. runDaemon 실패 경로 테스트는 initSignal 을 한 번 더 호출하지만 chSigs 에 쓰지 않으므로 chSigs 를 읽는 고루틴이 더 생기지만 chDone 은 용량 1 이고 단언은 chDone 수신으로만 하므로 테스트 순서에 의존하지 않습니다. 끝에 `signal.Stop(chSigs)` 와 chDone 비우기를 합니다.

## 5. verify 항목
- `go mod tidy && go mod vendor && go generate ./... && go test -count=1 -race ./... && golangci-lint run` 통과, 추가로 `go test -race -shuffle=on -count=1 ./cmd/...` 를 2회 실행해 순서 의존이 없음을 확인. 수기 합계 76%(722) 이상(전후 표는 scratch 의 계산 스크립트). go.mod/go.sum 무변경. vendor 는 커밋하지 않음.
- 오류 단언은 앞단 문자열(`failed to initialize handlers`, `could not connect to the database`, `database ping error`)과 errors.Is 로 하고 OS 문구는 쓰지 않습니다. runSubscribe 실패는 errors.Is 로 센티널을 확인하고 2번째 QueueBind 실패 시 3번째 호출과 ConsumeMessage 가 없음을 `Times` 로 단언합니다.
- 변이 표본(스크립트로 scratch 복사본에만 적용, 해당 테스트만 -run 으로 실행해 실패를 확인하고 원복. 변이 전후 go test 출력을 evidence 로 기록). 변이와 실패해야 할 테스트:
  - control resolveString 의 빈 값 검사 제거: Test_resolveString
  - control resolveUUID 의 uuid.Nil 비교 제거: Test_resolveUUID 의 잘못된 UUID 케이스
  - control printJSON 들여쓰기 변경: Test_printJSON
  - control cmdServiceStart 의 `transfer-type` 플래그 기본값을 `attended` 로 변경: Test_cmdFlags
  - control run* 의 errors.Wrap 제거: Test_run_initHandlerFailure
  - control initCache 의 errors.Wrapf 제거: Test_initCache_error
  - manager initProm 의 조건 `endpoint == "" || listen == ""` 를 정확한 부정 `endpoint != "" && listen != ""` 로 반전: Test_initProm_disabled(변이 판정 케이스는 endpoint 가 있고 listen 만 빈 케이스로 고정하며 DefaultServeMux 에 패턴이 없음을 단언)
  - manager initSignal 의 SIGHUP 을 Notify 목록에서 제거: Test_initSignal(실제 SIGHUP 으로 테스트 프로세스가 종료되어 실패하며 일반 FAIL 출력이 없음을 evidence 에 기록)
  - manager initSignal 의 chDone 전송 제거: Test_initSignal
  - manager runSubscribe 의 오류 wrap 을 `return nil` 로 바꿔 오류를 삼킴, 그리고 wrap 을 `return err` 로 바꿈: Test_runSubscribe_error(오류 문자열 앞단 `could not run the subscribe handler` 단언 포함, 두 변이 모두 실패해야 함)
  - manager runRequestListen 의 오류 wrap 을 `return nil` 로 바꿈, 그리고 `return err` 로 바꿈: Test_runRequestListen_error(앞단 `could not run the listen handler` 단언 포함)
  - (기존 pkg/subscribehandler 테스트도 같은 변이를 잡으며 cmd 테스트가 추가로 보장하는 것은 wrap 접두사와 실패 뒤 호출 부재임)
  - subscribehandler Run 의 QueueBind 오류 반환을 continue 로 변경: Test_runSubscribe_error 의 2번째 QueueBind 실패 케이스
  - manager runDaemon 의 DB 오류 wrap 제거: Test_runDaemon_dbFailure
- `git diff --name-only origin/main...HEAD` 가 bin-transfer-manager/ 안의 테스트 파일과 설계 문서뿐임.

## 6. 롤백
테스트 파일만 추가하므로 PR revert 로 되돌립니다. 운영 영향 없음.

# voip-kamailio-proxy 단위 테스트 커버리지 75% 설계 (VOIP-1652)

기준(origin/main accc03e)은 수기 코드 73.3% (242/330 statement) 이고 이 서비스에는 생성 코드와 mock 이 없어 원본과 같다. 목표는 75% 이상(248 이상)이며 CI 소수점 차이 여유를 위해 76% (251 이상)를 권장한다. 제품 코드는 바꾸지 않고 `_test.go` 파일만 추가하며 go.mod/go.sum 은 바꾸지 않는다(logrus test hook 은 이미 의존하는 logrus 모듈의 하위 패키지다).

## 미커버 분석

- 도달 가능: listenhandler Run 의 listenRun 오류 로그 (1), cmd main() 의 인터페이스 조회 실패 조기 반환 (8), initVariable 의 잘못된 SIP_TIMEOUT 폴백 (2), initProm 의 listen 실패 로그 (1). 합계 12 statement 이며 달성 시 254/330 = 77.0% 이며 76% 기준(251)까지 여유는 3 statement 다. 구현 뒤 실측으로 재확인한다.
- 도달 불가 또는 제외: initVariable 의 BindPFlag/BindEnv 오류 panic 분기(등록 직후 조회라 오류가 날 수 없음, 24), processRequest 의 handler 오류 -> 400 (3, 핸들러가 오류를 반환하지 않음), v1_providers.go 의 json.Marshal 실패 분기 (2, string 필드만 가진 구조체라 실패 불가. 체커 오류 -> 500 은 기존 테스트가 이미 커버), main() 의 RabbitMQ 연결 이후(40, 실제 브로커 필요), getKamailioID 의 net.Interfaces 오류 분기 (1), health.go 의 UDP dial, SetDeadline, Write 오류 분기 (6, OS 오류 주입 필요). 미커버 88 = 12 + 24 + 3 + 40 + 1 + 6 + 2.

## 검증 항목 (verify)

공통: 로그 검증은 logrus test hook 을 쓴다. 훅 격리는 `old := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))` 후 t.Cleanup 에서 `ReplaceHooks(old)` 로 복원하고 `test.NewGlobal()` 을 쓴다. 대기는 AllEntries 폴링에서 메시지를 부분 일치로 찾으며 상한은 5초 이상이다. 로그 단언은 "오류 로그가 남는다" 까지만 보증하므로, 레벨이 Error 이고 `func` 필드가 있는 로그(Run, main)는 기대값이며 합성 오류 문자열이 메시지에 들어가는지 함께 본다.

1. Run 오류 로그: QueueCreate 가 합성 오류(`declare failed`)를 반환하는 mock 으로 (a) Run 을 별도 goroutine 에서 호출하고, QueueCreate 가 gate 채널로 막혀 있는 동안 상한 시간(2초 이상) 안에 Run 이 nil 로 반환하는지 본다(`go` 제거 변이에서 동기 호출로 멈추지 않도록. 실패 경로에서도 gate 채널을 닫는다), (b) 막힘을 푼 뒤 goroutine 이 레벨 Error, func=Run, "Could not execute the listen handler" 와 `declare failed` 를 포함한 항목을 남기며, (c) ConsumeRPC 기대값을 두지 않아 호출되면 gomock 이 실패한다.
2. main() 조기 반환: interfaceName 을 존재하지 않는 이름으로 두고 main() 을 별도 goroutine 에서 호출해 done 채널을 5초 상한으로 기다리고, 시간 초과면 Fatal 한다(조기 반환이 깨지면 브로커 연결 재시도 루프에 들어가 동기 호출은 CI 가 멈추므로). 반환 뒤 레벨 Error, func=main, "Could not get kamailio ID" 와 해당 인터페이스 이름이 항목에 있는지 확인한다. 전역은 main 의 반환을 확인한 뒤에만 복원하고, 시간 초과 시에는 남은 goroutine 과 경합하지 않도록 전역을 건드리지 않은 채 Fatal 한다. rabbitMQAddress 는 init() 이후 기본값이 비어 있지 않으므로 goroutine 시작 전에 `""` 로 설정하고 main 반환 확인 뒤에만 복원한다.
3. initVariable: t.Parallel 을 쓰지 않는다. `pflag.CommandLine = pflag.NewFlagSet(name, pflag.ContinueOnError)` 로 새 FlagSet 을 두고(pflag 재등록 panic 과 ExitOnError 종료 방지), os.Args 를 `[]string{"x"}` 로 바꾸며(방어용), `viper.Reset()` 을 호출하고, CommandLine, os.Args, 패키지 전역 6개를 t.Cleanup 으로 복원한다. viper 전역은 복원 수단이 없으나 이후 viper 를 읽는 코드가 없어 영향이 없다(테스트 파일 주석에 viper.Reset 사용을 남긴다). Reset 뒤에도 AutomaticEnv 는 initVariable 안에서 다시 켜진다. 6개 환경변수는 t.Setenv 로 매 케이스마다 설정하거나 `""` 로 비워 호스트 환경 영향을 없앤다(viper 는 빈 값을 미설정으로 취급). (a) SIP_TIMEOUT=bogus 이면 sipTimeout 5s 폴백. (b) SIP_TIMEOUT=3s 이면 3s. (c) 환경변수 5개를 합성값으로 설정하면 해당 전역에 반영, 모두 비우면 initVariable 을 거친 전역 값이 기본값 상수와 같다(기존 Test_defaults 는 상수만 고정하므로 중복이 아님). (d) os.Args 에 6개 키 모두 서로 다른 합성값 플래그(`--rabbitmq_address=...` 등)를 주면 각 전역에 해당 값이 반영된다(키 교차 대입과 BindPFlag 키 오타를 잡기 위함). (e) 같은 키에 플래그와 환경변수를 함께 주면 플래그가 우선한다(우선순위 플래그 > 환경변수 > 기본값, 실측으로 확인됨).
4. initProm: `127.0.0.1:99999` 처럼 항상 실패하는 주소를 쓴다(`:0` 이나 임의 포트는 성공하므로 쓰지 않는다). init() 이 기본 주소를 이미 점유하므로 그에 기대지 않는다. 엔드포인트는 호출마다 다른 경로(원자 카운터)로 `-count>1` 의 기본 mux 중복 등록 panic 을 피한다. 항목은 레벨 Error, "Could not start prometheus server" 와 `99999`(invalid port 메시지)를 함께 포함해야 한다(init() 이 낸 같은 문구의 로그와 구분).
5. 체커 오류 500 경로는 기존 테스트("sip checker error returns 500")가 커버하므로 새 테스트를 만들지 않는다. 제외 항목은 위 분석의 도달 불가 분기이며 테스트하지 않는다.
6. 여유가 부족할 때의 후보: 실측이 254 보다 낮으면 먼저 원인을 확인한다. 추가 후보는 현재 없고, 76% 미달이면 default 에 보고한다.

## 변이 확인 계획

모든 변이는 scratch 복사본에서만 적용하고 끝난 뒤 프로세스 종료를 확인한다. 컴파일 실패로 막히지 않도록 적용 방식을 지정한다. (1) Run 의 `h.listenRun()` 을 `error(nil)` 로 치환, (2) Run 의 `log.Errorf` 를 `_ = log` 로 치환, (3) Run 의 `go` 키워드 제거(즉시 반환 관측이 잡아야 함), (4) main() 의 조기 return 제거(5초 상한에서 실패해야 함), (5) 폴백 값 5s 를 1s 로, (6) 키마다 BindPFlag 첫 번째 인자(viper 키) 문자열 오타 1건씩과 서로 다른 키 교차 대입(예: interfaceName 에 prometheus_endpoint 대입), (7) initProm 의 오류 로그 줄 삭제(조건식에 err 가 쓰여 컴파일됨). BindEnv 의 환경변수 이름 오타는 AutomaticEnv 가 같은 이름으로 매핑하므로 동치 변이이며 대상에서 제외한다. scratch 변이 실행은 `-timeout` 으로 감싼다. SIP_TIMEOUT=3s 반영 누락(ParseDuration 결과 무시)은 (a)(b) 케이스로 잡힌다. 확인 뒤 worktree 의 `git diff --stat origin/main...HEAD` 에 voip-kamailio-proxy 안의 `_test.go` 와 이 문서만 있음을 확인한다.

## 완료 기준

수기 합계 76% 이상, 기존 테스트 통과, `-race -shuffle=on -count=3` 통과, golangci-lint 통과, go.mod/go.sum 비변경, PR 파일은 voip-kamailio-proxy/ 안에만 있다.

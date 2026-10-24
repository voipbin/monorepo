# VOIP-1651 bin-webhook-manager 단위 테스트 커버리지 70% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main(99b59c386) 기준 수기 코드 68.6% (713/1039), 원본 60.9%를 실측하여 카드 수치와 일치.
- 관련 PR: VOIP-1647(#1515), VOIP-1649(#1561)가 머지되어 기준선에 반영됨. 열린 중복 PR 없음.
- 미커버 326 stmt 중 301 이 cmd/webhook-control(164), cmd/webhook-manager(137)이고 나머지 25 는 internal/config 3, listenhandler 11, webhookhandler 9(webhook.go 6, main.go, message.go, urlvalidator.go 각 1), accounthandler 1, cachehandler 1.
- 필요량: 70% 는 728 stmt(+15), 71% 는 738 stmt(+25).
- 구조상 테스트 불가(제외, 합 135 stmt): webhook-manager main 9, runDaemon 31(Connect 와 `<-chDone` 블로킹), run 49(sockhandler 구체 타입 직접 생성, 주입점 없음), initCache 4(Redis Connect). webhook-control main 3(Execute, log.Fatalf), initHandler 7(DB Connect), initCache 4, initWebhookHandler 28(구체 sockhandler 직접 생성). 추가로 runSendTo* 의 initHandler 호출 이후와 initCommand 의 CheckErr, BindPFlags 오류 경로, config 미커버 3 stmt(main.go:33-34, 65-66, 69-70)도 전부 도달 불가(bindConfig 가 플래그를 직접 정의한 직후 Lookup 하므로 BindPFlag 오류가 날 수 없고, BindEnv 키는 비어 있지 않으며, 플래그를 미리 정의하면 오류가 아니라 redefined panic). initHandler 이후는 로컬 DB 포트 상태에 따른 환경 의존이라 제외.
- 상한: 위 135 만 제외하면 (1039-135)/1039 = 87%. cmd 중 제외하지 않은 미커버는 166 stmt(control 122, manager 44)이며 이 중 runSendTo* 의 initHandler 이후 등 환경 의존분은 제외하므로 실제 도달 가능량은 약 80 stmt 이상(control 약 55 이상, manager 약 27). 87% 는 135 만 뺀 이론 상한이며 위 추가 불가 항목(listenhandler 11, config 3, runSendTo* 잔여 등)을 빼도 +25 는 cmd 보조 함수만으로 달성 가능.
- listenhandler 미커버 11 stmt: v1_webhooks.go 의 핸들러는 항상 (response, nil) 을 반환하고 Marshal 실패는 도달 불가이므로 processRequest 의 err 분기는 공개 경로로 도달하지 못함. 이 11 stmt 는 제외하고 계획에 넣지 않음(404 와 whHandler 500 은 기존 테스트가 있음).

## 2. 대상과 산정

| 대상 | 방법 |
|---|---|
| cmd/webhook-control | resolveUUID(빈 값, 잘못된 형식, 정상), printJSON(정상, 마샬 불가 값), runSendToCustomer/URI 의 입력 검증 오류(handler 초기화 이전에 반환), initCommand(하위 명령, 플래그 기본값, 루트 실행으로 PersistentPreRunE 와 검증 오류 도달) |
| cmd/webhook-manager | runListen/runSubscribe(sockhandler mock 으로 QueueCreate, TopicCreateWithKind, QueueBind 오류 전파와 성공 경로), initProm(비활성 경로, 루프백 주소 활성 경로), initSignal(SIGHUP 수신 시 chDone) |
| accounthandler, cachehandler | 도달 가능한 분기만: account 갱신 실패 시 계속 진행(account.go:57-59), 마샬 불가 값(cachehandler handler.go:31-32). config 와 webhookhandler 잔여는 도달 가능한 것만 구현 단계에서 판단 |

예상: cmd 보조 함수에서 약 70 stmt 이상(검토자의 시제품 측정으로 cmd 보조 함수만으로 80% 대 도달 확인), 수기 합계 약 75% 이상. 필요 추가 25(71% 기준).

## 3. Verify 항목 (설계 단계에서는 코드 없음)

1. 수기 합계 71% 이상. 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 gens/, mock_*.go, DO NOT EDIT 제외, 같은 블록의 중복은 최대 count 로 합산하는 스크립트(scratch)로 계산.
2. `go test -race -count=1 ./...` 와 `go test -race -count=2 ./cmd/...`(DefaultServeMux 중복 등록, 시그널 오염 검증), golangci-lint, `git fetch origin main` 후 scripts/check-test-conventions.sh 통과.
3. 비테스트 .go 변경 0 (`git diff --stat origin/main...HEAD` 가 bin-webhook-manager 의 _test.go 와 이 문서뿐이며 go.mod, go.sum, vendor 변경 0).
4. 변이 표본: 제품 코드를 임시로 바꿔(예: UUID nil 검사, runSubscribe 오류 전파, initProm 의 endpoint 분기, 시그널 처리) 새 테스트가 실패함을 확인하고 원복.
5. 자기비교 금지: 기대값은 리터럴로 쓰고 제품 함수 출력으로 만들지 않음. 네트워크는 쓰지 않음.
6. 전역 상태 대책 (모두 t.Parallel 금지):
   - viper: 테스트마다 `viper.Reset()` 후 cleanup 에서도 Reset. cobra `Execute` 는 반드시 `SetArgs` 지정(미지정 시 os.Args 의 -test.* 를 읽음).
   - config.LoadGlobalConfig 는 sync.Once 라 cmd 패키지에서 되돌릴 수 없으므로, 루트 실행 테스트는 한 번 실행되어도 다른 테스트의 기대에 영향이 없도록 config.Get() 값에 의존하는 테스트를 만들지 않음.
   - http.DefaultServeMux: 등록은 되돌릴 수 없고 같은 패턴 재등록은 panic. endpoint 마다 호출 1회로 제한하고 호출은 패키지 전역 sync.Once 안에서 하여 `-count=2` 에서도 재호출하지 않음. DefaultServeMux 검증(httptest)은 Once 밖에서 매 실행 수행. ListenAndServe 는 정상 주소(`127.0.0.1:0`)면 블로킹하므로 시작 로그 관찰까지만, 오류 분기는 잘못된 주소(`127.0.0.1:99999`)와 다른 endpoint 로 한 번 더 호출해 `Prometheus server error` 로그로 동기화. 표준 logrus 로거에 훅을 달고 cleanup 에서 제거(config.Bootstrap 이 표준 로거 포매터와 레벨을 바꾸므로 그 상태에 의존하지 않음).
   - signal: initSignal 테스트 전에 chDone 을 비우고, 타임아웃 있는 select 로 수신하며, 신호를 보낸 뒤 chDone 수신을 확인한 다음에 cleanup 에서 `signal.Stop(chSigs)`.
   - stdout: printJSON 캡처 시 os.Stdout 을 교체하고 cleanup 에서 복원.
   - runListen/runSubscribe 성공 경로: ConsumeRPC/ConsumeMessage 가 goroutine 에서 호출되므로 mock 의 DoAndReturn 에서 채널로 신호를 보내고 테스트가 그 신호를 기다린 뒤 종료(ctrl.Finish 는 goroutine 호출 이후). 소비자 오류 반환 경로도 같은 방식으로 추가.
7. 측정 후 `git status` 로 go.mod, go.sum, vendor 가 `GOFLAGS=-mod=mod` 때문에 바뀌지 않았음을 확인. cobra 출력은 SetOut/SetErr 로 버퍼에 받고 오류 메시지를 리터럴로 단언. initProm 비활성 경로는 Debug 로그만 남기므로 표준 로거 레벨을 Debug 로 올렸다 복원하거나 DefaultServeMux 에 endpoint 가 없음을 단언.

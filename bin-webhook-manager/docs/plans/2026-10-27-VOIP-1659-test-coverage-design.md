# VOIP-1659 bin-webhook-manager 단위 테스트 커버리지 85% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main a47f5a2a1 기준 수기 코드 82.3% (statement 1089, 미커버 193), 원본 73.4%.
- 85.0%까지 30개(926/1089), 85.5%까지 36개(932/1089)를 더 커버해야 함.
- 이 경로의 최근 커밋은 VOIP-1647, 1649, 1651 커버리지 작업, VOIP-1648 문서, 의존성/로그 변경, SQUARE-66 이며 열린 중복 PR 은 없음.
- 미커버 구성: cmd/webhook-manager 93, cmd/webhook-control 74, listenhandler 11, webhookhandler 12, config 3.
- 도달 가능 상한: cmd 시도의 실측으로 수기 약 87%이며 85% 이상이므로 block 하지 않음 (2절).
- 도달 불가: 2절 참조.

## 2. 대상과 방법 (비테스트 .go 변경 없음)

실측 근거: 격리된 scratch worktree 에서 아래 시도를 임시로 써서 측정하니 cmd 만으로 수기 86.9% (설계 확정용 측정이며 저장소에 포함하지 않음). 따라서 도달 가능 상한은 약 87~88%.

| 대상 | 방법 |
|---|---|
| cmd/webhook-manager | `*config.Get()` 에 닫힌 루프백 주소(net.Listen 으로 받은 포트를 즉시 Close)를 직접 설정하고 원복. runDaemon 이 DB 연결 실패를 감싸 반환, initCache 가 연결 실패 반환, miniredis 로 initCache 성공, main 은 os.Args 를 `--help` 로 두고 호출(구성/Bootstrap/Execute) |
| cmd/webhook-control | 같은 방식. initHandler 의 DB 실패, initCache 실패/성공, runSendToCustomer/runSendToURI 를 Execute 없이 직접 호출(viper.Set)해 "failed to initialize handlers" 오류 확인, main `--help` |
| pkg/webhookhandler | activeflowHandler.Get 오류, 잘못된 envelope JSON 의 조기 반환, 계정 조회 실패 시 서명 없이 전송, 잘못된 URL 거부(urlvalidator url.Parse 오류) |
| pkg/listenhandler | 기본 오류 처리는 처리 함수들이 항상 nil 오류를 반환해 도달 불가. 제외 |

도달 불가로 제외: config Bind 오류, json.Marshal 오류 분기, listenhandler 기본 오류 처리, 공인 IP dial, DB 연결 이후의 run/runSend* 구간, os.Exit/log.Fatalf 분기, nil httpClient 분기(webhook message.go:55), 고루틴 안 sendMessage 오류 로그(webhook.go 56, 113, 비동기 타이밍 의존이면 포기).

## 3. Verify 항목 (설계 단계에서는 코드 없음)

1. 수기 합계 85.5% 이상 (gens/, mock_*.go, DO NOT EDIT 제외, 중복 블록 최대 count).
2. `go test -race -count=1 ./...`, golangci-lint, scripts/check-test-conventions.sh 통과.
3. `git diff --stat origin/main...HEAD` 가 bin-webhook-manager 의 _test.go 와 이 문서뿐.
4. 변이 표본: 제품 코드를 임시로 바꿔(DB/캐시 오류 래핑 문구, 서명 분기, 오류 반환) 새 테스트가 실패함을 확인하고 원복.
5. 기대값은 리터럴로 쓰고 제품 함수 출력으로 만들지 않음. 합성값만 사용.
6. 격리: 연결은 닫힌 루프백 포트만(즉시 거부, 외부 네트워크 없음). run*/runDaemon 은 Execute 를 거치지 않고 직접 호출해 LoadGlobalConfig 의 sync.Once 가 config.Get() 을 덮어쓰지 않게 함(`--help` 는 PersistentPreRunE 를 실행하지 않음). 전역 config.Get(), viper, logrus 포매터/레벨, os.Args, 시그널(runDaemon 은 전체에서 1회만 호출하고, initSignal 이 남기는 고루틴은 종료되지 않으므로 signal.Stop 으로 등록만 해제하고 chSigs/chDone 을 비움)을 테스트 후 원복. runDaemon 은 prometheus endpoint 를 빈 값으로 둬 전역 http mux 중복 등록을 피함. go-redis 재시도로 지연이 생기면 해당 케이스를 최소화.

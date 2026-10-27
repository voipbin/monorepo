# VOIP-1659 bin-message-manager 단위 테스트 커버리지 상향 설계

## 1. 이슈 분석
- 기준선(origin/main a47f5a2a1): 수기 합계 83.36% (1147/1376). 목표 85.0% 이상, 권장 85.5%(1177 이상).
- 합계의 단위는 cover profile 의 NumStmt 가중 합이다(Go 는 직선 구간을 여러 블록으로 쪼개 블록마다 같은 NumStmt 를 붙이므로 cmd/ 비중이 크게 보인다).
- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 `gens/`, `mock_*.go`, 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일의 블록을 빼고, 같은 블록의 중복은 최대 count 로 합친다. `go test` 가 출력하는 패키지별 수치는 mock 을 포함하므로 이 값과 다르다.
- 패키지별 수기 미커버 statement: cmd/message-manager 106, cmd/message-control 76, pkg/requestexternal 18, pkg/listenhandler 13, pkg/dbhandler 8, internal/config 5, pkg/messagehandler 2, models/message 1.
- 미커버의 대부분은 외부 연결(DB, RabbitMQ)이 있어야 도달하는 cmd/ 분기와 외부 HTTP 응답 처리이며, 이는 대상에서 제외한다. 시그니처 변경 없이 도달할 수 있는 분기만 대상으로 한다. 제품 코드(비테스트 .go) 변경은 없고 추가 파일은 모두 `*_test.go` 이다.

## 2. 대상과 시험 경로 (예상 증가 statement)
| 대상 | 시험 경로 | 증가 |
|---|---|---|
| message-control `main` | `os.Args` 를 `--help` 로 실행 | 2 |
| message-control `initCache` | miniredis 성공, `127.0.0.1:0` 로 연결 거부 실패 | 4 |
| message-manager `main` | `--help` 실행 | 3 |
| message-manager `initProm` | 활성 경로 성공, 점유된 listener 주소로 바인드 실패 | 5 |
| Send 의 dbUpdateTargets 실패 | `MessageUpdateTargets` 가 오류를 반환 | 2 |
| listenhandler Run 의 ConsumeRPC 오류 | mock 이 sentinel 오류 반환 | 1 |
| listenhandler 응답 직렬화 실패 4곳(List, Post, Get, Delete) | mock handler 가 `TMCreate` 연도 10000 인 Message 반환, 상태 500 확인 | 8 |
| listenhandler `errorResponse` typed-nil | `(*VoipbinError)(nil)` 전달, 500 이고 DataType 이 VoipbinError 가 아님 | 2 |
| models/message `CreateWebhookEvent` 실패 | `TMCreate` 연도 10000 | 1 |
| dbhandler `MessageCreate` PrepareFields 실패 | `Targets[0].TMUpdate` 연도 10000 (최상위 시각은 TimeNow 로 덮이므로 Targets 안에 둔다) | 1 |
| dbhandler `MessageUpdate` PrepareFields 실패 | `fields[message.FieldTargets] = []target.Target{{TMUpdate: <연도 10000>}}` (최상위 `*time.Time` 필드는 변환 없이 통과하므로 실패하지 않는다) | 1 |
| dbhandler `MessageList` ToSql 실패 | filters 에 `commondatabasehandler.NotEq{Value: <Value() 가 오류를 반환하는 driver.Valuer>}`, token 은 비어 있지 않은 값(빈 값이면 `TimeGetCurTime` 기대가 필요), 단언 접두 `could not build query. MessageList` | 1 |
| dbhandler `MessageUpdate` ToSql 실패 | `message.FieldText` 같은 변환을 타지 않는 필드에 `ToSql()` 이 오류를 반환하는 string 기반 `sq.Sqlizer`, `TimeNow` `Times(1)`, 닫힌 DB 에서도 접두 `could not build query. MessageUpdate` 가 나오면 Exec 에 도달하지 않은 것 | 1 |
| requestexternal 요청 생성 실패 | messagebird 2, telnyx 1: `var nilCtx context.Context`, wrap 메시지로 단언 | 3 |
| 합계 | | 35 → 1182/1376 = 85.90% |
- 도달 불가로 제외: 응답 처리가 `client.Do` 성공을 요구하는 messagebird 65-86(자체 Transport 와 상수 URL이라 가로채려면 프로세스 전역 상태인 리졸버와 루트 인증서를 바꿔야 해서 제외), messagebird 의 취소된 ctx 시험은 기존 시험이 이미 있어 신규 대상이 아니다. telnyx 의 payload Marshal 실패. DB 접속이나 RabbitMQ 접속이 끝나야 실행이 완료되는 cmd/ 분기(접속이 없으면 무한 재시도하므로 블록 진입만 세는 시험은 goroutine 누수와 로그 소음 때문에 배제), config 의 바인딩 에러, 나머지 squirrel ToSql 에러(MessageCreate, MessageDelete, messageGetFromDB: 입력 타입이 고정), Delete 의 PrepareFields, listenhandler `processRequest` 의 VoipbinError, ErrNotFound 분기(handler 가 raw 오류로 돌려주는 경로는 `url.Parse` 오류뿐). loopback 가짜 MySQL 서버는 쓰지 않는다(필요량을 가짜 서버 없이 채울 수 있고 프로토콜 구현과 flaky 위험만 늘기 때문).

## 3. 설계 원칙
- 신규 시험 파일에 작은 `logrus.Hook` 구현을 두어 `Fire` 안에서 시험이 정한 유일 문자열을 포함한 메시지만 걸러 `Entry.Message` 를 문자열로 복사해 버퍼 채널로 비차단(`select` + `default`) 전송한다(다른 goroutine 의 로그로 버퍼가 차서 신호가 유실되는 것을 막는다). 시험은 `select` 로 수신하고 `time.After` 는 상한(실패 판정)으로만 쓴다. 등록은 `logrus.StandardLogger().ReplaceHooks` 로 하고 `t.Cleanup` 에서 원래 hook 맵, 로그 레벨, formatter 를 복원한다. 신호는 시험이 만든 유일한 값(경로, 주소, sentinel 문자열)을 담은 메시지로만 거른다.
- `initProm` 은 `http.Handle` 이 해제 불가한 전역 등록이므로 패키지 수준 `atomic.Int64` 카운터로 호출마다 유일한 경로(`/metrics-test-<n>`)를 만든다(`-count`, `-shuffle` 에서 중복 등록 panic 방지). `/metrics` 는 기존 `Test_initProm_disabled` 가 미등록을 단언하므로 신규 시험에서 쓰지 않는다. listen 주소는 `127.0.0.1:0`, 실패 시험은 `net.Listen("tcp","127.0.0.1:0")` 로 점유해 시험 끝까지 열어 둔 주소를 넘기고 로그는 접두 `Prometheus server error` 로 판별한다. 성공 단언은 시작 로그와 `http.DefaultServeMux.Handler` 가 반환한 핸들러를 `httptest.NewRecorder` 로 호출해 200 을 확인한다. 성공 경로의 listener goroutine 은 프로세스 종료까지 남는다.
- Send 시험은 동기 구간(`Create`)의 `MessageCreate`, `MessageGet`, `PublishWebhookEvent`(Created)를 `Times(1)` 로 기대하고 `BillingV1AccountIsValidBalanceByCustomerID` 는 valid=true 로 기대하고(`Create` 전에 호출된다), `CustomerV1CustomerGet` 은 내부 시스템 ID 가 아닌 일반 UUID 로 Verified 고객을 반환하게 하며, `id` 는 `uuid.Nil` 이 아닌 고정 값을 넘긴다(`UUIDCreate` 기대를 피한다). goroutine 에서는 `MessageUpdateTargets` 만 sentinel 오류를 `Times(1)` 로 반환하게 하고 Updated 쪽 `MessageGet`, `PublishWebhookEvent` 는 기대를 두지 않는다. provider handler 는 map 순회 순서가 무작위이므로 두 provider mock 의 성공을 `AnyTimes` 로 열고 `MessageUpdateTargets(ctx, id, gomock.Any(), gomock.Any())` 로 받는다. 신호는 goroutine 쪽 로그(`handler:` 와 sentinel 을 함께 포함한 메시지)에서만 받는다(`dbUpdateTargets` 내부 로그는 문구가 달라 먼저 깨어나는 것을 피한다).
- dbhandler 의 PrepareFields 시험은 기존 `error_paths_test.go` 의 `newClosedDB` 를 쓴다. 반환 오류가 `could not prepare fields. MessageCreate`(`MessageUpdate`)로 시작하는지 단언하면, PrepareFields 를 건너뛸 경우 closed-DB 오류가 나므로 Exec 에 도달하지 않음의 증거가 된다. `utilHandler.TimeNow` 는 `Times(1)` 로 기대한다.
- ConsumeRPC 시험은 sentinel 오류를 `Times(1)` 로 반환하게 하고 sentinel 을 포함한 로그를 기다린 뒤 종료한다.
- `main` 시험은 `os.Args`, `os.Stdout`(`os.Pipe`), logrus level, formatter, hooks, `viper.Reset()` 을 `t.Cleanup` 으로 복원한다. `resetControlState`, `captureStdout` 은 `package main` 안에 있어 import 할 수 없으므로 message-manager 쪽 시험 파일에 같은 패턴을 복제한다(도움말이 pipe 버퍼보다 작다는 전제를 주석에 남긴다). logrus hook 타입도 패키지마다 복제하며 이름만 고유하면 된다. 인자는 `--help` 하나로 고정한다(실패 시 `os.Exit` 로 시험 바이너리가 종료되는 위험을 주석에 남긴다). `--help` 는 `PersistentPreRunE` 를 실행하지 않아 `LoadGlobalConfig` 의 sync.Once 를 소모하지 않는다. 도움말 단언은 두 `main` 을 따로 한다(message-manager 는 서브커맨드가 없다): message-manager 는 `message-manager [flags]` 를, message-control 은 `message-control [command]` 와 `migrate` 를 포함하는지 확인한다(`Use` 이름 변경 변이가 이 `Usage:` 줄로 잡힌다). `t.Parallel` 을 쓰지 않는다.
- 로그 레벨이 Info 보다 낮아지는 경우에 대비해 hook 시험은 시작 시 `SetLevel(DebugLevel)` 을 명시하고 `t.Cleanup` 에서 복원한다. hook 필터는 성공 시험에서는 `starting on` 과 전체 경로(`strconv` 로 만든 `/metrics-test-<n>`)를 메시지 끝 일치(`HasSuffix`)로(`-1` 과 `-10` 의 부분 일치를 막는다), 바인드 실패 시험에서는 `Prometheus server error` 와 점유한 주소를 모두 포함하는 메시지로 정한다(시작 로그에도 주소가 들어가므로 주소만으로는 거르지 않는다).
- initCache 시험은 miniredis(`go.mod` 직접 의존성) 주소를 `config.Get()` 에 넣고 기존 원본 설정 복원 패턴을 따른다. 실패 시험은 go-redis 재시도로 수백 ms 지연될 수 있고, 단언은 wrap 접두 `could not connect to the cache` 로 한다. 빈 주소는 기본값 `localhost:6379` 로 접속하므로 쓰지 않는다. 비밀번호는 비우고 `user:pass@host` 형태 DSN 을 쓰지 않는다.
- telnyx 시험이 `http.DefaultTransport` 를 교체하는 기존 시험과 섞이지 않도록 `t.Parallel` 을 쓰지 않는다. 각 시험은 반환 오류의 wrap 메시지, mock 의 엄격한 호출 기대, 응답 상태 코드로 단언하며 자기비교 시험을 만들지 않는다.

## 4. Verify
- `GOFLAGS=-mod=mod go test -count=1 -race -coverprofile=cp.out ./...` 통과, 수기 합계 85.5% 이상(계산은 위 제외 규칙을 구현한 짧은 스크립트로 하며 스크립트와 `cp.out` 은 저장소에 커밋하지 않고 scratch 에 둔다).
- `go test -count=20 -race -shuffle=on ./...` 로(pkg/dbhandler 의 기존 시험은 공유 DB 에 같은 id 를 삽입해 `-count=2` 에서 이미 실패하므로 이 패키지는 `-count=1 -shuffle=on` 으로 따로 돌린다) 중복 등록 panic 과 flaky 를 확인하고 seed 를 기록한다(성공 시험마다 listener goroutine 이 남는다). 신규 시험 이름은 `Test_...` 규칙을 따르고 기존 시험은 수정하지 않는다.
- `golangci-lint run -v --timeout 5m`, `go vet`, `go generate ./...`, `go mod tidy && go mod vendor` 후 go.mod, go.sum diff 없음, `bash scripts/check-test-conventions.sh`, 추가 파일에 `grep -nE '://[^ ]*:[^ ]*@'` 결과 없음.
- 변이 표본(각각 실패해야 함): `initCache` 의 wrap 제거, `initProm` 의 핸들러 등록 제거, Send 의 dbUpdateTargets 오류 return 제거(goroutine 의 nil 역참조 panic 으로 시험 바이너리가 FAIL 하는 방식으로 검출되며, 시험이 먼저 끝나면 확률적이므로 `-count=20 -shuffle=on` 에서 최소 1회 FAIL 이 나오는 것을 완료 조건으로 한다), ConsumeRPC 오류 로그 제거, 직렬화 실패 시 500 을 200 으로 변경(4곳 각각), `MessageCreate`/`MessageUpdate` 의 PrepareFields wrap 제거, telnyx 와 messagebird 의 wrap 각각 제거, `CreateWebhookEvent` 오류 삼키기, `initProm` 의 Errorf 로그 제거, 두 `main` 의 `Use` 이름 변경, `MessageList`/`MessageUpdate` 의 ToSql wrap 제거, `errorResponse` 의 ToResponse 실패 분기 제거.
- `git diff --stat origin/main...HEAD` 가 bin-message-manager/ 안만 가리키고 비테스트 .go 변경이 없음.

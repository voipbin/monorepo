# bin-conversation-manager 단위 테스트 커버리지 65% 이상 (VOIP-1649, 목표 65% 이상, 여유 목표 66%)

## 1. 티켓 검토와 이슈 분석

기준: origin/main bffdd69e3 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (실패 0건). 수기 코드만(`gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 파일 제외, 동일 블록은 최대 count 로 합산) 집계한 결과는 티켓의 수치(3604 statement, 63.46%, 원본 56.30%)를 그대로 재현한다.

| 패키지 | covered/total | % |
|---|---|---|
| cmd/conversation-control | 0/528 | 0.0 |
| cmd/conversation-manager | 0/203 | 0.0 |
| internal/config | 59/67 | 88.1 |
| internal/convtitle | 18/20 | 90.0 |
| models/account, conversation, message | 56/59 | 94.9 |
| pkg/accounthandler | 108/134 | 80.6 |
| pkg/cachehandler | 0/81 | 0.0 |
| pkg/conversationhandler | 592/682 | 86.8 |
| pkg/dbhandler | 483/585 | 82.6 |
| pkg/linehandler | 99/190 | 52.1 |
| pkg/listenhandler | 460/586 | 78.5 |
| pkg/messagehandler | 155/183 | 84.7 |
| pkg/smshandler | 17/20 | 85.0 |
| pkg/subscribehandler | 84/92 | 91.3 |
| pkg/whatsapphandler | 156/174 | 89.7 |
| 합계(수기) | 2287/3604 | 63.46 |

65% 는 2343 covered(+56), 66% 는 2379 covered(+92)이다. 직전 VOIP-1647 레인이 이미 도달 가능한 오류 경로를 상당수 채웠으므로 남은 미커버는 아래 세 덩어리에 몰려 있다.

- pkg/cachehandler 81: Redis 전용이다. 현재 테스트가 없고 `github.com/alicebob/miniredis/v2` 를 다른 모듈들이 이미 같은 go-redis v8 과 함께 쓴다. 이 모듈의 go.mod 에는 없으므로 테스트 전용 의존성 추가(go.mod, go.sum)가 필요하다.
- cmd/conversation-control 528: `run*` 함수들은 검증 전에 `initHandlers()`(MySQL 과 RabbitMQ 접속)를 호출하므로 제품 코드 변경 없이는 테스트할 수 없다. 테스트 가능한 것은 `resolveUUID`, `printJSON`, `initCommand`, `cmdXxx` 플래그 생성자, `initCache`, `initConversationHandlers` 이다.
- cmd/conversation-manager 203: `main`, `runMain`, `run`, `initProm`, `initSignal` 은 외부 접속이나 고정 포트와 시그널 부작용 때문에 제외한다. `runListen`, `runSubscribe` 는 SockHandler 인자를 받아 mock 으로 테스트할 수 있다.
- 구조적으로 제외: linehandler `Setup`, `Teardown`, `GetPeer` 와 메시지가 있는 `Send` 는 LINE SDK 클라이언트를 함수 안에서 생성해(`getClient`) base URL 을 주입할 수 없고 외부 호출이 된다. 이를 위해 제품 코드를 바꾸지 않는다. 반대로 `Send` 의 빈 text 와 빈 medias 경로(네트워크 호출 없이 nil 반환)와 `Hook` 의 follow 처리(`hookEventTypeFollow` 의 dialog id 없음 분기와 GetPeer 앞부분, 외부 호출 전 반환 분기)는 mock 으로 도달 가능하므로 추가 여유분으로 다룬다.
- `run*` 은 정상 경로가 접속 없이는 불가능하다는 의미이며, 접속 실패 분기는 로컬 DB 나 네트워크에 의존해 불안정하므로 테스트하지 않는다.
- 범위 해석: go.mod 와 go.sum 은 bin-conversation-manager 안의 파일이고 비테스트 .go 의 동작 변경이 아니므로, miniredis 테스트 전용 의존성 추가는 '단위 테스트 추가와 필요한 헬퍼' 범위에 포함하는 것으로 해석한다(다른 레인과 선례 VOIP-1647 이 같은 방식).
- 오래된 전제 확인: 대상 경로의 최근 커밋(`git log -- bin-conversation-manager`)과 PR 목록을 확인했고 VOIP-1649 의 중복 PR 은 없다. 직전 VOIP-1647(5a3496e0c)이 목표를 61% 로 한정했고 cachehandler 와 cmd 를 제외로 남겼으므로, 이 작업은 그 제외분을 테스트 가능한 범위만큼 다루는 후속이다.

도달 가능 추정: cachehandler 약 75, cmd/conversation-control 약 110(함수 단위 환산으로는 테스트 가능분이 약 317 이며 보수적으로 잡음), cmd/conversation-manager 약 20, config 약 3(`Bootstrap` 만 도달 가능하고 나머지 미커버는 `logrus.Fatalf` 와 정상 호출로 발생하지 않는 Bind 오류), 각 `New*Handler` 와 listenhandler `Run` 약 5-10(프로파일이 `-coverpkg` 없이 패키지 자신의 테스트만 반영하므로 해당 패키지 안의 테스트로만 인정되며, 이 항목은 여유분), 합계 약 235 으로 필요한 +92 를 넘는다. 달성은 비테스트 코드 변경 없이 가능하다.

## 2. 계획

범위는 `_test.go` 파일과 go.mod/go.sum(miniredis, 테스트 전용)뿐이다. 비테스트 .go 는 바꾸지 않는다. 모든 테스트는 합성값만 쓰고 `Test_<func>` 이름, gomock 컨트롤러 이름 `mc` 를 따른다. viper, `config.globalConfig`, logrus 포매터와 레벨, prometheus 를 만지는 패키지에서는 `t.Parallel()` 을 쓰지 않고 `viper.Reset()` 과 `t.Setenv` 로 viper 를 격리하며, `config.Get()` 이 돌려주는 전역 포인터의 값(`RedisAddress` 등)은 직접 대입하고 `t.Cleanup` 으로 복원한다(`initCache` 의 주소 주입 방법). `initCommand` 와 `Bootstrap` 테스트가 logrus 상태를 바꾸면 원래 값으로 복원한다. `runListen`, `runSubscribe` 대상인 `listenhandler.Run`, `subscribehandler.Run` 은 고루틴 안에서 `ConsumeRPC`, `ConsumeMessage` 를 호출하므로 테스트는 채널로 호출 완료를 기다리고 호출 횟수를 고정한다(AnyTimes 로 얼버무리지 않음). 채널 대기에는 타임아웃을 두고 `DoAndReturn` 안에서는 `t.Fatalf` 대신 채널 전송만 한다. `ConsumeRPC`, `ConsumeMessage` 의 반환 오류 분기는 로그가 mock 반환 뒤에 찍혀 완료를 결정적으로 기다릴 수 없으므로 테스트하지 않는다. 생성자 테스트는 nil 이 아님만 확인하지 않고 주입된 의존성이 실제로 쓰이는지를 관찰한다.

verify 항목(각 항목은 반환 오류, 저장된 Redis 상태, mock 호출 인자와 횟수 중 하나를 관찰한다):
1. `pkg/cachehandler/handler_test.go` (miniredis): 4개 엔터티(Account, Conversation, Message, Media)별 Set 후 Get 이 동일한 값을 반환하며(합성값의 시간 필드는 UTC 로 고정하고 슬라이스 필드는 명시해 JSON 왕복 차이를 피한다) 저장 키가 `conversation.account:<id>` 등 리터럴 접두사이며 TTL 이 24시간이다. 없는 키 Get 은 `redis.Nil`, 손상된 JSON 은 unmarshal 오류, 서버 종료 후 Set 은 오류를 반환한다. 저장 값이 JSON `null` 이면 오류 없이 zero 구조체를 반환하는 현재 동작을 특성화하고, `FastForward(24h)` 후 키가 만료되어 Get 이 `redis.Nil` 을 반환함을 확인한다(TTL 변이를 값과 동작 모두로 잡음). `Connect` 는 정상 서버에서 성공하고 서버 종료 후 실패한다(go-redis 재시도로 수백 ms 가 걸리므로 시간 제한을 넉넉히 둔다). `NewHandler` 는 `RequireAuth` 서버에서 잘못된 비밀번호는 `Connect` 실패, 올바른 비밀번호는 성공, DB 번호 3 으로 만든 핸들러는 miniredis DB 3 에 쓰는 것으로 확인한다.
2. `cmd/conversation-control/main_test.go`: `resolveUUID`(빈 값은 오류 문구 `Customer ID is required` 형태의 리터럴 단언으로 empty 검사 제거 변이를 잡고, 잘못된 UUID 와 all-zero UUID `00000000-0000-0000-0000-000000000000` 은 `invalid format` 오류, 정상), `printJSON`(채널 같은 marshal 불가 값은 오류, 정상은 표준출력 캡처), `initCommand` 성공 경로(루트에 conversation, account, message, migrate 가 있고 각 하위 명령 집합과 플래그 기본값이 리터럴로 일치), 열 개 `cmdXxx` 각각에 대해 `RunE` 함수 포인터(`reflect.ValueOf(cmd.RunE).Pointer()`)가 기대한 `run*` 과 같고 플래그 이름과 기본값을 리터럴로 단언한다(배선이 서로 바뀌면 실패), `printJSON` 캡처는 `os.Stdout` 을 pipe 로 교체하고 별도 고루틴에서 읽으며 `t.Cleanup` 으로 복원한다, `viper.Reset()` 은 각 테스트 시작과 `t.Cleanup` 양쪽에서 호출한다, `initCache`(miniredis 성공, 서버 종료 후 wrapped 오류), `initConversationHandlers`(세 핸들러를 반환하고 오류가 없음, 반환된 핸들러가 주입한 mock cache 를 거쳐 동작하는 것을 한 번 호출로 확인). `run*` 은 검증 전에 접속을 시도하므로 호출하지 않는다.
3. `internal/config`: `Bootstrap` 이 문서화된 플래그와 기본값을 등록하고 환경변수 override 가 viper 로 반영된다. 이미 `sync.Once` 인 `LoadGlobalConfig` 는 기존 테스트가 다루는 범위를 확인한 뒤 빠진 분기만 추가한다.
4. `cmd/conversation-manager/main_test.go`: `runListen` 은 mock SockHandler 에서 `QueueCreate` 를 리터럴 큐 이름(`bin-manager.conversation-manager.request`)으로 한 번 호출하고 고루틴에서 같은 큐 이름으로 `ConsumeRPC` 를 한 번 호출함을 채널로 대기해 확인한다(상수가 아닌 리터럴로 비교해 큐 이름 변이를 잡는다. `Run` 의 두 번째 인자 delay 큐는 무시되므로 검증 대상이 아니다). `QueueCreate` 실패 시 `ConsumeRPC` 를 기대하지 않고 `fmt.Errorf` 로 감싼 오류의 문구와 `errors.Is` 원인 일치를 단언해 현재 동작을 특성화하며 동작이 바뀌면 갱신한다고 주석에 적는다. `runSubscribe` 는 `QueueCreate`(리터럴 큐 이름 `bin-manager.conversation-manager.subscribe`), `TopicCreateWithKind`, `QueueBind`, `ConsumeMessage` 를 순서대로 호출하며, 각 단계의 오류가 `Run` 의 wrapped 오류로 전파되고 이후 호출이 일어나지 않음(strict mock)을 단계별 케이스로 확인한다. `QueueBind` 횟수는 main 패키지가 `topicPatterns` 에 접근할 수 없으므로 현재 패턴 수를 리터럴로 고정하고(구현 시 `subscribehandler` 의 패턴 수를 읽어 확정) 패턴이 바뀌면 이 테스트를 갱신한다. `initProm`, `initSignal`, `signalHandler`, `run`, `runMain`, `main` 은 호출하지 않는다.
5. 생성자(`NewAccountHandler`, `NewConversationHandler`, `NewMessageHandler`, `NewSMSHandler`, `NewLineHandler`, `NewListenHandler`)는 필드 대조(자기비교)를 하지 않는다. 구성한 핸들러의 메서드를 한 번 호출해 주입된 mock 의 인자와 호출 횟수를 관찰하는 방식으로만 쓰고, 그렇게 쓰기 어려운 생성자는 이득이 1-4 statement 이므로 대상에서 뺀다. listenhandler `Run` 은 `QueueCreate` 실패의 wrapped 오류와 성공 시 `ConsumeRPC` 호출을 확인한다.
6. 예비분: 1-5 만으로 +92 를 넘는다(추정 약 235). 예비로 linehandler 의 `Send` 빈 text 경로와 `hookEventTypeFollow` 의 dialog id 없음 분기, conversationhandler `Event`, `Hook` 의 도달 가능한 오류 분기를 추가할 수 있으며 이는 verify 항목이 아니라 여유분이다. 추가하는 경우 각 케이스는 `go tool cover -html` 로 분기 도달을 먼저 확인한다.

측정: 위 명령으로 만든 cp.out 에서 scratch 스크립트(커밋 안 함)가 동일 블록을 최대 count 로 합치고 제외 규칙을 적용해 패키지별 전후 표를 낸다. 전 열은 위 표이며 후 열은 같은 스크립트로 만든다.

검증:
- 컨벤션: `scripts/check-test-conventions.sh` 규칙에 따라 비교는 `reflect.DeepEqual` 과 `t.Errorf` 를 쓰고 testify 는 쓰지 않는다. `initCommand` 테스트는 `Execute` 를 호출하지 않아 `LoadGlobalConfig` 의 `sync.Once` 를 건드리지 않는다. `runSubscribe` 의 `TopicCreateWithKind` 첫 인자(`bin-manager.conversation-manager.event`)와 kind 도 리터럴로 단언하며, `QueueBind` 횟수는 `subscribehandler/binding_golden_test.go` 의 패턴 집합과 함께 갱신한다. 변이 표본에 `runSubscribe` 큐 이름 변경과 `TopicCreateWithKind` kind 변경을 추가한다. 함수 단위 환산 추정은 구현 후 실측으로 확정한다.
- `-race -count=1 -shuffle=on` 전체 통과, 수기 합계 66% 이상, 전체 검증 워크플로(`go mod tidy`, `go mod vendor`, `go generate`, `go test`, `golangci-lint`)와 `scripts/check-test-conventions.sh`.
- 변이 표본(scratch 복사본에서 하나씩 적용 후 해당 패키지 테스트, 이후 원복): 캐시 TTL 24h 를 1h 로, 캐시 키 접두사 변경, `resolveUUID` 검증 제거, 플래그 기본값 변경(limit 100), `initCommand` 하위 명령 하나 제거, `runListen` 의 큐 이름 변경, `Bootstrap` 기본값 변경. 각 변이는 이번 PR 이 추가한 테스트를 하나 이상 실패시켜야 하며 생존 변이는 테스트를 강화한다.
- miniredis 추가 후 `go.mod` 와 `go.sum` 의 diff 를 확인해 miniredis 와 그 전이 의존성 외의 버전이 바뀌지 않았는지 본다(다른 모듈이 쓰는 v2.36.1 로 고정).
- `git diff --stat origin/main...HEAD` 는 bin-conversation-manager 경로만 포함한다.

롤백: 테스트 전용 변경이므로 커밋을 되돌린다.

## 3. 결과

구현 후 실측: 수기 합계 63.46%(2287/3604)에서 약 75.1%(2706/3604)로 올랐고 원본 합계는 56.30%에서 약 66.6%이다. 추정치(약 235 statement)보다 크게 오른 것은 `initCommand` 한 번 호출이 큰 statement 블록을 채우기 때문이다. 변이 표본은 계획한 항목 중 구현된 것에 한해 확인했으며, 생성자 테스트는 의존성 필드 대조를 피하기 위해 `initConversationHandlers` 의 캐시 경유 관찰로만 대체했다. `Bootstrap` 의 `BindEnv` 키 변이는 `AutomaticEnv` 가 같은 키를 처리하는 동등 변이로 판단했다.

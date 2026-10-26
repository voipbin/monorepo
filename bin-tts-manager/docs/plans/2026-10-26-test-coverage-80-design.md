# bin-tts-manager unit test coverage 80% (VOIP-1657)

## 1. 티켓 검토

- 검토 결과: 유효. origin/main 161e9748b 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 를 재측정했고 테스트 실패는 0건입니다.
- 수기 합계(gens/, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count 로 병합): 2215/2877 = 76.99%. 원본 합계는 PR 본문에 적습니다.
- 80.0% 는 2302개, 80.5% 는 2316개 이므로 추가로 87개(80.0%), 101개(80.5%) 를 커버해야 합니다.
- 같은 목적의 열린 PR 과 중복 없음. 최근 이 서비스 경로의 커밋은 VOIP-1648(streaming 오류 처리)과 VOIP-1647(커버리지 60%) 입니다. 제품 코드 변경은 하지 않습니다.
- 겹치는 후속 티켓: VOIP-1416(80% 미만 패키지 일반 상향)이 audiohandler, streaminghandler 를 포함할 수 있으며 흡수하지 않습니다.

## 2. 기준선 (수기 코드, 패키지별)

| 패키지 | 커버/전체 | % |
|---|---|---|
| cmd/tts-control | 53/110 | 48.2 |
| cmd/tts-manager | 4/142 | 2.8 |
| internal/config | 74/77 | 96.1 |
| models/message | 1/1 | 100.0 |
| models/speaking | 8/9 | 88.9 |
| pkg/audiohandler | 90/216 | 41.7 |
| pkg/buckethandler | 35/35 | 100.0 |
| pkg/cachehandler | 49/50 | 98.0 |
| pkg/dbhandler | 95/102 | 93.1 |
| pkg/listenhandler | 463/490 | 94.5 |
| pkg/speakinghandler | 176/177 | 99.4 |
| pkg/streaminghandler | 995/1276 | 78.0 |
| pkg/ttshandler | 172/192 | 89.6 |

## 3. 도달 가능 상한

외부 연결(GCP 클라이언트 생성, ElevenLabs 상수 호스트 접속, RabbitMQ, DB, Redis)이 필요한 생성자와 main, json.Marshal 및 squirrel 빌드 오류, PrepareFields 오류는 제외합니다. 시그니처 변경 없이 테스트 가능한 분기만 합산한 추정 상한은 아래 표의 합 약 122개(독립 재계산 122~147, 키프얼라이브 등 보충 항목 별도)로, 필요한 101개를 넘으므로 block 없이 진행합니다.

## 4. 추가할 테스트 (모두 로컬 전용, 외부 네트워크 없음)

| 항목 | 방법 | 추정 stmt |
|---|---|---|
| cmd/tts-control `initCommand` | 하위 명령 구성, `PersistentPreRunE` 실행, `tts create` 입력 검증 오류가 Execute 를 통해 전달됨 | 21 |
| cmd/tts-manager `runListenNormal`, `runListenPod`, `runListen` | mock sockhandler. QueueCreate 성공과 실패, 큐 이름 검증, 실패 시 panic | 18 |
| cmd/tts-manager `initSignal`, `initProm`(고유 endpoint, 로컬 listen), `runDaemon` 연결 오류 | `chSigs` 에 직접 신호 주입, initProm 은 잘못된 listen 주소로 ListenAndServe 오류 분기까지 결정적으로 도달, 잘못된 DSN(sql.Open 파싱 오류, 네트워크 없음) | 22 |
| streaminghandler `NewStreamingHandler`, `NewAWSHandler`, `NewGCPHandler`, `NewElevenlabsHandler` | 생성자가 만든 핸들러 필드 검증 | 12 |
| streaminghandler aws `Init`, `createClient` | 정적 합성 자격, 네트워크 호출 없음(호스트의 AWS_* 환경변수와 설정 파일 경로는 t.Setenv 로 격리). 이벤트 발행 mock 검증 | 25 |
| streaminghandler elevenlabs `getVoiceID` 분기 | 명시 voice, 변수 조회 오류, 변수 없음, 길이 불일치, 언어 중립 대체 | 5 |
| streaminghandler elevenlabs `runProcess`, `readConnWebsock` | 로컬 WebSocket 서버. ctx 취소, 읽기 오류 전달, 메시지 드롭 | 4 |
| streaminghandler gcp `runProcess` | 가짜 stream. 빈 오디오 건너뜀, 서버 측 종료 시 stream 정리, ctx 취소 종료, websocket 쓰기 오류 | 14 |
| streaminghandler websocket `websocketWrite` 사전 취소 ctx | 로컬 WebSocket, 이미 취소된 ctx 를 넘기면 프레임 0개 전송과 ctx 오류 반환을 단언(페이싱 select 의 ctx 분기는 ticker 와 경합하므로 best-effort 이며 합계에서 제외) | 0 |
| audiohandler `savePCMAsWAV` 쓰기 오류 | 쓰기가 실패하는 대상(Linux 의 /dev/full) | 1 |
| streaminghandler `runWebSocketRead` 정상 종료 분기 | 로컬 WebSocket 서버가 정상 종료 close 프레임을 보냄. logrus test hook 으로 Error 항목이 없음과 doneCh 닫힘을 단언 | 1 |

중단 기준: 각 묶음 뒤에 합계를 측정하고 80.5% 에 못 미치면 gcp `runKeepalive` 틱 분기(4초 간격 상수, 실제 대기 약 4초)와 aws SayFlush ctx 분기, start.go 성공 반환, listenhandler Run 의 ConsumeRPC 오류 분기를 추가합니다.

## 5. 테스트 규칙과 verify 항목

규칙 (공통)

- 기존 함수 이름과 mock 패턴을 따르며 테스트 이름은 `Test_` 로 시작하고 gomock 컨트롤러 변수는 `mc` 입니다. 합성 값만 쓰고 자격 값은 의미 없는 문자열을 주소와 다른 문장에서 대입합니다. 같은 규칙을 테스트 주석과 PR 본문에도 적용합니다.
- sleep 에 의존한 판정을 하지 않고 채널과 컨텍스트로 동기화합니다. 타임아웃은 안전망으로만 씁니다.
- 전역 상태는 cleanup 으로 복구합니다: viper 는 `viper.Reset`, logrus 의 formatter, level, hook 은 원래 값으로, `chSigs`/`chDone` 는 아래 규칙, 환경 변수는 `t.Setenv` 로 관련 키(DATABASE_DSN, RABBITMQ_ADDRESS 등)를 비웁니다. `config.LoadGlobalConfig` 는 once 이므로 프로세스 첫 호출의 값이 고정됩니다. runDaemon 테스트는 `config.Get()` 의 필드를 직접 설정하고 defer 로 원복합니다.

항목별 규칙

- tts-control initCommand: `SetArgs`, `SetOut`, `SetErr`, `SilenceUsage` 를 쓰고 `tts create` 는 검증 오류(call-id 누락, 잘못된 UUID, text 누락, 잘못된 provider)로만 종료시킵니다. 유효 입력은 initHandler 가 RabbitMQ 에 연결하므로 금지이고 `migrate` 하위 명령은 실행하지 않으며 `cobra.CheckErr` 와 `log.Fatalf` 가 `os.Exit` 를 부르는 경로는 피합니다. viper `AutomaticEnv` 로 호스트의 TEXT, PROVIDER, LANG 같은 환경 변수가 검증 경로에 새지 않도록 `t.Setenv` 로 비웁니다.
- tts-manager runListen*: 큐 이름은 `gomock.Any` 가 아닌 정확한 값(일반 큐 이름과 `%s.%s` pod 큐. `QueueCreate` 의 첫 인자로 단언하며 지연 큐 인자는 `Run` 이 쓰지 않으므로 대상이 아닙니다)으로 매칭합니다. `Run` 은 QueueCreate 성공 뒤 `go ConsumeRPC(...)` 를 띄우므로 ConsumeRPC 의 `DoAndReturn` 에서 채널로 동기화하고 normal 과 pod 두 번 모두 기다립니다. runListen 의 panic 은 recover 로 잡아 메시지를 단언합니다.
- initSignal: 호출마다 goroutine 하나와 `signal.Notify` 가 남으므로 호출 1회당 `chSigs` 에 정확히 1개를 주입하고 `chDone` 을 소비하며 cleanup 에서 `signal.Stop(chSigs)` 를 호출합니다. 실제 OS 시그널은 보내지 않습니다. runDaemon 오류 경로도 initSignal 을 부르므로 같은 규칙을 씁니다.
- initProm: `DefaultServeMux` 에 등록 해제 수단이 없고 같은 패턴 재등록은 panic 이므로 등록 호출을 한 건으로 최소화하고 endpoint 에 원자 카운터를 붙여 `-count` 반복마다 고유하게 합니다. ListenAndServe 오류는 잘못된 listen 주소(예: 포트 누락)로 즉시 발생시키며 logrus test hook 을 채널로 기다려 관측하고 cleanup 에서 hook 을 복구합니다.
- runDaemon: 잘못된 DSN 이 `sql.Open` 단계에서 실패하는 경로만 사용하고 네트워크 접속 경로는 쓰지 않습니다.
- aws Init/createClient: `t.Setenv` 로 `AWS_PROFILE`, `AWS_CONFIG_FILE`, `AWS_SHARED_CREDENTIALS_FILE`(임시 빈 파일), `AWS_EC2_METADATA_DISABLED=true`, `AWS_ENDPOINT_URL*` 를 지정해 호스트 설정을 차단합니다. 오류 분기는 존재하지 않는 `AWS_PROFILE` 로 유도해 Init 의 오류 래핑을 확인하고, 성공 경로에서는 `client.Options().Region` 이 `eu-central-1` 임을 단언하고 만든 `AWSConfig` 의 `Cancel` 은 cleanup 에서 호출합니다.
- gcp runProcess: 가짜 `StreamingSynthesizeClient` 의 `Recv` 는 채널 기반이고 ctx 취소로 풀립니다. `CloseSend` 를 구현해 호출 여부를 기록하고 `Client` 는 nil 로 둡니다. `GCPConfig` 의 `Ctx`, `StreamCtx`, `StreamCancel`, `processDone` 를 모두 초기화합니다. PlayStarted 와 PlayFinished 이벤트를 mock 으로 EXPECT 하고 끝에서 `processDone` 이 닫혔는지 채널로 기다립니다. 서버 종료 케이스는 `Stream == nil` 을, websocket 쓰기 오류는 닫힌 로컬 연결로 유도합니다.
- savePCMAsWAV 쓰기 오류: `/dev/full` 이 없거나 열 수 없으면 `t.Skip` 합니다(1 stmt 이므로 80.5% 에 영향 없음).
- 생성자 테스트: 반환 타입을 단언한 뒤 주입한 의존성, podID, 자격 값이 필드에 들어갔는지와 `mapStreaming` 이 nil 이 아닌 빈 맵인지 단언합니다.

verify 항목

1. 수기 합계 80.5% 이상. 측정은 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 gens/, mock_*.go, DO NOT EDIT 파일을 빼고 같은 블록은 최대 count 로 병합하는 고정 스크립트로 계산하며 묶음마다 반복합니다.
2. 변경 파일이 bin-tts-manager/ 안에만 있음, 저장소 루트에서 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 통과.
3. `go test -race -count=5 -shuffle=on ./cmd/... ./pkg/streaminghandler/ ./pkg/audiohandler/` 반복 통과.
4. 변이 검사(각각 대응 테스트가 실패해야 함): runListenNormal 큐 이름 변경 - runListen 큐 이름 테스트, initSignal 의 chDone 전달 제거 - initSignal 테스트, initProm 오류 분기 로그 제거 - initProm 오류 테스트, createClient 지역 상수 변경 또는 오류 래핑 제거 - aws Init 테스트, elevenlabs getVoiceID 우선순위 변경 - 우선순위 테스트, gcp runProcess 의 빈 오디오 건너뜀 반전 - gcp runProcess 테스트, runListenPod 큐 이름 변경 - pod 큐 테스트, tts-control provider 검증 분기 변경 - 검증 오류 테스트, readConnWebsock 의 드롭 분기 제거 - 드롭 테스트, 생성자 필드 대입 누락 - 생성자 테스트, websocketWrite 의 사전 취소 검사(ctx.Err() 반환) 제거 - 사전 취소 ctx 테스트, runWebSocketRead 정상 종료도 Error 로 기록 - 정상 종료 테스트, gcp runProcess 서버 종료 분기의 Stream 정리 제거 - 서버 종료 테스트, aws Init 의 Initiated 이벤트 발행 제거 - aws Init 테스트, tts-control 의 call-id 와 text 검증 제거 - 검증 오류 테스트.
5. `go mod tidy` 후 go.mod, go.sum 변경 없음. golangci-lint 통과.

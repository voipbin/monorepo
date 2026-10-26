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

외부 연결(GCP 클라이언트 생성, ElevenLabs 상수 호스트 접속, RabbitMQ, DB, Redis)이 필요한 생성자와 main, json.Marshal 및 squirrel 빌드 오류, PrepareFields 오류는 제외합니다. 시그니처 변경 없이 테스트 가능한 분기만 합산한 추정 상한은 아래 표의 합 약 124개(독립 재계산 124~147, 키프얼라이브 등 보충 항목 별도)로, 필요한 101개를 넘으므로 block 없이 진행합니다.

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
| streaminghandler websocket `websocketWrite` ctx 취소 | 로컬 WebSocket, 취소된 ctx | 1 |
| audiohandler `savePCMAsWAV` 쓰기 오류 | 쓰기가 실패하는 대상(Linux 의 /dev/full) | 1 |
| streaminghandler `websocketConnect` 미커버 분기, `runWebSocketRead` 정상 종료 분기 | 로컬 WebSocket 서버가 연결을 끊거나 바이너리 메시지를 보냄 | 2 |

중단 기준: 각 묶음 뒤에 합계를 측정하고 80.5% 에 못 미치면 gcp `runKeepalive` 틱 분기(4초 간격 상수, 실제 대기 약 4초)와 aws SayFlush ctx 분기, start.go 성공 반환, listenhandler Run 의 ConsumeRPC 오류 분기를 추가합니다.

## 5. 테스트 규칙과 verify 항목

- 기존 함수 이름과 mock 패턴을 따르며 테스트 이름은 `Test_` 로 시작하고 gomock 컨트롤러 변수는 `mc` 입니다.
- 시간과 외부 상태에 의존하지 않습니다. 동기화는 채널로 하고 sleep 에 의존한 판정은 하지 않습니다. 전역 상태(viper, `DefaultServeMux`, `chSigs`)는 테스트 안에서 복구하거나 고유 이름을 씁니다.
- 합성 값만 사용합니다. 자격 값은 의미 없는 문자열이며 주소와 같은 리터럴에 두지 않습니다.
- verify: (1) 수기 합계 80.5% 이상. (2) 변경 파일이 bin-tts-manager/ 안에만 있음. (3) `go test -race -count=5` 로 새 테스트 반복 통과. (4) 표본 변이(예: runProcess 의 빈 오디오 처리 반전, runListenNormal 의 큐 이름 변경)를 넣으면 새 테스트가 실패함. (5) `go mod tidy` 후 go.mod, go.sum 변경 없음. (6) golangci-lint 통과.

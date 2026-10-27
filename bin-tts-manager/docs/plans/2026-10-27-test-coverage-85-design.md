# bin-tts-manager unit test coverage 85% (VOIP-1659)

## 1. 티켓 검토

- 검토 결과: 유효. origin/main a47f5a2a1 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정한 수기 합계는 약 81.8% (2353~2354/2877, 실행 간 1 statement 편차), 테스트 실패 0건. 85.0% 까지 92~93개, 85.5% 까지 106개 statement 가 더 필요.
- 같은 목적의 열린 PR 과 중복 없음. 이전 커버리지 PR(VOIP-1647, 1652, 1657)이 streaminghandler 와 listenhandler 를 이미 올렸고 남은 큰 미커버는 audiohandler 와 gcp 경로.
- 제품 코드(비테스트 .go) 변경 없음. 테스트 파일만 추가한다.

## 2. 기준선 (수기 코드, gens/ mock_*.go DO NOT EDIT 제외, 같은 블록은 최대 count 로 병합)

| 패키지 | 전체 stmt | 커버 | % |
|---|---|---|---|
| cmd/tts-control | 110 | 71 | 64.5 |
| cmd/tts-manager | 142 | 55 | 38.7 |
| internal/config | 77 | 74 | 96.1 |
| pkg/audiohandler | 216 | 92 | 42.6 |
| pkg/buckethandler | 35 | 35 | 100.0 |
| pkg/cachehandler | 50 | 49 | 98.0 |
| pkg/dbhandler | 102 | 95 | 93.1 |
| pkg/listenhandler | 490 | 463 | 94.5 |
| pkg/speakinghandler | 177 | 176 | 99.4 |
| pkg/streaminghandler | 1276 | 1063 | 83.3 |
| pkg/ttshandler | 192 | 172 | 89.6 |

models/message(1/1)와 models/speaking(8/9)은 표에서 생략하고 합계에는 포함한다.

## 3. 구조 분석

미커버의 대부분이 pkg/audiohandler/gcp.go(약 103), pkg/audiohandler/main.go(16), pkg/ttshandler/main.go(20), streaminghandler 의 gcp.go 와 elevenlabs.go 에 있다. audiohandler 의 GCP 경로는 `*texttospeech.Client` 를 필드로 가지므로 테스트에서 로컬 gRPC 서버(127.0.0.1, 평문, 임의 포트)에 연결한 실제 클라이언트를 주입할 수 있어 제품 코드 변경이 필요 없다. 도달 가능 상한은 필요량(106)을 넘는다(audiohandler 와 ttshandler 만으로 약 130 statement). 서버 Run, main, 외부 연결 생성자, json.Marshal 실패 같은 분기는 대상에서 제외한다.

## 4. 추가할 테스트 (모두 로컬 전용, 외부 네트워크 없음)

1. audiohandler gcpAudioCreate: 가짜 `TextToSpeechServer`(`UnimplementedTextToSpeechServer` 임베딩, `127.0.0.1:0` 평문)로 성공 시 파일 내용과 요청의 Voice/AudioConfig/SSML 검증, voiceID 지정 시 그대로 사용, 빈 voiceID 에서 언어별 기본 보이스, 알 수 없는 언어에서 빈 이름, 서버 오류(재시도되지 않는 `InvalidArgument`)에서 에러 전파와 파일 미생성, 존재하지 않는 하위 디렉터리 경로(권한 비트에 의존하지 않음)에서 쓰기 에러. `AudioCreate` 의 gcp 위임과 unsupported provider. 클라이언트는 `option.WithEndpoint`, `WithoutAuthentication`, 평문 gRPC dial 옵션으로 만든다(go.mod 변경 없음).
2. gcpWarmUpConnection: 서버 성공, 서버 오류(`Unimplemented`, 비치명). gcpGetClient: 존재하지 않는 자격증명 파일을 가리키는 `GOOGLE_APPLICATION_CREDENTIALS` 로 에러 래핑, 런타임에 `t.TempDir()` 에 쓴 명백한 가짜 값의 `authorized_user` 파일과 `net.Listen("127.0.0.1:0")` 후 즉시 닫은 로컬 주소를 endpoint 로 하여 성공(경고만 남고 클라이언트 반환). 닫힌 포트의 warm-up 은 컨텍스트 deadline 까지 기다리므로 deadline 200ms 컨텍스트로 호출한다. 빈 endpoint 분기(실제 googleapis 기본 호스트)는 성공 경로로 테스트하지 않는다(실패 자격증명 케이스로만 통과).
3. NewAudioHandler: 자격증명 오류에서 nil, 가짜 자격증명과 200ms 컨텍스트로 성공. NewTTSHandler: 자격증명 오류에서 nil. NewTTSHandler 성공 경로는 `context.Background()` 고정으로 warm-up 이 약 10초 걸리므로 기본 계획에서 제외하고, 중단 기준 보충용으로만 둔다. 버킷 디렉터리는 `t.TempDir()`. awsGetClient 실패 분기는 존재하지 않는 `AWS_PROFILE` 을 케이스별 `t.Setenv` 로 지정해 NewAudioHandler 의 aws 실패와 함께 검증한다. 그 외 케이스의 AWS 설정 격리로 `AWS_CONFIG_FILE`, `AWS_SHARED_CREDENTIALS_FILE` 은 임시 경로, `AWS_EC2_METADATA_DISABLED=true`, `AWS_PROFILE` 비움을 `t.Setenv` 로 설정한다.
4. 보충(조건부): 1~3 뒤 합계가 85.5% 에 못 미칠 때만, 대상 블록 목록과 외부 접속 여부 점검을 이 문서에 먼저 추가한 뒤 streaminghandler 의 websocket.go, start.go 남은 분기와 gcp runProcess 의 asterisk 쓰기 실패, cmd/tts-control 의 도달 가능 분기를 진행한다. elevenlabs 와 SayAdd 재연결은 외부 접속 위험이 있어 제외가 기본값이다.

격리와 안정성: ADC 환경변수는 `t.Setenv` 를 쓰므로 해당 테스트는 `t.Parallel()` 을 쓰지 않고, 실패 케이스 외에는 항상 자격증명 파일을 지정한다(미지정 시 메타데이터 탐색 방지). 가짜 서버가 받은 요청은 mutex 또는 버퍼 채널로 테스트 고루틴에 전달해 `-race` 에서 안전하게 하고, 서버는 `t.Cleanup` 에서 `Stop` 후 `Serve` 고루틴 종료를 확인한다. sleep 없이 컨텍스트와 채널로 동기화한다. 합성 자격증명 값은 명백한 가짜 문자열(`fake-...`)이고 주소 리터럴과 같은 리터럴에 두지 않는다.

## 5. verify

- 수기 합계 85.5% 이상, 전체 `go test -race -count=3 ./...` 통과, `golangci-lint run` 통과.
- 각 신규 테스트가 실제 분기를 검증하는지 변이 표본(요청 필드 변경, 에러 전파 제거, 파일 쓰기 제거)을 scratch 복사본에서 실행해 실패하는 것을 확인.
- 합계는 mock_*.go, gens/, DO NOT EDIT 헤더 파일을 빼고 같은 블록은 최대 count 로 병합하는 참고 스크립트로 계산한다. 성공 경로 테스트는 `strace -f -e trace=connect` 로 loopback 외 연결이 없음을 한 번 확인한다.
- PR 파일 목록이 bin-tts-manager/ 안에만 있고 비테스트 .go 파일과 go.mod, go.sum 에 변경이 없음을 `git diff --stat origin/main...HEAD` 로 확인.

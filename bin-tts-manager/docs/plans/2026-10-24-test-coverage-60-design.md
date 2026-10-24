# bin-tts-manager unit test coverage 60% (VOIP-1647)

## 1. 티켓 검토

- 검토 결과: 유효. origin/main d68e695b5 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 을 재측정한 결과 수기 합계 47.87% (1360/2841), 원본 43.92%. 카드 수치와 일치.
- 60% 까지 필요한 추가 statement: 345개. 61% 목표는 약 374개.
- 같은 목적의 열린 PR 과 중복 없음 (`gh pr list` 에서 NOJIRA-Test-coverage-tts-manager #502 가 과거에 머지됨, 이후 streaminghandler 가 확장되어 다시 낮아짐).
- 제품 코드 변경 없음. 테스트 파일만 추가.

## 2. 기준선 (수기 코드, gens/ mock_*.go DO NOT EDIT 제외, 같은 블록은 최대 count 로 병합)

| 패키지 | 전체 stmt | 커버 | % |
|---|---|---|---|
| cmd/tts-control | 110 | 0 | 0.0 |
| cmd/tts-manager | 142 | 1 | 0.7 |
| internal/config | 77 | 74 | 96.1 |
| pkg/audiohandler | 216 | 32 | 14.8 |
| pkg/buckethandler | 35 | 24 | 68.6 |
| pkg/cachehandler | 50 | 0 | 0.0 |
| pkg/dbhandler | 102 | 90 | 88.2 |
| pkg/listenhandler | 490 | 392 | 80.0 |
| pkg/speakinghandler | 177 | 154 | 87.0 |
| pkg/streaminghandler | 1240 | 412 | 33.2 |
| pkg/ttshandler | 192 | 172 | 89.6 |

표에서 models/message (1/1) 와 models/speaking (8/9) 는 생략했으며 합계 2841 에는 포함. 독립 재측정에서 streaminghandler 가 421/1240 (합계 48.19%, 필요 약 336개) 으로 나와 약 9 statement 의 측정 편차가 있음. 시간에 민감한 기존 테스트로 보이며 보수적으로 345개 기준으로 계획하고, 최종 수치는 반복 측정해 PR 본문에 기록.

## 3. 구조 분석: 테스트 가능한 것과 어려운 것

- 어려움 (제외): `cmd/*/main`, `runDaemon`, `run` (프로세스 기동, 실제 DB/Redis/RabbitMQ), `cachehandler` (실제 Redis 필요, miniredis 가 go.mod 에 없어 의존성 추가가 필요하므로 이번 범위에서 제외), `gcpGetClient`/`awsGetClient`/`gcp connect` (외부 클라우드 인증과 네트워크).
- 쉬움: streaminghandler 의 순수 로직과 로컬 WebSocket 으로 가능한 경로.

## 4. 추가할 테스트 (우선순위순, 모두 로컬 전용, 외부 네트워크 없음)

함수별 도달 가능 statement 추정(합계 약 540, 필요 345, 목표 374):

| 항목 | 추정 stmt |
|---|---|
| elevenlabs (connect, Init, runKeepAlive 틱 분기 제외) | 약 170 |
| say.go | 약 48 |
| run.go, streaming.go | 약 40 |
| gcp (재연결 성공 경로와 connect 제외) | 약 100 |
| aws 스트리머 | 약 109 |
| audiohandler (savePCMAsWAV, AudioCreate, awsAudioCreate) | 약 58 |
| cmd (resolveUUID, printJSON, 검증 오류, initProm 빈 설정) | 약 25 (하한 계산에서는 제외) |

중단 기준: 각 항목 구현 뒤 합계를 측정해 61%(약 374) 에 못 미치면 listenhandler(v1_speakings, v1_streamings) 오류 경로와 websocket.go, start.go 미커버 분기로 보충한다.

1. elevenlabs: 로컬 `httptest` WebSocket 서버 연결을 `ElevenlabsConfig.ConnWebsock` 에 직접 주입해 `SayAdd`, `SayFlush`, `SayStop`, `SayFinish`, `terminate`, `Run`, `runProcess`(오디오 디코드 오류, JSON 파싱 오류, alignment 로 Finish 완료, 서버 close), `readConnWebsock`. `connect` 는 호스트가 상수라 제외, `Init` 도 connect 에 의존해 제외. 오류 경로: 잘못된 vendorConfig 타입, nil 연결, 닫힌 연결 쓰기 실패.
2. say.go: `SayStop`, `SayFlush`, `SayFinish`, `SayAdd` 오류 경로를 가짜 streamer 로 검증. "unsupported vendor" 분기는 `getStreamerByProvider` 의 default 가 elevenlabsHandler 를 반환하므로 핸들러의 elevenlabsHandler 필드를 nil 인터페이스로 둔 경우에만 도달한다(typed nil 은 불가).
3. run.go, streaming.go: `getStreamerByProvider` 전 분기, `runStreamer`(미설정 provider, Init 실패, 성공 시 Run 호출과 종료 뒤 VendorInfo 초기화), `UpdateConnAst`, `SetVendorInfo`.
4. gcp: `getVoiceID` 3단계 우선순위, `getVoiceIDByLangGender`, `extractLangCode`, `SayStop`, `SayFlush`, `SayFinish`, `closeStreamLocked` 를 가짜 `TextToSpeech_StreamingSynthesizeClient` 로 검증. `SayAdd` 는 "session context cancelled" 오류 경로만 허용하며 `processDone` 은 닫지 않은 채널, `Ctx` 는 취소 상태로 둔다(닫힌 processDone 이면 select 가 random 으로 갈라져 `connect` 가 실제 클라우드 클라이언트를 만들 수 있으므로 금지). 재연결 성공 경로는 제외.
5. aws 스트리머: `getVoiceID`, `getVoiceIDByLangGender`, `SayFlush`(채널 비우기), `SayStop`, `SayFinish`(Cancel 후 채널 close), `SayAdd` 는 `polly.New` 에 `127.0.0.1` httptest `BaseEndpoint` 와 합성 자격(`test-access-key` 류)을 주어 성공, 서비스 오류, 스트림 읽기 오류 확인.
6. audiohandler: `savePCMAsWAV` 는 리터럴 바이트로 RIFF/WAVE/fmt/data 마커, Subchunk1Size, AudioFormat, NumChannels, SampleRate, ByteRate, BlockAlign, BitsPerSample, ChunkSize, Subchunk2Size 와 PCM 본문 위치를 검증하고 입력 길이를 두 가지로 바꿔 기대값을 독립 계산한다. 쓰기 오류는 존재하지 않는 하위 디렉터리 경로로 유도(chmod 와 root 에 의존 금지). `AudioCreate` 미지원 provider 오류. `awsAudioCreate` 는 awsClient 필드에 로컬 엔드포인트 polly 클라이언트를 주입해 voice 미결정, 서비스 오류, 성공 확인(임시 디렉터리).
7. cmd/tts-control: `resolveUUID`, `printJSON`, `runCreate` 입력 검증 오류(initHandler 이전 반환 경로만). cmd/tts-manager: `initProm` 의 빈 설정 즉시 반환 경로만(비어 있지 않은 경로는 DefaultServeMux 등록이라 재실행 시 패닉). `initSignal` 은 전역 시그널 상태를 남기므로 제외.
8. websocket.go 의 `websocketConnect` 는 기존 로컬 테스트가 있으므로 미커버 분기(MEDIA_START 가 바이너리, 읽기 오류, 연결 거부)만 추가. start.go 미커버 분기는 mock 으로.

## 5. 테스트 작성 규칙과 verify 항목

규칙:
- 외부 호출 금지: 엔드포인트는 `127.0.0.1` httptest(`:0`) 뿐, 자격은 합성 문자열, `t.Cleanup` 으로 서버와 임시 디렉터리 정리.
- 동기화는 채널 기반만 허용하고 `time.Sleep` 금지. 고루틴이 발행하는 `PublishEvent` 호출은 gomock `DoAndReturn` 에서 채널을 닫아 테스트가 기다린 뒤 종료한다(ctrl.Finish 와의 경합 방지). 단언은 "반환했고 PlayFinished 가 발행됨" 수준으로 비결정 경로(닫힌 msgCh 선택)에 의존하지 않는다.
- `websocketWrite` 의 20ms pacing 을 피하도록 오디오 payload 는 프레임 크기 이하(ulaw 160, slin 320, slin16 640 바이트)로 한다. `runKeepAlive`(10초)와 gcp `runKeepalive`(4초) 의 틱 분기는 도달 불가이므로 ctx 취소 즉시 반환 경로만 다룬다.
- 전역 상태: viper 를 쓰는 테스트는 `viper.Reset()` 과 `t.Cleanup`, `t.Parallel` 금지. logrus 전역 레벨과 훅은 바꾸지 않는다. prometheus 메트릭은 재등록하지 않고 값은 전후 delta 로만 단언한다.
- 새 테스트 파일은 함수 이름 `Test_` 규칙과 gomock 컨트롤러 이름 `mc` 를 따른다(`scripts/check-test-conventions.sh`).

verify:
- 수기 합계 61% 이상(agg 와 같은 제외 규칙), 기존 테스트 전부 통과, `golangci-lint` 통과, check-test-conventions 통과.
- `go test -race -count=3 -shuffle=on ./...` 통과, 새 테스트 추가 시간은 패키지당 1초 이내.
- 변이 검사(scratch 복사본에서만 수행, 원 worktree 제품 코드 불변): 새 테스트 함수당 1개 이상, 표본은 elevenlabs `SayAdd` 의 TotalCount 증가 제거, `runProcess` 의 Finish 종료 조건 반전, aws `SayFinish` 의 Cancel 과 close 순서 반전, aws `SayFlush` drain 제거, gcp `getVoiceID` 우선순위 변경, run.go `runStreamer` 의 SetVendorInfo 인자 제거, say.go 메시지 ID 비교 반전과 `SayFlush` VendorLock, `savePCMAsWAV` 의 ChunkSize 계산 변경. 각 변이에서 실패한 테스트 이름을 PR 본문에 기록한다.
- 변경 파일은 `bin-tts-manager/` 안의 `*_test.go` 와 이 문서뿐.

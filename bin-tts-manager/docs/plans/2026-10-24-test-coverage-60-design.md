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

1. streaminghandler/elevenlabs: 로컬 `httptest` WebSocket 서버로 `SayAdd`, `SayFlush`, `SayStop`, `SayFinish`, `terminate`, `Run`, `runProcess`(오디오 디코드 오류, JSON 파싱 오류, 정상 종료 close, alignment 로 Finish 완료), `readConnWebsock`, `runKeepAlive`. `connect` 는 호스트가 상수로 고정되어 제외(제품 코드 변경 금지). 오류 경로: 잘못된 vendorConfig 타입, nil 연결, 닫힌 연결에서 쓰기 실패.
2. streaminghandler/say.go: `SayStop`, `SayFlush`, `SayFinish` 와 `SayAdd` 오류 경로(메시지 ID 불일치, VendorConfig nil, 알 수 없는 streaming). 가짜 streamer 로 위임과 오류 전파 확인.
3. streaminghandler/run.go, streaming.go: `getStreamerByProvider` 모든 분기, `runStreamer`(미설정 provider, Init 실패, 성공 시 Run 호출과 VendorInfo 초기화), `UpdateConnAst`, `SetVendorInfo`.
4. streaminghandler/gcp.go: `getVoiceID`(우선순위 3단계), `getVoiceIDByLangGender`, `extractLangCode`, `SayStop`, `SayFlush`, `SayFinish`, `closeStreamLocked` 를 가짜 `StreamingSynthesizeClient` 로 검증, `SayAdd` 의 Send 실패 후 재연결 불가 경로.
5. streaminghandler/aws.go: `getVoiceID`, `getVoiceIDByLangGender`, `SayFlush`(채널 비우기), `SayStop`, `SayFinish`(채널 close, 중복 호출 없음), `SayAdd` 는 polly 클라이언트를 `httptest` 로컬 엔드포인트와 고정 테스트 자격으로 구성해 성공과 서비스 오류 경로 확인.
6. audiohandler: `savePCMAsWAV` 헤더 바이트와 크기 필드 검증(임시 디렉터리), 쓰기 불가 경로 오류, `AudioCreate` 미지원 provider 오류.
6b. audiohandler `awsAudioCreate`: awsClient 필드에 로컬 httptest 엔드포인트와 고정 테스트 자격의 polly 클라이언트를 주입해 voice 미결정 오류, 서비스 오류, 성공 경로 확인(저장 파일은 임시 디렉터리).
7. cmd/tts-control: `resolveUUID` 의 빈 값, 잘못된 형식, 정상, `printJSON`, `runCreate` 입력 검증 오류. cmd/tts-manager: `initSignal`, `initProm` 의 빈 설정 즉시 반환 경로.
8. streaminghandler/websocket.go (`websocketConnect` 로컬 httptest WebSocket: MEDIA_START 정상, 바이너리 메시지, 연결 거부) 와 start.go 의 미커버 분기(mock 사용). 부족하면 listenhandler(v1_speakings, v1_streamings) 오류 경로로 보충.

## 5. verify 항목

- 수기 합계 61% 이상 (agg 스크립트와 동일한 제외 규칙), 기존 테스트 전부 통과, `golangci-lint` 통과, `bash scripts/check-test-conventions.sh` 통과(Test_ 명명, gomock 컨트롤러 mc).
- 각 새 테스트는 기대값을 입력과 독립적으로 계산한 리터럴(WAV 헤더 바이트, 메시지 문자열 등)로 확인하고, 자기비교식 단언을 쓰지 않는다.
- 변이 검사 표본: elevenlabs `SayAdd` 의 TotalCount 증가 제거, `runProcess` 의 Finish 종료 조건 반전, `savePCMAsWAV` 의 ChunkSize 계산 변경, say.go 의 메시지 ID 비교 반전. 각각 새 테스트가 실패해야 한다.
- `go test -race -count=3` 으로 동시성 경로의 불안정성 확인.
- 변경 파일은 `bin-tts-manager/` 안의 `*_test.go` 와 이 문서뿐.

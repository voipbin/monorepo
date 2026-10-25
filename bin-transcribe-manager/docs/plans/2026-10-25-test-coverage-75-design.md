# bin-transcribe-manager 단위 테스트 커버리지 75% 상향

## 1. 티켓 검토와 기준선

티켓 검토: 유효. origin/main accc03e 위에서 수기 합계 기준선을 73.61%(1852/2516)로 실측했고 카드의 73.6% 와 일치합니다(`GOFLAGS=-mod=mod go test -count=1 -coverprofile`, gens/, mock_*, DO NOT EDIT 제외, 같은 블록은 최대 count). 75.0% 는 1887 이상(+35), 76% 는 1913 이상(+61) 입니다. 직전 작업(VOIP-1647)은 외부 STT SDK 의 구체 타입 때문에 `aws.go`, `gcp.go`, `processFromRecording` 을 범위에서 제외했으나, 구체 `*speech.Client` 라도 in-memory gRPC 서버(bufconn)에 연결하면 네트워크 없이 구동할 수 있음을 spike 로 확인했습니다(`speech.NewClient` 에 `WithGRPCDialOption` 으로 bufconn dialer, `WithEndpoint("passthrough:///bufnet")`, `WithoutAuthentication` 을 주면 `LongRunningRecognize` 가 완료된 operation 을 받아 성공).

미커버 상위: cmd/transcribe-manager 179, cmd/transcribe-control 153, streaminghandler/aws.go 76, gcp.go 70, transcripthandler/process.go 35, dbhandler 42, listenhandler 25, start.go 12, recording.go 10. cmd 는 모든 `run*` 가 첫 단계에서 `initHandler`(DB 와 Redis 연결)를 호출해 제외합니다. aws.go 는 SDK 클라이언트의 HTTP 스트림 계약이 커서 이번 범위에서 제외하고 gcp.go 만 다룹니다(gcp 70 + process 35 + start 12 + recording 10 = 127 로 필요량 +61 을 충분히 넘음).

dbhandler 와 listenhandler 의 잔여 분기 중 일부는 도달 가능하므로 추가 대상에 넣습니다: 캐시 적중 경로의 `TranscribeGet`/`TranscriptGet`(`transcribe.go:110,117`, `transcript.go:96,103`), 빈 필드 맵의 Update(`transcribe.go:299`, `transcript.go:237`), 빈 token 의 기본 토큰(`transcribe.go:204`, `transcript.go:156`), 행 스캔 후 nil `StreamingIDs` 정규화(`transcribe.go:30`), `processV1*` 의 URI 항목 부족 400(`v1_transcribes.go` 30,121,155,190,225). `processRequest` 의 typed error 및 not found 분기(`main.go:292,294`)는 어떤 핸들러도 해당 오류를 `nil, err` 로 반환하지 않아 도달 불가이고, marshal 오류와 SQL 빌드 오류도 도달 불가라 제외합니다. `processFromRecording` 의 빈 응답은 results 가 비어 있는 경우로 한정합니다. `Recording` 의 transcript 생성 실패는 `createTranscripts` 가 해당 항목을 건너뛰는 동작이므로 대상에 포함합니다.

## 2. 테스트 계획

제품 코드는 바꾸지 않고 `_test.go` 만 새 파일로 추가합니다(기존 테스트 파일은 수정하지 않음). 예상 증가량은 gcp.go 약 55, process.go 35, recording.go 약 8, start.go 약 8 로 약 105 statement 이며 +61 을 넘습니다. 증가량이 없는 항목(`gcpProcessResult`, `createTranscripts`, `sortTranscriptsByOffset`, 지정 provider 미초기화 경로)은 새 테스트를 만들지 않습니다.

- 공용 fake: 각 패키지 테스트 파일 안에 bufconn 위의 `speechpb.SpeechServer` 구현을 둡니다. 테스트마다 새로 만들고 `t.Cleanup` 에서 서버 `Stop()` 과 클라이언트 `Close()` 를 호출합니다. 클라이언트는 `WithGRPCDialOption`(bufconn dialer, insecure), `WithEndpoint("passthrough:///bufnet")`, `WithoutAuthentication` 으로 생성합니다. 외부 네트워크와 자격 증명은 쓰지 않습니다.
- transcripthandler(`processFromRecording`): fake 가 받은 `LongRunningRecognize` 요청의 Uri, LanguageCode, Encoding, SampleRate, `EnableWordTimeOffsets`, `EnableAutomaticPunctuation` 을 mutex 로 보호한 기록에서 읽어 단언. 문장 분할은 한 케이스에서 `.`, `?`, `!` 세 종결 부호를 모두, 종결 없는 꼬리 문장, 여러 result 에 걸친 문장, 문장 시작 단어의 ms 오프셋, results 가 비어 있는 응답을 검증. 각 종결 단어 뒤에는 후속 단어를 두고 문장 오프셋이 서로 달라야 하며(마지막 단어이면 꼬리 flush 와 구분되지 않음), RPC 시작 오류와 완료된 operation 의 오류는 gax 재시도 대상이 아닌 코드로 만들어 nil 결과와 오류를 단언. `Recording`: 파일 3종(`_in`, `_out`, 접미사 없음)이 방향별로 `Create` mock 에 전달되고, 파일 간 오프셋이 교차하는 입력이 정렬되며 동일 오프셋은 입력 순서를 유지, 한 파일 실패 시 전체 오류와 `Create` 미호출.
- streaminghandler `gcpInit`: 서버가 첫 요청을 채널로 받아 언어, 인코딩, 샘플레이트, 채널 수, `UseEnhanced`, `EnableAutomaticPunctuation`, `InterimResults` 단언. 스트림 생성 실패는 `grpc.WithChainStreamInterceptor` 가 호출 수를 세고 `PermissionDenied` 를 반환하게 해 결정적으로 만든다(`Unavailable`, `DeadlineExceeded` 는 gax 가 재시도하므로 쓰지 않음).
- streaminghandler `gcpProcessMedia` 는 `gcpRun` 을 거치지 않고 직접 호출해 동기로 검증한다. 기존 `mockGCPStreamClient` 를 확장하지 않고 새 파일에 `Send` 인자를 기록하는 fake 를 두며(`io.EOF` 와 그 외 오류 케이스 포함), loopback websocket 서버는 기존 `newTestMediaServer`, `serverWriteFrame`, `serverWaitClosed` 를 같은 패키지에서 재사용한다. 텍스트 프레임을 먼저, 빈 바이너리 프레임, 이어 바이너리 프레임을 보내고 기록이 비어 있지 않은 바이너리 프레임 2건(순서 포함)뿐임을 단언. ctx 가 이미 취소된 경우 `Send` 가 호출되지 않음도 단언.
- streaminghandler `gcpRun` 과 `runSTT`: 서버가 final 응답을 보낸 뒤 스트림을 종료하고 클라이언트 `ConnAst` 는 그동안 열어 둔다(`Recv` 가 응답을 처리한 뒤 오류로 cancel 하므로 결과 처리가 취소보다 먼저 일어남). `notifyHandler` 와 `transcriptHandler` mock 호출이 `mc.Finish()` 전에 끝나는 것을 보장한다. `gcpRun` 반환 뒤 `ConnAst` 를 닫고 `runtime.Stack` 에서 `(*streamingHandler).gcpProcessMedia` 심볼이 사라질 때까지 기한 있는 폴링으로 확인한다(`NumGoroutine` 비교는 쓰지 않음). `runSTT`: gcp client 만 초기화한다. 성공 케이스(지정 provider gcp, provider 비어 있음)는 start.go 131-137 과 첫 handler 성공 후 반환을 커버하는 용도이고 중복 제외 변이는 식별하지 못한다. 중복 제외 변이는 스트림 생성 실패 케이스(provider=gcp)에서 interceptor 의 `NewStream` 호출 수가 1(변이 시 2)인 것으로만 검출한다. 이 interceptor 는 gax 재시도 대상이 아닌 코드(`PermissionDenied`)를 반환해 재시도를 막는다. 실패 후 "No handler executed successfully" 경고로 끝나는 경로도 같은 케이스에서 반환으로 확인한다. AWS client 는 만들지 않으며 두 provider 간 순서 관찰은 범위 밖이다.
- 의존성: `google.golang.org/grpc/test/bufconn`(이미 직접 의존하는 grpc 모듈 안), `cloud.google.com/go/longrunning`, `google.golang.org/api`, `google.golang.org/genproto/googleapis/rpc`, `google.golang.org/protobuf` 는 이미 indirect 로 존재하며 직접 import 로 바뀝니다(`go mod tidy`). 새 모듈 추가는 없습니다.

## 3. verify 항목

- 수기 합계 75% 이상(권장 76%), 기존 테스트 전부 통과, golangci-lint 와 `scripts/check-test-conventions.sh` 통과, `-race -count=20` 반복 통과. 합계 기준은 75%(1887 statement) 이상, 권장은 76%(1913) 이상. 새 테스트는 `t.Parallel` 을 쓰지 않고 미디어 연결을 여는 테스트는 모두 `ConnAst` 를 닫는다.
- go.mod 변경은 indirect 표기의 direct 이동 외에 없고 비테스트 .go 변경이 없음(`git diff --stat origin/main...HEAD`).
- 변이 표본(별도 복사본에서 적용해 신규 테스트가 FAIL 하는 출력을 evidence 로 남김): `process.go` 의 `.`, `?`, `!` 판정 각각 제거, 꼬리 문장 flush 제거, 문장 리셋 제거, 시작 오프셋 갱신 조건 변경, `Recording` 경유 방향 파싱 변경, 정렬 제거, `gcpProcessMedia` 의 비바이너리 필터 제거, 빈 프레임 skip 제거, `runSTT` 의 지정 provider 중복 제외 제거(호출 수 1 -> 2), `gcpInit` 설정 값 하나 변경.
- 금지: 빈 호출과 자기비교 테스트, 외부 네트워크 의존, 결함의 위치와 발생 조건을 적는 주석.

## 4. 구현 결과와 범위 조정

수기 합계는 73.61%(1852/2516)에서 78.50%(1975/2516)로 올랐고 목표 하한과 권장치를 넘어 1절에서 추가 대상으로 적은 dbhandler 와 listenhandler 보조 항목은 구현하지 않았습니다(요구 범위를 넘는 변경을 피함). 변이 표본 15건(종결 부호 3종, 꼬리 flush, 문장 리셋, 시작 오프셋, ms 환산, 방향, 정렬, 비바이너리 필터, 빈 프레임, interim, enhanced, 중복 제외, ctx 가드)은 컴파일되는 형태로 적용했을 때 모두 신규 또는 기존 테스트가 실패시켰습니다. 방향과 정렬 변이는 기존 단위 테스트도 잡습니다.

| 패키지 | 전 | 후 |
|---|---|---|
| pkg/streaminghandler | 63.5% | 80.6% |
| pkg/transcripthandler | 59.9% | 93.9% |
| 그 밖의 패키지 | 변화 없음 | 동일 |

# bin-transcribe-manager 단위 테스트 커버리지 75% 상향

## 1. 티켓 검토와 기준선

티켓 검토: 유효. origin/main accc03e 위에서 수기 합계 기준선을 73.61%(1852/2516)로 실측했고 카드의 73.6% 와 일치합니다(`GOFLAGS=-mod=mod go test -count=1 -coverprofile`, gens/, mock_*, DO NOT EDIT 제외, 같은 블록은 최대 count). 75.0% 는 1887 이상(+35), 76% 는 1913 이상(+61) 입니다. 직전 작업(VOIP-1647)은 외부 STT SDK 의 구체 타입 때문에 `aws.go`, `gcp.go`, `processFromRecording` 을 범위에서 제외했으나, 구체 `*speech.Client` 라도 in-memory gRPC 서버(bufconn)에 연결하면 네트워크 없이 구동할 수 있음을 spike 로 확인했습니다(`speech.NewClient` 에 `WithGRPCDialOption` 으로 bufconn dialer, `WithEndpoint("passthrough:///bufnet")`, `WithoutAuthentication` 을 주면 `LongRunningRecognize` 가 완료된 operation 을 받아 성공).

미커버 상위: cmd/transcribe-manager 179, cmd/transcribe-control 153, streaminghandler/aws.go 76, gcp.go 70, transcripthandler/process.go 35, dbhandler 42, listenhandler 25, start.go 12, recording.go 10. cmd 는 모든 `run*` 가 첫 단계에서 `initHandler`(DB 와 Redis 연결)를 호출해 제외합니다. aws.go 는 SDK 클라이언트의 HTTP 스트림 계약이 커서 이번 범위에서 제외하고 gcp.go 만 다룹니다(gcp 70 + process 35 + start 12 + recording 10 = 127 로 필요량 +61 을 충분히 넘음).

dbhandler 와 listenhandler 의 잔여 분기 중 일부는 도달 가능하므로 추가 대상에 넣습니다: 캐시 적중 경로의 `TranscribeGet`/`TranscriptGet`(`transcribe.go:110,117`, `transcript.go:96,103`), 빈 필드 맵의 Update(`transcribe.go:299`, `transcript.go:237`), 빈 token 의 기본 토큰(`transcribe.go:204`, `transcript.go:156`), 행 스캔 후 nil `StreamingIDs` 정규화(`transcribe.go:30`), `processV1*` 의 URI 항목 부족 400(`v1_transcribes.go` 30,121,155,190,225). `processRequest` 의 typed error 및 not found 분기(`main.go:292,294`)는 어떤 핸들러도 해당 오류를 `nil, err` 로 반환하지 않아 도달 불가이고, marshal 오류와 SQL 빌드 오류도 도달 불가라 제외합니다. `processFromRecording` 의 빈 응답은 results 가 비어 있는 경우로 한정합니다. `Recording` 의 transcript 생성 실패는 `createTranscripts` 가 해당 항목을 건너뛰는 동작이므로 대상에 포함합니다.

## 2. 테스트 계획

제품 코드는 바꾸지 않고 `_test.go` 만 새 파일로 추가합니다.

- 공용 fake: 각 패키지 테스트 파일 안에 bufconn 위의 `speechpb.SpeechServer` 구현을 둡니다(`LongRunningRecognize`, `StreamingRecognize`). 클라이언트는 `option.WithGRPCDialOption`, `WithEndpoint("passthrough:///...")`, `WithoutAuthentication` 으로 생성합니다. 외부 네트워크와 자격 증명은 쓰지 않습니다.
- transcripthandler: `processFromRecording` 의 문장 분할(`.`, `?`, `!` 종결, 종결 없는 꼬리 문장, 여러 result, 시작 오프셋 ms 환산, 빈 응답), RPC 시작 오류, 완료된 operation 의 오류. `Recording` 의 파일명별 방향(in/out/both), 오프셋 정렬, 한 파일 실패 시 전체 오류 전파와 Create 미호출, `createTranscripts` 가 생성 실패 항목만 건너뛰고 나머지 순서를 유지.
- streaminghandler: `gcpInit` 이 서버로 보내는 첫 요청의 설정(언어, 인코딩, 샘플레이트, 채널 수, interim) 단언, 스트림 생성 실패 오류. `gcpRun` 이 서버 스트림 종료 시 nil 로 반환, `gcpProcessResult` 가 final 결과를 결과 처리기로 전달, `gcpProcessMedia`(`ConnAst` 가 실제 `*websocket.Conn` 이므로 기존 loopback websocket 서버 헬퍼를 재사용) 가 바이너리 프레임만 전달하고 비바이너리와 빈 프레임은 건너뜀. `runSTT`(start.go 의 미커버 분기) 가 지정 provider 실패 후 다음 provider 로 넘어가는 순서.
- dbhandler: 캐시 적중 시 DB 미조회, 빈 필드 Update 가 nil 이며 DB 미접근, 빈 token List 의 기본 토큰 적용, nil `StreamingIDs` 의 빈 슬라이스 저장. listenhandler: URI 항목 부족 시 400.
- 의존성: `google.golang.org/grpc/test/bufconn`(이미 직접 의존하는 grpc 모듈 안), `cloud.google.com/go/longrunning` 은 이미 indirect 로 존재하며 직접 import 로 바뀝니다(`go mod tidy`). 새 모듈 추가는 없습니다.

## 3. verify 항목

- 수기 합계 76% 이상(1913/2516 이상, 최소 1887), 기존 테스트 전부 통과, golangci-lint 와 `scripts/check-test-conventions.sh` 통과, `-race -count=20` 반복 통과.
- go.mod 변경은 indirect 표기 이동 외에 없고 비테스트 .go 변경이 없음(`git diff --stat origin/main...HEAD`).
- 변이 표본: 문장 종결 판정 한 글자 제거, 오프셋 환산 제거, `Recording` 경유의 방향 파싱 반전, 정렬 제거, 비바이너리 프레임 필터 제거, provider 순서 변경 각각에서 신규 테스트가 실패해야 함.
- 금지: 빈 호출과 자기비교 테스트, 외부 네트워크 의존, 결함의 위치와 발생 조건을 적는 주석.

# bin-transcribe-manager 단위 테스트 커버리지 60% 상향

## 1. 배경과 측정

수기 코드(`mock_*.go`, `/gens/`, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록의 중복은 최대 count 로 합산)의 서비스 합계 커버리지를 60% 이상으로 올립니다. 측정 명령은 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -race -count=1 -coverprofile=cp.out ./...` 입니다(로컬 vendor 가 go.mod 와 어긋날 수 있어 `-mod=mod` 를 사용).

기준선(수정 전)은 수기 52.46%(1314/2505 statement), 생성 코드와 mock 을 포함한 원본 47.33% 였습니다. 60% 는 1503 statement 이상(+189), 61% 는 1529 이상(+215) 입니다.

## 2. 테스트하기 어려운 구간(이번 범위에서 제외)

- `cmd/transcribe-manager`: 전부 외부 연결(DB, Redis, RabbitMQ)과 프로세스 수명에 의존합니다.
- `pkg/streaminghandler/aws.go`, `gcp.go`: 외부 STT SDK 클라이언트(`*speech.Client`, `*transcribestreaming.Client`)를 구체 타입으로 보유해 주입 지점이 없습니다.
- `transcripthandler.processFromRecording`, `NewTranscriptHandler`: 같은 이유로 GCP 클라이언트에 의존합니다.
- `cmd/transcribe-control` 의 `run*` 본문: 첫 단계에서 `initHandler`(DB 연결)를 호출합니다. `runStart` 는 연결 이전의 입력 검증까지만 테스트합니다.
- `listenhandler.processRequest` 의 typed error 와 `ErrNotFound` 분기는 `processV1*` 가 해당 오류를 반환하지 않아 `processRequest` 경유로 도달할 수 없습니다.

## 3. 테스트 구성

제품 코드는 바꾸지 않고 `_test.go` 만 추가했습니다. 기존 테스트 파일은 수정하지 않고 `<소스>_<주제>_test.go` 형식의 새 파일을 만들었으며, 함수 이름은 `Test_<대상>_<시나리오>` 입니다. 오류 경로와 입력 검증에서 반환 오류의 종류, 응답 상태 코드, mock 호출 인자와 순서를 단언합니다.

- cachehandler: miniredis 로 Get/Set 왕복(키 형식, TTL 24시간), 키 없음, 깨진 JSON, 서버 종료, `Connect`, `NewHandler`.
- dbhandler: 닫힌 DB 에서의 각 쿼리 오류가 not found 나 duplicate 로 분류되지 않음, 캐시 쓰기 실패가 DB 쓰기를 실패시키지 않음, `ErrNotFound`, `isDuplicateKeyErr`(MySQL 1062 와 sqlite 문구). 공유 sqlite 를 오염시키지 않도록 고유 ID 와 별도 연결을 씁니다.
- listenhandler: `processRequest` 의 오류 응답 매핑(typed not found 404, 일반 오류 500, 잘못된 본문 400, 기본 404), `NewListenHandler`, `Run` 의 큐 선언 순서와 오류(`ConsumeRPC` goroutine 은 채널로 동기화).
- subscribehandler: `Run` 의 큐, 토픽, 바인딩 실패, `processEventRun` 이 오류를 삼킴, 알 수 없는 이벤트 무시.
- streaminghandler: 순수 TCP 로 구현한 loopback websocket 서버로 `websocketConnect`(정상, 바이너리 첫 메시지, MEDIA_START 전 종료, dial 실패), `Start` 의 실패와 성공 경로, `Stop` 의 연결 종료와 누락 streaming, `runSTT` 가 초기화된 provider 없이 반환. 서버는 `net/http` 를 import 하지 않습니다(import 하면 bodyclose lint 가 기존 `websocketConnect` 를 지적).
- transcribehandler: `Stop` 의 종료 후 flow 실행과 각 오류 삼킴, `isValidReference` 거절 경로, 조회, 생성, 삭제, 상태 갱신, 이벤트 핸들러, `variableSet`, `startLive` 의 부분 시작 롤백, `startRecording` 실패.
- transcripthandler: `Recording` 의 파일 목록 오류와 빈 목록, `parseDirection` 경계, `toOffsetMs`, `Create`/`List`/`Delete` 오류 전파.
- cmd/transcribe-control: `resolveUUID`, `printJSON`(stdout 캡처), 명령 정의(Use, Short, 플래그와 기본값), 명령 트리, `runStart` 의 입력 검증 순서와 메시지. viper 전역 상태는 테스트마다 초기화하며 병렬 실행하지 않습니다.

테스트 전용 의존성으로 `github.com/alicebob/miniredis/v2` v2.36.1 을 추가했고(다른 서비스가 같은 버전을 사용), `pflag` 는 테스트가 직접 import 하여 direct 로 이동했습니다.

## 4. 검증

- 수기 합계 73.21%(1834/2505), 원본 66.07%.
- `go test -race -count=1 ./...` 통과, 신규 테스트 `-race -count=20` 반복 통과, `golangci-lint` 0 issues, `scripts/check-test-conventions.sh` 통과.
- 변이 표본: 제품 코드를 별도 복사본에서 바꿔 새 테스트가 실패하는지 확인했습니다. 대표 사례는 `parseDirection` 정규식, `isValidReference` 의 recording 조회 오류, `Stop` 의 activeflow 생성 오류와 variable 오류 처리, `isDuplicateKeyErr` 의 MySQL 번호, `websocketConnect` 의 바이너리 거부, `listenhandler` 의 기본 404, `processEventRun` 의 오류 전파, `Stop` 의 `ConnAst.Close()` 제거입니다. 기존 테스트만으로는 통과하고 신규 테스트로 실패합니다. `Stop` 의 연결 종료 테스트는 클라이언트 연결을 `runtime.KeepAlive` 로 유지하고 서버 대기에 제한 시간을 두어, 종료 호출이 빠지면 가비지 컬렉터가 소켓을 닫아 통과하는 일 없이 단언 실패로 끝납니다.

## 5. 결과

| 패키지 | 전 | 후 |
|---|---|---|
| cmd/transcribe-control | 0.0% | 52.3% |
| cmd/transcribe-manager | 0.6% | 0.6% |
| internal/config | 89.6% | 89.6% |
| models/streaming, transcribe, transcript | 88.9~91.7% | 동일 |
| pkg/cachehandler | 0.0% | 98.6% |
| pkg/dbhandler | 79.4% | 87.2% |
| pkg/listenhandler | 72.2% | 91.1% |
| pkg/streaminghandler | 51.0% | 63.5% |
| pkg/subscribehandler | 77.3% | 91.8% |
| pkg/transcribehandler | 80.9% | 96.3% |
| pkg/transcripthandler | 21.7% | 59.9% |

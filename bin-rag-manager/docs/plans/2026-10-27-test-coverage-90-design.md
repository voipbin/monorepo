# bin-rag-manager 단위 테스트 커버리지 90% 계획 (VOIP-1663)

## 1. 티켓 검토 (이슈 분석)

판정: 부분 유효. origin/main 0bfbe2a1d 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 실행, 실패 0건. 수기 합계(gens, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count) 89.857% (2011/2238), 원본 80.1%. 측정 기준은 수기 합계이며 원본 합계도 함께 보고한다. 90.0% 까지 4, 90.5% 까지 15, 91% 까지 26 statement 추가 필요. 시그니처 변경 없이 도달 가능한 미커버가 필요량을 크게 넘으므로 목표는 91% 이상이다.

## 2. 설계 (테스트 계획)

범위: 테스트 파일 추가만. 비테스트 .go, go.mod, go.sum 변경 없음. 외부 네트워크와 DB 접속 없음. 새 `Test_` 함수는 표 기반, gomock controller 는 `mc`, 하드코딩 UUID, `t.Parallel` 미사용. 임시 파일은 `t.TempDir()` 와 `t.Setenv("TMPDIR", ...)` 로 격리. httptest 핸들러가 기록하는 요청 정보는 mutex 나 채널로 보호해 `-race` 를 통과시킨다.

| 패키지 | 추가 테스트 | 예상 증가(stmt) |
|---|---|---|
| pkg/bucketreader | 성공, 객체 없음, 임시 파일 생성 실패 | 약 12 |
| pkg/embedder | EmbedTexts, EmbedText, 생성자 오류 | 약 20 |
| cmd/rag-manager | run, runMigrations, runListen, initProm | 약 30 |
| pkg/raghandler | 기동 스윕, 주기 점검, RagAddSources URL 성공 | 약 4 |
| pkg/chunker, pkg/dbhandler | docx 비-Run 자식, NewHandler, 행 스캔 오류 | 약 3 |

합계 약 70 stmt, 약 93%. bucketreader 와 embedder 만으로도 91% 를 넘는다.

세부:
- pkg/bucketreader
  - `STORAGE_EMULATOR_HOST` 와 httptest 로 만든 `storage.Client` 사용.
  - 성공: 파일 내용과 요청 경로의 bucket, object 단언. 객체 없음 404 오류. TMPDIR 가 없는 경로일 때 임시 파일 생성 실패.
- pkg/embedder
  - 패키지 내부에서 `googleEmbedder` 를 `genai.BackendGeminiAPI` 와 `HTTPOptions.BaseURL`(httptest) 클라이언트로 구성.
  - 서버는 텍스트마다 1회 호출되는 `POST /v1beta/models/<model>:batchEmbedContents` 에 `{"embeddings":[{"values":[...]}]}` 로 응답하고, 요청 본문의 task type(`RETRIEVAL_DOCUMENT`, `RETRIEVAL_QUERY`)과 텍스트를 단언.
  - `EmbedTexts`(빈 입력은 nil 반환과 서버 요청 0건, 요청 본문 텍스트에 따라 다른 벡터를 돌려줘 다건 순서를 단언, 서버 오류, 빈 embeddings), `EmbedText`(성공, 서버 오류, 빈 embeddings), `NewGoogleEmbedder` project 누락 오류(`could not create genai client` 접두사만 비교). 테스트는 Gemini 백엔드 경로를 쓰고 생성자 자체는 Vertex 고정이라 생성자 성공 경로는 대상이 아니다.
  - genai 환경변수(GOOGLE_API_KEY, GEMINI_API_KEY, GOOGLE_GENAI_USE_VERTEXAI, GOOGLE_CLOUD_PROJECT, GOOGLE_CLOUD_LOCATION, GOOGLE_CLOUD_REGION)는 `t.Setenv` 로 비운다.
- cmd/rag-manager
  - `run`: 오류 문자열을 구분해 단언. 설정 초기화 오류는 플래그가 없는 새 `cobra.Command`(`failed to initialize config`), 나머지는 `init()` 이 플래그를 정의한 `rootCmd` 를 쓰고 값은 환경변수로만 준다(플래그 기본값은 env 가 이기므로 격리됨). 필수 설정 누락은 `invalid configuration` 과 `GCP_PROJECT_ID`, `GCP_REGION`, `POSTGRESQL_DSN` 각각의 `is required`(RabbitMQ 주소는 플래그 기본값이 있어 대상 아님). 누락 케이스는 나머지 필수 env 를 `t.Setenv(..., "")` 로 명시적으로 비워 호스트 환경과 무관하게 한다, 유효 설정에 사용자 정보 없는 `example` 계열 합성 DSN(`postgres://db.example.test:5432/rag`)을 주고 `t.Chdir` 로 빈 임시 디렉터리에서 실행하면 `could not run migrations` 와 `failed to open source` 를 함께 단언(소스 열기가 DB 연결보다 먼저라 네트워크 접속 없음, 순서를 단언으로 고정). `run` 케이스마다 `PROMETHEUS_ENDPOINT`(아래 카운터)와 `PROMETHEUS_LISTEN_ADDRESS=127.0.0.1:0` 을 `t.Setenv` 로 주며, `initProm` 이 띄운 리스너 고루틴은 프로세스 끝까지 남는다.
  - `runMigrations`: 빈 DSN 이면 건너뜀(nil).
  - `runListen`: `sockhandler.NewMockSockHandler`. `QueueCreate` 인자는 `string(commonoutline.QueueNameRagRequest)` 와 `"normal"` 로 정확히 매칭. 성공 시 `ConsumeRPC` 호출을 채널로 동기화. 실패 시 `ConsumeRPC` 기대치를 두지 않고, 항상 nil 을 반환하므로 logrus 훅으로 "Could not run the listenhandler correctly" 기록을 단언.
  - `initProm`: 호출 시 endpoint 는 직접 호출까지 포함해 호출마다 유일하게(패키지 atomic 카운터로 `/metrics-N`, `-count` 와 `-shuffle` 에서 `http.Handle` 중복 패닉 방지). 주소는 `PROMETHEUS_LISTEN_ADDRESS` 를 `127.0.0.1:0` 으로 고정. 오류 경로는 `127.0.0.1:99999`(DNS 없이 즉시 실패)로 호출하고 logrus 훅이 버퍼 채널로 "Could not start prometheus listener" 기록(1회 이상, 남은 고루틴 로그가 섞임)을 알리면 `select` 와 5초 상한 타임아웃으로 관측. 이 고루틴은 프로세스 끝까지 1초 주기로 재시도한다. 훅의 채널 전송은 non-blocking(`select`+`default`)이고 메시지 부분 문자열로 필터링하며, `t.Cleanup` 에서 `StandardLogger().ReplaceHooks` 로 원래 훅 집합을 되돌린다.
  - 전역 viper 는 테스트 시작 시 `viper.Reset()` 을 하고 `t.Cleanup(viper.Reset)` 로도 되돌리며 값은 `t.Setenv` 로 준다. `chSigs`, `chDone` 은 건드리지 않고 신호도 보내지 않는다.
- pkg/raghandler
  - 기동 스윕, 주기 점검, `RagAddSources` URL 성공 경로는 `documentIngest` 고루틴이 mock 을 호출한다. 문서를 반환하는 `ragHandler` 는 `ingestSem` 을 비어 있는 용량 1 의 non-nil 채널로 초기화한다(기존 `coverage_test.go` 방식, nil 이면 고루틴이 영구 대기). `DocumentGetPending` 은 첫 호출에서만 문서를 돌려주고 이후는 빈 슬라이스를 돌려준다. `DocumentClaimForProcessing` 이 오류를 반환하는 mock 에서 `sync.Once` 로 채널을 닫아 이중 close 를 막고, 테스트가 그 채널을 기다린다(`time.Sleep` 금지).
  - `RunIngestionTicker` 는 기대치를 `AnyTimes()` 로 두고 티커 간격은 10ms 로 두고, 첫 `DocumentGetPending` 안에서 `cancel` 한 뒤 함수 반환을 `done` 채널로, 문서를 처리하는 고루틴의 claim 호출을 claim 채널로 둘 다 기다린 뒤 끝낸다(문서는 `sync.Once` 로 첫 호출에서만 반환). 모든 채널 대기는 `select` 와 `time.After` 상한을 둔다.
- 새 테스트는 패키지별 신규 `_test.go` 파일(`google_test.go`, bucketreader `main_test.go`, cmd `main_test.go`, raghandler `ingest_dispatch_test.go`)에 둔다. docx 는 기존 `docx_test.go` 의 zip 헬퍼를 재사용한다.
- pkg/chunker: 하이퍼링크처럼 Run 이 아닌 문단 자식이 있는 docx 를 zip 으로 직접 만들어, 건너뛰고 나머지 텍스트를 이어붙임을 단언.
- pkg/dbhandler: 보유 sqlmock 헬퍼로 `NewHandler`(반환 핸들러로 쿼리 1건을 실행해 db 연결을 단언)와 문서 행 스캔 오류 경로.

verify: 변이 표본 확인은 raghandler 의 디스패치 호출 삭제, docx 비-Run 건너뛰기 제거, 스캔 오류 처리 삭제, task type 교체, 다건 순서 뒤집기, `runListen` 오류 분기 제거, `run` 오류 분기 제거, bucketreader 오류 래핑 제거에서 신규 테스트가 실패하는지 본다. 수기 합계 91% 이상, `-race -shuffle on -count=3` 통과, 루트 CLAUDE.md 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, golangci-lint) 후 go.mod, go.sum diff 없음, `scripts/check-test-conventions.sh` 통과.

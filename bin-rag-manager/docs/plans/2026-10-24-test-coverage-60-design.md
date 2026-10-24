# bin-rag-manager 단위 테스트 커버리지 60% 계획 (VOIP-1647)

## 1. 티켓 검토 (이슈 분석)

판정: 유효.

- 기준선 재현: origin/main d68e695b5 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile ./...` 실행, 실패 0건. 수기 합계(gens, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count) 37.67% (843/2238), 원본 33.56% (843/2512). 티켓 수치와 일치.
- 최근 변경 이력: `git log -- bin-rag-manager` 최근 커밋은 lint, 의존성 bump, 문서 정리뿐이며 테스트 전제를 바꾸는 변경 없음. 열린 PR 중 같은 목적의 PR 없음(`gh pr list --state all --search`는 과거 병합 PR #497 뿐).
- 패키지별 전(수기 statement, 커버, %):

| 패키지 | stmt | 커버 | % |
|---|---|---|---|
| cmd/rag-manager | 199 | 70 | 35.2 |
| internal/config | 99 | 0 | 0.0 |
| models/rag | 1 | 0 | 0.0 |
| pkg/bucketreader | 17 | 0 | 0.0 |
| pkg/chunker | 502 | 402 | 80.1 |
| pkg/dbhandler | 596 | 7 | 1.2 |
| pkg/embedder | 23 | 0 | 0.0 |
| pkg/listenhandler | 267 | 187 | 70.0 |
| pkg/raghandler | 534 | 177 | 33.1 |

- 테스트 난이도 분석:
  - cmd/rag-manager 의 main, run, runMigrations, runService, runListen, initProm 은 프로세스 기동과 외부 연결(RabbitMQ, PostgreSQL)이 필요해 단위 테스트 대상에서 제외.
  - bucketreader, embedder/google.go 는 GCS 와 Vertex 클라이언트를 직접 생성해 네트워크 의존. 제외.
  - 가장 큰 미커버는 pkg/dbhandler (589 stmt). 기존 테스트(rag_test, document_test, chunk_test)는 squirrel 로 같은 쿼리를 다시 만들어 문자열을 비교하는 방식이라 실제 핸들러 메서드를 호출하지 않는다(자기비교). `*sql.DB` 를 받는 구조이므로 `go-sqlmock` 으로 실제 메서드를 실행할 수 있다. 제품 코드 시그니처 변경 불필요.
  - pkg/raghandler 는 dbhandler 등이 인터페이스+mock 이라 RagCreate, RagList, RagUpdate, QueryRag, DocumentGet/List, documentIngestFail, buildDBChunks, embedTextsWithHeartbeat 를 단위 테스트할 수 있다.
  - internal/config 는 viper/cobra 기반 환경변수 로딩이며 로컬에서 결정적으로 테스트 가능.
- 산술: 60% 는 1343/2238, 필요한 추가 커버 약 500 stmt. dbhandler 를 실제 메서드 호출로 90% 이상 올리면 약 +530 이어서 단독으로 근접하고, raghandler 와 config 로 여유(목표 62% 이상)를 둔다.
- 확인 결과(고칠 결함 아님): `DocumentClaimForProcessing` 은 sql.ErrNoRows 를 ErrNotFound 로 바꾸지 않는다. 코드 주석(document.go 의 NOTE)이 의도라고 밝히고, 유일한 호출처 raghandler/document.go 는 모든 오류를 claim 실패로 건너뛴다. 테스트는 이 동작(ErrNoRows 그대로 전달)을 고정한다.
- 의존성: bin-rag-manager/go.mod 에 go-sqlmock 이 없다. 다른 서비스가 쓰는 v1.5.2 를 추가하며 go.mod, go.sum 이 이 PR 에 포함된다(모두 bin-rag-manager/ 안, 비테스트 .go 변경 없음).

## 2. 설계 (테스트 계획)

범위: `_test.go` 신규 파일과 go.mod/go.sum(go-sqlmock v1.5.2 를 direct 의존으로 추가하고, 테스트가 smfile 을 import 하면 `go mod tidy` 가 bin-storage-manager 를 indirect 에서 direct 로 바꾸는 변경 포함)뿐. 비테스트 .go 변경 없음, 시그니처 변경 없음.

공통 규칙: 새 `Test_` 함수는 표 기반이며 gomock controller 는 `mc := gomock.NewController(t)` + `defer mc.Finish()`, 실패 메시지는 `t.Errorf("Wrong match. expect: %v, got: %v", ...)` 형식, testify 미사용, 하드코딩 UUID(`uuid.FromStringOrNil`) 사용, `t.Parallel` 미사용. 반환 구조체는 `reflect.DeepEqual` 로 전체 비교하고 오류 케이스는 `errors.Is` 로 단언한다. `scripts/check-test-conventions.sh` 를 로컬에서 실행한다.

### 2.1 pkg/dbhandler (sqlmock)
- 헬퍼 `newTestHandler(t)`: `sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))` 로 mock DB, `&handler{utilHandler: mockUtil, db: db}`. `mockUtil` 은 `utilhandler.NewMockUtilHandler(mc)` 이며 `TimeNow()` 는 `Return(&now)`(`*time.Time` 반환), `TimeGetCurTime()` 은 고정 문자열을 반환한다. 각 케이스 끝에 `mock.ExpectationsWereMet()` 확인.
- SQL 은 테스트에 리터럴로 적고(squirrel 로 재생성하지 않음) `WithArgs` 로 인자(uuid, 고정 시각, 문자열)를 고정한다. `QueryRowContext` 를 쓰는 `ClaimForProcessing`, `Get` 은 `ExpectQuery`, Exec 계열은 `ExpectExec`.
- `RagList`/`DocumentList` 는 `range filters` 순서가 비결정적이므로 케이스당 필터를 1개만 쓴다.
- 대상과 케이스:
  - Rag: Create 성공/Exec 오류, Get 성공/ErrNoRows→ErrNotFound/기타 scan 오류, List 의 token 기본값과 size 기본 100, deleted=false/true/비bool 값, 임의 컬럼 필터, 쿼리 오류, 스캔 오류, rows.Err, Update 성공/오류, Delete 성공/오류.
  - Document: Create(StorageFileID Nil 이면 NULL 인자, 값이 있으면 uuid 인자), Get(NULL storage_file_id/source_url 스캔 분기, ErrNoRows 변환), List(Rag 와 동일 분기), Update, Delete, DeleteByRagID, ClaimForProcessing(성공, ErrNoRows 는 변환 없이 그대로 전달, 쿼리 오류), UpdateHeartbeat, GetStale(cutoff = 고정 시각 - threshold), GetPending(retry_count < 3), ResetStaleToPending(두 UPDATE 순서, 첫째 실패 시 둘째 미실행, 둘째 실패), GetsByRagID, GetsByRagIDs(빈 입력은 쿼리 없이 빈 맵, rag_id 별 그룹핑), GetsByRagIDAndSources(둘 다 비면 쿼리 없음, 한쪽만, 둘 다 OR).
  - Chunk: Create 성공/오류, CreateBatch(길이 불일치, 빈 입력 nil, Begin 실패, 중간 Exec 실패 + ExpectRollback, Commit 실패(Rollback 기대 없음), 성공 + 정확한 insert 인자 순서), SearchByRagID(score = 1 - distance, 쿼리 오류, 스캔 오류, rows.Err), DeleteByDocumentID/RagID, SoftDelete 2종의 성공/오류. `formatEmbedding` 은 기존 테스트가 덮으므로 추가하지 않는다.
  - `ToSql` 오류 분기(26개)는 도달 불가이므로 대상 외.
- 기존 자기비교 테스트(`Test_*_SQL`)는 건드리지 않는다(삭제는 범위 밖).

### 2.2 pkg/raghandler (gomock, 외부 네트워크 없이)
- RagCreate: DB 오류 전파, 소스 없는 성공(RagGet 까지), createDocumentsForSources 실패 시 RagDelete 롤백. RagList: DB 오류, 빈 결과는 DocumentGetsByRagIDs 미호출, 문서 조회 실패 시 rags 를 그대로 반환, 정상 시 Status/Sources 채움. RagUpdate: Update 오류, Get 오류, 성공. QueryRag: topK 0 과 음수일 때 mock 인자가 정확히 5, 임베딩 오류, 검색 오류, 문서 조회 실패 시 DocumentName 빈 값, 정상 매핑과 score. DocumentGet: ErrNotFound 는 cerrors NotFound(`errors.Is`), 기타 오류 래핑, 성공. DocumentList: 성공/오류. documentIngestFail: RetryCount 2(maxRetryCount-1)는 pending 복귀, 3(maxRetryCount)은 error 와 메시지. embedTextsWithHeartbeat: 10개와 11개 경계의 EmbedTexts/Heartbeat 호출 횟수, 임베딩 오류 위치 표기, heartbeat 오류 무시. buildDBChunks: 인덱스, 필드 복사, TokenCount = len/4.
- documentIngest 는 동기 함수이므로 직접 호출해 mock 으로 분기를 검증한다(claim 실패 시 조기 반환, 파일 확보 실패, 파싱 실패, 임베딩 실패, 저장 실패, 성공 시 ready 갱신). documentAcquireFile 과 documentDownloadGCS 는 reqHandler/bucketReader mock 으로, RunIngestionTicker 는 짧은 interval 과 ctx cancel 로, DocumentIngestPendingAll/ingestionTickerRun 은 pending 없는 경로와 오류 경로만(고루틴을 띄우는 경로는 만들지 않는다). documentDownloadURL 은 이 PR 에서 제외(구현 비용, httptest 서버 필요).
- 고루틴이 뜨는 호출 경로는 테스트하지 않으며 `go test -race` 로 확인한다. 기존 RagDelete 류 테스트는 건드리지 않는다.

- 보강 규칙: (a) embedder 는 gomock mock 이 없으므로(생성하면 비테스트 파일 변경) raghandler 의 `_test.go` 안에 fake embedder 를 둔다. 호출 횟수, 배치 인자 기록, n 번째 호출에서 오류 반환을 지원한다. (b) `documentIngest` 직접 호출 케이스는 `ragHandler` 리터럴에 `ingestSem: make(chan struct{}, 1)` 을 넣는다. (c) SourceURL 문서는 실제 네트워크를 타지 않도록 거부 스킴(`ftp://`)이나 빈 소스만 쓴다(documentDownloadURL 의 http/https 경로는 쓰지 않음). (d) `RunIngestionTicker` 는 interval 20ms 이상으로, 첫 mock 호출의 `DoAndReturn` 안에서 `cancel()` 을 호출해 tick 횟수를 결정적으로 만들고 `AnyTimes()` 는 쓰지 않는다. (e) `documentIngest` 분기에 청크 0개(빈 파일), 확장자 없을 때 contentType 폴백, 후속 heartbeat 오류 무시를 포함하고 `ChunkCreateBatch` 는 `DoAndReturn` 으로 청크 수, `DocumentID`, `RagID`, `CustomerID` 를 단언한다(`gomock.Any()` 금지). (f) `documentCreateInternal` 오류 경로와 `createDocumentsForSources` 의 실패 분기를 고루틴 없이 포함한다. (h) `documentDownloadGCS` 는 50MB 초과, `BucketName` 빈 값일 때 `h.bucketName` 폴백, `StorageV1FileGet` 오류, `DownloadToTempFile` 오류를 각각 케이스로 둔다. mock 의 `TimeNow` 호출 횟수는 `Times` 로 정확히 지정한다. (g) `ChunkCreateBatch` 길이 불일치 케이스는 오류 메시지의 "length mismatch" 부분 문자열을 단언하고 `Begin` 기대를 두지 않으며 `ExpectationsWereMet` 를 확인한다.

### 2.3 internal/config
- `Validate()` 의 필수값 누락별 오류와 정상, `Bootstrap` 의 플래그와 기본값 바인딩, env 가 플래그 기본값보다 우선하는지, `LoadGlobalConfig`(`viper.Reset()`, `once = sync.Once{}`, `t.Setenv` 로 초기화, 빈 문자열 env 는 쓰지 않음), `InitConfig` 의 성공 경로(`cmd.Flags().AddFlagSet(cmd.PersistentFlags())`)와 플래그를 하나씩 빼는 표 기반 오류 경로. 병렬 금지. `InitConfig` 가 덮어쓰는 `globalConfig` 와 viper 전역은 `t.Cleanup` 에서 `viper.Reset()` 과 값 복원으로 되돌린다. 테스트 시작 시 config 가 읽는 env 전체를 `os.Unsetenv` 로 비우고 `t.Cleanup` 으로 원래 값을 복원해 호출 환경의 env 에 영향받지 않게 한다.

### 2.4 제외
cmd 의 run 계열, bucketreader, embedder/google.go, documentDownloadURL, `ToSql` 오류 분기(도달 불가).

### 2.5 verify 항목과 목표
- 산술: 62% = 1388 stmt(+545). 계획분은 dbhandler 약 +530, raghandler 약 +250(기존 +166 에 documentIngest 등 추가), config 약 +80 으로 합계 약 +860, 수기 약 75%까지 가능하나 목표는 62% 이상이며 최소 60.0%.
- `go test -race -count=1 ./...` 통과, `go mod tidy`, `go mod vendor`, `go generate ./...`, `golangci-lint run` 통과, `bash scripts/check-test-conventions.sh` 통과.
- 변이 검사(구현 후 scratch 복사본, 각각 새 테스트가 실패해야 함): ① dbhandler `RagList` 의 `tm_delete IS NULL` 제거, ② `ChunkSearchByRagID` 의 `1-distance` 를 `distance` 로, ③ `ResetStaleToPending` 의 `retry_count < 3` 을 `<= 3` 으로, ④ `QueryRag` 의 topK 기본값 5 를 4 로, ⑤ `documentIngestFail` 의 `>=` 를 `>` 로, ⑥ `GetStale`/`ResetStale` 의 `Add(-threshold)` 부호 반전, ⑦ `Get*` 의 `ErrNoRows→ErrNotFound` 변환 제거, ⑧ `ChunkCreateBatch` 길이 불일치 검사 제거, ⑨ `RagList` 의 문서 조회 실패 시 rags 반환 경로를 오류 반환으로 변경, ⑩ `DocumentClaimForProcessing` 의 `status = pending` 조건 제거, ⑪ `DocumentList` 의 `deleted` 분기 반전, ⑫ config `Validate()` 필수값 검사 하나 제거, ⑬ `heartbeatInterval` 10 을 11 로, ⑭ `documentIngest` 의 청크 0개 검사 제거, ⑮ `GetPending` 의 `retry_count < 3` 을 `<= 3` 으로. ②의 검증을 위해 distance 는 0.2 처럼 0.5 가 아닌 값을 쓴다.
- 영향도: DB 스키마와 운영 동작 변경 없음(하드룰 17 해당 없음).

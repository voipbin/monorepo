# bin-rag-manager 단위 테스트 커버리지 85% 계획 (VOIP-1659)

## 1. 티켓 검토 (이슈 분석)

판정: 유효.

- 기준선 재현: origin/main a47f5a2a1 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 실행, 실패 0건. 수기 합계(gens, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count) 80.70% (1806/2238), 티켓 수치와 일치. 85.0% 까지 97 statement, 85.5% 까지 108 statement 추가 필요.
- 최근 이력: `git log -- bin-rag-manager` 에 커버리지 레인(VOIP-1647)과 listenhandler 테스트 수정(VOIP-1648) 이후 테스트 전제를 바꾸는 변경 없음. 같은 목적의 열린 PR 없음.
- 패키지별 전(수기 statement, 커버, %):

| 패키지 | stmt | 커버 | % |
|---|---|---|---|
| cmd/rag-manager | 199 | 70 | 35.2 |
| internal/config | 99 | 97 | 98.0 |
| models/rag | 1 | 0 | 0.0 |
| pkg/bucketreader | 17 | 0 | 0.0 |
| pkg/chunker | 502 | 402 | 80.1 |
| pkg/dbhandler | 596 | 568 | 95.3 |
| pkg/embedder | 23 | 0 | 0.0 |
| pkg/listenhandler | 267 | 187 | 70.0 |
| pkg/raghandler | 534 | 482 | 90.3 |

- 도달 가능성: 시그니처 변경 없이 도달 가능한 미커버가 필요량(108)을 크게 넘는다. 미커버 중 cmd 의 기동 함수, bucketreader, embedder 는 외부 연결이 필요해 대상에서 제외하고, dbhandler 의 남은 분기는 쿼리 빌더 오류 분기와 DB 연결 생성자(NewHandler)라 제외한다. 이를 빼고도 상한이 85% 를 넘으므로 block 없이 진행한다.

- 미커버 블록 실측(함수별 미커버 statement, 대상 파일만): listenhandler processRequest 22, processV1RagsPost 9, IDPut 9, SourcesPost 8, RagsGet 7, QueryPost 7, SourcesIDDelete 6, IDGet 4, IDDelete 4, errorResponse 2, jsonResponse 1(채널 값 등 직렬화 불가 값으로 도달 가능), Run 1(고루틴, 제외). chunker docx Chunk 32, pdf Chunk 23, json Chunk 7 / chunkObject 6 / chunkArray 8, csv 9, rst splitRSTSections 6 / Chunk 2, html 3, markdown 2, openapi 1, text 1. raghandler/document.go documentDownloadURL 42, documentIngest 2, ingestionTickerRun 2, DocumentIngestPendingAll 1, buildDBChunks 1, documentCreateInternal 1. models/rag ConvertWebhookMessage 1.
- 도달 불가로 뺀 추정치: listenhandler 약 9(processRequest 의 err != nil 블록 6 은 모든 process 핸들러가 nil 오류를 반환해 도달 불가, Run 고루틴 1. errorResponse 의 ToResponse 실패 2 는 도달 가능하나 여유가 커서 제외), chunker 약 5(json Marshal 실패 continue 2, pdf IsNull/GetPlainText 오류 2, docx Stat 오류 1), documentDownloadURL 3(NewRequestWithContext 오류 1, tmpFile.Close 오류 2). 이외에 uuid 생성 실패 분기(document.go, rag.go 의 uuid.NewV4 오류)와 `go h.documentIngest(...)` 고루틴 기동 줄은 도달 불가 또는 동기화 비용 때문에 제외한다(raghandler 미커버는 document.go 49 와 rag.go 3 을 합쳐 52). 제외분을 모두 빼도 도달 가능 합계는 약 205 로 필요량 108 의 약 1.9배다.

## 2. 설계 (테스트 계획)

범위: `_test.go` 신규 파일만. 비테스트 .go 변경 없음, 시그니처 변경 없음, go.mod/go.sum 변경 없음(go-docx, ledongthuc/pdf 는 이미 직접 의존). 새 `Test_` 함수는 표 기반, gomock controller 는 `mc`, 하드코딩 UUID, `t.Parallel` 미사용, 실패 메시지 `Wrong match. expect: %v, got: %v`. 임시 파일은 `t.TempDir()` 만 쓴다. `documentDownloadURL` 은 `os.CreateTemp("", ...)` 로 파일을 만들므로 해당 테스트는 `t.Setenv("TMPDIR", t.TempDir())` 로 격리한다(CreateTemp 오류 분기는 TMPDIR 를 존재하지 않는 경로로 두어 도달). 성공 응답은 mock 이 돌려준 객체를 기대값으로 재사용하지 않고 별도 JSON 리터럴과 비교한다. `scripts/check-test-conventions.sh` 를 로컬 실행한다.

### 2.1 pkg/listenhandler (미커버 약 80)
기존 `newTestHandler` 와 `raghandler.NewMockRagHandler` 를 재사용한다.
- processV1RagsPost: 잘못된 JSON, customer_id 누락, name 누락, 소스 둘 다 없음 각각 400 이고 mock 미호출. 성공 시 인자(customer, name, description, storage ids, urls) 를 정확히 단언하고 200 본문을 역직렬화해 비교.
- processV1RagsGet: page_size/page_token 쿼리 전달(정확한 pageSize, token), 필터 본문 파싱 실패 400, 필터 변환 오류 400(`customer_id` 에 UUID 로 파싱되지 않는 문자열만. `deleted` 의 비-bool 값은 변환 오류 없이 원값이 전달되므로 400 케이스가 아님), 알 수 없는 키는 무시되어 `RagList` 에 빈 필터 맵이 전달됨을 단언, 목록 성공, 핸들러 오류 500.
- processV1RagsIDGet/Put/Delete: URI 항목 부족과 UUID 파싱 실패 400, Put 은 잘못된 JSON 과 필드 없음 400, name 만/description 만/둘 다일 때 fields 맵 단언, 성공 200, Delete 성공은 본문 없는 200.
- processV1RagsIDSourcesPost/IDDelete: URI 부족, rag/source UUID 오류, 잘못된 JSON, 소스 없음 400, 성공 인자 단언.
- processV1QueryPost: 잘못된 JSON, rag_id 누락, query 누락 400, top_k 전달, 성공 본문.
- processRequest 디스패치: 라우트 8종(sources 삭제, sources 추가, id 조회/삭제/수정, 목록 생성/조회, query)이 각각 올바른 핸들러(mock 메서드)에 도달하는 URI to 핸들러 표(파싱된 값은 `gomock.Any()` 없이 정확히 단언하고 ctx 는 `context.Background()` 로 매칭)와, 메서드 불일치/미등록 URI 404. typed/일반 오류의 상태코드 변환은 기존 error_response_test, v1_rags_errmap_test 가 덮으므로 추가하지 않는다.
- jsonResponse: 직렬화 불가 값(채널)은 500. errorResponse 의 ToResponse 실패 분기는 `Details` 에 채널 값을 넣으면 도달 가능하나 이 계획에서는 제외. processV1RagsGet 의 URI 파싱 실패(제어문자 URI) 400 도 직접 호출로 추가.
- 검증 실패 케이스는 다른 필드를 모두 유효하게 두고 mock 에 기대를 두지 않아, 검증을 제거하는 변이가 mock 의 비기대 호출로 잡히게 한다.

### 2.2 pkg/chunker (미커버 약 100)
- docx: go-docx 로 테스트에서 실제 .docx 를 만들어 텍스트 추출, 빈 문서 결과는 `nil` 슬라이스로 단언(`[]Chunk{}` 와 DeepEqual 이 다름), 표 등 비문단 항목 무시, 손상 파일(zip 아님)은 parse 오류, 분할은 텍스트 문단 사이에 빈 문단을 넣어 `\n\n` 이 생기게 한 입력에서만 확인하며(문단 하나가 한도를 넘어도 분할되지 않음) 청크 수는 손으로 센 리터럴로 고정한다(분할 함수로 계산하지 않음).
- pdf: 테스트 안에서 xref 오프셋을 계산하고 Helvetica 폰트 객체를 포함하는 최소 PDF 바이트열 생성 헬퍼로 텍스트를 추출하며 추출 문자열을 구체 값으로 단언(폰트가 없으면 조용히 빈 결과가 되므로). 분할(Page group N)은 페이지 텍스트가 빈 줄을 만드는 입력을 먼저 측정한 뒤에만 단언한다, 텍스트 없는 PDF 는 nil, 손상 파일은 오류.
- json: 객체 키별 청크와 title, 한도 초과 키의 분할(`splitByTokenLimit` 은 빈 줄로만 나누고 MarshalIndent 출력에는 빈 줄이 없으므로 결과는 항상 `Key: k (part 1)` 1개. 한도 이내 키의 `Key: k` 제목과의 대비로 `<=` 경계를 제목으로 단언(`len(text) == maxTokens*4` 가 되도록 값을 패딩해 길이를 4 의 배수로 맞춘 단일 키 객체로 검증하며 다른 키를 섞지 않음), maxTokens 는 `MarshalIndent` 결과 길이에서 계산, 키 집합 비교는 `len > 0` 을 먼저 단언), openapi 의 paths 맵 순회는 정렬 후 비교하고, json 객체는 title 집합으로 비교한다. 배열의 크기 기반 분할과 마지막 묶음, JSON 이 아닌 입력의 텍스트 폴백.
- csv: parse 오류(열린 따옴표 입력), 빈 파일 nil, 토큰 초과 분할(open 오류는 기존 테스트가 덮음). rst: overline 제목 분기(`본문 / 빈 줄 / overline / 제목 / underline` 순서의 입력으로 이전 섹션 저장까지(빈 줄이 없으면 일반 제목 분기가 먼저 걸림)), 빈 섹션 건너뜀, 한도 초과 `(part N)` 제목. html: 파싱 오류(디렉터리 경로), 빈 결과 nil, script/style 제거. markdown: 읽기 오류, `(part N)` 제목. text: 공백만 있는 파일은 nil(읽기 오류는 기존 테스트가 덮음). openapi: 메서드 맵이 아닌 paths 항목 건너뜀. chunkArray 는 `currentSize+len(b) == maxChars` 경계 케이스를 둔다. 위 분기는 모두 커버 프로파일로 미커버임을 구현 전에 다시 확인한다.

### 2.3 pkg/raghandler (미커버 약 52)
- documentDownloadURL: `httptest.NewServer`(루프백, 외부 네트워크 없음)로 200 본문과 파일명/Content-Type 추출, 404 오류, `http://[::1` 같은 url.Parse 오류, 선언 길이보다 적게 보내고 연결을 끊는 서버(io.Copy 중간 오류), Content-Length 초과, 본문 초과(LimitReader 경계 maxFileSize 와 +1), 연결 오류(직접 `net.Listen` 후 닫은 주소 사용). `maxFileSize`(50MiB) 경계는 상수를 읽어 만들고, Content-Length 가 자동 부여되는 작은 응답은 `resp.ContentLength > maxFileSize` 분기만 타므로 `LimitReader` 경로는 `http.Flusher` 로 청크 전송(ContentLength == -1)해 도달한다. 본문은 메모리에 한 번에 올리지 않고 반복 쓰기로 스트리밍하며, `== maxFileSize` 성공 케이스와 `+1` 실패 케이스를 모두 둔다. 크기 경계는 줄마다 죽이는 케이스를 둔다: (a) `resp.ContentLength > maxFileSize` 는 Content-Length 를 정확히 maxFileSize 로 선언하고 본문을 끝까지 보내는 비청크 성공 케이스와 maxFileSize+1 을 헤더로만 선언하는 실패 케이스, (b) `n > maxFileSize` 는 청크 전송의 `==` 성공과 `+1` 실패, (c) GCS 경로의 `file.Filesize > maxFileSize` 는 기존 coverage_test 가 `==` 와 `+1` 을 이미 덮으므로 추가하지 않는다. 50MiB 본문 케이스는 최대 4개로 제한하고 `io.Copy` 로 스트리밍하며 `-race -count=3` 소요 시간을 측정해 보고한다. 타임아웃 경로(5분)는 테스트하지 않는다. 서버 주소는 단언에 하드코딩하지 않는다. 임시 파일 정리도 단언한다: 실패 케이스(상태코드, 크기 초과, io.Copy 오류)는 격리한 TMPDIR 가 비어 있음을, 성공 케이스는 반환 경로의 파일 내용이 기대값임을 확인한다. 청크 전송 핸들러는 Write 오류에서 반드시 반환하고 헤더 전송을 위해 Flush 하며, 모든 서버는 `t.Cleanup` 으로 닫는다.
- raghandler 의 그 밖의 미커버(documentIngest 2, buildDBChunks 1, documentCreateInternal 1, rag.go RagCreate 2 는 uuid 생성 실패 분기, ingestionTickerRun 2 와 DocumentIngestPendingAll 1 과 rag.go 1 은 pending 문서가 있어야 닿는 `go h.documentIngest` 고루틴 기동 줄로 동기화 비용 때문에 제외)는 전부 도달 불가 또는 제외 대상이다. raghandler 도달 가능분은 documentDownloadURL 약 39 이다.

### 2.4 models/rag
- ConvertWebhookMessage: 모든 필드(Status, Sources, TMCreate, TMUpdate 포함)를 서로 다른 0 이 아닌 값으로 채워 전체 구조체 DeepEqual 로 단언하되 입력과 기대값을 별도 리터럴로 작성한다.

### 2.5 제외
cmd 의 main, run, runService, runListen, initProm(외부 연결), pkg/bucketreader, pkg/embedder/google.go(GCS, Vertex 클라이언트), dbhandler 의 쿼리 빌더 오류 분기와 NewHandler. 기존 자기비교 테스트는 건드리지 않는다.

## 3. 검증(verify) 항목
0. 구현 전 `git fetch origin main`, 중간 측정은 chunker(docx, pdf, json)를 먼저 구현한 뒤 실시한다.
1. 수기 합계 85.0% 이상, 85.5% 이상 목표. 보고서에는 수기 합계와 `go tool cover` 원본 합계(mock 포함)를 병기한다. 산식은 scratch 의 계산 스크립트(gens/, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count)로 패키지별 전후 표를 만든다. 스크립트는 저장소에 넣지 않는다.
2. `go test -race -count=3 ./...` 통과(순서 의존과 시간 의존 없음).
3. 변이 검사: 표본 5곳 이상(예: Post 의 name 검증 제거, 필터 변환 오류 400 제거, 다운로드 크기 경계 `>` 를 `>=` 로, json 분할 조건 반전, docx 빈 문서 조기 반환 제거, ConvertWebhookMessage 필드 하나 누락, csv 분할 분기 제거, 다운로드 크기 경계 줄별 변이, 실패 경로의 `os.Remove` 제거)에서 새 테스트가 실패하는지 scratch 복사본에서 확인.
4. 루트 CLAUDE.md 의 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`)를 모두 실행하고 `go generate` 후 mock diff 가 없음을 확인한다. `scripts/check-test-conventions.sh` 도 실행한다. 변이 검사는 형제 디렉터리 replace 때문에 monorepo 레이아웃을 유지한 scratch worktree 에서 한다. 마지막으로 `go mod tidy` 후 go.mod/go.sum diff 없음, golangci-lint 통과, `git diff --stat origin/main...HEAD` 가 bin-rag-manager/ 안의 `_test.go` 와 이 문서뿐.

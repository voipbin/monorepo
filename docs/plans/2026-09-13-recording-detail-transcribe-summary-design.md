# Recording Detail: One-Click Transcribe & AI Summary — Design

Date: 2026-09-13
Author: CPO (Hermes) with pchero
Status: Design review

## 1. 배경 / 문제

admin 콘솔의 레코딩 상세 화면(`recordings_detail.js`)은 현재 Download / Delete 버튼만
제공한다. 사용자가 특정 레코딩을 텍스트로 변환(transcribe)하거나 AI 요약(summary)을
보려면 별도 메뉴(`transcribes`, `aisummaries`)로 이동해 recording ID를 수동 입력해
생성해야 한다. 컨택센터 운영자가 통화 녹음을 검토하는 실제 흐름("녹음 → 텍스트 → 요약")과
정반대이며, 결과를 한눈에 확인할 수 없다.

목표: 레코딩 상세 화면에서 그 레코딩의 transcript와 AI summary를 **한눈에 확인**하고,
없으면 **원클릭으로 바로 생성**한다.

## 2. 현황 조사 (코드 확인 결과)

### 2.1 백엔드 능력 (대부분 준비됨)

| 능력 | 상태 | 근거 |
|------|------|------|
| `POST /transcribes` (reference_type=recording) | 지원, recording은 **멱등** (동일 언어 기존 것 200 반환) | `openapi/paths/transcribes/main.yaml` POST description |
| `GET /transcribes?reference_type=recording&reference_id=` | **지원** | `openapi/paths/transcribes/main.yaml` GET params; `server/transcribes.go` `GetTranscribes`; `servicehandler.TranscribeList` |
| `GET /transcripts?transcribe_id=` | 지원 | `transcripts_list.js`가 이미 사용 |
| `POST /aisummaries` (reference_type=recording) | 지원 (enum에 recording 포함) | `openapi.yaml` `AIManagerSummaryReferenceType` |
| bin-ai-manager summary `reference_type`/`reference_id` 필터 | **이미 지원** | `bin-ai-manager/models/summary/filters.go`, `field.go` |
| `requesthandler.AIV1SummaryList(... filters)` | filters 파라미터 이미 받음 | `bin-common-handler/pkg/requesthandler/ai_summaries.go:20` |

### 2.2 빠진 부분 (gateway 노출 + 프론트)

1. **`GET /aisummaries`가 reference 필터를 노출하지 않음.** 체인 3곳이 customer_id만
   필터에 넣는다:
   - OpenAPI `paths/aisummaries/main.yaml` GET: `reference_type`/`reference_id` 파라미터 없음
   - `server/aisummaries.go` `GetAiSummaries`: 쿼리 파싱 안 함, `AISummaryGetsByCustomerID` 호출
   - `servicehandler.AISummaryGetsByCustomerID`: `filters`에 `customer_id`, `deleted`만 하드코딩
2. **`recordings_detail.js`에 transcribe/summary 진입점 전무.**

## 3. 설계 결정 (확정)

| # | 항목 | 결정 | 이유 |
|---|------|------|------|
| D1 | AI Summary 조회 | 백엔드 `GET /aisummaries`에 `reference_type + reference_id` 필터 추가 (transcribe와 동일 패턴) | 정석적, 클라이언트 전체 로드 필터링은 확장성 나쁨 |
| D2 | Summary 중복 방지 | 기존 요약 있으면 결과 표시, 재생성은 명시적 **Regenerate** 버튼으로 분리 | summary는 멱등 아님 (버튼 연타 시 누적) |
| D3 | 언어 선택 | 기본값 자동 지정 후 원클릭 실행, 드롭다운으로 변경 가능 | 원클릭 우선 (D1 대표님 확정) |
| D4 | Transcribe 중복 | 별도 처리 불필요. recording 대상은 백엔드가 이미 멱등 | 기존 동작 |

### 3.1 기본 언어 결정 (D3 세부)

레코딩 도메인 모델에는 언어 정보가 없다. 프론트에서 다음 순서로 기본값을 정한다:
1. `navigator.language`가 지원 목록(full BCP-47: en-US, ko-KR, ja-JP 등)에 정확히 있으면 그 값
2. `navigator.language`가 2글자 축약형(`en`, `ko`, `ja` 등)이면 정규화 매핑으로 변환
   (`en`→`en-US`, `ko`→`ko-KR`, `ja`→`ja-JP`, ...). 매핑 테이블은 지원 목록의
   각 언어군 대표 로케일로 구성한다.
3. 위 어느 것에도 안 맞으면 `en-US`

- **주의**: `navigator.language`는 브라우저가 `en`, `ko`처럼 2글자만 반환하는 경우가 많다.
  단순 "in 목록" 매칭은 대부분의 실제 로케일에서 조용히 en-US로 폴백하므로, 반드시 2단계
  정규화를 거친다.
- **제품 후속 과제(비범위)**: 운영자 UI 로케일 ≠ 녹취 대화 언어다. 향후 고객/큐 단위 기본 언어
  설정이 `navigator.language`보다 정확하다. 이번엔 브라우저 로케일 + 드롭다운 변경으로 충분히
  커버하되, 정확한 기본값은 후속으로 남긴다.

드롭다운은 transcribes/aisummaries create 화면과 동일한 언어 목록 + custom 입력을 재사용한다.

### 3.2 비동기 상태 갱신 (D5 확정) — 옵션 B

transcribe(recording 대상, 완료된 파일 STT)와 summary(LLM 호출) 모두 **비동기**다.
`POST` 직후 조회하면 대부분 `progressing`(summary) 또는 빈 transcript(transcribe)를 반환한다.
square-admin CLAUDE.md는 **manual Refresh 버튼·polling을 명시적으로 금지**한다.

**결정 (D5, 옵션 B): 조회 인라인 표시 + 원클릭 생성까지 완성. 실시간 자동 갱신은 후속 과제.**

- 대표님 확정(2026-09-13): 실시간 완료 자동 표시(WebSocket 구독)는 이번 범위에서 제외.
  근거: admin WS는 payload **shape로만** 이벤트를 분류하는데(wire에 `type` 필드 없음,
  `websocket.js:40-79`), transcribe의 `direction`은 이미 구독 중인 `call` payload에도,
  summary의 `reference_type`은 이미 구독 중인 `activeflow` payload에도 존재해 shape 분류가
  기존 프레임과 충돌한다. 또한 `subscribeTopic`(`websocket.js:85-92`)은 AMQP 구독 refcount만
  담당할 뿐 페이지별 메시지 콜백을 라우팅하지 않는다. 실시간 표시를 견고하게 하려면 WS에
  페이지별 리스너 레지스트리 신설 + shape 분류 강화 + 기존 call/activeflow 회귀 검증이 필요하며,
  이는 이번 스코프의 핵심 가치(조회+원클릭) 대비 복잡도/리스크가 과하다.

- **생성 흐름 (옵션 B)**:
  - `POST` 성공 시: `ActionFeedback`로 "생성이 시작되었습니다. 잠시 후 이 페이지를 다시 열면
    결과가 표시됩니다." 안내. 버튼은 중복 제출 방지를 위해 짧게 비활성화(로딩 스피너) 후 복원.
  - 페이지 **재진입(새 navigation)** 시 마운트 조회로 완료된 결과를 인라인 표시. 재진입은
    사용자 액션이므로 no-refresh 규약에 위배되지 않는다(polling도, Refresh 버튼도 아님).
  - summary가 조회 시점에 아직 `progressing`이면 content 대신 **"생성 중"** 배지 표시(R2 부분 완화).
    이 경우에도 페이지 재진입으로 갱신.

- **후속 과제(비범위)**: WebSocket `summary_updated`/`transcribe_done` 완료 이벤트 기반 실시간
  자동 갱신. 이를 위해서는 WS 리스너 레지스트리 배선 + shape 분류 강화가 선행되어야 한다.

## 4. 변경 범위 (레포별 1 PR)

### 4.1 monorepo (백엔드) — PR #1

**bin-openapi-manager**
- `openapi/paths/aisummaries/main.yaml` GET에 `reference_type`, `reference_id` query 파라미터 추가.
  transcribes GET의 문구/제약(둘은 쌍으로 제공, 하나만 주면 400)을 그대로 채용.
  현재 GET에는 `'400'` 응답 블록이 없으므로 `'400': BadRequest`를 함께 추가한다(쌍 검증 실패 노출).
- `go generate ./...`로 `gens/models/gen.go` 재생성.

**bin-api-manager**
- `server/aisummaries.go` `GetAiSummaries`: `params.ReferenceType`/`params.ReferenceId` 파싱,
  쌍 검증(한쪽만 오면 `INVALID_REFERENCE_FILTER` 400), 새 servicehandler 메서드 호출.
- `pkg/servicehandler/aisummary.go`: 기존 `AISummaryGetsByCustomerID`를 유지하되,
  reference 필터를 받는 형태로 **확장**한다. transcribe와 정확히 대칭이 되도록
  `AISummaryList(ctx, a, size, token, referenceType, referenceID)` 신설을 기본안으로 한다.
  - **인터페이스 변경 판단**: `AISummaryGetsByCustomerID`는 `server/aisummaries.go` GET 외
    다른 호출부가 없다(grep 확인 예정). 시그니처를 바꾸는 대신 새 메서드를 추가하고
    GET 핸들러만 새 메서드로 전환, 기존 메서드는 제거하여 중복 방지.
    (최종 형태는 설계 리뷰에서 확정)
  - `convertAISummaryFilters`에 reference 필터를 병합. `customer_id`, `deleted=false`는 유지.
- 인터페이스(`pkg/servicehandler/main.go`) + mock 재생성.
- `go generate ./...` (openapi 먼저, 그 다음 api-manager).

**RST 문서 동기화** (`bin-api-manager/docsdev/source/`)
- aisummaries(또는 speaking/summary 계열) overview/tutorial/struct에 GET reference 필터 예시 추가.
- clean rebuild + `git add -f build/`.

**검증**: bin-api-manager, bin-common-handler 대상 전체 verification workflow
(`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`).

### 4.2 monorepo-javascript (프론트) — PR #2

**square-admin `src/views/recordings/recordings_detail.js`**
- 레코딩 상세 카드 하단에 두 개 카드 추가:

**Transcript 카드**
- 마운트 시 `GET transcribes?reference_type=recording&reference_id={id}` 조회.
- 결과 있으면: **transcribe(언어)별로 그룹핑**하여 표시. 각 그룹은 언어 배지(예: `en-US`)를
  헤더로 갖고, 그 안에서 해당 transcribe의 `GET transcripts?transcribe_id={tid}` 텍스트를
  direction 구분 + `tm_transcript` 오름차순으로 표시. (R4: 여러 언어 transcribe를 시간순으로
  섞으면 동일 발화의 언어별 버전이 뒤섞여 가독성이 깨지므로, 언어별 그룹핑으로 분리한다.)
  transcribe가 하나면 그룹 헤더 없이 바로 표시해도 무방.
  - **진행 중 상태(R5)**: transcribe의 `status`가 `progressing`이거나, `done`이지만 transcripts가
    아직 비어있으면(생성 직후 재진입한 중간 상태), 해당 그룹에 텍스트 대신 **"생성 중"** 배지를
    표시한다. transcribe `WebhookMessage`는 `status` 필드(`progressing`/`done`)를 노출하므로
    summary와 대칭으로 판별 가능(`bin-transcribe-manager/models/transcribe/webhook.go`,
    `transcribe.go`의 `StatusProgressing`→`StatusDone`). 빈 그룹 dead-end를 방지한다.
- 결과 없으면: 언어 드롭다운(기본값 자동, 3.1) + **Transcribe** 버튼. 클릭 시 `POST /transcribes`
  → 성공 시 생성 시작 안내(아래 공통 안내 문구), 버튼 잠시 비활성화 후 복원 (D5 옵션 B).
  recording 대상은 멱등이라 재클릭해도 안전.

**AI Summary 카드**
- 마운트 시 `GET aisummaries?reference_type=recording&reference_id={id}` 조회 (D1 신규 필터).
  응답은 리스트이므로 `tm_create` 내림차순 정렬 후 **첫 건(최신)** 을 선택한다.
- 결과 있으면: 최신 1건의 `content`를 표시 + **Regenerate** 버튼(D2). status가 `progressing`이면
  content 대신 **"생성 중"** 배지 표시(R2: 페이지 재진입으로 갱신).
- 결과 없으면: 언어 드롭다운(기본값 자동) + **Summarize** 버튼. 클릭 시 `POST /aisummaries`
  (on_end_flow_id는 NIL_UUID) → 성공 시 생성 시작 안내 + 버튼 잠시 비활성화 (D5 옵션 B).
- Regenerate: `ConfirmDialog`로 확인 후 `POST /aisummaries` 재호출 → 위와 동일한 안내.
  확인 다이얼로그로 오연타 방지.

**진행 중 액션 잠금(R6)**: transcribe/summary가 "생성 중"(progressing 또는 빈 결과 중간 상태)인
동안에는 해당 카드의 생성/재생성 액션(Transcribe / Summarize / Regenerate)을 **비활성화**하여
중복 클릭을 막는다.

**공통 안내 문구(R3)**: 생성 POST 성공 시 `ActionFeedback`로 다음을 안내한다.
"생성을 시작했습니다. 트랜스크립션은 보통 수십 초, AI 요약은 최대 1분가량 걸릴 수 있습니다.
완료되면 목록에서 이 레코딩을 다시 선택해 결과를 확인하세요." (재진입 방법과 소요 시간 기대치를
명시하여, 실시간 자동 갱신 부재로 인한 오인을 방지.)

**UI 규약 준수** (square-admin CLAUDE.md)
- `alert()`/`window.confirm()` 금지 → `ActionFeedback`, `ConfirmDialog` 사용.
- `NIL_UUID`는 `src/constants`에서 import.
- Refresh/Reload 버튼 금지. 데이터 갱신은 사용자 액션(페이지 재진입, 재생성) 후 마운트 조회로만.
- `useDetailResource`의 transform은 referentially stable해야 함(무한 루프 주의).
- 타이포그래피 유틸(`SECTION_HEADER` 등) 사용.

> **참고 (D5 옵션 B)**: `src/websocket.js`는 **변경하지 않는다.** 실시간 완료 자동 갱신은 후속
> 과제이며, 그때 WS 리스너 레지스트리 배선 + shape 분류 강화를 별도로 설계한다.

**테스트**
- `__tests__/recordings_detail.test.js`에 케이스 추가:
  - transcript 있음(단일/다국어 그룹핑) / 없음
  - **transcribe progressing 또는 done이지만 transcripts 빈 중간 상태 → "생성 중" 배지**(R5)
  - summary 있음(done) / 없음 / **progressing 상태 렌더**(R2: content 대신 "생성 중" 배지)
  - summary 리스트 다건 → `tm_create` 최신 1건 선택
  - Transcribe / Summarize 버튼 클릭 → 각 POST 호출 + body 검증 + **생성 시작 안내 노출**
  - **생성 중 상태에서 생성/재생성 버튼 비활성화**(R6)
  - **POST 실패 시 ActionFeedback 오류 노출**
  - Regenerate 확인 다이얼로그 → 승인 시 POST 재호출
  - **기본 언어 산출 로직**(navigator.language: `ko`→`ko-KR`, `en-US`→그대로, 미지원→`en-US`)
- Test gate: 브랜치가 main 대비 실패 수 증가 없어야 PR (monorepo-javascript CLAUDE.md).

## 5. 비범위 (Out of scope)

- **실시간(WebSocket) 완료 자동 갱신** (`transcribe_done`/`summary_updated` 구독) — 후속 과제.
  현재 admin WS의 shape 기반 분류가 기존 call/activeflow 프레임과 충돌하고 페이지별 콜백 배선이
  없어, 리스너 레지스트리 신설 등 별도 설계가 필요하다 (3.2 D5 옵션 B).
- 실시간 transcript **인터림 스트리밍**(발화 중 단어 단위 표시) — recording은 완료된 파일이라 불필요.
- transcribe/summary 삭제 UI (기존 전용 메뉴에서 처리).
- summary 히스토리(여러 개) 목록 표시 — 최신 1건 + Regenerate로 충분(D2).
- service_agents(상담사) 화면 — 이번은 admin(Admin/Manager 권한)만.
- 고객/큐 단위 기본 언어 설정 — 후속 과제(3.1).

## 6. 리스크 / 주의

- **R1**: `AISummaryGetsByCustomerID` 시그니처 변경 시 다른 호출부 파급. → grep으로 호출부
  전수 확인 결과 호출부는 GET 핸들러뿐(다른 worktree 제외). 새 메서드 추가 + GET 전환 + 기존 제거로 단일화.
- **R2**: summary/transcribe는 비동기라 POST 직후 결과가 비어있음. → **옵션 B로 완화**: 생성 시작
  안내 후, 페이지 재진입 시 마운트 조회로 결과 표시. progressing이면 "생성 중" 배지. 실시간 자동
  갱신은 비범위(후속).
- **R3**: recording 언어 정보 부재로 잘못된 언어로 transcribe/summary 생성 가능. → 드롭다운으로
  변경 가능하게 하고, 기본값을 브라우저 로케일로(2단계 정규화). (D3/3.1)
- **R4**: transcribe가 여러 개(언어별) 존재 가능. → **해소**: 언어별 그룹핑 표시(4.2 Transcript 카드).
  시간순 병합으로 언어가 섞이지 않도록 함.
- **R5**: 원클릭 생성 후 재진입 시 transcribe 레코드는 있으나 transcripts가 아직 비어있는 중간
  상태 → 빈 그룹 dead-end. → **해소**: transcribe `status`/빈 결과 판별로 "생성 중" 배지 표시
  (summary와 대칭, 4.2 Transcript 카드 R5).
- **R6**: "생성 중" 동안 생성/재생성 버튼 노출 시 중복 클릭 위험. → **해소**: progressing/중간 상태
  동안 해당 액션 버튼 비활성화(4.2 R6).

## 7. 검증 계획

- 백엔드: 신규 servicehandler 메서드 단위 테스트(reference 필터가 실제 RPC filters로 전달되는지),
  server GET HTTP 테스트(쌍 검증 400 포함).
- 프론트: 위 4.2 테스트 케이스. 빌드 성공.
- 대표님 실브라우저 검증(리뷰 루프 통과 후에도 필수).

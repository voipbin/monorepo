# 설계: tm_transcript를 오프셋 숫자 컬럼으로 재설계 (VOIP-1528)

작성일: 2026-09-16
작성자: Hermes (CPO)
티켓: VOIP-1528
브랜치: VOIP-1528-Redesign-tm_transcript-offset-column
상태: Draft (설계 리뷰 대기)

이슈 분석 문서: `~/agent-hermes/notes/2026-09-15-transcript-zerodate-scan-analysis.md`
(근본 원인: tm_transcript가 세그먼트 발화 오프셋을 연도 0001 DATETIME으로 인코딩 →
DATETIME 범위 밖 → DB가 zero-value 치환 저장 → 읽기 파싱 실패 "month out of range" →
transcript scan 실패 → transcribe가 progressing 잔류. 이슈 분석은 4라운드 리뷰로
2연속 Approve 완료. 방향 C(오프셋 숫자 컬럼 재설계) 대표님 확정.)

## 1. 문제 정의

`transcribe_transcripts.tm_transcript`(DATETIME)는 벽시계 시각이 아니라 "전사 시작점으로부터
세그먼트 발화까지의 경과 오프셋"을 담는 필드다. 이 오프셋을 연도 0001 기준 시각으로 인코딩한
것이 타입 오용이며, DB DATETIME 범위(1000~9999) 밖이라 zero-value로 저장되어 읽기 파싱이
깨진다. 오프셋을 오프셋답게(숫자) 모델링해 근본 해소한다.

## 2. 목표 (numbered, testable)

1. `transcribe_transcripts`에서 시각 인코딩된 `tm_transcript`(datetime)를 오프셋 숫자 컬럼으로
   대체한다. transcript 조회가 scan 실패 없이 성공한다.
2. recording transcribe가 정상적으로 `done`까지 전이한다(장애 해소).
3. 세그먼트별 발화 오프셋(밀리초)이 API 응답과 프론트에 정확히 표시 가능해진다.
4. cross-service 소비자(bin-ai-manager 렌더러)가 새 오프셋 타입으로 정상 동작한다.
5. 전 서비스 build/test/lint 통과, OpenAPI/RST 문서 정합.

## 3. Decisions locked (대표님 확정 2026-09-16)

- 방향 C 채택(A/B 미채택). tm_transcript를 숫자 오프셋 컬럼으로 재설계.

## 4. 설계 결정 (리뷰에서 검증 대상)

### 4.1 새 컬럼명/타입 (설계 리뷰 R1·R2 반영 확정)
- 컬럼: `tm_transcript` (datetime(6)) → **`offset_ms` (bigint, 밀리초)** 로 대체.
  - **`offset`(무접미사)은 부적합 — 확정 `offset_ms`.** 근거 2가지(R2 지적, 실측 확인):
    (1) `offset`은 MySQL/MariaDB 예약어이고, dbhandler는 squirrel SetMap(PrepareFields)/
    GetDBFields로 식별자를 backtick 없이 평문 생성 → `INSERT INTO ... (offset,...)`가 SQL
    문법 오류 위험. (2) 기존 컨벤션이 밀리초 필드에 `_ms` 접미사를 씀(schedule의 duration_ms,
    timeout_ms 실측 확인). 예약어 회피 + 컨벤션 정합 모두 `offset_ms`를 가리킨다.
  - 값의 본질: "전사 시작점으로부터의 경과 밀리초". `bigint`(int64)면 통화/녹취 길이에 충분.
- Go 모델 필드: `TMTranscript *time.Time` → **`OffsetMs int64`** (models/transcript/transcript.go),
  json 태그 `offset_ms`.
  - **타입 폭 int64로 통일(R1 지적):** 컬럼 bigint / OpenAPI int64 / Go int64로 3계층 일치.
    밀리초 오프셋은 int32 범위(약 24.8일)를 초과할 수 있으므로 `int`가 아닌 `int64` 명시(32비트
    빌드 안전성 + 표기 일관성).
  - nullable 불필요(비포인터): 모든 세그먼트는 항상 오프셋을 가진다(0 이상). zero 값(0)은
    "전사 시작점"이라는 유효 의미를 가지므로 nil/포인터 불필요. (기존 컨벤션: groupcall
    CallCount/GroupcallCount(int), message.Sequence(int) 등 카운트/시퀀스 비포인터와 일치 —
    R1 확인.)

### 4.2 값 생성 (process.go)
- `convertTime(duration time.Duration) *time.Time` → **`toOffsetMs(duration time.Duration) int64`**
  (`return duration.Milliseconds()`).
- process.go:70,82의 `TMTranscript: convertTime(sentenceStart)` → `OffsetMs: toOffsetMs(sentenceStart)`.

### 4.3 정렬 (recording.go)
- `sortTranscriptsByTMTranscript`는 그대로 오프셋 기준 정렬로 유지하되 필드만 교체:
  `transcripts[i].OffsetMs < transcripts[j].OffsetMs` (int64 비교, nil 가드 불필요).
  - 함수명도 `sortTranscriptsByOffset`로 rename(선택, 일관성). 참고: 이 sort는 write 이전
    메모리 정렬용. read 순서는 여전히 tm_create(변경 없음).

### 4.4 dbhandler / handler 시그니처
- `TranscriptCreate`: `TMTranscript` 세팅 제거, `OffsetMs` 세팅. (Create 시 tm_create=now는 유지.)
- `transcriptHandler.Create` 시그니처: `tmTranscript *time.Time` → `offsetMs int64`.
  - 호출부 recording.go:47 `h.Create(..., tmp.OffsetMs)`, transcript.go:35 등 함께 수정.
- `field.go`: `FieldTMTranscript = "tm_transcript"` → `FieldOffsetMs = "offset_ms"`.

### 4.5 외부 노출 (WebhookMessage + OpenAPI)
- `WebhookMessage.TMTranscript *time.Time` → **`OffsetMs int64 json:"offset_ms"`**
  (models/transcript/webhook.go). ConvertWebhookMessage 매핑 수정.
- OpenAPI openapi.yaml: `tm_transcript` (string/date-time/x-go-type:string) → **`offset_ms`**:
  ```
  offset_ms:
    type: integer
    format: int64
    x-go-type: int64
    description: "Offset in milliseconds from the start of the transcription when this segment was spoken."
    example: 3500
  ```
  - **gen 타입 명세 (R3→R4 정정):** `x-go-type: int64`는 기저 타입만 정하며 **포인터 여부를
    결정하지 않는다**(실측: gen.go Id 필드가 x-go-type:string인데도 `*string,omitempty` 포인터로
    생성됨). 포인터/비포인터는 스키마 `required` 목록 또는 skip-optional-pointer 옵션이 정한다.
    다만 이는 **서버 응답 직렬화와 무관**하다(아래 핵심 참고). gen 모델 타입은 표기 일관성을 위해
    int64로 두되, 서버 테스트 리터럴 값을 결정하는 것은 gen이 아니라 WebhookMessage다.
- **핵심(R3→R4 정정): 서버 응답은 gen 모델이 아니라 WebhookMessage를 직렬화한다.**
  server/transcripts.go:60 `GenerateListResponse(tmps, ...)`는 transcribe-manager의
  `tmtranscript.WebhookMessage`(models/transcript/webhook.go)를 직렬화한다. 따라서 서버 테스트
  리터럴의 신규값 `"offset_ms":0`은 **`WebhookMessage.OffsetMs int64`(비포인터)** 가 결정한다
  (gen 모델 x-go-type/포인터 여부와 무관). 비포인터 int64라 zero 값도 항상 `"offset_ms":0`으로
  직렬화(null/생략 없음). 구현자는 gen 모델 포인터 여부를 서버 테스트 통과 조건으로 오인하지 말 것.
- bin-api-manager gen 타입 재생성. **redoc 산출물도 재생성 대상(R1 지적):**
  bin-api-manager/gens/openapi_redoc/openapi.json + api.html (tm_transcript + date-time 예시
  포함, 실측 확인). 재생성 후 force-add.
- **API 계약 변경(breaking)**: 필드명 tm_transcript→offset_ms, 타입 string→integer. 프론트가
  현재 tm_transcript를 실사용하지 않으므로(6절) 사용자 영향 없음. 단 계약 변경이므로 RST/OpenAPI
  갱신 필수. (외부 webhook 구독자 논거는 §9 참조.)

### 4.6 cross-service 소비자 (bin-ai-manager)
- `renderTranscriptLine`(tool_resource.go:515-521)이 `t.TMTranscript`를 디코딩해
  `[in 00:00:03] hello`를 만든다. → `t.OffsetMs`(밀리초)에서 직접 HH:MM:SS 계산으로 변경.
  **비포인터 int64로 바뀌므로 `--:--:--` 폴백은 도달 불가 데드코드 → 제거(R1 지적):**
  ```
  sec := t.OffsetMs / 1000
  offset := fmt.Sprintf("%02d:%02d:%02d", sec/3600, (sec%3600)/60, sec%60)
  return fmt.Sprintf("[%s %s] %s", t.Direction, offset, t.Message)
  ```
  - tool_resource.go:512-514의 zero-time 인코딩 전제 주석도 제거/갱신.
  - tool_resource_test.go의 `TMTranscript: nil → --:--:--` 기대 케이스(약 170-172행)는 비포인터
    전환으로 표현 불가 → 삭제/재작성.
  - bin-ai-manager는 tmtranscript 모델을 import하므로 모델 변경이 전파됨. 함께 빌드·테스트.

## 5. Affected files (repo별)

### monorepo (백엔드) — PR 1
| 파일 | 변경 |
|------|------|
| bin-dbscheme-manager/.../versions/<new>.py | ALTER **`transcribe_transcripts`**(레거시 `transcripts` 아님): tm_transcript drop, offset_ms bigint add. down_revision=현재 head, `after message` 위치 유지, downgrade 역작업 |
| bin-transcribe-manager/models/transcript/transcript.go | TMTranscript *time.Time → OffsetMs int64, **db 태그 `db:"tm_transcript"` → `db:"offset_ms"`**, 주석 갱신. (dbhandler는 PrepareFields/GetDBFields 리플렉션으로 db 태그에서 컬럼을 파생하므로 이 태그 변경만으로 INSERT/SELECT 컬럼이 자동 전파됨) |
| bin-transcribe-manager/models/transcript/field.go | FieldTMTranscript → FieldOffsetMs = "offset_ms" |
| bin-transcribe-manager/models/transcript/webhook.go | WebhookMessage 필드 + ConvertWebhookMessage |
| bin-transcribe-manager/pkg/transcripthandler/process.go | convertTime → toOffsetMs(int64), 세팅부 |
| bin-transcribe-manager/pkg/transcripthandler/recording.go | sort 필드/함수명 교체, Create 호출부 |
| bin-transcribe-manager/pkg/transcripthandler/transcript.go | Create 시그니처 offsetMs int64 |
| bin-transcribe-manager/pkg/dbhandler/transcript.go | Create 세팅, (List ORDER BY tm_create 유지) |
| **bin-transcribe-manager/scripts/database_scripts_test/table_transcripts.sql (R4 blocker)** | L11 `tm_transcript datetime(6)` → `offset_ms bigint`. main_test.go가 이 SQL을 SQLite에 로드하고 Test_TranscriptCreate가 INSERT하므로 미갱신 시 "no column named offset_ms"로 실패 |
| bin-transcribe-manager/pkg/transcripthandler/mock_main.go | Create 시그니처 변경으로 재생성(go generate 자동, 명시) |
| bin-transcribe-manager _test.go (컴파일 깨짐, 명시) | models/transcript/transcript_test.go(L35/L55/L56 zero/date round-trip 제거), pkg/dbhandler/transcript_test.go(L45/L57), pkg/transcripthandler/transcript_test.go(L57/L68), field_test.go, webhook_test.go, recording_test.go(date 정렬→offset), listenhandler/v1_transcripts_test.go(L58,90 `"tm_transcript":null` → `"offset_ms":0`) |
| bin-ai-manager/pkg/aicallhandler/tool_resource.go | renderTranscriptLine offset 디코딩 + 폴백 데드코드 제거 + 주석 |
| bin-ai-manager/pkg/aicallhandler/tool_resource_test.go | nil→`--:--:--` 케이스 삭제/재작성, 렌더 기대값 갱신 |
| bin-openapi-manager/openapi/openapi.yaml | tm_transcript → offset_ms (integer/int64/x-go-type:int64) |
| bin-openapi-manager/gens/models/gen.go | 재생성 |
| bin-api-manager/gens/openapi_server/gen.go | 재생성 |
| bin-api-manager/server _test.go (런타임 assertion, 명시 — R1·R2 공통 지적) | transcripts_test.go(L58·L156 `"tm_transcript":null` → `"offset_ms":0`), service_agents_transcripts_test.go(L60 동일). x-go-type:int64로 비포인터라 신규값은 `0`(null/생략 아님) |
| bin-api-manager/gens/openapi_redoc/openapi.json + api.html | 재생성 + force-add |
| RST source 9개(아래) | offset_ms로 갱신 + clean rebuild + build force-add |

**RST 대상 9개 파일(실측 확인, glob 아닌 명시 목록 — R1·R2 지적):**
transcript_struct_transcript.rst, transcribe_struct.rst(L141~166 0001-01-01 센티넬 설명 +
"sort by tm_transcript" — 이번 재설계로 완전히 틀리게 되는 핵심 문서),
transcribe_overview.rst(L281·378·386·402·410·659), transcribe_tutorial.rst,
quickstart_transcribe.rst, recording_tutorial.rst, call_scenarios.rst,
ai_voice_agent_integration_tutorial.rst, architecture_dataflow.rst.

### monorepo-javascript (프론트) — PR 2
| 파일 | 변경 |
|------|------|
| square-admin/src/types/api.ts | tm_transcript: string → offset_ms: number |
| square-admin/src/views/contacts/CaseCallTranscriptPanel.js | 주석 갱신 + offset_ms를 HH:MM:SS 배지로 표시(결정 (ii)) |
| square-admin/.../__tests__/CaseCallTranscriptPanel.test.js | fixture 필드 교체(offset_ms 숫자) + 타임코드 표시 테스트 |

## 6. 범위 / 프론트 처리 결정 (리뷰 확인)

현재 square-admin의 CaseCallTranscriptPanel.js는 tm_transcript를 **실사용하지 않고**
tm_create로 정렬한다(코드 확인). 따라서 프론트 PR의 필수 작업은 "타입 정의(api.ts)를 새 계약에
맞추고 fixture/주석을 갱신"하는 것까지다. 발화 오프셋을 UI에 실제로 표시(예: 각 줄 앞에
[00:00:03])하는 것은 목표3의 실현이지만 별도 UX 결정이 필요하다.

- 옵션 (i): 프론트 PR은 타입 정합만(offset 필드 수용, 표시는 미도입). 최소.
- 옵션 (ii): 프론트 PR에서 offset을 실제 표시(타임코드 배지)까지 구현. 목표3 완성.

**결정 (대표님 확정 2026-09-16): (ii) 채택.** 프론트 PR에서 각 transcript 줄 앞에
`[00:00:03]` 형태의 타임코드 배지를 실제로 표시하는 것까지 구현한다. CaseCallTranscriptPanel.js가
offset(밀리초)을 HH:MM:SS로 포맷해 각 라인에 렌더한다. api.ts 타입 정합 + fixture 갱신 +
표시 로직 + 관련 테스트가 프론트 PR 범위에 포함된다.

## 7. 마이그레이션 / 데이터

- Alembic: `alter table transcribe_transcripts drop column tm_transcript;` +
  `add column offset_ms bigint after message;` (신규 revision을 `alembic revision`으로 생성,
  down_revision=현재 head, downgrade는 역작업: offset_ms drop + tm_transcript datetime(6) 복원).
- 기존 데이터: 프로덕션 확인 결과 복구 대상 없음(과거 transcript 빈 배열, 장애 row는 조회 500).
  백필 불필요. 마이그레이션이 tm_transcript를 drop하므로 기존 zero-date row 잔재도 함께 제거됨.

## 8. 검증 계획

- **OpenAPI-first 순서(bin-openapi-manager CLAUDE.md 규칙):** openapi.yaml 수정 →
  bin-openapi-manager에서 `go generate ./...` FIRST → 이어서 bin-api-manager에서 `go generate ./...`.
  이번 변경은 신규 endpoint가 아니라 필드 타입 변경뿐이므로 server/servicehandler 시그니처 변경
  불필요(필드 rename만 gen에 반영).
- bin-transcribe-manager: 전체 verification workflow(go mod tidy/vendor/generate/test/lint).
- bin-ai-manager: renderTranscriptLine 오프셋 렌더 테스트 + 전체 verification.
- bin-api-manager: **전체 verification workflow(go mod tidy && go mod vendor** 포함) + gen(server
  + redoc) + RST clean rebuild(rm -rf build 후) + build force-add. (R2 지적: bin-api-manager는
  bin-transcribe-manager transcript 모델을 vendor하므로 vendor 갱신 없이는 gen/test가 stale
  TMTranscript 모델을 참조한다. mod tidy/vendor 필수.)
- bin-openapi-manager: oapi-codegen 검증 + consumer build.
- 회귀 테스트(비자명 지점 명시):
  - offset_ms 기반 Create→List round-trip, 오프셋 정렬, ai-manager 렌더 포맷([in 00:00:03]).
  - bin-transcribe-manager: listenhandler/v1_transcripts_test.go L58,90 `"tm_transcript":null` →
    `"offset_ms":0`. models/transcript/transcript_test.go zero/date round-trip 제거.
    pkg/dbhandler/transcript_test.go, pkg/transcripthandler/transcript_test.go 필드 교체.
    recording_test.go date 정렬 → offset_ms.
  - **bin-api-manager 서버 테스트(R1·R2 공통 지적): server/transcripts_test.go L58·L156,
    server/service_agents_transcripts_test.go L60의 `"tm_transcript":null` → `"offset_ms":0`.**
    (런타임 assertion 실패라 build gate가 안 잡음 — 반드시 명시적 수정.)
  - **테스트 스키마 픽스처(R4 blocker): scripts/database_scripts_test/table_transcripts.sql L11
    `tm_transcript datetime(6)` → `offset_ms bigint`.** 미갱신 시 dbhandler Create round-trip이
    "no column named offset_ms"로 실패. (offset_ms round-trip 테스트의 선행 조건.)
  - tool_resource_test.go의 nil→`--:--:--` 케이스 삭제/재작성.
- 프론트: 관련 vitest/jest 스위트 통과 + 타임코드 배지 표시 테스트, lint.

## 9. 롤아웃 / 리스크

- API 계약 breaking(tm_transcript→offset_ms, string→integer).
  - **프론트 영향 없음**: square-admin이 현재 tm_transcript 미사용(6절).
  - **외부 webhook 구독자 영향도 실질 위험 낮음(R2 지적 반영)**: 현재 tm_transcript는 버그로
    항상 zero-value(0001-01-01 손상값)를 반환해 사실상 무의미한 데이터를 준다. 외부 구독자도 이
    필드를 의미 있게 신뢰할 수 없었으므로 rename+retype의 실질 파괴 위험이 낮다. 필드가 비기능
    상태였으므로 deprecation window 불필요. RST/OpenAPI로 필드 변경을 명시하고 webhook 이벤트
    필드도 함께 바뀜을 문서화한다.
- 배포 순서: 백엔드(스키마+API) 먼저 배포·검증 후 프론트. 프론트는 offset_ms 부재도 방어적으로
  견디게(값 없으면 배지 미표시).
- cross-service: bin-ai-manager가 모델 변경에 직접 의존하므로 동일 PR(백엔드)에서 함께 수정
  (레포 동일 monorepo이므로 1 PR).

## 10. Open questions → 전부 해소 (설계 리뷰 + 대표님 확정)

1. 컬럼/필드명: **`offset_ms` 확정.** `offset`은 MySQL 예약어(squirrel backtick 미사용으로 SQL
   오류 위험) + `_ms` 접미사 컨벤션(duration_ms/timeout_ms 실측). (R2 blocking 해소)
2. 프론트 범위: **(ii) 확정** — 타임코드 배지 표시까지 구현. (대표님 2026-09-16)
3. 정밀도: **밀리초(ms) 확정.** UI 표시는 초 단위(HH:MM:SS)이고 정렬/식별에 ms면 충분. us 불필요.
   (R2: 비-blocking, §4.1 근거 충분)

## 11. 설계 리뷰 대응 이력

- 설계 R1(CHANGES_REQUESTED): RST glob 과소(9개 명시 필요), redoc 산출물 누락, 타입 폭
  불일치(int→int64), 렌더러 폴백 데드코드, listenhandler 테스트 리터럴 지점 → §4.1/4.5/4.6/5/8
  전부 반영.
- 설계 R2(CHANGES_REQUESTED): `offset` 예약어 blocking → `offset_ms` 확정(§4.1), RST 9개
  명시(§5), OpenAPI-first 순서 명시(§8), webhook 외부영향 논거(§9), open Q1 blocking 판정 →
  전부 반영.
- 설계 R3(CHANGES_REQUESTED, 개정본): bin-api-manager 서버 테스트 리터럴 3곳 누락(R1·R2
  공통) → §5/§8 추가. gen 계층 x-go-type:int64 미명시(포인터/값 결정) → §4.5 명시. 테스트
  파일명 정밀화(dbhandler/transcripthandler transcript_test.go) → §5. 마이그레이션 테이블명
  `transcribe_transcripts` 명시 → §5/§7. bin-api-manager vendor 갱신(mod tidy/vendor) → §8.
  전부 반영.
- 설계 R4(CHANGES_REQUESTED, 2차 개정본): (1) §4.5 x-go-type 근거 오류 정정 — x-go-type은
  포인터 여부를 정하지 않으며, 서버 응답은 gen 모델이 아니라 WebhookMessage를 직렬화하므로
  `"offset_ms":0`은 WebhookMessage.OffsetMs 비포인터가 결정(실측 반증 반영). (2) 테스트 스키마
  픽스처 scripts/database_scripts_test/table_transcripts.sql 누락(신규 blocker) → §5/§8 추가.
  (3) transcripthandler mock 재생성 명시 → §5. 전부 반영.

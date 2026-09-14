# Recording Detail: Show Transcripts As-Is (Owner-agnostic) — Design

Date: 2026-09-14
Author: CPO (Hermes) with pchero
Status: Design (review loop)
Related analysis: 2026-09-14-recording-summary-source-transcript-analysis.md (issue analysis, 2연속 Approve)

## 0. 결정 요약 (대표님 확정)

- **레이아웃**: E안 — 원문(Transcript)이 본문의 주인공, 상단에 AI Summary 배너. 4상태 정의(§4).
- **구현 방향**: 선택 B — "이 recording에 딸린 transcript를 소유자 무관하게 있는 그대로 조회해 표시".
  대표님 완화("Summary가 쓴 transcript와 표시 transcript가 반드시 일치할 필요 없다, 있는 그대로 보여주면 된다")로
  summary↔transcript 매칭/연결/동봉/재파생 로직이 전부 불필요해짐.
- **소유권 변경 아님**: transcribe 소유자(IDAIManager)는 그대로 둔다. 읽기 경로 하나만 recording 소유권으로 게이트해 연다.

## 1. 배경 (분석 확정 사실 요약)

- 요약이 내부적으로 만든 transcribe는 `cmcustomer.IDAIManager` 소유로 DB에 영구 저장된다
  (`bin-transcribe-manager`, reference_type=recording, reference_id=recordingID).
- 현행 프론트 두 카드는 이미 독립 조회다:
  - TranscriptCard: `GET /transcribes?reference_type=recording&reference_id={id}`
  - AISummaryCard: `GET /aisummaries?reference_type=recording&reference_id={id}`
- 그러나 gateway `TranscribeList`(servicehandler)가 `filters["customer_id"] = a.CustomerID`를 **무조건 강제**하므로,
  IDAIManager 소유 transcribe는 고객/admin 조회로 잡히지 않는다. 그래서 스크린샷에서 "No transcript yet".
- billing 영향 없음(신규 STT 없음, 기존 데이터 조회만). 분석 문서에서 코드 인용으로 확정.

## 2. 목표 / 비목표

**목표**
- 레코딩 상세에서, 그 recording에 존재하는 transcript를 **소유자 무관**하게 있는 그대로 표시한다
  (요약이 만든 IDAIManager 소유 transcript 포함).
- E안 레이아웃과 4상태(§4)를 구현한다.

**비목표**
- summary와 transcript를 정확히 매칭/연결하지 않는다(대표님 완화). "summary source" 뱃지 등 연결 표기 안 함.
- transcribe 소유권 변경 안 함. 고객 transcribe 목록(별도 메뉴)에 IDAIManager transcribe를 노출하지 않음
  (본 변경은 recording 상세의 조회 경로에 한정).
- SQUARE-69 이슈 B(고객 Transcribe 버튼 멱등키 문제)는 별도 트랙. 본 설계 범위 밖(§7).
- 실시간 WebSocket 자동 갱신 안 함(재진입 갱신). 이전 작업과 동일 정책.

## 3. 백엔드 설계 (선택 B)

### 3.1 새 조회 경로: recording 소유권으로 게이트된 owner-agnostic transcribe 조회

문제: 기존 `TranscribeList`는 `customer_id = a.CustomerID`를 강제 → IDAIManager transcribe 안 잡힘.
해결: **recording 소유권을 검증한 뒤, 그 recording에 딸린 transcribe를 customer_id 필터 없이 조회**하는
경로를 추가한다. 안전장치는 기존 `RecordingGet` 패턴을 그대로 재사용한다.

servicehandler 신규 메서드(안):
```
func (h *serviceHandler) RecordingTranscribeList(ctx, a, recordingID, size, token) ([]*tmtranscribe.WebhookMessage, error) {
    if a.IsDirect() { return nil, ErrDirectAccessNotSupported }

    // 1) recording을 가져와 소유권 게이트 (RecordingGet과 동일 패턴)
    rec, err := h.recordingGet(ctx, recordingID)
    if err != nil { return nil, err }
    if !h.hasPermission(ctx, a, rec.CustomerID, PermissionCustomerAdmin|PermissionCustomerManager) {
        return nil, ErrPermissionDenied
    }

    // 2) recording에 딸린 transcribe를 owner 무관 조회
    //    (customer_id 필터 없음. reference_type=recording, reference_id=recordingID로 한정)
    filters := { "deleted":"false", "reference_type":"recording", "reference_id": recordingID }
    tmps, err := h.reqHandler.TranscribeV1TranscribeList(ctx, token, size, typedFilters)
    ...
}
```

핵심 안전성 논거:
- **격리 우회 아님**: recording은 고객 소유 자원이고, §3.1은 recording의 소유 고객(`rec.CustomerID`)에 대해
  요청자가 admin/manager인지 검증한다. 그 recording에 딸린 transcribe만 반환하므로(reference_id 한정),
  타 고객 자원 노출 없음. IDAIManager transcribe라도 "이 고객 recording의 부산물"이므로 노출이 정당.
- **소유권 무변경**: transcribe.CustomerID는 IDAIManager 그대로. 조회 경로만 recording 게이트로 연다.
- 기존 `TranscribeList`(고객 소유 transcribe 조회, top-level /transcribes)는 **변경하지 않는다**.
  본 경로는 recording 하위 자원 조회로 분리한다(§3.2 엔드포인트).

### 3.2 API 엔드포인트 (OpenAPI-first)

recording 하위 자원 조회로 표현한다(REST 계층 구조 = 소유권 게이트를 URL에 반영):
```
GET /recordings/{id}/transcribes?page_size=&page_token=
```
- bin-openapi-manager: `openapi/paths/recordings/` 하위에 서브패스 추가. gen → api-manager gen.
- server 핸들러: `RecordingTranscribeList` 호출, page_size 서버 클램프(기존 transcripts 100 클램프와 동일 관례).
- 권한: PermissionCustomerAdmin|Manager (recording 소유 고객 기준). Agent-facing 아님(admin/manager 서피스).
- 대안 검토: 기존 `GET /transcribes`에 `owner_agnostic=true` 같은 플래그를 다는 방식은 top-level 조회의
  customer_id 강제 규약을 훼손하므로 배제. recording 하위 경로가 소유권 게이트를 구조적으로 강제해 더 안전.

### 3.3 transcript 텍스트 조회 (확정: 이 경로도 막힘 → 하위 경로 필요)

transcribe 목록을 얻은 뒤 각 transcribe의 transcript 텍스트는 `GET /transcripts?transcribe_id={id}`로 조회한다.
**코드 확인 결과, 이 경로도 IDAIManager transcribe에 대해 막힌다.**

`servicehandler.TranscriptList`(transcript.go:21~)는:
```
t, err := h.transcribeGet(ctx, transcribeID)          // transcribe를 먼저 조회
if !h.hasPermission(ctx, a, t.CustomerID, Admin|Manager) {  // 그 transcribe의 소유자로 게이트
    return nil, ErrPermissionDenied
}
```
즉 transcript 조회는 **transcribe 소유자(`t.CustomerID`) 기준** 권한을 요구한다. IDAIManager 소유
transcribe의 transcript를 실제 고객 admin이 조회하면 `t.CustomerID == IDAIManager`라 권한 거부된다.
→ transcript 텍스트 조회도 §3.1과 동일하게 **recording 소유권으로 게이트된 하위 경로**가 필요하다.

해결(안): recording 하위 경로로 transcript도 노출한다.
```
GET /recordings/{id}/transcripts?transcribe_id={tid}&page_size=&page_token=
```
servicehandler 신규 메서드(안):
```
func (h *serviceHandler) RecordingTranscriptList(ctx, a, recordingID, transcribeID, size, token) {
    // 1) recording 소유권 게이트 (RecordingGet 패턴)
    rec, err := h.recordingGet(ctx, recordingID)
    if !h.hasPermission(ctx, a, rec.CustomerID, Admin|Manager) { return ErrPermissionDenied }

    // 2) transcribeID가 정말 이 recording에 딸린 것인지 검증 (교차 recording 방지)
    t, err := h.transcribeGet(ctx, transcribeID)
    if t.ReferenceType != "recording" || t.ReferenceID != recordingID {
        return ErrPermissionDenied  // 이 recording 소속이 아님
    }

    // 3) transcript 텍스트 조회 (transcribe_id 한정, customer_id 필터 없음)
    filters := { "transcribe_id": transcribeID, "deleted":"false" }
    ...
}
```
핵심 안전장치: (2) transcribeID ↔ recordingID 소속 검증으로, 유효한 recording 권한을 악용해 다른
recording/고객의 transcribe transcript를 훔쳐보는 것을 차단한다. recording 소유권(1) + 소속 검증(2)
이중 게이트.

**엔드포인트 형태 확정(리뷰 R1 반영)**: 기본안인 **하위 경로 2개**(`GET /recordings/{id}/transcribes`,
`GET /recordings/{id}/transcripts?transcribe_id=`)로 확정한다. transcribe 목록 응답에 transcript를
인라인으로 싣는 대안은 배제한다. 이유: (1) transcribe-manager RPC의 transcript 인라인 지원이 불확실해
스코프 리스크가 크고, (2) 기존 프론트가 이미 transcribes → transcripts 2단 조회 패턴(fetchAllTranscripts)을
쓰고 있어 페치 계약 변경이 최소이며, (3) 인라인 채택 시에도 §3.3의 2중 게이트는 동일하게 필요해 이득이 작다.
→ 프론트 페치 계약(§4.2)은 이 2경로로 결정적으로 확정된다.

의사코드의 문자열 리터럴(`"recording"`)은 구현 시 typed 상수 `tmtranscribe.ReferenceTypeRecording`를 사용한다.
page_size 서버 클램프는 구현 시 기존 transcripts 핸들러(server/transcripts.go, 상한 100)의 실제 클램프 값을 확인해 동일 적용한다.
빈 토큰 기본값 처리도 기존 핸들러와 동일하게 한다: `token == "" → h.utilHandler.TimeGetCurTime()`
(transcribe.go/transcript.go의 기존 관례, VOIP-1480). 신규 메서드가 보내는 토큰이 입력만으로 결정되도록 servicehandler에서 설정한다.

## 4. 프론트 설계 (E안, square-admin)

`recordings_detail.js`에 통합 카드. 상단 Summary 영역 + 하단 Transcript 본문.

### 4.1 상태 모델 (리뷰 R1 반영: 2축 × progressing 서브상태)

상태는 두 축의 독립 조합이다. Transcript 축과 Summary 축은 소유자·수명주기가 독립이므로(summary=고객 소유,
transcribe=IDAIManager 소유), 각 축을 독립적으로 렌더하고 조합은 자연히 파생된다. 이분(있음/없음)이 아니라
**축별 3값**(none / progressing / ready)으로 정의한다.

**하단 본문 = Transcript 축** (E안의 주인공)
| Transcript 상태 | 하단 본문 렌더 |
|---|---|
| ready (transcribe done + transcript rows>0) | 원문 표시(언어별 그룹핑, 시간순) |
| progressing (transcribe status=progressing, 또는 done인데 rows=0 아직) | "Transcribing… 수십초~1분 후 목록에서 다시 선택" 배지 + 빈 본문 |
| none (이 recording에 transcribe 자체가 없음) | "No transcript yet" + 언어 선택 + Transcribe 버튼 |

**상단 영역 = Summary 축**
| Summary 상태 | 상단 영역 렌더 |
|---|---|
| ready (summary done, content 있음) | AI Summary 배너(내용) + Regenerate |
| progressing (summary status=progressing) | "Summarizing…" 배지 배너 (Summarize/Regenerate 비활성) |
| none (이 recording에 summary 없음) | "No summary yet · Summarize" CTA (언어 선택 + Summarize) |

**도달 가능한 조합 예시 (완전성 확인)**
- T:ready × S:ready = 상태1(요약+원문)
- T:ready × S:none = 상태2(원문만, Summarize 유도) ← 대표님 지적
- T:ready × S:progressing = 원문 아래에서 요약 생성 중
- **T:none × S:ready = 요약 배너 + "No transcript yet"** ← 리뷰 R1 지적. retention/TTL로 transcript가
  소거되고 summary(고객 소유)만 잔존 시 실재. 하단은 none 규칙대로 Transcribe 유도, 상단은 요약 표시.
- T:progressing × S:ready / S:progressing / S:none = 각 축 규칙대로 조합 렌더
- T:none × S:none = 상태4(아무것도 없음). 통합 empty. 액션은 **단일 "Summarize"** 만 제공(§4.2 결정 참조).
  Summarize가 내부적으로 transcribe를 생성하므로, 별도 "Transcribe" 없이도 요약 후 재진입 시 하단 원문이 채워진다.
  (원문만 원하는 경우를 위해 "Transcribe"도 함께 둘지는 §4.2에서 중복 방지와 함께 결정.)

핵심: **표를 "N상태 고정"이 아니라 "두 축의 독립 렌더 규칙"으로 정의**하므로, 모든 조합(none/progressing/ready
× none/progressing/ready = 9)이 규칙으로 커버된다. 특정 조합 누락이 원천적으로 발생하지 않는다.

**owner-agnostic 조회 원칙**: 하단 Transcript 축은 §3.2 신규 경로(`GET /recordings/{id}/transcribes`)로
조회하므로, 사용자 생성/요약 부산물을 구분하지 않고 "있는 그대로" 반환한다(대표님 완화). "summary source" 등
연결 표기는 하지 않는다.

### 4.2 컴포넌트 / 데이터 페치 계약 (확정)

- `recordings_detail.js`에 E안 통합 카드. 상단 Summary 영역 + 하단 Transcript 본문. 기존
  `TranscriptCard.js`/`AISummaryCard.js`는 각 축 컴포넌트로 재구성하되 한 카드 안에 배치.
- **페치 계약(확정, §3.3 결정 반영)**:
  - Transcript 목록: `GET /recordings/{id}/transcribes` (owner-agnostic)
  - Transcript 텍스트: `GET /recordings/{id}/transcripts?transcribe_id={tid}` (기존 fetchAllTranscripts의
    page_size=100 + next_page_token 루프 패턴 유지, 경로만 하위경로로 교체)
  - Summary: 기존 `GET /aisummaries?reference_type=recording&reference_id={id}` 그대로(고객 소유라 기존 경로로 커버)
- **두 독립 fetch의 loading/error 조율**: 상단(Summary)·하단(Transcript)이 각자 Loader2/에러 폴백을 갖는다.
  한 축의 로딩/실패가 다른 축 렌더를 막지 않는다(독립). `setFeedback`(부모 recordings_detail의 단일 feedback
  state, 현행 구조)은 액션 결과 토스트를 표시하되 **최신 1건만 표시(마지막 호출 승)** 임을 명시한다. 축별 상세
  에러는 각 축의 인라인 폴백에서 보여주고, 토스트는 사용자 액션(Summarize/Regenerate/Transcribe)의 성공/실패
  피드백 용도로 한정한다. 즉 "조율"은 토스트 단일화 + 축별 인라인 에러의 이원 구조다.
- **비동기 생성 직후 동작(리뷰 R2 정정)**: no-refresh 규약상 POST(`/aisummaries` 등) 성공 후 재조회나 낙관적
  로컬 상태 전환을 하지 **않는다**. 따라서 Summarize/Transcribe 클릭 직후 화면은 여전히 해당 축이 none/empty이며
  ActionFeedback 토스트("생성이 시작되었습니다. 수십초~1분 후 목록에서 다시 선택하면 결과가 표시됩니다")만 뜬다.
  progressing 배지·결과는 **사용자가 재진입(재조회)한 뒤** 나타난다. (progressing을 즉시 보이게 하려면 별도
  낙관적 상태 설계가 필요하나, 본 설계는 no-refresh 규약을 지켜 도입하지 않는다.)
- **상태4 액션 = 단일 Summarize (결합 액션 제거, 리뷰 R2 blocking 반영)**: Summarize(POST /aisummaries)는
  내부적으로 IDAIManager transcribe를 생성하므로, 재진입 시 owner-agnostic 조회로 하단 원문이 채워진다.
  따라서 "Transcribe & Summarize" 결합 버튼은 제거한다. 이유: 결합 버튼이 고객 POST /transcribes까지 함께
  발행하면 동일 recording에 transcribe 2건(고객 소유 + IDAIManager 소유)이 생겨 owner-agnostic 조회가 둘 다
  반환 → 하단에 동일 언어 그룹이 중복 렌더된다. 대표님 완화("있는 그대로")로 중복 억제 매칭을 배제했으므로,
  중복 소스 자체를 만들지 않는 것이 옳다.
  - 상태4에서 "Transcribe"(원문만) 버튼도 둘지: 둔다. 단 Summarize와 Transcribe를 **동시에 누르는 결합은 없다**
    (각각 별도 버튼). 사용자가 Transcribe만 누르면 고객 소유 transcribe 1건, Summarize만 누르면 IDAIManager
    transcribe 1건. 둘 다(순차로) 누르면 2건이 생겨 중복 렌더될 수 있으나, 이는 사용자의 명시적 2회 액션의
    "있는 그대로" 결과이며 설계가 만드는 자동 중복이 아니다. (자동 결합만 금지.)
- progressing 서브상태 렌더는 §4.1 축별 규칙을 따른다.
- Summarize / Regenerate: 기존 `POST /aisummaries` 그대로. Regenerate는 ConfirmDialog. progressing 중 비활성.
- no-refresh 규약 준수: 수동 Refresh 버튼/폴링 금지. 비동기 생성 후 재진입 갱신. alert/confirm 금지.

### 4.3 배너 summary 다중성 / 다국어 언어 규칙 (리뷰 R1 반영)

- **summary 다중성**: regenerate/다국어로 summary가 복수일 수 있다. 배너는 기존 AISummaryCard 규약대로
  `tm_create` desc 최신 1건을 표시한다(단일 배너). 언어별 다건 표시는 하지 않는다(E안은 요약을 "길잡이"로
  축약 제시하는 것이 목적). Regenerate는 최신 위에 새 summary를 쌓고 배너는 다시 최신을 표시.
- **다국어 transcript 기본 언어**: owner-agnostic 조회가 여러 언어 transcript를 반환할 수 있다. 하단 본문은
  언어별 그룹으로 모두 표시한다(섞지 않음). 상태2(S:none) Summarize의 기본 언어는 다음 우선순위로 결정:
  (1) 표시 중 transcript가 단일 언어면 그 언어, (2) 다국어면 navigator.language 정규화값(resolveDefaultLanguage,
  기존 로직), (3) 폴백 en-US. 강제 일치는 아님(사용자 변경 가능).
- **언어 데이터 흐름(리뷰 R2 minor)**: 위 (1)의 "표시 중 transcript 언어"는 Transcript 축이 소유한
  `transcribes[].language` 목록에서 나온다. 2축 독립 원칙을 지키기 위해, 이 언어 목록은 부모
  `recordings_detail`이 Transcript 축에서 받아(lift-up) Summary 축의 Summarize 기본 언어 계산에 전달한다.
  Summary 축이 Transcript 축을 직접 참조하지 않는다(부모 경유 단방향 전달).

## 5. 테스트 계획 (구현 시 작성)

- 백엔드:
  - `RecordingTranscribeList` 권한 게이트: recording 소유 고객 admin=허용, 타 고객=ErrPermissionDenied,
    direct access=ErrDirectAccessNotSupported.
  - owner-agnostic 조회: IDAIManager 소유 transcribe도 recording 하위로 반환됨을 검증(customer_id 필터 부재).
  - reference_id 한정: 다른 recording의 transcribe가 섞이지 않음.
  - page_size 서버 클램프.
- 프론트:
  - 상태 모델 렌더: T축(none/progressing/ready) × S축(none/progressing/ready) 조합, 특히
    T:none×S:ready(요약만+원문 없음), T:ready×S:none(원문만) 회귀 테스트.
  - owner-agnostic 조회로 요약 부산물 transcript도 하단에 채워짐.
  - 신규 경로 GET 호출 인자 검증(recordings/{id}/transcribes, recordings/{id}/transcripts).
    progressing 배지. Regenerate ConfirmDialog. 두 축 독립 loading/error.
  - 마운트 해제 race(mountedRef) 회귀 방지.

### 5.1 문서(RST) 갱신 (리뷰 R1 반영, 백엔드 PR 스코프 필수)

신규 user-visible 엔드포인트(`GET /recordings/{id}/transcribes`, `GET /recordings/{id}/transcripts`)이므로
root/bin-api-manager CLAUDE.md의 MANDATORY 규약대로 `bin-api-manager/docsdev/source/`의 recording 관련
overview/tutorial/struct RST를 갱신하고, `sphinx-build` clean rebuild 후 `build/`를 force-add로 커밋한다.
응답 바디는 transcribe/transcript의 WebhookMessage 형태이므로, struct RST는 CLAUDE.md 'RST struct docs
MUST match WebhookMessage' 규약에 따라 transcribe/transcript WebhookMessage 기준으로 문서화한다(신규 필드
추가 없음, 기존 구조 재사용이면 참조로 충분).

## 6. 마이그레이션 / 호환성

- 스키마 변경 없음(대안 a의 저장/백필 불필요). 기존 요약/기존 transcript 모두 즉시 커버.
- 기존 top-level `GET /transcribes`는 무변경 → 다른 화면(transcribe 메뉴) 회귀 없음.

## 7. 스코프 경계

- **본 PR 범위**: §3(백엔드 owner-agnostic recording transcribe 조회 + 엔드포인트), §4(프론트 E안 4상태).
  레포별 1 PR(백엔드 monorepo / 프론트 monorepo-javascript).
- **범위 밖**: SQUARE-69 이슈 B(고객 Transcribe 버튼 멱등키 customer_id 미포함 → 헛도는 문제). 본 설계는
  "표시"를 owner-agnostic 조회로 해결하므로, 고객이 직접 누른 Transcribe가 IDAIManager 기존 것을 재사용하는
  동작 자체는 그대로 남는다. 단, 표시 관점에서는 owner-agnostic 조회가 그 결과도 보여주므로 사용자 체감
  문제(빈 화면)는 해소된다. 멱등키 수정은 별도 트랙 유지.
- **상태2(원문O 요약X)의 재-Transcribe 경로(리뷰 R1 반영)**: 하단 Transcript 축이 ready이면 Transcribe
  버튼은 노출하지 않는다(none일 때만 노출). 따라서 상태2에서 고객이 재-Transcribe를 시도하는 UI 경로는 없고,
  상단의 Summarize CTA만 제공된다. 이로써 SQUARE-69 멱등키 충돌 표면이 UI에서 축소된다.

## 8. 설계 리뷰에서 확정할 Open Items

1. **transcribe 보존 정책 확인(구현 전)**: transcript/transcribe에 TTL/tm_delete가 걸리는지 실제 확인.
   owner-agnostic 조회의 "기존 데이터 커버"가 데이터 잔존에 의존하며, 소거 시 T:none×S:ready 조합(§4.1)으로
   자연 처리됨은 이미 설계에 반영. 보존 정책 확인은 커버리지 기대치 문서화용.
2. **엔드포인트 형태**: §3.3에서 하위 경로 2개로 **확정 완료**(인라인 대안 배제).
3. **다국어 언어 규칙**: §4.3에서 **확정 완료**(그룹 표시 + 기본 언어 우선순위).

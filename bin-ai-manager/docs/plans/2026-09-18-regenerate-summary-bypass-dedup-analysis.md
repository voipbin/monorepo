# 이슈 분석: AI Summary Regenerate가 dedup 때문에 실제 재생성되지 않음 (VOIP-1535)

작성일: 2026-09-18
작성자: Hermes (CPO)
브랜치: VOIP-1535-regenerate-summary-bypass-dedup
단계: 이슈 확인/분석 (design 이전)

## 1. 이슈 유효성 재확인 (실코드 근거)

### 1.1 증상
admin 콘솔(square-admin) recording 상세의 AI Summary 카드에서 Regenerate 버튼을 눌러도
요약이 실제로 재생성되지 않고 기존 캐시된 요약이 그대로 다시 표시된다. VOIP-1532에서
"ko-KR 요청인데 영어 요약" 증상의 직접 원인으로 지목됨.

### 1.2 프론트 경로 (실코드 확인)
`square-admin/src/views/recordings/AISummaryCard.js`:
- L100 `createSummary`가 Regenerate 버튼(L186)과 최초 생성 버튼(L209) 양쪽에 연결됨(L225 ConfirmDialog onConfirm도 동일).
- L107 `await ProviderPost('aisummaries', { on_end_flow_id, reference_type:'recording', reference_id, language })`.
- **삭제 호출 없음**: 파일 전체에 `ProviderDelete`/`Delete` 호출이 없음(grep 확인). 즉 Regenerate는
  기존 요약을 지우지 않고 POST만 재전송한다.
- `summary` state(L27)는 로드된 요약 객체를 보유하며 `summary.language`(L158), `summary.content`(L172)를
  참조 → **summary.id도 응답에 포함**(백엔드 WebhookMessage에 id 존재), 삭제 대상 식별 가능.

### 1.3 백엔드 dedup 경로 (실코드 확인)
`bin-ai-manager/pkg/summaryhandler/start.go`:
```go
func (h *summaryHandler) Start(ctx, customerID, activeflowID, onEndFlowID, referenceType, referenceID, language) {
    language = normalizeOutputLanguage(language)              // L54
    tmp, err := h.GetByCustomerIDAndReferenceIDAndLanguage(   // L56
        ctx, customerID, referenceID, language)
    if err == nil {
        return tmp, nil                                       // L58-59: 기존 요약 그대로 반환하고 종료
    }
    ... // startReferenceType* → contentGet (신규 언어 강제/검증 코드)
}
```
`GetByCustomerIDAndReferenceIDAndLanguage`(db.go:114)는 filters
`{deleted:false, customer_id, reference_id, language}`로 List 조회, 1건이라도 있으면 반환.

**결론**: (customer_id, reference_id, language) 조합으로 삭제되지 않은 요약이 이미 있으면,
POST가 와도 `contentGet`(VOIP-1532 언어 강제/검증 로직 포함)에 도달하지 못하고 기존 것을 반환한다.
Regenerate의 이름과 실제 동작이 불일치한다. **이슈는 유효하다.**

### 1.4 삭제 경로 존재 여부 (실코드 확인 — 해결책 실현성 판단 핵심)
전체 DELETE 체인이 **이미 구현되어 있음**:
- OpenAPI: `bin-openapi-manager/openapi/paths/aisummaries/id.yaml`에 `delete:` 정의 존재.
- api-manager: `bin-api-manager/server/aisummaries.go:151 DeleteAisummariesId`, gen.go 라우팅 존재.
- servicehandler: `bin-api-manager/pkg/servicehandler/aisummary.go:197 AISummaryDelete`.
- RPC client: `requesthandler/ai_summaries.go:110` DELETE `ai/summaries/<summary-id>`.
- ai-manager: `listenhandler/v1_summaries.go:151 processV1SummariesIDDelete` → `summaryHandler.Delete`(db.go:139).
- 프론트: `provider.js:174 export const Delete = (target) => fetchData('DELETE', target)` 존재.

즉 **옵션 A(삭제 후 재생성)는 백엔드/API 신규 개발 없이 프론트만으로 완결 가능**하다. 단 옵션 A는
아래 §2.4의 데이터 손실/레이스 리스크를 수반하므로, 최종 채택은 §3에서 리스크 가중 비교로 결정한다.

### 1.5 근본원인의 두 번째 축 — 언어 불일치 시 dedup 미적중 (R2 반영, 실코드 확인)
dedup 키는 (customer_id, reference_id, **language**)로 **언어별**이다(db.go:120-124). 따라서 이슈는
"동일 언어" 재요청에서만 발현한다. 그런데 요약 존재 분기 UI(AISummaryCard.js L154-188)에는 **언어
선택 UI가 없고**, Regenerate는 `summary.language`(L158에 표시되는 실제 요약 언어)가 아니라 컴포넌트의
`language` state(L30, defaultLanguage=표시 transcript 파생, L37-41에서 부모값 채택)를 POST한다(L110).

결과적으로 두 가지 다른 동작이 하나의 버튼에 섞여 있다:
- **language state == summary.language**: dedup 적중 → 기존 요약 그대로 반환(진짜 "재생성 안 됨" 버그).
- **language state != summary.language**: dedup 미적중 → "재생성"이 아니라 **다른 언어의 추가 요약**을
  신규 생성하고, loadSummary가 최신(L80-82)을 표시. 사용자는 "재생성됐다"고 인지하나 실제로는 언어가
  바뀐 별도 요약이다.

따라서 근본원인은 "dedup이 재생성을 막는다"(축 1)와 "Regenerate가 어떤 언어로 무엇을 해야 하는지
정의되지 않았다"(축 2)의 복합이다. **design은 Regenerate의 의미론을 먼저 정의해야 한다**: (a) 동일
언어 재생성인가(summary.language 고정), (b) 사용자가 언어를 다시 고를 수 있게 하는가. 이 결정 없이
dedup 우회만 구현하면 축 2가 미해결로 남는다.

## 2. 진행 타당성 분석

### 2.1 지금 진행이 타당한가
- **우선순위**: 사용자 가시성 있는 버그(버튼이 광고한 동작을 안 함). 다만 신규 데이터는 정상이므로
  P1이 아닌 P2 수준. 대표 지시로 착수 결정됨.
- **의존성**: VOIP-1532(언어 강제)가 이미 main 병합됨. 이 이슈가 해결되어야 VOIP-1532 로직이
  기존 캐시 리소스에도 도달한다(상호 보완).

### 2.2 리스크 재평가 (R2 반영 — 낮음으로 단정하지 않음)
- **데이터 영구 손실 (옵션 A 고유, 치명)**: DELETE 성공 후 POST 실패(recording 요약은
  startReferenceTypeRecording→contentGet(start.go:283)에서 **동기 LLM 호출**이라 실패 가능성 실재)
  시, 기존 요약은 이미 소프트삭제(deleted=true)됐고 신규 요약도 없는 **비복구 상태**로 남는다.
  square-admin no-refresh 규약(폴링/자동갱신 없음, AISummaryCard L22-24)상 자동 복구도 없어 사용자가
  수동 재시도해야 하며, 재시도 전까지 요약이 사라진 것으로 보인다. "짧은 공백"이 아니라 실질적
  데이터 손실 창이다.
- **동시성 레이스 (옵션 A 고유)**: DELETE+POST는 2단계 비원자 조작. 교차 관리자 세션 동시 Regenerate,
  또는 자동 트리거(on_end_flow, VOIP-1422 이중 conference_deleted, RPC 재시도)와 경합 시 삭제/생성이
  교차하며 중복 요약 또는 고아 레코드 가능. 프론트 isSubmitting(L179)은 단일 세션 더블클릭만 막고
  교차 세션/자동 경로는 못 막는다.
- **DELETE 이벤트 churn**: `Delete`(db.go:148)가 `EventTypeDeleted` 웹훅을 발행 → 매 Regenerate마다
  외부 구독자에게 delete+create 이벤트가 발생. 옵션 A는 이를 사용자 액션마다 유발, 옵션 B(서버 내부
  교체)는 이벤트 설계를 서버가 통제 가능.
- **dedup은 정상 보호 기능**: dedup(start.go:56-59)은 자동 트리거의 중복 요약/중복 transcribe start를
  막는 정상 기능이다. 해결책은 이를 전역 제거하지 말고 **명시적 사용자 Regenerate 액션만 우회/재생성**
  하고 자동 경로의 dedup은 유지하는 불변식을 지켜야 한다.
- **멱등성 참고**: SummaryDelete는 unconditional soft-delete(summary.go 기준 tm_delete set)라 비존재/
  이미삭제 id에 대략 멱등(0-row no-op)이나, POST(생성)는 비멱등이다.

### 2.3 해결 방향 후보 (design에서 리스크 가중 비교로 확정)
- **옵션 A (프론트만, 삭제 후 재생성)**:
  - 장점: 백엔드 무변경, 기존 API 재사용, 최소 범위.
  - 단점: §2.2의 **데이터 영구 손실 창**(DELETE 후 POST 실패)과 **비원자 레이스**를 구조적으로 안음.
    이를 완화하려면 "POST 성공 후 DELETE" 순서로 바꾸거나(그러면 dedup 때문에 POST가 기존 것을
    반환해 성립 불가 — 즉 옵션 A는 순서를 뒤집을 수 없음), 실패 시 롤백 불가를 UX 경고로만 처리.
  - 대표 원칙 상충: 삭제는 append가 아니며 파괴적. "비파괴·append 선호, 삭제는 예외" 원칙과 정면
    상충하므로 채택 시 예외 근거가 필요하다.
- **옵션 B (백엔드, create-then-replace)**: POST에 regenerate 플래그 → 서버가 **신규 레코드 생성 후
  구 레코드 삭제/교체**를 트랜잭션적으로 수행.
  - 장점: 단일 호출·원자적. 생성 성공 후에만 교체하므로 A의 비복구 창 제거.
  - 단점: API 계약 변경(OpenAPI/gen/RST/service_agents 동기화), 백엔드+프론트 양쪽 PR, 범위 큼.
    신규 레코드 방식이면 summary.id가 바뀌고(id churn), 구 레코드 처리에 따라 delete 이벤트 수반 가능.
- **옵션 C (백엔드, 동일 레코드 in-place 갱신) — R4 발견, 실코드 확인**: 기존 레코드 id를 유지한 채
  content만 원자적으로 덮어쓴다. 백엔드에 in-place 갱신 원시연산이 **이미 존재**한다:
  `UpdateStatusDone(id, content)`(db.go:170) → `SummaryUpdateStatusDoneIfNotDone`(dbhandler/summary.go:244)
  → **EventTypeUpdated 웹훅**(db.go 기준, 삭제 이벤트 아님).
  - 장점: (1) DELETE가 없어 §2.2 데이터 영구 손실 창이 **구조적으로 부재**(생성 성공 후에만 덮어씀),
    (2) EventTypeDeleted churn 없음(Updated 단일 이벤트), (3) **summary.id 안정성 유지**, (4) dedup을
    건드리지 않음(GetBy...가 기존 레코드를 찾고 그 id에 재생성), (5) 대표 "비파괴·append 선호" 원칙에
    가장 부합(삭제 없음).
  - 제약(실코드): `SummaryUpdateStatusDoneIfNotDone`은 `Where(NotEq{status: done})` 가드가 있어
    **이미 done인 요약은 갱신 못 함**(rowsAffected=0 → ErrSummaryAlreadyDone). Regenerate 대상은 이미
    done이므로, 이 함수를 그대로 쓸 수 없다. **done→done 재갱신을 허용하는 명시 재생성용 갱신 경로
    (가드 완화 또는 신규 메서드)가 필요**하다. 이 완화가 자동 경로(VOIP-1422 이중 conference_deleted 등)의
    IfNotDone 보호를 깨지 않도록, 완화는 명시 Regenerate 경로에만 적용해야 한다(자동 done 경로는 기존
    IfNotDone 유지).
  - 흐름: Start()가 dedup에서 기존 레코드를 찾으면 → 그 id로 contentGet 재실행(VOIP-1532 언어 강제
    포함) → 성공 시 완화된 갱신으로 content 덮어쓰기. 실패 시 기존 content 유지(비복구 창 없음).
  - 단점: 백엔드+프론트 PR(옵션 B와 동일 범위 수준), API에 regenerate 의미 노출 필요.

**후보 간 우열 관찰(참고, design에서 확정)**: 옵션 C가 데이터 손실·이벤트 churn·id 안정성·대표 원칙
부합에서 A/B보다 우월해 보인다. 단 "done 가드 완화 범위"와 "언어 의미론(축 2)"이 함께 설계되어야
하므로 최종 확정은 design 단계에서 한다.

### 2.4 스코프 경계
- 대량 backfill(과거 전체 요약 재생성)은 **비goal**. 사용자가 필요 시 개별 Regenerate로 갱신.
  (대표 확인: "새로운 데이터들은 문제 없으면 괜찮아".)

## 3. 분석 요약 (리뷰 대상)
- 이슈 유효성: **유효**. 근본원인은 복합이다 — (축 1) 프론트가 삭제 없이 POST만 → dedup(start.go:56-59)이
  기존 요약 반환 → 재생성/언어 강제 미실행; (축 2) Regenerate가 summary.language가 아닌 language state를
  보내 언어 불일치 시 dedup 미적중으로 다른 언어 요약을 추가 생성. 근거: AISummaryCard.js(삭제 없음,
  L110/L154-188 언어 UI 없음), start.go:54-59, db.go:114-124.
- 진행 타당성: **타당**. VOIP-1532와 상호 보완, 대표 착수 지시. 단 리스크는 "낮음"이 아니라 **옵션
  의존적**(옵션 A는 데이터 손실/레이스 창 있음).
- 옵션: **미확정 (3-way)**. 옵션 A(프론트 삭제+재생성), 옵션 B(백엔드 create-then-replace), 옵션 C
  (백엔드 in-place 갱신, UpdateStatusDone 계열 재사용+done 가드 완화). §2.2의 데이터 영구 손실·레이스·
  이벤트 churn 리스크와 대표 "비파괴" 원칙을 종합하면 **옵션 C가 유력**해 보이나(삭제 없음, id 안정,
  dedup 무손상), done 가드 완화 범위와 언어 의미론(축 2)을 함께 설계해야 하므로 최종 선택은 design에서
  리스크 가중 비교로 확정한다. design은 먼저 Regenerate 의미론(동일 언어 고정 vs 언어 재선택)을
  정의해야 한다(축 2).

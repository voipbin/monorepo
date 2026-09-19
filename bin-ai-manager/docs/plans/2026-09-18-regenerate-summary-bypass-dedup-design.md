# 설계: AI Summary Regenerate가 dedup을 우회해 실제 재생성하도록 수정 (VOIP-1535)

작성일: 2026-09-18
작성자: Hermes (CPO)
브랜치: VOIP-1535-regenerate-summary-bypass-dedup
상태: Draft (설계 리뷰 대기)
선행: 이슈 분석 문서 `2026-09-18-regenerate-summary-bypass-dedup-analysis.md` (리뷰 6회, R5/R6 2연속 APPROVED)

## 결정 (대표 확정, 2026-09-18)
- **구현 방식**: 옵션 C 계열(백엔드 in-place 갱신). 삭제 없음, summary.id 유지, 데이터 손실 창 없음.
- **언어 동작**: **언어 재선택 지원**. 요약 존재 상태에서도 언어를 바꿔 재생성 가능.
- **교체 규칙**: **recording당 요약 1개 유지**. 다른 언어로 재생성하면 기존 언어 요약을 새 언어 요약으로
  대체(공존 아님).

## 설계 피벗 (설계 리뷰 R1/R2 반영, 2026-09-18)
- **target을 클라이언트가 id로 지정** (R2-f, R1-4 반영, 대표 원칙 "재조회 회피=호출부 보유값 재사용,
  DB 재조회 거부"에 부합): 백엔드가 `GetByReferenceID`로 교체 대상을 재조회하지 않는다. 프론트가 이미
  보유한 `summary.id`(loadSummary가 유지)를 regenerate 요청에 실어 보내고, 백엔드는 그 id로 대상을
  로드해 in-place 갱신한다. 이로써 R1/R2가 지적한 최대 blocker(GetByReferenceID의 newest-1건 반환 →
  언어 공존 레거시에서 교체 대상 모호·orphan 잔존)가 원천 제거된다.
- **레거시 orphan 명시적 수용**: 과거 dedup은 (customer,reference,language)로 언어별 복수 요약을
  허용했으므로 한 reference에 언어별 복수 행이 이미 존재할 수 있다. 대량 backfill은 non-goal(대표 확인)
  이므로, regenerate는 클라이언트가 지정한 id만 교체하고 다른 언어의 레거시 행은 건드리지 않는다(잔존
  허용). 프론트 카드는 항상 1건(loadSummary가 최신 1건 표시)만 다루므로 사용자 경험상 "1개"는 유지된다.
- **scope: recording만 regenerate-then-update 성립**: call/conference는 Start 내에서
  contentGet를 동기 호출하지 않고 비동기(Event*→ContentProcess→UpdateStatusDone)로 content를 채우므로
  "생성 성공 후 갱신" 모델이 성립하지 않는다. transcribe는 content 산출 경로가 recording과 달라
  (getRecordingTranscripts가 아닌 별 경로) 본 PR에서 제외. 본 건은 **recording 전용**으로 설계한다.

## 1. 문제 정의

admin 콘솔 AI Summary Regenerate가 실제로 재생성되지 않는다. 근본원인 2축(분석 문서 §1):
- **축 1**: 프론트가 기존 요약을 삭제하지 않고 POST만 재전송 → 백엔드 `Start()`가 dedup(start.go:56-59,
  `GetByCustomerIDAndReferenceIDAndLanguage`)에서 기존 요약을 찾으면 그대로 반환하고 종료 →
  `contentGet`(VOIP-1532 언어 강제/검증 로직 포함)에 도달하지 못함.
- **축 2**: dedup 키가 (customer_id, reference_id, **language**)라 언어별인데, Regenerate가
  `summary.language`가 아닌 컴포넌트 `language` state를 POST(AISummaryCard.js L110). 언어가 다르면
  dedup 미적중으로 "재생성"이 아니라 다른 언어 요약이 추가 생성됨.

## 2. 목표 (numbered, testable)

1. 명시적 Regenerate 요청은 기존 요약을 **실제로 재생성**한다(VOIP-1532 언어 강제 로직 실행).
2. 재생성은 **삭제 없이 동일 레코드 id를 유지**하며 content(및 언어 변경 시 language)를 갱신한다.
3. 재생성 **실패 시 기존 요약 content가 보존**된다(데이터 손실 창 없음).
4. 사용자는 재생성 시 **언어를 재선택**할 수 있고, 다른 언어 선택 시 기존 요약이 새 언어로 **교체**된다
   (recording당 요약 1개 유지).
5. **자동 경로(비-Regenerate)의 dedup 및 IfNotDone 보호는 그대로 유지**된다(VOIP-1422 회귀 없음).

## 3. Non-goals

- 대량 backfill(과거 전체 요약 일괄 재생성). 사용자가 개별 Regenerate로 갱신(분석 §2.4, 대표 확인).
- 언어별 요약 공존(여러 요약 유지). 교체 방식으로 recording당 1개 유지.
- call/conference/transcribe 등 recording 외 reference type의 Regenerate UX. 본 건은 recording 상세
  카드 범위. (백엔드 재생성 경로는 reference type 무관하게 동작하나 UI는 recording 카드만 수정.)

## 4. 아키텍처: 재생성 경로 설계

### 4.1 핵심 원시연산 (실코드 확인)
- `dbhandler.SummaryUpdate(ctx, id, fields map[Field]any)`(summary.go:202): **임의 필드 갱신 가능**.
  `FieldLanguage`/`FieldContent`/`FieldStatus` 모두 갱신 가능(field.go:17-19). 캐시도 갱신
  (summaryUpdateToCache). WHERE는 id만 → status 가드 없음.
- `dbhandler.SummaryUpdateStatusDoneIfNotDone`(summary.go:244): `NotEq{status:done}` 가드 있어
  이미 done인 레코드는 갱신 못 함(VOIP-1422 이중 conference_deleted 방어). **Regenerate엔 부적합**.
- 비즈니스 레이어 `summaryHandler`에는 범용 Update가 아직 노출 안 됨(db.go: Create/Get/Delete/List/
  GetByCustomerIDAndReferenceIDAndLanguage/UpdateStatusDone만).

### 4.2 재생성 흐름 (id-targeting, regenerate-then-update, 데이터 손실 창 없음)
클라이언트가 재생성 대상 summary id를 지정하는 **전용 엔드포인트**를 신설한다:
`POST /v1/summaries/<summary-id>/regenerate` (body: `{ "language": "<bcp47>" }`).
이 방식은 (a) 교체 대상이 id로 명확(GetByReferenceID의 newest-추정·orphan 모호성 제거), (b) 기존
POST /summaries(생성)의 dedup 동작을 전혀 건드리지 않음(자동 경로 무손상), (c) REST상 "특정 요약
재생성"이 자연스러움. 신설 핸들러 의사코드:

```
Regenerate(ctx, summaryID uuid.UUID, language string):    // ai-manager summaryHandler
    // 1) 대상 로드 (클라이언트가 id 지정 → 재조회/추정 없음)
    existing, err := Get(ctx, summaryID)          // db.go:70 기존 Get 재사용
    if err != nil { return nil, err }             // 없는 id → 404

    // 2) 타입 검증 (scope 한정). 소유권 검증은 ai-manager가 아니라 servicehandler에서 수행한다
    //    (§4.4 참조). ai-manager는 내부 RPC라 customerID를 받지 않으며, api-manager auth 게이트를
    //    신뢰한다. 실코드 기존 패턴(AISummaryDelete)과 동일한 단일 권위(servicehandler) 원칙.
    if existing.ReferenceType != summary.ReferenceTypeRecording {
        return nil, errors.Errorf("regenerate supported only for recording")  // scope 한정
    }

    // 3) 언어 확정 (빈 값이면 기존 요약 언어 유지 → 축2 해소)
    lang := normalizeOutputLanguage(language)
    if language == "" { lang = existing.Language }   // 재선택 안 하면 기존 언어 유지

    // 4) content를 "먼저" 재산출 (VOIP-1532 언어 강제/transcript 재사용 포함).
    //    기존 헬퍼 조합(별도 추출 불필요): getRecordingTranscripts → contentGet.
    transcripts, err := h.getRecordingTranscripts(ctx, uuid.Nil, existing.ReferenceID)
    if err != nil { return nil, err }             // 실패 → 기존 레코드 무손상
    newContent, err := h.contentGet(ctx, uuid.Nil, summary.ReferenceTypeRecording, transcripts, lang)
    if err != nil { return nil, err }             // 실패 → 기존 레코드 무손상

    // 5) 성공 후에만 동일 레코드 in-place 갱신(language+content 동시 → 언어 교체도 같은 id).
    return h.UpdateContentLanguage(ctx, existing.ID, newContent, lang)   // §4.3
```

**소유권 검증은 servicehandler 단일 게이트**(실코드 AISummaryDelete 패턴, aisummary.go:197-209):
```
AISummaryRegenerate(ctx, a *auth.AuthIdentity, id uuid.UUID, language string):  // api-manager servicehandler
    if a.IsDirect() { return ErrDirectAccessNotSupported }
    c, err := h.aisummaryGet(ctx, id)                    // 대상 사전 조회(RPC AIV1SummaryGet)
    if err != nil { return nil, err }
    if !h.hasPermission(ctx, a, c.CustomerID,            // 소유권 게이트(단일 권위)
        amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
        return nil, ErrPermissionDenied
    }
    tmp, err := h.reqHandler.AIV1SummaryRegenerate(ctx, id, language)   // RPC
    ...
```
ai-manager Regenerate는 customerID를 받지 않으므로 소유권을 검사하지 않는다(검사 불가). 교차 커스터머
방어는 servicehandler의 fetch-then-permission이 유일 방어선이며 이를 확정 문구로 못박는다(R3/R4 반영).

핵심:
- **생성 성공 후에만 갱신** → 실패 시 기존 content 보존(목표 3). 삭제 없음(목표 2).
- **동일 레코드 id 유지** + language 함께 갱신 → 언어 교체도 같은 레코드에서 처리(목표 4).
- **activeflowID=uuid.Nil 전달**: regenerate는 flow 변수 컨텍스트가 없다(수동 액션). contentGet은
  activeflowID==Nil이면 변수 조회를 건너뜀(content.go 기존 분기). VOIP-1532 언어 강제는 RequestContent.
  OutputLanguage(=lang) 주입으로 동작하므로 activeflow 없이도 언어 강제됨.
- **dedup 무관**: 이 경로는 GetByCustomerIDAndReferenceIDAndLanguage(dedup)를 호출하지 않는다. 기존
  POST /summaries(생성)의 dedup 조기 반환은 **완전히 미변경** → 자동 경로 VOIP-1422 보호 유지(목표 5).
- **레거시 orphan**: 클라이언트가 지정한 id만 갱신. 다른 언어 레거시 행은 잔존 허용(설계 피벗 참조).
- **동시성(last-writer-wins)**: SummaryUpdate는 id-only WHERE라 낙관적 잠금이 없다. 동일 id 동시
  regenerate 시 마지막 쓰기가 이긴다. 수동 관리자 액션이고 프론트 isSubmitting으로 단일 세션은 막히며,
  교차 세션 경합은 드물고 결과가 "둘 중 하나의 최신 재생성"으로 수렴하므로 last-writer-wins를 수용한다
  (별도 잠금 미도입 — 오버엔지니어링 지양).

### 4.3 신설: `summaryHandler.UpdateContentLanguage`
비즈니스 레이어에 명시 재생성용 갱신 메서드를 신설한다(기존 `UpdateStatusDone`의 IfNotDone 가드를
쓰지 않고, 자동 경로는 건드리지 않음).

```go
func (h *summaryHandler) UpdateContentLanguage(ctx, id uuid.UUID, content, language string) (*summary.Summary, error) {
    fields := map[summary.Field]any{
        summary.FieldContent:  content,
        summary.FieldLanguage: language,
        summary.FieldStatus:   summary.StatusDone,   // 재생성 결과는 done
    }
    if err := h.db.SummaryUpdate(ctx, id, fields); err != nil {  // 기존 dbhandler 재사용, status 가드 없음
        return nil, errors.Wrapf(err, "could not update the summary")
    }
    res, err := h.db.SummaryGet(ctx, id)
    if err != nil { return nil, errors.Wrapf(err, "could not get updated summary") }
    h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeUpdated, res)  // Updated(삭제 아님)
    return res, nil
}
```
- 기존 `dbhandler.SummaryUpdate`(범용, status 가드 없음, 캐시 갱신 포함)를 재사용 → done 레코드도 갱신.
- `EventTypeUpdated` 발행(삭제 이벤트 churn 없음).
- 자동 경로가 쓰는 `UpdateStatusDone`/`SummaryUpdateStatusDoneIfNotDone`은 **미변경** → VOIP-1422 보호 유지.
- 메트릭: `promSummaryStartTotal`/`promSummaryDoneTotal` 모두 증가시키지 않음. regenerate는 신규 생성/
  최초 done이 아니라 기존 요약 갱신이므로 start/done 카운트를 중복 계수하지 않는다(관측성 일관, R1/R2
  관측성 지적 반영). 필요 시 별도 `promSummaryRegenerateTotal`을 둘 수 있으나 본 PR scope 밖(비goal).

### 4.4 API 계약 (신설 엔드포인트, regenerate)
- **엔드포인트**: `POST /aisummaries/{id}/regenerate` (신설). body: `{ "language": "<bcp47, optional>" }`.
  기존 POST /aisummaries(생성)와 DELETE는 미변경.
- **OpenAPI**: `paths/aisummaries/id_regenerate.yaml`(또는 유사) POST path 파일 신설 **+ openapi.yaml 루트에
  `/aisummaries/{id}/regenerate:` $ref 엔트리 추가**(루트는 자동 수집이 아니라 개별 $ref 등록 방식,
  L8502-8505 참조). `go generate`로 gen 반영.
- **api-manager**: 신설 핸들러 `PostAisummariesIdRegenerate`(gen 인터페이스) → servicehandler에
  `AISummaryRegenerate(ctx, auth, id, language)` 신설 → RPC.
- **RPC client**: `requesthandler/ai_summaries.go`에 `AIV1SummaryRegenerate(ctx, summaryID, language)` 신설
  → ai-manager `POST /v1/summaries/<id>/regenerate` 호출.
- **ai-manager listenhandler**: `regV1SummariesIDRegenerate` 라우팅 + `processV1SummariesIDRegeneratePost`
  신설 → `summaryHandler.Regenerate(ctx, id, language)` 호출. (customerID는 대상 레코드에서 읽되, api-manager
  auth가 이미 customer 스코프를 강제하므로 소유권 교차검증은 servicehandler 레이어에서 수행.)
- **Start 시그니처**: **변경 없음**(regenerate는 별도 엔드포인트/핸들러라 Start에 파라미터 추가 불필요).
  → R1-4의 Start 시그니처 파급(service.go ServiceStart 등 전 호출부 갱신) 문제도 원천 회피.
- **RST**: aisummaries 문서에 regenerate 엔드포인트 설명 추가(MANDATORY).

### 4.5 프론트 (square-admin, 별도 PR — 레포별 1 PR)
- `AISummaryCard.js` 요약 존재 분기(L154-188)에 **언어 드롭다운 추가**(무요약 분기 L196의 Select 재사용).
  **요약 존재 분기 전용 별도 state(예: `regenLanguage`)를 두고 `summary.language`로 시드**한다. 기존
  `language` state(L30, defaultLanguage=transcript 파생, L37-41 useEffect가 languageTouched 전까지 계속
  덮어씀)에 드롭다운을 바인딩하면 R1/R2가 지적한 "기본값=transcript 파생" 불일치가 재발하므로, regenLanguage는
  기존 defaultLanguage useEffect 영향 밖에 두고 summary 로드 시 summary.language로 초기화한다(축2 해소).
- Regenerate는 `POST aisummaries/{summary.id}/regenerate` + `{ language: regenLanguage }` 호출(생성 POST 아님).
- ConfirmDialog 문구에 "다른 언어 선택 시 기존 요약이 새 언어로 대체됩니다" 취지 안내.
- no-refresh 규약: POST 완료 응답 기반 프로그램 자동 1회 재로드(loadSummary) 유지(폴링 아님).

## 5. 영향 파일

### 5.1 백엔드 (monorepo, PR 1)
| 파일 | 변경 |
|------|------|
| bin-ai-manager/pkg/summaryhandler/main.go | Regenerate + UpdateContentLanguage 인터페이스 추가 (Start 시그니처 미변경) |
| bin-ai-manager/pkg/summaryhandler/start.go (또는 신규 regenerate.go) | Regenerate 핸들러 신설(id-targeting, regenerate-then-update) |
| bin-ai-manager/pkg/summaryhandler/db.go | UpdateContentLanguage 신설(SummaryUpdate 재사용, EventTypeUpdated) |
| bin-ai-manager/pkg/dbhandler/summary.go | SummaryUpdate의 "currently unreferenced outside its own test" 주석 갱신(이제 UpdateContentLanguage가 프로덕션 참조) |
| bin-ai-manager/pkg/summaryhandler/mock_main.go | go generate 반영 |
| bin-ai-manager/pkg/listenhandler/main.go | regV1SummariesIDRegenerate 라우팅 추가 |
| bin-ai-manager/pkg/listenhandler/v1_summaries.go | processV1SummariesIDRegeneratePost 신설 (기존 POST/DELETE 미변경) |
| bin-ai-manager/pkg/listenhandler/models/request/summaries.go | regenerate body DTO(language) 추가 |
| bin-ai-manager/pkg/summaryhandler/*_test.go, listenhandler/*_test.go | Regenerate 동일언어/교체/실패보존/소유권/타입 테스트 |
| bin-openapi-manager/openapi/paths/aisummaries/ | {id}/regenerate POST path 신설 |
| bin-openapi-manager (gen) | go generate 반영 |
| bin-api-manager/server/aisummaries.go + servicehandler/aisummary.go | PostAisummariesIdRegenerate + AISummaryRegenerate 신설(소유권 검증) |
| bin-api-manager/gens (openapi_server) | go generate 반영 |
| bin-api-manager (redoc gen) | 재생성 |
| bin-common-handler/pkg/requesthandler/ai_summaries.go | AIV1SummaryRegenerate 신설 |
| bin-api-manager/docsdev/source (RST) | aisummaries regenerate 엔드포인트 설명 |

주: **Start/ServiceStart 시그니처는 미변경**(regenerate가 별도 엔드포인트라 파급 없음). R1-4의 service.go
호출부 갱신 문제는 이 설계에서 발생하지 않는다.

### 5.2 프론트 (monorepo-javascript, PR 2)
| 파일 | 변경 |
|------|------|
| square-admin/src/views/recordings/AISummaryCard.js | 요약 존재 분기 언어 드롭다운(summary.language 시드), regenerate 엔드포인트 호출, 안내 문구 |
| square-admin/src/views/recordings/__tests__/AISummaryCard.test.js | regenerate 엔드포인트/language 전달 검증 |

## 6. 트레이드오프 / 리스크

- **범위**: 백엔드 다수 계층(ai-manager/openapi/api-manager/common-handler/RST) + 프론트. 레포별 1 PR
  (monorepo 1개, monorepo-javascript 1개). 옵션 C는 순수 프론트(옵션 A)보다 크나, 데이터 손실·이벤트
  churn·id churn을 구조적으로 제거하는 대가로 정당.
- **id-targeting 채택 근거**: 클라이언트가 이미 보유한 summary.id를 대상으로 지정 → GetByReferenceID의
  newest-추정·레거시 orphan 모호성을 원천 제거. 대표 원칙("재조회 회피=호출부 보유값 재사용, DB 재조회
  거부")과 일치. Start 시그니처 파급도 회피.
- **레거시 orphan**: 한 reference에 언어별 복수 요약이 이미 있어도 regenerate는 지정 id만 갱신. 다른
  언어 행은 잔존(대량 backfill non-goal). 프론트는 loadSummary가 최신 1건만 표시하므로 UX상 "1개" 유지.
- **동시성(last-writer-wins)**: SummaryUpdate는 id-only WHERE, 낙관적 잠금 없음. 동일 id 동시 regenerate
  시 마지막 쓰기가 이김. 수동 관리자 액션이고 프론트 isSubmitting으로 단일 세션 방지. 교차 세션 경합은
  드물고 결과가 유효한 재생성으로 수렴 → 수용(잠금 미도입, 오버엔지니어링 지양).
- **소유권 검증**: servicehandler AISummaryRegenerate가 auth customer로 대상 소유권 검증(교차 커스터머
  방어). ai-manager는 내부 RPC라 api-manager auth를 신뢰.
- **메트릭**: UpdateContentLanguage는 start/done 카운트를 증가시키지 않음(재생성은 신규 생성/최초 done이
  아님). 중복 계수 방지(관측성 일관).
- **scope 한정**: recording만 regenerate 지원(call/conference는 Start 내 동기 content 산출 없음). 다른
  reference type 요청은 명시적으로 거부(에러).
- **빈 콘텐츠 가드(데이터 손실 창 방지)**: contentGet은 hard error 없이 `("", nil)`을 반환할 수 있다(빈 LLM
  choices, 또는 비영어 타깃의 lastNonEmpty 빈값). Regenerate는 덮어쓸 기존 요약이 있으므로 err만 검사하면
  빈 결과가 기존 정상 content를 덮어쓰는 회귀가 생긴다. `newContent == ""`이면 갱신을 건너뛰고 기존 레코드를
  보존한 채 에러를 반환한다(신규 생성 Start 경로에는 없는, Regenerate 전용 가드).

## 7. 검증 계획

- 백엔드: `go build ./...`, `go test ./pkg/summaryhandler/... ./pkg/listenhandler/...`, `go vet`,
  `golangci-lint run`. 각 서비스(ai-manager/openapi-manager/api-manager/common-handler) verification
  워크플로우(go mod tidy/vendor/generate/test/lint).
- 테스트 케이스(summaryhandler Regenerate):
  1. 동일 언어(language=""): 기존 요약 언어 유지, content 재산출, 동일 id in-place 갱신, EventTypeUpdated.
  2. 다른 언어: language+content 갱신(교체), 동일 id 유지.
  3. content 재산출 실패(getRecordingTranscripts/contentGet 에러): UpdateContentLanguage 미호출, 기존
     레코드 무손상, 에러 반환.
  4. 없는 id: 404/에러.
  5. 소유권 불일치(다른 customer): 거부.
  6. reference_type != recording: 거부.
  7. UpdateContentLanguage: done 레코드도 갱신됨(IfNotDone 가드 없는 SummaryUpdate 사용 확인),
     EventTypeUpdated 발행.
  8. 자동 경로(UpdateStatusDone/IfNotDone) 미변경 회귀: VOIP-1422 기존 테스트 통과.
  9. 빈 콘텐츠 성공(contentGet가 `("", nil)` 반환): UpdateContentLanguage 미호출, 기존 레코드 무손상,
     에러 반환(빈 콘텐츠 가드).
- 프론트: `CI=true npx react-scripts test` — 언어 드롭다운(summary.language 시드), regenerate 엔드포인트
  호출·language 전달 검증.
- API: OpenAPI 스키마 검증, redoc 재생성. api-validator는 비용 유발(생성) 테스트 금지 규칙 준수(regenerate도
  LLM 비용 유발이므로 api-validator 테스트 미작성).

## 8. 리뷰 이력
- R1 (deleg_f648e3df, task0): CHANGES_REQUESTED. GetByReferenceID newest-추정/orphan(blocker),
  call/conference contentGet 비동기라 일반화 부정확, content-only 헬퍼 미명세, service.go 누락,
  메서드 명칭 모호, 소유권 검증, start_total 계측.
- R2 (deleg_f648e3df, task1): CHANGES_REQUESTED. 교체대상 모호성, 동시성 경합, 드롭다운 기본값 불일치,
  **클라이언트가 이미 target id 보유(id-targeting 대안, f)**, 폴백/원칙 정합은 양호.
- 정정(R1+R2 반영): **id-targeting 전용 엔드포인트로 피벗**(POST /aisummaries/{id}/regenerate). 이로써
  GetByReferenceID 재조회 제거(교체대상 모호성/orphan blocker 해소), Start 시그니처 미변경(service.go
  파급 회피), 기존 헬퍼(getRecordingTranscripts+contentGet) 조합으로 content-only 산출(별도 추출 불요),
  소유권/타입 검증 추가, 드롭다운 summary.language 시드 명시, 메트릭 start/done 미계수 명시, 동시성
  last-writer-wins 수용 명시, scope recording 한정 명시, 정확 메서드명(AISummaryRegenerate/
  AIV1SummaryRegenerate) 고정.
- (재리뷰 R3/R4 대기)
- R3 (deleg_c14732b3, task0): CHANGES_REQUESTED. 소유권 검증 계층 모순(ai-manager 의사코드 authCustomerID
  vs RPC 시그니처에 customerID 없음), transcribe scope 표기 모순, OpenAPI 루트 $ref 누락, servicehandler
  fetch-then-permission 절차 미상세. (id-targeting 흐름/activeflow Nil/자동경로 보호는 통과.)
- R4 (deleg_c14732b3, task1): CHANGES_REQUESTED. 소유권 검증 모순(동일), servicehandler fetch-then-permission
  명시, 프론트 드롭다운 별도 state(regenLanguage) 필요, SummaryUpdate "unreferenced" 주석 갱신.
- 정정(R3+R4 반영): 소유권 검증을 **servicehandler 단일 게이트**(AISummaryDelete 패턴 aisummary.go:197-209:
  aisummaryGet→hasPermission)로 통일, ai-manager 의사코드의 authCustomerID 검사 제거(타입 검증만). scope
  recording 전용으로 정정(transcribe 제외). OpenAPI 루트 openapi.yaml $ref 추가 명시. 프론트 regenLanguage
  별도 state(summary.language 시드, defaultLanguage useEffect 밖) 명시. §5.1에 SummaryUpdate 주석 갱신 추가.
- (재리뷰 R5/R6 대기)
- R5 (deleg_019ca383, task0): APPROVED. 소유권 단일 게이트(AISummaryDelete 패턴 일치), RPC 무-customerID,
  transcribe 제외 일관, OpenAPI 루트 $ref, regenLanguage state, SummaryUpdate 주석 모두 실코드 정합. 새 결함 없음.
- R6 (deleg_019ca383, task1): APPROVED. SummaryUpdate/IfNotDone 가드/AISummaryDelete 패턴/getRecordingTranscripts
  +contentGet/OpenAPI 루트/라우팅 $-anchored/프론트 축2/RST+OpenAPI MANDATORY/메트릭·잠금 defer 근거 모두
  확인. findings 없음.
- **설계 확정(R5/R6 2연속 APPROVED, 2026-09-18)**. 설계 리뷰 총 6회. 구현 착수.

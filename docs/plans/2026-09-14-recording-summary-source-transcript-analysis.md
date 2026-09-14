# Recording Detail: Show the Summary's Source Transcript — Analysis

Date: 2026-09-14
Author: CPO (Hermes) with pchero
Status: Issue analysis (review loop, rev.2)

## 1. 문제 (대표님 요구)

admin 레코딩 상세에서 AI Summary는 생성/표시되는데(스크린샷), 그 요약의 **근거가 된 트랜스크립트**는
Transcript 카드에 "No transcript yet"으로 비어 있다. 대표님 요구: "요약을 만들 때 쓴 트랜스크립트
내용도 이 화면에서 함께 보고 싶다." 소유권 변경은 원하지 않음(대표님 명시).

## 2. 이슈 유효성 재확인 (현재 코드 기준)

유효한 문제다. 실제 화면에서 재현되며, 아래 코드 사실로 뒷받침된다.

### 2.1 요약은 근거 transcript로의 "연결"을 저장하지 않는다 (핵심 근본원인)

`bin-ai-manager/models/summary/main.go`의 `Summary` 구조체가 저장하는 것은 `content`(최종 요약문)뿐이다.
근거 transcript 텍스트도, 그 transcribe의 id도 저장하지 않는다.

```
type Summary struct {
    ... ActiveflowID, OnEndFlowID,
    ReferenceType, ReferenceID,
    Status, Language, Content,   // <- Content(요약문)만. transcript 원문/transcribe_id 없음
    TMCreate, TMUpdate, TMDelete
}
```
`webhook.go`의 `WebhookMessage`(외부 노출)도 동일하게 `Content`까지만 노출한다.

**정정(리뷰 반영)**: 근거 transcript 원문은 "버려지지" 않는다. recording 요약 경로에서 transcribe-manager는
transcript를 DB에 **영구 저장**한다(`startRecording` → `transcriptHandler.Recording` → StatusDone). 다만
그 transcribe는 `IDAIManager`(시스템 계정) 소유로 남는다(§2.2). 즉 실제로 없는 것은 원문이 아니라
**summary → 근거 transcript로의 연결(transcribe_id / 원문 링크)** 이다. 이 "연결 부재"가 근본원인이다.

이 사실(원문이 IDAIManager 하위에 영구 잔존)은 오히려 §3의 대안 (b)/(c)를 강하게 뒷받침한다:
재-STT(재-transcription) 없이 **기존 데이터 재조회만으로** 요구를 충족할 수 있기 때문이다.

### 2.2 근거 transcribe는 IDAIManager 소유라 고객 경로로 못 찾는다 (부차)

요약이 내부적으로 만드는 transcribe는 `cmcustomer.IDAIManager`(시스템 계정) 소유로 생성된다.
소유권을 IDAIManager로 두는 분기는 3곳이다:
- `startReferenceTypeCall` (start.go:91)
- `startReferenceTypeConference` (start.go:155)
- `startReferenceTypeRecording` (start.go:257)
- (`startReferenceTypeTranscribe`는 transcribe를 새로 만들지 않고 기존 것을 조회만 하므로 제외)
- 비동기 재조회도 `content.go:150` `contentGetTranscripts`가 `FieldCustomerID: IDAIManager`로 필터.

백엔드 주석:
```
// note: here, we set the customer id as the ai manager id
// this is required because if we use the customer id, the created
// transcribe will be shown to the customer's transcribe list.
```

그래서 Transcript 카드의 `GET /transcribes?reference_type=recording&reference_id=`는
`customer_id = a.CustomerID`를 강제하므로(servicehandler `TranscribeList`, transcribe.go:86-100)
IDAIManager 소유 transcribe를 잡지 못한다. 단일 조회 `TranscribeGet`(transcribe.go:47)도
`hasPermission(a, tmp.CustomerID, ...)`로 소유 고객 매칭을 요구하므로 admin이라도 IDAIManager
리소스는 접근 불가(멀티테넌시 격리, 정상 설계).

**프레이밍**: §2.2의 소유권/격리는 "transcribe를 찾아가서 transcripts를 읽는다"는 특정 경로에서만
생기는 장애물이다. 대표님 요구(요약의 근거 텍스트를 화면에 표시)의 본질이 아니다. 대안 (a)/(b)는
IDAIManager transcribe를 조회 경로로 "찾아가지" 않고, 요약 생성 주체(ai-manager, IDAIManager 권한
보유)가 이미 확보한/재조회 가능한 transcript를 summary에 저장하거나 응답에 동봉하므로 소유권/격리를
전혀 건드리지 않는다. (반면 대안 (c)는 조회 경로에 admin-safe 구멍을 내므로 §2.2의 "정상 설계" 격리와
정면으로 긴장한다 — §3에서 별도 취급.)

### 2.3 요약 생성 흐름의 동기/비동기 비대칭 (대안 설계에 직접 영향)

reference_type별로 transcript 확보 시점과 저장 삽입 지점이 다르다. 이 비대칭을 대안이 반드시 다뤄야 한다.

| reference_type | 확보 시점 | Create 시 status | content 채워지는 곳 | 근거 transcript 저장 삽입 지점(대안 a) |
|---|---|---|---|---|
| recording | 동기 (start.go:277 TranscriptList) | StatusDone (start.go:288) | Create 시점 | Create 시점 (start.go) |
| transcribe | 동기 (start.go:209 TranscriptList) | StatusDone | Create 시점 | Create 시점 (start.go) |
| call | 비동기 | StatusProgressing, content="" | content.go ContentProcess → UpdateStatusDone | UpdateStatusDone 시점 (content.go) |
| conference | 비동기 | StatusProgressing, content="" | content.go ContentProcess → UpdateStatusDone | UpdateStatusDone 시점 (content.go) |

- 당면 요구는 **recording(동기)** 이라 즉시 블로킹은 아니나, 대안을 전 reference_type로 확장할 경우
  저장 삽입 지점이 두 곳(Create / UpdateStatusDone)으로 갈린다.
- `content.go`의 `ContentProcess`(비동기)는 **Call/Conference만** 처리하고 recording/transcribe는 default에서
  에러 반환(동기 처리). 따라서 대안 (b) 파생 조회를 채택하면 **recording용 재파생 분기를 신설**해야 한다.
- `contentGetTranscripts`의 조회 키가 reference_type별로 다르다: conference는 `reference_id`로
  `cf.ConfbridgeID`를 쓴다(content.go:106, summary.ReferenceID 아님). (b) 설계 시 이 간접참조 처리 필요.

### 2.4 멱등 재사용 경로 — 기존 summary는 근거 transcript를 확보하지 못한다 (핵심 리스크)

`Start()` 초입(start.go:29-33)에서 `GetByCustomerIDAndReferenceIDAndLanguage`로 **기존 summary가
발견되면 transcripts를 전혀 조회하지 않고 기존 row를 즉시 반환**한다.

```
tmp, err := h.GetByCustomerIDAndReferenceIDAndLanguage(ctx, customerID, referenceID, language)
if err == nil {
    return tmp, nil   // 기존 summary 재사용 — transcript 확보 안 함
}
```

함의:
- "요약 시점에 transcripts를 손에 쥐고 있다"는 신규 생성 경로에만 참이다. **재사용 경로에는 거짓**이다.
- 스크린샷의 요약처럼 **이미 존재하는 summary**는 (현행 흐름으로 생성돼 근거 연결이 없음) 대안별로
  다르게 처리해야 한다:
  - **대안 (a) 저장 방식**: 스키마/저장 로직 추가 이후 "신규 생성" summary만 근거를 갖는다. **기존
    summary는 백필(migration/backfill) 없이는 여전히 빈 화면**이 된다. → 백필 전략 또는 "신규 요약만
    표시" 한계를 설계에서 명시해야 함.
  - **대안 (b) 파생 조회 방식**: 재사용 경로여도 조회 시점에 IDAIManager 소유 transcribe/transcript를
    (§2.1의 영구 잔존 데이터로) 재발견해 실어 보내므로 **기존 summary도 커버 가능**. 단 §2.3의 recording
    재파생 분기 신설이 전제.

## 3. 진행 타당성 분석

- **우선순위**: 대표님이 실제 화면을 보고 직접 요청한 UX 개선. 가치 명확.
- **소유권 무변경 원칙 준수 가능성**: 가능하다. §2.1(원문 영구 잔존) + §2.2 프레이밍에 근거해, 요약이
  이미 확보한/재조회 가능한 transcript를 summary와 함께 남기거나 응답에 실으면 transcribe 소유권/목록/격리를
  건드리지 않는다.
- **billing 리스크**: 없음. `bin-billing-manager/models/billing/billing.go`의 `ReferenceType` 상수 집합
  (call, call_extension, sms, email, number, number_renew, credit_*, monthly_allowance, *_adjustment,
  speaking, recording, paddle_*)에 transcribe/stt 항목이 없다. subscribehandler에도 transcribe 과금
  구독 없음. 대안 (a)(b)(c) 모두 신규 STT를 만들지 않고 기존 데이터를 재사용하므로 과금 영향 없음.
  (주의: billing의 `ReferenceTypeRecording`/`ReferenceTypeSpeaking`은 각각 recording 생성/TTS 과금이며
  STT와 무관.)
- **대안 후보 (설계 단계에서 비교, 본 분석에서 확정하지 않음)**:
  - **(a) 저장**: 요약 생성 시 근거 transcript(원문 또는 transcribe_id)를 summary에 함께 저장. 조회 시
    그대로 표시. 파급: DB 스키마/저장 로직 변경(삽입 지점 recording=Create/call·conf=UpdateStatusDone,
    §2.3), **기존 summary 백필 필요**(§2.4). 소유권 무변경.
  - **(b) 파생 조회**: 저장 없이, summary 생성 응답 및 조회 시 근거 transcript를 재발견해 함께 반환.
    파급: recording 재파생 분기 신설 + conference confbridgeID 간접참조 처리(§2.3), 조회 비용. **기존
    summary도 커버**(§2.4). 소유권 무변경.
  - **(c) transcribe_id 저장 + admin-safe 조회 경로**: summary에 근거 transcribe_id만 저장하고 그
    transcripts를 admin-safe 경로로 읽음. §2.2의 "정상 설계" 격리에 구멍을 내므로 **격리 관점과 정면으로
    긴장**. 신중 검토 대상. 소유권은 안 바꾸되 조회 권한 경계를 변경.

## 4. 스코프 경계

- **본 이슈 범위**: "이미 만들어진(또는 새로 만드는) 요약의 근거 transcript를 레코딩 상세에 표시".
  우선 대상은 recording(동기 경로). call/conference 확장은 §2.3 비대칭 때문에 별도 판단.
- **별개 문제(SQUARE-69 이슈 B)**: §2.3의 멱등키 문제(recording.go:29 `GetByReferenceIDAndLanguage`가
  customer_id 미포함 → 고객 Transcribe 버튼이 IDAIManager 기존 transcribe를 재사용해 카드가 계속 빔)는
  본 이슈와 원인 계통은 겹치나 별도 트랙이다. 본 이슈를 대안 (a)/(b)로 풀면 "요약 근거 표시"는 해결되지만,
  "고객이 직접 누른 Transcribe가 헛도는" 문제는 별도로 남는다. 본 설계에서 함께 풀지, 분리할지는 설계 단계 결정.

## 5. 결론

- 이슈는 유효하고 진행 타당하다.
- 근본 원인은 "요약이 근거 transcript로의 **연결**을 저장하지 않음"(§2.1)이며, 원문 자체는 IDAIManager
  하위에 영구 잔존한다. IDAIManager 소유권(§2.2)은 대표님 요구의 본질이 아니라 특정 조회 경로의 부차적
  장애물이다.
- 설계는 §2.3(동기/비동기 저장 삽입 지점 비대칭)과 §2.4(멱등 재사용 경로 + 기존 summary 백필)를 반드시
  다뤄야 한다.
- 다음 단계: 소유권 무변경을 전제로 대안 (a)/(b)/(c)를 §2.3·§2.4 관점에서 비교하는 설계 문서를 작성하고
  설계 리뷰 루프 진행.

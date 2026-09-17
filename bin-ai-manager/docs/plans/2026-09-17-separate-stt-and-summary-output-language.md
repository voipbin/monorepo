# 설계: STT 언어와 AI Summary 출력 언어 분리 (VOIP-1532)

작성일: 2026-09-17
작성자: Hermes (CPO)
브랜치: VOIP-1532-separate-stt-and-summary-output-language
상태: Draft (설계 리뷰 대기)
결정(대표): STT는 기존 transcribe 재사용 우선. 스코프는 완전 해결(2축 분리) + 프론트 재노출.

## 1. 문제 (코드 확정)

`summaryHandler.Start`의 단일 `language` 파라미터가 세 가지를 동시에 결정한다:
1. STT 언어: `startReferenceTypeRecording`이 language를 `TranscribeV1TranscribeStart`에
   그대로 넘김 -> 영어 음성에 ko-KR을 주면 ko-KR STT가 영어를 인식 못해 transcript가 빔.
2. 요약 저장 language: `h.Create(... language ...)`로 summary 레코드에 기록.
3. transcribe dedup 키: `startRecording`의 `GetByReferenceIDAndLanguage`가 language까지
   일치해야 재사용 -> language가 다르면 admin이 만든 en-US transcribe를 못 쓰고 새로 생성.

한편 **요약 출력 언어는 이미 프롬프트가 분리 지원**한다(main.go:132 "Generate the summary
in the language specified in 'voipbin.ai_summary.language', regardless of the transcription's
language"). 그러나 `contentGet`이 만드는 RequestContent에는 language가 안 들어가고, 이
변수는 activeflow variables로만 채워진다. recording 경로는 activeflow가 없어(uuid.Nil) 이
변수가 비어 출력 언어 지시가 무력화된다.

실측(recording 882d1b70, 영어 음성): admin transcribe 65319f89(en-US) transcript 5개 정상,
summary가 만든 f9267345(ko-KR) transcript 1개(빈약), ko-KR 요약 5섹션 전부 None.

## 2. STT 자동감지 불가 (설계 제약)

Google STT `RecognitionConfig.LanguageCode`는 필수다(process.go:27, 단일 언어). 완전
자동감지는 현재 코드로 불가. 따라서 STT는 "어떤 언어로 인식할지"를 알아야 한다.
결론: summary가 STT 언어를 새로 정하지 말고, **recording에 이미 있는 transcribe를 재사용**
한다(그 transcribe의 언어=원음에 맞게 이미 정해진 것). 없을 때만 STT 언어를 정한다.

## 3. 목표 아키텍처 (2축 분리)

- **STT 축**: recording의 기존 transcribe(있으면)를 재사용. 없으면 STT 언어를 정해 1개
  생성. summary는 STT 언어를 좌우하지 않는다.
- **출력 축**: 사용자가 고른 출력 언어를 RequestContent에 실어 LLM이 그 언어로 요약(번역).
  activeflow variable 유무와 무관하게 동작.

두 축을 별도 파라미터로 표현한다:
- `outputLanguage`(요약 출력 언어): 사용자 선택. 없으면 기본값(en-US 또는 원음 언어).
- STT 언어: 파라미터로 받지 않는다. 기존 transcribe 언어를 따르거나(재사용), 없을 때만
  fallback STT 언어를 정한다(§4.2에서 결정).

## 4. 구현 설계 (phase별, 각 phase 독립 검증)

### Phase 1 (백엔드 bin-ai-manager): 출력 언어 주입 분리

RequestContent에 출력 언어를 실어 프롬프트가 확실히 그 언어로 요약하게 한다.

- `content.go` RequestContent에 `OutputLanguage string json:"output_language,omitempty"` 추가.
- `contentGet` 시그니처에 `outputLanguage string` 추가, RequestContent.OutputLanguage에 주입.
  호출부 4곳(content.go:65/112 call·conference, start.go:214 transcribe, start.go:282
  recording) 모두 해당 summary의 출력 언어를 전달.
- 프롬프트(main.go) Language 섹션을 `voipbin.ai_summary.language` 변수 의존에서
  RequestContent의 `output_language` 필드 우선으로 변경(변수는 fallback). recording처럼
  activeflow 없는 경로에서도 출력 언어가 확실히 적용됨.
- 이 phase만으로 call/conference/transcribe 경로는 출력 언어가 확실히 적용된다. **단
  recording 경로는 Phase 1 단독으로는 오히려 회귀**한다: §4.3에서 language 의미가 "출력
  언어"가 되면, recording 경로가 그 출력 언어(예: ko-KR)를 여전히 STT 언어로
  `TranscribeV1TranscribeStart`(start.go:262)에 넘겨 영어 음성을 ko-KR STT로 돌려 빈
  transcript가 된다. 따라서 **Phase 1과 Phase 2는 반드시 같은 PR에서 함께 배포**한다
  (§6 강제). "phase 독립 검증"은 테스트 단위 분리를 뜻하며, recording 경로의 배포 안전성은
  Phase 2까지 있어야 성립한다.

### Phase 1.1 (call/conference 경로 outputLanguage 출처)

call/conference는 contentGet를 동기 호출하지 않고 StatusProgressing으로 생성 후
ContentProcess -> contentProcessReferenceType*(content.go:65/112)에서 contentGet를 호출한다.
이 경로의 outputLanguage는 **summary 레코드에 저장된 sm.Language**에서 읽어 전달한다(summary가
language를 저장하므로 조회 가능). recording/transcribe 동기 경로는 Start의 language 인자를
직접 전달한다. 두 경로 모두 "그 summary의 출력 언어"라는 의미로 일관.

### Phase 2 (백엔드 bin-ai-manager): recording transcribe 재사용

`startReferenceTypeRecording`이 새 transcribe를 남발하지 않고, **원본(사용자/admin) transcribe를
재사용**하게 한다. summary가 과거에 만든 빈 transcribe를 재사용 후보에서 제외하는 것이 핵심.

- transcribe 시작 전, `TranscribeV1TranscribeList`로 recording의 기존 transcribe를 조회한다.
  - filters: `reference_id=recordingID`, `reference_type=recording`, `status=done`,
    `deleted=false`. **customer_id는 필터하지 않는다.** (중요: admin/사용자가 만든 transcribe는
    실제 customer_id를 갖고, summary가 만든 것만 IDAIManager다. 기존 헬퍼 contentGetTranscripts는
    customer_id=IDAIManager로 필터하므로 재사용에 그대로 쓰면 admin transcribe를 못 본다.
    Phase 2는 IDAIManager 필터를 쓰지 않는 별도 조회다.)
  - page_size는 넉넉히(예: 100) 받아 코드에서 후보를 고른다(1건 조회로 최신만 집으면 안 됨).
- **재사용 후보 선택 기준(회귀 방지 핵심, R1/R2 지적)**: "tm_create desc 최신 1개"는 금지.
  실측(§1)에서 admin en-US(먼저 생성, transcript 5개)보다 summary가 만든 빈 ko-KR(나중 생성,
  IDAIManager)이 더 최근이라 desc 최신은 빈 것을 고른다. 대신:
  1. `customer_id`가 IDAIManager가 **아닌** transcribe를 우선(원본 우선, summary가 만든 것
     배제). transcribe 모델의 판별 필드는 `customer_id`다(owner_id 필드는 없음). 배제는
     **customer_id 필터 없이 조회한 뒤 코드에서 IDAIManager를 제외**한다. (주의: List filter의
     NotEq는 string-kind 전용이라 uuid.UUID인 customer_id에 쓰면 바이트 표현 불일치로 조용히
     오작동한다 -- bin-common-handler/databasehandler/main.go:47-53 SCOPE WARNING. 따라서 DB단
     NotEq 배제는 금지, 코드 배제로 단일화한다.)
  2. 그 중 transcript가 실제로 존재하는(비어있지 않은) done transcribe.
  3. 후보가 여럿이면 tm_create desc 최신(원본 중 최신).
  - **단락(short-circuit) 규칙(N+1 완화)**: TranscribeList는 이미 tm_create DESC로 반환하므로
    (transcribe.go:212), 非IDAIManager 후보를 반환 순서(desc)대로 순회하며 첫 transcript>0에서
    즉시 채택하고 중단한다. 정상 케이스에서 TranscriptList 호출이 1회로 수렴.
  - IDAIManager가 아닌 후보가 없거나 전부 transcript가 비면, 재사용하지 않고 신규 생성(§4.2).
  - 즉 선택 순서: (원본 && transcript>0) 우선 -> 없으면 신규 생성. summary가 만든 IDAIManager
    transcribe는 재사용하지 않는다(오염원 배제).
- transcript 존재 검증: 후보 transcribe의 transcript를
  `TranscribeV1TranscriptList(transcribe_id=cand.ID)`로 조회해 개수>0을 확인한 뒤 그 transcript로
  요약. (기존 transcript 조회 로직 재사용, tr.ID만 후보로 교체.)
- 없으면 STT 언어를 정해(§4.2) 1개 생성 후 그 결과 사용.
- 결과: 영어 음성 recording에 ko-KR 요약을 눌러도 admin의 en-US transcript(원본, 非IDAIManager,
  transcript 5개)를 재사용하고 출력만 한국어로 번역 -> 빈 요약 회귀 없음.

### 4.2 기존 transcribe가 없을 때 STT fallback 언어

- STT fallback은 `outputLanguage`와 무관한 별도 값이다(출력 언어를 STT에 다시 쓰면 회귀).
- 이 이슈 스코프에서는 **en-US 고정 fallback**으로 시작(가장 단순, 오검출 리스크 최소).
  customer별 기본 STT 언어 설정은 별도 이슈로 분리(오버엔지니어링 방지).
- 즉 "재사용 후보 없음(원본 transcribe 없음) + 사용자가 ko-KR 출력 요청" -> en-US STT로
  인식 후 한국어 요약. 원음이 영어면 정상, 원음이 한국어면 en-US STT가 부정확할 수 있으나,
  이는 자동감지 불가의
  근본 한계이고 "기존 transcribe 재사용 우선" 원칙상 admin이 올바른 언어로 먼저 transcribe
  하면 해소된다. (fallback 언어 정교화는 후속.)

### Phase 3 (프론트 monorepo-javascript): 출력 언어 선택 재노출

VOIP-1531에서 임시 제거한 Regenerate 언어 선택을 "출력 언어"로 안전하게 재노출.

- AISummaryCard: Summarize 폼 + Regenerate 모두에 "출력 언어" 선택 제공(라벨을 STT가 아닌
  "Summary language"로 명확히). 선택값은 POST의 language(=출력 언어)로 전달.
- 백엔드가 이제 출력 언어를 STT와 분리하므로, 언어를 바꿔도 빈 요약이 안 남.
- defaultLanguage(표시된 transcript 언어 파생)는 그대로 기본 제안으로 유지.

### 4.3 API 계약 (language 파라미터 의미 변경)

- 기존 POST /aisummaries의 `language`는 "STT+출력 겸용"이었다. 이제 **"출력 언어"**로 의미가
  좁혀진다. STT 언어는 서버가 기존 transcribe 재사용/fallback으로 결정(클라이언트 미지정).
- OpenAPI 스펙(bin-api-manager) + RST 문서에서 language 설명을 "summary output language"로
  갱신(MANDATORY: 스키마/설명 변경 시 문서).
- 하위호환: 필드명 language는 유지(값 의미만 명확화). 신규 필드 추가 없음(최소 변경).

### 4.4 outputLanguage 확정 규칙 (저장/주입/dedup 3지점 일관성, R1/R2 지적)

outputLanguage는 세 지점에서 동일한 확정값을 써야 한다.
1. **확정 시점**: Start 진입 시 language 인자를 한 번 정규화한다. 비어있으면 en-US로
   확정(출력 언어 기본). **정규화는 반드시 dedup 조회(start.go:29
   GetByCustomerIDAndReferenceIDAndLanguage)보다 앞에서** 수행한다(빈 값과 en-US가 서로 다른
   요약으로 갈리는 것을 방지). 이하 모든 지점은 이 확정값을 쓴다.
2. **summary 저장**: `h.Create(..., language=확정 outputLanguage, ...)`. 저장값은 정규화 후
   값(빈 값이 아니라 en-US)이라, 같은 recording을 같은 출력 언어로 다시 요청하면 dedup이
   일관되게 동작.
3. **dedup 키**: Start의 `GetByCustomerIDAndReferenceIDAndLanguage`(start.go:29)가 이제 확정
   outputLanguage로 조회. 다른 출력 언어로 재요약하면 별도 summary 생성(의도된 동작 —
   언어별 요약 각각 보관). 빈 language와 en-US가 서로 다른 요약으로 갈리는 문제는 정규화로
   제거됨.
4. **프롬프트 주입**: RequestContent.OutputLanguage=확정 outputLanguage. activeflow 변수도
   비고 output_language도 비는 경우는 정규화로 발생하지 않음(항상 en-US 이상). 프롬프트는
   output_language 필드를 최우선으로 읽고, 그래도 비면(방어) en-US로 요약하도록 지시.
- STT 언어와는 무관: outputLanguage는 STT에 절대 전달하지 않는다(STT는 §4.2 재사용/fallback).

## 5. 스코프

In scope:
- bin-ai-manager: RequestContent.OutputLanguage, contentGet 시그니처+4호출부(call·conference는
  sm.Language, recording·transcribe는 Start language 인자), 프롬프트 Language 섹션,
  outputLanguage 정규화(§4.4), startReferenceTypeRecording transcribe 재사용(§Phase2).
- **테스트 갱신 사양**: content_test.go 5케이스(call/conference/recording/transcribe/none)
  전부 (a) contentGet 5-arg 호출로 갱신, (b) expectedRequestContent에 output_language 필드
  반영(none/빈 케이스는 정규화 후 en-US 기대). start_test.go
  Test_startReferenceTypeRecording에 재사용 경로 케이스 추가(원본 transcribe 있음->재사용,
  없음->신규 생성, IDAIManager 빈 것만 있음->신규 생성). **기존 normal 케이스(start_test.go
  Test_startReferenceTypeRecording, 현재 List 목 없음)는 순수 additive가 아니라, 재사용 조회
  TranscribeV1TranscribeList가 TranscribeStart 이전에 호출되므로 List가 빈 결과를 반환하는
  신규생성 케이스로 갱신(List 목 추가)해야 한다.** openai.Send는 marshaled
  RequestContent(output_language 포함) 검증 또는 gomock.Any() 유지 명시.
- bin-openapi-manager(있으면) + RST: language 설명 갱신.
- monorepo-javascript square-admin: 출력 언어 선택 재노출(라벨 명확화).

Out of scope:
- Google STT alternative_language_codes 자동감지 도입.
- customer별 기본 STT 언어 설정 체계.
- call/conference 경로의 transcribe 재사용(recording에 한정; 이들은 activeflow variable로
  이미 출력 언어가 주입되므로 Phase 1로 충분).

## 6. 리스크 / 트레이드오프

- **Phase 2 재사용 선택 기준(회귀 방지)**: "tm_create desc 최신 1개"는 금지(실측상 빈 ko-KR을
  고름). 확정 기준: `customer_id != IDAIManager`(원본 우선, summary가 만든 것 배제) && done &&
  transcript 개수>0. 후보 여럿이면 원본 중 최신(desc 순회 단락). 후보 없으면 신규 생성. 이로써
  admin이 만든 원음 transcript를 재사용하고, summary가 과거 오염시킨 빈 transcribe는 배제된다.
  transcribe 모델 판별 필드는 `customer_id`(owner_id 없음). §4.2/§Phase2 참조.
- **기존 오염 데이터**: 이미 존재하는 recording에 summary가 만든 빈 IDAIManager transcribe가
  남아 있어도, 재사용 로직이 그것을 배제하므로 새 요약은 정상. 오염 transcribe 자체의 정리는
  이 스코프 밖(무해, 배제됨).
- **Phase 1+2 분리 배포 금지(단정)**: Phase 1만 배포하면 recording 경로가 출력 언어를 STT로
  넘겨 오히려 회귀한다(§Phase1 말미). Phase 1과 Phase 2는 **반드시 같은 PR, 같은 배포**로
  나간다. 절대 분리하지 않는다.
- **PR 분리(레포 경계)**: 백엔드(monorepo) PR과 프론트(monorepo-javascript) PR은 레포가 달라
  각각. 백엔드 Phase 1+2는 한 PR. 프론트 Phase 3은 백엔드 배포 후 안전하게 재노출(순서 의존)
  -> 백엔드 먼저 병합/배포 후 프론트.

## 7. 리뷰 이력
- R1 (deleg_fe70d0d2, task0): CHANGES_REQUESTED. 근본원인/2축분리/fallback/API계약/스코프
  타당 확인. 블로커 2 + 수정 3: (재사용 desc 최신이 빈 ko-KR 선택 회귀 / customer_id 필터
  생략 명시 / status=done 필터 / phase 독립검증 recording 한정 / call·conference outputLanguage=
  sm.Language 출처).
- R2 (deleg_fe70d0d2, task1): CHANGES_REQUESTED. 동일 치명 결함(desc 최신=빈 ko-KR 회귀) +
  status/비어있음 가드 / customer_id 필터 / outputLanguage 3지점(저장·주입·dedup) 일관성 /
  content_test 5케이스 갱신 사양 / phase 분리배포 금지 강화.
- 정정(R1+R2 반영):
  * 재사용 선택 기준 재설계: customer_id != IDAIManager && done && transcript>0, 원본 중 최신,
    없으면 신규(§Phase2). desc 최신 1개 금지 명문화. (초안은 판별 필드를 owner_id로 적었으나
    R3 지적으로 customer_id로 정정.)
  * customer_id 필터 생략 이유 명시(admin transcribe는 실 customer_id), IDAIManager 필터
    쓰는 기존 헬퍼와 구분.
  * status=done 필터 + transcript 개수>0 검증 절차 추가.
  * §4.4 outputLanguage 정규화(en-US 기본) + 저장/주입/dedup 3지점 일관성.
  * §1.1 call/conference outputLanguage=sm.Language 출처 명시.
  * content_test 5케이스 + start_test 재사용 케이스 갱신 사양 명시(§5).
  * Phase 1+2 분리배포 금지 단정(§6).
- (재리뷰 R3/R4 대기)
- R3 (deleg_ecbf60b2, task0): CHANGES_REQUESTED. 회귀 해소 확인. 블로커 1: 판별 필드
  'owner_id' 표기 오류(transcribe 모델엔 customer_id만, owner_id 없음). 비차단: 정규화 위치
  dedup 이전 명시, status=done 레이스 폴백, RequestContent.OutputLanguage 실변경 필요.
- R4 (deleg_ecbf60b2, task1): APPROVED. 재사용 조회 현실성/language 일관성/Phase 1+2 배포강제
  확인. 권고(비차단): N+1 단락 규칙, customer_id 통일, NotEq 옵션, 기존 normal 테스트 List 목.
- 정정(R3 블로커 + R3/R4 권고 반영):
  * 문서 전체 owner_id -> customer_id 통일(§Phase2, §6).
  * customer_id 배제 방식 NotEq(DB단) 또는 코드필터 택1 명시.
  * N+1 단락 규칙(desc 순회, 첫 transcript>0 즉시 채택) 추가.
  * §4.4 정규화를 dedup 조회(start.go:29) 이전 수행 명시.
  * §5 기존 normal 테스트를 List 빈결과 신규생성 케이스로 갱신(additive 아님) 명시.
- (재리뷰 R5/R6 대기)
- R5 (deleg_19405df1, task0): APPROVED. R3 블로커 해소 + 권고 5건 반영 실코드 대조 확인.
  비차단: 이력 로그 owner_id 1건(정정), en-US fallback 혼용 코드리뷰 재확인 권고.
- R6 (deleg_19405df1, task1): CHANGES_REQUESTED. 블로커: customer_id 배제 옵션 (i)
  NotEq{IDAIManager}가 uuid.UUID에 미동작(databasehandler/main.go:47-53 SCOPE WARNING,
  string-kind 전용 -> 바이트 표현 불일치로 조용히 오작동, 회귀 재발). R5가 main.go:85-87만
  보고 위 경고를 놓침. 나머지 자기일관성/구현가능성 확인.
- 정정(R6 블로커 + R5 비차단 반영):
  * customer_id 배제를 옵션 (i) 삭제, **코드 배제(customer_id 필터 없이 조회 후 코드에서
    IDAIManager 제외)로 단일화**. NotEq UUID 미지원 경고 문서화(§Phase2).
  * 이력 로그 owner_id 잔존 1건 정정(초안 표기임을 명시).
- (재리뷰 R7/R8 대기)
- R7 (deleg_f6352e6b, task0): APPROVED. R6 블로커 해소 확인(코드 배제 단일화, DB단 NotEq
  잔존 없음). IDAIManager 코드 비교 실증(listen_trigger.go:646 동일 패턴), 구현 가능 최종 상태.
- R8 (deleg_f6352e6b, task1): APPROVED. 코드 배제 정합, page_size 유한 극단경우 실질 무해,
  자기일관성/구현가능성 확인, 실질 블로커 부재.
  **R7/R8 2연속 APPROVE -> 설계 리뷰 종료, 설계 확정.**
  (총 8회: R1/R2 CR, R3 CR/R4 A, R5 A/R6 CR, R7 A/R8 A. 실 회귀 결함 2건 차단:
   재사용 선택 기준, NotEq UUID 미동작.)

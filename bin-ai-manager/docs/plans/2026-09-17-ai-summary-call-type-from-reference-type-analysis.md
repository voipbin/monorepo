# 이슈 분석: AI Summary Call Type이 Unknown으로 나오는 문제 (VOIP-1533)

작성일: 2026-09-17
작성자: Hermes (CPO)
브랜치: VOIP-1533-AI-summary-call-type-from-reference-type
상태: Draft (이슈 분석 리뷰 대기)
성격: hardening / minimal-change (신규 엔티티/테이블/goroutine 없음)

## 1. 문제 요약

AI Summary 결과의 첫 섹션 "Call Type"이 자주 "Unknown"으로 나온다. 이 값은 코드가
이미 확정적으로 알고 있는 사실(summary.reference_type)인데도, 현재 프롬프트는
LLM이 transcript 텍스트만 보고 통화 유형을 추측하도록 설계돼 있다.

실측 (recording 3e5f1446):
```
Call Type:
- Unknown

Key Discussion Points:
- Pleasantries exchanged, including greetings.
- Mention of a test call concerning VoIP software.
(나머지 섹션은 transcript 내용으로 정상 채워짐)
```

reference_type=recording 임에도 Call Type이 Unknown이다. 나머지 4섹션이 정상이므로
transcript 자체는 유효하다. 문제는 오직 Call Type 판별 방식이다.

## 2. 근본 원인 (실코드로 확정)

파일: `bin-ai-manager/pkg/summaryhandler/content.go`

```go
type RequestContent struct {
	Prompt      string                    `json:"prompt,omitempty"`
	Transcripts []tmtranscript.Transcript `json:"transcripts,omitempty"`
	Variables   map[string]string         `json:"variables,omitempty"`
}
```

LLM에 넘어가는 것은 Prompt/Transcripts/Variables뿐이다. **reference_type이 포함되지
않는다.** 따라서 프롬프트에 "이 통화는 recording이다"라는 정보가 전혀 없다.

추가로, recording/transcribe 소스 summary는 activeflow가 없어
(activeflow_id=00000000) `contentGet`의 변수 주입 분기(`if activeflowID != uuid.Nil`)를
타지 않으므로 `Variables`도 비어, `voipbin.ai_summary.reference_type` 변수조차 주입되지
않는다.

프롬프트(main.go:136):
```
- Call Type: Identify if it was a 1:1 call, conference, support call, sales call,
  or recorded call. If it cannot be determined, write "- Unknown".
```

결국 LLM은 transcript만 보고 통화 유형을 추측해야 하고, 단서가 없으면 Unknown,
있어도 재생성마다 값이 달라지는 비결정적 출력이 된다.

## 3. 원칙 위반

VoIPBin 원칙: **코드가 사실을 확정하고 LLM은 서술만 한다(환각 금지).**
Call Type은 reference_type으로 결정되는 결정론적 메타데이터인데, LLM 추측에 맡겨
Unknown 또는 비결정 출력을 유발한다. 이는 원칙 위반이며, 이전에 "Recorded Call"로
나온 것도 LLM이 운좋게 맞춘 비결정 값이었다.

## 4. contentGet 호출부 (reference_type 확보 가능 여부 확인)

reference_type을 넘기려면 4개 호출부가 모두 그 값을 알고 있어야 한다. 실코드 확인:

| 호출부 | 파일:라인 | reference_type 확보 |
|--------|-----------|---------------------|
| call | content.go:64 (contentProcessReferenceTypeCall) | `sm.ReferenceType`(=call) 보유 |
| conference | content.go:111 (contentProcessReferenceTypeConference) | `sm.ReferenceType`(=conference) 보유 |
| transcribe | start.go:214 (startReferenceTypeTranscribe) | 상수 `summary.ReferenceTypeTranscribe` |
| recording | start.go:282 (startReferenceTypeRecording) | 상수 `summary.ReferenceTypeRecording` |

4곳 모두 reference_type을 이미 알고 있다. 파라미터 추가만으로 전달 가능하며, 추가
RPC/조회가 필요 없다.

summary.ReferenceType enum: None("")/call/conference/transcribe/recording (5종).
실제 Call Type으로 유의미한 값은 call/conference/recording/transcribe 4종.

## 5. 해결 방향 (옵션 1, 최소 변경)

### 5.1 코드 변경 (2곳)

1. `contentGet` 시그니처에 `referenceType summary.ReferenceType` 파라미터 추가.
   호출부 4곳에서 각자 보유한 reference_type 전달. **4곳 전수(누락 금지)**:
   - content.go:64 contentProcessReferenceTypeCall → `sm.ReferenceType`(call)
   - content.go:111 contentProcessReferenceTypeConference → `sm.ReferenceType`(conference)
   - start.go:214 startReferenceTypeTranscribe → 상수 transcribe
   - start.go:282 startReferenceTypeRecording → 상수 recording
   특히 **문제의 recording 버그 경로는 start.go:282**(startReferenceTypeRecording,
   StatusDone 동기 생성)이며 ContentProcess(content.go, call/conference 전용 switch)를
   거치지 않는다. content.go 2곳만 고치고 start.go 2곳을 빠뜨리면 정작 이 recording
   경로에 수정이 적용되지 않아 무효가 되므로, 4곳 모두 반드시 반영한다.
2. `RequestContent`에 `ReferenceType string` 필드 추가, contentGet에서 주입.

### 5.2 의미 축 정리 (중요)

reference_type은 기술적 "소스" 축(call/conference/transcribe/recording)이고, 기존
프롬프트의 "Call Type"은 비즈니스 분류 축(1:1/support/sales/recorded)을 섞어 지시했다.
이 혼재가 바로 Unknown의 원인이다. transcript만으로 support/sales 같은 비즈니스
성격을 판별하는 것은 비결정적이고 짧은 통화에서는 불가능하다.

결정: **Call Type 섹션을 reference_type 기반 라벨로 전면 대체한다.**
비즈니스 성격(support/sales 등)에 대한 서술은 Call Type이 아니라 Additional Notes/
Key Discussion Points에서 LLM이 자유롭게 다루면 된다. Call Type은 "이 요약이 무엇을
대상으로 하는가"를 reference_type으로 답하는 라벨로 정의한다.

용어 주의: "전면 대체"는 코드가 reference_type이라는 사실(fact)을 확정 제공한다는
의미이며, 최종 라벨 문자열은 여전히 LLM이 생성/번역한다(방식 A). 따라서 라벨 자체가
바이트 단위로 결정론인 것은 아니다(LLM 오라벨 가능). "결정론"이 적용되는 것은 입력
사실(reference_type)이 더 이상 transcript 추측이 아니라는 점이다. 완전한 문자열
결정론이 필요하면 방식 B(§Out of scope)로 분리한다.

enum → 사람이 읽는 라벨 매핑 (코드가 아니라 프롬프트 규칙으로, 방식 A):

| reference_type | Call Type 라벨(영문 canonical, LLM이 출력 언어로 렌더) |
|----------------|--------------------------------------------------|
| call | Call |
| conference | Conference |
| recording | Recorded Call |
| transcribe | Transcribed Call |
| "" (None) 또는 미지정 | Unknown |

recording → "Recorded Call"은 대표님이 이전에 정상으로 보셨던 값(§2 실측)과 일치한다.
call/conference/transcribe의 라벨(Call/Conference/Transcribed Call)은 이전 관측 정상값
대조 근거가 없는 **신규 제품 결정**이다. reference_type 값을 사람이 읽기 쉬운 형태로
매핑한 것으로, 필요 시 추후 문구 조정 가능.

### 5.3 프롬프트 변경 (Call Type 섹션)

현재:
```
- Call Type: Identify if it was a 1:1 call, conference, support call, sales call,
  or recorded call. If it cannot be determined, write "- Unknown".
```

변경(안):
```
- Call Type: A "reference_type" value is provided in the input. Use ONLY that value
  to state the call type, and translate the label into the summary's output language.
  Map exactly: "call" -> "Call", "conference" -> "Conference",
  "recording" -> "Recorded Call", "transcribe" -> "Transcribed Call".
  Do NOT infer the type from the transcription. If "reference_type" is missing or
  empty, write "- Unknown".
```

이 문구는 LLM에게 라벨링/번역만 맡기고(코드가 사실=reference_type 확정), transcript
추측을 금지한다. 비즈니스 하위 분류(support/sales)는 Call Type에서 제거된다. 이는
의도된 트레이드오프다: 비결정적 하위 분류를 잃는 대신, Call Type이 안정적이 되고
Unknown 남발이 사라진다. 하위 분류가 유의미하면 LLM이 다른 섹션에서 서술한다.

트레이드오프 명시: support/sales/1:1 하위 분류는 스키마 근거가 없는 LLM 추정이라
신뢰성이 낮았다. 이를 Call Type에서 제거하면 그 정보는 별도 구조화 필드로는 표출되지
않는다(Call Type 외 다른 필드로 support/sales를 결정론적으로 채우지 않는다). 다만
LLM이 Key Discussion Points/Additional Notes에서 자연어로 그 성격을 서술할 수는 있다.
이 상실은 known limitation으로 수용한다.

always-5-sections 무충돌: 재작성된 Call Type 지시는 프롬프트의 "Always produce all of
the following sections... Never skip a section"(VOIP-1531에서 확정된 5섹션 규칙) 블록
내부에 그대로 유지된다. Call Type은 여전히 5섹션 중 1번 섹션이고, None도 "- Unknown"
으로 항상 출력되므로 조건부 스킵을 재도입하지 않는다(§8 fallback 회귀와 무관).

### 5.4 None/default 방어

reference_type enum은 None("") 포함 5종이다. 현재 호출부 4곳은 모두 concrete 값을
전달하며, 추가로 `Start`(start.go:50)와 `ContentProcess`(content.go:38)가 빈
reference_type을 default 에러로 거부하므로, **None은 실런타임 경로로 contentGet에
도달하지 않는다**(dead-defensive). 그럼에도 향후 호출부 추가나 우발적 None 전달에
대비해:
- contentGet은 reference_type 원값을 그대로 RequestContent에 싣는다(코드가 라벨
  매핑을 하지 않으므로 switch/default 분기 자체가 없다 — 방식 A의 이점).
- 빈 값("")이 실리면 프롬프트 규칙이 "missing or empty -> Unknown"으로 처리한다.
즉 None 방어는 프롬프트 규칙 한 줄로 완결되며, 코드에 매핑 분기를 두지 않는다.
테스트의 None 케이스는 실사용 상태가 아니라 contentGet 단위 레벨의 방어 검증이다.

## 6. 스코프

In scope:
- contentGet에 reference_type 전달 + RequestContent 필드 추가
- 프롬프트 Call Type 섹션을 reference_type 기반으로 전면 대체
- 단위 테스트: contentGet이 넘긴 reference_type이 `engine_openai_handler.Send`로
  전달되는 marshaled RequestContent에 실리는지 검증. contentGet은 핸들러 내부 메서드라
  mock 대상이 아니므로, 기존 content_test.go 패턴(Send mock에 전달된 요청 본문
  assert)을 확장해 (1) 4종 reference_type 각각이 RequestContent.ReferenceType에
  올바르게 실리는지, (2) None("") 케이스 1건을 추가 검증한다. **content_test.go는
  변경 대상이다**: contentGet 시그니처에 referenceType이 추가되므로 기존 호출
  (content_test.go:132), expectedRequestContent, Send exact-match assert가 모두
  갱신되고, 현재 단일 케이스를 4종+None의 다중 케이스로 확장한다.

Out of scope:
- 방식 B(코드가 라벨을 직접 확정 + 다국어 라벨) — 후속 과제
- STT/출력 언어 분리 (VOIP-1532)
- summary가 recording마다 새 transcribe를 생성하는 문제 (별개, 이번 스코프 아님)
- 비즈니스 하위 분류(support/sales) 결정론화 — Call Type에서 제거, 필요 시 후속

## 7. 진행 타당성

- 이슈 유효: 실측(recording 3e5f1446)으로 재현 확인. 코드로 원인 확정.
- 최소 변경: 신규 테이블/엔티티/goroutine 없음. 시그니처 1개 + 구조체 1필드 +
  프롬프트 1섹션. hardening 성격으로 오버엔지니어링 없음.
- 원칙 정합: "코드가 사실 확정, LLM은 서술" 원칙에 정확히 부합.
- 진행 권장.

## 8. 리뷰 이력

- R1 (deleg_2006fa6f, task0): APPROVED. 이슈 유효성/근본원인/방식 A 선택 모두 타당.
- R2 (deleg_2006fa6f, task1): CHANGES_REQUESTED. 4지적:
  (a) None/default 처리 미명시 → §5.4 추가(프롬프트 규칙 한 줄로 처리, 코드 분기 없음).
  (b) 의미 축 혼동(reference_type=소스 축 vs Call Type=비즈니스 분류 축), enum→라벨
      매핑 테이블 필요 → §5.2 추가(Call Type을 reference_type 기반 결정론 라벨로 전면
      대체, 매핑 테이블 명시, 비즈니스 분류는 다른 섹션으로 이동).
  (c) 프롬프트 과도 제약(support/sales 하위 분류 상실) → §5.3에 의도된 트레이드오프로
      명시(결정론 확보 대신 하위 분류 제거, 필요 시 LLM이 다른 섹션 서술).
  (d) 테스트 전략 구체화(contentGet 내부 메서드라 mock 불가, Send mock 요청 본문
      assert) → §6 In scope 갱신(content_test.go 패턴 확장, 4종+None 검증).
- 정정 반영 완료. 재리뷰(R3/R4) 진행.
- R3 (deleg_f3d318fc, task0): APPROVED. R2 4지적 모두 적절히 반영, enum 일관성 확인.
  비차단: 라벨 canonical→번역 문구 명확화 권장 → §5.2/5.3에 canonical/렌더 구분 반영.
- R4 (deleg_f3d318fc, task1): CHANGES_REQUESTED. 단 finding 2·5는 R4가 엉뚱한 파일
  (VOIP-1531의 ai-summary-plaintext-format-analysis.md)을 리뷰한 오인 — 본 문서
  (call-type-from-reference-type-analysis.md)에는 해당 서술이 없음. 파일 무관한
  지적만 반영:
  (1) recording 버그 경로 start.go:282 강조, 4곳 전수 반영 명시 → §5.1 보강.
  (3) "결정론" 표현 정정(방식 A는 LLM 렌더, 입력 사실만 결정론) → §5.2 보강.
  (4) None이 Start/ContentProcess에서 거부돼 dead-defensive임을 명시 → §5.4 보강.
  (7) call/conference/transcribe 라벨은 신규 제품 결정 명시 → §5.2 보강.
  (8) 하위분류 상실이 다른 필드로 대체 불가함(known limitation) → §5.3 보강.
  (6 확인 OK) always-5-sections 무충돌 → §5.3에 명시.
  (2) content_test.go 변경 대상 명시(파일 오인 지적이나 명확화 차원) → §6 보강.
- 정정 반영 완료. 재리뷰(R5/R6) 대기.
- R5 (deleg_70d97add, task0): APPROVED. 실코드 검증(4호출부/recording 우회/enum/
  dead-defensive/content_test.go 범위) 모두 일치.
- R6 (deleg_70d97add, task1): APPROVED. 내부 일관성/구현가능성/테스트 현실성/최소변경
  확인. 2연속 Approve로 이슈 분석 리뷰 종료.

## 9. 코드 리뷰 이력

- R1 (deleg_b554b342, task0): APPROVED. build/test/lint 직접 통과, 4호출부/프롬프트
  매핑/None/always-5-sections/vendor 무오염 확인. 비차단: 테스트명 defends 오타.
- R2 (deleg_b554b342, task1): APPROVED. marshaling 일치, omitempty↔None 정합, 기존
  프로세스 테스트 무결성(gomock.Any), start.go summary import 존재 확인.
- 오타 수정(defends→defaults, 테스트 케이스명, 로직 무변경).
- R3 (deleg_29371ebe): APPROVED. build/test/lint 재실행 통과, 잔여 결함 없음.
  3회+2연속으로 코드 리뷰 종료.

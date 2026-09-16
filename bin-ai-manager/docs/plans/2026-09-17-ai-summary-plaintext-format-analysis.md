# 이슈 분석: AI Summary 결과 포맷을 평문 인용 친화적으로 개선

작성일: 2026-09-17
작성자: Hermes (CPO)
브랜치: VOIP-NEW-AI-summary-plaintext-prompt (백엔드), 별도 프론트 브랜치
상태: Draft (이슈 분석 리뷰 대기)

## 1. 문제 요약

admin 콘솔 recording 상세의 AI Summary 카드에 표시되는 요약 content에
마크다운 볼드 기호(`**...**`)가 그대로 섞여 나온다. 프론트는 이 content를
plain text(`whitespace-pre-wrap`)로 렌더하므로:

1. 화면에 `**Call Type**:`처럼 별표가 raw 문자로 노출된다(가독성 저하).
2. 고객이 요약을 복사해 이메일/티켓/CRM/메모에 붙여넣으면 `**` 기호가
   그대로 따라가 지저분해진다(평문 인용 저해).

즉 현재는 "마크다운으로 렌더하지도 않으면서 원본에 마크다운 기호가 들어가"
양쪽 다 손해인 상태다.

## 2. 실제 출력 예시 (프로덕션 확인)

```
**Call Type**: Recorded Call

**Key Discussion Points**:
- The call was brief and included basic greetings and a message suggesting
  it might be a test call for an open-source CPA.

**Important Decisions & Agreements**:
- No decisions or agreements were made during the call.

**Action Items & Next Steps**:
- No follow-up tasks or action items were identified from the call.

**Additional Notes**:
- The content of the transcription suggests this was a brief and possible
  test-related call...
```

섹션 제목에 `**` 볼드가, 항목에 `-` 하이픈이 들어 있다.

## 3. 근본 원인 (코드로 확정)

`bin-ai-manager/pkg/summaryhandler/main.go:121`의 `defaultSummaryGeneratePrompt`
지시문 **전체가 마크다운으로 작성**돼 있어 LLM이 출력에도 볼드 기호를 복제한다.
5개 출력 섹션 헤더(`**Call Type**` 등)뿐 아니라 지시부 자체(`**Language:**`,
`**Formatting:**`, `**Conditions:**`)와 번호 리스트까지 마크다운 표기를 쓴다.
프롬프트는 "plain text로 출력하라"거나 "마크다운 기호를 쓰지 말라"는 제약을
명시하지 않는다.

- 모델: `defaultModel = openai.GPT4Turbo` (main.go:117)
- 프롬프트가 요청하는 5개 섹션: Call Type / Key Discussion Points /
  Important Decisions & Agreements / Action Items & Next Steps / Additional Notes

## 3.1 summary content 소비처 전수조사

content는 AISummaryCard.js 한 곳이 아니라 다음에서 소비된다. 포맷 변경의
영향 범위를 완전하게 파악하기 위해 전수조사한다.

| 소비처 | 위치 | 렌더/사용 | 포맷 변경 영향 |
|--------|------|-----------|----------------|
| recording 상세 AI Summary 카드 | square-admin AISummaryCard.js:149-153 | `whitespace-pre-wrap` plain text | `**` raw 노출 → sanitize 필요 |
| aisummaries 상세 뷰 | square-admin aisummaries/detail.js:236-243 | `whitespace-pre-line` Textarea, raw | `**` raw 노출 → **동일 sanitize 필요** |
| flow 변수 주입 | bin-ai-manager variable.go:22 (`voipbin.ai_summary.content`) | on_end_flow의 TTS/SMS/Email이 소비 | 마크다운 제거 = TTS가 별표 안 읽음, SMS/이메일 깨끗 (긍정적 외부 영향) |
| webhook | summary_updated (PublishWebhookEvent) | 외부 구독자에게 content 전달 | 텍스트 스타일만 변경, 구조 불변(비파괴적) |

프론트 두 뷰가 모두 raw 렌더하므로 sanitize는 **공용 유틸(예: sanitizeSummaryText)**로
추출해 두 뷰에 적용한다. 특정 뷰 하나만 처리하면 다른 뷰에서 과거 데이터의
`**`가 남는다.

## 4. 제품 결정 (평문 인용 고객 최우선)

CPaaS 요약은 사람이 읽고 재사용하는 운영 데이터다. 고객은 요약을 텍스트로
복사·인용하는 것이 기본 사용 패턴이므로, content 원본을 깨끗한 plain text로
만드는 것이 정답이다. 화면 미관을 위한 마크다운 렌더링 추가 방향은 폐기한다
(원본에 마크다운 기호를 남기는 것 자체가 텍스트 이식성을 해침).

- **볼드 `**` 제거**: 섹션 제목은 평문 `Call Type:` 형태로.
- **불릿 `-` 유지**: 마크다운 전용 기호가 아니라 일반 텍스트 리스트 관습.
  이메일/메모에 붙여도 자연스럽고 가독성에 기여.
- **섹션 구분**: 빈 줄(현재도 그렇게 나옴).

## 5. 수정 대상

| 파일 | 변경 |
|------|------|
| bin-ai-manager/pkg/summaryhandler/main.go | `defaultSummaryGeneratePrompt`에서 **모든 마크다운 표기 제거**(5개 출력 섹션 헤더 `**...**` + 지시부 `**Language:**`/`**Formatting:**`/`**Conditions:**` + 번호 리스트 스타일), "plain text로만 출력, 마크다운 문법(볼드/헤더) 사용 금지, 섹션은 빈 줄 구분, 항목은 하이픈 리스트" 명시 |

- **content_test.go 변경 불필요**: content_test.go:73은 프롬프트를 리터럴이 아니라
  상수 심볼 `defaultSummaryGeneratePrompt`로 참조하므로, 상수 값을 바꿔도 테스트
  기대값이 자동으로 함께 바뀌어 깨지지 않는다. 수동 갱신 대상이 아니다.

프론트(square-admin)는 별도 PR/브랜치:
- **공용 유틸 sanitizeSummaryText(content)** 신규: 마크다운 볼드(`**`) 제거.
  과거 저장된 `**` 섞인 summary를 표시 계층에서 정리(원본 데이터 미변경).
- AISummaryCard.js: content 표시 시 sanitize 적용(line 149-153).
- aisummaries/detail.js: content Textarea에 sanitize 적용(line 236-243).
- Re-generate 언어 선택 UI 추가(같은 카드라 함께 처리). 순수 포맷 sanitize와
  신규 UX(언어 선택)를 한 PR에 담으므로 리뷰어에게 두 변경을 명시한다.

## 6. 과거 데이터 처리 및 소비처 영향

새 프롬프트는 앞으로 생성분에만 적용된다. 이미 저장된 마크다운 섞인 summary는
재생성 강요하거나 DB 마이그레이션하지 않는다(오버엔지니어링 회피).

- **프론트 표시(두 뷰)**: 공용 유틸 sanitizeSummaryText로 AISummaryCard.js와
  aisummaries/detail.js 양쪽에서 `**`를 벗겨, 과거분·미래분 모두 두 화면에서
  깨끗하게 보이도록 한다. 원본 데이터는 미변경.
- **flow 변수 / webhook(백엔드 소비처)**: 이 경로들은 원본 content를 그대로
  전달하므로 과거 저장분은 여전히 `**`가 남는다. 단, 새 프롬프트 적용 후
  생성분부터는 원본 자체가 깨끗해져 flow(TTS/SMS/Email)·webhook 소비자 모두
  자동으로 이득을 본다(프론트 sanitize와 무관하게 원본이 개선됨). 과거분의
  flow/webhook 잔존 `**`는 비마이그레이션 known limitation으로 수용한다
  (텍스트 스타일 차이일 뿐 기능/구조 영향 없음).

## 7. 진행 타당성

- 실제 프로덕션 출력으로 문제 재현 확인됨.
- 백엔드 프롬프트 수정은 저위험(문자열 상수 1곳). 프론트 sanitize는 방어적
  표시 처리로 회귀 위험 낮음.
- 평문 인용 고객 배려라는 명확한 제품 제약에 부합.
- flow(TTS/SMS/Email)·webhook 소비처에도 긍정적 외부 영향(원본 개선).
- 진행 타당.

## 8. Fallback 제거 및 항상-포맷-유지 (옵션 1, 배포 테스트 후 정정)

PR 브랜치를 프로덕션에 선배포해 테스트한 결과, 짧고 STT 품질이 낮은 통화
(recording 882d1b70, transcript = "This is boyfriend's test." / "Call void
been is an open source e PA s service." / "Thank you.")에서 요약이 통째로
"No meaningful content available for summary."로 나오는 회귀가 확인됐다.

원인: 재작성 과정에서 (1) 섹션이 "1~5 번호 출력 의무"에서 "하이픈 가이드"로
약화되고 (2) 상단 "follow strictly" 규칙으로 전체 지시 톤이 엄격해지면서,
프롬프트의 `Conditions` fallback("unrelated numbers or words without context")이
과하게 트리거됐다. 같은 입력에 옛 프롬프트는 요약을, 새 프롬프트는 fallback을
냈다.

결정(옵션 1):
- **fallback 문구 완전 제거**("No meaningful content..." / "do not generate").
- **항상 5개 섹션을 정해진 순서로 출력**. transcript가 짧거나 비어도 포맷 유지.
- 내용이 없는 섹션은 `- None`, Call Type을 못 정하면 `- Unknown`으로 채운다.
- 섹션 출력 의무를 명시("Never skip a section and never replace the whole
  summary with a single sentence")해 회귀 원인이던 섹션 약화를 되돌린다.
- 마크다운 금지·plain text·하이픈 불릿·language 변수명 교정은 그대로 유지.

코드 영향: 없음. fallback 문구는 프롬프트 문자열 안에만 있었고 content.go 등에서
그 문자열을 특별 처리하는 로직이 없으므로(grep 확인), 프롬프트 상수만 수정한다.

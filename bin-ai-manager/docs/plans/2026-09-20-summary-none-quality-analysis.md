# 이슈 분석: AI Summary가 간헐적으로 전 섹션 None을 반환 (VOIP-1536)

작성일: 2026-09-20
작성자: Hermes (CPO)
브랜치: VOIP-1536-summary-none-quality-guard
상태: 이슈 분석 (리뷰 대기)

## 1. 이슈 요약

AI Summary 생성/재생성 결과가 간헐적으로 모든 섹션이 "None"으로 채워진 무의미한 요약으로 나온다.
동일 recording을 다시 요약하면 정상 결과가 나온다.

## 2. 재현 및 확정된 사실 (프로덕션 Loki 로그, recording 9174a2ea)

2026-09-20 대표님이 로컬에서 Regenerate를 두 번 실행. 두 요청의 입력이 완전히 동일했으나 출력만 달랐다.

| | 1차 (17:21:02) | 2차 (17:21:18) |
|---|---|---|
| 입력 transcript | 7건 (동일) | 7건 (동일) |
| 프롬프트 / output_language | 동일 / en-US | 동일 / en-US |
| 모델 | gpt-4-turbo | gpt-4-turbo |
| completion_tokens | 38 | 128 |
| 결과 | 전 섹션 "- None" | 정상 요약 |

- 입력(transcript 7건)과 프롬프트가 동일한데 모델 출력만 달랐다 → LLM 출력 비결정성.
- regenerate 로직(VOIP-1535)은 정상. transcript도 정상 전달. 코드 흐름 버그 아님.

## 3. 실코드 근거 (근본 원인 3축)

### 3.1 temperature 미설정 (핵심 레버)
`content.go:221` 요약 생성 요청 `ChatCompletionRequest`에 `Temperature` 필드가 없다. go-openai는
Temperature 제로값(0.0)을 `json:"temperature,omitempty"`로 생략하므로 OpenAI 서버 기본값 **1.0**이
적용된다. 요약처럼 결정적이어야 하는 작업에 최대 무작위성이 걸려 있어, 같은 입력에도 "요약 포기(None)"
~ "정상 요약"까지 출력이 흔들린다.
- 근거: `content.go:221-233`(Temperature 미지정). 요약 생성의 실제 전송 경로는 `content.go:236`의
  `h.engineOpenaiHandler.Send` → `engine_openai_handler/send.go`(backoff.Retry 래퍼)이며, send.go도
  `*req`를 `CreateChatCompletion`에 그대로 전달하고 기본값 보정을 하지 않는다. (SendOnce/send_once.go는
  verifyOutputLanguage 검증 경로 전용이며 요약 생성에는 쓰이지 않는다.)

### 3.1-a 생성/재생성 공통 코드 (결함은 재생성 한정 아님)
`contentGet`(content.go:178)은 **생성(Start)과 재생성(Regenerate)의 유일 공통 진입점**이다.
- Start: recording(start.go:283), transcribe(start.go:241)는 contentGet를 동기 호출. call/conference는
  ContentProcess로 비동기(start.go:68/115).
- Regenerate: regenerate.go에서 contentGet 호출.
temperature 누락은 이 공통 코드(content.go:221) 한 곳이므로, **결함은 재생성에 국한되지 않고 생성 경로
(recording/transcribe 요약)에도 동일하게 존재**한다. 재현은 Regenerate로 됐지만 최초 생성도 같은 위험을
가진다. 역으로 content.go:221 한 곳 수정이 생성/재생성 모두에 동일 적용된다.

### 3.2 en(영어) 경로는 품질 검증이 전혀 없음
`shouldVerify`(content.go:293-296)는 영어 타깃을 스킵한다. en-US는 지배 트래픽인데, 검증 루프를
아예 안 타므로 단일 Send 결과가 None이어도 그대로 반환된다(content.go:251-252 early return).
비영어만 verifyOutputLanguage 재시도를 받는다.
- 근거: `content.go:251-252`(shouldVerify=false면 generateOnce 1회 후 즉시 반환).

### 3.3 검증 하네스는 "내용 실질성"을 검증하지 않음 (통과 기전: min-prose 스킵)
비영어 경로의 `verifyOutputLanguage`(content.go:316-373)도 출력이 대상 언어인지만 판정한다(VOIP-1532).
그러나 전 섹션 None 요약이 통과하는 실제 기전은 "언어 판정 통과"가 아니라 **최소 산문 가드에서 검증 자체가
스킵(pass)**되는 것이다. `proseLen`(content.go:387-406)은 헤더행(":"로 끝나는 줄)과 "- None" 항목을
제외하고 세므로, 전 섹션 None 요약은 proseLen이 0에 수렴한다. `verifyOutputLanguage`는
proseLen < languageVerifyMinProse(20룬)이면 "판정할 산문이 없다"며 검증을 스킵하고 true(pass)를
반환한다(content.go:329-332). 즉 None 요약은 "산문이라 통과"가 아니라 "산문이 없어 스킵-통과"된다.
- **design 함의**: None 감지를 추가한다면 손볼 대상은 "언어 검증(verifyOutputLanguage)"이 아니라
  "최소 산문 가드" 또는 별도 내용-실질성 판정이다. 어느 경로든(en 스킵, 비영어 min-prose 스킵) None을
  걸러내지 못한다는 최종 결론은 동일.

### 3.4 프롬프트는 이미 None 남발을 금지하고 있음 (프롬프트만으로 불충분)
`defaultSummaryGeneratePrompt`(main.go:162~)는 "짧거나 비어도 모든 섹션을 만들고, 의미 있는 내용은
반드시 캡처하라, 내용 없을 때만 - None"을 이미 명시한다. 그런데도 1차에서 전부 None이 나왔다 →
프롬프트 지시만으로는 temperature=1.0의 비결정성을 못 이긴다.

## 4. 진행 타당성

- **유효한 문제**: 사용자에게 무의미한 요약이 노출되는 실사용 결함. 대표님이 실제로 겪음.
- **우선순위**: 긴급은 아니나(재시도하면 정상), 신뢰도 훼손이라 방치 부적절.
- **의존성**: VOIP-1532(언어 하네스)/VOIP-1535(regenerate)와 같은 파일(content.go)을 다룸. 두 기능
  불변식(언어 강제, regenerate-then-update, 빈 콘텐츠 가드)을 깨지 않아야 함.
- **대안 검토**: 프롬프트 보강만으로는 부족함이 로그로 확인됨(3.4). temperature 조정이 가장 직접적이고
  비용 없는 개입. 품질 검증 추가는 오탐 위험이 있어 별도 신중 설계 필요.

## 5. 근본 원인 결론 (design에서 확정)

주 원인의 **최유력 가설**은 **3.1 temperature=1.0**이다(로그상 동일 입력·동일 프롬프트에서 출력만 상이;
단 대조 실험은 없으므로 인과 확정이 아니라 정황상 최유력). 요약 생성에 낮은 temperature(예: 0.2~0.3)를
명시하면 비결정성이 크게 줄어 None 재발률이 낮아진다. 이것이 가장 단순하고 저비용인 1차 개입이다.
- **한계 명시**: temperature<1.0도 샘플링 비결정성을 줄일 뿐 제거하지 못한다(완전 결정성은 근사도
  temperature=0 단독으로는 불가). 따라서 None 재발률은 0이 되지 않는다. 낮은 temperature는 재발 빈도를
  낮추는 완화책이지 근절책이 아니다.
- **omitempty 함정(중요)**: go-openai v1.41.2의 `ChatCompletionRequest.Temperature`는
  `float32 json:"temperature,omitempty"`이며 포인터/omitzero가 없다. 따라서 **값 0을 지정하면 제로값이
  JSON에서 생략되어 OpenAI가 다시 기본값 1.0을 적용** → §3.1이 지목한 현재 버그가 그대로 재발한다. 즉
  temperature=0은 이 필드로는 전송 불가능한 값이다. 반드시 **비영값(0.2~0.3 등)**을 써야 직렬화되어
  실제로 전송된다. (0을 꼭 써야 한다면 필드를 포인터로 바꾸거나 raw 요청을 구성하는 별도 워크어라운드가
  필요하나, 요약 목적상 불필요.)

품질 검증(전 섹션 None 감지 → 재생성)은 2차 안전망 후보이나, "요약 실패"의 판정 기준을 실측 없이
과하게 잡으면 정상 요약을 버리는 오탐이 생긴다(대표님 오버엔지니어링 지양 원칙). **이번에는 temperature
1차 개입만 적용하고 품질 검증은 미도입**하되, 재발이 실측되면 재검토한다(6절 관찰 트리거 참조). design
단계에서 이 방침을 리스크 가중해 확정한다.

## 6. 열린 질문 (design에서 해소)

- temperature 값: **비영값만 유효**(0.2 vs 0.3). 요약 다양성 vs 안정성 트레이드오프. 요약은 창의성보다
  일관성이 중요하므로 낮은 값(0.2~0.3)이 유력. **값 0은 후보에서 제외**(위 §5 omitempty 함정: 0은
  생략되어 1.0으로 되돌아가 버그 재발).
- 적용 범위: 요약 생성(defaultModel, content.go:221)에 우선 적용. **검증 모델(defaultVerifyModel=
  GPT4oMini, yes/no 판정, content.go:344)도 낮은 temperature가 부작용 없이 안전**하다(yes/no 판정은
  결정적일수록 유리하므로 다양성 손실이 없음) → 양쪽 다 적용 권고.
- 품질 검증 미도입 결정의 **관찰 트리거**: temperature 적용 후에도 전 섹션 None이 재발하는지 실측으로
  판단. 구체 신호 예: 프로덕션 로그에서 요약 응답이 all-None 패턴(proseLen==0)인 건수를 카운트,
  일정 빈도 이상이면 품질 검증(내용-실질성 판정 또는 min-prose 가드 재설계)을 후속 이슈로 착수. 지금은
  트리거만 정의하고 선제 구축하지 않는다(오버엔지니어링 지양).

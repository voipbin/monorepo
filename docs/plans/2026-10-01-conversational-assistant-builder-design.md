# VOIP-1558 Conversational Assistant Builder 디자인 (v8, 상태 없는 동기 멀티턴)

> **갱신(2026-10-02).** 이 문서의 "킬 스위치", "`ai_builder_enabled`", "기본 꺼짐", "`BUILDER_DISABLED`", "어두운 머지", status의 "꺼짐" 상태 서술은 작성 시점의 것이다. 대표님 지시로 켜기용 설정을 제거했고 구현에는 없다(4.6 참조).

Jira: VOIP-1558. 상태: **디자인 리뷰 루프 종료(6, 7회차 A, B 모두 APPROVE, 2회 연속). 7회차의 Medium과 Low는 v8에서 문서 수정으로 반영했다(설계 변경 없음).** 리뷰 이력(이 구조 기준): 1~4회차 A, B 모두 REQUEST_CHANGES, 5회차 A REQUEST_CHANGES와 B APPROVE, 6, 7회차 A, B 모두 APPROVE.

이전 구조(aicall 기반 세션)는 디자인 리뷰 7회를 거쳤으나 차단 지점이 계속 늘어 대표님이 상태 없는 방식으로 전환을 결정했다(2026-10-01). 이전 문서는 `2026-10-01-conversational-assistant-builder-design-v7-aicall-superseded.md`에 보존한다. 지적은 모두 코드로 재검증한 뒤 반영했다.

## 1. 문제와 확정 결정

square-admin의 AI 생성 화면은 하드코딩된 템플릿 8개(`square-admin/src/views/ais/prompt_templates.js`)만 제공한다. 사용자가 AI와 대화하며 요구사항을 정리하고 VoIPBin에 맞는 프롬프트, 도구, 이름을 제안받는 경로가 없다.

확정 결정(대표님):
1. 구조화 설문이 아니라 대화형(멀티턴)이다.
2. LLM 비용은 플랫폼이 부담한다. 고객 크레딧은 차감하지 않는다.
3. 생성 결과는 자동 저장하지 않는다. 기존 생성 폼에 채우고 사용자가 확인 후 저장한다.
4. 구조는 상태 없는 동기 호출이다(aicall, AI 행, activeflow를 만들지 않는다). 대화 이력은 브라우저가 보관한다.
5. **가장 중요한 요건: 질문이 미리 만든 확정형 목록이 아니라, AI가 사용자의 답에 따라 질문을 바꾸고 때로는 더 깊고 정교하게 파고들어야 한다**(Claude Code의 superpowers brainstorming과 같은 방식). 이 요건이 설계의 중심이며 2절이 가장 먼저 다룬다.

## 2. 적응형 질문 설계 (핵심)

### 2.1 원칙

- 질문 목록, 질문 순서, 단계 전이를 코드에 두지 않는다. 코드는 "알아야 할 것"(정보 차원)과 "좋은 인터뷰어의 행동 규칙"만 모델에 주고, 무엇을 물을지는 매 턴 모델이 판단한다.
- 코드가 사실을 확정하고 LLM은 서술만 한다는 원칙은 다음 경계에서 적용한다. 인터뷰 내용은 LLM이 정한다. 초안의 형식, 길이, 도구 허용 목록은 코드가 검증한다(4.4). 서버 동작은 응답의 어떤 서술 필드에도 의존하지 않는다.
- 질문을 코드 분기로 만들고 싶어지면 시스템 프롬프트의 규칙을 고친다. 고정 질문지로 되돌아가는 구현은 금지한다.

### 2.2 모델이 받는 지식 (시스템 프롬프트의 구성)

고정 문자열과 코드에서 생성하는 부분으로 구성하며 **가능한 한 작게** 유지한다(작은 모델이 지키기 어려워지는 것이 가장 큰 위험이므로 2.5에서 축소판과 비교한다). 비밀을 넣지 않는다(사용자가 추출해도 무해).

1. **역할**: VoIPBin의 AI 어시스턴트를 설계하는 인터뷰어이자 컨설턴트. 목표는 사용자의 상황에 맞는 이름, 설명, 프롬프트(`init_prompt`), 도구 선택. 사용자의 언어로 인터뷰하고 `init_prompt`는 영어로 쓰되 응답 언어를 한 줄로 명시한다.
2. **핵심 차원 4개(모두 채워지고 열린 갈림길이 없을 때 이해 확인으로 넘어간다)**: (가) 용도와 성공 기준, (나) 사용 채널(음성 통화, 채팅, SMS, 이메일. 채널이 문체를 바꾼다)과 **통화 상대의 언어**(인터뷰 언어와 다를 수 있다), (다) 대화 상대와 상황, (라) 반드시 해야 할 것과 하지 말 것.
3. **파고들기 원칙(도메인과 무관, 질문 목록이 아니다)**: 사용자의 답을 들을 때마다 "이 어시스턴트가 실제 통화에서 가장 실패하기 쉬운 지점이 어디인가"를 하나 고른다(예외, 경계, 실패 처리, 사람에게 넘기는 시점, 모를 때의 행동). 그 지점이 아직 정해지지 않았다면 그것을 묻는다. 이 판단은 답마다 새로 하며 미리 정해진 순서가 없다. 도메인 예시(예약이면 변경, 취소, 노쇼, 정보를 받아 적으면 검증과 재확인)는 이 원칙을 보이는 **예시일 뿐 체크리스트가 아니며 few-shot 안에만 둔다**(시스템 프롬프트 본문에 신호 표를 두지 않는다. 표가 있으면 모델이 항목을 순서대로 묻는 질문지로 퇴화한다는 2.5의 비교 평가 항목이다).
4. **VoIPBin 능력 카탈로그**: 초안에 쓸 수 있는 도구 6종의 이름과 사람이 쓴 한 줄 설명(builder 패키지에 직접 쓴다. `definitions.go` 첫 줄을 추출하지 않는다. `set_variables`의 "(internal tool)" 같은 문구가 도구 선택 판단을 흐릴 수 있고 추출 코드와 첫 줄 길이 테스트가 과하다). 도구가 없는 능력(결제, 외부 예약 시스템 연동 등)은 프롬프트에서 약속하지 않는다. 도구별 동작 두 문장(Flow에서 `stop_service`는 다음 노드로 넘어가고 `connect_call`은 전환한다)을 함께 둔다.
5. **프롬프트 작성 가이드**: 2.6(템플릿 섹션 헤더 골격만 포함, 본문 예시는 넣지 않는다).
6. **Few-shot 예시 2개**: 짧은 인터뷰 대화. 하나는 모호한 시작에서 위 원칙으로 파고드는 대화(도메인 예시는 여기에 담는다), 하나는 상세한 첫 설명에서 요약과 초안으로 바로 가는 대화. **예시의 도메인은 평가 시나리오의 도메인과 겹치지 않게 한다**(예: 부동산 중개 문의, 헬스장 회원 갱신 안내. 평가는 예약, 치과, 식당, 고객지원, 배달 안내, 설문, 약국 재고를 쓴다). 겹치면 평가가 적응성이 아니라 예시 암기를 측정한다.
7. **인터뷰 행동 규칙**(2.3).

### 2.3 인터뷰 행동 규칙 (superpowers 방식의 이식)

규칙은 **6개**다. 충돌 시 우선순위(질문 수 제한은 규칙 2의 한 턴 질문 수): **질문 수 제한 > 사용자의 말 > 깊이 파기(열린 갈림길이 있는 동안) > 이해 확인 > 종료 판단**. (3회차 B의 지적: 이해 확인이 깊이 파기보다 앞서면 4차원이 얕은 답 한 줄로 채워져 파고들기가 사라진다. 그래서 열린 갈림길이 있는 동안은 파는 쪽이 앞선다.)

1. **먼저 듣고 사용자의 말을 따른다.** 이미 말한 정보는 다시 묻지 않는다. 범위 안의 역질문은 먼저 답한 뒤 돌아온다. 정정하면 새 답을 따르고 이전 내용을 다시 지적하지 않는다. 실제 모순만 짚어 확인한다. 범위 밖이나 지시 무시 요청은 정중히 인터뷰로 되돌리고, 초안 이후의 **구조 변경 요청**(여러 AI로 분리, Flow 설계 등)은 지원 범위의 한계를 말하고 가능한 대안을 제시한다. 용어를 모른다는 답("모르겠어요")이 오면 기본안을 하나 제시하고 선택을 받는다.
2. **한 번에 하나, 많아야 둘.** 둘은 서로 독립적이고 가벼운 질문만 묶는다. 접근 2~3개를 제시하는 것도 그 턴의 질문 하나로 센다.
3. **답이 다음 질문을 정한다(파고들기 원칙, 2.2의 3).** 갈림길이 열려 있으면 그 주제를 1~2턴 더 판다. 한 주제는 3턴을 넘기지 않는다. **직전 두 답이 모두 질문에 새 정보를 주지 않았거나("모름", "알아서 해 줘") 선택을 위임했으면 최근 주제는 가정을 말하고 다음으로 넘어간다**(기준은 답의 길이가 아니라 정보량이다. "예약이요"처럼 정보가 있는 단답은 해당하지 않으며 계속 좁힌다. 주제별 추적이 아니라 최근 두 턴만 본다). 답이 단순하면 파지 않는다. **상한이나 가정으로 넘어간 갈림길은 닫힌 것으로 보고 `assumptions`에 넣는다.** **데이터 블록의 `checkpoint`가 true인 턴에만, `draft_exists`가 false이면 "지금 초안을 만들 수도 있고 더 다듬을 수도 있습니다"를, true이면 "더 다듬을지"를 응답 끝에 한 문장으로 덧붙인다**(질문 수에는 세지 않는 안내 문장이다. 파고들기 질문과 같은 턴에 할 수 있다. **이해 확인 요약을 하는 턴에는 이 안내를 생략한다**. 요약이 이미 "초안을 만들까요?"를 묻기 때문이다) (깊이 폭주 방어. 턴 수와 주기는 모델이 세지 않고 코드가 4.2의 데이터 블록으로 준다. `checkpoint`는 `user_turns >= 6`이고 `(user_turns - 6) % 4 == 0`일 때 true이므로 6, 10, 14, 18번째처럼 4턴마다 메시지 상한까지 반복되는 사용자 턴이다. 이력에서 안내 위치를 찾을 필요가 없다).
4. **전문가로서 의견을 낸다.** 모호하면 가정을 말하고 확인한다. 실제 갈림길이 있을 때만 접근과 권고를 짧게 제시하고 선택은 사용자에게 맡긴다.
5. **초안 전에 이해를 확인한다(요약은 대화당 최대 1회).** 핵심 4차원이 채워졌고 열린 갈림길이 없으면 "이렇게 이해했습니다: ... 이대로 초안을 만들까요?"로 한 번 요약한다. 요약에 대한 반응은 둘뿐이다. 명확한 반응(승인, 수정, 새 정보 모두)은 반영해 바로 초안을 내며 요약을 반복하지 않는다. 모호한 반응("음... 그런 것 같아요")이면 가장 불확실한 한 가지만 묻고 초안으로 간다. 요약 여부는 이력에서 판단하고(코드가 계산하지 않는 이유: 요약 문구가 자유 서술이라 코드로 식별할 수 없고, 어기는지는 시나리오 13이 확인한다) `draft_exists`가 true이면 요약하지 않고 규칙 6으로 간다. 예외: (가) 사용자가 고정 문구(프런트 버튼, 2.3의 고정 문구 2종) 또는 같은 뜻으로 전체를 맡기면 즉시 초안을 내고 가정을 `assumptions`에 명시한다. (나) **첫 사용자 메시지가 4차원 중 3개 이상을 구체적으로 답했고 열린 갈림길이 없다고 판단되면** 요약과 초안을 한 턴에 낸다(이 판단은 모델이 하며, 지키는지는 평가 2a와 2b가 확인한다. 초안이 곧 확인이다. 수정은 규칙 6).
6. **초안은 시작점이다.** 수정 요청이 오면 `current_draft`를 고쳐 다시 내고 바뀐 부분을 `message`에서 짚는다. 모르는 것은 빈 칸으로 두지 않고 `assumptions`로 알린다. 통화 상대의 언어를 확인하지 못했으면 반드시 `assumptions`에 쓴다.

**고정 문구 2종**: 한국어 "지금까지의 정보로 초안을 만들어 주세요", 영어 "Please create the draft with the information so far."(square-admin UI가 영어이므로 UI 언어에 맞춰 버튼이 둘 중 하나를 보낸다). 프런트와 프롬프트가 글자 단위로 일치해야 하므로 **골든 파일 하나를 두 PR의 체크리스트에서 문자열 비교한다.** 프롬프트는 "이 문구 또는 같은 뜻"으로 쓰기 때문에 공백이나 마침표 차이로 조용히 깨지지 않는다(정확 일치에 의존하지 않는다).

**종료 판단은 별도 규칙이 아니라 규칙 5의 진입 조건과 아래 문장이다.** **목표 턴 수를 두지 않는다.** "충분한 정보가 모이면 멈춘다"와 위 6턴 소프트 안내가 규칙이다. 서버는 이력 길이 상한(4.3)만 강제한다.

### 2.4 매 턴 응답 구조

모델은 JSON 객체를 돌려준다. **필드 순서는 `message`가 먼저 오도록 지시**한다(보증은 아니며 파서는 순서와 무관하다. 잘림이나 위반이 생겨도 앞부분이 쓸모 있도록).

| 필드 | 필수 | 설명 |
|---|---|---|
| `message` | 필수 | 사용자에게 보여 줄 평문(마크다운 금지, 복사, 인용 친화). 질문 또는 확인 또는 초안 설명 |
| `draft` | 선택 | `{name, detail, init_prompt, tool_names}`. 규칙 5의 승인 후, 예외, 수정 요청일 때만 |
| `assumptions` | `draft`와 함께 | 초안에서 가정한 항목 |

**`suggested_replies`, `captured`, `open_topics`는 v1에 없다.** 선택지 칩은 모델의 가정을 사용자에게 주입해 답을 닫을 수 있고(앵커링), 프롬프트와 프런트를 키운다. `captured`, `open_topics`는 상태가 없는 구조에서 매 턴 이력 전체에서 재계산되어 환각으로 흔들릴 수 있고 질문 품질에 기여한다는 근거가 없다. 사용자가 AI의 이해를 바로잡는 수단은 규칙 5의 이해 확인 요약이 맡는다. 세 필드 모두 2.5의 비교 평가에서 기여가 확인되면 도입한다(10절).

### 2.5 적응성 검증 (필수 수용 기준)

적응형 질문은 LLM 행동이므로 단위 테스트로 보증할 수 없다. 구현의 **첫 단계**에서 평가 하네스, 파서와 검증기(4.4, 순수 함수), 시스템 프롬프트를 먼저 만들어 반복한다. 스위트는 `builder_eval` 빌드 태그의 Go 테스트이며 실제 키가 필요해 CI에서 제외하고 수동 실행한다.

**사용자 시뮬레이터**: 고정 입력으로는 첫 질문이 달라질 때 이후 입력이 맥락과 어긋난다. 그래서 별도 LLM 호출이 **페르소나 시트**(사용자가 아는 사실, 말투, 장황함, 협조 정도)로 사용자 역을 한다. 시트에 없는 것은 "잘 모르겠어요"라고 답한다. **빌더와 다른 모델**을 쓰고(높은 temperature는 권장), 일부 페르소나는 **일부러 비협조적**(딴소리, 모순, 짧은 무응답, 용어를 모름)으로 둔다(협조적이고 일관된 사용자만 있으면 분기 쌍이 무의미해진다). 페르소나 시트의 어휘는 시스템 프롬프트의 few-shot 예시 어휘와 **겹치지 않게** 쓴다(예시 암기와 적응을 구분하기 위함).

**시나리오**(페르소나 시트로 정의, 3~8턴):
1. 모호한 요청("고객 지원 봇 만들고 싶어요"). 첫 질문이 용도 또는 채널을 좁히는 하나의 질문.
2a. 전문가가 첫 메시지에 핵심 정보를 모두 주고 **열린 갈림길이 없음**. 요약과 초안을 한 턴에(규칙 5 나) 또는 질문 1개 이내.
2b. 전문가가 풍부하게 주었으나 **갈림길이 열려 있음**(예: 사람에게 넘기는 조건이 미정). 한 턴 초안이 아니라 그 갈림길을 판다.
3. 답에서 복잡성이 드러남. 다음 질문이 그 하위 주제를 따라 깊어짐.
4. **분기 쌍 A**: 같은 첫 요청("예약 접수 봇")에 (A1) 병원 예약 변경과 취소와 연락 두절 환자를 다루는 페르소나, (A2) 단순 접수만 하는 식당 페르소나. 두 변형에서 이후 질문의 **주제**가 달라야 한다(문자열이 다르다는 것은 항상 참이므로 자동 판정하지 않고 사람이 주제를 본다).
5. **분기 쌍 B**: 같은 첫 요청에 (B1) 장황하고 상세한 페르소나, (B2) 짧고 모호한 페르소나. B1은 질문이 줄고 B2는 좁히는 질문이 나와야 한다. B2는 "예약이요" 같은 **정보가 있는 단답**에 대해 모델이 가정으로 넘어가지 않고 계속 좁히는지도 판정한다(규칙 3).
6. **축 밖 도메인**: 예시 어디에도 없는 도메인(약국 재고 문의, 배달 안내, 설문 수집). 파고들기 원칙으로 그 도메인 특유의 실패 지점을 묻는지.
7. 모순("영업시간 외에는 안 받는다"와 "24시간 응대"). 모순을 짚어 확인.
8. 정정과 역질문. 새 답을 따르고 이전 내용을 다시 지적하지 않음, 역질문에는 먼저 답하고 복귀.
9. 고정 문구(두 언어)와 "그냥 만들어 줘". 이해 확인을 건너뛰고 즉시 초안과 `assumptions`. 첫 응답에 `draft`가 있는지로 자동 판정.
10. 도구가 필요한 요구(문자로 확인서, 상담원 연결). `send_message`, `connect_call`을 근거와 함께 제안.
11. 불가능한 요구(결제 처리, 외부 예약 연동), 초안 후 구조 변경 요청. 약속하지 않고 한계와 대안 제시.
12. 범위 밖과 지시 무시 요청. 정중히 복귀.
13. **이해 확인 처리**: 요약 후 (a) 승인과 수정이 함께 옴, (b) 새 정보가 옴, (c) 모호한 승인. 규칙 5의 처리대로 동작하고 요약이 반복되지 않는다.
14. **비전문가와 위임 경계**: 용어를 모르고 "모르겠어요"를 반복하는 변형(기본안 제시, 가정 후 진행)과 함께, (a) 정보 있는 단답이 연속 2회 오면 계속 좁히는지, (b) 무정보 답이 연속 2회 오면 가정하고 `assumptions`에 기록하는지, (c) 첫 메시지의 "알아서 해 줘"는 규칙 5(가)의 전체 위임으로 받아 즉시 초안을 내는지, 중간에 한 주제에 대한 "알아서 해 줘"는 그 주제만 가정하고 넘어가는지를 판정한다(14는 이 4개 변형을 1회씩 실행하므로 평가 예산에 +3회).
15. **폭주 방어(시뮬레이터 없이 합성 이력 주입)**: `user_turns`, `draft_exists`, `checkpoint`를 (6, false, true), (6, true, true), (8, false, false), (10, true, true), 요약 턴과 겹치는 (6, false, true)로 직접 주입한 5개 케이스(마지막은 안내가 생략되는지 확인)(`checkpoint`가 false인 턴에는 안내가 없어야 한다). 첫 응답이 규칙 3대로 안내하는지(`draft_exists`가 true이면 "더 다듬을지")를 사람이 짧게 확인한다(15의 합성 케이스는 5건). 한국어 자연어 문구를 문자열 매칭하는 자동 판정은 하지 않는다.

**실행과 판정**
- **3회 반복**은 적응성의 정의에 해당하는 **2b, 3, 4의 A1과 A2, 6**(5개)에 한다(시나리오 6은 반복마다 다른 도메인, 즉 약국 재고, 배달 안내, 설문 수집을 하나씩 쓴다). **5의 B1과 B2는 각각 2회**, 나머지(1, 2a, 7~14)는 **1회**씩 실행한다.
- **자동 판정(스크립트)**: JSON 파싱 성공률, 시나리오 9의 첫 응답에 `draft` 존재, `init_prompt`에 허용 목록 밖 도구명 없음. (`?` 개수와 "질문 문자열 다름"은 한국어 질문이 `?`가 없을 수 있고 비결정성으로 항상 참이므로 쓰지 않는다.)
- **사람 판정(작성자가 아닌 독립 판정자)**: 3회 반복 5개(15건), B1과 B2(4건), 13과 9의 초안 품질(2건)은 전문을 읽는다(약 21건). 1, 2a, 7, 8, 10, 11, 12, 14는 1회분을 훑어본다(8건). 15는 합성 이력 5건을 훑어본다. 항목은 한 번에 질문이 몇 개인가(2 이하), 질문이 직전 답에서 파생됨, 이미 말한 것을 다시 묻지 않음, 도메인 특유의 실패 지점을 묻는가, 충분하면 멈춤, 한 주제를 3턴 넘게 파지 않음(규칙 3의 3턴 상한이며 "최근 두 턴" 규칙과는 별개다. 앞의 것은 주제별 깊이 상한이고 뒤의 것은 무정보 답에 대한 가정 전환이다), 초안이 대화를 반영함.
- **통과 조건(7절 1단계의 중단 기준)**: 자동 항목은 전부 통과해야 한다. 3회 반복 시나리오는 **각각 3회 중 2회 이상**, B1과 B2는 **2회 모두**(쌍 비교라 하나라도 실패하면 쌍이 무의미) 통과해야 한다. 나머지 시나리오는 1회가 통과해야 하며, **1회 실행이 미달이면 같은 프롬프트로 한 번 재실행해 노이즈인지 확인한 뒤에 프롬프트를 고친다**(노이즈에 맞춘 과적합 방지). 시나리오 9는 한국어 고정 문구, 영어 고정 문구, "그냥 만들어 줘" 세 입력을 각각 1회(3회 실행), 13은 (a), (b), (c)를 각각 1회(3회 실행)로 한다. **하나라도 미달이면 이후 단계로 가지 않고 프롬프트를 고쳐 재평가하며, 3회 반복해도 미달이면 대표님께 보고한다.**
- **비교 평가는 2축만 한다**: (가) `reasoning_effort`(**`none`과 낮은 effort를 시나리오 2b, 4, 6으로 반드시 비교한다.** 기본값 `none`의 근거는 JSON 잘림 방지(F5)뿐이며 갈림길 판정과 실패 지점 선택의 품질 근거가 아니다)와 모델 후보, (나) 파고들기 원칙만 vs 신호 표 병기(표가 질문지로 퇴화하는지 확인). 프롬프트는 **축소판을 기본**으로 시작하고 품질이 부족할 때만 키운다. 선택지 칩과 목표 턴 수는 이미 결정했으므로 비교하지 않는다. 모델 호출에 JSON 모드(`response_format`)를 쓸지 여부는 파싱 실패율 비교로 1단계에서 정한다.
- **평가 예산**: 3회 반복 5개 x 3 = 15회 + B1과 B2 2개 x 2 = 4회 + 나머지 10개 x 1 = 10회 = 약 38회 실행(9의 세 입력 +2, 13의 세 변형 +2, 14의 변형 +3 포함) x 평균 6턴 = 약 225 빌더 호출과 같은 수의 시뮬레이터 호출, 사람 판정은 전문 읽기 약 21건과 훑어보기 약 15건(15의 합성 5건 별도). `max_tokens`는 thinking 포함 여부를 1단계에서 실측한다. 미달 시 보고에는 판정 로그와 미달 항목을 함께 낸다. 이 예산과 통과 기준은 11절의 결정 사항이다.
- JSON 불량(파싱 실패, 잘림) 비율이 5%를 넘으면 서버 1회 재시도를 도입한다(10절).
- 시나리오, 페르소나 시트, 루브릭은 파일로 남겨 시스템 프롬프트를 바꿀 때마다 재실행하는 회귀 기준으로 쓴다.

### 2.6 프롬프트 작성 가이드 (생성되는 `init_prompt`)

실제 템플릿을 직접 확인한 사실에 맞춘다(F13). 템플릿의 `init_prompt`는 **영어이며 마크다운 헤더를 쓴다**(2488~5029자, 헤더 9~19개). 프롬프트 첫 줄은 `# <Agent Name>` 형태의 H1 제목이고(템플릿 공통), 섹션 구조는 `## Identity & Purpose`, `## Voice & Persona`(`### Personality`, `### Speech Characteristics`), `## Conversation Flow`(`### Introduction` 등), `## Response Guidelines`, `## Scenario Handling`이다.

- 시스템 프롬프트에는 **이 헤더 골격(약 600자)만** 넣고 템플릿 본문 예시는 넣지 않는다. 골격이 `PROMPT_TEMPLATES`와 같은 헤더 집합인지는 프런트 PR의 수동 체크리스트로 확인한다(저장소가 달라 자동 테스트가 불가능하다, 8절).
- **"음성 채널이면 마크다운 금지"는 사용자에게 보이는 `message`에 대한 규칙이며 `init_prompt`에는 해당하지 않는다.** `init_prompt`는 템플릿처럼 마크다운 헤더를 쓴다. 채널에 따라 달라지는 것은 내용이다(음성: 짧은 문장, 숫자를 풀어 말하기, 시각 요소 없음).
- **`## Tools & Capabilities` 섹션은 모델이 쓰지 않는다.** 이 섹션은 프런트의 `toolsSection()`이 `TOOL_LABELS`에서 만들어 붙인다(F13, F21). 모델이 썼을 때를 대비해 **서버가 `init_prompt`에서 해당 헤더 섹션을 제거하고 `draft_warnings`에 기록한다**(코드가 사실을 확정, 중복 방지). 모델은 `tool_names`만 정하고 `init_prompt` 본문에서 도구 이름을 사용자에게 말하지 말라는 관례를 따른다. 폼에서 도구를 바꿔도 이미 붙은 섹션은 갱신되지 않는다. 기존 템플릿과 같은 동작이며 수용한다.
- 초안 길이 목표는 템플릿과 같은 2.5K~6K자이며 코드 상한은 8000자다(4.3).

## 3. 코드로 확인한 전제 (main 4a14812da)

줄 번호는 코드 변경에 취약하므로 함수명과 파일 위주로 인용한다.

| 번호 | 사실 | 근거 |
|---|---|---|
| F1 | ai-manager에 상태 없는 LLM 호출 선례가 있다. analysis gateway는 모델 허용 목록, 입력 바이트 상한, 출력 토큰 상한, `reasoning_effort`를 받고 아무것도 저장하지 않는다. | `analysishandler/run.go`, `main.go` |
| F2 | analysis `Run`은 system 1개와 user 1개 메시지만 보낸다. 멀티턴 `messages`를 받지 않으므로 그대로 쓸 수 없다. 같은 엔진(`engine_openai_handler`)을 쓰는 새 핸들러가 필요하다. | `run.go` |
| F3 | 엔진은 `Send`(지수 백오프, 최대 1분)와 `SendOnce`(정확히 한 번, 호출자의 context deadline 존중)를 제공한다. `Send`는 `backoff.Retry`에 context를 쓰지 않아 deadline이 무력하다. 대화형에는 `SendOnce`가 맞으며 Gemini의 일시적 429, 503을 재시도하지 않으므로 UX의 "다시 시도"가 부담한다. | `engine_openai_handler/send.go`, `send_once.go` |
| F4 | analysis 엔진은 `AnalysisEngineBaseURL`에 "generativelanguage"가 포함되면 `cfg.GoogleAPIKey`를, 아니면 `cfg.EngineKeyChatGPT`를 쓴다(OpenAI 롤백 경로). Builder는 **analysis와 같은 엔진 인스턴스를 공유**하고 가용성은 그 엔진이 쓰는 키가 비어 있지 않은지로 판정한다(기동 시 계산). `main.go`의 기존 경고는 `GoogleAPIKey`만 본다. Gemini 호환 endpoint는 strict json_schema를 완전 지원하지 않아 analysis는 `Strict:false`를 쓴다. 따라서 모델이 스키마를 어길 수 있다. | `cmd/ai-manager/main.go` |
| F5 | analysis의 `reasoning_effort=none`은 Gemini thinking이 출력 예산을 소모해 JSON이 잘리는 문제(`finish_reason=length`)를 막는다. 기본 `analysis_max_output_tokens`는 16384다. 인터뷰 품질에 thinking이 도움이 되는지는 미확인이며 2.5로 측정한다. | `run.go`, `config/main.go` |
| F6 | RPC 기본 timeout은 3초이고 analysis RPC는 호출자가 timeout(ms)을 넘긴다(timeline-manager는 120초). Builder는 명시 timeout이 필요하다. | `requesthandler/main.go`, `ai_analysis.go` |
| F7 | api-manager `AICreate`는 `IsDirect()`만 거부하고 `hasPermission(CustomerAdmin or CustomerManager)`를 요구한다. **`AuthIdentity.HasPermission`은 accesskey에 `CustomerAdmin & p != 0`을, delegate에 같은 조건(프로젝트 권한 제외)을 적용해 true를 줄 수 있다.** 즉 "같은 권한"만으로는 accesskey와 delegate가 통과한다. Builder는 UI 전용이므로 servicehandler 첫머리에서 `!a.IsAgent()`를 거부한다. | `servicehandler/ai.go`, `models/auth/auth.go` |
| F8 | 초안 도구 후보 6종(`connect_call`, `stop_service`, `send_email`, `send_message`, `set_variables`, `case_create`)은 `AllToolNames`의 원소이고 기존 템플릿이 쓰는 도구 합집합이다. `Description` 첫 줄이 한 문장이지만 `set_variables`는 "(internal tool)"이 붙는다. 그래서 카탈로그 설명은 추출하지 않고 builder 패키지에 직접 쓴다. | `toolhandler/definitions.go`, `prompt_templates.js` |
| F9 | `cachehandler`의 `listenIncrExpireScript`는 INCR 후 매번 EXPIRE를 다시 건다(슬라이딩 TTL). 일일 고정 윈도우에는 그대로 쓸 수 없다. Redis EVAL은 원자 실행이므로 INCR과 EXPIRE 사이 크래시는 스크립트 안에서 발생하지 않는다. `listenTTLSeconds(ttl)`가 TTL 0을 1초로 올린다. | `cachehandler/listen.go` |
| F10 | ai-manager `processRequest`는 함수 최상단에서 `log`를 `"request": m`(Data 포함)로 만들고, 404 기본 분기와 말미의 `log.Errorf("Could not handle...")`가 이 `log`를 쓴다. `processV1ServicesTypeAnalysisPost`도 자체 `log`에 `"request": m`을 달아 `log.Errorf`를 호출한다. 성공 경로에는 본문 로그가 없다. 새 핸들러가 오류를 내면 **대화 본문 전체가 오류 로그에 남는다.** | `listenhandler/main.go`, `v1_services.go` |
| F11 | api-manager 라우트는 `v1.0` 그룹에서 `Authenticate`, `CustomerRateLimit`, `EnforceAccountStatus`, `DirectResourceScope` 미들웨어를 거치는 **인증 라우트**다(선례 `server/aipromptproposals.go`). `CustomerRateLimit`은 Redis 오류 시 fail-open이며 거부 시 429 `RESOURCE_EXHAUSTED`를 낸다. `ListenTurnCountIncr`도 오류 시 경고만 하고 진행한다. | `lib/middleware/customer_ratelimit.go`, `aicallhandler` |
| F12 | `analysis.Response`는 `prompt_tokens`, `output_tokens`, `finish_reason`, `truncated`를 돌려준다. 실제 토큰 사용량을 계측할 수 있다. | `models/analysis` |
| F13 | 템플릿 `init_prompt`는 영어, 마크다운 헤더, 2488~5029자. `## Tools & Capabilities`는 프런트 `toolsSection()`이 `TOOL_LABELS`에서 생성한다(통화 관점 문구). `DEFAULT_TOOLS_INTRO`에 "never mention tool names out loud" 관례가 있다. | `prompt_templates.js` |
| F14 | `POST /ais` 경로에는 name, detail, init_prompt 길이 검증 코드가 없다(servicehandler, aihandler grep 결과 없음). DB 컬럼은 `name varchar(255)`, `detail text`다. 따라서 "기존 검증과 일치"는 성립하지 않고 Builder가 상한을 새로 정한다. `init_prompt` 컬럼 길이는 구현 1단계에서 확인한다. | `servicehandler/ai.go`, dbscheme `ai_ais` |
| F15 | api-manager에는 파일 업로드 2곳 외에 요청 본문 크기 제한(`http.MaxBytesReader`)이 없다. `c.BindJSON` 전에 큰 본문이 메모리에 올라온다. | `server/service_agents_files.go`, `storage_files.go` |
| F16 | servicehandler 관례(`AICreate`)는 입력(`init_prompt` 등)을 `log.WithFields`에 싣는다. 그대로 복제하면 본문이 로그에 남는다. gin 액세스 로그는 본문을 기록하지 않는다. | `servicehandler/ai.go`, `cmd/api-manager/main.go` |
| F17 | **정정(3회차 리뷰 A H2)**: `rabbitmqhandler/consume.go`는 요청 파싱 실패 외에 **응답 본문(`res`, `Data`에 LLM 응답 포함)을 에러 문자열에 `%v`로 싣는 경로가 둘 있다**(`:317` marshal 실패, `:331` reply publish 실패). 에러는 `consumeRPCWorker`의 `log.Errorf("Could not execute the consumer. err: %v")`로 출력된다. 브로커 일시 장애로 reply publish가 실패하면 대화 파생 내용이 로그로 나갈 수 있다. 요청 쪽 파싱 실패 경로는 정상 JSON이라 해당하지 않는다. | `bin-common-handler/pkg/rabbitmqhandler/consume.go` |
| F18 | `cerrors`는 `InvalidArgument`, `PermissionDenied`, `ResourceExhausted`(429), `Unavailable`(503), `Internal`(500) 등을 제공하고 `HTTPStatusFor`가 상태를 HTTP로 매핑한다. **502, 504, 413은 만들 수 없다.** `error_translate.go`는 `VoipbinError`를 그대로 통과시키고 `context.DeadlineExceeded`는 503 `REQUEST_TIMEOUT`으로 바꾼다. 그래서 Builder의 모든 오류는 reason 코드로 구분하는 `Unavailable`(503), `ResourceExhausted`(429), `InvalidArgument`(400), `Internal`(500)만 쓴다. | `models/errors/{constructors,rpc}.go`, `server/error_translate.go` |
| F19 | ai-manager는 요청 큐 하나를 **워커 10개**(`ConsumeRPC(..., numWorkers=10, ...)`)로 소비하며 각 워커는 요청을 **직렬 처리**한다. 통화 중 도구 실행, aicall, AI 요청이 같은 큐를 쓴다. ai-manager는 **레플리카 2개**(`k8s/deployment.yml`)이므로 프로세스당 상한은 전역으로 2배다. 워커는 ack를 먼저 하고 처리한다. Builder 호출이 최대 LLM deadline(40초)과 파싱 시간 동안 워커를 점유할 수 있다. 선례로 timeline-manager의 analysis 호출(타임아웃 120초)도 같은 큐를 쓰지만 플랫폼 내부 파이프라인이 유발한다. Builder는 고객이 직접 유발한다. | `listenhandler/main.go`, `rabbitmqhandler/consume.go`, `timeline-manager analysishandler/main.go` |
| F20 | `geminiaudithandler.Sanitize`는 `---`를 `[DELIMITER_ESCAPED]`로 바꾸는 것이 전부다. 마크다운 `init_prompt`의 `---` 구분선이 변조되고 사용자가 같은 문자열을 입력하면 방어가 없다. 그래서 Builder는 `Sanitize`를 쓰지 않는다. `geminiproposalhandler.ParseProposalResponse`는 `json.Unmarshal`만 하며 코드펜스 제거나 JSON 추출 선례가 없다. | `geminiaudithandler/main.go`, `geminiproposalhandler/main.go` |
| F21 | 프런트(`prompt_templates.js`): `toolsSection`과 `TOOL_LABELS`는 `const`이며 **export되지 않는다**(export는 `TEMPLATE_ICONS`, `TEMPLATE_COLOR_CLASSES`, `PROMPT_TEMPLATES`). `ais_create.js`의 `handleTemplateSelect`는 name, engineModel, customModelInput, initPrompt, aiType, selectedTools, enableAllTools를 설정하고 다이얼로그를 닫는다. `detail`은 uncontrolled `ref_detail`이다. 다이얼로그는 `templateDialogOpen=true`로 시작한다. 템플릿의 engineModel은 `gemini.gemini-2.5-flash`다. | `square-admin/src/views/ais/prompt_templates.js`, `ais_create.js` |

## 4. 설계

### 4.1 구성

- **square-admin**: 템플릿 모달에 "Build with AI" 카드, 채팅 패널, 초안 미리보기, "이대로 만들기". 대화 이력은 `sessionStorage`에 보관(5절).
- **api-manager**: `POST /ai_builder/chat`, `GET /ai_builder/status`(인증 라우트). 인증, 권한, 본문 크기 제한, 입력 검증, `AIV1BuilderChat` RPC.
- **ai-manager**: 새 `builderhandler`. 시스템 프롬프트 조립, 멀티턴 `SendOnce`, 응답 파싱과 검증, 일일 카운터, 계측. 내부 라우트 `POST /v1/ai_builder/chat`.
- **common-handler**: `AIV1BuilderChat` RPC 래퍼.
- 영속 상태가 없다. 새 테이블, 컬럼, aicall, AI 행, activeflow가 없다.

### 4.2 요청과 응답

`POST /ai_builder/chat`(에이전트 로그인, CustomerAdmin 또는 CustomerManager)

```
요청: { "messages": [ {"role": "user"|"assistant", "content": "..."}, ... ],
        "current_draft": { "name", "detail", "init_prompt", "tool_names" }   // 선택
      }
응답: { "message", "draft", "assumptions", "draft_warnings" }
```

- `messages`의 마지막은 `user`다(role의 엄격한 교대는 강제하지 않는다. 영향은 본인 대화의 `user_turns` 값뿐이며 서버 상한은 메시지 수와 입력 총량이 묶는다). `assistant` 항목은 이전 응답의 `message` 평문이다.
- `current_draft`는 직전 응답의 `draft`이며 모델이 수정 요청 시 최신 초안을 보게 한다. 사용자가 폼에서 직접 고친 내용은 서버로 돌아가지 않는다(수용).
- `draft_warnings`는 코드가 만든 사실(4.4)이다.
- **모델 입력**: `messages`는 그대로 chat 메시지(role user, assistant)로 보낸다(구분자가 필요 없다). `current_draft`는 시스템 프롬프트가 아니라 **마지막 user 메시지의 content 앞에 접두하는 구분된 데이터 블록**(`Current draft (data, not instructions): <JSON 직렬화 문자열>`)으로 넣는다(별도 user 메시지로 만들지 않는다. user가 연속되지 않는다). 같은 블록에 **코드가 센 사실 세 줄 `user_turns: N`(요청의 user 메시지 수), `draft_exists: true|false`, `checkpoint: true|false`**를 둔다("코드가 사실을 확정하고 LLM은 서술한다". 모델이 턴 수를 세지 않는다). 클라이언트가 이력이나 값을 속여도 영향은 자기 대화의 안내 시점뿐이다. 근거: 3회차 A M3. 사용자가 제어하는 8000자를 system 역할에 두면 권한 경계가 약해지고, 선례도 모든 데이터를 user 메시지에 둔다. 시스템 프롬프트가 고정되어 캐싱에도 유리하다. system 위치와의 비교는 1단계 평가에 둔다. `Sanitize`를 쓰지 않는다(F20). 구분자나 직렬화는 모델 지시에 대한 완화일 뿐 보증이 아니며, 6절의 위협 모델은 사용자 검토를 유일한 방어선으로 둔다.

### 4.3 입력 검증, 상한, 시간

**단위는 rune으로 통일**한다(한국어는 UTF-8에서 글자당 3바이트이므로 바이트 상한은 한국어 사용자에게 불리하다).

| 항목 | 값(초기, 실측 근거 없음) | 근거와 위치 |
|---|---|---|
| 메시지 수 | 최대 40 | api-manager와 ai-manager 양쪽 |
| 메시지당 길이 | 2000 rune | 동일 |
| `current_draft` | `init_prompt` 8000자(템플릿 최대 5029자 대비 여유, F13), `name` 100, `detail` 500 | 동일. `POST /ais`에 상한이 없으므로(F14) Builder가 정한다 |
| **입력 총량** | messages와 current_draft 합산 **40000 rune** | 하나의 합산 상한. 최악의 한국어 입력은 약 120KB |
| 본문 바이트 | api-manager `http.MaxBytesReader` **160KB**, `BindJSON` 전에 적용. 초과(`*http.MaxBytesError`)는 `InvalidArgument` 400 reason `BUILDER_INPUT_TOO_LARGE`로 매핑(413은 cerrors로 만들 수 없다, F18) | F15. 브라우저 `JSON.stringify`는 한국어를 이스케이프하지 않으므로 합법 입력이 들어온다. `\uXXXX`로 이스케이프하는 비브라우저 클라이언트는 상한 내 입력도 거부될 수 있다(수용) |
| 출력 토큰 | `max_tokens` **4096**으로 시작(영어 프롬프트 5K자는 약 1.5K 토큰), 구현 1단계 실측으로 확정(잘림은 `truncated`로 감지) | F5, F13 |
| 고객당 일일 요청 수 | 200 | ai-manager, Redis |
| **ai-manager 동시 Builder 호출** | 프로세스당 **초기 3, 1단계 지연 실측 후 확정**(레플리카 2개이므로 전역 6, 워커 전체 20개) | 세마포어. 초과 시 즉시 429 `BUILDER_BUSY`(대기하지 않는다). F19의 워커 10개 중 최대 3개만 Builder가 점유해 통화 중 도구 실행 등이 굶지 않는다 |

계산 검증: 40메시지 x 2000 rune = 80000 rune이지만 합산 상한 40000 rune이 먼저 걸린다. 즉 40개 상한은 메시지가 짧을 때만 도달하며 긴 메시지가 섞이면 합산 상한이 먼저다. 40000 rune의 한국어는 약 120KB이고 `MaxBytesReader` 160KB 안에 들어온다.

**시간**: ai-manager LLM 호출 deadline **40초**(`SendOnce`의 context), RPC timeout **55초**, 클라이언트 timeout **65초**. 큐 대기가 없으면 서버가 먼저 끝난다. **워커가 모두 바쁠 때는 큐 대기 + 40초가 55초를 넘을 수 있고, 이 경우 api-manager가 먼저 타임아웃되어도 ai-manager는 ack를 먼저 한 뒤 처리하므로 LLM 호출과 카운트가 계속된다**(3회차 A M2). 세마포어는 이 상황의 발생 빈도를 낮추지만 없애지는 않는다. `requesthandler`의 circuit breaker(연속 5회 실패에 30초 open)는 큐 단위라 Builder 타임아웃이 쌓이면 ai-manager 큐 전체가 막힐 수 있으므로 이는 1단계 실측과 운영 메트릭(4.8)에서 확인한다. **api-manager 앞단(LB, ingress)의 요청 timeout이 65초 이상인지 인프라 설정에서 확인한다(코드 밖).** 지연이 상한에 닿으면 503 reason `BUILDER_TIMEOUT`과 "다시 시도"이며 연속 2회 실패하면 "잠시 후 다시 시도하세요"로 안내한다. 지연 실측(초안 턴, 수정 턴 포함)은 구현 1단계에 둔다. 40초가 부족하면 그 근거로 값을 올리되 "RPC timeout > LLM deadline + 큐 대기 여유" 관계는 유지한다.

**circuit breaker 결정(4회차 A H2, 코드 확인)**: breaker는 호출자(api-manager)에서 큐 단위로 동작하며 **전송 오류만 실패로 센다**(`send_request.go`). 전송 오류에는 RPC timeout(`context.DeadlineExceeded`)과 **요청 취소(`context.Canceled`, 사용자가 AbortController로 취소하거나 브라우저를 닫을 때)가 모두 포함된다**(`cctx`가 요청 ctx에서 파생된다). Builder 한 건의 기여는 작고 일반 AI 트래픽의 성공이 연속 실패를 끊으므로 별도 대응은 하지 않고 1단계 실측 항목으로 둔다. `BUILDER_BUSY`, `BUILDER_DAILY_LIMIT` 같은 응용 오류는 응답으로 돌아오므로 성공으로 센다. 연속 5회 실패에 30초 open이고 성공이 연속 실패를 끊으므로, 일반 AI 트래픽이 섞여 있는 한 breaker가 Builder 때문에 열리려면 Builder 타임아웃이 연속으로 5번 나야 한다. 이를 막는 방법은 (가) deadline을 낮춰 큐 대기에도 타임아웃이 나지 않게 하고(40초와 55초, 위) (나) 세마포어로 큐 점유를 제한하는 것이며, 둘이 breaker 오염의 방어다. 별도 큐와 breaker 제외 옵션은 실측 신호가 없는 선제적 분리라 하지 않는다. **관측과 변환(5회차 A)**: breaker를 여는 신호는 ai-manager가 돌려주는 `BUILDER_TIMEOUT`(응답이 오므로 성공으로 센다)이 아니라 **api-manager의 RPC timeout**이다. 그래서 api-manager의 Builder 서버 핸들러는 `context.DeadlineExceeded`를 **`BUILDER_TIMEOUT`(503)으로, `ErrCircuitOpen`을 503 `SERVICE_UNAVAILABLE`로 변환**하고(전역 `error_translate.go`는 각각 `REQUEST_TIMEOUT`, 500으로 바꾸므로 Builder 라우트 한정), **api-manager에 Builder timeout(`DeadlineExceeded`만)과 `ErrCircuitOpen` 카운터**를 둔다(클라이언트 취소 `Canceled`는 breaker 오염 신호이지만 별도 큐 트리거에는 섞지 않으며 카운터는 세지 않는다). 이 변환으로 프런트는 큐 대기 때문에 먼저 끊긴 경우에도 같은 `BUILDER_TIMEOUT` reason을 받는다. **별도 큐 트리거**: api-manager의 Builder timeout 카운터 또는 `ErrCircuitOpen`이 운영에서 관측되면 별도 큐를 설계한다(10절).

**호출 순서**: 입력 검증, 세마포어 획득(실패하면 429 `BUILDER_BUSY`, 카운트하지 않음), `BuilderChatCountIncr`(상한 초과면 429 `BUILDER_DAILY_LIMIT`, 세마포어 해제), LLM 호출, 해제(모든 오류 경로 포함, **해제는 `defer`로 보장**한다. ctx 취소와 panic은 4.5의 `recover`와 같은 defer 체인에서 처리). 처리되지 않은 요청이 일일 횟수를 소모하지 않게 하는 순서이며 8절 테스트에 포함한다.

**일일 카운터**: `cachehandler.BuilderChatCountIncr(ctx, customerID, ttl)`. 새 Lua 스크립트 `local n = redis.call("INCR",KEYS[1]) if redis.call("TTL",KEYS[1])<0 then redis.call("EXPIRE",KEYS[1],ARGV[1]) end return n`. INCR 직후 키는 존재하므로 TTL은 -1(만료 없음) 또는 양수다. 첫 INCR이거나 TTL이 없는 키(수동 조작, `PERSIST`)에서 TTL을 걸어 고객이 영구 429가 되는 상태가 자가 치유된다. 호출 방식은 기존 `ListenTurnCountIncr`(`h.Cache.Eval`)와 같고 같은 패키지의 `listenTTLSeconds`를 재사용한다. **"일일"은 첫 요청 후 24시간 고정 윈도우이며 UTC 날짜 경계가 아니다.** 검증이 카운트보다 먼저 실행되어 검증 실패는 카운트하지 않는다. LLM 응답 불량(`BUILDER_RESPONSE_INVALID`)은 플랫폼이 이미 비용을 지불했으므로 카운트한다(프런트의 "다시 시도" 안내에 "횟수가 차감될 수 있음"을 넣는다). Redis 오류 시 **fail-closed**(503 `BUILDER_UNAVAILABLE`)다. `CustomerRateLimit`과 `ListenTurnCountIncr`(둘 다 fail-open)와 다르며 의도적이다. 비용 방어가 우선이고 영향 범위가 Builder 기능뿐이다. 분당 폭주는 이 라우트에도 적용되는 `CustomerRateLimit`에 의존하며 구현 1단계에서 이 라우트의 tier 값이 LLM 호출 비용에 충분한지 확인한다.

### 4.4 응답 파싱과 검증 (코드가 사실 확정)

`Strict:false`이므로 모델은 스키마를 어길 수 있다(F4). 파싱은 **새 구현**이다(F20: 선례는 `json.Unmarshal`뿐이다).
- 알고리즘: 응답 문자열의 각 `{`(최대 5개)에서 `json.Decoder`로 **`map[string]json.RawMessage`** 하나를 디코드하고, **`message`가 비어 있지 않은 문자열인 첫 객체를 채택(`draft`가 있고 `assumptions`가 없으면 빈 배열로 취급한다. **`draft`는 `name`과 `init_prompt`가 비어 있지 않은 문자열일 때만 채택하고, 아니면 `draft`를 버리고 `draft_warnings`에 기록한다. `message`가 유효하면 응답은 유지한다**. 검사 순서: 필드 디코드, `draft` 필수 검사, `## Tools & Capabilities` 제거 후 `init_prompt` 재검사(비면 `draft`를 버림), 길이 절단. `draft`를 버리면 `assumptions`도 함께 버리고 `draft.tool_names` 같은 중첩 필드가 타입 불일치이면 `draft` 전체가 아니라 그 필드만 버린다. `draft`가 버려진 응답의 `message`가 초안을 언급할 수 있으므로 프런트는 `draft_warnings`가 있고 `draft`가 없으면 "초안이 생성되지 않았습니다. 다시 요청하세요"를 표시한다)**한다(앞 설명에 `{}`나 유효한 다른 JSON이 있어도 건너뛴다). 코드펜스와 앞뒤 설명은 무시된다. 이후 필드별로 디코드해 타입이 맞지 않는 필드만 버린다.
- 알 수 없는 필드는 무시한다. 필드 타입이 맞지 않으면 그 필드만 버리고 `message`가 유효하면 응답은 유지한다. 채택할 객체가 없거나 `truncated`이면 `Unavailable` 503 reason `BUILDER_RESPONSE_INVALID`다(502는 만들 수 없다, F18).
- `draft.tool_names`는 허용 목록 6종과의 교집합만 남긴다(fail-closed 허용 목록, 새 도구가 추가되어도 자동 포함되지 않는다). 제거된 도구와 잘린 필드는 코드가 `draft_warnings`에 기록한다.
- 길이 상한(8000, 100, 500)으로 자르고 잘림을 `draft_warnings`에 기록한다.
- `init_prompt`의 `## Tools & Capabilities` 섹션은 제거하고 경고한다(2.6). **제거 범위는 해당 헤더(대소문자와 앞뒤 공백 무시, `##`와 `###`, 여러 번 나오면 모두)부터 다음 같은 수준 이상의 헤더 직전 또는 문서 끝까지**이며 8절에서 변형을 테스트한다.
- **허용 목록 밖 도구명 경고**: `AllToolNames`에서 허용 6종을 뺀 집합의 이름이 `init_prompt`에 **단어 경계로** 나타나면(`stop_service`와 `stop_flow`는 접두사가 같으므로 부분 문자열 매칭을 하지 않는다) `draft_warnings`에 추가한다. 밑줄 포함 식별자라 오탐이 적고 코드가 사실을 확정하는 유일한 초안 내용 검사다. 사용자 인지용이며 보안 통제가 아니다. 개념적 약속(결제, 연동)은 문자열로 잡을 수 없어 시스템 프롬프트 규칙과 시나리오 11의 평가에 의존한다.
- 서버 동작은 응답의 서술 필드에 의존하지 않는다.

### 4.5 개인정보와 로그 (대화를 서버에 남기지 않는 약속의 이행)

서버는 대화를 저장하지 않는다. DB, Redis, 파일 어디에도 요청과 응답 본문을 쓰지 않는다. **RabbitMQ 큐는 요청을 `text/plain` 본문으로 전달하며 큐의 영속 설정에 따라 브로커 메모리나 디스크에 처리 전까지 일시적으로 남을 수 있다**(워커는 ack를 먼저 하므로 재전달로 다시 처리되지는 않는다). 이 범위는 "VoIPBin 애플리케이션이 저장하지 않는다"이며 브로커 내부 전달 상태는 포함하지 않는다. 일일 카운터 키는 customer id와 개수만 가진다. 이 약속은 **VoIPBin 서버 범위**이며 Google(외부 LLM)에 전송된 내용의 보관은 Google 정책을 따른다(고지, 5절).

코드로 확인한 유출 경로(F10, F16, F17)와 이행 방법. **규율이 아니라 구조로 막고, 막지 못하는 경로는 명시한다.**
- **ai-manager `processRequest`**: 최상단 `log`가 `m` 전체를 필드로 싣고 404 분기와 말미 오류 분기가 이를 쓴다. Builder 라우트는 **`processRequest`의 `log` 생성 이전에 별도 함수로 분기해 처리하고 반환**한다(그 경로에는 `m`을 담은 `log`가 존재하지 않는다). 그 함수 안에서 `m`이나 `m.Data`를 로그에 싣지 않고, **언마샬 실패를 포함한 모든 오류 경로를 `errorResponse(err)`로 변환해 `(response, nil)`을 반환**하며(이 계약이 4.7의 reason 보존 전제다. `processRequest` 말미의 오류 변환을 우회하므로 핸들러가 직접 `errorResponse(ve)`를 호출해야 한다. 8절에 ai-manager의 실제 `sock.Response`를 api-manager 쪽에서 복원하는 왕복 테스트를 둔다), `defer recover`로 panic도 본문 없이 500 응답으로 바꾼다. 선례 `processV1ServicesTypeAnalysisPost`의 자체 `log`(`"request": m`)는 복제하지 않는다. `SendOnce`가 돌려준 err는 provider의 4xx 응답이 입력을 에코할 수 있으므로 `%v`로 로깅하거나 응답에 싣지 않고 분류 코드(타임아웃, 인증, 한도, 기타)만 남긴다.
- **api-manager**: 새 서버 핸들러와 servicehandler는 `a.AgentID()`, `a.CustomerID`, 메시지 개수만 로깅한다. `AICreate`의 입력 로깅 관례(F16)를 복제하지 않는다. `BindJSON` 실패 로그에 본문이 들어가지 않게 한다.
- **RPC 계층(정정)**: 요청은 우리 RPC가 만든 정상 JSON이므로 요청 파싱 실패 경로는 해당하지 않는다. **그러나 응답 쪽에는 `consume.go:317`, `:331`이 응답 본문(`res`)을 에러 문자열에 싣고 `log.Errorf`로 출력하는 경로가 있다**(F17). 브로커 일시 장애로 reply publish가 실패하면 `draft`와 `init_prompt`가 로그로 나간다. 해결: 세 에러 문자열(`:283`의 요청 본문, `:317`과 `:331`의 응답 본문)에서 본문을 제거한다(`:283`은 큐가 공용이라 다른 발행자의 비정상 본문에서도 대화가 남을 수 있다)(공유 라이브러리의 소규모 변경이며, 응답 본문을 에러에 싣는 것은 어느 서비스에도 바람직하지 않다). 변경은 7절 단계 4의 common-handler에 포함하고 **vendor 정정(구현 계획 1회차에서 확인, 설계 변경 아님)**: 이 저장소는 vendor를 git에 커밋하지 않는다(루트 `CLAUDE.md`, `.gitignore`의 `vendor/`, Dockerfile이 빌드 시 `go mod vendor`를 실행하고 `go.mod`의 `replace`가 최신 `bin-common-handler` 소스를 가리킨다). 따라서 "vendor 갱신"은 커밋할 산출물이 아니고, 배포 이미지는 이미 `Qos(numWorkers)`와 `maxEventRetries`를 포함한 최신 `consume.go`로 빌드된다. 위의 "ai-manager vendor 사본은 main과 이미 다르다"는 로컬 체크아웃의 오래된 사본 이야기였다. 그래서 이번 PR이 소비 동작을 새로 바꾸지 않으며(이미 배포 상태), 영향 검증과 "갱신하지 않는 대안"은 이 PR의 게이트가 아니다. 로컬 검증에는 `go mod vendor`를 쓰되 커밋하지 않는다.
- 미확인(구현 1단계에서 확인하고 문서에 기록): 트레이스(otel, sentry) 유무, api-manager 인증 이후 미들웨어와 에러 미들웨어의 본문 로깅, Loki가 stdout 이외의 소스를 수집하는지(인프라 파이프라인).
- **테스트는 `processRequest` 전체를 통과시키는 형태**여야 한다(핸들러 함수만 호출하면 상위 `log`의 필드를 잡지 못한다). 오류(LLM 오류 mock, 파싱 실패, 검증 실패, 언마샬 실패)와 panic을 유발한 뒤 로그 캡처에 사용자 입력 문자열이 없음을 확인한다. api-manager에도 같은 테스트를 둔다.
- 오류 응답과 메트릭 라벨에 대화 내용이 들어가지 않는다.

### 4.6 설정

- **켜기와 끄기용 설정은 두지 않는다(대표님 지시, 2026-10-02).** 키가 있으면 사용 가능하다. **`GET /ai_builder/status`는 api-manager가 ai-manager의 전용 RPC `AIV1BuilderStatus`를 호출해 `available = 키 있음`을 돌려준다**(api-manager가 설정을 따로 가지지 않는다). **status는 api-manager의 프로세스 로컬 캐시(전역 값이라 레플리카별로 충분하며 api-manager `cachehandler`에는 범용 get/set이 없다)에 성공 결과 30초, 실패 결과 5초로 보관하고, `IsAgent`와 권한 검사는 캐시 앞에서 한다. status RPC timeout은 3~5초로 명시하며 어떤 오류에서도 `available=false`로 응답한다**(카드를 숨김). status 요청은 세마포어 대상이 아니며 모달을 열 때마다가 아니라 캐시 단위로만 워커를 쓴다. 키가 없으면 `available=false`이므로 카드가 숨겨진다.
- `ai_builder_model`(기본은 analysis 기본 모델), `ai_builder_reasoning_effort`(기본 `none`, 2.5의 비교로 확정), 4.3의 상한값들.
- 키는 analysis 엔진이 쓰는 키다(F4). 비어 있으면 호출이 동기 오류(`BUILDER_UNAVAILABLE`)로 즉시 돌아와 화면에 표시된다. 기동 시 키가 비어 있으면 정보 로그(info) 한 줄을, 키가 있는데 `ai_builder_model`이 base URL의 제공자와 맞지 않아 보이면(OpenAI 롤백 경로에서 Gemini 모델명 등) 경고 로그(warn)를 남긴다(`main.go` 수정, 7절). Builder는 `analysishandler`를 거치지 않고 엔진만 공유하므로 analysis의 모델 allow-set과 무관하다.
- `GET /ai_builder/status`: `{available, max_messages, max_message_chars}`. 에이전트 로그인이 아니거나 권한이 없거나 키가 없으면 `available=false`(카드 숨김). status도 `!a.IsAgent()` 규칙(F7)을 따른다.

### 4.7 오류 코드

| 상황 | 응답 | 프런트 동작 |
|---|---|---|
| 키 누락 | status `available=false`로 카드 숨김. 직접 호출하면 503 `BUILDER_UNAVAILABLE` | "AI 만들기를 사용할 수 없습니다"(셀프호스팅 운영자 안내는 문서) |
| 일일 상한 | 429 `BUILDER_DAILY_LIMIT` | 고정 안내와 support@voipbin.net |
| 동시 호출 상한 | 429 `BUILDER_BUSY` | "잠시 후 다시 시도하세요"(자동 재시도 없음) |
| status RPC 실패 | 200 `available=false`(서버가 확정, 실패 결과 캐시 5초) | 카드 숨김 |
| 입력 본문 과대 | 400 `BUILDER_INPUT_TOO_LARGE` | "새 대화"를 안내(상한 근접 UX와 동일) |
| breaker open | 503 `SERVICE_UNAVAILABLE` | "잠시 후 다시 시도하세요" |
| Redis 오류(fail-closed) | 503 `BUILDER_UNAVAILABLE` | "AI 만들기를 사용할 수 없습니다" |
| 입력 위반 | 400 `INVALID_ARGUMENT` | 길이 초과 시 "새 대화" 유도 |
| LLM 응답 불량 | 503 `BUILDER_RESPONSE_INVALID` | "다시 시도"(같은 요청 재전송) |
| LLM 지연 | 503 `BUILDER_TIMEOUT` | "다시 시도", 연속 2회면 대기 안내 |
| 비에이전트(chat 라우트 한정) | 403 | 카드 숨김. status 라우트는 비에이전트와 권한 없음을 200 `available=false`로 통일한다 |

reason 코드로 분기하고 문자열 매칭을 하지 않는다. 상태는 F18의 범위(400, 429, 503, 500) 안에서만 쓰며 429 두 종류와 503 reason들의 변환을 `error_translate.go` 테스트로 고정한다.

### 4.8 계측

Prometheus: 요청 수(결과별: ok, daily_limit, busy, unavailable, invalid_response, llm_error, invalid_argument, internal), 지연 히스토그램, `builder_tokens_total{kind="prompt"|"completion"}` 합계(F12. 구현에서 결과 라벨 2종과 `kind` 라벨 이름이 설계와 달라졌고 운영 문서가 구현을 따른다). 본문과 customer id는 라벨에 넣지 않는다. (Builder 라우트는 `processRequest` 앞 분기라 기존 `promReceivedRequestProcessTime`을 건너뛰므로 위 지연 히스토그램이 이를 대신한다.) 도구 제거 수와 경고 수는 이번 범위에서 계측하지 않는다.

## 5. 프런트 (square-admin, 별도 PR)

**Builder는 템플릿 다이얼로그 안의 한 화면이다**(라우터 state를 쓰지 않는다). 이유: 다이얼로그가 이미 `templateDialogOpen`으로 열려 있고 `handleTemplateSelect`가 폼 상태 setter를 직접 호출하므로, Builder의 "이대로 만들기"도 같은 setter를 호출하고 다이얼로그를 닫으면 된다(F21). 라우터 state 왕복, 모달 재개방 문제가 사라진다.

- 템플릿 그리드에 "Build with AI" 카드. `GET /ai_builder/status`가 `available=true`일 때만 표시한다. 404, 실패, 권한 없음이면 숨긴다. 카드를 누르면 다이얼로그가 Builder 화면으로 바뀌고 "템플릿으로 돌아가기"가 있다.
- **다이얼로그 구성(코드 확인)**: `DialogContent`는 `sm:max-w-[500px]`이므로 Builder 화면에서는 폭을 키운다(채팅, 5K자 초안, 경고, 확인을 담기 위해). `onOpenChange`로 바깥 클릭과 Esc에서 닫히므로 **요청 진행 중에는 닫기를 막고 확인을 받는다**(취소 시 이미 차감된 횟수와 응답이 사라진다). Builder 상태(`messages`, `current_draft`)는 `DialogContent` 하위가 아니라 `ais_create` 수준에 둬 닫아도 유지한다. `ref_detail`은 다이얼로그 아래의 폼에 항상 마운트되어 유효함을 코드로 확인했다(`ais_create.js`의 폼 영역). Builder 결과로 폼을 채울 때 `setSelectedTemplate(null)`로 템플릿 강조를 해제한다. 로그아웃 시 `sessionStorage`의 Builder 대화를 삭제한다.
- 채팅 패널: 메시지 목록, 입력, 전송. 전송 중 입력 잠금과 입력 중 표시, 초안 턴에는 "초안 작성 중" 문구. `message`는 평문으로 렌더링한다.
- 초안 미리보기: `draft`가 있으면 이름, 설명, 도구(라벨), `init_prompt`를 표시한다. `assumptions`와 `draft_warnings`를 눈에 띄게 표시하고 "초안은 검토 후 사용하세요. AI가 지원되지 않는 기능을 가정했을 수 있습니다" 안내를 둔다.
- **"지금 초안 만들기" 버튼**은 UI 언어에 맞는 고정 문구(2.3의 고정 문구, 한국어와 영어 2종)를 보낸다. **사용자 메시지가 한 번도 없으면 비활성화**하고, 사용자 답이 6번 이상 오가면 강조한다. 서버는 특별 취급하지 않는다.
- **"이대로 만들기"** 가 다음을 한다(F21): `setName(draft.name)`, `ref_detail.current.value = draft.detail`(폼이 다이얼로그 아래에 마운트되어 있어 ref가 유효한지는 구현 단계에서 확인하고 아니면 `defaultValue`와 key로 대체), `setInitPrompt(draft.init_prompt)` 뒤에 `toolsSection(draft.tool_names)`를 붙임, `setAiType('normal')`, `setEngineModel(...)`(`PROMPT_TEMPLATES`의 normal 템플릿 기본값을 읽어 쓰며 하드코딩하지 않는다), `setCustomModelInput('')`, `setSelectedTools(폼 선택 목록에 있는 도구만)`, `setEnableAllTools(false)`, 다이얼로그 닫기. TTS, STT, 언어는 폼 기본값을 그대로 둔다(Builder는 채널을 정보로 가지고 있지만 v1은 설정까지 채우지 않는다. 10절). 폼의 선택 목록에 없는 도구는 무시하고 경고한다. **이를 위해 `prompt_templates.js`가 `toolsSection`과 `TOOL_LABELS`를 export하도록 변경한다**(현재 `const`이며 export 안 됨, F21). 폼에 사용자 편집이 있으면 덮어쓰기 확인을 묻는다. **`assumptions`가 남아 있으면 "가정한 항목이 있습니다. 확인하셨습니까?"를 체크하도록 한 번 묻는다**(저장은 폼에서 사용자가 한다, 자동 저장 금지).
- 이력 보관: 컴포넌트 상태와 `sessionStorage`(탭 단위)에 `{messages, current_draft}`를 저장해 새로고침과 폼 이동 후 복귀에서 복원한다. "새 대화" 버튼으로 지운다. 복원 시 "이전 대화를 이어서 표시합니다"를 보여 준다. **AI 생성이 성공하면(폼 저장 성공) 저장된 대화를 지운다.** 서버가 보관하지 않으므로 다른 브라우저, 기기에서는 복원되지 않는다(수용).
- 전송 규칙: 한 번에 하나의 요청. 실패하면 마지막 사용자 메시지는 유지하고 "다시 시도(횟수가 차감될 수 있음)"로 같은 요청을 재전송한다. `AbortController`로 패널을 닫을 때 취소한다. 클라이언트 timeout 65초(4.3).
- 상한 근접(메시지 수 또는 합산 rune 90%): "지금까지로 초안 만들기"를 우선 제안한다. 새 대화를 선택하면 **새 대화는 빈 이력과 현재 `current_draft`로 시작**하고 첫 사용자 메시지는 고정 문구(한국어 "이전 초안을 이어서 다듬고 싶습니다."와 영어 "I would like to continue refining the previous draft."를 UI 언어에 맞춰 보내며 프롬프트의 "같은 뜻" 규칙에 의존하므로 골든 파일에 함께 둔다)다(이전 요약 메시지를 시드하지 않는다. 요약이 이력에 없을 수 있고 모델이 사용자 지시로 취급할 수 있다).
- 외부 LLM 전송 고지(4.5)를 첫 화면과 복원 화면에 상시 표시한다.
- 스트리밍과 선택지 칩은 v1에 없다(10절).

## 6. 위협 모델

| 위협 | 대응 |
|---|---|
| 플랫폼을 무료 LLM 프록시로 남용 | 에이전트 로그인(`IsAgent`), CustomerAdmin 또는 CustomerManager, 고정 시스템 프롬프트, 입력 합산 상한과 본문 바이트 상한, 출력 토큰 상한, 고객당 일일 상한 |
| Builder 동시 호출이 ai-manager RPC 워커 10개를 점유해 통화 중 도구 실행이 지연됨(F19) | 프로세스당 동시 호출 3개 세마포어, 초과 시 즉시 429 `BUILDER_BUSY`. 일일 상한과 별개의 통제 |
| accesskey, delegate, direct 토큰으로 호출(비용 노출 면적) | servicehandler 첫머리에서 `!a.IsAgent()` 거부(F7). 테스트에 accesskey, delegate, direct 케이스 명시 |
| 클라이언트가 조작한 이력, current_draft | 이력은 사용자 자신의 응답에만 영향. `current_draft`는 데이터 블록으로 구분. 서버는 이력을 신뢰한 권한, 상태 변경을 하지 않음. 비용은 상한이 묶음 |
| 프롬프트 주입 | 출력은 폼에 채워지는 초안뿐이고 사용자가 검토 후 저장한다. 서버가 도구 목록, 길이를 검증. 단 초안의 `init_prompt`는 저장 후 실제 통화 AI를 구동하므로 사용자의 검토가 유일한 방어선이다(`assumptions` 확인, 경고 안내) |
| 초안의 위험 도구 포함 | 허용 목록과의 교집합. `create_call` 등 제외. 허용 목록 밖 도구명 문자열은 경고 |
| 대화 내용 서버 노출 | 저장하지 않음. 로그에 본문 금지(4.5) |
| 계정을 여러 개 만드는 쿼터 우회 | 쿼터가 고객 단위라 플랫폼 전체의 가입 남용과 같은 위협이다. Builder 고유 대응은 없고 가입 단계 통제(이메일 인증)에 의존한다. 전역 일일 총량 상한은 활성화 전 결정(11절) |
| LLM이 도구가 없는 기능을 약속하는 초안 | 시스템 프롬프트 규칙과 시나리오 11의 평가. 코드로 보증할 수 없고 사용자가 검토 |

## 7. 구현 순서와 PR 구성

저장소 분리로 PR은 둘이다(monorepo, monorepo-javascript).

monorepo(백엔드, 한 PR). **프롬프트와 평가를 먼저 한다**(인프라를 만든 뒤 프롬프트가 안 통하면 낭비다):
1. **평가 하네스, 시스템 프롬프트, 사용자 시뮬레이터, 응답 파서와 검증기**(2.5, 4.4): 파서와 검증기는 순수 함수라 서버 인프라 없이 만든다(자동 판정이 이 둘에 의존한다). builder 패키지의 프롬프트 조립, `SendOnce`, 파서만으로 시작한다. 시나리오를 돌려 프롬프트를 반복하고 비교 평가(2.5의 2축)로 기본값을 정한다. 이 단계에서 첫 초안 턴과 수정 턴 지연, `max_tokens`, 세마포어 값을 실측한다. **2.5의 통과 조건을 만족하지 못하면 이후 단계로 가지 않고 3회 반복 후 대표님께 보고한다.**
2. 구현 확인(결과를 문서에 기록): `ai_ais.init_prompt` 컬럼 길이, 트레이스(otel, sentry) 유무, api-manager의 인증 이후 미들웨어와 에러 미들웨어의 본문 로깅, Loki 수집 소스, `CustomerRateLimit` tier 값, 앞단 LB와 ingress의 요청 timeout.
3. ai-manager: `builderhandler`(세마포어, 호출 순서 4.3), 설정, `BuilderChatCountIncr`, `processRequest` 앞의 Builder 라우트(본문 로그 금지), 메트릭, `main.go`의 키와 모델 정합성 경고, status RPC 처리.
4. common-handler: `AIV1BuilderChat`(명시 timeout 55초), `AIV1BuilderStatus`, **`rabbitmqhandler/consume.go`의 세 에러 문자열에서 본문 제거**(4.5). vendor는 커밋하지 않으므로 갱신 작업은 없다(4.5의 vendor 정정).
5. api-manager: `POST /ai_builder/chat`(Builder 라우트에서 RPC timeout을 `BUILDER_TIMEOUT` 503으로, `ErrCircuitOpen`을 503으로 변환하고 두 카운터를 둠, status 프로세스 로컬 캐시(만료 시 갱신은 `singleflight` 또는 뮤텍스 안에서 하나만 수행)와 실패 시 `available=false`), `GET /ai_builder/status`, `MaxBytesReader`, `!IsAgent` 거부, OpenAPI, servicehandler.
6. docsdev: AI 생성 문서, 셀프호스팅 설정(엔진 키), 배포 후 smoke 대화 1회.

monorepo-javascript(프런트): OpenAPI 동기화(`voipbin-openapi-sync-to-js`), `prompt_templates.js`의 export 추가(`ais_create.test.js`의 `jest.mock('../prompt_templates', ...)`에 `toolsSection`과 `TOOL_LABELS`를 추가하고 `src/provider` mock의 status 조회 undefined를 처리), 템플릿 다이얼로그의 Builder 화면, 미리보기, 폼 채우기, 테스트. 백엔드 배포 후에 노출한다(status가 `available=true`일 때만 카드).

배포: 백엔드, 프런트, 내부 검증. 켜기용 설정이 없어 키가 있는 환경에서는 백엔드가 올라가는 즉시 API로 사용 가능하다(플랫폼 비용 발생, 고객당 일일 상한만 적용).

## 8. 테스트

- ai-manager: 카탈로그 키와 허용 목록의 집합 일치(두 곳에 하드코딩된 6종이 서로 같음), 응답 파싱(정상, 코드펜스, 앞뒤 텍스트, 설명 텍스트에 `{}`, 비JSON, 잘림, 빈 message, 타입 불일치 필드, 알 수 없는 필드), 도구 교집합과 `draft_warnings`, 허용 목록 밖 도구명 경고(단어 경계, `stop_service`와 `stop_flow`), `## Tools & Capabilities` 섹션 제거, 길이 절단, 입력 검증(메시지 수, 길이, 합산 rune, 마지막 role), 일일 카운터(첫 INCR과 TTL 없는 키에서 TTL, 상한 경계, Redis 불가 시 fail-closed 503, 검증 실패 미카운트, `BUILDER_RESPONSE_INVALID`도 카운트), **동시 호출 세마포어(4번째 호출 즉시 429, 완료 후 해제, 오류 경로에서도 해제)**, 킬 스위치, 키 누락, `current_draft`가 JSON 직렬화 문자열로 들어감, **`processRequest` 전체를 통과하는 오류 유발(LLM 오류, 파싱 실패, 검증 실패, 언마샬 실패)과 panic 후 로그 캡처에 사용자 입력 문자열이 없음**, 처리 함수가 err를 반환하지 않음, **`draft`의 필수 필드 누락 케이스(`init_prompt` 없음, 타입 불일치, 섹션 제거 후 빈 `init_prompt`)와 `draft`가 버려질 때 `assumptions`도 함께 사라짐, `checkpoint` 계산(6, 10, 14, 18만 true, 7~9와 11~13은 false), api-manager의 RPC timeout이 `BUILDER_TIMEOUT`(503)으로, `ErrCircuitOpen`이 503으로 변환됨**, **BUSY, DAILY_LIMIT, TIMEOUT, RESPONSE_INVALID, UNAVAILABLE 각각이 ai-manager의 `sock.Response`에서 api-manager 쪽의 같은 reason으로 복원됨(왕복)**, `user_turns`, `draft_exists`, `checkpoint`가 코드가 센 값으로 주입됨, status 오류 시 `available=false`와 캐시, **reply publish 실패와 marshal 실패를 유도했을 때 로그에 응답 본문이 없음**(common-handler), **세마포어와 카운터 순서**(BUSY는 미카운트, 일일 초과 시 세마포어 해제), 파서의 `message` 비어 있는 선행 객체 건너뜀, `## Tools & Capabilities` 변형 제거(대소문자, `###`, 중복, 문서 끝), status RPC(꺼짐, 키 없음, 정상).
- api-manager: 비에이전트(accesskey, delegate, direct) 거부, 비권한 거부, status, `MaxBytesReader`가 `BindJSON` 전에 적용되고 초과가 구분 매핑됨, 입력 검증, 오류 매핑(429 변환), 서버 핸들러 로그 캡처에 본문 없음, RPC mock.
- 프런트: 폼 채우기(`prompt_templates.js` export, `toolsSection` 부착, 알 수 없는 도구 무시, 덮어쓰기 확인, `assumptions` 확인, `engineModel`과 `aiType` 기본값, `detail` ref), 초안 미리보기와 경고, sessionStorage 복원, AI 생성 성공 시 대화 삭제, 새 대화(빈 이력과 `current_draft`), 실패 후 재시도, 취소, 상한 근접 안내, status 404 시 카드 숨김, 고정 문구 버튼. **허용 6종이 `TOOL_LABELS`에 모두 있는지**(프런트 쪽 하드코딩 6종 사본으로 검사). **저장소가 달라 서버 카탈로그와 프런트의 자동 대조는 불가능하다.** 두 곳의 하드코딩 6종과 참조 헤더 골격(2.6)이 같은지는 두 PR의 **수동 체크리스트**로 확인한다.
- **적응성 평가 스위트(2.5)**: 실제 모델, 수용 기준(7절 1단계).

## 9. 이전 구조 대비 삭제되는 것

aicall과 AI 행 생성, activeflow, 세션 유일 제약, 슬롯 해소, 세션 재개, 차단 인프라(A1~A5, B1~B4), 신뢰 모델(G1~G3)과 webhook 발행 중앙화, `messagehandler.Create` 조회 비용과 일반 고객 webhook 누락 위험, ProcessTerminate와 activeflow 정리, pipecat vendor 갱신과 재시작, 폴링 상태 기계와 120초 타임아웃 UX, 키 누락 무신호, `turns_used` 서버 집계, 복원을 위한 서버 이력.

새로 생기는 것: 입력 합산 상한과 본문 바이트 상한, 새 Lua 카운터, 본문 로그 금지 처리, 프런트의 이력 보관, 평가 스위트.

## 10. 도입 트리거 (지금은 만들지 않는 것)

(재방문 조건이 구체적인 항목만 표에 둔다. 표에 없는 확장은 필요가 확인될 때 별도 설계한다.)

| 항목 | 도입 조건 |
|---|---|
| `suggested_replies`(선택지 칩), `captured`, `open_topics` 필드와 패널 | 2.5의 비교 평가에서 질문 품질 기여가 확인될 때(칩은 앵커링 위험과 비교) |
| 별도 큐와 breaker 제외(4.3) | api-manager의 Builder timeout 카운터 또는 `ErrCircuitOpen`이 관측될 때 |
| 응답 완료 후 TTS, STT, 언어 같은 폼 설정까지 채우기(v1은 미리보기에 통화 언어와 설정이 안 맞을 수 있다는 경고만 둔다) | 채널이 음성인 초안에서 설정 누락 문의가 확인될 때 |
| 도구 제거 수와 경고 수 계측 | 경고 빈도를 알아야 할 필요가 확인될 때 |
| 서버 1회 재시도(응답 불량 시) | JSON 불량 비율이 5%를 넘을 때(2.5) |
| 응답 스트리밍 | 지연에 대한 사용자 불만이 확인될 때 |
| 서버 측 대화 이력 보관(기기 간 복원, 분석) | 기기 간 이어하기 요구나 대화 품질 분석 필요가 확인될 때 |
| 플랫폼 전역 예산 카운터와 서킷브레이커 | 토큰 계측(F12)에서 Builder 비중이 문제로 관측될 때(전역 일일 총량 상한을 활성화 전에 둘지는 11절) |
| 평가의 자동 LLM 판정 | 시스템 프롬프트를 자주 바꾸게 되어 수동 판정이 부담될 때 |
| 초안 수정을 섹션 단위 diff로 요청 | 초안 수정 턴의 지연이나 비용이 문제로 관측될 때 |
| 기존 AI 편집 모드, insight 타입 초안, 음성 Builder | 요구가 확인될 때 |
| 초안 도구 허용 목록 확장(create_call 등) | 사용자가 제외된 도구를 요구하는 신호가 확인될 때 |
| 저장 전환율 측정(프런트 이벤트) | 성공 지표와 철수 기준이 필요해질 때 |

## 11. 열린 질문 (대표님 결정이 필요한 것)

| 질문 | 추천 |
|---|---|
| 인터뷰 깊이: 숫자 목표 없이 "충분하면 멈춘다"(현재안) vs 4~8턴 목표 | 현재안. 평가의 비교 결과를 보고 조정해 보고 |
| 활성화 전에 최소한의 전역 일일 총량 상한을 둘지 | 신규 계정 연쇄 가입이 비용 폭탄이 되는 유일한 경로이고 비용이 작으므로 도입을 권한다(보류를 택하시면 10절 트리거로 관리) |
| 호스팅 서비스에서 공개하는 시점과 범위(켜기용 설정 없음, 배포가 곧 공개) | 평가 통과 후 배포하는 것을 권한다 |
| 평가가 3회 반복해도 통과하지 못할 때 기능을 접을지 정적 템플릿 보강으로 대체할지 | 대표님 결정. 접거나 템플릿 보강(적응성 요건이 달성되지 않으면 기능 가치가 없다) |
| 평가 예산(약 38회 실행, 빌더 호출 약 225회와 시뮬레이터 호출, 사람 판정 전문 약 21건과 훑어보기 약 15건)과 통과 기준(2.5) | 현재안 승인 요청. 비용은 소액이며 판정자는 대표님 또는 지정인 |
| 성공 지표 | v1은 서버가 이미 가진 일일 요청 수, 오류 비율, 토큰 합계(4.8)만 본다. 폼 채우기 비율은 프런트 이벤트가 필요하므로 10절 트리거로 미루고, 임계값은 첫 2주 관찰 후 정한다 |
| 일일 요청 상한 초기값(200), 동시 호출 상한(프로세스당 3) | 보수적으로 시작, 토큰 계측을 보며 조정 |
| 기본 모델, `reasoning_effort`, 프롬프트 전체판 vs 축소판 | 2.5의 비교 평가 결과로 7절 1단계에서 결정해 보고 |
| `assumptions`가 남은 초안의 폼 채우기 허용 | 허용하되 확인 체크를 요구(5절) |
| 외부 LLM 전송 고지 문구와 약관, DPA 표기 | 첫 화면 한 줄 고지. 서버 비보관 사실을 약관에 명시할지 검토 |

## 12. 변경 이력

상세는 git 이력과 리뷰 기록에 맡기고 회차별 핵심만 남긴다.

- v1: (e) 구조(상태 없는 동기 멀티턴)로 작성. 이전 aicall 구조는 `...-v7-aicall-superseded.md`로 보존.
- v2(1회차 A, B REQUEST_CHANGES): `!IsAgent()` 거부(accesskey와 delegate 통과 확인), 카운터 Lua의 영구 락 결함 수정, rune 기반 상한과 `MaxBytesReader`, `processRequest` 로그 유출 방어, 적응성(규칙 충돌, 질문 목록화, 확인 게이트, 깊이 상한, few-shot, 평가 판정 가능성) 보강.
- v3(2회차 A, B REQUEST_CHANGES): ai-manager RPC 워커 10개 직렬 문제(F19)로 동시 호출 세마포어, `Sanitize` 사용 중단, 응답 파서 명세, 규칙 9개에서 6개, 신호 표와 `suggested_replies` 축소, 프런트 export와 다이얼로그 내부 화면, 평가를 구현 1단계로 이동, 사용자 시뮬레이터 도입.
- v4(3회차 A, B REQUEST_CHANGES): 502, 504, 413을 만들 수 없음(F18)을 확인해 오류 표를 400, 429, 503, 500과 reason으로 정정, 응답 본문이 `consume.go` 에러 로그로 새는 경로 확인(F17 정정), 우선순위를 `깊이 파기 > 이해 확인`으로 바꾸고 신호 표를 "실패 지점 탐색" 원칙으로 교체, 통과 조건을 숫자로.
- v5(4회차 A, B REQUEST_CHANGES): breaker 오염 방어(deadline 40, 55, 65초와 세마포어), status 캐시, `user_turns`와 `draft_exists`를 코드가 주입, `reasoning_effort` 비교 필수, 평가 축소와 철수 기준을 11절에.
- v6(5회차 A REQUEST_CHANGES, B APPROVE): A는 breaker 오염의 실제 신호가 api-manager의 RPC timeout이므로 변환과 카운터와 트리거를 그쪽으로 재정의(코드 확인), 취소도 실패로 셈, vendor 갱신이 소비 동작 변화를 동반하고 api-manager vendor도 필요함을 명시, status 캐시를 프로세스 로컬과 timeout과 실패 TTL로 확정, 4.2 문장과 블록 위치 확정, 문서 잔재 정리. B는 안내 규칙을 `user_turns>=6`과 `draft_exists` 조합으로 정리, 평가 수치를 실제 건수로 정정, 시나리오 15를 합성 이력으로 바꿈, 성공 지표를 서버가 가진 값으로 한정, `종료 판단`과 (나) 예외 서술 정직화, `assumptions` 누락 처리, 문서 압축(12절).
- v7(6회차 A, B 모두 APPROVE, Medium과 Low를 문서 수정으로 반영): `checkpoint`를 코드가 계산해 주입하고 규칙 3의 안내를 그 값에만 반응하도록(모델이 턴을 세지 않음, `draft_exists`와의 중복 안내 해소), 규칙 3의 "짧은 답"을 정보량 기준으로(정보가 있는 단답은 계속 좁힘, B2 판정에 추가), 평가 수치를 시나리오별로 다시 세어 정정(약 29회 실행, 전문 21건), 시나리오 3과 B1, B2의 반복 확대, 시나리오 6의 도메인 배분, `draft` 필수 필드 검증, 4.7 표에 신규 reason 추가와 status 실패 캐시 5초 일치, ai-manager vendor 갱신이 이벤트 구독 큐의 재시도도 바꾼다는 점과 갱신 범위, 카운터에서 취소 제외, 새 대화 시드 영어 문구, 문서 참조 정정.
- v8(7회차 A, B 모두 APPROVE, 설계 변경 없는 문서 수정): `checkpoint`가 상한까지 4턴마다 반복됨을 정정하고 요약 턴에는 안내 생략, 역할 교대 비강제 명시, 4.4의 `draft` 폐기 연쇄 규칙과 검사 순서와 프런트 안내, status 비에이전트 처리 통일, vendor 갱신 영향 검증(이벤트 핸들러 멱등성, delay exchange, `Qos`가 모든 소비 큐에 적용)과 영향이 크면 잔여 경로로 기록하고 분리하는 대안, 시나리오 14의 위임 경계 변형과 3턴 상한 판정 항목과 1회 시나리오 재실행 규칙, 평가 수치를 다시 셈(약 38회 실행).
- v8 정오 사항(구현 계획 1회차): 4.5와 7절 4단계의 "vendor 갱신 필수와 소비 동작 변화" 서술은 이 저장소의 vendor 비커밋 정책과 맞지 않아 위 정정으로 대체한다. 설계의 나머지는 변경이 없다.

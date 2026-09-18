# 설계: AI Summary 출력 언어 강제 (1차: 첫 생성 강제 + 2차: 검증 하네스) — VOIP-1532 후속

작성일: 2026-09-17
작성자: Hermes (CPO)
브랜치: VOIP-1532-separate-stt-and-summary-output-language (PR #1323에 흡수)
상태: Draft (설계 리뷰 대기)
결정(대표):
- (재정렬 2026-09-18) **주력은 "첫 생성 강제 강화"**: 언어 지시를 system 메시지로 분리하고
  출력 언어값을 지시문에 직접 주입해, 첫 생성에서 언어를 지키게 한다. GPT-4-turbo는 system
  지시를 강하게 따르므로 첫 생성 대부분이 준수 → 재시도가 거의 발동하지 않는다.
- **검증 하네스는 그 위의 얇은 안전망**: 강한 첫 생성으로도 LLM은 확률적이라 100% 보장은 불가.
  드물게 어길 때만 사후 검증(별도 LLM yes/no)으로 잡아 재생성. 첫 생성 강제 덕에 재시도가
  거의 안 일어나므로 검증 2배 호출 비용 우려가 대폭 완화된다.
- 재시도 상한: 최대 2회 재생성(총 3회 시도) 후 마지막(비어있지 않은) 결과 반환(빈 요약보다
  낫다, 실패는 로그).
- 이번엔 이 두 가지(첫 생성 강제 + 안전망 하네스)만. dedup 원인은 별도(배포 후 대표가 이번
  recording으로 재확인).

## 1. 문제

현재 출력 언어 강제는 프롬프트 안의 자연어 지시 한 줄이 전부다(main.go:132, "Generate the
summary in the language specified by the output_language field..."). LLM(GPT-4-turbo)에 보내는
것은 **system 메시지 없이 단일 user 메시지 하나**이며, 언어 지시가 그 user 메시지 JSON의 prompt
필드 안에 묻혀 있고 output_language를 **self-reference로 참조**한다(같은 JSON의 다른 필드를
보라는 간접 지시). 이 강제력은 약하며, LLM이 지시를 무시하고 transcript 언어(예: 영어)로 요약을
쓰는 경우가 발생한다.

두 겹으로 강제한다(주력 + 안전망):
1. **1차(주력) — 첫 생성 강제 강화**: 언어 지시를 별도 system 메시지로 올리고, 출력 언어값을
   지시문에 직접 문자열로 박아 넣는다(self-reference 제거). 근본 원인(약한 지시 위치/간접 참조)을
   직접 없앤다. 대부분의 요약이 첫 생성에서 언어를 지킨다.
2. **2차(안전망) — 검증 하네스**: 그래도 드물게 어기는 경우를 위해, 생성 결과를 별도 LLM으로
   재검증(yes/no)하고 불일치면 재생성. 통과/재시도 분기는 코드가 결정론적으로 판정(대표 원칙:
   코드가 사실 확정, LLM은 판정 재료 제공). 강한 1차 덕에 이 경로는 거의 발동하지 않는다.

## 2. 목표 (numbered, testable)

1. (1차) 언어 지시를 system 메시지로 분리하고 출력 언어값을 지시문에 직접 주입해 첫 생성에서
   언어를 강제한다. output_language self-reference 의존을 제거한다.
2. (2차) 요약 생성 후, 결과가 요청한 출력 언어로 쓰였는지 LLM으로 검증한다.
3. 일치하면 그대로 통과. 불일치하면 재생성한다.
4. 재시도 상한: 최대 2회 재생성(총 3회 생성 시도). 상한 도달 시 마지막 비어있지 않은 생성
   결과를 반환하고, 검증 실패를 로그로 남긴다(빈 요약/에러 반환 금지).
5. 검증 LLM의 출력은 결정론적으로 파싱 가능한 최소 형태(yes/no)로 강제하며, 통과/재시도 분기는
   코드가 판정한다.
6. 4개 요약 경로(call/conference/recording/transcribe) 모두에 자동 적용된다(공통 지점 삽입).

## 3. Non-goals (명시적 스코프 제외)

- **dedup 캐시 문제**(Start의 GetByCustomerIDAndReferenceIDAndLanguage가 기존 요약을 그대로
  반환 → 새 생성 로직 미실행)는 이 설계에서 다루지 않는다. 이번 증상(ko-KR인데 영어)의 직접
  원인이 dedup일 수 있으나, 대표 지시로 이번엔 언어 강제(1차+2차)만 구현하고 배포 후 이번
  recording으로 재확인한다. dedup이 원인이면 이 강제 로직은 이 recording에서 작동 기회조차
  없으므로, 배포가 이번 증상을 즉시 고친다고 오버클레임하지 않는다.
- 번역 전용 경로(재시도는 "재생성"이며, 기존 transcript+outputLanguage로 다시 생성한다).
- STT 언어/재사용 로직 변경(VOIP-1532 본체에서 이미 처리).

## 4. 영향 파일

| 파일 | 이유 |
|------|------|
| pkg/summaryhandler/content.go | (1차) 생성 요청을 [system,user] 2메시지로 조립(system에 outputLanguage 직접 주입). (2차) contentGet에 생성→검증→재시도 루프, verifyOutputLanguage/shouldVerify/canonPrimarySubtag/룬 안전 절단/최소 산문 가드 신설. |
| pkg/summaryhandler/main.go | languageSystemPromptFmt(1차 system 지시) 신설, Language 섹션을 "Follow the system message" 로 축약, 검증 프롬프트/재시도/샘플/타임아웃/모델/영어 subtag 상수 추가. |
| pkg/engine_openai_handler/main.go, send_once.go(신설), mock_main.go | SendOnce(ctx, req) 신설(backoff 없이 client.CreateChatCompletion 1회). 인터페이스+mock 갱신(go generate). R3-치명/R4-2: 검증 호출에 진짜 10s 상한을 두려면 backoff 우회 필요. |
| pkg/summaryhandler/content_test.go | 검증 통과/재시도/상한/빈결과보존/폴백/최소산문/영어스킵/1차강제 케이스 추가. **contentGet은 정규화를 하지 않으므로**(정규화는 상위 Start에서만) fixture의 outputLanguage 값이 그대로 shouldVerify를 결정한다. 기존 fixture 실제 언어: call=en-US(스킵), conference=ko-KR(검증), recording=ko-KR(검증), transcribe=en-US(스킵), none=""(빈 값). **ko-KR 2건(conference 라인 104, recording 라인 134)은 shouldVerify=true라 SendOnce(yes) EXPECT 추가 필요**. 모든 Send 단언을 [system,user] 2메시지로 갱신. |

## 5. 구현 설계

### 5.0 1차: 첫 생성 강제 강화 (주력) — system 메시지 분리 + 언어값 직접 주입

현재 생성 요청(content.go:208~)은 user 메시지 1개에 RequestContent JSON을 통째로 넣는다.
언어 지시가 그 JSON의 prompt 필드 안에 묻히고 output_language를 self-reference로 참조해 약하다.
이를 다음으로 바꾼다:

- **system 메시지 추가**: ChatCompletionRequest.Messages를 [system, user] 2개로 구성.
  system 메시지에 출력 언어를 **값으로 직접 박은** 강한 지시를 둔다:
  ```
  languageSystemPromptFmt = "You are a call summary assistant. You MUST write the ENTIRE summary,
  including every section body, in %s (BCP47). This language requirement is absolute and overrides
  the language of the transcript or any other input. Section labels follow the user instructions."
  ```
  실제 전달 시 %s에 outputLanguage(정규화된 값, 예 "ko-KR")를 fmt.Sprintf로 주입 →
  self-reference 제거(모델이 다른 필드를 찾을 필요 없이 지시문 자체에 언어가 있음).
- **user 메시지**: 기존 RequestContent JSON 유지(prompt/reference_type/output_language/
  transcripts/variables). output_language 필드는 남겨 두어도 무방(system과 일치). 단 main.go
  프롬프트의 Language 섹션은 system과 모순되지 않게 "Follow the system message's language
  requirement." 로 축약(자연어 self-reference 지시 제거).
- 근거: OpenAI Chat 모델은 system 역할 지시를 user 본문 안 지시보다 강하게 따른다. 언어값을
  지시문에 직접 넣으면 "어느 필드를 보라"는 간접 참조 실패가 사라진다. 이 2가지가 첫 생성
  준수율을 크게 올려 2차 재시도가 거의 발동하지 않게 한다.
- 적용 지점: §5.1 contentGet 내 요청 조립부(생성 시도마다 동일하게 system+user). 4경로 공통.
- outputLanguage가 en(영어)이어도 system 지시는 무해(영어로 쓰라는 지시). 즉 1차는 언어 무관
  하게 항상 적용. (2차 검증만 en을 스킵 — §5.1.1.)
- **빈 outputLanguage 방어(R6-3)**: contentGet은 상위 Start의 정규화를 거치지 않고 호출될 수
  있다(예: content_test.go "none" 케이스가 ""를 직접 전달, 또 방어적으로 프로덕션 외 경로).
  outputLanguage가 ""이면 system 지시가 "write ... in  (BCP47)"로 malformed가 되므로,
  **contentGet 진입 시 effectiveLang = outputLanguage 비었으면 defaultOutputLanguage(en-US)**로
  로컬 확정한 뒤 이 값을 system 주입/shouldVerify/RequestContent.OutputLanguage에 일관 사용한다.
  이는 기존 main.go 프롬프트의 empty→...→en-US fallback 체인을 대체(1차 지시가 self-ref 대신
  effectiveLang 값을 직접 담으므로). Start 경로는 이미 en-US로 정규화돼 오므로 이 로컬 방어는
  중복이나 무해하며, contentGet 단독 호출(테스트/미래 경로)의 안전을 보장한다.

### 5.1 삽입 지점 (2차 하네스)

contentGet(content.go:175~)이 4개 경로의 유일 공통 지점이다(호출부 4곳 모두 여기로 수렴:
content.go:66 call, content.go:113 conference, start.go:241 transcribe, start.go:283 recording).
생성→검증→재시도 루프를 contentGet 내부에 두면 4경로 전부 자동 적용된다(call/conference
비동기 경로 포함). contentGet의 시그니처(입출력)는 바꾸지 않는다 — 반환은 여전히
(string, error). 내부 동작만 루프로 바뀐다.

### 5.1.1 검증 대상 언어 결정 (en 계열만 스킵) — R2-3/R3-중 반영

검증은 별도 LLM yes/no 판정이다(§5.3). LLM은 "fr-FR 요청인데 영어 출력" 같은 라틴 대 라틴
불일치도 구분할 수 있으므로, "라틴 타깃은 검증 무의미"라는 (초안의) 근거는 틀렸다. 국제
CPaaS에서 스페인어/프랑스어/포르투갈어 타깃이 영어로 나오는 것도 실제 결함이며 검증 대상이다.

따라서 스킵 대상은 **영어(en 계열)만**으로 한정한다:

```
shouldVerify(outputLanguage) =
    canonPrimarySubtag(outputLanguage) != "" AND canonPrimarySubtag(outputLanguage) != "en"
```

- canonPrimarySubtag: outputLanguage에서 primary subtag를 추출하고 **소문자로 canonicalize**한다
  (예: "ko-KR"->"ko", "KO-KR"->"ko", "EN"->"en"). R4-3: BCP47은 대소문자 비구분이므로 반드시
  소문자화 후 비교. 정규화가 대소문자를 건드리지 않으므로("KO-KR" 그대로 저장 가능) 이 canonical
  화가 없으면 "KO"가 검증에서 조용히 누락된다.
- en(영어)만 스킵하는 이유(비용 트레이드오프로 명시, "무의미"가 아님):
  - 가장 흔한 기본 경로가 outputLanguage=en-US(정규화 기본값)다. 원 증상은 "비영어 타깃인데
    영어가 나온다"이므로 타깃이 영어면 애초에 그 증상이 성립하지 않는다. en 요약을 "영어가
    맞나" 검증하는 것은 거의 항상 통과하면서 지배적 트래픽에 LLM 호출을 2배로 만든다.
  - 즉 en 스킵은 "지배 비용 케이스 회피"라는 **명시적 비용/스코프 트레이드오프**로 수용한다
    (R2-3이 요구한 것이 정확히 이것). 비영어 타깃(라틴이든 비-라틴이든)은 모두 검증한다.
- 오탐 위험(섹션 헤더가 영어라 라틴 비중이 높음)은 스킵으로 회피하는 것이 아니라 §5.3
  검증 프롬프트의 "섹션 헤더/라벨/고유명사 무시, 산문만 판정" + 최소 산문 가드(§5.3)로 완화한다.

### 5.2 생성→검증→재시도 루프 (contentGet 내부)

현재 contentGet은 requestContent 조립 → 1회 Send → 결과 반환이다. 이를 아래로 바꾼다:

```
requestContent 조립 (기존과 동일; 매 시도 동일 입력 사용)

// 검증 대상이 아니면(라틴 타깃/빈 값) 기존 동작: 1회 생성 후 반환
if not shouldVerify(outputLanguage):
    return generateOnce()   // 기존 단일 Send 경로 그대로

maxAttempts = 1 + maxRegenerations (= 3)
var lastNonEmpty string     // 마지막 "비어있지 않은" 결과만 보존 (R2-1)
for attempt in 1..maxAttempts:
    content, err = generateOnce()   // Send 1회
    if err != nil:
        if attempt == 1:
            return "", err          // 첫 시도 실패는 기존 동작대로 전파(요약 자체가 없음)
        // R4-1: 재시도(attempt>=2)의 Send 하드 실패는 이미 확보한 결과를 버리지 않는다.
        //       하네스가 추가한 재생성 호출의 일시 장애가 정상 요약을 하드 실패로 뒤집지 않도록
        //       best-effort로 lastNonEmpty 폴백.
        log.Warnf("regeneration send failed at attempt %d/%d, falling back to last good summary. err=%v", attempt, maxAttempts, err)
        break
    if content == "":
        // 빈 결과는 검증 의미 없음. 앞선 비어있지 않은 결과를 덮지 않는다.
        break
    lastNonEmpty = content
    ok = verifyOutputLanguage(ctx, content, outputLanguage)
    if ok:
        return content, nil
    log.Warnf("summary output language mismatch, attempt %d/%d, want %s", attempt, maxAttempts, outputLanguage)
// 상한 도달 / 빈 결과 / 재시도 Send 실패로 break: 확보된 마지막 비어있지 않은 결과 반환(없으면 "")
return lastNonEmpty, nil
```

- **R2-1 회귀 방지**: 빈 결과가 앞선 비어있지 않은 결과를 덮지 않도록 lastNonEmpty만 갱신.
  attempt1이 (불일치) 요약 생성 → attempt2가 빈 결과여도 attempt1 결과를 반환.
- **R4-1 Send 실패 best-effort**: attempt1 Send 실패는 기존대로 error 전파(요약 자체가 없으니).
  그러나 attempt>=2의 Send 하드 실패(send.go backoff 1분 소진 후 err)는 lastNonEmpty로 폴백해
  전체 요약을 실패시키지 않는다. 하네스가 추가한 재생성 호출의 일시 장애가, 하네스 도입 전이면
  정상 완료됐을 요약을 하드 실패로 뒤집는 것을 방지. 비-라틴 고객에게 신뢰도가 오히려
  나빠지지 않도록.
- 검증 불확정(§5.3)은 "통과 간주"라 해당 시도 결과를 즉시 반환 → 정상 요약을 버리지 않음.

### 5.3 verifyOutputLanguage 헬퍼 (신설)

```
func (h *summaryHandler) verifyOutputLanguage(ctx, content string, outputLanguage string) bool
```

- 별도 ChatCompletion 호출. system 없이 user 메시지 1개. 짧게 구성(비용 억제):
  - 프롬프트(languageVerifyPrompt): "You are a strict language detector. Answer with exactly one
    word: yes or no. Ignore section headings/labels, proper nouns, product names, URLs, code, and
    technical terms — judge only the natural-language prose. Is the following text written mainly
    in the language with BCP47 code %s? TEXT:\n%s"
    (섹션 헤더/라벨 무시를 명시 — R2-4 오탐 완화)
  - content 앞부분 languageVerifySampleLen(1500) **룬** 만 사용. **바이트 슬라이싱 금지**(R3-경):
    `[]rune(content)` 변환 후 `if len(r) > N { r = r[:N] }` 로 절단(멀티바이트 한글/일본어
    중간 잘림·out-of-range 패닉 방지).
- **최소 산문 가드(R4-4)**: 검증에 넘길 샘플에서 섹션 헤더 라인과 "- None" 항목을 제외한
  실제 산문 길이가 languageVerifyMinProse(예: 20 룬) 미만이면 **검증을 스킵하고 통과 간주**
  (true 반환). 근거: transcript가 비음성/무의미해 요약이 "헤더 + 여러 개의 - None"만인 경우
  판정할 산문이 사실상 없어, 검출기가 자신있게 no를 뱉어 3회 재시도를 낭비할 수 있다(§6의
  "명백한 불일치에만 재시도" 원칙과 배치). 산문이 거의 없으면 애초에 "언어"를 논할 수 없으므로
  통과로 둔다.
- **검증 요청은 저렴한 모델 + 진짜 짧은 상한**(R1-3/R2-6/R3-치명/R4-2):
  - Model: defaultVerifyModel(예: openai.GPT4oMini). 생성용 defaultModel(GPT4Turbo)과 분리.
    yes/no 판정에 최상위 모델 불필요.
  - **backoff 우회가 핵심**: 기존 engineOpenaiHandler.Send는 send.go에서
    cenkalti/backoff.Retry를 backoff.WithContext 없이 쓰고 MaxElapsedTime=1분이다. 따라서
    context.WithTimeout(ctx, 10s)로 감싸도 backoff 루프가 ctx-cancel 에러를 일반 에러로 보고
    1분까지 재시도한다 → 10초 상한이 서지 않는다(초안의 거짓 주장). 검증 경로는 Send를 쓰지
    않고, engineOpenaiHandler에 신설하는 **Verify 전용 메서드(백오프 없이 client.
    CreateChatCompletion을 1회만 호출)**를 verifyTimeout(10s) context로 호출한다. 판정 호출은
    fail-open이 안전하므로(실패=통과 간주) 정확성을 위한 재시도가 불필요하고, backoff 없이 1회만
    호출해 실제 10초 상한이 선다.
    - 구현: engine_openai_handler에 `SendOnce(ctx, req) (*resp, error)` 추가(backoff 없이
      client.CreateChatCompletion(ctx, req) 1회). 인터페이스+mock 갱신. verifyOutputLanguage는
      이 SendOnce를 verifyTimeout context로 호출.
  - ctx: context.WithTimeout(parentCtx, verifyTimeout)(=10s). 타임아웃/에러 시 아래 "통과
    간주"로 빠짐.
  - **fail-open 트레이드오프 명시(R6-2)**: SendOnce는 backoff가 없어 429(rate limit)/일시
    네트워크 오류가 즉시 "통과 간주"로 귀결된다. 그럼에도 **재시도를 추가하지 않는다**: (a) 검증
    실패가 통과로 처리돼도 정확성엔 안전(정상 요약을 버리지 않음), (b) 주력은 1차 강제이고 검증은
    드물게만 발동하는 안전망이라 일부 스킵이 치명적이지 않음, (c) 429 급증은 실측 신호가 아닌
    가정이므로 선제적 재시도(신규 config 표면)를 두는 것은 과설계다. 이 잔존 위험은 §6에 명시.
- 응답을 trim+lowercase 후 "yes"로 시작하면 true, "no"로 시작하면 false.
- 그 외(빈 응답, 예상 못한 출력, SendOnce 에러/타임아웃) → **true 반환(통과 간주)**. 검증
  불확정이 정상 요약을 버리지 않도록(보수적, 오탐 방지). 이 경우 Debugf 로그.
- 결정론성: 통과/재시도 분기는 이 bool을 코드가 판정. LLM은 yes/no 재료만 제공.

### 5.4 상수 (main.go)

- languageSystemPromptFmt = (위 §5.0 1차 system 지시 포맷; %s에 effectiveLang 주입) — **신설**
- defaultOutputLanguage: **start.go L22에 이미 선언됨("en-US") → 재사용(중복 선언 금지, R7)**.
  contentGet 빈 값 방어(§5.0)와 정규화가 같은 상수를 참조.
- maxSummaryRegenerations = 2 (재생성 횟수; 총 시도 = 3) — 신설
- languageVerifySampleLen = 1500 (룬 기준) — 신설
- languageVerifyMinProse = 20 (룬; 이 미만이면 검증 스킵-통과, R4-4) — 신설
- defaultVerifyModel = openai.GPT4oMini (검증 전용 저렴 모델; go-openai에 실재 확인) — 신설
- verifyTimeout = 10 * time.Second — 신설
- languageVerifyPrompt = (위 §5.3 문구) — 신설
- englishPrimarySubtag = "en" (이것과 같으면 검증 스킵; §5.1.1) — 신설

### 5.5 재생성이 다른 결과를 내는 근거 (temperature) — R1-2 반영

재시도가 의미를 가지려면 동일 입력의 재생성이 다른 출력을 낼 수 있어야 한다. 현재
ChatCompletionRequest는 Temperature를 설정하지 않아 OpenAI 기본값(비-0)에 의존한다. 즉
재생성 변동성의 근거는 "암묵적 비-0 temperature"다. 이 전제를 명시한다:

- 만약 향후 생성 요청 Temperature를 0으로 고정하면 재시도가 동일 오언어를 반복해 순수 낭비가
  된다. 그 시점에는 "재시도 시 언어 지시 강화(강조 문구 prepend)" 같은 변동 유발책이 함께
  필요하다. 현재는 기본 temperature에 의존하며, 이 의존성을 §6 리스크에 남긴다.
- 이번 스코프에서는 재시도 시 입력을 바꾸지 않는다(단순성). 강조-prepend 대안은 후속 검토.

## 6. 트레이드오프 / 리스크

- **비용/지연(비영어 타깃 + 강한 1차로 재시도 희소)**: 검증(2차)은 비영어 타깃(en 스킵)에만
  수행(§5.1.1). 게다가 1차(system 강제)로 첫 생성 준수율이 높아 **검증이 no를 내는 일이 드물어
  재시도가 거의 발동하지 않는다**. 지배 경로 en-US는 검증 스킵(오버헤드 0). 비영어 타깃 정상
  통과 시 생성1+검증1=2회. 최악(2회 재생성 전부 불일치) 시 생성3(GPT4Turbo)+검증3(GPT4oMini)=6회
  이나, 강한 1차로 이 최악 케이스 확률은 낮다. 검증은 저렴 모델 + 1500룬 샘플로 억제.
- **검증 백오프 지연 → 실제 상한 확보(R1-3/R3-치명/R4-2)**: 기존 Send는 backoff.WithContext
  없이 backoff.Retry(MaxElapsedTime 1분)라 context.WithTimeout(10s)로 감싸도 10초 상한이 서지
  않는다(1분까지 재시도). → 검증은 Send가 아닌 신설 SendOnce(backoff 없이 1회 호출, §5.3)를
  10s context로 호출해 실제 상한을 둔다. 타임아웃 시 "통과 간주"로 빠져 정상 요약을 버리지 않음.
  강한 1차 덕에 검증 자체가 드물게만 실행되므로 이 지연 성분의 기대값은 매우 작다.
- **검증 fail-open의 잔존 위험(R6-2)**: SendOnce는 backoff를 제거했으므로 429(rate limit)/일시
  오류가 즉시 "통과 간주"로 귀결된다. 이런 오류가 트래픽 급증 시(=오언어 가능성이 높은 때)
  집중되면 일부 검증이 통과로 스킵될 수 있다. 그럼에도 **재시도를 추가하지 않는 것이 의도된
  선택**이다: (a) 정확성엔 안전(검증 실패가 정상 요약을 버리지 않음), (b) 주력은 1차 강제이고
  검증은 드물게만 발동하는 안전망이라 급증 구간의 일부 스킵이 치명적이지 않음, (c) 429 급증은
  현재 실측 신호가 아닌 가정이므로, 선제적 재시도(신규 config 표면·복잡도)를 두는 것은 과설계다.
  실제로 급증 구간 스킵이 관측되면 그때 bounded 재시도를 도입한다(트리거 기반).
- **검증 LLM 오탐(R2-4)**: 섹션 헤더가 영어로 유지되어 비-라틴 요약도 라틴 비중이 높다.
  검증 프롬프트가 "섹션 헤더/라벨/고유명사/URL/기술용어 무시, 산문만 판정"을 명시(§5.3)해
  완화. 불확정은 통과 간주(§5.3)라 정상 요약을 버리지 않음. 하네스는 "명백한 불일치"에만
  재시도.
- **재시도 무익 케이스 + temperature 의존(R1-2)**: 재생성 변동성은 암묵적 비-0 temperature에
  의존(§5.5). transcript가 극히 짧거나 비음성이면 LLM이 계속 오언어를 뱉을 수 있음. 상한(3시도)
  후 마지막 비어있지 않은 결과 반환 → 무한 루프 없음. 실패는 Warn 로그로 관측.
- **dedup 미해결**: 이번 증상이 dedup이면 이 강제 로직은 작동 기회 없음(§3). 배포 후 대표
  재확인으로 원인 분리. 배포가 이번 증상을 즉시 고친다고 단정하지 않음.
- **오버엔지니어링 아님**: 실측 신호(대표가 프로덕션에서 영어 출력 확인)에 기반한 대응. 주력은
  프롬프트 구조 개선(system 분리)으로 신규 인프라 없음. 안전망 검증도 기존 LLM 엔진 재사용 +
  비영어 타깃 한정 + 강한 1차로 발동 희소.

## 7. 검증 계획

- go build ./... / go test ./pkg/summaryhandler/... / go vet — 전부 green.
- **gomock 생성/검증 호출 구분(R1-1/R2-5)**: 생성은 engineOpenaiHandler.Send, 검증은 신설
  engineOpenaiHandler.SendOnce로 **메서드 자체가 달라** 구분이 명확하다(matcher 부담 감소).
  - 생성 Send: req.Messages가 [system, user] 2개이고 Messages[0].Role==system, Model==defaultModel
    인지 확인. system 메시지에 outputLanguage 값이 들어갔는지도 assert(1차 강제 검증).
  - 검증 SendOnce: req.Messages[0].Content에 languageVerifyPrompt 조각("language detector")이
    포함되고 Model==defaultVerifyModel인지 확인.
  - gomock.InOrder로 Send→SendOnce→(재)Send→SendOnce 순서 강제. 각 EXPECT는 .Times(1) +
    DoAndReturn으로 호출별 다른 응답. 이로써 어느 응답이 생성/검증에 매핑되는지 결정론적 보장.
- 테스트 케이스(content_test.go), 검증 경로는 비영어 타깃(ko-KR)로 진입:
  1. 검증 통과(1회): Send(생성)+SendOnce(yes) → 그 결과 반환. InOrder.
  2. 불일치 후 재생성 통과: Send#1(영어)+SendOnce(no) → Send#2(한국어)+SendOnce(yes) → Send#2
     결과 반환. InOrder 4개. DoAndReturn으로 호출 순서별 응답.
  3. 상한 도달: Send3+SendOnce3 전부 no → 마지막(Send#3) 비어있지 않은 결과 반환, 에러 없음.
  4. 검증 불확정(빈 응답): SendOnce가 yes/no 아님 → 통과 간주, Send#1 결과 반환.
  5. 빈 결과 보존(R2-1): Send#1(비어있지 않음)+SendOnce(no) → Send#2가 "" 반환 → Send#1 결과 반환.
  6. 재시도 Send 하드 실패 폴백(R4-1): Send#1+SendOnce(no) → Send#2 error → Send#1 결과 반환(에러 아님).
  7. 최소 산문 스킵(R4-4): 요약이 헤더+"- None"만(산문<20룬) → SendOnce EXPECT 없음, Send#1 반환.
  8. 영어 타깃 스킵(en-US): 검증 스킵, Send 1회만 → 기존 동작. SendOnce EXPECT 없음.
  9. 1차 강제 검증: Send에 넘어간 req의 Messages[0](system)에 outputLanguage가 박혔는지 assert.
  10. 기존 fixture(실제 언어 기준): call=en-US·transcribe=en-US는 검증 스킵(Send 1회, SendOnce
     없음). **conference=ko-KR(라인 104)·recording=ko-KR(라인 134)는 shouldVerify=true라
     SendOnce(yes) EXPECT 추가 필요**(R3-경/R6-치명). none=""는 effectiveLang→en-US 방어로 검증
     스킵(Send 1회). 단 모든 케이스의 Send는 이제 [system,user] 2메시지 → expectedRequest 단언을
     2메시지로 갱신(Messages[0]=system에 effectiveLang 포함, Messages[1]=user JSON).
- 주: "outputLanguage 빈 값" 케이스는 정규화로 프로덕션 도달 불가(§3/R2-2). shouldVerify의
  primarySubtag!="" 가드는 방어 코드로만 두고, 단위 테스트는 영어 스킵(케이스 8)으로 대체한다.

## 8. 리뷰 이력
- R1 (deleg_cf71493e, task0): CHANGES_REQUESTED. 삽입지점/재시도상한/보수적검증/dedup경계 통과.
  지적 3: (1)테스트 gomock 생성/검증 Send 구분 미명세 (2)temperature 의존 미문서화
  (3)검증 백오프 지연 과소계상.
- R2 (deleg_cf71493e, task1): CHANGES_REQUESTED. 지적 6: (1)빈 결과가 앞 비어있지 않은 결과
  덮는 회귀 (2)outputLanguage 빈값 스킵은 정규화로 죽은 코드-근거 정정 (3)en-US 검증이 항상
  수행돼 지배 비용 케이스-스킵 미검토 (4)섹션 헤더 영어라 비-라틴 오탐 위험 (5)gomock 구분
  미명세 (6)검증에 GPT4Turbo 과함-저렴 모델 옵션.
- 정정(R1+R2 반영):
  * §5.1.1 신설: 비-라틴 타깃만 검증(en-US/라틴 스킵) → 지배 비용/오탐 동시 해결(R2-3/4).
  * §5.2 lastNonEmpty로 빈 결과 회귀 방지(R2-1).
  * §5.3 검증 프롬프트에 섹션 헤더/라벨 무시 명시(R2-4), 저렴 모델 defaultVerifyModel(R2-6),
    verifyTimeout 10s context(R1-3).
  * §5.5 temperature 의존 명시(R1-2).
  * §7 gomock InOrder + payload matcher로 생성/검증 구분(R1-1/R2-5), 빈값 스킵 근거 정정(R2-2).
- (재리뷰 R3/R4 대기)
- R3 (deleg_5410fa3f, task0): CHANGES_REQUESTED. [치명] verifyTimeout(10s)가 실제 안 먹음
  (send.go backoff.WithContext 없음 → 1분 재시도). [중] 라틴 전면 스킵 근거 오류(LLM은 fr→en도
  구분, 비영어 라틴 사각지대). [경] 룬 절단 패닉, 기존 ko-KR fixture 2건 검증 Send 추가.
- R4 (deleg_5410fa3f, task1): CHANGES_REQUESTED. [신규] 재시도 Send 하드 실패가 확보한 정상
  요약을 버림(R2-1과 대칭). verifyTimeout 과대표기(실제 ~60s). 대소문자 취약(KO-KR 스킵 누락).
  극단 요약(산문 0) 검증 오탐. zh-Hant/오버엔지니어링은 이슈 아님 확인.
- **재정렬(대표, 2026-09-18)**: "처음 요약 질의부터 강제하면 재시도 불필요" 지적 반영. 주력을
  1차(첫 생성 강제)로 이동, 검증 하네스는 얇은 안전망으로 격하. 결정: 첫 생성 강제(system 분리
  +언어값 직접 주입) + 안전망 하네스 둘 다.
- 정정(R3+R4 + 재정렬 반영):
  * §5.0 신설: 1차 첫 생성 강제(system 메시지 분리, outputLanguage 값 직접 주입, self-ref 제거).
    §1/§2/§3/제목 재정렬(주력=1차, 하네스=안전망).
  * §5.1.1: 라틴 전면 스킵 → **en 계열만 스킵**(비영어 라틴 사각지대 제거, R3-중). canonical
    primary subtag 소문자화(R4-3 대소문자).
  * §5.2: 재시도 Send 하드 실패 best-effort 폴백(R4-1, attempt>=2는 lastNonEmpty 반환).
  * §5.3: verifyTimeout 실효화 → 신설 SendOnce(backoff 우회)로 진짜 10s 상한(R3-치명/R4-2).
    룬 안전 절단(R3-경), 최소 산문 가드(R4-4).
  * §4: SendOnce(engine_openai_handler) 신설, main.go system 프롬프트.
  * §7: Send/SendOnce 메서드 분리로 gomock 구분, 기존 ko-KR fixture 2건 명시(R3-경),
    [system,user] 2메시지 단언, 케이스 재구성(폴백/최소산문/1차 강제 검증 추가).
- (재리뷰 R5/R6 대기)
- R5 (deleg_d8250e32, task0): APPROVED. §5.0 system 강제 실효(ChatMessageRoleSystem 실재,
  mock 호환), SendOnce 근거 정확(send.go backoff.WithContext 없음), en만 스킵, 자기일관성,
  temperature 근거, R1~R4 9건 반영 모두 확인. Minor(비차단): §5.4에 languageSystemPromptFmt
  누락, 다국어 None 라벨, 파일명 개명 제안.
- R6 (deleg_d8250e32, task1): CHANGES_REQUESTED. [치명] §4/§7이 recording fixture를 en-US로
  오기(실제 ko-KR, 라인 134) → contentGet은 정규화 안 하므로 shouldVerify=true, SendOnce 필수.
  ko-KR 2건=conference(104)+recording(134). [안전망] SendOnce 무재시도 fail-open이 429/급증
  구간에 대량 스킵 위험. [프롬프트] 빈 outputLanguage system 지시 malformed + empty-fallback 유실.
- 정정(R5 minor + R6 반영):
  * §4/§7: recording=ko-KR 정정, ko-KR 2건=conference+recording로 통일(R6-치명). fixture 실제
    언어 표기(call/transcribe=en-US 스킵, conference/recording=ko-KR 검증, none="" 방어).
  * §5.0: contentGet 진입 시 effectiveLang 빈값→en-US 로컬 방어, system 주입/shouldVerify/
    RequestContent에 일관 사용(R6-프롬프트). main.go empty-fallback 대체.
  * §5.3/§5.4/§6: verifyTimeout 예산 내 bounded 재시도(languageVerifyMaxTries=2, RetryWait=500ms)로
    429/일시 오류 흡수, 잔존 스킵은 §6 트레이드오프로 명시(R6-안전망).
  * §5.4: languageSystemPromptFmt/defaultOutputLanguage 상수 추가(R5-minor).
- (재리뷰 R7/R8 대기)
- R7 (deleg_92b1f81b, task0): CHANGES_REQUESTED. [확인] R6-치명 정정 정확(fixture 언어 일치,
  contentGet 정규화 없음, bounded 재시도 상한, effectiveLang 일관). [결함] §5.4가 defaultOutputLanguage
  "추가" 지시하나 start.go L22에 이미 선언됨 → 중복 선언 시 컴파일 에러. "재사용"으로 정정 필요.
- R8 (deleg_92b1f81b, task1): CHANGES_REQUESTED. [경] §5.3 "재시도 불필요" 서술이 R6-2 bounded
  재시도와 표면 모순. [경] R6-2 bounded 재시도가 "안전망 위의 안전망"으로 오버엔지니어링 의심
  (새 config 표면 2개, 실측 신호 아닌 429 가정). 나머지(canonical/최소산문)는 이슈 아님.
- 정정(R7 + R8 반영, CPO 판단):
  * **R8-2 채택 → bounded 재시도 제거**: 대표 원칙(실측 신호 없는 선제 방어 지양, 신규 config
    표면 지양)에 부합. §5.3/§5.4/§6에서 languageVerifyMaxTries/RetryWait 제거, SendOnce 1회 +
    fail-open으로 단순화. 잔존 위험은 §6에 트레이드오프로 명시(트리거 기반 향후 도입).
  * R8-1 동시 해소: §5.3 "재시도 불필요" 서술이 이제 실제 무재시도와 일관(모순 제거).
  * R7: §5.4 defaultOutputLanguage를 "신설"→"start.go 기존 상수 재사용"으로 정정.
- (재리뷰 R9/R10 대기)
- R9 (deleg_5dd954aa, task0): APPROVED. bounded 재시도 완전 제거(활성 본문 잔재 없음), fail-open
  서술 일관(R8-1 해소), defaultOutputLanguage 재사용(R7 해소, start.go L22 실재), send.go backoff
  우회 근거·fixture 언어 실코드 일치. 새 결함 없음.
- R10 (deleg_5dd954aa, task1): APPROVED. send.go backoff/go-openai 상수(GPT4oMini, RoleSystem)
  실재, start.go L22 재사용 정확, fixture 언어 일치, contentGet 시그니처 불변, 단순화 일관,
  오버엔지니어링 잔재 없음, 구현 착수 가능. 새 이슈 없음.
- **설계 확정(R9/R10 2연속 APPROVED, 2026-09-18)**. 설계 리뷰 총 10회. 구현 착수.

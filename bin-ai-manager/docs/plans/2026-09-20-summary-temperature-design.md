# 설계: AI Summary temperature 고정으로 None 요약 완화 (VOIP-1536)

작성일: 2026-09-20
작성자: Hermes (CPO)
브랜치: VOIP-1536-summary-none-quality-guard
상태: 설계 (리뷰 대기)
선행: 이슈 분석 문서 `2026-09-20-summary-none-quality-analysis.md` (리뷰 6회, 2연속 Approve 확정)

## 1. 목표

AI Summary 생성/재생성이 간헐적으로 전 섹션 "None"을 반환하는 비결정성을, 요약 생성·검증 요청의
temperature를 낮은 비영값으로 고정해 완화한다. 근본 원인은 temperature 미설정(OpenAI 기본값 1.0)이며,
이는 생성/재생성 공통 코드(content.go:221)에 존재한다.

## 2. 범위

포함:
- 요약 생성 요청(content.go:221 `ChatCompletionRequest`)에 낮은 temperature 명시
- 언어 검증 요청(content.go:343 `ChatCompletionRequest`)에 낮은 temperature 명시
- temperature 상수 신설(main.go 상수 블록)

제외 (이슈 분석 §5/§6 확정):
- 품질 검증(전 섹션 None 감지 → 재생성) 미도입. 관찰 트리거만 정의(재발 실측 시 후속 이슈).
- VOIP-1532 언어 강제 하네스 로직 변경 없음(shouldVerify/verifyOutputLanguage/proseLen 불변).
- VOIP-1535 regenerate 로직 변경 없음.
- 프롬프트 변경 없음(이미 None 남발 금지 명시, 프롬프트만으로 불충분함이 이슈 분석에서 확인).

## 3. 설계 결정

### 3.1 temperature 값: 0.2 (비영값)
- 요약은 창의성보다 일관성이 중요 → 낮은 값 채택.
- **값 0 금지**: go-openai v1.41.2 `Temperature float32 json:"temperature,omitempty"`(포인터/omitzero
  없음)라 0은 JSON에서 생략되어 OpenAI가 기본값 1.0을 재적용 → 버그 재발. 반드시 비영값이어야 직렬화된다.
- 0.2 vs 0.3 중 0.2 선택: 더 낮을수록 재현 안정성이 높고, 요약 태스크에서 0.2는 충분히 자연스러운 출력을
  낸다(일반적 실무 관례). 극단값 회피(0.1 미만은 반복/degeneration 위험).

### 3.2 적용 범위: 생성 + 검증 양쪽
- 생성(defaultModel=GPT4Turbo, content.go:221): 주 개입 대상.
- 검증(defaultVerifyModel=GPT4oMini, content.go:343): yes/no 판정은 결정적일수록 유리하고 다양성 손실이
  없으므로 낮은 temperature가 부작용 없이 안전. 동일 상수 재사용.

### 3.3 상수 단일 정의 (단일 write-through)
main.go 상수 블록에 `summaryTemperature float32 = 0.2` 1개만 정의하고 두 요청 지점에서 재사용한다.
생성/검증에 별도 상수를 두지 않는다(둘 다 "낮게 고정"이 목적이라 값 분기 불필요; 향후 분리 필요가 실측되면
그때 나눈다(선제 분리는 오버엔지니어링).

## 4. 구현 상세

### 4.1 main.go 상수 신설
`defaultModel` 인근 상수 블록에 추가:
```go
// summaryTemperature pins the sampling temperature for summary generation and
// language verification. Summaries must be consistent, not creative, so a low
// value curbs the nondeterminism that intermittently produced all-"None"
// output (VOIP-1536). Must be non-zero: go-openai's Temperature field is
// `json:"temperature,omitempty"`, so a 0 value is dropped and OpenAI re-applies
// its default of 1.0.
summaryTemperature float32 = 0.2
```

### 4.2 content.go 생성 요청 (221)
```go
req := &openai.ChatCompletionRequest{
    Model:       defaultModel,
    Temperature: summaryTemperature,
    Messages:    []openai.ChatCompletionMessage{ ... },
}
```

### 4.3 content.go 검증 요청 (343)
```go
req := &openai.ChatCompletionRequest{
    Model:       defaultVerifyModel,
    Temperature: summaryTemperature,
    Messages:    []openai.ChatCompletionMessage{ ... },
}
```

## 5. 테스트 전략

**핵심: 기존 exact-match 테스트 fixture를 반드시 갱신해야 한다(미갱신 시 FAIL).**
`content_test.go:253-265`의 `tmpRequestContent`는 `mockOpenai.EXPECT().Send(ctx, tmpRequestContent)`로
요청 전체를 gomock 정확일치(gomock.Eq → `reflect.DeepEqual`로 `*ChatCompletionRequest` **Go 구조체**를
비교; JSON 직렬화가 아님)로 매칭한다. 이 fixture는 현재 `Model: defaultModel`만 있고 Temperature가 없다.
content.go:221에 `Temperature: summaryTemperature`를 추가하는 순간 실제 요청의 Temperature(0.2) ≠
expected(0.0)가 되어 이 기존 테스트가 즉시 FAIL한다. 따라서 **fixture(253-265)에도 Temperature를 반드시
추가**해야 한다.

- **fixture는 리터럴 `0.2`로 고정(중요)**: fixture에 `Temperature: summaryTemperature`(상수 참조)를 쓰면,
  상수 자체를 0으로 바꾸는 회귀(§3.1이 가장 우려하는 "값 0")를 잡지 못한다. 실요청과 expected가 같은
  상수를 참조해 함께 0이 되므로 DeepEqual이 여전히 통과하여 mutation이 생존한다. 따라서 fixture는
  **리터럴 `Temperature: 0.2`**로 고정해 값을 이중 기입한다(실코드는 상수, 테스트는 리터럴). 이렇게 해야
  상수를 0으로 바꾸는 회귀도 실요청 0.0 ≠ expected 0.2로 FAIL한다.
- **mutation 가드 귀속과 실패 기전(정정)**: Temperature 회귀를 잡는 것은 위 exact-match 테스트
  (content_test.go:253-266)다. 실패 원인은 구조체 필드 비교(실요청 Temperature ≠ expected Temperature)이지
  omitempty 직렬화와 무관하다(omitempty는 §3.1의 프로덕션 버그 발생 원인일 뿐 테스트 실패 원인이 아니다).
  mutation: content.go:221의 Temperature 라인 제거 → 실요청 0.0 ≠ expected 0.2 → FAIL. 상수를 0으로
  변경 → 실요청 0.0 ≠ expected(리터럴) 0.2 → FAIL. 반면 `Test_contentGet_verificationHarness`의
  genDo/verifyDo(content_test.go:308-357)는 요청을 `gomock.Any()`로 받고 req.Model/Messages만 assert하므로
  Temperature 변화를 잡지 못한다.
- **검증 요청 커버리지**: 검증 경로 테스트는 field-selective Do 매처라 Temperature를 검사하지 않는다.
  검증 요청(content.go:343)의 Temperature까지 회귀 보장하려면 verify 경로 Do 매처에 Temperature assert를
  추가하는 것을 검토(구현 시 기존 패턴 범위 내에서 판단, 신규 추상화 신설 지양). 최소 요건은 생성 경로
  exact-match fixture 갱신이며, 검증 경로 커버리지는 저위험(yes/no 판정)이라 선택 사항으로 둔다.
- **기존 하네스/regenerate 테스트**: 위 fixture 갱신을 제외하면 VOIP-1532 언어 강제 하네스와 VOIP-1535
  regenerate 테스트의 로직 검증은 그대로 유지된다(제어흐름 미변경).
- mock engine이 요청을 정확일치로 캡처하는 구조(content_test.go:266)임이 확인되었으므로 신규 추상화는
  불필요하다.

## 6. 리스크 및 완화

- **완화책이지 근절책 아님**: temperature 0.2도 비결정성을 줄일 뿐 제거하지 못한다. None 재발률이 0이
  되지는 않는다. → 관찰 트리거(§7)로 재발 모니터링.
- **요약 품질 변화**: 0.2로 낮추면 출력이 더 결정적/보수적이 된다. 요약 태스크에선 바람직하나, 만약
  단조로움이 관찰되면 값을 0.3으로 상향(상수 1곳 변경).
- **검증 하네스 영향**: 검증 요청도 0.2가 되나 yes/no 판정이라 영향 미미(오히려 안정). 기존 fail-open
  동작 불변.

## 7. 관찰 트리거 (품질 검증 후속 판단 신호)

temperature 적용 후에도 전 섹션 None이 재발하는지 실측으로 판단한다. 신호: 프로덕션 로그에서 요약 응답이
all-None 패턴(proseLen==0)인 건수. 일정 빈도 이상이면 품질 검증(내용-실질성 판정 또는 min-prose 가드
재설계)을 후속 이슈로 착수. 지금은 트리거만 정의하고 선제 구축하지 않는다(오버엔지니어링 지양).

## 8. 문서/파급

- RST: 이 변경은 API 스키마 변경이 아님(내부 LLM 파라미터). OpenAPI/RST 문서 갱신 불요.
- 다른 서비스 파급 없음(bin-ai-manager 내부 상수).
- gen 파일 영향 없음.

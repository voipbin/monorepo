# VOIP-1537 추가 이슈 분석: Korean regenerate 빈 결과 + 503 (send.go 에러 삼킴)

## 1. 배경 / 증상

대표(admin.voipbin.net)가 recording `9174a2ea-778d-4e47-90bd-a637e451f058`를
Korean으로 regenerate 했으나:
- (초기) 영어로만 나옴
- (최근) regenerate를 눌러도 요약이 안 바뀜 + 프론트에서 503 REQUEST_TIMEOUT

summary id: `7c90c237-069c-4dd1-9202-327ae2b3f5af`

## 2. Loki 로그 증거 (VPN, service_name=bin-ai-manager.*)

라벨 스킴 변경 확인: `compose_service` → `service_name`, Grafana 포트 3000→80, Loki datasource id 1→2.

시간대별:
| 시각 | output_language | 소요 | 결과 |
|------|-----------------|------|------|
| 02:21 (2회) | **en-US** | ~5초 | choices=1 성공, 영어 요약 반환 |
| 06:53 (3회) | **ko-KR** | ~64초 | response=nil → empty choices → 빈 콘텐츠 거부 |
| 09:25~09:27 (2회) | **ko-KR** | ~64초 | 동일 |

핵심 관찰:
- en-US 요청은 검증 스킵(shouldVerify=false) + 즉시 성공 → 영어 반환. 이건 "정상 동작"이며, 초기 영어 현상은 **프론트가 en-US를 보냈기 때문**(최근엔 ko-KR 정상 전달됨).
- ko-KR 요청은 100% `response=nil`, 각 64초 소요, 에러 로그 0건.
- 64초 ≈ send.go의 backoff MaxElapsedTime=1분과 일치.

## 3. 근본 원인 (코드 확정)

### 3.1 주 버그: send.go 에러 삼킴 (변수 shadowing)

`bin-ai-manager/pkg/engine_openai_handler/send.go`:

```go
func (h *engineOpenaiHandler) send(ctx, req) (*ChatCompletionResponse, error) {
    ...
    var resp openai.ChatCompletionResponse
    var err error                         // (A) 바깥 err — 항상 nil로 남음
    operation := func() error {
        var err error                     // (B) 안쪽 err — (A)를 shadow
        resp, err = h.client.CreateChatCompletion(ctx, *req)
        if err != nil { return err }      // (B)만 backoff로 전달
        return nil
    }
    if errRetry := backoff.Retry(operation, expBackoff); errRetry != nil {
        return nil, err                   // (C) 버그: errRetry가 아니라 (A) err(nil) 반환
    }
    return &resp, nil
}
```

- operation 내부 err은 `var err error`로 바깥 (A)를 shadow → 바깥 err은 영원히 nil.
- 재시도 최종 실패 시 errRetry는 non-nil이지만, `return nil, err`가 (A) nil을 반환.
- 결과: 실패한 LLM 호출이 `(nil, nil)`로 반환됨.

호출부 `content.go` generateOnce:
```go
tmpRes, errSend := h.engineOpenaiHandler.Send(ctx, req)
if errSend != nil { return "", errors.Wrapf(...) }   // errSend=nil이라 통과
if tmpRes == nil || len(tmpRes.Choices) == 0 {
    log.Debugf("Received response with empty choices")
    return "", nil                                    // 빈 문자열 반환
}
```
→ nil 응답을 "빈 응답"으로 오인 → 빈 콘텐츠.

regenerate.go:
```go
if newContent == "" {
    return nil, errors.Errorf("regenerated summary content is empty; keeping the existing summary")
}
```
→ 빈 콘텐츠 거부 → 요약 안 바뀜.

### 3.2 파생 증상 1: 503 REQUEST_TIMEOUT
- 실패한 Send가 backoff 1분을 다 돌아 64초 소요.
- (직전 커밋 cb77a5d8d 이전) regenerate RPC timeout이 3초 → 64초를 못 기다림 → 503.
- timeout hotfix(cb77a5d8d, 50초)로 완화되나, 64초 > 50초라 **여전히 timeout 가능**. 근본은 이 버그.

### 3.3 파생 증상 2: 원인 은폐
- 에러가 삼켜져 로그에 LLM 실패 사유(429/500/timeout/context length 등)가 안 남음.
- send.go 수정이 **진짜 LLM 실패 원인을 드러내는 전제조건**.

### 3.4 동일 패턴 2차 발생 지점
`bin-ai-manager/pkg/engine_dialogflow_handler/send.go:30` 동일 shadowing 버그
(`return nil, err`, err는 항상 nil). 같은 수정 필요.

## 4. 왜 ko-KR만 실패하나 (미확정, send.go 수정 후 확인)

- en-US(5초 성공) vs ko-KR(64초 실패)의 차이는 시스템 프롬프트 언어 지정뿐.
- 현재 로그로는 LLM 실패 사유를 알 수 없음(3.3).
- 가설: (a) ko-KR 출력 강제 프롬프트에서 gpt-4-turbo가 특정 에러(반복 timeout), (b) rate limit이 우연히 ko-KR 시도 시각에 겹침, (c) 프로바이더 간헐 장애.
- **판단**: send.go 수정을 먼저 배포 → 에러 로그로 진짜 사유 확인 → 필요 시 추가 대응. VOIP-1537의 gemini-3.8-flash 전환이 gpt-4-turbo 특유 문제라면 함께 해소될 수 있음(엔진 교체).

## 5. 진행 타당성

- **유효**: 코드로 재현 확정(shadowing 버그 실재), 로그로 증상 확정. 프로덕션 장애.
- **스코프**: bin-ai-manager, PR #1327(미머지)에 흡수(대표 지시). gemini 전환과 같은 레포·같은 엔진 계층.
- **리스크**: 낮음. 버그 수정은 에러를 올바로 전파할 뿐, 정상 경로 무영향. 정상 응답 시 동작 불변.
- **완전성**: send.go + dialogflow send.go 두 곳 동시 수정(같은 버그 클래스). 부분 수정 지양.

## 6. 수정안 (설계 단계에서 확정)

1. send.go: `return nil, err` → `return nil, errRetry` (또는 operation 내부 shadow 제거로 바깥 err에 담기). errRetry 반환이 최소·명확.
2. engine_dialogflow_handler/send.go: 동일.
3. 단위 테스트: CreateChatCompletion이 계속 에러 → Send가 non-nil 에러 반환하는지(현재는 nil 반환 버그). mock으로 검증 가능.
4. gemini 전환(VOIP-1537 기존 변경)과 함께 배포 시, 에러 전파로 ko-KR 실패 사유가 로그에 남는지 카나리 확인.

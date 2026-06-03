# Fix: confbridge StartContextIncoming ChannelJoin error silencing

**Date:** 2026-06-03
**Service:** bin-call-manager
**File:** `pkg/confbridgehandler/start.go`
**Severity:** Critical — SIP call setup always fails for incoming ext-to-ext calls

---

## Problem Statement

모든 extension-to-extension 인바운드 통화에서 caller(7001)가 30초 timeout 후 FAIL로 종료된다. pjsua 입장에서는 SIP INVITE 이후 100 Trying조차 수신하지 못한 것처럼 보이나, 실제로는 VoIPBin 백엔드가 confbridge bridging 중 에러를 무시하고 있다.

### Root Cause

`pkg/confbridgehandler/start.go:45`에 다음 버그가 존재한다.

```go
// 현재 코드 (버그)
if errJoin := h.bridgeHandler.ChannelJoin(ctx, cb.BridgeID, channelID, "", false, false); errJoin != nil {
    log.Errorf("Could not add the channel to the bridge. err: %v", err)   // ← err (= nil) 참조
    return errors.Wrap(err, "could not add the channel to the bridge")    // ← Wrap(nil) = nil 반환
}
```

- `ChannelJoin`이 에러를 반환하더라도, 에러 변수 `errJoin`이 아닌 이전 라인의 `err`(이미 `nil`로 처리됨)를 참조한다.
- `errors.Wrap(nil, msg)` = `nil`이므로 함수가 정상 반환한다.
- 에러 로그도 찍히지 않는다.

### Symptom Chain

1. `StartContextIncoming` 호출: `conf-in-000001a0` (outgoing leg 7002) 채널이 `StasisStart` 진입
2. `bridgeHandler.ChannelJoin()` 호출: bridge `1baeb03b`에 join 시도
3. `ChannelJoin` 에러 발생 — 원인 미표시 (로그 미출력)
4. 버그로 인해 `nil` 반환 → 에러 무시
5. `Joined` 이벤트 발화 없음 → `joinedTypeConnect`의 2번째 채널 분기 진입 불가
6. `confbridge.Answer()` 미호출 → incoming 7001 channel 영구 ring 상태
7. 7001 pjsua 30s timeout → `FAIL: Call setup failure — timeout waiting for CONFIRMED`

### Evidence from Logs

- `call-manager` 로그: `StartContextIncoming` → `Found confbridge` 이후 `Joined` 이벤트 없음
- `asterisk-conference` 로그: `conf-in-000001a0 joined bridge` 직후 `left bridge` — join 후 즉시 이탈
- `RTPAUDIOQOS` 수치 존재 (7002 측 RTP 정상 처리됨) — backend 처리는 진행됐으나 incoming leg answer만 누락

---

## Scope

### In Scope
- `start.go` 2라인 변수명 수정 (`err` → `errJoin`)

### Out of Scope
- `bridgeHandler.ChannelJoin` 내부 실패 원인 분석 (별도 이슈)
- `ChannelJoin`에 재시도 로직 추가
- 다른 핸들러에서 유사 패턴 점검 (별도 audit 이슈로 분리)

---

## Fix

### Before

```go
// pkg/confbridgehandler/start.go:44-48
if errJoin := h.bridgeHandler.ChannelJoin(ctx, cb.BridgeID, channelID, "", false, false); errJoin != nil {
    log.Errorf("Could not add the channel to the bridge. err: %v", err)
    return errors.Wrap(err, "could not add the channel to the bridge")
}
```

### After

```go
// pkg/confbridgehandler/start.go:44-48
if errJoin := h.bridgeHandler.ChannelJoin(ctx, cb.BridgeID, channelID, "", false, false); errJoin != nil {
    log.Errorf("Could not add the channel to the bridge. err: %v", errJoin)
    return errors.Wrap(errJoin, "could not add the channel to the bridge")
}
```

변경 사항: 2라인, `err` → `errJoin` (각 1단어씩).

---

## Impact Analysis

### Fix 적용 후 동작 변화

| 시나리오 | 수정 전 | 수정 후 |
|---------|---------|---------|
| `ChannelJoin` 성공 | 동일 (정상) | 동일 (정상) |
| `ChannelJoin` 실패 | 에러 무시, nil 반환, 통화 hang | 에러 로그 출력 + 에러 반환 → `StartContextIncoming` caller에서 처리 |

### Error propagation 경로 확인

`StartContextIncoming` → `confbridgehandler/ari_event.go:23` → `EventHandlerStasisStart`

```go
// ari_event.go
case channel.ContextConfIncoming:
    return h.StartContextIncoming(ctx, cn)
```

`StartContextIncoming` 에러 반환 시 Stasis 이벤트 핸들러가 에러를 로그로 남기고 채널을 정리한다. 이는 기존 패턴과 일치한다.

### 수정 후 `ChannelJoin` 실패 시 예상 흐름

1. `ChannelJoin` 에러 반환
2. `log.Errorf("Could not add the channel to the bridge. err: %v", errJoin)` 출력
3. `errors.Wrap(errJoin, ...)` 반환
4. ARI 이벤트 핸들러에서 에러 수신 → 채널 hangup 처리
5. outgoing call (7002) hangup → groupcall이 전체 통화 정리
6. incoming 7001 channel: hangup_cause가 명확히 설정됨 (현재는 timeout 30s 후 603 Decline)

수정 후에도 통화 실패는 발생하나, 실패 이유가 로그에 기록되고 cleanup이 즉시 이루어진다.

---

## Test Verification

수정 후 `test_sip_watchdog.py` 실행 시 확인 항목:

1. `ChannelJoin` 에러가 있다면 call-manager 로그에 `Could not add the channel to the bridge` 출력 확인
2. `ChannelJoin` 정상이라면 통화 CONFIRMED → `joinedTypeConnect` 2번째 채널 → `Answer` 호출 → pjsua CONFIRMED 수신

`ChannelJoin` 자체가 실패하고 있는지 성공하는지는 이 fix 적용 후 로그에서 최초로 확인 가능하다.

---

## Related Files

| 파일 | 변경 내용 |
|------|----------|
| `bin-call-manager/pkg/confbridgehandler/start.go` | 2라인 변수명 수정 |

---

## Error Propagation Path (v2 추가)

리뷰어가 지적한 가장 중요한 미분석 항목: `StartContextIncoming`이 에러를 반환했을 때 ARI 이벤트 핸들러가 어떻게 처리하는가.

### 호출 경로

```
EventHandlerStasisStart (ari_event.go)
  → channel.ContextConfIncoming 케이스
  → StartContextIncoming(ctx, cn) 에러 반환 시
  → EventHandlerStasisStart 에서 에러 수신
```

`ari_event.go`의 `EventHandlerStasisStart` 에러 처리 패턴:
- 에러 반환 시 채널에 대해 `channelHandler.HangingUp()` 호출 — 해당 채널 정리
- confbridge의 반대편 채널(outgoing leg, 7002)은 groupcall / chained_call hangup 로직이 연쇄적으로 정리 (기존 로그 확인됨: `hangup_by=remote, cause=16`)

**결론:** fix 적용 후 `ChannelJoin` 에러 반환 시 해당 conf-in 채널은 hangup되고, outgoing leg도 이미 존재하는 cleanup 로직으로 정리됨. 무한 stasis 상태로 방치되지 않는다.

단, 구현 PR에서 `ari_event.go`의 에러 핸들링 코드를 직접 확인 후 명시적으로 문서화할 것.

## Required in Implementation PR

| 항목 | 이유 |
|------|------|
| 단위 테스트: `ChannelJoin` 에러 경로 | 동일 회귀 방지, `joined_test.go` gomock 패턴 이미 존재 |
| `confbridgehandler/*.go` 파일 스캔 | 동일 패턴 재발 방지, 전체 codebase 감사보다 좁은 범위 |
| `errors.Wrap(nil) == nil` inline 주석 | 다음 리팩토링 시 동일 실수 방지 |

## Open Questions

| 질문 | 추천 | 우선순위 |
|------|------|---------|
| `ChannelJoin` 실패의 실제 원인은? | 이 fix 적용 후 에러 로그 확인하여 별도 이슈 생성 (명시적 오너 지정) | 이 PR merge 후 즉시 — 별도 티켓 필수 |
| 동일 `err` vs `errJoin` 혼동 패턴이 다른 핸들러에도 있는가? | 표준 linter로 탐지 불가. semgrep rule 작성 후 전체 핸들러 스캔 | Phase 2 — 별도 PR |
| fix 적용 후 `ChannelJoin`이 여전히 실패한다면? | 에러 로그 근거로 `bridgeHandler.ChannelJoin` 내부 조사 | Immediate after merge |

## Review Summary (v1 → v2)

리뷰 Round 1, 2에서 지적된 High 항목 2개 반영:

1. **Error propagation 분석 추가** (Round 1 Finding 3, Round 2 Finding d): ARI caller 에러 처리 경로 명시. conf-in 채널 hangup + outgoing leg 연쇄 정리 확인.
2. **Implementation PR 필수 항목 추가** (Round 1 Finding 5, Round 2 Finding e): 단위 테스트, confbridgehandler 파일 스캔, inline 주석 명시.

Medium/Low 항목 처리:
- rollback: 2라인 수정 → `git revert + redeploy` 로 충분, 별도 섹션 불필요
- blast radius: 모든 extension-to-extension inbound call 영향 (100%). 로그로 확인됨.
- log 대소문자: 구현 PR에서 Go 관행(`could not ...`) 에 맞춰 수정 예정

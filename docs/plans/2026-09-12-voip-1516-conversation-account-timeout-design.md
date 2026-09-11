# Design: bin-conversation-manager POST /conversation_accounts (type=line) 간헐적 503 REQUEST_TIMEOUT 수정

- Ticket: VOIP-1516
- GitHub Issue: voipbin/monorepo#1294 (근본원인 정정 코멘트 추가됨)
- Worktree: `~/gitvoipbin/monorepo/.worktrees/VOIP-1516-conversation-account-timeout`
- 작성일: 2026-09-12

## 1. 배경 / 문제

api-validator 정기 테스트에서 `POST /conversation_accounts` (payload `type: "line"`) 호출이 간헐적으로 503 `REQUEST_TIMEOUT`을 반환한다. 3라운드 독립 리뷰 루프(delegate_task fresh reviewer, Round1 CHANGES_REQUESTED / Round2 APPROVE / Round3 APPROVE)로 근본원인을 코드 기준 검증 완료했다.

## 2. 근본원인 (검증 완료, 재론 없음)

RabbitMQ RPC는 프로세스 경계를 넘어 Go `context.Context`(취소 신호/데드라인)를 전달하지 못한다.

- 발신측(`bin-api-manager`): `ConversationV1AccountCreate` → `sendRequestConversation(ctx, ..., 30000, ...)` → `send_request.go:37`에서 `context.WithTimeout(ctx, 30초)`. 이는 발신측 프로세스 내부에서 응답을 기다리는 **로컬 타이머**일 뿐, wire로 전달되지 않는다.
- 수신측(`bin-conversation-manager`): `pkg/listenhandler/main.go:183`에서 `processRequest`가 **`ctx := context.Background()`**(데드라인 없음)를 새로 생성한다. 발신측 30초와 완전히 무관.
- 이 ctx가 `accountHandler.Create`(`pkg/accounthandler/db.go:52`, `h.setup(ctx, ac)`) → `lineHandler.Setup`(`pkg/linehandler/setup.go:19`) → LINE SDK(`vendor/.../linebot/client.go:210-213`, `req.WithContext(ctx)`)까지 그대로 전파된다. `context.Background()`는 절대 취소되지 않으므로, 이 경로에서 실질적으로 유효한 유일한 타임아웃은 `pkg/linehandler/send.go:64`의 `http.Client{Timeout: 60초}`뿐이다.
- 결과: 클라이언트는 발신측 30초 로컬 대기 초과로 503을 받지만, 그 시점에도 conversation-manager 프로세스는 최대 60초까지 LINE 웹훅 등록 → `AccountCreate` DB insert(`db.go:58`) → `Get`(`db.go:63`) → `PublishWebhookEvent`(`db.go:68`)를 계속 진행할 수 있다. 클라이언트에는 실패로 보이는데 서버에는 계정이 뒤늦게 생성되는 경쟁 상태가 존재한다.
- Circuit Breaker는 무관함을 배제 확인(에러 시그니처가 `REQUEST_TIMEOUT`이며 CB open이라면 `INTERNAL`로 나와야 함).

## 3. 스코프 결정

**이번 수정은 `bin-conversation-manager`의 `POST /accounts`(LINE 계정 생성) 경로로 한정한다.**

`pkg/listenhandler/main.go`에서 `ctx := context.Background()` 패턴은 모노레포 전역 30개 서비스 listenhandler에 동일하게 존재하는 표준 패턴이며(grep 확인, 대안 없이 관행적으로 쓰임), 이번 이슈에서 실제로 문제가 되는 것은 그 ctx가 **외부 서드파티 동기 HTTP 호출**(LINE API)까지 전파되는 유일한 케이스라는 점이다. 전역 listenhandler 패턴을 바꾸는 것은 별도의 아키텍처 논의가 필요한 큰 스코프이며, 실측 신호(다른 서비스에서 유사 장애가 보고된 바 없음) 없이 선제적으로 확장하지 않는다. 오버엔지니어링 지양 원칙에 따라 지금은 관측된 실패 경로만 고친다.

## 4. 수정 방안

### 4.1 선택: 방안 1 (수신측 로컬 타임아웃 부여)

`accountHandler.Create`에서 `setup(ctx, ac)`를 호출하기 전에, 발신측 RPC 예산(30초)보다 짧은 로컬 데드라인을 명시적으로 부여한다.

```go
// pkg/accounthandler/db.go, Create() 내부
setupCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
defer cancel()

if errSetup := h.setup(setupCtx, ac); errSetup != nil {
    log.Errorf("Could not setup the account. err: %v", errSetup)
    return nil, errors.Wrap(errSetup, "could not setup the account")
}
```

- 값 선택: 발신측 RPC 예산 30초보다 명확히 짧은 **20초**로 설정한다. (LINE API 정상 응답은 수 초 이내가 통상적이며, 60초 `http.Client.Timeout`은 그대로 안전망으로 유지 — `send.go:64`는 수정하지 않는다.)
- 효과: LINE 응답이 20초를 넘기면 `setupCtx`가 먼저 취소되어 `lineHandler.Setup`이 즉시 에러를 반환하고, `AccountCreate` DB insert 이전 단계에서 실패로 종료된다. **DB insert 이전에 실패하므로 "클라이언트는 실패로 보는데 서버는 계정을 생성한다"는 경쟁 상태가 원천적으로 제거된다.**
- 발신측 30초 vs 수신측 20초 여유(10초)는 RabbitMQ 큐잉/네트워크 왕복 지연을 감안한 버퍼다.
- wire 프로토콜 변경 불필요 (Round 2/3 리뷰로 확인됨).
- 주의: 원래 `ctx`는 `context.Background()`이며 애초에 데드라인이 없다. "발신측 30초 예산을 상속받아 20초로 줄인다"는 것이 아니라, **원래 무기한이던 것에 처음으로 명시적 데드라인을 새로 부여**하는 것이다.

### 4.2 채택하지 않는 대안

- **방안 2(비동기화)**: 계정 생성을 즉시 응답하고 webhook 설정을 별도 비동기 처리로 옮기는 근본적 재설계. 효과는 크지만 API 응답 계약(현재는 생성된 계정을 동기 응답으로 반환) 변경이 필요해 스코프가 크다. 이번 티켓 범위를 벗어나며, 재발 빈도(753건 중 2건)를 감안할 때 지금 시점에 정당화되지 않는다. 향후 유사 패턴(WhatsApp 등)이 반복되면 재검토.
- **전역 listenhandler ctx 패턴 개편**: §3에서 서술한 대로 스코프 아웃.

## 5. 영향 범위

- 변경 파일: `bin-conversation-manager/pkg/accounthandler/db.go` (Create 함수만).
- `lineHandler.Setup`/`Teardown`, LINE SDK 호출부는 변경 없음 — ctx가 이미 파라미터로 전달되므로 상위에서 데드라인을 씌우는 것만으로 하위 전체에 자동 적용됨(Go `context.WithTimeout`의 자식 ctx 취소 전파 표준 동작).
- `account.TypeWhatsApp`(`whatsappHandler.Setup`, `pkg/whatsapphandler/setup.go:12`)는 시그니처가 `Setup(_ context.Context, ac *account.Account) error`로 **ctx를 아예 사용하지 않는다**(provider_data 로컬 검증만 수행, 외부 API 호출 없음). 따라서 이번 수정의 영향을 받지 않는다 — "부가 이득" 주장은 정정한다: 영향 없음이 정확한 서술이다.
- `account.TypeSMS`는 `setup()`에서 no-op이므로 영향 없음.
- **원래 `ctx`(발신측에서 전달된 것처럼 보이지만 실제로는 `pkg/listenhandler/main.go:183`에서 생성된 `context.Background()`, 즉 데드라인이 전혀 없는 컨텍스트)를 `AccountCreate`/`Get`/`PublishWebhookEvent`(58, 63, 68행)에는 그대로 사용한다.** §2에서 "발신측 30초 예산을 쓴다"는 표현은 부정확했으므로 정정한다 — 정확히는 "원래 ctx(무기한, 데드라인 없음)를 그대로 사용하며, 이 구간은 순수 로컬 DB/이벤트 처리이므로 외부 API처럼 무한 대기할 위험이 없다"는 것이 근거다.

## 6. 테스트 계획

- 기존 유닛 테스트(`pkg/accounthandler/db_test.go`)에 `setup()` 호출이 타임아웃되는 경로(context deadline exceeded를 반환하는 mock)에 대한 케이스 추가.
- `go test ./...`, `golangci-lint`는 표준 검증 워크플로우로 실행.
- api-validator 회귀 테스트는 실제 LINE API 호출을 포함하므로 이번 PR로 직접 재현 검증은 어렵다(외부 의존). 코드 레벨 유닛 테스트로 타임아웃 동작을 검증하고, 배포 후 api-validator 정기 실행에서 재발 여부를 모니터링한다.

## 7. 리스크

- 20초 로컬 타임아웃이 정상 LINE 응답 시간보다 지나치게 짧을 경우 정상 요청도 실패시킬 수 있다. 다만 기존에도 60초 안에 실패하던 경로이므로, 20초는 정상 케이스(수 초)에 영향 없고 비정상 지연 케이스만 더 빨리 차단한다.
- 추후 실제 LINE API 응답 지연 분포 데이터가 쌓이면 20초 값을 재조정할 수 있다(현재는 발신측 30초 예산 대비 안전 마진 기준의 추정치).

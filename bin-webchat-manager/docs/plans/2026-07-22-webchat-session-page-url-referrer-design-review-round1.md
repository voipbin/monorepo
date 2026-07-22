# Review: 2026-07-22-webchat-session-page-url-referrer-design.md (Round 1)

이 리뷰는 Round 0(`...review-round0.md`)에서 지적된 두 결함 — (A) §5의 "오버사이즈 URL은 Gin 바인딩 계층에서 400으로 거부된다"는 허위 주장, (B) §7 파일 목록에서 `bin-webchat-manager` 내부 RPC 수신 체인 누락 — 이 이번 개정판에서 실제로 해소되었는지를 코드로 직접 재검증한다.

## 1. §4.4 신규 RPC 체인 목록의 완결성 재검증

문서 §4.4가 나열한 체인을 실제 코드와 하나씩 대조했다:

- `bin-api-manager/server/webchat_sessions.go` (`PostWebchatSessions`, 65-101행) — `c.BindJSON(&req)` 후 `h.serviceHandler.WebchatSessionCreate(c.Request.Context(), a, widgetID)` 호출. 문서 서술과 일치.
- `bin-api-manager/pkg/servicehandler/main.go:906` — `WebchatSessionCreate(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID) (*wcsession.WebhookMessage, error)`. 문서 서술과 일치.
- `bin-api-manager/pkg/servicehandler/webchat_session.go:138-214` — `WebchatSessionCreate` 구현, 205행에서 `h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID)` 호출. 문서 서술과 일치.
- `bin-common-handler/pkg/requesthandler/main.go:1536` — `WebchatV1SessionCreate(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID) (*wcsession.Session, error)`. 라인 번호까지 정확.
- `bin-common-handler/pkg/requesthandler/webchat_session.go:17-23` — `WebchatV1SessionCreate`가 `&wcrequest.V1DataSessionsPost{CustomerID: customerID, WidgetID: widgetID}`만 구성. 문서가 "Round 1 신규 발견"으로 지목한 그대로, 현재 `PageURL` 필드가 없다. 확인됨.
- `bin-webchat-manager/pkg/listenhandler/models/request/v1_sessions.go` — `V1DataSessionsPost{CustomerID uuid.UUID; WidgetID uuid.UUID}` (11-12행), `PageURL` 없음. 확인됨.
- `bin-webchat-manager/pkg/listenhandler/v1_sessions.go:33` — `processV1SessionsPost`가 정확히 `h.sessionHandler.Create(ctx, req.CustomerID, req.WidgetID)`를 호출. 문서가 인용한 그대로.
- `bin-webchat-manager/pkg/sessionhandler/main.go:21` — `SessionHandler` 인터페이스 `Create(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID) (*session.Session, error)`, `pageURL` 파라미터 없음. 확인됨. `//go:generate mockgen`(3행) 지시어도 실재하여, `mock_main.go` 재생성 필요성 서술이 타당함을 뒷받침한다.
- `bin-webchat-manager/pkg/sessionhandler/create.go:29` — `Create(ctx, customerID, widgetID)` 시그니처, 문서 서술과 일치.

체인 상의 추가 은닉 지점(캐시 계층)도 확인했다: `bin-webchat-manager/pkg/cachehandler`의 `SessionSet`/`SessionGet`은 `*session.Session` 포인터를 통째로 직렬화/역직렬화하며 필드별 리터럴 재구성을 하지 않으므로, `session.go`에 `PageURL` 필드만 추가되면 캐시 계층은 자동으로 이를 포함한다 — 문서가 캐시 계층을 별도로 언급하지 않은 것은 결함이 아니라 정확한 판단이다.

**결론: §4.4가 나열한 체인은 실제 코드의 모든 중간 지점을 빠짐없이 커버한다. Round 0 지적 (B)는 실질적으로 해소되었다.**

## 2. §7 파일 목록이 §4.4의 발견사항을 반영하는지

§7 "bin-common-handler" 섹션에 `pkg/requesthandler/main.go`(1536행 명시), `pkg/requesthandler/webchat_session.go`, `pkg/requesthandler/mock_main.go` 3개가 추가되어 있고, "bin-webchat-manager" 섹션에 `pkg/listenhandler/models/request/v1_sessions.go`, `pkg/listenhandler/v1_sessions.go`, `pkg/sessionhandler/main.go`, `pkg/sessionhandler/mock_main.go` 4개가 추가되어 있다. §4.4에서 식별된 7개 신규 파일과 정확히 1:1 대응한다. 누락 없음.

## 3. §5 수정된 길이 검증 주장의 타당성 — 새로운 결함 발견

§5는 이제 "Gin 바인딩 계층에서 400 거부"라는 허위 주장을 버리고, `pkg/servicehandler/webchat_session.go`의 `WebchatSessionCreate`에 명시적 `validatePageURL` 체크를 추가하는 것으로 수정했다. 이 자체 방향은 타당하고, `bin-api-manager/server/webchat_sessions.go`가 정말 `c.BindJSON`만 호출한다는 Round 0의 관찰도 재확인했다(위 §1).

그런데 §5는 이렇게 쓰고 있다:

> "a private `validatePageURL(pageURL string) error` returning `serviceerrors`-wrapped error if `len(pageURL) > 2048` ... returning the existing `serviceerrors.ErrBadRequest`-style pattern rather than a new error type"

`bin-api-manager/pkg/serviceerrors/sentinels.go`를 직접 읽어 확인한 결과, **`serviceerrors.ErrBadRequest`라는 심볼은 존재하지 않는다.** 해당 패키지의 실제 sentinel 목록은:

```go
ErrPermissionDenied, ErrNotFound, ErrAuthenticationRequired,
ErrDirectAccessNotSupported, ErrInvalidArgument, ErrInternal,
ErrIdentityVerificationRequired, ErrStateInvalid, ErrServiceUnavailable,
ErrInsufficientBalance
```

`ErrBadRequest`는 `bin-common-handler/pkg/requesthandler` 패키지에 있는 별개의 심볼(백엔드가 bare 4xx status만 반환했을 때를 매핑하는 RPC 계층 sentinel, `server/error_translate.go:74`에서 사용)이며, `serviceerrors` 패키지 소속이 아니다. 문서가 참고 사례로 직접 지목한 `validateDelegateReason`(`pkg/servicehandler/auth_delegate.go:138-151`, §5가 "reject-don't-truncate precedent"의 근거로 인용한 바로 그 함수) 자체도 `serviceerrors.ErrBadRequest`가 아니라 `serviceerrors.ErrInvalidArgument`를 사용한다 (`auth_delegate.go:71`: `fmt.Errorf("%w: %v", serviceerrors.ErrInvalidArgument, err)`). 그리고 `ErrInvalidArgument`는 `server/error_translate.go:68-69`에서 400/`INVALID_ARGUMENT`로 매핑된다 — 즉 문서가 의도한 "400으로 거부" 동작 자체는 `ErrInvalidArgument`로 정확히 달성되지만, 문서에 적힌 심볼 이름 `ErrBadRequest`는 존재하지 않아 컴파일되지 않는다.

이는 Round 0에서 지적된 (A)와 같은 계열의 결함(사실이 아닌 코드 근거로 구현 지침을 내림)이 정정된 절 안에서 형태를 바꿔 재발한 것이다. 심각도는 Round 0의 원 결함보다 낮다 — 구현자가 문서가 직접 인용한 `validateDelegateReason`의 실제 코드를 보면 즉시 `ErrInvalidArgument`가 맞는 심볼임을 알 수 있고, 이는 하나의 오탈자 수준 수정으로 해결된다. 그러나 "형식적 승인 금지"라는 이번 리뷰의 요구 수준에 비추어 이 부정확한 인용은 반드시 고쳐야 한다: §5의 `serviceerrors.ErrBadRequest`를 `serviceerrors.ErrInvalidArgument`로 정정할 것.

## 4. Round 0에서 이미 검증된 항목 재확인 — 새로 깨진 부분 없음

- **코드 인용 정확성**: `session.go`, `field.go`, `webhook.go`, `dbhandler/session.go`를 다시 읽었다. 이번 개정에서 이 파일들에 대한 문서 서술(§4.3)에 변경이 없고, 실제 코드도 Round 0 확인 시점과 동일하다 — `PageURL` 필드는 아직 추가되지 않은 설계 단계 그대로다. 문제 없음.
- **Peer/Local 기각 근거(§6)**: `sessionhandler/create.go:77-78`의 `self`/`peer` 지역변수 재확인, 문서 서술과 일치. 변경 없음, 문제 없음.
- **referrer 선택 근거(§3)**: `window.location.href` vs `document.referrer` 논리는 이번 라운드에서 수정되지 않았고, 재검토해도 여전히 타당하다.
- **프라이버시(§4.6)**: 서술도 변경 없음, 논리적 결함 없음.
- **iframe/scheme 엣지케이스(§5 나머지 항목)**: `LogoURL`의 `https://`만 허용 주석을 `bin-webchat-manager/models/widget/widget.go:99`에서 재확인(`// https URL only`) — §5가 대조 근거로 인용한 그대로. 문제 없음.

## 5. 결론

Round 0의 두 핵심 결함(A: 허위 400 검증 주장, B: RPC 체인 파일 누락) 중 **B는 완전히 해소**되었다 — §4.4/§7의 신규 체인 목록은 실제 코드의 모든 중간 파일을 정확히 커버한다. **A도 방향은 올바르게 수정**되었으나(명시적 servicehandler 레벨 길이 체크 도입 자체는 타당), 그 수정문 안에서 **존재하지 않는 심볼 `serviceerrors.ErrBadRequest`를 실제 검증 패턴인 것처럼 인용**하는 새로운 사실 오류가 발생했다. 문서가 직접 참고 사례로 지목한 `validateDelegateReason`이 실제로 사용하는 심볼은 `serviceerrors.ErrInvalidArgument`이다.

이 오류는 Round 0의 원 결함들보다 훨씬 경미하고(구현자가 인용된 참고 함수 코드를 보면 즉시 바로잡을 수 있는 심볼명 오기), 설계의 방향성이나 RPC 체인 완결성 같은 구조적 문제는 아니다. 하지만 "코드 근거로 제시된 심볼이 실재하는가"는 이 리뷰 라운드의 핵심 검증 대상이었고, 실제로 실재하지 않는 것으로 확인되었으므로 정정 없이 승인할 수 없다.

**요청 사항 (구현 착수 전 수정 필요):**
1. §5에서 `serviceerrors.ErrBadRequest` → `serviceerrors.ErrInvalidArgument`로 정정 (validateDelegateReason이 실제로 사용하는 심볼과 일치시킬 것).

VERDICT: CHANGES_REQUESTED

# Review: 2026-07-22-webchat-session-page-url-referrer-design.md (Round 2)

이 리뷰는 Round 1(`...review-round1.md`)에서 지적된 유일한 결함 — §5가 인용한
`serviceerrors.ErrBadRequest` 심볼이 실제로는 존재하지 않으며, `validateDelegateReason`이
실제로 사용하는 심볼은 `serviceerrors.ErrInvalidArgument`라는 지적 — 이 이번 개정판에서
해소되었는지, 그리고 문서 전체에 잔존/신규 결함이 없는지를 코드와 직접 대조해
처음부터 끝까지 재검증한다.

## 1. Round 1 지적사항의 해소 여부

`bin-api-manager/pkg/serviceerrors/sentinels.go`를 다시 읽어 확인:

```go
ErrPermissionDenied, ErrNotFound, ErrAuthenticationRequired,
ErrDirectAccessNotSupported, ErrInvalidArgument, ErrInternal,
ErrIdentityVerificationRequired, ErrStateInvalid, ErrServiceUnavailable,
ErrInsufficientBalance, ErrCaseClosed, ErrCaseDestinationNotAssociated,
ErrCaseSourceNotOwned
```

`ErrBadRequest`는 여전히 존재하지 않는다(Round 1 확인과 동일). 문서 §5(287-304행,
"**Round 2 correction:**" 단락)는 이제 다음과 같이 서술한다:

> "confirmed against `bin-api-manager/pkg/serviceerrors/sentinels.go` that the correct
> sentinel is `ErrInvalidArgument = stderrors.New("invalid argument")` ... an earlier
> draft of this section incorrectly named a nonexistent `ErrBadRequest` symbol."

`ErrInvalidArgument = stderrors.New("invalid argument")` 문구를 sentinels.go 19행과
글자 그대로 대조 — 정확히 일치한다.

문서가 근거로 재확인한 `auth_delegate.go`도 다시 읽었다:
- 64-72행: `validateDelegateReason(reason)`가 에러를 반환하면
  `fmt.Errorf("%w: %v", serviceerrors.ErrInvalidArgument, err)`로 래핑 — 문서가
  "the caller-facing wrap at the `servicehandler` boundary is `ErrInvalidArgument`"라고
  서술한 그대로. 라인 번호(§5는 "auth_delegate.go:138-151"을 `validateDelegateReason`
  함수 자체의 위치로 인용)도 실제 함수 정의(137-151행, 0-index 보정 감안) 위치와 일치한다.
- `validateDelegateReason` 함수 자체(138-151행)는 `fmt.Errorf`로 plain 에러만 반환하고
  `serviceerrors` sentinel로 래핑하지 않는다는 §5의 서술("`auth_delegate.go`'s error
  messages use plain `fmt.Errorf`, but the caller-facing wrap ... is `ErrInvalidArgument`")도
  정확 — 래핑은 호출부(71행)에서 일어난다.

`server/error_translate.go:68`에서 `ErrInvalidArgument`가 400으로 매핑됨을 재확인
(`case stderrors.Is(err, serviceerrors.ErrInvalidArgument):`) — §5가 의도한 "400으로
거부" 동작이 정정된 심볼로 실제로 달성됨을 뒷받침한다.

**결론: Round 1의 유일한 지적사항은 완전히 해소되었다. 존재하지 않는 심볼 인용은
남아있지 않다.**

## 2. 문서 전체 재검증 (신규/잔존 결함 탐색)

Round 0/Round 1에서 이미 검증되었고 이번 개정에서 서술이 바뀌지 않은 항목은
코드 재확인만 하고 새로 나열하지 않는다(§3 referrer 선택 근거, §4.6 프라이버시,
§6 Peer/Local 기각 근거, iframe/scheme 엣지케이스). 아래는 이번 라운드에서 처음부터
끝까지 다시 짚은 항목이다.

- **§4.1 client.js 인용**: `_doStart()`의 `POST /webchat_sessions` 호출부를
  `webchat-widget-runtime/client.js:316-319`에서 재확인 — 현재 body는
  `{ widget_id: this.resourceId }`만 담고 있으며, 문서가 제안하는 `page_url` 필드는
  아직 추가되지 않은 설계 단계 그대로다(설계 문서이므로 당연히 미구현 상태). 문서가
  인용한 라인 번호(315-319)도 실제 코드(315행 로그, 316-319행 fetch 호출)와 일치한다.
- **§4.2 OpenAPI 인용**: `bin-openapi-manager/openapi/paths/webchat_sessions/main.yaml`을
  전체 재확인 — `post:` 블록의 requestBody가 현재 `widget_id`만 정의하고 있음을
  확인(36-51행), 문서가 새 필드로 추가하려는 `page_url`은 아직 없다. §7의
  `bin-openapi-manager` 파일 목록(`openapi/paths/webchat_sessions/main.yaml`,
  `openapi/openapi.yaml`)도 정확 — `openapi.yaml:2510`에 `WebchatManagerSession:`
  스키마가 실재함을 확인했다(§7이 "WebchatManagerSession schema" 변경 대상으로 지목한
  스키마).
- **§4.3 backend 모델 인용**: `models/session/session.go`(19-28행 필드 목록),
  `models/session/field.go`(6-22행), `models/session/webhook.go`(13-38행)를 모두
  재확인 — 문서가 "PageURL 필드 없음" 전제로 서술한 그대로, 세 파일 모두 현재
  `PageURL`/`FieldPageURL` 관련 내용이 전혀 없다. `dbhandler/session.go`에 대한
  §4.3의 "no logic change expected -- PrepareFields/GetDBFields are struct-tag-driven"
  주장도 재확인: `SessionCreate`(42행 `PrepareFields(s)`)와
  `sessionGetFromDB`(102행 `GetDBFields(&session.Session{})`)가 정말로 구조체
  전체를 리플렉션 기반으로 스캔하며, `PageURL` 같은 신규 필드가 struct tag만
  갖추면 이 두 함수의 코드 변경 없이 자동으로 포함된다는 판단은 타당하다.
- **§4.4 RPC 체인 재검증**: Round 1에서 검증된 7개 신규/변경 파일을 이번에도 코드
  대 코드로 재대조했다.
  - `requesthandler/webchat_session.go:16-23` — `WebchatV1SessionCreate`가
    여전히 `customerID, widgetID uuid.UUID`만 받고 `&wcrequest.V1DataSessionsPost{
    CustomerID: customerID, WidgetID: widgetID}`만 구성 (PageURL 없음). 문서 서술과 일치.
  - `listenhandler/models/request/v1_sessions.go:10-13` — `V1DataSessionsPost{
    CustomerID uuid.UUID; WidgetID uuid.UUID}`, PageURL 없음. 일치.
  - `listenhandler/v1_sessions.go:33` — `processV1SessionsPost`가 정확히
    `h.sessionHandler.Create(ctx, req.CustomerID, req.WidgetID)` 호출. 일치.
  - `sessionhandler/main.go:20-26` — `SessionHandler` 인터페이스의 `Create(...)`
    시그니처, pageURL 파라미터 없음. `//go:generate mockgen`(3행) 지시어 실재 확인.
    일치.
  - `requesthandler/main.go:1536` — `WebchatV1SessionCreate(ctx context.Context,
    customerID uuid.UUID, widgetID uuid.UUID) (*wcsession.Session, error)`.
    문서가 인용한 라인 번호까지 정확히 일치.
  - `server/webchat_sessions.go:65-101`(`PostWebchatSessions`) — `c.BindJSON(&req)`
    후 `h.serviceHandler.WebchatSessionCreate(c.Request.Context(), a, widgetID)` 호출
    (93행). 문서 서술과 일치.
  - `sessionhandler/create.go` — `Create()` 시그니처(29행)가 여전히
    `(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID)`이고, 구조체
    리터럴(40-48행)이 `WidgetID`/`Status`만 설정. 문서가 "stored on the
    `session.Session{}` literal at construction (line 40-48)"이라 인용한 라인
    번호와 정확히 일치.
  누락된 중간 파일 없음. Round 0/1에서 지적된 결함(A, B)의 재발 징후 없음.
- **§5 iframe/scheme 엣지케이스**: `widget.go:99`의 `LogoURL` 주석
  (`// https URL only`)을 재확인 — §5가 "differs from `ThemeConfig.LogoURL`'s
  existing `https://`-only validation"의 근거로 인용한 그대로.
- **§6 Peer/Local 기각 근거**: `sessionhandler/create.go:77-78`의 `self`/`peer`
  지역변수를 재확인 — `self := commonaddress.Address{Type: commonaddress.TypeWebchat,
  Target: widgetID.String()}`, `peer := commonaddress.Address{Type:
  commonaddress.TypeWebchat, Target: id.String()}`. 문서가 "Peer.Target would be
  Session.ID itself", "Local.Target would be Widget.ID's string form"이라 서술한
  그대로 — Peer는 `id`(세션 자기 자신의 PK), Local(self)은 `widgetID`(이미
  `Session.WidgetID`로 저장되는 값)와 일치. 기각 논리에 결함 없음.
- **§7 파일 목록 완결성**: monorepo-javascript 쪽 인용 파일도 실재 확인 —
  `square-admin/src/webchat-widget-runtime/client.js`,
  `square-admin/src/views/webchat_widgets/message_timeline.js`(§4.5가 서술한 대로
  `session={selectedSession}` prop을 받아 트랜스크립트를 렌더링하는 구조 확인,
  27행 `const WebchatMessageTimeline = ({ session, onOpenChange })`),
  `square-admin/src/views/webchat_widgets/sessions_list.js`가 모두 실재.
  bin-common-handler 섹션(`pkg/requesthandler/main.go`, `webchat_session.go`,
  `mock_main.go`)과 bin-webchat-manager 섹션(`pkg/listenhandler/models/request/
  v1_sessions.go`, `pkg/listenhandler/v1_sessions.go`, `pkg/sessionhandler/main.go`,
  `pkg/sessionhandler/mock_main.go`)이 §4.4에서 식별된 7개 신규 파일과 여전히
  1:1 대응한다. 누락 없음.
- **환각 인용 전수 재점검**: 문서 전체에서 코드/라인 번호를 직접 인용하는 모든
  지점(§4.1 client.js:315-319, §4.3 create.go:40-48, §4.4의 각 파일+라인,
  §5 sentinels.go 인용, §5 widget.go:99, §6 create.go:77-78, §7의 openapi.yaml
  `WebchatManagerSession` 스키마)을 하나씩 실제 코드와 대조했고, 모두 실재하며
  서술과 일치한다. 새로 인용된 심볼/함수/파일 중 존재하지 않는 것은 없다.

## 3. 결론

Round 1의 유일한 결함(존재하지 않는 `serviceerrors.ErrBadRequest` 심볼 인용)은
정확히 `serviceerrors.ErrInvalidArgument`로 정정되었고, 이 정정이 `sentinels.go`의
실제 정의 및 `auth_delegate.go`의 실제 사용 패턴과 글자 그대로 일치함을 코드로
확인했다. Round 0의 두 결함(허위 400 검증 주장, RPC 체인 파일 누락)도 이전 라운드에서
해소된 상태가 이번 재검증에서도 유지되고 있다. 문서를 처음부터 끝까지 재점검한
결과, §4.4 RPC 체인의 모든 중간 파일, §7 파일 목록, §5/§6의 코드 근거 인용 모두
현재 코드베이스와 정확히 일치하며 새로운 사실 오류나 논리적 비약은 발견되지 않았다.

VERDICT: APPROVED

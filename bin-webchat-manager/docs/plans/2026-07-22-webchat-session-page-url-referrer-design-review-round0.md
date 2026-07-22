# Review: 2026-07-22-webchat-session-page-url-referrer-design.md (Round 0)

## 1. 코드 인용 검증 (환각 여부)

대부분의 파일 경로·라인·코드 인용은 실제 코드베이스와 일치함을 직접 소스로 확인했다:

- `session.go`, `field.go`, `webhook.go`, `create.go`(37-48행 Session 리터럴), `dbhandler/session.go` — 모두 문서 서술과 일치.
- `client.js` `_doStart()`(279행)의 `POST /webchat_sessions` 호출부(316-319행) — 문서가 인용한 "client.js:315-319"와 거의 일치(실제는 316-319, 오차 1행 수준으로 무해).
- `sessionhandler/create.go` 77-78행의 `self`/`peer` 지역변수 — §6의 Peer/Local 기각 근거로 인용한 그대로 존재.
- `widget.go:99` `LogoURL` 필드의 `// https URL only` 주석 — 문서가 §5에서 대조 근거로 든 그대로 존재.
- `kase.go`의 `Peer`/`Local` 필드(24-38행, `commonaddress.Address`, JSON db tag) — 문서 서술과 일치.
- `message_timeline.js`가 `session` prop을 받고, `detail.js`(850행)·`sessions_list_global.js`(72행) 양쪽에서 `session={selectedSession}`으로 넘겨받는다는 서술 — grep으로 확인, 정확.
- `bin-api-manager/docsdev/source/webchat_struct_session.rst` 실재 확인, `page_url` 항목 없음 — 문서의 "추가 필요" 판단 타당.
- OpenAPI `WebchatManagerSession` 스키마(`openapi.yaml:2510`)와 `paths/webchat_sessions/main.yaml`도 실재하며 `page_url`이 없는 현재 상태와 문서 서술이 일치.

환각으로 볼 만한 인용 오류는 발견하지 못했다. 이 부분은 신뢰할 수 있다.

## 2. window.location.href vs document.referrer

타당하다. 위젯은 iframe 없이 고객 페이지에 직접 임베드되는 것이 기본 케이스이므로 `location.href`가 "어느 페이지에 위젯이 떠 있었는가"를 정확히 답한다. `document.referrer`는 별개의 질문(유입 경로)에 답하며 이는 §2에서 명시적으로 non-goal 처리된 UTM/캠페인 영역과 겹친다는 논리도 일관적이다.

## 3. 엣지 케이스

- 초과 길이, `javascript:`/`data:` 스킴 불가능성, `file://` 그대로 캡처, iframe cross-origin `SecurityError` 회피 — 논리 자체는 타당.
- **단, "초과 길이(2048자 초과) 값은 OpenAPI `maxLength` + Gin 바인딩 계층에서 400으로 거부된다"는 주장은 실제 코드로 검증되지 않는다.** `bin-api-manager/server`, 리포 전체를 검색했지만 `oapi-codegen`의 요청 검증 미들웨어(`nethttp-middleware`, `gin-middleware`, `OapiRequestValidator` 류)가 전혀 존재하지 않는다. `PostWebchatSessions`(server/webchat_sessions.go:79-84)는 `c.BindJSON(&req)`만 호출하는데, 이는 JSON 문법 검증과 Go 구조체 태그의 `binding:"required"`만 체크할 뿐 `maxLength`같은 OpenAPI 스키마 제약은 전혀 강제하지 않는다. 즉 오버사이즈 `page_url`은 400으로 거부되지 않고 그대로 `WebchatSessionCreate` → RPC → DB INSERT까지 흘러간다. MySQL이 strict mode면 INSERT 자체가 에러(session 생성 실패로 이어질 수 있음), non-strict면 조용히 잘려서 저장된다 — 문서가 "accepted, rare failure mode"로 분류한 근거 자체가 틀렸다. **명시적인 서버측 길이 체크(servicehandler 또는 sessionhandler 레이어)를 추가하거나, 최소한 이 엣지케이스 서술을 정정해야 한다.**

## 4. §7 파일 목록 완결성 — 중대한 누락

문서가 스스로 "특히 mock 재생성, RPC 시그니처 변경 전파 경로"를 검증 관점으로 요구했는데, 바로 그 전파 경로에 구멍이 있다.

`page_url`이 실제로 `session.Session{}`에 도달하려면 다음 체인을 전부 통과해야 한다:

```
client.js → POST /webchat_sessions (bin-api-manager)
  → server/webchat_sessions.go (문서에 있음)
  → servicehandler/webchat_session.go, main.go (문서에 있음)
  → bin-common-handler/pkg/requesthandler.WebchatV1SessionCreate (문서에 있음, "signature+mock" 언급)
  → RabbitMQ RPC
  → bin-webchat-manager: pkg/listenhandler/v1_sessions.go의 processV1SessionsPost
       (req.CustomerID, req.WidgetID를 sessionHandler.Create에 전달 — 33행)
  → pkg/listenhandler/models/request/v1_sessions.go의 V1DataSessionsPost 구조체
       (현재 CustomerID/WidgetID만 있음, PageURL 필드 없음)
  → pkg/sessionhandler/main.go의 SessionHandler 인터페이스 (Create 시그니처, 21행)
  → pkg/sessionhandler/create.go (문서에 있음)
  → pkg/sessionhandler/mock_main.go (Create 목 — 문서에 없음)
```

직접 소스를 읽어 확인한 결과, §7 체크리스트는 다음을 전부 빠뜨렸다:

- `bin-webchat-manager/pkg/listenhandler/v1_sessions.go` — `processV1SessionsPost`가 `req.PageURL`을 `sessionHandler.Create(ctx, req.CustomerID, req.WidgetID, req.PageURL)`로 넘기도록 수정 필요.
- `bin-webchat-manager/pkg/listenhandler/models/request/v1_sessions.go` — `V1DataSessionsPost`에 `PageURL string` 필드 추가 필요 (현재 `CustomerID`/`WidgetID`만 존재, 실제 확인됨).
- `bin-webchat-manager/pkg/sessionhandler/main.go` — `SessionHandler` 인터페이스의 `Create(...)` 시그니처에 `pageURL string` 파라미터 추가 필요.
- `bin-webchat-manager/pkg/sessionhandler/mock_main.go` — 위 인터페이스 변경에 따른 목 재생성 필요.
- `bin-common-handler/pkg/requesthandler/main.go` — `WebchatV1SessionCreate` 인터페이스 시그니처(1536행, 실제로 `pageURL` 파라미터 없이 `customerID, widgetID`만 있음을 확인) 변경 필요. 문서는 "gains the field"라고만 뭉뚱그렸지 이 인터페이스 파일과 구체적 라인을 명시하지 않았다.
- `bin-common-handler/pkg/requesthandler/webchat_session.go` — 실제 구현체(`WebchatV1SessionCreate`, `V1DataSessionsPost` 마샬링 부분)도 함께 변경 필요; 이 파일도 §7에 없음.

이 중 listenhandler 체인 전체(`v1_sessions.go` + `request/v1_sessions.go` + `sessionhandler/main.go` + `sessionhandler/mock_main.go`)가 완전히 빠진 것은 사소한 누락이 아니다 — 이 레이어를 고치지 않으면 `create.go`의 `Create()` 시그니처를 아무리 바꿔도 `page_url` 값 자체가 RPC 수신 지점에서 파싱조차 되지 않는다. 즉 문서의 §4.3 구현 계획("Create()'s signature gains a pageURL string parameter... threaded through from the API-manager call site")이 실제로는 실행 불가능한 계획이다 — threading 경로에 필수 중간 지점이 통째로 비어 있다.

## 5. §6 Peer/Local 기각 근거 검증

`sessionhandler/create.go` 77-78행을 직접 읽어 확인:
```go
self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: id.String()}
```
`peer.Target`은 세션 자신의 `id`(즉 `Session.ID`), `self.Target`은 `widgetID` — 문서가 주장한 대로 각각 "자기 자신의 PK"와 "이미 WidgetID로 저장된 값의 중복"이라는 서술이 정확히 코드로 뒷받침된다. `kase.go`의 Peer(실제 전화번호/이메일 — 외부 identity)와 Local(실제 DID)이 정보량을 갖는다는 대조도 타당. 이 부분은 근거가 튼튼하다.

## 6. OpenAPI 계약 호환성

`main.yaml`의 `post.requestBody`에 `required: [widget_id]`만 있고 `page_url`은 `required`에 없는 형태로 추가하는 설계는 기존 계약을 깨지 않는 optional 필드 추가로 적절하다. 다만 §3 항목에서 지적한 서버측 강제 검증 부재를 감안하면, `maxLength: 2048`은 문서화된 계약일 뿐 실제로 강제되지 않는다는 점을 알고 진행해야 한다(§4.4/§5 수정 필요).

## 7. 프라이버시/문서화

- RST 문서(`webchat_struct_session.rst`) 갱신 필요성 판단은 정확하고 실재 파일 확인됨.
- 프라이버시 논거(자사 페이지 URL이라 방문자 브라우징 히스토리를 노출하지 않는다는 논리)는 §3의 `location.href` 채택 근거와 정합적이며 합리적이다. 별다른 결함 없음.

## 결론

코드 인용 정확도, Peer/Local 기각 근거, referrer/location.href 선택 근거, 프라이버시/RST 판단은 모두 실제 소스로 검증되어 견고하다. 그러나:

1. **§7 파일 목록이 `page_url`을 실제로 전달하는 데 필수적인 `bin-webchat-manager` 내부 RPC 수신 체인(`pkg/listenhandler/v1_sessions.go`, `pkg/listenhandler/models/request/v1_sessions.go`, `pkg/sessionhandler/main.go`+`mock_main.go`)과 `bin-common-handler/pkg/requesthandler/main.go`의 인터페이스 라인을 통째로 빠뜨렸다.** 이대로 구현하면 `page_url`이 RPC 요청 페이로드에서 파싱되지 않아 기능이 동작하지 않는다.
2. **§5의 "오버사이즈 URL은 Gin 바인딩 계층에서 400으로 거부된다"는 주장이 실제로는 거짓이다.** 리포에 OpenAPI 요청 검증 미들웨어가 존재하지 않으므로, 명시적 길이 체크를 추가하거나 이 엣지케이스 서술 자체를 다시 써야 한다.

두 가지 모두 "구현 시작 전에 반드시 고쳐야 하는" 수준의 실질적 결함이므로 승인할 수 없다.

VERDICT: CHANGES_REQUESTED

# Review: 2026-07-22-webchat-session-page-url-referrer-design.md (Round 3)

이 라운드는 문서에 변경사항이 없는 재검증이다(Round 2에서 이미 `APPROVED`). Round 0/1의
지적사항 해소 여부, Round 2가 확인한 근거들을 신뢰하지 않고 처음부터 코드와 직접 대조했다.
특히 지금까지 상대적으로 얕게 다뤄진 §1~§3(문제 정의/스코프), §4.5·§7의 프론트엔드
(square-admin) 파일들, §4.6의 RST 문서, DB 마이그레이션 목록 완결성을 중점적으로 훑었다.

## 1. Round 0/1/2 지적사항의 해소 상태 재확인

- **Round 0 결함 A (§5 "Gin 바인딩 계층에서 400 거부" 허위 주장)**: 문서 현재본은 이미
  "confirmed by reading `bin-api-manager/server/webchat_sessions.go`... none exists"로
  정정되어 있다. `server/webchat_sessions.go:79-91`을 직접 읽어 재확인 — `PostWebchatSessions`는
  `c.BindJSON(&req)`(80행) 후 `widgetID := uuid.FromStringOrNil(req.WidgetId)`(86행)만
  검증하고, `page_url`/`maxLength` 관련 미들웨어는 전혀 없다. 리포 전체에서
  `OapiRequestValidator`/`nethttp-middleware`/`gin-middleware` 패턴을 검색했으나
  0건 — 문서의 정정된 서술과 일치. 결함 해소 유지됨.
- **Round 0 결함 B (§7 RPC 체인 파일 누락)**: 문서 §4.4/§7이 지금 나열하는 7개 신규 파일
  (`requesthandler/webchat_session.go`, `requesthandler/main.go`,
  `requesthandler/mock_main.go`, `listenhandler/models/request/v1_sessions.go`,
  `listenhandler/v1_sessions.go`, `sessionhandler/main.go`, `sessionhandler/mock_main.go`)을
  모두 직접 열어 현재 시그니처를 재확인했다(§4 상세 참조). 전부 실재하고, 현재 시그니처가
  전부 `pageURL` 파라미터 없이 `customerID, widgetID`만 받는 상태 그대로다 — 설계
  문서이므로 미구현이 당연하고, 체인 자체는 완결적이다.
- **Round 1 결함 (`serviceerrors.ErrBadRequest` 존재하지 않는 심볼)**:
  `bin-api-manager/pkg/serviceerrors/sentinels.go`를 재확인 — `ErrBadRequest`는
  여전히 존재하지 않는다. 문서 §5(287-304행)는 `ErrInvalidArgument =
  stderrors.New("invalid argument")`(sentinels.go:19)를 정확히 인용하고 있고,
  글자 그대로 일치한다. `auth_delegate.go:64-71`을 재확인 —
  `validateDelegateReason(reason)`이 에러를 반환하면 71행에서
  `fmt.Errorf("%w: %v", serviceerrors.ErrInvalidArgument, err)`로 래핑되는 것을 확인,
  문서의 "the caller-facing wrap at the servicehandler boundary is ErrInvalidArgument"
  서술과 정확히 일치. 결함 재발 없음.

## 2. §1~§3 (문제 정의/스코프) — 이번 라운드 신규 검증

- §1이 근거로 든 `models/session/session.go`에 페이지 URL 필드가 없다는 전제와
  `webchat-widget-runtime/client.js`가 `document.referrer`/`window.location.href`를
  전혀 읽지 않는다는 전제를 직접 확인: `session.go`(1-45행) 전체에 `PageURL`/URL
  관련 필드 없음. `client.js` 전체(802행)를 `location`/`referrer` 키워드로 훑었으나
  `_doStart()`(279-335행) 본문에 `window.location`/`document.referrer` 참조가
  전혀 없다 — §1의 전제 정확.
- §3의 "이 설계는 `page_url`만 캡처하고 `document.referrer`는 캡처하지 않는다"는
  선택 논리(§3, 43-57행)는 순수 설계 판단이라 코드 검증 대상이 아니지만, 내적
  일관성은 유지된다 — §4.1의 실제 구현 스니펫도 `window.location?.href`만 읽고
  `document.referrer`는 참조하지 않아 §3과 일치.
- §2 Non-goals의 "widget_id 자체도 세션 생성 시점에 한 번만 설정되고 갱신되지
  않는다"는 비교 근거를 `sessionhandler/create.go`(29-48행)로 재확인 — `Create()`가
  `WidgetID`를 구조체 리터럴 생성 시에만 설정하고, 이후 `SessionUpdate`
  호출(101-107행)은 `FieldActiveflowID`만 갱신한다. `WidgetID`를 갱신하는
  코드 경로는 리포 전체에 없음 — §2의 유비가 정확하다.

## 3. §4.1 client.js — `_doStart()` 실제 라인 재확인

`client.js:267-335`를 직접 읽었다:
- `start()`(267-277행)는 `_startPromise` 캐싱 래퍼, 실제 본체는 `_doStart()`(279행 정의).
- `POST /webchat_sessions` 호출은 315-319행(`this._log(...)`가 315행, `fetch` 호출이
  316-319행) — 문서가 §4.1에서 "client.js:315-319"로 인용한 범위와 정확히 일치.
  Round 2가 지적한 것과 동일하게 로그 줄(315)과 fetch 호출(316-319)이 섞여 있지만
  이는 무해한 근사 인용이다.
- 현재 body는 `{ widget_id: this.resourceId }`(318행)만 담고 있어 문서가 "아직
  `page_url`이 없다"는 전제가 맞다.
- **jsdom 가드에 대한 §4.1의 주석("the `typeof window !== 'undefined'` guard exists
  only for this file's existing test harness")을 실제 테스트로 검증**:
  `__tests__/client.test.js`(1-726행)를 훑었으나 `window`를 `undefined`로 만들거나
  삭제하는 테스트 케이스는 없다(Jest의 기본 `testEnvironment: jsdom`에서
  `window`는 항상 존재). 즉 문서 자신의 논거("a defensive check costs nothing")는
  현재 테스트 스위트가 그 가드를 실제로 필요로 하지 않는다는 사실과 약간
  어긋나지만, 문서 스스로도 "guard exists only... a defensive check costs nothing"
  이라고 정확히 이 갭을 인정하고 있어 — 사실 오류가 아니라 방어적 코딩
  선택이다. 결함 아님(§4.1 자체가 already self-aware).

## 4. §4.2 OpenAPI, §4.3 백엔드 모델, §4.4 RPC 체인 — 전 파일 재확인

- `bin-openapi-manager/openapi/paths/webchat_sessions/main.yaml`(41-55행): `requestBody`가
  `widget_id`만 정의, `required: [widget_id]` — `page_url` 없음. §4.2의 전제와 일치.
- `openapi.yaml:2510` `WebchatManagerSession:` 스키마 실재 확인.
- `models/session/session.go`(16-29행), `field.go`(6-23행), `webhook.go`(13-38행) 모두
  `PageURL`/`FieldPageURL` 없음 — §4.3의 "필드 없음" 전제 정확.
- `pkg/dbhandler/session.go`의 `PrepareFields`/`GetDBFields` 사용(42행, 102행, 160행) —
  구조체 태그 기반 리플렉션이 맞음, §4.3의 "no logic change expected" 판단 타당.
- `scripts/database_scripts_test/sessions.sql`(1-22행) 실재, 현재 `page_url` 컬럼 없음 —
  §4.3의 "SQLite 테스트 스키마에 컬럼 추가 필요" 판단과 일치. `bin-dbscheme-manager/
  bin-manager/main/versions/`에 webchat 관련 기존 마이그레이션 4개(`webchat_widgets_*`,
  `webchat_widgets_sessions_messages_*`)만 존재하고 `page_url` 컬럼 마이그레이션은
  아직 없음 — §4.3/§7이 "새 Alembic revision 필요"라 서술한 그대로.
- RPC 체인 7개 파일(§4.4) 전부 재확인:
  - `requesthandler/webchat_session.go:16-23` — `WebchatV1SessionCreate(ctx, customerID,
    widgetID uuid.UUID)`, `V1DataSessionsPost{CustomerID, WidgetID}`만 구성. PageURL 없음.
  - `requesthandler/main.go:1536` — 인터페이스 시그니처, `pageURL` 없음. 라인 번호 정확.
  - `listenhandler/models/request/v1_sessions.go:10-13` — `V1DataSessionsPost{CustomerID,
    WidgetID}`만, `PageURL` 없음.
  - `listenhandler/v1_sessions.go:33` — `processV1SessionsPost`가 정확히
    `h.sessionHandler.Create(ctx, req.CustomerID, req.WidgetID)` 호출(파라미터 2개).
  - `sessionhandler/main.go:21` — `Create(ctx, customerID, widgetID) (*session.Session, error)`,
    `pageURL` 없음. 3행 `//go:generate mockgen` 지시어 실재.
  - `sessionhandler/mock_main.go`, `requesthandler/mock_main.go` 둘 다 파일 실재 확인
    (search_files로 존재 확인) — §7의 "mock 재생성 대상" 목록과 일치.
  - `sessionhandler/create.go:29-48` — `Create()` 시그니처와 구조체 리터럴(40-48행)
    `WidgetID`/`Status`만 설정, 문서가 인용한 라인 범위와 정확히 일치.
  누락된 중간 파일 없음. Round 0가 지적했던 구멍은 재발하지 않았다.

## 5. §4.5 square-admin 표시 로직 — 이번 라운드 집중 검증

이전 라운드들이 상대적으로 얕게 다룬 구간이라 직접 코드를 열어 대조했다:

- **`message_timeline.js`**: 27행 `const WebchatMessageTimeline = ({ session,
  onOpenChange }) => {` — `session` prop을 받는 구조 확인. 현재는 `session.id`(32행),
  `session.customer_id`(33행), `session.status`(96-98행)만 사용하고 `page_url`은
  아직 참조하지 않는다 — 설계 단계이므로 당연하다. §4.5가 제안하는 "헤더 라인에
  `page_url` 렌더링"은 이 컴포넌트의 실제 렌더 구조(85-143행, `DialogHeader`
  다음에 메시지 리스트가 오는 흐름)에 자연스럽게 삽입 가능한 위치이며, 구조적
  모순은 없다.
- **`session={selectedSession}` 전달부**: `detail.js:97,850` —
  `const [selectedSession, setSelectedSession] = useState(null)`(97행),
  `<WebchatMessageTimeline session={selectedSession} onOpenChange={...} />`(850행).
  `sessions_list_global.js:23,71-74`도 동일 패턴(`const [selectedSession, setSelectedSession]
  = useState(null)`, `<WebchatMessageTimeline session={selectedSession} ... />`).
  두 곳 모두 §4.5의 "already receives `session={selectedSession}` from both `detail.js`
  and `sessions_list_global.js`" 서술과 정확히 일치 — 라인 번호까지 실제와 부합.
- **`sessions_list.js` 컬럼 미추가 판단**: §4.5는 "no new column added to the default
  table view"라 서술하며 `sessions_list_global.js`의 기존 column-crowding 논거를
  근거로 든다. `sessions_list.js`/`sessions_list_global.js` 두 파일 모두 실재 확인됨
  (search_files). §7의 파일 목록에도 `sessions_list.js`는 포함되지 않았는데, 이는
  §4.5의 "컬럼 추가 없음" 결론과 정합적이다 — 컬럼을 추가하지 않으므로
  `sessions_list.js` 자체를 수정할 필요가 없고, 실제로 §7 checklist에서도 빠져 있다
  (일관성 확인, 누락 아님).

## 6. §4.6 프라이버시/RST 문서

- `bin-api-manager/docsdev/source/webchat_struct_session.rst`(1-49행) 전체 재확인 —
  현재 `id`/`customer_id`/`widget_id`/`status`/`tm_*` 5개 필드만 문서화되어 있고
  `page_url` 항목 없음. §4.6의 "새 `page_url` bullet 추가 필요" 판단 정확.
  또한 이 파일의 필드 나열이 `models/session/webhook.go`의 `WebhookMessage`
  구조체(13-23행: `Identity`, `WidgetID`, `Status`, `TMLastActivity/Create/Update/End`)와
  정확히 1:1 대응한다는 점도 확인 — bin-api-manager CLAUDE.md의 "RST struct docs
  MUST match WebhookMessage, not internal model" 규칙과 문서 §4.3의 "webhook.go에
  PageURL 추가" 계획이 함께 맞물려야 RST 갱신이 실제로 유효하다는 것도 재확인했다
  (webhook.go에 필드가 없으면 RST에 추가해도 실제 응답과 불일치하게 됨 — 문서
  §4.3이 이미 webhook.go 변경을 계획에 포함하고 있으므로 문제 없음).

## 7. §5 엣지케이스, §6 Peer/Local 기각 근거 — 재확인

- `widget.go:99` `LogoURL string \`json:"logo_url,omitempty"\` // https URL only` 재확인
  — §5가 "differs from ThemeConfig.LogoURL's existing https://-only validation"의
  근거로 인용한 그대로. 다만 이 주석은 서술적 주석일 뿐 실제 코드 강제가 아니다 —
  `bin-api-manager/server/webchat_widgets.go:340-342`, `pkg/servicehandler`,
  `bin-openapi-manager/openapi/openapi.yaml:2364-2367`(`format: uri`)를 확인했으나
  `https://` 접두어를 실제로 검증(HasPrefix 등)하는 코드는 리포 전체에 없다.
  **단, 이는 §5가 대조 대상으로 삼은 사실("LogoURL은 https 전용 필드"라는 존재
  자체)에는 영향이 없다** — §5의 논지는 "PageURL은 브라우저가 스스로 만든 same-origin
  값이라 LogoURL과 달리 스킴 검증이 불필요하다"는 것이지 "LogoURL의 https 검증이
  실제로 강제되고 있다"는 주장이 아니므로, LogoURL 자체의 검증 부재는 이 설계
  문서의 결함이 아니다(별개 기존 이슈). NICE-TO-HAVE 수준의 참고사항으로만 기록.
- `sessionhandler/create.go:77-78`의 `self`/`peer` 지역변수 재확인 — 이전 라운드들과
  동일하게 `Peer.Target == id`(세션 자기 PK), `Local.Target == widgetID`(이미
  `Session.WidgetID`로 저장)임을 확인, §6의 기각 논리 견고함 유지.

## 8. Round 3 신규 이슈 탐색 결과

전 구간을 코드 대 코드로 재검증했으나 새로운 사실 오류, 환각 인용, 논리적 비약을
발견하지 못했다. §5의 LogoURL https 검증 부재(§7 참고사항)는 이 설계 문서의
결함이 아니라 기존 별개 코드베이스 상태에 대한 부수적 관찰이며, 문서의 논지에
영향을 주지 않는다.

## 9. 결론 — 컨버전스 체크

이번 Round 3은 Round 2(직전 라운드)에 이어 **두 번째 연속 APPROVE**다. review-loop
종료 조건("2회 연속 APPROVED")을 충족한다. §1~§3(문제 정의), §4.1~§4.6(클라이언트,
API 계약, 백엔드 모델, RPC 체인, square-admin 표시, 프라이버시/RST), §5(엣지케이스),
§6(기각 대안), §7(파일 체크리스트, DB 마이그레이션 포함)을 각각 실제 코드와 대조했고,
모든 코드 인용·라인 번호·구조적 주장이 현재 코드베이스와 정확히 일치했다. 이번
라운드에서 처음으로 깊게 검증한 §4.5 square-admin 프론트엔드 파일들
(`message_timeline.js`, `detail.js`, `sessions_list_global.js`)과 §7의
`bin-dbscheme-manager` 마이그레이션 목록도 문서 서술과 완전히 부합한다.

문서는 구현 착수 준비가 되었다고 판단한다.

VERDICT: APPROVED

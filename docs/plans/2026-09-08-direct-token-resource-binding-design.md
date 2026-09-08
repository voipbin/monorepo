# Direct 토큰 리소스 바인딩 설계

- 티켓: VOIP-1501 (REST 소유권 미검증), VOIP-1498 (WebSocket 토픽 와일드카드)
- 관련: VOIP-1502 (reference_type 미검증 + GetByReferenceID 무필터). 본 설계는 direct 경로만 차단하며, 나머지는 그 티켓에서 처리합니다
- 작성일: 2026-09-08
- 개정: rev7 (설계 리뷰 라운드 6 반영. 양측 Approve. 라운드 5·6 연속 승인으로 디자인 리뷰 루프 완료)
- 상태: 디자인 리뷰 완료. 대표님 최종 검토 대기

## 1. 배경

공개 웹페이지에 임베드된 AI 챗봇 위젯과 webchat 위젯은 `direct hash`로 인증합니다. 이 해시는 페이지의 JavaScript 전역(`window._env_.REACT_DIRECT_HASH`)에 노출되어 있고, `POST /auth/boot`은 인증 없이 4시간 JWT를 발급합니다. **해시당·세션당 발급 한도는 없습니다.** (클라이언트 IP당 초당 10회·버스트 20의 리미터는 있습니다. `cmd/api-manager/main.go:291`, `lib/middleware/ratelimit.go:47-80`. **다만 이 버킷은 프로세스 메모리에 있어 replica마다 별개이므로, fleet 전체로는 약 2배입니다.** rev4까지 "횟수 제한이 없습니다"라고 쓴 것은 부정확했습니다.) 즉 **그 페이지를 여는 누구나 토큰을 얻습니다.** 이것은 의도된 설계입니다. 익명 방문자가 챗봇을 쓰려면 필요합니다.

문제는 그 토큰의 권한 범위입니다.

### 1.1 토큰은 스코프를 들고 있으나 아무도 읽지 않습니다

`AuthBoot`(`bin-api-manager/pkg/servicehandler/boot.go:81`)이 만드는 JWT에는 이미 스코프가 들어 있습니다.

```go
scope := &auth.DirectScope{
    CustomerID:           d.CustomerID,
    ResourceType:         d.ResourceType,   // "ai", "ai_team", "webchat_widget" 세 가지
    ResourceID:           d.ResourceID,     // 그 AI 또는 위젯의 id
    AllowedResourceTypes: allowedTypes,     // ["aicall"] 또는 ["webchat_session"]
}
```

`AllowedResourceTypes`는 실제로 검사됩니다. `ResourceID`는 **생성 경로에서만** 검사됩니다.

- `AIcallCreate`(`pkg/servicehandler/aicall.go:70`): `a.DirectScope.ResourceID != assistanceID`면 거부. **올바릅니다.**
- `WebchatSessionCreate`(`pkg/servicehandler/webchat_session.go:197`): 동일. **올바릅니다.**

생성 이후 경로에는 그 검사가 없습니다.

- `AIcallGet`(`aicall.go:206`), `AIcallDelete`(`:231`), `AIcallTerminate`(`:261`): `CustomerID` 일치만 확인
- `AImessageGet`(`aimessage.go:125`): 동일
- `AIcallGetsByCustomerID`(`aicall.go:137`): id 검사가 아예 없음. **테넌트 전체 목록을 반환합니다.**
- `AIcallParticipantGets`(`ai_participants.go:15`): `AIcallGet` 경유. 동일한 결함
- `AImessageDelete`(`aimessage.go:139`): `AImessageGet` 경유. 동일한 결함

결과적으로 공개 링크 보유자가 그 고객사의 **임의 aicall을 열거·조회하고, 삭제하거나 강제 종료**할 수 있습니다. 정보 노출을 넘어 무결성·가용성 문제입니다.

### 1.2 webchat도 같은 결함이 있습니다 (한 단계 약한 형태)

`WebchatMessageList`(`webchat_message.go:104-133`)와 `WebchatSessionEnd`(`webchat_session.go:322`, direct 분기는 `:339`)는 세션을 조회해 `s.WidgetID != a.DirectScope.ResourceID`를 확인합니다. 이것은 **"이 세션이 내 위젯의 것인가"**까지만 봅니다.

같은 위젯 페이지를 연 모든 방문자가 같은 해시를 받으므로 `ResourceID`가 전부 동일합니다. 따라서 **방문자 갑이 방문자 을의 `session_id`를 알면 을의 대화 내용을 전부 읽을 수 있습니다.**

aicall보다 실질 위험이 낮은 이유는 하나뿐입니다. `GET /webchat_sessions`가 direct에게 막혀 있어 세션 id를 얻을 경로가 없습니다. 근본 결함은 같습니다.

### 1.3 WebSocket은 별개 경로로 같은 결함을 가집니다

`validateTopics`(`pkg/websockhandler/etc.go:39-46`)는 4조각 토픽 `customer_id:<cid>:<type>:<id>`에서 `tmps[3]`(리소스 id)을 검증하지 않습니다. 주석이 그 사실을 명시합니다. 그리고 전달 매칭이 `strings.HasPrefix`(`pkg/pubsubhandler/subscriber.go:32`)입니다.

따라서 `customer_id:<cid>:aicall:`처럼 뒤를 비운 토픽이 4조각으로 통과하고, prefix 매칭으로 **그 고객사의 모든 aicall 이벤트**를 받습니다. 이것이 VOIP-1498입니다.

### 1.4 두 티켓은 같은 뿌리입니다

REST든 WebSocket이든 원인이 하나입니다. **토큰이 "무엇에 접근할 수 있는가"를 들고 있는데, 접근 시점에 그것을 대조하지 않습니다.**

## 2. 목표와 비목표

### 목표

1. direct 토큰이 **자기에게 배정된 리소스 하나**에만 접근할 수 있게 한다.
2. 같은 위젯을 연 방문자끼리도 서로 격리한다.
3. 목록 열거를 차단해 다른 방문자의 리소스 id를 알아낼 경로를 없앤다.
4. 새 API를 추가할 때 개발자가 별도 목록에 등록할 필요가 없게 한다.

### 비목표

- direct 토큰 외 신원(상담사, 관리자, accesskey, delegate)의 권한 변경. 이번 범위 밖이며 동작이 바뀌어서도 안 된다.
- 기존 핸들러의 `CustomerID` 검사 제거. 심층 방어로 유지한다.
- 319개 핸들러의 `ErrDirectAccessNotSupported` 거부 가드 정리. 그대로 둔다.

**클라이언트 무변경은 목표가 아닙니다.** rev2는 이것을 목표 5로 두고 "감지 훅이 이미 있으므로 연결만 하면 된다"고 서술했으나, 라운드 2 검증 결과 **다섯 가지 재부팅 계기 중 하나만 훅이 있고 나머지는 클라이언트에 처리 자체가 없습니다**(3.6). 클라이언트 변경은 이 설계의 필수 구성요소이며, 그 범위를 3.6에 명시합니다.

## 3. 설계 결정

2026-09-08 대표 결정 사항입니다.

### 3.1 부팅 시점에 리소스 번호를 미리 배정하고, 소진되면 재부팅한다

`POST /auth/boot`이 토큰을 만들 때, **이 부팅 세션이 사용할 리소스 UUID를 하나 생성**해 JWT와 응답에 함께 넣습니다. 리소스 자체는 만들지 않습니다. 번호만 예약합니다.

부팅만 하고 떠난 방문자는 DB에 아무 흔적을 남기지 않습니다. 리소스를 실제로 만들 때 그 번호가 쓰입니다.

#### rev2의 근거는 사실이 아니었습니다. 결론은 유지하되 근거를 교체합니다

rev2는 "유휴 타임아웃 1800초 × 토큰 14400초이므로 한 토큰 안에서 최대 8개 세션"이라고 서술했습니다. **그 산술이 딛고 선 동작은 구현되어 있지 않습니다.**

- `Widget.SessionIdleTimeout`(`bin-webchat-manager/models/widget/widget.go:51`, 기본 1800)은 CRUD와 웹훅 DTO에만 존재합니다. 이 값을 읽어 세션을 종료시키는 코드가 `bin-webchat-manager` 어디에도 없습니다.
- `TMLastActivity`는 세션 생성 시 한 번 기록되고(`pkg/dbhandler/session.go:36`) 이후 갱신되지 않습니다. 만료 판정에 쓰이지도 않습니다.
- `sessionHandler.End`의 호출자는 **하나뿐**입니다. 명시적 종료 RPC(`pkg/listenhandler/v1_sessions.go:199`)입니다. 유휴 리퍼가 없습니다.
- 자동 재생성 경로도 없습니다. `WebchatMessageCreate`는 `session_id`를 요구하고, 종료된 세션에 대해 두 분기 모두 `ErrNotFound`를 반환합니다.

즉 `models/session/session.go:57-64`의 주석과 `docsdev/source/webchat_overview.rst:67`은 **의도된 동작을 서술한 것이며, 현재 코드는 그렇게 동작하지 않습니다.** rev2가 문서 서술을 실측으로 오인했습니다. rev1이 8.4에서 지적받은 것과 같은 유형의 실수입니다.

**그럼에도 1회용 배정과 재부팅이 옳습니다. 근거는 이것입니다.**

- **명시적 종료는 실재합니다.** `POST /webchat_sessions/{id}/end`, `POST /aicalls/{id}/terminate`, 그리고 서버가 발행하는 `aicall_status_terminated`. 종료 후 같은 토큰으로 새 대화를 시작하려면 새 번호가 필요합니다.
- **유휴 타임아웃이 나중에 구현되면** 토큰 하나에 여러 세션이 실제로 발생합니다. 재부팅 계약은 그때 수정 없이 그대로 동작합니다. 전방 호환입니다.
- **대안은 더 비쌉니다.** 토큰에 방문자 식별자를 넣는 방식은 리소스 테이블 컬럼 추가와 마이그레이션, 그리고 미들웨어의 매 요청 조회를 요구합니다.

**두 번째 생성 시도의 서버 동작.** 배정 번호가 이미 쓰였는데 같은 토큰으로 다시 생성하려 하면 기본키 충돌이 납니다. **명시적 에러(409)**를 반환합니다. 클라이언트는 이를 재부팅 신호로 처리합니다(3.6). 무음 실패나 기존 리소스 반환은 하지 않습니다.

**검토했으나 채택하지 않은 대안**

- *부팅이 리소스까지 생성*: 방문자마다 빈 리소스가 쌓입니다.
- *생성 후 토큰 재발급*: 토큰이 두 번 발급되고 클라이언트가 교체 로직을 가져야 합니다.
- *미들웨어가 매번 리소스를 조회해 소유자 대조*: 같은 위젯의 방문자끼리 구분되지 않습니다. 소유자가 전부 같은 AI이기 때문입니다.
- *토큰에 방문자 식별자를 넣고 리소스에 기록*: 개수 제한이 없어지지만 스키마 변경과 매 요청 조회가 필요합니다.

### 3.2 필드 이름은 `allowed_resource_id`

기존 `allowed_resource_types`(어떤 종류를 다룰 수 있는가)와 짝을 이룹니다. 새 필드는 **그중 어느 인스턴스인가**를 나타냅니다.

`resource_id`(부모 리소스, AI 또는 위젯의 id)는 **의미와 값을 그대로 유지**합니다. 기존 클라이언트가 그 값을 `assistance_id`/`widget_id`로 쓰고 있으므로 건드리면 깨집니다.

### 3.3 URL에 대상이 있는 요청은 미들웨어가 대조한다

`DELETE /aicalls/{id}`처럼 대상이 경로에 있는 요청은, 인증 미들웨어가 경로 파라미터를 토큰의 `allowed_resource_id`와 대조합니다. 불일치면 핸들러에 도달하기 전에 거부합니다.

경로 파라미터는 direct 도달 경로 전부에서 이름이 `:id`이고 위치가 같으므로(4.3), 라우트별 매핑이 필요 없습니다.

### 3.4 URL에 대상이 없는 요청은 서버가 덮어쓴다

`POST /aimessages { aicall_id }`처럼 대상이 본문이나 쿼리에 있는 요청은 **대조하지 않습니다.** 대신 해당 핸들러가 클라이언트가 보낸 값을 무시하고 `allowed_resource_id`를 사용합니다.

**대조하지 않는 이유.** 대조하려면 미들웨어가 "이 API에서는 어느 필드가 대상인가"를 알아야 합니다. 필드 이름이 API마다 다르므로(`aicall_id`, `session_id`) 라우트별 매핑이 필요해지고, 새 API마다 등록해야 합니다. 목표 4에 어긋납니다.

**덮어쓰기가 대조보다 강합니다.** 잘못된 값을 걸러내는 것이 아니라 값을 읽지 않습니다. 검사를 빠뜨릴 여지가 없습니다. 그리고 본문을 읽고 재주입하는 코드가 필요 없어집니다. 이 저장소에는 그 전례가 한 건도 없습니다(`io.NopCloser` 0건).

### 3.5 목록 조회는 direct에서 차단한다

`GET /aicalls`는 대상 id가 없고 덮어쓸 것도 없습니다. direct 신원에서 전면 거부합니다. 위젯이 사용하지 않으며, **다른 방문자의 리소스 id를 알아낼 유일한 경로**이므로 차단 효과가 큽니다.

`GET /webchat_sessions`는 이미 direct를 거부하고 있습니다. aicall 쪽에만 빠져 있습니다.

### 3.6 클라이언트 재부팅 계약 (다섯 가지 계기)

**이 절이 rev3의 핵심 추가이며, rev4에서 다시 손봤습니다.** rev2는 재부팅 계기를 "리소스 종료" 하나로만 서술했고 그것이 이미 훅으로 존재한다고 썼습니다. 라운드 2가 이를 반박했고, 라운드 3이 rev3의 네 계기 표에서 **다섯 번째(403)를 빠뜨렸다**고 지적했습니다.

| # | 계기 | 서버 신호 | 현재 클라이언트 | 필요한 작업 |
|---|---|---|---|---|
| 1 | 리소스 종료 | `webchat_session_ended` / `aicall_status_terminated` WS 이벤트 | webchat: `onSessionEnded` 훅 있고 **프로덕션에서 소비 중**(`square-admin/src/webchat-widget-runtime/widget.js:464-469`, "This session has ended." 표시). AI 위젯: `data.aicall_id` 가드에서 걸러냄(`square-main/.../useChatState.js:39-47`) | 표시 후 재부팅을 연결 |
| 2 | 배정 소진 | `POST /aicalls` 또는 `POST /webchat_sessions` 409 | **없음.** 양쪽 모두 `!res.ok`에서 그냥 throw | 409를 재부팅 신호로 분기 |
| 3 | 토큰 무효 | 401 (배포 시점 구형 토큰, 8.1) | **없음.** `square-main/src/lib/api.js:26`, `square-admin/.../client.js:303` 둘 다 throw만 함 | 401을 재부팅 신호로 분기 |
| 4 | 만료 임박 갱신 | 없음 (클라이언트 타이머) | AI 위젯: `scheduleRefresh`가 만료 5분 전 `POST /auth/boot` 재호출 후 토큰만 교체(`square-main/src/lib/auth.js:45-62`). webchat 위젯: **갱신 로직 자체가 없음** | 재부팅이 아니라 **갱신 전용 경로**. 아래 참조 |
| 5 | 배정 불일치 | 403 (미들웨어) 또는 생성 대상 not-found | **없음** | 403을 재부팅 신호로 분기. **재부팅 루프 방지 필수**. 아래 참조 |

#### 계기 5를 추가하는 이유 (라운드 3 지적)

정상 흐름에서 403은 나오지 않습니다. 그러나 **롤링 배포 중에는 나옵니다.** 매니저가 2 replica이므로, 강제가 켜진 인스턴스와 아직 켜지지 않은 인스턴스가 잠시 공존합니다. 그 창에서 부팅한 방문자는 번호를 들고 있으나 대화는 서버 생성 id로 만들어질 수 있고, 이후 403을 받습니다. 계기 1·2·3 어느 것도 걸리지 않습니다.

#### 재부팅 후 동작과 루프 방지 (rev4의 서술은 구현 불가였습니다)

rev4는 "재부팅 후 재시도했는데 또 403이면 에러 표시"라고 썼습니다. **그 재시도는 반드시 실패합니다.**

**위젯이 실제로 만나는 403은 경로 파라미터 불일치입니다**(4.3 6단계). 본문·쿼리 경로는 대조가 아니라 덮어쓰기이므로(3.4) 거기서는 403이 날 수 없습니다. (다른 403도 존재합니다. 4.3의 3·4·5단계, 4.5의 목록 차단, `AIcallCreate`와 `WebchatSessionCreate`의 스코프 검사, 그리고 `ErrDirectAccessNotSupported` 계열이 그것입니다. 전부 해시에서 결정되는 확정적 거부이므로, 아래 결론은 그 전부에 대해 성립합니다.) 불일치 403은 **낡은 경로 id**를 들고 있고, 재부팅은 그와 절대 일치하지 않는 **새 번호**를 발급합니다. 실패한 요청을 그대로 재생하면 100% 다시 403입니다. 재부팅이 위젯을 완전히 복구한 경우에도 방문자는 에러를 보게 됩니다.

**재부팅 후의 올바른 동작은 실패한 요청의 재생이 아니라 대화 재초기화입니다.** 낡은 번호를 가리키던 상태(대화 id, 구독 토픽, 화면의 메시지 목록)를 버리고 새 번호로 새 대화를 시작합니다.

**루프 방지는 계기 5가 아니라 공용 핸들러에 겁니다.** rev4는 403에만 걸었는데, 정작 대량으로 발생하는 것은 401입니다. 9.2가 강제 시작 시점에 모든 구버전 토큰을 일제히 401시키기 때문입니다.

| 항목 | 규칙 |
|---|---|
| 적용 범위 | 공용 재부팅 핸들러. 계기 1·2·3·5 전부 |
| 상한 | 연속 재부팅 3회 |
| 백오프 | 재부팅 사이에 지수 백오프(0s, 1s, 4s) |
| 리셋 | **재부팅 이후 인증된 요청이 한 번 성공하면 0으로 초기화.** 이 리셋이 없으면 대화를 두 번 종료한 방문자(계기 1)가 영구히 죽습니다 |
| 소진 시 | 방문자에게 에러 표시. 더 이상 부팅하지 않음 |
| 저장 위치 | **페이지 로드 단위 메모리.** `localStorage`에 두지 마십시오. 영속화하면 "에러 표시"가 "캐시를 지울 때까지 영구 고장"이 됩니다 |
| 429 | 부팅 자체가 429를 받을 수 있습니다(IP당 10rps·버스트 20). 백오프 후 재시도하며, 상한을 공유합니다 |

**계기 1에 상한을 걸 때 주의합니다.** 대화 종료는 정상 동작이며 한 방문자가 여러 번 겪을 수 있습니다. 리셋 조건이 있어야 계기 1이 죽지 않습니다.

#### 계기 4는 재부팅으로 처리하면 안 됩니다

`scheduleRefresh`(`square-main/src/lib/auth.js:45-62`)는 만료 5분 전에 `bootAuth(directHash)`를 다시 호출합니다. 이것은 **인증 없는 신규 부팅**이며, 새 토큰을 `setAuthToken`/`wsUpdateToken`으로 갈아끼우기만 하고 진행 중인 대화 번호는 그대로 둡니다.

이 설계를 그대로 적용하면 **3시간 55분째에 진행 중이던 대화가 죽습니다.**

- 새 토큰이 새 `AllowedResourceID`를 들고 옵니다. 그 번호의 리소스는 존재하지 않습니다.
- `POST /aimessages`: 4.4가 `aicall_id`를 그 새 번호로 덮어쓰고, `AIcallGet`이 실패합니다. 이후 모든 전송이 실패합니다.
- 언마운트 시 `DELETE /aicalls/{기존 id}`: 미들웨어가 403.
- WebSocket은 살아 있습니다. 토큰 갱신 시 소켓을 의도적으로 유지하기 때문입니다(`square-main/src/lib/websocket.js:147-163`). 방문자에게는 **연결은 멀쩡한데 메시지만 안 나가는 상태**로 보입니다.

rev2의 §8.2는 "위젯은 새 대화를 시작하게 됩니다"라고 썼습니다. **그렇게 동작하는 코드가 없습니다.**

**해결: 인증된 갱신 엔드포인트를 신설하고, 배정 번호를 그대로 이어받습니다.**

`POST /auth/boot/refresh`를 `authProtected` 그룹(`cmd/api-manager/main.go:306-312`)에 추가합니다. 그 그룹의 체인은 `RateLimit("auth_protected")` + `Authenticate()` + `EnforceAccountStatus()`입니다.

#### 갱신 엔드포인트의 검사 순서 (전부 필수)

라운드 3에서 두 리뷰어가 독립적으로 **갱신이 토큰 무효화를 무력화한다**고 지적했습니다. rev3의 "새 권한이 생기지 않습니다"라는 논증은 **토큰을 얻는 것**에 대해서만 참이고 **토큰을 유지하는 것**에 대해서는 거짓이었습니다. 아래 검사가 그 구멍을 닫습니다.

```
1. !a.IsDirect() 이면 403
      authProtected는 agent/accesskey/delegate도 통과시킨다.
      그들에게 DirectScope는 nil이므로, 이 검사가 없으면 nil 역참조로 500이 난다
2. scope.AllowedResourceID == uuid.Nil 이면 401
      구형 토큰의 Nil을 새 토큰으로 세탁하지 않는다. 8.1과 같은 값 기준 술어
      여기서 새 UUID를 발급하면 방문자의 진행 중 대화가 조용히 고아가 된다. 하지 않는다
   scope.DirectID == uuid.Nil 또는 scope.HashFingerprint == "" 이어도 401
      4·5단계가 우연히 fail-closed 되는 것에 기대지 않는다. 명시적으로 거부한다
3. now > scope.BootExpire 이면 401
      최초 부팅 시각 기준 절대 상한. 갱신으로 밀리지 않는다
4. DirectV1DirectGet(scope.DirectID)가 실패하면 401
      레코드 없음과 RPC 오류를 구분하지 않는다. 둘 다 401 (fail closed)
      direct 레코드 삭제를 즉시 반영한다
5. fingerprint(record.Hash) != scope.HashFingerprint 이면 401
      해시 재발급(무효화)을 즉시 반영한다
6. CustomerV1CustomerGet(scope.CustomerID)가 실패하거나
   Status != StatusActive 이면 401
      AuthBoot(boot.go:96-111)과 동일하게 fail closed.
      EnforceAccountStatus(authenticate.go:249-253)의 fail-open 분기를 재사용하지 말 것.
      같은 패키지 140줄 아래에 정반대 전례가 있으므로 명시한다
7. 통과. DirectScope 전체를 그대로 복사하고, 4단계에서 이미 받아 둔 레코드로
   CustomerID/ResourceType/ResourceID를 덮어쓰며 AllowedResourceTypes를
   directResourceMapping으로 재계산한다. 바깥 expire만 새로 계산한다
      expire = min(now + BootExpiration, scope.BootExpire)
      directResourceMapping에 record.ResourceType이 없으면 401
         AuthBoot(boot.go:113-117)은 이 경우를 에러로 처리한다.
         재계산이 nil을 낳아 "조용히 쓸모없는 토큰"이 되게 두지 않는다
```

**7단계는 "전체 복사"입니다. 필드를 열거하지 마십시오.** rev4는 다섯 개만 나열해 `CustomerID`, `DirectID`, `HashFingerprint`, `ScopeVersion`을 빠뜨렸고, 그대로 구현하면 **첫 갱신에서 토큰이 망가집니다.** `CustomerID`가 Nil이 되면 `validateTopics`의 고객사 대조(`etc.go:43`)가 방문자 자신의 토픽을 거부해 소켓을 끊고, `CustomerRateLimit`의 키(`customer_ratelimit.go:188`)도 무너집니다. `DirectID`/`HashFingerprint`가 빠지면 **두 번째 갱신**이 4·5단계를 못 해 401이 됩니다. 절대 상한이 실질적으로 두 배가 됩니다.

**레코드에서 다시 뽑는 이유.** 클레임을 그대로 믿어도 현재는 차이가 없습니다. direct 레코드의 `customer_id`/`resource_id`를 바꾸는 코드 경로가 없기 때문입니다(`DirectUpdate`는 `FieldHash`로만 호출됩니다, `handler.go:212-214`). 그 불변성에 조용히 의존하지 않기 위해, 이미 손에 든 레코드에서 다시 뽑습니다. 비용이 0입니다.

**절단을 걸 자리가 없다는 점에 주의합니다.** `authJWTGenerateWithExpiration`은 만료를 내부에서 `TimeGetCurTimeAdd(duration)`으로 계산하며, 절대 시각을 받는 인자가 없습니다(`pkg/servicehandler/boot.go:165-183`). 그대로 재사용하면 절단이 조용히 사라집니다. **duration 쪽에서 자르십시오.**

```
duration = min(BootExpiration, parse(scope.BootExpire) - now)
```

**미세한 초과는 남습니다.** `min`은 `now`로 계산하는데 `TimeGetCurTimeAdd`가 `time.Now()`를 다시 읽으므로, 결과가 `BootExpire`를 마이크로초 단위로 넘길 수 있습니다. 실무상 무의미하나, 8.2가 24시간을 정확한 경계로 서술하므로 적어 둡니다.

**`expire`를 상한으로 자릅니다.** `BootExpire`는 **갱신 자격**만 제한하고 토큰 수명은 제한하지 않습니다. 만료 판정은 바깥 `expire` 클레임으로만 이뤄지기 때문입니다(`pkg/servicehandler/auth.go:72`). 상한 1분 전에 갱신하면 거기서 4시간이 더 붙어 실제 최악이 28시간이 됩니다. `min`으로 자르지 않으면 24시간 상한이라는 서술이 거짓입니다.

**두 필드의 형식을 못 박습니다.**

- `BootExpire`: 저장소 관례대로 마이크로초 고정폭 ISO-8601 UTC 문자열(`bin-common-handler/pkg/utilhandler/time.go:10,53-55`). 사전식 비교가 성립하므로 3단계를 문자열 비교로 구현할 수 있습니다.
- `HashFingerprint`: **해시 문자열 전체의 SHA-256 hex, 절단 없음.** 솔트나 시간 의존 요소를 넣으면 5단계가 replica 간에 성립하지 않습니다. 절단하지 않는 이유는 원본 해시의 엔트로피가 48비트뿐이기 때문입니다(`generateHash`가 6바이트, `bin-direct-manager/pkg/directhandler/handler.go:24-30`).
- 지문을 JWT에 넣어도 되는 이유는 **원본이 이미 공개**이기 때문입니다(부팅 가능한 세 리소스 타입의 해시는 페이지 JS에 노출되도록 설계됨). **비밀 자격증명에 이 패턴을 복사하지 마십시오.**

**4·5단계가 없으면 갱신이 무효화를 이깁니다.** 공개 링크가 유출됐을 때의 대응 수단은 해시 재발급인데, 그 동작은 **해시만 바꾸고 고객사 id와 리소스 id는 그대로 둡니다**(`bin-direct-manager/pkg/directhandler/handler.go:196-215`). 따라서 rev3이 §4.1에 쓴 "direct 레코드 재조회"는 두 가지로 틀렸습니다. 토큰에 direct id도 해시도 없어 **조회 자체가 불가능**했고, 설령 고객사·리소스로 조회하더라도 **재발급을 탐지하지 못합니다.**

지금은 유출된 링크의 토큰이 최대 4시간이면 죽습니다. 검사 없는 갱신은 그것을 무한으로 만듭니다. 이는 명백한 후퇴입니다.

**3단계의 절대 상한.** 4·5·6단계가 있으면 무효화는 즉시 반영되므로 상한이 없어도 이론상 안전합니다. 그래도 둡니다. 검사 하나가 나중에 제거되거나 조회가 실패했을 때의 최종 방어선입니다. **`BootExpire`는 최초 부팅 시각 + 24시간이며, 갱신 시 그대로 복사합니다.** 갱신으로 밀리지 않아야 상한의 의미가 있습니다.

#### 토큰에 추가되는 필드

`DirectScope`(`bin-api-manager/models/auth/auth.go:35-40`)에 다섯을 추가합니다.

| 필드 | 용도 |
|---|---|
| `AllowedResourceID` | 배정된 리소스 번호 (3.1) |
| `DirectID` | 갱신 시 direct 레코드 조회 키 (검사 4) |
| `HashFingerprint` | 해시 재발급 탐지 (검사 5). 해시 원문이 아니라 지문을 넣습니다 |
| `BootExpire` | 절대 수명 상한 (검사 3) |
| `ScopeVersion` | 착륙 단계 구분 (9절). 강제 시작 시 이전 단계 토큰을 일괄 무효화합니다 |

**갱신은 `ScopeVersion`을 올리지 않고 그대로 복사합니다.** 올리면 9절의 무효화 장치가 무력해집니다.

`AuthBoot`은 해시로 direct 레코드를 이미 조회하므로(`boot.go:81`) 네 값을 모두 그 자리에서 채울 수 있습니다.

**응답은 `BootResponse`에서 `resource_data`만 뺀 형태입니다.** 다른 필드는 그대로 둡니다. `cachedBoot` 소비자가 `resource_type`/`resource_id`/`customer_id`를 읽기 때문입니다(`square-main/src/components/widget/useChatState.js:130-137`).

`resource_data`를 빼는 이유는 이렇습니다. `resourceDisplayConfigFetchers`가 부팅마다 `WebchatV1WidgetGet` RPC를 한 번 호출하는데(`boot.go:148-160`), 클라이언트가 이미 들고 있는 값을 3시간 55분마다 다시 가져올 이유가 없습니다.

**미들웨어 적용 범위.** 4.3의 `DirectResourceScope()`는 `v1.0` 그룹에만 등록하므로 이 경로에 걸리지 않습니다. 의도한 것입니다. 다만 `authProtected`는 `CustomerRateLimit`이 아니라 `RateLimit("auth_protected")` 버킷을 씁니다. 이것은 **전역이 아니라 클라이언트 IP당** 토큰 버킷입니다(`lib/middleware/ratelimit.go:70-80`). 따라서 상담사 트래픽과의 충돌은 같은 IP를 공유할 때만 일어납니다.

**상한 직전에는 갱신이 몰립니다.** `scheduleRefresh`의 지연 하한이 60초이므로(`square-main/src/lib/auth.js:53`), `BootExpire` 직전 약 5분 동안은 절단된 `expire`가 5분 미만의 여유만 남겨 타이머가 60초마다 발화합니다. 즉 방문자당 마지막 꼬리에서 갱신이 1회가 아니라 약 5회입니다. 유계이고 fail-closed(결국 3단계에서 401 후 재부팅)이지만, 아래 "3시간 55분에 한 번"은 그 꼬리를 제외한 값입니다.

**수용 근거를 정직하게 씁니다.** "방문자당 3시간 55분에 한 번"은 정상 클라이언트 모델이지 공격자 모델이 아닙니다. 무료로 얻은 토큰 하나로 IP 한도까지 갱신을 두드릴 수 있고, 각 호출이 내부 RPC 2회를 씁니다. 그럼에도 수용하는 이유는 **`/auth/boot` 자체가 이미 무인증·무료이고 RPC 2~3회를 쓰기 때문**입니다. 새로운 증폭 계층이 아닙니다.

**클라이언트 변경.** `scheduleRefresh`의 `bootAuth(directHash)` 호출을 `refreshAuth()`로 교체합니다. **`refreshAuth()`는 토큰만 바꾸는 것으로 부족합니다.** `bootAuth`는 `cachedBoot`를 갱신하고 `scheduleRefresh`를 다시 걸어 주는데(`auth.js:25-28`), 이를 빠뜨리면 타이머가 **한 번만 동작하고 다시 걸리지 않아** 4시간 절벽이 그대로 돌아옵니다. 갱신 실패 시에는 계기 3(401)으로 떨어져 재부팅합니다.

**AI 위젯의 `aicall_id` 가드는 "완화"하지 않습니다.** 그 가드는 상태 객체가 가짜 말풍선으로 렌더링되는 것을 막는 목적이 문서화되어 있습니다(`useChatState.js:39-43`). 가드를 느슨하게 하는 대신 **가드보다 앞에 `aicall_status_terminated` 분기를 명시적으로 추가**합니다.

**이것은 rev2에 없던 범위 추가이며, 계기 4를 안전하게 처리할 다른 방법이 없어 불가피합니다.** 대안은 갱신 시점에 대화를 강제 종료하고 새로 시작하는 것인데, 아무 이유 없이 3시간 55분째 대화를 끊는 것이므로 현재보다 나쁩니다.

#### 계기별 클라이언트 구현은 한 곳에 모읍니다

다섯 계기를 각각 다른 위치에서 처리하면 하나를 빠뜨렸을 때 조용히 깨집니다. 각 앱에 **재부팅 핸들러 하나**를 두고 계기 1·2·3·5가 모두 그것을 호출하게 합니다. 계기 4만 갱신 경로입니다.

**webchat 위젯은 갱신 로직이 아예 없습니다.** `square-admin/src/webchat-widget-runtime/client.js`에 만료 타이머가 없어, 오늘도 4시간이 지나면 토큰이 죽고 복구 경로가 없습니다. 이것은 이 설계가 만든 문제가 아니라 기존 결함이며, 계기 3·4 구현으로 함께 해소됩니다.

## 4. 아키텍처

### 4.1 부팅에서 번호 배정

`AuthBoot`(`boot.go:81`)에서 `allowedTypes`를 조회한 직후 UUID를 생성해 스코프에 넣습니다.

```go
scope := &auth.DirectScope{
    CustomerID:           d.CustomerID,
    ResourceType:         d.ResourceType,
    ResourceID:           d.ResourceID,
    AllowedResourceTypes: allowedTypes,
    AllowedResourceID:    h.utilHandler.UUIDCreate(),   // 추가
}
```

`BootResponse`(`boot.go:27-46`)에도 같은 값을 `allowed_resource_id`로 반환합니다. 클라이언트는 이 값을 쓸 필요가 없지만(4.4), 디버깅과 향후 확장을 위해 노출합니다.

`DirectScope`(`bin-api-manager/models/auth/auth.go:35-40`)에 필드를 추가합니다. JWT 클레임은 이 구조체를 그대로 직렬화하므로 별도 작업이 없습니다.

`DirectID`, `HashFingerprint`, `BootExpire`도 같은 자리에서 채웁니다(3.6). 넷 다 `AuthBoot`이 이미 조회한 direct 레코드와 현재 시각으로 계산할 수 있어 추가 조회가 없습니다.

**갱신 엔드포인트**(3.6)는 새 UUID를 만들지 않고 토큰의 값을 복사하되, **복사 전에 3.6의 7단계 검사를 전부 수행합니다.** rev3은 여기에 "direct 레코드를 재조회해 확인"이라고만 썼는데, 토큰에 조회 키가 없어 실행 불가능했고 해시 재발급도 탐지하지 못했습니다. 그 문장을 3.6의 검사 목록으로 대체합니다.

### 4.2 배정된 번호를 리소스 생성까지 전달

`bin-ai-manager`의 aicall id는 `pkg/aicallhandler/db.go:41`(`Create`)과 `:115`(`CreateByMessaging`)에서 `h.utilHandler.UUIDCreate()`로 생성됩니다. 호출자가 지정할 수 없습니다.

**저장소에 동일 패턴의 전례가 있습니다.** `bin-transcribe-manager`가 이미 이 구조를 씁니다.

```go
// pkg/listenhandler/models/request/v1_transcribes.go:13
ID uuid.UUID `json:"id,omitempty"`  // optional caller-specified id; zero => server generates

// pkg/transcribehandler/start.go:80-91
callerSpecifiedID := id != uuid.Nil
if !callerSpecifiedID {
    id = h.utilHandler.UUIDCreate()
} else if _, err := h.Get(ctx, id); err == nil {
    return nil, errTranscribeIDAlreadyExists()
} else if !stderrors.Is(err, dbhandler.ErrNotFound) {
    return nil, err
}
```

같은 패턴을 aicall과 webchat session에 적용합니다. **DB 스키마는 변경하지 않습니다.** `PrepareFields`가 구조체의 `ID`를 그대로 insert하므로, 값만 채워 넣으면 됩니다.

**OpenAPI에는 `id`를 노출하지 않습니다.** 외부 호출자가 리소스 id를 지정할 이유가 없고, 노출하면 새 공개 표면이 생깁니다. `bin-transcribe-manager` 선례는 공개 스펙에도 노출했으나, 그 구현의 중복 확인(`h.Get(ctx, id)`)이 고객사 필터 없이 동작해 **교차 테넌트 존재 여부 오라클**이 됩니다. 같은 실수를 반복하지 않습니다.

#### 4.2.1 변경 대상 (rev2의 목록은 불완전하고 자기모순이었습니다)

rev2의 표는 6단계로 줄었다고 썼으나, `Start`의 호출자를 세 곳 빠뜨렸고 "서버 핸들러는 손대지 않는다"는 문장이 1단계와 모순됐습니다. 용어를 먼저 구분합니다.

- **OpenAPI 서버 핸들러** = `bin-api-manager/server/aicalls.go`. 요청 바인딩 계층. **손대지 않습니다.** `id`를 스펙에 노출하지 않으므로 바인딩할 것이 없습니다.
- **서비스 핸들러** = `bin-api-manager/pkg/servicehandler/aicall.go`. 인가와 오케스트레이션. **여기서 토큰의 `AllowedResourceID`를 채워 넣습니다.**

aicall 기준 전체 변경 대상입니다.

| # | 대상 | 내용 |
|---|---|---|
| 1 | `pkg/servicehandler/aicall.go:22` `AIcallCreate` | direct 분기에서 `AllowedResourceID`를 전달 |
| 2 | `bin-common-handler/pkg/requesthandler/ai_aicalls.go:18` `AIV1AIcallStart` + 인터페이스(`main.go:283`) + mock | `id` 파라미터 추가 |
| 3 | `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go:284` | **rev2 누락.** 같은 `AIV1AIcallStart`를 호출하므로 `uuid.Nil`을 넘기도록 수정 |
| 4 | `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go` | `ID` 필드 추가 |
| 5 | `bin-ai-manager/pkg/listenhandler/v1_aicalls.go:91` | `req.ID`를 `Start`에 전달 |
| 6 | `pkg/aicallhandler/start.go` `Start`(:170) + 인터페이스 + mock | `id` 파라미터 추가 |
| 7 | `pkg/aicallhandler/service.go:54`, `:96` | **rev2 누락.** `Start`의 내부 호출자 2곳. `uuid.Nil` 전달 |
| 8 | `pkg/aicallhandler/start.go` `startReferenceType*` 4개(:204/:245/:405/:606), `startAIcallByRealtime`(:993), `startAIcallByMessaging`(:1050) | `id` 관통 |
| 9 | `pkg/aicallhandler/start.go:1106` `StartTask`(내부에서 `:1121`이 호출) | **rev2 누락.** `startAIcallByMessaging`의 또 다른 호출자. `uuid.Nil` 전달 |
| 10 | `pkg/aicallhandler/db.go:41`, `:115` | `uuid.Nil`이면 생성, 아니면 그대로 사용 |

webchat session도 같은 형태를 거칩니다(`bin-webchat-manager/pkg/sessionhandler/create.go:37`이 생성 지점). **구현 전에 `Create`의 호출자를 같은 방식으로 전수 조사합니다.** aicall에서 세 곳을 놓친 것이 이 조사를 생략했기 때문입니다.

#### 4.2.2 중복 판정은 기본키 제약이 권위입니다

rev2는 "중복 확인을 고객사 범위로 한정한다"고만 썼습니다. 그것만으로는 부족합니다.

**고객사 범위 사전 확인은 교차 테넌트 오라클을 없애지만, 기본키에 대해서는 불완전합니다.** 다른 테넌트가 쥔 id에 대해 "없음"을 반환하고, 그 뒤 INSERT가 드라이버 원시 에러로 실패합니다. 의도한 409가 나오지 않습니다.

따라서 이렇게 정의합니다.

- **기본키 제약이 정답입니다.** 중복 여부의 최종 판정은 INSERT 실패입니다.
- **사전 확인은 깨끗한 409를 렌더링하기 위한 것뿐입니다.** 보안 경계가 아닙니다.
- INSERT가 기본키로 실패하면 사전 확인 결과와 무관하게 409로 매핑합니다.

**409는 서비스 경계를 넘어와야 합니다.** 기본키 실패는 `bin-ai-manager`에서 일어나므로, RabbitMQ RPC를 건너 `bin-api-manager`까지 전달돼야 합니다. 기존 경로를 씁니다. `StatusAlreadyExists`(`bin-common-handler/models/errors/status.go:32`)로 반환하면 `rpc.go:66-67`이 409로 매핑합니다. 새로 만들지 마십시오.

**PK 충돌과 유니크 인덱스 충돌을 구분하는 방법.** 8.4가 지적하듯 `IsErrDuplicate`의 문자열 매칭으로는 둘을 구분할 수 없습니다. 구현 가능한 판별식은 이것입니다.

> **호출자가 id를 지정한 요청에서 발생한 중복 에러**는 PK 충돌이다.

성립하는 이유는 그 경로가 direct 전용이고, 4.4.1이 direct의 `reference_id`를 `uuid.Nil`로 강제하므로 **생성 컬럼이 NULL이 되어 유니크 인덱스 충돌 자체가 불가능**하기 때문입니다. 즉 남는 가능성이 PK 하나뿐입니다.

`ai_aicalls`의 유니크 제약은 기본키와 `uq_aicall_active_reference_key` 둘뿐이고(`bin-ai-manager/scripts/database_scripts_test/table_ai_aicalls.sql:57,60`), **`webchat_sessions`에는 기본키 외 유니크 인덱스가 아예 없습니다**(`bin-webchat-manager/scripts/database_scripts_test/sessions.sql:20`, 프로덕션 마이그레이션 `c9602a744cb3:82-107`도 동일하게 유니크 인덱스 없음). 따라서 webchat 쪽은 판별식이 더 단순하게 성립합니다.

이 판별식은 4.4.1에 의존하므로, 4.4.1을 되돌리면 함께 무너집니다. **무너지는 방향은 어느 쪽을 되돌리느냐에 따라 다릅니다.**

- **`reference_id` 강제만 되돌린 경우**: 안전합니다. `reference_type`이 여전히 `none`이므로 `startReferenceTypeNone`(`start.go:606`)을 타고, 그 함수는 `GetByReferenceID`를 호출하지 않습니다. 유니크 충돌이 409로 오분류되고, 클라이언트가 재부팅해 같은 충돌을 만나고, 상한 3에서 멈춰 에러를 표시합니다. UX 저하입니다.
- **`reference_type` 강제까지 되돌린 경우**: **안전하지 않습니다.** direct가 `contact_case`를 보낼 수 있게 되고, 유니크 위반이 `Start`의 분류에 닿기 전에 재사용 경로에 가로채입니다(`start.go:443-455`가 `IsErrDuplicate` 확인 후 곧바로 `GetByReferenceID` 호출). 4.4.1이 서술한 그대로 **타인의 aicall이 반환됩니다.**

즉 4.4.1의 두 강제는 분리 불가이며, 그중 `reference_type` 쪽이 보안 경계입니다. rev6은 이 구분 없이 "UX 저하이지 보안 구멍이 아니다"라고 써서 4.4.1 본문과 모순됐습니다.

#### 분류는 db 계층이 아니라 핸들러 계층에서 합니다

**`bin-transcribe-manager` 선례의 db 계층 형태를 복사하면 기존 동작이 깨집니다.**

```go
// bin-transcribe-manager/pkg/dbhandler/transcribe.go:67-70
if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
    if isDuplicateKeyErr(err) {
        return ErrDuplicateID          // 원본 문자열을 버린다
    }
```

`bin-ai-manager`의 `IsErrDuplicate`는 `"Duplicate entry"` **문자열 매칭**이고(`pkg/dbhandler/main.go:114`), 살아 있는 소비자가 둘 있습니다.

- `pkg/aicallhandler/start.go:444` (contact_case 재사용·경합 처리)
- `pkg/aihandler/db.go:229`

현재 `AIcallCreate`는 `fmt.Errorf("...: %v", err)`로 감싸 문자열을 보존합니다(`pkg/dbhandler/aicall.go:36-39`). **여기에 bare 센티널을 넣으면 두 소비자가 조용히 동작을 멈춥니다.** direct와 무관한 경로가 깨집니다.

따라서 이렇게 합니다.

- `dbhandler.AIcallCreate`의 반환은 **바꾸지 않습니다.**
- 분류는 `aicallhandler.Start`에서 `callerSpecifiedID`가 참일 때만 수행합니다. **`Start`(`start.go:170-201`)는 분기만 하고 자식의 에러를 그대로 반환하는 디스패처이므로, 분류는 개별 분기 안이 아니라 switch 이후 반환값에 겁니다.** transcribe도 db 계층이 아니라 핸들러 계층에서, 같은 형태로 게이트합니다(`pkg/transcribehandler/start.go:112-125`).
- 센티널은 드라이버 에러를 `%w`/`%v`로 감싸지 않은 **새 `cerrors.AlreadyExists`**여야 합니다(`errTranscribeIDAlreadyExists`와 같은 형태, `start.go:135-141`). 감싸면 `IsErrDuplicate`에 도로 걸려 재시도 루프에 삼켜지고, 위 "삼켜지지 않는다"는 서술이 뒤집힙니다.

**`IsErrDuplicate`와의 결합을 명시합니다.** 전용 센티널(`errAIcallIDAlreadyExists`)은 `bin-ai-manager/pkg/dbhandler/main.go:114`의 문자열 매칭(`"Duplicate entry"`)에 걸리지 않습니다. 따라서 `startReferenceTypeContactCase`의 재시도 루프가 이를 삼키지 않고 그대로 올려보냅니다. **이것이 의도한 동작입니다.** 다만 그것이 성립하는 이유는 4.4.1이 direct를 그 경로에서 배제하기 때문이며, 우연이 아니라 명시된 결합입니다.

### 4.3 미들웨어: 경로 대상 대조

`bin-api-manager/lib/middleware/`에 `DirectResourceScope()`를 신설하고, `cmd/api-manager/main.go`의 `v1` 그룹에 등록합니다.

```go
v1.Use(middleware.Authenticate())            // :333
v1.Use(middleware.CustomerRateLimit(...))    // :334
v1.Use(middleware.EnforceAccountStatus())    // :335
v1.Use(middleware.DirectResourceScope())     // 추가
openapi_server.RegisterHandlersWithOptions(v1, ...)  // :336
```

`RegisterHandlersWithOptions` **이전**에 등록해야 합니다. gin의 `RouterGroup.Use`는 라우트 등록 시점에 핸들러 슬라이스를 스냅샷하므로, 이후에 추가하면 적용되지 않습니다.

동작:

```
0. auth_identity가 없거나 타입이 다르면 401 (abort)
      EnforceAccountStatus(authenticate.go:87-99)와 같은 처리. 통과시키지 않는다
1. auth_identity가 direct가 아니면 즉시 통과
2. c.Params가 비어 있으면 통과 (생성·전송 계열. 4.4가 처리)
3. c.Param("id")가 비어 있으면 403        ← params는 있는데 id가 아닌 경우
4. c.Param("id")를 uuid.FromString으로 파싱. 실패하면 403
5. a.DirectScope == nil 또는 a.DirectScope.AllowedResourceID == uuid.Nil 이면 403
      nil 역참조는 4.6과 같은 종류다. gin.Recovery가 잡아 500을 내는 것보다 403이 낫다
      Nil 검사는 미들웨어를 자기완결적으로 만든다. 이것이 없으면
      경로 id가 00000000-...-000000000000인 요청이 6단계에서 "일치"로 통과한다.
      8.1이 그런 토큰을 원천 차단하므로 도달 불가이나,
      4.4가 Nil 가드를 남겨 두는 것과 같은 이유로 여기도 남긴다
6. 파싱한 UUID != AllowedResourceID 이면 403
7. 통과
```

**3단계가 fail-closed의 핵심입니다.** 단순히 "id가 없으면 통과"로 두면, 나중에 누가 `GET /aicalls/:aicall_id/foo` 같은 라우트를 추가하면서 direct를 허용하면 미들웨어가 조용히 통과시킵니다. 파라미터가 있는데 그중 `id`가 없다는 것은 이 미들웨어가 모르는 형태라는 뜻이므로 거부합니다. 목표 4가 이 방식으로 유지됩니다.

**4단계를 명시합니다(rev2 누락).** rev2는 "없음"과 "불일치"만 다루고 **"있는데 파싱 불가"**를 정의하지 않았습니다. `uuid.FromStringOrNil`을 쓰면 파싱 실패가 `uuid.Nil`이 되고, 그것이 `AllowedResourceID`와 우연히 같아지는 상황(즉 `AllowedResourceID`가 Nil인 토큰)에서만 통과합니다. 8.1이 그런 토큰을 원천 차단하므로 결과적으로는 안전하지만, **두 절에 걸친 암묵적 결합에 안전성을 의존시키지 않습니다.** 파싱 실패는 그 자리에서 403이고, `AllowedResourceID`가 Nil이면 어떤 값과도 비교하지 않습니다.

**6단계는 문자열이 아니라 파싱한 UUID를 비교합니다.** 핸들러가 `uuid.FromStringOrNil`로 파싱하므로(`server/aicalls.go:112`) 대소문자나 중괄호 형식을 받아들입니다. 문자열 비교를 하면 미들웨어와 핸들러가 "같은 id"에 대해 다른 판정을 내립니다.

**경로 파라미터 실태.** rev3의 "전부 `:id`"는 부정확했습니다. 라운드 3·4에서 리뷰어들이 서로 다른 수를 제시했는데, **등록 수(메서드+경로)와 고유 경로 수를 각각 센 결과**였습니다. 혼선을 막기 위해 기준을 고정합니다. 아래는 `gens/openapi_server/gen.go`의 **고유 경로** 기준입니다.

| 구분 | 수 |
|---|---|
| 고유 라우트 경로 | 264 |
| `:id`를 포함 | 159 |
| `:id` + 두 번째 파라미터 | 10 (`/ais/:id/prompt_histories/:history_id`, `/contacts/:id/addresses/:address_id`, `/contacts/:id/tags/:tag_id`, `/contact_cases/:id/notes/:note_id`, `/outdials/:id/targets/:target_id`, `/rags/:id/sources/:source_id`, `/service_agents/...` 4개) |
| 파라미터는 있으나 `:id` 없음 | 4 (`/billings/:billing-id`, `/timelines/:resource_type/:resource_id/events`, `/timelines/calls/:call_id/pcap`, `/timelines/calls/:call_id/sip-analysis`) |

결과는 그대로 fail-closed입니다. `:id` 없는 라우트는 3단계에서 403이고, 두 번째 파라미터를 가진 라우트는 `:id`가 부모 리소스 id이므로 6단계에서 403입니다.

**다만 한 가지 형태가 미래의 구멍입니다.** 누군가 `/aicalls/:id/<child>/:child_id` 같은 라우트를 추가하면 `:id`는 일치하고 **`:child_id`는 아무도 검사하지 않습니다.** 현재 그런 라우트는 없습니다. 이 사실을 미들웨어 주석에 남기고, 8.8의 후속 과제로 올립니다.

**자동으로 얻어지는 효과.** `GET /aimessages/{message_id}`는 메시지 id가 오는데, 그 값은 대화 번호와 절대 같을 수 없으므로 6단계에서 거부됩니다. 별도 예외 목록이 필요 없습니다. 위젯이 사용하지 않는 경로이므로 영향도 없습니다.

**적용 범위 한계.** 이 미들웨어는 `v1.0` 그룹만 덮습니다. `authProtected` 그룹(`main.go:306-312`)의 `/auth/unregister`, `/auth/delegate`는 별개이나 셋 다 direct를 downstream에서 이미 거부합니다. 3.6이 추가하는 `/auth/boot/refresh`는 **의도적으로 이 미들웨어 밖**입니다. 경로 파라미터가 없어 2단계에서 통과하겠지만, 그룹이 다르다는 사실을 명시해 둡니다.

### 4.4 핸들러: 본문·쿼리 값 덮어쓰기

대상이 본문이나 쿼리에 있는 경로는 해당 핸들러의 **기존 direct 분기 안에서** 값을 덮어씁니다. 새 분기를 만들지 않습니다.

| 핸들러 | direct 분기 | 처리 |
|---|---|---|
| `AIcallCreate`(`aicall.go:22`) | 있음 | 생성할 aicall의 id를 `AllowedResourceID`로 지정. 기존 `ResourceID != assistanceID` 검사 유지. **추가로 `referenceType`을 `none`, `referenceID`를 `Nil`로 강제**(4.4.1) |
| `AImessageCreate`(`aimessage.go:26`) | **없음. 신설** | `AIcallGet` 위임 이전에 `aicallID`를 `AllowedResourceID`로 치환 |
| `AImessageGetsByAIcallID`(`aimessage.go:50`) | **없음. 신설** | 동일 |
| `WebchatSessionCreate`(`webchat_session.go:140`) | 있음 | 생성할 session의 id를 `AllowedResourceID`로 지정 |
| `WebchatMessageCreate`(`webchat_message.go:183`) | 있음 | 본문의 `session_id`를 무시하고 `AllowedResourceID` 사용 |
| `WebchatMessageList`(`webchat_message.go:88`) | 있음 | 쿼리의 `session_id`를 무시하고 `AllowedResourceID` 사용. 기존 `sessionID == uuid.Nil` 거부 가드(`:108-113`)는 **제거하지 않고 유지** |

6곳입니다. `aimessage.go` 전체에 `IsDirect()`는 `AImessageGet`(`:125`) 한 곳뿐이므로, 위 두 항목은 분기 신설입니다.

**여섯 곳 모두 덮어쓰기 직전에 `AllowedResourceID == uuid.Nil`이면 거부합니다.** rev4는 `WebchatMessageList`의 기존 Nil 가드를 "죽은 코드가 되므로 제거"한다고 썼습니다. 제거하지 않습니다. 8.1이 Nil 토큰을 원천 차단하므로 실행되지 않는 것이 맞지만, **8.1을 제거하거나 우회하는 변경이 들어왔을 때 마지막으로 붙잡는 자리**가 바로 이곳입니다. 비용이 조건문 하나입니다. 라운드 4 보안 리뷰가 같은 취지를 지적했습니다.

**`WebchatSessionEnd`(`webchat_session.go:322`)는 이 표에 없습니다.** 경로 파라미터 `:id`를 가지므로 4.3의 미들웨어가 덮습니다. 8.6에 함께 적습니다.

#### 4.4.1 direct 토큰의 `reference_type`과 `reference_id`를 **둘 다** 강제한다

`AIcallCreate`의 direct 분기는 현재 `referenceType`과 `referenceID`를 **전혀 검사하지 않습니다**(`aicall.go:66-72`). 값은 요청 본문에서 문자열 캐스트로 그대로 전달되며(`server/aicalls.go:41`), 생성된 `Valid()` 헬퍼는 호출되지 않습니다.

공개 링크 보유자가 `reference_type: "contact_case"`와 임의의 `reference_id`를 보내면 `startReferenceTypeContactCase`(`start.go:405`)로 분기하고, `uq_aicall_active_reference_key` 위반 시 재사용 경로가 `AIcallGetByReferenceID`를 호출합니다. **그 함수에는 고객사 필터가 없어 타인의 aicall이 반환되고 세션 상태가 변경됩니다.** VOIP-1502로 등록했습니다.

**미들웨어는 이 경로를 막지 못합니다.** `POST /aicalls`에 경로 파라미터가 없어 통과시키기 때문입니다.

따라서 direct 분기에서 `referenceType = ReferenceTypeNone`, `referenceID = uuid.Nil`을 강제합니다. 두 위젯 모두 실제로 그 값만 사용하므로 동작 변경이 없습니다.

**두 강제는 서로 다른 것을 막으며, 어느 하나도 뺄 수 없습니다(rev2가 뭉뚱그렸던 부분).**

- `referenceType = none`: 분기를 `startReferenceTypeNone`(`start.go:606`)으로 고정합니다. 이 함수는 `AIcallGetByReferenceID`를 호출하지 않습니다. `GetByReferenceID`에 닿는 start 경로는 `startReferenceTypeConversation`(`:274`)과 `startReferenceTypeContactCase`(`:451`) 둘뿐이므로, 이것으로 무필터 조회가 차단됩니다.
- `referenceID = uuid.Nil`: 유니크 인덱스 표면을 없앱니다. 생성 컬럼의 정의가 **타입이 아니라 `reference_id`를 조건으로 삼기 때문**입니다(마이그레이션 `a5a40c93d3e6:143`).

```sql
active_reference_key BINARY(32) GENERATED ALWAYS AS (
    IF(status NOT IN ('terminated','terminating')
       AND reference_id != UNHEX('00000000000000000000000000000000'),
       UNHEX(SHA2(CONCAT_WS('|', customer_id, reference_type, reference_id), 256)),
       NULL)
) STORED
```

즉 `reference_type`만 `none`으로 바꾸고 공격자의 `reference_id`를 통과시키면 `active_reference_key`가 여전히 계산되어 충돌 표면이 남습니다. **`reference_id`를 Nil로 만드는 쪽이 키를 NULL로 만드는 실제 조치입니다.**

이 조치는 direct 경로만 닫습니다. `reference_type` 열거 검증, `reference_id` 소유권 확인, `AIcallGetByReferenceID`의 고객사 필터는 direct 외 신원에도 해당하므로 VOIP-1502에서 처리합니다.

**기존 검사는 제거하지 않습니다.** `HasAllowedResourceType`, `CustomerID` 일치, `WidgetID` 일치, 위젯 soft-delete 확인은 전부 유지합니다. 심층 방어입니다.

### 4.5 목록 조회 차단

`AIcallGetsByCustomerID`(`aicall.go:137`)의 direct 분기를 전면 거부로 바꿉니다.

```go
case a.IsDirect():
    return nil, serviceerrors.ErrPermissionDenied
```

`GET /webchat_sessions`는 이미 direct를 거부하고 있어 변경이 없습니다. 이 비대칭이 aicall 쪽 결함의 원인이었습니다.

### 4.6 WebSocket 토픽 검증 (VOIP-1498)

WebSocket 연결 요청(`GET /v1.0/ws`, `GET /v1.0/service_agents/ws`)에는 대상 id가 없습니다. 토픽은 연결 후 프레임으로 도착하므로 미들웨어가 처리할 수 없습니다. `validateTopics`에서 별도로 다룹니다.

`etc.go`의 4조각 분기, `a.IsDirect()` 안에 두 가지를 추가합니다.

```go
if a.IsDirect() {
    // DirectScope nil 가드. HasAllowedResourceType은 이미 방어하고 있으나
    // 아래 필드 접근은 그렇지 않다. 8.5 참조
    if a.DirectScope == nil {
        return false
    }
    if !a.HasAllowedResourceType(tmps[2]) {
        return false
    }
    // 추가 1: 리소스 id가 정규 UUID여야 함 (와일드카드 차단)
    parsed, err := uuid.FromString(tmps[3])
    if err != nil || parsed.String() != tmps[3] {
        return false
    }
    // 추가 2: 배정된 리소스여야 함
    if parsed != a.DirectScope.AllowedResourceID {
        return false
    }
}
```

**두 검사의 역할이 다릅니다.** 첫째는 `customer_id:<cid>:aicall:`(빈 문자열)이 prefix 와일드카드로 동작하는 것을 막습니다. 둘째는 다른 방문자의 토픽 구독을 막습니다.

**`etc.go:39-42`의 기존 주석을 반드시 고쳐 씁니다.** 그 주석은 "리소스 id를 검증하지 않는다"고 명시하고 있어, 이 변경 이후 사실과 반대가 됩니다. 잘못된 주석은 다음 사람이 검사를 지우는 근거가 됩니다.

**정규화 판정이 미들웨어보다 엄격한 것은 의도적입니다.** 4.3은 `uuid.FromString` 결과만 비교하므로 중괄호·대문자 형식을 허용하지만, 여기서는 `parsed.String() != tmps[3]`으로 정규 표기만 통과시킵니다. 토픽은 발행 측 문자열과 바이트 단위로 일치해야 하기 때문입니다. 둘 다 안전하며, 나중에 "일관성"을 이유로 WS 쪽을 느슨하게 맞추면 안 됩니다.

**비-direct 분기는 건드리지 않습니다.** 관리자·매니저는 2·3조각 토픽으로 이미 합법적으로 고객사 전체를 구독할 수 있으므로, 4조각 trailing colon이 새 권한을 주지 않습니다. 그리고 그 형태는 `docsdev/source/transcribe_tutorial.rst:273-280`에서 **지원되는 prefix 구독 문법으로 문서화**되어 있습니다. 무조건 거부하면 문서화된 기능이 깨지고, 토픽 거부가 소켓 전체를 끊으므로(`subscription.go:181-184` → `:158-161` → `:140`) 피해가 큽니다.

**`validateTopic`(단수, `etc.go:87-150`)은 `validateTopics`의 거의 동일한 복사본이며 `etc_test.go:277`에서만 호출됩니다.** 인가 술어의 사본이 갈라지면 같은 결함이 조용히 재도입되므로, 이번에 제거하고 그 11개 테스트 케이스를 `Test_validateTopics`로 이관합니다. 현재 `Test_validateTopics`는 4개뿐이라 삭제만 하면 인가 커버리지 대부분을 잃습니다.

## 5. 데이터 흐름

방문자가 AI 챗 위젯이 있는 페이지를 엽니다.

1. `POST /auth/boot { direct_hash }`: 서버가 direct 레코드를 조회해 `ResourceID`(AI id)를 얻고, 별도로 `AllowedResourceID`를 새로 생성합니다. 둘 다 JWT에 넣고 응답에도 반환합니다.
2. `POST /v1.0/aicalls`: 미들웨어는 경로 파라미터가 없으므로 통과시킵니다. 핸들러의 direct 분기가 기존 검사(`ResourceID != assistanceID`)를 수행하고, `referenceType`/`referenceID`를 강제한 뒤, 생성할 aicall의 id를 `AllowedResourceID`로 지정합니다.
3. `POST /v1.0/aimessages`: 미들웨어 통과. 핸들러가 본문의 `aicall_id`를 무시하고 `AllowedResourceID`를 씁니다.
4. WebSocket 구독 `customer_id:<cid>:aicall:<AllowedResourceID>`: `validateTopics`가 UUID 형태와 배정 일치를 확인합니다.
5. (3시간 55분 경과) `POST /auth/boot/refresh`: 현재 토큰을 Bearer로 제시합니다. 서버가 3.6의 7단계 검사(direct 신원, Nil 아님, 절대 상한 이내, direct 레코드 생존, 해시 지문 일치, 고객사 활성, 리소스 타입 매핑 존재)를 수행한 뒤 **같은 `AllowedResourceID`**로 새 토큰을 발급합니다. 대화가 끊기지 않습니다.
6. `DELETE /v1.0/aicalls/{AllowedResourceID}`: 미들웨어가 경로 파라미터를 대조해 통과시킵니다.
7. (대화 종료 후 재시작) `aicall_status_terminated` 수신 → `POST /auth/boot` 재호출 → 새 번호 배정.

공격자가 같은 페이지에서 토큰을 받아도, 자신의 `AllowedResourceID` 하나만 다룰 수 있습니다.

- `GET /aicalls`: 4.5가 거부
- `DELETE /aicalls/{남의 id}`: 미들웨어가 거부
- `POST /aimessages { aicall_id: 남의 id }`: 값이 무시되고 자기 대화에 기록
- `POST /aicalls { reference_type: "contact_case", reference_id: 남의 case }`: 둘 다 강제로 덮어써짐(4.4.1)
- `customer_id:<cid>:aicall:` 구독: UUID 형태 검사에서 거부
- `customer_id:<cid>:aicall:<남의 id>` 구독: 배정 불일치로 거부
- `POST /auth/boot/refresh { allowed_resource_id: 남의 id }`: 본문을 읽지 않고 토큰에서만 값을 취하므로 무효

## 6. 에러 처리

| 상황 | 응답 | 클라이언트 |
|---|---|---|
| 미들웨어 대조 불일치 | 403 | 재부팅 (3.6 계기 5). **정상 흐름에서는 나오지 않으나 롤링 배포 창에서는 나옵니다.** rev3이 "로그만"이라고 쓴 것은 틀렸습니다 |
| `AllowedResourceID`가 Nil인 토큰 | 401 (8.1) | 재부팅 (3.6 계기 3) |
| `ScopeVersion < 2` 토큰 | 401 (9.2) | 재부팅 (3.6 계기 3). **강제 시작 시점에 대량 발생합니다** |
| 부팅 자체가 한도 초과 | 429 | 백오프 후 재시도. 재부팅 상한을 공유 (3.6) |
| 배정 id 소진 후 재생성 | 409 | 재부팅 (3.6 계기 2) |
| 리소스 종료 | WS 이벤트 | 재부팅 (3.6 계기 1) |
| 토큰 만료 임박 | 없음 | 갱신 (3.6 계기 4) |
| 갱신 거부 (검사 2~7 중 하나) | 401 | 재부팅 시도. 부팅도 실패하면(해시 무효화·고객사 동결) 에러 표시 |
| 갱신을 direct 아닌 신원이 호출 | 403 | 해당 없음 |
| 토픽 검증 실패 | 소켓 종료 | 8.3 |

## 7. 테스트 전략

**서버**

- `AuthBoot`이 `AllowedResourceID`를 생성해 JWT와 응답에 넣는지. 매 부팅마다 다른 값인지.
- `AuthBoot`이 `DirectID`/`HashFingerprint`/`BootExpire`를 채우는지. `BootExpire`가 최초 부팅 시각 기준 24시간인지.
- `POST /auth/boot/refresh`가 **같은** `AllowedResourceID`와 **같은** `BootExpire`를 유지하고 `expire`만 갱신하는지. 갱신을 반복해도 `BootExpire`가 밀리지 않는지.
- `POST /auth/boot/refresh`가 인증 없이는 거부되는지. 본문에 `allowed_resource_id`를 넣어도 무시되는지.
- 갱신 검사 1: agent·accesskey·delegate 신원이 403인지. **패닉이나 500이 아닌지**(각 신원별 케이스).
- 갱신 검사 2: `AllowedResourceID`가 Nil인 토큰이 401인지. **새 UUID를 발급하지 않는지.**
- 갱신 검사 3: `BootExpire` 경과 토큰이 401인지.
- 갱신 검사 4: direct 레코드가 삭제된 토큰이 401인지.
- 갱신 검사 5: **해시 재발급 후 기존 토큰의 갱신이 401인지.** 무효화가 갱신을 이기는지 확인하는 핵심 케이스.
- 갱신 검사 6: 동결·삭제 고객사의 토큰이 401인지.
- 갱신 응답에 `resource_data`가 없고 **나머지 `BootResponse` 필드는 전부 있는지**(`customer_id`/`resource_type`/`resource_id`).
- **갱신을 2회 연속** 수행했을 때 `DirectScope` 아홉 필드가 `expire` 외에 전부 동일한지. rev4가 `CustomerID`/`DirectID`/`HashFingerprint`/`ScopeVersion`을 빠뜨린 회귀 고정.
- 갱신 후 토큰으로 WS 토픽 구독이 여전히 통과하는지(`CustomerID` 유실 시 자기 토픽이 거부되는 것을 잡는 케이스).
- 갱신 검사 4·6: **RPC가 오류를 반환할 때도 401인지**(fail closed). `EnforceAccountStatus`의 fail-open 분기를 재사용하지 않았는지.
- 갱신 검사 2: `DirectID`가 Nil이거나 `HashFingerprint`가 빈 문자열일 때 401인지.
- `expire` 절단: `BootExpire` 직전에 갱신했을 때 새 `expire`가 `BootExpire`를 넘지 않는지. **duration 쪽에서 잘렸는지**(`authJWTGenerateWithExpiration`을 그대로 써서 절단이 사라지지 않았는지).
- 갱신 7단계: `directResourceMapping`에 없는 `resource_type`이면 401인지(빈 `AllowedResourceTypes` 토큰을 발급하지 않는지).
- 미들웨어 0단계: `auth_identity`가 없거나 타입이 다르면 401로 abort하는지(통과가 아님).
- 미들웨어 5단계: `DirectScope`가 nil일 때 패닉 없이 403인지.
- 미들웨어 5단계: `AllowedResourceID`가 Nil인 토큰이 경로 id `00000000-0000-0000-0000-000000000000`으로 들어와도 403인지(미들웨어 자기완결성).
- 미들웨어: direct가 아닌 신원(agent/accesskey/delegate)은 그대로 통과하는지.
- 미들웨어: 경로 파라미터 일치 시 통과, 불일치 시 403.
- 미들웨어: 경로 파라미터가 없으면 통과하는지(생성·전송 계열).
- 미들웨어: 파라미터가 있는데 `id`가 아닌 경우 거부되는지(fail-closed).
- 미들웨어: `id`가 파싱 불가능한 문자열일 때 403인지(4.3 4단계).
- 미들웨어: 대소문자·중괄호 형식 UUID가 핸들러와 동일하게 판정되는지(파싱 비교).
- 미들웨어: `GET /aimessages/{message_id}`가 거부되는지(종류 불일치 자동 차단).
- 미들웨어 등록 위치: `RegisterHandlersWithOptions` 이전이어야 적용됨을 회귀로 고정.
- 6개 핸들러 전부: 덮어쓰기 직전 `AllowedResourceID == uuid.Nil` 가드가 동작하는지. `WebchatMessageList`의 기존 `sessionID == uuid.Nil` 가드가 **제거되지 않았는지**.
- 6개 핸들러: 클라이언트가 보낸 대상 id가 무시되고 `AllowedResourceID`가 쓰이는지. **남의 id를 보내도 자기 리소스에 기록되는지**를 명시적으로 검증.
- `AIcallGetsByCustomerID`가 direct를 거부하는지.
- aicall/webchat session 생성 시 배정 id가 실제로 그 id로 저장되는지.
- 배정 번호를 소진한 토큰으로 두 번째 생성 시 409가 반환되는지. 무음 실패나 기존 리소스 반환이 아님을 확인.
- **기본키 충돌이 사전 확인을 통과한 경우에도 409로 매핑되는지**(4.2.2). 사전 확인이 보안 경계가 아님을 고정.
- **`IsErrDuplicate`의 기존 소비자 2곳이 그대로 동작하는지**(4.2.2). `aicallhandler/start.go:444`의 contact_case 재사용 경로와 `aihandler/db.go:229`. db 계층에 센티널을 넣는 구현이 들어오면 실패해야 하는 회귀 테스트.
- 센티널이 드라이버 에러를 감싸지 않아 `IsErrDuplicate`에 걸리지 않는지.
- `AIcallCreate`: direct 신원에서 `reference_type: "contact_case"`가 `none`으로 **강제되는지**. 회귀 고정.
- `AIcallCreate`: direct 신원에서 임의 `reference_id`가 `uuid.Nil`로 **강제되는지**. 4.4.1의 두 번째 강제를 별도 케이스로 검증(하나만 있으면 유니크 인덱스 표면이 남음).
- `buildJWTIdentity`: `AllowedResourceID`가 없는 direct 토큰이 거부되는지. `v1.0`·`authProtected`·WS 업그레이드 세 경로 모두.
- `validateTopics`: 빈 문자열·부분 문자열 리소스 id 거부, 정규 UUID가 아닌 값 거부.
- `validateTopics`: 배정된 id는 통과, 다른 id는 거부.
- `validateTopics`: 비정규 표기(중괄호·대문자)가 거부되는지. 미들웨어와의 의도적 비대칭 고정.
- `validateTopics`: **관리자·매니저의 4조각 trailing colon 구독이 여전히 통과하는지**(문서화된 기능 회귀 방지).
- `validateTopics`: `webchat_widget` direct 토큰이 `aicall` 타입 구독을 여전히 거부하는지.
- `validateTopics`: `DirectScope`가 nil일 때 패닉하지 않고 거부하는지.
- `validateTopic`(단수) 제거 후 11개 케이스가 `Test_validateTopics`로 이관되어 통과하는지.
- `buildJWTIdentity`: `ScopeVersion < 2` 토큰이 401인지. 1·2단계 토큰이 3단계에서 전부 무효화되는지(9.2).
- 갱신이 `ScopeVersion`을 올리지 않고 복사하는지. 갱신으로 무효화를 우회할 수 없는지.
- `AIV1AIcallStart`/`Start`의 **기존 호출자 4곳**(serviceagent_aicall.go:284, service.go:54, service.go:96, StartTask)이 `uuid.Nil`을 넘겨 서버 생성 경로를 그대로 타는지.

**클라이언트 (3.6)**

- 계기 1: 종료 이벤트 수신 시 재부팅이 호출되는지. AI 위젯의 `aicall_id` 가드가 종료 이벤트를 삼키지 않는지.
- 계기 2: 409 응답이 재부팅으로 이어지는지. 일반 에러 토스트로 끝나지 않는지.
- 계기 3: 401 응답이 재부팅으로 이어지는지. 두 앱 모두.
- 계기 4: 만료 타이머가 `/auth/boot`이 아니라 `/auth/boot/refresh`를 호출하는지. **대화 id가 유지되는지**(B2 회귀 고정).
- 계기 4 실패 시 계기 3으로 강등되어 재부팅하는지.
- 계기 4: `refreshAuth()`가 `cachedBoot`를 갱신하고 **다음 타이머를 다시 거는지.** 갱신이 1회로 끝나지 않는지.
- 계기 5: 403이 재부팅으로 이어지고, **실패한 요청을 재생하지 않고 대화를 재초기화하는지.**
- 루프 방지: 연속 재부팅이 3회에서 멈추는지. 백오프가 적용되는지.
- 루프 방지 리셋: 재부팅 후 인증 요청이 한 번 성공하면 카운터가 0이 되는지. **대화를 두 번 종료한 방문자(계기 1)가 계속 동작하는지.**
- 429 응답에서 즉시 재시도하지 않고 백오프하는지.
- 계기 1: AI 위젯이 `aicall_id` 가드를 우회하지 않고, 가드 **앞의** 명시적 분기로 종료를 처리하는지. 상태 객체가 말풍선으로 렌더링되지 않는지.

**프로덕션 E2E**

- 브라우저 두 개로 같은 위젯을 열고, 한쪽이 다른 쪽의 리소스에 접근할 수 없음을 확인.
- 세션을 종료시킨 뒤 재부팅으로 새 대화가 정상 시작되는지.
- **토큰 갱신 시점을 앞당긴 빌드로 갱신 전후 대화 연속성을 확인.** 실제 4시간을 기다리지 않습니다.

## 8. 리스크와 미해결 항목

### 8.1 배포 직후 구형 토큰은 토큰 해석 시점에 거부한다

배포 시점에 이미 발급된 토큰에는 `AllowedResourceID`가 없습니다. 최대 4시간 동안 존재합니다.

**술어는 키의 유무가 아니라 값입니다: `scope.AllowedResourceID == uuid.Nil`이면 거부.** 이 구분이 중요합니다. `DirectScope`의 필드에는 `omitempty`가 없으므로(`models/auth/auth.go:35-40`), 구형 토큰이 갱신을 거치면 `"allowed_resource_id":"00000000-0000-0000-0000-000000000000"`처럼 **키는 있고 값은 0인** 형태로 직렬화됩니다. 키 유무로 판정하면 그 토큰이 강제를 통과해 버리고, 4.4의 덮어쓰기가 `uuid.Nil`을 실제 리소스 id로 저장합니다. 3.6의 갱신 검사 2가 애초에 그런 토큰을 만들지 않지만, 두 곳 모두 값 기준이어야 합니다.

이것이 특히 나쁜 이유는 **4.4가 그 상황을 잡아낼 마지막 가드를 동시에 제거하기 때문**입니다. `WebchatMessageList`의 `sessionID == uuid.Nil` 거부(`webchat_message.go:108-113`)가 그것입니다.

**거부합니다.** 부팅은 인증이 없고 무료입니다. 4시간 동안 같은 테넌트 읽기·삭제·종료 권한이 열려 있는 쪽이 훨씬 나쁩니다.

**단, rev2가 "위젯이 재부팅하면 즉시 복구됩니다"라고 쓴 것은 사실이 아니었습니다.** 두 앱 모두 401 처리가 없어(`square-main/src/lib/api.js:26`, `square-admin/src/webchat-widget-runtime/client.js:303`) 그냥 throw하고 끝납니다. 방문자가 페이지를 새로고침하기 전까지 위젯이 죽어 있습니다.

따라서 **3.6 계기 3(401을 재부팅으로 처리)이 이 절의 전제조건입니다.** 클라이언트 변경 없이 서버만 배포하면 최대 4시간의 위젯 장애가 발생합니다. 배포 순서는 9절을 따릅니다.

**거부 위치가 중요합니다. 미들웨어가 아니라 `buildJWTIdentity`(`lib/middleware/authenticate.go:122-144`)의 direct 분기입니다.**

미들웨어의 경로 파라미터 분기에 두면 구멍이 생깁니다. 구형 토큰이 경로 파라미터 없는 요청(`POST /aicalls` 등)으로 들어오면 2단계에서 통과하고, 4.4의 덮어쓰기가 **`uuid.Nil`을 실제 리소스 id로 저장**합니다. 첫 요청이 0으로 된 UUID 행을 만들고 이후 요청은 전부 중복 키 충돌이 납니다. 보안 결함을 고치려다 데이터 무결성 결함을 넣는 셈입니다.

토큰 해석 시점에 거부하면 `v1.0` 그룹, `authProtected` 그룹, WebSocket 업그레이드가 한 곳에서 함께 덮이고, 4.4의 덮어쓰기가 무조건 안전해집니다.

### 8.2 토큰 갱신과 만료

**갱신(3시간 55분).** 3.6이 `/auth/boot/refresh`로 처리합니다. 배정 번호가 유지되므로 대화가 끊기지 않습니다.

**갱신 실패 또는 완전 만료.** 재부팅으로 새 번호가 배정되므로 이전 대화에 접근할 수 없습니다. 위젯은 새 대화를 시작합니다. **이번에는 그렇게 동작하는 코드를 실제로 작성합니다**(3.6 계기 3).

재부팅 시 이전 id를 넘겨받는 방식은 **채택할 수 없습니다.** 인증 없는 엔드포인트이므로 아무 id나 지정할 수 있게 됩니다. 갱신 경로가 인증을 요구하는 이유가 정확히 이것입니다.

**갱신이 만료를 대체하지는 않습니다.** `BootExpire`(최초 부팅 + 24시간)가 절대 상한이며, 그 이후에는 갱신이 401이고 재부팅만 남습니다. 갱신은 4시간 만료를 24시간으로 늘리는 장치이지 무기한 연장 장치가 아닙니다.

**단, 3.6의 `min` 절단이 있어야 그 서술이 참입니다.** 만료 판정은 바깥 `expire` 클레임으로만 이뤄지므로(`pkg/servicehandler/auth.go:72`), 절단하지 않으면 상한 직전 갱신이 4시간을 더 붙여 실제 최악이 28시간이 됩니다. 라운드 4 보안 리뷰의 지적입니다.

**무효화 지연은 24시간이 아니라 갱신 주기입니다.** 검사 4·5·6이 있으므로 해시 재발급·direct 삭제·고객사 동결은 다음 갱신(최대 3시간 55분)에 반영됩니다. `BootExpire`는 그 검사들이 나중에 제거됐을 때를 위한 최종 방어선입니다.

### 8.3 토픽 거부가 소켓 전체를 끊습니다

`validateTopics`가 false를 반환하면 연결이 종료되어 그 소켓의 다른 구독까지 죽습니다(`subscription.go:140`). 이 설계는 그 동작을 바꾸지 않습니다. 정상 클라이언트는 배정된 토픽만 구독하므로 실무상 문제가 없으나, **거부된 토픽만 건너뛰고 연결을 유지하도록 고치는 것을 후속 티켓으로 등록**합니다.

### 8.4 contact_case 경로의 중복 판별

`bin-ai-manager/pkg/dbhandler/main.go:114` `IsErrDuplicate`가 문자열 매칭입니다.

```go
return strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "UNIQUE constraint failed")
```

`startReferenceTypeContactCase`(`start.go:405`)의 재시도 루프가 이를 사용하는데, **기본키(id) 충돌과 `uq_aicall_active_reference_key` 충돌을 구분하지 못합니다.**

**rev1에서 "direct 토큰은 contact_case를 쓰지 않는다"고 서술했으나 사실이 아니었습니다.** 아무것도 그것을 강제하지 않았습니다. 4.4.1이 direct 분기에서 두 값을 모두 강제하므로 이제 참이 됩니다. 그 강제를 회귀 테스트로 고정합니다. 가정이 아니라 시행된 규칙임을 검증하는 형태여야 합니다.

### 8.5 `validateTopics`의 nil 역참조는 프로세스를 죽인다

`a.DirectScope`가 nil인 상태에서 `a.DirectScope.AllowedResourceID`를 읽으면 패닉입니다. `NewDirectIdentity`가 항상 채우므로 도달 불가로 보이지만, `validateTopics`는 `go h.subscriptionRunWebsock(...)`(`websockhandler/subscription.go:83`) 안에서 실행됩니다. **그 goroutine에는 recover가 없고 `gin.Recovery`의 사정권 밖입니다.**

즉 이 위치의 패닉은 소켓 하나가 끊기는 것이 아니라 **api-manager 프로세스 전체를 종료시킵니다.** 4.6의 nil 가드는 방어적 코드가 아니라 필수입니다.

`subscriptionRunWebsock`에 `defer recover()`를 추가하는 것은 별도 하드닝 티켓으로 등록합니다.

### 8.6 direct 토큰 REST 문서

`GET /aicalls` 차단과 `POST /auth/boot/refresh` 신설은 외부에 보이는 동작 변경입니다. RST 문서에 direct 토큰의 접근 범위와 갱신 절차를 명시합니다. 루트 CLAUDE.md의 문서 동기화 규칙 대상입니다.

`etc.go:39-42`의 코드 주석도 같은 이유로 이번 변경에 포함됩니다(4.6).

### 8.7 이 설계가 고치지 않는 것

읽는 사람이 고쳐졌다고 오해하지 않도록 명시합니다.

- **`AIcallCreate`의 인가 switch에 `default`가 없습니다.** delegate 신원은 `IsAgent()`/`IsAccesskey()`/`IsDirect()` 어디에도 걸리지 않고 switch를 그대로 빠져나가 activeflow와 aicall 생성으로 진입합니다. `customerID`는 AI에서 해석되며 호출자와 대조되지 않습니다. `aicall.go:137`도 같은 형태입니다. **비목표(2절)에 따라 이번 범위 밖이나, 생성 경로가 완전히 잠겼다고 오해하면 안 됩니다.** 후속 티켓으로 등록합니다.
- **동결·삭제된 고객사의 우회.** direct 토큰은 `EnforceAccountStatus`를 통과합니다(`authenticate.go:227-230`에서 즉시 `false`). 즉 **direct 요청 경로는 발급 이후 DB를 한 번도 다시 보지 않습니다.** 살아 있는 토큰을 끊는 유일한 수단이 4시간 만료입니다. 삭제된 고객사의 영구 해시로도 부팅과 사용이 됩니다. **rev3은 여기에 "갱신 경로가 재조회하므로 구멍을 넓히지 않는다"고 썼으나 거짓이었습니다.** rev3의 갱신에는 고객사 상태 검사가 없었습니다. 3.6의 검사 3·4·5·6이 그것을 고칩니다. 그 검사들이 있어야만 이 항목이 "넓어지지 않음"으로 유지됩니다.
- **WebSocket 인가는 구독 시점 1회입니다.** `validateTopics`는 구독 프레임마다 실행되고(`subscription.go:181`), 토큰 만료나 재부팅 시 기존 구독을 재검증하지 않습니다. 클라이언트가 토큰 교체 시 소켓을 유지하므로(`websocket.js:147-163`), 재부팅 후에도 **이전 번호에 대한 구독이 계속 배달됩니다.** 자기 자신의 이전 리소스이므로 교차 방문자 유출은 아니지만, VOIP-1498이 WS 티켓인 만큼 명시합니다. 후속 과제입니다.
- **direct 신원의 공유 레이트 리밋.** `ratelimit:v1_customer_direct:<customerID>`(`customer_ratelimit.go:188`)로 한 고객사의 모든 익명 방문자가 한 버킷을 씁니다. 한 명이 전체를 고갈시킬 수 있습니다.
- **해시의 영속성.** 토큰 하나의 파급 범위는 좁아지지만, 발급 가능한 토큰 수는 제한되지 않습니다.
- **유휴 타임아웃 미구현.** `Widget.SessionIdleTimeout`이 아무 효과가 없다는 사실(3.1)은 이 설계가 고치지 않습니다. 별도 티켓입니다.
- **`reference_type` 열거 검증, `reference_id` 소유권 확인, `AIcallGetByReferenceID`의 고객사 필터.** direct 경로만 4.4.1로 닫습니다. 나머지는 VOIP-1502입니다.
- **미들웨어가 덮는 경로들.** `GET /aicalls/{id}/participants`, `DELETE /aimessages/{id}`, `POST /webchat_sessions/{id}/end`, 그리고 `WebchatMessageGet`(`webchat_message.go:43`), `WebchatMessageDelete`(`:322`), `WebchatSessionGet`(`webchat_session.go:46`), `WebchatSessionDelete`(`:285`). 뒤 넷은 이미 `ErrDirectAccessNotSupported`로 거부하고 있어 회귀가 없습니다. 4.4의 목록이 전부라고 오해하지 마십시오.

### 8.8 후속 과제

- `AIcallCreate`/`AIcallGetsByCustomerID` 인가 switch의 `default` 처리 (8.7)
- 토픽 거부 시 소켓 유지 (8.3)
- `subscriptionRunWebsock`에 recover 추가 (8.5)
- `IsErrDuplicate`를 드라이버 에러 코드 기반으로 교체 (8.4)
- `Widget.SessionIdleTimeout` 실제 구현 (3.1)
- 4조각 이외 토픽 형태의 리소스 id 검증 확대 검토
- 미들웨어가 조회한 리소스를 컨텍스트로 넘겨 핸들러의 중복 조회 제거
- `/aicalls/:id/<child>/:child_id` 형태가 추가될 경우의 2차 파라미터 검증 (4.3)
- WebSocket 구독의 재검증 (재부팅 후 이전 구독이 살아남는 문제, 8.7)
- `AuthJWTParse`의 무검사 타입 단언(`pkg/servicehandler/auth.go:72`, `res["expire"].(string)`)을 `, ok` 형태로 교체. **`BootExpire`와는 무관합니다.** 그 값은 `direct` 클레임 객체 안으로 들어가 `buildJWTIdentity`가 재직렬화·역직렬화하므로(`authenticate.go:132-142`) 이 단언을 거치지 않습니다. 독립된 하드닝 항목입니다

## 9. 착륙 순서

**서버를 먼저 배포하면 최대 4시간의 위젯 장애가 납니다**(8.1). 그리고 **rev3의 단계 구성은 그 장애를 스스로 만들어냈습니다.** 라운드 3의 지적입니다.

### 9.1 rev3의 단계 구성이 틀린 이유

rev3의 1단계는 `AllowedResourceID`를 **발급하되 강제하지 않았습니다.** 그 창에서 부팅한 방문자는 유효한 번호를 든 토큰을 갖지만, 대화는 서버가 생성한 다른 id로 만들어집니다.

```js
// square-main/src/components/widget/useChatState.js:129-138
const aicall = await apiPost('aicalls', {...});
aicallIdRef.current = aicall.id;    // 서버 생성 id. AllowedResourceID와 다름
```

3단계에서 강제가 켜지면 그 방문자는 **다섯 계기 중 어느 것에도 걸리지 않습니다.** 토큰에 번호가 있으니 401이 아니고, 두 번째 생성이 없으니 409가 아니고, 종료된 것이 없으니 WS 이벤트도 없고, 갱신은 틀린 번호를 성실히 이어받습니다. 남는 것은 조용히 망가진 위젯입니다.

rev3의 "3단계 시점의 구형 토큰은 401을 받는다"는 서술은 **1단계 이전에 발급된 토큰**에만 참인데, 그 토큰들은 4시간 만료로 3단계 전에 이미 전부 사라집니다. **단계를 나눈 행위가 깨지는 집단을 새로 만들어낸 것입니다.**

### 9.2 수정된 착륙 순서

`ScopeVersion`으로 단계별 토큰 집단을 명시적으로 구분하고, 강제 시작 시 이전 집단을 일괄 무효화합니다.

| 단계 | 대상 | 내용 | 발급 토큰 | 이 시점의 상태 |
|---|---|---|---|---|
| 1 | monorepo | `DirectScope` 필드 5종 추가, `AuthBoot`이 전부 채움, `/auth/boot/refresh` 신설(3.6의 7단계 검사 포함). **강제·차단·덮어쓰기는 아직 없음** | `ScopeVersion: 1` | 구·신 클라이언트 모두 정상. 새 필드는 아무도 읽지 않음 |
| 2 | monorepo-javascript | **HTTP 상태 코드를 보존하는 전송 계층 리팩터링이 선행**(아래). 그 위에 3.6의 재부팅 핸들러와 계기 1·2·3·5 연결, 갱신 경로를 `/auth/boot/refresh`로 교체(재무장 포함) | 변화 없음 | 클라이언트가 모든 계기를 처리. 갱신이 정상 동작하므로 **긴 대화가 끊기지 않음** |
| 3 | monorepo | `buildJWTIdentity` 거부, 미들웨어, 핸들러 덮어쓰기, 4.4.1 강제, 목록 차단, WS 검증 | `ScopeVersion: 2` | 강제 시작 |

**3단계의 `buildJWTIdentity` 거부 조건은 둘입니다.**

```
scope.AllowedResourceID == uuid.Nil   → 401   (1단계 이전 토큰)
scope.ScopeVersion < 2                → 401   (1·2단계 토큰)
```

이로써 3단계 배포 시점에 살아 있는 **모든** 이전 토큰이 401로 수렴하고, 2단계에서 이미 배포된 계기 3 핸들러가 재부팅합니다. 방문자에게는 짧은 재연결로 보입니다. 9.1의 조용히 망가지는 집단이 존재하지 않습니다.

**롤링 배포 창.** 매니저가 2 replica이므로 3단계 배포 중 강제 인스턴스와 비강제 인스턴스가 잠시 공존합니다. 그 창에서 두 가지가 나타납니다.

- **401이 주된 경로입니다.** 구버전 인스턴스가 아직 `ScopeVersion: 1`을 발급하므로, 그 토큰이 신버전 인스턴스에 닿으면 401입니다. 재부팅해도 다시 구버전 인스턴스에 걸리면 또 401입니다. 균등 분배 기준 기대 반복 2회로 수렴하지만 **최악의 경우 상한이 없으므로**, 3.6의 재부팅 상한·백오프가 이 경로에 반드시 적용돼야 합니다. rev4가 루프 방지를 403에만 걸어 둔 것이 이 지점에서 부족했습니다.
- **403은 부수적입니다.** 강제 이전에 만들어진 대화의 id로 요청이 갈 때 발생하며, 계기 5가 처리합니다.

**"짧은 재연결로 보입니다"는 401 경로에 상한·백오프가 있을 때만 참입니다.** 없으면 부팅이 IP 한도(초당 10회)에 걸려 429가 나오고, 방문자는 재연결이 아니라 정지를 봅니다.

**그리고 전원이 회복되지는 않습니다.** 일부는 상한을 소진하고 에러 화면을 봅니다. 페이지를 새로고침하면 회복됩니다(카운터가 메모리에 있으므로). 유계이고 fail-closed이지만, "전부 짧은 재연결"이라고 쓰면 과장입니다.

**비율은 리셋 규칙을 넣고 계산해야 합니다.** replica 2개, 상한 3 기준입니다.

- 리셋이 없다고 가정하면 주기당 실패 확률 1/2, 소진 확률 **(1/2)³ = 1/8**. 이것이 상한입니다.
- 리셋을 반영하면 주기당 실패는 **부팅이 구버전에 걸리고(1/2) 그 다음 요청이 신버전에 걸릴(1/2)** 때뿐이므로 1/4이고, 소진 확률은 **(1/4)³ ≈ 1/64**입니다.

rev6은 1/8만 적어 두 문단 뒤의 리셋 규칙과 모순됐습니다. **실제 기대값은 1/64 쪽이고, 1/8은 리셋이 동작하지 않을 때의 보수적 상한입니다.**

**리셋 조건이 여기서 두 번째로 중요해집니다.** 구버전 인스턴스에 걸린 요청은 **성공**하고, 그 성공이 카운터를 0으로 되돌립니다. 따라서 상한 3은 "연속 3회 401"에서만 걸리며, 기대 반복 2회로 수렴하는 정상 경로는 상한에 닿지 않습니다.

**2단계의 숨은 선행 작업.** 계기 2·3·5는 전부 HTTP 상태 코드를 봐야 하는데, **두 클라이언트 모두 상태 코드를 버립니다.** `square-main/src/lib/api.js:26-30`은 상태를 `console.error`에만 남기고 일반 `Error`를 던지고, `square-admin/.../client.js:303-306`은 메시지 문자열에 섞어 넣습니다. 따라서 2단계는 재부팅 핸들러를 붙이기 전에 **전송 계층에서 상태 코드를 구조화해 올려 보내는 리팩터링**부터 시작합니다. 구현 도중에 발견되지 않도록 여기에 적어 둡니다.

### 9.3 PR 분할

1단계와 3단계를 하나의 PR로 묶으면 2단계가 낄 자리가 없습니다. **monorepo 쪽은 두 번에 나눠 착륙합니다.** 저장소가 다른 경우와 달리 이것은 같은 저장소 안의 분할이므로, **PR 분할 규칙상 대표님 승인 대상입니다.** 리뷰 완료 후 별도로 여쭙니다.

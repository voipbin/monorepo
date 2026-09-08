# VOIP-1491: 만료 계정 API 게이트 차단

- 작성일: 2026-09-09
- 티켓: VOIP-1491
- 선행: VOIP-1490(복구 경로) 2026-09-08 머지·배포 완료
- 관련: VOIP-1492(고아 리소스 정리), VOIP-1504(보존 정책)

## 1. 이슈 유효성: 유효. 단 **티켓 원문의 범위는 과하다**

`status=expired` 고객이 API를 그대로 사용할 수 있다. 프로덕션에 193건이 있고
살아있는 accesskey 97개(최단 만료 2026-12-31, 최장 2027-09-06, 90일 내 자연 만료 0건),
살아있는 agent 92개를 보유한다. 참고로 `active` 고객은 132명이다.

**HTTP 요청 경로에 한해** 원인은 한 곳뿐이다. `bin-api-manager/lib/middleware/authenticate.go`의
`isBlockedAccountStatus` switch가 `StatusFrozen`과 `StatusDeleted`만 처리하고
`StatusExpired`는 `default: return false`로 통과시킨다.

`cscustomer.StatusExpired`는 api-manager / agent-manager / registrar-manager 어디에도
(vendor·test·`gens/` 생성 코드 제외) 등장하지 않는다. 즉 **요청 경로에서 `expired`를 해석하는 지점은 이 switch 하나뿐이다.**

단 요청 경로 밖에도 미처리 지점이 있다. billing top-up cron이 만료 계정에 매달 지급하는 문제
(VOIP-1506)는 `customer_expired` 이벤트 부재가 원인이며 이 게이트로 막히지 않는다.
"게이트가 모든 것을 닫는다"로 읽히지 않도록 명시해 둔다.

## 2. 티켓 원문에서 철회하는 범위

티켓과 그 코멘트는 "게이트만으로는 자격증명이 닫히지 않는다"며 `/auth/login`,
`/auth/password-forgot`, `/auth/password-reset`, `/auth/unregister`, direct token,
fail-open까지 범위에 넣었다. **검증 결과 그 근거는 성립하지 않는다. 철회한다.**

### 2-1. 인증을 받고 게이트를 건너뛰는 경로는 하나도 없다

`middleware.Authenticate()`는 서비스 전체에서 **정확히 두 곳**에만 등록되며,
**양쪽 모두 바로 다음 줄에 `EnforceAccountStatus()`가 붙어 있다.**

| 위치 | 체인 |
|---|---|
| `cmd/api-manager/main.go:309-310` | `authProtected` 그룹 |
| `cmd/api-manager/main.go:334-336` | `v1.0` 그룹 (414개 라우트) |

세 번째 등록 지점은 없다.

**한 가지 예외적 캐비앳**: 게이트는 HTTP 업그레이드 시점에만 동작한다. 이미 수립된 websocket
구독(`pkg/websockhandler/run.go:18`의 `RunSubscription`, 호출은 `pkg/servicehandler/websock.go:26`)은 배포 후에도 계속 스트리밍된다.
휴면 집단이라 실질 영향은 미미하나 "실질 접근은 0"이 절대적 서술은 아니다.

### 2-2. 개별 근거 반박

| 원 근거 | 검증 결과 |
|---|---|
| `/auth/login`이 JWT를 발급한다 | **철회하지 않는다. 아래 2-4 참조.** 게이트만 놓고 보면 JWT가 갈 곳이 없는 것은 맞으나, fail-open과 조합하면 증폭 경로가 된다 |
| `password-forgot` -> `password-reset`로 로그인 가능 | 메커니즘이 login과 다르다. agent의 `password_hash`는 가입 시 부여된 랜덤값이라 사용자가 모르므로(`bin-agent-manager/pkg/agenthandler/event.go:145-151`), reset은 **없던 자격증명을 새로 만들어낸다.** 다만 2-4의 조치로 로그인 자체가 막히면 비밀번호를 설정해도 쓸 곳이 없으므로 **범위 외로 유지한다** |
| direct token이 검사를 건너뛴다 | **`AuthBoot`이 이미 `StatusActive`를 요구한다**(`pkg/servicehandler/boot.go:108`). 만료 계정은 발급 자체가 불가. 토큰 수명 4시간이라 193건은 전부 소멸 |
| `/auth/unregister` 예외 | 만료 계정에서는 **이미 죽은 분기**. POST(`CustomerSelfFreeze`)는 `dbhandler/customer.go:301`의 `WHERE status='active'` CAS 때문에 0행 갱신 -> `ErrNotFound` -> 400. DELETE(`CustomerSelfRecover`)도 `dbhandler/customer.go:354`가 `status='frozen'`으로 CAS하므로 동일하게 `ErrNotFound`. `immediate: true`의 `FreezeAndDelete`도 내부에서 `Freeze()`를 먼저 호출한다(`freeze.go:80`). **세 경로 전부 막힌다.** 이 예외로 스스로 `active`로 되돌릴 수 없음을 확인했다 |

### 2-3. `/provisioning/extension`도 별도 조치가 불필요하다

`GET /provisioning/extension`은 인증 없이 SIP 자격증명 XML을 반환하며 고객 상태를 보지 않는다
(`pkg/servicehandler/extension.go`의 `ExtensionProvisioningXMLGet`은 `e.TMDelete`만 확인).

그러나 **토큰 공급이 `v1.0` 경로**(`POST /v1.0/extensions/:id/provisioning-token`)이고
`ProvisioningTokenTTL = 10 * time.Minute`(`pkg/servicehandler/extension.go:23`)이다.
게이트를 막으면 신규 발급이 끊기고 기존 토큰은 10분 내 소멸한다.

### 2-4. `/auth/login`은 철회하지 않는다: fail-open과 조합되면 증폭된다

초안은 login을 "게이트가 어차피 막으므로 불필요"로 철회했다. **각각을 따로 논증하고 조합을 보지 않은 오류다.**

- `AuthLogin`(`bin-api-manager/pkg/servicehandler/auth.go:14-41`)은 고객 상태를 확인하지 않는다
- 발급되는 JWT의 수명은 **7일**(`pkg/servicehandler/main.go:114`, `TokenExpiration`)
- 3절의 결론대로 fail-open은 유지한다
- 따라서 만료 계정 보유자는 **7일짜리 JWT를 언제든 새로 찍어두고**, customer-manager나
  RabbitMQ가 흔들릴 때까지 재시도할 수 있다. 그 순간 414개 v1 라우트가 전부 열린다

3절이 fail-open을 정당화하며 든 근거가 "장애 중 살아남는 것은 로컬 HMAC으로 파싱되는 JWT 트래픽뿐"인데,
**그 트래픽 클래스가 바로 만료 계정이 무제한으로 생산할 수 있는 것이다.** 드물고 유계인 창이 아니라,
보유자가 앉아서 기다리면 되는 구조다.

따라서 **`AuthLogin`에 고객 상태 검사를 추가한다.** 이미 `AuthBoot`이 같은 패턴을 쓰고 있고
(`pkg/servicehandler/boot.go:108`), 호출 지점이 하나뿐인 저빈도 경로라 fail-closed의 대가가 작다.
이것으로 증폭 경로가 사라지고, `password-forgot`/`password-reset`도 자동으로 무력해진다.

**이것은 범위 확대가 아니라 축소 과정에서 생긴 구멍을 메우는 것이다.** 대표님이 우려하신
fail-open 전환(전면 장애 위험)과는 성격이 다르다. 로그인 경로 하나에 상태 검사를 넣는 것이다.

### 2-5. 2026-09-08 분석서와의 관계

`docs/plans/2026-09-08-expired-account-cleanup-analysis.md`(커밋 `c6d9ab767`) §7과 §9 item 3은
"login/password-forgot 경로와 fail-open까지 VOIP-1491 범위에 포함시켜야 실제로 닫힌다"고 서술한다.
**이 문서는 그 권고를 부분적으로 정정한다.**

| 항목 | 09-08 분석서 | 이 문서 | 근거 |
|---|---|---|---|
| login | 포함 필요 | **포함 (유지)** | 2-4. 근거는 "게이트를 우회한다"가 아니라 "fail-open과 조합해 증폭한다"로 바뀐다 |
| password-forgot/reset | 포함 필요 | **제외** | login이 막히면 설정한 비밀번호를 쓸 곳이 없다 |
| unregister | 포함 필요 | **제외** | 이미 죽은 분기 (2-2) |
| direct token | 포함 필요 | **제외** | `AuthBoot`이 `StatusActive` 요구 (2-2) |
| fail-open 전환 | 포함 필요 | **제외, 관측만 추가** | 3절. 트레이드오프가 과도 |

09-08 분석서의 해당 절에 이 문서를 가리키는 정정 주석을 함께 추가한다.

## 3. fail-open은 이번 범위에서 제외한다

`authenticate.go`의 고객 조회 RPC 실패 시 fail-open을 fail-closed로 바꾸는 것은
**트레이드오프가 너무 크다.**

- api-manager에 캐시가 없다. 게이트마다 customer-manager로 RPC를 친다
  (`pkg/servicehandler/customer.go`의 `customerGet`, 3초 타임아웃, 재시도 없음).
  캐시는 customer-manager 내부에만 있다
- fail-closed로 바꾸면 customer-manager나 RabbitMQ가 흔들릴 때 **414개 v1 라우트 전부와
  `/auth/delegate`가 함께 죽는다**
- 비대칭이 있다. **accesskey 인증은 이미 fail-closed다.** `AccesskeyRawGetByToken`도
  customer-manager RPC이고 `Authenticate()`가 실패 시 401로 중단한다.
  장애 중 살아남는 것은 로컬 HMAC으로 파싱되는 JWT identity(agent, delegate) 트래픽뿐인데,
  (direct token은 게이트 이전에 return하므로 애초에 해당하지 않는다)
  fail-closed 전환은 그 마지막 생존 경로마저 없앤다

**결정적으로, 이 분기에는 로그도 메트릭도 없어 발동 빈도를 측정할 수 없다.**
판단 근거가 없는 상태에서 전면 장애 위험을 감수할 이유가 없다.

따라서 이번에는 **관측만 추가**하고 전환 여부는 데이터가 쌓인 뒤 별도로 판단한다.

## 4. 설계

### 4-1. 변경 지점 4개

| # | 위치 | 변경 |
|---|---|---|
| a | `authenticate.go`의 `isBlockedAccountStatus` switch | `case cscustomer.StatusExpired:` 추가. 403 + `ACCOUNT_EXPIRED` |
| b | `pkg/servicehandler/auth.go`의 `AuthLogin` | 고객 상태 검사 추가. **거부목록**: `StatusExpired`와 `StatusDeleted`를 거부하고 `active`/`initial`/`frozen`은 통과 (4-1-1) |
| c | `lib/service/auth.go`의 `PostLogin` | 상태 거부를 403 엔벨로프로 매핑. expired -> `ACCOUNT_EXPIRED`(+`details`), deleted -> `ACCOUNT_DELETED`(`details` 없음). 자격증명 오류는 기존 400 유지 |
| d | `authenticate.go`의 고객 조회 실패 분기 | 로그 + Prometheus 카운터 추가. **동작 변경 없음**. 사양은 4-1-3 |

### 4-1-1. (b)의 판정식은 거부목록이어야 한다. `AuthBoot`을 그대로 베끼면 안 된다

`AuthBoot`은 `if cu.Status != cscustomer.StatusActive { reject }`, 즉 `active`만 허용하는
**허용목록**이다(`boot.go:108`). **이것을 그대로 가져오면 두 가지가 깨진다.**

1. **`initial`이 막힌다.** 가입 후 72시간 유예 중인 정상 사용자가 `initial`이며
   (`cleanup.go:39`가 이 상태를 선택하고 `:56`이 `expired`로 바꾼다), 이들을 막으면 VOIP-1490이 만든 온보딩 창이
   무의미해진다.
2. **`frozen`이 막히고, 그 결과 frozen 자가 복구 UX가 끊긴다.** frozen 분기가 `details`로 안내하는
   복구 엔드포인트는 `DELETE /auth/unregister`이고(`authenticate.go:273`), 그 경로는
   `authProtected` 그룹(`cmd/api-manager/main.go:312`)이라 **인증이 필요하다.**
   `DeleteAuthUnregister`는 `auth_identity`가 없으면 401로 중단한다(`lib/service/unregister.go:164-175`).
   즉 **배포된 브라우저 UX가 그 경로에 도달하는 유일한 방법이 `/auth/login`이다.**
(엄밀히는 accesskey로도 인증할 수 있고 `PermissionCustomerAdmin`을 만족하지만
(`models/auth/auth.go:80-81`), frozen 복구 UX는 accesskey를 쓰지 않는다.)
   여기를 막으면 **이미 배포된 frozen 복구 경로가 통째로 사라진다.** 고치려는 버그보다 나쁜 회귀다.

따라서 판정식은 `isBlockedAccountStatus`와 같은 **거부목록**으로 한다.
**`StatusExpired`와 `StatusDeleted`를 거부하고 `active` / `initial` / `frozen`은 통과시킨다.**

`StatusDeleted`를 포함하는 이유는 `Authenticate()`가 걸러줄 것이라 기대할 수 없기 때문이다.
로그인 경로의 실제 필터는 `AgentGetByUsername`의 `FieldDeleted: false`
(`bin-agent-manager/pkg/dbhandler/agent.go:427-429`)인데, 이는 캐스케이드가 agent를 실제로
soft-delete했을 때만 작동한다. **그런데 `authenticate.go`의 `StatusDeleted` 케이스가 존재하는
이유 자체가 "캐스케이드가 일부 고객을 놓친다"(VOIP-1395)는 것이다.** 놓친 고객은 agent가 살아있어
로그인이 되고, 2-4가 닫으려는 것과 동일한 증폭 구조가 남는다. 케이스 하나를 더 추가하는 비용은
`ACCOUNT_DELETED` 매핑뿐이므로 함께 닫는다.
`AuthBoot`에서 가져오는 것은 *메커니즘*(`a.CustomerID`로 고객을 조회해 상태를 본다)이지
그 판정식이 아니다.

**고객 조회 실패 시에는 fail-closed로 거부한다.** 근거는 빈도가 아니다.
3절이 게이트의 fail-open을 유지하기로 한 이유는 장애 중 기존 트래픽을 살리기 위해서인데,
로그인까지 fail-open으로 두면 **장애 창이 곧 자격증명 발급 창**이 되어 2-4가 닫으려는 증폭이
정확히 그 순간 되살아난다. 게이트를 열어두는 대가를 치르는 이상 발급구는 닫아야 한다.

**대가는 명시해 둔다.** customer-manager나 RabbitMQ가 흔들리면 **아무도 새 세션을 얻지 못한다.**
활성 고객 132명 전원과 admin.voipbin.net / talk.voipbin.net이 해당된다.
3절의 비대칭 논거가 "장애 중 살아남는 것은 기존 JWT 트래픽뿐"이었는데, 이 변경은 그것을
**유일한** 생존 경로로 만든다. 수용하되 알고 수용한다.

### 4-1-2. (c)가 없으면 (a)와 (b)가 서로를 상쇄한다

`PostLogin`은 `AuthLogin`의 **모든** 에러를 본문 없는 400으로 뭉갠다.

```go
token, err := serviceHandler.AuthLogin(...)
if err != nil {
    log.Debugf("Login failed. err: %v", err)
    c.AbortWithStatus(400)
    return
}
```
(`bin-api-manager/lib/service/auth.go:56-61`)

엔벨로프도, 사유도, `details`도 없고 403도 아니다. 그러면 (a)+(b)만 적용했을 때
**만료 계정 193건은 로그인을 잃고 아무 안내도 받지 못한다.** 4-2에서 공들여 설계한
`details.recovery_endpoint`는 v1 게이트에서만 나오는데, 로그인이 막혀 거기 도달할 수 없다.
**설계의 두 절반이 서로를 무효화한다.**

따라서 `PostLogin`에서 만료 거부만 403 `cerrors.PermissionDenied` 엔벨로프로 분리해
`ACCOUNT_EXPIRED`와 동일한 `details`를 싣는다. 자격증명 오류는 기존대로 불투명한 400을 유지한다.
`StatusDeleted` 거부도 403 엔벨로프로 매핑하되 코드는 `ACCOUNT_DELETED`, `details`는 싣지 않는다.
게이트가 이미 같은 상태를 `ACCOUNT_DELETED` 403으로 내고 있으므로(`authenticate.go:291`) 일관되고,
expired와 달리 안내할 복구 경로가 없으므로 `details`가 비어 있는 것이 맞다.

센티널은 `pkg/serviceerrors/sentinels.go`에 `ErrAccountExpired`와 `ErrAccountDeleted`로 추가한다.
`abortWithMappedStatus`(`lib/service/unregister.go:55-74`)는 재사용할 수 없다. 그것은 엔벨로프 없이
상태 코드만 낸다.

**중요: 상태 검사는 `AgentV1Login`이 성공한 *뒤에* 수행한다**(`auth.go:22`).
비밀번호 검증 전에 상태를 보면 존재하지 않는 계정과 만료 계정을 구분할 수 있게 되어
username enumeration oracle이 생긴다. `AuthPasswordForgot`이 정확히 그 이유로 모든 오류를
삼키고 있다(`auth.go:79-80`).

### 4-1-3. (d) 관측 사양

3절이 fail-open 전환 판단을 이 카운터에 위임했으므로, 판단에 필요한 차원을 갖춰야 한다.
단순 카운터로는 3절이 던진 질문에 답할 수 없다.

- 위치와 스타일: `lib/middleware/` 패키지 로컬 `prometheus.NewCounterVec` + `MustRegister`.
  같은 패키지의 `ratelimit.go:22-31`이 이미 쓰는 방식을 따른다
  (`pkg/servicehandler/auth_delegate.go:19`의 `promauto` 스타일이 아니다)
- 라벨: **`identity_type`**(agent / accesskey / delegate)과 **에러 클래스**(timeout / not_found / other).
  에러 클래스 판정 순서를 못박는다.
  `errors.Is(err, circuitbreakerhandler.ErrCircuitOpen)` -> `circuit_open`,
  `errors.Is(err, context.DeadlineExceeded)` -> `timeout`,
  `errors.Is(err, commonrequesthandler.ErrNotFound)` -> `not_found`, 그 외 `other`.

  **`circuit_open`을 별도로 두는 것이 중요하다.** 요청 경로에는 회로차단기가 항상 걸려 있고
  (`requesthandler/main.go:1608`, `send_request.go:51-55`), 3절이 이 카운터에 위임한 바로 그
  장애 시나리오에서 차단기가 열리면 이후 호출이 전부 `ErrCircuitOpen`
  (`circuitbreakerhandler/main.go:101`)을 반환한다. 이를 `other`로 뭉치면 정작 판단이 필요한
  구간이 관측되지 않는다. 연결/채널 실패는 `%v`로 포맷되어 unwrap이 안 되므로
  (`rabbitmqhandler/publish.go:55-58, 64-74, 106-108`) 어차피 `other`로 남는다는 점도 감안할 것

`identity_type`이 핵심이다. fail-closed 전환이 안전한지는 **어떤 identity가 이 분기를 타느냐**에 달려 있다.
accesskey가 이 분기를 타는 일은 드물다. `AccesskeyRawGetByToken`도 같은 customer-manager를 호출하므로
(`accesskeys.go:116`) 전면 장애에서는 그 전에 401로 걸린다. 다만 부분 장애에서는
`identity_type="accesskey"` 행이 나올 수 있으므로 "불가능"이 아니라 "드물다"로 본다.
위협은 장애 순간의 만료 agent JWT다. 라벨 없이 총량만 세면 그 구분이 불가능하다.

### 4-2. 응답 형태

기존 `StatusDeleted` 케이스를 그대로 따른다. `cerrors.PermissionDenied`로 403,
에러코드 `ACCOUNT_EXPIRED`.

메시지에는 **복구 경로를 명시한다.** 이 티켓의 목적은 차단이지 방치가 아니다.
VOIP-1490이 `POST /auth/email-verify-resend`를 만들어 두었으므로 그것을 안내한다.

**`details` 배열을 넣는다.** `StatusFrozen` 케이스가 `details`를 싣는 이유는
"admin.voipbin.net, talk.voipbin.net이 삭제 예정일과 복구 엔드포인트로 UX를 렌더링하기 때문"이다
(`authenticate.go:259-263`). expired에도 렌더링할 것이 있다. **복구 엔드포인트다.**
`details`가 없으면 클라이언트가 메시지 문자열을 매칭해야 한다.

```
details: [{"recovery_endpoint": "POST /auth/email-verify-resend"}]
```

**단, 프론트엔드 렌더링은 이 티켓 범위가 아니다.** 4-1(c)로 로그인 실패가 403 `ACCOUNT_EXPIRED`
엔벨로프로 나가므로 사용자는 로그인 화면에서 복구 안내를 받게 되며, 대시보드에 들어와 모든 호출이
403나는 상황은 발생하지 않는다. 프론트가 이 `details`를 실제로 쓰게 하는 작업은 별도 티켓으로 남긴다.

### 4-3. 복구 경로 안내 시 주의: 193건은 두 갈래다

| 구분 | 건수 | 복구 경로 |
|---|---|---|
| 살아있는 agent 보유 | 92 | `POST /auth/email-verify-resend` 로 자가 복구 |
| agent 없음 + `tm_delete` 설정됨 (VOIP-1490 이전 만료) | 101 | 재발송 불가(`resendTarget`이 nil). **재가입은 가능** |
| agent 없음 + `tm_delete` NULL (VOIP-1490 이후 만료) | 현재 0 | **세 문이 모두 닫힌다.** 아래 참조 |

**세 번째 집단이 존재한다.** 초안은 두 갈래로만 봤는데 틀렸다.

`validateCreate`의 고객 중복 검사는 `FieldDeleted: false` + `FieldEmail`이다
(`customerhandler/customer.go:63-75`). 즉 재가입 가능 여부는 전적으로 `tm_delete`에 달려 있다.

- **VOIP-1490 이전** 만료 행은 `tm_delete`가 설정되어 있어 이 필터에 걸리지 않는다 -> 재가입 가능
- **VOIP-1490 이후** 만료 행은 의도적으로 `tm_delete`를 설정하지 않는다(`cleanup.go:52-57`).
  `deleted=false, status=expired`이므로 중복 검사에 **걸린다** -> 재가입 불가

여기에 agent까지 없으면 재발송도 거부되고(`customerhandler/signup.go:449-450`), 이 설계가 로그인까지 막는다.
**세 문이 전부 닫힌다.**

현재 모수는 0이다(agent 없는 101건은 전부 VOIP-1490 이전 행이다). 그러나 앞으로 만료되는 계정 중
agent 생성이 실패한 건이 나오면 그때부터 발생한다. VOIP-1490 설계서 3-4-1이 이미 이 구멍을
기록해 두었고, **이 설계는 그것을 더 확실하게 만든다.**

따라서 4-2의 안내 문구에서 **재가입을 무조건 약속하지 않는다.** 재발송을 우선 안내하고,
그것이 통하지 않으면 문의처로 보낸다. 이 막다른 길 자체는 5절에 후속으로 기록한다.

**초안은 "게이트가 agent 유무를 모른다"고 썼는데 부정확하다.** `a.Type == TypeAgent`이면
그 identity 자체가 해당 고객의 agent이므로 추가 RPC 없이 알 수 있다. 반대로 agent가 없는 101건은 실무상 accesskey만 제시하게 된다
(엄밀히는 agent JWT가 클레임으로 구성되고 생존 조회를 하지 않으므로 soft-delete된 agent의 토큰도
`TypeAgent`로 제시될 수 있으나, 이 집단은 애초에 토큰을 받은 적이 없다).
즉 게이트는 사실상 무료로 두 집단을 구분할 수 있다.

(92라는 수치는 `resendTarget`이 `FieldUsername: c.Email`로 매칭하므로(`customerhandler/signup.go:439-442`)
customer_id 조인으로 구한 값이라면 상한이다. 아래 결론은 어느 쪽이든 바뀌지 않는다.)

**그럼에도 분기하지 않기로 한다.** 이유는 불가능해서가 아니라 메시지 복잡도에 비해 이득이 작기
때문이다. 문구 하나로 충분하다
("인증 메일 재발송을 요청하십시오. 해결되지 않으면 support@voipbin.net으로 문의하십시오").
**재가입을 약속하는 문구는 쓰지 않는다.** 위 세 번째 집단에게는 거짓이 된다.

### 4-4. 테스트

`authenticate_test.go`에 `expired` 케이스를 추가한다. 기존 frozen/deleted 테스트 구조를 따른다.

| 조합 | 기대 |
|---|---|
| expired + agent JWT | 403 `ACCOUNT_EXPIRED` |
| expired + accesskey | 403 `ACCOUNT_EXPIRED` |
| expired + direct token | **통과** (direct는 게이트 이전에 return) |
| expired + `PermissionProjectSuperAdmin` | **통과** (면제) |
| expired + `/auth/unregister` | **통과** (경로 예외) |
| expired + delegate identity | 403 `ACCOUNT_EXPIRED` (frozen/deleted가 VOIP-1292 회귀 커버리지로 delegate를 명시 검증하므로 동일하게 맞춘다) |
| active / initial | 통과 (`initial`은 기존에 없던 **신규 케이스**다) |
| frozen / deleted | 기존 동작 유지 (회귀) |
| **`AuthLogin`** + expired | 거부 (4-1b) |
| **`AuthLogin`** + active / initial | 통과 (회귀) |
| **`AuthLogin`** + **frozen** | **통과 (필수 회귀 가드).** 막으면 `DELETE /auth/unregister` 복구 경로가 끊긴다. 4-1-1 참조 |
| **`AuthLogin`** + deleted | 거부 + `ACCOUNT_DELETED`. 캐스케이드 누락 고객은 agent가 살아있어 로그인이 되므로 여기서 닫는다 (4-1-1) |
| **`AuthLogin`** + 고객 조회 RPC 실패 | **거부**(fail-closed). `PostLogin`은 만료 판정이 아니므로 403 엔벨로프가 아니라 기존 불투명 400을 반환한다 (4-1-1) |
| **`PostLogin`** + expired | 403 + `ACCOUNT_EXPIRED` 엔벨로프 + `details` (4-1c) |
| **`PostLogin`** + deleted | 403 + `ACCOUNT_DELETED` 엔벨로프, `details` 없음 (4-1-2) |
| **`PostLogin`** + 잘못된 비밀번호 | 기존대로 본문 없는 400 (enumeration 방지 회귀) |
| **fail-open 분기** | 동작은 그대로 통과하되 카운터가 증가하는지 검증 (4-1c). 기존 `authenticate_test.go`의 "CustomerRawSelfGet error - fail open" 케이스를 확장한다. 변경 (d) / 4-1-3 |

**구현 주의:** `authenticate_test.go`의 단언 헬퍼가 `tt.customerStatus == StatusDeleted`인지로
2분기해 기대 에러코드를 정하고 기본값이 `ACCOUNT_FROZEN`이다. expired 행을 "기존 구조 그대로"
추가하면 `ACCOUNT_FROZEN`을 단언해 실패한다. **3분기 매핑으로 바꿔야 한다.**

`initial`이 통과해야 한다는 점이 중요하다. 가입 직후 72시간 유예 중인 정상 사용자가
`initial` 상태이며, 이들을 막으면 VOIP-1490이 만든 온보딩 창을 무의미하게 만든다.

## 5. 범위 외 (후속 티켓 후보)

1. **direct token의 런타임 상태 예외.** `authenticate.go`가 `IsDirect()`면 게이트 이전에 return한다.
   활성 중 발급된 토큰은 고객이 frozen/expired가 되어도 최대 4시간 유효하고,
   direct hash 재발급이 기존 토큰을 회수하지 않는다(`bin-direct-manager/pkg/directhandler/handler.go`).
2. **`Freeze()`의 `status='active'` CAS**로 인해 expired 계정의 `/auth/unregister`가 400 막다른 길이 된다
   (`bin-customer-manager/pkg/dbhandler/customer.go:301`).
   **출처는 커밋 메시지가 아니라 코드 주석이다**(`bin-customer-manager/pkg/customerhandler/cleanup.go:52-54`).
   그 주석이 `Delete()`와 `Freeze()`가 모두 도달 가능해진다고 서술하나 `Freeze()`는 여전히 막힌다.
   다음 독자를 오도하는 것은 그 주석이므로 후속 티켓은 주석을 가리켜야 한다.
3. `bin-customer-manager/pkg/listenhandler/v1_customers_freeze.go`가 모든 오류를 400으로 뭉갠다.
4. fail-open -> fail-closed 전환 판단. 이번에 추가하는 카운터에 데이터가 쌓인 뒤.
5. **VOIP-1490 이후 만료되고 agent가 없는 계정은 복구 경로가 0개다 (티켓 미등록).**
   재발송 불가(agent 없음), 재가입 불가(`tm_delete` NULL이라 `validateCreate`에 걸림),
   로그인 불가(이 설계). 현재 모수 0이나 구조적으로 발생 가능하다. 4-3 참조.
   VOIP-1492/1504와 함께 검토하는 것이 자연스럽다.
6. **registrar/kamailio 경로의 `StatusExpired` 미처리 (티켓 미등록).**
   SIP 등록은 api-manager를 거치지 않으므로 이 게이트로는 닫히지 않는다. 193건의 SIP 자격증명은
   그대로 살아있다. 남은 표면 중 가장 크다.
   **그럼에도 이번에 다루지 않는 이유**: (i) 별개 서비스 경로라 이 티켓의 변경과 공유하는 코드가 없고,
   (ii) VOIP-1489~1496에서 보았듯 SIP 경로에 상태 게이트를 넣는 변경은 자체 blast radius가 크며
   (계정 상태 거부가 SIP 500으로 표면화된 것이 VOIP-1490의 발단이었다),
   (iii) `/provisioning/extension`과 달리 공급을 끊는 방식으로 우회할 수 없어 실제 설계가 필요하다.
   별도 티켓으로 등록할 것.

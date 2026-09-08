# VOIP-1490: 이메일 미인증 만료 계정 복구 경로

- 작성일: 2026-09-08
- 티켓: VOIP-1490
- 관련: VOIP-1489(SIP 4xx), VOIP-1491(StatusExpired 게이트), VOIP-1492(고아 자격증명 정리)

## 1. 배경

2026-09-07 고객 서포트 문의(SIP 500)를 조사하다 발견한 결함이다. 가입 후 1시간 안에
이메일 인증을 마치지 못한 고객은 그 상태에서 벗어날 방법이 사실상 없다. 정확히는 agent row가
살아있는 경우 재가입까지 막혀 완전히 갇히고, agent가 없는 경우에만 재가입이 우연히 열려 있다
(1절 표와 3-4-1 참조). (b) 이후 신규 만료 계정은 예외 없이 전자에 해당하게 된다.

프로덕션 기준(2026-09-08 조회) `status=expired` 고객이 **193명** 누적되어 있으며, 193건 전부
`tm_delete`가 설정되어 있고 전부 미인증이다. 그 계정들에 살아있고 미만료인 accesskey가 97건,
살아있는 agent가 92건이다. 살아있는 agent 92건은 동시에 해당 이메일 92개의 재가입을 영구
차단하고 있다.

**193건은 균질하지 않다.** agent row 유무로 두 코호트로 갈린다.

| agent | 건수 | `tm_create` 범위 |
|---|---|---|
| 있음 | 92 | 2026-04-03 ~ 2026-09-06 |
| 없음 | 101 | 2026-02-23 ~ 2026-04-26 |

agent가 없는 101건은 `EventCustomerCreated`의 agent 자동 생성이 안정화되기 이전(대략 2026-04)에
가입한 행들이다. 이 구분이 설계에 직접 영향을 준다(3-4-1 참조).

### 1-1. 도입 이력

| 시점 | 커밋 | 내용 |
|---|---|---|
| 2026-02-08 | `01a1e0c6d` | 최초 도입. `CustomerHardDelete`로 미인증 고객을 DB에서 영구 삭제 |
| 2026-02-23 | `f1aeb4a59` | hard delete -> soft delete로 변경. `status=expired` + `tm_delete=now` |

2/23 커밋 메시지는 "Change cleanup from hard delete to soft delete with status=expired"
한 줄이 전부이며 이유가 기록되어 있지 않다(NOJIRA). 완전 삭제일 때는 재가입이 자연스럽게
열려 있었으나, `tm_delete`를 남기도록 바뀌면서 캐스케이드 봉인과 accesskey 잔존이 발생했다.

### 1-2. 복구 불가 근거 (전부 코드로 확인)

1. 인증 링크 TTL(`emailVerifyTokenTTL = 1시간`, customerhandler/signup.go:20)이 cleanup
   윈도우(`unverifiedMaxAge = 1시간`, cleanup.go:13)와 동일해, 만료 시 Redis 토큰도 함께 소멸한다.
2. 재발송 엔드포인트가 없다. `sendSignupVerification`은 `Signup`에서만 호출되며,
   customer-manager RPC도 토큰을 소비하는 `/v1/customers/email_verify`만 있다.
   (openapi 문서화된 auth 경로는 boot / signup / email-verify / unregister / password-forgot /
   password-reset 이고, gin 라우터에는 이 외에 `/auth/login`과 `/auth/delegate`도 등록되어 있다.)
3. `/auth/unregister`도 실패한다. `unregister.go:138,145` -> `CustomerSelfFreezeAndDelete` /
   `CustomerSelfFreeze` -> `freeze.go:70` / `:16`으로 가는데, `Freeze()`가 `freeze.go:37-40`에서
   `TMDelete != nil`이면 `"cannot freeze a deleted customer"` 에러를 반환한다. 조용한 no-op이
   아니라 `abortWithMappedStatus`(unregister.go:55-74)로 HTTP 오류가 사용자에게 나간다.
4. `EmailVerify`(signup.go:128~)는 `email_verified`와 `status`만 설정하고 `tm_delete`를
   복구하지 않는다.
5. 동일 이메일 재가입도 막힌다. `validateCreate`(customerhandler/customer.go:51-98)의 고객
   중복 검사(:63-75)는 `FieldDeleted:false` 필터라 soft-delete된 행에 걸리지 않지만, 에이전트
   중복 검사(:78-95)는 `FieldUsername: email, FieldDeleted:false`이고 cleanup이 agent를
   건드리지 않으므로 매칭된다.

### 1-3. 부수 확인 사항

`customerhandler/signup.go:117` 주석은 "If the verification email fails, the customer can
re-request signup to trigger a new email."이라고 서술하지만, 위 5번 때문에 구현된 적이 없다.

**가입 시 메일은 한 통만 나간다.** `Signup`은 `customer_created` 이벤트를 `Headless: true`로
발행하고(signup.go:111), agent-manager는 `if !headless`일 때만 Welcome 비밀번호 메일을 보낸다
(agenthandler/event.go:177-185). 따라서 가입 시 발송되는 것은 인증 메일 하나뿐이며,
`EmailVerify`가 호출하는 `AgentV1PasswordForgot`(signup.go:202)이 **고객이 비밀번호를 설정할 수
있는 유일한 메일**이다. 이 호출은 중복이 아니므로 제거해서는 안 된다.

비밀번호 복구 자체는 `/auth/password-forgot`으로 가능하다(agent row가 살아있어
`AgentGetByUsername`이 성공한다). 따라서 **막혀 있는 것은 이메일 인증 하나뿐이다.**

## 2. 목표와 비목표

### 목표
- 미인증 계정이 복구 불가능해지는 상태를 제거한다.
- 이미 만료된 193건 중 **agent가 살아있는 92건**을 별도 데이터 마이그레이션 없이 복구 가능하게 한다.

### 비목표
- 만료 계정의 API 접근 차단(VOIP-1491)
- 기존 고아 자격증명 정리 및 agent 없는 만료 계정(현재 101건 및 향후 동종 케이스) 처리(VOIP-1492)
- 인증 전 리소스 사용 게이팅
- hard delete 복원
- `/auth/unregister`가 expired 계정에서 동작하게 만드는 것 (3-8 참조)

## 3. 설계

### 3-1. 변경 지점

| # | 위치 | 변경 |
|---|---|---|
| a | `customerhandler/signup.go:20`, `cleanup.go:13` | `emailVerifyTokenTTL` 1h -> 24h, `unverifiedMaxAge` 1h -> 72h |
| b | `cleanup.go:28-31`, `:44-51` | 필터에 `status=initial` 추가, `tm_delete` 설정 제거 |
| c | `signup.go`의 `EmailVerify` | 상태 가드 추가 + 복구 시 `tm_delete = NULL` 함께 설정 |
| d | 신규 | `POST /auth/email-verify-resend` + RPC `/v1/customers/email_verify_resend` |
| e | `signup.go:114-117` 주석, `signup.go:253` 메일 본문, `lib/service/signup.go:170` HTML | 문구 수정 |
| f | `bin-openapi-manager`, `bin-api-manager/docsdev` | 신규 엔드포인트 문서화 |

(b)의 `status=initial` 필터는 필수다. `cleanup.go:28-31`의 현재 필터는
`{email_verified:false, deleted:false}`뿐이라 `tm_delete`만이 행을 결과에서 제외한다.
`tm_delete` 설정을 제거하면서 필터를 그대로 두면 만료된 행이 15분 주기 cron
(`0c037bf0a362_..._ticker_.py:129`)마다 영원히 재선택되어, 매회 UPDATE + Redis 캐시 재기록 +
로그 + 비정상 카운트를 발생시킨다.
필터는 `status=initial` 허용목록으로 표현한다. `ApplyFields`는 `NotEq`
(bin-common-handler/pkg/databasehandler/main.go:41-57, 처리부 :85-87)로 제외 조건도 표현할 수
있으나, 새로 추가될 상태를 자동으로 배제하지 않는 허용목록이 더 안전하고
`idx_customer_customers_status` 인덱스와도 맞는다.

(c) 자체는 `tm_delete`가 설정된 193건 모두에 적용되나, 재발송이 열리는 것은 agent가 살아있는
92건이다(3-4-1). **별도 데이터 마이그레이션이 필요 없는 이유가 이것이다.**

### 3-2. (b)가 실제로 얻는 것과 잃는 것

초안에서 "(b)가 캐스케이드 봉인, 재가입 차단, Delete()/Freeze() 불능을 동시에 해소한다"고
서술했으나 이는 부정확했다. 정정한다.

**해소되는 것**
- `Delete()`(customerhandler/customer.go:36-40)의 조기 return이 풀려 `customer_deleted`
  캐스케이드가 정상 동작한다.
- 문서화된 불변식이 **복원된다.** 마이그레이션 `dafeedbccfa5`는
  `update customer_customers set status = 'deleted' where tm_delete is not null`로
  `tm_delete 설정 <-> status=deleted` 대응을 세웠다. 현재 cleanup은 `status=expired`인데
  `tm_delete`를 설정해 이 불변식을 깨고 있다. (b)는 이를 되돌린다.

**해소되지 않는 것 (초안의 오류)**
- **재가입 차단은 풀리지 않는다.** 차단 주체는 agent 중복 검사(customer.go:78-95)이고
  cleanup은 agent를 건드리지 않는다. 오히려 `tm_delete`가 없어지면 고객 중복 검사
  (customer.go:63-75)에도 걸리게 되어 이중으로 막힌다. 재가입은 이 티켓의 복구 수단이 아니므로
  (재발송이 그 역할을 한다) 실질 영향은 없다.
- **`Freeze()`는 여전히 실패하며, 사용자 관점의 변화도 전혀 없다.** 현재는 `freeze.go:37-40`이
  `TMDelete != nil`로 거부하고, (b) 이후에는 `dbhandler.CustomerFreeze`의 `WHERE status='active'`
  가드(dbhandler/customer.go:301)에 걸려 0행 갱신 -> `ErrNotFound`가 된다. 그런데
  `v1_customers_freeze.go:29-32`가 `Freeze`의 **모든** 오류를 `simpleResponse(400)`으로 뭉개고
  (`:65-67`의 `FreezeAndDelete`도 동일), 이것이 `parseResponse` -> `ErrBadRequest`
  (bin-common-handler/pkg/requesthandler/common.go:20,108-119,153) ->
  `abortWithMappedStatus`(unregister.go:69-70)를 거쳐 **HTTP 400**이 된다.
  즉 expired 계정의 `/auth/unregister`는 (b) 전후 모두 400이며 관측 가능한 변화가 없다.
  3-8 참조.

**받아들이는 부작용**
- 신규 만료 고객이 ProjectSuperAdmin 고객 목록에 나타난다(server/customers.go:95-97이
  `deleted:false`를 하드코딩). 193건이 아무도 모르게 쌓인 원인이 이 비가시성이었으므로
  가시화는 바람직한 방향으로 판단한다. 다만 관리자에게는 보이는 목록이 늘어나는 변화다.
- expired 고객의 `customer_*` 웹훅 페이로드에서 `tm_delete`가 `null`로 직렬화된다.

### 3-2-1. 유예 연장(1h -> 72h)의 보안 영향

API 접근은 변화가 없다. `isBlockedAccountStatus`(bin-api-manager/lib/middleware/authenticate.go:255-300)는
`frozen`과 `deleted`만 차단하고 `initial`과 `expired`를 똑같이 통과시키므로, 현재 `expired` 상태
자체가 아무것도 게이팅하지 않는다(그 문제가 VOIP-1491이다). 가입 시 발급되는 accesskey도 만료
여부와 무관하게 1년간 살아있다. 아웃바운드 통화도 `active`를 요구하므로
(bin-call-manager/pkg/callhandler/validate.go:39) 영향이 없다.

**유일한 델타는 인바운드 통화다.** `ValidateCustomerStatusIncoming`(validate.go:66)은 `active`와
`initial`을 허용하고 `expired`를 거부하므로, 미인증 계정이 인바운드 통화를 받을 수 있는 기간이
1시간에서 72시간으로 늘어난다.

이 델타는 수용한다. 남용 시나리오는 가짜 이메일로 가입해 72시간 동안 인바운드 통화를 받는
것인데, 동일 계정이 이미 1년짜리 accesskey와 API 접근을 만료 여부와 무관하게 보유하고 있어
한계 이득이 작고, 통화량은 free 플랜 토큰 잔액에 묶인다. 대신 이 델타를 명시해 VOIP-1491이
게이트를 설계할 때 함께 고려하도록 한다.

### 3-3. 경로명 근거

`/auth` 그룹은 전부 2단계 평면 kebab-case이며 3단계가 하나도 없다. 따라서
`/auth/email-verify/resend`는 이 그룹의 규칙에 맞지 않는다.
(v1.0 그룹에는 `{id}` 없는 3단계 경로가 일부 존재하므로 -- `/service_agents/contacts/lookup`,
`/numbers/renew` 등 -- "전 API에 그런 경로가 없다"는 주장은 성립하지 않는다. 근거는
`/auth` 그룹 내부의 일관성이다.)

`/auth/email-verify-resend`는 2단계 평면 kebab-case로 `/auth` 규칙에 맞고, `email-verify`
접두사를 공유해 문서와 라우트 목록에서 두 경로가 인접 정렬된다. 기존에 이미 대칭 쌍이 있으며
비어 있던 왼쪽 칸을 채우는 작업이다.

| 발송(요청) | 토큰 소비 |
|---|---|
| `password-forgot` | `password-reset` |
| `email-verify-resend` (신규) | `email-verify` |

RPC는 기존 규칙(snake_case)에 맞춰 `/v1/customers/email_verify_resend`,
requesthandler 메서드는 `CustomerV1CustomerEmailVerifyResend`로 한다.

### 3-4. 엔드포인트 계약

```
POST /auth/email-verify-resend
Request : {"email": "user@example.com"}
Response: 항상 200 {}
```

라우트는 **공개 `/auth` 그룹**(cmd/api-manager/main.go:290-299)에 등록한다. 이 그룹이
`RateLimit("auth_public")`을 제공한다. 인증이 필요한 `authProtected` 그룹(:306-312)에 넣으면
안 된다. 복구 대상 사용자는 인증을 통과하지 못할 수 있기 때문이다.

응답을 항상 200으로 고정하는 것은 `POST /auth/signup`의 기존 enumeration 방지 정책과 동일하다
(lib/service/signup.go:70-73은 실패 시에도 200과 빈 바디를 반환한다). 형식이 잘못된 이메일도
200을 반환한다. api-manager 바인딩은 `binding:"required"`만 하고 형식 검증은
`utilHandler.EmailIsValid`로 customer-manager에서 이뤄지므로, 400을 내려면 별도 검증기를
추가해야 하는데 그 이득이 없다.

**고객 선택 규칙.** 이메일로 조회하되 `deleted` 필터를 넣지 않는다(레거시 193건이 `tm_delete`
설정 상태이기 때문). 이메일에 유일성 제약이 없고 필터도 없으므로 복수 행이 나올 수 있다
(soft-delete 후 재가입, VOIP-1492 정리 이후 등). 따라서 `status=deleted`인 행을 제외한 뒤
`tm_create`가 가장 최근인 행 하나를 대상으로 한다. 남는 행이 없으면 아무것도 하지 않는다.

내부 분기:

| 대상 고객 상태 | 동작 |
|---|---|
| 해당 고객 없음 | 아무것도 하지 않음 |
| `email_verified == true` | 아무것도 하지 않음 |
| `status` 가 `frozen` 또는 `deleted` | 아무것도 하지 않음 |
| 살아있는 agent row가 없음 | 아무것도 하지 않음 (3-4-1) |
| `status` 가 `initial` 또는 `expired` 이고 미인증 | 쿨다운/상한 확인 후 토큰 발급 + 발송 |

### 3-4-1. agent가 없는 고객에는 재발송하지 않는다

만료 193건 중 101건은 살아있는 agent row가 없다. 이들에게 재발송을 허용하면 복구가 아니라
**개악이 된다.**

- 인증이 끝나면 `EmailVerify`가 `AgentV1PasswordForgot`(signup.go:202)을 호출하는데,
  `AgentGetByUsername`(agenthandler/agent.go:478-482)이 실패해 `"agent not found"`로 끝난다.
  로그만 남고(signup.go:203) 비밀번호를 설정할 경로가 없으므로 대시보드 로그인이 불가능하다.
  `/auth/password-forgot`도 같은 이유로 실패한다.
- 더 나쁜 것은 현재 열려 있는 탈출구가 닫힌다는 점이다. 이 101건은 `tm_delete`가 설정되어 있어
  `validateCreate`의 고객 중복 검사(customer.go:63-75, `deleted:false`)에 걸리지 않고 agent도
  없으므로(:78-95), **지금은 동일 이메일 재가입이 가능하다.** 인증으로 `tm_delete`를 지우면
  고객 중복 검사에 걸려 재가입이 영구 차단된다.

따라서 재발송 대상에서 제외한다. 이 101건은 현행 재가입 경로를 그대로 두고, 근본 정리는
VOIP-1492에서 다룬다.

**남는 구멍 하나를 명시해 둔다.** (b) 이후에 만들어지는 만료 행은 `tm_delete`가 없으므로 고객
중복 검사에 걸려 재가입이 막힌다. 여기에 더해 agent까지 없으면 재발송도 거부되어 완전히 갇힌다.
`customer_created`는 제한된 재시도 후 메시지를 폐기하므로
(bin-common-handler/pkg/rabbitmqhandler/consume.go:145-152) 이론적으로 발생 가능하다. 다만
프로덕션에서 agent 없는 `active` 고객은 6건인데 전부 2026-04-25 이전 가입이므로, agent 자동
생성 안정화 이후로 한정하면 **0건**이다. 재발송 시 agent를 재프로비저닝하는 처리는 VOIP-1492 범위로 넘긴다. 판정은 customer-manager에서 `AgentV1AgentList`로 수행하며, 이는
`validateCreate`(customer.go:78-95)가 이미 쓰는 방식이라 새로운 결합이 아니다.

### 3-5. 남용 방지

`RateLimit("auth_public")`(lib/middleware/ratelimit.go:48-98,142-144)은 IP 기준이며 `ipLimiter`
맵을 쓰는 replica별 in-process 구현이다. 따라서 (1) 같은 이메일에 대한 반복 요청(메일 폭탄)을
막지 못하고 (2) 2 replica 환경에서 실효 한도가 2배가 된다. 이메일별 제한은 Redis를 쓰는
customer-manager 계층에 두어야 한다.

**키는 이메일이 아니라 `customer_id`로 한다.** 프로덕션 `customer_customers.email`의 collation은
`utf8mb3_uca1400_ai_ci`로 대소문자/악센트 비구분이다. 따라서 `Victim@x.com`과 `victim@x.com`은
같은 고객 행을 가리키지만 이메일을 키로 쓰면 서로 다른 Redis 키가 되어 쿨다운과 상한이
그대로 우회된다. 고객을 먼저 확정한 뒤 그 UUID로 제한하면 정규화 문제가 사라진다.
기존 캐시 키가 모두 UUID 기반(`customer:<id>`, `verify_lock:<id>`)인 것과도 일관된다.

| 키 | 용도 | 한도 |
|---|---|---|
| `email_verify_resend_cd:<customer_id>` | 쿨다운 | TTL 60초 |
| `email_verify_resend_n:<customer_id>` | 일일 상한 | TTL 24시간, 5회 |

**원자성.** 쿨다운은 기존 `SetNX`(cachehandler/handler.go:131)로 충분하다. `SetNX`가 false를
반환하면 발송을 건너뛴다. 상한 카운터는 GET -> 비교 -> SET 형태로 구현하면 2 replica에서 경쟁이
발생하므로 다음 순서를 따른다.

1. `INCR n_key` 실행
2. 결과가 1이면 `EXPIRE n_key 86400`
3. `TTL n_key` 가 음수이면(만료 설정 누락) 방어적으로 `EXPIRE n_key 86400` 재설정
4. 결과가 5를 초과하면 발송을 건너뜀

3번이 없으면 2번의 `EXPIRE`가 실패하거나 그 사이 프로세스가 죽었을 때 키가 TTL 없이 남아
해당 고객이 **영구히 재발송 불가**가 된다. 남용 방지 장치가 이 티켓이 만들려는 복구 경로 자체를
차단하는 형태가 되므로 반드시 포함한다.

`cachehandler`에 `Incr`, `Expire`, `TTL` 프리미티브가 없으므로 추가한다
(현재 `main.go:25-40` 인터페이스에 없음).

어느 제한에 걸려도 **발송만 건너뛰고 응답은 200을 유지한다.**

일일 상한 5회와 토큰 TTL 24시간은 서로 독립적인 창이므로, 창 경계에서는 동시 유효 토큰이
최대 10개까지 나올 수 있다. 무한정이 아니라 상수로 묶인다는 점이 중요하므로, 이전 토큰을
무효화하기 위한 customer -> token 역인덱스는 별도로 두지 않는다.

### 3-6. `EmailVerify` 상태 가드

현재 `EmailVerify`는 `c.EmailVerified`만 확인한다(signup.go:166). 토큰 수명이 1시간에서
24시간으로 늘고 재발송까지 생기면 토큰이 살아있는 동안 고객 상태가 바뀔 여지가 훨씬 커진다.
(c)가 `tm_delete=NULL`까지 설정하므로, 가드가 없으면 `status=deleted`이고 PII가 익명화된 행까지
`active`로 되살릴 수 있다.

가드는 **거부목록**으로 정의한다. `status`가 `deleted` 또는 `frozen`이면 거부하고, 그 외에는
통과시킨다. 재발송 분기표(3-4)에만 두지 않고 `EmailVerify` 자체에 둔다. 토큰을 발급받은 시점과
사용하는 시점 사이에 상태가 바뀔 수 있기 때문이다.

**가드는 기존 `if c.EmailVerified` 조기 return(signup.go:166-173)보다 앞에 둔다.** 뒤에 두면
인증 후 삭제되어 PII가 익명화된 고객의 낡은 토큰이 그 행을 그대로 호출자에게 반환한다.

허용목록(`initial`/`expired`만 통과)으로 만들면 안 된다. 3-5가 동시 유효 토큰을 최대 10개까지
허용하므로, 사용자가 새 링크로 인증한 뒤 오래된 링크를 클릭하는 경우가 실제로 발생한다. 그때
고객은 이미 `active` + `email_verified=true`이므로 허용목록에서 탈락해 거부되고,
`lib/service/signup.go:170`이 "링크가 만료되었거나 유효하지 않습니다"를 띄운다. 낡은 토큰도
삭제되지 않는다(현재는 `customerhandler/signup.go:170`에서 삭제된다). 이미 인증된 사용자가 재발송을 다시 요청해 5회
상한을 소모하게 되는 흐름이라, 이 티켓이 만들려는 경로를 스스로 망가뜨린다.

### 3-7. 상태 전이

```
signup -> initial (email_verified=false)
  |- 24h 내 링크 클릭 -> active, email_verified=true, tm_delete=NULL
  |- 링크 만료        -> resend로 새 링크 (72h 유예 내내 가능)
  |- 72h 경과         -> expired (tm_delete 없음)
                         resend 계속 동작(agent가 살아있는 경우, 3-4-1), 인증 시 active 복귀
```

레거시 경로(agent가 살아있는 92건):
```
expired + tm_delete 설정됨 -> resend -> 링크 클릭
  -> active, email_verified=true, tm_delete=NULL (완전 복구)
```

### 3-8. `/auth/unregister`를 이 티켓에서 고치지 않는 이유

expired 계정의 `/auth/unregister`는 (b) 전후 모두 **HTTP 400**을 반환한다(3-2 참조).
실패 지점만 `freeze.go:37-40`에서 `CustomerFreeze`의 상태 가드로 옮겨갈 뿐, 사용자에게 보이는
동작은 동일하다. 즉 (b)는 이 경로를 개선하지도 악화시키지도 않는다.

이를 고치려면 `CustomerFreeze`의 상태 가드를 완화하거나 expired 전용 경로를 만들어야 하는데,
이는 계정 수명주기 정책 변경이라 VOIP-1491(StatusExpired 게이트)의 영역이다. 이 티켓의 복구
경로는 재발송 -> 인증이며 unregister에 의존하지 않고, (b)가 이 경로를 악화시키지도 않으므로
범위 밖으로 둔다. VOIP-1491에 기록한다.

### 3-9. 컴포넌트 경계

| 계층 | 책임 |
|---|---|
| `bin-api-manager` route + `lib/service` | 요청 바인딩, 항상 200 반환. 상태 판단 없음. 인증 실패 HTML의 재발송 폼(3-10-1) 포함 |
| `bin-api-manager/pkg/servicehandler` | RPC 호출 위임 |
| `bin-common-handler/pkg/requesthandler` | `CustomerV1CustomerEmailVerifyResend` |
| `bin-customer-manager/pkg/listenhandler` | `/v1/customers/email_verify_resend` 라우팅 |
| `bin-customer-manager/pkg/customerhandler` | 고객 조회/선택, 상태 분기, 쿨다운/상한, 토큰 발급, 발송 |
| `bin-customer-manager/pkg/cachehandler` | 쿨다운/상한 키. 쿨다운용 신규 메서드(기존 `VerifyLockAcquire`는 잠금 전용이라 재사용 불가)와 상한용 `Incr`/`Expire`/`TTL` 프리미티브 추가 |
| `bin-openapi-manager` | `openapi/paths/auth/email-verify-resend.yaml` 추가 |
| `bin-api-manager/docsdev` | `auth_overview.rst` 갱신 |

상태 판단과 남용 방지 로직 전부를 customer-manager에 두어 api-manager는 얇게 유지한다.
발송 자체는 기존 `sendSignupVerification`을 재사용한다.

### 3-10. 문서 및 사용자 문구

기존 문구 중 이번 변경으로 사실과 어긋나게 되는 것들을 함께 고친다.

| 위치 | 현재 | 문제 |
|---|---|---|
| `customerhandler/signup.go:253` | 메일 본문 "(expires in 1 hour)" | TTL이 24h로 바뀜 |
| `lib/service/signup.go:170` | 인증 실패 HTML "Please sign up again." | 재가입은 동작하지 않는다(1-2.5). 아래 3-10-1 참조 |
| `customerhandler/signup.go:114-117` | "can re-request signup to trigger a new email" | 구현된 적 없음 |
| `docsdev/source/auth_overview.rst` | 엔드포인트 표(L25-68), 엔드포인트 절(L175), 수명주기 다이어그램(L84-92), rate limit 설명(L72) | 신규 엔드포인트 누락, 다이어그램이 만료를 종착 상태로 표현, L72는 모든 `/auth/*`가 IP 제한만 공유한다고 서술 |

### 3-10-1. 인증 실패 화면에 재발송 폼을 넣는다

`GetCustomerEmailVerify`가 서빙하는 HTML(lib/service/signup.go:124-183)의 실패 분기는 이 티켓의
대상 사용자가 **실제로 도달하는 유일한 화면**이다. 그런데 이 페이지는 토큰만 들고 있고 사용자
이메일을 모르는 반면, `POST /auth/email-verify-resend`는 이메일을 요구한다. 따라서 문구만 고치면
사용자는 같은 막다른 길에 그대로 남는다.

실패 분기에 이메일 입력 필드와 재발송 버튼을 추가하고, 버튼이
`fetch('/auth/email-verify-resend', {method:'POST', body:{email}})`를 호출하도록 한다.
응답이 무엇이든 화면 문구는 "해당 주소로 계정이 있다면 새 인증 메일을 보냈습니다" 형태로
**항상 동일하게** 표시한다. 3-4의 enumeration 방지 정책이 응답 본문뿐 아니라 화면 문구에도
그대로 적용되어야 하기 때문이다.

`emailVerifyHTML`은 `%s` 하나를 갖는 `fmt.Sprintf` 템플릿이고 CSS의 퍼센트가 `%%`로
이스케이프되어 있으므로(lib/service/signup.go:120,133,136), 마크업 추가 시 이 규칙을 유지해야 한다.

참고로 이 화면은 형식이 올바르지만 만료/무효인 토큰으로만 도달한다. 형식이 깨진 토큰은
`signup.go:115-117`의 `validVerifyToken` 검사에서 렌더링 전에 400으로 걸러진다.

이로 인해 `lib/service`의 책임이 "요청 바인딩"에 그치지 않고 정적 HTML 한 벌을 더 갖게 된다.
3-9 표에 반영한다.

### 3-11. 에러 처리

고객 조회 실패, `AgentV1AgentList` 실패, Redis 실패, 이메일 발송 실패 모두 로그만 남기고 200을
반환한다. agent 조회가 실패하면 **fail-closed로 발송을 건너뛴다.** 같은 RPC를 쓰는
`validateCreate`(customer.go:86-90)의 기존 처리와 일치한다. 재발송은
best-effort 액션이며, 실패를 노출하면 enumeration 통로가 된다.

## 4. 테스트

| 대상 | 검증 |
|---|---|
| `CleanupUnverified` | `tm_delete`를 설정하지 않고 `status=expired`만 기록 |
| `CleanupUnverified` | 이미 `expired`인 행을 두 번째 실행에서 다시 선택하지 않음 |
| `EmailVerify` | `tm_delete`가 설정된 expired 행을 완전 복구(레거시 92건 시나리오) |
| `EmailVerify` | `deleted` / `frozen` 상태는 거부 |
| `EmailVerify` | 이미 `active` + 인증된 고객의 낡은 토큰은 기존대로 성공 처리하고 토큰을 삭제(멱등성) |
| `EmailVerifyResend` | initial / expired / verified / frozen / 미존재 / **agent 없음** 6개 분기가 전부 200 |
| `EmailVerifyResend` | agent 없는 expired 행에는 메일을 보내지 않음(3-4-1) |
| `EmailVerifyResend` | 동일 이메일의 대소문자 변형이 같은 카운터를 사용(customer_id 키잉) |
| `EmailVerifyResend` | 복수 행 매칭 시 `deleted` 제외 후 최신 `tm_create` 선택 |
| 쿨다운 | 60초 내 2회차는 발송 생략, 응답은 200 |
| 일일 상한 | 5회 초과 시 발송 생략. 카운터에 항상 TTL이 설정됨(영구 잠금 없음) |
| 동시성 | 2 replica 동시 요청이 쿨다운 하나를 공유 |
| 회귀 | `Signup` 정상 경로, 기존 `EmailVerify` 정상 경로 |

기존 `cleanup_test.go:96-105`는 `FieldTMDelete`가 설정되는 것을 단언하고 있으므로 함께 수정한다.

## 5. 범위 외 (후속 티켓 후보)

1. `/auth/password-forgot`도 이메일별 쿨다운이 없어 동일한 메일 폭탄 노출이 있다.
2. `CustomerHardDelete`(dbhandler/customer.go:545)는 2026-02-23 이후 호출자가 없는 dead code다.
3. expired 계정의 `/auth/unregister`가 400을 반환하는 문제(3-8). VOIP-1491에 기록한다.

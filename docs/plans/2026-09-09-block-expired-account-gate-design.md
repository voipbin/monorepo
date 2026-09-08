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
(vendor·test 제외) 등장하지 않는다. 즉 **요청 경로에서 `expired`를 해석하는 지점은 이 switch 하나뿐이다.**

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
구독(`pkg/servicehandler/websock.go`의 `RunSubscription`)은 배포 후에도 계속 스트리밍된다.
휴면 집단이라 실질 영향은 미미하나 "실질 접근은 0"이 절대적 서술은 아니다.

### 2-2. 개별 근거 반박

| 원 근거 | 검증 결과 |
|---|---|
| `/auth/login`이 JWT를 발급한다 | **철회하지 않는다. 아래 2-4 참조.** 게이트만 놓고 보면 JWT가 갈 곳이 없는 것은 맞으나, fail-open과 조합하면 증폭 경로가 된다 |
| `password-forgot` -> `password-reset`로 로그인 가능 | 메커니즘이 login과 다르다. agent의 `password_hash`는 가입 시 부여된 랜덤값이라 사용자가 모르므로(`bin-agent-manager/pkg/agenthandler/event.go:145-151`), reset은 **없던 자격증명을 새로 만들어낸다.** 다만 2-4의 조치로 로그인 자체가 막히면 비밀번호를 설정해도 쓸 곳이 없으므로 **범위 외로 유지한다** |
| direct token이 검사를 건너뛴다 | **`AuthBoot`이 이미 `StatusActive`를 요구한다**(`pkg/servicehandler/boot.go:108`). 만료 계정은 발급 자체가 불가. 토큰 수명 4시간이라 193건은 전부 소멸 |
| `/auth/unregister` 예외 | 만료 계정에서는 **이미 죽은 분기**. POST(`CustomerSelfFreeze`)는 `dbhandler/customer.go:301`의 `WHERE status='active'` CAS 때문에 0행 갱신 -> `ErrNotFound` -> 400. DELETE(`CustomerSelfRecover`)도 `dbhandler/customer.go:353`이 `status='frozen'`으로 CAS하므로 동일하게 `ErrNotFound`. `immediate: true`의 `FreezeAndDelete`도 내부에서 `Freeze()`를 먼저 호출한다(`freeze.go:80`). **세 경로 전부 막힌다.** 이 예외로 스스로 `active`로 되돌릴 수 없음을 확인했다 |

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

### 2-3. `/provisioning/extension`도 별도 조치가 불필요하다

`GET /provisioning/extension`은 인증 없이 SIP 자격증명 XML을 반환하며 고객 상태를 보지 않는다
(`pkg/servicehandler/extension.go`의 `ExtensionProvisioningXMLGet`은 `e.TMDelete`만 확인).

그러나 **토큰 공급이 `v1.0` 경로**(`POST /v1.0/extensions/:id/provisioning-token`)이고
`ProvisioningTokenTTL = 10 * time.Minute`(`pkg/servicehandler/extension.go:22`)이다.
게이트를 막으면 신규 발급이 끊기고 기존 토큰은 10분 내 소멸한다.

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

### 4-1. 변경 지점 2개

| # | 위치 | 변경 |
|---|---|---|
| a | `authenticate.go`의 `isBlockedAccountStatus` switch | `case cscustomer.StatusExpired:` 추가. 403 + `ACCOUNT_EXPIRED` |
| b | `pkg/servicehandler/auth.go`의 `AuthLogin` | 고객 상태 검사 추가. `AuthBoot`(boot.go:108) 패턴을 따른다. 2-4 참조 |
| c | `authenticate.go`의 고객 조회 실패 분기 | 로그 + Prometheus 카운터 추가. **동작 변경 없음** |

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

**단, 프론트엔드 렌더링은 이 티켓 범위가 아니다.** 2-4의 조치로 로그인 자체가 막히므로
사용자는 로그인 화면에서 403을 받게 되며, 대시보드에 들어와 모든 호출이 403나는 상황은
발생하지 않는다. 프론트가 이 `details`를 실제로 쓰게 하는 작업은 별도 티켓으로 남긴다.

### 4-3. 복구 경로 안내 시 주의: 193건은 두 갈래다

| 구분 | 건수 | 복구 경로 |
|---|---|---|
| 살아있는 agent 보유 | 92 | `POST /auth/email-verify-resend` 로 자가 복구 |
| agent 없음 | 101 | 재발송 불가(`resendTarget`이 nil 반환). **재가입만 가능** |

에러 메시지가 재발송만 안내하면 101건에게는 막다른 길이 된다.

**초안은 "게이트가 agent 유무를 모른다"고 썼는데 부정확하다.** `a.Type == TypeAgent`이면
그 identity 자체가 해당 고객의 agent이므로 추가 RPC 없이 알 수 있다. 반대로 agent가 없는 101건은
agent JWT를 가질 수 없고 accesskey만 제시할 수 있다. 즉 게이트는 이미 무료로 두 집단을 구분할 수 있다.

**그럼에도 분기하지 않기로 한다.** 이유는 불가능해서가 아니라 메시지 복잡도에 비해 이득이 작기
때문이다. 두 경로를 모두 언급하는 문구 하나로 충분하다
("인증 메일 재발송을 요청하거나, 계정이 없다면 다시 가입하십시오").

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
| **fail-open 분기** | 동작은 그대로 통과하되 카운터가 증가하는지 검증 (4-1c). 기존 `authenticate_test.go`의 "CustomerRawSelfGet error - fail open" 케이스를 확장한다 |

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
5. registrar/kamailio 경로에는 `StatusExpired` 처리가 없다. SIP 등록은 api-manager를 거치지 않는다.

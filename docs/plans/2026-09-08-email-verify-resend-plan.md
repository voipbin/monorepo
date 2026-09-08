# VOIP-1490 Email Verification Resend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 이메일 인증을 놓친 가입자가 영구히 갇히지 않도록, 인증 메일 재발송 경로(`POST /auth/email-verify-resend`)를 추가하고 만료 처리에서 `tm_delete` 설정을 제거한다.

**Architecture:** api-manager는 요청 바인딩과 항상 200 반환만 담당하는 얇은 계층으로 두고, 고객 선택 / 상태 분기 / 남용 방지 / 토큰 발급 / 발송 판단은 전부 customer-manager에 둔다. 남용 방지는 IP 기준 미들웨어로는 불가능하므로(replica별 in-process) Redis에 `customer_id` 기준 쿨다운과 일일 상한을 둔다. 기존 `sendSignupVerification`을 그대로 재사용한다.

**Tech Stack:** Go, gin(api-manager), RabbitMQ RPC(sock), MariaDB(squirrel), Redis(go-redis v8), gomock, oapi-codegen

**설계 문서:** `docs/plans/2026-09-08-email-verify-resend-design.md` — 판단 근거는 전부 여기에 있다. 구현 중 "왜 이렇게 하는가"가 막히면 설계 문서를 먼저 읽을 것.

---

## 사전 지식 (이 저장소를 모르는 사람을 위한 것)

- **워크트리에서 작업한다.** 이 계획은 `~/gitvoipbin/monorepo/.worktrees/VOIP-1490-Add-email-verify-resend-endpoint`에서 실행한다. 메인 저장소(`~/gitvoipbin/monorepo`)의 파일을 절대 수정하지 않는다.
- **검증 워크플로는 생략 불가다.** 코드를 바꾼 서비스 디렉토리에서 매번 아래 5단계를 전부 돌린다. "사소한 변경"은 생략 사유가 되지 않는다.
  ```bash
  cd bin-<service-name>
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```
  `vendor/`는 git에 커밋하지 않는다(.gitignore). `go.mod`/`go.sum` 변경분은 커밋한다.
- **mock은 `go generate ./...`가 만든다.** 인터페이스에 메서드를 추가하면 반드시 해당 서비스에서 `go generate ./...`를 돌려 `mock_*.go`를 갱신한 뒤 커밋한다.
- **커밋 메시지 형식**은 요약 한 줄 + `- 서비스명: 변경` 불릿이다. AI attribution(`Co-Authored-By`, `Generated with`)은 절대 넣지 않는다.
- **서비스 간 호출은 RabbitMQ RPC**다. `bin-common-handler/pkg/requesthandler`가 클라이언트, 각 서비스의 `pkg/listenhandler`가 서버다.

## 파일 구조

| 파일 | 책임 | 작업 |
|---|---|---|
| `bin-customer-manager/pkg/customerhandler/signup.go` | 상수, 가입, 인증, 재발송 | 수정 |
| `bin-customer-manager/pkg/customerhandler/cleanup.go` | 미인증 만료 처리 | 수정 |
| `bin-customer-manager/pkg/customerhandler/main.go` | CustomerHandler 인터페이스 | 수정 |
| `bin-customer-manager/pkg/cachehandler/handler.go` | Redis 접근 구현 | 수정 |
| `bin-customer-manager/pkg/cachehandler/main.go` | CacheHandler 인터페이스 | 수정 |
| `bin-customer-manager/pkg/listenhandler/main.go` | RPC 라우팅 | 수정 |
| `bin-customer-manager/pkg/listenhandler/v1_customers_signup.go` | signup/verify RPC 핸들러 | 수정 |
| `bin-customer-manager/pkg/listenhandler/models/request/customers.go` | RPC 요청 구조체 | 수정 |
| `bin-common-handler/pkg/requesthandler/customer_customer.go` | customer-manager RPC 클라이언트 | 수정 |
| `bin-common-handler/pkg/requesthandler/main.go` | RequestHandler 인터페이스 | 수정 |
| `bin-api-manager/pkg/servicehandler/customer.go` | 서비스 계층 | 수정 |
| `bin-api-manager/pkg/servicehandler/main.go` | ServiceHandler 인터페이스 | 수정 |
| `bin-api-manager/lib/service/signup.go` | HTTP 핸들러 + 인증 HTML | 수정 |
| `bin-api-manager/cmd/api-manager/main.go` | 라우트 등록 | 수정 |
| `bin-openapi-manager/openapi/paths/auth/email-verify-resend.yaml` | 신규 엔드포인트 스펙 | 생성 |
| `bin-openapi-manager/openapi/openapi.yaml` | path/schema 등록 | 수정 |
| `bin-api-manager/docsdev/source/auth_overview.rst` | RST 문서 | 수정 |

## 태스크 순서와 의존성

```
Task 1 (상수/문구)  ─┐
Task 2 (cleanup)    ─┼─ 서로 독립, customer-manager 안에서 완결
Task 3 (EmailVerify)─┘
Task 4 (cachehandler 프리미티브)
     └─ Task 5 (EmailVerifyResend 핸들러)
            └─ Task 6 (listenhandler RPC)
                   └─ Task 7 (common-handler RPC 클라이언트)
                          └─ Task 8 (api-manager 서비스 + 라우트)
                                 └─ Task 9 (인증 실패 화면 재발송 폼)
Task 10 (openapi + RST)  ← 마지막
```

---
### Task 1: 상수 연장과 사실과 어긋난 문구 수정

유예 창을 1시간에서 72시간으로, 인증 링크 수명을 1시간에서 24시간으로 늘린다. 같은 파일에 있는, 이 변경으로 사실과 어긋나게 되는 문구 두 개도 함께 고친다.

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go` (상수 :19-23, 주석 :114-117, 메일 본문 :253)
- Modify: `bin-customer-manager/pkg/customerhandler/cleanup.go` (상수 :12-14)

- [ ] **Step 1: 현재 상수를 확인한다**

```bash
cd bin-customer-manager
sed -n '18,24p' pkg/customerhandler/signup.go
sed -n '11,15p' pkg/customerhandler/cleanup.go
```

기대 출력에 `emailVerifyTokenTTL = time.Hour`, `unverifiedMaxAge = time.Hour`가 보여야 한다.

- [ ] **Step 2: 상수를 바꾼다**

`pkg/customerhandler/signup.go`의 상수 블록:

```go
	emailVerifyTokenTTL    = 24 * time.Hour
	emailVerifyTokenLen    = 32                   // 32 bytes = 64 hex chars
	defaultAccesskeyExpire = 365 * 24 * time.Hour // 1 year
```

`pkg/customerhandler/cleanup.go`의 상수 블록:

```go
const (
	// unverifiedMaxAge is how long an unverified signup is left usable before it
	// is moved to StatusExpired. It is deliberately longer than
	// emailVerifyTokenTTL so a user whose link expired can still ask for a new
	// one (POST /auth/email-verify-resend) before the account is expired.
	unverifiedMaxAge = 72 * time.Hour
)
```

- [ ] **Step 3: 메일 본문의 "1 hour"를 고친다**

`pkg/customerhandler/signup.go`의 `sendVerificationEmail` 안:

```go
	subject := "VoIPBin - Verify Your Email"
	content := fmt.Sprintf(
		"Welcome to VoIPBin!\n\n"+
			"Click the link below to verify your email address (expires in 24 hours):\n\n"+
			"%s\n\n"+
			"If the link has expired, you can request a new one from the verification page.\n\n"+
			"If you did not create this account, you can safely ignore this email.",
		verifyLink,
	)
```

- [ ] **Step 4: 사실과 반대인 주석을 고친다**

`Signup` 안, `sendSignupVerification` 호출 직전 주석을 아래로 교체한다. 기존 문장 "If the verification email fails, the customer can re-request signup to trigger a new email."은 구현된 적이 없으므로 삭제한다(재가입은 `validateCreate`의 중복 검사에 막힌다).

```go
	// Best-effort verification email: generate token, store in Redis, and send email.
	// Failures here are non-fatal because the customer and access key are already committed
	// and cannot be rolled back. The client needs the SignupResult to authenticate.
	// If the verification email fails or expires, the customer requests a new one via
	// POST /auth/email-verify-resend. Re-running signup does NOT work: validateCreate
	// rejects the duplicate email.
```

- [ ] **Step 5: 검증 워크플로를 돌린다**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

기대: 전부 통과. `cleanup_test.go`는 아직 옛 동작을 단언하지만 이 시점에는 동작이 그대로이므로 통과해야 정상이다.

- [ ] **Step 6: 커밋**

커밋 메시지(요약 한 줄 + 서비스 접두사 불릿):

```
Extend email verification window and fix stale copy

- bin-customer-manager: Extend emailVerifyTokenTTL from 1h to 24h
- bin-customer-manager: Extend unverifiedMaxAge from 1h to 72h
- bin-customer-manager: Correct verification email body to say 24 hours
- bin-customer-manager: Replace the signup comment that claimed re-signup resends the email
```

---

### Task 2: cleanup이 tm_delete를 설정하지 않게 한다

`tm_delete` 설정을 제거해 삭제 캐스케이드 봉인을 푼다. 동시에 필터에 `status=initial`을 추가한다. 현재 필터는 `{email_verified:false, deleted:false}`뿐이라 `tm_delete`만이 행을 결과에서 빼주는데, 그걸 없애면 만료된 행이 15분 주기 cron마다 영원히 재선택된다.

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/cleanup.go:28-51`
- Test: `bin-customer-manager/pkg/customerhandler/cleanup_test.go`

- [ ] **Step 1: 실패하는 테스트를 쓴다**

`pkg/customerhandler/cleanup_test.go`에 추가한다. 기존 테스트의 import와 mock 생성 방식을 그대로 따를 것(파일 상단 참고).

```go
func Test_CleanupUnverified_DoesNotSetTMDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &customerHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("8f2b7f2c-1c4a-4a3b-9d61-0f2b8a1c7e11")

	mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), map[customer.Field]any{
		customer.FieldEmailVerified: false,
		customer.FieldDeleted:       false,
		customer.FieldStatus:        string(customer.StatusInitial),
	}).Return([]*customer.Customer{{ID: customerID, Email: "a@test.com"}}, nil)

	// tm_delete must NOT be part of the update. Leaving it unset is what keeps the
	// customer_deleted cascade reachable and restores the documented invariant
	// "tm_delete set <=> status=deleted" from migration dafeedbccfa5.
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, map[customer.Field]any{
		customer.FieldStatus: string(customer.StatusExpired),
	}).Return(nil)

	got, err := h.CleanupUnverified(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if got != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", got)
	}
}
```

`mockUtil`이 실제로 안 쓰이면 선언에서 빼고 `customerHandler` 리터럴에서도 뺀다. Step 3에서 `TimeNow()` 호출이 사라지기 때문이다.

- [ ] **Step 2: 실패를 확인한다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run Test_CleanupUnverified_DoesNotSetTMDelete -v
```

기대: FAIL. `CustomerList`가 `status` 없는 필터로 호출되고 `CustomerUpdate`가 `tm_delete`를 포함해 호출되므로 gomock이 unexpected call로 실패한다.

- [ ] **Step 3: 구현한다**

`pkg/customerhandler/cleanup.go`의 필터:

```go
	// status=initial is required, not cosmetic: this filter is the only thing that
	// stops an already-expired row from being re-selected on every run now that
	// tm_delete is no longer set. Expressed as an allowlist rather than an
	// exclusion so a future status is not silently swept in.
	filters := map[customer.Field]any{
		customer.FieldEmailVerified: false,
		customer.FieldDeleted:       false,
		customer.FieldStatus:        string(customer.StatusInitial),
	}
```

갱신 루프:

```go
	expired := 0
	for _, c := range customers {
		log.Infof("Expiring unverified customer. customer_id: %s, email: %s", c.ID, c.Email)

		// tm_delete is deliberately NOT set. Setting it (the behavior introduced by
		// f1aeb4a59) makes Delete() and Freeze() permanently unreachable for this
		// row and blocks the customer_deleted cascade forever.
		fields := map[customer.Field]any{
			customer.FieldStatus: string(customer.StatusExpired),
		}
		if err := h.db.CustomerUpdate(ctx, c.ID, fields); err != nil {
			log.Errorf("Could not expire customer. customer_id: %s, err: %v", c.ID, err)
			continue
		}
		expired++
	}
```

`h.utilHandler.TimeNow()` 호출이 사라지므로, 그 변수(`now`)와 미사용 import를 정리한다.

- [ ] **Step 4: 기존 테스트를 새 기대값에 맞춘다**

`cleanup_test.go`에서 `FieldTMDelete`를 단언하던 기존 케이스의 기대 필드를 `{FieldStatus: expired}`만 남기도록 고치고, `CustomerList` 기대 필터에도 `FieldStatus: initial`을 추가한다.

- [ ] **Step 5: 테스트를 돌린다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run Test_CleanupUnverified -v
```

기대: PASS.

- [ ] **Step 6: 검증 워크플로**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: 커밋**

```
Stop setting tm_delete when expiring unverified customers

- bin-customer-manager: Drop tm_delete from CleanupUnverified so the customer_deleted cascade stays reachable
- bin-customer-manager: Add status=initial to the cleanup filter so expired rows are not re-selected every run
- bin-customer-manager: Update cleanup tests to assert the new field set
```

---

### Task 3: EmailVerify에 상태 거부목록과 tm_delete 복구를 넣는다

인증 성공 시 `tm_delete`를 `NULL`로 되돌린다. 이것이 기존 만료 행(프로덕션 92건)을 별도 마이그레이션 없이 복구시키는 장치다. 동시에 `deleted`/`frozen` 상태를 거부하는 가드를 넣는다.

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go` (`EmailVerify`, :128~)
- Test: `bin-customer-manager/pkg/customerhandler/signup_test.go`

- [ ] **Step 1: 실패하는 테스트 두 개를 쓴다**

```go
func Test_EmailVerify_ClearsTMDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &customerHandler{
		cache:      mockCache,
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	tmDelete := "2026-09-06 19:00:03.569129"

	// a legacy row: expired AND soft-deleted by the old cleanup behavior
	expiredCustomer := &customer.Customer{
		ID:            customerID,
		Email:         "legacy@test.com",
		EmailVerified: false,
		Status:        customer.StatusExpired,
		TMDelete:      &tmDelete,
	}
	recovered := &customer.Customer{
		ID:            customerID,
		Email:         "legacy@test.com",
		EmailVerified: true,
		Status:        customer.StatusActive,
	}

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, token).Return(customerID, nil)
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, gomock.Any()).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(expiredCustomer, nil)

	// tm_delete must be cleared, otherwise the row stays half-recovered: active but
	// still invisible to every deleted:false filter and still blocking re-signup.
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, map[customer.Field]any{
		customer.FieldEmailVerified: true,
		customer.FieldStatus:        string(customer.StatusActive),
		customer.FieldTMDelete:      nil,
	}).Return(nil)

	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, token).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(recovered, nil)
	mockReq.EXPECT().AgentV1PasswordForgot(ctx, gomock.Any(), "legacy@test.com").Return(nil)

	res, err := h.EmailVerify(ctx, token)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Customer.Status != customer.StatusActive {
		t.Errorf("Wrong match. expect: active, got: %v", res.Customer.Status)
	}
}

func Test_EmailVerify_RejectsDeletedAndFrozen(t *testing.T) {
	tests := []struct {
		name   string
		status customer.Status
	}{
		{"deleted", customer.StatusDeleted},
		{"frozen", customer.StatusFrozen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{cache: mockCache, db: mockDB}

			ctx := context.Background()
			token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

			mockCache.EXPECT().EmailVerifyTokenGet(ctx, token).Return(customerID, nil)
			mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, gomock.Any()).Return(true, nil)
			mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
				ID:     customerID,
				Status: tt.status,
			}, nil)

			// no CustomerUpdate expected: a deleted (PII-anonymized) or frozen row
			// must never be revived by a stale token.
			if _, err := h.EmailVerify(ctx, token); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
```

- [ ] **Step 2: 실패를 확인한다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run 'Test_EmailVerify_(ClearsTMDelete|RejectsDeletedAndFrozen)' -v
```

기대: FAIL. 현재 구현은 `tm_delete`를 갱신 필드에 넣지 않고 상태 가드도 없다.

- [ ] **Step 3: 구현한다**

`EmailVerify`에서 `CustomerGet` 직후, **기존 `if c.EmailVerified` 조기 return보다 앞에** 가드를 넣는다.

```go
	// Deny-list, and it must come before the EmailVerified early return.
	// Placing it after would let a stale token hand back a deleted (PII-anonymized)
	// row. It is a deny-list rather than an allow-list on purpose: an allow-list of
	// {initial, expired} would reject an already-active customer clicking an older
	// link, which the early return below is there to handle idempotently.
	if c.Status == customer.StatusDeleted || c.Status == customer.StatusFrozen {
		log.Infof("Customer is not eligible for verification. customer_id: %s, status: %s", c.ID, c.Status)
		metricshandler.EmailVerificationTotal.WithLabelValues("ineligible").Inc()
		return nil, fmt.Errorf("customer is not eligible for verification")
	}
```

갱신 필드에 `tm_delete` 초기화를 추가한다.

```go
	// mark as verified and activate.
	// tm_delete is cleared so a row expired by the pre-VOIP-1490 cleanup (which set
	// it) comes back fully, not half-recovered. processMapValues preserves nil, so
	// this emits SET tm_delete = NULL.
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
		customer.FieldStatus:        string(customer.StatusActive),
		customer.FieldTMDelete:      nil,
	}
```

- [ ] **Step 4: 테스트를 돌린다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run Test_EmailVerify -v
```

기대: PASS. 기존 `EmailVerify` 테스트가 갱신 필드를 단언하고 있으면 `FieldTMDelete: nil`을 추가해 고친다.

- [ ] **Step 5: 검증 워크플로**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 6: 커밋**

```
Clear tm_delete and guard status on email verification

- bin-customer-manager: Clear tm_delete in EmailVerify so legacy expired rows recover fully
- bin-customer-manager: Reject deleted and frozen customers before the already-verified early return
- bin-customer-manager: Add tests for legacy row recovery and the status deny-list
```

---
### Task 4: cachehandler에 쿨다운과 일일 카운터 프리미티브를 추가한다

Redis 기반 남용 방지의 저장 계층이다. 키는 이메일이 아니라 `customer_id`로 잡는다. 프로덕션 `customer_customers.email`의 collation이 `utf8mb3_uca1400_ai_ci`(대소문자 비구분)라, 이메일을 키로 쓰면 `Victim@x.com`과 `victim@x.com`이 같은 고객을 가리키면서 서로 다른 키가 되어 제한이 통째로 우회된다.

**Files:**
- Modify: `bin-customer-manager/pkg/cachehandler/main.go` (CacheHandler 인터페이스)
- Modify: `bin-customer-manager/pkg/cachehandler/handler.go` (키 접두사 상수 + 구현)

- [ ] **Step 1: 인터페이스에 메서드를 추가한다**

`pkg/cachehandler/main.go`의 `CacheHandler` 인터페이스에 추가한다. 기존 `VerifyLockAcquire`는 잠금 전용이라 재사용하지 않는다.

```go
	ResendCooldownAcquire(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (bool, error)
	ResendCountIncr(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (int64, error)
```

- [ ] **Step 2: 키 접두사 상수를 추가한다**

`pkg/cachehandler/handler.go` 상단, 기존 상수 옆:

```go
const resendCooldownKeyPrefix = "email_verify_resend_cd:"
const resendCountKeyPrefix = "email_verify_resend_n:"
```

- [ ] **Step 3: 구현한다**

`pkg/cachehandler/handler.go`에 추가한다.

```go
// ResendCooldownAcquire returns true when a verification resend may be sent for
// this customer right now, and arms the cooldown. It returns false while a
// previous send is still inside the cooldown window.
func (h *handler) ResendCooldownAcquire(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (bool, error) {
	key := resendCooldownKeyPrefix + customerID.String()

	ok, err := h.Cache.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}

	return ok, nil
}

// ResendCountIncr increments this customer's rolling resend counter and returns
// the new value.
//
// The TTL repair below is not defensive noise. INCR creates the key without an
// expiry, so if the follow-up EXPIRE fails or the process dies between the two
// commands, the counter would live forever and permanently bar that customer
// from the one recovery path this endpoint exists to provide. Re-arming a
// missing TTL on every call makes that failure self-healing.
func (h *handler) ResendCountIncr(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (int64, error) {
	key := resendCountKeyPrefix + customerID.String()

	n, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if n == 1 {
		if errExpire := h.Cache.Expire(ctx, key, ttl).Err(); errExpire != nil {
			return n, errExpire
		}
		return n, nil
	}

	remain, err := h.Cache.TTL(ctx, key).Result()
	if err != nil {
		return n, err
	}
	// go-redis returns a negative duration when the key has no expiry (-1) or is
	// already gone (-2).
	if remain < 0 {
		if errExpire := h.Cache.Expire(ctx, key, ttl).Err(); errExpire != nil {
			return n, errExpire
		}
	}

	return n, nil
}
```

- [ ] **Step 4: mock을 재생성한다**

```bash
cd bin-customer-manager
go generate ./...
git status --short pkg/cachehandler/
```

기대: `pkg/cachehandler/mock_main.go`가 수정됨.

- [ ] **Step 5: 검증 워크플로**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 6: 커밋**

```
Add resend cooldown and daily counter cache primitives

- bin-customer-manager: Add ResendCooldownAcquire keyed on customer_id
- bin-customer-manager: Add ResendCountIncr with TTL repair to avoid permanent lockout
- bin-customer-manager: Regenerate cachehandler mock
```

---

### Task 5: customerhandler에 EmailVerifyResend를 구현한다

이 태스크가 이 티켓의 핵심이다. 고객 선택, 상태 분기, agent 존재 확인, 남용 방지, 발송이 전부 여기 있다.

**중요:** 이 함수는 어떤 경우에도 "그 이메일이 존재하는가"를 호출자에게 알려주면 안 된다. 실패는 전부 조용히 `nil`을 반환한다.

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (CustomerHandler 인터페이스)
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go` (상수 + 함수)
- Test: `bin-customer-manager/pkg/customerhandler/signup_test.go`

- [ ] **Step 1: 인터페이스에 추가한다**

`pkg/customerhandler/main.go`의 `CustomerHandler` 인터페이스:

```go
	EmailVerifyResend(ctx context.Context, email string) error
```

- [ ] **Step 2: 상수를 추가한다**

`pkg/customerhandler/signup.go`의 상수 블록:

```go
	resendCooldownTTL = 60 * time.Second
	resendCountTTL    = 24 * time.Hour
	resendCountMax    = 5
)
```

- [ ] **Step 3: 실패하는 테스트를 쓴다**

핵심 분기 6개를 표로 돌린다. 어느 분기든 반환은 `nil`이어야 한다.

```go
func Test_EmailVerifyResend_SendsForExpiredCustomerWithAgent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &customerHandler{db: mockDB, cache: mockCache, reqHandler: mockReq}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	email := "legacy@test.com"

	// no deleted filter: legacy expired rows still carry tm_delete
	mockDB.EXPECT().CustomerList(ctx, uint64(100), "", map[customer.Field]any{
		customer.FieldEmail: email,
	}).Return([]*customer.Customer{
		{ID: customerID, Email: email, EmailVerified: false, Status: customer.StatusExpired},
	}, nil)

	mockReq.EXPECT().AgentV1AgentList(ctx, "", uint64(1), map[amagent.Field]any{
		amagent.FieldDeleted:  false,
		amagent.FieldUsername: email,
	}).Return([]amagent.Agent{{}}, nil)

	mockCache.EXPECT().ResendCooldownAcquire(ctx, customerID, gomock.Any()).Return(true, nil)
	mockCache.EXPECT().ResendCountIncr(ctx, customerID, gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), customerID, gomock.Any()).Return(nil)
	mockReq.EXPECT().EmailV1EmailSend(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	if err := h.EmailVerifyResend(ctx, email); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EmailVerifyResend_SkipsWithoutSending(t *testing.T) {
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	email := "x@test.com"

	tests := []struct {
		name      string
		customers []*customer.Customer
		agents    []amagent.Agent
		expectAgentLookup bool
	}{
		{
			name:      "no customer",
			customers: []*customer.Customer{},
		},
		{
			name:      "already verified",
			customers: []*customer.Customer{{ID: customerID, Email: email, EmailVerified: true, Status: customer.StatusActive}},
		},
		{
			name:      "frozen",
			customers: []*customer.Customer{{ID: customerID, Email: email, EmailVerified: false, Status: customer.StatusFrozen}},
		},
		{
			name:      "deleted",
			customers: []*customer.Customer{{ID: customerID, Email: email, EmailVerified: false, Status: customer.StatusDeleted}},
		},
		{
			name:              "no live agent",
			customers:         []*customer.Customer{{ID: customerID, Email: email, EmailVerified: false, Status: customer.StatusExpired}},
			agents:            []amagent.Agent{},
			expectAgentLookup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &customerHandler{db: mockDB, cache: mockCache, reqHandler: mockReq}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), "", gomock.Any()).Return(tt.customers, nil)
			if tt.expectAgentLookup {
				mockReq.EXPECT().AgentV1AgentList(ctx, "", uint64(1), gomock.Any()).Return(tt.agents, nil)
			}

			// no cooldown, no token, no email in any of these branches
			if err := h.EmailVerifyResend(ctx, email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EmailVerifyResend_RespectsCooldownAndCap(t *testing.T) {
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	email := "x@test.com"

	tests := []struct {
		name          string
		cooldownOK    bool
		count         int64
		expectCounter bool
	}{
		{name: "inside cooldown", cooldownOK: false, expectCounter: false},
		{name: "over daily cap", cooldownOK: true, count: 6, expectCounter: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &customerHandler{db: mockDB, cache: mockCache, reqHandler: mockReq}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), "", gomock.Any()).Return([]*customer.Customer{
				{ID: customerID, Email: email, EmailVerified: false, Status: customer.StatusInitial},
			}, nil)
			mockReq.EXPECT().AgentV1AgentList(ctx, "", uint64(1), gomock.Any()).Return([]amagent.Agent{{}}, nil)
			mockCache.EXPECT().ResendCooldownAcquire(ctx, customerID, gomock.Any()).Return(tt.cooldownOK, nil)
			if tt.expectCounter {
				mockCache.EXPECT().ResendCountIncr(ctx, customerID, gomock.Any()).Return(tt.count, nil)
			}

			// EmailVerifyTokenSet / EmailV1EmailSend must NOT be called
			if err := h.EmailVerifyResend(ctx, email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

`AgentV1AgentList`의 실제 시그니처와 반환 타입(`[]amagent.Agent` vs `[]*amagent.Agent`)은 `bin-common-handler/pkg/requesthandler/main.go`에서 확인하고 테스트를 맞춘다. `validateCreate`(customerhandler/customer.go:78-95)가 이미 같은 호출을 하므로 그 사용법을 그대로 따른다.

- [ ] **Step 4: 실패를 확인한다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run Test_EmailVerifyResend -v
```

기대: FAIL, `h.EmailVerifyResend undefined`.

- [ ] **Step 5: 구현한다**

`pkg/customerhandler/signup.go`에 추가한다.

```go
// EmailVerifyResend re-issues and re-sends the signup verification email for the
// given address.
//
// It returns nil in every case that is not an internal fault, including "no such
// customer". Reporting anything else would turn this public, unauthenticated
// endpoint into an email-existence oracle.
func (h *customerHandler) EmailVerifyResend(ctx context.Context, email string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EmailVerifyResend",
		"email": email,
	})
	log.Debug("Processing email verification resend.")

	c, err := h.resendTarget(ctx, email)
	if err != nil {
		log.Errorf("Could not resolve the resend target. err: %v", err)
		return nil
	}
	if c == nil {
		log.Debug("No eligible customer for the given email. Skipping.")
		return nil
	}

	ok, err := h.cache.ResendCooldownAcquire(ctx, c.ID, resendCooldownTTL)
	if err != nil {
		log.Errorf("Could not acquire the resend cooldown. err: %v", err)
		return nil
	}
	if !ok {
		log.Infof("Resend is still within the cooldown window. customer_id: %s", c.ID)
		return nil
	}

	n, err := h.cache.ResendCountIncr(ctx, c.ID, resendCountTTL)
	if err != nil {
		log.Errorf("Could not increase the resend counter. err: %v", err)
		return nil
	}
	if n > resendCountMax {
		log.Infof("Resend daily cap reached. customer_id: %s, count: %d", c.ID, n)
		return nil
	}

	if errSend := h.sendSignupVerification(ctx, c.ID, c.Email); errSend != nil {
		log.Errorf("Could not send the verification email. customer_id: %s, err: %v", c.ID, errSend)
		return nil
	}

	log.Infof("Resent the verification email. customer_id: %s", c.ID)
	return nil
}

// resendTarget picks the customer a resend should act on, or nil when there is
// none. Returning (nil, nil) is the normal "nothing to do" outcome; a non-nil
// error means the lookup itself failed.
func (h *customerHandler) resendTarget(ctx context.Context, email string) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "resendTarget",
		"email": email,
	})

	// No deleted filter on purpose: rows expired before VOIP-1490 still carry
	// tm_delete, and those are exactly the ones that need recovering. CustomerList
	// orders by tm_create DESC, so the newest match comes first.
	filters := map[customer.Field]any{
		customer.FieldEmail: email,
	}
	tmps, err := h.db.CustomerList(ctx, 100, "", filters)
	if err != nil {
		return nil, err
	}

	var c *customer.Customer
	for _, tmp := range tmps {
		if tmp.Status == customer.StatusDeleted {
			continue
		}
		c = tmp
		break
	}
	if c == nil {
		return nil, nil
	}

	if c.EmailVerified {
		return nil, nil
	}
	if c.Status == customer.StatusFrozen || c.Status == customer.StatusDeleted {
		return nil, nil
	}

	// Refuse when the customer has no live agent. Verification ends by calling
	// AgentV1PasswordForgot, which looks the agent up by username; without one the
	// customer would be activated with no way to set a password and no way to log
	// in. Worse, clearing tm_delete would also close the re-signup path those rows
	// still have today. See design 3-4-1.
	filterAgent := map[amagent.Field]any{
		amagent.FieldDeleted:  false,
		amagent.FieldUsername: c.Email,
	}
	agents, err := h.reqHandler.AgentV1AgentList(ctx, "", 1, filterAgent)
	if err != nil {
		// fail closed, matching validateCreate's handling of the same call
		log.Errorf("Could not get the agent info. err: %v", err)
		return nil, err
	}
	if len(agents) == 0 {
		log.Infof("Customer has no live agent. Skipping resend. customer_id: %s", c.ID)
		return nil, nil
	}

	return c, nil
}
```

- [ ] **Step 6: 테스트를 돌린다**

```bash
cd bin-customer-manager
go test ./pkg/customerhandler/ -run Test_EmailVerifyResend -v
```

기대: PASS.

- [ ] **Step 7: mock 재생성 + 검증 워크플로**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 8: 커밋**

```
Add EmailVerifyResend to customer-manager

- bin-customer-manager: Add EmailVerifyResend with enumeration-safe silent outcomes
- bin-customer-manager: Skip customers without a live agent so recovery never strands them
- bin-customer-manager: Apply a 60s cooldown and a 5-per-day cap keyed on customer_id
- bin-customer-manager: Regenerate customerhandler mock
```

---

### Task 6: listenhandler에 RPC 경로를 추가한다

**Files:**
- Modify: `bin-customer-manager/pkg/listenhandler/models/request/customers.go`
- Modify: `bin-customer-manager/pkg/listenhandler/main.go` (정규식 :64-68, 디스패치 :202-205 부근)
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_signup.go`

- [ ] **Step 1: 요청 구조체를 추가한다**

`pkg/listenhandler/models/request/customers.go`, 기존 `V1DataCustomersEmailVerifyPost` 바로 아래:

```go
// V1DataCustomersEmailVerifyResendPost is request struct for POST /v1/customers/email_verify_resend
type V1DataCustomersEmailVerifyResendPost struct {
	Email string `json:"email"`
}
```

- [ ] **Step 2: 정규식을 등록한다**

`pkg/listenhandler/main.go`의 customers 정규식 블록:

```go
	regV1CustomersSignup            = regexp.MustCompile("/v1/customers/signup$")
	regV1CustomersEmailVerify       = regexp.MustCompile("/v1/customers/email_verify$")
	regV1CustomersEmailVerifyResend = regexp.MustCompile("/v1/customers/email_verify_resend$")
```

**주의:** `regV1CustomersEmailVerify`는 `$` 앵커가 있으므로 `email_verify_resend`와 충돌하지 않는다. 앵커를 지우지 말 것.

- [ ] **Step 3: 디스패치를 추가한다**

`pkg/listenhandler/main.go`의 switch, `email_verify` case 바로 아래:

```go
	// POST /customers/email_verify_resend
	case regV1CustomersEmailVerifyResend.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersEmailVerifyResendPost(ctx, m)
		requestType = "/v1/customers/email_verify_resend"
```

- [ ] **Step 4: 핸들러를 구현한다**

`pkg/listenhandler/v1_customers_signup.go` 끝에 추가한다.

```go
// processV1CustomersEmailVerifyResendPost handles POST /v1/customers/email_verify_resend request
func (h *listenHandler) processV1CustomersEmailVerifyResendPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersEmailVerifyResendPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersEmailVerifyResendPost.")

	var reqData request.V1DataCustomersEmailVerifyResendPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// EmailVerifyResend swallows every non-fault outcome, so a 200 here says
	// "request accepted", never "that address exists".
	if err := h.customerHandler.EmailVerifyResend(ctx, reqData.Email); err != nil {
		log.Errorf("Could not resend the verification email. err: %v", err)
		return simpleResponse(500), nil
	}

	return simpleResponse(200), nil
}
```

- [ ] **Step 5: 검증 워크플로**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 6: 커밋**

```
Add email_verify_resend RPC route to customer-manager

- bin-customer-manager: Add V1DataCustomersEmailVerifyResendPost request struct
- bin-customer-manager: Route POST /v1/customers/email_verify_resend to EmailVerifyResend
```

---

### Task 7: bin-common-handler에 RPC 클라이언트를 추가한다

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (RequestHandler 인터페이스)

- [ ] **Step 1: 인터페이스에 추가한다**

`pkg/requesthandler/main.go`에서 `CustomerV1CustomerEmailVerify` 선언 옆:

```go
	CustomerV1CustomerEmailVerifyResend(ctx context.Context, email string) error
```

- [ ] **Step 2: 구현한다**

`pkg/requesthandler/customer_customer.go`, `CustomerV1CustomerEmailVerify` 바로 아래. 반환값이 없는 RPC이므로 `AgentV1PasswordForgot`(pkg/requesthandler/agent_password.go:14-36) 패턴을 따른다.

```go
// CustomerV1CustomerEmailVerifyResend asks customer-manager to re-send the signup
// verification email for the given address.
func (r *requestHandler) CustomerV1CustomerEmailVerifyResend(ctx context.Context, email string) error {
	uri := "/v1/customers/email_verify_resend"

	reqData := csrequest.V1DataCustomersEmailVerifyResendPost{
		Email: email,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/email_verify_resend", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
```

- [ ] **Step 3: mock 재생성 + 검증 워크플로**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

기대: `pkg/requesthandler/mock_main.go`에 새 메서드가 추가된다.

- [ ] **Step 4: 소비자 빌드를 확인한다**

```bash
cd bin-customer-manager && go build ./... && cd ../bin-api-manager && go build ./...
```

- [ ] **Step 5: 커밋**

```
Add customer email verify resend RPC client

- bin-common-handler: Add CustomerV1CustomerEmailVerifyResend request method
- bin-common-handler: Regenerate requesthandler mock
```

---
### Task 8: api-manager에 서비스 계층과 라우트를 추가한다

api-manager는 얇게 둔다. 상태 판단은 전혀 하지 않고, 바인딩 실패를 포함해 **무조건 200**을 반환한다.

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (ServiceHandler 인터페이스)
- Modify: `bin-api-manager/pkg/servicehandler/customer.go`
- Modify: `bin-api-manager/lib/service/signup.go`
- Modify: `bin-api-manager/cmd/api-manager/main.go` (:290-299 공개 `/auth` 그룹)
- Test: `bin-api-manager/lib/service/signup_test.go`

- [ ] **Step 1: ServiceHandler 인터페이스에 추가한다**

`pkg/servicehandler/main.go`에서 `CustomerEmailVerify` 선언 옆:

```go
	CustomerEmailVerifyResend(ctx context.Context, email string) error
```

- [ ] **Step 2: servicehandler를 구현한다**

`pkg/servicehandler/customer.go`, `CustomerEmailVerify`(:738-753) 바로 아래:

```go
// CustomerEmailVerifyResend re-sends the signup verification email.
// This is a public endpoint — no authentication required.
func (h *serviceHandler) CustomerEmailVerifyResend(ctx context.Context, email string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "CustomerEmailVerifyResend",
	})
	log.Debug("Processing customer email verification resend.")

	if err := h.reqHandler.CustomerV1CustomerEmailVerifyResend(ctx, email); err != nil {
		log.Errorf("Could not resend the customer verification email. err: %v", err)
		return err
	}

	return nil
}
```

- [ ] **Step 3: 실패하는 HTTP 핸들러 테스트를 쓴다**

`lib/service/signup_test.go`에 추가한다. 기존 signup 테스트의 gin 테스트 컨텍스트 구성 방식을 그대로 따를 것.

```go
func Test_PostCustomerEmailVerifyResend_AlwaysReturns200(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		rpcErr  error
		expectCall bool
	}{
		{name: "normal", body: `{"email":"a@test.com"}`, expectCall: true},
		{name: "downstream error", body: `{"email":"a@test.com"}`, rpcErr: fmt.Errorf("boom"), expectCall: true},
		{name: "malformed json", body: `{`, expectCall: false},
		{name: "missing email", body: `{}`, expectCall: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			if tt.expectCall {
				mockSvc.EXPECT().CustomerEmailVerifyResend(gomock.Any(), "a@test.com").Return(tt.rpcErr)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(common.OBJServiceHandler, mockSvc)
			c.Request = httptest.NewRequest("POST", "/auth/email-verify-resend", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			PostCustomerEmailVerifyResend(c)

			// Every branch must look identical from the outside. A 400 on a malformed
			// body would still be a signal an attacker can differentiate on, and the
			// downstream error must never surface either.
			if w.Code != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", w.Code)
			}
		})
	}
}
```

- [ ] **Step 4: 실패를 확인한다**

```bash
cd bin-api-manager
go test ./lib/service/ -run Test_PostCustomerEmailVerifyResend -v
```

기대: FAIL, `PostCustomerEmailVerifyResend undefined`.

- [ ] **Step 5: HTTP 핸들러를 구현한다**

`lib/service/signup.go`에 추가한다. **`c.BindJSON`이 아니라 `c.ShouldBindJSON`을 쓴다.** `BindJSON`은 실패 시 gin이 자동으로 400을 써버려 always-200 정책을 깨뜨린다.

```go
// RequestBodyEmailVerifyResendPOST is request body for POST /auth/email-verify-resend
type RequestBodyEmailVerifyResendPOST struct {
	Email string `json:"email" binding:"required"`
}

// PostCustomerEmailVerifyResend handles POST /auth/email-verify-resend request.
// It always returns 200, including on a malformed body, so that no response
// distinguishes a known address from an unknown one.
func PostCustomerEmailVerifyResend(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerEmailVerifyResend",
		"request_address": c.ClientIP(),
	})
	log.Debug("Processing email verification resend.")

	// ShouldBindJSON, not BindJSON: BindJSON writes a 400 itself on failure, which
	// would leak the difference between a malformed request and an accepted one.
	var req RequestBodyEmailVerifyResendPOST
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.JSON(200, gin.H{})
		return
	}

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := sh.CustomerEmailVerifyResend(c.Request.Context(), req.Email); err != nil {
		log.Debugf("Email verification resend failed. err: %v", err)
	}

	c.JSON(200, gin.H{})
}
```

- [ ] **Step 6: 라우트를 등록한다**

`cmd/api-manager/main.go`의 **공개** `/auth` 그룹(:290-299), `email-verify` 줄 바로 아래:

```go
	auth.POST("/email-verify-resend", service.PostCustomerEmailVerifyResend)
```

**주의:** `authProtected` 그룹(:306-312)에 넣으면 안 된다. 복구 대상 사용자는 인증을 통과하지 못할 수 있다. 공개 그룹이어야 `RateLimit("auth_public")`도 함께 적용된다.

- [ ] **Step 7: 테스트를 돌린다**

```bash
cd bin-api-manager
go test ./lib/service/ -run Test_PostCustomerEmailVerifyResend -v
```

기대: PASS.

- [ ] **Step 8: 검증 워크플로**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 9: 커밋**

```
Add POST /auth/email-verify-resend endpoint

- bin-api-manager: Add CustomerEmailVerifyResend service handler
- bin-api-manager: Add PostCustomerEmailVerifyResend that always returns 200
- bin-api-manager: Register the route on the public auth group so auth_public rate limiting applies
- bin-api-manager: Regenerate servicehandler mock
```

---

### Task 9: 인증 실패 화면에 재발송 폼을 넣는다

`GET /auth/email-verify`가 서빙하는 HTML의 실패 분기는 이 티켓의 대상 사용자가 **실제로 도달하는 유일한 화면**이다. 그런데 이 페이지는 토큰만 갖고 이메일을 모르는 반면 재발송 API는 이메일을 요구한다. 문구만 고치면 사용자는 같은 막다른 길에 남는다.

**Files:**
- Modify: `bin-api-manager/lib/service/signup.go` (`emailVerifyHTML`, :120-183)

- [ ] **Step 1: 템플릿 이스케이프 규칙을 확인한다**

```bash
cd bin-api-manager
grep -n '%%' lib/service/signup.go | head
grep -n 'fmt.Sprintf(emailVerifyHTML' lib/service/signup.go
```

`emailVerifyHTML`은 `%s` **하나**만 갖는 `fmt.Sprintf` 템플릿이고, CSS의 퍼센트는 전부 `%%`로 이스케이프되어 있다. 마크업을 추가할 때 이 규칙을 반드시 유지한다. 새로 넣는 CSS에 `%`가 있으면 `%%`로 쓴다. 새 `%s`를 추가하면 안 된다.

- [ ] **Step 2: 실패 분기에 재발송 UI를 추가한다**

실패 분기(`else` 절, 현재 :169-173)를 아래로 바꾼다. `resendBox`는 기본 숨김 상태로 body에 미리 넣어 두고 여기서 노출시킨다.

```javascript
      } else {
        msgEl.textContent = 'This verification link is no longer valid.';
        msgEl.className = 'message error';
        btn.style.display = 'none';
        document.getElementById('resend-box').style.display = 'block';
      }
```

body에 추가할 마크업(버튼 아래):

```html
<div id="resend-box" style="display:none">
  <p>Enter your email address and we will send a new verification link.</p>
  <input id="resend-email" type="email" placeholder="you@example.com" autocomplete="email">
  <button id="resend-btn" onclick="resend()">Send new link</button>
  <p id="resend-msg" class="message"></p>
</div>
```

스크립트에 추가할 함수:

```javascript
  function resend() {
    var emailEl = document.getElementById('resend-email');
    var outEl = document.getElementById('resend-msg');
    var rbtn = document.getElementById('resend-btn');
    rbtn.disabled = true;
    fetch('/auth/email-verify-resend', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: emailEl.value })
    }).then(function() {
      // Always the same copy. The API deliberately returns 200 for unknown
      // addresses, and branching the message here would undo that.
      outEl.textContent = 'If an account exists for that address, a new verification link is on its way.';
      outEl.className = 'message success';
    }).catch(function() {
      outEl.textContent = 'If an account exists for that address, a new verification link is on its way.';
      outEl.className = 'message success';
    });
  }
```

**중요:** 성공/실패 어느 쪽이든 문구가 같아야 한다. 여기서 분기하면 API가 always-200으로 막아 둔 enumeration을 화면이 되살린다.

- [ ] **Step 3: 렌더링을 눈으로 확인한다**

```bash
cd bin-api-manager
go build ./... && go vet ./lib/service/
```

`fmt.Sprintf` 인자 개수 불일치가 있으면 `go vet`이 잡는다. 잡히지 않으면 `%%` 이스케이프가 깨졌는지 다시 본다.

- [ ] **Step 4: 검증 워크플로**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 5: 커밋**

```
Offer verification resend on the failed verification page

- bin-api-manager: Add an email input and resend action to the verification failure branch
- bin-api-manager: Keep the resend result copy identical for every outcome to preserve enumeration safety
```

---

### Task 10: OpenAPI 스펙과 RST 문서를 갱신한다

`/auth/*` 여섯 개 엔드포인트는 전부 OpenAPI와 RST 양쪽에 문서화되어 있다. 신규 엔드포인트도 같은 수준으로 맞춘다.

**Files:**
- Create: `bin-openapi-manager/openapi/paths/auth/email-verify-resend.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (paths 항목 + `components/schemas` :8064 부근)
- Modify: `bin-api-manager/docsdev/source/auth_overview.rst` (표 L25-68, rate limit 설명 L72, 수명주기 다이어그램 L84-92, 엔드포인트 절 L175)

- [ ] **Step 1: path 파일을 만든다**

`bin-openapi-manager/openapi/paths/auth/email-verify-resend.yaml`:

```yaml
post:
  summary: Resend the customer email verification link.
  description: |
    Issues a new email verification token and sends it to the given address.

    This endpoint always responds `200` with an empty body, whether or not an
    account exists for the address. That is deliberate: a differentiated response
    would let an unauthenticated caller test which addresses are registered.

    Sending is rate limited per account (a short cooldown between sends and a
    daily cap). Requests that exceed either limit still return `200` without
    sending.
  tags:
    - Auth
  requestBody:
    required: true
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/RequestBodyAuthEmailVerifyResendPOST'
  responses:
    '200':
      description: |
        Request accepted. The body is always an empty object. This response does
        not indicate whether an account exists or whether an email was sent.
      content:
        application/json:
          schema:
            type: object
```

- [ ] **Step 2: openapi.yaml에 path와 schema를 등록한다**

`paths:` 아래, `/auth/email-verify` 항목 옆:

```yaml
  /auth/email-verify-resend:
    $ref: './paths/auth/email-verify-resend.yaml'
```

`components/schemas:` 아래, `RequestBodyAuthEmailVerifyPOST`(:8064) 옆:

```yaml
    RequestBodyAuthEmailVerifyResendPOST:
      type: object
      description: Request body for POST /auth/email-verify-resend (verification email resend).
      required:
        - email
      properties:
        email:
          type: string
          format: email
          description: The email address the account was registered with.
          example: simone.costa@example.com
```

기존 `/auth/email-verify` 항목의 `$ref` 표기 방식을 그대로 따를 것. `openapi.yaml`에서 실제 표기를 먼저 확인한다.

- [ ] **Step 3: 타입을 재생성한다**

```bash
cd bin-openapi-manager
go generate ./...
git status --short gens/
```

기대: `gens/models/gen.go`가 수정된다. 이 파일은 손으로 고치지 않는다.

- [ ] **Step 4: 소비자 빌드를 확인한다**

```bash
cd bin-api-manager && go build ./...
```

- [ ] **Step 5: RST 문서를 갱신한다**

`bin-api-manager/docsdev/source/auth_overview.rst`에서 네 곳을 고친다.

1. 엔드포인트 표(L25-68)에 `POST /auth/email-verify-resend` 행 추가
2. rate limit 설명(L72) — 현재 모든 `/auth/*`가 IP 기준 제한만 공유한다고 서술되어 있다. `/auth/email-verify-resend`는 그에 더해 계정별 쿨다운과 일일 상한을 갖는다는 문장을 추가한다
3. 수명주기 다이어그램(L84-92) — 현재 만료를 종착 상태로 그리고 있다. 만료에서 재발송을 거쳐 활성으로 돌아오는 경로를 추가한다
4. 엔드포인트 절(L175) — 신규 엔드포인트 설명 추가

- [ ] **Step 6: 검증 워크플로 (두 서비스 모두)**

```bash
cd bin-openapi-manager && go generate ./... && go build ./...
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**순서 주의:** codegen이 두 저장소에 걸쳐 있으므로 `bin-openapi-manager`에서 먼저 `go generate ./...`를 돌린 뒤 `bin-api-manager`에서 돌린다.

- [ ] **Step 7: 커밋**

```
Document the email verification resend endpoint

- bin-openapi-manager: Add the /auth/email-verify-resend path and request schema
- bin-openapi-manager: Regenerate model types
- bin-api-manager: Add the endpoint to the auth overview table, rate limit note, and lifecycle diagram
```

---

## 최종 확인

- [ ] **전체 테스트를 다시 돌린다**

```bash
for s in bin-customer-manager bin-common-handler bin-api-manager bin-openapi-manager; do
  echo "== $s"
  (cd "$s" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || echo "FAILED: $s"
done
```

- [ ] **`git status`가 깨끗한지 확인한다**

```bash
git status --short
```

`vendor/`는 `.gitignore`에 걸려 나오면 안 된다. 나오면 `git add -f`를 쓰지 말고 왜 추적되는지 확인한다.

- [ ] **main과의 충돌을 확인한다**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflict"
git log --oneline HEAD..origin/main
```

충돌이 있으면 rebase 후 검증 워크플로를 전부 다시 돌린다.

- [ ] **PR을 만든다**

제목은 브랜치명과 일치시킨다: `VOIP-1490-Add-email-verify-resend-endpoint`

본문 형식(마크다운 헤더 없이 서술 한 문단 + 서비스 접두사 불릿):

```
Add a recovery path for email-unverified signups so that missing the
verification window no longer leaves an account permanently unrecoverable.

- bin-customer-manager: Add POST /v1/customers/email_verify_resend and EmailVerifyResend
- bin-customer-manager: Extend the verification token TTL to 24h and the unverified window to 72h
- bin-customer-manager: Stop setting tm_delete when expiring unverified customers
- bin-customer-manager: Clear tm_delete and guard status on email verification
- bin-customer-manager: Add per-customer resend cooldown and daily cap cache primitives
- bin-common-handler: Add CustomerV1CustomerEmailVerifyResend request method
- bin-api-manager: Add POST /auth/email-verify-resend on the public auth group
- bin-api-manager: Offer resend on the failed verification page
- bin-openapi-manager: Document the new endpoint and regenerate model types
```

## 배포 후 확인 (참고)

이 계획의 범위는 아니지만, 배포 후 아래를 보면 동작을 확인할 수 있다.

- `CleanupUnverified` 로그가 같은 customer_id를 반복하지 않는지 (Task 2 회귀 확인)
- 만료 고객 수 추이 — `select status, count(*) from customer_customers group by status`
- 재발송 후 실제 복구가 되는지: 대상은 agent가 살아있는 만료 고객 92건

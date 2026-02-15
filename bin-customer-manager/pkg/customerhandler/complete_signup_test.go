package customerhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

func Test_CompleteSignup(t *testing.T) {

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000001")
	accesskeyID := uuid.FromStringOrNil("aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee")

	tests := []struct {
		name string

		tempToken string
		code      string

		responseSession *cachehandler.SignupSession
		responseCount   int64

		expectCustomerID string
		expectErr        bool
	}{
		{
			name: "normal - happy path",

			tempToken: "tmp_abcdef1234567890",
			code:      "123456",

			responseSession: &cachehandler.SignupSession{
				CustomerID:  customerID,
				OTPCode:     "123456",
				VerifyToken: "verifytoken123",
			},
			responseCount: 1,

			expectCustomerID: customerID.String(),
			expectErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &customerHandler{
				reqHandler:       mockReq,
				db:               mockDB,
				cache:            mockCache,
				notifyHandler:    mockNotify,
				accesskeyHandler: mockAccesskey,
			}
			ctx := context.Background()

			// rate limit check
			mockCache.EXPECT().SignupAttemptIncrement(ctx, tt.tempToken, gomock.Any()).Return(tt.responseCount, nil)

			// get signup session
			mockCache.EXPECT().SignupSessionGet(ctx, tt.tempToken).Return(tt.responseSession, nil)

			// verification lock
			mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
			mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)

			// double-verification guard — customer not yet verified
			mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
				ID:            customerID,
				EmailVerified: false,
			}, nil)

			// customer update
			mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)

			// create access key (now before Redis cleanup)
			mockAccesskey.EXPECT().Create(ctx, customerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(&accesskey.Accesskey{
				ID: accesskeyID,
			}, nil)

			// cleanup Redis keys (now after access key creation)
			mockCache.EXPECT().SignupSessionDelete(ctx, tt.tempToken).Return(nil)
			mockCache.EXPECT().SignupAttemptDelete(ctx, tt.tempToken).Return(nil)
			mockCache.EXPECT().EmailVerifyTokenDelete(ctx, tt.responseSession.VerifyToken).Return(nil)

			// get customer for event
			mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
				ID:            customerID,
				EmailVerified: true,
			}, nil)

			// publish event with headless=true
			mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

			res, err := h.CompleteSignup(ctx, tt.tempToken, tt.code)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Fatalf("Wrong match. expect: result, got: nil")
			}

			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong customer_id. expect: %s, got: %s", tt.expectCustomerID, res.CustomerID)
			}

			if res.Accesskey == nil {
				t.Errorf("Wrong match. expect: accesskey in result, got: nil")
			}
		})
	}
}

func Test_CompleteSignup_rateLimitExceeded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// count exceeds maxSignupAttempts (5)
	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_rate_limited", gomock.Any()).Return(int64(6), nil)

	_, err := h.CompleteSignup(ctx, "tmp_rate_limited", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "too many attempts" {
		t.Errorf("Wrong error message. expect: too many attempts, got: %v", err)
	}
}

func Test_CompleteSignup_rateLimitAtBoundary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000002")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// exactly at maxSignupAttempts (5) — should still be allowed
	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_boundary", gomock.Any()).Return(int64(5), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_boundary").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "654321",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — customer not yet verified
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)
	mockAccesskey.EXPECT().Create(ctx, customerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(&accesskey.Accesskey{}, nil)
	mockCache.EXPECT().SignupSessionDelete(ctx, "tmp_boundary").Return(nil)
	mockCache.EXPECT().SignupAttemptDelete(ctx, "tmp_boundary").Return(nil)
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "vt").Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{ID: customerID}, nil)
	mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

	res, err := h.CompleteSignup(ctx, "tmp_boundary", "654321")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
}

func Test_CompleteSignup_rateLimitIncrementError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_redis_err", gomock.Any()).Return(int64(0), fmt.Errorf("redis connection error"))

	_, err := h.CompleteSignup(ctx, "tmp_redis_err", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_CompleteSignup_invalidTempToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_invalid", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_invalid").Return(nil, fmt.Errorf("signup session not found or expired"))

	_, err := h.CompleteSignup(ctx, "tmp_invalid", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "invalid or expired temp_token" {
		t.Errorf("Wrong error message. expect: invalid or expired temp_token, got: %v", err)
	}
}

func Test_CompleteSignup_invalidOTPCode(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000003")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_wrong_otp", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_wrong_otp").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)

	_, err := h.CompleteSignup(ctx, "tmp_wrong_otp", "999999")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "invalid verification code" {
		t.Errorf("Wrong error message. expect: invalid verification code, got: %v", err)
	}
}

func Test_CompleteSignup_customerUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000004")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_update_err", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_update_err").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — customer not yet verified
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(fmt.Errorf("db error"))

	_, err := h.CompleteSignup(ctx, "tmp_update_err", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "could not verify customer" {
		t.Errorf("Wrong error message. expect: could not verify customer, got: %v", err)
	}
}

func Test_CompleteSignup_accesskeyCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000005")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_ak_err", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_ak_err").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — customer not yet verified
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)
	// AccessKey creation fails — Redis keys should NOT be cleaned up (user can retry)
	mockAccesskey.EXPECT().Create(ctx, customerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(nil, fmt.Errorf("accesskey creation failed"))

	_, err := h.CompleteSignup(ctx, "tmp_ak_err", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "could not create access key" {
		t.Errorf("Wrong error message. expect: could not create access key, got: %v", err)
	}
}

func Test_CompleteSignup_customerGetFailureNonFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000006")
	accesskeyID := uuid.FromStringOrNil("aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_get_fail", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_get_fail").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — customer not yet verified
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)
	mockAccesskey.EXPECT().Create(ctx, customerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(&accesskey.Accesskey{
		ID: accesskeyID,
	}, nil)
	mockCache.EXPECT().SignupSessionDelete(ctx, "tmp_get_fail").Return(nil)
	mockCache.EXPECT().SignupAttemptDelete(ctx, "tmp_get_fail").Return(nil)
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "vt").Return(nil)

	// CustomerGet fails — event should NOT be published, but result should still be returned
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(nil, fmt.Errorf("db get error"))

	res, err := h.CompleteSignup(ctx, "tmp_get_fail", "123456")
	if err != nil {
		t.Errorf("Wrong match. expect: ok (CustomerGet failure is non-fatal), got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
	if res.Accesskey == nil {
		t.Errorf("Wrong match. expect: accesskey in result, got: nil")
	}
	if res.CustomerID != customerID.String() {
		t.Errorf("Wrong customer_id. expect: %s, got: %s", customerID.String(), res.CustomerID)
	}
}

func Test_CompleteSignup_alreadyVerified(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000007")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_already_verified", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_already_verified").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — customer already verified
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: true,
	}, nil)
	// Redis cleanup on early return
	mockCache.EXPECT().SignupSessionDelete(ctx, "tmp_already_verified").Return(nil)
	mockCache.EXPECT().SignupAttemptDelete(ctx, "tmp_already_verified").Return(nil)
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "vt").Return(nil)

	res, err := h.CompleteSignup(ctx, "tmp_already_verified", "123456")
	if err != nil {
		t.Errorf("Wrong match. expect: ok (already verified returns success), got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
	if res.CustomerID != customerID.String() {
		t.Errorf("Wrong customer_id. expect: %s, got: %s", customerID.String(), res.CustomerID)
	}
	// already-verified path should NOT return an accesskey
	if res.Accesskey != nil {
		t.Errorf("Wrong match. expect: nil accesskey (already verified), got: %v", res.Accesskey)
	}
}

func Test_CompleteSignup_guardCustomerGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000008")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().SignupAttemptIncrement(ctx, "tmp_guard_err", gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().SignupSessionGet(ctx, "tmp_guard_err").Return(&cachehandler.SignupSession{
		CustomerID:  customerID,
		OTPCode:     "123456",
		VerifyToken: "vt",
	}, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// double-verification guard — CustomerGet fails
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(nil, fmt.Errorf("db error"))

	_, err := h.CompleteSignup(ctx, "tmp_guard_err", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "customer not found" {
		t.Errorf("Wrong error message. expect: customer not found, got: %v", err)
	}
}

func Test_EmailVerify_accesskeyCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("e1e2e3e4-0000-0000-0000-000000000001")
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_ak_fail").Return(customerID, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "token_ak_fail").Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: true,
	}, nil)
	// AccessKey creation fails — should be non-fatal
	mockAccesskey.EXPECT().Create(ctx, customerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(nil, fmt.Errorf("accesskey error"))
	mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

	res, err := h.EmailVerify(ctx, "token_ak_fail")
	if err != nil {
		t.Errorf("Wrong match. expect: ok (accesskey failure is non-fatal), got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
	if res.Customer == nil {
		t.Fatalf("Wrong match. expect: customer in result, got: nil")
	}
	if res.Accesskey != nil {
		t.Errorf("Wrong match. expect: nil accesskey (creation failed), got: %v", res.Accesskey)
	}
}

func Test_EmailVerify_customerUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("e1e2e3e4-0000-0000-0000-000000000002")
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_update_fail").Return(customerID, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(fmt.Errorf("db update error"))

	_, err := h.EmailVerify(ctx, "token_update_fail")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailVerify_customerGetAfterUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("e1e2e3e4-0000-0000-0000-000000000003")
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_get_fail").Return(customerID, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
	}, nil)
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(nil)
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "token_get_fail").Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(nil, fmt.Errorf("db get error after update"))

	_, err := h.EmailVerify(ctx, "token_get_fail")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_sendVerificationEmail_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &customerHandler{
		reqHandler: mockReq,
	}
	ctx := context.Background()

	mockReq.EXPECT().EmailV1EmailSend(
		ctx,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(nil, fmt.Errorf("email service error"))

	err := h.sendVerificationEmail(ctx, "test@voipbin.net", "testtoken", "123456")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

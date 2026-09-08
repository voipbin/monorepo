package customerhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

func Test_Signup(t *testing.T) {

	tests := []struct {
		name string

		userName      string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		responseUUID     uuid.UUID
		responseCustomer *customer.Customer
	}{
		{
			name: "normal",

			userName:      "test signup",
			detail:        "signup detail",
			email:         "signup@voipbin.net",
			phoneNumber:   "+821100000001",
			address:       "somewhere",
			webhookMethod: customer.WebhookMethodPost,
			webhookURI:    "test.com",

			responseUUID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			responseCustomer: &customer.Customer{
				ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				Name:          "test signup",
				Detail:        "signup detail",
				Email:         "signup@voipbin.net",
				PhoneNumber:   "+821100000001",
				Address:       "somewhere",
				WebhookMethod: customer.WebhookMethodPost,
				WebhookURI:    "test.com",
				EmailVerified: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &customerHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				cache:            mockCache,
				notifyHandler:    mockNotify,
				accesskeyHandler: mockAccesskey,
			}
			ctx := context.Background()

			// validateCreate expectations
			mockUtil.EXPECT().EmailIsValid(tt.email).Return(true)
			mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
			mockReq.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)

			// create customer
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().StringGenerateRandom(webhookSecretSize).Return("test-webhook-secret", nil)
			mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *customer.Customer) error {
				if c.TermsAgreedIP != "192.168.1.1" {
					t.Errorf("Expected TermsAgreedIP=192.168.1.1, got: %s", c.TermsAgreedIP)
				}
				if c.TermsAgreedVersion == "" {
					t.Errorf("Expected TermsAgreedVersion to be set, got empty")
				}
				if c.Status != customer.StatusInitial {
					t.Errorf("Expected Status=initial, got: %s", c.Status)
				}
				return nil
			})
			mockDB.EXPECT().CustomerGet(ctx, tt.responseUUID).Return(tt.responseCustomer, nil)

			// create access key + publish event (now at signup time)
			mockAccesskey.EXPECT().Create(ctx, tt.responseUUID, "default", "Auto-provisioned API key", defaultAccesskeyExpire).Return(&accesskey.Accesskey{ID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

			// token + email
			mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), tt.responseUUID, gomock.Any()).Return(nil)
			mockReq.EXPECT().EmailV1EmailSend(ctx, customer.IDSystem, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

			res, err := h.Signup(ctx, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, "192.168.1.1")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Errorf("Wrong match. expect: result, got: nil")
			}

			if res != nil && res.Customer == nil {
				t.Errorf("Wrong match. expect: customer in result, got: nil")
			}
		})
	}
}

func Test_Signup_invalidEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockUtil.EXPECT().EmailIsValid("invalid-email").Return(false)

	_, err := h.Signup(ctx, "test", "detail", "invalid-email", "", "", "", "", "192.168.1.1")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Signup_duplicateEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockUtil.EXPECT().EmailIsValid("existing@voipbin.net").Return(true)
	mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{
		{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")},
	}, nil)

	_, err := h.Signup(ctx, "test", "detail", "existing@voipbin.net", "", "", "", "", "192.168.1.1")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailVerify(t *testing.T) {

	tests := []struct {
		name  string
		token string

		responseCustomerID uuid.UUID
		responseCustomer   *customer.Customer
		responseUpdated    *customer.Customer
	}{
		{
			name:  "normal",
			token: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",

			responseCustomerID: uuid.FromStringOrNil("b1b2c3d4-0000-0000-0000-000000000001"),
			responseCustomer: &customer.Customer{
				ID:            uuid.FromStringOrNil("b1b2c3d4-0000-0000-0000-000000000001"),
				Email:         "verify@voipbin.net",
				EmailVerified: false,
			},
			responseUpdated: &customer.Customer{
				ID:            uuid.FromStringOrNil("b1b2c3d4-0000-0000-0000-000000000001"),
				Email:         "verify@voipbin.net",
				EmailVerified: true,
			},
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

			mockCache.EXPECT().EmailVerifyTokenGet(ctx, tt.token).Return(tt.responseCustomerID, nil)
			// verification lock
			mockCache.EXPECT().VerifyLockAcquire(ctx, tt.responseCustomerID, 30*time.Second).Return(true, nil)
			mockCache.EXPECT().VerifyLockRelease(ctx, tt.responseCustomerID).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseCustomerID).Return(tt.responseCustomer, nil)
			mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomerID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[customer.Field]any) error {
					if fields[customer.FieldStatus] != string(customer.StatusActive) {
						t.Errorf("Expected status=active, got: %v", fields[customer.FieldStatus])
					}
					if fields[customer.FieldEmailVerified] != true {
						t.Errorf("Expected email_verified=true, got: %v", fields[customer.FieldEmailVerified])
					}
					return nil
				},
			)
			mockCache.EXPECT().EmailVerifyTokenDelete(ctx, tt.token).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseCustomerID).Return(tt.responseUpdated, nil)
			// password reset email for browser users
			mockReq.EXPECT().AgentV1PasswordForgot(ctx, 30000, tt.responseUpdated.Email).Return(nil)

			res, err := h.EmailVerify(ctx, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Errorf("Wrong match. expect: result, got: nil")
			}

			if res != nil && res.Customer == nil {
				t.Errorf("Wrong match. expect: customer in result, got: nil")
			}
		})
	}
}

func Test_EmailVerify_invalidToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

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

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "invalidtoken").Return(uuid.Nil, fmt.Errorf("token not found or expired"))

	_, err := h.EmailVerify(ctx, "invalidtoken")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailVerify_alreadyVerified(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	customerID := uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000001")

	h := &customerHandler{
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "sometoken").Return(customerID, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: true,
	}, nil)
	// Token cleanup on already-verified path
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, "sometoken").Return(nil)

	res, err := h.EmailVerify(ctx, "sometoken")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}

	if res.Customer == nil {
		t.Fatalf("Wrong match. expect: customer in result, got: nil")
	}

	if !res.Customer.EmailVerified {
		t.Errorf("Wrong match. expect: email_verified=true, got: false")
	}
}

func Test_Signup_customerCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// validateCreate passes
	mockUtil.EXPECT().EmailIsValid("test@voipbin.net").Return(true)
	mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
	mockReq.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)

	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001"))
	mockUtil.EXPECT().StringGenerateRandom(webhookSecretSize).Return("test-webhook-secret", nil)
	mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(fmt.Errorf("db create error"))

	_, err := h.Signup(ctx, "test", "detail", "test@voipbin.net", "", "", "", "", "192.168.1.1")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Signup_accesskeyCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	responseUUID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000005")
	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// validateCreate passes
	mockUtil.EXPECT().EmailIsValid("test@voipbin.net").Return(true)
	mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
	mockReq.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)

	// create customer succeeds
	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockUtil.EXPECT().StringGenerateRandom(webhookSecretSize).Return("test-webhook-secret", nil)
	mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, responseUUID).Return(&customer.Customer{ID: responseUUID}, nil)

	// AccessKey creation fails
	mockAccesskey.EXPECT().Create(ctx, responseUUID, "default", "Auto-provisioned API key", defaultAccesskeyExpire).Return(nil, fmt.Errorf("accesskey create error"))

	_, err := h.Signup(ctx, "test", "detail", "test@voipbin.net", "", "", "", "", "192.168.1.1")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Signup_emailVerifyTokenSetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	responseUUID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000002")
	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// validateCreate passes
	mockUtil.EXPECT().EmailIsValid("test@voipbin.net").Return(true)
	mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
	mockReq.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)

	// create customer succeeds
	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockUtil.EXPECT().StringGenerateRandom(webhookSecretSize).Return("test-webhook-secret", nil)
	mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, responseUUID).Return(&customer.Customer{ID: responseUUID}, nil)

	// AccessKey + event (happen before token storage)
	mockAccesskey.EXPECT().Create(ctx, responseUUID, "default", "Auto-provisioned API key", defaultAccesskeyExpire).Return(&accesskey.Accesskey{}, nil)
	mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

	// Redis EmailVerifyTokenSet fails — non-fatal, result still returned
	mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), responseUUID, gomock.Any()).Return(fmt.Errorf("redis error"))

	res, err := h.Signup(ctx, "test", "detail", "test@voipbin.net", "", "", "", "", "192.168.1.1")
	if err != nil {
		t.Errorf("Wrong match. expect: ok (redis failure non-fatal), got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
	if res.Customer == nil {
		t.Errorf("Wrong match. expect: customer in result, got: nil")
	}
}

func Test_Signup_emailSendFailureNonFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	responseUUID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000004")
	responseCustomer := &customer.Customer{
		ID:    responseUUID,
		Email: "test@voipbin.net",
	}
	h := &customerHandler{
		utilHandler:      mockUtil,
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		notifyHandler:    mockNotify,
		accesskeyHandler: mockAccesskey,
	}
	ctx := context.Background()

	// validateCreate passes
	mockUtil.EXPECT().EmailIsValid("test@voipbin.net").Return(true)
	mockDB.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
	mockReq.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)

	// create customer succeeds
	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockUtil.EXPECT().StringGenerateRandom(webhookSecretSize).Return("test-webhook-secret", nil)
	mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, responseUUID).Return(responseCustomer, nil)

	// AccessKey + event (happen before token storage)
	mockAccesskey.EXPECT().Create(ctx, responseUUID, "default", "Auto-provisioned API key", defaultAccesskeyExpire).Return(&accesskey.Accesskey{}, nil)
	mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

	// token storage succeeds
	mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), responseUUID, gomock.Any()).Return(nil)

	// email send FAILS — should be non-fatal
	mockReq.EXPECT().EmailV1EmailSend(ctx, customer.IDSystem, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("email service down"))

	res, err := h.Signup(ctx, "test", "detail", "test@voipbin.net", "", "", "", "", "192.168.1.1")
	if err != nil {
		t.Errorf("Wrong match. expect: ok (email failure non-fatal), got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: result, got: nil")
	}
	if res.Customer == nil {
		t.Errorf("Wrong match. expect: customer in result, got: nil")
	}
}

func Test_EmailVerify_updateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000001")
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

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_update_err").Return(customerID, nil)
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: false,
		Status:        customer.StatusInitial,
	}, nil)
	// CustomerUpdate fails — status transition to active fails
	mockDB.EXPECT().CustomerUpdate(ctx, customerID, gomock.Any()).Return(fmt.Errorf("db update error"))

	_, err := h.EmailVerify(ctx, "token_update_err")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailVerify_lockNotAcquired(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000002")
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

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_lock_fail").Return(customerID, nil)
	// lock not acquired — concurrent verification in progress
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(false, nil)

	_, err := h.EmailVerify(ctx, "token_lock_fail")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailVerify_customerGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("e1e2e3e4-0000-0000-0000-000000000010")
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

	// token lookup succeeds
	mockCache.EXPECT().EmailVerifyTokenGet(ctx, "token_get_err").Return(customerID, nil)
	// verification lock
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, 30*time.Second).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	// first CustomerGet fails
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(nil, fmt.Errorf("db get error"))

	_, err := h.EmailVerify(ctx, "token_get_err")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_sendVerificationEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &customerHandler{
		reqHandler: mockReq,
	}
	ctx := context.Background()

	mockReq.EXPECT().EmailV1EmailSend(
		ctx,
		customer.IDSystem,
		uuid.Nil,
		[]commonaddress.Address{
			{
				Type:   commonaddress.TypeEmail,
				Target: "test@voipbin.net",
			},
		},
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	err := h.sendVerificationEmail(ctx, "test@voipbin.net", "testtoken123")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
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

	err := h.sendVerificationEmail(ctx, "test@voipbin.net", "testtoken")
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
	// customer.Customer.TMDelete is *time.Time, not *string
	tmDelete := time.Date(2026, 9, 6, 19, 0, 3, 0, time.UTC)

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
		name          string
		status        customer.Status
		emailVerified bool
	}{
		{"deleted", customer.StatusDeleted, false},
		{"frozen", customer.StatusFrozen, false},
		// Pins the guard ORDERING, not just its existence: with email_verified true
		// the already-verified early return would happily hand back this deleted
		// (PII-anonymized) row if the deny-list were moved below it.
		{"deleted and already verified", customer.StatusDeleted, true},
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
				ID:            customerID,
				Status:        tt.status,
				EmailVerified: tt.emailVerified,
			}, nil)

			// no CustomerUpdate expected: a deleted (PII-anonymized) or frozen row
			// must never be revived by a stale token. No EmailVerifyTokenDelete
			// either: reaching the already-verified early return is itself the
			// failure this case guards against.
			if _, err := h.EmailVerify(ctx, token); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_EmailVerify_ActiveVerifiedTokenStillSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{cache: mockCache, db: mockDB}

	ctx := context.Background()
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	customerID := uuid.FromStringOrNil("22222222-3333-4444-5555-666666666666")

	mockCache.EXPECT().EmailVerifyTokenGet(ctx, token).Return(customerID, nil)
	mockCache.EXPECT().VerifyLockAcquire(ctx, customerID, gomock.Any()).Return(true, nil)
	mockCache.EXPECT().VerifyLockRelease(ctx, customerID).Return(nil)
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		Status:        customer.StatusActive,
		EmailVerified: true,
	}, nil)
	// the stale token must still be consumed
	mockCache.EXPECT().EmailVerifyTokenDelete(ctx, token).Return(nil)

	// An allow-list of {initial, expired} would reject this and break the
	// idempotent path the design's deny-list decision exists to protect.
	if _, err := h.EmailVerify(ctx, token); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

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
		name              string
		customers         []*customer.Customer
		agents            []amagent.Agent
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
		expectSend    bool
	}{
		{name: "inside cooldown", cooldownOK: false, expectCounter: false, expectSend: false},
		// The two rows below straddle the cap boundary. resendCountMax is 5 and the
		// counter is post-increment, so the 5th send must still go out and the 6th
		// must not. Asserting only the rejecting side would let "n > resendCountMax"
		// silently become "n >= resendCountMax" (an effective cap of 4).
		{name: "at daily cap - still sends", cooldownOK: true, count: 5, expectCounter: true, expectSend: true},
		{name: "over daily cap", cooldownOK: true, count: 6, expectCounter: true, expectSend: false},
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
			// When expectSend is false, EmailVerifyTokenSet / EmailV1EmailSend are left
			// unexpected on purpose: gomock fails the test if they are called anyway.
			if tt.expectSend {
				mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), customerID, gomock.Any()).Return(nil)
				mockReq.EXPECT().EmailV1EmailSend(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			}

			if err := h.EmailVerifyResend(ctx, email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestResendConstants(t *testing.T) {
	// Pinned exactly, not only relative to each other. Both values are published as
	// fact in bin-api-manager/docs/operations.md, docsdev/source/auth_overview.rst and
	// docsdev/source/quickstart_signup.rst; every other test passes gomock.Any() for
	// the TTL, so without these assertions a change here would silently make the
	// user-facing documentation wrong.
	if resendCooldownTTL != 60*time.Second {
		t.Errorf("resendCooldownTTL = %v, expected %v", resendCooldownTTL, 60*time.Second)
	}
	if resendCountMax != 5 {
		t.Errorf("resendCountMax = %v, expected %v", resendCountMax, 5)
	}
	// The counter window is what makes the cap a *daily* cap.
	if resendCountTTL != 24*time.Hour {
		t.Errorf("resendCountTTL = %v, expected %v", resendCountTTL, 24*time.Hour)
	}
}

func Test_EmailVerifyResend_SkipsDeletedRowAndPicksOlderRecoverable(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &customerHandler{db: mockDB, cache: mockCache, reqHandler: mockReq}

	ctx := context.Background()
	newerDeletedID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	olderExpiredID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")
	email := "shared@test.com"

	// CustomerList returns tm_create DESC, so the deleted row comes first.
	mockDB.EXPECT().CustomerList(ctx, uint64(100), "", gomock.Any()).Return([]*customer.Customer{
		{ID: newerDeletedID, Email: email, EmailVerified: false, Status: customer.StatusDeleted},
		{ID: olderExpiredID, Email: email, EmailVerified: false, Status: customer.StatusExpired},
	}, nil)

	mockReq.EXPECT().AgentV1AgentList(ctx, "", uint64(1), gomock.Any()).Return([]amagent.Agent{{}}, nil)

	// the older recoverable row must be the one acted on
	mockCache.EXPECT().ResendCooldownAcquire(ctx, olderExpiredID, gomock.Any()).Return(true, nil)
	mockCache.EXPECT().ResendCountIncr(ctx, olderExpiredID, gomock.Any()).Return(int64(1), nil)
	mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), olderExpiredID, gomock.Any()).Return(nil)
	mockReq.EXPECT().EmailV1EmailSend(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	if err := h.EmailVerifyResend(ctx, email); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

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
			mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseUUID).Return(tt.responseCustomer, nil)

			// token + signup session + email
			mockCache.EXPECT().EmailVerifyTokenSet(ctx, gomock.Any(), tt.responseUUID, gomock.Any()).Return(nil)
			mockCache.EXPECT().SignupSessionSet(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().EmailV1EmailSend(ctx, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

			res, err := h.Signup(ctx, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
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

	_, err := h.Signup(ctx, "test", "detail", "invalid-email", "", "", "", "")
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

	_, err := h.Signup(ctx, "test", "detail", "existing@voipbin.net", "", "", "", "")
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
			mockDB.EXPECT().CustomerGet(ctx, tt.responseCustomerID).Return(tt.responseCustomer, nil)
			mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomerID, gomock.Any()).Return(nil)
			mockCache.EXPECT().EmailVerifyTokenDelete(ctx, tt.token).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseCustomerID).Return(tt.responseUpdated, nil)
			mockAccesskey.EXPECT().Create(ctx, tt.responseCustomerID, "default", "Auto-provisioned API key", time.Duration(0)).Return(&accesskey.Accesskey{ID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

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
	mockDB.EXPECT().CustomerGet(ctx, customerID).Return(&customer.Customer{
		ID:            customerID,
		EmailVerified: true,
	}, nil)

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

func Test_cleanupUnverified(t *testing.T) {

	tests := []struct {
		name string

		responseCustomers []*customer.Customer
	}{
		{
			name:              "empty",
			responseCustomers: []*customer.Customer{},
		},
		{
			name: "2 unverified customers",
			responseCustomers: []*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000001"),
					Email:         "expired1@voipbin.net",
					EmailVerified: false,
				},
				{
					ID:            uuid.FromStringOrNil("d1d2d3d4-0000-0000-0000-000000000002"),
					Email:         "expired2@voipbin.net",
					EmailVerified: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseCustomers, nil)

			for _, c := range tt.responseCustomers {
				mockDB.EXPECT().CustomerHardDelete(ctx, c.ID).Return(nil)
			}

			h.cleanupUnverified(ctx)
		})
	}
}

func Test_cleanupUnverified_listError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		db: mockDB,
	}
	ctx := context.Background()

	mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("db error"))

	// should not panic
	h.cleanupUnverified(ctx)
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
		uuid.Nil,
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

	err := h.sendVerificationEmail(ctx, "test@voipbin.net", "testtoken123", "123456")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

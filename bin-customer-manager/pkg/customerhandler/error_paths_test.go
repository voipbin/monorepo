package customerhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

type errPathMocks struct {
	util      *utilhandler.MockUtilHandler
	req       *requesthandler.MockRequestHandler
	db        *dbhandler.MockDBHandler
	cache     *cachehandler.MockCacheHandler
	notify    *notifyhandler.MockNotifyHandler
	accesskey *accesskeyhandler.MockAccesskeyHandler
}

func newErrPathHandler(mc *gomock.Controller) (*customerHandler, *errPathMocks) {
	m := &errPathMocks{
		util:      utilhandler.NewMockUtilHandler(mc),
		req:       requesthandler.NewMockRequestHandler(mc),
		db:        dbhandler.NewMockDBHandler(mc),
		cache:     cachehandler.NewMockCacheHandler(mc),
		notify:    notifyhandler.NewMockNotifyHandler(mc),
		accesskey: accesskeyhandler.NewMockAccesskeyHandler(mc),
	}
	h := &customerHandler{
		utilHandler:      m.util,
		reqHandler:       m.req,
		db:               m.db,
		cache:            m.cache,
		notifyHandler:    m.notify,
		accesskeyHandler: m.accesskey,
	}
	return h, m
}

var (
	errPathCustomerID = uuid.FromStringOrNil("7a1c0d3e-0000-4000-8000-000000000001")
	errPathBillingID  = uuid.FromStringOrNil("7a1c0d3e-0000-4000-8000-000000000002")
)

func Test_Get_notFoundIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrPathHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, errPathCustomerID)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: *VoipbinError, got: %v", err)
	}
	if ve.Reason != "CUSTOMER_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
		t.Errorf("Wrong match. expect: NOT_FOUND/CUSTOMER_NOT_FOUND, got: %v/%v", ve.Status, ve.Reason)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect the original error to stay in the chain, got: %v", err)
	}
}

func Test_Get_otherErrorPassesThrough(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrPathHandler(mc)
	ctx := context.Background()

	sentinel := fmt.Errorf("db unavailable")
	m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(nil, sentinel)

	_, err := h.Get(ctx, errPathCustomerID)
	if err != sentinel { //nolint:errorlint // identity check: the error must not be wrapped
		t.Errorf("Wrong match. expect: %v, got: %v", sentinel, err)
	}

	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) {
		t.Errorf("Wrong match. expect: untyped error, got: %v", ve)
	}
}

func Test_Create_errors(t *testing.T) {
	email := "customer-a@example.com"

	tests := []struct {
		name  string
		setup func(m *errPathMocks)
	}{
		{
			name: "webhook secret generation fails",
			setup: func(m *errPathMocks) {
				m.util.EXPECT().UUIDCreate().Return(errPathCustomerID)
				m.util.EXPECT().StringGenerateRandom(webhookSecretSize).Return("", fmt.Errorf("random error"))
			},
		},
		{
			name: "customer create fails",
			setup: func(m *errPathMocks) {
				m.util.EXPECT().UUIDCreate().Return(errPathCustomerID)
				m.util.EXPECT().StringGenerateRandom(webhookSecretSize).Return("secret", nil)
				m.db.EXPECT().CustomerCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert error"))
			},
		},
		{
			name: "created customer lookup fails",
			setup: func(m *errPathMocks) {
				m.util.EXPECT().UUIDCreate().Return(errPathCustomerID)
				m.util.EXPECT().StringGenerateRandom(webhookSecretSize).Return("secret", nil)
				m.db.EXPECT().CustomerCreate(gomock.Any(), gomock.Any()).Return(nil)
				m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(nil, fmt.Errorf("get error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			ctx := context.Background()

			// validateCreate passes; any later call without an expectation fails the test
			m.util.EXPECT().EmailIsValid(email).Return(true)
			m.db.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
			m.req.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)
			tt.setup(m)

			res, err := h.Create(ctx, "name", "detail", email, "", "", customer.WebhookMethodPost, "")
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	existing := &customer.Customer{ID: errPathCustomerID}

	tests := []struct {
		name  string
		setup func(m *errPathMocks)
	}{
		{
			name: "customer lookup fails",
			setup: func(m *errPathMocks) {
				m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(nil, fmt.Errorf("get error"))
			},
		},
		{
			name: "delete fails",
			setup: func(m *errPathMocks) {
				m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(existing, nil)
				m.db.EXPECT().CustomerDelete(gomock.Any(), errPathCustomerID).Return(fmt.Errorf("delete error"))
			},
		},
		{
			name: "lookup after delete fails",
			setup: func(m *errPathMocks) {
				gomock.InOrder(
					m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(existing, nil),
					m.db.EXPECT().CustomerDelete(gomock.Any(), errPathCustomerID).Return(nil),
					m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(nil, fmt.Errorf("get error")),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			tt.setup(m)

			// no PublishEvent expectation: a failure must not publish
			res, err := h.Delete(context.Background(), errPathCustomerID)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_updates_errors(t *testing.T) {
	name := "new name"

	tests := []struct {
		name       string
		call       func(h *customerHandler) (*customer.Customer, error)
		updateFail bool
	}{
		{
			name: "basic info update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateBasicInfo(context.Background(), errPathCustomerID, &name, nil, nil, nil, nil, nil, nil)
			},
			updateFail: true,
		},
		{
			name: "basic info lookup after update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateBasicInfo(context.Background(), errPathCustomerID, &name, nil, nil, nil, nil, nil, nil)
			},
		},
		{
			name: "billing account update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateBillingAccountID(context.Background(), errPathCustomerID, errPathBillingID)
			},
			updateFail: true,
		},
		{
			name: "billing account lookup after update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateBillingAccountID(context.Background(), errPathCustomerID, errPathBillingID)
			},
		},
		{
			name: "metadata update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateMetadata(context.Background(), errPathCustomerID, customer.Metadata{})
			},
			updateFail: true,
		},
		{
			name: "metadata lookup after update fails",
			call: func(h *customerHandler) (*customer.Customer, error) {
				return h.UpdateMetadata(context.Background(), errPathCustomerID, customer.Metadata{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)

			if tt.updateFail {
				// no CustomerGet expectation: a failed update must not re-read
				m.db.EXPECT().CustomerUpdate(gomock.Any(), errPathCustomerID, gomock.Any()).Return(fmt.Errorf("update error"))
			} else {
				m.db.EXPECT().CustomerUpdate(gomock.Any(), errPathCustomerID, gomock.Any()).Return(nil)
				m.db.EXPECT().CustomerGet(gomock.Any(), errPathCustomerID).Return(nil, fmt.Errorf("get error"))
			}

			// no PublishEvent expectation: a failure must not publish
			res, err := tt.call(h)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_validateCreate_dependencyErrors(t *testing.T) {
	email := "customer-b@example.com"

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *errPathMocks)
	}{
		{
			name: "customer list fails",
			setup: func(ctx context.Context, m *errPathMocks) {
				m.db.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("list error"))
			},
		},
		{
			name: "agent list fails",
			setup: func(ctx context.Context, m *errPathMocks) {
				m.db.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
				m.req.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("rpc error"))
			},
		},
		{
			name: "email already used by an agent",
			setup: func(ctx context.Context, m *errPathMocks) {
				m.db.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
				m.req.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{{}}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			ctx := context.Background()

			m.util.EXPECT().EmailIsValid(email).Return(true)
			tt.setup(ctx, m)

			// the validation result must be false, so Create must stop before allocating an id
			res, err := h.Create(ctx, "name", "detail", email, "", "", customer.WebhookMethodPost, "")
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Signup_errors(t *testing.T) {
	email := "customer-c@example.com"

	validate := func(ctx context.Context, m *errPathMocks) {
		m.util.EXPECT().EmailIsValid(email).Return(true)
		m.db.EXPECT().CustomerList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]*customer.Customer{}, nil)
		m.req.EXPECT().AgentV1AgentList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return([]amagent.Agent{}, nil)
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *errPathMocks)
	}{
		{
			name: "validation fails",
			setup: func(ctx context.Context, m *errPathMocks) {
				m.util.EXPECT().EmailIsValid(email).Return(false)
			},
		},
		{
			name: "webhook secret generation fails",
			setup: func(ctx context.Context, m *errPathMocks) {
				validate(ctx, m)
				m.util.EXPECT().UUIDCreate().Return(errPathCustomerID)
				m.util.EXPECT().StringGenerateRandom(webhookSecretSize).Return("", fmt.Errorf("random error"))
			},
		},
		{
			name: "created customer lookup fails",
			setup: func(ctx context.Context, m *errPathMocks) {
				validate(ctx, m)
				m.util.EXPECT().UUIDCreate().Return(errPathCustomerID)
				m.util.EXPECT().StringGenerateRandom(webhookSecretSize).Return("secret", nil)
				m.db.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(nil, fmt.Errorf("get error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			// no accesskey, event or cache expectations: a failure must stop before them
			res, err := h.Signup(ctx, "name", "detail", email, "", "", customer.WebhookMethodPost, "", "203.0.113.1")
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_EmailVerify_lockAcquireError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrPathHandler(mc)
	ctx := context.Background()

	m.cache.EXPECT().EmailVerifyTokenGet(ctx, "token-lock-error").Return(errPathCustomerID, nil)
	m.cache.EXPECT().VerifyLockAcquire(ctx, errPathCustomerID, 30*time.Second).Return(false, fmt.Errorf("redis error"))

	// no CustomerGet expectation: the flow must stop before reading the customer
	res, err := h.EmailVerify(ctx, "token-lock-error")
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if err == nil || err.Error() != "internal error" {
		t.Errorf("Wrong match. expect: internal error, got: %v", err)
	}
}

func Test_EmailVerify_nonFatalSideEffects(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrPathHandler(mc)
	ctx := context.Background()

	unverified := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusInitial, EmailVerified: false}
	verified := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusActive, EmailVerified: true, Email: "customer-d@example.com"}

	m.cache.EXPECT().EmailVerifyTokenGet(ctx, "token-side-effects").Return(errPathCustomerID, nil)
	m.cache.EXPECT().VerifyLockAcquire(ctx, errPathCustomerID, 30*time.Second).Return(true, nil)
	m.cache.EXPECT().VerifyLockRelease(ctx, errPathCustomerID).Return(nil)
	gomock.InOrder(
		m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(unverified, nil),
		m.db.EXPECT().CustomerUpdate(ctx, errPathCustomerID, gomock.Any()).Return(nil),
		m.cache.EXPECT().EmailVerifyTokenDelete(ctx, "token-side-effects").Return(fmt.Errorf("redis error")),
		m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(verified, nil),
		m.req.EXPECT().AgentV1PasswordForgot(ctx, 30000, verified.Email).Return(fmt.Errorf("rpc error")),
	)

	res, err := h.EmailVerify(ctx, "token-side-effects")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.Customer != verified {
		t.Errorf("Wrong match. expect: verified customer, got: %v", res)
	}
}

func Test_Freeze_concurrentRace(t *testing.T) {
	active := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusActive}
	frozen := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusFrozen}

	tests := []struct {
		name          string
		freezeErr     error
		refetch       *customer.Customer
		refetchErr    error
		expectRefetch bool
		expectRes     *customer.Customer
		expectErr     bool
	}{
		{
			name:          "already frozen by another request returns the frozen customer",
			freezeErr:     dbhandler.ErrNotFound,
			refetch:       frozen,
			expectRefetch: true,
			expectRes:     frozen,
		},
		{
			name:          "refetched customer in another state returns the original error",
			freezeErr:     dbhandler.ErrNotFound,
			refetch:       active,
			expectRefetch: true,
			expectErr:     true,
		},
		{
			name:          "refetch failure returns the original error",
			freezeErr:     dbhandler.ErrNotFound,
			refetchErr:    fmt.Errorf("get error"),
			expectRefetch: true,
			expectErr:     true,
		},
		{
			name:      "other errors do not trigger a refetch",
			freezeErr: fmt.Errorf("db error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			ctx := context.Background()

			gomock.InOrder(
				m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(active, nil),
				m.db.EXPECT().CustomerFreeze(ctx, errPathCustomerID).Return(tt.freezeErr),
			)
			if tt.expectRefetch {
				m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(tt.refetch, tt.refetchErr)
			}

			res, err := h.Freeze(ctx, errPathCustomerID)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
				}
				if !errors.Is(err, tt.freezeErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.freezeErr, err)
				}
				return
			}
			if err != nil || res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_Recover_concurrentRace(t *testing.T) {
	frozen := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusFrozen}
	active := &customer.Customer{ID: errPathCustomerID, Status: customer.StatusActive}

	tests := []struct {
		name          string
		recoverErr    error
		refetch       *customer.Customer
		refetchErr    error
		expectRefetch bool
		expectRes     *customer.Customer
		expectErr     bool
	}{
		{
			name:          "already recovered by another request returns the active customer",
			recoverErr:    dbhandler.ErrNotFound,
			refetch:       active,
			expectRefetch: true,
			expectRes:     active,
		},
		{
			name:          "refetched customer in another state returns the original error",
			recoverErr:    dbhandler.ErrNotFound,
			refetch:       frozen,
			expectRefetch: true,
			expectErr:     true,
		},
		{
			name:          "refetch failure returns the original error",
			recoverErr:    dbhandler.ErrNotFound,
			refetchErr:    fmt.Errorf("get error"),
			expectRefetch: true,
			expectErr:     true,
		},
		{
			name:       "other errors do not trigger a refetch",
			recoverErr: fmt.Errorf("db error"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrPathHandler(mc)
			ctx := context.Background()

			gomock.InOrder(
				m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(frozen, nil),
				m.db.EXPECT().CustomerRecover(ctx, errPathCustomerID).Return(tt.recoverErr),
			)
			if tt.expectRefetch {
				m.db.EXPECT().CustomerGet(ctx, errPathCustomerID).Return(tt.refetch, tt.refetchErr)
			}

			res, err := h.Recover(ctx, errPathCustomerID)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
				}
				if !errors.Is(err, tt.recoverErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.recoverErr, err)
				}
				return
			}
			if err != nil || res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expectRes, res, err)
			}
		})
	}
}

package customerdomainhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func setTestBaseDomains(t *testing.T) {
	t.Helper()

	common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("reg.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}
	t.Cleanup(common.ResetBaseDomainNamesForTest)
}

func Test_EnsureByCustomerID_existing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("11111111-7f82-11ee-8f5a-1b2c3d4e5f60")
	existing := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ab12",
		Realm:       "ab12.reg.voipbin.net",
	}

	// existing row: returned as-is, NO create
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(existing, nil)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(existing, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", existing, res)
	}
}

func Test_EnsureByCustomerID_create(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("22222222-7f82-11ee-8f5a-1b2c3d4e5f60")

	var created *customerdomain.CustomerDomain
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
		created = cd
		return nil
	})
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
		return created, nil
	})

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if res.CustomerID != customerID {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID, res.CustomerID)
	}
	if len(res.DomainLabel) != labelLength {
		t.Errorf("Wrong match. expect label length: %d, got: %d (%s)", labelLength, len(res.DomainLabel), res.DomainLabel)
	}
	for _, c := range res.DomainLabel {
		if !strings.ContainsRune(labelCharset, c) {
			t.Errorf("Wrong match. label contains a non-base36 char: %q", c)
		}
	}
	if isReservedLabel(res.DomainLabel) {
		t.Errorf("Wrong match. label is reserved: %s", res.DomainLabel)
	}
	expectRealm := res.DomainLabel + ".reg.voipbin.net"
	if res.Realm != expectRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", expectRealm, res.Realm)
	}
}

func Test_EnsureByCustomerID_labelCollisionRetry(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("33333333-7f82-11ee-8f5a-1b2c3d4e5f60")

	dupErr := fmt.Errorf("could not execute query: %w", dbhandler.ErrDuplicate)

	var created *customerdomain.CustomerDomain

	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		// first attempt: label collision -> row for the customer still missing
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(dupErr),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		// second attempt: succeeds with a NEW label
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
			created = cd
			return nil
		}),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
			return created, nil
		}),
	)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res.DomainLabel) != labelLength {
		t.Errorf("Wrong match. expect label length: %d, got: %d", labelLength, len(res.DomainLabel))
	}
}

func Test_EnsureByCustomerID_concurrentCreateRace(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("44444444-7f82-11ee-8f5a-1b2c3d4e5f60")
	winner := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "zz99",
		Realm:       "zz99.reg.voipbin.net",
	}

	dupErr := fmt.Errorf("could not execute query: %w", dbhandler.ErrDuplicate)

	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		// duplicate on customer_id: a concurrent create won -> return the winner's row
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(dupErr),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(winner, nil),
	)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(winner, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", winner, res)
	}
}

func Test_EnsureByCustomerID_attemptsExhausted(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("55555555-7f82-11ee-8f5a-1b2c3d4e5f60")

	dupErr := fmt.Errorf("could not execute query: %w", dbhandler.ErrDuplicate)

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound).Times(labelGenerateMaxAttempts + 1)
	mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(dupErr).Times(labelGenerateMaxAttempts)

	if _, err := h.EnsureByCustomerID(ctx, customerID); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_EnsureByCustomerID_createError(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("66666666-7f82-11ee-8f5a-1b2c3d4e5f60")

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(fmt.Errorf("db down"))

	if _, err := h.EnsureByCustomerID(ctx, customerID); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_EnsureByCustomerID_nilCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	if _, err := h.EnsureByCustomerID(ctx, uuid.Nil); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_GetByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("77777777-7f82-11ee-8f5a-1b2c3d4e5f60")
	expectRes := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ab12",
		Realm:       "ab12.reg.voipbin.net",
	}

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(expectRes, nil)

	res, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}

	// not found is passed through (lookup-only: never creates)
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	if _, errGet := h.GetByCustomerID(ctx, customerID); errGet == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_GetByRealm(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	expectRes := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("88888888-7f82-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "cd34",
		Realm:       "cd34.reg.voipbin.net",
	}

	mockDB.EXPECT().CustomerDomainGetByRealm(ctx, "cd34.reg.voipbin.net").Return(expectRes, nil)

	res, err := h.GetByRealm(ctx, "cd34.reg.voipbin.net")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}

	// empty realm rejected without a DB call
	if _, errGet := h.GetByRealm(ctx, ""); errGet == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}

	// miss passed through
	mockDB.EXPECT().CustomerDomainGetByRealm(ctx, "nope.reg.voipbin.net").Return(nil, dbhandler.ErrNotFound)
	if _, errGet := h.GetByRealm(ctx, "nope.reg.voipbin.net"); errGet == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_DeleteByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("99999999-7f82-11ee-8f5a-1b2c3d4e5f60")

	// normal delete
	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(nil)
	if err := h.DeleteByCustomerID(ctx, customerID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// idempotent: missing row is not an error
	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(fmt.Errorf("could not get: %w", dbhandler.ErrNotFound))
	if err := h.DeleteByCustomerID(ctx, customerID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// other errors propagate
	mockDB.EXPECT().CustomerDomainDelete(ctx, customerID).Return(fmt.Errorf("db down"))
	if err := h.DeleteByCustomerID(ctx, customerID); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_RealmGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-7f82-11ee-8f5a-1b2c3d4e5f60")
	existing := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ef56",
		Realm:       "ef56.reg.voipbin.net",
	}

	// RealmGet ensures (get-or-create); with the row present it just returns the realm
	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(existing, nil)

	res, err := h.RealmGet(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != "ef56.reg.voipbin.net" {
		t.Errorf("Wrong match. expect: ef56.reg.voipbin.net, got: %s", res)
	}
}

func Test_EndpointGet(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		extension  string

		responseDomain *customerdomain.CustomerDomain
		responseErr    error

		expectRes string
		expectErr bool
	}{
		{
			name: "domain row exists",

			customerID: uuid.FromStringOrNil("bbbbbbbb-7f82-11ee-8f5a-1b2c3d4e5f60"),
			extension:  "1001",

			responseDomain: &customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("bbbbbbbb-7f82-11ee-8f5a-1b2c3d4e5f60"),
				DomainLabel: "gh78",
				Realm:       "gh78.reg.voipbin.net",
			},

			expectRes: "1001@gh78.reg.voipbin.net",
		},
		{
			name: "no row: error (post-cutover, no computed fallback)",

			customerID: uuid.FromStringOrNil("cccccccc-7f82-11ee-8f5a-1b2c3d4e5f60"),
			extension:  "1002",

			responseErr: dbhandler.ErrNotFound,

			expectErr: true,
		},
		{
			name: "db error propagates",

			customerID: uuid.FromStringOrNil("eeeeeeee-7f82-11ee-8f5a-1b2c3d4e5f60"),
			extension:  "1004",

			responseErr: fmt.Errorf("db down"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &customerDomainHandler{dbBin: mockDB}
			ctx := context.Background()

			if tt.responseErr != nil {
				mockDB.EXPECT().CustomerDomainGet(ctx, tt.customerID).Return(nil, tt.responseErr)
			} else {
				mockDB.EXPECT().CustomerDomainGet(ctx, tt.customerID).Return(tt.responseDomain, nil)
			}

			res, err := h.EndpointGet(ctx, tt.customerID, tt.extension)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_MigrateToShortDomain(t *testing.T) {
	t.Run("backfilled long-label row is rewritten", func(t *testing.T) {
		setTestBaseDomains(t)

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &customerDomainHandler{dbBin: mockDB}
		ctx := context.Background()

		customerID := uuid.FromStringOrNil("f1f1f1f1-7f82-11ee-8f5a-1b2c3d4e5f60")
		backfilled := &customerdomain.CustomerDomain{
			CustomerID:  customerID,
			DomainLabel: customerID.String(),
			Realm:       customerID.String() + ".registrar.voipbin.net",
		}

		var updatedFields map[customerdomain.Field]any
		updated := &customerdomain.CustomerDomain{CustomerID: customerID}

		gomock.InOrder(
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilled, nil),
			mockDB.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[customerdomain.Field]any) error {
				updatedFields = fields
				updated.DomainLabel = fields[customerdomain.FieldDomainLabel].(string)
				updated.Realm = fields[customerdomain.FieldRealm].(string)
				return nil
			}),
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
				return updated, nil
			}),
		)

		res, err := h.MigrateToShortDomain(ctx, customerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		label := updatedFields[customerdomain.FieldDomainLabel].(string)
		if len(label) != labelLength {
			t.Errorf("Wrong match. expect label length: %d, got: %d", labelLength, len(label))
		}
		if res.Realm != label+".reg.voipbin.net" {
			t.Errorf("Wrong match. expect: %s.reg.voipbin.net, got: %s", label, res.Realm)
		}
	})

	t.Run("already-short row is a no-op", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &customerDomainHandler{dbBin: mockDB}
		ctx := context.Background()

		customerID := uuid.FromStringOrNil("f2f2f2f2-7f82-11ee-8f5a-1b2c3d4e5f60")
		short := &customerdomain.CustomerDomain{
			CustomerID:  customerID,
			DomainLabel: "ij90",
			Realm:       "ij90.reg.voipbin.net",
		}

		// no Update expectation: any update call fails the test
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(short, nil)

		res, err := h.MigrateToShortDomain(ctx, customerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(short, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", short, res)
		}
	})

	t.Run("missing row creates a short row directly even with the flag disabled", func(t *testing.T) {
		setTestBaseDomains(t)

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		// flag DISABLED: the batch path must still produce a SHORT label
		h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
		ctx := context.Background()

		customerID := uuid.FromStringOrNil("f3f3f3f3-7f82-11ee-8f5a-1b2c3d4e5f60")

		var created *customerdomain.CustomerDomain
		gomock.InOrder(
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound), // MigrateToShortDomain
			mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
				created = cd
				return nil
			}),
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
				return created, nil
			}),
		)

		res, err := h.MigrateToShortDomain(ctx, customerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res.DomainLabel) != labelLength {
			t.Errorf("Wrong match. expect label length: %d, got: %d", labelLength, len(res.DomainLabel))
		}
		if isReservedLabel(res.DomainLabel) {
			t.Errorf("Wrong match. label is reserved: %s", res.DomainLabel)
		}
	})

	t.Run("label collision on update retries", func(t *testing.T) {
		setTestBaseDomains(t)

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &customerDomainHandler{dbBin: mockDB}
		ctx := context.Background()

		customerID := uuid.FromStringOrNil("f4f4f4f4-7f82-11ee-8f5a-1b2c3d4e5f60")
		backfilled := &customerdomain.CustomerDomain{
			CustomerID:  customerID,
			DomainLabel: customerID.String(),
			Realm:       customerID.String() + ".registrar.voipbin.net",
		}

		dupErr := fmt.Errorf("exec failed: %w", dbhandler.ErrDuplicate)
		updated := &customerdomain.CustomerDomain{CustomerID: customerID, DomainLabel: "uv12", Realm: "uv12.reg.voipbin.net"}

		gomock.InOrder(
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilled, nil),
			mockDB.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).Return(dupErr),
			mockDB.EXPECT().CustomerDomainUpdate(ctx, customerID, gomock.Any()).Return(nil),
			mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(updated, nil),
		)

		res, err := h.MigrateToShortDomain(ctx, customerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(updated, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
		}
	})
}

func Test_UpdateByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f5f5f5f5-7f82-11ee-8f5a-1b2c3d4e5f60")
	expectRes := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       customerID.String() + ".registrar.voipbin.net",
	}

	expectFields := map[customerdomain.Field]any{
		customerdomain.FieldDomainLabel: expectRes.DomainLabel,
		customerdomain.FieldRealm:       expectRes.Realm,
	}

	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainUpdate(ctx, customerID, expectFields).Return(nil),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(expectRes, nil),
	)

	res, err := h.UpdateByCustomerID(ctx, customerID, expectRes.DomainLabel, expectRes.Realm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}

	// empty label/realm rejected
	if _, errUpdate := h.UpdateByCustomerID(ctx, customerID, "", "realm"); errUpdate == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if _, errUpdate := h.UpdateByCustomerID(ctx, customerID, "label", ""); errUpdate == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

// ============================================================================
// short-label flag (VOIP-1385 deploy-order gate)
// ============================================================================

// Test_EnsureByCustomerID_shortLabelDisabled_createsLegacyRow pins the deploy
// window behavior: with domain_short_label_enabled=false (the default), a NEW
// customer gets the LEGACY uuid label/realm — exactly the backfill shape — so
// pre-cutover frontends that build the domain from the customer uuid keep
// working.
func Test_EnsureByCustomerID_shortLabelDisabled_createsLegacyRow(t *testing.T) {
	common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}
	t.Cleanup(common.ResetBaseDomainNamesForTest)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a7a7a7a7-7f88-11ee-8f5a-1b2c3d4e5f60")

	var created *customerdomain.CustomerDomain
	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
			created = cd
			return nil
		}),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
			return created, nil
		}),
	)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// legacy shape: label = customer uuid string, realm = <uuid>.<base>
	if res.DomainLabel != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), res.DomainLabel)
	}
	expectRealm := customerID.String() + ".registrar.voipbin.net"
	if res.Realm != expectRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", expectRealm, res.Realm)
	}
}

// Test_EnsureByCustomerID_shortLabelDisabled_duplicateRace: the legacy
// label/realm are deterministic per customer, so a duplicate means a
// concurrent create won — return the winner's row, no retry loop.
func Test_EnsureByCustomerID_shortLabelDisabled_duplicateRace(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("b8b8b8b8-7f88-11ee-8f5a-1b2c3d4e5f60")
	winner := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       customerID.String() + ".reg.voipbin.net",
	}

	dupErr := fmt.Errorf("could not execute query: %w", dbhandler.ErrDuplicate)

	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).Return(dupErr),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(winner, nil),
	)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(winner, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", winner, res)
	}
}

// Test_EnsureByCustomerID_shortLabelDisabled_existingRowUntouched: an existing
// row (any shape) is returned as-is under either flag state.
func Test_EnsureByCustomerID_shortLabelDisabled_existingRowUntouched(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c9c9c9c9-7f88-11ee-8f5a-1b2c3d4e5f60")
	existing := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ab12", // already migrated to short: must NOT be rewritten
		Realm:       "ab12.reg.voipbin.net",
	}

	mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(existing, nil)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(existing, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", existing, res)
	}
}

// Test_RealmGet_shortLabelDisabled_lazyCreateLegacy: the extension-create
// lazy-create path under the pre-cutover default yields the legacy realm.
func Test_RealmGet_shortLabelDisabled_lazyCreateLegacy(t *testing.T) {
	common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}
	t.Cleanup(common.ResetBaseDomainNamesForTest)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: false}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("dadadada-7f88-11ee-8f5a-1b2c3d4e5f60")

	var created *customerdomain.CustomerDomain
	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
			created = cd
			return nil
		}),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
			return created, nil
		}),
	)

	realm, err := h.RealmGet(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if realm != customerID.String()+".registrar.voipbin.net" {
		t.Errorf("Wrong match. expect: %s.registrar.voipbin.net, got: %s", customerID.String(), realm)
	}
}

// Test_EnsureByCustomerID_shortLabelEnabled_createsShortRow: flag flipped on
// (frontend deployed): new customers get the short label/realm.
func Test_EnsureByCustomerID_shortLabelEnabled_createsShortRow(t *testing.T) {
	setTestBaseDomains(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &customerDomainHandler{dbBin: mockDB, shortLabelEnabled: true}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("ebebebeb-7f88-11ee-8f5a-1b2c3d4e5f60")

	var created *customerdomain.CustomerDomain
	gomock.InOrder(
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound),
		mockDB.EXPECT().CustomerDomainCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, cd *customerdomain.CustomerDomain) error {
			created = cd
			return nil
		}),
		mockDB.EXPECT().CustomerDomainGet(ctx, customerID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*customerdomain.CustomerDomain, error) {
			return created, nil
		}),
	)

	res, err := h.EnsureByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res.DomainLabel) != labelLength {
		t.Errorf("Wrong match. expect label length: %d, got: %d (%s)", labelLength, len(res.DomainLabel), res.DomainLabel)
	}
	if res.Realm != res.DomainLabel+".reg.voipbin.net" {
		t.Errorf("Wrong match. expect: %s.reg.voipbin.net, got: %s", res.DomainLabel, res.Realm)
	}
}

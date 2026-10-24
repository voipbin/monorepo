package trunkhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedErrCase = errors.New("injected failure")

type trunkErrFixture struct {
	h *trunkHandler

	util   *utilhandler.MockUtilHandler
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
}

func newTrunkErrFixture(mc *gomock.Controller) *trunkErrFixture {
	f := &trunkErrFixture{
		util:   utilhandler.NewMockUtilHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}
	f.h = &trunkHandler{
		utilHandler:   f.util,
		reqHandler:    f.req,
		db:            f.db,
		notifyHandler: f.notify,
	}

	return f
}

func errCaseTrunk() *trunk.Trunk {
	return &trunk.Trunk{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001"),
		},
		Name:       "case name",
		DomainName: "case-trunk",
		AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
		Realm:      "case-trunk.trunk.example.invalid",
		Username:   "caseuser",
		Password:   "case-password",
	}
}

// Test_Create_validation_errors covers the checks that run before any trunk is stored.
func Test_Create_validation_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")

	t.Run("billing lookup fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		f := newTrunkErrFixture(mc)
		ctx := context.Background()

		f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeTrunk).Return(false, errInjectedErrCase)

		res, err := f.h.Create(ctx, customerID, "n", "d", "case-trunk", nil, "u", "p", nil)
		if !errors.Is(err, errInjectedErrCase) {
			t.Errorf("Wrong match. expect: injected error, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})

	t.Run("resource limit exceeded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		f := newTrunkErrFixture(mc)
		ctx := context.Background()

		// no database access is allowed once the limit is exceeded.
		f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeTrunk).Return(false, nil)

		res, err := f.h.Create(ctx, customerID, "n", "d", "case-trunk", nil, "u", "p", nil)
		if err == nil || !strings.Contains(err.Error(), "resource limit exceeded") {
			t.Errorf("Wrong match. expect: resource limit exceeded, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})
}

// Test_Create_domainNameLength_errors checks the boundaries of the accepted domain name
// length (2 to 31 characters, starting with a letter).
func Test_Create_domainNameLength_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")

	tests := []struct {
		name       string
		domainName string

		expectLookup bool
	}{
		{name: "one character is rejected", domainName: "a", expectLookup: false},
		{name: "two characters is the lower bound", domainName: "ab", expectLookup: true},
		{name: "thirty one characters is the upper bound", domainName: "a" + strings.Repeat("b", 30), expectLookup: true},
		{name: "thirty two characters is rejected", domainName: "a" + strings.Repeat("b", 31), expectLookup: false},
		{name: "starting with a digit is rejected", domainName: "1abc", expectLookup: false},
		{name: "underscore is rejected", domainName: "ab_cd", expectLookup: false},
		{name: "empty is rejected", domainName: "", expectLookup: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newTrunkErrFixture(mc)
			ctx := context.Background()

			f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeTrunk).Return(true, nil)
			if tt.expectLookup {
				// a lookup that finds a trunk stops the creation, so nothing is written.
				existing := errCaseTrunk()
				f.db.EXPECT().TrunkGetByDomainName(ctx, tt.domainName).Return(existing, nil)
			}

			res, err := f.h.Create(ctx, customerID, "n", "d", tt.domainName, nil, "u", "p", nil)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}

			wantMessage := "invalid domain name"
			if tt.expectLookup {
				wantMessage = "already exists"
			}
			if err != nil && !strings.Contains(err.Error(), wantMessage) {
				t.Errorf("Wrong match. expect: %q, got: %v", wantMessage, err)
			}
		})
	}
}

func Test_Create_failureAfterLookup_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")
	trunkID := uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001")
	domainName := "case-trunk"

	steps := []string{"trunk_create", "trunk_get", "sipauth_create"}

	for failAt, failStep := range steps {
		t.Run(failStep, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newTrunkErrFixture(mc)
			ctx := context.Background()

			defer common.ResetBaseDomainNamesForTest()
			if errSet := common.SetBaseDomainNames("registrar.example.invalid", "trunk.example.invalid"); errSet != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
			}

			f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeTrunk).Return(true, nil)
			f.db.EXPECT().TrunkGetByDomainName(ctx, domainName).Return(nil, dbhandler.ErrNotFound)
			f.util.EXPECT().UUIDCreate().Return(trunkID)

			for idx := 0; idx <= failAt; idx++ {
				var stepErr error
				if idx == failAt {
					stepErr = errInjectedErrCase
				}

				switch steps[idx] {
				case "trunk_create":
					f.db.EXPECT().TrunkCreate(ctx, gomock.Any()).Return(stepErr)
				case "trunk_get":
					if stepErr != nil {
						f.db.EXPECT().TrunkGet(ctx, trunkID).Return(nil, stepErr)
					} else {
						f.db.EXPECT().TrunkGet(ctx, trunkID).Return(errCaseTrunk(), nil)
					}
				case "sipauth_create":
					f.db.EXPECT().SIPAuthCreate(ctx, gomock.Any()).Return(stepErr)
				}
			}

			res, err := f.h.Create(ctx, customerID, "n", "d", domainName, []sipauth.AuthType{sipauth.AuthTypeBasic}, "u", "p", nil)
			if !errors.Is(err, errInjectedErrCase) {
				t.Errorf("Wrong match. expect: injected error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_Update_failureAtEachStep_errors(t *testing.T) {
	id := uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001")
	fields := map[trunk.Field]any{trunk.FieldName: "n"}

	steps := []string{"update", "get", "sipauth_update"}

	for failAt, failStep := range steps {
		t.Run(failStep, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newTrunkErrFixture(mc)
			ctx := context.Background()

			for idx := 0; idx <= failAt; idx++ {
				var stepErr error
				if idx == failAt {
					stepErr = errInjectedErrCase
				}

				switch steps[idx] {
				case "update":
					f.db.EXPECT().TrunkUpdate(ctx, id, fields).Return(stepErr)
				case "get":
					if stepErr != nil {
						f.db.EXPECT().TrunkGet(ctx, id).Return(nil, stepErr)
					} else {
						f.db.EXPECT().TrunkGet(ctx, id).Return(errCaseTrunk(), nil)
					}
				case "sipauth_update":
					f.db.EXPECT().SIPAuthUpdate(ctx, id, gomock.Any()).Return(stepErr)
				}
			}

			res, err := f.h.Update(ctx, id, fields)
			if !errors.Is(err, errInjectedErrCase) {
				t.Errorf("Wrong match. expect: injected error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_Delete_failureAtEachStep_errors(t *testing.T) {
	id := uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001")

	steps := []string{"delete", "get", "sipauth_delete"}

	for failAt, failStep := range steps {
		t.Run(failStep, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newTrunkErrFixture(mc)
			ctx := context.Background()

			for idx := 0; idx <= failAt; idx++ {
				var stepErr error
				if idx == failAt {
					stepErr = errInjectedErrCase
				}

				switch steps[idx] {
				case "delete":
					f.db.EXPECT().TrunkDelete(ctx, id).Return(stepErr)
				case "get":
					if stepErr != nil {
						f.db.EXPECT().TrunkGet(ctx, id).Return(nil, stepErr)
					} else {
						f.db.EXPECT().TrunkGet(ctx, id).Return(errCaseTrunk(), nil)
					}
				case "sipauth_delete":
					f.db.EXPECT().SIPAuthDelete(ctx, id).Return(stepErr)
				}
			}

			res, err := f.h.Delete(ctx, id)
			if !errors.Is(err, errInjectedErrCase) {
				t.Errorf("Wrong match. expect: injected error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_List_dbError_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newTrunkErrFixture(mc)
	ctx := context.Background()

	filters := map[trunk.Field]any{trunk.FieldDeleted: false}
	f.db.EXPECT().TrunkList(ctx, uint64(10), "token", filters).Return(nil, errInjectedErrCase)

	res, err := f.h.List(ctx, "token", 10, filters)
	if !errors.Is(err, errInjectedErrCase) {
		t.Errorf("Wrong match. expect: injected error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_CountByCustomerID_dbError_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newTrunkErrFixture(mc)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")

	f.db.EXPECT().TrunkCountByCustomerID(ctx, customerID).Return(7, errInjectedErrCase)

	count, err := f.h.CountByCustomerID(ctx, customerID)
	if !errors.Is(err, errInjectedErrCase) {
		t.Errorf("Wrong match. expect: injected error, got: %v", err)
	}
	if count != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", count)
	}
}

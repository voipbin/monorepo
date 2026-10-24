package extensionhandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedErrCase = errors.New("injected failure")

// extErrFixture bundles the handler under test with its mocks.
type extErrFixture struct {
	h *extensionHandler

	util         *utilhandler.MockUtilHandler
	req          *requesthandler.MockRequestHandler
	dbAst        *dbhandler.MockDBHandler
	dbBin        *dbhandler.MockDBHandler
	notify       *notifyhandler.MockNotifyHandler
	customerDmns *customerdomainhandler.MockCustomerDomainHandler
}

// chainCallsErrCase requires the given expected calls to happen in the given order.
func chainCallsErrCase(calls []*gomock.Call) {
	for i := 1; i < len(calls); i++ {
		calls[i].After(calls[i-1])
	}
}

func newExtErrFixture(mc *gomock.Controller) *extErrFixture {
	f := &extErrFixture{
		util:         utilhandler.NewMockUtilHandler(mc),
		req:          requesthandler.NewMockRequestHandler(mc),
		dbAst:        dbhandler.NewMockDBHandler(mc),
		dbBin:        dbhandler.NewMockDBHandler(mc),
		notify:       notifyhandler.NewMockNotifyHandler(mc),
		customerDmns: customerdomainhandler.NewMockCustomerDomainHandler(mc),
	}
	f.h = &extensionHandler{
		utilHandler:           f.util,
		reqHandler:            f.req,
		dbAst:                 f.dbAst,
		dbBin:                 f.dbBin,
		notifyHandler:         f.notify,
		customerDomainHandler: f.customerDmns,
	}

	return f
}

func errCaseExtension() *extension.Extension {
	return &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001"),
		},
		Name:       "case name",
		AuthID:     "1001@case.example.invalid",
		EndpointID: "1001@case.example.invalid",
		AORID:      "1001@case.example.invalid",
		Extension:  "1001",
		Realm:      "case.example.invalid",
		Username:   "1001",
		Password:   "case-password",
		DirectID:   uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000001"),
		DirectHash: "casehash0001",
	}
}

// Test_Create_failureAtEachStep_errors injects a failure at each step of Create
// and checks that the error is returned, no result is returned, the later steps
// are not reached and no event is published.
func Test_Create_failureAtEachStep_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")
	extID := uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")
	directID := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000001")
	ext := "1001"
	realm := "case.example.invalid"
	aorID := "1001@case.example.invalid"

	// steps run in this order. A failure at "extension_get" or "sipauth_create" has no
	// direct-manager cleanup call expected, only "extension_create" has one.
	steps := []string{
		"billing", "realm", "endpoint", "aor", "auth", "endpoint_create",
		"direct_create", "extension_create", "extension_get", "sipauth_create",
	}

	for failAt, failStep := range steps {
		t.Run(failStep, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newExtErrFixture(mc)
			ctx := context.Background()

			created := errCaseExtension()
			direct := &dmdirect.Direct{
				Identity: commonidentity.Identity{ID: directID, CustomerID: customerID},
				Hash:     "casehash0001",
			}

			// each step registers its expectation: the failing step returns the injected
			// error and every step after it gets no expectation at all.
			result := func(idx int) error {
				if idx == failAt {
					return errInjectedErrCase
				}
				return nil
			}
			for idx := 0; idx <= failAt; idx++ {
				switch steps[idx] {
				case "billing":
					f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeExtension).Return(result(idx) == nil, result(idx))
				case "realm":
					f.customerDmns.EXPECT().RealmGet(ctx, customerID).Return(realm, result(idx))
				case "endpoint":
					f.customerDmns.EXPECT().EndpointGet(ctx, customerID, ext).Return(aorID, result(idx))
				case "aor":
					f.dbAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(result(idx))
				case "auth":
					f.dbAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(result(idx))
				case "endpoint_create":
					f.dbAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(result(idx))
				case "direct_create":
					f.util.EXPECT().UUIDCreate().Return(extID)
					if result(idx) != nil {
						f.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, extID).Return(nil, result(idx))
					} else {
						f.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, extID).Return(direct, nil)
					}
				case "extension_create":
					f.dbBin.EXPECT().ExtensionCreate(ctx, gomock.Any()).Return(result(idx))
					if result(idx) != nil {
						f.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(direct, nil)
					}
				case "extension_get":
					if result(idx) != nil {
						f.dbBin.EXPECT().ExtensionGet(ctx, extID).Return(nil, result(idx))
					} else {
						f.dbBin.EXPECT().ExtensionGet(ctx, extID).Return(created, nil)
					}
				case "sipauth_create":
					f.dbBin.EXPECT().SIPAuthCreate(ctx, gomock.Any()).Return(result(idx))
				}
			}

			res, err := f.h.Create(ctx, customerID, "case name", "case detail", ext, "case-password")
			if !errors.Is(err, errInjectedErrCase) {
				t.Errorf("Wrong match. expect: injected error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_Create_resourceLimitExceeded_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newExtErrFixture(mc)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")

	// no realm lookup or asterisk write may happen once the limit is exceeded.
	f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeExtension).Return(false, nil)

	res, err := f.h.Create(ctx, customerID, "n", "d", "1001", "case-password")
	if err == nil || !strings.Contains(err.Error(), "resource limit exceeded") {
		t.Errorf("Wrong match. expect: resource limit exceeded, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

// Test_Create_directCleanup_errors checks what happens to the direct hash that was
// created before the extension row failed to be stored.
func Test_Create_directCleanup_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")
	extID := uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")
	directID := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000001")
	errCleanup := errors.New("cleanup failure")

	tests := []struct {
		name       string
		errCleanup error
	}{
		{name: "cleanup succeeds", errCleanup: nil},
		{name: "cleanup fails and the original error is still returned", errCleanup: errCleanup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newExtErrFixture(mc)
			ctx := context.Background()

			direct := &dmdirect.Direct{Identity: commonidentity.Identity{ID: directID}}

			f.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeExtension).Return(true, nil)
			f.customerDmns.EXPECT().RealmGet(ctx, customerID).Return("case.example.invalid", nil)
			f.customerDmns.EXPECT().EndpointGet(ctx, customerID, "1001").Return("1001@case.example.invalid", nil)
			f.dbAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
			f.dbAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
			f.dbAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
			f.util.EXPECT().UUIDCreate().Return(extID)
			f.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, extID).Return(direct, nil)
			f.dbBin.EXPECT().ExtensionCreate(ctx, gomock.Any()).Return(errInjectedErrCase)
			f.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, tt.errCleanup)

			res, err := f.h.Create(ctx, customerID, "n", "d", "1001", "case-password")
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
	id := uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")
	current := errCaseExtension()
	password := "new-case-password"
	newPassword := password
	newAuth := &astauth.AstAuth{ID: &current.AuthID, Password: &newPassword}

	tests := []struct {
		name   string
		fields map[extension.Field]any
		// failStep: get, auth_update, update, get_updated, sipauth_update
		failStep string
	}{
		{name: "first get fails", fields: map[extension.Field]any{extension.FieldName: "n"}, failStep: "get"},
		{name: "asterisk auth update fails", fields: map[extension.Field]any{extension.FieldPassword: password}, failStep: "auth_update"},
		{name: "extension update fails and no asterisk auth update is made without a password", fields: map[extension.Field]any{extension.FieldName: "n"}, failStep: "update"},
		{name: "extension update fails after the asterisk auth was updated", fields: map[extension.Field]any{extension.FieldPassword: password}, failStep: "update"},
		{name: "get after update fails", fields: map[extension.Field]any{extension.FieldName: "n"}, failStep: "get_updated"},
		{name: "sipauth update fails", fields: map[extension.Field]any{extension.FieldName: "n"}, failStep: "sipauth_update"},
	}

	order := []string{"get", "auth_update", "update", "get_updated", "sipauth_update"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newExtErrFixture(mc)
			ctx := context.Background()

			_, withPassword := tt.fields[extension.FieldPassword]

			for _, step := range order {
				failing := step == tt.failStep
				var stepErr error
				if failing {
					stepErr = errInjectedErrCase
				}

				switch step {
				case "get":
					if failing {
						f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr)
					} else {
						f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(current, nil)
					}
				case "auth_update":
					if withPassword {
						f.dbAst.EXPECT().AstAuthUpdate(ctx, newAuth).Return(stepErr)
					}
				case "update":
					f.dbBin.EXPECT().ExtensionUpdate(ctx, id, tt.fields).Return(stepErr)
				case "get_updated":
					if failing {
						f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr)
					} else {
						f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(current, nil)
					}
				case "sipauth_update":
					f.dbBin.EXPECT().SIPAuthUpdate(ctx, gomock.Any(), gomock.Any()).Return(stepErr)
				}

				if failing {
					break
				}
			}

			res, err := f.h.Update(ctx, id, tt.fields)
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
	id := uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")

	tests := []struct {
		name     string
		failStep string
	}{
		{name: "get fails", failStep: "get"},
		{name: "extension delete fails and no asterisk row is deleted", failStep: "extension_delete"},
		{name: "endpoint delete fails", failStep: "endpoint_delete"},
		{name: "auth delete fails", failStep: "auth_delete"},
		{name: "aor delete fails", failStep: "aor_delete"},
		{name: "get after delete fails", failStep: "get_deleted"},
		{name: "sipauth delete fails", failStep: "sipauth_delete"},
	}

	order := []string{"get", "extension_delete", "endpoint_delete", "auth_delete", "aor_delete", "get_deleted", "sipauth_delete"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newExtErrFixture(mc)
			ctx := context.Background()
			ext := errCaseExtension()

			// the direct hash is deleted best-effort before the extension row.
			var calls []*gomock.Call
			for _, step := range order {
				failing := step == tt.failStep
				var stepErr error
				if failing {
					stepErr = errInjectedErrCase
				}

				switch step {
				case "get":
					if failing {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr))
					} else {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(ext, nil))
						calls = append(calls, f.req.EXPECT().DirectV1DirectDelete(ctx, ext.DirectID).Return(nil, nil))
					}
				case "extension_delete":
					calls = append(calls, f.dbBin.EXPECT().ExtensionDelete(ctx, ext.ID).Return(stepErr))
				case "endpoint_delete":
					calls = append(calls, f.dbAst.EXPECT().AstEndpointDelete(ctx, ext.EndpointID).Return(stepErr))
				case "auth_delete":
					calls = append(calls, f.dbAst.EXPECT().AstAuthDelete(ctx, ext.AuthID).Return(stepErr))
				case "aor_delete":
					calls = append(calls, f.dbAst.EXPECT().AstAORDelete(ctx, ext.AORID).Return(stepErr))
				case "get_deleted":
					if failing {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr))
					} else {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(ext, nil))
					}
				case "sipauth_delete":
					calls = append(calls, f.dbBin.EXPECT().SIPAuthDelete(ctx, ext.ID).Return(stepErr))
				}

				if failing {
					break
				}
			}
			chainCallsErrCase(calls)

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

// Test_Delete_withoutDirect_skipsDirectDelete checks that no direct-manager request is
// made for an extension that has no direct hash.
func Test_Delete_withoutDirect_skipsDirectDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newExtErrFixture(mc)
	ctx := context.Background()

	ext := errCaseExtension()
	ext.DirectID = uuid.Nil

	// DirectV1DirectDelete has no expectation: any call fails the test.
	gomock.InOrder(
		f.dbBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(ext, nil),
		f.dbBin.EXPECT().ExtensionDelete(ctx, ext.ID).Return(nil),
		f.dbAst.EXPECT().AstEndpointDelete(ctx, ext.EndpointID).Return(nil),
		f.dbAst.EXPECT().AstAuthDelete(ctx, ext.AuthID).Return(nil),
		f.dbAst.EXPECT().AstAORDelete(ctx, ext.AORID).Return(nil),
		f.dbBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(ext, nil),
		f.dbBin.EXPECT().SIPAuthDelete(ctx, ext.ID).Return(nil),
		f.notify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionDeleted, ext),
	)

	res, err := f.h.Delete(ctx, ext.ID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, ext) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", ext, res)
	}
}

func Test_DirectHashRegenerate_failureAtEachStep_errors(t *testing.T) {
	id := uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")
	directID := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000001")
	direct := &dmdirect.Direct{
		Identity: commonidentity.Identity{ID: directID, CustomerID: customerID},
		Hash:     "casehash0002",
	}
	wantFields := map[extension.Field]any{
		extension.FieldDirectID:   directID,
		extension.FieldDirectHash: "casehash0002",
	}

	tests := []struct {
		name     string
		directID uuid.UUID
		failStep string
	}{
		{name: "get fails", directID: directID, failStep: "get"},
		{name: "regenerate fails", directID: directID, failStep: "direct"},
		{name: "create fails for an extension without a direct hash", directID: uuid.Nil, failStep: "direct"},
		{name: "extension update fails", directID: directID, failStep: "update"},
		{name: "final get fails", directID: directID, failStep: "get_updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newExtErrFixture(mc)
			ctx := context.Background()

			ext := errCaseExtension()
			ext.DirectID = tt.directID

			order := []string{"get", "direct", "update", "get_updated"}
			var calls []*gomock.Call
			for _, step := range order {
				failing := step == tt.failStep
				var stepErr error
				if failing {
					stepErr = errInjectedErrCase
				}

				switch step {
				case "get":
					if failing {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr))
					} else {
						calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(ext, nil))
					}
				case "direct":
					var d *dmdirect.Direct
					if !failing {
						d = direct
					}
					if tt.directID != uuid.Nil {
						calls = append(calls, f.req.EXPECT().DirectV1DirectRegenerate(ctx, tt.directID).Return(d, stepErr))
					} else {
						calls = append(calls, f.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, id).Return(d, stepErr))
					}
				case "update":
					calls = append(calls, f.dbBin.EXPECT().ExtensionUpdate(ctx, id, wantFields).Return(stepErr))
				case "get_updated":
					calls = append(calls, f.dbBin.EXPECT().ExtensionGet(ctx, id).Return(nil, stepErr))
				}

				if failing {
					break
				}
			}
			chainCallsErrCase(calls)

			res, err := f.h.DirectHashRegenerate(ctx, id)
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

	f := newExtErrFixture(mc)
	ctx := context.Background()

	filters := map[extension.Field]any{extension.FieldDeleted: false}
	f.dbBin.EXPECT().ExtensionList(ctx, uint64(10), "token", filters).Return(nil, errInjectedErrCase)

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

	f := newExtErrFixture(mc)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001")

	f.dbBin.EXPECT().ExtensionCountByCustomerID(ctx, customerID).Return(7, errInjectedErrCase)

	count, err := f.h.CountByCustomerID(ctx, customerID)
	if !errors.Is(err, errInjectedErrCase) {
		t.Errorf("Wrong match. expect: injected error, got: %v", err)
	}
	if count != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", count)
	}
}

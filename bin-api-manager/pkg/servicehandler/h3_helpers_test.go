package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	h3CustomerID      = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	h3OtherCustomerID = uuid.FromStringOrNil("9a1b2c3d-8e5f-11ee-97b2-cfe7337b7099")
	h3AgentID         = uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	h3ResourceID      = uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425")
	errH3             = errors.New("h3 fake rpc error")
)

func h3Agent(customerID uuid.UUID, perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: h3AgentID, CustomerID: customerID},
		Permission: perm,
	})
}

func h3Admin() *auth.AuthIdentity {
	return h3Agent(h3CustomerID, amagent.PermissionCustomerAdmin)
}

func h3NoPerm() *auth.AuthIdentity {
	return h3Agent(h3CustomerID, amagent.PermissionNone)
}

func h3Direct() *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           h3CustomerID,
		ResourceType:         "ai",
		ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
		AllowedResourceTypes: []string{"aicall"},
		AllowedResourceID:    uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000001"),
	})
}

func h3NewHandler(mc *gomock.Controller) (*serviceHandler, *requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	return &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: mockUtil,
	}, mockReq, mockUtil
}

// h3Row is a single guard/propagation case for a handler method returning only an error.
type h3Row struct {
	name    string
	a       *auth.AuthIdentity
	setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	call    func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error
	wantErr error
}

func h3Run(t *testing.T, rows []h3Row) {
	t.Helper()
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockReq, _ := h3NewHandler(mc)
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(mockReq, ctx)
			}

			err := tt.call(h, ctx, tt.a)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// h3Errs are the sentinel errors reused across the h3 tests.
var (
	h3ErrDirect = serviceerrors.ErrDirectAccessNotSupported
	h3ErrDenied = serviceerrors.ErrPermissionDenied
)

func h3ErrorsIs(err, target error) bool { return errors.Is(err, target) }

// h3ResRow is a case for a handler method returning (T, error).
type h3ResRow[T any] struct {
	name    string
	a       *auth.AuthIdentity
	setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	call    func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (T, error)
	want    T
	wantErr error
}

func h3RunRes[T any](t *testing.T, rows []h3ResRow[T]) {
	t.Helper()
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockReq, _ := h3NewHandler(mc)
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(mockReq, ctx)
			}

			res, err := tt.call(h, ctx, tt.a)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
				}
				var zero T
				if !reflect.DeepEqual(res, zero) {
					t.Errorf("Wrong match. expect zero result on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.want, res)
			}
		})
	}
}

// h3Denied is an identity that must be rejected, plus the rpc setup reached before the rejection.
type h3Denied struct {
	name  string
	a     *auth.AuthIdentity
	setup func(m *requesthandler.MockRequestHandler, ctx context.Context)
}

// h3Ops describes a typical resource operation: direct reject, denied identities, get/op failure, success.
type h3Ops[T any] struct {
	call   func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (T, error)
	a      *auth.AuthIdentity // identity expected to succeed
	denied []h3Denied
	// getFail is set when a lookup rpc precedes the op; nil otherwise.
	getFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
	// opPre sets the lookup expectation needed before the op (may be nil).
	opPre  func(m *requesthandler.MockRequestHandler, ctx context.Context)
	opFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
	opOK   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	want   T
}

func h3GuardRows[T any](o h3Ops[T]) []h3ResRow[T] {
	rows := []h3ResRow[T]{
		{name: "direct identity is rejected", a: h3Direct(), call: o.call, wantErr: h3ErrDirect},
	}
	for _, d := range o.denied {
		rows = append(rows, h3ResRow[T]{name: d.name, a: d.a, setup: d.setup, call: o.call, wantErr: h3ErrDenied})
	}
	if o.getFail != nil {
		rows = append(rows, h3ResRow[T]{name: "lookup error is propagated", a: o.a, setup: o.getFail, call: o.call, wantErr: errH3})
	}
	rows = append(rows,
		h3ResRow[T]{
			name: "rpc error is propagated", a: o.a, call: o.call, wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				if o.opPre != nil {
					o.opPre(m, ctx)
				}
				o.opFail(m, ctx)
			},
		},
		h3ResRow[T]{
			name: "success", a: o.a, call: o.call, want: o.want,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				if o.opPre != nil {
					o.opPre(m, ctx)
				}
				o.opOK(m, ctx)
			},
		},
	)
	return rows
}

// h3OwnedDenied builds the two standard denied rows (other tenant, no permission) for a lookup-first op.
func h3OwnedDenied(lookup func(m *requesthandler.MockRequestHandler, ctx context.Context)) []h3Denied {
	return []h3Denied{
		{"cross tenant is denied", h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin), lookup},
		{"no permission is denied", h3NoPerm(), lookup},
	}
}

// h3SuperDenied is the denied row for super-admin only operations (a customer admin must not pass).
func h3SuperDenied(setup func(m *requesthandler.MockRequestHandler, ctx context.Context)) []h3Denied {
	return []h3Denied{
		{"customer admin is denied", h3Admin(), setup},
		{"no permission is denied", h3NoPerm(), setup},
	}
}

func h3Super() *auth.AuthIdentity {
	return h3Agent(h3CustomerID, amagent.PermissionProjectSuperAdmin)
}

// h3NoPermOnly is the denied row for operations that do not look anything up before the permission check.
func h3NoPermOnly() []h3Denied {
	return []h3Denied{{name: "no permission is denied", a: h3NoPerm()}}
}

type h3GomockCall = gomock.Call

func h3NewController(t *testing.T) *gomock.Controller {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	return mc
}

func h3DeepEqual(a, b any) bool { return reflect.DeepEqual(a, b) }

func h3NilActions() []fmaction.Action { return nil }

func h3InOrder(calls ...any) { gomock.InOrder(calls...) }

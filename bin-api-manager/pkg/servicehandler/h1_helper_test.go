package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

const (
	h1CustomerID = "6a1f0b10-0000-4000-8000-000000000001"
	h1OtherID    = "6a1f0b10-0000-4000-8000-000000000002"
	h1ResourceID = "6a1f0b10-0000-4000-8000-0000000000a1"
	h1AgentID    = "6a1f0b10-0000-4000-8000-0000000000b1"
)

var errH1RPC = errors.New("h1 rpc failure")

type h1Mocks struct {
	req  *requesthandler.MockRequestHandler
	util *utilhandler.MockUtilHandler
	db   *dbhandler.MockDBHandler
}

type h1Case struct {
	name    string
	setup   func(m *h1Mocks)
	call    func(ctx context.Context, h *serviceHandler) (any, error)
	wantErr error
	// wantErrText expects a non-nil error whose message contains this text (for errors with no sentinel,
	// e.g. a parse error or a plain not-found message).
	wantErrText string
	wantRes     any
}

func h1UUID(s string) uuid.UUID {
	return uuid.FromStringOrNil(s)
}

// h1Agent returns an agent identity of the given customer with the given permission.
func h1Agent(customerID string, perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1AgentID),
			CustomerID: h1UUID(customerID),
		},
		Permission: perm,
	})
}

func h1Admin() *auth.AuthIdentity {
	return h1Agent(h1CustomerID, amagent.PermissionCustomerAdmin)
}

func h1Manager() *auth.AuthIdentity {
	return h1Agent(h1CustomerID, amagent.PermissionCustomerManager)
}

func h1NoPerm() *auth.AuthIdentity {
	return h1Agent(h1CustomerID, amagent.PermissionNone)
}

func h1Foreign() *auth.AuthIdentity {
	return h1Agent(h1OtherID, amagent.PermissionCustomerAdmin)
}

func h1Super() *auth.AuthIdentity {
	return h1Agent(h1OtherID, amagent.PermissionProjectSuperAdmin)
}

func h1Direct() *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{CustomerID: h1UUID(h1CustomerID)})
}

func h1IsNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return rv.IsNil()
	case reflect.String:
		return rv.Len() == 0
	}
	return false
}

func h1Run(t *testing.T, tests []h1Case) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &h1Mocks{
				req:  requesthandler.NewMockRequestHandler(mc),
				util: utilhandler.NewMockUtilHandler(mc),
				db:   dbhandler.NewMockDBHandler(mc),
			}
			h := &serviceHandler{
				reqHandler:  m.req,
				utilHandler: m.util,
				dbHandler:   m.db,
			}
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := tt.call(context.Background(), h)
			if tt.wantErrText != "" {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
				}
				if !strings.Contains(err.Error(), tt.wantErrText) {
					t.Errorf("Wrong error text. expect contains: %q, got: %v", tt.wantErrText, err)
				}
				if errors.Is(err, serviceerrors.ErrPermissionDenied) {
					t.Errorf("Wrong error kind. expect: not a permission error, got: %v", err)
				}
				if !h1IsNilValue(res) {
					t.Errorf("Wrong result on error. expect: nil, got: %v", res)
				}
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.wantErr, err)
				}
				if !h1IsNilValue(res) {
					t.Errorf("Wrong result on error. expect: nil, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.wantRes, res)
			}
		})
	}
}

// h1Ctx matches the context argument of an RPC call.
func h1Ctx() gomock.Matcher {
	return gomock.Any()
}

type h1Identity = auth.AuthIdentity

func h1Now() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

// h1ActSpec describes a "direct check -> get resource -> permission -> RPC action"
// operation so that the shared guard table can be generated.
type h1ActSpec struct {
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error)
	// getOK expects the resource lookup to succeed with a resource owned by customerID.
	getOK func(m *h1Mocks, customerID string)
	// getErr expects the resource lookup to fail with err.
	getErr func(m *h1Mocks, err error)
	// getDeleted expects the lookup to return a soft-deleted resource (optional).
	getDeleted func(m *h1Mocks)
	// actExpect expects the action RPC; it returns a resource of customerID, or err.
	actExpect func(m *h1Mocks, customerID string, err error)
	// okAgent is the identity that is permitted (defaults to customer admin).
	okAgent *h1Identity
	// deniedSameTenant is an identity of the same tenant that must be refused (defaults to no permission).
	deniedSameTenant *h1Identity
	want             any
	// notFoundCrossTenant makes the cross tenant case expect ErrNotFound instead of ErrPermissionDenied.
	notFoundCrossTenant bool
	// directErr is the error expected for a direct identity (defaults to ErrDirectAccessNotSupported).
	directErr error
}

func h1ActCases(s h1ActSpec) []h1Case {
	ok := s.okAgent
	if ok == nil {
		ok = h1Admin()
	}
	denied := s.deniedSameTenant
	if denied == nil {
		denied = h1NoPerm()
	}
	crossErr := serviceerrors.ErrPermissionDenied
	if s.notFoundCrossTenant {
		crossErr = serviceerrors.ErrNotFound
	}

	directErr := s.directErr
	if directErr == nil {
		directErr = serviceerrors.ErrDirectAccessNotSupported
	}

	tests := []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, h1Direct()) },
			wantErr: directErr,
		},
		{
			name:    "get error",
			setup:   func(m *h1Mocks) { s.getErr(m, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, ok) },
			wantErr: errH1RPC,
		},
	}
	if s.getDeleted != nil {
		tests = append(tests, h1Case{
			name:    "deleted resource is not found",
			setup:   s.getDeleted,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, ok) },
			wantErr: serviceerrors.ErrNotFound,
		})
	}
	tests = append(tests,
		h1Case{
			name:    "no permission",
			setup:   func(m *h1Mocks) { s.getOK(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, denied) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		h1Case{
			name:    "cross tenant",
			setup:   func(m *h1Mocks) { s.getOK(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, h1Foreign()) },
			wantErr: crossErr,
		},
		h1Case{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				s.getOK(m, h1CustomerID)
				s.actExpect(m, h1CustomerID, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, ok) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "success",
			setup: func(m *h1Mocks) {
				s.getOK(m, h1CustomerID)
				s.actExpect(m, h1CustomerID, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return s.call(ctx, h, ok) },
			wantRes: s.want,
		},
	)

	if s.actExpect == nil {
		// lookup-only operation: drop the action-failure case, success needs only the lookup
		var kept []h1Case
		for _, c := range tests {
			if c.name == "action rpc error" {
				continue
			}
			if c.name == "success" {
				c.setup = func(m *h1Mocks) { s.getOK(m, h1CustomerID) }
			}
			kept = append(kept, c)
		}
		return kept
	}

	return tests
}

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

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	h2CustomerID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000001")
)

// h2AgentIdentity returns an agent identity with the given customer and permission.
func h2AgentIdentity(customerID uuid.UUID, perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000a1"),
			CustomerID: customerID,
		},
		Permission: perm,
	})
}

// h2DirectIdentity returns a direct-scope identity.
func h2DirectIdentity(customerID uuid.UUID) *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{CustomerID: customerID})
}

// h2Case is one table row of a servicehandler test.
type h2Case struct {
	name    string
	setup   func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, db *dbhandler.MockDBHandler)
	call    func(ctx context.Context, h *serviceHandler) (any, error)
	wantErr error
	wantRes any
}

// h2RunCases runs the table with a fresh handler and mocks per row.
func h2RunCases(t *testing.T, tests []h2Case) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				utilHandler: mockUtil,
				dbHandler:   mockDB,
			}
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, mockReq, mockUtil, mockDB)
			}

			res, err := tt.call(ctx, h)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.wantErr, err)
				}
				if res != nil {
					rv := reflect.ValueOf(res)
					switch rv.Kind() {
					case reflect.Pointer, reflect.Slice, reflect.Map:
						if !rv.IsNil() {
							t.Errorf("Expected nil result on error, got: %v", res)
						}
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("Wrong match.\nexpect: %#v\ngot: %#v", tt.wantRes, res)
			}
		})
	}
}

var (
	h2AgentID         = uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000a1")
	h2OtherCustomerID = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000c9")
	errH2RPC          = errors.New("h2 rpc failure")
)

// h2AgentResp is the agent-manager response for h2AgentIdentity.
func h2AgentResp(customerID uuid.UUID) *amagent.Agent {
	return &amagent.Agent{
		Identity: commonidentity.Identity{ID: h2AgentID, CustomerID: customerID},
	}
}

// h2LookupFn sets the expectation of the ownership lookup. When failure is
// true the lookup returns errH2RPC, otherwise a resource owned by customerID.
type h2LookupFn func(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool)

// h2RPCFn sets the expectation of the final RPC. When failure is true the RPC
// returns errH2RPC, otherwise it succeeds with the value of wantRes.
type h2RPCFn func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool)

// h2StdCases builds the standard guard/error/success table of a handler:
// direct rejection, lookup failure, cross tenant, missing permission, RPC
// failure and success. A nil lookup skips the lookup based rows.
func h2StdCases(
	call func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error),
	lookup h2LookupFn,
	rpc h2RPCFn,
	wantRes any,
) []h2Case {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	noPerm := h2AgentIdentity(h2CustomerID, amagent.PermissionNone)

	setupLookup := func(customerID uuid.UUID, failure bool) func(context.Context, *requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler, *dbhandler.MockDBHandler) {
		return func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
			if lookup != nil {
				lookup(ctx, req, customerID, failure)
			}
		}
	}
	setupAll := func(failure bool) func(context.Context, *requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler, *dbhandler.MockDBHandler) {
		return func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
			if lookup != nil {
				lookup(ctx, req, h2CustomerID, false)
			}
			rpc(ctx, req, failure)
		}
	}

	res := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2DirectIdentity(h2CustomerID))
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
	}
	if lookup != nil {
		res = append(res,
			h2Case{
				name:  "lookup fails",
				setup: setupLookup(h2CustomerID, true),
				call: func(ctx context.Context, h *serviceHandler) (any, error) {
					return call(ctx, h, admin)
				},
				wantErr: errH2RPC,
			},
			h2Case{
				name:  "cross tenant",
				setup: setupLookup(h2OtherCustomerID, false),
				call: func(ctx context.Context, h *serviceHandler) (any, error) {
					return call(ctx, h, admin)
				},
				wantErr: serviceerrors.ErrPermissionDenied,
			},
		)
	}
	res = append(res,
		h2Case{
			name:  "no permission",
			setup: setupLookup(h2CustomerID, false),
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, noPerm)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		h2Case{
			name:  "rpc error",
			setup: setupAll(true),
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantErr: errH2RPC,
		},
		h2Case{
			name:  "success",
			setup: setupAll(false),
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantRes: wantRes,
		},
	)
	return res
}

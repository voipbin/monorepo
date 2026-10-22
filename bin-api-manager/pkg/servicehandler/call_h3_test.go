package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// h3CallRows builds the standard guard rows for a call operation that first loads the call.
func h3CallRows(op func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error, expectOp func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call) []h3Row {
	owned := &cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}
	deleted := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return []h3Row{
		{
			name:    "direct identity is rejected before any rpc",
			a:       h3Direct(),
			call:    op,
			wantErr: h3ErrDirect,
		},
		{
			name: "call lookup error is propagated",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(nil, errH3)
			},
			call:    op,
			wantErr: errH3,
		},
		{
			name: "deleted call is a state error",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(&cmcall.Call{
					Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
					TMDelete: &deleted,
				}, nil)
			},
			call:    op,
			wantErr: serviceerrors.ErrStateInvalid,
		},
		{
			name: "other customer is denied",
			a:    h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
			call:    op,
			wantErr: h3ErrDenied,
		},
		{
			name: "no permission is denied",
			a:    h3NoPerm(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
			call:    op,
			wantErr: h3ErrDenied,
		},
		{
			name: "rpc error is propagated",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
				expectOp(m, ctx).Return(errH3)
			},
			call:    op,
			wantErr: errH3,
		},
		{
			name: "success",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
				expectOp(m, ctx).Return(nil)
			},
			call: op,
		},
	}
}

func Test_CallHold_MuteMOHSilence_h3(t *testing.T) {
	type opCase struct {
		name     string
		op       func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error
		expectOp func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call
	}
	ops := []opCase{
		{
			"CallHoldOn",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallHoldOn(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallHoldOn(ctx, h3ResourceID)
			},
		},
		{
			"CallHoldOff",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallHoldOff(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallHoldOff(ctx, h3ResourceID)
			},
		},
		{
			"CallMuteOn",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallMuteOn(ctx, a, h3ResourceID, cmcall.MuteDirectionIn)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallMuteOn(ctx, h3ResourceID, cmcall.MuteDirectionIn)
			},
		},
		{
			"CallMuteOff",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallMuteOff(ctx, a, h3ResourceID, cmcall.MuteDirectionOut)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallMuteOff(ctx, h3ResourceID, cmcall.MuteDirectionOut)
			},
		},
		{
			"CallMOHOn",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallMOHOn(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallMusicOnHoldOn(ctx, h3ResourceID)
			},
		},
		{
			"CallMOHOff",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallMOHOff(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallMusicOnHoldOff(ctx, h3ResourceID)
			},
		},
		{
			"CallSilenceOn",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallSilenceOn(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallSilenceOn(ctx, h3ResourceID)
			},
		},
		{
			"CallSilenceOff",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallSilenceOff(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallSilenceOff(ctx, h3ResourceID)
			},
		},
		{
			"CallTalk",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
				return h.CallTalk(ctx, a, h3ResourceID, "hello", "en-US", "gcp", "voice-1")
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallTalk(ctx, h3ResourceID, "hello", "en-US", "gcp", "voice-1", 10000)
			},
		},
	}

	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			h3Run(t, h3CallRows(o.op, o.expectOp))
		})
	}
}

func Test_CallGet_h3(t *testing.T) {
	owned := &cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}
	deleted := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		a       *auth.AuthIdentity
		setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
		wantRes *cmcall.WebhookMessage
		wantErr error
	}{
		{name: "direct", a: h3Direct(), wantErr: h3ErrDirect},
		{
			name: "rpc error",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(nil, errH3)
			},
			wantErr: errH3,
		},
		{
			name: "deleted",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(&cmcall.Call{Identity: owned.Identity, TMDelete: &deleted}, nil)
			},
			wantErr: serviceerrors.ErrStateInvalid,
		},
		{
			name: "cross tenant",
			a:    h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
			wantErr: h3ErrDenied,
		},
		{
			name: "no permission",
			a:    h3NoPerm(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
			wantErr: h3ErrDenied,
		},
		{
			name: "success",
			a:    h3Admin(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
			wantRes: owned.ConvertWebhookMessage(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockReq, _ := h3NewHandler(mc)
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(mockReq, ctx)
			}
			res, err := h.CallGet(ctx, tt.a, h3ResourceID)
			if tt.wantErr != nil {
				if !h3ErrorsIs(err, tt.wantErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantRes, res)
			}
		})
	}
}

func Test_CallDeleteHangup_h3(t *testing.T) {
	owned := &cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}
	deleted := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result := &cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}, Status: cmcall.StatusHangup}

	type fn func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmcall.WebhookMessage, error)
	ops := []struct {
		name     string
		op       fn
		expectOp func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call
	}{
		{
			"CallDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmcall.WebhookMessage, error) {
				return h.CallDelete(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallDelete(ctx, h3ResourceID)
			},
		},
		{
			"CallHangup",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmcall.WebhookMessage, error) {
				return h.CallHangup(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallHangup(ctx, h3ResourceID)
			},
		},
		{
			"CallRecordingStop",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmcall.WebhookMessage, error) {
				return h.CallRecordingStop(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallRecordingStop(ctx, h3ResourceID)
			},
		},
		{
			"CallRecordingStart",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmcall.WebhookMessage, error) {
				return h.CallRecordingStart(ctx, a, h3ResourceID, cmrecording.FormatWAV, 5, "#", 60, uuid.Nil)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) *gomock.Call {
				return m.EXPECT().CallV1CallRecordingStart(ctx, h3ResourceID, cmrecording.FormatWAV, 5, "#", 60, uuid.Nil)
			},
		},
	}

	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			tests := []struct {
				name    string
				a       *auth.AuthIdentity
				setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
				wantRes *cmcall.WebhookMessage
				wantErr error
			}{
				{name: "direct", a: h3Direct(), wantErr: h3ErrDirect},
				{
					name: "get error",
					a:    h3Admin(),
					setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
						m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(nil, errH3)
					},
					wantErr: errH3,
				},
				{
					name: "deleted",
					a:    h3Admin(),
					setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
						m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(&cmcall.Call{Identity: owned.Identity, TMDelete: &deleted}, nil)
					},
					wantErr: serviceerrors.ErrStateInvalid,
				},
				{
					name: "cross tenant",
					a:    h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin),
					setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
						m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
					},
					wantErr: h3ErrDenied,
				},
				{
					name: "rpc error",
					a:    h3Admin(),
					setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
						m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
						o.expectOp(m, ctx).Return(nil, errH3)
					},
					wantErr: errH3,
				},
				{
					name: "success",
					a:    h3Admin(),
					setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
						m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
						o.expectOp(m, ctx).Return(result, nil)
					},
					wantRes: result.ConvertWebhookMessage(),
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					mc := gomock.NewController(t)
					defer mc.Finish()
					h, mockReq, _ := h3NewHandler(mc)
					ctx := context.Background()
					if tt.setup != nil {
						tt.setup(mockReq, ctx)
					}
					res, err := o.op(h, ctx, tt.a)
					if tt.wantErr != nil {
						if !h3ErrorsIs(err, tt.wantErr) {
							t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
						}
						if res != nil {
							t.Errorf("Wrong match. expect nil result on error, got: %v", res)
						}
						return
					}
					if err != nil {
						t.Fatalf("Wrong match. expect: ok, got: %v", err)
					}
					if !reflect.DeepEqual(res, tt.wantRes) {
						t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantRes, res)
					}
				})
			}
		})
	}
}

func Test_CallMediaStreamStart_guards_h3(t *testing.T) {
	owned := &cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}
	op := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
		return h.CallMediaStreamStart(ctx, a, h3ResourceID, "audiosocket", nil, nil)
	}
	h3Run(t, []h3Row{
		{name: "direct", a: h3Direct(), call: op, wantErr: h3ErrDirect},
		{
			name: "get error", a: h3Admin(), call: op, wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(nil, errH3)
			},
		},
		{
			name: "no permission", a: h3NoPerm(), call: op, wantErr: h3ErrDenied,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, h3ResourceID).Return(owned, nil)
			},
		},
	})
}

func Test_CallList_h3(t *testing.T) {
	tests := []struct {
		name    string
		a       *auth.AuthIdentity
		token   string
		setup   func(m *requesthandler.MockRequestHandler, u func(string))
		wantRes []*cmcall.WebhookMessage
		wantErr error
	}{
		{name: "direct", a: h3Direct(), wantErr: h3ErrDirect},
		{name: "no permission", a: h3NoPerm(), token: "t", wantErr: h3ErrDenied},
		{
			name:  "rpc error",
			a:     h3Admin(),
			token: "2024-01-01 00:00:00.000000",
			setup: func(m *requesthandler.MockRequestHandler, _ func(string)) {
				m.EXPECT().CallV1CallList(gomock.Any(), "2024-01-01 00:00:00.000000", uint64(10), map[cmcall.Field]any{
					cmcall.FieldCustomerID: h3CustomerID,
					cmcall.FieldDeleted:    false,
				}).Return(nil, errH3)
			},
			wantErr: errH3,
		},
		{
			name:  "success",
			a:     h3Admin(),
			token: "2024-01-01 00:00:00.000000",
			setup: func(m *requesthandler.MockRequestHandler, _ func(string)) {
				m.EXPECT().CallV1CallList(gomock.Any(), "2024-01-01 00:00:00.000000", uint64(10), map[cmcall.Field]any{
					cmcall.FieldCustomerID: h3CustomerID,
					cmcall.FieldDeleted:    false,
				}).Return([]cmcall.Call{{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}}, nil)
			},
			wantRes: []*cmcall.WebhookMessage{(&cmcall.Call{Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID}}).ConvertWebhookMessage()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockReq, _ := h3NewHandler(mc)
			if tt.setup != nil {
				tt.setup(mockReq, nil)
			}
			res, err := h.CallList(context.Background(), tt.a, 10, tt.token)
			if tt.wantErr != nil {
				if !h3ErrorsIs(err, tt.wantErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantRes, res)
			}
		})
	}
}

func Test_callGetsByFilters_h3(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockReq, mockUtil := h3NewHandler(mc)

	// invalid filter key must fail in conversion, before any rpc
	_, err := h.callGetsByFilters(context.Background(), 10, "tok", map[string]string{"customer_id": "not-a-uuid"})
	if err == nil {
		t.Errorf("Wrong match. expect: error for invalid customer_id filter, got: nil")
	}

	// empty token is replaced by the current time
	mockUtil.EXPECT().TimeGetCurTime().Return("2024-02-02 00:00:00.000000")
	mockReq.EXPECT().CallV1CallList(gomock.Any(), "2024-02-02 00:00:00.000000", uint64(5), map[cmcall.Field]any{
		cmcall.FieldCustomerID: h3CustomerID,
	}).Return([]cmcall.Call{}, nil)
	res, err := h.callGetsByFilters(context.Background(), 5, "", map[string]string{"customer_id": h3CustomerID.String()})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, []*cmcall.WebhookMessage{}) {
		t.Errorf("Wrong match. expect: empty slice, got: %v", res)
	}
}

func Test_CallCreate_guards_h3(t *testing.T) {
	tel := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15555550100"}}
	sip := []commonaddress.Address{{Type: commonaddress.TypeSIP, Target: "user@example.com"}}
	flowID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	type fn func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error)
	mk := func(dest []commonaddress.Address) fn {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error) {
			return h.CallCreate(ctx, a, flowID, nil, nil, dest, "", nil)
		}
	}
	deletedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		a       *auth.AuthIdentity
		call    fn
		setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
		wantErr error
	}{
		{name: "direct", a: h3Direct(), call: mk(sip), wantErr: h3ErrDirect},
		{name: "no permission", a: h3NoPerm(), call: mk(sip), wantErr: h3ErrDenied},
		{
			name: "customer lookup fails for tel", a: h3Admin(), call: mk(tel), wantErr: serviceerrors.ErrInternal,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CustomerV1CustomerGet(ctx, h3CustomerID).Return(nil, errH3)
			},
		},
		{
			name: "unverified customer cannot call pstn", a: h3Admin(), call: mk(tel), wantErr: serviceerrors.ErrIdentityVerificationRequired,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CustomerV1CustomerGet(ctx, h3CustomerID).Return(&cscustomer.Customer{
					ID: h3CustomerID, IdentityVerificationStatus: cscustomer.IdentityVerificationStatusNone,
				}, nil)
			},
		},
		{
			name: "flow lookup fails", a: h3Admin(), call: mk(sip), wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowGet(ctx, flowID).Return(nil, errH3)
			},
		},
		{
			name: "deleted flow", a: h3Admin(), call: mk(sip), wantErr: serviceerrors.ErrNotFound,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowGet(ctx, flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: h3CustomerID}, TMDelete: &deletedAt,
				}, nil)
			},
		},
		{
			name: "flow of another customer", a: h3Admin(), call: mk(sip), wantErr: h3ErrDenied,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowGet(ctx, flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: h3OtherCustomerID},
				}, nil)
			},
		},
		{
			name: "create rpc error", a: h3Admin(), call: mk(sip), wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowGet(ctx, flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: h3CustomerID},
				}, nil)
				m.EXPECT().CallV1CallsCreate(ctx, h3CustomerID, flowID, uuid.Nil, (*commonaddress.Address)(nil), sip, false, false, "", nil, map[string]string(nil)).Return(nil, nil, errH3)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockReq, _ := h3NewHandler(mc)
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(mockReq, ctx)
			}
			calls, groups, err := tt.call(h, ctx, tt.a)
			if !h3ErrorsIs(err, tt.wantErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
			}
			if calls != nil || groups != nil {
				t.Errorf("Wrong match. expect nil results on error, got: %v %v", calls, groups)
			}
		})
	}
}

package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
)

func h3Groupcall() *cmgroupcall.Groupcall {
	return &cmgroupcall.Groupcall{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
	}
}

func h3GroupcallLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().CallV1GroupcallGet(ctx, h3ResourceID).Return(h3Groupcall(), nil)
}

func h3GroupcallLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().CallV1GroupcallGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_GroupcallGet_guards_h3(t *testing.T) {
	o := h3Ops[*cmgroupcall.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
			return h.GroupcallGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(h3GroupcallLookup),
		opFail: h3GroupcallLookupFail,
		opOK:   h3GroupcallLookup,
		want:   h3Groupcall().ConvertWebhookMessage(),
	}
	rows := h3GuardRows(o)
	// every non-direct row must go through the lookup; the rpc-error row covers lookup failure.
	h3RunRes(t, rows)
}

func Test_GroupcallHangupDelete_h3(t *testing.T) {
	ops := []struct {
		name   string
		call   func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error)
		opFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
		opOK   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	}{
		{
			"GroupcallHangup",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
				return h.GroupcallHangup(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1GroupcallHangup(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1GroupcallHangup(ctx, h3ResourceID).Return(h3Groupcall(), nil)
			},
		},
		{
			"GroupcallDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
				return h.GroupcallDelete(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1GroupcallDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().CallV1GroupcallDelete(ctx, h3ResourceID).Return(h3Groupcall(), nil)
			},
		},
	}
	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			h3RunRes(t, h3GuardRows(h3Ops[*cmgroupcall.WebhookMessage]{
				call:    o.call,
				a:       h3Admin(),
				denied:  h3OwnedDenied(h3GroupcallLookup),
				getFail: h3GroupcallLookupFail,
				opPre:   h3GroupcallLookup,
				opFail:  o.opFail,
				opOK:    o.opOK,
				want:    h3Groupcall().ConvertWebhookMessage(),
			}))
		})
	}
}

func Test_GroupcallList_h3(t *testing.T) {
	filters := map[cmgroupcall.Field]any{
		cmgroupcall.FieldCustomerID: h3CustomerID,
		cmgroupcall.FieldDeleted:    false,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*cmgroupcall.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*cmgroupcall.WebhookMessage, error) {
			return h.GroupcallList(ctx, a, 10, "tok")
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1GroupcallList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1GroupcallList(ctx, "tok", uint64(10), filters).Return([]cmgroupcall.Groupcall{*h3Groupcall()}, nil)
		},
		want: []*cmgroupcall.WebhookMessage{h3Groupcall().ConvertWebhookMessage()},
	}))
}

func Test_GroupcallList_emptyToken_h3(t *testing.T) {
	{
		mc := h3NewController(t)
		h, mockReq, mockUtil := h3NewHandler(mc)
		ctx := context.Background()
		mockUtil.EXPECT().TimeGetCurTime().Return("2024-03-03 00:00:00.000000")
		mockReq.EXPECT().CallV1GroupcallList(ctx, "2024-03-03 00:00:00.000000", uint64(3), map[cmgroupcall.Field]any{
			cmgroupcall.FieldCustomerID: h3CustomerID,
			cmgroupcall.FieldDeleted:    false,
		}).Return([]cmgroupcall.Groupcall{}, nil)
		res, err := h.GroupcallList(ctx, h3Admin(), 3, "")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: empty non-nil slice, got: %v", res)
		}
	}
}

func Test_GroupcallCreate_guards_h3(t *testing.T) {
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15555550100"}
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15555550101"}}
	flowID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	call := func(fid uuid.UUID) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
			return h.GroupcallCreate(ctx, a, src, dests, fid, nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers)
		}
	}
	create := func(m *requesthandler.MockRequestHandler, ctx context.Context) *h3GomockCall {
		return m.EXPECT().CallV1GroupcallCreate(ctx, uuid.Nil, h3CustomerID, flowID, src, dests, uuid.Nil, uuid.Nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers, "", map[string]string(nil))
	}
	h3RunRes(t, h3GuardRows(h3Ops[*cmgroupcall.WebhookMessage]{
		call:   call(flowID),
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			create(m, ctx).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			create(m, ctx).Return(h3Groupcall(), nil)
		},
		want: h3Groupcall().ConvertWebhookMessage(),
	}))

	// nil flow id: a temporary flow is created first; its failure aborts the groupcall.
	h3RunRes(t, []h3ResRow[*cmgroupcall.WebhookMessage]{
		{
			name: "tmp flow creation error aborts", a: h3Admin(), call: call(uuid.Nil), wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowCreate(ctx, h3CustomerID, fmflow.TypeFlow, "tmp", "tmp outbound flow", h3NilActions(), uuid.Nil, false).Return(nil, errH3)
			},
		},
	})
}

func Test_GroupcallCreate_tmpFlow_h3(t *testing.T) {
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15555550100"}
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15555550101"}}
	tmpFlowID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	h3RunRes(t, []h3ResRow[*cmgroupcall.WebhookMessage]{
		{
			name: "tmp flow id is used for the groupcall", a: h3Admin(),
			call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmgroupcall.WebhookMessage, error) {
				return h.GroupcallCreate(ctx, a, src, dests, uuid.Nil, nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers)
			},
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().FlowV1FlowCreate(ctx, h3CustomerID, fmflow.TypeFlow, "tmp", "tmp outbound flow", h3NilActions(), uuid.Nil, false).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: tmpFlowID, CustomerID: h3CustomerID},
				}, nil)
				m.EXPECT().CallV1GroupcallCreate(ctx, uuid.Nil, h3CustomerID, tmpFlowID, src, dests, uuid.Nil, uuid.Nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers, "", map[string]string(nil)).Return(h3Groupcall(), nil)
			},
			want: h3Groupcall().ConvertWebhookMessage(),
		},
	})
}

func Test_convertGroupcallFilters_h3(t *testing.T) {
	mc := h3NewController(t)
	h, _, _ := h3NewHandler(mc)

	if _, err := h.convertGroupcallFilters(map[string]string{"customer_id": "not-a-uuid"}); err == nil {
		t.Errorf("Wrong match. expect: error for invalid customer_id, got: nil")
	}

	res, err := h.convertGroupcallFilters(map[string]string{"customer_id": h3CustomerID.String(), "deleted": "false"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	want := map[cmgroupcall.Field]any{cmgroupcall.FieldCustomerID: h3CustomerID, cmgroupcall.FieldDeleted: false}
	if !h3DeepEqual(res, want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, res)
	}
}

package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"go.uber.org/mock/gomock"
)

func Test_Tag_Guards(t *testing.T) {
	foreignTag := &tmtag.Tag{Identity: commonidentity.Identity{ID: guardResourceID, CustomerID: guardOtherCustomerID}}
	ownTag := &tmtag.Tag{Identity: commonidentity.Identity{ID: guardResourceID, CustomerID: guardCustomerID}}

	admin := guardAgent(amagent.PermissionCustomerAdmin)
	none := guardAgent(amagent.PermissionNone)

	create := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.TagCreate(ctx, a, "n", "d")
		return err
	}
	get := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.TagGet(ctx, a, guardResourceID)
		return err
	}
	list := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.TagList(ctx, a, 10, "2020-09-20T03:23:20.995000Z")
		return err
	}
	del := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.TagDelete(ctx, a, guardResourceID)
		return err
	}
	update := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.TagUpdate(ctx, a, guardResourceID, "n", "d")
		return err
	}
	tagGetOK := func(res *tmtag.Tag) func(m *requesthandler.MockRequestHandler) {
		return func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TagV1TagGet(gomock.Any(), guardResourceID).Return(res, nil)
		}
	}
	tagGetErr := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().TagV1TagGet(gomock.Any(), guardResourceID).Return(nil, errGuardDownstream)
	}

	tests := []guardCase{
		{name: "create direct", agent: guardDirect(), call: create, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "create no permission", agent: none, call: create, wantErr: serviceerrors.ErrPermissionDenied},
		{name: "create downstream error", agent: admin, call: create, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TagV1TagCreate(gomock.Any(), guardCustomerID, "n", "d").Return(nil, errGuardDownstream)
		}},
		{name: "create allowed for manager only", agent: guardAgent(amagent.PermissionCustomerManager), call: create, wantErr: nil, setup: func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TagV1TagCreate(gomock.Any(), guardCustomerID, "n", "d").Return(&tmtag.Tag{}, nil)
		}},
		{name: "get direct", agent: guardDirect(), call: get, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "get lookup error", agent: admin, call: get, wantErr: errGuardDownstream, setup: tagGetErr},
		{name: "get foreign tenant", agent: admin, call: get, wantErr: serviceerrors.ErrPermissionDenied, setup: tagGetOK(foreignTag)},
		{name: "list direct", agent: guardDirect(), call: list, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "list no permission", agent: none, call: list, wantErr: serviceerrors.ErrPermissionDenied},
		{name: "list downstream error", agent: admin, call: list, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TagV1TagList(gomock.Any(), "2020-09-20T03:23:20.995000Z", uint64(10), map[tmtag.Field]any{tmtag.FieldCustomerID: guardCustomerID}).Return(nil, errGuardDownstream)
		}},
		{name: "delete direct", agent: guardDirect(), call: del, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "delete lookup error", agent: admin, call: del, wantErr: errGuardDownstream, setup: tagGetErr},
		{name: "delete foreign tenant", agent: admin, call: del, wantErr: serviceerrors.ErrPermissionDenied, setup: tagGetOK(foreignTag)},
		{name: "delete downstream error", agent: admin, call: del, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			tagGetOK(ownTag)(m)
			m.EXPECT().TagV1TagDelete(gomock.Any(), guardResourceID).Return(nil, errGuardDownstream)
		}},
		{name: "update direct", agent: guardDirect(), call: update, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "update lookup error", agent: admin, call: update, wantErr: errGuardDownstream, setup: tagGetErr},
		{name: "update foreign tenant", agent: admin, call: update, wantErr: serviceerrors.ErrPermissionDenied, setup: tagGetOK(foreignTag)},
		{name: "update downstream error", agent: admin, call: update, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			tagGetOK(ownTag)(m)
			m.EXPECT().TagV1TagUpdate(gomock.Any(), guardResourceID, "n", "d").Return(nil, errGuardDownstream)
		}},
	}

	runGuardCases(t, tests)
}

package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Outdialtarget_Guards(t *testing.T) {
	targetID := uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-000000000010")
	otherOutdialID := uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-000000000011")

	ownOutdial := &omoutdial.Outdial{Identity: commonidentity.Identity{ID: guardResourceID, CustomerID: guardCustomerID}}
	foreignOutdial := &omoutdial.Outdial{Identity: commonidentity.Identity{ID: guardResourceID, CustomerID: guardOtherCustomerID}}

	admin := guardAgent(amagent.PermissionCustomerAdmin)

	outdialOK := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().OutdialV1OutdialGet(gomock.Any(), guardResourceID).Return(ownOutdial, nil)
	}
	outdialForeign := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().OutdialV1OutdialGet(gomock.Any(), guardResourceID).Return(foreignOutdial, nil)
	}
	outdialErr := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().OutdialV1OutdialGet(gomock.Any(), guardResourceID).Return(nil, errGuardDownstream)
	}
	targetOfOtherOutdial := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().OutdialV1OutdialtargetGet(gomock.Any(), targetID).Return(&omoutdialtarget.OutdialTarget{OutdialID: otherOutdialID}, nil)
	}
	targetErr := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().OutdialV1OutdialtargetGet(gomock.Any(), targetID).Return(nil, errGuardDownstream)
	}
	chain := func(fns ...func(m *requesthandler.MockRequestHandler)) func(m *requesthandler.MockRequestHandler) {
		return func(m *requesthandler.MockRequestHandler) {
			for _, f := range fns {
				f(m)
			}
		}
	}

	create := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.OutdialtargetCreate(ctx, a, guardResourceID, "n", "d", "data", nil, nil, nil, nil, nil)
		return err
	}
	get := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.OutdialtargetGet(ctx, a, guardResourceID, targetID)
		return err
	}
	list := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.OutdialtargetGetsByOutdialID(ctx, a, guardResourceID, 10, "2020-09-20T03:23:20.995000Z")
		return err
	}
	del := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.OutdialtargetDelete(ctx, a, guardResourceID, targetID)
		return err
	}

	tests := []guardCase{
		{name: "create direct", agent: guardDirect(), call: create, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "create outdial lookup error", agent: admin, call: create, wantErr: errGuardDownstream, setup: outdialErr},
		{name: "create foreign tenant", agent: admin, call: create, wantErr: serviceerrors.ErrPermissionDenied, setup: outdialForeign},
		{name: "create downstream error", agent: admin, call: create, wantErr: errGuardDownstream, setup: chain(outdialOK, func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().OutdialV1OutdialtargetCreate(gomock.Any(), guardResourceID, "n", "d", "data", nil, nil, nil, nil, nil).Return(nil, errGuardDownstream)
		})},
		{name: "get direct", agent: guardDirect(), call: get, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "get outdial lookup error", agent: admin, call: get, wantErr: errGuardDownstream, setup: outdialErr},
		{name: "get foreign tenant", agent: admin, call: get, wantErr: serviceerrors.ErrPermissionDenied, setup: outdialForeign},
		{name: "get target lookup error", agent: admin, call: get, wantErr: errGuardDownstream, setup: chain(outdialOK, targetErr)},
		{name: "get target of other outdial", agent: admin, call: get, wantErr: serviceerrors.ErrInvalidArgument, setup: chain(outdialOK, targetOfOtherOutdial)},
		{name: "list direct", agent: guardDirect(), call: list, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "list outdial lookup error", agent: admin, call: list, wantErr: errGuardDownstream, setup: outdialErr},
		{name: "list foreign tenant", agent: admin, call: list, wantErr: serviceerrors.ErrPermissionDenied, setup: outdialForeign},
		{name: "list downstream error", agent: admin, call: list, wantErr: errGuardDownstream, setup: chain(outdialOK, func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().OutdialV1OutdialtargetGetsByOutdialID(gomock.Any(), guardResourceID, "2020-09-20T03:23:20.995000Z", uint64(10)).Return(nil, errGuardDownstream)
		})},
		{name: "delete direct", agent: guardDirect(), call: del, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "delete outdial lookup error", agent: admin, call: del, wantErr: errGuardDownstream, setup: outdialErr},
		{name: "delete foreign tenant", agent: admin, call: del, wantErr: serviceerrors.ErrPermissionDenied, setup: outdialForeign},
		{name: "delete target lookup error", agent: admin, call: del, wantErr: errGuardDownstream, setup: chain(outdialOK, targetErr)},
		{name: "delete target of other outdial", agent: admin, call: del, wantErr: serviceerrors.ErrInvalidArgument, setup: chain(outdialOK, targetOfOtherOutdial)},
		{name: "delete downstream error", agent: admin, call: del, wantErr: errGuardDownstream, setup: chain(outdialOK,
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().OutdialV1OutdialtargetGet(gomock.Any(), targetID).Return(&omoutdialtarget.OutdialTarget{OutdialID: guardResourceID}, nil)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().OutdialV1OutdialtargetDelete(gomock.Any(), targetID).Return(nil, errGuardDownstream)
			})},
	}

	runGuardCases(t, tests)
}

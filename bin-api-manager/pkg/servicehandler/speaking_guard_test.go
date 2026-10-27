package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"go.uber.org/mock/gomock"
)

func Test_Speaking_Guards(t *testing.T) {
	const podID = "pod-guard-1"
	admin := guardAgent(amagent.PermissionCustomerAdmin)
	none := guardAgent(amagent.PermissionNone)

	spk := func(status tmspeaking.Status) *tmspeaking.Speaking {
		return &tmspeaking.Speaking{
			Identity: commonidentity.Identity{ID: guardResourceID, CustomerID: guardCustomerID},
			Status:   status,
			PodID:    podID,
		}
	}
	foreign := spk(tmspeaking.StatusActive)
	foreign.CustomerID = guardOtherCustomerID

	getOK := func(s *tmspeaking.Speaking) func(m *requesthandler.MockRequestHandler) {
		return func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TTSV1SpeakingGet(gomock.Any(), guardResourceID).Return(s, nil)
		}
	}
	getErr := func(m *requesthandler.MockRequestHandler) {
		m.EXPECT().TTSV1SpeakingGet(gomock.Any(), guardResourceID).Return(nil, errGuardDownstream)
	}

	create := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingCreate(ctx, a, tmstreaming.ReferenceTypeCall, guardResourceID, "en-US", "gcp", "voice", tmstreaming.DirectionOutgoing)
		return err
	}
	get := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingGet(ctx, a, guardResourceID)
		return err
	}
	list := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingList(ctx, a, 10, "2020-09-20T03:23:20.995000Z")
		return err
	}
	say := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingSay(ctx, a, guardResourceID, "hello")
		return err
	}
	flush := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingFlush(ctx, a, guardResourceID)
		return err
	}
	stop := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingStop(ctx, a, guardResourceID)
		return err
	}
	del := func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error {
		_, err := h.SpeakingDelete(ctx, a, guardResourceID)
		return err
	}

	tests := []guardCase{
		{name: "create direct", agent: guardDirect(), call: create, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "create no permission", agent: none, call: create, wantErr: serviceerrors.ErrPermissionDenied},
		{name: "create downstream error", agent: admin, call: create, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TTSV1SpeakingCreate(gomock.Any(), guardCustomerID, tmstreaming.ReferenceTypeCall, guardResourceID, "en-US", "gcp", "voice", tmstreaming.DirectionOutgoing).Return(nil, errGuardDownstream)
		}},
		{name: "get direct", agent: guardDirect(), call: get, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "get lookup error", agent: admin, call: get, wantErr: errGuardDownstream, setup: getErr},
		{name: "get foreign tenant", agent: admin, call: get, wantErr: serviceerrors.ErrPermissionDenied, setup: getOK(foreign)},
		{name: "list direct", agent: guardDirect(), call: list, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "list no permission", agent: none, call: list, wantErr: serviceerrors.ErrPermissionDenied},
		{name: "list downstream error", agent: admin, call: list, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			m.EXPECT().TTSV1SpeakingGets(gomock.Any(), "2020-09-20T03:23:20.995000Z", uint64(10), map[tmspeaking.Field]any{
				tmspeaking.FieldCustomerID: guardCustomerID,
				tmspeaking.FieldDeleted:    false,
			}).Return(nil, errGuardDownstream)
		}},
		{name: "say direct", agent: guardDirect(), call: say, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "say lookup error", agent: admin, call: say, wantErr: errGuardDownstream, setup: getErr},
		{name: "say foreign tenant", agent: admin, call: say, wantErr: serviceerrors.ErrPermissionDenied, setup: getOK(foreign)},
		{name: "say downstream error", agent: admin, call: say, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			getOK(spk(tmspeaking.StatusActive))(m)
			m.EXPECT().TTSV1SpeakingSay(gomock.Any(), podID, guardResourceID, "hello").Return(nil, errGuardDownstream)
		}},
		{name: "flush direct", agent: guardDirect(), call: flush, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "flush lookup error", agent: admin, call: flush, wantErr: errGuardDownstream, setup: getErr},
		{name: "flush foreign tenant", agent: admin, call: flush, wantErr: serviceerrors.ErrPermissionDenied, setup: getOK(foreign)},
		{name: "flush downstream error", agent: admin, call: flush, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			getOK(spk(tmspeaking.StatusActive))(m)
			m.EXPECT().TTSV1SpeakingFlush(gomock.Any(), podID, guardResourceID).Return(nil, errGuardDownstream)
		}},
		{name: "stop direct", agent: guardDirect(), call: stop, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "stop lookup error", agent: admin, call: stop, wantErr: errGuardDownstream, setup: getErr},
		{name: "stop foreign tenant", agent: admin, call: stop, wantErr: serviceerrors.ErrPermissionDenied, setup: getOK(foreign)},
		{name: "stop downstream error", agent: admin, call: stop, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			getOK(spk(tmspeaking.StatusActive))(m)
			m.EXPECT().TTSV1SpeakingStop(gomock.Any(), podID, guardResourceID).Return(nil, errGuardDownstream)
		}},
		{name: "delete direct", agent: guardDirect(), call: del, wantErr: serviceerrors.ErrDirectAccessNotSupported},
		{name: "delete lookup error", agent: admin, call: del, wantErr: errGuardDownstream, setup: getErr},
		{name: "delete foreign tenant", agent: admin, call: del, wantErr: serviceerrors.ErrPermissionDenied, setup: getOK(foreign)},
		{name: "delete downstream error", agent: admin, call: del, wantErr: errGuardDownstream, setup: func(m *requesthandler.MockRequestHandler) {
			getOK(spk(tmspeaking.StatusStopped))(m)
			m.EXPECT().TTSV1SpeakingDelete(gomock.Any(), guardResourceID).Return(nil, errGuardDownstream)
		}},
		{name: "delete continues when stopping the active session fails", agent: admin, call: del, wantErr: nil, setup: func(m *requesthandler.MockRequestHandler) {
			getOK(spk(tmspeaking.StatusActive))(m)
			m.EXPECT().TTSV1SpeakingStop(gomock.Any(), podID, guardResourceID).Return(nil, errGuardDownstream)
			m.EXPECT().TTSV1SpeakingDelete(gomock.Any(), guardResourceID).Return(&tmspeaking.Speaking{}, nil)
		}},
	}

	runGuardCases(t, tests)
}

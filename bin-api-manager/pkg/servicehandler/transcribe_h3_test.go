package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
)

func h3Transcribe() *tmtranscribe.Transcribe {
	return &tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
		HostID:   uuid.FromStringOrNil("12121212-0000-0000-0000-000000000012"),
	}
}

func h3TranscribeLookup(m *h3Mock, ctx context.Context) {
	m.EXPECT().TranscribeV1TranscribeGet(ctx, h3ResourceID).Return(h3Transcribe(), nil)
}

func h3TranscribeLookupFail(m *h3Mock, ctx context.Context) {
	m.EXPECT().TranscribeV1TranscribeGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_TranscribeGet_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[*tmtranscribe.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error) {
			return h.TranscribeGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(h3TranscribeLookup),
		opFail: h3TranscribeLookupFail,
		opOK:   h3TranscribeLookup,
		want:   h3Transcribe().ConvertWebhookMessage(),
	}))
}

func Test_TranscribeStopDelete_h3(t *testing.T) {
	cases := []struct {
		name   string
		call   func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error)
		opFail func(m *h3Mock, ctx context.Context)
		opOK   func(m *h3Mock, ctx context.Context)
	}{
		{
			"TranscribeStop",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error) {
				return h.TranscribeStop(ctx, a, h3ResourceID)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TranscribeV1TranscribeStop(ctx, h3Transcribe().HostID, h3ResourceID).Return(nil, errH3)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TranscribeV1TranscribeStop(ctx, h3Transcribe().HostID, h3ResourceID).Return(h3Transcribe(), nil)
			},
		},
		{
			"TranscribeDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error) {
				return h.TranscribeDelete(ctx, a, h3ResourceID)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TranscribeV1TranscribeDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TranscribeV1TranscribeDelete(ctx, h3ResourceID).Return(h3Transcribe(), nil)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h3RunRes(t, h3GuardRows(h3Ops[*tmtranscribe.WebhookMessage]{
				call:    c.call,
				a:       h3Admin(),
				denied:  h3OwnedDenied(h3TranscribeLookup),
				getFail: h3TranscribeLookupFail,
				opPre:   h3TranscribeLookup,
				opFail:  c.opFail,
				opOK:    c.opOK,
				want:    h3Transcribe().ConvertWebhookMessage(),
			}))
		})
	}
}

func Test_TranscribeList_guards_h3(t *testing.T) {
	refID := uuid.FromStringOrNil("13131313-0000-0000-0000-000000000013")
	filters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:    h3CustomerID,
		tmtranscribe.FieldDeleted:       false,
		tmtranscribe.FieldReferenceType: "call",
		tmtranscribe.FieldReferenceID:   refID,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*tmtranscribe.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tmtranscribe.WebhookMessage, error) {
			return h.TranscribeList(ctx, a, 10, "tok", "call", refID)
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *h3Mock, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscribeList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *h3Mock, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscribeList(ctx, "tok", uint64(10), filters).Return([]tmtranscribe.Transcribe{*h3Transcribe()}, nil)
		},
		want: []*tmtranscribe.WebhookMessage{h3Transcribe().ConvertWebhookMessage()},
	}))
}

func Test_TranscribeStart_h3(t *testing.T) {
	refID := uuid.FromStringOrNil("13131313-0000-0000-0000-000000000013")
	confbridgeID := uuid.FromStringOrNil("14141414-0000-0000-0000-000000000014")
	endFlow := uuid.FromStringOrNil("15151515-0000-0000-0000-000000000015")
	callFn := func(refType string) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tmtranscribe.WebhookMessage, error) {
			return h.TranscribeStart(ctx, a, h3ResourceID, refType, refID, "en-US", tmtranscribe.DirectionBoth, endFlow, tmtranscribe.ProviderGCP)
		}
	}
	start := func(m *h3Mock, ctx context.Context, refType tmtranscribe.ReferenceType, id uuid.UUID) *h3GomockCall {
		return m.EXPECT().TranscribeV1TranscribeStart(ctx, h3ResourceID, h3CustomerID, uuid.Nil, endFlow, refType, id, "en-US", tmtranscribe.DirectionBoth, tmtranscribe.ProviderGCP, 60000)
	}
	ownCall := &cmcall.Call{Identity: commonidentity.Identity{ID: refID, CustomerID: h3CustomerID}}
	otherCall := &cmcall.Call{Identity: commonidentity.Identity{ID: refID, CustomerID: h3OtherCustomerID}}
	ownConf := &cfconference.Conference{Identity: commonidentity.Identity{ID: refID, CustomerID: h3CustomerID}, ConfbridgeID: confbridgeID}
	otherConf := &cfconference.Conference{Identity: commonidentity.Identity{ID: refID, CustomerID: h3OtherCustomerID}, ConfbridgeID: confbridgeID}
	ownRec := &cmrecording.Recording{Identity: commonidentity.Identity{ID: refID, CustomerID: h3CustomerID}}
	otherRec := &cmrecording.Recording{Identity: commonidentity.Identity{ID: refID, CustomerID: h3OtherCustomerID}}
	result := h3Transcribe()
	want := result.ConvertWebhookMessage()

	h3RunRes(t, []h3ResRow[*tmtranscribe.WebhookMessage]{
		{name: "direct", a: h3Direct(), call: callFn("call"), wantErr: h3ErrDirect},
		{name: "no permission", a: h3NoPerm(), call: callFn("call"), wantErr: h3ErrDenied},
		{name: "unsupported reference type", a: h3Admin(), call: callFn("flow"), wantErr: serviceerrors.ErrInvalidArgument},
		{
			name: "call lookup error", a: h3Admin(), call: callFn("call"), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { m.EXPECT().CallV1CallGet(ctx, refID).Return(nil, errH3) },
		},
		{
			name: "call of another customer", a: h3Admin(), call: callFn("call"), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) { m.EXPECT().CallV1CallGet(ctx, refID).Return(otherCall, nil) },
		},
		{
			name: "conference lookup error", a: h3Admin(), call: callFn("conference"), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceGet(ctx, refID).Return(nil, errH3)
			},
		},
		{
			name: "conference of another customer", a: h3Admin(), call: callFn("conference"), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceGet(ctx, refID).Return(otherConf, nil)
			},
		},
		{
			name: "recording lookup error", a: h3Admin(), call: callFn("recording"), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { m.EXPECT().CallV1RecordingGet(ctx, refID).Return(nil, errH3) },
		},
		{
			name: "recording of another customer", a: h3Admin(), call: callFn("recording"), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) { m.EXPECT().CallV1RecordingGet(ctx, refID).Return(otherRec, nil) },
		},
		{
			name: "start rpc error", a: h3Admin(), call: callFn("call"), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, refID).Return(ownCall, nil)
				start(m, ctx, tmtranscribe.ReferenceTypeCall, refID).Return(nil, errH3)
			},
		},
		{
			name: "call reference", a: h3Admin(), call: callFn("call"), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().CallV1CallGet(ctx, refID).Return(ownCall, nil)
				start(m, ctx, tmtranscribe.ReferenceTypeCall, refID).Return(result, nil)
			},
		},
		{
			name: "conference reference is mapped to its confbridge", a: h3Admin(), call: callFn("conference"), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceGet(ctx, refID).Return(ownConf, nil)
				start(m, ctx, tmtranscribe.ReferenceTypeConfbridge, confbridgeID).Return(result, nil)
			},
		},
		{
			name: "recording reference", a: h3Admin(), call: callFn("recording"), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().CallV1RecordingGet(ctx, refID).Return(ownRec, nil)
				start(m, ctx, tmtranscribe.ReferenceTypeRecording, refID).Return(result, nil)
			},
		},
	})
}

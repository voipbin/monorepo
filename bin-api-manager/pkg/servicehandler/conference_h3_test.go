package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gofrs/uuid"
)

func h3Conf() *cfconference.Conference {
	return &cfconference.Conference{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
		Name:     "h3 conf",
	}
}

func h3ConfLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().ConferenceV1ConferenceGet(ctx, h3ResourceID).Return(h3Conf(), nil)
}

func h3ConfLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().ConferenceV1ConferenceGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_ConferenceOps_guards_h3(t *testing.T) {
	type call = func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error)
	name := "n"
	cases := []struct {
		name   string
		call   call
		opFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
		opOK   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	}{
		{
			"ConferenceGet",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceGet(ctx, a, h3ResourceID)
			},
			nil, nil,
		},
		{
			"ConferenceDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceDelete(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceDelete(ctx, h3ResourceID).Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceUpdate",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceUpdate(ctx, a, h3ResourceID, &name, nil, nil, nil, nil, nil)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceUpdate(ctx, h3ResourceID, &name, nil, nil, nil, nil, nil).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceUpdate(ctx, h3ResourceID, &name, nil, nil, nil, nil, nil).Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceRecordingStart",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceRecordingStart(ctx, a, h3ResourceID, cmrecording.FormatWAV, 30, uuid.Nil)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceRecordingStart(ctx, h3ResourceID, uuid.Nil, cmrecording.FormatWAV, 30, uuid.Nil).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceRecordingStart(ctx, h3ResourceID, uuid.Nil, cmrecording.FormatWAV, 30, uuid.Nil).Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceRecordingStop",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceRecordingStop(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceRecordingStop(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceRecordingStop(ctx, h3ResourceID).Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceTranscribeStart",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceTranscribeStart(ctx, a, h3ResourceID, "en-US")
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceTranscribeStart(ctx, h3ResourceID, "en-US").Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceTranscribeStart(ctx, h3ResourceID, "en-US").Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceTranscribeStop",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceTranscribeStop(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceTranscribeStop(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceTranscribeStop(ctx, h3ResourceID).Return(h3Conf(), nil)
			},
		},
		{
			"ConferenceDirectHashRegenerate",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
				return h.ConferenceDirectHashRegenerate(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceDirectHashRegenerate(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().ConferenceV1ConferenceDirectHashRegenerate(ctx, h3ResourceID).Return(h3Conf(), nil)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := h3Ops[*cfconference.WebhookMessage]{
				call:    c.call,
				a:       h3Admin(),
				denied:  h3OwnedDenied(h3ConfLookup),
				getFail: h3ConfLookupFail,
				opPre:   h3ConfLookup,
				opFail:  c.opFail,
				opOK:    c.opOK,
				want:    h3Conf().ConvertWebhookMessage(),
			}
			if c.opFail == nil {
				// ConferenceGet has no op rpc after the lookup: the lookup itself is the op.
				o.opPre = nil
				o.opFail = h3ConfLookupFail
				o.opOK = h3ConfLookup
			}
			h3RunRes(t, h3GuardRows(o))
		})
	}
}

func Test_ConferenceList_h3(t *testing.T) {
	listCall := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*cfconference.WebhookMessage, error) {
		return h.ConferenceList(ctx, a, 10, "tok")
	}
	filters := map[cfconference.Field]any{
		cfconference.FieldCustomerID: h3CustomerID,
		cfconference.FieldDeleted:    false,
		cfconference.FieldType:       "conference",
	}
	o := h3Ops[[]*cfconference.WebhookMessage]{
		call:   listCall,
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().ConferenceV1ConferenceList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().ConferenceV1ConferenceList(ctx, "tok", uint64(10), filters).Return([]cfconference.Conference{*h3Conf()}, nil)
		},
		want: []*cfconference.WebhookMessage{h3Conf().ConvertWebhookMessage()},
	}
	h3RunRes(t, h3GuardRows(o))
}

func Test_ConferenceCreate_h3(t *testing.T) {
	pre, post := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000011"), uuid.FromStringOrNil("11111111-0000-0000-0000-000000000012")
	data := map[string]any{"k": "v"}
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cfconference.WebhookMessage, error) {
		return h.ConferenceCreate(ctx, a, h3ResourceID, cfconference.TypeConference, "n", "d", data, 60, pre, post)
	}
	o := h3Ops[*cfconference.WebhookMessage]{
		call:   call,
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().ConferenceV1ConferenceCreate(ctx, h3ResourceID, h3CustomerID, cfconference.TypeConference, "n", "d", data, 60, pre, post).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().ConferenceV1ConferenceCreate(ctx, h3ResourceID, h3CustomerID, cfconference.TypeConference, "n", "d", data, 60, pre, post).Return(h3Conf(), nil)
		},
		want: h3Conf().ConvertWebhookMessage(),
	}
	h3RunRes(t, h3GuardRows(o))
}

func Test_ConferenceMediaStreamStart_guards_h3(t *testing.T) {
	op := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) error {
		return h.ConferenceMediaStreamStart(ctx, a, h3ResourceID, "audiosocket", nil, nil)
	}
	h3Run(t, []h3Row{
		{name: "direct", a: h3Direct(), call: op, wantErr: h3ErrDirect},
		{name: "lookup error", a: h3Admin(), call: op, wantErr: errH3, setup: h3ConfLookupFail},
		{name: "cross tenant", a: h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin), call: op, wantErr: h3ErrDenied, setup: h3ConfLookup},
		{name: "no permission", a: h3NoPerm(), call: op, wantErr: h3ErrDenied, setup: h3ConfLookup},
	})
}

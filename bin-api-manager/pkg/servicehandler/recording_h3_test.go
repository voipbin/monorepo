package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

func h3Recording() *cmrecording.Recording {
	return &cmrecording.Recording{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
	}
}

func h3RecordingLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().CallV1RecordingGet(ctx, h3ResourceID).Return(h3Recording(), nil)
}

func h3RecordingLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().CallV1RecordingGet(ctx, h3ResourceID).Return(nil, errH3)
}

func h3RecordingDeleted(m *requesthandler.MockRequestHandler, ctx context.Context) {
	rec := h3Recording()
	rec.TMDelete = timePtr("2024-01-01T00:00:00.000000Z")
	m.EXPECT().CallV1RecordingGet(ctx, h3ResourceID).Return(rec, nil)
}

func Test_RecordingGet_guards_h3(t *testing.T) {
	o := h3Ops[*cmrecording.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmrecording.WebhookMessage, error) {
			return h.RecordingGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(h3RecordingLookup),
		opFail: h3RecordingLookupFail,
		opOK:   h3RecordingLookup,
		want:   h3Recording().ConvertWebhookMessage(),
	}
	h3RunRes(t, append(h3GuardRows(o), h3ResRow[*cmrecording.WebhookMessage]{
		name: "deleted recording is not found", a: h3Admin(), call: o.call, setup: h3RecordingDeleted, wantErr: serviceerrors.ErrNotFound,
	}))
}

func Test_RecordingDelete_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[*cmrecording.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*cmrecording.WebhookMessage, error) {
			return h.RecordingDelete(ctx, a, h3ResourceID)
		},
		a:       h3Admin(),
		denied:  h3OwnedDenied(h3RecordingLookup),
		getFail: h3RecordingLookupFail,
		opPre:   h3RecordingLookup,
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1RecordingDelete(ctx, h3ResourceID).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1RecordingDelete(ctx, h3ResourceID).Return(h3Recording(), nil)
		},
		want: h3Recording().ConvertWebhookMessage(),
	}))
}

func Test_RecordingList_guards_h3(t *testing.T) {
	filters := map[cmrecording.Field]any{
		cmrecording.FieldCustomerID: h3CustomerID,
		cmrecording.FieldDeleted:    false,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*cmrecording.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*cmrecording.WebhookMessage, error) {
			return h.RecordingList(ctx, a, 10, "tok")
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1RecordingList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().CallV1RecordingList(ctx, "tok", uint64(10), filters).Return([]cmrecording.Recording{*h3Recording()}, nil)
		},
		want: []*cmrecording.WebhookMessage{h3Recording().ConvertWebhookMessage()},
	}))
}

func Test_RecordingPlayfilesGet_guards_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]openapi_server.ApiManagerRecordingPlayfile, error) {
		return h.RecordingPlayfilesGet(ctx, a, h3ResourceID)
	}
	fileFilters := map[smfile.Field]any{
		smfile.FieldReferenceType: smfile.ReferenceTypeRecording,
		smfile.FieldReferenceID:   h3ResourceID,
		smfile.FieldDeleted:       false,
	}
	h3RunRes(t, []h3ResRow[[]openapi_server.ApiManagerRecordingPlayfile]{
		{name: "direct", a: h3Direct(), call: call, wantErr: h3ErrDirect},
		{name: "lookup error", a: h3Admin(), call: call, wantErr: errH3, setup: h3RecordingLookupFail},
		{name: "cross tenant", a: h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin), call: call, wantErr: h3ErrDenied, setup: h3RecordingLookup},
		{name: "no permission", a: h3NoPerm(), call: call, wantErr: h3ErrDenied, setup: h3RecordingLookup},
		{
			name: "file list error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				h3RecordingLookup(m, ctx)
				m.EXPECT().StorageV1FileList(ctx, "", uint64(100), fileFilters).Return(nil, errH3)
			},
		},
	})
}

func Test_RecordingTranscribeList_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tmtranscribe.WebhookMessage, error) {
		return h.RecordingTranscribeList(ctx, a, h3ResourceID, 10, "tok")
	}
	tr := &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004"), CustomerID: h3CustomerID}}
	filters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldReferenceType: "recording",
		tmtranscribe.FieldReferenceID:   h3ResourceID,
		tmtranscribe.FieldDeleted:       false,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*tmtranscribe.WebhookMessage]{
		call:    call,
		a:       h3Admin(),
		denied:  h3OwnedDenied(h3RecordingLookup),
		getFail: h3RecordingLookupFail,
		opPre:   h3RecordingLookup,
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscribeList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscribeList(ctx, "tok", uint64(10), filters).Return([]tmtranscribe.Transcribe{*tr}, nil)
		},
		want: []*tmtranscribe.WebhookMessage{tr.ConvertWebhookMessage()},
	}))
}

func Test_RecordingTranscriptList_h3(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tmtranscript.WebhookMessage, error) {
		return h.RecordingTranscriptList(ctx, a, h3ResourceID, transcribeID, 10, "tok")
	}
	ownTranscribe := &tmtranscribe.Transcribe{
		Identity:      commonidentity.Identity{ID: transcribeID, CustomerID: h3CustomerID},
		ReferenceType: tmtranscribe.ReferenceTypeRecording,
		ReferenceID:   h3ResourceID,
	}
	transcribeLookup := func(tr *tmtranscribe.Transcribe) func(m *requesthandler.MockRequestHandler, ctx context.Context) {
		return func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			h3RecordingLookup(m, ctx)
			m.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(tr, nil)
		}
	}
	mismatchRef := *ownTranscribe
	mismatchRef.ReferenceID = uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")
	mismatchType := *ownTranscribe
	mismatchType.ReferenceType = tmtranscribe.ReferenceTypeCall
	ts := tmtranscript.Transcript{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006"), CustomerID: h3CustomerID}, TranscribeID: transcribeID}
	filters := map[tmtranscript.Field]any{
		tmtranscript.FieldTranscribeID: transcribeID,
		tmtranscript.FieldDeleted:      false,
	}

	rows := h3GuardRows(h3Ops[[]*tmtranscript.WebhookMessage]{
		call:    call,
		a:       h3Admin(),
		denied:  h3OwnedDenied(h3RecordingLookup),
		getFail: h3RecordingLookupFail,
		opPre:   transcribeLookup(ownTranscribe),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscriptList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().TranscribeV1TranscriptList(ctx, "tok", uint64(10), filters).Return([]tmtranscript.Transcript{ts}, nil)
		},
		want: []*tmtranscript.WebhookMessage{ts.ConvertWebhookMessage()},
	})
	rows = append(rows,
		h3ResRow[[]*tmtranscript.WebhookMessage]{
			name: "transcribe lookup error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				h3RecordingLookup(m, ctx)
				m.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(nil, errH3)
			},
		},
		h3ResRow[[]*tmtranscript.WebhookMessage]{
			name: "transcribe of another recording is denied", a: h3Admin(), call: call, wantErr: h3ErrDenied,
			setup: transcribeLookup(&mismatchRef),
		},
		h3ResRow[[]*tmtranscript.WebhookMessage]{
			name: "transcribe of another reference type is denied", a: h3Admin(), call: call, wantErr: h3ErrDenied,
			setup: transcribeLookup(&mismatchType),
		},
	)
	h3RunRes(t, rows)
}

func Test_fileDirection_playfileSortKey_h3(t *testing.T) {
	in, out, none := openapi_server.ApiManagerRecordingPlayfileDirectionIn, openapi_server.ApiManagerRecordingPlayfileDirectionOut, openapi_server.ApiManagerRecordingPlayfileDirectionNone
	dirTests := []struct {
		name     string
		filename string
		eligible bool
		want     openapi_server.ApiManagerRecordingPlayfileDirection
	}{
		{"not eligible", "a_in.wav", false, none},
		{"in", "a_in.wav", true, in},
		{"out", "a_out.wav", true, out},
		{"mixed", "a.wav", true, none},
	}
	for _, tt := range dirTests {
		t.Run("fileDirection/"+tt.name, func(t *testing.T) {
			if got := fileDirection(tt.filename, tt.eligible); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}

	keyTests := []struct {
		name string
		p    openapi_server.ApiManagerRecordingPlayfile
		want int
	}{
		{"nil direction", openapi_server.ApiManagerRecordingPlayfile{}, 2},
		{"in", openapi_server.ApiManagerRecordingPlayfile{Direction: &in}, 0},
		{"out", openapi_server.ApiManagerRecordingPlayfile{Direction: &out}, 1},
		{"none", openapi_server.ApiManagerRecordingPlayfile{Direction: &none}, 2},
	}
	for _, tt := range keyTests {
		t.Run("playfileSortKey/"+tt.name, func(t *testing.T) {
			if got := playfileSortKey(tt.p); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}

package servicehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
)

func h3File(customerID, ownerID uuid.UUID) *smfile.File {
	return &smfile.File{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: ownerID},
		Filename: "h3.wav",
	}
}

func h3FileRPC(file *smfile.File, err error) func(m *h3Mock, ctx context.Context) {
	return func(m *h3Mock, ctx context.Context) { m.EXPECT().StorageV1FileGet(ctx, h3ResourceID).Return(file, err) }
}

func Test_ServiceAgentFileCreate_guards_h3(t *testing.T) {
	// the storage client is not configured on the handler: the authentication guard must reject before it is touched.
	call := func(a *auth.AuthIdentity) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*smfile.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*smfile.WebhookMessage, error) {
			return h.ServiceAgentFileCreate(ctx, a, nil, smfile.Type("h3"), "n", "d", "f.wav")
		}
	}
	h3RunRes(t, []h3ResRow[*smfile.WebhookMessage]{
		{name: "non-agent is unauthenticated", a: h3Direct(), call: call(nil), wantErr: serviceerrors.ErrAuthenticationRequired},
	})
}

func Test_ServiceAgentFileGet_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*smfile.WebhookMessage, error) {
		return h.ServiceAgentFileGet(ctx, a, h3ResourceID)
	}
	own := h3File(h3CustomerID, h3AgentID)
	deleted := h3File(h3CustomerID, h3AgentID)
	deleted.TMDelete = timePtr("2024-01-01T00:00:00.000000Z")
	h3RunRes(t, []h3ResRow[*smfile.WebhookMessage]{
		{name: "non-agent is unauthenticated", a: h3Direct(), call: call, wantErr: serviceerrors.ErrAuthenticationRequired},
		{name: "lookup error", a: h3Admin(), call: call, wantErr: errH3, setup: h3FileRPC(nil, errH3)},
		{name: "deleted file is not found", a: h3Admin(), call: call, wantErr: serviceerrors.ErrNotFound, setup: h3FileRPC(deleted, nil)},
		{name: "file of another customer", a: h3Agent(h3OtherCustomerID, 0), call: call, wantErr: h3ErrDenied, setup: h3FileRPC(own, nil)},
		{name: "success", a: h3Admin(), call: call, want: own.ConvertWebhookMessage(), setup: h3FileRPC(own, nil)},
	})
}

func Test_ServiceAgentFileDelete_guards_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*smfile.WebhookMessage, error) {
		return h.ServiceAgentFileDelete(ctx, a, h3ResourceID)
	}
	own := h3File(h3CustomerID, h3AgentID)
	foreignOwner := h3File(h3CustomerID, h3TalkOtherID)
	h3RunRes(t, []h3ResRow[*smfile.WebhookMessage]{
		{name: "non-agent is unauthenticated", a: h3Direct(), call: call, wantErr: serviceerrors.ErrAuthenticationRequired},
		{name: "lookup error", a: h3Admin(), call: call, wantErr: errH3, setup: h3FileRPC(nil, errH3)},
		{name: "only the owner may delete", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: h3FileRPC(foreignOwner, nil)},
		{
			name: "delete rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3FileRPC(own, nil)(m, ctx)
				m.EXPECT().StorageV1FileDelete(ctx, h3ResourceID, 60000).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: own.ConvertWebhookMessage(),
			setup: func(m *h3Mock, ctx context.Context) {
				h3FileRPC(own, nil)(m, ctx)
				m.EXPECT().StorageV1FileDelete(ctx, h3ResourceID, 60000).Return(own, nil)
			},
		},
	})
}

func Test_ServiceAgentFileList_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*smfile.WebhookMessage, error) {
		return h.ServiceAgentFileList(ctx, a, 10, "tok")
	}
	filters := map[smfile.Field]any{
		smfile.FieldCustomerID: h3CustomerID,
		smfile.FieldDeleted:    false,
		smfile.FieldOwnerID:    h3AgentID,
	}
	own := h3File(h3CustomerID, h3AgentID)
	h3RunRes(t, []h3ResRow[[]*smfile.WebhookMessage]{
		{name: "non-agent is unauthenticated", a: h3Direct(), call: call, wantErr: serviceerrors.ErrAuthenticationRequired},
		{
			name: "rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().StorageV1FileList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
			},
		},
		{
			name: "success lists only the caller's own files", a: h3Admin(), call: call, want: []*smfile.WebhookMessage{own.ConvertWebhookMessage()},
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().StorageV1FileList(ctx, "tok", uint64(10), filters).Return([]smfile.File{*own}, nil)
			},
		},
	})
}

func Test_ServiceAgentFileDownloadRedirect_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (string, error) {
		return h.ServiceAgentFileDownloadRedirect(ctx, a, h3ResourceID)
	}
	expired := h3File(h3CustomerID, h3AgentID)
	past := time.Now().Add(-time.Hour)
	expired.TMDownloadExpire = &past
	expired.URIDownload = "https://example.com/old"
	h3RunRes(t, []h3ResRow[string]{
		{name: "non-agent is unauthenticated", a: h3Direct(), call: call, wantErr: serviceerrors.ErrAuthenticationRequired},
		{name: "lookup error", a: h3Admin(), call: call, wantErr: errH3, setup: h3FileRPC(nil, errH3)},
		{name: "file of another customer", a: h3Agent(h3OtherCustomerID, 0), call: call, wantErr: h3ErrDenied, setup: h3FileRPC(expired, nil)},
		{
			name: "refresh error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3FileRPC(expired, nil)(m, ctx)
				m.EXPECT().StorageV1FileDownloadURIRefresh(ctx, h3ResourceID).Return("", errH3)
			},
		},
		{
			name: "expired uri is refreshed", a: h3Admin(), call: call, want: "https://example.com/new",
			setup: func(m *h3Mock, ctx context.Context) {
				h3FileRPC(expired, nil)(m, ctx)
				m.EXPECT().StorageV1FileDownloadURIRefresh(ctx, h3ResourceID).Return("https://example.com/new", nil)
			},
		},
	})
}

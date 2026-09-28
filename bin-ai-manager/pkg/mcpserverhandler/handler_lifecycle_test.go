package mcpserverhandler

import (
	"errors"
	"testing"
	"time"

	"context"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
)

// deletedServer is a soft-deleted row: the READ path deliberately still
// returns it (GET after DELETE answers 200), which is exactly why the write
// paths cannot rely on Get alone to refuse.
func deletedServer(id, customerID uuid.UUID) *mcpserver.McpServer {
	ts := time.Now()
	return &mcpserver.McpServer{
		Identity: identityFor(id, customerID),
		Status:   mcpserver.StatusActive,
		TMDelete: &ts,
	}
}

// Test_Update_DeletedServerIsNotMutable covers D2/D20: a soft-deleted server
// used to remain fully editable, and an empty PUT bypassed every gate by
// short-circuiting to Get() before any write was attempted.
//
// The three cases are the three distinct code paths through Update, all of
// which must now refuse BEFORE deciding anything else:
//   - a normal field write (must not reach McpServerUpdate),
//   - an empty PUT (must not short-circuit to a 200),
//   - a malformed URL (must report the missing resource, not the bad input:
//     validation must not get to decide the status code for a row that no
//     longer exists).
func Test_Update_DeletedServerIsNotMutable(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string

		fieldName *string
		fieldURL  *string
	}{
		{
			name:      "a field write is refused",
			fieldName: strPtr("new name"),
		},
		{
			name: "an empty PUT is refused instead of short-circuiting to 200",
		},
		{
			name:     "a malformed URL reports the missing server, not the bad URL",
			fieldURL: strPtr("not-a-url"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			// Exactly once: the existence gate reads the row, sees tm_delete and
			// stops. No McpServerUpdate and no webhook are expected at all, so an
			// unexpected call to either fails the test.
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).
				Return(deletedServer(id, customerID), nil)

			_, err := h.Update(context.Background(), id,
				tt.fieldName, nil, tt.fieldURL, nil, nil, nil, nil)
			if err == nil {
				t.Fatalf("expected an error for a soft-deleted server, got nil")
			}
			if !isNotFound(err) {
				t.Errorf("expected a NotFound error, got: %v", err)
			}
		})
	}
}

// Test_Update_OAuthAuthTypeIsNotDirectlySettable covers D17. `oauth` is a
// state the OAuth flow produces; accepting it on a PUT yields a server whose
// auth_type promises a bearer token it has no way to obtain, and the published
// docs enumerate only "", "bearer" and "api_key" as settable.
func Test_Update_OAuthAuthTypeIsNotDirectlySettable(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

	// The existence gate runs first and succeeds; the auth_type refusal is what
	// must stop the write. No McpServerUpdate is expected.
	mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(&mcpserver.McpServer{
		Identity: identityFor(id, customerID),
		Status:   mcpserver.StatusActive,
	}, nil)

	_, err := h.Update(context.Background(), id,
		nil, nil, nil, nil, authTypePtr(mcpserver.AuthTypeOAuth), nil, nil)
	if err == nil {
		t.Fatalf("expected auth_type \"oauth\" to be refused on a direct update")
	}
}

// Test_Create_OAuthAuthTypeIsNotDirectlySettable is the create half of D17:
// gating only Update would leave the same violating row reachable by POST.
func Test_Create_OAuthAuthTypeIsNotDirectlySettable(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

	// No McpServerCreate is expected: the refusal precedes every write.
	_, err := h.Create(context.Background(), customerID,
		"name", "detail", "https://mcp.example.com/mcp",
		mcpserver.StatusActive, mcpserver.AuthTypeOAuth, "", "")
	if err == nil {
		t.Fatalf("expected auth_type \"oauth\" to be refused on create")
	}
}

// Test_Delete_IsIdempotentAndPublishesOnce covers D16. A repeat DELETE reaches
// no live row, which dbhandler now reports as ErrNotFound. The handler must
// still answer 200 -- DELETE is idempotent by contract and GET on a
// soft-deleted server answers 200, so 404ing only the second delete would be
// incoherent -- while NOT re-publishing the deleted event.
func Test_Delete_IsIdempotentAndPublishesOnce(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name        string
		deleteErr   error
		wantPublish bool
	}{
		{
			name:        "first delete publishes the event",
			deleteErr:   nil,
			wantPublish: true,
		},
		{
			name:        "repeat delete still returns the row but publishes nothing",
			deleteErr:   dbhandler.ErrNotFound,
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			mockDB.EXPECT().McpServerDelete(gomock.Any(), id).Return(tt.deleteErr)
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).
				Return(deletedServer(id, customerID), nil)
			if tt.wantPublish {
				mockNotify.EXPECT().PublishWebhookEvent(
					gomock.Any(), customerID, mcpserver.EventTypeDeleted, gomock.Any())
			}

			res, err := h.Delete(context.Background(), id)
			if err != nil {
				t.Fatalf("delete must stay idempotent, got error: %v", err)
			}
			if res == nil || res.ID != id {
				t.Errorf("expected the row to be returned, got: %v", res)
			}
		})
	}
}

// Test_Delete_PropagatesRealErrors guards the swallow above from widening: only
// ErrNotFound may be absorbed, because only that one means "already deleted".
// Any other failure must surface rather than be reported as a success.
func Test_Delete_PropagatesRealErrors(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

	// No read-back and no webhook are expected: the error must stop the flow.
	mockDB.EXPECT().McpServerDelete(gomock.Any(), id).
		Return(errors.New("connection refused"))

	if _, err := h.Delete(context.Background(), id); err == nil {
		t.Fatalf("expected a non-ErrNotFound delete failure to propagate")
	}
}

func isNotFound(err error) bool {
	var ve *commonerrors.VoipbinError
	if errors.As(err, &ve) {
		return ve.Status == commonerrors.StatusNotFound
	}
	return false
}

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

// Test_Update_OAuthAuthTypeTransition covers D17 on the update path. `oauth`
// is a state the OAuth flow produces, so MOVING a row into it directly would
// yield a server whose auth_type promises a bearer token it has no way to
// obtain.
//
// What the gate must NOT do is reject the value outright. square-admin builds
// the PUT body from form state rather than from a diff
// (mcpservers_detail.js:156 sends auth_type unconditionally, :91 hydrates it
// from the GET), so a value gate would 400 every rename, URL change, and
// enable/disable of an already-connected server -- permanently, with no way
// for the customer to avoid it. Hence the stored-row comparison, and hence the
// "stays editable" case below, which is the regression guard: it fails if the
// gate is ever narrowed back to the value.
func Test_Update_OAuthAuthTypeTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string

		storedAuthType  mcpserver.AuthType
		requestAuthType *mcpserver.AuthType

		wantRefused bool
	}{
		{
			// The transition the gate exists for.
			name:            "moving a bearer server into oauth is refused",
			storedAuthType:  mcpserver.AuthTypeBearer,
			requestAuthType: authTypePtr(mcpserver.AuthTypeOAuth),
			wantRefused:     true,
		},
		{
			name:            "moving a no-auth server into oauth is refused",
			storedAuthType:  mcpserver.AuthType(""),
			requestAuthType: authTypePtr(mcpserver.AuthTypeOAuth),
			wantRefused:     true,
		},
		{
			// The regression guard. Not a transition: the row is already
			// oauth and the request re-submits the same value, which is what
			// the admin UI does on every save.
			name:            "an already-connected oauth server stays editable",
			storedAuthType:  mcpserver.AuthTypeOAuth,
			requestAuthType: authTypePtr(mcpserver.AuthTypeOAuth),
			wantRefused:     false,
		},
		{
			// Leaving oauth is a legitimate downgrade: the customer is
			// replacing the connection with a static credential.
			name:            "moving an oauth server to bearer is allowed",
			storedAuthType:  mcpserver.AuthTypeOAuth,
			requestAuthType: authTypePtr(mcpserver.AuthTypeBearer),
			wantRefused:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			stored := &mcpserver.McpServer{
				Identity: identityFor(id, customerID),
				Status:   mcpserver.StatusActive,
				AuthType: tt.storedAuthType,
			}

			if tt.wantRefused {
				// Only the existence gate's read happens; the refusal
				// precedes the write.
				mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(stored, nil)
			} else {
				// Allowed: the write and its read-back must both happen, so a
				// gate that refused everything could not pass this case.
				mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(stored, nil).Times(2)
				mockDB.EXPECT().McpServerUpdate(gomock.Any(), id, gomock.Any()).Return(nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, mcpserver.EventTypeUpdated, gomock.Any())
			}

			res, err := h.Update(context.Background(), id,
				nil, nil, nil, nil, tt.requestAuthType, nil, nil)

			if !tt.wantRefused {
				if err != nil {
					t.Fatalf("expected the update to be allowed, got: %v", err)
				}
				if res == nil {
					t.Fatal("expected the updated server to be returned")
				}
				return
			}

			if err == nil {
				t.Fatalf("expected the transition into oauth to be refused")
			}
			// Assert the REASON, not just that something failed: this error
			// reaches the customer as an HTTP status and a code the admin UI
			// and the published docs both key off.
			assertInvalidArgument(t, err, "INVALID_MCP_SERVER_AUTH_TYPE")
		})
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
	assertInvalidArgument(t, err, "INVALID_MCP_SERVER_AUTH_TYPE")
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

// assertInvalidArgument pins both halves of what the customer actually
// receives: the status (which rpc.go maps to HTTP 400) and the reason code.
// Asserting only that some error occurred would let a refusal silently become
// a 404 or a 500 -- one misleads, the other pages an on-call engineer for a
// rejected request -- and would let the reason code, which square-admin and
// the published docs both key off, be renamed with no signal.
func assertInvalidArgument(t *testing.T, err error, wantReason string) {
	t.Helper()

	var ve *commonerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a VoipbinError, got %T: %v", err, err)
	}
	if ve.Status != commonerrors.StatusInvalidArgument {
		t.Errorf("expected status %v (HTTP 400), got %v", commonerrors.StatusInvalidArgument, ve.Status)
	}
	if ve.Reason != wantReason {
		t.Errorf("expected reason %q, got %q", wantReason, ve.Reason)
	}
}

// Test_Delete_ReadBackFailures covers the two paths the read-back after a
// delete can take when it fails. Neither had a test: Delete could be changed
// to return (nil, nil) -- success with no server -- and every caller would nil
// dereference res.ID on the very next line.
func Test_Delete_ReadBackFailures(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string

		getErr error

		// A vanished row is the customer's 404; anything else is a real
		// failure that must surface as one rather than being flattened.
		wantNotFound bool
	}{
		{
			name:         "a row that vanished between the delete and the read-back is a 404",
			getErr:       dbhandler.ErrNotFound,
			wantNotFound: true,
		},
		{
			name:         "a transport failure on the read-back propagates",
			getErr:       errors.New("connection refused"),
			wantNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			mockDB.EXPECT().McpServerDelete(gomock.Any(), id).Return(nil)
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(nil, tt.getErr)
			// Nothing was successfully deleted from the caller's point of
			// view, so no event may be published.
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res, err := h.Delete(context.Background(), id)
			if err == nil {
				t.Fatalf("expected the read-back failure to surface, got res=%v", res)
			}
			// The assertion that matters: a failure must not be reported as a
			// success with a nil server.
			if res != nil {
				t.Errorf("expected no server alongside the error, got %v", res)
			}
			if isNotFound(err) != tt.wantNotFound {
				t.Errorf("expected notFound=%v, got err=%v", tt.wantNotFound, err)
			}
		})
	}
}

package mcpserverhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// strPtr is a small helper so table entries can express nil vs. &"" vs.
// &"value" for the secret parameter without a separate named var per case.
func strPtr(s string) *string { return &s }

// newTestHandlerWithCrypto builds an *mcpServerHandler wired to real mocks
// plus a real *SecretCrypto (not mocked -- this suite verifies the actual
// encrypt/store/clear translation, which is the point of these tests).
func newTestHandlerWithCrypto(t *testing.T, db *dbhandler.MockDBHandler, notify *notifyhandler.MockNotifyHandler) *mcpServerHandler {
	t.Helper()
	crypto, err := NewSecretCrypto("1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	if err != nil {
		t.Fatalf("could not build test SecretCrypto: %v", err)
	}
	return &mcpServerHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notify,
		db:            db,
		crypto:        crypto,
	}
}

// Test_Update_SecretPointerSemantics pins the exact DB field values Update
// writes for each of the three *string secret states design §10 defines:
// nil (untouched -- no secret fields written at all), &"" (explicit clear --
// ciphertext/nonce/version all zeroed), and &"value" (replace -- re-encrypted
// and the new key version recorded). This is the property PR review flagged
// as untested at the handler layer (v1_mcpservers_test.go only asserts the
// listenhandler passes the right pointer into a MOCKED Update, never that
// Update itself does the right thing with it).
func Test_Update_SecretPointerSemantics(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name   string
		secret *string

		// wantSecretFieldsPresent asserts whether the three secret-related
		// fields appear in the map passed to McpServerUpdate at all.
		wantSecretFieldsPresent bool
		// wantCiphertextNil / wantVersionZero only checked when
		// wantSecretFieldsPresent is true.
		wantCiphertextNil bool
		wantVersionZero   bool
	}{
		{
			name:                    "nil secret leaves the encrypted secret untouched",
			secret:                  nil,
			wantSecretFieldsPresent: false,
		},
		{
			name:                    "pointer to empty string explicitly clears the secret",
			secret:                  strPtr(""),
			wantSecretFieldsPresent: true,
			wantCiphertextNil:       true,
			wantVersionZero:         true,
		},
		{
			name:                    "pointer to a non-empty value re-encrypts and replaces the secret",
			secret:                  strPtr("sk-new-secret-value"),
			wantSecretFieldsPresent: true,
			wantCiphertextNil:       false,
			wantVersionZero:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			var gotFields map[mcpserver.Field]any
			mockDB.EXPECT().McpServerUpdate(gomock.Any(), id, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[mcpserver.Field]any) error {
					gotFields = fields
					return nil
				},
			)
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(&mcpserver.McpServer{
				Identity: identityFor(id, customerID),
				Status:   mcpserver.StatusActive,
			}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, mcpserver.EventTypeUpdated, gomock.Any())

			_, err := h.Update(context.Background(), id, "name", "detail", "https://mcp.example.com/", mcpserver.StatusActive, mcpserver.AuthTypeNone, "", tt.secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, hasCiphertext := gotFields[mcpserver.FieldSecretCiphertext]
			_, hasNonce := gotFields[mcpserver.FieldSecretNonce]
			_, hasVersion := gotFields[mcpserver.FieldKeyVersion]

			if !tt.wantSecretFieldsPresent {
				if hasCiphertext || hasNonce || hasVersion {
					t.Errorf("expected no secret fields in the update when secret is nil, got fields: %v", gotFields)
				}
				return
			}

			if !hasCiphertext || !hasNonce || !hasVersion {
				t.Fatalf("expected all three secret fields present, got fields: %v", gotFields)
			}

			ciphertext, _ := gotFields[mcpserver.FieldSecretCiphertext].([]byte)
			version, _ := gotFields[mcpserver.FieldKeyVersion].(int)

			if tt.wantCiphertextNil && len(ciphertext) != 0 {
				t.Errorf("expected an explicit clear to zero the ciphertext, got %d bytes", len(ciphertext))
			}
			if !tt.wantCiphertextNil && len(ciphertext) == 0 {
				t.Errorf("expected a replace to write a non-empty ciphertext, got none")
			}
			if tt.wantVersionZero && version != 0 {
				t.Errorf("expected an explicit clear to zero the key version, got %d", version)
			}
			if !tt.wantVersionZero && version == 0 {
				t.Errorf("expected a replace to record a non-zero key version, got 0")
			}
		})
	}
}

// Test_Create_SecretEncryption pins that Create only encrypts and stores a
// secret when one is actually supplied -- an empty secret at creation time
// must leave the row with no ciphertext, not a ciphertext of the empty
// string.
func Test_Create_SecretEncryption(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name   string
		secret string

		wantCiphertextEmpty bool
	}{
		{name: "empty secret at creation stores no ciphertext", secret: "", wantCiphertextEmpty: true},
		{name: "non-empty secret at creation is encrypted and stored", secret: "sk-initial-secret", wantCiphertextEmpty: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)
			h.utilHandler = mockUtil

			newID := uuid.Must(uuid.NewV4())
			mockUtil.EXPECT().UUIDCreate().Return(newID)

			var created *mcpserver.McpServer
			mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m *mcpserver.McpServer) error {
					created = m
					return nil
				},
			)
			mockDB.EXPECT().McpServerGet(gomock.Any(), newID).DoAndReturn(
				func(_ context.Context, _ uuid.UUID) (*mcpserver.McpServer, error) {
					return created, nil
				},
			)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, mcpserver.EventTypeCreated, gomock.Any())

			_, err := h.Create(context.Background(), customerID, "name", "detail", "https://mcp.example.com/", mcpserver.StatusActive, mcpserver.AuthTypeNone, "", tt.secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			isEmpty := len(created.SecretCiphertext) == 0
			if isEmpty != tt.wantCiphertextEmpty {
				t.Errorf("expected ciphertext-empty=%v, got %v (len=%d)", tt.wantCiphertextEmpty, isEmpty, len(created.SecretCiphertext))
			}
		})
	}
}

// identityFor is a tiny fixture helper mirroring the shape
// pkg/aicallhandler's test file uses for the same purpose.
func identityFor(id, customerID uuid.UUID) identity.Identity {
	return identity.Identity{ID: id, CustomerID: customerID}
}

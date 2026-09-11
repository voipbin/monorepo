package mcpserverhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
)

// Test_Update_PartialFields pins the fix for the bug found during CPO
// review of PR #1285/#1290's follow-up work: every non-secret field used
// to be unconditionally written to the SQL UPDATE field map, so omitting
// name/detail/api_key_header silently wiped them, omitting url/status
// 400'd (ValidateURL("")/!status.IsValid()), and omitting auth_type
// silently set it to AuthTypeNone (a valid enum value) -- the worst case,
// since it silently disabled authentication with no error. See
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md.
//
// Table-driven over which pointers are nil (omitted) vs non-nil (set,
// including pointers to the zero value): asserts the exact set of keys
// present in the map passed to McpServerUpdate, so a regression that
// re-includes an omitted field is caught, not just "no error occurred".
func Test_Update_PartialFields(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string

		fieldName   *string
		fieldDetail *string
		fieldURL    *string
		fieldStatus *mcpserver.Status
		fieldAuth   *mcpserver.AuthType
		fieldAPIKey *string

		wantKeys []mcpserver.Field
	}{
		{
			name:      "all fields nil except name -> only FieldName present",
			fieldName: strPtr("renamed"),
			wantKeys:  []mcpserver.Field{mcpserver.FieldName},
		},
		{
			name:        "status nil -> no validation, FieldStatus absent (regression test for the 400-on-omission bug)",
			fieldName:   strPtr("name only plus url"),
			fieldURL:    strPtr("https://mcp.example.com/"),
			fieldStatus: nil,
			wantKeys:    []mcpserver.Field{mcpserver.FieldName, mcpserver.FieldURL},
		},
		{
			name:      "auth_type nil -> FieldAuthType absent (regression test for the silent-deauth bug -- the most severe case)",
			fieldName: strPtr("name only"),
			fieldAuth: nil,
			wantKeys:  []mcpserver.Field{mcpserver.FieldName},
		},
		{
			name:      "auth_type explicitly set to the zero value (AuthTypeNone) is a real, validated value, distinct from omitted",
			fieldAuth: authTypePtr(mcpserver.AuthTypeNone),
			wantKeys:  []mcpserver.Field{mcpserver.FieldAuthType},
		},
		{
			name:     "url nil alongside secret set -> secret-only rotation succeeds without resending url (the original bug report's literal repro case)",
			fieldURL: nil,
			wantKeys: []mcpserver.Field{mcpserver.FieldSecretCiphertext, mcpserver.FieldSecretNonce, mcpserver.FieldKeyVersion},
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

			var secret *string
			if tt.name == "url nil alongside secret set -> secret-only rotation succeeds without resending url (the original bug report's literal repro case)" {
				secret = strPtr("«redacted:new-secret»")
			}

			_, err := h.Update(context.Background(), id, tt.fieldName, tt.fieldDetail, tt.fieldURL, tt.fieldStatus, tt.fieldAuth, tt.fieldAPIKey, secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(gotFields) != len(tt.wantKeys) {
				t.Fatalf("expected %d fields %v, got %d fields: %v", len(tt.wantKeys), tt.wantKeys, len(gotFields), gotFields)
			}
			for _, k := range tt.wantKeys {
				if _, ok := gotFields[k]; !ok {
					t.Errorf("expected field %s present, got fields: %v", k, gotFields)
				}
			}
		})
	}
}

// Test_Update_AllFieldsOmitted_IsANoOp pins the len(fields)==0 short-circuit:
// a PUT with every field omitted must NOT call db.McpServerUpdate at all
// (asserted by simply not setting up an EXPECT() for it -- gomock's strict
// mode fails the test if it's called unexpectedly), and must return the
// current row via the same Get() path a GET request would use, reusing
// Get's existing ErrNotFound -> cerrors.NotFound mapping.
func Test_Update_AllFieldsOmitted_IsANoOp(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

	current := &mcpserver.McpServer{
		Identity: identityFor(id, customerID),
		Status:   mcpserver.StatusActive,
	}
	// db.McpServerGet is expected exactly once (via the Get() fallback);
	// db.McpServerUpdate and notifyHandler.PublishWebhookEvent are NOT
	// expected at all -- an unexpected call to either fails the test.
	mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(current, nil)

	res, err := h.Update(context.Background(), id, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != current {
		t.Errorf("expected the current row to be returned unchanged, got: %v", res)
	}
}

package dbhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_InteractionCreate_And_InteractionGet_CaseID verifies Task 3.4's
// case_id linkage column round-trips through InteractionCreate/Get via
// the reflection-based PrepareFields/ScanRow mapping (same mechanism
// proven for Conversation.Metadata in Phase 1 Task 1.3) -- no explicit
// dbhandler scan/prepare code needed, just the db tag on the model.
func Test_InteractionCreate_And_InteractionGet_CaseID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000001")
	tm := timePtr(time.Date(2026, 6, 28, 15, 0, 0, 0, time.UTC))

	withCase := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000002"),
		CustomerID:    uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000003"),
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+15551200001",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000004"),
		CaseID:        &caseID,
		TMCreate:      tm,
	}
	if err := h.InteractionCreate(ctx, withCase); err != nil {
		t.Fatalf("InteractionCreate(withCase) error = %v", err)
	}

	withoutCase := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000005"),
		CustomerID:    uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000003"),
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+15551200002",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("f1b2c3d4-8002-8002-8002-000000000006"),
		TMCreate:      tm,
	}
	if err := h.InteractionCreate(ctx, withoutCase); err != nil {
		t.Fatalf("InteractionCreate(withoutCase) error = %v", err)
	}

	resWith, err := h.InteractionGet(ctx, withCase.ID)
	if err != nil {
		t.Fatalf("InteractionGet(withCase) error = %v", err)
	}
	if resWith.CaseID == nil || *resWith.CaseID != caseID {
		t.Errorf("expected case_id: %v, got: %v", caseID, resWith.CaseID)
	}

	resWithout, err := h.InteractionGet(ctx, withoutCase.ID)
	if err != nil {
		t.Fatalf("InteractionGet(withoutCase) error = %v", err)
	}
	if resWithout.CaseID != nil {
		t.Errorf("expected nil case_id, got: %v", *resWithout.CaseID)
	}
}

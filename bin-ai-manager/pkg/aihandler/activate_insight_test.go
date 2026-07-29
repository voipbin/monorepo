package aihandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_ActivateInsight_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	aiID := uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000002")

	activated := &ai.AI{
		Identity:        identity.Identity{ID: aiID, CustomerID: customerID},
		Type:            ai.TypeInsight,
		IsInsightActive: true,
	}

	mockDB.EXPECT().AIActivateInsight(gomock.Any(), aiID).Return(activated, nil, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, ai.EventTypeUpdated, activated).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	res, err := h.ActivateInsight(context.Background(), aiID)
	if err != nil {
		t.Fatalf("expected ok, got: %v", err)
	}
	if !res.IsInsightActive {
		t.Errorf("returned ai: IsInsightActive = false, want true")
	}
}

// A swap deactivates a second row. That row's is_insight_active flipped
// true -> false, so it MUST get its own ai_updated webhook -- otherwise every
// external subscriber keeps a stale is_insight_active=true for it.
func Test_ActivateInsight_PublishesUpdatedForDeactivatedAI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	customerID := uuid.FromStringOrNil("c5c5c5c5-0000-0000-0000-000000000001")
	activatedID := uuid.FromStringOrNil("c5c5c5c5-0000-0000-0000-000000000002")
	previousID := uuid.FromStringOrNil("c5c5c5c5-0000-0000-0000-000000000003")

	activated := &ai.AI{
		Identity:        identity.Identity{ID: activatedID, CustomerID: customerID},
		Type:            ai.TypeInsight,
		IsInsightActive: true,
	}
	previous := &ai.AI{
		Identity:        identity.Identity{ID: previousID, CustomerID: customerID},
		Type:            ai.TypeInsight,
		IsInsightActive: false,
	}

	mockDB.EXPECT().AIActivateInsight(gomock.Any(), activatedID).Return(activated, previous, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, ai.EventTypeUpdated, previous).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, ai.EventTypeUpdated, activated).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	if _, err := h.ActivateInsight(context.Background(), activatedID); err != nil {
		t.Fatalf("expected ok, got: %v", err)
	}
}

// Re-activating the already-active AI displaces nothing, so exactly one event
// is published. A spurious second event would tell subscribers a config they
// still hold as active was deactivated.
func Test_ActivateInsight_NoPreviousPublishesSingleEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	customerID := uuid.FromStringOrNil("c6c6c6c6-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("c6c6c6c6-0000-0000-0000-000000000002")

	activated := &ai.AI{
		Identity:        identity.Identity{ID: aiID, CustomerID: customerID},
		Type:            ai.TypeInsight,
		IsInsightActive: true,
	}

	mockDB.EXPECT().AIActivateInsight(gomock.Any(), aiID).Return(activated, nil, nil).Times(1)
	// Exactly one event total -- mc.Finish() fails on any extra publish.
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, ai.EventTypeUpdated, activated).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	if _, err := h.ActivateInsight(context.Background(), aiID); err != nil {
		t.Fatalf("expected ok, got: %v", err)
	}
}

// A type=normal target must be rejected with a 400 InvalidArgument, not a 500.
func Test_ActivateInsight_RejectsNormalType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	aiID := uuid.FromStringOrNil("c2c2c2c2-0000-0000-0000-000000000001")
	mockDB.EXPECT().AIActivateInsight(gomock.Any(), aiID).Return(nil, nil, dbhandler.ErrNotInsightAI).Times(1)
	// Negative control: no webhook may be published for a rejected activation.

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.ActivateInsight(context.Background(), aiID)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a *cerrors.VoipbinError, got: %v (%T)", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("expected Status=%v, got %v", cerrors.StatusInvalidArgument, ve.Status)
	}
	if ve.Reason != "AI_NOT_INSIGHT_TYPE" {
		t.Errorf("expected Reason=AI_NOT_INSIGHT_TYPE, got %v", ve.Reason)
	}
}

func Test_ActivateInsight_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	aiID := uuid.FromStringOrNil("c3c3c3c3-0000-0000-0000-000000000001")
	mockDB.EXPECT().AIActivateInsight(gomock.Any(), aiID).Return(nil, nil, dbhandler.ErrNotFound).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.ActivateInsight(context.Background(), aiID)
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a *cerrors.VoipbinError, got: %v (%T)", err, err)
	}
	if ve.Status != cerrors.StatusNotFound {
		t.Errorf("expected Status=%v, got %v", cerrors.StatusNotFound, ve.Status)
	}
	if ve.Reason != "AI_NOT_FOUND" {
		t.Errorf("expected Reason=AI_NOT_FOUND, got %v", ve.Reason)
	}
}

// The unique-index collision from a concurrent activation must surface as the
// dedicated activation-conflict reason, NOT SQUARE-23's now-removed
// AI_INSIGHT_ALREADY_EXISTS (whose "delete it before creating another" advice
// is wrong under the multi-config design).
func Test_ActivateInsight_ConcurrentActivationConflict(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	aiID := uuid.FromStringOrNil("c4c4c4c4-0000-0000-0000-000000000001")
	mockDB.EXPECT().AIActivateInsight(gomock.Any(), aiID).
		Return(nil, nil, fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_ai_active_insight_key'")).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.ActivateInsight(context.Background(), aiID)
	assertAlreadyExists(t, err, "AI_INSIGHT_ACTIVATION_CONFLICT")
}

// buildUpdateFields must clear is_insight_active for every non-insight type,
// unconditionally -- never as a read-then-write, which would race a concurrent
// activation.
func Test_buildUpdateFields_ClearsIsInsightActiveForNonInsightTypes(t *testing.T) {
	h := &aiHandler{utilHandler: utilhandler.NewUtilHandler()}

	tests := []struct {
		name        string
		aiType      ai.Type
		expectField bool
	}{
		{"normal type clears the flag", ai.TypeNormal, true},
		{"insight type leaves the flag untouched", ai.TypeInsight, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := h.buildUpdateFields("n", "d", tt.aiType, ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil, "",
				ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)

			got, ok := fields[ai.FieldIsInsightActive]
			if ok != tt.expectField {
				t.Fatalf("FieldIsInsightActive present = %v, want %v", ok, tt.expectField)
			}
			if ok && got != false {
				t.Errorf("FieldIsInsightActive = %v, want false", got)
			}
		})
	}
}

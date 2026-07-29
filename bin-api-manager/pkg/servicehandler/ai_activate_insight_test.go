package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func newAIServiceHandler(mc *gomock.Controller) (*serviceHandler, *requesthandler.MockRequestHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	return &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}, mockReq
}

func Test_AIActivateInsight(t *testing.T) {
	customerID := uuid.FromStringOrNil("e1e1e1e1-0000-4000-8000-000000000001")
	aiID := uuid.FromStringOrNil("e1e1e1e1-0000-4000-8000-000000000002")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("e1e1e1e1-0000-4000-8000-00000000000a"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	target := &amai.AI{
		Identity: commonidentity.Identity{ID: aiID, CustomerID: customerID},
		Type:     amai.TypeInsight,
	}
	activated := &amai.AI{
		Identity:        commonidentity.Identity{ID: aiID, CustomerID: customerID},
		Type:            amai.TypeInsight,
		IsInsightActive: true,
	}

	mockReq.EXPECT().AIV1AIGet(ctx, aiID).Return(target, nil)
	mockReq.EXPECT().AIV1AIActivateInsight(ctx, aiID).Return(activated, nil)

	res, err := h.AIActivateInsight(ctx, agent, aiID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := activated.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
	if !res.IsInsightActive {
		t.Errorf("response: IsInsightActive = false, want true")
	}
}

// Cross-customer activation must be rejected at the API boundary. bin-ai-manager
// scopes its transaction from the TARGET row's own customer_id, so without this
// check an agent could clear another customer's active Insight AI.
func Test_AIActivateInsight_RejectsCrossCustomerTarget(t *testing.T) {
	aiID := uuid.FromStringOrNil("e2e2e2e2-0000-4000-8000-000000000001")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("e2e2e2e2-0000-4000-8000-00000000000a"),
			CustomerID: uuid.FromStringOrNil("e2e2e2e2-0000-4000-8000-0000000000aa"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	// The target belongs to a DIFFERENT customer.
	mockReq.EXPECT().AIV1AIGet(ctx, aiID).Return(&amai.AI{
		Identity: commonidentity.Identity{
			ID:         aiID,
			CustomerID: uuid.FromStringOrNil("e2e2e2e2-0000-4000-8000-0000000000bb"),
		},
		Type: amai.TypeInsight,
	}, nil)

	// Negative control: the activation RPC must never be reached.

	_, err := h.AIActivateInsight(ctx, agent, aiID)
	if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Fatalf("Wrong match. expect: ErrPermissionDenied, got: %v", err)
	}
}

func Test_AIActivateInsight_RejectsInsufficientPermission(t *testing.T) {
	customerID := uuid.FromStringOrNil("e3e3e3e3-0000-4000-8000-0000000000aa")
	aiID := uuid.FromStringOrNil("e3e3e3e3-0000-4000-8000-000000000001")

	// A plain agent (not admin/manager) of the SAME customer.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("e3e3e3e3-0000-4000-8000-00000000000a"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	mockReq.EXPECT().AIV1AIGet(ctx, aiID).Return(&amai.AI{
		Identity: commonidentity.Identity{ID: aiID, CustomerID: customerID},
		Type:     amai.TypeInsight,
	}, nil)

	_, err := h.AIActivateInsight(ctx, agent, aiID)
	if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Fatalf("Wrong match. expect: ErrPermissionDenied, got: %v", err)
	}
}

// --- resolveInsightAIID: two-query resolution ---

// The activated AI must win over a more-recently-created but inactive one.
func Test_resolveInsightAIID_ActiveWinsOverMoreRecentInactive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e4e4e4e4-0000-4000-8000-0000000000aa")
	activeID := uuid.FromStringOrNil("e4e4e4e4-0000-4000-8000-000000000001")

	mockReq.EXPECT().
		AIV1AIList(ctx, "", uint64(1), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ uint64, filters map[amai.Field]any) ([]amai.AI, error) {
			// The dedicated query MUST carry all four filters. Losing
			// is_insight_active silently degrades this to "most recent".
			if got, ok := filters[amai.FieldIsInsightActive]; !ok || got != true {
				t.Errorf("active query filters: is_insight_active = %v (present: %v), want true", got, ok)
			}
			if got := filters[amai.FieldType]; got != string(amai.TypeInsight) && got != amai.TypeInsight {
				t.Errorf("active query filters: type = %v, want insight", got)
			}
			if got, ok := filters[amai.FieldDeleted]; !ok || got != false {
				t.Errorf("active query filters: deleted = %v (present: %v), want false", got, ok)
			}
			if got, ok := filters[amai.FieldCustomerID]; !ok || got != customerID {
				t.Errorf("active query filters: customer_id = %v (present: %v), want %v", got, ok, customerID)
			}
			return []amai.AI{{
				Identity:        commonidentity.Identity{ID: activeID, CustomerID: customerID},
				Type:            amai.TypeInsight,
				IsInsightActive: true,
			}}, nil
		})

	// Negative control: the 100-row fallback must NOT run when an active AI
	// exists — mc.Finish() fails on an unexpected call.

	got, err := h.resolveInsightAIID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got != activeID {
		t.Errorf("resolved ai id = %v, want %v", got, activeID)
	}
}

// The active AI must still be found when it is NOT in the top-100-by-recency
// page — the exact failure the dedicated query exists to prevent.
func Test_resolveInsightAIID_ActiveBeyondTop100PageStillFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e5e5e5e5-0000-4000-8000-0000000000aa")
	oldActiveID := uuid.FromStringOrNil("e5e5e5e5-0000-4000-8000-000000000999")

	// Simulate a customer with far more than 100 Insight AIs whose active one
	// is the OLDEST: a single 100-row tm_create-desc list would never see it.
	mockReq.EXPECT().
		AIV1AIList(ctx, "", uint64(1), gomock.Any()).
		Return([]amai.AI{{
			Identity:        commonidentity.Identity{ID: oldActiveID, CustomerID: customerID},
			Type:            amai.TypeInsight,
			IsInsightActive: true,
		}}, nil)

	got, err := h.resolveInsightAIID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got != oldActiveID {
		t.Errorf("resolved ai id = %v, want the active-but-old %v", got, oldActiveID)
	}
}

func Test_resolveInsightAIID_ZeroActiveFallsBackToMostRecent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e6e6e6e6-0000-4000-8000-0000000000aa")
	newestID := uuid.FromStringOrNil("e6e6e6e6-0000-4000-8000-000000000001")
	olderID := uuid.FromStringOrNil("e6e6e6e6-0000-4000-8000-000000000002")

	gomock.InOrder(
		mockReq.EXPECT().AIV1AIList(ctx, "", uint64(1), gomock.Any()).Return([]amai.AI{}, nil),
		mockReq.EXPECT().
			AIV1AIList(ctx, "", uint64(100), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ uint64, filters map[amai.Field]any) ([]amai.AI, error) {
				// The fallback query must NOT carry is_insight_active.
				if _, ok := filters[amai.FieldIsInsightActive]; ok {
					t.Errorf("fallback query must not filter on is_insight_active, got: %v", filters)
				}
				return []amai.AI{
					{Identity: commonidentity.Identity{ID: newestID, CustomerID: customerID}, Type: amai.TypeInsight},
					{Identity: commonidentity.Identity{ID: olderID, CustomerID: customerID}, Type: amai.TypeInsight},
				}, nil
			}),
	)

	got, err := h.resolveInsightAIID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got != newestID {
		t.Errorf("resolved ai id = %v, want the most-recently-created %v", got, newestID)
	}
}

func Test_resolveInsightAIID_NoInsightAIsAtAll(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newAIServiceHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e7e7e7e7-0000-4000-8000-0000000000aa")

	gomock.InOrder(
		mockReq.EXPECT().AIV1AIList(ctx, "", uint64(1), gomock.Any()).Return([]amai.AI{}, nil),
		mockReq.EXPECT().AIV1AIList(ctx, "", uint64(100), gomock.Any()).Return([]amai.AI{}, nil),
	)

	_, err := h.resolveInsightAIID(ctx, customerID)
	if !errors.Is(err, serviceerrors.ErrNotFound) {
		t.Fatalf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

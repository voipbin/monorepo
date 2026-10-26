package aicallhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
)

// Test_startReferenceTypeContactCase_otherCustomerRow pins that the duplicate
// key re-read is scoped to the AI's customer and that a row of another
// customer is refused: the error is returned at once, nothing is updated or
// terminated, and the create is not retried.
func Test_startReferenceTypeContactCase_otherCustomerRow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &aicallHandler{
		utilHandler:    mockUtil,
		reqHandler:     mockReq,
		notifyHandler:  mockNotify,
		db:             mockDB,
		aiHandler:      mockAI,
		messageHandler: mockMessage,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c0000000-0001-11f0-9000-000000000001")
	otherID := uuid.FromStringOrNil("c0000000-0002-11f0-9000-000000000001")
	referenceID := uuid.FromStringOrNil("c0000000-0003-11f0-9000-000000000001")
	a := &ai.AI{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0004-11f0-9000-000000000001"), CustomerID: customerID},
		EngineModel: ai.EngineModelOpenaiGPT5,
	}

	// exactly one create attempt: it collides with an existing active key
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("c0000000-0005-11f0-9000-000000000001"))
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("c0000000-0006-11f0-9000-000000000001"))
	mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_aicall_active_reference_key'")).Times(1)

	// the re-read is scoped to the AI's customer; the unscoped lookup and any
	// update/terminate of the returned row have no expectation and would fail
	mockDB.EXPECT().AIcallGetByCustomerIDAndReferenceID(ctx, customerID, referenceID).Return(&aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c0000000-0007-11f0-9000-000000000001"),
			CustomerID: otherID,
		},
		Status: aicall.StatusProgressing,
	}, nil)

	res, err := h.startReferenceTypeContactCase(ctx, uuid.Nil, a, aicall.AssistanceTypeAI, a.ID, uuid.FromStringOrNil("c0000000-0008-11f0-9000-000000000001"), referenceID, nil, uuid.Nil)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

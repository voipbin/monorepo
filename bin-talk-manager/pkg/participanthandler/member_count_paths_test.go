package participanthandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
)

func newParticipantHandlerWithMocks(t *testing.T) (*participantHandler, *MockDBHandler, *notifyhandler.MockNotifyHandler, *commonutil.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	mockDB := NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := commonutil.NewMockUtilHandler(mc)

	h := New(mockDB, NewMockSockHandler(mc), mockNotify, mockUtil).(*participantHandler)

	return h, mockDB, mockNotify, mockUtil
}

func Test_New(t *testing.T) {
	// a collaborator that New() leaves unset would panic on first use, so each one is exercised
	h, mockDB, mockNotify, mockUtil := newParticipantHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d31")
	chatID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d32")
	ownerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d33")
	participantID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d34")

	mockDB.EXPECT().ChatGet(ctx, chatID).Return(&chat.Chat{Identity: commonidentity.Identity{ID: chatID, CustomerID: customerID}}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(participantID)
	mockDB.EXPECT().ParticipantCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ChatMemberCountIncrement(ctx, chatID).Return(nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, participant.EventParticipantAdded, gomock.Any())

	res, err := h.ParticipantAdd(ctx, chatID, ownerID, "agent")
	if err != nil || res == nil || res.ID != participantID {
		t.Fatalf("ParticipantAdd() = %v, %v, want the participant %s", res, err, participantID)
	}
}

func Test_ParticipantAdd_chatNotFound_error(t *testing.T) {
	h, mockDB, _, _ := newParticipantHandlerWithMocks(t)
	ctx := context.Background()

	chatID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d35")
	ownerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d36")

	// no chat and no error from the database: nothing may be created or published
	mockDB.EXPECT().ChatGet(ctx, chatID).Return(nil, nil)

	res, err := h.ParticipantAdd(ctx, chatID, ownerID, "agent")
	if err == nil || res != nil {
		t.Errorf("ParticipantAdd() = %v, %v, want an error and no participant", res, err)
	}
}

// A failing member count update is logged but does not undo the participant change.
func Test_ParticipantAdd_memberCountIncrementFails(t *testing.T) {
	h, mockDB, mockNotify, mockUtil := newParticipantHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d37")
	chatID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d38")
	ownerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d39")
	participantID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d3a")

	mockDB.EXPECT().ChatGet(ctx, chatID).Return(&chat.Chat{Identity: commonidentity.Identity{ID: chatID, CustomerID: customerID}}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(participantID)
	mockDB.EXPECT().ParticipantCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ChatMemberCountIncrement(ctx, chatID).Return(errors.New("counter failure"))
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, participant.EventParticipantAdded, gomock.Any())

	res, err := h.ParticipantAdd(ctx, chatID, ownerID, "agent")
	if err != nil || res == nil || res.ID != participantID {
		t.Errorf("ParticipantAdd() = %v, %v, want the participant %s and no error", res, err, participantID)
	}
}

func Test_ParticipantRemove_memberCountDecrementFails(t *testing.T) {
	h, mockDB, mockNotify, _ := newParticipantHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d3b")
	chatID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d3c")
	participantID := uuid.FromStringOrNil("2f8d6a40-9a7e-11ef-8c2b-0f6f3c1a7d3d")

	p := &participant.Participant{
		Identity: commonidentity.Identity{ID: participantID, CustomerID: customerID},
		ChatID:   chatID,
	}

	mockDB.EXPECT().ParticipantGet(ctx, participantID).Return(p, nil)
	mockDB.EXPECT().ParticipantDelete(ctx, participantID).Return(nil)
	mockDB.EXPECT().ChatMemberCountDecrement(ctx, chatID).Return(errors.New("counter failure"))
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, participant.EventParticipantRemoved, p)

	if err := h.ParticipantRemove(ctx, customerID, participantID); err != nil {
		t.Errorf("ParticipantRemove() error = %v, want nil", err)
	}
}

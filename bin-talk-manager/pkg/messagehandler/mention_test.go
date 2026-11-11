package messagehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/mention"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

func mentionTestSetup(t *testing.T, ownerID, chatID, customerID uuid.UUID) (*gomock.Controller, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler, *messageHandler) {
	mc := gomock.NewController(t)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := commonutil.NewMockUtilHandler(mc)
	h := &messageHandler{dbHandler: mockDB, notifyHandler: mockNotify, utilHandler: mockUtil}

	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55")).AnyTimes()
	mockDB.EXPECT().ChatGet(gomock.Any(), chatID).Return(&chat.Chat{Identity: commonidentity.Identity{ID: chatID, CustomerID: customerID}, Type: chat.TypeTalk}, nil)
	mockDB.EXPECT().ParticipantListByChatIDs(gomock.Any(), []uuid.UUID{chatID}).Return([]*participant.Participant{
		{Owner: commonidentity.Owner{OwnerType: "agent", OwnerID: ownerID}, ChatID: chatID},
	}, nil)

	return mc, mockDB, mockNotify, h
}

func Test_MessageCreate_mentions(t *testing.T) {
	customerID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	chatID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	a1 := uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	a2 := uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")

	t.Run("normal message with agent mentions is stored with its mentions in one call", func(t *testing.T) {
		mc, mockDB, mockNotify, h := mentionTestSetup(t, ownerID, chatID, customerID)
		defer mc.Finish()
		ctx := context.Background()

		mockDB.EXPECT().MessageCreateWithMentions(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m *message.Message, ms []*mention.Mention) error {
				if len(ms) != 2 || ms[0].AgentID != a1 || ms[1].AgentID != a2 {
					t.Errorf("Wrong mentions. got: %v", ms)
				}
				for _, x := range ms {
					if x.CustomerID != customerID || x.ChatID != chatID || x.ID == uuid.Nil {
						t.Errorf("Wrong mention identity. got: %v", x)
					}
					if x.MessageID != m.ID {
						t.Errorf("Wrong message id. expect: %s, got: %s", m.ID, x.MessageID)
					}
				}
				return nil
			})
		mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, gomock.Any())

		res, err := h.MessageCreate(ctx, MessageCreateRequest{
			ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: message.TypeNormal,
			Text: "[@A](mention:" + a1.String() + ") [@B](mention:" + a2.String() + ") [@me](mention:" + ownerID.String() + ")",
		})
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("normal message without agent mention uses the plain insert", func(t *testing.T) {
		mc, mockDB, mockNotify, h := mentionTestSetup(t, ownerID, chatID, customerID)
		defer mc.Finish()
		ctx := context.Background()

		mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
		mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, gomock.Any())

		if _, err := h.MessageCreate(ctx, MessageCreateRequest{
			ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: message.TypeNormal,
			Text: "[@Bob](mention:contact:" + a1.String() + ")",
		}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("system message is not scanned for mentions", func(t *testing.T) {
		mc, mockDB, mockNotify, h := mentionTestSetup(t, ownerID, chatID, customerID)
		defer mc.Finish()
		ctx := context.Background()

		mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
		mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, gomock.Any())

		if _, err := h.MessageCreate(ctx, MessageCreateRequest{
			ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: message.TypeSystem,
			Text: "[@A](mention:" + a1.String() + ")",
		}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a failing store fails the creation and publishes no event", func(t *testing.T) {
		mc, mockDB, _, h := mentionTestSetup(t, ownerID, chatID, customerID)
		defer mc.Finish()
		ctx := context.Background()

		mockDB.EXPECT().MessageCreateWithMentions(ctx, gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert failed"))

		res, err := h.MessageCreate(ctx, MessageCreateRequest{
			ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: message.TypeNormal,
			Text: "[@A](mention:" + a1.String() + ")",
		})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil result, got: %v, %v", res, err)
		}
	})
}

func Test_MentionList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &messageHandler{dbHandler: mockDB}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	agentID := uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	expect := []*message.Message{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("33333333-3333-4333-8333-333333333333")}}}

	mockDB.EXPECT().MentionList(ctx, customerID, agentID, "tok", uint64(10)).Return(expect, nil)

	res, err := h.MentionList(ctx, customerID, agentID, "tok", 10)
	if err != nil || !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v, err: %v", expect, res, err)
	}

	mockDB.EXPECT().MentionList(ctx, customerID, agentID, "", uint64(10)).Return(nil, fmt.Errorf("boom"))
	if _, err := h.MentionList(ctx, customerID, agentID, "", 10); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

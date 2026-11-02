package pinhandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/pin"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

var (
	testCustomerID = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000001")
	testOtherCust  = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000002")
	testAgentID    = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000003")
	testChatID     = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000004")
	testMessageID  = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000005")
	testPinID      = uuid.FromStringOrNil("11111111-0000-4000-8000-000000000006")
)

func testChat(chatType chat.Type, customerID uuid.UUID, deleted bool) *chat.Chat {
	c := &chat.Chat{
		Identity: commonidentity.Identity{ID: testChatID, CustomerID: customerID},
		Type:     chatType,
	}
	if deleted {
		now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
		c.TMDelete = &now
	}
	return c
}

func testMessage(customerID uuid.UUID, deleted bool) *message.Message {
	m := &message.Message{
		Identity: commonidentity.Identity{ID: testMessageID, CustomerID: customerID},
		ChatID:   testChatID,
	}
	if deleted {
		now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
		m.TMDelete = &now
	}
	return m
}

func testPin(targetType pin.TargetType, targetID uuid.UUID) *pin.Pin {
	return &pin.Pin{
		Identity:   commonidentity.Identity{ID: testPinID, CustomerID: testCustomerID},
		Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: testAgentID},
		ChatID:     testChatID,
		TargetType: targetType,
		TargetID:   targetID,
	}
}

func isStatus(err error, status cerrors.Status) bool {
	var ve *cerrors.VoipbinError
	return stderrors.As(err, &ve) && ve.Status == status
}

func Test_PinCreate(t *testing.T) {
	participantFilter := map[participant.Field]any{
		participant.FieldChatID:    testChatID,
		participant.FieldOwnerType: "agent",
		participant.FieldOwnerID:   testAgentID,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		ownerType  string
		ownerID    uuid.UUID
		targetType pin.TargetType
		targetID   uuid.UUID

		responseChat         *chat.Chat
		responseChatErr      error
		responseMessage      *message.Message
		responseMessageErr   error
		responseParticipants []*participant.Participant
		expectParticipants   bool
		expectCreate         *pin.Pin
		responseCreateErr    error

		expectStatus cerrors.Status
		expectErr    bool
	}{
		{
			name:         "chat pin of a talk chat",
			customerID:   testCustomerID,
			ownerType:    "agent",
			ownerID:      testAgentID,
			targetType:   pin.TargetTypeChat,
			targetID:     testChatID,
			responseChat: testChat(chat.TypeTalk, testCustomerID, false),
			expectCreate: testPin(pin.TargetTypeChat, testChatID),
		},
		{
			name:            "message pin fills the chat id of the message",
			customerID:      testCustomerID,
			ownerType:       "agent",
			ownerID:         testAgentID,
			targetType:      pin.TargetTypeMessage,
			targetID:        testMessageID,
			responseMessage: testMessage(testCustomerID, false),
			responseChat:    testChat(chat.TypeTalk, testCustomerID, false),
			expectCreate:    testPin(pin.TargetTypeMessage, testMessageID),
		},
		{
			name:                 "direct chat, participant",
			customerID:           testCustomerID,
			ownerType:            "agent",
			ownerID:              testAgentID,
			targetType:           pin.TargetTypeChat,
			targetID:             testChatID,
			responseChat:         testChat(chat.TypeDirect, testCustomerID, false),
			expectParticipants:   true,
			responseParticipants: []*participant.Participant{{}},
			expectCreate:         testPin(pin.TargetTypeChat, testChatID),
		},
		{
			name:                 "group chat, not a participant",
			customerID:           testCustomerID,
			ownerType:            "agent",
			ownerID:              testAgentID,
			targetType:           pin.TargetTypeChat,
			targetID:             testChatID,
			responseChat:         testChat(chat.TypeGroup, testCustomerID, false),
			expectParticipants:   true,
			responseParticipants: []*participant.Participant{},
			expectStatus:         cerrors.StatusPermissionDenied,
			expectErr:            true,
		},
		{
			name:         "deleted chat",
			customerID:   testCustomerID,
			ownerType:    "agent",
			ownerID:      testAgentID,
			targetType:   pin.TargetTypeChat,
			targetID:     testChatID,
			responseChat: testChat(chat.TypeTalk, testCustomerID, true),
			expectStatus: cerrors.StatusNotFound,
			expectErr:    true,
		},
		{
			name:         "chat of another customer",
			customerID:   testCustomerID,
			ownerType:    "agent",
			ownerID:      testAgentID,
			targetType:   pin.TargetTypeChat,
			targetID:     testChatID,
			responseChat: testChat(chat.TypeTalk, testOtherCust, false),
			expectStatus: cerrors.StatusNotFound,
			expectErr:    true,
		},
		{
			name:            "chat not found",
			customerID:      testCustomerID,
			ownerType:       "agent",
			ownerID:         testAgentID,
			targetType:      pin.TargetTypeChat,
			targetID:        testChatID,
			responseChatErr: dbhandler.ErrNotFound,
			expectStatus:    cerrors.StatusNotFound,
			expectErr:       true,
		},
		{
			name:            "deleted message",
			customerID:      testCustomerID,
			ownerType:       "agent",
			ownerID:         testAgentID,
			targetType:      pin.TargetTypeMessage,
			targetID:        testMessageID,
			responseMessage: testMessage(testCustomerID, true),
			expectStatus:    cerrors.StatusNotFound,
			expectErr:       true,
		},
		{
			name:            "message of another customer",
			customerID:      testCustomerID,
			ownerType:       "agent",
			ownerID:         testAgentID,
			targetType:      pin.TargetTypeMessage,
			targetID:        testMessageID,
			responseMessage: testMessage(testOtherCust, false),
			expectStatus:    cerrors.StatusNotFound,
			expectErr:       true,
		},
		{
			name:               "message not found",
			customerID:         testCustomerID,
			ownerType:          "agent",
			ownerID:            testAgentID,
			targetType:         pin.TargetTypeMessage,
			targetID:           testMessageID,
			responseMessageErr: dbhandler.ErrNotFound,
			expectStatus:       cerrors.StatusNotFound,
			expectErr:          true,
		},
		{
			name:            "message in a deleted chat",
			customerID:      testCustomerID,
			ownerType:       "agent",
			ownerID:         testAgentID,
			targetType:      pin.TargetTypeMessage,
			targetID:        testMessageID,
			responseMessage: testMessage(testCustomerID, false),
			responseChat:    testChat(chat.TypeTalk, testCustomerID, true),
			expectStatus:    cerrors.StatusNotFound,
			expectErr:       true,
		},
		{
			name:         "storage error",
			customerID:   testCustomerID,
			ownerType:    "agent",
			ownerID:      testAgentID,
			targetType:   pin.TargetTypeChat,
			targetID:     testChatID,
			responseChat: testChat(chat.TypeTalk, testCustomerID, false),
			expectCreate: testPin(pin.TargetTypeChat, testChatID),

			responseCreateErr: stderrors.New("db down"),
			expectErr:         true,
		},
		{"nil customer", uuid.Nil, "agent", testAgentID, pin.TargetTypeChat, testChatID, nil, nil, nil, nil, nil, false, nil, nil, cerrors.StatusInvalidArgument, true},
		{"owner type not agent", testCustomerID, "customer", testAgentID, pin.TargetTypeChat, testChatID, nil, nil, nil, nil, nil, false, nil, nil, cerrors.StatusInvalidArgument, true},
		{"nil owner", testCustomerID, "agent", uuid.Nil, pin.TargetTypeChat, testChatID, nil, nil, nil, nil, nil, false, nil, nil, cerrors.StatusInvalidArgument, true},
		{"unknown target type", testCustomerID, "agent", testAgentID, pin.TargetType("call"), testChatID, nil, nil, nil, nil, nil, false, nil, nil, cerrors.StatusInvalidArgument, true},
		{"nil target", testCustomerID, "agent", testAgentID, pin.TargetTypeChat, uuid.Nil, nil, nil, nil, nil, nil, false, nil, nil, cerrors.StatusInvalidArgument, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)
			h := New(mockDB, mockUtil)
			ctx := context.Background()

			switch tt.targetType {
			case pin.TargetTypeMessage:
				if tt.responseMessage != nil || tt.responseMessageErr != nil {
					mockDB.EXPECT().MessageGet(ctx, tt.targetID).Return(tt.responseMessage, tt.responseMessageErr)
				}
				if tt.responseChat != nil {
					mockDB.EXPECT().ChatGet(ctx, testChatID).Return(tt.responseChat, tt.responseChatErr)
				}
			case pin.TargetTypeChat:
				if tt.responseChat != nil || tt.responseChatErr != nil {
					mockDB.EXPECT().ChatGet(ctx, tt.targetID).Return(tt.responseChat, tt.responseChatErr)
				}
			}
			if tt.expectParticipants {
				mockDB.EXPECT().ParticipantList(ctx, participantFilter).Return(tt.responseParticipants, nil)
			}
			if tt.expectCreate != nil {
				mockUtil.EXPECT().UUIDCreate().Return(testPinID)
				mockDB.EXPECT().PinCreate(ctx, tt.expectCreate).DoAndReturn(func(_ context.Context, p *pin.Pin) (*pin.Pin, error) {
					if tt.responseCreateErr != nil {
						return nil, tt.responseCreateErr
					}
					return p, nil
				})
			}

			res, err := h.PinCreate(ctx, tt.customerID, tt.ownerType, tt.ownerID, tt.targetType, tt.targetID)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: %v", res)
				}
				if tt.expectStatus != "" && !isStatus(err, tt.expectStatus) {
					t.Errorf("Wrong match. expect: %s, got: %v", tt.expectStatus, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectCreate, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCreate, res)
			}
		})
	}
}

func Test_PinList(t *testing.T) {
	pins := []*pin.Pin{testPin(pin.TargetTypeChat, testChatID)}

	tests := []struct {
		name string

		customerID uuid.UUID
		ownerType  string
		ownerID    uuid.UUID
		targetType pin.TargetType
		token      string
		size       uint64

		expectSize uint64
		expectErr  bool
	}{
		{"normal", testCustomerID, "agent", testAgentID, "", "", 10, 10, false},
		{"filtered by target type", testCustomerID, "agent", testAgentID, pin.TargetTypeMessage, "tok", 5, 5, false},
		{"size zero gets the default", testCustomerID, "agent", testAgentID, "", "", 0, defaultPageSize, false},
		{"nil customer", uuid.Nil, "agent", testAgentID, "", "", 10, 0, true},
		{"nil owner", testCustomerID, "agent", uuid.Nil, "", "", 10, 0, true},
		{"owner type not agent", testCustomerID, "customer", testAgentID, "", "", 10, 0, true},
		{"unknown target type", testCustomerID, "agent", testAgentID, pin.TargetType("call"), "", 10, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)
			h := New(mockDB, mockUtil)
			ctx := context.Background()

			if !tt.expectErr {
				mockDB.EXPECT().PinList(ctx, tt.customerID, tt.ownerType, tt.ownerID, tt.targetType, testChatID, tt.token, tt.expectSize).Return(pins, nil)
			}

			res, err := h.PinList(ctx, tt.customerID, tt.ownerType, tt.ownerID, tt.targetType, testChatID, tt.token, tt.size)
			if tt.expectErr {
				if err == nil || !isStatus(err, cerrors.StatusInvalidArgument) {
					t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(pins, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", pins, res)
			}
		})
	}
}

func Test_PinList_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := New(mockDB, commonutil.NewMockUtilHandler(mc))
	ctx := context.Background()

	mockDB.EXPECT().PinList(ctx, testCustomerID, "agent", testAgentID, pin.TargetType(""), uuid.Nil, "", uint64(10)).Return(nil, stderrors.New("db down"))

	if _, err := h.PinList(ctx, testCustomerID, "agent", testAgentID, "", uuid.Nil, "", 10); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

// Verify (5): the pin of another owner or customer is not found; an own pin that is no longer
// visible (for example the agent left the chat) can still be deleted.
func Test_PinDelete(t *testing.T) {
	own := testPin(pin.TargetTypeChat, testChatID)

	otherAgent := testPin(pin.TargetTypeChat, testChatID)
	otherAgent.OwnerID = uuid.FromStringOrNil("11111111-0000-4000-8000-0000000000ff")

	otherCustomer := testPin(pin.TargetTypeChat, testChatID)
	otherCustomer.CustomerID = testOtherCust

	otherType := testPin(pin.TargetTypeChat, testChatID)
	otherType.OwnerType = commonidentity.OwnerType("customer")

	tests := []struct {
		name string

		responseGet    *pin.Pin
		responseGetErr error
		expectDelete   bool
		deleteErr      error

		expectStatus cerrors.Status
		expectErr    bool
	}{
		{name: "own pin", responseGet: own, expectDelete: true},
		{name: "pin of another agent", responseGet: otherAgent, expectStatus: cerrors.StatusNotFound, expectErr: true},
		{name: "pin of another customer", responseGet: otherCustomer, expectStatus: cerrors.StatusNotFound, expectErr: true},
		{name: "pin of another owner type", responseGet: otherType, expectStatus: cerrors.StatusNotFound, expectErr: true},
		{name: "unknown pin", responseGetErr: dbhandler.ErrNotFound, expectStatus: cerrors.StatusNotFound, expectErr: true},
		{name: "deleted concurrently", responseGet: own, expectDelete: true, deleteErr: dbhandler.ErrNotFound, expectStatus: cerrors.StatusNotFound, expectErr: true},
		{name: "storage error on get", responseGetErr: stderrors.New("db down"), expectErr: true},
		{name: "storage error on delete", responseGet: own, expectDelete: true, deleteErr: stderrors.New("db down"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := New(mockDB, commonutil.NewMockUtilHandler(mc))
			ctx := context.Background()

			mockDB.EXPECT().PinGet(ctx, testPinID).Return(tt.responseGet, tt.responseGetErr)
			if tt.expectDelete {
				mockDB.EXPECT().PinDelete(ctx, testPinID, testCustomerID, "agent", testAgentID).Return(tt.deleteErr)
			}

			res, err := h.PinDelete(ctx, testPinID, testCustomerID, "agent", testAgentID)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: %v", res)
				}
				if tt.expectStatus != "" && !isStatus(err, tt.expectStatus) {
					t.Errorf("Wrong match. expect: %s, got: %v", tt.expectStatus, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(own, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", own, res)
			}
		})
	}
}

func Test_PinDelete_invalid(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := New(dbhandler.NewMockDBHandler(mc), commonutil.NewMockUtilHandler(mc))
	ctx := context.Background()

	for name, args := range map[string][4]any{
		"nil id":       {uuid.Nil, testCustomerID, "agent", testAgentID},
		"nil customer": {testPinID, uuid.Nil, "agent", testAgentID},
		"nil owner":    {testPinID, testCustomerID, "agent", uuid.Nil},
		"owner type":   {testPinID, testCustomerID, "customer", testAgentID},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.PinDelete(ctx, args[0].(uuid.UUID), args[1].(uuid.UUID), args[2].(string), args[3].(uuid.UUID))
			if !isStatus(err, cerrors.StatusInvalidArgument) {
				t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
			}
		})
	}
}

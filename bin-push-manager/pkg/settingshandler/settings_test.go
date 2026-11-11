package settingshandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/models/settings"
	"monorepo/bin-push-manager/pkg/dbhandler"
)

var (
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testAgentID    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testAgentID2   = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e53")
	testAgentID3   = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e54")
	testChatID     = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")
)

var errBoom = errors.New("boom")

func newTestHandler(mc *gomock.Controller) (*settingsHandler, *dbhandler.MockDBHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)

	return &settingsHandler{db: mockDB}, mockDB
}

func assertReason(t *testing.T, err error, reason string) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: a typed error, got: %v", err)
	}
	if ve.Reason != reason {
		t.Errorf("Wrong match. expect reason: %s, got: %s", reason, ve.Reason)
	}
}

func Test_Get(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB := newTestHandler(mc)
	ctx := context.Background()

	t.Run("nothing stored", func(t *testing.T) {
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return([]notification.Kind{}, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return([]uuid.UUID{}, nil)

		res, err := h.Get(ctx, testCustomerID, testAgentID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(res, settings.New(nil, nil)) {
			t.Errorf("Wrong match. got: %v", res)
		}
		if res.MutedChatIDs == nil {
			t.Errorf("Wrong match. the muted list must not be nil")
		}
	})

	t.Run("stored values", func(t *testing.T) {
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return([]notification.Kind{notification.KindCaseNote}, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return([]uuid.UUID{testChatID}, nil)

		res, err := h.Get(ctx, testCustomerID, testAgentID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.Kinds[notification.KindCaseNote] || !res.Kinds[notification.KindChatMessage] || len(res.MutedChatIDs) != 1 {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	t.Run("database errors", func(t *testing.T) {
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return(nil, errBoom)
		if _, err := h.Get(ctx, testCustomerID, testAgentID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}

		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return([]notification.Kind{}, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return(nil, errBoom)
		if _, err := h.Get(ctx, testCustomerID, testAgentID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})

	t.Run("the owner is required", func(t *testing.T) {
		_, err := h.Get(ctx, uuid.Nil, testAgentID)
		assertReason(t, err, "INVALID_OWNER")
		_, err = h.Get(ctx, testCustomerID, uuid.Nil)
		assertReason(t, err, "INVALID_OWNER")
	})
}

func Test_Update(t *testing.T) {
	ctx := context.Background()

	expectGet := func(mockDB *dbhandler.MockDBHandler, disabled []notification.Kind) {
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return(disabled, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return([]uuid.UUID{}, nil)
	}

	t.Run("on and off, in the order of the kind, then the settings", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		gomock.InOrder(
			mockDB.EXPECT().KindPrefDisable(ctx, testCustomerID, testAgentID, notification.KindCaseNote).Return(nil),
			mockDB.EXPECT().KindPrefEnable(ctx, testCustomerID, testAgentID, notification.KindChatMessage).Return(nil),
		)
		expectGet(mockDB, []notification.Kind{notification.KindCaseNote})

		res, err := h.Update(ctx, testCustomerID, testAgentID, map[notification.Kind]bool{
			notification.KindChatMessage: true,
			notification.KindCaseNote:    false,
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.Kinds[notification.KindCaseNote] || !res.Kinds[notification.KindChatMessage] {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	t.Run("an empty change returns the current settings", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		for _, kinds := range []map[notification.Kind]bool{nil, {}} {
			expectGet(mockDB, nil)
			if _, err := h.Update(ctx, testCustomerID, testAgentID, kinds); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
		}
	})

	t.Run("a call kind or an unknown kind is rejected before anything is written", func(t *testing.T) {
		for _, bad := range []notification.Kind{notification.KindCallIncoming, notification.KindCallCancelled, "unknown_kind", ""} {
			mc := gomock.NewController(t)
			h, _ := newTestHandler(mc) // the strict mock has no expectation: any write fails the test

			_, err := h.Update(ctx, testCustomerID, testAgentID, map[notification.Kind]bool{
				notification.KindChatMessage: false,
				bad:                          false,
			})
			assertReason(t, err, "INVALID_KIND")
			mc.Finish()
		}
	})

	t.Run("a failed write stops and returns the error, the same request can be sent again", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		kinds := map[notification.Kind]bool{notification.KindCaseNote: false, notification.KindChatMessage: false}

		gomock.InOrder(
			mockDB.EXPECT().KindPrefDisable(ctx, testCustomerID, testAgentID, notification.KindCaseNote).Return(nil),
			mockDB.EXPECT().KindPrefDisable(ctx, testCustomerID, testAgentID, notification.KindChatMessage).Return(errBoom),
		)
		if _, err := h.Update(ctx, testCustomerID, testAgentID, kinds); err == nil {
			t.Fatalf("Wrong match. expect: error")
		}

		// the retry applies everything again and every write is idempotent
		gomock.InOrder(
			mockDB.EXPECT().KindPrefDisable(ctx, testCustomerID, testAgentID, notification.KindCaseNote).Return(nil),
			mockDB.EXPECT().KindPrefDisable(ctx, testCustomerID, testAgentID, notification.KindChatMessage).Return(nil),
		)
		expectGet(mockDB, []notification.Kind{notification.KindCaseNote, notification.KindChatMessage})
		if _, err := h.Update(ctx, testCustomerID, testAgentID, kinds); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("the read after the writes fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefEnable(ctx, testCustomerID, testAgentID, notification.KindCaseNote).Return(nil)
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return(nil, errBoom)

		if _, err := h.Update(ctx, testCustomerID, testAgentID, map[notification.Kind]bool{notification.KindCaseNote: true}); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})

	t.Run("the owner is required", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newTestHandler(mc)

		_, err := h.Update(ctx, testCustomerID, uuid.Nil, map[notification.Kind]bool{notification.KindCaseNote: true})
		assertReason(t, err, "INVALID_OWNER")
	})
}

func Test_Mute(t *testing.T) {
	ctx := context.Background()

	t.Run("add and remove return the settings", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().MuteCreate(ctx, testCustomerID, testAgentID, testChatID).Return(nil)
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return([]notification.Kind{}, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return([]uuid.UUID{testChatID}, nil)

		res, err := h.MuteAdd(ctx, testCustomerID, testAgentID, testChatID)
		if err != nil || len(res.MutedChatIDs) != 1 || res.MutedChatIDs[0] != testChatID {
			t.Fatalf("Wrong match. got: %v, %v", res, err)
		}

		mockDB.EXPECT().MuteDelete(ctx, testCustomerID, testAgentID, testChatID).Return(nil)
		mockDB.EXPECT().KindPrefListDisabledByAgent(ctx, testCustomerID, testAgentID).Return([]notification.Kind{}, nil)
		mockDB.EXPECT().MuteListByAgent(ctx, testCustomerID, testAgentID).Return([]uuid.UUID{}, nil)

		res, err = h.MuteRemove(ctx, testCustomerID, testAgentID, testChatID)
		if err != nil || len(res.MutedChatIDs) != 0 {
			t.Fatalf("Wrong match. got: %v, %v", res, err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().MuteCreate(ctx, testCustomerID, testAgentID, testChatID).Return(errBoom)
		if _, err := h.MuteAdd(ctx, testCustomerID, testAgentID, testChatID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
		mockDB.EXPECT().MuteDelete(ctx, testCustomerID, testAgentID, testChatID).Return(errBoom)
		if _, err := h.MuteRemove(ctx, testCustomerID, testAgentID, testChatID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}

		_, err := h.MuteAdd(ctx, testCustomerID, testAgentID, uuid.Nil)
		assertReason(t, err, "INVALID_CHAT")
		_, err = h.MuteRemove(ctx, testCustomerID, testAgentID, uuid.Nil)
		assertReason(t, err, "INVALID_CHAT")
		_, err = h.MuteAdd(ctx, uuid.Nil, testAgentID, testChatID)
		assertReason(t, err, "INVALID_OWNER")
	})
}

func Test_FilterRecipients(t *testing.T) {
	ctx := context.Background()
	all := []uuid.UUID{testAgentID, testAgentID2, testAgentID3}

	t.Run("a kind that is off removes the agent, the order is kept", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefListDisabledByAgents(ctx, testCustomerID, all, notification.KindCaseNote).Return([]uuid.UUID{testAgentID2}, nil)

		res, err := h.FilterRecipients(ctx, testCustomerID, all, notification.KindCaseNote, uuid.Nil)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(res, []uuid.UUID{testAgentID, testAgentID3}) {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	t.Run("chat_message also removes the agents that muted the chat", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefListDisabledByAgents(ctx, testCustomerID, all, notification.KindChatMessage).Return([]uuid.UUID{testAgentID}, nil)
		mockDB.EXPECT().MuteListMutedByAgents(ctx, testCustomerID, all, testChatID).Return([]uuid.UUID{testAgentID3}, nil)

		res, err := h.FilterRecipients(ctx, testCustomerID, all, notification.KindChatMessage, testChatID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(res, []uuid.UUID{testAgentID2}) {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	t.Run("a kind other than chat_message never reads the mutes, even with a chat id", func(t *testing.T) {
		for _, kind := range []notification.Kind{notification.KindChatMentioned, notification.KindConversationMessage, notification.KindCaseAssigned, notification.KindCaseNote} {
			mc := gomock.NewController(t)
			h, mockDB := newTestHandler(mc) // MuteListMutedByAgents has no expectation

			mockDB.EXPECT().KindPrefListDisabledByAgents(ctx, testCustomerID, all, kind).Return(nil, nil)

			res, err := h.FilterRecipients(ctx, testCustomerID, all, kind, testChatID)
			if err != nil || !reflect.DeepEqual(res, all) {
				t.Errorf("Wrong match. kind: %s, got: %v, %v", kind, res, err)
			}
			mc.Finish()
		}
	})

	t.Run("no agent needs no query", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newTestHandler(mc)

		res, err := h.FilterRecipients(ctx, testCustomerID, nil, notification.KindChatMessage, testChatID)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	t.Run("database errors are returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefListDisabledByAgents(ctx, testCustomerID, all, notification.KindChatMessage).Return(nil, errBoom)
		if _, err := h.FilterRecipients(ctx, testCustomerID, all, notification.KindChatMessage, testChatID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}

		mockDB.EXPECT().KindPrefListDisabledByAgents(ctx, testCustomerID, all, notification.KindChatMessage).Return(nil, nil)
		mockDB.EXPECT().MuteListMutedByAgents(ctx, testCustomerID, all, testChatID).Return(nil, errBoom)
		if _, err := h.FilterRecipients(ctx, testCustomerID, all, notification.KindChatMessage, testChatID); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})
}

func Test_EventAgentDeleted(t *testing.T) {
	ctx := context.Background()
	a := &amagent.Agent{Identity: commonidentity.Identity{ID: testAgentID, CustomerID: testCustomerID}}

	t.Run("both tables are cleaned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefDeleteByAgentID(ctx, testAgentID).Return(nil)
		mockDB.EXPECT().MuteDeleteByAgentID(ctx, testAgentID).Return(nil)

		if err := h.EventAgentDeleted(ctx, a); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a failure of the first does not skip the second", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newTestHandler(mc)

		mockDB.EXPECT().KindPrefDeleteByAgentID(ctx, testAgentID).Return(errBoom)
		mockDB.EXPECT().MuteDeleteByAgentID(ctx, testAgentID).Return(nil)

		if err := h.EventAgentDeleted(ctx, a); !errors.Is(err, errBoom) {
			t.Errorf("Wrong match. expect: boom, got: %v", err)
		}
	})
}

func Test_EventCustomerDeleted(t *testing.T) {
	ctx := context.Background()
	c := &cmcustomer.Customer{ID: testCustomerID}

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockDB := newTestHandler(mc)

	mockDB.EXPECT().KindPrefDeleteByCustomerID(ctx, testCustomerID).Return(errBoom)
	mockDB.EXPECT().MuteDeleteByCustomerID(ctx, testCustomerID).Return(errBoom)

	if err := h.EventCustomerDeleted(ctx, c); !errors.Is(err, errBoom) {
		t.Errorf("Wrong match. expect: boom, got: %v", err)
	}
}

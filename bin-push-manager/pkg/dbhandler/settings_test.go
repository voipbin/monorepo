package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/notification"
)

var (
	testChatA  = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")
	testChatB  = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e12")
	testAgentC = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e54")
)

func newSettingsTestHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler) {
	t.Helper()

	for _, table := range []string{"push_kind_prefs", "push_chat_mutes"} {
		if _, err := dbTest.Exec("delete from " + table); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

	return &handler{utilHandler: mockUtil, db: dbTest}, mockUtil
}

func countRows(t *testing.T, table string) int {
	t.Helper()

	var n int
	if err := dbTest.QueryRow("select count(*) from " + table).Scan(&n); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	return n
}

func Test_KindPref_DisableEnableList(t *testing.T) {
	h, _ := newSettingsTestHandler(t)
	ctx := context.Background()

	// nothing stored
	got, err := h.KindPrefListDisabledByAgent(ctx, testCustomerA, testAgentA)
	if err != nil || len(got) != 0 {
		t.Fatalf("Wrong match. expect: empty, got: %v, %v", got, err)
	}

	// disabling twice is the same as once (idempotent) and leaves one row
	for i := 0; i < 2; i++ {
		if err := h.KindPrefDisable(ctx, testCustomerA, testAgentA, notification.KindChatMessage); err != nil {
			t.Fatalf("Wrong match. try: %d, expect: ok, got: %v", i, err)
		}
	}
	if n := countRows(t, "push_kind_prefs"); n != 1 {
		t.Errorf("Wrong match. expect: 1 row, got: %d", n)
	}

	if err := h.KindPrefDisable(ctx, testCustomerA, testAgentA, notification.KindCaseNote); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	// another agent
	if err := h.KindPrefDisable(ctx, testCustomerA, testAgentB, notification.KindChatMessage); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err = h.KindPrefListDisabledByAgent(ctx, testCustomerA, testAgentA)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, []notification.Kind{notification.KindCaseNote, notification.KindChatMessage}) {
		t.Errorf("Wrong match. got: %v", got)
	}

	// the list is scoped by the customer
	got, err = h.KindPrefListDisabledByAgent(ctx, testCustomerB, testAgentA)
	if err != nil || len(got) != 0 {
		t.Errorf("Wrong match. expect: empty for another customer, got: %v, %v", got, err)
	}

	// enabling deletes the row, and enabling a kind that is on succeeds
	for i := 0; i < 2; i++ {
		if err := h.KindPrefEnable(ctx, testCustomerA, testAgentA, notification.KindChatMessage); err != nil {
			t.Fatalf("Wrong match. try: %d, expect: ok, got: %v", i, err)
		}
	}
	got, _ = h.KindPrefListDisabledByAgent(ctx, testCustomerA, testAgentA)
	if !reflect.DeepEqual(got, []notification.Kind{notification.KindCaseNote}) {
		t.Errorf("Wrong match. got: %v", got)
	}

	// enabling with the wrong customer does not touch the row of the agent
	if err := h.KindPrefEnable(ctx, testCustomerB, testAgentA, notification.KindCaseNote); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	got, _ = h.KindPrefListDisabledByAgent(ctx, testCustomerA, testAgentA)
	if len(got) != 1 {
		t.Errorf("Wrong match. the row of another customer must stay. got: %v", got)
	}
}

func Test_KindPref_ListDisabledByAgents(t *testing.T) {
	h, _ := newSettingsTestHandler(t)
	ctx := context.Background()

	_ = h.KindPrefDisable(ctx, testCustomerA, testAgentA, notification.KindChatMessage)
	_ = h.KindPrefDisable(ctx, testCustomerA, testAgentB, notification.KindCaseNote)
	_ = h.KindPrefDisable(ctx, testCustomerB, testAgentC, notification.KindChatMessage)

	t.Run("only the agents with the kind off", func(t *testing.T) {
		got, err := h.KindPrefListDisabledByAgents(ctx, testCustomerA, []uuid.UUID{testAgentA, testAgentB, testAgentC}, notification.KindChatMessage)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		// testAgentC belongs to another customer in the table, so it is not found for customer A
		if !reflect.DeepEqual(got, []uuid.UUID{testAgentA}) {
			t.Errorf("Wrong match. got: %v", got)
		}
	})

	t.Run("an empty agent list needs no query", func(t *testing.T) {
		got, err := h.KindPrefListDisabledByAgents(ctx, testCustomerA, nil, notification.KindChatMessage)
		if err != nil || len(got) != 0 {
			t.Errorf("Wrong match. got: %v, %v", got, err)
		}
	})
}

func Test_Mute_CreateDeleteList(t *testing.T) {
	h, _ := newSettingsTestHandler(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := h.MuteCreate(ctx, testCustomerA, testAgentA, testChatA); err != nil {
			t.Fatalf("Wrong match. try: %d, expect: ok, got: %v", i, err)
		}
	}
	if n := countRows(t, "push_chat_mutes"); n != 1 {
		t.Errorf("Wrong match. expect: 1 row, got: %d", n)
	}
	if err := h.MuteCreate(ctx, testCustomerA, testAgentA, testChatB); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.MuteCreate(ctx, testCustomerA, testAgentB, testChatA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err := h.MuteListByAgent(ctx, testCustomerA, testAgentA)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, []uuid.UUID{testChatA, testChatB}) {
		t.Errorf("Wrong match. got: %v", got)
	}

	if got, _ := h.MuteListByAgent(ctx, testCustomerB, testAgentA); len(got) != 0 {
		t.Errorf("Wrong match. expect: empty for another customer, got: %v", got)
	}

	muted, err := h.MuteListMutedByAgents(ctx, testCustomerA, []uuid.UUID{testAgentA, testAgentB, testAgentC}, testChatA)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(muted, []uuid.UUID{testAgentA, testAgentB}) {
		t.Errorf("Wrong match. got: %v", muted)
	}
	if muted, _ := h.MuteListMutedByAgents(ctx, testCustomerA, nil, testChatA); len(muted) != 0 {
		t.Errorf("Wrong match. got: %v", muted)
	}

	// unmuting twice succeeds
	for i := 0; i < 2; i++ {
		if err := h.MuteDelete(ctx, testCustomerA, testAgentA, testChatA); err != nil {
			t.Fatalf("Wrong match. try: %d, expect: ok, got: %v", i, err)
		}
	}
	got, _ = h.MuteListByAgent(ctx, testCustomerA, testAgentA)
	if !reflect.DeepEqual(got, []uuid.UUID{testChatB}) {
		t.Errorf("Wrong match. got: %v", got)
	}
}

func Test_Settings_failuresAreErrors(t *testing.T) {
	h, _ := newSettingsTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := h.KindPrefDisable(ctx, testCustomerA, testAgentA, notification.KindChatMessage); err == nil {
		t.Errorf("Wrong match. a canceled context must not look like a success")
	}
	if err := h.MuteCreate(ctx, testCustomerA, testAgentA, testChatA); err == nil {
		t.Errorf("Wrong match. a canceled context must not look like a success")
	}
	if _, err := h.KindPrefListDisabledByAgent(ctx, testCustomerA, testAgentA); err == nil {
		t.Errorf("Wrong match. expect: error")
	}
	if _, err := h.MuteListByAgent(ctx, testCustomerA, testAgentA); err == nil {
		t.Errorf("Wrong match. expect: error")
	}
}

func Test_Settings_deleteByOwner(t *testing.T) {
	h, _ := newSettingsTestHandler(t)
	ctx := context.Background()

	seed := func() {
		_ = h.KindPrefDisable(ctx, testCustomerA, testAgentA, notification.KindChatMessage)
		_ = h.KindPrefDisable(ctx, testCustomerA, testAgentB, notification.KindChatMessage)
		_ = h.KindPrefDisable(ctx, testCustomerB, testAgentC, notification.KindChatMessage)
		_ = h.MuteCreate(ctx, testCustomerA, testAgentA, testChatA)
		_ = h.MuteCreate(ctx, testCustomerA, testAgentB, testChatA)
		_ = h.MuteCreate(ctx, testCustomerB, testAgentC, testChatA)
	}

	seed()
	if err := h.KindPrefDeleteByAgentID(ctx, testAgentA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.MuteDeleteByAgentID(ctx, testAgentA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if n := countRows(t, "push_kind_prefs"); n != 2 {
		t.Errorf("Wrong match. expect: 2 rows, got: %d", n)
	}
	if n := countRows(t, "push_chat_mutes"); n != 2 {
		t.Errorf("Wrong match. expect: 2 rows, got: %d", n)
	}

	if err := h.KindPrefDeleteByCustomerID(ctx, testCustomerA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.MuteDeleteByCustomerID(ctx, testCustomerA); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if n := countRows(t, "push_kind_prefs"); n != 1 {
		t.Errorf("Wrong match. expect: 1 row (customer B), got: %d", n)
	}
	if n := countRows(t, "push_chat_mutes"); n != 1 {
		t.Errorf("Wrong match. expect: 1 row (customer B), got: %d", n)
	}
}

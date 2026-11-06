package dbhandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/cachehandler"
)

func newIfTestMessage(id string, direction message.Direction, targets []target.Target) *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil("6a5a0b1e-1c3f-11ee-9d6c-0b3a8c9a1b01"),
		},
		Type:      message.TypeSMS,
		Source:    &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		Targets:   targets,
		Text:      "hello",
		Medias:    []string{},
		Direction: direction,
	}
}

func Test_MessageUpdateTargetsIf(t *testing.T) {
	t0 := timePtr(time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC))
	t1 := timePtr(time.Date(2026, 11, 1, 10, 0, 1, 123456000, time.UTC))
	t2 := timePtr(time.Date(2026, 11, 1, 10, 0, 2, 0, time.UTC))

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	m := newIfTestMessage("11111111-0000-4000-8000-000000000001", message.DirectionOutbound, []target.Target{
		{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000002"}, Status: target.StatusQueued},
	})
	mockUtil.EXPECT().TimeNow().Return(t0)
	mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
	if err := h.MessageCreate(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}

	newTargets := []target.Target{
		{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000002"}, Status: target.StatusSent, ProviderReferenceID: "ref-1"},
	}
	provider := message.ProviderNameTelnyx

	// 1. the message was never updated, so tm_update IS NULL matches.
	mockUtil.EXPECT().TimeNow().Return(t1)
	mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
	ok, err := h.MessageUpdateTargetsIf(ctx, m.ID, nil, &provider, newTargets)
	if err != nil || !ok {
		t.Fatalf("Wrong match. expect: true, got: %v, err: %v", ok, err)
	}

	// 2. a writer that still holds the old (NULL) tm_update loses.
	mockUtil.EXPECT().TimeNow().Return(t2)
	ok, err = h.MessageUpdateTargetsIf(ctx, m.ID, nil, nil, newTargets)
	if err != nil || ok {
		t.Fatalf("Wrong match. expect: false, got: %v, err: %v", ok, err)
	}

	// 3. the writer that read the row after the first write wins. The tm_update is read back from the database.
	got, err := h.messageGetFromDB(ctx, m.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TMUpdate == nil || !got.TMUpdate.Equal(*t1) {
		t.Fatalf("Wrong tm_update. expect: %v, got: %v", t1, got.TMUpdate)
	}
	if got.ProviderName != message.ProviderNameTelnyx || got.Targets[0].ProviderReferenceID != "ref-1" {
		t.Fatalf("Wrong stored row. got: %v", got)
	}

	mockUtil.EXPECT().TimeNow().Return(t2)
	mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
	ok, err = h.MessageUpdateTargetsIf(ctx, m.ID, got.TMUpdate, nil, newTargets)
	if err != nil || !ok {
		t.Fatalf("Wrong match. expect: true, got: %v, err: %v", ok, err)
	}

	// the provider name was not given, so it is kept.
	got, _ = h.messageGetFromDB(ctx, m.ID)
	if got.ProviderName != message.ProviderNameTelnyx {
		t.Errorf("Wrong provider name. got: %s", got.ProviderName)
	}
}

func Test_MessageListByProviderReference(t *testing.T) {
	now := timePtr(time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC))
	old := timePtr(time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC))
	ref := "4031a1af-0000-4000-8000-00000000aaaa"

	tgt := func(r string) []target.Target {
		return []target.Target{{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000003"}, Status: target.StatusQueued, ProviderReferenceID: r}}
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	create := func(m *message.Message, at *time.Time) {
		mockUtil.EXPECT().TimeNow().Return(at)
		mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
		if err := h.MessageCreate(ctx, m); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	hit := newIfTestMessage("22222222-0000-4000-8000-000000000001", message.DirectionOutbound, tgt(ref))
	create(hit, now)
	create(newIfTestMessage("22222222-0000-4000-8000-000000000002", message.DirectionOutbound, tgt("other")), now)
	create(newIfTestMessage("22222222-0000-4000-8000-000000000003", message.DirectionInbound, tgt(ref)), now)
	create(newIfTestMessage("22222222-0000-4000-8000-000000000004", message.DirectionOutbound, tgt(ref)), old)
	deleted := newIfTestMessage("22222222-0000-4000-8000-000000000005", message.DirectionOutbound, tgt(ref))
	create(deleted, now)
	mockUtil.EXPECT().TimeNow().Return(now)
	mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
	if err := h.MessageDelete(ctx, deleted.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	res, err := h.MessageListByProviderReference(ctx, ref, time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(res) != 1 || res[0].ID != hit.ID {
		t.Errorf("Wrong match. expect only the outbound, not deleted, recent message. got: %d rows", len(res))
	}

	for _, bad := range []string{"", "a%b", "a_b", `a\b`} {
		if _, err := h.MessageListByProviderReference(ctx, bad, time.Time{}); err == nil {
			t.Errorf("Wrong match. expect: error for %q", bad)
		}
	}
}

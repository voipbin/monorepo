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

func Test_InteractionCreate(t *testing.T) {
	tests := []struct {
		name        string
		interaction *interaction.Interaction
	}{
		{
			name: "normal incoming call",

			interaction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001"),
				CustomerID:    uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000002"),
				Direction:     "incoming",
				PeerType:      "tel",
				PeerTarget:    "peerTarget-call-incoming",
				LocalType:     "tel",
				LocalTarget:   "localTarget-call-incoming",
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000003"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC); return &t }(),
			},
		},
		{
			name: "outgoing conversation message LINE",

			interaction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000001"),
				CustomerID:    uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000002"),
				Direction:     "outgoing",
				PeerType:      "line",
				PeerTarget:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				LocalType:     "line",
				LocalTarget:   "",
				ReferenceType: "conversation_message",
				ReferenceID:   uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000003"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC); return &t }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// InteractionCreate does NOT call utilHandler or cache.
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			if err := h.InteractionCreate(ctx, tt.interaction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_InteractionCreate_duplicate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	i := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000001"),
		CustomerID:    uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000002"),
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "uniquePeerTarget-idem-001",
		LocalType:     "tel",
		LocalTarget:   "uniqueLocalTarget-idem-001",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000003"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC); return &t }(),
	}

	// first insert
	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// duplicate insert — must NOT error (idempotent at-least-once guard)
	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Errorf("duplicate insert should be idempotent (no error), got: %v", err)
	}
}

func Test_InteractionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	tm := func() *time.Time { t := time.Date(2026, 6, 28, 14, 0, 0, 0, time.UTC); return &t }()
	i := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0001-0001-0001-000000000001"),
		CustomerID:    uuid.FromStringOrNil("e1b2c3d4-0001-0001-0001-000000000002"),
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+15550001001",
		LocalType:     "tel",
		LocalTarget:   "+15559990001",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0001-0001-0001-000000000003"),
		TMCreate:      tm,
	}

	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Fatalf("InteractionCreate() error = %v", err)
	}

	// get by ID → verify fields
	got, err := h.InteractionGet(ctx, i.ID)
	if err != nil {
		t.Fatalf("InteractionGet() error = %v", err)
	}
	if got.ID != i.ID {
		t.Errorf("InteractionGet() ID = %v, want %v", got.ID, i.ID)
	}
	if got.PeerTarget != i.PeerTarget {
		t.Errorf("InteractionGet() PeerTarget = %v, want %v", got.PeerTarget, i.PeerTarget)
	}
	if got.Direction != i.Direction {
		t.Errorf("InteractionGet() Direction = %v, want %v", got.Direction, i.Direction)
	}

	// get non-existent ID → verify ErrNotFound
	_, err = h.InteractionGet(ctx, uuid.FromStringOrNil("e1b2c3d4-ffff-ffff-ffff-ffffffffffff"))
	if err != ErrNotFound {
		t.Errorf("InteractionGet() expected ErrNotFound, got: %v", err)
	}
}

func Test_InteractionList_byPeer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e1b2c3d4-0002-0002-0002-000000000001")
	peerType := "tel"
	peerTarget := "+15550002001"

	i1 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0002-0002-0002-000000000002"),
		CustomerID:    customerID,
		Direction:     "incoming",
		PeerType:      peerType,
		PeerTarget:    peerTarget,
		LocalType:     "tel",
		LocalTarget:   "+15559990002",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0002-0002-0002-000000000003"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 15, 0, 0, 0, time.UTC); return &t }(),
	}
	i2 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0002-0002-0002-000000000004"),
		CustomerID:    customerID,
		Direction:     "outgoing",
		PeerType:      peerType,
		PeerTarget:    peerTarget,
		LocalType:     "tel",
		LocalTarget:   "+15559990002",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0002-0002-0002-000000000005"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 16, 0, 0, 0, time.UTC); return &t }(),
	}

	if err := h.InteractionCreate(ctx, i1); err != nil {
		t.Fatalf("InteractionCreate(i1) error = %v", err)
	}
	if err := h.InteractionCreate(ctx, i2); err != nil {
		t.Fatalf("InteractionCreate(i2) error = %v", err)
	}

	// InteractionList with peerType+peerTarget → both returned
	res, err := h.InteractionList(ctx, customerID, 10, "", peerType, peerTarget, nil)
	if err != nil {
		t.Fatalf("InteractionList() error = %v", err)
	}
	if len(res) != 2 {
		t.Errorf("InteractionList() len = %d, want 2", len(res))
	}

	// InteractionList with wrong peer → empty
	res, err = h.InteractionList(ctx, customerID, 10, "", peerType, "+19990000000", nil)
	if err != nil {
		t.Fatalf("InteractionList() wrong peer error = %v", err)
	}
	if len(res) != 0 {
		t.Errorf("InteractionList() wrong peer len = %d, want 0", len(res))
	}
}

func Test_InteractionList_byAddressSet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e1b2c3d4-0003-0003-0003-000000000001")

	i1 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0003-0003-0003-000000000002"),
		CustomerID:    customerID,
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+15550003001",
		LocalType:     "tel",
		LocalTarget:   "+15559990003",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0003-0003-0003-000000000003"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 17, 0, 0, 0, time.UTC); return &t }(),
	}
	i2 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0003-0003-0003-000000000004"),
		CustomerID:    customerID,
		Direction:     "outgoing",
		PeerType:      "line",
		PeerTarget:    "Ufake12345",
		LocalType:     "line",
		LocalTarget:   "",
		ReferenceType: "conversation_message",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0003-0003-0003-000000000005"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 18, 0, 0, 0, time.UTC); return &t }(),
	}

	if err := h.InteractionCreate(ctx, i1); err != nil {
		t.Fatalf("InteractionCreate(i1) error = %v", err)
	}
	if err := h.InteractionCreate(ctx, i2); err != nil {
		t.Fatalf("InteractionCreate(i2) error = %v", err)
	}

	// InteractionList with addressSet containing both pairs → both returned
	bothPairs := []AddressPair{
		{Type: "tel", Target: "+15550003001"},
		{Type: "line", Target: "Ufake12345"},
	}
	res, err := h.InteractionList(ctx, customerID, 10, "", "", "", bothPairs)
	if err != nil {
		t.Fatalf("InteractionList() both pairs error = %v", err)
	}
	if len(res) != 2 {
		t.Errorf("InteractionList() both pairs len = %d, want 2", len(res))
	}

	// InteractionList with addressSet containing only first pair → 1 returned
	onePair := []AddressPair{
		{Type: "tel", Target: "+15550003001"},
	}
	res, err = h.InteractionList(ctx, customerID, 10, "", "", "", onePair)
	if err != nil {
		t.Fatalf("InteractionList() one pair error = %v", err)
	}
	if len(res) != 1 {
		t.Errorf("InteractionList() one pair len = %d, want 1", len(res))
	}
}

func Test_InteractionList_empty(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e1b2c3d4-0004-0004-0004-000000000001")

	// InteractionList with all-empty filters → no error, nil result
	res, err := h.InteractionList(ctx, customerID, 10, "", "", "", nil)
	if err != nil {
		t.Fatalf("InteractionList() empty filters error = %v", err)
	}
	if res != nil {
		t.Errorf("InteractionList() empty filters expected nil, got %v", res)
	}
}

func Test_InteractionListByIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000001")
	wrongCustomerID := uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000099")

	i1 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000002"),
		CustomerID:    customerID,
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "+15550005001",
		LocalType:     "tel",
		LocalTarget:   "+15559990005",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000003"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 19, 0, 0, 0, time.UTC); return &t }(),
	}
	i2 := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000004"),
		CustomerID:    customerID,
		Direction:     "outgoing",
		PeerType:      "tel",
		PeerTarget:    "+15550005002",
		LocalType:     "tel",
		LocalTarget:   "+15559990005",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("e1b2c3d4-0005-0005-0005-000000000005"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC); return &t }(),
	}

	if err := h.InteractionCreate(ctx, i1); err != nil {
		t.Fatalf("InteractionCreate(i1) error = %v", err)
	}
	if err := h.InteractionCreate(ctx, i2); err != nil {
		t.Fatalf("InteractionCreate(i2) error = %v", err)
	}

	// list by their IDs → both returned
	res, err := h.InteractionListByIDs(ctx, customerID, []uuid.UUID{i1.ID, i2.ID})
	if err != nil {
		t.Fatalf("InteractionListByIDs() error = %v", err)
	}
	if len(res) != 2 {
		t.Errorf("InteractionListByIDs() len = %d, want 2", len(res))
	}

	// list with wrong customerID → empty (tenant guard)
	res, err = h.InteractionListByIDs(ctx, wrongCustomerID, []uuid.UUID{i1.ID, i2.ID})
	if err != nil {
		t.Fatalf("InteractionListByIDs() wrong customer error = %v", err)
	}
	if len(res) != 0 {
		t.Errorf("InteractionListByIDs() wrong customer len = %d, want 0", len(res))
	}
}

// Test_InteractionList_pagination verifies that InteractionList correctly
// returns size+1 rows when callers pass size+1, enabling buildListResponse to
// detect hasMore and emit a non-empty NextPageToken.
func Test_InteractionList_pagination(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	// UUID namespace: f1b2c3d4-* (pagination tests — 'f' is valid hex, unlike 'p')
	customerID := uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000000")

	// Insert 3 interactions with the same peer so they all match.
	rows := []struct {
		id    uuid.UUID
		refID uuid.UUID
		tmC   time.Time
	}{
		{
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000001"),
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000011"),
			time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		},
		{
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000002"),
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000012"),
			time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC),
		},
		{
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000003"),
			uuid.FromStringOrNil("f1b2c3d4-0001-0001-0001-000000000013"),
			time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC),
		},
	}
	for _, r := range rows {
		tm := r.tmC
		i := &interaction.Interaction{
			ID:            r.id,
			CustomerID:    customerID,
			Direction:     "incoming",
			PeerType:      "tel",
			PeerTarget:    "+15550001111",
			LocalType:     "tel",
			LocalTarget:   "+15559999",
			ReferenceType: "call",
			ReferenceID:   r.refID,
			TMCreate:      &tm,
		}
		if err := h.InteractionCreate(ctx, i); err != nil {
			t.Fatalf("InteractionCreate error = %v", err)
		}
	}

	// pageSize=2, pass size+1=3 → expect 3 rows returned (probe row present)
	const pageSize uint64 = 2
	res, err := h.InteractionList(ctx, customerID, pageSize+1, "", "tel", "+15550001111", nil)
	if err != nil {
		t.Fatalf("InteractionList() error = %v", err)
	}
	// Should get 3 rows (all 3 exist); caller uses len>pageSize to detect hasMore.
	if uint64(len(res)) != pageSize+1 {
		t.Fatalf("InteractionList() len = %d, want %d (probe row present)", len(res), pageSize+1)
	}

	// First page: take first pageSize rows, encode token from last.
	page1 := res[:pageSize]
	probe := res[pageSize]
	_ = probe // confirms probe row exists

	lastOfPage1 := page1[len(page1)-1]
	if lastOfPage1.TMCreate == nil {
		t.Fatal("last item TMCreate is nil; cannot build page token")
	}
	nextToken := EncodePageToken(lastOfPage1.TMCreate, lastOfPage1.ID)
	if nextToken == "" {
		t.Fatal("EncodePageToken returned empty string for non-nil TMCreate")
	}

	// Second page: pass token, expect 1 row (the remaining one).
	res2, err := h.InteractionList(ctx, customerID, pageSize+1, nextToken, "tel", "+15550001111", nil)
	if err != nil {
		t.Fatalf("InteractionList page2 error = %v", err)
	}
	// Only 1 row should remain past the cursor; len ≤ pageSize → no more pages.
	if len(res2) == 0 {
		t.Error("InteractionList page2 returned 0 rows; expected 1")
	}
	if uint64(len(res2)) > pageSize {
		t.Errorf("InteractionList page2 len = %d; expected ≤ %d (no more data)", len(res2), pageSize)
	}
}

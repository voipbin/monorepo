package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_AIPromptHistoryCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC); return &t }()

	tests := []struct {
		name string

		prompt *aiprompthistory.AIPromptHistory

		responseCurTime *time.Time
		expectRes       *aiprompthistory.AIPromptHistory
	}{
		{
			name: "normal",

			prompt: &aiprompthistory.AIPromptHistory{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000002"),
				},
				AIID:   uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000003"),
				Prompt: "You are a helpful assistant.",
			},

			responseCurTime: curTime,
			expectRes: &aiprompthistory.AIPromptHistory{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000002"),
				},
				AIID:     uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000003"),
				Prompt:   "You are a helpful assistant.",
				TMCreate: curTime,
			},
		},
		{
			name: "empty prompt",

			prompt: &aiprompthistory.AIPromptHistory{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
				},
				AIID:   uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000003"),
				Prompt: "",
			},

			responseCurTime: curTime,
			expectRes: &aiprompthistory.AIPromptHistory{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
				},
				AIID:     uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000003"),
				Prompt:   "",
				TMCreate: curTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.AIPromptHistoryCreate(ctx, tt.prompt); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			res, err := h.AIPromptHistoryGet(ctx, tt.prompt.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptHistoryGet_NotFound(t *testing.T) {
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

	_, err := h.AIPromptHistoryGet(ctx, uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"))
	if err != ErrNotFound {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_AIPromptHistoryGetsByAIID(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC); return &t }()
	laterTime := func() *time.Time { t := time.Date(2025, 2, 1, 0, 0, 1, 0, time.UTC); return &t }()
	evenLaterTime := func() *time.Time { t := time.Date(2025, 2, 1, 0, 0, 2, 0, time.UTC); return &t }()

	aiID := uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000002")

	prompts := []*aiprompthistory.AIPromptHistory{
		{
			Identity: identity.Identity{
				ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000011"),
				CustomerID: customerID,
			},
			AIID:   aiID,
			Prompt: "v1",
		},
		{
			Identity: identity.Identity{
				ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000012"),
				CustomerID: customerID,
			},
			AIID:   aiID,
			Prompt: "v2",
		},
		{
			Identity: identity.Identity{
				ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000013"),
				CustomerID: customerID,
			},
			AIID:   aiID,
			Prompt: "v3",
		},
	}

	times := []*time.Time{curTime, laterTime, evenLaterTime}

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

	// Insert 3 entries with distinct timestamps
	for i, p := range prompts {
		mockUtil.EXPECT().TimeNow().Return(times[i])
		if err := h.AIPromptHistoryCreate(ctx, p); err != nil {
			t.Fatalf("setup create failed: %v", err)
		}
	}

	// Fetch all 3 — newest first (v3, v2, v1)
	res, err := h.AIPromptHistoryGetsByAIID(ctx, aiID, 10, utilhandler.TimeGetCurTime())
	if err != nil {
		t.Fatalf("AIPromptHistoryGetsByAIID failed: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("Wrong count. expect: 3, got: %d", len(res))
	}
	// newest first
	if res[0].Prompt != "v3" {
		t.Errorf("Wrong order. expect first: v3, got: %s", res[0].Prompt)
	}
	if res[2].Prompt != "v1" {
		t.Errorf("Wrong order. expect last: v1, got: %s", res[2].Prompt)
	}
}

func Test_AIPromptHistoryGetsByAIID_Empty(t *testing.T) {
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

	res, err := h.AIPromptHistoryGetsByAIID(ctx, uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"), 10, utilhandler.TimeGetCurTime())
	if err != nil {
		t.Fatalf("AIPromptHistoryGetsByAIID failed: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong count. expect: 0, got: %d", len(res))
	}
}

func Test_AIPromptHistoryGetsByAIID_Pagination(t *testing.T) {

	t0 := func() *time.Time { t := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC); return &t }()
	t1 := func() *time.Time { t := time.Date(2025, 3, 1, 0, 0, 1, 0, time.UTC); return &t }()
	t2 := func() *time.Time { t := time.Date(2025, 3, 1, 0, 0, 2, 0, time.UTC); return &t }()
	t3 := func() *time.Time { t := time.Date(2025, 3, 1, 0, 0, 3, 0, time.UTC); return &t }()
	t4 := func() *time.Time { t := time.Date(2025, 3, 1, 0, 0, 4, 0, time.UTC); return &t }()

	aiID := uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000002")

	type entry struct {
		id     uuid.UUID
		prompt string
		ts     *time.Time
	}
	entries := []entry{
		{uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000011"), "p1", t0},
		{uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000012"), "p2", t1},
		{uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000013"), "p3", t2},
		{uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000014"), "p4", t3},
		{uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000015"), "p5", t4},
	}

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

	for _, e := range entries {
		p := &aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{
				ID:         e.id,
				CustomerID: customerID,
			},
			AIID:   aiID,
			Prompt: e.prompt,
		}
		mockUtil.EXPECT().TimeNow().Return(e.ts)
		if err := h.AIPromptHistoryCreate(ctx, p); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	// First page: size=2 => p5, p4
	page1, err := h.AIPromptHistoryGetsByAIID(ctx, aiID, 2, utilhandler.TimeGetCurTime())
	if err != nil {
		t.Fatalf("page1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("Wrong page1 count. expect: 2, got: %d", len(page1))
	}
	if page1[0].Prompt != "p5" {
		t.Errorf("Wrong page1[0]. expect: p5, got: %s", page1[0].Prompt)
	}
	if page1[1].Prompt != "p4" {
		t.Errorf("Wrong page1[1]. expect: p4, got: %s", page1[1].Prompt)
	}

	// Second page: token = tm_create of p4 (last in page1).
	// Use the SQLite-compatible datetime format for comparison in tests.
	token := page1[len(page1)-1].TMCreate.UTC().Format("2006-01-02 15:04:05-07:00")
	page2, err := h.AIPromptHistoryGetsByAIID(ctx, aiID, 2, token)
	if err != nil {
		t.Fatalf("page2 failed: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("Wrong page2 count. expect: 2, got: %d", len(page2))
	}
	if page2[0].Prompt != "p3" {
		t.Errorf("Wrong page2[0]. expect: p3, got: %s", page2[0].Prompt)
	}
}

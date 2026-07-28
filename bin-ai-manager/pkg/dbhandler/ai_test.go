package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_AICreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string

		ai *ai.AI

		responseCurTime *time.Time

		expectRes *ai.AI
	}{
		{
			name: "have all",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				Parameter: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "test engine key",
				InitPrompt: "test init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "test tts voice id",
				STTType:    ai.STTTypeElevenLabs,
			},

			responseCurTime: curTime,
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				Parameter: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "test engine key",
				InitPrompt: "test init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "test tts voice id",
				STTType:    ai.STTTypeElevenLabs,

				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
		{
			name: "empty",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
			},

			responseCurTime: curTime,
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
				Parameter: nil,
				TMCreate:   curTime,
				TMUpdate:   nil,
				TMDelete:   nil,
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
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if err := h.AICreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIGet(ctx, tt.ai.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			res, err := h.AIGet(ctx, tt.ai.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIDelete(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *ai.AI

		id uuid.UUID

		responseCurTime *time.Time
		expectRes       *ai.AI
	}{
		{
			name: "normal",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
			},

			id: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),

			responseCurTime: curTime,
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
				Parameter: nil,
				TMCreate:   curTime,
				TMUpdate:   curTime,
				TMDelete:   curTime,
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
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if err := h.AICreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if errDel := h.AIDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().AIGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			res, err := h.AIGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_AIList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ais  []*ai.AI

		count   int
		filters map[ai.Field]any

		responseCurTime *time.Time
		expectRes       []*ai.AI
	}{
		{
			name: "normal",
			ais: []*ai.AI{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
				},
			},

			count: 10,
			filters: map[ai.Field]any{
				ai.FieldDeleted:    false,
				ai.FieldCustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			},

			responseCurTime: curTime,
			expectRes: []*ai.AI{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					Parameter: nil,
					TMCreate:   curTime,
					TMUpdate:   nil,
					TMDelete:   nil,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					Parameter: nil,
					TMCreate:   curTime,
					TMUpdate:   nil,
					TMDelete:   nil,
				},
			},
		},
		{
			name: "empty",
			ais:  []*ai.AI{},

			count: 0,
			filters: map[ai.Field]any{
				ai.FieldDeleted:    false,
				ai.FieldCustomerID: uuid.FromStringOrNil("b31d32ae-7f45-11ec-82c6-936e22306376"),
			},

			responseCurTime: curTime,
			expectRes:       []*ai.AI{},
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

			for _, cf := range tt.ais {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AISet(ctx, gomock.Any())
				if errCreate := h.AICreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.AIList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_AIUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *ai.AI

		id     uuid.UUID
		fields map[ai.Field]any

		responseCurTime *time.Time
		expectRes       *ai.AI
	}{
		{
			name: "normal",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
			},

			id: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
			fields: map[ai.Field]any{
				ai.FieldName:        "new name",
				ai.FieldDetail:      "new detail",
				ai.FieldEngineModel: ai.EngineModelOpenaiGPT5Nano,
				ai.FieldParameter: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				ai.FieldEngineKey:  "new engine key",
				ai.FieldInitPrompt: "new init prompt",
				ai.FieldTTSType:    ai.TTSTypeCartesia,
				ai.FieldTTSVoiceID: "new tts voice id",
				ai.FieldSTTType:    ai.STTTypeElevenLabs,
			},

			responseCurTime: curTime,
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
				Name:        "new name",
				Detail:      "new detail",
				EngineModel: ai.EngineModelOpenaiGPT5Nano,
				Parameter: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "new engine key",
				InitPrompt: "new init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "new tts voice id",
				STTType:    ai.STTTypeElevenLabs,
				TMCreate:   curTime,
				TMUpdate:   curTime,
				TMDelete:   nil,
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
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if err := h.AICreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if errUpdate := h.AIUpdate(ctx, tt.id, tt.fields); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
			}

			mockCache.EXPECT().AIGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			res, err := h.AIGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_AICreate_InsightUniquePerCustomer(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().AISet(ctx, gomock.Any()).AnyTimes()

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	customerID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001")

	// first insight AI for the customer succeeds
	firstID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000002")
	t1 := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t1)
	first := &ai.AI{
		Identity: identity.Identity{ID: firstID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	if err := h.AICreate(ctx, first); err != nil {
		t.Fatalf("first insight AICreate: expected ok, got: %v", err)
	}

	// second insight AI for the SAME customer must be rejected
	secondID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000003")
	t2 := time.Date(2026, 7, 29, 0, 0, 1, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t2)
	second := &ai.AI{
		Identity: identity.Identity{ID: secondID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	err := h.AICreate(ctx, second)
	if err == nil {
		t.Fatal("second insight AICreate: expected an error, got nil")
	}
	if !IsErrDuplicate(err) {
		t.Errorf("second insight AICreate: expected IsErrDuplicate(err)=true, got err: %v", err)
	}

	// a normal-type AI for the same customer is NOT affected by the constraint
	normalID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000004")
	t3 := time.Date(2026, 7, 29, 0, 0, 2, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t3)
	normal := &ai.AI{
		Identity: identity.Identity{ID: normalID, CustomerID: customerID},
		Type:     ai.TypeNormal,
	}
	if err := h.AICreate(ctx, normal); err != nil {
		t.Errorf("normal-type AICreate for same customer: expected ok, got: %v", err)
	}

	// soft-delete the first insight AI, freeing the slot.
	// AIDelete's real cache interaction is TimeNow + aiUpdateToCache (which
	// reads back via a DB query, not cache.AIGet, then calls cache.AISet) —
	// see bin-ai-manager/pkg/dbhandler/ai.go's AIDelete and aiUpdateToCache.
	// The AISet call is already covered by the mockCache.EXPECT().AISet(...).AnyTimes()
	// set up above; AIDelete never calls cache.AIGet, so no such expectation
	// is set here (unlike the separate, subsequent h.AIGet call pattern used
	// in the existing Test_AIDelete test, which does call cache.AIGet).
	t4 := time.Date(2026, 7, 29, 0, 0, 3, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t4)
	if err := h.AIDelete(ctx, firstID); err != nil {
		t.Fatalf("AIDelete(first): expected ok, got: %v", err)
	}

	// a new insight AI for the same customer now succeeds
	thirdID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000005")
	t5 := time.Date(2026, 7, 29, 0, 0, 4, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t5)
	third := &ai.AI{
		Identity: identity.Identity{ID: thirdID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	if err := h.AICreate(ctx, third); err != nil {
		t.Errorf("insight AICreate after soft-delete: expected ok, got: %v", err)
	}
}

package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_AICreate(t *testing.T) {

	tests := []struct {
		name string

		ai *ai.AI

		responseCurTime string

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
				EngineType: ai.EngineTypeNone,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "test engine key",
				InitPrompt: "test init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "test tts voice id",
				STTType:    ai.STTTypeElevenLabs,
			},

			responseCurTime: "2023-01-03T21:35:02.809Z",
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				EngineType: ai.EngineTypeNone,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "test engine key",
				InitPrompt: "test init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "test tts voice id",
				STTType:    ai.STTTypeElevenLabs,

				TMCreate: "2023-01-03T21:35:02.809Z",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "empty",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
			},

			responseCurTime: "2023-01-03T21:35:02.809Z",
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
				EngineData: nil,
				TMCreate:   "2023-01-03T21:35:02.809Z",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

	tests := []struct {
		name string
		ai   *ai.AI

		id uuid.UUID

		responseCurTime string
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

			responseCurTime: "2023-01-03T21:35:02.809Z",
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
				EngineData: nil,
				TMCreate:   "2023-01-03T21:35:02.809Z",
				TMUpdate:   "2023-01-03T21:35:02.809Z",
				TMDelete:   "2023-01-03T21:35:02.809Z",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if err := h.AICreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

	tests := []struct {
		name string
		ais  []*ai.AI

		count   int
		filters map[ai.Field]any

		responseCurTime string
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

			responseCurTime: "2023-01-03T21:35:02.809Z",
			expectRes: []*ai.AI{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					EngineData: nil,
					TMCreate:   "2023-01-03T21:35:02.809Z",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					EngineData: nil,
					TMCreate:   "2023-01-03T21:35:02.809Z",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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

			responseCurTime: "2023-01-03T21:35:02.809Z",
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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

	tests := []struct {
		name string
		ai   *ai.AI

		id     uuid.UUID
		fields map[ai.Field]any

		responseCurTime string
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
				ai.FieldEngineType:  ai.EngineTypeNone,
				ai.FieldEngineModel: ai.EngineModelOpenaiGPT3Dot5Turbo,
				ai.FieldEngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				ai.FieldEngineKey:  "new engine key",
				ai.FieldInitPrompt: "new init prompt",
				ai.FieldTTSType:    ai.TTSTypeCartesia,
				ai.FieldTTSVoiceID: "new tts voice id",
				ai.FieldSTTType:    ai.STTTypeElevenLabs,
			},

			responseCurTime: "2023-01-03T21:35:02.809Z",
			expectRes: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
				Name:        "new name",
				Detail:      "new detail",
				EngineType:  ai.EngineTypeNone,
				EngineModel: ai.EngineModelOpenaiGPT3Dot5Turbo,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				EngineKey:  "new engine key",
				InitPrompt: "new init prompt",
				TTSType:    ai.TTSTypeCartesia,
				TTSVoiceID: "new tts voice id",
				STTType:    ai.STTTypeElevenLabs,
				TMCreate:   "2023-01-03T21:35:02.809Z",
				TMUpdate:   "2023-01-03T21:35:02.809Z",
				TMDelete:   DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AISet(ctx, gomock.Any())
			if err := h.AICreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

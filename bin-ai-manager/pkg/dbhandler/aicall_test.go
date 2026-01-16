package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_AIcallCreate(t *testing.T) {

	tests := []struct {
		name string

		ai *aicall.AIcall

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			name: "have all",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
					CustomerID: uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				},
				AIID:          uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				AIEngineType:  ai.EngineTypeNone,
				AIEngineModel: ai.EngineModelOpenaiGPT4Turbo,
				AIEngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "elevenlabs-voice-001",
				AISTTType:     ai.STTTypeDeepgram,
				ActiveflowID:  uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:  uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				PipecatcallID: uuid.FromStringOrNil("c8f5048e-afbc-11f0-b7de-3f3a52b42500"),
				Status:        aicall.StatusInitiating,
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
					CustomerID: uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				},
				AIID:          uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				AIEngineType:  ai.EngineTypeNone,
				AIEngineModel: ai.EngineModelOpenaiGPT4Turbo,
				AIEngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "elevenlabs-voice-001",
				AISTTType:     ai.STTTypeDeepgram,
				ActiveflowID:  uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:  uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				PipecatcallID: uuid.FromStringOrNil("c8f5048e-afbc-11f0-b7de-3f3a52b42500"),
				Status:        aicall.StatusInitiating,
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				TMEnd:         DefaultTimeStamp,
				TMCreate:      "2023-01-03 21:35:02.809",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			"empty",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				},
			},

			"2023-01-03 21:35:02.809",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				},
				AIEngineData: nil,
				TMEnd:        DefaultTimeStamp,
				TMCreate:     "2023-01-03 21:35:02.809",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIcallGet(ctx, tt.ai.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			res, err := h.AIcallGet(ctx, tt.ai.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallGetByReferenceID(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		referenceID uuid.UUID

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			"normal",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a8b26464-a5e2-11ed-bce7-83b475b0c53d"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),
			},

			uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),

			"2023-01-03 21:35:02.809",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a8b26464-a5e2-11ed-bce7-83b475b0c53d"),
				},
				AIEngineData:  nil,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),
				TMEnd:         DefaultTimeStamp,
				TMCreate:      "2023-01-03 21:35:02.809",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			res, err := h.AIcallGetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallUpdate(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id     uuid.UUID
		fields map[aicall.Field]any

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			name: "update pipecatcall_id",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				},
				PipecatcallID: uuid.FromStringOrNil("f6ee5c0a-afbc-11f0-8049-c7a79d2e4fe8"),
			},

			id: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
			fields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("f720a0d4-afbc-11f0-954f-6ff64a2d4520"),
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				},
				AIEngineData:  nil,
				PipecatcallID: uuid.FromStringOrNil("f720a0d4-afbc-11f0-954f-6ff64a2d4520"),
				TMEnd:         DefaultTimeStamp,
				TMCreate:      "2023-01-03 21:35:02.809",
				TMUpdate:      "2023-01-03 21:35:02.809",
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			name: "update status",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
				},
			},

			id: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
			fields: map[aicall.Field]any{
				aicall.FieldStatus: aicall.StatusProgressing,
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
				},
				AIEngineData: nil,
				Status:       aicall.StatusProgressing,
				TMEnd:        DefaultTimeStamp,
				TMCreate:     "2023-01-03 21:35:02.809",
				TMUpdate:     "2023-01-03 21:35:02.809",
				TMDelete:     DefaultTimeStamp,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			res, err := h.AIcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallDelete(t *testing.T) {

	tests := []struct {
		name   string
		aicall *aicall.AIcall

		id uuid.UUID

		responseCurTime string
		expectRes       *aicall.AIcall
	}{
		{
			name: "normal",
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				},
			},

			id: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				},
				AIEngineData: nil,
				TMEnd:        DefaultTimeStamp,
				TMCreate:     "2023-01-03 21:35:02.809",
				TMUpdate:     "2023-01-03 21:35:02.809",
				TMDelete:     "2023-01-03 21:35:02.809",
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.aicall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if errDel := h.AIcallDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().AIcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			res, err := h.AIcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_AIcallList(t *testing.T) {

	tests := []struct {
		name    string
		aicalls []*aicall.AIcall

		count   int
		filters map[aicall.Field]any

		responseCurTime string
		expectRes       []*aicall.AIcall
	}{
		{
			name: "normal",
			aicalls: []*aicall.AIcall{
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
			filters: map[aicall.Field]any{
				aicall.FieldDeleted:    false,
				aicall.FieldCustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: []*aicall.AIcall{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					AIEngineData: nil,
					TMEnd:        DefaultTimeStamp,
					TMCreate:     "2023-01-03 21:35:02.809",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					AIEngineData: nil,
					TMEnd:        DefaultTimeStamp,
					TMCreate:     "2023-01-03 21:35:02.809",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			name:    "empty",
			aicalls: []*aicall.AIcall{},

			count: 0,
			filters: map[aicall.Field]any{
				aicall.FieldDeleted:    false,
				aicall.FieldCustomerID: uuid.FromStringOrNil("a819a17a-0ba7-11f0-94b8-77c77a198260"),
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes:       []*aicall.AIcall{},
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

			for _, cc := range tt.aicalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
				if errCreate := h.AIcallCreate(ctx, cc); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.AIcallList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

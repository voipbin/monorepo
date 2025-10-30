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
				ActiveflowID:      uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType:     aicall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:      uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				TranscribeID:      uuid.FromStringOrNil("e2c7cd7a-a5e1-11ed-9c3a-ef9305cb70cd"),
				PipecatcallID:     uuid.FromStringOrNil("a654893a-b530-11f0-8d2e-2f3b9545f1bb"),
				Status:            aicall.StatusInitiating,
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				TTSStreamingID:    uuid.FromStringOrNil("4c120406-5ba9-11f0-836b-039c8c430c18"),
				TTSStreamingPodID: "4c3f9c22-5ba9-11f0-bd29-e3a3162f5277",
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
				ActiveflowID:      uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType:     aicall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:      uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				TranscribeID:      uuid.FromStringOrNil("e2c7cd7a-a5e1-11ed-9c3a-ef9305cb70cd"),
				PipecatcallID:     uuid.FromStringOrNil("a654893a-b530-11f0-8d2e-2f3b9545f1bb"),
				Status:            aicall.StatusInitiating,
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				TTSStreamingID:    uuid.FromStringOrNil("4c120406-5ba9-11f0-836b-039c8c430c18"),
				TTSStreamingPodID: "4c3f9c22-5ba9-11f0-bd29-e3a3162f5277",
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
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
				AIEngineData: map[string]any{},
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
				AIEngineData:  map[string]any{},
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

func Test_AIcallGetByTranscribeID(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		transcribeID uuid.UUID

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			"normal",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ee65f8bc-a5e3-11ed-bc48-4fd434eda48d"),
				},
				TranscribeID: uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),
			},

			uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),

			"2023-01-03 21:35:02.809",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ee65f8bc-a5e3-11ed-bc48-4fd434eda48d"),
				},
				AIEngineData: map[string]any{},
				TranscribeID: uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),
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

			mockCache.EXPECT().AIcallGetByTranscribeID(ctx, tt.transcribeID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			res, err := h.AIcallGetByTranscribeID(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallUpdateStatusProgressing(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id           uuid.UUID
		transcribeID uuid.UUID

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			name: "normal",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
				},
			},

			id:           uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
			transcribeID: uuid.FromStringOrNil("e6342714-a5e2-11ed-a3dd-cbe7bf0cbcb0"),

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
				},
				AIEngineData: map[string]any{},
				TranscribeID: uuid.FromStringOrNil("e6342714-a5e2-11ed-a3dd-cbe7bf0cbcb0"),
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
			if err := h.AIcallUpdateStatusProgressing(ctx, tt.id, tt.transcribeID); err != nil {
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

func Test_AIcallUpdateStatusPausing(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id uuid.UUID

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			name: "normal",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("7664a584-0414-11f0-9866-0ff182c4e5cd"),
				},
			},

			id: uuid.FromStringOrNil("7664a584-0414-11f0-9866-0ff182c4e5cd"),

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("7664a584-0414-11f0-9866-0ff182c4e5cd"),
				},
				AIEngineData: map[string]any{},
				TranscribeID: uuid.Nil,
				Status:       aicall.StatusPausing,
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
			if err := h.AIcallUpdateStatusPausing(ctx, tt.id); err != nil {
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

func Test_AIcallUpdateStatusEnd(t *testing.T) {

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id uuid.UUID

		responseCurTime string

		expectRes *aicall.AIcall
	}{
		{
			name: "normal",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),
				},
				TranscribeID: uuid.FromStringOrNil("e9a4d8c2-e7ca-11ef-b80a-43dbe39bcce9"),
			},

			id: uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),
				},
				AIEngineData: map[string]any{},
				TranscribeID: uuid.Nil,
				Status:       aicall.StatusTerminated,
				TMEnd:        "2023-01-03 21:35:02.809",
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
			if err := h.AIcallUpdateStatusTerminated(ctx, tt.id); err != nil {
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
				AIEngineData: map[string]any{},
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

func Test_AIcallGets(t *testing.T) {

	tests := []struct {
		name    string
		aicalls []*aicall.AIcall

		count   int
		filters map[string]string

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
			filters: map[string]string{
				"deleted":     "false",
				"customer_id": "6d35368c-a76d-11ed-9699-235c9e4a0117",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: []*aicall.AIcall{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					AIEngineData: map[string]any{},
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
					AIEngineData: map[string]any{},
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
			filters: map[string]string{
				"deleted":     "false",
				"customer_id": "a819a17a-0ba7-11f0-94b8-77c77a198260",
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

			res, err := h.AIcallGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

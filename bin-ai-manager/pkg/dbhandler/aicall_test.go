package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_AIcallCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string

		ai *aicall.AIcall

		responseCurTime *time.Time

		expectRes *aicall.AIcall
	}{
		{
			name: "have all",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
					CustomerID: uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				AIEngineModel: ai.EngineModelOpenaiGPT5Dot1,
				Parameter: map[string]any{
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

				STTLanguage:   "en-US",
			},

			responseCurTime: curTime,
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
					CustomerID: uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				AIEngineModel: ai.EngineModelOpenaiGPT5Dot1,
				Parameter: map[string]any{
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

				STTLanguage:   "en-US",
				TMEnd:         nil,
				TMCreate:      curTime,
				TMUpdate:      nil,
				TMDelete:      nil,
			},
		},
		{
			"empty",
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				},
				ReferenceID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
			},

			curTime,
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				},
				Parameter:   nil,
				ReferenceID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				TMEnd:        nil,
				TMCreate:     curTime,
				TMUpdate:     nil,
				TMDelete:     nil,
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

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *aicall.AIcall

		referenceID uuid.UUID

		responseCurTime *time.Time

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

			curTime,
			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a8b26464-a5e2-11ed-bce7-83b475b0c53d"),
				},
				Parameter:  nil,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),
				TMEnd:         nil,
				TMCreate:      curTime,
				TMUpdate:      nil,
				TMDelete:      nil,
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

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id     uuid.UUID
		fields map[aicall.Field]any

		responseCurTime *time.Time

		expectRes *aicall.AIcall
	}{
		{
			name: "update pipecatcall_id",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				},
				ReferenceID:   uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				PipecatcallID: uuid.FromStringOrNil("f6ee5c0a-afbc-11f0-8049-c7a79d2e4fe8"),
			},

			id: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
			fields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("f720a0d4-afbc-11f0-954f-6ff64a2d4520"),
			},

			responseCurTime: curTime,
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				},
				Parameter:     nil,
				ReferenceID:   uuid.FromStringOrNil("f6c9d56a-afbc-11f0-bb5f-1b20049b3cfb"),
				PipecatcallID: uuid.FromStringOrNil("f720a0d4-afbc-11f0-954f-6ff64a2d4520"),
				TMEnd:         nil,
				TMCreate:      curTime,
				TMUpdate:      curTime,
				TMDelete:      nil,
			},
		},
		{
			name: "update status",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
				},
				ReferenceID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
			},

			id: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
			fields: map[aicall.Field]any{
				aicall.FieldStatus: aicall.StatusProgressing,
			},

			responseCurTime: curTime,
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
				},
				Parameter:   nil,
				ReferenceID: uuid.FromStringOrNil("f7c0bf02-b083-11f0-99e0-ffcbb19dc61e"),
				Status:      aicall.StatusProgressing,
				TMEnd:       nil,
				TMCreate:    curTime,
				TMUpdate:    curTime,
				TMDelete:    nil,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		aicall *aicall.AIcall

		id uuid.UUID

		responseCurTime *time.Time
		expectRes       *aicall.AIcall
	}{
		{
			name: "normal",
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				},
				ReferenceID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
			},

			id: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),

			responseCurTime: curTime,
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				},
				Parameter:   nil,
				ReferenceID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				TMEnd:        nil,
				TMCreate:     curTime,
				TMUpdate:     curTime,
				TMDelete:     curTime,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.aicall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

func Test_AIcallUpdateIfActive(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id     uuid.UUID
		fields map[aicall.Field]any

		responseCurTime *time.Time

		expectRowsAffected int64
	}{
		{
			name: "active aicall — update succeeds",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
				Status: aicall.StatusProgressing,
			},

			id: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			fields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},

			responseCurTime:    curTime,
			expectRowsAffected: 1,
		},
		{
			name: "terminated aicall — update is a no-op",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
				Status: aicall.StatusTerminated,
			},

			id: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			fields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			},

			responseCurTime:    curTime,
			expectRowsAffected: 0,
		},
		{
			name: "terminating aicall — update is a no-op",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				},
				Status: aicall.StatusTerminating,
			},

			id: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			fields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
			},

			responseCurTime:    curTime,
			expectRowsAffected: 0,
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
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if tt.expectRowsAffected > 0 {
				mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			}
			rowsAffected, err := h.AIcallUpdateIfActive(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if rowsAffected != tt.expectRowsAffected {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectRowsAffected, rowsAffected)
			}
		})
	}
}

func Test_IsErrDuplicate(t *testing.T) {

	tests := []struct {
		name string
		err  error

		expectRes bool
	}{
		{
			name:      "nil error",
			err:       nil,
			expectRes: false,
		},
		{
			name:      "mysql duplicate entry error",
			err:       fmt.Errorf("Error 1062: Duplicate entry 'abc' for key 'uq_aicall_active_reference_key'"),
			expectRes: true,
		},
		{
			name:      "sqlite unique constraint error",
			err:       fmt.Errorf("UNIQUE constraint failed: ai_aicalls.active_reference_key"),
			expectRes: true,
		},
		{
			name:      "unrelated error",
			err:       fmt.Errorf("connection refused"),
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsErrDuplicate(tt.err)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		aicalls []*aicall.AIcall

		count   int
		filters map[aicall.Field]any

		responseCurTime *time.Time
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
					ReferenceID: uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					ReferenceID: uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
				},
			},

			count: 10,
			filters: map[aicall.Field]any{
				aicall.FieldDeleted:    false,
				aicall.FieldCustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			},

			responseCurTime: curTime,
			expectRes: []*aicall.AIcall{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					Parameter:   nil,
					ReferenceID: uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
					TMEnd:        nil,
					TMCreate:     curTime,
					TMUpdate:     nil,
					TMDelete:     nil,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					Parameter:   nil,
					ReferenceID: uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					TMEnd:        nil,
					TMCreate:     curTime,
					TMUpdate:     nil,
					TMDelete:     nil,
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

			responseCurTime: curTime,
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
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

// Test_AIcallUpdateNoTouchTMUpdate pins that this variant leaves tm_update
// alone, unlike AIcallUpdate which always bumps it.
//
// This is not a micro-optimisation. Send()'s cooldown reads tm_update to decide
// whether to reject a message. Listening stops on call hangup -- exactly when an
// agent is most likely to ask the Insight AI a follow-up question -- so a
// tm_update bump from listen's own stop-time bookkeeping would reject a genuine
// question the agent just typed, for no reason the agent could understand.
func Test_AIcallUpdateNoTouchTMUpdate(t *testing.T) {

	createTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()
	updateTime := func() *time.Time { t := time.Date(2024, 6, 7, 11, 12, 13, 140000000, time.UTC); return &t }()

	tests := []struct {
		name string
		ai   *aicall.AIcall

		id     uuid.UUID
		fields map[aicall.Field]any

		expectListenCallID uuid.UUID
	}{
		{
			name: "listen_call_id is written and tm_update is left untouched",
			ai: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6f9a6d10-8a12-11f0-9e3a-8f2b1c4d5e60"),
				},
				ReferenceID: uuid.FromStringOrNil("6f9a6d10-8a12-11f0-9e3a-8f2b1c4d5e60"),
			},

			id: uuid.FromStringOrNil("6f9a6d10-8a12-11f0-9e3a-8f2b1c4d5e60"),
			fields: map[aicall.Field]any{
				aicall.FieldListenCallID: uuid.FromStringOrNil("70f01b34-8a12-11f0-a4c2-b30d9e2f61a7"),
			},

			expectListenCallID: uuid.FromStringOrNil("70f01b34-8a12-11f0-a4c2-b30d9e2f61a7"),
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

			mockUtil.EXPECT().TimeNow().Return(createTime)
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallCreate(ctx, tt.ai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Bump tm_update once through the ordinary path, so the assertion
			// below distinguishes "never set" from "not touched by this call".
			mockUtil.EXPECT().TimeNow().Return(updateTime)
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallUpdate(ctx, tt.id, map[aicall.Field]any{
				aicall.FieldStatus: aicall.StatusProgressing,
			}); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			before, err := h.AIcallGet(ctx, tt.id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// NO mockUtil.EXPECT().TimeNow() here on purpose: gomock fails the
			// test if AIcallUpdateNoTouchTMUpdate reaches for the clock at all,
			// which is the most direct possible assertion that it does not set
			// tm_update.
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			if err := h.AIcallUpdateNoTouchTMUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AIcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
			after, err := h.AIcallGet(ctx, tt.id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if after.ListenCallID != tt.expectListenCallID {
				t.Errorf("ListenCallID mismatch. expected: %s, got: %s", tt.expectListenCallID, after.ListenCallID)
			}
			if !reflect.DeepEqual(before.TMUpdate, after.TMUpdate) {
				t.Errorf("tm_update must be untouched. before: %v, after: %v", before.TMUpdate, after.TMUpdate)
			}
		})
	}
}

// Test_AIcallGetSkipCache pins that this variant never reads the cache, and
// still refreshes it with what it read -- a caller reaching for this has just
// established the cached copy was suspect, so leaving the stale copy in place
// would make the next ordinary AIcallGet wrong again.
func Test_AIcallGetSkipCache(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	id := uuid.FromStringOrNil("9c4b3a58-8a13-11f0-8f52-cf3a4e5b6d71")

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

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
	if err := h.AIcallCreate(ctx, &aicall.AIcall{
		Identity:    identity.Identity{ID: id},
		ReferenceID: id,
	}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// NO mockCache.EXPECT().AIcallGet here on purpose: gomock fails the test if
	// the cache is consulted at all. Only the write-back is expected.
	mockCache.EXPECT().AIcallSet(ctx, gomock.Any())
	res, err := h.AIcallGetSkipCache(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id {
		t.Errorf("id mismatch. expected: %s, got: %s", id, res.ID)
	}
}

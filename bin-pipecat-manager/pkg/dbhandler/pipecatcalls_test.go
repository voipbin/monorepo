package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-pipecat-manager/pkg/cachehandler"
)

func Test_PipecatcallsCreate(t *testing.T) {

	tests := []struct {
		name string

		pipecatcall *pipecatcall.Pipecatcall

		responseCurTime string

		expectedRes *pipecatcall.Pipecatcall
	}{
		{
			name: "have all",

			pipecatcall: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},

				ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),

				HostID: "1.2.3.4",

				LLMType:    pipecatcall.LLMType("openai.gpt-4"),
				STTType:    pipecatcall.STTTypeDeepgram,
				TTSType:    pipecatcall.TTSTypeElevenLabs,
				TTSVoiceID: "test-voice-id",
				LLMMessages: []map[string]any{
					{
						"role":    "system",
						"content": "You are a helpful assistant.",
					},
					{
						"role":    "user",
						"content": "Hello, world!",
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),

				HostID: "1.2.3.4",

				LLMType:    pipecatcall.LLMType("openai.gpt-4"),
				STTType:    pipecatcall.STTTypeDeepgram,
				TTSType:    pipecatcall.TTSTypeElevenLabs,
				TTSVoiceID: "test-voice-id",
				LLMMessages: []map[string]any{
					{
						"role":    "system",
						"content": "You are a helpful assistant.",
					},
					{
						"role":    "user",
						"content": "Hello, world!",
					},
				},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			name: "empty",

			pipecatcall: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
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
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			if err := h.PipecatcallCreate(ctx, tt.pipecatcall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().PipecatcallGet(ctx, tt.pipecatcall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			res, err := h.PipecatcallGet(ctx, tt.pipecatcall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_PipecatcallUpdate(t *testing.T) {

	tests := []struct {
		name string
		flow *pipecatcall.Pipecatcall

		id     uuid.UUID
		fields map[pipecatcall.Field]any

		responseCurTime string

		expectedRes *pipecatcall.Pipecatcall
	}{
		{
			name: "test normal",
			flow: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("90dcceac-b48e-11f0-83e3-d30fce348344"),
				},
			},

			id: uuid.FromStringOrNil("90dcceac-b48e-11f0-83e3-d30fce348344"),
			fields: map[pipecatcall.Field]any{
				pipecatcall.FieldActiveflowID:  uuid.FromStringOrNil("908f40d8-b48e-11f0-bd85-7ba4b165fc16"),
				pipecatcall.FieldReferenceType: pipecatcall.ReferenceTypeAICall,
				pipecatcall.FieldLLMMessages: []map[string]any{
					{
						"role":    "system",
						"content": "You are a helpful assistant.",
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("90dcceac-b48e-11f0-83e3-d30fce348344"),
				},
				ActiveflowID:  uuid.FromStringOrNil("908f40d8-b48e-11f0-bd85-7ba4b165fc16"),
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				LLMMessages: []map[string]any{
					{
						"role":    "system",
						"content": "You are a helpful assistant.",
					},
				},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: commondatabasehandler.DefaultTimeStamp,
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
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			if err := h.PipecatcallCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			if err := h.PipecatcallUpdate(context.Background(), tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().PipecatcallGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			res, err := h.PipecatcallGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_FlowDelete(t *testing.T) {

	tests := []struct {
		name string
		flow *pipecatcall.Pipecatcall

		responseCurTime string

		expectedRes *pipecatcall.Pipecatcall
	}{
		{
			name: "normal",
			flow: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4fa80722-b492-11f0-b3bd-7ff9a5ee49c7"),
					CustomerID: uuid.FromStringOrNil("4fd5c734-b492-11f0-8d98-7f3010848eb1"),
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4fa80722-b492-11f0-b3bd-7ff9a5ee49c7"),
					CustomerID: uuid.FromStringOrNil("4fd5c734-b492-11f0-8d98-7f3010848eb1"),
				},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: "2020-04-18 03:22:17.995000",
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
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			if err := h.PipecatcallCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any())
			if err := h.PipecatcallDelete(ctx, tt.flow.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().PipecatcallGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any()).Return(nil)
			res, err := h.PipecatcallGet(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

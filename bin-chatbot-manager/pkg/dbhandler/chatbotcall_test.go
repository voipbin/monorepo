package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/cachehandler"
)

func Test_ChatbotcallCreate(t *testing.T) {

	tests := []struct {
		name string

		chatbot *chatbotcall.Chatbotcall

		responseCurTime string

		expectRes *chatbotcall.Chatbotcall
	}{
		{
			name: "have all",
			chatbot: &chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
				CustomerID:    uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				ChatbotID:     uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				ActiveflowID:  uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:  uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				TranscribeID:  uuid.FromStringOrNil("e2c7cd7a-a5e1-11ed-9c3a-ef9305cb70cd"),
				Status:        chatbotcall.StatusInitiating,
				Gender:        chatbotcall.GenderFemale,
				Language:      "en-US",
				Messages:      []chatbotcall.Message{},
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("b11ef334-a5e1-11ed-8006-bf175306f060"),
				CustomerID:    uuid.FromStringOrNil("b147c35e-a5e1-11ed-bd07-e789c0df6bca"),
				ChatbotID:     uuid.FromStringOrNil("b171a2be-a5e1-11ed-a547-cf7c662e9b6b"),
				ActiveflowID:  uuid.FromStringOrNil("d23695e0-fba4-11ed-a802-4ba57348a125"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b198e572-a5e1-11ed-acc0-5fc5c1482647"),
				ConfbridgeID:  uuid.FromStringOrNil("24c07cfb-92b0-4334-b5e8-fea9b8c5fdbd"),
				TranscribeID:  uuid.FromStringOrNil("e2c7cd7a-a5e1-11ed-9c3a-ef9305cb70cd"),
				Status:        chatbotcall.StatusInitiating,
				Gender:        chatbotcall.GenderFemale,
				Language:      "en-US",
				Messages:      []chatbotcall.Message{},
				TMEnd:         DefaultTimeStamp,
				TMCreate:      "2023-01-03 21:35:02.809",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			"empty",
			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
			},

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:       uuid.FromStringOrNil("e2fa5772-a5e1-11ed-94a9-f72c152d4780"),
				Messages: []chatbotcall.Message{},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotcallGet(ctx, tt.chatbot.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGet(ctx, tt.chatbot.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallGetByReferenceID(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbotcall.Chatbotcall

		referenceID uuid.UUID

		responseCurTime string

		expectRes *chatbotcall.Chatbotcall
	}{
		{
			"normal",
			&chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("a8b26464-a5e2-11ed-bce7-83b475b0c53d"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),
			},

			uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("a8b26464-a5e2-11ed-bce7-83b475b0c53d"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a8ebd744-a5e2-11ed-bc18-d3a88a0f1ffa"),
				Messages:      []chatbotcall.Message{},
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotcallGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallGetByTranscribeID(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbotcall.Chatbotcall

		transcribeID uuid.UUID

		responseCurTime string

		expectRes *chatbotcall.Chatbotcall
	}{
		{
			"normal",
			&chatbotcall.Chatbotcall{
				ID:           uuid.FromStringOrNil("ee65f8bc-a5e3-11ed-bc48-4fd434eda48d"),
				TranscribeID: uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),
			},

			uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:           uuid.FromStringOrNil("ee65f8bc-a5e3-11ed-bc48-4fd434eda48d"),
				TranscribeID: uuid.FromStringOrNil("ee91df04-a5e3-11ed-91f2-a36948c67a14"),
				Messages:     []chatbotcall.Message{},
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotcallGetByTranscribeID(ctx, tt.transcribeID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGetByTranscribeID(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallUpdateStatusProgressing(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbotcall.Chatbotcall

		id           uuid.UUID
		transcribeID uuid.UUID

		responseCurTime string

		expectRes *chatbotcall.Chatbotcall
	}{
		{
			"normal",
			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
			},

			uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
			uuid.FromStringOrNil("e6342714-a5e2-11ed-a3dd-cbe7bf0cbcb0"),

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:           uuid.FromStringOrNil("e5f5d64e-a5e2-11ed-bd71-538bb8b1fd91"),
				TranscribeID: uuid.FromStringOrNil("e6342714-a5e2-11ed-a3dd-cbe7bf0cbcb0"),
				Status:       chatbotcall.StatusProgressing,
				Messages:     []chatbotcall.Message{},
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallUpdateStatusProgressing(ctx, tt.id, tt.transcribeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallUpdateStatusEnd(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbotcall.Chatbotcall

		id uuid.UUID

		responseCurTime string

		expectRes *chatbotcall.Chatbotcall
	}{
		{
			"normal",
			&chatbotcall.Chatbotcall{
				ID:           uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),
				TranscribeID: uuid.FromStringOrNil("e9a4d8c2-e7ca-11ef-b80a-43dbe39bcce9"),
			},

			uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:           uuid.FromStringOrNil("a210c140-a5e3-11ed-80e0-b726a1acfc64"),
				TranscribeID: uuid.Nil,
				Status:       chatbotcall.StatusEnd,
				Messages:     []chatbotcall.Message{},
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallUpdateStatusEnd(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallDelete(t *testing.T) {

	tests := []struct {
		name        string
		chatbotcall *chatbotcall.Chatbotcall

		id uuid.UUID

		responseCurTime string
		expectRes       *chatbotcall.Chatbotcall
	}{
		{
			"normal",
			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
			},

			uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),

			"2023-01-03 21:35:02.809",
			&chatbotcall.Chatbotcall{
				ID:       uuid.FromStringOrNil("78f9a8fc-a5e4-11ed-95aa-133c8380df73"),
				Messages: []chatbotcall.Message{},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: "2023-01-03 21:35:02.809",
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if err := h.ChatbotcallCreate(ctx, tt.chatbotcall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if errDel := h.ChatbotcallDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ChatbotcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatbotcallGets(t *testing.T) {

	tests := []struct {
		name         string
		chatbotcalls []*chatbotcall.Chatbotcall

		customerID uuid.UUID
		count      int
		filters    map[string]string

		responseCurTime string
		expectRes       []*chatbotcall.Chatbotcall
	}{
		{
			"normal",
			[]*chatbotcall.Chatbotcall{
				{
					ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
					CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
				},
				{
					ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
				},
			},

			uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			10,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*chatbotcall.Chatbotcall{
				{
					ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
					CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					Messages:   []chatbotcall.Message{},
					TMEnd:      DefaultTimeStamp,
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					Messages:   []chatbotcall.Message{},
					TMEnd:      DefaultTimeStamp,
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*chatbotcall.Chatbotcall{},

			uuid.FromStringOrNil("b31d32ae-7f45-11ec-82c6-936e22306376"),
			0,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*chatbotcall.Chatbotcall{},
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

			for _, cc := range tt.chatbotcalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
				if errCreate := h.ChatbotcallCreate(ctx, cc); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ChatbotcallGets(ctx, tt.customerID, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatbotcallSetMessages(t *testing.T) {

	tests := []struct {
		name        string
		chatbotcall *chatbotcall.Chatbotcall

		id       uuid.UUID
		messages []chatbotcall.Message

		responseCurTime string
		expectRes       *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",
			chatbotcall: &chatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("978f40ac-f665-11ed-92d8-7735094e2d1b"),
				CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			},

			id: uuid.FromStringOrNil("978f40ac-f665-11ed-92d8-7735094e2d1b"),
			messages: []chatbotcall.Message{
				{
					Role:    "system",
					Content: "test system message",
				},
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("978f40ac-f665-11ed-92d8-7735094e2d1b"),
				CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
				Messages: []chatbotcall.Message{
					{
						Role:    "system",
						Content: "test system message",
					},
				},

				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: DefaultTimeStamp,
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
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if errCreate := h.ChatbotcallCreate(ctx, tt.chatbotcall); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			if errSet := h.ChatbotcallSetMessages(ctx, tt.id, tt.messages); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().ChatbotcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotcallSet(ctx, gomock.Any())
			res, err := h.ChatbotcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

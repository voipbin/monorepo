package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

func Test_ConversationCreate(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation

		responseCurTime string
		expectRes       *conversation.Conversation
	}{
		{
			name: "have all",

			conversation: &conversation.Conversation{
				ID:            uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea112"),
				CustomerID:    uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				AccountID:     uuid.FromStringOrNil("5d634a2a-fdec-11ed-b49e-07e9ef4b45cf"),
				Name:          "conversation name",
				Detail:        "conversation detail",
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "9bf1d18c-f116-11ec-896c-636b8bfbe1a1",
				},
				Participants: []commonaddress.Address{
					{
						Type:       commonaddress.TypeLine,
						Target:     "e9d6a222-e42a-11ec-a678-57ec5f8add13",
						TargetName: "test user",
					},
				},
			},

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: &conversation.Conversation{
				ID:            uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea112"),
				CustomerID:    uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				AccountID:     uuid.FromStringOrNil("5d634a2a-fdec-11ed-b49e-07e9ef4b45cf"),
				Name:          "conversation name",
				Detail:        "conversation detail",
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "9bf1d18c-f116-11ec-896c-636b8bfbe1a1",
				},
				Participants: []commonaddress.Address{
					{
						Type:       commonaddress.TypeLine,
						Target:     "e9d6a222-e42a-11ec-a678-57ec5f8add13",
						TargetName: "test user",
					},
				},
				TMCreate: "2022-04-18 03:22:17.995000",
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
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
			if err := h.ConversationCreate(ctx, tt.conversation); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConversationGet(ctx, tt.conversation.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any()).Return(nil)
			res, err := h.ConversationGet(ctx, tt.conversation.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationGetByReferenceInfo(t *testing.T) {

	tests := []struct {
		name         string
		conversation *conversation.Conversation

		customerID    uuid.UUID
		referenceType conversation.ReferenceType
		referenceID   string

		responseCurTime string
		expectRes       *conversation.Conversation
	}{
		{
			name: "normal",
			conversation: &conversation.Conversation{
				ID:            uuid.FromStringOrNil("400d2aaa-e429-11ec-92ee-9779b9418690"),
				CustomerID:    uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				Name:          "conversation name",
				Detail:        "conversation detail",
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "612435d0-e429-11ec-845d-bba00000504b",
				Source:        &commonaddress.Address{},
				Participants:  []commonaddress.Address{},
			},

			customerID:    uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
			referenceType: conversation.ReferenceTypeLine,
			referenceID:   "612435d0-e429-11ec-845d-bba00000504b",

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: &conversation.Conversation{
				ID:            uuid.FromStringOrNil("400d2aaa-e429-11ec-92ee-9779b9418690"),
				CustomerID:    uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				Name:          "conversation name",
				Detail:        "conversation detail",
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "612435d0-e429-11ec-845d-bba00000504b",
				Source:        &commonaddress.Address{},
				Participants:  []commonaddress.Address{},
				TMCreate:      "2022-04-18 03:22:17.995000",
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
			mockCache.EXPECT().ConversationSet(gomock.Any(), gomock.Any())
			if err := h.ConversationCreate(ctx, tt.conversation); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConversationGetByReferenceInfo(ctx, tt.customerID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name          string
		conversations []*conversation.Conversation

		customerID uuid.UUID
		token      string
		limit      uint64

		responseCurTime string
		expectRes       []*conversation.Conversation
	}{
		{
			name: "normal",
			conversations: []*conversation.Conversation{
				{
					ID:            uuid.FromStringOrNil("3358a1b2-e42a-11ec-9052-23951983d6b2"),
					CustomerID:    uuid.FromStringOrNil("2e7d337e-e42a-11ec-b705-07b2b80e4ad5"),
					Name:          "conversation name",
					Detail:        "conversation detail",
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "38a2bdf6-e42a-11ec-b5a9-43316ee06787",
					Source:        &commonaddress.Address{},
					Participants:  []commonaddress.Address{},
				},
				{
					ID:            uuid.FromStringOrNil("2a99bbd8-e42a-11ec-ae36-576f6e89b025"),
					CustomerID:    uuid.FromStringOrNil("2e7d337e-e42a-11ec-b705-07b2b80e4ad5"),
					Name:          "conversation name",
					Detail:        "conversation detail",
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "387f1afe-e42a-11ec-ad8f-1340414f9a51",
					Source:        &commonaddress.Address{},
					Participants:  []commonaddress.Address{},
				},
			},

			customerID: uuid.FromStringOrNil("2e7d337e-e42a-11ec-b705-07b2b80e4ad5"),
			token:      "2022-06-18 03:22:17.995000",
			limit:      100,

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: []*conversation.Conversation{
				{
					ID:            uuid.FromStringOrNil("3358a1b2-e42a-11ec-9052-23951983d6b2"),
					CustomerID:    uuid.FromStringOrNil("2e7d337e-e42a-11ec-b705-07b2b80e4ad5"),
					Name:          "conversation name",
					Detail:        "conversation detail",
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "38a2bdf6-e42a-11ec-b5a9-43316ee06787",
					Source:        &commonaddress.Address{},
					Participants:  []commonaddress.Address{},
					TMCreate:      "2022-04-18 03:22:17.995000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("2a99bbd8-e42a-11ec-ae36-576f6e89b025"),
					CustomerID:    uuid.FromStringOrNil("2e7d337e-e42a-11ec-b705-07b2b80e4ad5"),
					Name:          "conversation name",
					Detail:        "conversation detail",
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "387f1afe-e42a-11ec-ad8f-1340414f9a51",
					Source:        &commonaddress.Address{},
					Participants:  []commonaddress.Address{},
					TMCreate:      "2022-04-18 03:22:17.995000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
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

			for _, c := range tt.conversations {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConversationSet(gomock.Any(), gomock.Any())
				if err := h.ConversationCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ConversationGetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationSet(t *testing.T) {
	tests := []struct {
		name         string
		conversation *conversation.Conversation

		id               uuid.UUID
		conversationName string
		detail           string

		responseCurTime string
		expectRes       *conversation.Conversation
	}{
		{
			name: "normal",
			conversation: &conversation.Conversation{
				ID:         uuid.FromStringOrNil("fbb24a9a-0068-11ee-985d-fffb84d2b682"),
				CustomerID: uuid.FromStringOrNil("fbdb45f8-0068-11ee-9984-63f5b1d1e1c4"),
			},

			id:               uuid.FromStringOrNil("fbb24a9a-0068-11ee-985d-fffb84d2b682"),
			conversationName: "test name",
			detail:           "test detail",

			responseCurTime: "2020-04-18T03:22:17.995000",
			expectRes: &conversation.Conversation{
				ID:         uuid.FromStringOrNil("fbb24a9a-0068-11ee-985d-fffb84d2b682"),
				CustomerID: uuid.FromStringOrNil("fbdb45f8-0068-11ee-9984-63f5b1d1e1c4"),
				Name:       "test name",
				Detail:     "test detail",
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMUpdate:   "2020-04-18T03:22:17.995000",
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
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
			if err := h.ConversationCreate(ctx, tt.conversation); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConversationSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.ConversationSet(ctx, tt.id, tt.conversationName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConversationGet(ctx, tt.conversation.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
			res, err := h.ConversationGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

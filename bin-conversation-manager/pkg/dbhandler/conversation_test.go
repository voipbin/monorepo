package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea112"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("9a0591de-3d35-11ef-9856-8ffd2949633a"),
				},

				AccountID: uuid.FromStringOrNil("5d634a2a-fdec-11ed-b49e-07e9ef4b45cf"),
				Name:      "conversation name",
				Detail:    "conversation detail",
				Type:      conversation.TypeLine,
				DialogID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Self: commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "9bf1d18c-f116-11ec-896c-636b8bfbe1a1",
				},
				Peer: commonaddress.Address{
					Type:       commonaddress.TypeLine,
					Target:     "e9d6a222-e42a-11ec-a678-57ec5f8add13",
					TargetName: "test user",
				},
			},

			responseCurTime: "2022-04-18T03:22:17.995000Z",
			expectRes: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea112"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("9a0591de-3d35-11ef-9856-8ffd2949633a"),
				},
				AccountID: uuid.FromStringOrNil("5d634a2a-fdec-11ed-b49e-07e9ef4b45cf"),
				Name:      "conversation name",
				Detail:    "conversation detail",
				Type:      conversation.TypeLine,
				DialogID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Self: commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "9bf1d18c-f116-11ec-896c-636b8bfbe1a1",
				},
				Peer: commonaddress.Address{
					Type:       commonaddress.TypeLine,
					Target:     "e9d6a222-e42a-11ec-a678-57ec5f8add13",
					TargetName: "test user",
				},
				TMCreate: "2022-04-18T03:22:17.995000Z",
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

func Test_ConversationList(t *testing.T) {

	tests := []struct {
		name          string
		conversations []*conversation.Conversation

		token   string
		limit   uint64
		filters map[conversation.Field]any

		responseCurTime string
		expectRes       []*conversation.Conversation
	}{
		{
			name: "normal",
			conversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a4b1d416-3e12-11ef-9007-3bcec17f6287"),
						CustomerID: uuid.FromStringOrNil("a55f730a-3e12-11ef-adec-df6b60fe6b19"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("a5932312-3e12-11ef-ba41-3720b253edff"),
					},
					Name:     "conversation name",
					Detail:   "conversation detail",
					Type:     conversation.TypeLine,
					DialogID: "38a2bdf6-e42a-11ec-b5a9-43316ee06787",
					Self:     commonaddress.Address{},
					Peer:     commonaddress.Address{},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a52972d2-3e12-11ef-ab7c-9303bf77ff4d"),
						CustomerID: uuid.FromStringOrNil("a55f730a-3e12-11ef-adec-df6b60fe6b19"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("a5932312-3e12-11ef-ba41-3720b253edff"),
					},
					Name:     "conversation name",
					Detail:   "conversation detail",
					Type:     conversation.TypeLine,
					DialogID: "387f1afe-e42a-11ec-ad8f-1340414f9a51",
					Self:     commonaddress.Address{},
					Peer:     commonaddress.Address{},
				},
			},

			token: "2022-06-18T03:22:17.995000Z",
			limit: 100,
			filters: map[conversation.Field]any{
				conversation.FieldDeleted:    false,
				conversation.FieldCustomerID: uuid.FromStringOrNil("a55f730a-3e12-11ef-adec-df6b60fe6b19"),
				conversation.FieldType:       conversation.TypeLine,
			},

			responseCurTime: "2022-04-18T03:22:17.995000Z",
			expectRes: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a4b1d416-3e12-11ef-9007-3bcec17f6287"),
						CustomerID: uuid.FromStringOrNil("a55f730a-3e12-11ef-adec-df6b60fe6b19"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("a5932312-3e12-11ef-ba41-3720b253edff"),
					},

					Name:     "conversation name",
					Detail:   "conversation detail",
					Type:     conversation.TypeLine,
					DialogID: "38a2bdf6-e42a-11ec-b5a9-43316ee06787",
					Self:     commonaddress.Address{},
					Peer:     commonaddress.Address{},
					TMCreate: "2022-04-18T03:22:17.995000Z",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a52972d2-3e12-11ef-ab7c-9303bf77ff4d"),
						CustomerID: uuid.FromStringOrNil("a55f730a-3e12-11ef-adec-df6b60fe6b19"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("a5932312-3e12-11ef-ba41-3720b253edff"),
					},

					Name:     "conversation name",
					Detail:   "conversation detail",
					Type:     conversation.TypeLine,
					DialogID: "387f1afe-e42a-11ec-ad8f-1340414f9a51",
					Self:     commonaddress.Address{},
					Peer:     commonaddress.Address{},
					TMCreate: "2022-04-18T03:22:17.995000Z",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
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

			res, err := h.ConversationList(ctx, tt.limit, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_ConversationUpdate(t *testing.T) {
	tests := []struct {
		name         string
		conversation *conversation.Conversation

		id    uuid.UUID
		field map[conversation.Field]any

		responseCurTime string
		expectRes       *conversation.Conversation
	}{
		{
			name: "normal",
			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00f151ba-2199-11f0-85be-9b26b400d0c2"),
					CustomerID: uuid.FromStringOrNil("010eac88-2199-11f0-ad61-671cf62bcc31"),
				},
			},

			id: uuid.FromStringOrNil("00f151ba-2199-11f0-85be-9b26b400d0c2"),
			field: map[conversation.Field]any{
				conversation.FieldOwnerType: "agent",
				conversation.FieldOwnerID:   uuid.FromStringOrNil("f74ef31a-2198-11f0-8a23-0b555d83cce8"),
				conversation.FieldAccountID: uuid.FromStringOrNil("012c01ac-2199-11f0-a5e5-7f1895af8640"),
				conversation.FieldName:      "update name",
				conversation.FieldDetail:    "update detail",
				conversation.FieldSelf: commonaddress.Address{
					Target: "+123456789",
				},
			},

			responseCurTime: "2020-04-18T03:22:17.995000Z",
			expectRes: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00f151ba-2199-11f0-85be-9b26b400d0c2"),
					CustomerID: uuid.FromStringOrNil("010eac88-2199-11f0-ad61-671cf62bcc31"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("f74ef31a-2198-11f0-8a23-0b555d83cce8"),
				},
				AccountID: uuid.FromStringOrNil("012c01ac-2199-11f0-a5e5-7f1895af8640"),
				Name:      "update name",
				Detail:    "update detail",
				Self: commonaddress.Address{
					Target: "+123456789",
				},

				TMCreate: "2020-04-18T03:22:17.995000Z",
				TMUpdate: "2020-04-18T03:22:17.995000Z",
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
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
			if err := h.ConversationCreate(ctx, tt.conversation); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConversationSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.ConversationUpdate(ctx, tt.id, tt.field); err != nil {
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

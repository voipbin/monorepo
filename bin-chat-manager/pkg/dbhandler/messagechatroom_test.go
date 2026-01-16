package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/cachehandler"
)

func Test_MessagechatroomCreate(t *testing.T) {

	tests := []struct {
		name string

		msg *messagechatroom.Messagechatroom
	}{
		{
			"empty all",

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79d814a8-20b9-11ed-b450-d7f8fee3b91c"),
				},
			},
		},
		{
			"all items",

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7a0c38f0-20b9-11ed-8e47-b7f77cbe161e"),
					CustomerID: uuid.FromStringOrNil("7a369a6e-20b9-11ed-adca-5713ccb2dc5e"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("77f158b0-da31-11ee-8884-abbae9b72b3c"),
				},

				ChatroomID: uuid.FromStringOrNil("7a9865d2-20b9-11ed-8243-3faf34c97731"),

				Source: &commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "test target",
					Name:       "test name",
					Detail:     "test detail",
				},
				Type:   messagechatroom.TypeNormal,
				Text:   "test message",
				Medias: []media.Media{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
			if err := h.MessagechatroomCreate(ctx, tt.msg); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessagechatroomGet(ctx, tt.msg.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
			res, err := h.MessagechatroomGet(ctx, tt.msg.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.msg, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.msg, res)
			}
		})
	}
}

func Test_MessagechatroomList(t *testing.T) {

	tests := []struct {
		name string
		data []*messagechatroom.Messagechatroom

		customerID uuid.UUID
		size       uint64
		filters    map[messagechatroom.Field]any

		expectRes []*messagechatroom.Messagechatroom
	}{
		{
			"normal",
			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7afc1816-20b9-11ed-9ff2-838e10ae7629"),
						CustomerID: uuid.FromStringOrNil("7acab154-20b9-11ed-9a1e-0738cbdd7876"),
					},
					ChatroomID:    uuid.FromStringOrNil("c84b4484-bad2-11ee-be7f-1bb4bcbab992"),
					MessagechatID: uuid.FromStringOrNil("c87ad8d4-bad2-11ee-964d-2f7ccf4aa5aa"),

					TMCreate: "2020-04-19T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7b2f6bd0-20b9-11ed-9c4d-e3fcfa19401a"),
						CustomerID: uuid.FromStringOrNil("7acab154-20b9-11ed-9a1e-0738cbdd7876"),
					},
					ChatroomID:    uuid.FromStringOrNil("c84b4484-bad2-11ee-be7f-1bb4bcbab992"),
					MessagechatID: uuid.FromStringOrNil("c8a3c082-bad2-11ee-bb40-878bc8a951a1"),

					TMCreate: "2020-04-18T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("7acab154-20b9-11ed-9a1e-0738cbdd7876"),
			10,
			map[messagechatroom.Field]any{
				messagechatroom.FieldDeleted:       false,
				messagechatroom.FieldChatroomID:    uuid.FromStringOrNil("c84b4484-bad2-11ee-be7f-1bb4bcbab992"),
				messagechatroom.FieldMessagechatID: uuid.FromStringOrNil("c87ad8d4-bad2-11ee-964d-2f7ccf4aa5aa"),
			},

			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7afc1816-20b9-11ed-9ff2-838e10ae7629"),
						CustomerID: uuid.FromStringOrNil("7acab154-20b9-11ed-9a1e-0738cbdd7876"),
					},
					ChatroomID:    uuid.FromStringOrNil("c84b4484-bad2-11ee-be7f-1bb4bcbab992"),
					MessagechatID: uuid.FromStringOrNil("c87ad8d4-bad2-11ee-964d-2f7ccf4aa5aa"),

					TMCreate: "2020-04-19T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
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

			for _, c := range tt.data {
				mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
				mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
				if err := h.MessagechatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.MessagechatroomList(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Clear timestamps for comparison (they are auto-generated by Create)
			for _, c := range cs {
				c.TMCreate = ""
				c.TMUpdate = ""
				c.TMDelete = ""
			}
			for _, c := range tt.expectRes {
				c.TMCreate = ""
				c.TMUpdate = ""
				c.TMDelete = ""
			}

			if reflect.DeepEqual(cs, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cs)
			}
		})
	}
}

func Test_MessagechatroomDelete(t *testing.T) {

	tests := []struct {
		name string

		msg *messagechatroom.Messagechatroom
	}{
		{
			"normal",

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0abd8bb0-20ba-11ed-80da-5ff9e046c29f"),
					CustomerID: uuid.FromStringOrNil("0aee266c-20ba-11ed-9784-cb1e7c4182d4"),
				},
				TMCreate: "2020-04-19T03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
			if err := h.MessagechatroomCreate(ctx, tt.msg); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
			if errDel := h.MessagechatroomDelete(ctx, tt.msg.ID); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().MessagechatroomGet(ctx, tt.msg.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessagechatroomSet(ctx, gomock.Any())
			res, err := h.MessagechatroomGet(ctx, tt.msg.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect")
			}
		})
	}
}

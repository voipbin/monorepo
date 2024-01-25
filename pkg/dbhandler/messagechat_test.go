package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/cachehandler"
)

func Test_MessagechatCreate(t *testing.T) {

	tests := []struct {
		name string

		msg *messagechat.Messagechat
	}{
		{
			"empty all",

			&messagechat.Messagechat{
				ID: uuid.FromStringOrNil("d98a9ab4-20a6-11ed-a34f-631a552935e3"),
			},
		},
		{
			"all items",

			&messagechat.Messagechat{
				ID:         uuid.FromStringOrNil("d9ec3b66-20a6-11ed-9bbf-cbca7996ec6f"),
				CustomerID: uuid.FromStringOrNil("da1f519a-20a6-11ed-8883-ebcee5b506e9"),

				ChatID: uuid.FromStringOrNil("da4e1f7a-20a6-11ed-8bf2-ef7f2470a441"),

				Source: &commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "test target",
					Name:       "test name",
					Detail:     "test detail",
				},
				Type:   messagechat.TypeSystem,
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

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
			if err := h.MessagechatCreate(ctx, tt.msg); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessagechatGet(ctx, tt.msg.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
			res, err := h.MessagechatGet(ctx, tt.msg.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.msg, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.msg, res)
			}
		})
	}
}

func Test_MessagechatGets(t *testing.T) {

	tests := []struct {
		name string

		chatID  uuid.UUID
		limit   uint64
		filters map[string]string

		messagechats []*messagechat.Messagechat
	}{
		{
			"normal",
			uuid.FromStringOrNil("127a78d4-20b7-11ed-9060-8bc587ee87d7"),
			10,
			map[string]string{
				"customer_id": "c12a4082-20b5-11ed-8c34-07c2afd5a6ab",
				"deleted":     "false",
				"chat_id":     "127a78d4-20b7-11ed-9060-8bc587ee87d7",
			},

			[]*messagechat.Messagechat{
				{
					ID:         uuid.FromStringOrNil("1224803c-20b7-11ed-9d6c-f3d58383f709"),
					CustomerID: uuid.FromStringOrNil("c12a4082-20b5-11ed-8c34-07c2afd5a6ab"),
					ChatID:     uuid.FromStringOrNil("127a78d4-20b7-11ed-9060-8bc587ee87d7"),
					TMCreate:   "2020-04-19T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("1255f1b2-20b7-11ed-a560-e74d6ef11737"),
					CustomerID: uuid.FromStringOrNil("c12a4082-20b5-11ed-8c34-07c2afd5a6ab"),
					ChatID:     uuid.FromStringOrNil("127a78d4-20b7-11ed-9060-8bc587ee87d7"),
					TMCreate:   "2020-04-18T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.messagechats {
				mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
				if err := h.MessagechatCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.MessagechatGets(ctx, utilhandler.TimeGetCurTime(), tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(cs, tt.messagechats) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.messagechats, cs)
			}
		})
	}
}


func Test_MessagechatDelete(t *testing.T) {

	tests := []struct {
		name string

		msg *messagechat.Messagechat
	}{
		{
			"normal",

			&messagechat.Messagechat{

				ID:         uuid.FromStringOrNil("3275aa6a-20b6-11ed-941f-ffa864a78e87"),
				CustomerID: uuid.FromStringOrNil("c12a4082-20b5-11ed-8c34-07c2afd5a6ab"),
				TMCreate:   "2020-04-19T03:22:17.995000",
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

			mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
			if err := h.MessagechatCreate(ctx, tt.msg); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
			if errDel := h.MessagechatDelete(ctx, tt.msg.ID); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().MessagechatGet(ctx, tt.msg.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessagechatSet(ctx, gomock.Any())
			res, err := h.MessagechatGet(ctx, tt.msg.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect")
			}
		})
	}
}

package messagechatroomhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessagechatroom *messagechatroom.Messagechatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("f9a22990-32b0-11ed-911a-8baec663a128"),

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f9a22990-32b0-11ed-911a-8baec663a128"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatroomGet(ctx, tt.id).Return(tt.responseMessagechatroom, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechatroom, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		chatroomID uuid.UUID
		token      string
		limit      uint64
		filters    map[string]string

		responseMessagechatroom []*messagechatroom.Messagechatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("df481ecc-32b2-11ed-9274-6b15aa52d410"),
			"2022-04-18 03:22:17.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			[]*messagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("df481ecc-32b2-11ed-9274-6b15aa52d410"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatroomGets(ctx, tt.token, tt.limit, tt.filters).Return(tt.responseMessagechatroom, nil)

			res, err := h.Gets(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechatroom, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		agentID       uuid.UUID
		chatroomID    uuid.UUID
		messagechatID uuid.UUID
		source        *commonaddress.Address
		messageType   messagechatroom.Type
		text          string
		medias        []media.Media

		responseUUID            uuid.UUID
		responseCurTime         string
		responseMessagechatroom *messagechatroom.Messagechatroom

		expectMessagechatroom *messagechatroom.Messagechatroom
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("65ac45e2-32b3-11ed-b720-973a629c7807"),
			agentID:       uuid.FromStringOrNil("0698e5d4-daae-11ee-be05-cb7440513a2f"),
			chatroomID:    uuid.FromStringOrNil("65d8b7e4-32b3-11ed-8846-97d903739f2c"),
			messagechatID: uuid.FromStringOrNil("662ecbfc-32b3-11ed-ae98-a71aa0c6ca99"),
			source: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "test target",
				Name:       "test name",
				Detail:     "test detail",
			},
			messageType: messagechatroom.TypeNormal,
			text:        "test text",
			medias:      []media.Media{},

			responseUUID:    uuid.FromStringOrNil("68922a7a-daae-11ee-8e4d-6fbcd4b11b39"),
			responseCurTime: "2024-03-05 05:10:04.781006734",
			responseMessagechatroom: &messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6692d52a-32b3-11ed-9cf8-ef08221492ce"),
				},
			},

			expectMessagechatroom: &messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("68922a7a-daae-11ee-8e4d-6fbcd4b11b39"),
					CustomerID: uuid.FromStringOrNil("65ac45e2-32b3-11ed-b720-973a629c7807"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("0698e5d4-daae-11ee-be05-cb7440513a2f"),
				},
				ChatroomID:    uuid.FromStringOrNil("65d8b7e4-32b3-11ed-8846-97d903739f2c"),
				MessagechatID: uuid.FromStringOrNil("662ecbfc-32b3-11ed-ae98-a71aa0c6ca99"),
				Source: &commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "test target",
					Name:       "test name",
					Detail:     "test detail",
				},
				Type:     messagechatroom.TypeNormal,
				Text:     "test text",
				Medias:   []media.Media{},
				TMCreate: "2024-03-05 05:10:04.781006734",
				TMUpdate: dbhandler.DefaultTimeStamp,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatroomHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockDB.EXPECT().MessagechatroomCreate(ctx, tt.expectMessagechatroom).Return(nil)
			mockDB.EXPECT().MessagechatroomGet(ctx, gomock.Any()).Return(tt.responseMessagechatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechatroom.CustomerID, messagechatroom.EventTypeMessagechatroomCreated, tt.responseMessagechatroom)

			res, err := h.Create(ctx, tt.customerID, tt.agentID, tt.chatroomID, tt.messagechatID, tt.source, tt.messageType, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechatroom, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessagechatroom *messagechatroom.Messagechatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("f1b4ce88-32b3-11ed-910f-c300da3e58d5"),

			&messagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b4ce88-32b3-11ed-910f-c300da3e58d5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messagechatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessagechatroomDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessagechatroomGet(ctx, tt.responseMessagechatroom.ID).Return(tt.responseMessagechatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessagechatroom.CustomerID, messagechatroom.EventTypeMessagechatroomDeleted, tt.responseMessagechatroom)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseMessagechatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseMessagechatroom, res)
			}
		})
	}
}

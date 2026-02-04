package groupcallhandler

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
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id                uuid.UUID
		customerID        uuid.UUID
		ownerType         commonidentity.OwnerType
		ownerID           uuid.UUID
		flowID            uuid.UUID
		source            *commonaddress.Address
		destinations      []commonaddress.Address
		callIDs           []uuid.UUID
		groupcallIDs      []uuid.UUID
		masterCallID      uuid.UUID
		masterGroupcallID uuid.UUID
		ringMethod        groupcall.RingMethod
		answerMethod      groupcall.AnswerMethod

		expectGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("708d695e-e457-11ed-a7eb-dfe8cc1bbd99"),
			customerID: uuid.FromStringOrNil("c345ddd8-bb27-11ed-812c-df4f74c7c1a1"),
			ownerType:  commonidentity.OwnerTypeAgent,
			ownerID:    uuid.FromStringOrNil("88177492-2c00-11ef-b655-af61ed389cee"),
			flowID:     uuid.FromStringOrNil("9aa1067e-e4bc-4ec0-8251-75b266330514"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			callIDs: []uuid.UUID{
				uuid.FromStringOrNil("c38fe31a-bb27-11ed-9e5c-bf52e856e97c"),
				uuid.FromStringOrNil("c3c0630a-bb27-11ed-a026-236fa4f96287"),
			},
			groupcallIDs: []uuid.UUID{
				uuid.FromStringOrNil("7e9cf26e-e469-11ed-89d5-83e996b75aca"),
				uuid.FromStringOrNil("7ee564c2-e469-11ed-8d10-1b6a574c6bda"),
			},
			masterCallID:      uuid.FromStringOrNil("427ff710-e11a-11ed-b73f-fb15f8d543a3"),
			masterGroupcallID: uuid.FromStringOrNil("43b938f6-e455-11ed-84f1-a347bf43faa8"),
			ringMethod:        groupcall.RingMethodRingAll,
			answerMethod:      groupcall.AnswerMethodHangupOthers,

			expectGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("708d695e-e457-11ed-a7eb-dfe8cc1bbd99"),
					CustomerID: uuid.FromStringOrNil("c345ddd8-bb27-11ed-812c-df4f74c7c1a1"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("88177492-2c00-11ef-b655-af61ed389cee"),
				},
				Status: groupcall.StatusProgressing,
				FlowID: uuid.FromStringOrNil("9aa1067e-e4bc-4ec0-8251-75b266330514"),
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				MasterCallID:      uuid.FromStringOrNil("427ff710-e11a-11ed-b73f-fb15f8d543a3"),
				MasterGroupcallID: uuid.FromStringOrNil("43b938f6-e455-11ed-84f1-a347bf43faa8"),
				RingMethod:        groupcall.RingMethodRingAll,
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerCallID:      [16]byte{},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("c38fe31a-bb27-11ed-9e5c-bf52e856e97c"),
					uuid.FromStringOrNil("c3c0630a-bb27-11ed-a026-236fa4f96287"),
				},
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("7e9cf26e-e469-11ed-89d5-83e996b75aca"),
					uuid.FromStringOrNil("7ee564c2-e469-11ed-8d10-1b6a574c6bda"),
				},

				CallCount:      2,
				GroupcallCount: 2,
				DialIndex:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallCreate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.expectGroupcall.ID).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectGroupcall.CustomerID, groupcall.EventTypeGroupcallCreated, tt.expectGroupcall)

			res, err := h.Create(ctx, tt.id, tt.customerID, tt.ownerType, tt.ownerID, tt.flowID, tt.source, tt.destinations, tt.callIDs, tt.groupcallIDs, tt.masterCallID, tt.masterGroupcallID, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupcall, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[groupcall.Field]any

		responseGroupcalls []*groupcall.Groupcall
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-01-18T03:22:18.995000Z",
			filters: map[groupcall.Field]any{
				groupcall.FieldCustomerID: uuid.FromStringOrNil("b3944c9c-bd7c-11ed-874c-6b6fb342a46d"),
			},

			responseGroupcalls: []*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3cc0d12-bd7c-11ed-9b14-0bbea6e65d74"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3f8f2c8-bd7c-11ed-b479-dbff552bfa7e"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallList(ctx, tt.size, tt.token, gomock.Any()).Return(tt.responseGroupcalls, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcalls, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("678717bc-bb29-11ed-81c0-c3d2e4da7296"),
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("678717bc-bb29-11ed-81c0-c3d2e4da7296"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateAnswerCallID(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		callID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("ac6ae0fc-bb29-11ed-9f2a-6b95feacf142"),
			callID: uuid.FromStringOrNil("ac9c3c42-bb29-11ed-aa47-47441a47de62"),

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ac6ae0fc-bb29-11ed-9f2a-6b95feacf142"),
				},
				AnswerCallID: uuid.FromStringOrNil("ac9c3c42-bb29-11ed-aa47-47441a47de62"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetAnswerCallID(ctx, tt.id, tt.callID).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.responseGroupcall)

			res, err := h.UpdateAnswerCallID(ctx, tt.id, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateAnswerGroupcallID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		groupcallID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("6f148790-2cd4-44ed-962b-fdb7a5f4a28c"),
			groupcallID: uuid.FromStringOrNil("ad78b600-f184-4ed9-8296-fbe56f1cc0d2"),

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6f148790-2cd4-44ed-962b-fdb7a5f4a28c"),
				},
				AnswerGroupcallID: uuid.FromStringOrNil("ad78b600-f184-4ed9-8296-fbe56f1cc0d2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetAnswerGroupcallID(ctx, tt.id, tt.groupcallID).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.responseGroupcall)

			res, err := h.UpdateAnswerGroupcallID(ctx, tt.id, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("6d83f5fe-bd7c-11ed-83eb-af7a569ac7ad"),
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6d83f5fe-bd7c-11ed-83eb-af7a569ac7ad"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallDeleted, tt.responseGroupcall)

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_DecreaseCallCount(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("41d22862-d899-11ed-9791-2355b45b9efc"),
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("41d22862-d899-11ed-9791-2355b45b9efc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDecreaseCallCount(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.DecreaseCallCount(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_DecreaseGroupcallCount(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("6d5bdb48-e2c3-11ed-8bdc-67d0aa1f514d"),
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6d5bdb48-e2c3-11ed-8bdc-67d0aa1f514d"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDecreaseGroupcallCount(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.DecreaseGroupcallCount(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateCallIDsAndCallCountAndDialIndex(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		callIDs   []uuid.UUID
		callCount int
		dialIndex int

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("0f286b57-2114-486b-a6f9-bd0f3641b506"),
			callIDs: []uuid.UUID{
				uuid.FromStringOrNil("c3b4e769-2844-4353-b005-dff5c393528f"),
				uuid.FromStringOrNil("f009f9ba-272b-4697-8b5a-a0ee904655a8"),
			},
			callCount: 2,
			dialIndex: 2,

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0f286b57-2114-486b-a6f9-bd0f3641b506"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, tt.id, tt.callIDs, tt.callCount, tt.dialIndex).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.UpdateCallIDsAndCallCountAndDialIndex(ctx, tt.id, tt.callIDs, tt.callCount, tt.dialIndex)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateGroupcallIDsAndGroupcallCountAndDialIndex(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		groupcallIDs   []uuid.UUID
		groupcallCount int
		dialIndex      int

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("6aecfce2-d32c-4716-9fa4-bbe87ff5e088"),
			groupcallIDs: []uuid.UUID{
				uuid.FromStringOrNil("ea5cfb70-7044-4ac5-8392-e8df751d5522"),
				uuid.FromStringOrNil("c733796b-2c63-40ec-90c5-64908e1ab247"),
			},
			groupcallCount: 2,
			dialIndex:      2,

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6aecfce2-d32c-4716-9fa4-bbe87ff5e088"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, tt.id, tt.groupcallIDs, tt.groupcallCount, tt.dialIndex).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.UpdateGroupcallIDsAndGroupcallCountAndDialIndex(ctx, tt.id, tt.groupcallIDs, tt.groupcallCount, tt.dialIndex)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status groupcall.Status

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("bc5c0a83-6785-4fc1-9d62-940e53abcb70"),
			status: groupcall.StatusHangup,

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bc5c0a83-6785-4fc1-9d62-940e53abcb70"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

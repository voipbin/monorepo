package messagehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/messagehandlermessagebird"
)

func Test_dbCreate(t *testing.T) {

	tests := []struct {
		name        string
		message     *message.Message
		responseGet *message.Message
		expectRes   *message.Message
	}{
		{
			"normal",
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1703a4fe-a225-11ec-b393-f7ff27e9f57d"),
				},
			},
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1703a4fe-a225-11ec-b393-f7ff27e9f57d"),
				},
			},
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1703a4fe-a225-11ec-b393-f7ff27e9f57d"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,

				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageCreate(ctx, tt.message).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.message.ID).Return(tt.responseGet, nil)

			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageCreated, tt.responseGet)

			res, err := h.dbCreate(ctx, tt.message)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateTargets(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		providerName message.ProviderName
		targets      []target.Target

		responseGet *message.Message
		expectRes   *message.Message
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("ca37640c-a225-11ec-8cbf-fbf3ceb420d5"),
			providerName: message.ProviderNameTelnyx,
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					Status: target.StatusSent,
				},
			},

			responseGet: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1703a4fe-a225-11ec-b393-f7ff27e9f57d"),
				},
			},
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1703a4fe-a225-11ec-b393-f7ff27e9f57d"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:                        mockDB,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.providerName, tt.targets).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageUpdated, tt.responseGet)

			res, err := h.dbUpdateTargets(ctx, tt.id, tt.providerName, tt.targets)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGet *message.Message
		expectRes   *message.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4"),

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4"),
				},
			},
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:                        mockDB,
				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)

			res, err := h.dbGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[message.Field]any

		responseGets []*message.Message
		expectRes    []*message.Message
	}{
		{
			"normal",

			"2021-02-26T18:26:49.000Z",
			10,
			map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("90c4eadc-a298-11ec-ab3a-8b21b05640ec"),
			},

			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4"),
					},
				},
			},
			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ab6eef5a-a297-11ec-87f8-dbbd4c5d2ff4"),
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
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:                        mockDB,
				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageList(ctx, tt.token, tt.size, tt.filters).Return(tt.responseGets, nil)

			res, err := h.dbList(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGet *message.Message
		expectRes   *message.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("1568bbd0-a2c9-11ec-b6a2-a7205ff6e321"),

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1568bbd0-a2c9-11ec-b6a2-a7205ff6e321"),
				},
			},
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1568bbd0-a2c9-11ec-b6a2-a7205ff6e321"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:                        mockDB,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageDeleted, tt.responseGet)

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

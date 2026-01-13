package messagehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/messagehandlermessagebird"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		source       *commonaddress.Address
		targets      []target.Target
		providerName message.ProviderName
		text         string
		direction    message.Direction

		responseUUID    uuid.UUID
		responseMessage *message.Message
		expectMessage   *message.Message
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("6dd9d746-197d-11ee-a39d-0ffbf2a45563"),
			customerID: uuid.FromStringOrNil("755c7b90-197d-11ee-9cb0-a3ddbcba0c6f"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					Status: target.StatusReceived,
				},
			},
			providerName: message.ProviderNameTelnyx,
			text:         "test message",
			direction:    message.DirectionInbound,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6dd9d746-197d-11ee-a39d-0ffbf2a45563"),
				},
			},
			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6dd9d746-197d-11ee-a39d-0ffbf2a45563"),
					CustomerID: uuid.FromStringOrNil("755c7b90-197d-11ee-9cb0-a3ddbcba0c6f"),
				},
				Type: message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusReceived,
					},
				},
				ProviderName: message.ProviderNameTelnyx,
				Text:         "test message",
				Medias:       []string{},
				Direction:    message.DirectionInbound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				messageHandlerMessagebird: mockMessagebird,
			}
			ctx := context.Background()

			if tt.id == uuid.Nil {
				mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUID)
			}

			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage.Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectMessage.ID.Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)

			res, err := h.Create(ctx, tt.id, tt.customerID, tt.source, tt.targets, tt.providerName, tt.text, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseMessage, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

package messagehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/messagehandlermessagebird"
)

func Test_Send(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		text         string

		responseMessage *message.Message
		responseSend    []target.Target

		expectMessage *message.Message
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
			customerID: uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			text: "hello world",

			responseMessage: &message.Message{
				ID:         uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
				CustomerID: uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"),
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
						// Status:   target.StatusSent,
						// Parts:    1,
						// TMUpdate: "2022-03-18 03:22:17.995000",
					},
				},
			},
			responseSend: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					Status:   target.StatusSent,
					Parts:    1,
					TMUpdate: "2022-03-18 03:22:17.995000",
				},
			},

			expectMessage: &message.Message{
				ID:         uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
				CustomerID: uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"),
				Type:       message.TypeSMS,
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
						Status: target.StatusQueued,
						Parts:  0,
					},
				},
				ProviderName: message.ProviderNameMessagebird,
				Text:         "hello world",
				Medias:       []string{},
				Direction:    message.DirectionOutbound,
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
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				utilHandler:               mockUtil,
				db:                        mockDB,
				reqHandler:                mockReq,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockMessagebird,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID, bmbilling.ReferenceTypeSMS, "", len(tt.destinations)).Return(true, nil)

			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)

			mockMessagebird.EXPECT().SendMessage(tt.id, tt.responseMessage.CustomerID, tt.responseMessage.Source, tt.responseMessage.Targets, tt.responseMessage.Text).Return(tt.responseSend, nil)
			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.responseSend).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageUpdated, tt.responseMessage)

			res, err := h.Send(ctx, tt.id, tt.customerID, tt.source, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.responseMessage, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

func Test_sendMessage(t *testing.T) {

	tests := []struct {
		name string

		providerName message.ProviderName
		id           uuid.UUID
		customerID   uuid.UUID
		source       *commonaddress.Address
		targets      []target.Target
		text         string

		responseSend []target.Target

		responseGet *message.Message
	}{
		{
			name: "normal",

			providerName: message.ProviderNameMessagebird,
			id:           uuid.FromStringOrNil("f9eaa2ba-a2d7-11ec-a29e-cf6eefb11b42"),
			customerID:   uuid.FromStringOrNil("fa365854-a2d7-11ec-8fe6-3b93248d4ab9"),
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
				},
			},
			text: "hello world",

			responseSend: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					Status:   target.StatusSent,
					Parts:    1,
					TMUpdate: "2022-03-18 03:22:17.995000",
				},
			},

			responseGet: &message.Message{
				ID:         uuid.FromStringOrNil("f9eaa2ba-a2d7-11ec-a29e-cf6eefb11b42"),
				CustomerID: uuid.FromStringOrNil("fa365854-a2d7-11ec-8fe6-3b93248d4ab9"),
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
						Status:   target.StatusSent,
						Parts:    1,
						TMUpdate: "2022-03-18 03:22:17.995000",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

			h := &messageHandler{
				db:                        mockDB,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockMessagebird,
			}

			ctx := context.Background()

			mockMessagebird.EXPECT().SendMessage(tt.id, tt.customerID, tt.source, tt.targets, tt.text).Return(tt.responseSend, nil)
			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.responseSend).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageUpdated, tt.responseGet)

			res, err := h.sendMessage(ctx, tt.providerName, tt.id, tt.customerID, tt.source, tt.targets, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGet, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGet, res)
			}
		})
	}
}

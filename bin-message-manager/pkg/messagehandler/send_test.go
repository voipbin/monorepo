package messagehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
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
		expectTargets []target.Target
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
					CustomerID: uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"),
				},
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
					CustomerID: uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"),
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
						Status: target.StatusQueued,
						Parts:  0,
					},
				},
				ProviderName: message.ProviderNameMessagebird,
				Text:         "hello world",
				Medias:       []string{},
				Direction:    message.DirectionOutbound,
			},
			expectTargets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					Status: target.StatusQueued,
					Parts:  0,
				},
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
			mockBird := NewMockMessageHandlerMessagebird(mc)
			mockTelnyx := NewMockMessageHandlerTelnyx(mc)

			h := &messageHandler{
				utilHandler:               mockUtil,
				db:                        mockDB,
				reqHandler:                mockReq,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockBird,
				messageHandlerTelnyx:      mockTelnyx,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID, bmbilling.ReferenceTypeSMS, "", len(tt.destinations)).Return(true, nil)

			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)

			mockTelnyx.EXPECT().SendMessage(ctx, tt.id, tt.responseMessage.Source, tt.expectTargets, tt.text).Return(tt.responseSend, nil).AnyTimes()

			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, gomock.AnyOf(message.ProviderNameTelnyx, message.ProviderNameMessagebird), tt.responseSend).Return(nil)
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

// func Test_sendMessage(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		providerName message.ProviderName
// 		id           uuid.UUID
// 		customerID   uuid.UUID
// 		source       *commonaddress.Address
// 		targets      []target.Target
// 		text         string

// 		responseSend []target.Target

// 		responseGet *message.Message
// 	}{
// 		{
// 			name: "normal",

// 			providerName: message.ProviderNameMessagebird,
// 			id:           uuid.FromStringOrNil("f9eaa2ba-a2d7-11ec-a29e-cf6eefb11b42"),
// 			customerID:   uuid.FromStringOrNil("fa365854-a2d7-11ec-8fe6-3b93248d4ab9"),
// 			source: &commonaddress.Address{
// 				Type:   commonaddress.TypeTel,
// 				Target: "+821100000001",
// 			},
// 			targets: []target.Target{
// 				{
// 					Destination: commonaddress.Address{
// 						Type:   commonaddress.TypeTel,
// 						Target: "+821100000002",
// 					},
// 				},
// 			},
// 			text: "hello world",

// 			responseSend: []target.Target{
// 				{
// 					Destination: commonaddress.Address{
// 						Type:   commonaddress.TypeTel,
// 						Target: "+821100000002",
// 					},
// 					Status:   target.StatusSent,
// 					Parts:    1,
// 					TMUpdate: "2022-03-18 03:22:17.995000",
// 				},
// 			},

// 			responseGet: &message.Message{
// 				Identity: commonidentity.Identity{
// 					ID:         uuid.FromStringOrNil("f9eaa2ba-a2d7-11ec-a29e-cf6eefb11b42"),
// 					CustomerID: uuid.FromStringOrNil("fa365854-a2d7-11ec-8fe6-3b93248d4ab9"),
// 				},
// 				Source: &commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+821100000001",
// 				},
// 				Targets: []target.Target{
// 					{
// 						Destination: commonaddress.Address{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821100000002",
// 						},
// 						Status:   target.StatusSent,
// 						Parts:    1,
// 						TMUpdate: "2022-03-18 03:22:17.995000",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockMessagebird := messagehandlermessagebird.NewMockMessageHandlerMessagebird(mc)

// 			h := &messageHandler{
// 				db:                        mockDB,
// 				notifyHandler:             mockNotify,
// 				messageHandlerMessagebird: mockMessagebird,
// 			}

// 			ctx := context.Background()

// 			mockMessagebird.EXPECT().SendMessage(ctx, tt.id, tt.source, tt.targets, tt.text).Return(tt.responseSend, nil)
// 			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.responseSend).Return(nil)
// 			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageUpdated, tt.responseGet)

// 			res, err := h.sendMessage(ctx, tt.providerName, tt.id, tt.customerID, tt.source, tt.targets, tt.text)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(tt.responseGet, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGet, res)
// 			}
// 		})
// 	}
// }

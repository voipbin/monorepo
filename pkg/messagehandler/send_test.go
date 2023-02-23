package messagehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
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
		responseSend    *message.Message

		expectMessage *message.Message
		expectRes     *message.Message
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
				ID: uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
			},
			responseSend: &message.Message{
				ID: uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
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
			expectRes: &message.Message{
				ID: uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7"),
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
				utilHandler:               mockUtil,
				db:                        mockDB,
				notifyHandler:             mockNotify,
				messageHandlerMessagebird: mockMessagebird,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)

			mockMessagebird.EXPECT().SendMessage(tt.id, tt.customerID, tt.source, tt.destinations, tt.text).Return(tt.responseSend, nil)
			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.responseSend.Targets).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageUpdated, tt.responseMessage)

			res, err := h.Send(ctx, tt.id, tt.customerID, tt.source, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_sendMessage(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		text         string

		responseSend *message.Message

		responseGet *message.Message
		expectRes   *message.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("f9eaa2ba-a2d7-11ec-a29e-cf6eefb11b42"),
			uuid.FromStringOrNil("fa365854-a2d7-11ec-8fe6-3b93248d4ab9"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			"hello world",

			&message.Message{
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

			&message.Message{
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
			&message.Message{
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

			mockMessagebird.EXPECT().SendMessage(tt.id, tt.customerID, tt.source, tt.destinations, tt.text).Return(tt.responseSend, nil)
			mockDB.EXPECT().MessageUpdateTargets(ctx, tt.id, tt.responseSend.Targets).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, message.EventTypeMessageUpdated, tt.responseGet)

			res, err := h.sendMessage(ctx, tt.id, tt.customerID, tt.source, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

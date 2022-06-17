package linehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

func Test_Event(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		data       []byte

		responseAccount *account.Account

		expectResConversations []*conversation.Conversation
		expectResMessages      []*message.Message
	}{
		{
			name: "received message",

			customerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			expectResMessages: []*message.Message{
				{
					CustomerID:    uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
					Status:        message.StatusReceived,
					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					SourceTarget:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Text:          "Hello",
					Medias:        []media.Media{},
				},
			},
		},
		{
			name: "received follow",

			customerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "follow",
						"webhookEventId": "01G49KGV3YYCWA0CPZHP9AA6H9",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884873348,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "44b7e0b5fa034a58bfd75c9e256ad2ed",
						"mode": "active"
					}
				]
			}`),

			responseAccount: &account.Account{
				ID:         uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				LineSecret: "ba5f0575d826d5b4a052a43145ef1391",
				LineToken:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},

			expectResConversations: []*conversation.Conversation{
				{
					CustomerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),

					Name:   "Sungtae Kim",
					Detail: "Conversation with Sungtae Kim",

					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

					Participants: []commonaddress.Address{
						{
							Type:       commonaddress.TypeLine,
							Target:     "",
							TargetName: "Me",
						},
						{
							Type:       commonaddress.TypeLine,
							Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
							TargetName: "Sungtae Kim",
						},
					},
				},
			},
		},
		{
			name: "received follow and message",

			customerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "follow",
						"webhookEventId": "01G49KGV3YYCWA0CPZHP9AA6H9",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884873348,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "44b7e0b5fa034a58bfd75c9e256ad2ed",
						"mode": "active"
					},
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			responseAccount: &account.Account{
				ID:         uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				LineSecret: "ba5f0575d826d5b4a052a43145ef1391",
				LineToken:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},

			expectResConversations: []*conversation.Conversation{
				{
					CustomerID: uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),

					Name:   "Sungtae Kim",
					Detail: "Conversation with Sungtae Kim",

					ReferenceType: conversation.ReferenceTypeLine,
					ReferenceID:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

					Participants: []commonaddress.Address{
						{
							Type:       commonaddress.TypeLine,
							Target:     "",
							TargetName: "Me",
						},
						{
							Type:       commonaddress.TypeLine,
							Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
							TargetName: "Sungtae Kim",
						},
					},
				},
			},
			expectResMessages: []*message.Message{
				{
					ID:             [16]byte{},
					CustomerID:     uuid.FromStringOrNil("8c9be70a-e7a6-11ec-8686-abd2812afa1e"),
					ConversationID: [16]byte{},
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					SourceTarget:   "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Text:           "Hello",
					Medias:         []media.Media{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := lineHandler{
				accountHandler: mockAccount,
			}

			ctx := context.Background()

			if len(tt.expectResConversations) > 0 {
				mockAccount.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)
			}

			resConversations, resMessages, err := h.Event(ctx, tt.customerID, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(tt.expectResConversations) > 0 && !reflect.DeepEqual(resConversations, tt.expectResConversations) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectResConversations[0], resConversations[0])
			}
			if len(tt.expectResMessages) > 0 && !reflect.DeepEqual(resMessages, tt.expectResMessages) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectResMessages[0], resMessages[0])
			}
		})
	}
}

func Test_getParticipant(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		id         string

		responseAccount *account.Account

		expectRes *commonaddress.Address
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
			id:         "Ud871bcaf7c3ad13d2a0b0d78a42a287f",

			responseAccount: &account.Account{
				ID:         uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
				LineSecret: "ba5f0575d826d5b4a052a43145ef1391",
				LineToken:  "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
			},

			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TargetName: "Sungtae Kim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := lineHandler{
				accountHandler: mockAccount,
			}

			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.customerID).Return(tt.responseAccount, nil)

			res, err := h.GetParticipant(ctx, tt.customerID, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\nres: %v\n", tt.expectRes, res)
			}
		})
	}
}
